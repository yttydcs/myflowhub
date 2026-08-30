package node

import (
	"context"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestUntrustedChildCanJoinOnlyWithParentIssuedPermit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parentIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	store, _ := keystore.New(t.TempDir())
	parentTrust, _ := auth.LoadTrustStore(store)
	if err := parentTrust.Add(parentIdentity.NodeID, parentIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	admission, _ := auth.LoadAdmission(parentIdentity, store, auth.AdmissionConfig{})
	permit, err := admission.Issue(childIdentity.NodeID, childIdentity.PublicKey, "device", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	childTrust := auth.NewTrustStore()
	if err := childTrust.Add(parentIdentity.NodeID, parentIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	parent, err := New(ctx, Config{Identity: parentIdentity, Trust: parentTrust, Policy: auth.AllowAll{}, Admission: admission})
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	child, err := New(ctx, Config{Identity: childIdentity, Trust: childTrust, Policy: auth.AllowAll{}, JoinPermit: &permit})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := parent.Listen(network, "permit-parent")
	if err != nil {
		t.Fatal(err)
	}
	if err := child.ConnectParent(ctx, network, endpoint, parent.ID()); err != nil {
		t.Fatal(err)
	}
	waitRoute(t, parent.Tree(), child.ID())
	if _, trusted := parentTrust.PublicKey(child.ID()); !trusted {
		t.Fatal("admitted child was not persisted to trust store")
	}
}

func TestListenerClosesStalledInitialHandshakeAtJoinTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	parent, err := New(ctx, Config{
		Identity: identity, Trust: trust, Policy: auth.AllowAll{}, JoinTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := parent.Listen(network, "stalled-handshake")
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := network.Dial(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer pipe.Close()
	readResult := make(chan error, 1)
	go func() {
		var value [1]byte
		_, readErr := pipe.Read(value[:])
		readResult <- readErr
	}()
	select {
	case readErr := <-readResult:
		if readErr == nil {
			t.Fatal("stalled pre-authentication pipe closed without a read error")
		}
	case <-time.After(time.Second):
		t.Fatal("stalled pre-authentication pipe was not closed at JoinTimeout")
	}
}
