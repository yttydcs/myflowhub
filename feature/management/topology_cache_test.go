package management

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

func TestTopologyApplyEnvelopeOverflowAndRecovery(t *testing.T) {
	id := protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology}
	domain := protocol.ManagementTopologyV1{Version: 1, Epoch: 1, Nodes: []protocol.TopologyNodeV1{{NodeID: "1", Role: "root", Generation: 1}}}
	probe, err := newJSONVariable(1, id.Name, protocol.SchemaManagementTopologyV1, "management.topology.read", domain)
	if err != nil {
		t.Fatal(err)
	}
	state := probe.Snapshot().State
	encoded, err := protocol.EncodeJSONPayload(&state, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	base, err := resource.NewVariable(resource.VariableDescriptor(id, "management.topology.read", len(encoded)+32), *state.Definition, resource.VariableState{Status: "ready", Value: state.Value}, nil)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := newTopologyResource(base)
	if err != nil {
		t.Fatal(err)
	}
	treeState := tree.TopologySnapshot{Local: 1, Version: tree.TopologyVersion{MembershipEpoch: 1}}
	current, err := buildTopologySnapshot(treeState, "", wrapped.instanceID, 1, *state.Definition)
	if err != nil {
		t.Fatal(err)
	}
	wrapped.current.Store(current)
	if err := wrapped.publish(current); err != nil {
		t.Fatal(err)
	}
	var observations []resource.Observation
	_, cancel, err := wrapped.Observe(func(event resource.Observation) { observations = append(observations, event) })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	before := base.Snapshot()
	if _, err := base.Apply(before.Revision, resource.VariableState{Value: before.State.Value, Status: "ready"}); !errors.Is(err, resource.ErrRevisionRegression) {
		t.Fatalf("regressed Apply accepted: %v", err)
	}
	if len(observations) != 0 {
		t.Fatal("regressed Apply emitted an event")
	}
	treeState.Relations = []tree.Relation{{Node: 2, Parent: 1}}
	next, err := buildTopologySnapshot(treeState, "", wrapped.instanceID, 2, *state.Definition)
	if err != nil || next.valueErr != nil {
		t.Fatalf("domain value must fit before envelope: %v", err)
	}
	wrapped.current.Store(next)
	if err := wrapped.publish(next); !errors.Is(err, protocol.ErrPayloadTooLarge) {
		t.Fatalf("missing complete envelope overflow: %v", err)
	}
	after := base.Snapshot()
	if after.Revision != before.Revision || !bytes.Equal(after.State.Value, before.State.Value) {
		t.Fatal("overflow changed committed variable state")
	}
	if len(observations) != 1 || observations[0].Failure == nil || observations[0].Failure.Code != protocol.CodeOverflow {
		t.Fatalf("overflow emitted successful value or omitted terminal failure: %+v", observations)
	}
	if _, err := wrapped.Operate(context.Background(), resource.OperationRequest{Capability: protocol.CapabilityRead}); !errors.Is(err, protocol.ErrPayloadTooLarge) {
		t.Fatalf("read silently returned old value after overflow: %v", err)
	}
	if _, stop, err := wrapped.Observe(func(resource.Observation) {}); !errors.Is(err, protocol.ErrPayloadTooLarge) {
		if stop != nil {
			stop()
		}
		t.Fatalf("Observe silently returned old value: %v", err)
	}
	if _, err := wrapped.Operate(context.Background(), resource.OperationRequest{Capability: protocol.CapabilityChildren, Payload: []byte(`{"version":1,"depth":1}`)}); err != nil {
		t.Fatalf("overflow disabled shallow query: %v", err)
	}
	treeState.Relations = nil
	recovered, err := buildTopologySnapshot(treeState, "", wrapped.instanceID, 3, *state.Definition)
	if err != nil {
		t.Fatal(err)
	}
	wrapped.current.Store(recovered)
	if err := wrapped.publish(recovered); err != nil {
		t.Fatal(err)
	}
	result, err := wrapped.Operate(context.Background(), resource.OperationRequest{Capability: protocol.CapabilityRead})
	if err != nil {
		t.Fatal(err)
	}
	var final protocol.VariableValueV1
	if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &final); err != nil {
		t.Fatal(err)
	}
	if result.Schema != protocol.SchemaVariableValueV1 || final.Definition == nil || final.Revision != "4" || final.InstanceID != before.State.InstanceID {
		t.Fatalf("invalid recovered envelope: %+v", final)
	}
	wrapped.Retire()
	if !wrapped.Retired() {
		t.Fatal("wrapper lost retirement state")
	}
	if _, err := base.Set(resource.VariableState{Value: final.Value, Status: "ready"}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("retired provider accepted Set: %v", err)
	}
}

func TestTopologyDepthCacheIsBoundedAndIsolated(t *testing.T) {
	definition, err := protocol.DataSchemaByID(protocol.SchemaManagementTopologyV1)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := buildTopologySnapshot(tree.TopologySnapshot{Local: 1, Version: tree.TopologyVersion{MembershipEpoch: 1}}, "", strings.Repeat("a", 32), 1, definition)
	if err != nil {
		t.Fatal(err)
	}
	for depth := 2; depth < 50; depth++ {
		result := snapshot.query(depth)
		if result.err != nil {
			t.Fatal(result.err)
		}
		if len(snapshot.cache) > topologyCacheEntries || snapshot.cacheBytes > topologyCacheBytes {
			t.Fatal("depth cache grew without bound")
		}
	}
	for _, depth := range []int{0, 1} {
		if _, exists := snapshot.cache[depth]; exists {
			t.Fatal("precomputed depths consumed dynamic slots")
		}
	}
}

func BenchmarkTopologySnapshot(b *testing.B) {
	definition, err := protocol.DataSchemaByID(protocol.SchemaManagementTopologyV1)
	if err != nil {
		b.Fatal(err)
	}
	for _, count := range []int{16, 4000} {
		state := tree.TopologySnapshot{Local: 1, Version: tree.TopologyVersion{MembershipEpoch: 1}}
		state.Relations = append(state.Relations, tree.Relation{Node: 2, Parent: 1})
		for id := 3; id <= count; id++ {
			state.Relations = append(state.Relations, tree.Relation{Node: protocol.NodeID(id), Parent: 2})
		}
		snapshot, err := buildTopologySnapshot(state, "", strings.Repeat("a", 32), 1, definition)
		if err != nil {
			b.Fatal(err)
		}
		for _, name := range []string{"shallow-hit", "full-hit", "depth-miss", "rebuild"} {
			b.Run(strings.Join([]string{strconv.Itoa(count) + "-nodes", name}, "/"), func(b *testing.B) {
				b.ReportAllocs()
				b.ReportMetric(float64(len(snapshot.shallow.payload)), "shallow-bytes")
				b.ReportMetric(float64(len(snapshot.value)), "value-bytes")
				for i := 0; i < b.N; i++ {
					switch name {
					case "shallow-hit":
						snapshot.query(1)
					case "full-hit":
						snapshot.query(0)
					case "depth-miss":
						snapshot.encode(2)
					case "rebuild":
						if _, err := buildTopologySnapshot(state, "", snapshot.instanceID, 1, definition); err != nil {
							b.Fatal(err)
						}
					}
				}
			})
		}
	}
}
