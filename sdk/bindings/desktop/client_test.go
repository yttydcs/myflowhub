package desktop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/sdk/bindings"
)

func TestAttachedClientLifecycleAndPollingBoundary(t *testing.T) {
	if _, err := NewAttachedClient(nil); err == nil {
		t.Fatal("nil attached core was accepted")
	}
	host, err := nodehost.New(context.Background(), nodehost.Config{
		StateDirectory: t.TempDir(), NodeID: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	core, err := bindings.NewAttachedClient(host.Client(), bindings.PublicIdentity{
		NodeID: host.ID(), PublicKey: host.PublicKey(),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(core)
	if err != nil {
		t.Fatal(err)
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

	// The facade is non-owning: its Close must not close the Host client or
	// prevent the Host from starting its own lifecycle.
	if id, err := host.Client().NodeID(); err != nil || id != protocol.NodeID(2) {
		t.Fatalf("facade close affected host client: id=%d err=%v", id, err)
	}
	if err := host.Start(); err != nil {
		t.Fatalf("facade close affected host lifecycle: %v", err)
	}
}
