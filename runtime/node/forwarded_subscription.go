package node

import (
	"errors"
	"fmt"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
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
	sourceLink string
	targetLink string
}

func (n *Node) trackForwardedSubscription(request protocol.Envelope, generation uint64) error {
	if generation == 0 {
		return errors.New("forwarded subscription policy generation is required")
	}
	sourceRoute, sourceErr := n.tree.RouteTo(request.Source)
	targetRoute, targetErr := n.tree.RouteTo(request.Target)
	if sourceErr != nil || targetErr != nil {
		return fmt.Errorf("track subscription routes: source=%v target=%v", sourceErr, targetErr)
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
	current := forwardedSubscription{
		id: request.MessageID, source: request.Source, principal: request.Principal, target: request.Target,
		resource: request.Resource, capability: request.Capability, generation: generation,
	}
	if session := n.sessions[sourceRoute.NextHop]; session != nil {
		current.sourceLink = session.linkID
	}
	if session := n.sessions[targetRoute.NextHop]; session != nil {
		current.targetLink = session.linkID
	}
	n.forwardedSubscriptions[request.MessageID] = current
	return nil
}

// Replacement and disconnect cleanup are owned by the exact old link, even
// when a new connection for the same NodeID is already installed.
func (n *Node) cleanupForwardedLink(linkID string) {
	n.mu.Lock()
	ended := make([]forwardedSubscription, 0)
	for id, current := range n.forwardedSubscriptions {
		if current.sourceLink == linkID || current.targetLink == linkID {
			ended = append(ended, current)
			delete(n.forwardedSubscriptions, id)
		}
	}
	n.mu.Unlock()
	if len(ended) == 0 {
		return
	}
	n.launch(func() {
		for _, current := range ended {
			if current.sourceLink != linkID {
				request := protocol.Envelope{MessageID: current.id, Source: current.source, Target: current.target, Resource: current.resource}
				if err := n.sendError(request, protocol.CodeExpired, "subscription link disconnected"); err != nil && n.ctx.Err() == nil {
					n.emit(fmt.Errorf("end disconnected subscription: %w", err))
				}
			}
			if current.targetLink != linkID {
				n.cancelForwardedSubscription(current)
			}
		}
	})
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
	n.mu.RLock()
	candidates := make([]forwardedSubscription, 0, len(n.forwardedSubscriptions))
	for _, current := range n.forwardedSubscriptions {
		candidates = append(candidates, current)
	}
	n.mu.RUnlock()
	expired := make([]forwardedSubscription, 0)
	for _, current := range candidates {
		if current.generation == generation {
			continue
		}
		if n.authorizeForwardedSubscription(current) == nil {
			continue
		}
		if n.untrackForwardedSubscription(current.id, current.source, current.target) {
			expired = append(expired, current)
		}
	}
	if len(expired) == 0 {
		return
	}
	n.launch(func() {
		for _, current := range expired {
			n.expireForwardedSubscription(current)
		}
	})
}

func (n *Node) authorizeForwardedSubscription(current forwardedSubscription) error {
	subject := current.principal
	if subject == 0 {
		subject = current.source
	}
	return n.policy.Authorize(n.ctx, auth.Request{Subject: subject, Action: auth.Action(current.capability), Capability: current.capability, Resource: current.resource})
}

func (n *Node) expireForwardedSubscription(current forwardedSubscription) {
	request := protocol.Envelope{
		MessageID: current.id, Source: current.source, Target: current.target, Resource: current.resource,
	}
	if err := n.sendError(request, protocol.CodeExpired, "authority policy generation changed"); err != nil && n.ctx.Err() == nil {
		n.emit(fmt.Errorf("expire forwarded subscription response: %w", err))
	}
	n.cancelForwardedSubscription(current)
}

func (n *Node) cancelForwardedSubscription(current forwardedSubscription) {
	messageID, err := protocol.NewMessageID()
	if err != nil {
		n.emit(err)
		return
	}
	unsubscribe := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseControl, Operation: protocol.OperationUnsubscribe,
		MessageID: messageID, CorrelationID: current.id, Source: n.ID(), Principal: current.principal,
		Target: current.target, Resource: current.resource, Capability: current.capability,
		ContentType: "application/json", Schema: "unsubscribe.v1", Payload: []byte("{}"),
	}
	if unsubscribe.Principal == 0 {
		unsubscribe.Principal = current.source
	}
	if err := n.routeEnvelope(n.ctx, unsubscribe, nil); err != nil && n.ctx.Err() == nil {
		n.emit(fmt.Errorf("cancel forwarded subscription: %w", err))
	}
}

// A sibling's disconnected link does not change the consumer's parent status.
// End its forwarded interests immediately instead of waiting for lease expiry.
func (n *Node) cleanupForwardedRoutes(removed []protocol.NodeID) {
	if len(removed) == 0 {
		return
	}
	nodes := make(map[protocol.NodeID]bool, len(removed))
	for _, id := range removed {
		nodes[id] = true
	}
	n.mu.Lock()
	ended := make([]forwardedSubscription, 0)
	for id, current := range n.forwardedSubscriptions {
		if nodes[current.source] || nodes[current.target] {
			ended = append(ended, current)
			delete(n.forwardedSubscriptions, id)
		}
	}
	n.mu.Unlock()
	n.launch(func() {
		for _, current := range ended {
			if !nodes[current.source] {
				request := protocol.Envelope{MessageID: current.id, Source: current.source, Target: current.target, Resource: current.resource}
				if err := n.sendError(request, protocol.CodeExpired, "subscription route disconnected"); err != nil && n.ctx.Err() == nil {
					n.emit(fmt.Errorf("end disconnected subscription: %w", err))
				}
			}
			if !nodes[current.target] {
				n.cancelForwardedSubscription(current)
			}
		}
	})
}
