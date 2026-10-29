package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	MaxPollSubscriptions = 4096
	MaxPollEvents        = 256
	MaxPollBatchBytes    = 256 << 10
	MaxPollSingleBytes   = (1 << 20) + 8192
)

type SubscriptionPoll struct {
	SubscriptionID int64           `json:"subscription_id"`
	Kind           string          `json:"kind"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

// PollSubscriptions drains bounded, fairly selected events. A zero timeout is
// an explicit nonblocking probe; idle waits are capped at 100ms. A legal large
// event may occupy one batch alone, never be dequeued and silently discarded.
func (c *Client) PollSubscriptions(subscriptionIDsJSON string, timeoutMS int64) (string, error) {
	if c == nil {
		return "", errors.New("desktop binding client is not open")
	}
	if len(subscriptionIDsJSON) > 128<<10 || timeoutMS < 0 || timeoutMS > 100 {
		return "", errors.New("batch poll requires at most 128 KiB of IDs and timeout in 0..100ms")
	}
	var ids []int64
	if err := json.Unmarshal([]byte(subscriptionIDsJSON), &ids); err != nil {
		return "", fmt.Errorf("decode poll IDs: %w", err)
	}
	if len(ids) < 1 || len(ids) > MaxPollSubscriptions {
		return "", errors.New("batch poll requires 1..4096 subscription IDs")
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 || id > 1<<53-1 {
			return "", errors.New("batch subscription IDs must be positive JS-safe integers")
		}
		if _, exists := seen[id]; exists {
			return "", errors.New("duplicate batch subscription ID")
		}
		seen[id] = struct{}{}
	}
	c.batchMu.Lock()
	defer c.batchMu.Unlock()
	// Frontend waiters are reinserted after delivery; their input order changes.
	// Fairness must follow stable IDs rather than an index into that order.
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	defer timer.Stop()
	for {
		items := make([]SubscriptionPoll, 0, MaxPollEvents)
		bytes := 2
		start := sort.Search(len(ids), func(i int) bool { return ids[i] > c.batchAfter }) % len(ids)
		for offset := 0; offset < len(ids) && len(items) < MaxPollEvents; offset++ {
			index := (start + offset) % len(ids)
			id := ids[index]
			c.mu.Lock()
			current := c.subscriptions[id]
			closed := c.core == nil
			c.mu.Unlock()
			if closed {
				return "", errors.New("desktop binding client is closed")
			}
			var item *SubscriptionPoll
			if current == nil {
				item = &SubscriptionPoll{SubscriptionID: id, Kind: "error", Payload: json.RawMessage(`{"code":"gone","message":"subscription is closed or unknown"}`)}
			} else {
				if !current.pollMu.TryLock() {
					return "", errors.New("desktop subscription already has an active poll")
				}
				item = current.takePoll(id)
			}
			if item == nil {
				if current != nil {
					current.pollMu.Unlock()
				}
				continue
			}
			encoded, err := json.Marshal(item)
			if err != nil {
				if current != nil {
					current.pollMu.Unlock()
				}
				return "", fmt.Errorf("encode subscription poll: %w", err)
			}
			if len(encoded)+2 > MaxPollSingleBytes {
				if current != nil {
					current.pollMu.Unlock()
				}
				return "", errors.New("subscription poll exceeds the single-event limit")
			}
			if bytes+len(encoded)+1 > MaxPollBatchBytes-MaxPollEvents*16 && len(items) > 0 {
				if current != nil {
					current.pending = item
					current.pollMu.Unlock()
				}
				break
			}
			if current != nil {
				current.pollMu.Unlock()
			}
			items = append(items, *item)
			bytes += len(encoded) + 1
			c.batchAfter = id
		}
		if len(items) > 0 || timeoutMS == 0 {
			encoded, err := json.Marshal(items)
			return string(encoded), err
		}
		select {
		case <-c.ready:
		case <-timer.C:
			return "[]", nil
		}
	}
}

// Called with pollMu held. Terminal/error events take priority over cached data.
func (s *subscription) takePoll(id int64) *SubscriptionPoll {
	select {
	case failure := <-s.errors:
		s.pending = nil
		return &SubscriptionPoll{SubscriptionID: id, Kind: "error", Payload: json.RawMessage(failure)}
	default:
	}
	select {
	case <-s.done:
		s.pending = nil
		return &SubscriptionPoll{SubscriptionID: id, Kind: "closed"}
	default:
	}
	if s.pending != nil {
		item := s.pending
		s.pending = nil
		return item
	}
	select {
	case event := <-s.events:
		return &SubscriptionPoll{SubscriptionID: id, Kind: "event", Payload: json.RawMessage(event)}
	default:
		return nil
	}
}
