package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/yttydcs/myflowhub/protocol"
)

// Handler performs one descriptor-declared operation. The request and result
// pass through HandlerResource's schema, size, and defensive-copy boundary.
type Handler func(context.Context, OperationRequest) (OperationResult, error)

// HandlerResource dispatches arbitrary operation capabilities without adding
// type- or product-specific switches to the Registry.
type HandlerResource struct {
	descriptor Descriptor
	handlers   map[protocol.CapabilityID]Handler
}

func NewHandlerResource(descriptor Descriptor, handlers map[protocol.CapabilityID]Handler) (*HandlerResource, error) {
	normalized, err := normalizeDescriptor(descriptor)
	if err != nil {
		return nil, err
	}
	declared := make(map[protocol.CapabilityID]protocol.CapabilityDescriptorV2, len(normalized.Capabilities))
	for _, capability := range normalized.Capabilities {
		declared[capability.Name] = capability
		if eventOnlyCapability(capability) {
			if handler, exists := handlers[capability.Name]; exists {
				if handler == nil {
					return nil, fmt.Errorf("handler for event-only capability %q is nil", capability.Name)
				}
				return nil, fmt.Errorf("event-only capability %q must not have an operation handler", capability.Name)
			}
			continue
		}
		handler, exists := handlers[capability.Name]
		if !exists {
			return nil, fmt.Errorf("operation capability %q requires exactly one handler", capability.Name)
		}
		if handler == nil {
			return nil, fmt.Errorf("handler for capability %q is required", capability.Name)
		}
	}
	cloned := make(map[protocol.CapabilityID]Handler, len(handlers))
	for capability, handler := range handlers {
		declaredCapability, exists := declared[capability]
		if !exists {
			return nil, fmt.Errorf("handler capability %q is not declared by the resource", capability)
		}
		if eventOnlyCapability(declaredCapability) {
			return nil, fmt.Errorf("event-only capability %q must not have an operation handler", capability)
		}
		if handler == nil {
			return nil, fmt.Errorf("handler for capability %q is required", capability)
		}
		cloned[capability] = handler
	}
	return &HandlerResource{descriptor: normalized, handlers: cloned}, nil
}

func (r *HandlerResource) Descriptor() Descriptor {
	if r == nil {
		return Descriptor{}
	}
	return cloneDescriptor(r.descriptor)
}

func (r *HandlerResource) Operate(ctx context.Context, request OperationRequest) (OperationResult, error) {
	if ctx == nil {
		return OperationResult{}, errors.New("handler resource context is required")
	}
	if r == nil {
		return OperationResult{}, errors.New("handler resource is required")
	}
	capability, declared := r.descriptor.Capability(request.Capability)
	if !declared || eventOnlyCapability(capability) {
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
	handler, exists := r.handlers[request.Capability]
	if !exists || handler == nil {
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
	if err := validateOperationInput(r.descriptor, capability, &request); err != nil {
		return OperationResult{}, err
	}
	result, err := handler(ctx, request)
	if err != nil {
		return OperationResult{}, err
	}
	if err := validateOperationResult(r.descriptor, capability, &result); err != nil {
		return OperationResult{}, err
	}
	return result, nil
}

func eventOnlyCapability(capability protocol.CapabilityDescriptorV2) bool {
	return capability.EventSchema != "" && capability.InputSchema == "" && capability.OutputSchema == ""
}

func validateOperationInput(descriptor Descriptor, capability protocol.CapabilityDescriptorV2, request *OperationRequest) error {
	if len(request.Payload) > capability.MaxPayloadBytes || len(request.Payload) > descriptor.Limits.MaxPayloadBytes {
		return fmt.Errorf("%w: got %d", ErrValueTooLarge, len(request.Payload))
	}
	if request.Schema == "" {
		request.Schema = capability.InputSchema
	} else if capability.InputSchema != "" && request.Schema != capability.InputSchema {
		return fmt.Errorf("%w: got %q, want %q", ErrInvalidSchema, request.Schema, capability.InputSchema)
	}
	request.Payload = append([]byte(nil), request.Payload...)
	return nil
}

func validateOperationResult(descriptor Descriptor, capability protocol.CapabilityDescriptorV2, result *OperationResult) error {
	if len(result.Payload) > capability.MaxPayloadBytes || len(result.Payload) > descriptor.Limits.MaxPayloadBytes {
		return fmt.Errorf("%w: operation result got %d", ErrValueTooLarge, len(result.Payload))
	}
	if result.Schema == "" {
		result.Schema = capability.OutputSchema
	} else if capability.OutputSchema != "" && result.Schema != capability.OutputSchema {
		return fmt.Errorf("%w: result got %q, want %q", ErrInvalidSchema, result.Schema, capability.OutputSchema)
	}
	result.Payload = append([]byte(nil), result.Payload...)
	return nil
}
