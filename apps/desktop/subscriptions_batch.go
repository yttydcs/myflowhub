package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

type batchSubscriptionClient interface {
	PollSubscriptions(string, int64) (string, error)
}
type pollGroup struct {
	client     subscriptionClient
	ids        []int64
	desktopIDs map[int64]int64
}

func (a *App) PollSubscriptionsJSON(subscriptionIDsJSON string, timeoutMS int64) (string, error) {
	return a.subscriptions.pollMany(subscriptionIDsJSON, timeoutMS)
}

type SubscriptionBatchItem struct {
	SubscriptionID int64  `json:"subscription_id"`
	Kind           string `json:"kind"`
	Payload        any    `json:"payload,omitempty"`
}

// The native bridge serializes the batch once. RawMessage remains structured
// JSON here; the network/SDK boundary has already validated its wire payload.
func (a *App) PollSubscriptionsBatch(subscriptionIDsJSON string, timeoutMS int64) ([]SubscriptionBatchItem, error) {
	items, err := a.subscriptions.pollManyItems(subscriptionIDsJSON, timeoutMS)
	if err != nil {
		return nil, err
	}
	batch := make([]SubscriptionBatchItem, len(items))
	for i, item := range items {
		batch[i] = SubscriptionBatchItem{SubscriptionID: item.SubscriptionID, Kind: item.Kind}
		if len(item.Payload) > 0 {
			batch[i].Payload = item.Payload
		}
	}
	return batch, nil
}

func (s *desktopSubscriptions) pollMany(raw string, timeoutMS int64) (string, error) {
	items, err := s.pollManyItems(raw, timeoutMS)
	if err != nil {
		return "", err
	}
	if items == nil {
		return "[]", nil
	}
	encoded, err := json.Marshal(items)
	return string(encoded), err
}

func (s *desktopSubscriptions) pollManyItems(raw string, timeoutMS int64) ([]desktopbinding.SubscriptionPoll, error) {
	if len(raw) > 128<<10 || timeoutMS < 0 || timeoutMS > 100 {
		return nil, errors.New("batch poll requires at most 128 KiB of IDs and timeout in 0..100ms")
	}
	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("decode desktop poll IDs: %w", err)
	}
	if len(ids) < 1 || len(ids) > desktopbinding.MaxPollSubscriptions {
		return nil, errors.New("batch poll requires 1..4096 subscription IDs")
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 || id > 1<<53-1 {
			return nil, errors.New("invalid desktop subscription ID")
		}
		if _, ok := seen[id]; ok {
			return nil, errors.New("duplicate desktop subscription ID")
		}
		seen[id] = struct{}{}
	}
	s.batchMu.Lock()
	defer s.batchMu.Unlock()
	// Coalesce sparse arrivals without depending on WebView background timers.
	// Waiting holds only the poll gate; control and cancellation stay independent.
	if delay := time.Until(s.nextBatch); delay > 0 {
		time.Sleep(delay)
	}
	s.nextBatch = time.Now().Add(16 * time.Millisecond)
	s.mu.Lock()
	groups := make([]pollGroup, 0, 1)
	byClient := make(map[subscriptionClient]int)
	missing := make([]desktopbinding.SubscriptionPoll, 0)
	for _, id := range ids {
		entry, ok := s.entries[id]
		if !ok {
			if len(missing) < desktopbinding.MaxPollEvents {
				missing = append(missing, desktopbinding.SubscriptionPoll{SubscriptionID: id, Kind: "error", Payload: json.RawMessage(`{"code":"gone","message":"desktop subscription is closed or unknown"}`)})
			}
			continue
		}
		index, ok := byClient[entry.client]
		if !ok {
			index = len(groups)
			byClient[entry.client] = index
			groups = append(groups, pollGroup{client: entry.client, desktopIDs: make(map[int64]int64)})
		}
		groups[index].ids = append(groups[index].ids, entry.id)
		groups[index].desktopIDs[entry.id] = id
	}
	s.mu.Unlock()
	if len(missing) > 0 {
		return missing, nil
	}
	start := s.batchCursor % len(groups)
	probe := func(index int, wait int64) ([]desktopbinding.SubscriptionPoll, error) {
		group := groups[index]
		client, ok := group.client.(batchSubscriptionClient)
		if !ok {
			return nil, errors.New("subscription client does not support batch polling")
		}
		encoded, err := json.Marshal(group.ids)
		if err != nil {
			return nil, err
		}
		result, err := client.PollSubscriptions(string(encoded), wait)
		if err != nil {
			failure, _ := json.Marshal(map[string]string{"code": "closed", "message": err.Error()})
			items := make([]desktopbinding.SubscriptionPoll, 0, min(len(group.ids), desktopbinding.MaxPollEvents))
			for _, id := range group.ids[:min(len(group.ids), desktopbinding.MaxPollEvents)] {
				items = append(items, desktopbinding.SubscriptionPoll{SubscriptionID: group.desktopIDs[id], Kind: "error", Payload: failure})
			}
			return items, nil
		}
		if len(result) > desktopbinding.MaxPollSingleBytes {
			return nil, errors.New("batch response exceeds the single-event limit")
		}
		var items []desktopbinding.SubscriptionPoll
		if err := json.Unmarshal([]byte(result), &items); err != nil {
			return nil, fmt.Errorf("decode subscription batch: %w", err)
		}
		if len(items) > desktopbinding.MaxPollEvents {
			return nil, errors.New("subscription batch exceeds the event limit")
		}
		for i := range items {
			id, ok := group.desktopIDs[items[i].SubscriptionID]
			if !ok {
				return nil, errors.New("subscription batch returned an unrequested ID")
			}
			items[i].SubscriptionID = id
		}
		return items, nil
	}
	for offset := 0; offset < len(groups); offset++ {
		index := (start + offset) % len(groups)
		items, err := probe(index, 0)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			s.batchCursor = (index + 1) % len(groups)
			return items, nil
		}
	}
	if timeoutMS == 0 {
		return nil, nil
	}
	items, err := probe(start, timeoutMS)
	s.batchCursor = (start + 1) % len(groups)
	if err != nil {
		return nil, err
	}
	return items, nil
}
