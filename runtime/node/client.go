package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

func (n *Node) Subscribe(ctx context.Context, resource protocol.ResourceID, lease time.Duration, queue int) (*RemoteSubscription, error) {
	if ctx == nil {
		return nil, errors.New("subscribe context is required")
	}
	if err := resource.Validate(); err != nil {
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
	payload, err := encodeJSON(subscribePayload{LeaseMS: lease.Milliseconds(), Queue: queue})
	if err != nil {
		n.unregisterPending(id, err)
		return nil, err
	}
	request := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseRequest, Operation: protocol.OperationSubscribe,
		MessageID: id, Source: n.ID(), Target: resource.Owner, Resource: resource, DeadlineUnixMS: time.Now().Add(lease).UnixMilli(),
		ContentType: "application/json", Schema: "subscribe.v1", Payload: payload,
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
	if first.Operation != protocol.OperationSubscribeAck && first.Operation != protocol.OperationVariableSnapshot {
		err := fmt.Errorf("unexpected subscribe response operation %d", first.Operation)
		n.unregisterPending(id, err)
		return nil, err
	}
	remoteCtx, remoteCancel := context.WithCancel(ctx)
	events := make(chan subscription.Event, queue)
	errorsOut := make(chan error, 1)
	if first.Operation == protocol.OperationVariableSnapshot {
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
		ID: id, Resource: resource, Events: events, Errors: errorsOut,
		cancel: func() { cancelOnce.Do(remoteCancel) },
	}
	if !n.launch(func() { n.runRemoteSubscription(remoteCtx, id, resource, pending, events, errorsOut) }) {
		remoteCancel()
		n.unregisterPending(id, errors.New("node is closed"))
		close(events)
		close(errorsOut)
		return nil, errors.New("node is closed")
	}
	return value, nil
}

func (n *Node) runRemoteSubscription(ctx context.Context, id protocol.MessageID, resource protocol.ResourceID, pending *pendingEntry, events chan<- subscription.Event, errorsOut chan<- error) {
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
				n.sendUnsubscribe(id, resource)
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
			n.sendUnsubscribe(id, resource)
			return
		}
	}
}

func (n *Node) sendUnsubscribe(subscriptionID protocol.MessageID, resource protocol.ResourceID) {
	envelope, err := n.newEnvelope(protocol.PhaseRequest, protocol.OperationUnsubscribe, resource.Owner, resource, subscriptionID, 0, "application/json", "unsubscribe.v1", []byte("{}"))
	if err != nil {
		n.emit(err)
		return
	}
	if err := n.routeEnvelope(n.ctx, envelope, nil); err != nil && n.ctx.Err() == nil {
		n.emit(fmt.Errorf("unsubscribe: %w", err))
	}
}

func (n *Node) Invoke(ctx context.Context, resource protocol.ResourceID, input []byte) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("invoke context is required")
	}
	if err := resource.Validate(); err != nil {
		return nil, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		return nil, err
	}
	pending, err := n.registerPending(id, 1)
	if err != nil {
		return nil, err
	}
	defer n.unregisterPending(id, nil)
	request := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseRequest, Operation: protocol.OperationCommandCall,
		MessageID: id, Source: n.ID(), Target: resource.Owner, Resource: resource, DeadlineUnixMS: deadline.UnixMilli(),
		ContentType: "application/octet-stream", Schema: "command.raw.v1", Payload: append([]byte(nil), input...),
	}
	if sendErr := n.routeEnvelope(ctx, request, nil); sendErr != nil {
		select {
		case response := <-pending.frames:
			return commandResponse(response)
		default:
			return nil, sendErr
		}
	}
	response, err := awaitFrame(ctx, pending)
	if err != nil {
		return nil, err
	}
	return commandResponse(response)
}

func commandResponse(response protocol.Envelope) ([]byte, error) {
	switch response.Operation {
	case protocol.OperationCommandResult:
		return append([]byte(nil), response.Payload...), nil
	case protocol.OperationError:
		failure, err := protocol.DecodeErrorPayload(response.Payload)
		if err != nil {
			return nil, err
		}
		return nil, failure
	default:
		return nil, fmt.Errorf("unexpected command response operation %d", response.Operation)
	}
}
