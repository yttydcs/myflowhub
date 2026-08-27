package clipboard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestControllerPublishesBodyOnlyInProtectedEventAndHistory(t *testing.T) {
	controller, runtime, _ := newClipboardController(t, 2, nil)
	defer runtime.Close()
	defer controller.Close()
	enableClipboard(t, controller, false, false, nil)
	secret := "private clipboard body"
	streamEvents := make(chan resource.StreamEvent, 1)
	cancelWatch, err := clipboardStream(t, runtime).Watch(func(event resource.StreamEvent) { streamEvents <- event })
	if err != nil {
		t.Fatal(err)
	}
	defer cancelWatch()
	decision, err := controller.SendText(context.Background(), secret)
	if err != nil || decision.Action != "published" {
		t.Fatalf("unexpected send decision: %+v (%v)", decision, err)
	}
	if snapshot := <-streamEvents; !strings.Contains(string(snapshot.Value), secret) {
		t.Fatalf("stream event omitted clipboard body: %s", snapshot.Value)
	}
	status := resourceJSON(t, runtime, ResourceStatus)
	config := resourceJSON(t, runtime, ResourceConfig)
	if strings.Contains(status, secret) || strings.Contains(config, secret) {
		t.Fatalf("clipboard body leaked into status/config: status=%s config=%s", status, config)
	}
	history := controller.History()
	if len(history) != 1 || history[0].Text != secret {
		t.Fatalf("unexpected local history: %+v", history)
	}
}

func TestControllerValidatesOversizeDedupePendingApplyAndLoopSuppression(t *testing.T) {
	adapter := newFakeAdapter()
	controller, runtime, _ := newClipboardController(t, 2, adapter)
	defer runtime.Close()
	defer controller.Close()
	enableClipboard(t, controller, false, false, []PeerV1{{NodeID: "3", Receive: true}})
	if _, err := controller.SendText(context.Background(), strings.Repeat("x", 64<<10+1)); err == nil {
		t.Fatal("configured oversize clipboard value was accepted")
	}
	event := clipboardEvent(3, "remote secret", time.Now())
	decision, err := controller.Receive(context.Background(), 3, event)
	if err != nil || decision.Action != "pending" || controller.Status().PendingCount != 1 {
		t.Fatalf("unexpected receive decision: %+v (%v)", decision, err)
	}
	duplicate, err := controller.Receive(context.Background(), 3, event)
	if err != nil || duplicate.Action != "ignored" {
		t.Fatalf("duplicate was not ignored: %+v (%v)", duplicate, err)
	}
	applied, err := controller.ApplyPending(context.Background(), event.EventID)
	if err != nil || applied.Action != "applied" || adapter.text() != event.Text {
		t.Fatalf("pending body was not applied: %+v (%v)", applied, err)
	}
	suppressed, err := controller.SendText(context.Background(), event.Text)
	if err != nil || suppressed.Action != "ignored" || !strings.Contains(suppressed.Reason, "recently applied") {
		t.Fatalf("clipboard write-back loop was not suppressed: %+v (%v)", suppressed, err)
	}
}

