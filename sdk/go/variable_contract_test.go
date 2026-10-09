package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/transport/memory"
)

const testVariableInstance = "0123456789abcdef0123456789abcdef"

func TestVariableSDKSharedValidationVectors(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "protocol", "testdata", "variable_validation_vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	type vector struct {
		Name       string          `json:"name"`
		Definition json.RawMessage `json:"definition"`
		Value      json.RawMessage `json:"value"`
		Valid      bool            `json:"valid"`
	}
	var vectors struct {
		Version     int      `json:"version"`
		Definitions []vector `json:"definitions"`
		Values      []vector `json:"values"`
	}
	if err := json.Unmarshal(content, &vectors); err != nil {
		t.Fatal(err)
	}
	if vectors.Version != 1 || len(vectors.Definitions) == 0 || len(vectors.Values) == 0 {
		t.Fatal("shared vectors missing")
	}
	for _, group := range []struct {
		name    string
		vectors []vector
	}{{"definition", vectors.Definitions}, {"value", vectors.Values}} {
		for _, v := range group.vectors {
			t.Run(group.name+"/"+v.Name, func(t *testing.T) {
				stateFields := `"status":"unavailable"`
				if group.name == "value" {
					stateFields = `"status":"ready","value":` + string(v.Value)
				}
				payload := fmt.Sprintf(`{"version":1,"instance_id":%q,"revision":"1","definition":%s,%s,"observed_at_unix_ms":1}`, testVariableInstance, v.Definition, stateFields)
				_, err := decodeVariableEnvelope(protocol.SchemaVariableValueV1, []byte(payload), true)
				if (err == nil) != v.Valid {
					t.Fatalf("SDK boundary valid=%v want %v: %v", err == nil, v.Valid, err)
				}
			})
		}
	}
}

func testVariableEnvelope(definition *protocol.DataSchemaDefinition, value string, revision uint64) protocol.VariableValueV1 {
	return protocol.VariableValueV1{Version: 1, InstanceID: testVariableInstance, Revision: strconv.FormatUint(revision, 10), Definition: definition, Value: json.RawMessage(value), Status: "ready", ObservedAtUnixMS: 1}
}

