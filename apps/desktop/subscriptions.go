package main

import (
	"errors"
	"fmt"
	"github.com/yttydcs/myflowhub/host/nodehost"
	"sync"
	"time"
)

type subscriptionClient interface {
	PollSubscription(int64, int64) (string, error)
	CancelSubscription(int64)
}

type desktopSubscription struct {
	client subscriptionClient
	id     int64
}

// SDK subscription IDs belong to their client. Desktop IDs retain that client
// so a late poll/cancel from an old Profile cannot address a new Profile's ID.
type desktopSubscriptions struct {
	batchMu     sync.Mutex
	batchCursor int
	nextBatch   time.Time
	mu          sync.Mutex
	next        int64
	limit       int
	entries     map[int64]desktopSubscription
}

func (s *desktopSubscriptions) add(client subscriptionClient, id int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := s.limit
	if limit == 0 {
		limit = nodehost.DefaultProductSubscriptions
	}
	if len(s.entries) >= limit {
		return 0, fmt.Errorf("desktop supports at most %d active subscriptions", limit)
	}
	if s.next >= 1<<53-1 {
		return 0, errors.New("desktop subscription IDs exhausted")
	}
	if s.entries == nil {
		s.entries = make(map[int64]desktopSubscription)
	}
	s.next++
	s.entries[s.next] = desktopSubscription{client: client, id: id}
	return s.next, nil
}

func (s *desktopSubscriptions) poll(id, timeout int64) (string, error) {
	s.mu.Lock()
	entry, exists := s.entries[id]
	s.mu.Unlock()
	if !exists {
		return "", errors.New("desktop subscription is closed or unknown")
	}
	return entry.client.PollSubscription(entry.id, timeout)
}

func (s *desktopSubscriptions) cancel(id int64) {
	s.mu.Lock()
	entry, exists := s.entries[id]
	delete(s.entries, id)
	s.mu.Unlock()
	if exists {
		entry.client.CancelSubscription(entry.id)
	}
}

func (s *desktopSubscriptions) close() {
	s.mu.Lock()
	entries := s.entries
	s.entries = nil
	s.mu.Unlock()
	for _, entry := range entries {
		entry.client.CancelSubscription(entry.id)
	}
}
