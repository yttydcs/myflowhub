package resource

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yttydcs/myflowhub/protocol"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// VariableState contains owner-supplied data. Identity, revision and observation
// time are assigned by the shared runtime, never by a remote writer.
type VariableState struct {
	Value            json.RawMessage
	Status           string
	SourceTimeUnixMS *int64
	Error            *protocol.VariableError
}
type VariableUpdate struct {
	Revision     uint64
	State        protocol.VariableValueV1
	Failure      *protocol.ErrorPayload
	eventPayload []byte
}

// VariableWriteHandler returns the confirmed state after applying a typed JSON
// value. Numbers are json.Number. It must not mutate the same Variable itself.
type VariableWriteHandler func(context.Context, any) (VariableState, error)
type variableWatcher struct {
	id        uint64
	callback  func(VariableUpdate)
	cancelled atomic.Bool
}
type variableNotification struct {
	update   VariableUpdate
	watchers []*variableWatcher
}
type preparedVariable struct {
	update  VariableUpdate
	payload []byte
}
type Variable struct {
	definitionJSON []byte
	gate           chan struct{}
	mu             sync.RWMutex
	descriptor     Descriptor
	definition     protocol.DataSchemaDefinition
	instanceID     string
	current        preparedVariable
	handler        VariableWriteHandler
	retired        bool
	nextWatch      uint64
	watchers       map[uint64]*variableWatcher
	pending        []variableNotification
	draining       bool
}

