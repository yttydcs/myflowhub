package file

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

type uploadResource struct {
	controller *Controller
	descriptor resource.Descriptor
}

func newUploadResource(controller *Controller) (*uploadResource, error) {
	if controller == nil || controller.node == nil {
		return nil, errors.New("file upload resource requires a controller")
	}
	descriptor := resource.SessionDescriptor(
		protocol.ResourceID{Owner: controller.node.ID(), Name: protocol.BuiltinFileUpload},
		protocol.ResourceTypeFile, "application/json", protocol.SchemaFileOfferV1,
		protocol.SchemaFileProgressV1, "file.write", protocol.DefaultMaxPayload,
	)
	return &uploadResource{controller: controller, descriptor: descriptor}, nil
}

func (r *uploadResource) Descriptor() resource.Descriptor { return r.descriptor }

func (r *uploadResource) Operate(context.Context, resource.OperationRequest) (resource.OperationResult, error) {
	return resource.OperationResult{}, fmt.Errorf("%w: file upload uses an open session", resource.ErrUnsupportedCapability)
}

func (r *uploadResource) OpenSession(_ context.Context, request resource.SessionOpenRequest) (resource.ActiveSession, error) {
	if request.Capability != protocol.CapabilityOpen {
		return nil, fmt.Errorf("%w: %s", resource.ErrUnsupportedCapability, request.Capability)
	}
	if err := request.Subject.Validate(); err != nil {
		return nil, fmt.Errorf("file session subject: %w", err)
	}
	var offer protocol.FileOfferV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &offer); err != nil {
		return nil, err
	}
	if _, err := r.controller.offerFor(request.Subject, offer); err != nil {
		return nil, err
	}
	return &uploadSession{controller: r.controller, owner: request.Subject, offer: offer}, nil
}

type uploadSession struct {
	controller *Controller
	owner      protocol.NodeID
	offer      protocol.FileOfferV1

	mu     sync.Mutex
	closed bool
}

func (s *uploadSession) Grant() resource.SessionGrant {
	return resource.SessionGrant{
		Capability: protocol.CapabilityOpen, MaxChunkBytes: s.offer.ChunkSize,
		MaxTotalBytes: s.offer.Size, ExpiresAtUnixMS: s.offer.ExpiresAtUnixMS,
	}
}

func (s *uploadSession) Write(_ context.Context, data resource.SessionData) (resource.OperationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return resource.OperationResult{}, errors.New("file upload session is closed")
	}
	payload, err := s.controller.chunkFor(s.owner, protocol.FileChunkV1{
		Version: protocol.SchemaVersionV1, TransferID: s.offer.TransferID, Offset: data.Offset,
		Data: append([]byte(nil), data.Payload...), SHA256: data.Checksum,
	})
	if err != nil {
		return resource.OperationResult{}, err
	}
	return resource.OperationResult{Schema: protocol.SchemaFileProgressV1, Payload: payload}, nil
}

func (s *uploadSession) Close(_ context.Context, request resource.SessionCloseRequest) (resource.OperationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return resource.OperationResult{}, errors.New("file upload session is closed")
	}
	s.closed = true
	if request.Commit {
		payload, err := s.controller.completeFor(s.owner, protocol.FileCompleteV1{
			Version: protocol.SchemaVersionV1, TransferID: s.offer.TransferID, Size: s.offer.Size, SHA256: s.offer.SHA256,
		})
		return resource.OperationResult{Schema: protocol.SchemaFileProgressV1, Payload: payload}, err
	}
	reason := "upload session closed without commit"
	if len(request.Payload) > 0 {
		reason = string(request.Payload)
	}
	payload, err := s.controller.cancelFor(s.owner, protocol.FileCancelV1{
		Version: protocol.SchemaVersionV1, TransferID: s.offer.TransferID, Reason: reason,
	})
	return resource.OperationResult{Schema: protocol.SchemaFileProgressV1, Payload: payload}, err
}

func (s *uploadSession) Abort(reason error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.controller.abortFor(s.owner, s.offer.TransferID, reason)
}
