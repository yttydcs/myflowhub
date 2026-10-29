package main

import (
	"encoding/json"
	"testing"
	"time"
)

type eagerBatchClient struct{ testSubscriptionClient }

func (*eagerBatchClient) PollSubscriptions(string, int64) (string, error) {
	return `[{"subscription_id":1,"kind":"event","payload":true}]`, nil
}

func TestDesktopBatchPollCoalescesEagerResponses(t *testing.T) {
	var sessions desktopSubscriptions
	client := &eagerBatchClient{}
	if _, err := sessions.add(client, 1); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	for i := 0; i < 5; i++ {
		raw, err := sessions.pollMany(`[1]`, 0)
		if err != nil {
			t.Fatal(err)
		}
		var items []struct {
			ID   int64  `json:"subscription_id"`
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal([]byte(raw), &items); err != nil || len(items) != 1 || items[0].ID != 1 || items[0].Kind != "event" {
			t.Fatalf("eager event was lost: %s, %v", raw, err)
		}
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Fatalf("eager IPC responses bypassed coalescing: %s", elapsed)
	}
	sessions.cancel(1)
	raw, err := sessions.pollMany(`[1]`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var items []struct{ Kind string }
	if err := json.Unmarshal([]byte(raw), &items); err != nil || len(items) != 1 || items[0].Kind != "error" {
		t.Fatalf("cancelled handle still returned data: %s, %v", raw, err)
	}
}

func TestNativeBatchPreservesStructuredPayloadAndLegacyJSON(t *testing.T) {
	app := &App{}
	if _, err := app.subscriptions.add(&eagerBatchClient{}, 1); err != nil {
		t.Fatal(err)
	}
	batch, err := app.PollSubscriptionsBatch(`[1]`, 0)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := app.PollSubscriptionsJSON(`[1]`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != legacy || legacy != `[{"subscription_id":1,"kind":"event","payload":true}]` {
		t.Fatalf("native payload was stringified or base64 encoded: %s / %s", encoded, legacy)
	}
	if _, err := app.PollSubscriptionsBatch(`[1,1]`, 0); err == nil {
		t.Fatal("invalid IDs accepted")
	}
}
