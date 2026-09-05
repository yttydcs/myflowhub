package sdk

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
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
	EventSnapshot EventKind = "snapshot"
	EventData     EventKind = "data"
	EventGap      EventKind = "gap"
	EventExpired  EventKind = "expired"
)

type Event struct {
	Kind              EventKind
	Resource          protocol.ResourceID
	Capability        protocol.CapabilityID
	Schema            string
	Revision          uint64
	Sequence          uint64
	Publisher         protocol.NodeID
	PublisherSequence uint64
	GapFrom           uint64
	GapTo             uint64
	Value             []byte
	Reason            string
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
	runtime *node.Node
}

// NewAttachedClient creates an operation client for an existing node.
// The caller owns the node and its connections for the entire client lifetime.
func NewAttachedClient(runtime *node.Node) (*Client, error) {
	if runtime == nil {
		return nil, errors.New("SDK client requires a node runtime")
	}
	return &Client{runtime: runtime}, nil
}

// NodeID returns the identity of the node used by this operation client.
func (c *Client) NodeID() (protocol.NodeID, error) {
	runtime, err := c.runtimeNode()
	if err != nil {
		return 0, err
	}
	return runtime.ID(), nil
}

func (c *Client) Subscribe(ctx context.Context, resourceID protocol.ResourceID, lease time.Duration, queue int) (*Subscription, error) {
	return c.SubscribeCapability(ctx, resourceID, protocol.CapabilitySubscribe, lease, queue)
}

func (c *Client) SubscribeCapability(ctx context.Context, resourceID protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int) (*Subscription, error) {
	if ctx == nil {
		return nil, errors.New("SDK subscription context is required")
	}
	runtime, err := c.runtimeNode()
	if err != nil {
		return nil, err
	}
	remote, err := runtime.SubscribeCapability(ctx, resourceID, capability, lease, queue)
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
	result, err := c.Operate(ctx, resourceID, protocol.CapabilityRead, "", nil)
	if err != nil {
		return Event{}, err
	}
	return Event{Kind: EventSnapshot, Resource: resourceID, Capability: protocol.CapabilityRead, Schema: result.Schema, Value: result.Payload}, nil
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

func (c *Client) WriteVariable(ctx context.Context, resourceID protocol.ResourceID, expectedRevision uint64, value []byte) (resource.OperationResult, error) {
	request, err := protocol.EncodeJSONPayload(&protocol.VariableWriteV2{
		Version: protocol.SchemaVersionV2, ExpectedRevision: expectedRevision, Value: append([]byte(nil), value...),
	}, protocol.DefaultMaxPayload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	return c.Operate(ctx, resourceID, protocol.CapabilityWrite, protocol.SchemaVariableWriteV2, request)
}

func (c *Client) Catalog(ctx context.Context, owner protocol.NodeID) (protocol.ResourceCatalogV2, error) {
	var catalog protocol.ResourceCatalogV2
	err := c.DecodeSnapshot(ctx, protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog}, &catalog)
	return catalog, err
}

func (c *Client) Invoke(ctx context.Context, resourceID protocol.ResourceID, input []byte) ([]byte, error) {
	result, err := c.Operate(ctx, resourceID, protocol.CapabilityInvoke, "", input)
	return result.Payload, err
}

func (c *Client) Operate(ctx context.Context, resourceID protocol.ResourceID, capability protocol.CapabilityID, schema string, input []byte) (resource.OperationResult, error) {
	runtime, err := c.runtimeNode()
	if err != nil {
		return resource.OperationResult{}, err
	}
	output, err := runtime.Operate(ctx, resourceID, capability, schema, input)
	return output, wrapError(err)
}

// OperatePayload validates and encodes a typed request, performs one operation
// through the client's existing Node path, then validates and decodes the
// typed response. The descriptor remains authoritative for wire schemas, so
// the request schema is intentionally left empty.
func (c *Client) OperatePayload(ctx context.Context, resourceID protocol.ResourceID, capability protocol.CapabilityID, request, response protocol.ValidatedPayload) error {
	if ctx == nil {
		return errors.New("SDK operation context is required")
	}
	if c == nil {
		return errors.New("SDK client is closed")
	}
	if err := resourceID.Validate(); err != nil {
		return fmt.Errorf("SDK operation resource: %w", err)
	}
	if err := capability.Validate(); err != nil {
		return fmt.Errorf("SDK operation capability: %w", err)
	}
	if isNilValidatedPayload(request) || isNilValidatedPayload(response) {
		return errors.New("SDK operation request and response payloads are required")
	}
	input, err := protocol.EncodeJSONPayload(request, protocol.DefaultMaxPayload)
	if err != nil {
		return &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("encode %s %s request: %v", resourceID.Name, capability, err), cause: err}
	}
	output, err := c.Operate(ctx, resourceID, capability, "", input)
	if err != nil {
		return err
	}
	if err := protocol.DecodeJSONPayload(output.Payload, protocol.DefaultMaxPayload, response); err != nil {
		return &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("decode %s %s response: %v", resourceID.Name, capability, err), cause: err}
	}
	return nil
}

func isNilValidatedPayload(value protocol.ValidatedPayload) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func (c *Client) InvokePayload(ctx context.Context, resourceID protocol.ResourceID, request, response protocol.ValidatedPayload) error {
	return c.OperatePayload(ctx, resourceID, protocol.CapabilityInvoke, request, response)
}

func (c *Client) runtimeNode() (*node.Node, error) {
	if c == nil {
		return nil, errors.New("SDK client is closed")
	}
	runtime := c.runtime
	if runtime == nil {
		return nil, errors.New("SDK client is closed")
	}
	select {
	case <-runtime.Done():
		return nil, errors.New("SDK node is closed")
	default:
		return runtime, nil
	}
}

func convertEvent(event subscription.Event) Event {
	kind := EventExpired
	switch event.Kind {
	case subscription.EventSnapshot:
		kind = EventSnapshot
	case subscription.EventData:
		kind = EventData
	case subscription.EventGap:
		kind = EventGap
	}
	return Event{
		Kind: kind, Resource: event.Resource, Capability: event.Capability, Schema: event.Schema,
		Revision: event.Revision, Sequence: event.Sequence, Publisher: event.Publisher,
		PublisherSequence: event.PublisherSequence, GapFrom: event.GapFrom, GapTo: event.GapTo,
		Value: append([]byte(nil), event.Value...), Reason: event.Reason,
	}
}
