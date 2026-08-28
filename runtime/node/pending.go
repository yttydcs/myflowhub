package node

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

var ErrPendingOverflow = errors.New("pending response queue overflow")

type pendingEntry struct {
	frames chan protocol.Envelope
	done   chan struct{}

	mu   sync.Mutex
	err  error
	once sync.Once
}

func newPending(queue int) *pendingEntry {
	if queue < 1 {
		queue = 1
	}
	return &pendingEntry{frames: make(chan protocol.Envelope, queue), done: make(chan struct{})}
}

func (p *pendingEntry) deliver(envelope protocol.Envelope) {
	select {
	case <-p.done:
		return
	default:
	}
	select {
	case p.frames <- envelope:
	default:
		p.fail(ErrPendingOverflow)
	}
}

func (p *pendingEntry) fail(err error) {
	p.once.Do(func() {
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
		close(p.done)
	})
}

func (p *pendingEntry) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

type RemoteSubscription struct {
	ID         protocol.MessageID
	Resource   protocol.ResourceID
	Capability protocol.CapabilityID
	Events     <-chan subscription.Event
	Errors     <-chan error
	cancel     func()
}

func (s *RemoteSubscription) Cancel() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
}

func subscriptionEvent(envelope protocol.Envelope) (subscription.Event, error) {
	switch envelope.Operation {
	case protocol.OperationResourceEvent, protocol.OperationResourceGap:
		var payload resourceEventPayload
		if err := decodeJSON(envelope.Payload, &payload); err != nil {
			return subscription.Event{}, err
		}
		if payload.Version != protocol.SchemaVersionV2 {
			return subscription.Event{}, fmt.Errorf("resource event version must be %d", protocol.SchemaVersionV2)
		}
		kind := subscription.EventData
		if payload.Snapshot {
			kind = subscription.EventSnapshot
		}
		if envelope.Operation == protocol.OperationResourceGap {
			kind = subscription.EventGap
		}
		return subscription.Event{
			Kind: kind, Resource: envelope.Resource, Capability: envelope.Capability, Schema: payload.Schema,
			Revision: payload.Revision, Sequence: payload.Sequence, Publisher: payload.Publisher,
			PublisherSequence: payload.PublisherSequence, GapFrom: payload.GapFrom, GapTo: payload.GapTo,
			Value: payload.Value, Reason: payload.Reason,
		}, nil
	case protocol.OperationError:
		failure, err := protocol.DecodeErrorPayload(envelope.Payload)
		if err != nil {
			return subscription.Event{}, err
		}
		return subscription.Event{}, failure
	default:
		return subscription.Event{}, fmt.Errorf("unexpected subscription operation %d", envelope.Operation)
	}
}

func awaitFrame(ctx context.Context, pending *pendingEntry) (protocol.Envelope, error) {
	select {
	case envelope := <-pending.frames:
		return envelope, nil
	case <-pending.done:
		if err := pending.Err(); err != nil {
			return protocol.Envelope{}, err
		}
		return protocol.Envelope{}, errors.New("pending request closed")
	case <-ctx.Done():
		return protocol.Envelope{}, ctx.Err()
	}
}
