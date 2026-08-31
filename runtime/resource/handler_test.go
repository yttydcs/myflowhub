package resource

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func handlerDescriptor() Descriptor {
	descriptor := Descriptor{
		ID: protocol.ResourceID{Owner: 1, Name: "items"}, Type: protocol.ResourceTypeCollection, TypeVersion: 1,
		Capabilities: []protocol.CapabilityDescriptorV2{
			{Name: protocol.CapabilityGet, Permission: "items.get", InputSchema: protocol.SchemaCollectionMemberRequestV1, OutputSchema: protocol.SchemaCollectionMemberV1, MaxPayloadBytes: 64},
			{Name: protocol.CapabilityList, Permission: "items.list", InputSchema: protocol.SchemaCollectionListRequestV1, OutputSchema: protocol.SchemaCollectionPageV1, MaxPayloadBytes: 64},
			{Name: protocol.CapabilitySubscribe, Permission: "items.subscribe", EventSchema: protocol.SchemaCollectionMemberV1, MaxPayloadBytes: 64},
		},
		Schemas: []protocol.SchemaDescriptorV2{
			{ID: protocol.SchemaCollectionListRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionMemberRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionMemberV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionPageV1, ContentType: "application/json"},
		},
		Limits: protocol.ResourceLimitsV2{MaxPayloadBytes: 64},
	}
	descriptor.Sort()
	return descriptor
}

func validHandlers() map[protocol.CapabilityID]Handler {
	return map[protocol.CapabilityID]Handler{
		protocol.CapabilityGet: func(_ context.Context, request OperationRequest) (OperationResult, error) {
			return OperationResult{Payload: append([]byte("get:"), request.Payload...)}, nil
		},
		protocol.CapabilityList: func(_ context.Context, request OperationRequest) (OperationResult, error) {
			return OperationResult{Payload: append([]byte("list:"), request.Payload...)}, nil
		},
	}
}

func TestHandlerResourceRequiresExactOperationHandlers(t *testing.T) {
	tests := []struct {
		name     string
		handlers map[protocol.CapabilityID]Handler
	}{
		{"missing", map[protocol.CapabilityID]Handler{protocol.CapabilityGet: validHandlers()[protocol.CapabilityGet]}},
		{"extra", func() map[protocol.CapabilityID]Handler {
			value := validHandlers()
			value["delete"] = value[protocol.CapabilityGet]
			return value
		}()},
		{"nil", func() map[protocol.CapabilityID]Handler {
			value := validHandlers()
			value[protocol.CapabilityGet] = nil
			return value
		}()},
		{"event-only", func() map[protocol.CapabilityID]Handler {
			value := validHandlers()
			value[protocol.CapabilitySubscribe] = value[protocol.CapabilityGet]
			return value
		}()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewHandlerResource(handlerDescriptor(), test.handlers); err == nil {
				t.Fatalf("invalid handler map was accepted: %#v", test.handlers)
			}
		})
	}
	resource, err := NewHandlerResource(handlerDescriptor(), validHandlers())
	if err != nil {
		t.Fatal(err)
	}
	if resource.Descriptor().Type != protocol.ResourceTypeCollection {
		t.Fatal("handler resource changed the descriptor type")
	}
}

func TestHandlerResourceRejectsNilContextUnsupportedAndEventOnly(t *testing.T) {
	resource, err := NewHandlerResource(handlerDescriptor(), validHandlers())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resource.Operate(nil, OperationRequest{Capability: protocol.CapabilityGet}); err == nil {
		t.Fatal("nil context was accepted")
	}
	for _, capability := range []protocol.CapabilityID{protocol.CapabilitySubscribe, "delete"} {
		if _, err := resource.Operate(context.Background(), OperationRequest{Capability: capability}); !errors.Is(err, ErrUnsupportedCapability) {
			t.Fatalf("capability %q did not return unsupported: %v", capability, err)
		}
	}
}

