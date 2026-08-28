package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

type ownedResourceSession struct {
	id               protocol.MessageID
	subject          protocol.NodeID
	resource         protocol.ResourceID
	capability       protocol.CapabilityID
	linkID           string
	topologyEpoch    uint64
	policyGeneration uint64
	expiresAt        time.Time
	session          resource.ActiveSession
}

type RemoteSession struct {
	node       *Node
	id         protocol.MessageID
	resource   protocol.ResourceID
	capability protocol.CapabilityID
	grant      protocol.SessionGrantV2

	mu          sync.Mutex
	closed      bool
	closeResult resource.OperationResult
	closeErr    error
}

func (s *RemoteSession) Grant() protocol.SessionGrantV2 { return s.grant }

func (n *Node) OpenSession(ctx context.Context, resourceID protocol.ResourceID, capability protocol.CapabilityID, schema string, payload []byte) (*RemoteSession, error) {
	if ctx == nil {
		return nil, errors.New("open session context is required")
	}
	if err := resourceID.Validate(); err != nil {
		return nil, err
	}
	if err := capability.Validate(); err != nil {
		return nil, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	messageID, err := protocol.NewMessageID()
	if err != nil {
		return nil, err
	}
	pending, err := n.registerPending(messageID, 1)
	if err != nil {
		return nil, err
	}
	defer n.unregisterPending(messageID, nil)
	request := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseRequest, Operation: protocol.OperationSessionOpen,
		MessageID: messageID, Source: n.ID(), Target: resourceID.Owner, Resource: resourceID, Capability: capability,
		DeadlineUnixMS: deadline.UnixMilli(), ContentType: "application/json", Schema: schema, Payload: append([]byte(nil), payload...),
	}
	if err := n.routeEnvelope(ctx, request, nil); err != nil {
		return nil, err
	}
	response, err := awaitFrame(ctx, pending)
	if err != nil {
		return nil, err
	}
	if response.Operation == protocol.OperationError {
		failure, decodeErr := protocol.DecodeErrorPayload(response.Payload)
		if decodeErr != nil {
			return nil, decodeErr
		}
		return nil, failure
	}
	if response.Operation != protocol.OperationSessionOpenResult {
		return nil, fmt.Errorf("unexpected session open response %d", response.Operation)
	}
	var grant protocol.SessionGrantV2
	if err := protocol.DecodeJSONPayload(response.Payload, protocol.DefaultMaxPayload, &grant); err != nil {
		return nil, err
	}
	sessionID, err := protocol.ParseMessageID(grant.SessionID)
	if err != nil {
		return nil, err
	}
	return &RemoteSession{node: n, id: sessionID, resource: resourceID, capability: capability, grant: grant}, nil
}

