package sdk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/node"
)

type DurableSubscription struct {
	Resource protocol.ResourceID
	Events   <-chan Event
	Errors   <-chan error
	Ready    <-chan struct{}
	cancel   context.CancelFunc
	once     sync.Once
}

func (s *DurableSubscription) Cancel() {
	if s != nil && s.cancel != nil {
		s.once.Do(s.cancel)
	}
}

func (c *Client) SubscribeDurable(ctx context.Context, supervisor *node.ParentSupervisor, resource protocol.ResourceID, lease time.Duration, queue int) (*DurableSubscription, error) {
	return c.SubscribeDurableCapability(ctx, supervisor, resource, protocol.CapabilitySubscribe, lease, queue)
}

func (c *Client) SubscribeDurableCapability(ctx context.Context, supervisor *node.ParentSupervisor, resource protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int) (*DurableSubscription, error) {
	if supervisor == nil {
		return nil, errors.New("durable subscription requires a parent supervisor")
	}
	return c.SubscribeDurableStatusCapability(ctx, runtimeConnectionStatus{supervisor: supervisor}, resource, capability, lease, queue)
}

// SubscribeDurableStatus keeps a subscription active across changes reported
// by a read-only Host-owned connection status.
func (c *Client) SubscribeDurableStatus(ctx context.Context, status ConnectionStatus, resource protocol.ResourceID, lease time.Duration, queue int) (*DurableSubscription, error) {
	return c.SubscribeDurableStatusCapability(ctx, status, resource, protocol.CapabilitySubscribe, lease, queue)
}

// SubscribeDurableStatusCapability is the non-owning connection-status form of
// durable subscription used by attached bindings.
func (c *Client) SubscribeDurableStatusCapability(ctx context.Context, status ConnectionStatus, resource protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int) (*DurableSubscription, error) {
	if ctx == nil {
		return nil, errors.New("durable subscription context is required")
	}
	if status == nil {
		return nil, errors.New("durable subscription requires parent connection status")
	}
	if _, err := c.runtimeNode(); err != nil {
		return nil, err
	}
	if err := resource.Validate(); err != nil {
		return nil, err
	}
	if err := capability.Validate(); err != nil {
		return nil, err
	}
	if lease <= 0 || queue < 1 || queue > 1024 {
		return nil, errors.New("durable subscription lease or queue is invalid")
	}
	runCtx, cancel := context.WithCancel(ctx)
	events := make(chan Event, queue)
	errorsOut := make(chan error, 1)
	ready := make(chan struct{})
	result := &DurableSubscription{Resource: resource, Events: events, Errors: errorsOut, Ready: ready, cancel: cancel}
	go c.runDurableSubscription(runCtx, status, resource, capability, lease, queue, events, errorsOut, ready)
	return result, nil
}

func (c *Client) SubscribeDurableConnection(ctx context.Context, connection *Connection, resource protocol.ResourceID, lease time.Duration, queue int) (*DurableSubscription, error) {
	return c.SubscribeDurableConnectionCapability(ctx, connection, resource, protocol.CapabilitySubscribe, lease, queue)
}

func (c *Client) SubscribeDurableConnectionCapability(ctx context.Context, connection *Connection, resource protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int) (*DurableSubscription, error) {
	if connection == nil || connection.supervisor == nil {
		return nil, errors.New("durable subscription requires a managed connection")
	}
	return c.SubscribeDurableStatusCapability(ctx, connection, resource, capability, lease, queue)
}

