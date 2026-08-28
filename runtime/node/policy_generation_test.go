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
	variable, _ := resource.NewVariable(resource.VariableDescriptor(resourceID, "application/octet-stream", "test.raw.v1", "test.read", 64), []byte("ready"))
	_ = parent.Registry().Register(variable)
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
