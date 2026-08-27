package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

func (n *Node) handleInbound(inbound *peerSession, envelope protocol.Envelope) error {
	switch envelope.Operation {
	case protocol.OperationHeartbeat:
		return nil
	case protocol.OperationRouteAnnounce:
		if inbound.role != link.RoleChild || envelope.Target != n.ID() {
			return errors.New("route announcement must arrive from a direct child")
		}
		var payload routePayload
		if err := decodeJSON(envelope.Payload, &payload); err != nil {
			return err
		}
		if err := n.tree.Announce(inbound.peer, payload.Node, inbound.epoch); err != nil {
			return err
		}
		n.announceUp(payload.Node)
		return nil
	case protocol.OperationRouteWithdraw:
		if inbound.role != link.RoleChild || envelope.Target != n.ID() {
			return errors.New("route withdrawal must arrive from a direct child")
		}
		var payload routePayload
		if err := decodeJSON(envelope.Payload, &payload); err != nil {
			return err
		}
		if err := n.tree.WithdrawRoute(inbound.peer, payload.Node, inbound.epoch); err != nil {
			return err
		}
		n.withdrawUp(payload.Node)
		return nil
	default:
		return n.routeEnvelope(n.ctx, envelope, inbound)
	}
}

func (n *Node) routeEnvelope(ctx context.Context, envelope protocol.Envelope, inbound *peerSession) error {
	if envelope.Target == n.ID() {
		return n.handleLocal(envelope, inbound)
	}
	route, err := n.tree.RouteTo(envelope.Target)
	if err != nil {
		if envelope.Phase == protocol.PhaseRequest {
			_ = n.sendError(envelope, protocol.CodeNotFound, err.Error())
		}
		return err
	}
	if route.Kind == tree.DirectionLocal {
		return n.handleLocal(envelope, inbound)
	}
	if inbound != nil && route.NextHop == inbound.peer {
		return errors.New("routing loop detected")
	}
	if route.Kind == tree.DirectionDown {
		switch envelope.Phase {
		case protocol.PhaseRequest:
			if envelope.Source != n.ID() {
				if err := auth.AuthorizeRequest(ctx, n.policy, envelope); err != nil {
					_ = n.sendError(envelope, protocol.CodeForbidden, err.Error())
					return err
				}
			}
			envelope, err = auth.PromoteControl(envelope, route.Epoch)
			if err != nil {
				return err
			}
		case protocol.PhaseControl:
			envelope, err = auth.PromoteControl(envelope, route.Epoch)
			if err != nil {
				return err
			}
		}
	} else if envelope.Phase == protocol.PhaseControl {
		return errors.New("control frame cannot be routed toward parent")
	}
	n.mu.RLock()
	next := n.sessions[route.NextHop]
	n.mu.RUnlock()
	if next == nil || next.session.State() != link.StateActive {
		return fmt.Errorf("next hop %d has no active session", route.NextHop)
	}
	return next.session.Send(ctx, envelope)
}

func (n *Node) announceUp(descendant protocol.NodeID) {
	n.sendRouteEvent(protocol.OperationRouteAnnounce, descendant)
}

func (n *Node) withdrawUp(descendant protocol.NodeID) {
	n.sendRouteEvent(protocol.OperationRouteWithdraw, descendant)
}

func (n *Node) sendRouteEvent(operation protocol.Operation, descendant protocol.NodeID) {
	parent, ok := n.tree.Parent()
	if !ok {
		return
	}
	payload, err := encodeJSON(routePayload{Node: descendant})
	if err != nil {
		n.emit(err)
		return
	}
	messageID, err := protocol.NewMessageID()
	if err != nil {
		n.emit(err)
		return
	}
	envelope := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseEvent, Operation: operation,
		MessageID: messageID, Source: n.ID(), Target: parent.Node, TopologyEpoch: parent.Epoch,
		ContentType: "application/json", Schema: "route.v1", Payload: payload,
	}
	n.mu.RLock()
	session := n.sessions[parent.Node]
	n.mu.RUnlock()
	if session == nil {
		n.emit(errors.New("cannot announce route without active parent session"))
		return
	}
	if err := session.session.Send(n.ctx, envelope); err != nil && n.ctx.Err() == nil {
		n.emit(fmt.Errorf("send route event: %w", err))
	}
}

func (n *Node) sendError(request protocol.Envelope, code protocol.ErrorCode, message string) error {
	payload, err := protocol.EncodeErrorPayload(protocol.ErrorPayload{Code: code, Message: message})
	if err != nil {
		return err
	}
	messageID, err := protocol.NewMessageID()
	if err != nil {
		return err
	}
	response := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseResponse, Operation: protocol.OperationError,
		MessageID: messageID, CorrelationID: request.MessageID, Source: n.ID(), Target: request.Source,
		ContentType: "application/json", Schema: "error.v1", Payload: payload,
	}
	return n.routeEnvelope(n.ctx, response, nil)
}

func (n *Node) newEnvelope(phase protocol.Phase, operation protocol.Operation, target protocol.NodeID, resource protocol.ResourceID, correlation protocol.MessageID, deadline int64, contentType, schema string, payload []byte) (protocol.Envelope, error) {
	messageID, err := protocol.NewMessageID()
	if err != nil {
		return protocol.Envelope{}, err
	}
	return protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: phase, Operation: operation, MessageID: messageID, CorrelationID: correlation,
		Source: n.ID(), Target: target, Resource: resource, DeadlineUnixMS: deadline, ContentType: contentType, Schema: schema, Payload: payload,
	}, nil
}
