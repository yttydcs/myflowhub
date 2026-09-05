package subscription

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestDeliveryFailureDiscardsQueuedData(t *testing.T) {
	id := protocol.ResourceID{Owner: 1, Name: "test"}
	d := newDelivery(context.Background(), id, 1)
	defer d.stop()
	d.enqueueSnapshot(Event{Kind: EventSnapshot, Value: []byte("snapshot")})
	d.enqueueVariable(Event{Kind: EventData, Revision: 2, Value: []byte("latest")})
	d.enqueueStream(Event{Kind: EventData, Sequence: 1})
	d.enqueueStream(Event{Kind: EventData, Sequence: 3})
	failure := protocol.ErrorPayload{Code: protocol.CodeOverflow, Message: "too many nodes", Retryable: true, Details: map[string]string{"depth": "0"}}
	d.fail(Event{Kind: EventFailure, Resource: id, Failure: &failure})
	failure.Code = protocol.CodeInternal
	failure.Details["depth"] = "mutated"
	d.fail(Event{Kind: EventFailure, Failure: &failure})
	d.expire("lease_expired")
	d.enqueueSnapshot(Event{Kind: EventSnapshot})
	d.enqueueVariable(Event{Kind: EventData, Revision: 3})
	d.enqueueStream(Event{Kind: EventData, Sequence: 4})
	event, ok, terminal := d.next()
	if !ok || !terminal || event.Kind != EventFailure || event.Resource != id || event.Failure == nil ||
		event.Failure.Code != protocol.CodeOverflow || event.Failure.Message != "too many nodes" ||
		!event.Failure.Retryable || event.Failure.Details["depth"] != "0" {
		t.Fatalf("failure was lost or aliased: %+v, ok=%v terminal=%v", event, ok, terminal)
	}
	if event, ok, _ := d.next(); ok {
		t.Fatalf("queued data survived failure: %+v", event)
	}
}

func TestDeliveryFailureInterruptsBlockedValue(t *testing.T) {
	d := newDelivery(context.Background(), protocol.ResourceID{Owner: 1, Name: "test"}, 1)
	defer d.stop()
	d.enqueueSnapshot(Event{Kind: EventSnapshot, Value: []byte("stale")})
	done := make(chan struct{})
	d.start(func() { close(done) })
	awaitFailureCondition(t, func() bool {
		d.mu.Lock()
		defer d.mu.Unlock()
		return len(d.pending) == 0
	})
	// No reader exists while the old snapshot is blocked on the output channel.
	d.fail(Event{Kind: EventFailure, Failure: &protocol.ErrorPayload{Code: protocol.CodeGone, Message: "provider stopped"}})
	awaitFailureCondition(t, func() bool {
		d.mu.Lock()
		defer d.mu.Unlock()
		return d.terminal == nil
	})
	if event := receive(t, d.out); event.Kind != EventFailure || event.Failure.Code != protocol.CodeGone {
		t.Fatalf("blocked value preceded terminal failure: %+v", event)
	}
	assertFailureClosed(t, d.out)
	select {
	case <-done:
	default:
		t.Fatal("delivery did not finish cleanup")
	}
	if d.ctx.Err() == nil {
		t.Fatal("delivery context was retained after terminal close")
	}
}

func TestObservationFailureClosesSubscriptionAndCleansUp(t *testing.T) {
	for _, initialFailure := range []bool{false, true} {
		name := "active"
		if initialFailure {
			name = "initial"
		}
		t.Run(name, func(t *testing.T) {
			manager, observable, id := newFailureManager(t)
			failure := protocol.ErrorPayload{Code: protocol.CodeOverflow, Message: "topology exceeds limit", Retryable: true, Details: map[string]string{"limit": "4096"}}
			if initialFailure {
				observable.initial = &resource.Observation{Snapshot: true, Failure: &failure}
			}
			sub, err := manager.Subscribe(request(id, time.Minute, 2))
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Cancel()
			var delivery *delivery
			if !initialFailure {
				if event := receive(t, sub.Events); event.Kind != EventSnapshot {
					t.Fatalf("missing initial snapshot: %+v", event)
				}
				manager.mu.Lock()
				delivery = manager.entries[sub.ID].delivery
				manager.mu.Unlock()
				observable.emit(resource.Observation{Failure: &failure})
			}
			event := receive(t, sub.Events)
			if event.Kind != EventFailure || event.Capability != protocol.CapabilitySubscribe || !reflect.DeepEqual(event.Failure, &failure) {
				t.Fatalf("failure payload changed: %+v", event)
			}
			assertFailureClosed(t, sub.Events)
			if manager.Count() != 0 || len(manager.Interests(id)) != 0 {
				t.Fatal("terminal subscription retained its entry or interest")
			}
			select {
			case <-observable.cancelled:
			default:
				t.Fatal("provider observer was not cancelled")
			}
			if delivery != nil && delivery.ctx.Err() == nil {
				t.Fatal("delivery and lease contexts were not cancelled")
			}
		})
	}
}

