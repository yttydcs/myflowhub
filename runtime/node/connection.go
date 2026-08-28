package node

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

func (n *Node) ConnectParent(ctx context.Context, driver link.Driver, endpoint link.Endpoint, expectedParent protocol.NodeID) error {
	if ctx == nil {
		return errors.New("connect context is required")
	}
	if err := link.ValidateDriver(driver); err != nil {
		return err
	}
	if err := expectedParent.Validate(); err != nil {
		return err
	}
	pipe, err := driver.Dial(ctx, endpoint)
	if err != nil {
		return err
	}
	session, err := link.NewSession(n.ctx, pipe, n.config.Session)
	if err != nil {
		_ = pipe.Close()
		return err
	}
	success := false
	defer func() {
		if !success {
			_ = session.Close()
		}
	}()
	var claim auth.JoinClaim
	if n.config.JoinPermit == nil {
		claim, err = auth.NewJoinClaim(n.identity, n.tree.NextEpoch())
	} else {
		claim, err = auth.NewJoinClaimWithPermit(n.identity, n.tree.NextEpoch(), *n.config.JoinPermit)
	}
	if err != nil {
		return err
	}
	payload, err := json.Marshal(claim)
	if err != nil {
		return err
	}
	messageID, err := protocol.NewMessageID()
	if err != nil {
		return err
	}
	join := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseRequest, Operation: protocol.OperationJoin,
		MessageID: messageID, Source: n.ID(), Target: expectedParent, ContentType: "application/json", Schema: "join.v1", Payload: payload,
	}
	if err := session.Send(ctx, join); err != nil {
		return fmt.Errorf("send join: %w", err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, n.config.JoinTimeout)
	defer cancel()
	var response protocol.Envelope
	select {
	case value, ok := <-session.Inbound():
		if !ok {
			return fmt.Errorf("join session closed: %w", session.Err())
		}
		response = value
	case <-waitCtx.Done():
		return fmt.Errorf("wait for join acknowledgement: %w", waitCtx.Err())
	case <-n.ctx.Done():
		return errors.New("node closed while joining parent")
	}
	if response.Operation != protocol.OperationJoinAck || response.CorrelationID != messageID || response.Source != expectedParent || response.Target != n.ID() {
		return errors.New("join acknowledgement does not match request")
	}
	var ack auth.JoinAck
	if err := decodeJSON(response.Payload, &ack); err != nil {
		return err
	}
	if err := n.trust.VerifyJoinAck(ack, expectedParent, n.ID(), claim.Nonce); err != nil {
		return fmt.Errorf("verify join acknowledgement: %w", err)
	}
	epoch := ack.TopologyEpoch
	treeChanged := false
	if epoch != claim.TopologyEpoch {
		return errors.New("join topology epoch mismatch")
	}
	if err := n.tree.ActivateParent(expectedParent, epoch); err != nil {
		return fmt.Errorf("activate parent tree edge: %w", err)
	}
	treeChanged = true
	defer func() {
		if !success && treeChanged {
			n.tree.DetachParentEpoch(expectedParent, epoch)
			n.closeSessions(link.RoleParent, 0)
			n.failPending(errors.New("parent join failed after topology changed"))
		}
	}()
	if err := session.Activate(expectedParent, link.RoleParent, epoch); err != nil {
		return err
	}
	current := &peerSession{peer: expectedParent, role: link.RoleParent, epoch: epoch, linkID: fmt.Sprintf("parent:%d:%d", expectedParent, epoch), session: session}
	if err := n.registerSession(current); err != nil {
		return err
	}
	for _, relation := range n.tree.Relations() {
		n.announceUp(relation.Node, relation.Parent)
	}
	success = true
	return nil
}

