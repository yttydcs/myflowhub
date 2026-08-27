package clipboardmobile

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClientIdentityIsDurableAndBoundaryIsStrict(t *testing.T) {
	client := NewClient()
	request, _ := json.Marshal(identityRequestV1{Version: 1, StateDirectory: filepath.Join(t.TempDir(), "state"), NodeID: "81"})
	first, err := client.Identity(string(request))
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Identity(string(request))
	if err != nil || first != second {
		t.Fatalf("identity changed: %q %q (%v)", first, second, err)
	}
	if _, err := client.Identity(`{"version":1,"state_directory":"x","node_id":"1","unknown":true}`); err == nil {
		t.Fatal("unknown mobile boundary field was accepted")
	}
	if _, err := client.History(); err == nil {
		t.Fatal("idle client exposed history")
	}
}

func TestMobileAdapterWriteCompletionAndErrorPrivacy(t *testing.T) {
	adapter := newMobileAdapter()
	defer adapter.Close()
	done := make(chan error, 1)
	go func() { done <- adapter.WriteText(context.Background(), "mobile private body") }()
	deadline := time.Now().Add(time.Second)
	var action writeActionV1
	for time.Now().Before(deadline) {
		if value, ok := adapter.next(); ok {
			action = value
			break
		}
		time.Sleep(time.Millisecond)
	}
	if action.Text != "mobile private body" {
		t.Fatalf("write action was not delivered: %+v", action)
	}
	if err := adapter.complete(action.ActionID, "failed with mobile private body"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil || strings.Contains(err.Error(), action.Text) {
		t.Fatalf("platform error was not safely redacted: %v", err)
	}
}

func TestMobileAdapterQueuesAreBounded(t *testing.T) {
	adapter := newMobileAdapter()
	defer adapter.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for index := 0; index < actionQueueCapacity; index++ {
		go func(value int) { _ = adapter.WriteText(ctx, string(rune('a'+value%20))) }(index)
	}
	deadline := time.Now().Add(time.Second)
	count := 0
	for time.Now().Before(deadline) {
		adapter.mu.Lock()
		count = len(adapter.outstanding)
		adapter.mu.Unlock()
		if count == actionQueueCapacity {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if count != actionQueueCapacity {
		t.Fatalf("mobile write queue did not fill: %d", count)
	}
	if err := adapter.WriteText(context.Background(), "overflow"); err == nil {
		t.Fatal("mobile write queue exceeded its capacity")
	}
}
