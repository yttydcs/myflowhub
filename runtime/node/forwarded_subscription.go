package node

import (
	"errors"
	"fmt"

	"github.com/yttydcs/myflowhub/protocol"
)

var ErrForwardedSubscriptionLimit = errors.New("forwarded subscription limit reached")

type forwardedSubscription struct {
	id         protocol.MessageID
	source     protocol.NodeID
	principal  protocol.NodeID
	target     protocol.NodeID
	resource   protocol.ResourceID
	capability protocol.CapabilityID
	generation uint64
}

func (n *Node) trackForwardedSubscription(request protocol.Envelope, generation uint64) error {
	if generation == 0 {
		return errors.New("forwarded subscription policy generation is required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return errors.New("node is closed")
	}
	if len(n.forwardedSubscriptions) >= n.config.MaxPending {
		return ErrForwardedSubscriptionLimit
	}
	if _, exists := n.forwardedSubscriptions[request.MessageID]; exists {
		return errors.New("forwarded subscription ID already exists")
	}
	n.forwardedSubscriptions[request.MessageID] = forwardedSubscription{
		id: request.MessageID, source: request.Source, principal: request.Principal, target: request.Target,
		resource: request.Resource, capability: request.Capability, generation: generation,
	}
	return nil
}

func (n *Node) untrackForwardedSubscription(id protocol.MessageID, source, target protocol.NodeID) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	current, exists := n.forwardedSubscriptions[id]
	if !exists || current.source != source || current.target != target {
		return false
	}
	delete(n.forwardedSubscriptions, id)
	return true
}

func (n *Node) expireForwardedSubscriptions(generation uint64) {
	n.mu.Lock()
	expired := make([]forwardedSubscription, 0, len(n.forwardedSubscriptions))
	for id, current := range n.forwardedSubscriptions {
		if current.generation != generation {
			expired = append(expired, current)
			delete(n.forwardedSubscriptions, id)
		}
	}
	n.mu.Unlock()
	if len(expired) == 0 {
		return
	}
	n.launch(func() {
		for _, current := range expired {
			n.expireForwardedSubscription(current)
		}
	})
}

func (n *Node) expireForwardedSubscription(current forwardedSubscription) {
	request := protocol.Envelope{
		MessageID: current.id, Source: current.source, Target: current.target, Resource: current.resource,
	}
	if err := n.sendError(request, protocol.CodeExpired, "authority policy generation changed"); err != nil && n.ctx.Err() == nil {
		n.emit(fmt.Errorf("expire forwarded subscription response: %w", err))
	}
	messageID, err := protocol.NewMessageID()
	if err != nil {
		n.emit(err)
		return
	}
	unsubscribe := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseControl, Operation: protocol.OperationUnsubscribe,
		MessageID: messageID, CorrelationID: current.id, Source: current.source, Principal: current.principal,
		Target: current.target, Resource: current.resource, Capability: current.capability,
		ContentType: "application/json", Schema: "unsubscribe.v1", Payload: []byte("{}"),
	}
	if err := n.routeEnvelope(n.ctx, unsubscribe, nil); err != nil && n.ctx.Err() == nil {
		n.emit(fmt.Errorf("cancel forwarded subscription: %w", err))
	}
}
