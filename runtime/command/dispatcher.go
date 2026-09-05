package command

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

var (
	ErrDuplicate          = errors.New("duplicate command message")
	ErrPendingLimit       = errors.New("command concurrency limit reached")
	ErrUnauthorizedOrigin = errors.New("command authorization origin is invalid")
)

type retryableError struct{ error }

func (retryableError) Retryable() bool { return true }

func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return retryableError{error: err}
}

type Origin uint8

const (
	OriginAdjudicated Origin = iota + 1
	OriginParentControl
)

type Call struct {
	MessageID  protocol.MessageID
	Source     protocol.NodeID
	Resource   protocol.ResourceID
	Capability protocol.CapabilityID
	Schema     string
	Input      []byte
	Deadline   time.Time
	Origin     Origin
}

type Delegation struct {
	subject protocol.NodeID
	valid   bool
}

type callContextKey struct{}

func DelegationFromContext(ctx context.Context) (Delegation, bool) {
	if ctx == nil {
		return Delegation{}, false
	}
	call, ok := ctx.Value(callContextKey{}).(Call)
	if !ok || call.Source == 0 {
		return Delegation{}, false
	}
	return Delegation{subject: call.Source, valid: true}, true
}

func (d Delegation) Subject() (protocol.NodeID, bool) { return d.subject, d.valid }

type Result struct {
	Schema  string
	Output  []byte
	Failure *protocol.ErrorPayload
}

func (r Result) clone() Result {
	r.Output = append([]byte(nil), r.Output...)
	if r.Failure != nil {
		failure := *r.Failure
		if r.Failure.Details != nil {
			failure.Details = make(map[string]string, len(r.Failure.Details))
			for key, value := range r.Failure.Details {
				failure.Details[key] = value
			}
		}
		r.Failure = &failure
	}
	return r
}

type Authorizer func(context.Context, Call) error
type LateResultObserver func(Call, Result)

type Config struct {
	MaxActive    int
	MaxDedup     int
	DedupTTL     time.Duration
	Authorizer   Authorizer
	OnLateResult LateResultObserver
}

func (c Config) normalized() (Config, error) {
	if c.MaxActive <= 0 {
		c.MaxActive = 64
	}
	if c.MaxDedup <= 0 {
		c.MaxDedup = 4096
	}
	if c.DedupTTL <= 0 {
		c.DedupTTL = 5 * time.Minute
	}
	if c.MaxActive > c.MaxDedup {
		return Config{}, errors.New("command max active cannot exceed dedup capacity")
	}
	return c, nil
}

type cacheRecord struct {
	id      protocol.MessageID
	result  Result
	expires time.Time
}

type Dispatcher struct {
	registry *resource.Registry
	config   Config
	active   chan struct{}

	mu       sync.Mutex
	pending  map[protocol.MessageID]struct{}
	cache    map[protocol.MessageID]*list.Element
	cacheLRU *list.List
}

func NewDispatcher(registry *resource.Registry, config Config) (*Dispatcher, error) {
	if registry == nil {
		return nil, errors.New("command registry is required")
	}
	normalized, err := config.normalized()
	if err != nil {
		return nil, err
	}
	return &Dispatcher{
		registry: registry, config: normalized, active: make(chan struct{}, normalized.MaxActive),
		pending: make(map[protocol.MessageID]struct{}), cache: make(map[protocol.MessageID]*list.Element), cacheLRU: list.New(),
	}, nil
}

func (d *Dispatcher) Invoke(parent context.Context, call Call) (Result, error) {
	if parent == nil {
		return Result{}, errors.New("command context is required")
	}
	if call.MessageID.IsZero() {
		return Result{}, errors.New("command message ID is required")
	}
	if err := call.Source.Validate(); err != nil {
		return Result{}, err
	}
	if err := call.Resource.Validate(); err != nil {
		return Result{}, err
	}
	if err := call.Capability.Validate(); err != nil {
		return Result{}, err
	}
	if call.Deadline.IsZero() || !call.Deadline.After(time.Now()) {
		return timeoutResult("command deadline has expired"), context.DeadlineExceeded
	}
	if call.Origin != OriginAdjudicated && call.Origin != OriginParentControl {
		return forbiddenResult("command has no trusted authorization origin"), ErrUnauthorizedOrigin
	}
	if call.Origin == OriginAdjudicated {
		if d.config.Authorizer == nil {
			return forbiddenResult("command authorizer is not configured"), ErrUnauthorizedOrigin
		}
		if err := d.config.Authorizer(parent, call); err != nil {
			return forbiddenResult(err.Error()), err
		}
	}
	if cached, duplicate := d.begin(call.MessageID); duplicate {
		return cached, ErrDuplicate
	}
	select {
	case d.active <- struct{}{}:
	case <-parent.Done():
		d.endPending(call.MessageID)
		return resultForContext(parent.Err()), parent.Err()
	default:
		d.endPending(call.MessageID)
		failure := protocol.ErrorPayload{Code: protocol.CodeOverflow, Message: ErrPendingLimit.Error(), Retryable: true}
		return Result{Failure: &failure}, ErrPendingLimit
	}
	ctx, cancel := context.WithDeadline(parent, call.Deadline)
	defer cancel()
	resultChannel := make(chan Result)
	go func() {
		defer func() { <-d.active }()
		result := d.execute(ctx, call)
		select {
		case resultChannel <- result:
		case <-ctx.Done():
			if d.config.OnLateResult != nil {
				d.config.OnLateResult(call, result.clone())
			}
		}
	}()
	select {
	case result := <-resultChannel:
		d.complete(call.MessageID, result)
		return result.clone(), nil
	case <-ctx.Done():
		result := resultForContext(ctx.Err())
		d.complete(call.MessageID, result)
		return result, ctx.Err()
	}
}

