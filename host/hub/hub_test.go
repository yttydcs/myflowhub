package hub

import (
	"bytes"
	"context"
	"testing"

	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/memory"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func TestStartAndClose(t *testing.T) {
	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	_ = trust.Add(identity.NodeID, identity.PublicKey)
	network := memory.NewNetwork()
	defer network.Close()
	value, err := Start(context.Background(), Config{Node: node.Config{Identity: identity, Trust: trust}, Driver: network, Endpoint: "hub"})
	if err != nil {
		t.Fatal(err)
	}
	if value.Endpoint != "hub" {
		t.Fatalf("unexpected endpoint %q", value.Endpoint)
	}
	if err := value.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStartSupportsMultipleListeners(t *testing.T) {
	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	_ = trust.Add(identity.NodeID, identity.PublicKey)
	network := memory.NewNetwork()
	defer network.Close()
	value, err := Start(context.Background(), Config{Node: node.Config{Identity: identity, Trust: trust}, Listeners: []ListenerConfig{
		{Driver: network, Endpoint: "memory-hub"},
		{Driver: tcp.Driver{}, Endpoint: "127.0.0.1:0"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer value.Close()
	if len(value.Endpoints) != 2 || value.Endpoints[0] != "memory-hub" || value.Endpoints[1] == "127.0.0.1:0" {
		t.Fatalf("unexpected endpoints: %#v", value.Endpoints)
	}
}

func TestPersistentHubKeepsIdentityAndRollsBackPartialListeners(t *testing.T) {
	directory := t.TempDir()
	network := memory.NewNetwork()
	defer network.Close()
	config := PersistentConfig{StateDirectory: directory, NodeID: 1, Listeners: []ListenerConfig{{Driver: network, Endpoint: "persistent"}, {Driver: tcp.Driver{}, Endpoint: "127.0.0.1:0"}}}
	first, err := StartPersistent(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	key := append([]byte(nil), first.Runtime.Identity.PublicKey...)
	if first.Management == nil || len(first.Endpoints) != 2 {
		t.Fatal("persistent composition is incomplete")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := StartPersistent(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key, second.Runtime.Identity.PublicKey) {
		t.Fatal("persistent Hub identity changed across restart")
	}
	_ = second.Close()

	bad := PersistentConfig{StateDirectory: directory, NodeID: 1, Listeners: []ListenerConfig{{Driver: network, Endpoint: "rollback"}, {Driver: network, Endpoint: "rollback"}}}
	if _, err := StartPersistent(context.Background(), bad); err == nil {
		t.Fatal("duplicate listener unexpectedly started")
	}
	listener, err := network.Listen(context.Background(), "rollback")
	if err != nil {
		t.Fatalf("first listener was not rolled back: %v", err)
	}
	_ = listener.Close()
}

func TestInvalidConfigDoesNotStartPartialHub(t *testing.T) {
	identity, _ := auth.GenerateIdentity(1)
	if _, err := Start(context.Background(), Config{Node: node.Config{Identity: identity}}); err == nil {
		t.Fatal("invalid hub configuration was accepted")
	}
}
