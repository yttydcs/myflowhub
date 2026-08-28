package resource

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

var ErrPublishRate = errors.New("topic publisher rate limit reached")

type TopicConfig struct {
	MaxPublishers      int
	MaxSubscribers     int
	MaxEventsPerSecond int
}

func (c TopicConfig) normalized(descriptor Descriptor) (TopicConfig, error) {
	if c.MaxPublishers <= 0 {
		c.MaxPublishers = 1024
	}
	if c.MaxSubscribers <= 0 {
		c.MaxSubscribers = descriptor.Limits.MaxSubscribers
	}
	if c.MaxSubscribers <= 0 {
		c.MaxSubscribers = 1024
	}
	if c.MaxEventsPerSecond <= 0 {
		c.MaxEventsPerSecond = 256
	}
	if c.MaxPublishers > protocol.MaxItems || c.MaxSubscribers > protocol.MaxItems || c.MaxEventsPerSecond > protocol.MaxItems {
		return TopicConfig{}, errors.New("topic limits exceed protocol maximum")
	}
	return c, nil
}

type TopicEvent struct {
	Sequence          uint64
	Publisher         protocol.NodeID
	PublisherSequence uint64
	PublishedAt       time.Time
	Value             []byte
}

type publisherState struct {
	sequence    uint64
	windowStart time.Time
	windowCount int
}

type Topic struct {
	mu         sync.RWMutex
	descriptor Descriptor
	config     TopicConfig
	sequence   uint64
	publishers map[protocol.NodeID]publisherState
	nextWatch  uint64
	watchers   map[uint64]func(TopicEvent)
}

func NewTopic(descriptor Descriptor, config TopicConfig) (*Topic, error) {
	var err error
	descriptor, err = normalizeDescriptor(descriptor)
	if err != nil {
		return nil, err
	}
	if descriptor.Type != protocol.ResourceTypeTopic {
		return nil, errors.New("topic descriptor must use mfh.topic type")
	}
	if _, ok := descriptor.Capability(protocol.CapabilityPublish); !ok {
		return nil, errors.New("topic descriptor requires publish capability")
	}
	if _, ok := descriptor.Capability(protocol.CapabilitySubscribe); !ok {
		return nil, errors.New("topic descriptor requires subscribe capability")
	}
	config, err = config.normalized(descriptor)
	if err != nil {
		return nil, err
	}
	return &Topic{descriptor: descriptor, config: config, publishers: make(map[protocol.NodeID]publisherState), watchers: make(map[uint64]func(TopicEvent))}, nil
}

func (t *Topic) Descriptor() Descriptor { return cloneDescriptor(t.descriptor) }

func (t *Topic) Operate(ctx context.Context, request OperationRequest) (OperationResult, error) {
	if ctx == nil {
		return OperationResult{}, errors.New("topic context is required")
	}
	if request.Capability != protocol.CapabilityPublish {
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
	if _, err := t.Publish(request.Subject, request.Payload); err != nil {
		return OperationResult{}, err
	}
	return OperationResult{}, nil
}

func (t *Topic) Observe(observer func(Observation)) (*Observation, func(), error) {
	cancel, err := t.Watch(func(event TopicEvent) {
		observer(Observation{
			Sequence: event.Sequence, Publisher: event.Publisher, PublisherSequence: event.PublisherSequence,
			Schema: t.eventSchema(), Value: event.Value,
		})
	})
	return nil, cancel, err
}

func (t *Topic) Publish(publisher protocol.NodeID, value []byte) (TopicEvent, error) {
	if err := publisher.Validate(); err != nil {
		return TopicEvent{}, fmt.Errorf("topic publisher: %w", err)
	}
	if err := validatePayload(t.descriptor, value); err != nil {
		return TopicEvent{}, err
	}
	now := time.Now().UTC()
	t.mu.Lock()
	state, exists := t.publishers[publisher]
	if !exists && len(t.publishers) >= t.config.MaxPublishers {
		t.mu.Unlock()
		return TopicEvent{}, errors.New("topic publisher limit reached")
	}
	if state.windowStart.IsZero() || now.Sub(state.windowStart) >= time.Second {
		state.windowStart = now
		state.windowCount = 0
	}
	if state.windowCount >= t.config.MaxEventsPerSecond {
		t.mu.Unlock()
		return TopicEvent{}, ErrPublishRate
	}
	if t.sequence == math.MaxUint64 || state.sequence == math.MaxUint64 {
		t.mu.Unlock()
		return TopicEvent{}, errors.New("topic sequence exhausted")
	}
	t.sequence++
	state.sequence++
	state.windowCount++
	t.publishers[publisher] = state
	event := TopicEvent{
		Sequence: t.sequence, Publisher: publisher, PublisherSequence: state.sequence,
		PublishedAt: now, Value: append([]byte(nil), value...),
	}
	watchers := t.copyWatchersLocked()
	t.mu.Unlock()
	for _, watcher := range watchers {
		watcher(TopicEvent{
			Sequence: event.Sequence, Publisher: event.Publisher, PublisherSequence: event.PublisherSequence,
			PublishedAt: event.PublishedAt, Value: append([]byte(nil), event.Value...),
		})
	}
	return event, nil
}

func (t *Topic) Watch(observer func(TopicEvent)) (func(), error) {
	if observer == nil {
		return nil, errors.New("topic observer is required")
	}
	t.mu.Lock()
	if len(t.watchers) >= t.config.MaxSubscribers {
		t.mu.Unlock()
		return nil, errors.New("topic subscriber limit reached")
	}
	t.nextWatch++
	id := t.nextWatch
	t.watchers[id] = observer
	t.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			t.mu.Lock()
			delete(t.watchers, id)
			t.mu.Unlock()
		})
	}, nil
}

func (t *Topic) copyWatchersLocked() []func(TopicEvent) {
	result := make([]func(TopicEvent), 0, len(t.watchers))
	for _, watcher := range t.watchers {
		result = append(result, watcher)
	}
	return result
}

func (t *Topic) eventSchema() string {
	capability, _ := t.descriptor.Capability(protocol.CapabilitySubscribe)
	return capability.EventSchema
}