func TestHandlerResourceAndRegistryPreserveSchemaAndSizeBoundaries(t *testing.T) {
	descriptor := handlerDescriptor()
	handlers := validHandlers()
	resource, err := NewHandlerResource(descriptor, handlers)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewRegistry(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(resource); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Operate(context.Background(), descriptor.ID, OperationRequest{
		Capability: protocol.CapabilityGet, Schema: "wrong.v1", Payload: []byte("a"),
	}); !errors.Is(err, ErrInvalidSchema) {
		t.Fatalf("registry accepted wrong input schema: %v", err)
	}
	if _, err := resource.Operate(context.Background(), OperationRequest{
		Capability: protocol.CapabilityGet, Payload: make([]byte, 65),
	}); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("handler resource accepted oversized input: %v", err)
	}

	badOutputHandlers := validHandlers()
	badOutputHandlers[protocol.CapabilityGet] = func(context.Context, OperationRequest) (OperationResult, error) {
		return OperationResult{Schema: "wrong.v1", Payload: []byte("value")}, nil
	}
	badOutput, err := NewHandlerResource(descriptor, badOutputHandlers)
	if err != nil {
		t.Fatal(err)
	}
	registry, _ = NewRegistry(1)
	if err := registry.Register(badOutput); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Operate(context.Background(), descriptor.ID, OperationRequest{
		Capability: protocol.CapabilityGet, Payload: []byte("a"),
	}); !errors.Is(err, ErrInvalidSchema) {
		t.Fatalf("registry accepted wrong output schema: %v", err)
	}

	largeOutputHandlers := validHandlers()
	largeOutputHandlers[protocol.CapabilityGet] = func(context.Context, OperationRequest) (OperationResult, error) {
		return OperationResult{Payload: make([]byte, 65)}, nil
	}
	largeOutput, err := NewHandlerResource(descriptor, largeOutputHandlers)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := largeOutput.Operate(context.Background(), OperationRequest{Capability: protocol.CapabilityGet}); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("handler resource accepted oversized output: %v", err)
	}
}

func TestHandlerResourceDefensivelyCopiesInputOutputAndDescriptor(t *testing.T) {
	descriptor := handlerDescriptor()
	sharedOutput := []byte("member")
	handlers := validHandlers()
	handlers[protocol.CapabilityGet] = func(_ context.Context, request OperationRequest) (OperationResult, error) {
		request.Payload[0] = 'X'
		return OperationResult{Payload: sharedOutput}, nil
	}
	resource, err := NewHandlerResource(descriptor, handlers)
	if err != nil {
		t.Fatal(err)
	}
	input := []byte("value")
	result, err := resource.Operate(context.Background(), OperationRequest{Capability: protocol.CapabilityGet, Payload: input})
	if err != nil {
		t.Fatal(err)
	}
	sharedOutput[0] = 'M'
	descriptor.Capabilities[0].Name = "changed"
	if string(input) != "value" || string(result.Payload) != "member" || resource.Descriptor().Capabilities[0].Name == "changed" {
		t.Fatalf("defensive copy failed: input=%q output=%q descriptor=%#v", input, result.Payload, resource.Descriptor())
	}
	if result.Schema != protocol.SchemaCollectionMemberV1 {
		t.Fatalf("default output schema was not applied: %q", result.Schema)
	}
}

func TestHandlerResourceConcurrentReadOnlyDispatch(t *testing.T) {
	resource, err := NewHandlerResource(handlerDescriptor(), validHandlers())
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	for index := range 128 {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			payload := []byte(fmt.Sprintf("%d", index))
			result, err := resource.Operate(context.Background(), OperationRequest{Capability: protocol.CapabilityList, Payload: payload})
			if err != nil || string(result.Payload) != "list:"+string(payload) || result.Schema != protocol.SchemaCollectionPageV1 {
				t.Errorf("dispatch %d: %#v %v", index, result, err)
			}
		}(index)
	}
	wait.Wait()
}
