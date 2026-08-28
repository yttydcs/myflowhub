package subscription

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

var (
	ErrSubscriptionLimit = errors.New("subscription limit reached")
	ErrInvalidLease      = errors.New("invalid subscription lease")
	ErrNotSubscribable   = errors.New("resource is not subscribable")
)

type Config struct {
	MaxSubscriptions int
	DefaultQueue     int
	MaxQueue         int
	MinLease         time.Duration
	MaxLease         time.Duration
}

func (c Config) normalized() (Config, error) {
	if c.MaxSubscriptions <= 0 {
		c.MaxSubscriptions = 1024
	}
	if c.DefaultQueue <= 0 {
		c.DefaultQueue = 16
	}
	if c.MaxQueue <= 0 {
		c.MaxQueue = 256
	}
	if c.MinLease <= 0 {
		c.MinLease = time.Second
	}
	if c.MaxLease <= 0 {
		c.MaxLease = 24 * time.Hour
	}
	if c.DefaultQueue > c.MaxQueue || c.MinLease > c.MaxLease {
		return Config{}, errors.New("invalid subscription configuration")
	}
	return c, nil
}

type Request struct {
	ID               protocol.MessageID
	Subscriber       protocol.NodeID
	Resource         protocol.ResourceID
	Capability       protocol.CapabilityID
	LinkID           string
	NextHop          protocol.NodeID
	Lease            time.Duration
	AuthorizedUntil  time.Time
	TopologyEpoch    uint64
	PolicyGeneration uint64
	Queue            int
}

type Subscription struct {
	ID         protocol.MessageID
	Subscriber protocol.NodeID
	Resource   protocol.ResourceID
	Capability protocol.CapabilityID
	LeaseUntil time.Time
	Events     <-chan Event
	cancel     func()
}

func (s *Subscription) Cancel() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
}

type entry struct {
	request      Request
	leaseUntil   time.Time
	delivery     *delivery
	watchCancel  func()
	expireCancel context.CancelFunc
}

type Manager struct {
	ctx      context.Context
	cancel   context.CancelFunc
	config   Config
	registry *resource.Registry
	interest *InterestTable

	mu      sync.Mutex
	entries map[protocol.MessageID]*entry
	closed  bool
}

func NewManager(parent context.Context, registry *resource.Registry, config Config) (*Manager, error) {
	if parent == nil || registry == nil {
		return nil, errors.New("subscription context and registry are required")
	}
	normalized, err := config.normalized()
	if err != nil {
		return nil, err
	}
	interest, err := NewInterestTable(normalized.MaxSubscriptions)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	return &Manager{ctx: ctx, cancel: cancel, config: normalized, registry: registry, interest: interest, entries: make(map[protocol.MessageID]*entry)}, nil
}

