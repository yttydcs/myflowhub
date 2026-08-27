package node

import (
	"errors"
	"fmt"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

func (n *Node) handleLocal(envelope protocol.Envelope, inbound *peerSession) error {
	if (envelope.Phase == protocol.PhaseResponse || envelope.Phase == protocol.PhaseEvent) && n.deliverPending(envelope) {
		return nil
	}
	switch envelope.Operation {
	case protocol.OperationSubscribe:
		return n.handleSubscribe(envelope, inbound)
	case protocol.OperationUnsubscribe:
		return n.handleUnsubscribe(envelope)
	case protocol.OperationCommandCall:
		return n.handleCommand(envelope)
	case protocol.OperationError:
		return nil
	case protocol.OperationSubscribeAck, protocol.OperationVariableSnapshot, protocol.OperationVariableUpdate,
		protocol.OperationStreamEvent, protocol.OperationStreamGap, protocol.OperationCommandResult:
		return nil
	default:
		return fmt.Errorf("operation %d has no local handler", envelope.Operation)
	}
}

func (n *Node) authorizeLocal(envelope protocol.Envelope) error {
	if envelope.Phase == protocol.PhaseControl || envelope.Subject() == n.ID() {
		return nil
	}
	return auth.AuthorizeRequest(n.ctx, n.policy, envelope)
}

func (n *Node) handleSubscribe(envelope protocol.Envelope, inbound *peerSession) error {
	if err := n.authorizeLocal(envelope); err != nil {
		_ = n.sendError(envelope, protocol.CodeForbidden, err.Error())
		return err
	}
	var payload subscribePayload
	if err := decodeJSON(envelope.Payload, &payload); err != nil {
		_ = n.sendError(envelope, protocol.CodeMalformed, err.Error())
		return err
	}
	if payload.LeaseMS <= 0 {
		err := errors.New("subscription lease must be positive")
		_ = n.sendError(envelope, protocol.CodeMalformed, err.Error())
		return err
	}
	lease := time.Duration(payload.LeaseMS) * time.Millisecond
	linkID := "local"
	nextHop := n.ID()
	topologyEpoch := n.tree.Epoch()
	if inbound != nil {
		linkID = inbound.linkID
		nextHop = inbound.peer
		topologyEpoch = inbound.epoch
	}
	current, err := n.subscriptions.Subscribe(subscription.Request{
		ID: envelope.MessageID, Subscriber: envelope.Source, Resource: envelope.Resource,
		LinkID: linkID, NextHop: nextHop, Lease: lease,
		TopologyEpoch: topologyEpoch, PolicyGeneration: auth.PolicyGeneration(n.policy), Queue: payload.Queue,
	})
	if err != nil {
		code := protocol.CodeConflict
		if errors.Is(err, resource.ErrNotFound) {
			code = protocol.CodeNotFound
		}
		_ = n.sendError(envelope, code, err.Error())
		return err
	}
	value, _ := n.registry.Resolve(envelope.Resource)
	if _, ok := value.(*resource.Stream); ok {
		ack, err := n.newEnvelope(protocol.PhaseResponse, protocol.OperationSubscribeAck, envelope.Source, envelope.Resource, envelope.MessageID, 0, "application/json", "subscribe-ack.v1", []byte("{}"))
		if err != nil {
			current.Cancel()
			return err
		}
		if err := n.routeEnvelope(n.ctx, ack, nil); err != nil {
			current.Cancel()
			return err
		}
	}
	if !n.launch(func() {
		n.forwardSubscription(envelope, current)
	}) {
		current.Cancel()
		return errors.New("node is closed")
	}
	return nil
}

func (n *Node) forwardSubscription(request protocol.Envelope, current *subscription.Subscription) {
	for event := range current.Events {
		var operation protocol.Operation
		var phase protocol.Phase
		var payload []byte
		var err error
		switch event.Kind {
		case subscription.EventVariableSnapshot:
			operation, phase = protocol.OperationVariableSnapshot, protocol.PhaseResponse
			payload, err = encodeJSON(variablePayload{Revision: event.Revision, Value: event.Value})
		case subscription.EventVariableUpdate:
			operation, phase = protocol.OperationVariableUpdate, protocol.PhaseEvent
			payload, err = encodeJSON(variablePayload{Revision: event.Revision, Value: event.Value})
		case subscription.EventStream:
			operation, phase = protocol.OperationStreamEvent, protocol.PhaseEvent
			payload, err = encodeJSON(streamPayload{Sequence: event.Sequence, Value: event.Value})
		case subscription.EventStreamGap:
			operation, phase = protocol.OperationStreamGap, protocol.PhaseEvent
			payload, err = encodeJSON(streamPayload{GapFrom: event.GapFrom, GapTo: event.GapTo})
		case subscription.EventExpired:
			_ = n.sendError(request, protocol.CodeExpired, event.Reason)
			return
		default:
			continue
		}
		if err != nil {
			n.emit(err)
			current.Cancel()
			return
		}
		envelope, err := n.newEnvelope(phase, operation, request.Source, request.Resource, request.MessageID, 0, "application/json", "subscription-event.v1", payload)
		if err == nil {
			err = n.routeEnvelope(n.ctx, envelope, nil)
		}
		if err != nil {
			if errors.Is(err, link.ErrQueueFull) {
				_ = n.sendError(request, protocol.CodeOverflow, err.Error())
			}
			n.emit(fmt.Errorf("forward subscription: %w", err))
			current.Cancel()
			return
		}
	}
}

func (n *Node) handleUnsubscribe(envelope protocol.Envelope) error {
	if envelope.CorrelationID.IsZero() {
		err := errors.New("unsubscribe requires the subscription correlation ID")
		_ = n.sendError(envelope, protocol.CodeMalformed, err.Error())
		return err
	}
	if err := n.authorizeLocal(envelope); err != nil {
		_ = n.sendError(envelope, protocol.CodeForbidden, err.Error())
		return err
	}
	if !n.subscriptions.UnsubscribeFor(envelope.CorrelationID, envelope.Source) {
		err := errors.New("subscription not found for caller")
		_ = n.sendError(envelope, protocol.CodeNotFound, err.Error())
		return err
	}
	ack, err := n.newEnvelope(protocol.PhaseResponse, protocol.OperationSubscribeAck, envelope.Source, envelope.Resource, envelope.MessageID, 0, "application/json", "unsubscribe-ack.v1", []byte("{}"))
	if err != nil {
		return err
	}
	return n.routeEnvelope(n.ctx, ack, nil)
}

func (n *Node) handleCommand(envelope protocol.Envelope) error {
	deadline := time.UnixMilli(envelope.DeadlineUnixMS)
	if envelope.DeadlineUnixMS == 0 {
		err := errors.New("command deadline is required")
		_ = n.sendError(envelope, protocol.CodeMalformed, err.Error())
		return err
	}
	origin := command.OriginAdjudicated
	if envelope.Phase == protocol.PhaseControl || envelope.Subject() == n.ID() {
		origin = command.OriginParentControl
	}
	result, invokeErr := n.commands.Invoke(n.ctx, command.Call{
		MessageID: envelope.MessageID, Source: envelope.Subject(), Resource: envelope.Resource,
		Input: envelope.Payload, Deadline: deadline, Origin: origin,
	})
	if result.Failure != nil {
		_ = n.sendFailure(envelope, *result.Failure)
		return invokeErr
	}
	response, err := n.newEnvelope(protocol.PhaseResponse, protocol.OperationCommandResult, envelope.Source, envelope.Resource, envelope.MessageID, 0, envelope.ContentType, envelope.Schema, result.Output)
	if err != nil {
		return err
	}
	return n.routeEnvelope(n.ctx, response, nil)
}
