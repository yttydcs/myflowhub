package sdk

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestClientCatalogSubscriptionInvokeAndTypedErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	root, child, network := connectedSDKNodes(t, ctx, "sdk-contract")
	defer network.Close()
	defer root.Close()
	defer child.Close()

	statusID := protocol.ResourceID{Owner: root.ID(), Name: "test/status"}
	status, err := resource.NewVariable(resource.VariableDescriptor(statusID, "application/json", "test.status.v1", "test.read", 128), []byte(`{"version":1,"state":"ready"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Registry().Register(status); err != nil {
		t.Fatal(err)
	}
	writableID := protocol.ResourceID{Owner: root.ID(), Name: "test/writable"}
	writable, err := resource.NewVariable(resource.WritableVariableDescriptor(writableID, "application/json", "test.settings.v1", "test.read", "test.write", 256), []byte(`{"enabled":false}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Registry().Register(writable); err != nil {
		t.Fatal(err)
	}
	echoID := protocol.ResourceID{Owner: root.ID(), Name: "test/echo"}
	echo, _ := resource.NewCommand(resource.CommandDescriptor(echoID, "application/octet-stream", "test.raw.v1", "test.invoke", 128), func(_ context.Context, input []byte) ([]byte, error) {
		return append([]byte("echo:"), input...), nil
	})
	if err := root.Registry().Register(echo); err != nil {
		t.Fatal(err)
	}
	retryID := protocol.ResourceID{Owner: root.ID(), Name: "test/retry"}
	retry, _ := resource.NewCommand(resource.CommandDescriptor(retryID, "application/octet-stream", "test.raw.v1", "test.invoke", 128), func(context.Context, []byte) ([]byte, error) {
		return nil, command.Retryable(errors.New("try later"))
	})
	if err := root.Registry().Register(retry); err != nil {
		t.Fatal(err)
	}
	malformedID := protocol.ResourceID{Owner: root.ID(), Name: "test/malformed"}
	malformed, _ := resource.NewVariable(resource.VariableDescriptor(malformedID, "application/json", "test.malformed.v1", "test.read", 128), []byte(`{"version":1,"unexpected":true}`))
	if err := root.Registry().Register(malformed); err != nil {
		t.Fatal(err)
	}

	client, err := NewAttachedClient(child)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := client.Catalog(ctx, root.ID())
	if err != nil {
		t.Fatal(err)
	}
	if !catalogHas(catalog, statusID.Name, protocol.ResourceTypeVariable) || !catalogHas(catalog, echoID.Name, protocol.ResourceTypeCommand) {
		t.Fatalf("catalog is missing SDK test resources: %+v", catalog.Resources)
	}

	current, err := client.Subscribe(ctx, statusID, time.Minute, 4)
	if err != nil {
		t.Fatal(err)
	}
	if event := receiveEvent(t, current); event.Kind != EventSnapshot || string(event.Value) != `{"version":1,"state":"ready"}` {
		t.Fatalf("unexpected snapshot: %+v", event)
	}
	if _, err := status.Set([]byte(`{"version":1,"state":"updated"}`)); err != nil {
		t.Fatal(err)
	}
	if event := receiveEvent(t, current); event.Kind != EventData || event.Revision != 2 {
		t.Fatalf("unexpected update: %+v", event)
	}
	current.Cancel()
	current.Cancel()
	select {
	case _, ok := <-current.Events:
		if ok {
			t.Fatal("subscription delivered an event after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not close after cancellation")
	}
	writableSub, err := client.Subscribe(ctx, writableID, time.Minute, 2)
	if err != nil {
		t.Fatal(err)
	}
	writableSnapshot := receiveEvent(t, writableSub)
	writableSub.Cancel()
	if _, err := client.WriteVariable(ctx, writableID, writableSnapshot.Revision, []byte(`{"enabled":true}`)); err != nil {
		t.Fatalf("conditional SDK write failed: %v", err)
	}
	if string(writable.Snapshot().Value) != `{"enabled":true}` {
		t.Fatalf("conditional SDK write did not update owner: %s", writable.Snapshot().Value)
	}
	_, err = client.WriteVariable(ctx, writableID, writableSnapshot.Revision, []byte(`{"enabled":false}`))
	assertSDKError(t, err, protocol.CodeConflict, false)

	output, err := client.Invoke(ctx, echoID, []byte("hello"))
	if err != nil || string(output) != "echo:hello" {
		t.Fatalf("unexpected invoke result %q: %v", output, err)
	}
	_, err = client.Invoke(ctx, retryID, []byte("retry"))
	assertSDKError(t, err, protocol.CodeInternal, true)
	_, err = client.Invoke(ctx, protocol.ResourceID{Owner: root.ID(), Name: "test/missing"}, []byte("missing"))
	assertSDKError(t, err, protocol.CodeNotFound, false)

	var health protocol.ManagementHealthV1
	err = client.DecodeSnapshot(ctx, malformedID, &health)
	assertSDKError(t, err, protocol.CodeMalformed, false)
	var notification protocol.NotificationEventV1
	err = client.InvokePayload(ctx, echoID, &protocol.NotificationPublishV1{}, &notification)
	assertSDKError(t, err, protocol.CodeMalformed, false)
}

func connectedSDKNodes(t *testing.T, ctx context.Context, endpoint string) (*node.Node, *node.Node, *memory.Network) {
	t.Helper()
	rootIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	_ = trust.Add(rootIdentity.NodeID, rootIdentity.PublicKey)
	_ = trust.Add(childIdentity.NodeID, childIdentity.PublicKey)
	config := func(identity auth.Identity) node.Config {
		return node.Config{Identity: identity, Trust: trust, Policy: auth.AllowAll{}, Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute}}
	}
	root, err := node.New(ctx, config(rootIdentity))
	if err != nil {
		t.Fatal(err)
	}
	child, err := node.New(ctx, config(childIdentity))
	if err != nil {
		_ = root.Close()
		t.Fatal(err)
	}
	network := memory.NewNetwork()
	address, err := root.Listen(network, link.Endpoint(endpoint))
	if err != nil {
		_ = child.Close()
		_ = root.Close()
		network.Close()
		t.Fatal(err)
	}
	if err := child.ConnectParent(ctx, network, address, root.ID()); err != nil {
		_ = child.Close()
		_ = root.Close()
		network.Close()
		t.Fatal(err)
	}
	return root, child, network
}

func catalogHas(catalog protocol.ResourceCatalogV2, name string, typeID protocol.ResourceTypeID) bool {
	for _, descriptor := range catalog.Resources {
		if descriptor.ID.Name == name && descriptor.Type == typeID {
			return true
		}
	}
	return false
}

func receiveEvent(t *testing.T, current *Subscription) Event {
	t.Helper()
	select {
	case event, ok := <-current.Events:
		if !ok {
			t.Fatal("subscription closed")
		}
		return event
	case err := <-current.Errors:
		t.Fatalf("subscription failed: %v", err)
		return Event{}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SDK event")
		return Event{}
	}
}

func assertSDKError(t *testing.T, err error, code protocol.ErrorCode, retryable bool) {
	t.Helper()
	var value *Error
	if !errors.As(err, &value) {
		t.Fatalf("expected SDK Error, got %T: %v", err, err)
	}
	if value.Code != code || value.Retryable != retryable {
		t.Fatalf("unexpected SDK Error: %+v", value)
	}
}
