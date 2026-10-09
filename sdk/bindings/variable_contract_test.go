package bindings

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

func TestBindingVariableSnapshotAndWriteKeepEnvelopeAndPrecision(t *testing.T) {
	const owner = 9007199254740993
	host, err := nodehost.New(context.Background(), nodehost.Config{StateDirectory: t.TempDir(), NodeID: owner})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	id := protocol.ResourceID{Owner: host.ID(), Name: "unknown/value"}
	variable, err := resource.NewVariable(resource.WritableVariableDescriptor(id, "read", "write", 2048), protocol.DataSchemaDefinition{Type: "boolean"}, resource.VariableState{Status: "ready", Value: json.RawMessage("false")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := variable.Apply(1<<53+1, resource.VariableState{Status: "ready", Value: json.RawMessage("false")}); err != nil {
		t.Fatal(err)
	}
	if err := host.Resources().Register(variable); err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(host.Client(), PublicIdentity{NodeID: host.ID(), PublicKey: host.PublicKey()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	catalogJSON, err := client.CatalogJSON(owner, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var catalog bindingCatalog
	if err := json.Unmarshal([]byte(catalogJSON), &catalog); err != nil {
		t.Fatal(err)
	}
	for _, descriptor := range catalog.Resources {
		if descriptor.ID.OwnerNodeID != "9007199254740993" {
			t.Fatalf("binding Catalog rounded NodeID: %s", catalogJSON)
		}
	}
	encoded, err := client.SnapshotJSON(owner, id.Name, 1000)
	if err != nil {
		t.Fatal(err)
	}
	state := bindingTestState(t, encoded)
	if state.Revision != "9007199254740993" || string(state.Value) != "false" {
		t.Fatalf("snapshot lost native JSON/precision: %s", encoded)
	}
	encoded, err = client.WriteVariableJSON(owner, id.Name, state.InstanceID, state.Revision, "true", 1000)
	if err != nil {
		t.Fatal(err)
	}
	updated := bindingTestState(t, encoded)
	if updated.Revision != "9007199254740994" || updated.InstanceID != state.InstanceID || string(updated.Value) != "true" {
		t.Fatalf("write lost envelope/precision: %s", encoded)
	}
	for _, value := range []string{`"true"`, `"dHJ1ZQ=="`, "{"} {
		if _, err := client.WriteVariableJSON(owner, id.Name, updated.InstanceID, updated.Revision, value, 1000); err == nil {
			t.Fatalf("invalid business value accepted: %s", value)
		}
	}
	raw, err := client.OperateJSON(owner, id.Name, "read", "", "", 1000)
	if err != nil {
		t.Fatal(err)
	}
	var operation struct {
		Schema  string          `json:"schema"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(raw), &operation); err != nil || operation.Schema != protocol.SchemaVariableValueV1 {
		t.Fatalf("bad raw operation: %s %v", raw, err)
	}
	if bindingTestState(t, string(operation.Payload)).Revision != updated.Revision {
		t.Fatalf("raw operation unwrapped standard envelope: %s", raw)
	}
}

func TestBindingVariableEventsAreJSONAndRejectInvalidState(t *testing.T) {
	id := protocol.ResourceID{Owner: 97, Name: "unknown/value"}
	initial := protocol.VariableValueV1{Version: 1, InstanceID: strings.Repeat("a", 32), Revision: "9007199254740993", Definition: &protocol.DataSchemaDefinition{Type: "boolean"}, Value: json.RawMessage("false"), Status: "ready", ObservedAtUnixMS: 1}
	for _, test := range []struct {
		name    string
		invalid bool
	}{
		{name: "standard"}, {name: "invalid data", invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			events := make(chan sdk.Event, 2)
			errorsOut := make(chan error)
			{
				payload, err := json.Marshal(initial)
				if err != nil {
					t.Fatal(err)
				}
				events <- sdk.Event{Kind: sdk.EventSnapshot, Resource: id, Revision: 1<<53 + 1, Schema: protocol.SchemaVariableValueV1, Value: payload}
				if test.invalid {
					next := initial
					next.Definition = nil
					next.Revision = "9007199254740994"
					next.Value = json.RawMessage(`"false"`)
					payload, err := json.Marshal(next)
					if err != nil {
						t.Fatal(err)
					}
					events <- sdk.Event{Kind: sdk.EventData, Resource: id, Revision: 1<<53 + 2, Schema: protocol.SchemaVariableValueV1, Value: payload}
				}
			}
			close(errorsOut)
			close(events)
			client := &Client{subscriptions: map[int64]context.CancelFunc{}}
			listener := &testListener{events: make(chan string, 2), errors: make(chan string, 2)}
			client.wg.Add(1)
			client.forwardSubscription(1, context.Background(), &sdk.DurableSubscription{Resource: id, Events: events, Errors: errorsOut}, listener)
			{
				select {
				case encoded := <-listener.events:
					var event struct {
						Revision string                   `json:"revision"`
						Value    protocol.VariableValueV1 `json:"value"`
					}
					if err := json.Unmarshal([]byte(encoded), &event); err != nil || event.Revision != initial.Revision || string(event.Value.Value) != "false" || event.Value.Definition == nil {
						t.Fatalf("event was base64/rounded/unwrapped: %s %v", encoded, err)
					}
				default:
					t.Fatal("snapshot event lost")
				}
			}
			if test.invalid {
				select {
				case <-listener.errors:
				default:
					t.Fatal("invalid variable event did not report an error")
				}
			}
			select {
			case event := <-listener.events:
				t.Fatalf("unexpected event: %s", event)
			default:
			}
		})
	}
}

func TestBindingStreamValuesKeepByteTransport(t *testing.T) {
	for name, value := range map[string][]byte{
		"binary": {0xff, 0x00, 0x80, 0xfe},
		"JSON":   []byte(`{"version":1,"value":false}`),
		"text":   []byte("hello\nworld"),
		"empty":  {},
	} {
		t.Run(name, func(t *testing.T) {
			id := protocol.ResourceID{Owner: 97, Name: "stream/bytes"}
			events := make(chan sdk.Event, 1)
			events <- sdk.Event{Kind: sdk.EventData, Resource: id, Capability: protocol.CapabilitySubscribe, Schema: "test.stream.v1", Sequence: 1, Value: value}
			failures := make(chan error)
			close(failures)
			close(events)
			client := &Client{subscriptions: map[int64]context.CancelFunc{}}
			listener := &testListener{events: make(chan string, 1), errors: make(chan string, 1)}
			client.wg.Add(1)
			client.forwardSubscription(1, context.Background(), &sdk.DurableSubscription{Resource: id, Events: events, Errors: failures}, listener)
			select {
			case encoded := <-listener.events:
				var event struct {
					Value    []byte `json:"value,omitempty"`
					Sequence uint64 `json:"sequence"`
				}
				if err := json.Unmarshal([]byte(encoded), &event); err != nil || !bytes.Equal(event.Value, value) || event.Sequence != 1 {
					t.Fatalf("stream byte transport changed: %s %v", encoded, err)
				}
			default:
				t.Fatal("legal stream event was discarded")
			}
			select {
			case failure := <-listener.errors:
				t.Fatalf("legal stream event raised an error: %s", failure)
			default:
			}
		})
	}
}

func TestBindingForwardsTerminalVariableValidationError(t *testing.T) {
	events := make(chan sdk.Event)
	failures := make(chan error, 1)
	failures <- &sdk.Error{Code: protocol.CodeMalformed, Message: "invalid variable"}
	close(failures)
	close(events)
	client := &Client{subscriptions: map[int64]context.CancelFunc{}}
	listener := &testListener{events: make(chan string, 1), errors: make(chan string, 1)}
	client.wg.Add(1)
	client.forwardSubscription(1, context.Background(), &sdk.DurableSubscription{Events: events, Errors: failures}, listener)
	select {
	case failure := <-listener.errors:
		if !strings.Contains(failure, "invalid variable") {
			t.Fatal(failure)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal error lost when event channel closed")
	}
}
