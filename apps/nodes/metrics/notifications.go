package metrics

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

const defaultNotificationCapacity = 64

type NotificationStatus struct {
	QueueDepth int    `json:"queue_depth"`
	Dropped    uint64 `json:"dropped"`
	LastError  string `json:"last_error,omitempty"`
}

type NotificationInbox struct {
	client     *sdk.Client
	connection *sdk.Connection
	parent     protocol.NodeID
	controller *Controller
	capacity   int

	mu        sync.Mutex
	queue     []protocol.NotificationEventV1
	dropped   uint64
	lastError string
	cancel    context.CancelFunc
	done      chan struct{}
	closeOnce sync.Once
}

func startNotificationInbox(ctx context.Context, client *sdk.Client, connection *sdk.Connection, parent protocol.NodeID, controller *Controller, capacity int) (*NotificationInbox, error) {
	if ctx == nil || client == nil || connection == nil || controller == nil {
		return nil, errors.New("metrics notification inbox dependencies are required")
	}
	if capacity == 0 {
		capacity = defaultNotificationCapacity
	}
	if capacity < 1 || capacity > 1024 {
		return nil, errors.New("metrics notification inbox capacity must be between 1 and 1024")
	}
	runCtx, cancel := context.WithCancel(ctx)
	value := &NotificationInbox{
		client: client, connection: connection, parent: parent, controller: controller, capacity: capacity,
		queue: make([]protocol.NotificationEventV1, 0, capacity), cancel: cancel, done: make(chan struct{}),
	}
	go value.run(runCtx)
	return value, nil
}

func (i *NotificationInbox) Status() NotificationStatus {
	if i == nil {
		return NotificationStatus{}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	return NotificationStatus{QueueDepth: len(i.queue), Dropped: i.dropped, LastError: i.lastError}
}

func (i *NotificationInbox) Dequeue() []protocol.NotificationEventV1 {
	if i == nil {
		return nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	result := append([]protocol.NotificationEventV1(nil), i.queue...)
	i.queue = i.queue[:0]
	return result
}

func (i *NotificationInbox) Next() (protocol.NotificationEventV1, bool) {
	if i == nil {
		return protocol.NotificationEventV1{}, false
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if len(i.queue) == 0 {
		return protocol.NotificationEventV1{}, false
	}
	result := i.queue[0]
	copy(i.queue, i.queue[1:])
	i.queue = i.queue[:len(i.queue)-1]
	return result, true
}

func (i *NotificationInbox) Close() {
	if i == nil {
		return
	}
	i.closeOnce.Do(func() {
		i.cancel()
		<-i.done
	})
}

func (i *NotificationInbox) run(ctx context.Context) {
	defer close(i.done)
	for {
		config := i.controller.Config()
		if len(config.NotificationChannels) == 0 {
			if !waitNotificationRetry(ctx, 250*time.Millisecond) {
				return
			}
			continue
		}
		channels := make(map[string]struct{}, len(config.NotificationChannels))
		for _, channel := range config.NotificationChannels {
			channels[channel] = struct{}{}
		}
		subscription, err := i.client.SubscribeDurableConnection(ctx, i.connection, protocol.ResourceID{
			Owner: i.parent, Name: protocol.BuiltinNotificationEvents,
		}, 30*time.Second, i.capacity)
		if err != nil {
			i.setError(err)
			if !waitNotificationRetry(ctx, time.Second) {
				return
			}
			continue
		}
		revision := config.Revision
		retry := i.consume(ctx, subscription, revision, channels)
		subscription.Cancel()
		if !retry {
			return
		}
	}
}

func (i *NotificationInbox) consume(ctx context.Context, subscription *sdk.DurableSubscription, revision uint64, channels map[string]struct{}) bool {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case event, ok := <-subscription.Events:
			if !ok {
				return ctx.Err() == nil
			}
			if event.Kind == sdk.EventGap {
				i.setError(fmt.Errorf("notification stream gap: %s", event.Reason))
				continue
			}
			if event.Kind != sdk.EventData {
				continue
			}
			var notification protocol.NotificationEventV1
			if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &notification); err != nil {
				i.setError(fmt.Errorf("decode notification event: %w", err))
				continue
			}
			if _, enabled := channels[notification.Channel]; enabled {
				i.enqueue(notification)
			}
		case err, ok := <-subscription.Errors:
			if ok && err != nil {
				i.setError(err)
			}
			return ctx.Err() == nil
		case <-ticker.C:
			if i.controller.Config().Revision != revision {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
}

func (i *NotificationInbox) enqueue(event protocol.NotificationEventV1) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if len(i.queue) == i.capacity {
		copy(i.queue, i.queue[1:])
		i.queue = i.queue[:len(i.queue)-1]
		i.dropped++
	}
	i.queue = append(i.queue, event)
	i.lastError = ""
}

func (i *NotificationInbox) setError(err error) {
	if err == nil {
		return
	}
	i.mu.Lock()
	i.lastError = truncateError(err.Error())
	i.mu.Unlock()
}

func waitNotificationRetry(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