func NewVariable(descriptor Descriptor, definition protocol.DataSchemaDefinition, initial VariableState, handler VariableWriteHandler) (*Variable, error) {
	var err error
	descriptor, err = normalizeDescriptor(descriptor)
	if err != nil {
		return nil, err
	}
	if err := validateVariableDescriptor(descriptor); err != nil {
		return nil, err
	}
	if handler != nil {
		if _, ok := descriptor.Capability(protocol.CapabilityWrite); !ok {
			return nil, errors.New("variable write handler requires write capability")
		}
	}
	if err := protocol.ValidateVariableDefinition(definition); err != nil {
		return nil, fmt.Errorf("variable definition: %w", err)
	}
	definition = cloneVariableDefinition(definition)
	var identity [16]byte
	if _, err := rand.Read(identity[:]); err != nil {
		return nil, fmt.Errorf("variable instance identity: %w", err)
	}
	v := &Variable{gate: make(chan struct{}, 1), descriptor: descriptor, definition: definition, instanceID: hex.EncodeToString(identity[:]), handler: handler, watchers: make(map[uint64]*variableWatcher)}
	v.definitionJSON, err = json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("encode variable definition: %w", err)
	}
	v.current, err = v.prepare(1, initial)
	if err != nil {
		return nil, err
	}
	return v, nil
}
func validateVariableDescriptor(descriptor Descriptor) error {
	if descriptor.Type != protocol.ResourceTypeVariable || descriptor.TypeVersion != 2 {
		return fmt.Errorf("%w: variable requires mfh.variable type_version 2", ErrInvalidSchema)
	}
	read, ok := descriptor.Capability(protocol.CapabilityRead)
	if !ok || read.InputSchema != "" || read.OutputSchema != protocol.SchemaVariableValueV1 {
		return fmt.Errorf("%w: variable requires standard read capability", ErrInvalidSchema)
	}
	subscribe, ok := descriptor.Capability(protocol.CapabilitySubscribe)
	if !ok || subscribe.EventSchema != protocol.SchemaVariableValueV1 {
		return fmt.Errorf("%w: variable requires standard subscribe capability", ErrInvalidSchema)
	}
	if write, ok := descriptor.Capability(protocol.CapabilityWrite); ok && (write.InputSchema != protocol.SchemaVariableWriteV3 || write.OutputSchema != protocol.SchemaVariableValueV1) {
		return fmt.Errorf("%w: variable requires standard write capability", ErrInvalidSchema)
	}
	return nil
}
func (v *Variable) Descriptor() Descriptor { return cloneDescriptor(v.descriptor) }
func (v *Variable) Operate(ctx context.Context, request OperationRequest) (OperationResult, error) {
	if ctx == nil {
		return OperationResult{}, errors.New("variable context is required")
	}
	if err := ctx.Err(); err != nil {
		return OperationResult{}, err
	}
	switch request.Capability {
	case protocol.CapabilityRead:
		if len(request.Payload) != 0 || request.Schema != "" {
			return OperationResult{}, fmt.Errorf("%w: variable read takes no input", ErrInvalidSchema)
		}
		v.mu.RLock()
		defer v.mu.RUnlock()
		if v.retired {
			return OperationResult{}, ErrNotFound
		}
		return OperationResult{Schema: protocol.SchemaVariableValueV1, Payload: bytes.Clone(v.current.payload)}, nil
	case protocol.CapabilityWrite:
		if _, ok := v.descriptor.Capability(protocol.CapabilityWrite); !ok {
			return OperationResult{}, fmt.Errorf("%w: write", ErrUnsupportedCapability)
		}
		if request.Schema != "" && request.Schema != protocol.SchemaVariableWriteV3 {
			return OperationResult{}, ErrInvalidSchema
		}
		var write protocol.VariableWriteV3
		if err := protocol.DecodeJSONPayload(request.Payload, v.descriptor.Limits.MaxPayloadBytes, &write); err != nil {
			return OperationResult{}, err
		}
		revision, err := protocol.ParseVariableRevision(write.ExpectedRevision)
		if err != nil {
			return OperationResult{}, err
		}
		update, err := v.CompareAndSet(ctx, write.ExpectedInstanceID, revision, write.Value)
		if err != nil {
			return OperationResult{}, err
		}
		payload, err := protocol.EncodeJSONPayload(&update.State, v.descriptor.Limits.MaxPayloadBytes)
		return OperationResult{Schema: protocol.SchemaVariableValueV1, Payload: payload}, err
	default:
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
}
func (v *Variable) acquire(ctx context.Context) error {
	if ctx == nil {
		return errors.New("variable context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case v.gate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (v *Variable) release() { <-v.gate }
func (v *Variable) CompareAndSet(ctx context.Context, expectedInstanceID string, expectedRevision uint64, value json.RawMessage) (VariableUpdate, error) {
	if err := v.acquire(ctx); err != nil {
		return VariableUpdate{}, err
	}
	update, drain, err := v.compareAndSet(ctx, expectedInstanceID, expectedRevision, bytes.Clone(value))
	v.release()
	if drain {
		v.drain()
	}
	return update, err
}
func (v *Variable) compareAndSet(ctx context.Context, instance string, revision uint64, value json.RawMessage) (VariableUpdate, bool, error) {
	if err := ctx.Err(); err != nil {
		return VariableUpdate{}, false, err
	}
	v.mu.RLock()
	current, retired := v.current.update.Revision, v.retired
	v.mu.RUnlock()
	if retired {
		return VariableUpdate{}, false, ErrNotFound
	}
	if revision == 0 || instance != v.instanceID || revision != current {
		return VariableUpdate{}, false, fmt.Errorf("%w: variable instance or revision changed", ErrRevisionConflict)
	}
	if current == math.MaxUint64 {
		return VariableUpdate{}, false, errors.New("variable revision exhausted")
	}
	state := VariableState{Value: value, Status: "ready"}
	// Preflight the full envelope and queue before external effects. Rejected
	// proposals must never reach the actuator.
	prepared, err := v.prepare(current+1, state)
	if err != nil {
		return VariableUpdate{}, false, err
	}
	if err := v.canCommit(); err != nil {
		return VariableUpdate{}, false, err
	}
	if v.handler != nil {
		decoder := json.NewDecoder(bytes.NewReader(value))
		decoder.UseNumber()
		var typed any
		if err := decoder.Decode(&typed); err != nil {
			return VariableUpdate{}, false, err
		}
		if err := ctx.Err(); err != nil {
			return VariableUpdate{}, false, err
		}
		state, err = invokeVariableHandler(v.handler, ctx, typed)
		if err != nil {
			return VariableUpdate{}, false, err
		}
		if err := ctx.Err(); err != nil {
			return VariableUpdate{}, false, fmt.Errorf("variable write outcome unconfirmed: %w", err)
		}
		if state.Status != "ready" {
			return VariableUpdate{}, false, errors.New("variable write handler did not confirm a ready value")
		}
		prepared, err = v.prepare(current+1, state)
		if err != nil {
			return VariableUpdate{}, false, fmt.Errorf("variable write result could not be published: %w", err)
		}
	}
	return v.commit(prepared)
}
func (v *Variable) Snapshot() VariableUpdate {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cloneVariableUpdate(v.current.update)
}
func (v *Variable) Set(state VariableState) (VariableUpdate, error) { return v.apply(0, state) }
func (v *Variable) Apply(revision uint64, state VariableState) (VariableUpdate, error) {
	if revision == 0 {
		return VariableUpdate{}, ErrRevisionRegression
	}
	return v.apply(revision, state)
}
func (v *Variable) apply(revision uint64, state VariableState) (VariableUpdate, error) {
	if err := v.acquire(context.Background()); err != nil {
		return VariableUpdate{}, err
	}
	prepared, err := v.prepareNext(revision, state)
	var update VariableUpdate
	var drain bool
	if err == nil {
		update, drain, err = v.commit(prepared)
	}
	v.release()
	if drain {
		v.drain()
	}
	return update, err
}

// Registry holds the mutation gate until resource visibility and catalog state
// commit together. All callbacks run after both registry and variable locks exit.
func (v *Variable) prepareNext(revision uint64, state VariableState) (preparedVariable, error) {
	v.mu.RLock()
	current, retired := v.current.update.Revision, v.retired
	v.mu.RUnlock()
	if retired {
		return preparedVariable{}, ErrNotFound
	}
	if current == math.MaxUint64 {
		return preparedVariable{}, errors.New("variable revision exhausted")
	}
	if revision == 0 {
		revision = current + 1
	}
	if revision <= current {
		return preparedVariable{}, ErrRevisionRegression
	}
	if err := v.canCommit(); err != nil {
		return preparedVariable{}, err
	}
	return v.prepare(revision, state)
}
func (v *Variable) prepare(revision uint64, state VariableState) (preparedVariable, error) {
	if err := validatePayload(v.descriptor, state.Value); err != nil {
		return preparedVariable{}, err
	}
	envelope := protocol.VariableValueV1{Version: 1, InstanceID: v.instanceID, Revision: strconv.FormatUint(revision, 10), Value: bytes.Clone(state.Value), Status: state.Status, SourceTimeUnixMS: state.SourceTimeUnixMS, ObservedAtUnixMS: time.Now().UnixMilli(), Error: state.Error}
	if err := protocol.ValidateVariableState(v.definition, envelope); err != nil {
		return preparedVariable{}, fmt.Errorf("%w: %v", protocol.ErrInvalidPayload, err)
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return preparedVariable{}, fmt.Errorf("encode variable state: %w", err)
	}
	// The immutable definition is encoded once at construction. The data event
	// and complete snapshot are both prepared before the state becomes visible.
	payload := make([]byte, 0, len(data)+len(v.definitionJSON)+14)
	payload = append(payload, data[:len(data)-1]...)
	payload = append(payload, `,"definition":`...)
	payload = append(payload, v.definitionJSON...)
	payload = append(payload, '}')
	if len(payload) > v.payloadLimit() {
		return preparedVariable{}, fmt.Errorf("%w: complete variable envelope", ErrValueTooLarge)
	}
	definition := v.definition
	envelope.Definition = &definition
	return preparedVariable{update: cloneVariableUpdate(VariableUpdate{Revision: revision, State: envelope, eventPayload: data}), payload: payload}, nil
}
func (v *Variable) canCommit() error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.retired {
		return ErrNotFound
	}
	if len(v.pending) >= v.descriptor.Limits.MaxQueue {
		return errors.New("variable notification queue is full")
	}
	return nil
}
func (v *Variable) commit(prepared preparedVariable) (VariableUpdate, bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.retired {
		return VariableUpdate{}, false, ErrNotFound
	}
	if len(v.pending) >= v.descriptor.Limits.MaxQueue {
		return VariableUpdate{}, false, errors.New("variable notification queue is full")
	}
	v.current = prepared
	drain := v.enqueueLocked(prepared.update)
	return cloneVariableUpdate(prepared.update), drain, nil
}
func (v *Variable) enqueueLocked(update VariableUpdate) bool {
	if len(v.watchers) == 0 {
		return false
	}
	watchers := make([]*variableWatcher, 0, len(v.watchers))
	for _, watcher := range v.watchers {
		watchers = append(watchers, watcher)
	}
	v.pending = append(v.pending, variableNotification{update: update, watchers: watchers})
	if v.draining {
		return false
	}
	v.draining = true
	return true
}
func (v *Variable) drain() {
	for {
		v.mu.Lock()
		if len(v.pending) == 0 {
			v.draining = false
			v.mu.Unlock()
			return
		}
		notification := v.pending[0]
		v.pending[0] = variableNotification{}
		v.pending = v.pending[1:]
		v.mu.Unlock()
		for _, watcher := range notification.watchers {
			if !watcher.cancelled.Load() {
				v.callWatcher(watcher, notification.update)
			}
		}
	}
}
func (v *Variable) Watch(observer func(VariableUpdate)) (VariableUpdate, func(), error) {
	if observer == nil {
		return VariableUpdate{}, nil, errors.New("variable observer is required")
	}
	v.mu.Lock()
	if v.retired {
		v.mu.Unlock()
		return VariableUpdate{}, nil, ErrNotFound
	}
	if len(v.watchers) >= v.descriptor.Limits.MaxSubscribers {
		v.mu.Unlock()
		return VariableUpdate{}, nil, errors.New("variable subscriber limit reached")
	}
	v.nextWatch++
	id := v.nextWatch
	watcher := &variableWatcher{id: id, callback: observer}
	v.watchers[id] = watcher
	snapshot := cloneVariableUpdate(v.current.update)
	v.mu.Unlock()
	var once sync.Once
	cancel := func() {
		once.Do(func() { watcher.cancelled.Store(true); v.mu.Lock(); delete(v.watchers, id); v.mu.Unlock() })
	}
	return snapshot, cancel, nil
}
func (v *Variable) Observe(observer func(Observation)) (*Observation, func(), error) {
	if observer == nil {
		return nil, nil, errors.New("variable observer is required")
	}
	snapshot, cancel, err := v.Watch(func(update VariableUpdate) {
		if update.Failure != nil {
			observer(Observation{Failure: update.Failure})
			return
		}
		observer(Observation{Revision: update.Revision, Schema: protocol.SchemaVariableValueV1, Value: bytes.Clone(update.eventPayload)})
	})
	if err != nil {
		return nil, nil, err
	}
	payload, err := protocol.EncodeJSONPayload(&snapshot.State, v.descriptor.Limits.MaxPayloadBytes)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return &Observation{Snapshot: true, Revision: snapshot.Revision, Schema: protocol.SchemaVariableValueV1, Value: payload}, cancel, nil
}

// Retire invalidates this registration instance and terminates its observers.
// Re-registering requires a new Variable, hence a new identity.
func (v *Variable) Retire() { v.PrepareRetire()() }

// PrepareRetire invalidates identity synchronously, so Registry can bind it to
// its catalog transaction. The returned completion must run after registry
// locks exit: it waits for an in-flight writer and drains terminal notifications.
func (v *Variable) PrepareRetire() func() {
	v.mu.Lock()
	if v.retired {
		v.mu.Unlock()
		return func() {}
	}
	v.retired = true
	drain := v.enqueueLocked(VariableUpdate{Failure: &protocol.ErrorPayload{Code: protocol.CodeGone, Message: "variable instance was removed"}})
	v.watchers = make(map[uint64]*variableWatcher)
	v.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = v.acquire(context.Background())
			v.release()
			if drain {
				v.drain()
			}
		})
	}
}

