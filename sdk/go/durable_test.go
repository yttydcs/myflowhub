package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestDurableVariableSubscriptionRecoversAfterParentRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	parentIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	_ = trust.Add(parentIdentity.NodeID, parentIdentity.PublicKey)
	_ = trust.Add(childIdentity.NodeID, childIdentity.PublicKey)
	network := memory.NewNetwork()
	defer network.Close()
	resourceID := protocol.ResourceID{Owner: 1, Name: "state"}
	startParent := func(value string) *node.Node {
		parent, err := node.New(ctx, node.Config{Identity: parentIdentity, Trust: trust, Policy: auth.AllowAll{}, Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute}})
		if err != nil {
			t.Fatal(err)
		}
		variable, _ := resource.NewVariable(resource.VariableDescriptor(resourceID, "application/octet-stream", "test.raw.v1", "test.read", 64), []byte(value))
		if err := parent.Registry().Register(variable); err != nil {
			t.Fatal(err)
		}
		if _, err := parent.Listen(network, "durable-parent"); err != nil {
			t.Fatal(err)
		}
		return parent
	}
	parent := startParent("one")
	child, _ := node.New(ctx, node.Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}, Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute}})
	defer child.Close()
	client, _ := NewAttachedClient(child)
	supervisor, err := child.SuperviseParent(ctx, network, "durable-parent", 1, node.SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Stop()
	waitSDKConnected(t, supervisor)
	durable, err := client.SubscribeDurable(ctx, supervisor, resourceID, time.Minute, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer durable.Cancel()
	select {
	case <-durable.Ready:
	case <-time.After(2 * time.Second):
		t.Fatal("durable subscription did not become ready")
	}
	if event := receiveDurable(t, durable); event.Kind != EventSnapshot || string(event.Value) != "one" {
		t.Fatalf("unexpected first snapshot: %+v", event)
	}
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}
	parent = startParent("two")
	defer parent.Close()
	if event := receiveDurable(t, durable); event.Kind != EventSnapshot || string(event.Value) != "two" {
		t.Fatalf("unexpected recovered snapshot: %+v", event)
	}
}

func receiveDurable(t *testing.T, durable *DurableSubscription) Event {
	t.Helper()
	select {
	case event, ok := <-durable.Events:
		if !ok {
			select {
			case err := <-durable.Errors:
				t.Fatalf("durable subscription failed: %v", err)
			default:
				t.Fatal("durable subscription closed")
			}
		}
		return event
	case err := <-durable.Errors:
		t.Fatalf("durable subscription failed: %v", err)
		return Event{}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for durable event")
		return Event{}
	}
}

func waitSDKConnected(t *testing.T, supervisor *node.ParentSupervisor) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if supervisor.Snapshot().State == node.ConnectionConnected {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("parent did not connect: %+v", supervisor.Snapshot())
}
