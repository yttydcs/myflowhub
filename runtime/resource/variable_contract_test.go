package resource

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yttydcs/myflowhub/protocol"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestBoolean(t *testing.T, handler VariableWriteHandler) *Variable {
	t.Helper()
	v, err := NewVariable(WritableVariableDescriptor(protocol.ResourceID{Owner: 1, Name: "independent/state"}, "read", "write", 1024), protocol.DataSchemaDefinition{Type: "boolean"}, testReady("false"), handler)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestVariableEveryMutationRejectsBeforeStateAndObservation(t *testing.T) {
	invalid := []VariableState{testReady(`"false"`), testReady(`null`), {Status: "ready"}, {Status: "unavailable", Value: json.RawMessage("false")}, {Status: "stale"}, {Status: "unknown", Value: json.RawMessage("false")}, {Status: "stale", Value: json.RawMessage("false"), Error: &protocol.VariableError{Code: "failed", Message: strings.Repeat("x", 1025)}}}
	for _, state := range invalid {
		v := newTestBoolean(t, nil)
		before := v.Snapshot()
		count := 0
		_, cancel, err := v.Observe(func(Observation) { count++ })
		if err != nil {
			t.Fatal(err)
		}
		if _, err := v.Set(state); err == nil {
			t.Fatalf("Set accepted %+v", state)
		}
		if _, err := v.Apply(9, state); err == nil {
			t.Fatalf("Apply accepted %+v", state)
		}
		if count != 0 || !reflect.DeepEqual(before, v.Snapshot()) {
			t.Fatal("invalid local state became observable")
		}
		if string(state.Value) != "false" {
			if _, err := v.CompareAndSet(context.Background(), before.State.InstanceID, before.Revision, state.Value); err == nil {
				t.Fatalf("CAS accepted invalid value %s", state.Value)
			}
			if count != 0 || !reflect.DeepEqual(before, v.Snapshot()) {
				t.Fatal("invalid CAS changed state")
			}
		}
		cancel()
	}
}
func TestVariableFullEnvelopeLimitPrecedesEveryMutation(t *testing.T) {
	descriptor := WritableVariableDescriptor(protocol.ResourceID{Owner: 1, Name: "text"}, "read", "write", 512)
	v, err := NewVariable(descriptor, protocol.DataSchemaDefinition{Type: "string"}, testReady(`""`), nil)
	if err != nil {
		t.Fatal(err)
	}
	before := v.Snapshot()
	// Business bytes alone fit. The shared envelope makes the state too large.
	state := testReady(`"` + strings.Repeat("x", 400) + `"`)
	for _, mutate := range []func() error{
		func() error { _, err := v.Set(state); return err },
		func() error { _, err := v.Apply(11, state); return err },
		func() error {
			_, err := v.CompareAndSet(context.Background(), before.State.InstanceID, 1, state.Value)
			return err
		},
	} {
		if err := mutate(); !errors.Is(err, ErrValueTooLarge) {
			t.Fatalf("expected envelope size rejection, got %v", err)
		}
	}
	if !reflect.DeepEqual(before, v.Snapshot()) {
		t.Fatal("failed envelope encoding mutated state")
	}
}
func TestVariableConcurrentCASInvokesOwnerOnceAndConfirmsActualValue(t *testing.T) {
	var calls atomic.Int32
	v := newTestBoolean(t, func(_ context.Context, value any) (VariableState, error) {
		if _, ok := value.(bool); !ok {
			t.Errorf("handler got untyped %T", value)
		}
		calls.Add(1)
		return testReady("false"), nil
	})
	before := v.Snapshot()
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := v.CompareAndSet(context.Background(), before.State.InstanceID, 1, json.RawMessage("true"))
			if err == nil {
				successes.Add(1)
				if string(result.State.Value) != "false" {
					t.Error("request value reported as actual")
				}
			} else if !errors.Is(err, ErrRevisionConflict) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 || successes.Load() != 1 {
		t.Fatalf("actuator %d, success %d", calls.Load(), successes.Load())
	}
	current := v.Snapshot()
	if _, err := v.CompareAndSet(context.Background(), current.State.InstanceID, current.Revision, json.RawMessage(`"true"`)); err == nil {
		t.Fatal("invalid input accepted")
	}
	if calls.Load() != 1 {
		t.Fatal("invalid input reached owner")
	}
}
func TestVariableHandlerDoesNotLockSnapshotAndCancelledWriterDoesNotExecute(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	v := newTestBoolean(t, func(context.Context, any) (VariableState, error) {
		calls.Add(1)
		close(entered)
		<-release
		return testReady("true"), nil
	})
	before := v.Snapshot()
	done := make(chan error, 1)
	go func() {
		_, err := v.CompareAndSet(context.Background(), before.State.InstanceID, 1, json.RawMessage("true"))
		done <- err
	}()
	<-entered
	readDone := make(chan struct{})
	go func() {
		if v.Snapshot().Revision != 1 {
			t.Error("unconfirmed state was visible")
		}
		close(readDone)
	}()
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("handler held snapshot lock")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := v.CompareAndSet(ctx, before.State.InstanceID, 1, json.RawMessage("true")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waiting writer ignored context: %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("cancelled write invoked owner")
	}
}
func TestVariableReentrantWatchersRemainOrdered(t *testing.T) {
	v := newTestBoolean(t, nil)
	var observations [2][]uint64
	for i := range observations {
		_, cancel, err := v.Watch(func(update VariableUpdate) {
			observations[i] = append(observations[i], update.Revision)
			if i == 0 && update.Revision == 2 {
				if _, err := v.Set(testReady("false")); err != nil {
					t.Error(err)
				}
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		defer cancel()
	}
	if _, err := v.Set(testReady("true")); err != nil {
		t.Fatal(err)
	}
	for _, seen := range observations {
		if !reflect.DeepEqual(seen, []uint64{2, 3}) {
			t.Fatalf("reentrant delivery out of order: %v", seen)
		}
	}
}
func TestVariableIdentityRetirementAndExactLargeRevision(t *testing.T) {
	registry, err := NewRegistry(1)
	if err != nil {
		t.Fatal(err)
	}
	old := newTestBoolean(t, nil)
	if err := registry.Register(old); err != nil {
		t.Fatal(err)
	}
	events := make(chan Observation, 2)
	_, cancel, err := old.Observe(func(event Observation) { events <- event })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	oldState := old.Snapshot()
	if err := registry.Remove(old.Descriptor().ID); err != nil {
		t.Fatal(err)
	}
	if event := <-events; event.Failure == nil || event.Failure.Code != protocol.CodeGone {
		t.Fatalf("removed instance not terminated: %+v", event)
	}
	if err := registry.Register(old); err == nil {
		t.Fatal("retired instance re-registered")
	}
	replacement := newTestBoolean(t, nil)
	if err := registry.Register(replacement); err != nil {
		t.Fatal(err)
	}
	if _, err := replacement.CompareAndSet(context.Background(), oldState.State.InstanceID, 1, json.RawMessage("true")); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("old identity accepted: %v", err)
	}
	if _, err := old.Set(testReady("true")); !errors.Is(err, ErrNotFound) {
		t.Fatal("retired owner could update")
	}
	update, err := replacement.Apply(9007199254740993, testReady("true"))
	if err != nil {
		t.Fatal(err)
	}
	if update.State.Revision != "9007199254740993" {
		t.Fatal("revision lost precision")
	}
}
func TestRegistryCatalogFailureAndReentrantPublication(t *testing.T) {
	registry, err := NewRegistry(1)
	if err != nil {
		t.Fatal(err)
	}
	value := newTestBoolean(t, nil)
	before := registry.Catalog().Snapshot()
	count := 0
	_, cancel, err := registry.Catalog().Watch(func(VariableUpdate) { count++ })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	savedLimit := registry.catalog.descriptor.Limits.MaxPayloadBytes
	registry.catalog.descriptor.Limits.MaxPayloadBytes = len(registry.catalog.current.payload) + 1
	if err := registry.Register(value); err == nil {
		t.Fatal("catalog envelope overflow accepted")
	}
	if _, ok := registry.Resolve(value.Descriptor().ID); ok {
		t.Fatal("failed registration became visible")
	}
	if count != 0 || !reflect.DeepEqual(before, registry.catalog.Snapshot()) {
		t.Fatal("failed registration published catalog")
	}
	registry.catalog.descriptor.Limits.MaxPayloadBytes = savedLimit
	if err := registry.Register(value); err != nil {
		t.Fatal(err)
	}
	before = registry.catalog.Snapshot()
	count = 0
	registry.catalog.descriptor.Limits.MaxPayloadBytes = 1
	if err := registry.Remove(value.Descriptor().ID); err == nil {
		t.Fatal("catalog removal overflow accepted")
	}
	if _, ok := registry.Resolve(value.Descriptor().ID); !ok {
		t.Fatal("failed removal hid resource")
	}
	if value.Retired() || count != 0 || !reflect.DeepEqual(before, registry.catalog.Snapshot()) {
		t.Fatal("failed removal changed instance or catalog")
	}
	registry.catalog.descriptor.Limits.MaxPayloadBytes = savedLimit
	cancel()
	_, cancel, err = registry.catalog.Watch(func(update VariableUpdate) {
		if update.Revision == 3 {
			other, _ := NewStream(streamDescriptor(1, "observer/events", 32))
			if err := registry.Register(other); err != nil {
				t.Error(err)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if err := registry.Remove(value.Descriptor().ID); err != nil {
		t.Fatal(err)
	}
	if registry.catalog.Snapshot().Revision != 4 {
		t.Fatal("catalog callback could not mutate registry")
	}
}
func TestVariableHandlerPanicAndUnknownOutcomeKeepLastConfirmedState(t *testing.T) {
	var calls int
	v := newTestBoolean(t, func(context.Context, any) (VariableState, error) {
		calls++
		if calls == 1 {
			panic("external failure")
		}
		return VariableState{Status: "unavailable"}, nil
	})
	before := v.Snapshot()
	for range 2 {
		if _, err := v.CompareAndSet(context.Background(), before.State.InstanceID, 1, json.RawMessage("true")); err == nil {
			t.Fatal("unconfirmed write reported success")
		}
	}
	if !reflect.DeepEqual(before, v.Snapshot()) {
		t.Fatal("unconfirmed outcome changed confirmed state")
	}
	if calls != 2 {
		t.Fatal("panic stranded mutation gate")
	}
}

func TestVariableRemovalInvalidatesBeforeBlockedWriterFinishes(t *testing.T) {
	registry, err := NewRegistry(1)
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	v := newTestBoolean(t, func(context.Context, any) (VariableState, error) {
		close(entered)
		<-release
		return testReady("true"), nil
	})
	if err := registry.Register(v); err != nil {
		t.Fatal(err)
	}
	initial := v.Snapshot()
	written := make(chan error, 1)
	go func() {
		_, err := v.CompareAndSet(context.Background(), initial.State.InstanceID, 1, json.RawMessage("true"))
		written <- err
	}()
	<-entered
	removed := make(chan error, 1)
	go func() { removed <- registry.Remove(v.Descriptor().ID) }()
	deadline := time.After(time.Second)
	for {
		if _, exists := registry.Resolve(v.Descriptor().ID); !exists {
			break
		}
		select {
		case <-deadline:
			t.Fatal("removal failed to make progress")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := registry.Register(v); err == nil {
		t.Fatal("old instance re-registered while removal waited for owner")
	}
	close(release)
	if err := <-removed; err != nil {
		t.Fatal(err)
	}
	if err := <-written; !errors.Is(err, ErrNotFound) {
		t.Fatalf("retired writer published state: %v", err)
	}
	if v.Snapshot().Revision != initial.Revision {
		t.Fatal("retired write changed revision")
	}
	if _, exists := registry.Resolve(v.Descriptor().ID); exists {
		t.Fatal("catalog resolves retired old instance")
	}
}
func TestVariableObserverPanicDetachesOnlyFaultyObserver(t *testing.T) {
	v := newTestBoolean(t, nil)
	_, cancel, err := v.Watch(func(VariableUpdate) { panic("faulty observer") })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	observed := []uint64{}
	_, healthyCancel, err := v.Watch(func(update VariableUpdate) { observed = append(observed, update.Revision) })
	if err != nil {
		t.Fatal(err)
	}
	defer healthyCancel()
	for range 3 {
		if _, err := v.Set(testReady("true")); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(observed, []uint64{2, 3, 4}) {
		t.Fatalf("observer panic stalled delivery: %v", observed)
	}
	if v.draining || len(v.pending) != 0 || len(v.watchers) != 1 {
		t.Fatal("failed observer left delivery state stranded")
	}
}
