package node

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestPolicyGenerationChangeExpiresExistingSubscription(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parentIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	_ = trust.Add(parentIdentity.NodeID, parentIdentity.PublicKey)
	_ = trust.Add(childIdentity.NodeID, childIdentity.PublicKey)
	store, _ := keystore.New(t.TempDir())
	policy, _ := auth.LoadPolicyState(store)
	resourceID := protocol.ResourceID{Owner: 1, Name: "state"}
	request := auth.Request{Subject: 2, Action: auth.ActionSubscribe, Resource: resourceID}
	if err := policy.Grant(request); err != nil {
		t.Fatal(err)
	}
	parent, _ := New(ctx, Config{Identity: parentIdentity, Trust: trust, Policy: policy, Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute}})
	defer parent.Close()
	variable, err := resource.NewVariable(resource.VariableDescriptor(resourceID, "test.read", 512), protocol.DataSchemaDefinition{Type: "string"}, resource.VariableState{Value: []byte(`"ready"`), Status: "ready"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := parent.Registry().Register(variable); err != nil {
		t.Fatal(err)
	}
	child, _ := New(ctx, Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}, Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute}})
	defer child.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, _ := parent.Listen(network, "policy-parent")
	if err := child.ConnectParent(ctx, network, endpoint, parent.ID()); err != nil {
		t.Fatal(err)
	}
	remote, err := child.Subscribe(ctx, resourceID, time.Minute, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Cancel()
	if event := receiveRemote(t, remote); event.Kind != subscription.EventSnapshot {
		t.Fatalf("unexpected first event: %+v", event)
	}
	if err := policy.Revoke(request); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-remote.Errors:
		var payload protocol.ErrorPayload
		if !errors.As(err, &payload) || payload.Code != protocol.CodeExpired {
			t.Fatalf("unexpected policy generation error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not expire after policy generation changed")
	}
}

type policyReadingSession struct {
	resource.ActiveSession
	policy  *auth.PolicyState
	aborted chan struct{}
}

func (s *policyReadingSession) Abort(error) { _ = s.policy.Generation(); close(s.aborted) }

func TestPolicyCommitDoesNotHoldItsLockDuringProviderCleanup(t *testing.T) {
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := auth.LoadPolicyState(store)
	if err != nil {
		t.Fatal(err)
	}
	defer policy.Close()
	identity, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	current, err := New(context.Background(), Config{Identity: identity, Trust: auth.NewTrustStore(), Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	session := &policyReadingSession{policy: policy, aborted: make(chan struct{})}
	id := protocol.MustMessageID()
	current.mu.Lock()
	current.resourceSessions[id] = &ownedResourceSession{id: id, policyGeneration: policy.Generation(), session: session}
	current.mu.Unlock()
	committed := make(chan error, 1)
	go func() {
		committed <- policy.Grant(auth.Request{Subject: 2, Action: auth.ActionRead, Resource: protocol.ResourceID{Owner: 1, Name: "other"}})
	}()
	select {
	case err := <-committed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("policy commit called a provider while retaining its lock")
	}
	select {
	case <-session.aborted:
	case <-time.After(time.Second):
		t.Fatal("provider cleanup did not finish")
	}
}
