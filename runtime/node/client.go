package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

func (n *Node) Subscribe(ctx context.Context, resourceID protocol.ResourceID, lease time.Duration, queue int) (*RemoteSubscription, error) {
	return n.SubscribeCapability(ctx, resourceID, protocol.CapabilitySubscribe, lease, queue)
}

func (n *Node) SubscribeCapability(ctx context.Context, resourceID protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int) (*RemoteSubscription, error) {
	return n.subscribe(ctx, 0, resourceID, capability, lease, queue)
}

func (n *Node) SubscribeDelegated(ctx context.Context, delegation command.Delegation, resourceID protocol.ResourceID, lease time.Duration, queue int) (*RemoteSubscription, error) {
	principal, err := n.delegatedPrincipal(delegation)
	if err != nil {
		return nil, err
	}
	return n.subscribe(ctx, principal, resourceID, protocol.CapabilitySubscribe, lease, queue)
}

func (n *Node) subscribe(ctx context.Context, principal protocol.NodeID, resourceID protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int) (*RemoteSubscription, error) {
	if ctx == nil {
		return nil, errors.New("subscribe context is required")
	}
	if err := resourceID.Validate(); err != nil {
		return nil, err
	}
	if err := capability.Validate(); err != nil {
		return nil, err
	}
	if lease <= 0 {
		return nil, errors.New("subscribe lease must be positive")
	}
	if queue < 1 || queue > 1024 {
		return nil, errors.New("subscribe queue must be between 1 and 1024")
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		return nil, err
	}
	pending, err := n.registerPending(id, max(queue*2, 4))
	if err != nil {
		return nil, err
	}
	payload, err := encodeJSON(subscribePayload{Version: protocol.SchemaVersionV2, LeaseMS: lease.Milliseconds(), Queue: queue})
	if err != nil {
		n.unregisterPending(id, err)
		return nil, err
	}
	request := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseRequest, Operation: protocol.OperationSubscribe,
		MessageID: id, Source: n.ID(), Principal: principal, Target: resourceID.Owner, Resource: resourceID,
		DeadlineUnixMS: time.Now().Add(lease).UnixMilli(), ContentType: "application/json", Schema: "mfh.subscribe.v2",
		Capability: capability, Payload: payload,
	}
	var first protocol.Envelope
	haveFirst := false
	if sendErr := n.routeEnvelope(ctx, request, nil); sendErr != nil {
		select {
		case first = <-pending.frames:
			haveFirst = true
		default:
			n.unregisterPending(id, sendErr)
			return nil, sendErr
		}
	}
	if !haveFirst {
		first, err = awaitFrame(ctx, pending)
		if err != nil {
			n.unregisterPending(id, err)
			return nil, err
		}
	}
	if first.Operation == protocol.OperationError {
		failure, decodeErr := protocol.DecodeErrorPayload(first.Payload)
		n.unregisterPending(id, failure)
		if decodeErr != nil {
			return nil, decodeErr
		}
		return nil, failure
	}
	if first.Operation != protocol.OperationSubscribeAck && first.Operation != protocol.OperationResourceEvent {
		err := fmt.Errorf("unexpected subscribe response operation %d", first.Operation)
		n.unregisterPending(id, err)
		return nil, err
	}
	remoteCtx, remoteCancel := context.WithCancel(ctx)
	events := make(chan subscription.Event, queue)
	errorsOut := make(chan error, 1)
	if first.Operation == protocol.OperationResourceEvent {
		event, err := subscriptionEvent(first)
		if err != nil {
			remoteCancel()
			n.unregisterPending(id, err)
			return nil, err
		}
		events <- event
	}
	var cancelOnce sync.Once
	value := &RemoteSubscription{
		ID: id, Resource: resourceID, Capability: capability, Events: events, Errors: errorsOut,
		cancel: func() { cancelOnce.Do(remoteCancel) },
	}
	if !n.launch(func() { n.runRemoteSubscription(remoteCtx, id, resourceID, capability, pending, events, errorsOut) }) {
		remoteCancel()
		n.unregisterPending(id, errors.New("node is closed"))
		close(events)
		close(errorsOut)
		return nil, errors.New("node is closed")
	}
	return value, nil
}

