package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/memory"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

// Each provider supplies its complete definition. Neither discovery nor value
// decoding uses a product path, semantic ID registry, or builtin value schema.
func TestUnfamiliarVariableProviderTypes(t *testing.T) {
	forVariableTransports(t, func(t *testing.T, ctx context.Context, graph topology, policy *auth.StaticPolicy, client *sdk.Client) {
		minimum, maximum := 0.0, 100.0
		tests := []struct {
			definition protocol.DataSchemaDefinition
			initial    string
			proposed   string
			confirmed  string
			invalid    string
			typed      any
		}{
			{protocol.DataSchemaDefinition{Type: "boolean"}, "false", "true", "true", `"false"`, true},
			{protocol.DataSchemaDefinition{Type: "number", Unit: "%", Minimum: &minimum, Maximum: &maximum}, "0", "75", "74", "101", json.Number("75")},
			{protocol.DataSchemaDefinition{Type: "string", Enum: []string{"", "auto", "manual"}}, `""`, `"auto"`, `"auto"`, `"other"`, "auto"},
			{protocol.DataSchemaDefinition{Type: "object", Properties: []protocol.DataSchemaProperty{
				{Name: "enabled", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "boolean"}},
				{Name: "level", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "integer"}},
			}}, `{"enabled":false,"level":0}`, `{"enabled":true,"level":7}`, `{"enabled":true,"level":7}`, `{"enabled":true}`, map[string]any{"enabled": true, "level": json.Number("7")}},
			{protocol.DataSchemaDefinition{Type: "array", Items: &protocol.DataSchemaDefinition{Type: "integer"}}, "[]", "[0,7]", "[0,7]", `["7"]`, []any{json.Number("0"), json.Number("7")}},
			{protocol.DataSchemaDefinition{Type: "string", Format: "base64", ContentType: "application/octet-stream"}, `""`, `"AP+A"`, `"AP+A"`, `"!"`, "AP+A"},
			{protocol.DataSchemaDefinition{Type: "string", Format: "uint64"}, `"0"`, `"18446744073709551615"`, `"18446744073709551615"`, `"18446744073709551616"`, "18446744073709551615"},
			{protocol.DataSchemaDefinition{Type: "string", Nullable: true}, "null", `""`, `""`, "false", ""},
		}
		for index, test := range tests {
			t.Run(fmt.Sprintf("%d-%s-%s", index, test.definition.Type, test.definition.Format), func(t *testing.T) {
				id := protocol.ResourceID{Owner: graph.leafB.ID(), Name: fmt.Sprintf("opaque/%d/%d", time.Now().UnixNano(), index)}
				definition := test.definition
				definition.ID = fmt.Sprintf("unregistered.%d.v1", time.Now().UnixNano())
				grantVariable(policy, graph.leafA.ID(), id, auth.ActionRead, auth.ActionWrite, auth.ActionSubscribe)
				var calls atomic.Int32
				variable, err := graph.leafB.Registry().Variable(resource.VariableSpec{
					Name: id.Name, Definition: definition, ReadPermission: "opaque.read", WritePermission: "opaque.write", MaxPayloadBytes: 2048,
					Initial: resource.VariableState{Value: json.RawMessage(test.initial), Status: "ready"},
					WriteHandler: func(_ context.Context, value any) (resource.VariableState, error) {
						calls.Add(1)
						if !reflect.DeepEqual(value, test.typed) {
							return resource.VariableState{}, fmt.Errorf("typed owner input = %#v, want %#v", value, test.typed)
						}
						return resource.VariableState{Value: json.RawMessage(test.confirmed), Status: "ready"}, nil
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				assertVariableDescriptor(t, variable.Descriptor(), true)
				state, err := client.Snapshot(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				assertCompleteVariable(t, state, definition, test.initial)
				remote, err := client.SubscribeVariable(ctx, id, time.Minute, 4)
				if err != nil {
					t.Fatal(err)
				}
				defer remote.Cancel()
				first := decodeVariableEvent(t, receiveProductEvent(t, ctx, remote), sdk.EventSnapshot)
				if !reflect.DeepEqual(first, state) {
					t.Fatalf("read and initial snapshot differ: read=%+v snapshot=%+v", state, first)
				}
				if _, err := client.WriteVariable(ctx, id, state.InstanceID, state.Revision, json.RawMessage(test.invalid)); err == nil {
					t.Fatal("invalid business value was accepted")
				}
				if calls.Load() != 0 || !reflect.DeepEqual(variable.Snapshot().State, state) {
					t.Fatal("rejected value invoked the owner or changed state")
				}
				confirmed, err := client.WriteVariable(ctx, id, state.InstanceID, state.Revision, json.RawMessage(test.proposed))
				if err != nil {
					t.Fatal(err)
				}
				assertCompleteVariable(t, confirmed, definition, test.confirmed)
				if calls.Load() != 1 || confirmed.InstanceID != state.InstanceID || confirmed.Revision != "2" {
					t.Fatalf("write was not confirmed exactly once: %+v, calls=%d", confirmed, calls.Load())
				}
				update := decodeVariableEvent(t, receiveProductEvent(t, ctx, remote), sdk.EventData)
				update.Definition = &definition
				if !reflect.DeepEqual(update, confirmed) {
					t.Fatalf("write and observed state differ: write=%+v data=%+v", confirmed, update)
				}
				current, err := client.Snapshot(ctx, id)
				if err != nil || !reflect.DeepEqual(current, confirmed) {
					t.Fatalf("read did not return complete current envelope: %+v (%v)", current, err)
				}
			})
		}
	})
}

func TestUnfamiliarVariableCASPermissionsAndRecreation(t *testing.T) {
	forVariableTransports(t, func(t *testing.T, ctx context.Context, graph topology, policy *auth.StaticPolicy, client *sdk.Client) {
		id := protocol.ResourceID{Owner: graph.leafB.ID(), Name: fmt.Sprintf("opaque/%d/state", time.Now().UnixNano())}
		grantVariable(policy, graph.leafA.ID(), id, auth.ActionRead, auth.ActionWrite, auth.ActionSubscribe)
		grantVariable(policy, graph.branchA.ID(), id, auth.ActionRead)
		oldCommand := protocol.ResourceID{Owner: id.Owner, Name: id.Name + "/set"}
		grantVariable(policy, graph.branchA.ID(), oldCommand, auth.ActionInvoke)
		var calls atomic.Int32
		definition := protocol.DataSchemaDefinition{Type: "integer"}
		variable, err := graph.leafB.Registry().Variable(resource.VariableSpec{
			Name: id.Name, Definition: definition, ReadPermission: "opaque.read", WritePermission: "opaque.write", MaxPayloadBytes: 512,
			Initial: resource.VariableState{Value: json.RawMessage("0"), Status: "ready"},
			WriteHandler: func(_ context.Context, value any) (resource.VariableState, error) {
				calls.Add(1)
				data, err := json.Marshal(value)
				return resource.VariableState{Value: data, Status: "ready"}, err
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		const highRevision uint64 = 9007199254740993
		if _, err := variable.Apply(highRevision, resource.VariableState{Value: json.RawMessage("0"), Status: "ready"}); err != nil {
			t.Fatal(err)
		}
		state, err := client.Snapshot(ctx, id)
		if err != nil || state.Revision != "9007199254740993" {
			t.Fatalf("large revision lost precision: %+v (%v)", state, err)
		}
		readClient, err := sdk.NewAttachedClient(graph.branchA)
		if err != nil {
			t.Fatal(err)
		}
		readOnly, err := readClient.Snapshot(ctx, id)
		if err != nil || !reflect.DeepEqual(readOnly, state) {
			t.Fatalf("read-only principal could not read definition/version: %+v (%v)", readOnly, err)
		}
		if denied, err := readClient.SubscribeVariable(ctx, id, time.Minute, 4); !isSDKCode(err, protocol.CodeForbidden) {
			if denied != nil {
				denied.Cancel()
			}
			t.Fatalf("read grant authorized subscribe: %v", err)
		}
		if _, err := readClient.WriteVariable(ctx, id, state.InstanceID, state.Revision, json.RawMessage("1")); !isSDKCode(err, protocol.CodeForbidden) {
			t.Fatalf("read/old command grant authorized write: %v", err)
		}
		if _, err := readClient.Invoke(ctx, oldCommand, nil); err == nil {
			t.Fatal("old /set command unexpectedly exists")
		}
		if calls.Load() != 0 || !reflect.DeepEqual(variable.Snapshot().State, state) {
			t.Fatal("denied operations caused side effects")
		}
		remote, err := client.SubscribeVariable(ctx, id, time.Minute, 8)
		if err != nil {
			t.Fatal(err)
		}
		defer remote.Cancel()
		initial := decodeVariableEvent(t, receiveProductEvent(t, ctx, remote), sdk.EventSnapshot)
		if !reflect.DeepEqual(initial, state) {
			t.Fatal("subscription snapshot lost the large revision or definition")
		}
		for _, input := range []struct{ schema, payload string }{
			{"mfh.variable-write.v2", `{"version":2,"expected_revision":1,"value":"MQ=="}`},
			{protocol.SchemaVariableWriteV3, `{"version":2,"expected_revision":1,"value":"MQ=="}`},
			{protocol.SchemaVariableWriteV3, fmt.Sprintf(`{"version":3,"expected_instance_id":%q,"expected_revision":%q,"value":1,"status":"ready"}`, state.InstanceID, state.Revision)},
		} {
			if _, err := client.Operate(ctx, id, protocol.CapabilityWrite, input.schema, []byte(input.payload)); err == nil {
				t.Fatalf("obsolete or overprivileged write accepted: %s", input.payload)
			}
		}
		if calls.Load() != 0 || !reflect.DeepEqual(variable.Snapshot().State, state) {
			t.Fatal("invalid wire input changed owner state")
		}
		type result struct {
			state protocol.VariableValueV1
			err   error
		}
		results := make(chan result, 2)
		start := make(chan struct{})
		for _, value := range []string{"11", "22"} {
			go func(value string) {
				<-start
				next, err := client.WriteVariable(ctx, id, state.InstanceID, state.Revision, json.RawMessage(value))
				results <- result{next, err}
			}(value)
		}
		close(start)
		var winner protocol.VariableValueV1
		succeeded, conflicts := 0, 0
		for range 2 {
			select {
			case got := <-results:
				if got.err == nil {
					winner = got.state
					succeeded++
				} else if isSDKCode(got.err, protocol.CodeConflict) {
					conflicts++
				} else {
					t.Fatalf("unexpected CAS error: %v", got.err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}
		if succeeded != 1 || conflicts != 1 || calls.Load() != 1 || winner.Revision != "9007199254740994" {
			t.Fatalf("CAS results: successes=%d conflicts=%d calls=%d state=%+v", succeeded, conflicts, calls.Load(), winner)
		}
		assertCompleteVariable(t, winner, definition, string(winner.Value))
		update := decodeVariableEvent(t, receiveProductEvent(t, ctx, remote), sdk.EventData)
		update.Definition = &definition
		current, err := client.Snapshot(ctx, id)
		if err != nil || !reflect.DeepEqual(update, winner) || !reflect.DeepEqual(current, winner) {
			t.Fatalf("CAS read/write/event envelopes diverged: %+v / %+v / %+v (%v)", current, winner, update, err)
		}
		assertNoVariableEvent(t, remote)
		if err := graph.leafB.Registry().Remove(id); err != nil {
			t.Fatal(err)
		}
		if err := receiveProductError(t, ctx, remote); !isSDKCode(err, protocol.CodeGone) {
			t.Fatalf("retired variable did not terminate subscription: %v", err)
		}
		replacement, err := graph.leafB.Registry().Variable(resource.VariableSpec{
			Name: id.Name, Definition: protocol.DataSchemaDefinition{Type: "boolean"}, ReadPermission: "opaque.read", WritePermission: "opaque.write", MaxPayloadBytes: 512,
			Initial: resource.VariableState{Value: json.RawMessage("false"), Status: "ready"},
			WriteHandler: func(_ context.Context, value any) (resource.VariableState, error) {
				calls.Add(1)
				data, err := json.Marshal(value)
				return resource.VariableState{Value: data, Status: "ready"}, err
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		newState, err := client.Snapshot(ctx, id)
		if err != nil || newState.InstanceID == state.InstanceID || newState.Revision != "1" {
			t.Fatalf("replacement reused registration identity: %+v (%v)", newState, err)
		}
		if _, err := client.WriteVariable(ctx, id, state.InstanceID, newState.Revision, json.RawMessage("true")); !isSDKCode(err, protocol.CodeConflict) {
			t.Fatalf("old instance wrote to replacement: %v", err)
		}
		if _, err := readClient.WriteVariable(ctx, id, newState.InstanceID, newState.Revision, json.RawMessage("true")); !isSDKCode(err, protocol.CodeForbidden) {
			t.Fatalf("replacement changed principal permissions: %v", err)
		}
		if calls.Load() != 1 || !reflect.DeepEqual(replacement.Snapshot().State, newState) {
			t.Fatal("rejected replacement writes invoked owner or changed state")
		}
		fresh, err := client.SubscribeVariable(ctx, id, time.Minute, 4)
		if err != nil {
			t.Fatal(err)
		}
		defer fresh.Cancel()
		if snapshot := decodeVariableEvent(t, receiveProductEvent(t, ctx, fresh), sdk.EventSnapshot); !reflect.DeepEqual(snapshot, newState) {
			t.Fatalf("replacement subscription did not refresh definition: %+v", snapshot)
		}
		if _, err := client.WriteVariable(ctx, id, newState.InstanceID, newState.Revision, json.RawMessage("true")); err != nil {
			t.Fatal(err)
		}
		if next := decodeVariableEvent(t, receiveProductEvent(t, ctx, fresh), sdk.EventData); string(next.Value) != "true" || next.InstanceID != newState.InstanceID || calls.Load() != 2 {
			t.Fatalf("replacement write failed: %+v calls=%d", next, calls.Load())
		}
		readOnlyID := protocol.ResourceID{Owner: id.Owner, Name: id.Name + "/readonly"}
		readOnlyVariable, err := graph.leafB.Registry().Variable(resource.VariableSpec{
			Name: readOnlyID.Name, Definition: definition, ReadPermission: "opaque.read", MaxPayloadBytes: 512,
			Initial: resource.VariableState{Value: json.RawMessage("0"), Status: "ready"},
		})
		if err != nil {
			t.Fatal(err)
		}
		assertVariableDescriptor(t, readOnlyVariable.Descriptor(), false)
		grantVariable(policy, graph.leafA.ID(), readOnlyID, auth.ActionRead, auth.ActionWrite)
		readOnlyState, err := client.Snapshot(ctx, readOnlyID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.WriteVariable(ctx, readOnlyID, readOnlyState.InstanceID, readOnlyState.Revision, json.RawMessage("1")); !isSDKCode(err, protocol.CodeUnsupported) {
			t.Fatalf("grant created a missing write capability: %v", err)
		}
		if !reflect.DeepEqual(readOnlyVariable.Snapshot().State, readOnlyState) {
			t.Fatal("read-only variable changed after rejected write")
		}
	})
}

func TestUnfamiliarVariableCompleteEnvelopeLimit(t *testing.T) {
	forVariableTransports(t, func(t *testing.T, ctx context.Context, graph topology, policy *auth.StaticPolicy, client *sdk.Client) {
		id := protocol.ResourceID{Owner: graph.leafB.ID(), Name: "opaque/bounded"}
		grantVariable(policy, graph.leafA.ID(), id, auth.ActionRead, auth.ActionWrite, auth.ActionSubscribe)
		const limit = 512
		definition := protocol.DataSchemaDefinition{Type: "string", Title: strings.Repeat("t", 160)}
		var calls atomic.Int32
		variable, err := graph.leafB.Registry().Variable(resource.VariableSpec{
			Name: id.Name, Definition: definition, ReadPermission: "opaque.read", WritePermission: "opaque.write", MaxPayloadBytes: limit,
			Initial: resource.VariableState{Value: json.RawMessage(`""`), Status: "ready"},
			WriteHandler: func(_ context.Context, value any) (resource.VariableState, error) {
				calls.Add(1)
				data, err := json.Marshal(value)
				return resource.VariableState{Value: data, Status: "ready"}, err
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		state := variable.Snapshot().State
		encoded, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		// Compute from the complete envelope; an otherwise legal business value
		// and V3 request can both fit while its full published state exceeds 512.
		value := json.RawMessage(strconv.Quote(strings.Repeat("x", limit-len(encoded)+1)))
		proposed := state
		proposed.Value = value
		overflow, err := json.Marshal(proposed)
		if err != nil || len(value) >= limit || len(overflow) != limit+1 {
			t.Fatalf("invalid boundary fixture: business=%d envelope=%d (%v)", len(value), len(overflow), err)
		}
		request, err := protocol.EncodeJSONPayload(&protocol.VariableWriteV3{Version: 3, ExpectedInstanceID: state.InstanceID, ExpectedRevision: state.Revision, Value: value}, limit)
		if err != nil || len(request) >= limit {
			t.Fatalf("write request itself must fit: %d (%v)", len(request), err)
		}
		remote, err := client.SubscribeVariable(ctx, id, time.Minute, 4)
		if err != nil {
			t.Fatal(err)
		}
		defer remote.Cancel()
		_ = decodeVariableEvent(t, receiveProductEvent(t, ctx, remote), sdk.EventSnapshot)
		for _, mutate := range []func() error{
			func() error {
				_, err := variable.Set(resource.VariableState{Value: value, Status: "ready"})
				return err
			},
			func() error {
				_, err := variable.Apply(2, resource.VariableState{Value: value, Status: "ready"})
				return err
			},
			func() error { _, err := variable.CompareAndSet(ctx, state.InstanceID, 1, value); return err },
			func() error {
				_, err := client.Operate(ctx, id, protocol.CapabilityWrite, protocol.SchemaVariableWriteV3, request)
				return err
			},
		} {
			if err := mutate(); err == nil {
				t.Fatal("oversized complete envelope was accepted")
			}
			if calls.Load() != 0 || !reflect.DeepEqual(variable.Snapshot().State, state) {
				t.Fatal("overflow invoked owner or changed revision/state")
			}
		}
		assertNoVariableEvent(t, remote)
		catalogBefore := graph.leafB.Registry().Catalog().Snapshot()
		rejectedID := protocol.ResourceID{Owner: id.Owner, Name: "opaque/rejected"}
		if _, err := graph.leafB.Registry().Variable(resource.VariableSpec{Name: rejectedID.Name, Definition: definition, ReadPermission: "opaque.read", MaxPayloadBytes: limit, Initial: resource.VariableState{Value: value, Status: "ready"}}); err == nil {
			t.Fatal("oversized initial envelope was registered")
		}
		if _, exists := graph.leafB.Registry().Resolve(rejectedID); exists || !reflect.DeepEqual(graph.leafB.Registry().Catalog().Snapshot(), catalogBefore) {
			t.Fatal("rejected registration partially changed visibility or catalog")
		}
		// One byte less fits the exact same resource limit and is observable.
		fit := json.RawMessage(strconv.Quote(strings.Repeat("x", limit-len(encoded))))
		confirmed, err := client.WriteVariable(ctx, id, state.InstanceID, state.Revision, fit)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := json.Marshal(confirmed)
		if err != nil || len(wire) != limit || !bytes.Equal(confirmed.Value, fit) || calls.Load() != 1 {
			t.Fatalf("exact-limit state was not committed: bytes=%d calls=%d (%v)", len(wire), calls.Load(), err)
		}
		_ = decodeVariableEvent(t, receiveProductEvent(t, ctx, remote), sdk.EventData)
	})
}

func forVariableTransports(t *testing.T, run func(*testing.T, context.Context, topology, *auth.StaticPolicy, *sdk.Client)) {
	t.Helper()
	for _, name := range []string{"memory", "tcp"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			var driver link.Driver
			endpoint := func(string) link.Endpoint { return "127.0.0.1:0" }
			if name == "memory" {
				network := memory.NewNetwork()
				defer network.Close()
				driver = network
				endpoint = func(name string) link.Endpoint { return link.Endpoint("variable-" + name) }
			} else {
				driver = tcp.Driver{}
			}
			policy := auth.NewStaticPolicy()
			graph := buildTopology(t, ctx, driver, endpoint, policy)
			defer closeTopology(graph)
			client, err := sdk.NewAttachedClient(graph.leafA)
			if err != nil {
				t.Fatal(err)
			}
			run(t, ctx, graph, policy, client)
		})
	}
}

func grantVariable(policy *auth.StaticPolicy, subject protocol.NodeID, id protocol.ResourceID, actions ...auth.Action) {
	for _, action := range actions {
		policy.Allow(auth.Request{Subject: subject, Action: action, Resource: id})
	}
}

func assertVariableDescriptor(t *testing.T, descriptor resource.Descriptor, writable bool) {
	t.Helper()
	read, hasRead := descriptor.Capability(protocol.CapabilityRead)
	subscribe, hasSubscribe := descriptor.Capability(protocol.CapabilitySubscribe)
	write, hasWrite := descriptor.Capability(protocol.CapabilityWrite)
	if descriptor.Type != protocol.ResourceTypeVariable || descriptor.TypeVersion != 2 || !hasRead || !hasSubscribe || read.OutputSchema != protocol.SchemaVariableValueV1 || subscribe.EventSchema != protocol.SchemaVariableValueV1 || hasWrite != writable {
		t.Fatalf("nonstandard variable descriptor: %+v", descriptor)
	}
	if writable && (write.InputSchema != protocol.SchemaVariableWriteV3 || write.OutputSchema != protocol.SchemaVariableValueV1) {
		t.Fatalf("nonstandard variable write: %+v", write)
	}
}

func assertCompleteVariable(t *testing.T, state protocol.VariableValueV1, definition protocol.DataSchemaDefinition, value string) {
	t.Helper()
	if err := state.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.Definition, &definition) || string(state.Value) != value || state.Status != "ready" || state.ObservedAtUnixMS <= 0 {
		t.Fatalf("unexpected complete variable state: %+v, want definition=%+v value=%s", state, definition, value)
	}
}

func decodeVariableEvent(t *testing.T, event sdk.Event, kind sdk.EventKind) protocol.VariableValueV1 {
	t.Helper()
	if event.Kind != kind || event.Schema != protocol.SchemaVariableValueV1 {
		t.Fatalf("unexpected variable event: %+v", event)
	}
	var state protocol.VariableValueV1
	if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &state); err != nil {
		t.Fatal(err)
	}
	revision, err := protocol.ParseVariableRevision(state.Revision)
	if err != nil || revision != event.Revision || (kind == sdk.EventSnapshot) != (state.Definition != nil) {
		t.Fatalf("wire definition or revision mismatch: %+v / %+v (%v)", event, state, err)
	}
	return state
}

func assertNoVariableEvent(t *testing.T, remote *sdk.Subscription) {
	t.Helper()
	select {
	case event := <-remote.Events:
		t.Fatalf("rejected mutation published an event: %+v", event)
	case err := <-remote.Errors:
		t.Fatalf("rejected mutation terminated subscription: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
}
