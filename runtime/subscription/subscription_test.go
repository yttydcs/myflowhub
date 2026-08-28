package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func newManager(t *testing.T, typeID protocol.ResourceTypeID, lease time.Duration) (*Manager, resource.Resource, protocol.ResourceID) {
	t.Helper()
	id := protocol.ResourceID{Owner: 1, Name: "test"}
	registry, _ := resource.NewRegistry(1)
	var value resource.Resource
	switch typeID {
	case protocol.ResourceTypeVariable:
		value, _ = resource.NewVariable(resource.VariableDescriptor(id, "application/octet-stream", "test.raw.v1", "test.read", 64), []byte("initial"))
	case protocol.ResourceTypeStream:
		value, _ = resource.NewStream(resource.StreamDescriptor(id, "application/octet-stream", "test.raw.v1", "test.read", 64))
	}
	if err := registry.Register(value); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(context.Background(), registry, Config{MinLease: time.Millisecond, MaxLease: time.Minute, DefaultQueue: 1, MaxQueue: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	return manager, value, id
}

func request(resourceID protocol.ResourceID, lease time.Duration, queue int) Request {
	return Request{Subscriber: 2, Resource: resourceID, Capability: protocol.CapabilitySubscribe, LinkID: "child-2", Lease: lease, Queue: queue, TopologyEpoch: 1, PolicyGeneration: 1}
}

func receive(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("subscription closed")
		}
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscription event")
		return Event{}
	}
}

func TestVariableSnapshotBeforeCoalescedChange(t *testing.T) {
	manager, value, id := newManager(t, protocol.ResourceTypeVariable, time.Second)
	subscription, err := manager.Subscribe(request(id, time.Second, 1))
	if err != nil {
		t.Fatal(err)
	}
	variable := value.(*resource.Variable)
	for _, next := range []string{"one", "two", "three"} {
		if _, err := variable.Set([]byte(next)); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := receive(t, subscription.Events)
	if snapshot.Kind != EventSnapshot || string(snapshot.Value) != "initial" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	update := receive(t, subscription.Events)
	if update.Kind != EventData || string(update.Value) != "three" || update.Revision != 4 {
		t.Fatalf("unexpected coalesced update: %#v", update)
	}
}

func TestSlowStreamEmitsGap(t *testing.T) {
	manager, value, id := newManager(t, protocol.ResourceTypeStream, time.Second)
	subscription, err := manager.Subscribe(request(id, time.Second, 1))
	if err != nil {
		t.Fatal(err)
	}
	stream := value.(*resource.Stream)
	for i := 0; i < 20; i++ {
		if _, err := stream.Publish([]byte{byte(i)}); err != nil {
			t.Fatal(err)
		}
	}
	foundGap := false
	deadline := time.After(time.Second)
	for !foundGap {
		select {
		case event := <-subscription.Events:
			if event.Kind == EventGap {
				foundGap = true
				if event.GapFrom == 0 || event.GapTo < event.GapFrom {
					t.Fatalf("invalid gap: %#v", event)
				}
			}
		case <-deadline:
			t.Fatal("stream gap was not observable")
		}
	}
}

func TestLeaseExpiryAndLinkCleanup(t *testing.T) {
	manager, _, id := newManager(t, protocol.ResourceTypeVariable, 20*time.Millisecond)
	subscription, err := manager.Subscribe(request(id, 20*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	_ = receive(t, subscription.Events)
	expired := receive(t, subscription.Events)
	if expired.Kind != EventExpired {
		t.Fatalf("expected expiry, got %#v", expired)
	}
	select {
	case _, ok := <-subscription.Events:
		if ok {
			t.Fatal("events remained open after expiry")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not close")
	}
	second, err := manager.Subscribe(request(id, time.Second, 2))
	if err != nil {
		t.Fatal(err)
	}
	_ = receive(t, second.Events)
	if got := manager.CleanupLink("child-2"); got != 1 {
		t.Fatalf("want one cleanup, got %d", got)
	}
	select {
	case _, ok := <-second.Events:
		if ok {
			t.Fatal("link-bound subscription remained open")
		}
	case <-time.After(time.Second):
		t.Fatal("link cleanup did not close subscription")
	}
}

func TestInterestAggregationKeepsIndependentBoundaries(t *testing.T) {
	table, _ := NewInterestTable(4)
	resourceID := protocol.ResourceID{Owner: 1, Name: "state"}
	now := time.Now()
	one := Interest{ID: protocol.MustMessageID(), Subscriber: 2, Resource: resourceID, Capability: protocol.CapabilitySubscribe, LinkID: "a", LeaseUntil: now.Add(time.Minute), AuthorizedUntil: now.Add(time.Minute), TopologyEpoch: 1}
	two := Interest{ID: protocol.MustMessageID(), Subscriber: 3, Resource: resourceID, Capability: protocol.CapabilitySubscribe, LinkID: "b", LeaseUntil: now.Add(2 * time.Minute), AuthorizedUntil: now.Add(2 * time.Minute), TopologyEpoch: 2}
	if first, err := table.Add(one); err != nil || !first {
		t.Fatalf("first interest: %v %v", first, err)
	}
	if first, err := table.Add(two); err != nil || first {
		t.Fatalf("aggregated interest: %v %v", first, err)
	}
	interests := table.Interests(resourceID)
	if len(interests) != 2 || interests[0].Subscriber == interests[1].Subscriber {
		t.Fatalf("authorization boundaries were lost: %#v", interests)
	}
}

func TestMultipleSubscribersReceiveIndependentUpdates(t *testing.T) {
	manager, value, id := newManager(t, protocol.ResourceTypeVariable, time.Second)
	one, err := manager.Subscribe(request(id, time.Second, 2))
	if err != nil {
		t.Fatal(err)
	}
	twoRequest := request(id, time.Second, 2)
	twoRequest.Subscriber = 3
	twoRequest.LinkID = "child-3"
	two, err := manager.Subscribe(twoRequest)
	if err != nil {
		t.Fatal(err)
	}
	_ = receive(t, one.Events)
	_ = receive(t, two.Events)
	_, _ = value.(*resource.Variable).Set([]byte("shared"))
	if got := receive(t, one.Events); string(got.Value) != "shared" {
		t.Fatalf("first subscriber got %#v", got)
	}
	if got := receive(t, two.Events); string(got.Value) != "shared" {
		t.Fatalf("second subscriber got %#v", got)
	}
}