func (d *Dispatcher) execute(ctx context.Context, call Call) (result Result) {
	defer func() {
		if recovered := recover(); recovered != nil {
			failure := protocol.ErrorPayload{Code: protocol.CodeInternal, Message: fmt.Sprintf("command handler panic: %v", recovered)}
			result = Result{Failure: &failure}
		}
	}()
	operation, err := d.registry.Operate(context.WithValue(ctx, callContextKey{}, call), call.Resource, resource.OperationRequest{
		Subject: call.Source, Capability: call.Capability, Schema: call.Schema, Payload: call.Input,
	})
	if err != nil {
		failure := protocol.ErrorPayload{Code: protocol.CodeInternal, Message: err.Error()}
		if errors.Is(err, resource.ErrNotFound) {
			failure.Code = protocol.CodeNotFound
		}
		if errors.Is(err, resource.ErrUnsupportedCapability) {
			failure.Code = protocol.CodeUnsupported
		}
		if errors.Is(err, resource.ErrInvalidSchema) || errors.Is(err, resource.ErrValueTooLarge) {
			failure.Code = protocol.CodeMalformed
		}
		if errors.Is(err, protocol.ErrInvalidPayload) {
			failure.Code = protocol.CodeMalformed
		}
		if errors.Is(err, protocol.ErrPayloadTooLarge) {
			failure.Code = protocol.CodeOverflow
		}
		if errors.Is(err, resource.ErrRevisionConflict) {
			failure.Code = protocol.CodeConflict
		}
		var temporary interface{ Retryable() bool }
		if errors.As(err, &temporary) && temporary.Retryable() {
			failure.Retryable = true
		}
		if errors.Is(err, context.DeadlineExceeded) {
			failure.Code = protocol.CodeTimeout
		}
		return Result{Failure: &failure}
	}
	return Result{Schema: operation.Schema, Output: operation.Payload}
}

func (d *Dispatcher) begin(id protocol.MessageID) (Result, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pruneLocked(time.Now())
	if _, exists := d.pending[id]; exists {
		return duplicateResult(), true
	}
	if element, exists := d.cache[id]; exists {
		record := element.Value.(cacheRecord)
		d.cacheLRU.MoveToBack(element)
		duplicate := duplicateResult()
		if record.result.Failure != nil {
			duplicate.Failure.Details = map[string]string{"previous_code": string(record.result.Failure.Code)}
		}
		return duplicate, true
	}
	d.pending[id] = struct{}{}
	return Result{}, false
}

func (d *Dispatcher) endPending(id protocol.MessageID) {
	d.mu.Lock()
	delete(d.pending, id)
	d.mu.Unlock()
}

func (d *Dispatcher) complete(id protocol.MessageID, result Result) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.pending, id)
	d.pruneLocked(time.Now())
	if element, exists := d.cache[id]; exists {
		d.cacheLRU.Remove(element)
		delete(d.cache, id)
	}
	record := cacheRecord{id: id, result: result.clone(), expires: time.Now().Add(d.config.DedupTTL)}
	d.cache[id] = d.cacheLRU.PushBack(record)
	for len(d.cache) > d.config.MaxDedup {
		d.removeOldestLocked()
	}
}

func (d *Dispatcher) pruneLocked(now time.Time) {
	for {
		front := d.cacheLRU.Front()
		if front == nil || front.Value.(cacheRecord).expires.After(now) {
			return
		}
		d.cacheLRU.Remove(front)
		delete(d.cache, front.Value.(cacheRecord).id)
	}
}

func (d *Dispatcher) removeOldestLocked() {
	front := d.cacheLRU.Front()
	if front == nil {
		return
	}
	d.cacheLRU.Remove(front)
	delete(d.cache, front.Value.(cacheRecord).id)
}

func timeoutResult(message string) Result {
	failure := protocol.ErrorPayload{Code: protocol.CodeTimeout, Message: message, Retryable: true}
	return Result{Failure: &failure}
}

func forbiddenResult(message string) Result {
	failure := protocol.ErrorPayload{Code: protocol.CodeForbidden, Message: message}
	return Result{Failure: &failure}
}

func duplicateResult() Result {
	failure := protocol.ErrorPayload{Code: protocol.CodeConflict, Message: ErrDuplicate.Error()}
	return Result{Failure: &failure}
}

func resultForContext(err error) Result {
	if errors.Is(err, context.DeadlineExceeded) {
		return timeoutResult(err.Error())
	}
	failure := protocol.ErrorPayload{Code: protocol.CodeExpired, Message: err.Error(), Retryable: true}
	return Result{Failure: &failure}
}

func (d *Dispatcher) Pending() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.pending)
}
