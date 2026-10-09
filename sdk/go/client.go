package sdk

import (
	"bytes"
	"context"
	"encoding/json"
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
	return c.subscribe(ctx, resourceID, capability, lease, queue, false)
}

// SubscribeVariable requires a standard variable snapshot before any data.
// Events keep their full wire envelopes; the definition cache is subscription-local.
func (c *Client) SubscribeVariable(ctx context.Context, resourceID protocol.ResourceID, lease time.Duration, queue int) (*Subscription, error) {
	return c.subscribe(ctx, resourceID, protocol.CapabilitySubscribe, lease, queue, true)
}

func (c *Client) subscribe(ctx context.Context, resourceID protocol.ResourceID, capability protocol.CapabilityID, lease time.Duration, queue int, variableOnly bool) (*Subscription, error) {
	if ctx == nil {
		return nil, errors.New("SDK subscription context is required")
	}
	if lease <= 0 || queue < 1 || queue > 1024 {
		return nil, errors.New("SDK subscription lease or queue is invalid")
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
		defer result.Cancel()
		defer close(events)
		defer close(errorsOut)
		var decoder VariableEventDecoder
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
				if variableOnly || converted.Kind == EventSnapshot || converted.Revision != 0 || converted.Schema == protocol.SchemaVariableValueV1 || decoder.initialized {
					variableOnly = true
					if converted.Resource != resourceID {
						select {
						case errorsOut <- &Error{Code: protocol.CodeMalformed, Message: "variable event resource does not match subscription"}:
						case <-runCtx.Done():
						}
						return
					}
					if _, err := decoder.Decode(converted); err != nil {
						select {
						case errorsOut <- err:
						case <-runCtx.Done():
						}
						return
					}
				}
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

// Snapshot reads and validates the complete standard variable state without
// requiring Catalog discovery or subscribe permission.
func (c *Client) Snapshot(ctx context.Context, resourceID protocol.ResourceID) (protocol.VariableValueV1, error) {
	if ctx == nil {
		return protocol.VariableValueV1{}, errors.New("SDK snapshot context is required")
	}
	result, err := c.Operate(ctx, resourceID, protocol.CapabilityRead, "", nil)
	if err != nil {
		return protocol.VariableValueV1{}, err
	}
	return decodeVariableEnvelope(result.Schema, result.Payload, true)
}

func (c *Client) DecodeSnapshot(ctx context.Context, resourceID protocol.ResourceID, target protocol.ValidatedPayload) error {
	if isNilValidatedPayload(target) {
		return errors.New("SDK snapshot target is required")
	}
	state, err := c.Snapshot(ctx, resourceID)
	if err != nil {
		return err
	}
	if len(state.Value) == 0 {
		return &Error{Message: fmt.Sprintf("%s variable value is unavailable", resourceID.Name)}
	}
	if err := protocol.DecodeVariablePayload(*state.Definition, state.Value, protocol.DefaultMaxPayload, target); err != nil {
		return &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("decode %s snapshot: %v", resourceID.Name, err), cause: err}
	}
	return nil
}

// WriteVariable submits a V3 conditional write and returns the owner's confirmed
// state. JSON numbers and revisions are never converted through float64.
func (c *Client) WriteVariable(ctx context.Context, resourceID protocol.ResourceID, expectedInstanceID, expectedRevision string, value json.RawMessage) (protocol.VariableValueV1, error) {
	if ctx == nil {
		return protocol.VariableValueV1{}, errors.New("SDK variable write context is required")
	}
	request, err := protocol.EncodeJSONPayload(&protocol.VariableWriteV3{
		Version: 3, ExpectedInstanceID: expectedInstanceID, ExpectedRevision: expectedRevision, Value: value,
	}, protocol.DefaultMaxPayload)
	if err != nil {
		return protocol.VariableValueV1{}, &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("encode variable write: %v", err), cause: err}
	}
	result, err := c.Operate(ctx, resourceID, protocol.CapabilityWrite, protocol.SchemaVariableWriteV3, request)
	if err != nil {
		return protocol.VariableValueV1{}, err
	}
	state, err := decodeVariableEnvelope(result.Schema, result.Payload, true)
	if err != nil {
		return protocol.VariableValueV1{}, err
	}
	revision, err := protocol.ParseVariableRevision(state.Revision)
	if err != nil {
		return protocol.VariableValueV1{}, err
	}
	expected, err := protocol.ParseVariableRevision(expectedRevision)
	if err != nil {
		return protocol.VariableValueV1{}, err
	}
	if state.InstanceID != expectedInstanceID || revision <= expected {
		return protocol.VariableValueV1{}, &Error{Code: protocol.CodeMalformed, Message: "variable write response does not confirm the expected instance and a newer revision"}
	}
	if state.Status != "ready" {
		return protocol.VariableValueV1{}, &Error{Code: protocol.CodeMalformed, Message: "variable write response does not confirm a ready value"}
	}
	return state, nil
}

