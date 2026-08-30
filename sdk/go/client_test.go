package sdk

import (
	"context"
	"errors"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestClientRequiresRuntime(t *testing.T) {
	if _, err := NewClient(nil); err == nil {
		t.Fatal("nil runtime was accepted")
	}
	if _, err := NewAttachedClient(nil); err == nil {
		t.Fatal("nil attached runtime was accepted")
	}
}

func TestAttachedClientCloseCannotCloseOrDetachRuntime(t *testing.T) {
	runtime := newLocalNode(t, 91)
	defer runtime.Close()
	variableID := protocol.ResourceID{Owner: runtime.ID(), Name: "test/status"}
	variable, err := resource.NewVariable(resource.VariableDescriptor(variableID, "text/plain", "test.status.v1", "test.read", 64), []byte("ready"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Registry().Register(variable); err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := client.NodeID(); err != nil || id != runtime.ID() {
		t.Fatalf("unexpected attached client NodeID %d: %v", id, err)
	}
	if err := client.Close(); !errors.Is(err, ErrAttachedClientClose) {
		t.Fatalf("attached Close returned %v", err)
	}
	if err := client.Connect(context.Background(), nil, "unused", 1); !errors.Is(err, ErrAttachedClientConnection) {
		t.Fatalf("attached Connect returned %v", err)
	}
	if _, err := client.ConnectManaged(context.Background(), nil, "unused", 1, node.SupervisorConfig{}); !errors.Is(err, ErrAttachedClientConnection) {
		t.Fatalf("attached ConnectManaged returned %v", err)
	}
	select {
	case <-runtime.Done():
		t.Fatal("attached client Close closed the node")
	default:
	}
	event, err := client.Snapshot(context.Background(), variableID)
	if err != nil || string(event.Value) != "ready" {
		t.Fatalf("attached client was not usable after guarded Close: event=%+v err=%v", event, err)
	}
}

func TestLegacyClientCloseStillOwnsRuntime(t *testing.T) {
	runtime := newLocalNode(t, 92)
	client, err := NewClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-runtime.Done():
	default:
		t.Fatal("legacy client Close did not close its node")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("repeated legacy Close returned %v", err)
	}
}

func newLocalNode(t *testing.T, id protocol.NodeID) *node.Node {
	t.Helper()
	identity, err := auth.GenerateIdentity(id)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := node.New(context.Background(), node.Config{
		Identity: identity,
		Trust:    auth.NewTrustStore(),
		Policy:   auth.AllowAll{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}
