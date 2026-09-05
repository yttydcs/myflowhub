package sdk_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/sdk/bindings"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestManagementQueryTopologyAttachedHostPermissionsAndLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	network := memory.NewNetwork()
	defer network.Close()
	root, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: t.TempDir(), NodeID: 1, RefreshInterval: time.Hour,
		Listeners: []hub.ListenerConfig{{Driver: network, Endpoint: "sdk-topology-host"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	directory := t.TempDir()
	state, err := auth.OpenState(directory, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Policy.Close()
	permit, err := root.Runtime.Admission.Issue(2, state.Identity.PublicKey, "sdk-topology", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Policy.Close(); err != nil {
		t.Fatal(err)
	}
	host, err := nodehost.New(ctx, nodehost.Config{
		StateDirectory: directory, NodeID: 2,
		Parent: &nodehost.ParentConfig{
			NodeID: 1, PublicKey: root.Runtime.Identity.PublicKey, Permit: &permit,
			Driver: network, Endpoint: root.Endpoint,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	if err := host.Start(); err != nil {
		t.Fatal(err)
	}
	connection, ok := host.ParentStatus()
	if !ok {
		t.Fatal("attached Host has no parent status")
	}
	facade, err := bindings.NewAttachedClient(host.Client(), bindings.PublicIdentity{NodeID: host.ID(), PublicKey: host.PublicKey()}, connection)
	if err != nil {
		t.Fatal(err)
	}
	defer facade.Close()
	if err := facade.WaitConnected(5_000); err != nil {
		t.Fatal(err)
	}
	if err := root.Management.Refresh(); err != nil {
		t.Fatal(err)
	}
	management, err := host.Client().Management(1)
	if err != nil {
		t.Fatal(err)
	}
	resourceID := protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology}
	grant := func(capability protocol.CapabilityID) {
		t.Helper()
		if err := root.Runtime.Policy.Grant(auth.Request{Subject: 2, Resource: resourceID, Capability: capability}); err != nil {
			t.Fatal(err)
		}
	}
	forbidden := func(err error) {
		t.Helper()
		var failure *sdk.Error
		if !errors.As(err, &failure) || failure.Code != protocol.CodeForbidden || failure.Retryable {
			t.Fatalf("expected non-retryable SDK Forbidden, got %T: %v", err, err)
		}
	}
	_, err = management.QueryTopology(ctx, 1)
	forbidden(err)
	grant(protocol.CapabilityChildren)
	shallow, err := management.QueryTopology(ctx, 1)
	if err != nil || shallow.RootNodeID != "1" || shallow.Depth != 1 || len(shallow.Nodes) != 2 {
		t.Fatalf("unexpected attached Host topology: %+v (%v)", shallow, err)
	}
	if shallow.Nodes[0].NodeID != "1" || shallow.Nodes[0].ParentID != "" || !shallow.Nodes[0].HasChildren ||
		shallow.Nodes[1].NodeID != "2" || shallow.Nodes[1].ParentID != "1" || shallow.Nodes[1].HasChildren {
		t.Fatalf("query did not reflect the connected Host relation: %+v", shallow.Nodes)
	}
	for _, depth := range []int{0, 2} {
		_, err := management.QueryTopology(ctx, depth)
		forbidden(err)
	}
	_, err = management.Topology(ctx)
	forbidden(err)
	current, err := host.Client().Subscribe(ctx, resourceID, time.Minute, 2)
	if current != nil {
		current.Cancel()
	}
	forbidden(err)
	grant(protocol.CapabilitySubtree)
	for _, depth := range []int{0, 2, protocol.MaxItems} {
		value, err := management.QueryTopology(ctx, depth)
		if err != nil || value.Depth != depth || len(value.Nodes) != 2 || value.InstanceID != shallow.InstanceID || value.Revision != shallow.Revision {
			t.Fatalf("unexpected subtree depth %d: %+v (%v)", depth, value, err)
		}
	}
	grant(protocol.CapabilityRead)
	legacy, err := management.Topology(ctx)
	if err != nil || len(legacy.Nodes) != 2 || legacy.Nodes[0].NodeID != "1" {
		t.Fatalf("legacy topology compatibility failed: %+v (%v)", legacy, err)
	}
	// Closing the existing portable facade must leave the Host's attached Go
	// helper and connection usable; the new query adds no owning lifecycle.
	if err := facade.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-host.Node().Done():
		t.Fatal("facade Close stopped the Host")
	default:
	}
	if _, err := management.QueryTopology(ctx, 1); err != nil {
		t.Fatalf("Host query failed after facade Close: %v", err)
	}
}