func variableEvent(t *testing.T, kind EventKind, state protocol.VariableValueV1) Event {
	t.Helper()
	// Marshal deliberately bypasses Validate to exercise the consumer boundary.
	payload, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := strconv.ParseUint(state.Revision, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return Event{Kind: kind, Resource: protocol.ResourceID{Owner: 91, Name: "random/path"}, Capability: protocol.CapabilitySubscribe, Revision: revision, Schema: protocol.SchemaVariableValueV1, Value: payload}
}

func TestVariableSnapshotStrictEnvelopeAndNativeValues(t *testing.T) {
	for _, test := range []struct{ kind, value string }{
		{"boolean", "false"}, {"integer", "0"}, {"number", "0.3"}, {"string", `""`}, {"null", "null"},
		{"object", `{}`}, {"array", `[]`},
	} {
		t.Run(test.kind, func(t *testing.T) {
			definition := protocol.DataSchemaDefinition{ID: "unknown.random.v1", Type: test.kind}
			if test.kind == "array" {
				definition.Items = &protocol.DataSchemaDefinition{Type: "boolean"}
			}
			event := variableEvent(t, EventSnapshot, testVariableEnvelope(&definition, test.value, 1<<53+1))
			state, err := decodeVariableEnvelope(event.Schema, event.Value, true)
			if err != nil || state.Revision != "9007199254740993" || string(state.Value) != test.value || state.Definition == nil {
				t.Fatalf("native state changed: %+v (%v)", state, err)
			}
		})
	}
	definition := protocol.DataSchemaDefinition{Type: "boolean"}
	valid := string(variableEvent(t, EventSnapshot, testVariableEnvelope(&definition, "false", 1)).Value)
	for name, raw := range map[string]string{
		"old business payload": `{"version":1,"state":"ready"}`,
		"missing definition":   strings.Replace(valid, `"definition":{"type":"boolean"},`, "", 1),
		"null definition":      strings.Replace(valid, `"definition":{"type":"boolean"}`, `"definition":null`, 1),
		"wrong value type":     strings.Replace(valid, `"value":false`, `"value":"false"`, 1),
		"missing ready value":  strings.Replace(valid, `"value":false,`, "", 1),
		"numeric revision":     strings.Replace(valid, `"revision":"1"`, `"revision":1`, 1),
		"zero revision":        strings.Replace(valid, `"revision":"1"`, `"revision":"0"`, 1),
		"unknown version":      strings.Replace(valid, `"version":1`, `"version":2`, 1),
		"duplicate field":      strings.Replace(valid, `"value":false`, `"value":false,"value":true`, 1),
		"unknown field":        strings.Replace(valid, `"value":false`, `"value":false,"extra":0`, 1),
		"case alias":           strings.Replace(valid, `"value":false`, `"Value":false`, 1),
		"missing observed":     strings.Replace(valid, `,"observed_at_unix_ms":1`, "", 1),
		"trailing JSON":        valid + `{}`,
		"oversize":             strings.Repeat(" ", protocol.DefaultMaxPayload) + valid,
	} {
		t.Run(name, func(t *testing.T) {
			state, err := decodeVariableEnvelope(protocol.SchemaVariableValueV1, []byte(raw), true)
			assertSDKError(t, err, protocol.CodeMalformed, false)
			if !reflect.DeepEqual(state, protocol.VariableValueV1{}) {
				t.Fatalf("invalid response exposed a partial state: %+v", state)
			}
		})
	}
	_, err := decodeVariableEnvelope("test.private.v1", []byte(valid), true)
	assertSDKError(t, err, protocol.CodeUnsupported, false)
}

func TestVariableEventDecoderCacheAndBoundary(t *testing.T) {
	definition := protocol.DataSchemaDefinition{Type: "boolean"}
	initial := variableEvent(t, EventSnapshot, testVariableEnvelope(&definition, "false", 1<<53+1))
	data := variableEvent(t, EventData, testVariableEnvelope(nil, "true", 1<<53+2))
	var decoder VariableEventDecoder
	if _, err := decoder.Decode(data); err == nil {
		t.Fatal("data before snapshot was accepted")
	}
	state, err := decoder.Decode(initial)
	if err != nil {
		t.Fatal(err)
	}
	state.Definition.Type = "string"
	state, err = decoder.Decode(data)
	if err != nil || state.Definition.Type != "boolean" || state.Revision != "9007199254740994" || string(state.Value) != "true" {
		t.Fatalf("cache or exact revision lost: %+v (%v)", state, err)
	}
	if strings.Contains(string(data.Value), "definition") {
		t.Fatal("decoder modified wire data")
	}
	if _, err := decoder.Decode(data); err == nil {
		t.Fatal("duplicate revision accepted")
	}
	for name, mutate := range map[string]func(*Event, *protocol.VariableValueV1){
		"outer revision mismatch": func(e *Event, _ *protocol.VariableValueV1) { e.Revision++ },
		"instance change":         func(_ *Event, v *protocol.VariableValueV1) { v.InstanceID = strings.Repeat("b", 32) },
		"definition change": func(_ *Event, v *protocol.VariableValueV1) {
			v.Definition = &protocol.DataSchemaDefinition{Type: "string"}
			v.Value = json.RawMessage(`"bad"`)
		},
		"invalid business value": func(_ *Event, v *protocol.VariableValueV1) { v.Value = json.RawMessage(`"false"`) },
		"resource change":        func(e *Event, _ *protocol.VariableValueV1) { e.Resource.Owner++ },
		"schema change":          func(e *Event, _ *protocol.VariableValueV1) { e.Schema = "private.v1" },
	} {
		t.Run(name, func(t *testing.T) {
			nextState := testVariableEnvelope(nil, "false", 1<<53+3)
			next := variableEvent(t, EventData, nextState)
			mutate(&next, &nextState)
			next.Value, err = json.Marshal(nextState)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decoder.Decode(next); err == nil {
				t.Fatal("invalid variable event accepted")
			}
			if decoder.revision != 1<<53+2 || decoder.instanceID != testVariableInstance {
				t.Fatal("invalid event polluted session state")
			}
		})
	}
	for _, kind := range []EventKind{EventGap, EventExpired} {
		if _, err := decoder.Decode(Event{Kind: kind, Resource: initial.Resource}); err != nil {
			t.Fatal(err)
		}
		if _, err := decoder.Decode(data); err == nil {
			t.Fatal("invalidated definition reused")
		}
		if _, err := decoder.Decode(initial); err != nil {
			t.Fatal(err)
		}
	}
	newState := testVariableEnvelope(&protocol.DataSchemaDefinition{Type: "string"}, `"new"`, 1)
	newState.InstanceID = strings.Repeat("c", 32)
	if _, err := decoder.Decode(variableEvent(t, EventSnapshot, newState)); err != nil {
		t.Fatalf("new registration snapshot rejected: %v", err)
	}
}

func TestVariableUnavailableAndStaleRemainDistinct(t *testing.T) {
	definition := protocol.DataSchemaDefinition{Type: "boolean"}
	var decoder VariableEventDecoder
	initial := testVariableEnvelope(&definition, "", 1)
	initial.Status = "unavailable"
	if state, err := decoder.Decode(variableEvent(t, EventSnapshot, initial)); err != nil || state.Value != nil {
		t.Fatalf("unavailable became a value: %+v %v", state, err)
	}
	next := testVariableEnvelope(nil, "false", 2)
	next.Status = "stale"
	next.Error = &protocol.VariableError{Code: "sample_failed", Message: "no new reading"}
	if state, err := decoder.Decode(variableEvent(t, EventData, next)); err != nil || state.Status != "stale" || string(state.Value) != "false" {
		t.Fatalf("stale false was lost: %+v %v", state, err)
	}
}

func TestVariableDataChecksFullEnvelopeLimitWithCachedDefinition(t *testing.T) {
	definition := protocol.DataSchemaDefinition{Type: "string", Description: strings.Repeat("d", 1024)}
	var decoder VariableEventDecoder
	initial := variableEvent(t, EventSnapshot, testVariableEnvelope(&definition, `""`, 1))
	if _, err := decoder.Decode(initial); err != nil {
		t.Fatal(err)
	}
	largeValue := `"` + strings.Repeat("v", protocol.DefaultMaxPayload-500) + `"`
	data := variableEvent(t, EventData, testVariableEnvelope(nil, largeValue, 2))
	if len(data.Value) >= protocol.DefaultMaxPayload {
		t.Fatal("fixture's definition-less data must fit the payload limit")
	}
	if _, err := decoder.Decode(data); err == nil {
		t.Fatal("data accepted an oversized full envelope")
	}
	if decoder.revision != 1 {
		t.Fatal("oversized state advanced decoder revision")
	}
}

func TestVariableWritePreservesExactRevisionAndRawJSON(t *testing.T) {
	runtime := newLocalNode(t, 91)
	defer runtime.Close()
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	id := protocol.ResourceID{Owner: runtime.ID(), Name: "random/cas"}
	var calls atomic.Int32
	var received any
	variable, err := resource.NewVariable(resource.WritableVariableDescriptor(id, "read", "write", 2048), protocol.DataSchemaDefinition{Type: "number"}, resource.VariableState{Status: "ready", Value: json.RawMessage("0")}, func(_ context.Context, value any) (resource.VariableState, error) {
		calls.Add(1)
		received = value
		return resource.VariableState{Status: "ready", Value: json.RawMessage("7")}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := variable.Apply(1<<53+1, resource.VariableState{Status: "ready", Value: json.RawMessage("0")}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Registry().Register(variable); err != nil {
		t.Fatal(err)
	}
	initial, err := client.Snapshot(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	confirmed, err := client.WriteVariable(context.Background(), id, initial.InstanceID, initial.Revision, json.RawMessage("0.12345678901234567"))
	if err != nil || confirmed.Revision != "9007199254740994" || string(confirmed.Value) != "7" {
		t.Fatalf("write did not preserve owner's confirmed result: %+v %v", confirmed, err)
	}
	if number, ok := received.(json.Number); !ok || number.String() != "0.12345678901234567" {
		t.Fatalf("number passed through float64: %T %v", received, received)
	}
	for _, request := range []struct{ instance, revision, value string }{
		{initial.InstanceID, initial.Revision, "1"},
		{testVariableInstance, confirmed.Revision, "1"},
		{initial.InstanceID, "01", "1"},
		{initial.InstanceID, confirmed.Revision, `"1"`},
		{initial.InstanceID, confirmed.Revision, "9007199254740992"},
		{initial.InstanceID, confirmed.Revision, "{"},
	} {
		if _, err := client.WriteVariable(context.Background(), id, request.instance, request.revision, json.RawMessage(request.value)); err == nil {
			t.Fatalf("invalid CAS/value accepted: %+v", request)
		}
	}
	if calls.Load() != 1 || variable.Snapshot().State.Revision != confirmed.Revision {
		t.Fatal("rejected write caused a side effect")
	}
	for _, old := range []string{`{"version":2,"expected_revision":1,"value":"MQ=="}`, `{"version":3,"expected_revision":"1","value":1}`} {
		if _, err := client.Operate(context.Background(), id, protocol.CapabilityWrite, protocol.SchemaVariableWriteV3, []byte(old)); err == nil {
			t.Fatal("old write format accepted")
		}
	}
}

type variableReadPolicy struct {
	reads atomic.Int32
}

func TestVariableWriteRejectsUnconfirmedProviderResponse(t *testing.T) {
	runtime := newLocalNode(t, 91)
	defer runtime.Close()
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*protocol.VariableValueV1){
		"revision":   func(v *protocol.VariableValueV1) { v.Revision = "1" },
		"instance":   func(v *protocol.VariableValueV1) { v.InstanceID = strings.Repeat("b", 32) },
		"definition": func(v *protocol.VariableValueV1) { v.Definition = nil },
		"status":     func(v *protocol.VariableValueV1) { v.Status = "stale" },
	} {
		t.Run(name, func(t *testing.T) {
			state := testVariableEnvelope(&protocol.DataSchemaDefinition{Type: "boolean"}, "false", 2)
			mutate(&state)
			payload, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			id := protocol.ResourceID{Owner: runtime.ID(), Name: "unconfirmed/" + name}
			fixture := &variableWireFixture{descriptor: resource.WritableVariableDescriptor(id, "read", "write", 2048), payload: payload}
			if err := runtime.Registry().Register(fixture); err != nil {
				t.Fatal(err)
			}
			result, err := client.WriteVariable(context.Background(), id, testVariableInstance, "1", json.RawMessage("true"))
			assertSDKError(t, err, protocol.CodeMalformed, false)
			if !reflect.DeepEqual(result, protocol.VariableValueV1{}) {
				t.Fatal("unconfirmed result leaked usable state")
			}
		})
	}
}

func (p *variableReadPolicy) Authorize(_ context.Context, request auth.Request) error {
	if request.Capability != protocol.CapabilityRead {
		return auth.ErrForbidden
	}
	p.reads.Add(1)
	return nil
}

func TestVariableCatalogBootstrapAndReadNeedsNoSubscribe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	policy := &variableReadPolicy{}
	identity, err := auth.GenerateIdentity(91)
	if err != nil {
		t.Fatal(err)
	}
	childIdentity, err := auth.GenerateIdentity(92)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	if err := trust.Add(childIdentity.NodeID, childIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	runtime, err := node.New(ctx, node.Config{Identity: identity, Trust: trust, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	child, err := node.New(ctx, node.Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	network := memory.NewNetwork()
	defer network.Close()
	address, err := runtime.Listen(network, "sdk-read-only")
	if err != nil {
		t.Fatal(err)
	}
	if err := child.ConnectParent(ctx, network, address, runtime.ID()); err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(child)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := client.Catalog(context.Background(), runtime.ID())
	if err != nil || len(catalog.Resources) != 1 || policy.reads.Load() != 1 {
		t.Fatalf("Catalog bootstrap performed extra reads or failed to unwrap: %+v %v, reads=%d", catalog, err, policy.reads.Load())
	}
	id := protocol.ResourceID{Owner: runtime.ID(), Name: protocol.BuiltinResourceCatalog}
	state, err := client.Snapshot(context.Background(), id)
	if err != nil || state.Revision != "1" || state.Definition == nil {
		t.Fatalf("read-only snapshot failed: %+v %v", state, err)
	}
	raw, err := client.Operate(context.Background(), id, protocol.CapabilityRead, "", nil)
	if err != nil || raw.Schema != protocol.SchemaVariableValueV1 || !strings.Contains(string(raw.Payload), `"instance_id"`) {
		t.Fatalf("raw operation unwrapped variable: %+v %v", raw, err)
	}
	if _, err := client.SubscribeVariable(context.Background(), id, time.Minute, 2); err == nil {
		t.Fatal("subscribe succeeded under read-only policy")
	}
	var nilTarget *protocol.ResourceCatalogV2
	if err := client.DecodeSnapshot(context.Background(), id, nilTarget); err == nil {
		t.Fatal("typed nil target accepted")
	}
}

func TestVariableCatalogPreservesLargeNodeIDs(t *testing.T) {
	const owner protocol.NodeID = 9007199254740993
	runtime := newLocalNode(t, owner)
	defer runtime.Close()
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := client.Catalog(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Resources) != 1 || catalog.Resources[0].ID.Owner != owner {
		t.Fatalf("catalog NodeID lost precision: %+v", catalog)
	}
	id := protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog}
	state, err := client.Snapshot(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Resources []struct {
			ID struct {
				Owner json.RawMessage `json:"owner"`
			} `json:"id"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(state.Value, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Resources) != 1 || string(wire.Resources[0].ID.Owner) != `"9007199254740993"` {
		t.Fatalf("catalog variable value exposed an unsafe numeric NodeID: %s", state.Value)
	}
}

type readyVariableConnection struct{}

func (readyVariableConnection) Snapshot() ConnectionSnapshot {
	return ConnectionSnapshot{State: ConnectionConnected, Generation: 1}
}

func (readyVariableConnection) WaitChange(ctx context.Context, _ uint64) (ConnectionSnapshot, error) {
	<-ctx.Done()
	return ConnectionSnapshot{}, ctx.Err()
}

func TestDurableVariableRejectsMalformedSnapshotWithoutRetry(t *testing.T) {
	runtime := newLocalNode(t, 91)
	defer runtime.Close()
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	id := protocol.ResourceID{Owner: runtime.ID(), Name: "custom/durable"}
	fixture := &variableWireFixture{descriptor: resource.VariableDescriptor(id, "read", 2048), payload: []byte(`{"version":1,"value":false}`), observeRevision: 1}
	if err := runtime.Registry().Register(fixture); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sub, err := client.SubscribeDurableStatus(ctx, readyVariableConnection{}, id, time.Minute, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Cancel()
	select {
	case err := <-sub.Errors:
		assertSDKError(t, err, protocol.CodeMalformed, false)
	case <-ctx.Done():
		t.Fatal("durable subscription retried or lost malformed snapshot error")
	}
	for event := range sub.Events {
		t.Fatalf("durable subscription delivered malformed event: %+v", event)
	}
}

type variableWireFixture struct {
	descriptor      resource.Descriptor
	payload         []byte
	observeRevision uint64
	requests        atomic.Int32
}

func (v *variableWireFixture) Descriptor() resource.Descriptor { return v.descriptor }
func (v *variableWireFixture) Operate(context.Context, resource.OperationRequest) (resource.OperationResult, error) {
	v.requests.Add(1)
	return resource.OperationResult{Schema: protocol.SchemaVariableValueV1, Payload: v.payload}, nil
}
func (v *variableWireFixture) Observe(func(resource.Observation)) (*resource.Observation, func(), error) {
	return &resource.Observation{Snapshot: true, Revision: v.observeRevision, Schema: protocol.SchemaVariableValueV1, Value: v.payload}, func() {}, nil
}

func TestVariableCustomProviderReadAndSubscribeRejectOldPayload(t *testing.T) {
	runtime := newLocalNode(t, 91)
	defer runtime.Close()
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	for index, payload := range []string{`{"version":1,"value":false}`, fmt.Sprintf(`{"version":1,"instance_id":%q,"revision":"2","definition":{"type":"boolean"},"value":false,"status":"ready","observed_at_unix_ms":1}`, testVariableInstance)} {
		id := protocol.ResourceID{Owner: runtime.ID(), Name: fmt.Sprintf("custom/%d", index)}
		fixture := &variableWireFixture{descriptor: resource.VariableDescriptor(id, "read", 2048), payload: []byte(payload), observeRevision: 1}
		if err := runtime.Registry().Register(fixture); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			_, err := client.Snapshot(context.Background(), id)
			assertSDKError(t, err, protocol.CodeMalformed, false)
		}
		sub, err := client.SubscribeVariable(context.Background(), id, time.Minute, 2)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-sub.Errors:
			assertSDKError(t, err, protocol.CodeMalformed, false)
		case <-time.After(time.Second):
			t.Fatal("invalid subscription did not report its boundary error")
		}
		for event := range sub.Events {
			t.Fatalf("invalid event was delivered: %+v", event)
		}
		sub.Cancel()
	}
}
