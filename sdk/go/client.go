package sdk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

type Error struct {
	Code      protocol.ErrorCode
	Message   string
	Retryable bool
	Details   map[string]string
	cause     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func wrapError(err error) error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return err
	}
	var remote protocol.ErrorPayload
	if errors.As(err, &remote) {
		details := make(map[string]string, len(remote.Details))
		for key, value := range remote.Details {
			details[key] = value
		}
		return &Error{Code: remote.Code, Message: remote.Message, Retryable: remote.Retryable, Details: details, cause: err}
	}
	return &Error{Message: err.Error(), cause: err}
}

type EventKind string

const (
	EventVariableSnapshot EventKind = "variable_snapshot"
	EventVariableUpdate   EventKind = "variable_update"
	EventStream           EventKind = "stream"
	EventStreamGap        EventKind = "stream_gap"
	EventExpired          EventKind = "expired"
)

type Event struct {
	Kind     EventKind
	Resource protocol.ResourceID
	Revision uint64
	Sequence uint64
	GapFrom  uint64
	GapTo    uint64
	Value    []byte
	Reason   string
}

type Subscription struct {
	Resource protocol.ResourceID
	Events   <-chan Event
	Errors   <-chan error
	cancel   func()
	once     sync.Once
}

func (s *Subscription) Cancel() {
	if s != nil && s.cancel != nil {
		s.once.Do(s.cancel)
	}
}

type Client struct {
	mu      sync.RWMutex
	runtime *node.Node
}

func NewClient(runtime *node.Node) (*Client, error) {
	if runtime == nil {
		return nil, errors.New("SDK client requires a node runtime")
	}
	return &Client{runtime: runtime}, nil
}

func (c *Client) Connect(ctx context.Context, driver link.Driver, endpoint link.Endpoint, parent protocol.NodeID) error {
	runtime, err := c.runtimeNode()
	if err != nil {
		return err
	}
	return wrapError(runtime.ConnectParent(ctx, driver, endpoint, parent))
}

func (c *Client) Subscribe(ctx context.Context, resourceID protocol.ResourceID, lease time.Duration, queue int) (*Subscription, error) {
	if ctx == nil {
		return nil, errors.New("SDK subscription context is required")
	}
	runtime, err := c.runtimeNode()
	if err != nil {
		return nil, err
	}
	remote, err := runtime.Subscribe(ctx, resourceID, lease, queue)
	if err != nil {
		return nil, wrapError(err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	events := make(chan Event, queue)
	errorsOut := make(chan error, 1)
	result := &Subscription{Resource: resourceID, Events: events, Errors: errorsOut, cancel: func() {
		cancel()
		remote.Cancel()
	}}
	go func() {
		defer close(events)
		defer close(errorsOut)
		eventSource := remote.Events
		errorSource := remote.Errors
		for eventSource != nil || errorSource != nil {
			select {
			case event, ok := <-eventSource:
				if !ok {
					eventSource = nil
					continue
				}
				converted := convertEvent(event)
				select {
				case events <- converted:
				case <-runCtx.Done():
					remote.Cancel()
					return
				}
			case err, ok := <-errorSource:
				if !ok {
					errorSource = nil
					continue
				}
				if err != nil {
					select {
					case errorsOut <- wrapError(err):
					case <-runCtx.Done():
						remote.Cancel()
						return
					}
				}
			case <-runCtx.Done():
				remote.Cancel()
				return
			}
		}
	}()
	return result, nil
}

func (c *Client) Snapshot(ctx context.Context, resourceID protocol.ResourceID) (Event, error) {
	if ctx == nil {
		return Event{}, errors.New("SDK snapshot context is required")
	}
	current, err := c.Subscribe(ctx, resourceID, time.Minute, 4)
	if err != nil {
		return Event{}, err
	}
	defer current.Cancel()
	events := current.Events
	errorsOut := current.Errors
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return Event{}, errors.New("SDK snapshot subscription closed")
			}
			if event.Kind != EventVariableSnapshot {
				return Event{}, fmt.Errorf("resource %s is not a Variable", resourceID.Name)
			}
			return event, nil
		case err, ok := <-errorsOut:
			if !ok {
				errorsOut = nil
			} else if err != nil {
				return Event{}, err
			}
		case <-ctx.Done():
			return Event{}, wrapError(ctx.Err())
		}
	}
}

func (c *Client) DecodeSnapshot(ctx context.Context, resourceID protocol.ResourceID, target protocol.ValidatedPayload) error {
	if target == nil {
		return errors.New("SDK snapshot target is required")
	}
	event, err := c.Snapshot(ctx, resourceID)
	if err != nil {
		return err
	}
	if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, target); err != nil {
		return &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("decode %s snapshot: %v", resourceID.Name, err), cause: err}
	}
	return nil
}

func (c *Client) Catalog(ctx context.Context, owner protocol.NodeID) (protocol.ResourceCatalogV1, error) {
	var catalog protocol.ResourceCatalogV1
	err := c.DecodeSnapshot(ctx, protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog}, &catalog)
	return catalog, err
}

func (c *Client) Invoke(ctx context.Context, resourceID protocol.ResourceID, input []byte) ([]byte, error) {
	runtime, err := c.runtimeNode()
	if err != nil {
		return nil, err
	}
	output, err := runtime.Invoke(ctx, resourceID, input)
	return output, wrapError(err)
}

func (c *Client) InvokePayload(ctx context.Context, resourceID protocol.ResourceID, request, response protocol.ValidatedPayload) error {
	if request == nil || response == nil {
		return errors.New("SDK request and response payloads are required")
	}
	input, err := protocol.EncodeJSONPayload(request, protocol.DefaultMaxPayload)
	if err != nil {
		return &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("encode %s request: %v", resourceID.Name, err), cause: err}
	}
	output, err := c.Invoke(ctx, resourceID, input)
	if err != nil {
		return err
	}
	if err := protocol.DecodeJSONPayload(output, protocol.DefaultMaxPayload, response); err != nil {
		return &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("decode %s response: %v", resourceID.Name, err), cause: err}
	}
	return nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	runtime := c.runtime
	c.runtime = nil
	c.mu.Unlock()
	if runtime == nil {
		return nil
	}
	return runtime.Close()
}

func (c *Client) runtimeNode() (*node.Node, error) {
	if c == nil {
		return nil, errors.New("SDK client is closed")
	}
	c.mu.RLock()
	runtime := c.runtime
	c.mu.RUnlock()
	if runtime == nil {
		return nil, errors.New("SDK client is closed")
	}
	return runtime, nil
}

func convertEvent(event subscription.Event) Event {
	kind := EventExpired
	switch event.Kind {
	case subscription.EventVariableSnapshot:
		kind = EventVariableSnapshot
	case subscription.EventVariableUpdate:
		kind = EventVariableUpdate
	case subscription.EventStream:
		kind = EventStream
	case subscription.EventStreamGap:
		kind = EventStreamGap
	}
	return Event{Kind: kind, Resource: event.Resource, Revision: event.Revision, Sequence: event.Sequence, GapFrom: event.GapFrom, GapTo: event.GapTo, Value: append([]byte(nil), event.Value...), Reason: event.Reason}
}
