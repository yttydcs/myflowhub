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
	case protocol.OperationOperate:
		return n.handleOperate(envelope)
	case protocol.OperationSessionOpen:
		return n.handleSessionOpen(envelope, inbound)
	case protocol.OperationSessionData:
		return n.handleSessionData(envelope, inbound)
	case protocol.OperationSessionClose:
		return n.handleSessionClose(envelope, inbound)
	case protocol.OperationError:
		return nil
	case protocol.OperationSubscribeAck, protocol.OperationResourceEvent, protocol.OperationResourceGap,
		protocol.OperationOperateResult, protocol.OperationSessionOpenResult:
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
	if payload.Version != protocol.SchemaVersionV2 || payload.LeaseMS <= 0 {
		err := errors.New("subscription version or lease is invalid")
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
		ID: envelope.MessageID, Subscriber: envelope.Subject(), Resource: envelope.Resource, Capability: envelope.Capability,
		LinkID: linkID, NextHop: nextHop, Lease: lease, TopologyEpoch: topologyEpoch,
		PolicyGeneration: auth.PolicyGeneration(n.policy), Queue: payload.Queue,
	})
	if err != nil {
		code := protocol.CodeConflict
		switch {
		case errors.Is(err, resource.ErrNotFound):
			code = protocol.CodeNotFound
		case errors.Is(err, subscription.ErrNotSubscribable):
			code = protocol.CodeUnsupported
		case errors.Is(err, subscription.ErrSubscriptionLimit):
			code = protocol.CodeOverflow
		}
		_ = n.sendError(envelope, code, err.Error())
		return err
	}
	ack, err := n.newEnvelope(
		protocol.PhaseResponse, protocol.OperationSubscribeAck, envelope.Source, envelope.Resource,
		envelope.Capability, envelope.MessageID, 0, "application/json", "mfh.subscribe-ack.v2", []byte("{}"),
	)
	if err != nil {
		current.Cancel()
		return err
	}
	if err := n.routeEnvelope(n.ctx, ack, nil); err != nil {
		current.Cancel()
		return err
	}
	if !n.launch(func() { n.forwardSubscription(envelope, current) }) {
		current.Cancel()
		return errors.New("node is closed")
	}
	return nil
}

func (n *Node) forwardSubscription(request protocol.Envelope, current *subscription.Subscription) {
	for event := range current.Events {
		operation := protocol.OperationResourceEvent
		payload := resourceEventPayload{
			Version: protocol.SchemaVersionV2, Snapshot: event.Kind == subscription.EventSnapshot,
			Revision: event.Revision, Sequence: event.Sequence, Publisher: event.Publisher,
			PublisherSequence: event.PublisherSequence, GapFrom: event.GapFrom, GapTo: event.GapTo,
			Reason: event.Reason, Schema: event.Schema, Value: event.Value,
		}
		if event.Kind == subscription.EventGap {
			operation = protocol.OperationResourceGap
		}
		if event.Kind == subscription.EventExpired {
			_ = n.sendError(request, protocol.CodeExpired, event.Reason)
			return
		}
		encoded, err := encodeJSON(payload)
		if err != nil {
			n.emit(err)
			current.Cancel()
			return
		}
		envelope, err := n.newEnvelope(
			protocol.PhaseEvent, operation, request.Source, request.Resource, request.Capability,
			request.MessageID, 0, "application/json", "mfh.resource-event.v2", encoded,
		)
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
	if !n.subscriptions.UnsubscribeFor(envelope.CorrelationID, envelope.Subject()) {
		err := errors.New("subscription not found for caller")
		_ = n.sendError(envelope, protocol.CodeNotFound, err.Error())
		return err
	}
	ack, err := n.newEnvelope(
		protocol.PhaseResponse, protocol.OperationSubscribeAck, envelope.Source, envelope.Resource,
		envelope.Capability, envelope.MessageID, 0, "application/json", "mfh.unsubscribe-ack.v2", []byte("{}"),
	)
	if err != nil {
		return err
	}
	return n.routeEnvelope(n.ctx, ack, nil)
}

func (n *Node) handleOperate(envelope protocol.Envelope) error {
	if envelope.DeadlineUnixMS == 0 {
		err := errors.New("resource operation deadline is required")
		_ = n.sendError(envelope, protocol.CodeMalformed, err.Error())
		return err
	}
	origin := command.OriginAdjudicated
	if envelope.Phase == protocol.PhaseControl || envelope.Subject() == n.ID() {
		origin = command.OriginParentControl
	}
	result, operationErr := n.commands.Invoke(n.ctx, command.Call{
		MessageID: envelope.MessageID, Source: envelope.Subject(), Resource: envelope.Resource,
		Capability: envelope.Capability, Schema: envelope.Schema, Input: envelope.Payload,
		Deadline: time.UnixMilli(envelope.DeadlineUnixMS), Origin: origin,
	})
	if result.Failure != nil {
		_ = n.sendFailure(envelope, *result.Failure)
		return operationErr
	}
	response, err := n.newEnvelope(
		protocol.PhaseResponse, protocol.OperationOperateResult, envelope.Source, envelope.Resource,
		envelope.Capability, envelope.MessageID, 0, envelope.ContentType, result.Schema, result.Output,
	)
	if err != nil {
		return err
	}
	return n.routeEnvelope(n.ctx, response, nil)
}