func (m *Manager) Subscribe(request Request) (*Subscription, error) {
	if request.ID.IsZero() {
		id, err := protocol.NewMessageID()
		if err != nil {
			return nil, err
		}
		request.ID = id
	}
	if err := request.Subscriber.Validate(); err != nil {
		return nil, err
	}
	if err := request.Resource.Validate(); err != nil {
		return nil, err
	}
	if err := request.Capability.Validate(); err != nil {
		return nil, err
	}
	if request.LinkID == "" {
		return nil, errors.New("subscription link ID is required")
	}
	if request.Lease < m.config.MinLease || request.Lease > m.config.MaxLease {
		return nil, ErrInvalidLease
	}
	queue := request.Queue
	if queue == 0 {
		queue = m.config.DefaultQueue
	}
	if queue < 1 || queue > m.config.MaxQueue {
		return nil, errors.New("subscription queue is outside configured limits")
	}
	now := time.Now()
	leaseUntil := now.Add(request.Lease)
	if request.AuthorizedUntil.IsZero() {
		request.AuthorizedUntil = leaseUntil
	}
	if request.AuthorizedUntil.Before(leaseUntil) {
		return nil, errors.New("subscription authorization does not cover requested lease")
	}
	value, ok := m.registry.Resolve(request.Resource)
	if !ok {
		return nil, resource.ErrNotFound
	}
	valueDelivery := newDelivery(m.ctx, request.Resource, queue)
	current := &entry{request: request, leaseUntil: leaseUntil, delivery: valueDelivery}
	ready := make(chan struct{})
	readyClosed := false
	defer func() {
		if !readyClosed {
			close(ready)
		}
	}()
	descriptor := value.Descriptor()
	capability, exists := descriptor.Capability(request.Capability)
	if !exists || capability.EventSchema == "" {
		return nil, ErrNotSubscribable
	}
	observable, ok := value.(resource.Observable)
	if !ok {
		return nil, ErrNotSubscribable
	}
	snapshot, cancel, err := observable.Observe(func(observation resource.Observation) {
		<-ready
		current.enqueueObservation(request, observation)
	})
	if err != nil {
		return nil, err
	}
	current.watchCancel = cancel
	if snapshot != nil {
		current.enqueueObservation(request, *snapshot)
	}
	interest := Interest{
		ID: request.ID, Subscriber: request.Subscriber, Resource: request.Resource, LinkID: request.LinkID, NextHop: request.NextHop,
		Capability: request.Capability,
		LeaseUntil: leaseUntil, AuthorizedUntil: request.AuthorizedUntil, TopologyEpoch: request.TopologyEpoch, PolicyGeneration: request.PolicyGeneration,
	}
	if _, err := m.interest.Add(interest); err != nil {
		current.watchCancel()
		return nil, err
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		_, _, _ = m.interest.Remove(request.ID)
		current.watchCancel()
		return nil, errors.New("subscription manager is closed")
	}
	if _, exists := m.entries[request.ID]; exists || len(m.entries) >= m.config.MaxSubscriptions {
		m.mu.Unlock()
		_, _, _ = m.interest.Remove(request.ID)
		current.watchCancel()
		if exists {
			return nil, errors.New("subscription ID already exists")
		}
		return nil, ErrSubscriptionLimit
	}
	m.entries[request.ID] = current
	m.mu.Unlock()
	close(ready)
	readyClosed = true
	valueDelivery.start(func() { m.remove(request.ID, false) })
	expireCtx, expireCancel := context.WithCancel(m.ctx)
	current.expireCancel = expireCancel
	go func() {
		timer := time.NewTimer(time.Until(leaseUntil))
		defer timer.Stop()
		select {
		case <-timer.C:
			valueDelivery.expire("lease_expired")
		case <-expireCtx.Done():
		}
	}()
	return &Subscription{
		ID: request.ID, Subscriber: request.Subscriber, Resource: request.Resource, Capability: request.Capability, LeaseUntil: leaseUntil, Events: valueDelivery.out,
		cancel: func() { m.Unsubscribe(request.ID) },
	}, nil
}

func (m *Manager) Unsubscribe(id protocol.MessageID) bool { return m.remove(id, true) }

func (m *Manager) UnsubscribeFor(id protocol.MessageID, subscriber protocol.NodeID) bool {
	m.mu.Lock()
	current, ok := m.entries[id]
	m.mu.Unlock()
	if !ok || current.request.Subscriber != subscriber {
		return false
	}
	return m.remove(id, true)
}

func (m *Manager) remove(id protocol.MessageID, stop bool) bool {
	m.mu.Lock()
	current, ok := m.entries[id]
	if ok {
		delete(m.entries, id)
	}
	m.mu.Unlock()
	if !ok {
		return false
	}
	_, _, _ = m.interest.Remove(id)
	if current.expireCancel != nil {
		current.expireCancel()
	}
	if current.watchCancel != nil {
		current.watchCancel()
	}
	if stop {
		current.delivery.stop()
	}
	return true
}

func (m *Manager) CleanupLink(linkID string) int {
	interests := m.interest.CleanupLink(linkID)
	for _, interest := range interests {
		m.remove(interest.ID, true)
	}
	return len(interests)
}

func (m *Manager) CleanupPolicyGeneration(generation uint64) int {
	interests := m.interest.CleanupPolicyGeneration(generation)
	for _, interest := range interests {
		m.mu.Lock()
		current := m.entries[interest.ID]
		m.mu.Unlock()
		if current != nil {
			current.delivery.expire("policy_generation_changed")
		}
	}
	return len(interests)
}

func (m *Manager) Interests(resourceID protocol.ResourceID) []Interest {
	return m.interest.Interests(resourceID)
}

func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	ids := make([]protocol.MessageID, 0, len(m.entries))
	for id := range m.entries {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.remove(id, true)
	}
	m.cancel()
	return nil
}

func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

func ValidateRequest(request Request) error {
	if request.ID.IsZero() || request.LinkID == "" {
		return fmt.Errorf("subscription ID and link are required")
	}
	if err := request.Resource.Validate(); err != nil {
		return err
	}
	return request.Capability.Validate()
}

func (e *entry) enqueueObservation(request Request, observation resource.Observation) {
	event := Event{
		Kind: EventData, Resource: request.Resource, Capability: request.Capability, Schema: observation.Schema,
		Revision: observation.Revision, Sequence: observation.Sequence, Publisher: observation.Publisher,
		PublisherSequence: observation.PublisherSequence, Value: observation.Value,
	}
	if observation.Snapshot {
		event.Kind = EventSnapshot
		e.delivery.enqueueSnapshot(event)
		return
	}
	if observation.Revision != 0 {
		e.delivery.enqueueVariable(event)
		return
	}
	e.delivery.enqueueStream(event)
}
