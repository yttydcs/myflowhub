package sdk

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
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

func (c *Client) runDurableSubscription(ctx context.Context, status ConnectionStatus, resource protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int, events chan<- Event, errorsOut chan<- error, ready chan struct{}) {
	defer close(events)
	defer close(errorsOut)
	var readyOnce sync.Once
	var lastSequence uint64
	sawStream := false
	sawValue := false
	invalidate := func() bool {
		if !sawValue {
			return true
		}
		gap := Event{Kind: EventGap, Resource: resource, Reason: "subscription_interrupted_unknown_range"}
		if sawStream {
			gap.GapFrom = lastSequence + 1
		}
		select {
		case events <- gap:
			sawValue = false
			return true
		case <-ctx.Done():
			return false
		}
	}
	retryDelay := 100 * time.Millisecond
	for {
		// Renewal retires the old handle without an interruption event. If
		// its replacement cannot start, invalidate the last value before waiting.
		snapshot, err := waitConnected(ctx, status, invalidate)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				sendTerminalError(errorsOut, wrapError(err))
			}
			return
		}
		remote, err := c.SubscribeCapability(ctx, resource, capability, lease, queue)
		if err != nil {
			if permanentSubscriptionError(err) {
				sendTerminalError(errorsOut, wrapError(err))
				return
			}
			if !invalidate() {
				return
			}
			// A remote sibling can recover while this client's own parent stays
			// connected. Retry only subscription setup, with bounded backoff.
			retryCtx, stopRetry := context.WithTimeout(ctx, retryDelay)
			_, waitErr := status.WaitChange(retryCtx, snapshot.Generation)
			stopRetry()
			if waitErr != nil && !errors.Is(waitErr, context.Canceled) && !errors.Is(waitErr, context.DeadlineExceeded) {
				sendTerminalError(errorsOut, waitErr)
				return
			}
			if retryDelay < time.Second {
				retryDelay = min(retryDelay*2, time.Second)
			}
			continue
		}
		readyOnce.Do(func() { close(ready) })
		retryDelay = 100 * time.Millisecond
		// Spread a large watch set's renewals across 65..79% of its lease.
		hash := fnv.New32a()
		_, _ = fmt.Fprintf(hash, "%d/%s/%s", resource.Owner, resource.Name, capability)
		renewAfter := lease * time.Duration(65+hash.Sum32()%15) / 100
		if renewAfter <= 0 {
			renewAfter = lease
		}
		timer := time.NewTimer(renewAfter)
		attemptDone := false
		interrupted := false
		eventSource := remote.Events
		errorSource := remote.Errors
		for !attemptDone {
			select {
			case event, ok := <-eventSource:
				if !ok {
					// Subscribe closes Errors before Events. Drain a buffered
					// validation failure before considering this a reconnect.
					if errorSource != nil {
						for failure := range errorSource {
							if permanentSubscriptionError(failure) {
								remote.Cancel()
								timer.Stop()
								sendTerminalError(errorsOut, wrapError(failure))
								return
							}
						}
					}
					attemptDone = true
					interrupted = true
					eventSource = nil
					break
				}
				converted := event
				sawValue = true
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
					interrupted = true
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
		if interrupted && !invalidate() {
			return
		}
	}
}

func waitConnected(ctx context.Context, status ConnectionStatus, invalidate func() bool) (ConnectionSnapshot, error) {
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
		if !invalidate() {
			return ConnectionSnapshot{}, ctx.Err()
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
	var sdkError *Error
	if errors.As(err, &sdkError) {
		switch sdkError.Code {
		case protocol.CodeMalformed, protocol.CodeUnsupported, protocol.CodeForbidden, protocol.CodeNotFound, protocol.CodeConflict:
			return true
		}
	}
	var payload protocol.ErrorPayload
	if errors.As(err, &payload) {
		switch payload.Code {
		case protocol.CodeMalformed, protocol.CodeUnsupported, protocol.CodeForbidden, protocol.CodeNotFound, protocol.CodeConflict:
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
