package subscription

import (
	"errors"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

var ErrInterestLimit = errors.New("subscription interest limit reached")

type Interest struct {
	ID               protocol.MessageID
	Subscriber       protocol.NodeID
	Resource         protocol.ResourceID
	LinkID           string
	NextHop          protocol.NodeID
	LeaseUntil       time.Time
	AuthorizedUntil  time.Time
	TopologyEpoch    uint64
	PolicyGeneration uint64
}

type InterestTable struct {
	mu    sync.RWMutex
	max   int
	byID  map[protocol.MessageID]Interest
	byKey map[protocol.ResourceID]map[protocol.MessageID]struct{}
}

func NewInterestTable(max int) (*InterestTable, error) {
	if max <= 0 {
		return nil, errors.New("interest limit must be positive")
	}
	return &InterestTable{max: max, byID: make(map[protocol.MessageID]Interest), byKey: make(map[protocol.ResourceID]map[protocol.MessageID]struct{})}, nil
}

func (t *InterestTable) Add(interest Interest) (bool, error) {
	if interest.ID.IsZero() {
		return false, errors.New("interest ID is required")
	}
	if err := interest.Subscriber.Validate(); err != nil {
		return false, err
	}
	if err := interest.Resource.Validate(); err != nil {
		return false, err
	}
	if interest.LinkID == "" || interest.LeaseUntil.IsZero() || interest.AuthorizedUntil.Before(interest.LeaseUntil) {
		return false, errors.New("interest requires a link and authorization covering its lease")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.byID[interest.ID]; exists {
		return false, errors.New("interest ID already exists")
	}
	if len(t.byID) >= t.max {
		return false, ErrInterestLimit
	}
	first := len(t.byKey[interest.Resource]) == 0
	if t.byKey[interest.Resource] == nil {
		t.byKey[interest.Resource] = make(map[protocol.MessageID]struct{})
	}
	t.byID[interest.ID] = interest
	t.byKey[interest.Resource][interest.ID] = struct{}{}
	return first, nil
}

func (t *InterestTable) Remove(id protocol.MessageID) (Interest, bool, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	interest, ok := t.byID[id]
	if !ok {
		return Interest{}, false, false
	}
	delete(t.byID, id)
	group := t.byKey[interest.Resource]
	delete(group, id)
	last := len(group) == 0
	if last {
		delete(t.byKey, interest.Resource)
	}
	return interest, true, last
}

func (t *InterestTable) CleanupLink(linkID string) []Interest {
	t.mu.RLock()
	ids := make([]protocol.MessageID, 0)
	for id, interest := range t.byID {
		if interest.LinkID == linkID {
			ids = append(ids, id)
		}
	}
	t.mu.RUnlock()
	removed := make([]Interest, 0, len(ids))
	for _, id := range ids {
		if interest, ok, _ := t.Remove(id); ok {
			removed = append(removed, interest)
		}
	}
	return removed
}

func (t *InterestTable) CleanupExpired(now time.Time) []Interest {
	t.mu.RLock()
	ids := make([]protocol.MessageID, 0)
	for id, interest := range t.byID {
		if !now.Before(interest.LeaseUntil) || !now.Before(interest.AuthorizedUntil) {
			ids = append(ids, id)
		}
	}
	t.mu.RUnlock()
	removed := make([]Interest, 0, len(ids))
	for _, id := range ids {
		if interest, ok, _ := t.Remove(id); ok {
			removed = append(removed, interest)
		}
	}
	return removed
}

func (t *InterestTable) CleanupPolicyGeneration(generation uint64) []Interest {
	t.mu.RLock()
	ids := make([]protocol.MessageID, 0)
	for id, interest := range t.byID {
		if interest.PolicyGeneration != generation {
			ids = append(ids, id)
		}
	}
	t.mu.RUnlock()
	removed := make([]Interest, 0, len(ids))
	for _, id := range ids {
		if interest, ok, _ := t.Remove(id); ok {
			removed = append(removed, interest)
		}
	}
	return removed
}

func (t *InterestTable) Interests(resource protocol.ResourceID) []Interest {
	t.mu.RLock()
	defer t.mu.RUnlock()
	group := t.byKey[resource]
	result := make([]Interest, 0, len(group))
	for id := range group {
		result = append(result, t.byID[id])
	}
	return result
}
