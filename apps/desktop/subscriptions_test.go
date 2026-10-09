package main

import "testing"

type testSubscriptionClient struct {
	label     string
	cancelled []int64
}

func (c *testSubscriptionClient) PollSubscription(id, timeout int64) (string, error) {
	return c.label, nil
}
func (c *testSubscriptionClient) CancelSubscription(id int64) { c.cancelled = append(c.cancelled, id) }

func TestDesktopSubscriptionsKeepOriginalProfileClient(t *testing.T) {
	var sessions desktopSubscriptions
	old := &testSubscriptionClient{label: "old"}
	current := &testSubscriptionClient{label: "current"}
	first, err := sessions.add(old, 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := sessions.add(current, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("client-local IDs collided")
	}
	if value, err := sessions.poll(first, 10); err != nil || value != "old" {
		t.Fatalf("old poll used wrong client: %s %v", value, err)
	}
	sessions.cancel(first)
	sessions.cancel(first)
	if len(old.cancelled) != 1 || len(current.cancelled) != 0 {
		t.Fatal("old cancellation reached the current Profile")
	}
	if _, err := sessions.poll(first, 10); err == nil {
		t.Fatal("cancelled subscription remained readable")
	}
	sessions.close()
	if len(current.cancelled) != 1 || len(sessions.entries) != 0 {
		t.Fatal("remaining subscription was not released")
	}
}

func TestDesktopSubscriptionsBoundedAndReusable(t *testing.T) {
	var sessions desktopSubscriptions
	client := &testSubscriptionClient{}
	for i := 0; i < 256; i++ {
		if _, err := sessions.add(client, int64(i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sessions.add(client, 999); err == nil {
		t.Fatal("unbounded subscription cache")
	}
	sessions.close()
	if _, err := sessions.add(client, 999); err != nil {
		t.Fatal(err)
	}
}