func (s *RemoteSession) Send(ctx context.Context, offset int64, data []byte, checksum string) (resource.OperationResult, error) {
	if s == nil || s.node == nil {
		return resource.OperationResult{}, errors.New("resource session is unavailable")
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return resource.OperationResult{}, errors.New("resource session is closed")
	}
	if len(data) > s.grant.MaxChunkBytes {
		return resource.OperationResult{}, fmt.Errorf("session chunk exceeds grant: got %d, max %d", len(data), s.grant.MaxChunkBytes)
	}
	payload, err := protocol.EncodeJSONPayload(&protocol.SessionDataV2{
		Version: protocol.SchemaVersionV2, Offset: offset, Checksum: checksum, Data: append([]byte(nil), data...),
	}, protocol.DefaultMaxPayload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	return s.exchange(ctx, protocol.OperationSessionData, protocol.SchemaSessionDataV2, payload)
}

func (s *RemoteSession) Close(ctx context.Context, commit bool, schema string, payload []byte) (resource.OperationResult, error) {
	if s == nil || s.node == nil {
		return resource.OperationResult{}, errors.New("resource session is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.closeResult, s.closeErr
	}
	encoded, err := protocol.EncodeJSONPayload(&protocol.SessionCloseV2{
		Version: protocol.SchemaVersionV2, Commit: commit, Schema: schema, Payload: append([]byte(nil), payload...),
	}, protocol.DefaultMaxPayload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	result, err := s.exchange(ctx, protocol.OperationSessionClose, protocol.SchemaSessionCloseV2, encoded)
	s.closed = true
	s.closeResult = result
	s.closeErr = err
	return result, err
}

func (s *RemoteSession) exchange(ctx context.Context, operation protocol.Operation, schema string, payload []byte) (resource.OperationResult, error) {
	if ctx == nil {
		return resource.OperationResult{}, errors.New("resource session context is required")
	}
	messageID, err := protocol.NewMessageID()
	if err != nil {
		return resource.OperationResult{}, err
	}
	pending, err := s.node.registerPending(messageID, 1)
	if err != nil {
		return resource.OperationResult{}, err
	}
	defer s.node.unregisterPending(messageID, nil)
	request := protocol.Envelope{
		Version: protocol.CurrentVersion, Phase: protocol.PhaseRequest, Operation: operation,
		MessageID: messageID, CorrelationID: s.id, Source: s.node.ID(), Target: s.resource.Owner,
		Resource: s.resource, Capability: s.capability, ContentType: "application/json", Schema: schema,
		Payload: append([]byte(nil), payload...),
	}
	if err := s.node.routeEnvelope(ctx, request, nil); err != nil {
		return resource.OperationResult{}, err
	}
	response, err := awaitFrame(ctx, pending)
	if err != nil {
		return resource.OperationResult{}, err
	}
	if response.Operation == protocol.OperationError {
		failure, decodeErr := protocol.DecodeErrorPayload(response.Payload)
		if decodeErr != nil {
			return resource.OperationResult{}, decodeErr
		}
		return resource.OperationResult{}, failure
	}
	if response.Operation != operation {
		return resource.OperationResult{}, fmt.Errorf("unexpected session response %d", response.Operation)
	}
	return resource.OperationResult{Schema: response.Schema, Payload: append([]byte(nil), response.Payload...)}, nil
}

func (n *Node) handleSessionOpen(envelope protocol.Envelope, inbound *peerSession) error {
	if err := n.authorizeLocal(envelope); err != nil {
		_ = n.sendError(envelope, protocol.CodeForbidden, err.Error())
		return err
	}
	session, err := n.registry.OpenSession(n.ctx, envelope.Resource, resource.SessionOpenRequest{
		Subject: envelope.Subject(), Capability: envelope.Capability, Schema: envelope.Schema, Payload: envelope.Payload,
	})
	if err != nil {
		code := protocol.CodeConflict
		switch {
		case errors.Is(err, resource.ErrNotFound):
			code = protocol.CodeNotFound
		case errors.Is(err, resource.ErrUnsupportedCapability):
			code = protocol.CodeUnsupported
		case errors.Is(err, resource.ErrInvalidSchema), errors.Is(err, resource.ErrValueTooLarge):
			code = protocol.CodeMalformed
		}
		_ = n.sendError(envelope, code, err.Error())
		return err
	}
	grant := session.Grant()
	if grant.MaxChunkBytes <= 0 || grant.MaxChunkBytes > protocol.MaxSessionChunkBytes || grant.MaxTotalBytes < 0 || grant.ExpiresAtUnixMS <= time.Now().UnixMilli() {
		session.Abort(errors.New("resource returned an invalid session grant"))
		err := errors.New("resource returned an invalid session grant")
		_ = n.sendError(envelope, protocol.CodeInternal, err.Error())
		return err
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		session.Abort(err)
		return err
	}
	linkID := "local"
	topologyEpoch := n.tree.Epoch()
	if inbound != nil {
		linkID = inbound.linkID
		topologyEpoch = inbound.epoch
	}
	owned := &ownedResourceSession{
		id: id, subject: envelope.Subject(), resource: envelope.Resource, capability: envelope.Capability,
		linkID: linkID, topologyEpoch: topologyEpoch, policyGeneration: auth.PolicyGeneration(n.policy),
		expiresAt: time.UnixMilli(grant.ExpiresAtUnixMS), session: session,
	}
	n.mu.Lock()
	if n.closed || len(n.resourceSessions) >= n.config.MaxResourceSessions {
		n.mu.Unlock()
		session.Abort(errors.New("resource session limit reached"))
		err := errors.New("resource session limit reached")
		_ = n.sendError(envelope, protocol.CodeOverflow, err.Error())
		return err
	}
	n.resourceSessions[id] = owned
	n.mu.Unlock()
	responsePayload, err := protocol.EncodeJSONPayload(&protocol.SessionGrantV2{
		Version: protocol.SchemaVersionV2, SessionID: id.String(), Capability: envelope.Capability,
		MaxChunkBytes: grant.MaxChunkBytes, MaxTotalBytes: grant.MaxTotalBytes, ExpiresAtUnixMS: grant.ExpiresAtUnixMS,
	}, protocol.DefaultMaxPayload)
	if err != nil {
		n.removeResourceSession(id, err)
		return err
	}
	response, err := n.newEnvelope(
		protocol.PhaseResponse, protocol.OperationSessionOpenResult, envelope.Source, envelope.Resource,
		envelope.Capability, envelope.MessageID, 0, "application/json", protocol.SchemaSessionGrantV2, responsePayload,
	)
	if err != nil {
		n.removeResourceSession(id, err)
		return err
	}
	if err := n.routeEnvelope(n.ctx, response, nil); err != nil {
		n.removeResourceSession(id, err)
		return err
	}
	n.launch(func() {
		timer := time.NewTimer(time.Until(owned.expiresAt))
		defer timer.Stop()
		select {
		case <-timer.C:
			n.removeResourceSession(id, errors.New("resource session expired"))
		case <-n.ctx.Done():
		}
	})
	return nil
}

func (n *Node) handleSessionData(envelope protocol.Envelope, inbound *peerSession) error {
	owned, err := n.validateResourceSession(envelope, inbound)
	if err != nil {
		_ = n.sendError(envelope, protocol.CodeExpired, err.Error())
		return err
	}
	var payload protocol.SessionDataV2
	if err := protocol.DecodeJSONPayload(envelope.Payload, protocol.DefaultMaxPayload, &payload); err != nil {
		_ = n.sendError(envelope, protocol.CodeMalformed, err.Error())
		return err
	}
	result, err := owned.session.Write(n.ctx, resource.SessionData{Offset: payload.Offset, Payload: payload.Data, Checksum: payload.Checksum})
	if err != nil {
		_ = n.sendError(envelope, protocol.CodeConflict, err.Error())
		return err
	}
	return n.sendSessionResponse(envelope, protocol.OperationSessionData, result)
}

func (n *Node) handleSessionClose(envelope protocol.Envelope, inbound *peerSession) error {
	owned, err := n.validateResourceSession(envelope, inbound)
	if err != nil {
		_ = n.sendError(envelope, protocol.CodeExpired, err.Error())
		return err
	}
	var payload protocol.SessionCloseV2
	if err := protocol.DecodeJSONPayload(envelope.Payload, protocol.DefaultMaxPayload, &payload); err != nil {
		_ = n.sendError(envelope, protocol.CodeMalformed, err.Error())
		return err
	}
	n.mu.Lock()
	delete(n.resourceSessions, owned.id)
	n.mu.Unlock()
	result, err := owned.session.Close(n.ctx, resource.SessionCloseRequest{Commit: payload.Commit, Schema: payload.Schema, Payload: payload.Payload})
	if err != nil {
		owned.session.Abort(err)
		_ = n.sendError(envelope, protocol.CodeConflict, err.Error())
		return err
	}
	return n.sendSessionResponse(envelope, protocol.OperationSessionClose, result)
}

func (n *Node) sendSessionResponse(request protocol.Envelope, operation protocol.Operation, result resource.OperationResult) error {
	response, err := n.newEnvelope(
		protocol.PhaseResponse, operation, request.Source, request.Resource, request.Capability,
		request.MessageID, 0, "application/octet-stream", result.Schema, result.Payload,
	)
	if err != nil {
		return err
	}
	return n.routeEnvelope(n.ctx, response, nil)
}

func (n *Node) validateResourceSession(envelope protocol.Envelope, inbound *peerSession) (*ownedResourceSession, error) {
	if err := n.authorizeLocal(envelope); err != nil {
		return nil, err
	}
	n.mu.RLock()
	owned := n.resourceSessions[envelope.CorrelationID]
	n.mu.RUnlock()
	if owned == nil {
		return nil, errors.New("resource session not found")
	}
	linkID := "local"
	if inbound != nil {
		linkID = inbound.linkID
	}
	if owned.subject != envelope.Subject() || owned.resource != envelope.Resource || owned.capability != envelope.Capability || owned.linkID != linkID {
		return nil, errors.New("resource session binding does not match request")
	}
	if !time.Now().Before(owned.expiresAt) || owned.policyGeneration != auth.PolicyGeneration(n.policy) {
		n.removeResourceSession(owned.id, errors.New("resource session authorization expired"))
		return nil, errors.New("resource session authorization expired")
	}
	return owned, nil
}

func (n *Node) removeResourceSession(id protocol.MessageID, reason error) bool {
	n.mu.Lock()
	owned := n.resourceSessions[id]
	if owned != nil {
		delete(n.resourceSessions, id)
	}
	n.mu.Unlock()
	if owned == nil {
		return false
	}
	owned.session.Abort(reason)
	return true
}

func (n *Node) cleanupResourceSessionsLink(linkID string, reason error) int {
	n.mu.RLock()
	ids := make([]protocol.MessageID, 0)
	for id, owned := range n.resourceSessions {
		if owned.linkID == linkID {
			ids = append(ids, id)
		}
	}
	n.mu.RUnlock()
	for _, id := range ids {
		n.removeResourceSession(id, reason)
	}
	return len(ids)
}

func (n *Node) cleanupResourceSessionsPolicy(generation uint64) int {
	n.mu.RLock()
	ids := make([]protocol.MessageID, 0)
	for id, owned := range n.resourceSessions {
		if owned.policyGeneration != generation {
			ids = append(ids, id)
		}
	}
	n.mu.RUnlock()
	for _, id := range ids {
		n.removeResourceSession(id, errors.New("resource session policy generation changed"))
	}
	return len(ids)
}