func (v *Variable) callWatcher(watcher *variableWatcher, update VariableUpdate) {
	defer func() {
		if recovered := recover(); recovered != nil {
			watcher.cancelled.Store(true)
			v.mu.Lock()
			delete(v.watchers, watcher.id)
			v.mu.Unlock()
			// The faulty observer has no usable error callback. Report its removal
			// without logging arbitrary callback panic payloads or resource values.
			slog.Error("variable observer panicked; observer detached", "owner", v.descriptor.ID.Owner, "resource", v.descriptor.ID.Name)
		}
	}()
	watcher.callback(cloneVariableUpdate(update))
}

func cloneVariableDefinition(definition protocol.DataSchemaDefinition) protocol.DataSchemaDefinition {
	definition.Precision = cloneVariablePointer(definition.Precision)
	definition.Minimum = cloneVariablePointer(definition.Minimum)
	definition.Maximum = cloneVariablePointer(definition.Maximum)
	definition.MultipleOf = cloneVariablePointer(definition.MultipleOf)
	definition.MinLength = cloneVariablePointer(definition.MinLength)
	definition.MaxLength = cloneVariablePointer(definition.MaxLength)
	definition.MinItems = cloneVariablePointer(definition.MinItems)
	definition.MaxItems = cloneVariablePointer(definition.MaxItems)
	definition.Enum = append([]string(nil), definition.Enum...)
	definition.Properties = append([]protocol.DataSchemaProperty(nil), definition.Properties...)
	for i := range definition.Properties {
		if definition.Properties[i].Schema != nil {
			child := cloneVariableDefinition(*definition.Properties[i].Schema)
			definition.Properties[i].Schema = &child
		}
	}
	if definition.Items != nil {
		child := cloneVariableDefinition(*definition.Items)
		definition.Items = &child
	}
	if definition.AdditionalProperties != nil {
		child := cloneVariableDefinition(*definition.AdditionalProperties)
		definition.AdditionalProperties = &child
	}
	return definition
}
func cloneVariablePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func invokeVariableHandler(handler VariableWriteHandler, ctx context.Context, value any) (state VariableState, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = errors.New("variable write handler panicked; external outcome is unconfirmed")
		}
	}()
	return handler(ctx, value)
}

