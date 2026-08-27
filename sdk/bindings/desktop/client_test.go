package desktop

import (
	"encoding/json"
	"testing"
)

func TestClientLifecycleAndPollingBoundary(t *testing.T) {
	var client Client
	if _, err := client.IdentityJSON(); err == nil {
		t.Fatal("unopened desktop client was accepted")
	}
	if err := client.Open(t.TempDir(), 2); err != nil {
		t.Fatal(err)
	}
	if err := client.Open(t.TempDir(), 3); err == nil {
		t.Fatal("desktop client was opened twice")
	}
	identity, err := client.IdentityJSON()
	if err != nil || !json.Valid([]byte(identity)) {
		t.Fatalf("invalid desktop identity: %v (%s)", err, identity)
	}
	status, err := client.StatusJSON()
	if err != nil || !json.Valid([]byte(status)) {
		t.Fatalf("invalid desktop status: %v (%s)", err, status)
	}

	current := newSubscription()
	client.mu.Lock()
	client.subscriptions[1] = current
	client.mu.Unlock()
	current.OnEvent(`{"kind":"stream","resource_name":"test/events"}`)
	result, err := client.PollSubscription(1, 1_000)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(result), &envelope); err != nil || envelope.Kind != "event" || !json.Valid(envelope.Payload) {
		t.Fatalf("invalid desktop poll result: %v (%s)", err, result)
	}
	client.CancelSubscription(1)
	if _, err := client.PollSubscription(1, 1_000); err == nil {
		t.Fatal("cancelled desktop subscription remained available")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}
