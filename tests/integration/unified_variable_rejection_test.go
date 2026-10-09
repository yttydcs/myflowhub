package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

func TestUnfamiliarVariableRejectsLocalInvalidState(t *testing.T) {
	forVariableTransports(t, func(t *testing.T, ctx context.Context, graph topology, policy *auth.StaticPolicy, client *sdk.Client) {
		id := protocol.ResourceID{Owner: graph.leafB.ID(), Name: "opaque/local-validation"}
		grantVariable(policy, graph.leafA.ID(), id, auth.ActionRead, auth.ActionSubscribe)
		variable, err := graph.leafB.Registry().Variable(resource.VariableSpec{
			Name: id.Name, Definition: protocol.DataSchemaDefinition{Type: "boolean"}, ReadPermission: "opaque.read", MaxPayloadBytes: 512,
			Initial: resource.VariableState{Value: json.RawMessage("false"), Status: "ready"},
		})
		if err != nil {
			t.Fatal(err)
		}
		remote, err := client.SubscribeVariable(ctx, id, time.Minute, 4)
		if err != nil {
			t.Fatal(err)
		}
		defer remote.Cancel()
		state := decodeVariableEvent(t, receiveProductEvent(t, ctx, remote), sdk.EventSnapshot)
		for _, invalid := range []resource.VariableState{
			{Value: json.RawMessage(`"false"`), Status: "ready"},
			{Value: json.RawMessage("null"), Status: "ready"},
			{Value: json.RawMessage("false")},
			{Status: "stale"},
			{Value: json.RawMessage("false"), Status: "unavailable"},
			{Value: json.RawMessage("false"), Status: "ready", Error: &protocol.VariableError{Code: "failed", Message: "cannot be ready"}},
		} {
			if _, err := variable.Set(invalid); err == nil {
				t.Fatalf("Set accepted invalid state: %+v", invalid)
			}
			if _, err := variable.Apply(2, invalid); err == nil {
				t.Fatalf("Apply accepted invalid state: %+v", invalid)
			}
			if !reflect.DeepEqual(variable.Snapshot().State, state) {
				t.Fatalf("local rejection changed state/revision: %+v", variable.Snapshot())
			}
		}
		if _, err := variable.Apply(1, resource.VariableState{Value: json.RawMessage("true"), Status: "ready"}); err == nil {
			t.Fatal("Apply accepted a non-increasing revision")
		}
		current, err := client.Snapshot(ctx, id)
		if err != nil || !reflect.DeepEqual(current, state) {
			t.Fatalf("remote reader observed rejected local state: %+v (%v)", current, err)
		}
		assertNoVariableEvent(t, remote)
	})
}

func TestUnfamiliarVariableRejectsObsoletePayload(t *testing.T) {
	forVariableTransports(t, func(t *testing.T, ctx context.Context, graph topology, policy *auth.StaticPolicy, client *sdk.Client) {
		for index, payload := range []string{`{"enabled":false}`, `{"version":1,"revision":1,"value":"ZmFsc2U="}`} {
			id := protocol.ResourceID{Owner: graph.leafB.ID(), Name: fmt.Sprintf("opaque/obsolete/%d", index)}
			grantVariable(policy, graph.leafA.ID(), id, auth.ActionRead, auth.ActionSubscribe)
			base, err := resource.NewVariable(resource.VariableDescriptor(id, "opaque.read", 512), protocol.DataSchemaDefinition{Type: "boolean"}, resource.VariableState{Value: json.RawMessage("false"), Status: "ready"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := graph.leafB.Registry().Register(&obsoletePayloadFixture{Variable: base, payload: []byte(payload)}); err != nil {
				t.Fatal(err)
			}
			// Raw operations preserve the original wire. Only the standard variable
			// boundary validates and rejects this deliberately malformed provider.
			raw, err := client.Operate(ctx, id, protocol.CapabilityRead, "", nil)
			if err != nil || string(raw.Payload) != payload {
				t.Fatalf("raw operation altered wire payload: %+v (%v)", raw, err)
			}
			if _, err := client.Snapshot(ctx, id); !isSDKCode(err, protocol.CodeMalformed) {
				t.Fatalf("obsolete read payload was not rejected: %v", err)
			}
			remote, err := client.SubscribeVariable(ctx, id, time.Minute, 4)
			if err != nil {
				t.Fatal(err)
			}
			defer remote.Cancel()
			if err := receiveProductError(t, ctx, remote); !isSDKCode(err, protocol.CodeMalformed) {
				t.Fatalf("obsolete snapshot was not rejected: %v", err)
			}
			select {
			case event, open := <-remote.Events:
				if open {
					t.Fatalf("obsolete snapshot reached consumer: %+v", event)
				}
			case <-ctx.Done():
				t.Fatal("rejected provider subscription did not close")
			}
		}
	})
}

// Static negative fixture only; it contains no obsolete encoder or decoder.
type obsoletePayloadFixture struct {
	*resource.Variable
	payload []byte
}

func (f *obsoletePayloadFixture) Operate(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if request.Capability == protocol.CapabilityRead {
		return resource.OperationResult{Schema: protocol.SchemaVariableValueV1, Payload: f.payload}, nil
	}
	return f.Variable.Operate(ctx, request)
}

func (f *obsoletePayloadFixture) Observe(func(resource.Observation)) (*resource.Observation, func(), error) {
	return &resource.Observation{Snapshot: true, Revision: 1, Schema: protocol.SchemaVariableValueV1, Value: f.payload}, func() {}, nil
}
