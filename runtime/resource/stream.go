package resource

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
)

type StreamEvent struct {
	Sequence uint64
	Value    []byte
}

type Stream struct {
	mu         sync.RWMutex
	descriptor Descriptor
	sequence   uint64
	nextWatch  uint64
	watchers   map[uint64]func(StreamEvent)
}

func NewStream(descriptor Descriptor) (*Stream, error) {
	var err error
	descriptor, err = normalizeDescriptor(descriptor)
	if err != nil {
		return nil, err
	}
	if descriptor.Type != protocol.ResourceTypeStream {
		return nil, errors.New("stream descriptor must use mfh.stream type")
	}
	if _, ok := descriptor.Capability(protocol.CapabilitySubscribe); !ok {
		return nil, errors.New("stream descriptor requires subscribe capability")
	}
	return &Stream{descriptor: descriptor, watchers: make(map[uint64]func(StreamEvent))}, nil
}

func (s *Stream) Descriptor() Descriptor { return cloneDescriptor(s.descriptor) }

func (s *Stream) Operate(ctx context.Context, request OperationRequest) (OperationResult, error) {
	if ctx == nil {
		return OperationResult{}, errors.New("stream context is required")
	}
	if request.Capability != protocol.CapabilityPublish {
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
	if _, ok := s.descriptor.Capability(protocol.CapabilityPublish); !ok {
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
	event, err := s.Publish(request.Payload)
	if err != nil {
		return OperationResult{}, err
	}
	return OperationResult{Schema: request.Schema, Payload: event.Value}, nil
}

func (s *Stream) Observe(observer func(Observation)) (*Observation, func(), error) {
	cancel, err := s.Watch(func(event StreamEvent) {
		observer(Observation{Sequence: event.Sequence, Schema: s.eventSchema(), Value: event.Value})
	})
	return nil, cancel, err
}

func (s *Stream) Sequence() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sequence
}

func (s *Stream) Publish(value []byte) (StreamEvent, error) {
	if err := validatePayload(s.descriptor, value); err != nil {
		return StreamEvent{}, err
	}
	s.mu.Lock()
	if s.sequence == math.MaxUint64 {
		s.mu.Unlock()
		return StreamEvent{}, errors.New("stream sequence exhausted")
	}
	s.sequence++
	event := StreamEvent{Sequence: s.sequence, Value: append([]byte(nil), value...)}
	watchers := s.copyWatchersLocked()
	s.mu.Unlock()
	notifyStream(watchers, event)
	return event, nil
}

func (s *Stream) Apply(sequence uint64, value []byte) (StreamEvent, error) {
	if err := validatePayload(s.descriptor, value); err != nil {
		return StreamEvent{}, err
	}
	s.mu.Lock()
	if sequence <= s.sequence {
		s.mu.Unlock()
		return StreamEvent{}, ErrRevisionRegression
	}
	s.sequence = sequence
	event := StreamEvent{Sequence: sequence, Value: append([]byte(nil), value...)}
	watchers := s.copyWatchersLocked()
	s.mu.Unlock()
	notifyStream(watchers, event)
	return event, nil
}

func (s *Stream) eventSchema() string {
	capability, _ := s.descriptor.Capability(protocol.CapabilitySubscribe)
	return capability.EventSchema
}

func (s *Stream) Watch(observer func(StreamEvent)) (func(), error) {
	if observer == nil {
		return nil, errors.New("stream observer is required")
	}
	s.mu.Lock()
	s.nextWatch++
	id := s.nextWatch
	s.watchers[id] = observer
	s.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.watchers, id)
			s.mu.Unlock()
		})
	}, nil
}

func (s *Stream) copyWatchersLocked() []func(StreamEvent) {
	result := make([]func(StreamEvent), 0, len(s.watchers))
	for _, watcher := range s.watchers {
		result = append(result, watcher)
	}
	return result
}

func notifyStream(watchers []func(StreamEvent), event StreamEvent) {
	for _, watcher := range watchers {
		watcher(StreamEvent{Sequence: event.Sequence, Value: append([]byte(nil), event.Value...)})
	}
}
