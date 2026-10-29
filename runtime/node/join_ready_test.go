package node

import (
	"context"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestParentCanReadChildImmediatelyAfterJoinAcknowledgement(t *testing.T) {
	for round := range 30 {
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			parentIdentity, err := auth.GenerateIdentity(1)
			if err != nil {
				t.Fatal(err)
			}
			childIdentity, err := auth.GenerateIdentity(2)
			if err != nil {
				t.Fatal(err)
			}
			trust := auth.NewTrustStore()
			for _, identity := range []auth.Identity{parentIdentity, childIdentity} {
				if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
					t.Fatal(err)
				}
			}
			parent, err := New(ctx, Config{Identity: parentIdentity, Trust: trust, Policy: auth.AllowAll{}})
			if err != nil {
				t.Fatal(err)
			}
			defer parent.Close()
			child, err := New(ctx, Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}})
			if err != nil {
				t.Fatal(err)
			}
			defer child.Close()
			id := protocol.ResourceID{Owner: 2, Name: "test/join-ready"}
			variable, err := resource.NewVariable(resource.VariableDescriptor(id, "test.read", 512),
				protocol.DataSchemaDefinition{Type: "boolean"}, resource.VariableState{Status: "ready", Value: []byte(`true`)}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := child.Registry().Register(variable); err != nil {
				t.Fatal(err)
			}
			network := memory.NewNetwork()
			defer network.Close()
			endpoint, err := parent.Listen(network, "join-ready")
			if err != nil {
				t.Fatal(err)
			}
			if err := child.ConnectParent(ctx, network, endpoint, parent.ID()); err != nil {
				t.Fatal(err)
			}
			if _, err := parent.Operate(ctx, id, protocol.CapabilityRead, "", nil); err != nil {
				t.Fatalf("round %d: acknowledged child is not routable: %v", round+1, err)
			}
		}()
	}
}
