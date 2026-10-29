package desktop

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/sdk/bindings"
)

func batchClient() *Client {
	return &Client{core: new(bindings.Client), subscriptions: make(map[int64]*subscription), ready: make(chan struct{}, 1)}
}
func readBatch(t *testing.T, c *Client, raw string, wait int64) []SubscriptionPoll {
	t.Helper()
	result, err := c.PollSubscriptions(raw, wait)
	if err != nil {
		t.Fatal(err)
	}
	var items []SubscriptionPoll
	if err := json.Unmarshal([]byte(result), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) > MaxPollEvents || (len(items) > 1 && len(result) > MaxPollBatchBytes) {
		t.Fatalf("unbounded batch: %d events, %d bytes", len(items), len(result))
	}
	return items
}

func TestBatchPollFairlyDrains2000RealQueues(t *testing.T) {
	c := batchClient()
	ids := make([]int64, 2000)
	for i := range ids {
		id := int64(i + 1)
		ids[i] = id
		s := newSubscription()
		s.ready = c.ready
		c.subscriptions[id] = s
		s.OnEvent(`{"kind":"data","value":false}`)
	}
	raw, _ := json.Marshal(ids)
	seen := make(map[int64]bool)
	for len(seen) < len(ids) {
		items := readBatch(t, c, string(raw), 0)
		if len(items) == 0 {
			t.Fatal("active queues were skipped")
		}
		for _, item := range items {
			if seen[item.SubscriptionID] {
				t.Fatal("duplicate delivery")
			}
			seen[item.SubscriptionID] = true
		}
	}
	if items := readBatch(t, c, string(raw), 0); len(items) != 0 {
		t.Fatal("already consumed events repeated")
	}
}

func TestBatchPollRetainsDeferredLargeEventsAndPrioritizesErrors(t *testing.T) {
	c := batchClient()
	for _, id := range []int64{1, 2} {
		c.subscriptions[id] = newSubscription()
		c.subscriptions[id].OnEvent(`{"value":"` + strings.Repeat("x", 200<<10) + `"}`)
	}
	first := readBatch(t, c, `[1,2]`, 0)
	second := readBatch(t, c, `[1,2]`, 0)
	if len(first) != 1 || len(second) != 1 || first[0].SubscriptionID == second[0].SubscriptionID {
		t.Fatal("byte-limited event was lost or repeated")
	}
	c.subscriptions[1].OnEvent(`{"value":"` + strings.Repeat("x", 300<<10) + `"}`)
	if items := readBatch(t, c, `[1]`, 0); len(items) != 1 {
		t.Fatal("legal single event was starved by the soft byte budget")
	}
	c.subscriptions[1].OnEvent(`{"value":"old"}`)
	c.subscriptions[1].OnError(`{"code":"forbidden"}`)
	if items := readBatch(t, c, `[1]`, 0); len(items) != 1 || items[0].Kind != "error" {
		t.Fatal("stale data hid terminal permission error")
	}
}

func TestBatchPollValidatesInputsAndWakesForEvents(t *testing.T) {
	c := batchClient()
	s := newSubscription()
	s.ready = c.ready
	c.subscriptions[1] = s
	for _, raw := range []string{`null`, `[]`, `[0]`, `[1,1]`, `[1.5]`, `[9007199254740992]`, `[1] true`} {
		if _, err := c.PollSubscriptions(raw, 0); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, wait := range []int64{-1, 101, 1 << 62} {
		if _, err := c.PollSubscriptions(`[1]`, wait); err == nil {
			t.Fatal("invalid timeout")
		}
	}
	go func() { time.Sleep(5 * time.Millisecond); s.OnEvent(`{"value":true}`) }()
	if items := readBatch(t, c, `[1]`, 100); len(items) != 1 || items[0].Kind != "event" {
		t.Fatal("new event did not wake batch poll")
	}
}
