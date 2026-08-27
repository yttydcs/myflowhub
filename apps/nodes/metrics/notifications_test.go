package metrics

import (
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestNotificationInboxDropsOldestAtCapacity(t *testing.T) {
	inbox := &NotificationInbox{capacity: 2, queue: make([]protocol.NotificationEventV1, 0, 2)}
	for _, id := range []string{"one", "two", "three"} {
		inbox.enqueue(protocol.NotificationEventV1{EventID: id})
	}
	status := inbox.Status()
	if status.QueueDepth != 2 || status.Dropped != 1 {
		t.Fatalf("unexpected inbox status: %+v", status)
	}
	events := inbox.Dequeue()
	if len(events) != 2 || events[0].EventID != "two" || events[1].EventID != "three" {
		t.Fatalf("unexpected retained events: %+v", events)
	}
}
