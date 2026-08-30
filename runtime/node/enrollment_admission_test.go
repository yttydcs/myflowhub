package node_test

import (
	"context"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/enrollment"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestListenerEnrollsDeviceBeforeOrdinaryJoin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parentIdentity, _ := auth.GenerateIdentity(1)
	device, _ := auth.GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	parentTrust, _ := auth.LoadTrustStore(store)
	if err := parentTrust.Add(parentIdentity.NodeID, parentIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	authority, err := auth.LoadEnrollmentAuthority(parentIdentity, store, auth.EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := authority.IssuePermit(
		"40000000000000000000000000000001",
		auth.DevicePublicKeyFingerprint(device.PublicKey),
		parentIdentity.NodeID,
		false,
		"desktop",
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	enrollmentServer, err := enrollment.NewServer(enrollment.ServerConfig{
		Parent: parentIdentity, Trust: parentTrust, Broker: enrollment.LocalBroker{Authority: authority},
	})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := node.New(ctx, node.Config{
		Identity: parentIdentity, Trust: parentTrust, Policy: auth.AllowAll{}, Enrollment: enrollmentServer,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := parent.Listen(network, "enrollment-parent")
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := network.Dial(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	result, err := enrollment.Enroll(ctx, pipe, device, enrollment.ClientOptions{
		RequestID: "40000000000000000000000000000002", Permit: &permit,
	})
	_ = pipe.Close()
	if err != nil {
		t.Fatal(err)
	}
	if result.Identity == nil {
		t.Fatal("enrollment did not return an identity")
	}
	childTrust := auth.NewTrustStore()
	if err := childTrust.Add(parentIdentity.NodeID, parentIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	child, err := node.New(ctx, node.Config{Identity: *result.Identity, Trust: childTrust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	if err := child.ConnectParent(ctx, network, endpoint, parent.ID()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := parent.Tree().RouteTo(child.ID()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("enrolled child did not enter the ordinary node tree")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