func cloneVariableUpdate(update VariableUpdate) VariableUpdate {
	update.State.Value = bytes.Clone(update.State.Value)
	if update.State.Definition != nil {
		definition := cloneVariableDefinition(*update.State.Definition)
		update.State.Definition = &definition
	}
	if update.State.SourceTimeUnixMS != nil {
		timestamp := *update.State.SourceTimeUnixMS
		update.State.SourceTimeUnixMS = &timestamp
	}
	if update.State.Error != nil {
		failure := *update.State.Error
		update.State.Error = &failure
	}
	if update.Failure != nil {
		failure := *update.Failure
		update.Failure = &failure
	}
	return update
}

func (v *Variable) Retired() bool { v.mu.RLock(); defer v.mu.RUnlock(); return v.retired }

// A successful publication must fit every capability that exposes the state,
// including a custom descriptor with a tighter per-capability output limit.
func (v *Variable) payloadLimit() int {
	limit := v.descriptor.Limits.MaxPayloadBytes
	for _, capability := range v.descriptor.Capabilities {
		if (capability.Name == protocol.CapabilityRead || capability.Name == protocol.CapabilitySubscribe || capability.Name == protocol.CapabilityWrite) && capability.MaxPayloadBytes < limit {
			limit = capability.MaxPayloadBytes
		}
	}
	return limit
}
