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
)

type Request struct {
	Subject  protocol.NodeID
	Action   Action
	Resource protocol.ResourceID
}

type Policy interface {
	Authorize(context.Context, Request) error
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
	p.mu.Lock()
	p.allowed[request] = struct{}{}
	p.mu.Unlock()
}

func (p *StaticPolicy) Authorize(_ context.Context, request Request) error {
	p.mu.RLock()
	_, ok := p.allowed[request]
	p.mu.RUnlock()
	if !ok {
		return fmt.Errorf("%w: subject %d cannot %s %s", ErrForbidden, request.Subject, request.Action, request.Resource.Name)
	}
	return nil
}

func ActionFor(operation protocol.Operation) (Action, error) {
	switch operation {
	case protocol.OperationSubscribe, protocol.OperationUnsubscribe:
		return ActionSubscribe, nil
	case protocol.OperationCommandCall:
		return ActionInvoke, nil
	default:
		return "", fmt.Errorf("operation %d does not require resource authorization", operation)
	}
}

func AuthorizeRequest(ctx context.Context, policy Policy, envelope protocol.Envelope) error {
	if envelope.Phase != protocol.PhaseRequest {
		return errors.New("only request phase can be adjudicated")
	}
	if policy == nil {
		return errors.New("authorization policy is required")
	}
	action, err := ActionFor(envelope.Operation)
	if err != nil {
		return err
	}
	return policy.Authorize(ctx, Request{Subject: envelope.Source, Action: action, Resource: envelope.Resource})
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
	return state.ValidateSource(child, envelope.Source, childEpoch)
}

func ValidateInboundParent(state *tree.State, parent protocol.NodeID, envelope protocol.Envelope) error {
	if state == nil {
		return errors.New("tree state is required")
	}
	if envelope.Phase == protocol.PhaseControl {
		return state.ValidateParentControl(parent, envelope.TopologyEpoch)
	}
	if envelope.Phase == protocol.PhaseRequest && envelope.Source != parent {
		return fmt.Errorf("%w: unadjudicated parent request cannot represent another subject", ErrForbidden)
	}
	return nil
}
