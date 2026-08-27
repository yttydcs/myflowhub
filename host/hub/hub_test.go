package hub

import (
	"context"
	"testing"

	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/memory"
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

func TestInvalidConfigDoesNotStartPartialHub(t *testing.T) {
	identity, _ := auth.GenerateIdentity(1)
	if _, err := Start(context.Background(), Config{Node: node.Config{Identity: identity}}); err == nil {
		t.Fatal("invalid hub configuration was accepted")
	}
}
