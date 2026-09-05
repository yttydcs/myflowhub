package node

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestSubscriptionFailurePreservesWireErrorAndCloses(t *testing.T) {
	for _, code := range []protocol.ErrorCode{protocol.CodeOverflow, protocol.CodeGone} {
		t.Run(string(code), func(t *testing.T) {
			ctx, root, child, observable, id := newSubscriptionFailureNodes(t)
			remote, err := child.Subscribe(ctx, id, time.Minute, 4)
			if err != nil {
				t.Fatal(err)
			}
			defer remote.Cancel()
			if event := receiveRemote(t, remote); event.Kind != subscription.EventSnapshot {
				t.Fatalf("missing initial snapshot: %+v", event)
			}
			want := protocol.ErrorPayload{Code: code, Message: "provider cannot publish snapshot", Retryable: true, Details: map[string]string{"limit": "4096"}}
			observable.emit(resource.Observation{Failure: &want})
			select {
			case err, ok := <-remote.Errors:
				var got protocol.ErrorPayload
				if !ok || !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
					t.Fatalf("wire failure lost code or metadata: %v, want %+v", err, want)
				}
			case <-ctx.Done():
				t.Fatal("remote subscription did not receive failure")
			}
			select {
			case event, ok := <-remote.Events:
				if ok {
					t.Fatalf("data after failure: %+v", event)
				}
			case <-ctx.Done():
				t.Fatal("remote events remained open")
			}
			select {
			case err, ok := <-remote.Errors:
				if ok {
					t.Fatalf("second terminal error: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("remote errors remained open")
			}
			select {
			case <-observable.cancelled:
			case <-ctx.Done():
				t.Fatal("server observer was retained")
			}
			if root.subscriptions.Count() != 0 || len(root.subscriptions.Interests(id)) != 0 {
				t.Fatal("server retained terminal subscription state")
			}
			child.mu.RLock()
			_, pending := child.pending[remote.ID]
			child.mu.RUnlock()
			if pending {
				t.Fatal("client retained terminal pending request")
			}
		})
	}
}

func TestNewSubscriptionOverflowKeepsWireCode(t *testing.T) {
	ctx, root, child, observable, id := newSubscriptionFailureNodes(t)
	observable.mu.Lock()
	observable.observeErr = protocol.ErrPayloadTooLarge
	observable.mu.Unlock()
	remote, err := child.Subscribe(ctx, id, time.Minute, 4)
	if remote != nil {
		remote.Cancel()
		t.Fatal("overflow returned a live subscription")
	}
	var failure protocol.ErrorPayload
	if !errors.As(err, &failure) || failure.Code != protocol.CodeOverflow {
		t.Fatalf("new subscription overflow code changed: %v", err)
	}
	if root.subscriptions.Count() != 0 || len(root.subscriptions.Interests(id)) != 0 {
		t.Fatal("rejected subscription retained server state")
	}
}

type subscriptionFailureObservable struct {
	*resource.Variable
	mu         sync.Mutex
	observer   func(resource.Observation)
	observeErr error
	cancelled  chan struct{}
}

func (o *subscriptionFailureObservable) Observe(observer func(resource.Observation)) (*resource.Observation, func(), error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.observeErr != nil {
		return nil, nil, o.observeErr
	}
	o.observer = observer
	snapshot := o.Snapshot()
	var once sync.Once
	return &resource.Observation{Snapshot: true, Revision: snapshot.Revision, Schema: "test.raw.v1", Value: snapshot.Value}, func() {
		once.Do(func() {
			o.mu.Lock()
			o.observer = nil
			o.mu.Unlock()
			close(o.cancelled)
		})
	}, nil
}

func (o *subscriptionFailureObservable) emit(observation resource.Observation) {
	o.mu.Lock()
	observer := o.observer
	o.mu.Unlock()
	if observer != nil {
		observer(observation)
	}
}

func newSubscriptionFailureNodes(t *testing.T) (context.Context, *Node, *Node, *subscriptionFailureObservable, protocol.ResourceID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	trust := auth.NewTrustStore()
	newNode := func(id protocol.NodeID) *Node {
		identity, err := auth.GenerateIdentity(id)
		if err != nil {
			t.Fatal(err)
		}
		if err := trust.Add(id, identity.PublicKey); err != nil {
			t.Fatal(err)
		}
		value, err := New(ctx, Config{Identity: identity, Trust: trust, Policy: auth.AllowAll{}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = value.Close() })
		return value
	}
	root, child := newNode(1), newNode(2)
	network := memory.NewNetwork()
	t.Cleanup(func() { _ = network.Close() })
	endpoint, err := root.Listen(network, "failure-root")
	if err != nil {
		t.Fatal(err)
	}
	if err := child.ConnectParent(ctx, network, endpoint, root.ID()); err != nil {
		t.Fatal(err)
	}
	waitRoute(t, root.Tree(), child.ID())
	id := protocol.ResourceID{Owner: root.ID(), Name: "failure-state"}
	variable, err := resource.NewVariable(resource.VariableDescriptor(id, "application/octet-stream", "test.raw.v1", "test.read", 64), []byte("ready"))
	if err != nil {
		t.Fatal(err)
	}
	observable := &subscriptionFailureObservable{Variable: variable, cancelled: make(chan struct{})}
	if err := root.Registry().Register(observable); err != nil {
		t.Fatal(err)
	}
	return ctx, root, child, observable, id
}
