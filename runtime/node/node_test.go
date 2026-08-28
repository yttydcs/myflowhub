package node

import (
	"context"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/runtime/tree"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestJoinedNodeUsesRemoteVariableAndCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rootIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	_ = trust.Add(rootIdentity.NodeID, rootIdentity.PublicKey)
	_ = trust.Add(childIdentity.NodeID, childIdentity.PublicKey)
	root, err := New(ctx, Config{Identity: rootIdentity, Trust: trust, Policy: auth.AllowAll{}, Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute}})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	child, err := New(ctx, Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}, Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute}})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := root.Listen(network, "root")
	if err != nil {
		t.Fatal(err)
	}
	if err := child.ConnectParent(ctx, network, endpoint, root.ID()); err != nil {
		t.Fatal(err)
	}
	waitRoute(t, root.Tree(), child.ID())
	variableID := protocol.ResourceID{Owner: root.ID(), Name: "status"}
	variable, _ := resource.NewVariable(resource.VariableDescriptor(variableID, "application/octet-stream", "test.raw.v1", "test.read", 64), []byte("ready"))
	if err := root.Registry().Register(variable); err != nil {
		t.Fatal(err)
	}
	commandID := protocol.ResourceID{Owner: root.ID(), Name: "echo"}
	command, _ := resource.NewCommand(resource.CommandDescriptor(commandID, "application/octet-stream", "test.raw.v1", "test.invoke", 64), func(_ context.Context, input []byte) ([]byte, error) {
		return append([]byte("echo:"), input...), nil
	})
	if err := root.Registry().Register(command); err != nil {
		t.Fatal(err)
	}
	remote, err := child.Subscribe(ctx, variableID, time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Cancel()
	if event := receiveRemote(t, remote); event.Kind != subscription.EventSnapshot || string(event.Value) != "ready" {
		t.Fatalf("unexpected snapshot: %#v", event)
	}
	_, _ = variable.Set([]byte("updated"))
	if event := receiveRemote(t, remote); event.Kind != subscription.EventData || string(event.Value) != "updated" {
		t.Fatalf("unexpected update: %#v", event)
	}
	output, err := child.Invoke(ctx, commandID, []byte("hi"))
	if err != nil || string(output) != "echo:hi" {
		t.Fatalf("unexpected command result %q: %v", output, err)
	}
}

func TestReconnectCleansSubscriptionsBoundToReplacedLink(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rootIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	_ = trust.Add(rootIdentity.NodeID, rootIdentity.PublicKey)
	_ = trust.Add(childIdentity.NodeID, childIdentity.PublicKey)
	config := func(identity auth.Identity) Config {
		return Config{
			Identity: identity, Trust: trust, Policy: auth.AllowAll{},
			Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute},
		}
	}
	root, err := New(ctx, config(rootIdentity))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	child, err := New(ctx, config(childIdentity))
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := root.Listen(network, "root-reconnect")
	if err != nil {
		t.Fatal(err)
	}
	if err := child.ConnectParent(ctx, network, endpoint, root.ID()); err != nil {
		t.Fatal(err)
	}
	waitRoute(t, root.Tree(), child.ID())
	variableID := protocol.ResourceID{Owner: child.ID(), Name: "status"}
	variable, _ := resource.NewVariable(resource.VariableDescriptor(variableID, "application/octet-stream", "test.raw.v1", "test.read", 64), []byte("ready"))
	if err := child.Registry().Register(variable); err != nil {
		t.Fatal(err)
	}
	remote, err := root.Subscribe(ctx, variableID, time.Minute, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Cancel()
	if event := receiveRemote(t, remote); event.Kind != subscription.EventSnapshot {
		t.Fatalf("unexpected snapshot: %#v", event)
	}
	if child.subscriptions.Count() != 1 {
		t.Fatalf("want one subscription before reconnect, got %d", child.subscriptions.Count())
	}
	if err := child.ConnectParent(ctx, network, endpoint, root.ID()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && child.subscriptions.Count() != 0 {
		time.Sleep(time.Millisecond)
	}
	if child.subscriptions.Count() != 0 {
		t.Fatalf("subscription from replaced link was retained: %d", child.subscriptions.Count())
	}
}

func waitRoute(t *testing.T, state *tree.State, target protocol.NodeID) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if route, err := state.RouteTo(target); err == nil && route.Kind == tree.DirectionDown {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("route to %d was not installed", target)
}

func receiveRemote(t *testing.T, remote *RemoteSubscription) subscription.Event {
	t.Helper()
	select {
	case event, ok := <-remote.Events:
		if !ok {
			select {
			case err := <-remote.Errors:
				t.Fatalf("remote subscription failed: %v", err)
			default:
				t.Fatal("remote subscription closed")
			}
		}
		return event
	case err := <-remote.Errors:
		t.Fatalf("remote subscription failed: %v", err)
		return subscription.Event{}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for remote event")
		return subscription.Event{}
	}
}
