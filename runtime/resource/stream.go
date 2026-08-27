package resource

import (
	"errors"
	"math"
	"sync"
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
	if descriptor.Kind != KindStream {
		return nil, errors.New("stream descriptor must use stream kind")
	}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	return &Stream{descriptor: descriptor, watchers: make(map[uint64]func(StreamEvent))}, nil
}

func (s *Stream) Descriptor() Descriptor { return s.descriptor }

func (s *Stream) Sequence() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sequence
}

func (s *Stream) Publish(value []byte) (StreamEvent, error) {
	if err := s.descriptor.validateValue(value); err != nil {
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
	if err := s.descriptor.validateValue(value); err != nil {
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