func TestObservationFailureRejectsConcurrentLateData(t *testing.T) {
	manager, observable, id := newFailureManager(t)
	sub, err := manager.Subscribe(request(id, time.Minute, 2))
	if err != nil {
		t.Fatal(err)
	}
	_ = receive(t, sub.Events)
	var writers sync.WaitGroup
	start := make(chan struct{})
	for range 4 {
		writers.Add(1)
		go func() {
			defer writers.Done()
			<-start
			for i := range 100 {
				observable.emit(resource.Observation{Revision: uint64(i + 1), Value: []byte("late")})
			}
		}()
	}
	observable.emit(resource.Observation{Failure: &protocol.ErrorPayload{Code: protocol.CodeOverflow, Message: "too large"}})
	close(start)
	writers.Wait()
	if event := receive(t, sub.Events); event.Kind != EventFailure {
		t.Fatalf("late data replaced failure: %+v", event)
	}
	assertFailureClosed(t, sub.Events)
}

func TestObservationRegistrationFailureLeavesNoSubscriptionState(t *testing.T) {
	manager, observable, id := newFailureManager(t)
	observable.observeErr = protocol.ErrPayloadTooLarge
	if _, err := manager.Subscribe(request(id, time.Minute, 2)); !errors.Is(err, protocol.ErrPayloadTooLarge) {
		t.Fatalf("registration error changed: %v", err)
	}
	if manager.Count() != 0 || len(manager.Interests(id)) != 0 {
		t.Fatal("failed registration retained subscription state")
	}
}

type failureObservable struct {
	*resource.Variable
	mu         sync.Mutex
	observer   func(resource.Observation)
	initial    *resource.Observation
	observeErr error
	cancelled  chan struct{}
}

func (o *failureObservable) Observe(observer func(resource.Observation)) (*resource.Observation, func(), error) {
	if o.observeErr != nil {
		return nil, nil, o.observeErr
	}
	o.mu.Lock()
	o.observer = observer
	o.mu.Unlock()
	initial := o.initial
	if initial == nil {
		snapshot := o.Snapshot()
		initial = &resource.Observation{Snapshot: true, Revision: snapshot.Revision, Value: snapshot.Value}
	}
	var once sync.Once
	return initial, func() {
		once.Do(func() {
			o.mu.Lock()
			o.observer = nil
			o.mu.Unlock()
			close(o.cancelled)
		})
	}, nil
}

func (o *failureObservable) emit(observation resource.Observation) {
	o.mu.Lock()
	observer := o.observer
	o.mu.Unlock()
	if observer != nil {
		observer(observation)
	}
}

func newFailureManager(t *testing.T) (*Manager, *failureObservable, protocol.ResourceID) {
	t.Helper()
	manager, value, id := newManager(t, protocol.ResourceTypeVariable, time.Minute)
	if err := manager.registry.Remove(id); err != nil {
		t.Fatal(err)
	}
	observable := &failureObservable{Variable: value.(*resource.Variable), cancelled: make(chan struct{})}
	if err := manager.registry.Register(observable); err != nil {
		t.Fatal(err)
	}
	return manager, observable, id
}

func assertFailureClosed(t *testing.T, events <-chan Event) {
	t.Helper()
	select {
	case event, ok := <-events:
		if ok {
			t.Fatalf("event after terminal failure: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not close after failure")
	}
}

func awaitFailureCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for delivery state")
		}
		runtime.Gosched()
	}
}
