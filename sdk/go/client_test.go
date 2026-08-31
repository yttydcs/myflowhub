package sdk

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestClientRequiresRuntime(t *testing.T) {
	if _, err := NewClient(nil); err == nil {
		t.Fatal("nil runtime was accepted")
	}
	if _, err := NewAttachedClient(nil); err == nil {
		t.Fatal("nil attached runtime was accepted")
	}
}

func TestAttachedClientCloseCannotCloseOrDetachRuntime(t *testing.T) {
	runtime := newLocalNode(t, 91)
	defer runtime.Close()
	variableID := protocol.ResourceID{Owner: runtime.ID(), Name: "test/status"}
	variable, err := resource.NewVariable(resource.VariableDescriptor(variableID, "text/plain", "test.status.v1", "test.read", 64), []byte("ready"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Registry().Register(variable); err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := client.NodeID(); err != nil || id != runtime.ID() {
		t.Fatalf("unexpected attached client NodeID %d: %v", id, err)
	}
	if err := client.Close(); !errors.Is(err, ErrAttachedClientClose) {
		t.Fatalf("attached Close returned %v", err)
	}
	if err := client.Connect(context.Background(), nil, "unused", 1); !errors.Is(err, ErrAttachedClientConnection) {
		t.Fatalf("attached Connect returned %v", err)
	}
	if _, err := client.ConnectManaged(context.Background(), nil, "unused", 1, node.SupervisorConfig{}); !errors.Is(err, ErrAttachedClientConnection) {
		t.Fatalf("attached ConnectManaged returned %v", err)
	}
	select {
	case <-runtime.Done():
		t.Fatal("attached client Close closed the node")
	default:
	}
	event, err := client.Snapshot(context.Background(), variableID)
	if err != nil || string(event.Value) != "ready" {
		t.Fatalf("attached client was not usable after guarded Close: event=%+v err=%v", event, err)
	}
}

func TestLegacyClientCloseStillOwnsRuntime(t *testing.T) {
	runtime := newLocalNode(t, 92)
	client, err := NewClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-runtime.Done():
	default:
		t.Fatal("legacy client Close did not close its node")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("repeated legacy Close returned %v", err)
	}
}

func TestTypedCollectionOperationsShareDirectAndAttachedNodePath(t *testing.T) {
	runtime := newLocalNode(t, 93)
	defer runtime.Close()
	resourceID := protocol.ResourceID{Owner: runtime.ID(), Name: "test/collection"}
	descriptor := resource.Descriptor{
		ID: resourceID, Type: protocol.ResourceTypeCollection, TypeVersion: 1,
		Capabilities: []protocol.CapabilityDescriptorV2{
			{Name: protocol.CapabilityGet, Permission: "test.get", InputSchema: protocol.SchemaCollectionMemberRequestV1, OutputSchema: protocol.SchemaCollectionMemberV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
			{Name: protocol.CapabilityList, Permission: "test.list", InputSchema: protocol.SchemaCollectionListRequestV1, OutputSchema: protocol.SchemaCollectionPageV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
		},
		Schemas: []protocol.SchemaDescriptorV2{
			{ID: protocol.SchemaCollectionListRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionMemberRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionMemberV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionPageV1, ContentType: "application/json"},
		},
		Limits: protocol.ResourceLimitsV2{MaxPayloadBytes: protocol.DefaultMaxPayload},
	}
	descriptor.Sort()
	collectionResource, err := resource.NewHandlerResource(descriptor, map[protocol.CapabilityID]resource.Handler{
		protocol.CapabilityList: func(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
			var list protocol.CollectionListRequestV1
			if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &list); err != nil {
				return resource.OperationResult{}, err
			}
			if list.Cursor == "malformed" {
				return resource.OperationResult{Payload: []byte("{")}, nil
			}
			page := protocol.CollectionPageV1{Version: 1, Revision: 1, Members: []protocol.CollectionMemberV1{{
				Key: "item", Kind: "test-item", Label: "Item", Capabilities: []protocol.CapabilityID{protocol.CapabilityGet},
			}}}
			payload, err := protocol.EncodeJSONPayload(&page, protocol.DefaultMaxPayload)
			return resource.OperationResult{Payload: payload}, err
		},
		protocol.CapabilityGet: func(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
			var memberRequest protocol.CollectionMemberRequestV1
			if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &memberRequest); err != nil {
				return resource.OperationResult{}, err
			}
			member := protocol.CollectionMemberV1{Key: memberRequest.Key, Kind: "test-item", Label: "Item"}
			payload, err := protocol.EncodeJSONPayload(&member, protocol.DefaultMaxPayload)
			return resource.OperationResult{Payload: payload}, err
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Registry().Register(collectionResource); err != nil {
		t.Fatal(err)
	}
	direct, err := NewClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	attached, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	for name, client := range map[string]*Client{"direct": direct, "attached": attached} {
		t.Run(name, func(t *testing.T) {
			collection, err := client.Collection(resourceID)
			if err != nil {
				t.Fatal(err)
			}
			page, err := collection.List(context.Background(), protocol.CollectionListRequestV1{Version: 1, Limit: 1})
			if err != nil || len(page.Members) != 1 || page.Members[0].Key != "item" {
				t.Fatalf("unexpected typed list: %#v (%v)", page, err)
			}
			member, err := collection.Get(context.Background(), protocol.CollectionMemberRequestV1{Version: 1, Key: "item"})
			if err != nil || member.Key != "item" {
				t.Fatalf("unexpected typed get: %#v (%v)", member, err)
			}
		})
	}

	request := protocol.CollectionListRequestV1{Version: 1, Limit: 1}
	var response protocol.CollectionPageV1
	if err := direct.OperatePayload(nil, resourceID, protocol.CapabilityList, &request, &response); err == nil {
		t.Fatal("typed operation accepted nil context")
	}
	var nilClient *Client
	if err := nilClient.OperatePayload(context.Background(), resourceID, protocol.CapabilityList, &request, &response); err == nil {
		t.Fatal("typed operation accepted nil client")
	}
	if err := direct.OperatePayload(context.Background(), resourceID, protocol.CapabilityList, nil, &response); err == nil {
		t.Fatal("typed operation accepted nil request")
	}
	if err := direct.OperatePayload(context.Background(), resourceID, protocol.CapabilityList, &request, nil); err == nil {
		t.Fatal("typed operation accepted nil response")
	}
	var typedNilRequest *protocol.CollectionListRequestV1
	if err := direct.OperatePayload(context.Background(), resourceID, protocol.CapabilityList, typedNilRequest, &response); err == nil {
		t.Fatal("typed operation accepted typed nil request")
	}
	var typedNilResponse *protocol.CollectionPageV1
	if err := direct.OperatePayload(context.Background(), resourceID, protocol.CapabilityList, &request, typedNilResponse); err == nil {
		t.Fatal("typed operation accepted typed nil response")
	}
	malformed := protocol.CollectionListRequestV1{Version: 1, Cursor: "malformed", Limit: 1}
	if err := direct.OperatePayload(context.Background(), resourceID, protocol.CapabilityList, &malformed, &response); err == nil {
		t.Fatal("typed operation accepted malformed response")
	} else {
		assertSDKError(t, err, protocol.CodeMalformed, false)
	}
	if err := direct.OperatePayload(context.Background(), resourceID, "delete", &request, &response); err == nil {
		t.Fatal("typed operation accepted unknown capability")
	} else {
		assertSDKError(t, err, protocol.CodeUnsupported, false)
	}
	payload, err := protocol.EncodeJSONPayload(&request, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := direct.Operate(context.Background(), resourceID, protocol.CapabilityList, "wrong.schema.v1", payload); err == nil {
		t.Fatal("direct operation accepted wrong schema")
	} else {
		assertSDKError(t, err, protocol.CodeMalformed, false)
	}
	if err := direct.Close(); err != nil {
		t.Fatal(err)
	}
	if err := direct.OperatePayload(context.Background(), resourceID, protocol.CapabilityList, &request, &response); err == nil {
		t.Fatal("typed operation accepted closed client")
	}
}

func TestFlowClientPublicMethodsRejectNilReceiver(t *testing.T) {
	var flows *FlowClient
	checks := []func() error{
		func() error {
			_, err := flows.ListDefinitions(context.Background(), protocol.CollectionListRequestV1{})
			return err
		},
		func() error {
			_, err := flows.ListRuns(context.Background(), protocol.CollectionListRequestV1{})
			return err
		},
		func() error {
			_, err := flows.GetDefinition(context.Background(), protocol.CollectionMemberRequestV1{})
			return err
		},
		func() error {
			_, err := flows.GetRun(context.Background(), protocol.CollectionMemberRequestV1{})
			return err
		},
		func() error { _, err := flows.Events(context.Background(), time.Second, 1); return err },
		func() error { _, err := flows.Create(context.Background(), protocol.FlowDefinitionV1{}); return err },
		func() error { _, err := flows.Update(context.Background(), protocol.FlowDefinitionV1{}); return err },
		func() error { _, err := flows.Run(context.Background(), protocol.FlowRunV1{}); return err },
		func() error { _, err := flows.Cancel(context.Background(), protocol.FlowCancelV1{}); return err },
		func() error { return flows.Archive(context.Background(), protocol.FlowArchiveV1{}) },
	}
	for index, check := range checks {
		if err := check(); err == nil {
			t.Fatalf("nil Flow client method %d returned no error", index)
		}
	}
}

func newLocalNode(t *testing.T, id protocol.NodeID) *node.Node {
	t.Helper()
	identity, err := auth.GenerateIdentity(id)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := node.New(context.Background(), node.Config{
		Identity: identity,
		Trust:    auth.NewTrustStore(),
		Policy:   auth.AllowAll{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}