func (c *Client) runDurableSubscription(ctx context.Context, status ConnectionStatus, resource protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int, events chan<- Event, errorsOut chan<- error, ready chan struct{}) {
	defer close(events)
	defer close(errorsOut)
	var readyOnce sync.Once
	var lastSequence uint64
	sawStream := false
	recovering := false
	for {
		snapshot, err := waitConnected(ctx, status)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				sendTerminalError(errorsOut, wrapError(err))
			}
			return
		}
		if recovering && sawStream {
			gap := Event{Kind: EventGap, Resource: resource, GapFrom: lastSequence + 1, GapTo: 0, Reason: "connection_recovered_unknown_range"}
			select {
			case events <- gap:
			case <-ctx.Done():
				return
			}
		}
		runtime, runtimeErr := c.runtimeNode()
		if runtimeErr != nil {
			sendTerminalError(errorsOut, wrapError(runtimeErr))
			return
		}
		remote, err := runtime.SubscribeCapability(ctx, resource, capability, lease, queue)
		if err != nil {
			if permanentSubscriptionError(err) {
				sendTerminalError(errorsOut, wrapError(err))
				return
			}
			recovering = true
			if _, waitErr := status.WaitChange(ctx, snapshot.Generation); waitErr != nil && !errors.Is(waitErr, context.Canceled) {
				sendTerminalError(errorsOut, waitErr)
				return
			}
			continue
		}
		readyOnce.Do(func() { close(ready) })
		recovering = false
		renewAfter := lease * 4 / 5
		if renewAfter <= 0 {
			renewAfter = lease
		}
		timer := time.NewTimer(renewAfter)
		attemptDone := false
		eventSource := remote.Events
		errorSource := remote.Errors
		for !attemptDone {
			select {
			case event, ok := <-eventSource:
				if !ok {
					attemptDone = true
					eventSource = nil
					break
				}
				converted := convertEvent(event)
				if converted.Kind == EventData && converted.Sequence != 0 {
					sawStream = true
					lastSequence = converted.Sequence
				}
				select {
				case events <- converted:
				case <-ctx.Done():
					remote.Cancel()
					timer.Stop()
					return
				}
			case err, ok := <-errorSource:
				if !ok {
					errorSource = nil
					continue
				}
				if ok && err != nil && permanentSubscriptionError(err) {
					remote.Cancel()
					timer.Stop()
					sendTerminalError(errorsOut, wrapError(err))
					return
				}
				if err != nil {
					remote.Cancel()
					attemptDone = true
				}
			case <-timer.C:
				remote.Cancel()
				attemptDone = true
			case <-ctx.Done():
				remote.Cancel()
				timer.Stop()
				return
			}
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		remote.Cancel()
		recovering = sawStream || status.Snapshot().State != ConnectionConnected
	}
}

func waitConnected(ctx context.Context, status ConnectionStatus) (ConnectionSnapshot, error) {
	for {
		current := status.Snapshot()
		switch current.State {
		case ConnectionConnected:
			return current, nil
		case ConnectionFailed:
			return ConnectionSnapshot{}, fmt.Errorf("parent connection failed: %s", current.LastError)
		case ConnectionStopped:
			return ConnectionSnapshot{}, errors.New("parent connection supervisor stopped")
		}
		next, err := status.WaitChange(ctx, current.Generation)
		if err != nil {
			return ConnectionSnapshot{}, err
		}
		if next.Generation <= current.Generation {
			return ConnectionSnapshot{}, errors.New("parent supervisor generation did not advance")
		}
	}
}

type runtimeConnectionStatus struct {
	supervisor *node.ParentSupervisor
}

func (s runtimeConnectionStatus) Snapshot() ConnectionSnapshot {
	return convertConnection(s.supervisor.Snapshot())
}

func (s runtimeConnectionStatus) WaitChange(ctx context.Context, after uint64) (ConnectionSnapshot, error) {
	value, err := s.supervisor.WaitChange(ctx, after)
	if err != nil {
		return ConnectionSnapshot{}, err
	}
	return convertConnection(value), nil
}

func permanentSubscriptionError(err error) bool {
	var payload protocol.ErrorPayload
	if errors.As(err, &payload) {
		switch payload.Code {
		case protocol.CodeMalformed, protocol.CodeForbidden, protocol.CodeNotFound, protocol.CodeConflict:
			return true
		}
	}
	return false
}

func sendTerminalError(target chan<- error, err error) {
	select {
	case target <- err:
	default:
	}
}