func (n *Node) acceptChild(pipe link.Pipe) error {
	session, err := link.NewSession(n.ctx, pipe, n.config.Session)
	if err != nil {
		_ = pipe.Close()
		return err
	}
	success := false
	defer func() {
		if !success {
			_ = session.Close()
		}
	}()
	waitCtx, cancel := context.WithTimeout(n.ctx, n.config.JoinTimeout)
	defer cancel()
	var request protocol.Envelope
	select {
	case value, ok := <-session.Inbound():
		if !ok {
			return fmt.Errorf("child join session closed: %w", session.Err())
		}
		request = value
	case <-waitCtx.Done():
		return fmt.Errorf("wait for child join: %w", waitCtx.Err())
	}
	if request.Operation != protocol.OperationJoin || request.Phase != protocol.PhaseRequest || request.Target != n.ID() {
		return errors.New("first child frame must be a join request for this node")
	}
	var claim auth.JoinClaim
	if err := decodeJSON(request.Payload, &claim); err != nil {
		return err
	}
	if claim.NodeID != request.Source {
		return errors.New("join claim node does not match envelope source")
	}
	if err := n.trust.VerifyJoin(claim); err != nil {
		if !errors.Is(err, auth.ErrUntrustedIdentity) || n.admission == nil || claim.Permit == nil {
			return fmt.Errorf("verify child join: %w", err)
		}
		if err := auth.VerifyJoinClaim(claim); err != nil {
			return fmt.Errorf("verify untrusted child claim: %w", err)
		}
		if err := n.admission.Consume(claim.NodeID, ed25519.PublicKey(claim.PublicKey), *claim.Permit); err != nil {
			return fmt.Errorf("admit child: %w", err)
		}
		if err := n.trust.Add(claim.NodeID, ed25519.PublicKey(claim.PublicKey)); err != nil {
			return fmt.Errorf("persist admitted child trust: %w", err)
		}
	}
	reattached := false
	if err := n.tree.AttachChild(claim.NodeID, claim.TopologyEpoch); err != nil {
		if !errors.Is(err, tree.ErrChildExists) {
			return fmt.Errorf("attach child: %w", err)
		}
		if err := n.tree.ReattachChild(claim.NodeID, claim.TopologyEpoch); err != nil {
			return fmt.Errorf("reattach child: %w", err)
		}
		reattached = true
	}
	attached := true
	defer func() {
		if !success && attached {
			removed := n.tree.WithdrawChildEpoch(claim.NodeID, claim.TopologyEpoch)
			if reattached {
				n.closeSessions(link.RoleChild, claim.NodeID)
				for _, descendant := range removed {
					n.withdrawUp(descendant)
				}
			}
		}
	}()
	ack, err := auth.SignJoinAck(n.identity, claim.NodeID, claim.Nonce, claim.TopologyEpoch)
	if err != nil {
		return err
	}
	payload, err := encodeJSON(ack)
	if err != nil {
		return err
	}
	ackID, err := protocol.NewMessageID()
	if err != nil {
		return err
	}
	response := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseResponse, Operation: protocol.OperationJoinAck,
		MessageID: ackID, CorrelationID: request.MessageID, Source: n.ID(), Target: claim.NodeID,
		ContentType: "application/json", Schema: "join-ack.v1", Payload: payload,
	}
	if err := session.Send(waitCtx, response); err != nil {
		return fmt.Errorf("send join acknowledgement: %w", err)
	}
	if err := session.Activate(claim.NodeID, link.RoleChild, claim.TopologyEpoch); err != nil {
		return err
	}
	current := &peerSession{peer: claim.NodeID, role: link.RoleChild, epoch: claim.TopologyEpoch, linkID: fmt.Sprintf("child:%d:%d", claim.NodeID, claim.TopologyEpoch), session: session}
	if err := n.registerSession(current); err != nil {
		return err
	}
	n.announceUp(claim.NodeID, n.ID())
	attached = false
	success = true
	return nil
}

func (n *Node) registerSession(current *peerSession) error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return errors.New("node is closed")
	}
	old := n.sessions[current.peer]
	topologyChanged := false
	if current.role == link.RoleParent {
		for peer, session := range n.sessions {
			if session.role == link.RoleParent && session != old {
				delete(n.sessions, peer)
				go session.session.Close()
				topologyChanged = true
			}
		}
		if old != nil {
			topologyChanged = true
		}
	}
	n.sessions[current.peer] = current
	n.wg.Add(1)
	n.mu.Unlock()
	if current.role == link.RoleParent {
		n.signalParentChange()
	}
	if topologyChanged {
		n.failPending(errors.New("topology changed during reparent"))
	}
	go n.runSession(current)
	if old != nil && old != current {
		go old.session.Close()
	}
	return nil
}

func (n *Node) runSession(current *peerSession) {
	defer n.wg.Done()
	for envelope := range current.session.Inbound() {
		var err error
		if current.role == link.RoleChild {
			err = auth.ValidateInboundChild(n.tree, current.peer, current.epoch, envelope)
		} else {
			err = auth.ValidateInboundParent(n.tree, current.peer, envelope)
		}
		if err != nil {
			n.emit(fmt.Errorf("peer %d: %w", current.peer, err))
			_ = current.session.Close()
			break
		}
		if err := n.handleInbound(current, envelope); err != nil {
			n.emit(fmt.Errorf("peer %d: %w", current.peer, err))
		}
	}
	n.subscriptions.CleanupLink(current.linkID)
	n.cleanupResourceSessionsLink(current.linkID, errors.New("resource session link disconnected"))
	n.mu.Lock()
	isCurrent := n.sessions[current.peer] == current
	if isCurrent {
		delete(n.sessions, current.peer)
	}
	n.mu.Unlock()
	if !isCurrent {
		return
	}
	if current.role == link.RoleChild {
		removed := n.tree.WithdrawChildEpoch(current.peer, current.epoch)
		for _, descendant := range removed {
			n.withdrawUp(descendant)
		}
	} else {
		n.tree.DetachParentEpoch(current.peer, current.epoch)
		n.failPending(errors.New("parent link disconnected"))
		n.signalParentChange()
	}
}

func (n *Node) closeSessions(role link.Role, peer protocol.NodeID) {
	n.mu.RLock()
	values := make([]*link.Session, 0, len(n.sessions))
	for id, current := range n.sessions {
		if current.role == role && (peer == 0 || id == peer) {
			values = append(values, current.session)
		}
	}
	n.mu.RUnlock()
	for _, current := range values {
		_ = current.Close()
	}
}

func (n *Node) failPending(err error) {
	n.mu.Lock()
	values := make([]*pendingEntry, 0, len(n.pending))
	for id, value := range n.pending {
		values = append(values, value)
		delete(n.pending, id)
	}
	n.mu.Unlock()
	for _, value := range values {
		value.fail(err)
	}
}