func (n *Node) runRemoteSubscription(ctx context.Context, id protocol.MessageID, resourceID protocol.ResourceID, capability protocol.CapabilityID, pending *pendingEntry, events chan<- subscription.Event, errorsOut chan<- error) {
	defer close(events)
	defer close(errorsOut)
	defer n.unregisterPending(id, nil)
	for {
		select {
		case envelope := <-pending.frames:
			if envelope.Operation == protocol.OperationSubscribeAck {
				continue
			}
			event, err := subscriptionEvent(envelope)
			if err != nil {
				select {
				case errorsOut <- err:
				default:
				}
				return
			}
			select {
			case events <- event:
			case <-ctx.Done():
				n.sendUnsubscribe(id, resourceID, capability)
				return
			}
		case <-pending.done:
			if err := pending.Err(); err != nil {
				select {
				case errorsOut <- err:
				default:
				}
			}
			return
		case <-ctx.Done():
			n.sendUnsubscribe(id, resourceID, capability)
			return
		}
	}
}

func (n *Node) sendUnsubscribe(subscriptionID protocol.MessageID, resourceID protocol.ResourceID, capability protocol.CapabilityID) {
	envelope, err := n.newEnvelope(protocol.PhaseRequest, protocol.OperationUnsubscribe, resourceID.Owner, resourceID, capability, subscriptionID, 0, "application/json", "mfh.unsubscribe.v2", []byte("{}"))
	if err != nil {
		n.emit(err)
		return
	}
	if err := n.routeEnvelope(n.ctx, envelope, nil); err != nil && n.ctx.Err() == nil {
		n.emit(fmt.Errorf("unsubscribe: %w", err))
	}
}

func (n *Node) Operate(ctx context.Context, resourceID protocol.ResourceID, capability protocol.CapabilityID, schema string, input []byte) (resource.OperationResult, error) {
	return n.operate(ctx, 0, resourceID, capability, schema, input)
}

func (n *Node) Invoke(ctx context.Context, resourceID protocol.ResourceID, input []byte) ([]byte, error) {
	result, err := n.Operate(ctx, resourceID, protocol.CapabilityInvoke, "", input)
	return result.Payload, err
}

func (n *Node) InvokeDelegated(ctx context.Context, delegation command.Delegation, resourceID protocol.ResourceID, input []byte) ([]byte, error) {
	principal, err := n.delegatedPrincipal(delegation)
	if err != nil {
		return nil, err
	}
	result, err := n.operate(ctx, principal, resourceID, protocol.CapabilityInvoke, "", input)
	return result.Payload, err
}

func (n *Node) operate(ctx context.Context, principal protocol.NodeID, resourceID protocol.ResourceID, capability protocol.CapabilityID, schema string, input []byte) (resource.OperationResult, error) {
	if ctx == nil {
		return resource.OperationResult{}, errors.New("operate context is required")
	}
	if err := resourceID.Validate(); err != nil {
		return resource.OperationResult{}, err
	}
	if err := capability.Validate(); err != nil {
		return resource.OperationResult{}, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		return resource.OperationResult{}, err
	}
	pending, err := n.registerPending(id, 1)
	if err != nil {
		return resource.OperationResult{}, err
	}
	defer n.unregisterPending(id, nil)
	request := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseRequest, Operation: protocol.OperationOperate,
		MessageID: id, Source: n.ID(), Principal: principal, Target: resourceID.Owner, Resource: resourceID,
		DeadlineUnixMS: deadline.UnixMilli(), ContentType: "application/octet-stream", Schema: schema,
		Capability: capability, Payload: append([]byte(nil), input...),
	}
	if sendErr := n.routeEnvelope(ctx, request, nil); sendErr != nil {
		select {
		case response := <-pending.frames:
			return operationResponse(response)
		default:
			return resource.OperationResult{}, sendErr
		}
	}
	response, err := awaitFrame(ctx, pending)
	if err != nil {
		return resource.OperationResult{}, err
	}
	return operationResponse(response)
}

func (n *Node) delegatedPrincipal(delegation command.Delegation) (protocol.NodeID, error) {
	subject, ok := delegation.Subject()
	if !ok {
		return 0, errors.New("delegated operation requires an authenticated operation context")
	}
	if _, hasParent := n.tree.Parent(); hasParent {
		return 0, errors.New("only an authority root can originate delegated operations")
	}
	if subject == n.ID() {
		return 0, nil
	}
	route, err := n.tree.RouteTo(subject)
	if err != nil || route.Kind != tree.DirectionDown {
		return 0, errors.New("delegated principal is not in the authority root subtree")
	}
	return subject, nil
}

func operationResponse(response protocol.Envelope) (resource.OperationResult, error) {
	switch response.Operation {
	case protocol.OperationOperateResult:
		return resource.OperationResult{Schema: response.Schema, Payload: append([]byte(nil), response.Payload...)}, nil
	case protocol.OperationError:
		failure, err := protocol.DecodeErrorPayload(response.Payload)
		if err != nil {
			return resource.OperationResult{}, err
		}
		return resource.OperationResult{}, failure
	default:
		return resource.OperationResult{}, fmt.Errorf("unexpected operation response %d", response.Operation)
	}
}
