package management_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/feature/management"
	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/enrollment"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestRemoteParentUsesCentralAuthorityAndReceivesRevocation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rootState, err := hostconfig.Open(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	parentState, err := hostconfig.Open(t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := rootState.Trust.Add(parentState.Identity.NodeID, parentState.Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	if err := parentState.Trust.Add(rootState.Identity.NodeID, rootState.Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	authority, err := auth.LoadEnrollmentAuthority(rootState.Identity, rootState.Store, auth.EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	root, err := node.New(ctx, node.Config{Identity: rootState.Identity, Trust: rootState.Trust, Policy: auth.AllowAll{}, Admission: rootState.Admission})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := management.Register(management.Config{
		Node: root, Admission: rootState.Admission, EnrollmentAuthority: authority,
		Trust: rootState.Trust, Policy: rootState.Policy, Settings: rootState.Settings, RevokeNode: rootState.RevokeNode,
	}); err != nil {
		t.Fatal(err)
	}
	remoteBroker := &enrollment.RemoteBroker{
		AuthorityNodeID: root.ID(), AuthorityPublicKey: authority.PublicKey(),
	}
	parentEnrollment, err := enrollment.NewServer(enrollment.ServerConfig{
		Parent: parentState.Identity, Trust: parentState.Trust, Broker: remoteBroker,
	})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := node.New(ctx, node.Config{
		Identity: parentState.Identity, Trust: parentState.Trust, Policy: auth.AllowAll{},
		Admission: parentState.Admission, Enrollment: parentEnrollment,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	remoteBroker.Node = parent
	if _, err := management.Register(management.Config{
		Node: parent, Admission: parentState.Admission, AuthorityNodeID: root.ID(),
		Trust: parentState.Trust, Policy: parentState.Policy, Settings: parentState.Settings, RevokeNode: parentState.RevokeNode,
	}); err != nil {
		t.Fatal(err)
	}
	network := memory.NewNetwork()
	defer network.Close()
	rootEndpoint, err := root.Listen(network, "authority-root")
	if err != nil {
		t.Fatal(err)
	}
	if err := parent.ConnectParent(ctx, network, rootEndpoint, root.ID()); err != nil {
		t.Fatal(err)
	}
	waitForManagementRoute(t, root, parent.ID())
	device, _ := auth.GenerateDeviceIdentity()
	wrongParent, _ := auth.GenerateIdentity(parent.ID())
	forgedSubmission := protocol.AdmissionSubmitV1{
		Version: 1, RequestID: "60000000000000000000000000000000",
		DevicePublicKey: base64.RawStdEncoding.EncodeToString(device.PublicKey), ParentNodeID: "2",
		ParentPublicKey: base64.RawStdEncoding.EncodeToString(wrongParent.PublicKey), TranscriptDigest: strings.Repeat("01", 32),
	}
	if _, err := parent.Invoke(ctx, protocol.ResourceID{Owner: root.ID(), Name: protocol.BuiltinAdmissionSubmitEnrollment}, encode(t, &forgedSubmission)); err == nil {
		t.Fatal("authenticated parent submitted a different parent public key")
	}
	if len(authority.Requests()) != 0 {
		t.Fatal("forged parent key created an Enrollment request")
	}
	parentEndpoint, err := parent.Listen(network, "remote-parent")
	if err != nil {
		t.Fatal(err)
	}
	requestID := "60000000000000000000000000000001"
	firstPipe, err := network.Dial(ctx, parentEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	first, err := enrollment.Enroll(ctx, firstPipe, device, enrollment.ClientOptions{RequestID: requestID, AllowTOFU: true})
	_ = firstPipe.Close()
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome.Status != "pending" || first.Identity != nil || len(authority.Requests()) != 1 {
		t.Fatalf("remote parent did not create one central Pending request: %#v", first)
	}
	grant, err := authority.Approve("60000000000000000000000000000002", requestID, "remote-leaf")
	if err != nil {
		t.Fatal(err)
	}
	secondPipe, err := network.Dial(ctx, parentEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	second, err := enrollment.Enroll(ctx, secondPipe, device, enrollment.ClientOptions{
		RequestID: requestID, ExpectedParentNodeID: first.ParentNodeID,
		ExpectedParentPublicKey: first.ParentPublicKey, ExpectedAuthorityPublicKey: first.AuthorityPublicKey,
	})
	_ = secondPipe.Close()
	if err != nil {
		t.Fatal(err)
	}
	if second.Identity == nil || second.Outcome.Grant == nil || second.Outcome.Grant.NodeID != grant.NodeID {
		t.Fatalf("approved remote Enrollment did not return the central Grant: %#v", second)
	}
	if _, trusted := parentState.Trust.PublicKey(second.Identity.NodeID); !trusted {
		t.Fatal("remote parent did not cache the central Grant")
	}
	revoke := protocol.AdmissionRevokeEnrollmentV1{
		Version: 1, RequestID: "60000000000000000000000000000003",
		EnrollmentID: grant.EnrollmentID, Reason: "compromised",
	}
	invoke(t, root, protocol.BuiltinAdmissionRevokeEnrollment, encode(t, &revoke))
	if _, trusted := parentState.Trust.PublicKey(second.Identity.NodeID); trusted {
		t.Fatal("Authority revocation was not routed back to the direct parent")
	}
}

func waitForManagementRoute(t *testing.T, runtime *node.Node, target protocol.NodeID) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := runtime.Tree().RouteTo(target); err == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("route to %d was not installed", target)
}
