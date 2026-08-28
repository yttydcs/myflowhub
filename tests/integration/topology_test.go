package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/runtime/tree"
	"github.com/yttydcs/myflowhub/transport/memory"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type topology struct {
	root, branchA, leafA, branchB, leafB, root2 *node.Node
	root2Endpoint                               link.Endpoint
}

func TestCrossSubtreeVerticalSlice(t *testing.T) {
	tests := []struct {
		name     string
		driver   func() (link.Driver, func())
		endpoint func(string) link.Endpoint
	}{
		{
			name: "memory",
			driver: func() (link.Driver, func()) {
				network := memory.NewNetwork()
				return network, func() { _ = network.Close() }
			},
			endpoint: func(name string) link.Endpoint { return link.Endpoint(name) },
		},
		{
			name:     "tcp",
			driver:   func() (link.Driver, func()) { return tcp.Driver{}, func() {} },
			endpoint: func(string) link.Endpoint { return "127.0.0.1:0" },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, closeDriver := test.driver()
			defer closeDriver()
			runVerticalSlice(t, driver, test.endpoint)
		})
	}
}

func runVerticalSlice(t *testing.T, driver link.Driver, endpoint func(string) link.Endpoint) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	variableID := protocol.ResourceID{Owner: 5, Name: "state"}
	streamID := protocol.ResourceID{Owner: 5, Name: "events"}
	topicID := protocol.ResourceID{Owner: 5, Name: "shared/events"}
	echoID := protocol.ResourceID{Owner: 5, Name: "echo"}
	deniedID := protocol.ResourceID{Owner: 5, Name: "denied"}
	slowID := protocol.ResourceID{Owner: 5, Name: "slow"}
	policy := auth.NewStaticPolicy()
	for _, resourceID := range []protocol.ResourceID{variableID, streamID} {
		policy.Allow(auth.Request{Subject: 3, Action: auth.ActionSubscribe, Resource: resourceID})
	}
	policy.Allow(auth.Request{Subject: 2, Action: auth.ActionSubscribe, Resource: topicID})
	policy.Allow(auth.Request{Subject: 2, Action: auth.ActionPublish, Resource: topicID})
	for _, resourceID := range []protocol.ResourceID{echoID, slowID} {
		policy.Allow(auth.Request{Subject: 3, Action: auth.ActionInvoke, Resource: resourceID})
	}
	graph := buildTopology(t, ctx, driver, endpoint, policy)
	defer closeTopology(graph)
	variable, _ := resource.NewVariable(resource.VariableDescriptor(variableID, "application/octet-stream", "test.raw.v1", "test.read", 128), []byte("initial"))
	stream, _ := resource.NewStream(resource.StreamDescriptor(streamID, "application/octet-stream", "test.raw.v1", "test.read", 128))
	topic, _ := resource.NewTopic(resource.TopicDescriptor(topicID, "application/octet-stream", "test.raw.v1", "test.publish", "test.subscribe", 128), resource.TopicConfig{})
	echo, _ := resource.NewCommand(resource.CommandDescriptor(echoID, "application/octet-stream", "test.raw.v1", "test.invoke", 128), func(_ context.Context, input []byte) ([]byte, error) {
		return append([]byte("ok:"), input...), nil
	})
	denied, _ := resource.NewCommand(resource.CommandDescriptor(deniedID, "application/octet-stream", "test.raw.v1", "test.invoke", 128), func(_ context.Context, input []byte) ([]byte, error) {
		return input, nil
	})
	slow, _ := resource.NewCommand(resource.CommandDescriptor(slowID, "application/octet-stream", "test.raw.v1", "test.invoke", 128), func(ctx context.Context, _ []byte) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	for _, value := range []resource.Resource{variable, stream, topic, echo, denied, slow} {
		if err := graph.leafB.Registry().Register(value); err != nil {
			t.Fatal(err)
		}
	}
	variableSub, err := graph.leafA.Subscribe(ctx, variableID, 5*time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	if event := receiveEvent(t, variableSub); event.Kind != subscription.EventSnapshot || string(event.Value) != "initial" {
		t.Fatalf("unexpected variable snapshot: %#v", event)
	}
	_, _ = variable.Set([]byte("changed"))
	if event := receiveEvent(t, variableSub); event.Kind != subscription.EventData || string(event.Value) != "changed" {
		t.Fatalf("unexpected variable update: %#v", event)
	}
	streamSub, err := graph.leafA.Subscribe(ctx, streamID, 5*time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer streamSub.Cancel()
	_, _ = stream.Publish([]byte("one"))
	if event := receiveEvent(t, streamSub); event.Kind != subscription.EventData || event.Sequence != 1 {
		t.Fatalf("unexpected stream event: %#v", event)
	}
	_, _ = stream.Apply(3, []byte("three"))
	if event := receiveEvent(t, streamSub); event.Kind != subscription.EventGap || event.GapFrom != 2 || event.GapTo != 2 {
		t.Fatalf("unexpected stream gap: %#v", event)
	}
	if event := receiveEvent(t, streamSub); event.Kind != subscription.EventData || event.Sequence != 3 {
		t.Fatalf("unexpected post-gap event: %#v", event)
	}
	topicSub, err := graph.root.SubscribeCapability(ctx, topicID, protocol.CapabilitySubscribe, 5*time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer topicSub.Cancel()
	topicSub2, err := graph.branchA.SubscribeCapability(ctx, topicID, protocol.CapabilitySubscribe, 5*time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer topicSub2.Cancel()
	if _, err := graph.branchA.Operate(ctx, topicID, protocol.CapabilityPublish, "test.raw.v1", []byte("branch")); err != nil {
		t.Fatalf("first topic publisher: %v", err)
	}
	if _, err := graph.root.Operate(ctx, topicID, protocol.CapabilityPublish, "test.raw.v1", []byte("root")); err != nil {
		t.Fatalf("second topic publisher: %v", err)
	}
	firstTopic := receiveEvent(t, topicSub)
	secondTopic := receiveEvent(t, topicSub)
	firstTopic2 := receiveEvent(t, topicSub2)
	secondTopic2 := receiveEvent(t, topicSub2)
	if firstTopic.Publisher != graph.branchA.ID() || firstTopic.PublisherSequence != 1 || firstTopic.Sequence != 1 || string(firstTopic.Value) != "branch" {
		t.Fatalf("unexpected first topic event: %#v", firstTopic)
	}
	if secondTopic.Publisher != graph.root.ID() || secondTopic.PublisherSequence != 1 || secondTopic.Sequence != 2 || string(secondTopic.Value) != "root" {
		t.Fatalf("unexpected second topic event: %#v", secondTopic)
	}
	if firstTopic2.Publisher != firstTopic.Publisher || firstTopic2.Sequence != firstTopic.Sequence || secondTopic2.Publisher != secondTopic.Publisher || secondTopic2.Sequence != secondTopic.Sequence {
		t.Fatalf("topic subscribers diverged: %#v %#v / %#v %#v", firstTopic, secondTopic, firstTopic2, secondTopic2)
	}
	if _, err := graph.leafA.Operate(ctx, topicID, protocol.CapabilityPublish, "test.raw.v1", []byte("denied")); errorCode(err) != protocol.CodeForbidden {
		t.Fatalf("expected forbidden topic publisher, got %v", err)
	}
	if deniedSub, err := graph.leafA.SubscribeCapability(ctx, topicID, protocol.CapabilitySubscribe, 5*time.Second, 1); errorCode(err) != protocol.CodeForbidden {
		if deniedSub != nil {
			deniedSub.Cancel()
		}
		t.Fatalf("expected forbidden topic subscriber, got %v", err)
	}
	output, err := graph.leafA.Invoke(ctx, echoID, []byte("call"))
	if err != nil || string(output) != "ok:call" {
		t.Fatalf("unexpected command result %q: %v", output, err)
	}
	if _, err := graph.leafA.Invoke(ctx, deniedID, nil); errorCode(err) != protocol.CodeForbidden {
		t.Fatalf("expected forbidden command, got %v", err)
	}
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer timeoutCancel()
	if _, err := graph.leafA.Invoke(timeoutCtx, slowID, nil); err == nil {
		t.Fatal("slow command did not time out")
	}
	oldParent, ok := graph.leafA.Tree().Parent()
	if !ok || oldParent.Node != graph.branchA.ID() {
		t.Fatalf("unexpected old parent: %#v", oldParent)
	}
	if err := graph.leafA.ConnectParent(ctx, driver, graph.root2Endpoint, graph.root2.ID()); err != nil {
		t.Fatalf("reparent: %v", err)
	}
	select {
	case err := <-variableSub.Errors:
		if err == nil {
			t.Fatal("old-topology subscription ended without a reason")
		}
	case <-time.After(time.Second):
		t.Fatal("reparent did not invalidate old subscription")
	}
	if err := graph.leafA.Tree().ValidateParentControl(graph.branchA.ID(), oldParent.Epoch); !errors.Is(err, tree.ErrNotReachable) {
		t.Fatalf("old parent retained control: %v", err)
	}
	waitUnreachable(t, graph.root.Tree(), graph.leafA.ID())
}

func buildTopology(t *testing.T, ctx context.Context, driver link.Driver, endpoint func(string) link.Endpoint, rootPolicy auth.Policy) topology {
	t.Helper()
	identities := make([]auth.Identity, 6)
	trust := auth.NewTrustStore()
	for index := range identities {
		identity, err := auth.GenerateIdentity(protocol.NodeID(index + 1))
		if err != nil {
			t.Fatal(err)
		}
		identities[index] = identity
		if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
			t.Fatal(err)
		}
	}
	newNode := func(index int, policy auth.Policy) *node.Node {
		value, err := node.New(ctx, node.Config{
			Identity: identities[index-1], Trust: trust, Policy: policy,
			Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute, DefaultQueue: 2, MaxQueue: 16},
			Session:       link.SessionConfig{ControlQueue: 8, DataQueue: 8, InboundQueue: 32},
		})
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	graph := topology{
		root: newNode(1, rootPolicy), branchA: newNode(2, auth.AllowAll{}), leafA: newNode(3, auth.AllowAll{}),
		branchB: newNode(4, auth.AllowAll{}), leafB: newNode(5, auth.AllowAll{}), root2: newNode(6, auth.AllowAll{}),
	}
	rootEndpoint, err := graph.root.Listen(driver, endpoint("root"))
	if err != nil {
		t.Fatal(err)
	}
	branchAEndpoint, err := graph.branchA.Listen(driver, endpoint("branch-a"))
	if err != nil {
		t.Fatal(err)
	}
	branchBEndpoint, err := graph.branchB.Listen(driver, endpoint("branch-b"))
	if err != nil {
		t.Fatal(err)
	}
	graph.root2Endpoint, err = graph.root2.Listen(driver, endpoint("root-2"))
	if err != nil {
		t.Fatal(err)
	}
	for _, connection := range []struct {
		child    *node.Node
		address  link.Endpoint
		parentID protocol.NodeID
	}{
		{graph.branchA, rootEndpoint, graph.root.ID()},
		{graph.branchB, rootEndpoint, graph.root.ID()},
		{graph.leafA, branchAEndpoint, graph.branchA.ID()},
		{graph.leafB, branchBEndpoint, graph.branchB.ID()},
	} {
		if err := connection.child.ConnectParent(ctx, driver, connection.address, connection.parentID); err != nil {
			t.Fatal(err)
		}
	}
	waitDownRoute(t, graph.root.Tree(), graph.leafA.ID())
	waitDownRoute(t, graph.root.Tree(), graph.leafB.ID())
	return graph
}

func closeTopology(graph topology) {
	for _, value := range []*node.Node{graph.leafA, graph.leafB, graph.branchA, graph.branchB, graph.root, graph.root2} {
		if value != nil {
			_ = value.Close()
		}
	}
}

func waitDownRoute(t *testing.T, state *tree.State, target protocol.NodeID) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if route, err := state.RouteTo(target); err == nil && route.Kind == tree.DirectionDown {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("down route to %d was not installed", target)
}

func waitUnreachable(t *testing.T, state *tree.State, target protocol.NodeID) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := state.RouteTo(target); errors.Is(err, tree.ErrNotReachable) {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("route to %d was not withdrawn", target)
}

func receiveEvent(t *testing.T, remote *node.RemoteSubscription) subscription.Event {
	t.Helper()
	select {
	case event, ok := <-remote.Events:
		if !ok {
			t.Fatal("remote subscription closed")
		}
		return event
	case err := <-remote.Errors:
		t.Fatalf("remote subscription error: %v", err)
		return subscription.Event{}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for remote event")
		return subscription.Event{}
	}
}

func errorCode(err error) protocol.ErrorCode {
	if err == nil {
		return ""
	}
	var payload protocol.ErrorPayload
	if errors.As(err, &payload) {
		return payload.Code
	}
	return protocol.ErrorCode(fmt.Sprintf("%T", err))
}