func decodeVariableEnvelope(schema string, payload []byte, requireDefinition bool) (protocol.VariableValueV1, error) {
	if schema != protocol.SchemaVariableValueV1 {
		return protocol.VariableValueV1{}, &Error{Code: protocol.CodeUnsupported, Message: fmt.Sprintf("unsupported variable schema %q", schema)}
	}
	var state protocol.VariableValueV1
	if err := protocol.DecodeJSONPayload(payload, protocol.DefaultMaxPayload, &state); err != nil {
		return protocol.VariableValueV1{}, &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("decode variable state: %v", err), cause: err}
	}
	if requireDefinition && state.Definition == nil {
		return protocol.VariableValueV1{}, &Error{Code: protocol.CodeMalformed, Message: "variable snapshot requires definition"}
	}
	return state, nil
}

// VariableEventDecoder validates one resource subscription, retaining only its
// current instance definition. Use a separate decoder for each subscription;
// it is not safe for concurrent calls. Its zero value expects an initial snapshot.
type VariableEventDecoder struct {
	resource    protocol.ResourceID
	instanceID  string
	revision    uint64
	definition  []byte
	initialized bool
}

// Decode returns a validated state with the session definition populated. It
// leaves the wire event untouched. Gap/expired events invalidate the cache and
// return an empty state. A fresh snapshot is then required before more data.
func (d *VariableEventDecoder) Decode(event Event) (protocol.VariableValueV1, error) {
	malformed := func(message string) (protocol.VariableValueV1, error) {
		return protocol.VariableValueV1{}, &Error{Code: protocol.CodeMalformed, Message: message}
	}
	if d == nil {
		return malformed("variable event decoder is required")
	}
	if d.initialized && event.Resource != d.resource {
		return malformed("variable event changed resource")
	}
	if event.Kind == EventGap || event.Kind == EventExpired {
		*d = VariableEventDecoder{}
		return protocol.VariableValueV1{}, nil
	}
	if event.Kind != EventSnapshot && event.Kind != EventData {
		return malformed("unsupported variable event kind")
	}
	if err := event.Resource.Validate(); err != nil {
		return malformed(fmt.Sprintf("variable event resource: %v", err))
	}
	if event.Kind == EventData && !d.initialized {
		return malformed("variable subscription requires an initial snapshot")
	}
	state, err := decodeVariableEnvelope(event.Schema, event.Value, event.Kind == EventSnapshot)
	if err != nil {
		return protocol.VariableValueV1{}, err
	}
	revision, err := protocol.ParseVariableRevision(state.Revision)
	if err != nil {
		return malformed(fmt.Sprintf("variable revision: %v", err))
	}
	if revision != event.Revision {
		return malformed("variable envelope revision differs from event revision")
	}
	if d.initialized && event.Kind == EventData && state.InstanceID != d.instanceID {
		return malformed("variable data changed instance without a snapshot")
	}
	if d.initialized && state.InstanceID == d.instanceID && (revision < d.revision || event.Kind == EventData && revision == d.revision) {
		return malformed("variable event revision did not advance")
	}
	definition := d.definition
	if state.Definition != nil {
		definition, err = json.Marshal(state.Definition)
		if err != nil {
			return malformed(fmt.Sprintf("variable definition: %v", err))
		}
		if d.initialized && state.InstanceID == d.instanceID && !bytes.Equal(definition, d.definition) {
			return malformed("variable definition changed within an instance")
		}
	} else {
		if err := json.Unmarshal(definition, &state.Definition); err != nil {
			return malformed(fmt.Sprintf("variable cached definition: %v", err))
		}
		// Envelopes carrying a definition were already fully validated by
		// DecodeJSONPayload. Only definition-less data needs cached validation.
		if err := protocol.ValidateVariableState(*state.Definition, state); err != nil {
			return malformed(fmt.Sprintf("variable cached state: %v", err))
		}
		full, err := json.Marshal(state)
		if err != nil {
			return malformed(fmt.Sprintf("encode complete variable state: %v", err))
		}
		if len(full) > protocol.DefaultMaxPayload {
			return malformed("complete variable state exceeds protocol payload limit")
		}
	}
	*d = VariableEventDecoder{resource: event.Resource, instanceID: state.InstanceID, revision: revision, definition: definition, initialized: true}
	return state, nil
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
