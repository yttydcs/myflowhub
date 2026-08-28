package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

var ErrForbidden = errors.New("forbidden")

type Action string

const (
	ActionSubscribe Action = "subscribe"
	ActionInvoke    Action = "invoke"
	ActionRead      Action = "read"
	ActionWrite     Action = "write"
	ActionPublish   Action = "publish"
	ActionOpen      Action = "open"
)

type Request struct {
	Subject    protocol.NodeID       `json:"subject"`
	Action     Action                `json:"action"`
	Capability protocol.CapabilityID `json:"capability"`
	Resource   protocol.ResourceID   `json:"resource"`
}

type Policy interface {
	Authorize(context.Context, Request) error
}

func PolicyGeneration(policy Policy) uint64 {
	if generated, ok := policy.(interface{ Generation() uint64 }); ok {
		return generated.Generation()
	}
	return 1
}

type AllowAll struct{}

func (AllowAll) Authorize(context.Context, Request) error { return nil }

type StaticPolicy struct {
	mu      sync.RWMutex
	allowed map[Request]struct{}
}

func NewStaticPolicy() *StaticPolicy {
	return &StaticPolicy{allowed: make(map[Request]struct{})}
}

func (p *StaticPolicy) Allow(request Request) {
	request, err := normalizeRequest(request)
	if err != nil {
		return
	}
	p.mu.Lock()
	p.allowed[request] = struct{}{}
	p.mu.Unlock()
}

func (p *StaticPolicy) Authorize(_ context.Context, request Request) error {
	request, err := normalizeRequest(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	}
	p.mu.RLock()
	_, ok := p.allowed[request]
	p.mu.RUnlock()
	if !ok {
		return fmt.Errorf("%w: subject %d cannot %s %s", ErrForbidden, request.Subject, request.Action, request.Resource.Name)
	}
	return nil
}

func normalizeRequest(request Request) (Request, error) {
	if err := request.Subject.Validate(); err != nil {
		return Request{}, err
	}
	if request.Capability == "" {
		request.Capability = protocol.CapabilityID(request.Action)
	}
	if request.Action == "" {
		request.Action = Action(request.Capability)
	}
	if err := request.Capability.Validate(); err != nil {
		return Request{}, err
	}
	if request.Action != Action(request.Capability) {
		return Request{}, errors.New("policy action and capability must match")
	}
	if err := request.Resource.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func ActionFor(operation protocol.Operation) (Action, error) {
	return ActionForCapability(operation, "")
}

func ActionForCapability(operation protocol.Operation, capability protocol.CapabilityID) (Action, error) {
	switch operation {
	case protocol.OperationSubscribe, protocol.OperationUnsubscribe:
		if capability == "" {
			capability = protocol.CapabilitySubscribe
		}
	case protocol.OperationOperate, protocol.OperationSessionOpen, protocol.OperationSessionData, protocol.OperationSessionClose:
		if err := capability.Validate(); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("operation %d does not require resource authorization", operation)
	}
	return Action(capability), nil
}

func AuthorizeRequest(ctx context.Context, policy Policy, envelope protocol.Envelope) error {
	if envelope.Phase != protocol.PhaseRequest {
		return errors.New("only request phase can be adjudicated")
	}
	if policy == nil {
		return errors.New("authorization policy is required")
	}
	action, err := ActionForCapability(envelope.Operation, envelope.Capability)
	if err != nil {
		return err
	}
	return policy.Authorize(ctx, Request{Subject: envelope.Subject(), Action: action, Capability: envelope.Capability, Resource: envelope.Resource})
}

func PromoteControl(envelope protocol.Envelope, childEpoch uint64) (protocol.Envelope, error) {
	if envelope.Phase != protocol.PhaseRequest && envelope.Phase != protocol.PhaseControl {
		return protocol.Envelope{}, errors.New("only request or control can be routed downward")
	}
	if childEpoch == 0 {
		return protocol.Envelope{}, errors.New("child epoch is required")
	}
	envelope.Phase = protocol.PhaseControl
	envelope.TopologyEpoch = childEpoch
	return envelope, nil
}

func ValidateInboundChild(state *tree.State, child protocol.NodeID, childEpoch uint64, envelope protocol.Envelope) error {
	if state == nil {
		return errors.New("tree state is required")
	}
	if envelope.Phase == protocol.PhaseControl {
		return fmt.Errorf("%w: child cannot originate control", ErrForbidden)
	}
	if envelope.Principal != 0 {
		return fmt.Errorf("%w: child cannot originate a delegated principal", ErrForbidden)
	}
	return state.ValidateSource(child, envelope.Source, childEpoch)
}

func ValidateInboundParent(state *tree.State, parent protocol.NodeID, envelope protocol.Envelope) error {
	if state == nil {
		return errors.New("tree state is required")
	}
	if envelope.Phase == protocol.PhaseControl {
		return state.ValidateParentControl(parent, envelope.TopologyEpoch)
	}
	if envelope.Principal != 0 {
		return fmt.Errorf("%w: unadjudicated parent request cannot delegate a principal", ErrForbidden)
	}
	if envelope.Phase == protocol.PhaseRequest && envelope.Source != parent {
		return fmt.Errorf("%w: unadjudicated parent request cannot represent another subject", ErrForbidden)
	}
	return nil
}