func TestHistoryRetentionIsBoundedDurableAndClearable(t *testing.T) {
	controller, runtime, store := newClipboardController(t, 2, nil)
	enableClipboardWithHistory(t, controller, "metadata", 2, 1024, time.Hour)
	for _, text := range []string{"one", "two", "three"} {
		if _, err := controller.SendText(context.Background(), text); err != nil {
			t.Fatal(err)
		}
	}
	history := controller.History()
	if len(history) != 2 || history[0].Text != "" || history[1].Text != "" {
		t.Fatalf("metadata history was not bounded/redacted: %+v", history)
	}
	if err := controller.Close(); err != nil {
		t.Fatal(err)
	}
	_ = runtime.Close()
	restarted := newClipboardNode(t, 2)
	defer restarted.Close()
	loaded, err := Register(ControllerConfig{Node: restarted, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	defer loaded.Close()
	if len(loaded.History()) != 2 {
		t.Fatalf("history did not survive restart: %+v", loaded.History())
	}
	input, _ := protocol.EncodeJSONPayload(&ClearV1{Version: 1}, protocol.DefaultMaxPayload)
	value, _ := restarted.Registry().Resolve(protocol.ResourceID{Owner: 2, Name: CommandHistoryClear})
	result, err := value.(*resource.Command).Invoke(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	var cleared ClearResultV1
	if err := protocol.DecodeJSONPayload(result, protocol.DefaultMaxPayload, &cleared); err != nil || cleared.Removed != 2 || len(loaded.History()) != 0 {
		t.Fatalf("history clear failed: %+v (%v)", cleared, err)
	}
}

func TestPendingQueueAndHistoryTTLRemainBounded(t *testing.T) {
	controller, runtime, _ := newClipboardController(t, 2, newFakeAdapter())
	defer runtime.Close()
	defer controller.Close()
	enableClipboard(t, controller, false, false, []PeerV1{{NodeID: "3", Receive: true}})
	var firstEventID string
	for index := 0; index <= MaxPendingEvents; index++ {
		event := clipboardEvent(3, fmt.Sprintf("pending-%d", index), time.Now())
		if index == 0 {
			firstEventID = event.EventID
		}
		if decision, err := controller.Receive(context.Background(), 3, event); err != nil || decision.Action != "pending" {
			t.Fatalf("receive pending event %d: %+v (%v)", index, decision, err)
		}
	}
	if controller.Status().PendingCount != MaxPendingEvents {
		t.Fatalf("pending queue exceeded its bound: %+v", controller.Status())
	}
	if _, err := controller.ApplyPending(context.Background(), firstEventID); err == nil {
		t.Fatal("oldest pending event was not evicted")
	}

	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.HistoryTTLMS = 1000
	now := time.Now().UTC()
	history, err := loadHistory(store, config, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.add(config, clipboardEvent(3, "expires", now), "received", now); err != nil {
		t.Fatal(err)
	}
	if err := history.normalize(config, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(history.snapshot()) != 0 {
		t.Fatalf("expired clipboard history was retained: %+v", history.snapshot())
	}
}

func TestWatcherPublishesWithoutLeakingAdapterErrors(t *testing.T) {
	adapter := newFakeAdapter()
	controller, runtime, _ := newClipboardController(t, 2, adapter)
	defer runtime.Close()
	defer controller.Close()
	enableClipboard(t, controller, true, false, nil)
	adapter.observe("watched private body")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && len(controller.History()) == 0 {
		time.Sleep(time.Millisecond)
	}
	if len(controller.History()) != 1 {
		t.Fatal("watcher did not publish observed clipboard body")
	}
	adapter.fail(errors.New("platform failed while handling watched private body"))
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) && controller.Status().LastError == "" {
		time.Sleep(time.Millisecond)
	}
	if strings.Contains(controller.Status().LastError, "watched private body") {
		t.Fatalf("clipboard body leaked through adapter error: %q", controller.Status().LastError)
	}
}

type fakeAdapter struct {
	mu     sync.Mutex
	value  string
	events chan TextObservation
	errors chan error
}

func newFakeAdapter() *fakeAdapter {
	return &fakeAdapter{events: make(chan TextObservation, 8), errors: make(chan error, 8)}
}

func (a *fakeAdapter) ReadText(context.Context) (string, error) { return a.text(), nil }
func (a *fakeAdapter) WriteText(_ context.Context, value string) error {
	a.mu.Lock()
	a.value = value
	a.mu.Unlock()
	return nil
}
func (a *fakeAdapter) WatchText(context.Context) (<-chan TextObservation, <-chan error, error) {
	return a.events, a.errors, nil
}
func (a *fakeAdapter) Close() error { return nil }
func (a *fakeAdapter) text() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.value
}
func (a *fakeAdapter) observe(value string) {
	a.events <- TextObservation{Text: value, ObservedAt: time.Now()}
}
func (a *fakeAdapter) fail(err error) { a.errors <- err }

func newClipboardController(t *testing.T, id protocol.NodeID, adapter Adapter) (*Controller, *node.Node, *keystore.Store) {
	t.Helper()
	runtime := newClipboardNode(t, id)
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	controller, err := Register(ControllerConfig{Node: runtime, Store: store, Adapter: adapter})
	if err != nil {
		_ = runtime.Close()
		t.Fatal(err)
	}
	return controller, runtime, store
}

func newClipboardNode(t *testing.T, id protocol.NodeID) *node.Node {
	t.Helper()
	identity, err := auth.GenerateIdentity(id)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	_ = trust.Add(id, identity.PublicKey)
	runtime, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func enableClipboard(t *testing.T, controller *Controller, watch, apply bool, peers []PeerV1) {
	t.Helper()
	config := controller.Config()
	_, err := controller.ApplyConfig(context.Background(), ConfigUpdateV1{
		Version: 1, ExpectedRevision: config.Revision, Enabled: true, MaxInlineBytes: config.MaxInlineBytes,
		AutoWatch: watch, AutoApply: apply, HistoryRetention: config.HistoryRetention, HistoryLimit: config.HistoryLimit,
		HistoryMaxBytes: config.HistoryMaxBytes, HistoryTTLMS: config.HistoryTTLMS, Peers: peers,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func enableClipboardWithHistory(t *testing.T, controller *Controller, retention string, limit, bytes int, ttl time.Duration) {
	t.Helper()
	config := controller.Config()
	_, err := controller.ApplyConfig(context.Background(), ConfigUpdateV1{
		Version: 1, ExpectedRevision: config.Revision, Enabled: true, MaxInlineBytes: config.MaxInlineBytes,
		HistoryRetention: retention, HistoryLimit: limit, HistoryMaxBytes: bytes, HistoryTTLMS: ttl.Milliseconds(), Peers: []PeerV1{},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func clipboardEvent(origin protocol.NodeID, text string, now time.Time) TextEventV1 {
	id, _ := protocol.NewMessageID()
	return TextEventV1{
		Version: 1, EventID: id.String(), OriginNodeID: formatNodeID(origin), CreatedAtUnixMS: now.UnixMilli(),
		ContentType: "text/plain; charset=utf-8", Text: text, SizeBytes: len([]byte(text)), SHA256: textHash(text),
	}
}

func clipboardStream(t *testing.T, runtime *node.Node) *resource.Stream {
	t.Helper()
	value, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: ResourceEvents})
	if !ok {
		t.Fatal("clipboard event stream is missing")
	}
	return value.(*resource.Stream)
}

func resourceJSON(t *testing.T, runtime *node.Node, name string) string {
	t.Helper()
	value, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: name})
	if !ok {
		t.Fatalf("clipboard resource %s is missing", name)
	}
	return string(value.(*resource.Variable).Snapshot().Value)
}
