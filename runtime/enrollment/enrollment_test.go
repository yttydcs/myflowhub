package enrollment

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
)

func TestPermitEnrollmentAnchorsAuthorityAndCreatesParentTrust(t *testing.T) {
	authorityIdentity, _ := auth.GenerateIdentity(1)
	parentIdentity, _ := auth.GenerateIdentity(2)
	device, _ := auth.GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	authority, err := auth.LoadEnrollmentAuthority(authorityIdentity, store, auth.EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := authority.IssuePermit(
		"00000000000000000000000000000001",
		auth.DevicePublicKeyFingerprint(device.PublicKey),
		parentIdentity.NodeID,
		false,
		"headless-leaf",
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	server, err := NewServer(ServerConfig{Parent: parentIdentity, Trust: trust, Broker: LocalBroker{Authority: authority}})
	if err != nil {
		t.Fatal(err)
	}
	result, serverErr, clientErr := exchange(t, server, device, ClientOptions{
		RequestID: "00000000000000000000000000000002",
		Permit:    &permit,
	})
	if clientErr != nil {
		t.Fatal(clientErr)
	}
	if serverErr != nil {
		t.Fatal(serverErr)
	}
	if result.Outcome.Status != "granted" || result.Identity == nil {
		t.Fatalf("unexpected enrollment result: %#v", result)
	}
	trustedKey, trusted := trust.PublicKey(result.Identity.NodeID)
	if !trusted || string(trustedKey) != string(device.PublicKey) {
		t.Fatal("parent did not persist the granted child trust")
	}
	if string(result.AuthorityPublicKey) != string(authority.PublicKey()) {
		t.Fatal("client did not retain the Permit-anchored Authority key")
	}
}

func TestPendingEnrollmentGetsNodeIDOnlyAfterApproval(t *testing.T) {
	authorityIdentity, _ := auth.GenerateIdentity(1)
	parentIdentity, _ := auth.GenerateIdentity(2)
	device, _ := auth.GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	authority, _ := auth.LoadEnrollmentAuthority(authorityIdentity, store, auth.EnrollmentAuthorityConfig{})
	trust := auth.NewTrustStore()
	server, _ := NewServer(ServerConfig{Parent: parentIdentity, Trust: trust, Broker: LocalBroker{Authority: authority}})
	requestID := "10000000000000000000000000000001"
	first, serverErr, clientErr := exchange(t, server, device, ClientOptions{RequestID: requestID, AllowTOFU: true})
	if clientErr != nil || serverErr != nil {
		t.Fatalf("pending exchange errors: client=%v server=%v", clientErr, serverErr)
	}
	if first.Outcome.Status != "pending" || first.Identity != nil || len(authority.Enrollments()) != 0 {
		t.Fatalf("unapproved enrollment created an identity: %#v", first)
	}
	grant, err := authority.Approve("10000000000000000000000000000002", requestID, "approved-leaf")
	if err != nil {
		t.Fatal(err)
	}
	second, serverErr, clientErr := exchange(t, server, device, ClientOptions{
		RequestID:                  requestID,
		ExpectedParentNodeID:       first.ParentNodeID,
		ExpectedParentPublicKey:    first.ParentPublicKey,
		ExpectedAuthorityPublicKey: first.AuthorityPublicKey,
	})
	if clientErr != nil || serverErr != nil {
		t.Fatalf("approved exchange errors: client=%v server=%v", clientErr, serverErr)
	}
	if second.Identity == nil || second.Outcome.Grant == nil || second.Outcome.Grant.NodeID != grant.NodeID || second.Identity.NodeID == 0 {
		t.Fatalf("approved enrollment did not return the durable Grant: %#v", second)
	}
}

func TestEnrollmentRequiresExplicitTOFUWithoutAnAnchor(t *testing.T) {
	authorityIdentity, _ := auth.GenerateIdentity(1)
	parentIdentity, _ := auth.GenerateIdentity(2)
	device, _ := auth.GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	authority, _ := auth.LoadEnrollmentAuthority(authorityIdentity, store, auth.EnrollmentAuthorityConfig{})
	server, _ := NewServer(ServerConfig{Parent: parentIdentity, Trust: auth.NewTrustStore(), Broker: LocalBroker{Authority: authority}})
	_, _, clientErr := exchange(t, server, device, ClientOptions{RequestID: "20000000000000000000000000000001"})
	if !errors.Is(clientErr, ErrTrustConfirmationRequired) {
		t.Fatalf("expected explicit TOFU confirmation error, got %v", clientErr)
	}
	if len(authority.Requests()) != 0 {
		t.Fatal("client created a Pending request before confirming trust")
	}
}

func TestAuthorityKeyAloneDoesNotBypassParentTOFU(t *testing.T) {
	authorityIdentity, _ := auth.GenerateIdentity(1)
	parentIdentity, _ := auth.GenerateIdentity(2)
	device, _ := auth.GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	authority, _ := auth.LoadEnrollmentAuthority(authorityIdentity, store, auth.EnrollmentAuthorityConfig{})
	server, _ := NewServer(ServerConfig{Parent: parentIdentity, Trust: auth.NewTrustStore(), Broker: LocalBroker{Authority: authority}})
	_, _, clientErr := exchange(t, server, device, ClientOptions{
		RequestID:                  "20000000000000000000000000000002",
		ExpectedAuthorityPublicKey: authority.PublicKey(),
	})
	if !errors.Is(clientErr, ErrTrustConfirmationRequired) {
		t.Fatalf("expected parent TOFU confirmation with only an Authority pin, got %v", clientErr)
	}
	if len(authority.Requests()) != 0 {
		t.Fatal("client created a Pending request before confirming the unpinned parent")
	}
}

func TestPermitRejectsDifferentDeviceBeforeEnrollment(t *testing.T) {
	authorityIdentity, _ := auth.GenerateIdentity(1)
	first, _ := auth.GenerateDeviceIdentity()
	second, _ := auth.GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	authority, _ := auth.LoadEnrollmentAuthority(authorityIdentity, store, auth.EnrollmentAuthorityConfig{})
	permit, _ := authority.IssuePermit(
		"30000000000000000000000000000001",
		auth.DevicePublicKeyFingerprint(first.PublicKey),
		2,
		false,
		"leaf",
		time.Hour,
	)
	clientPipe, serverPipe := net.Pipe()
	defer clientPipe.Close()
	defer serverPipe.Close()
	_, err := Enroll(context.Background(), clientPipe, second, ClientOptions{
		RequestID: "30000000000000000000000000000002",
		Permit:    &permit,
	})
	if err == nil {
		t.Fatal("Permit bound to another device was accepted")
	}
}

func TestEnrollmentCancellationClosesBlockedHandshake(t *testing.T) {
	device, _ := auth.GenerateDeviceIdentity()
	clientPipe, serverPipe := net.Pipe()
	defer serverPipe.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := Enroll(ctx, clientPipe, device, ClientOptions{RequestID: "40000000000000000000000000000001", AllowTOFU: true})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked Enrollment returned %v, want context deadline", err)
	}
}

func TestEnrollmentRejectsResultForAnotherRequest(t *testing.T) {
	parent, _ := auth.GenerateIdentity(2)
	authority, _ := auth.GenerateIdentity(1)
	device, _ := auth.GenerateDeviceIdentity()
	clientPipe, serverPipe := net.Pipe()
	serverErrors := make(chan error, 1)
	go func() {
		defer serverPipe.Close()
		codec := protocol.EnrollmentCodec{}
		initFrame, err := codec.Decode(serverPipe)
		if err != nil {
			serverErrors <- err
			return
		}
		var init protocol.EnrollmentClientInitV1
		if err := protocol.DecodeJSONPayload(initFrame.Payload, protocol.EnrollmentMaxPayload, &init); err != nil {
			serverErrors <- err
			return
		}
		unsigned := unsignedChallenge{
			Version: 1, RequestID: init.RequestID, ParentNodeID: "2",
			ParentPublicKey: base64.RawStdEncoding.EncodeToString(parent.PublicKey), AuthorityNodeID: "1",
			AuthorityPublicKey: base64.RawStdEncoding.EncodeToString(authority.PublicKey), Nonce: base64.RawStdEncoding.EncodeToString(make([]byte, 32)),
			ExpiresAtUnixMS: time.Now().Add(time.Minute).UnixMilli(),
		}
		challenge := protocol.EnrollmentChallengeV1{
			Version: unsigned.Version, RequestID: unsigned.RequestID, ParentNodeID: unsigned.ParentNodeID,
			ParentPublicKey: unsigned.ParentPublicKey, AuthorityNodeID: unsigned.AuthorityNodeID, AuthorityPublicKey: unsigned.AuthorityPublicKey,
			Nonce: unsigned.Nonce, ExpiresAtUnixMS: unsigned.ExpiresAtUnixMS, Signature: base64.RawStdEncoding.EncodeToString(signChallenge(parent.PrivateKey, unsigned)),
		}
		challengePayload, err := protocol.EncodeJSONPayload(challenge, protocol.EnrollmentMaxPayload)
		if err == nil {
			err = codec.Encode(serverPipe, protocol.EnrollmentFrame{Version: 1, Type: protocol.EnrollmentMessageChallenge, Payload: challengePayload})
		}
		if err == nil {
			_, err = codec.Decode(serverPipe)
		}
		resultPayload, encodeErr := protocol.EncodeJSONPayload(protocol.EnrollmentResultV1{
			Version: 1, RequestID: "40000000000000000000000000000003", Status: "pending", RetryAfterMS: 1,
		}, protocol.EnrollmentMaxPayload)
		if err == nil && encodeErr == nil {
			err = codec.Encode(serverPipe, protocol.EnrollmentFrame{Version: 1, Type: protocol.EnrollmentMessageResult, Payload: resultPayload})
		}
		if err == nil {
			err = encodeErr
		}
		serverErrors <- err
	}()
	_, err := Enroll(context.Background(), clientPipe, device, ClientOptions{
		RequestID: "40000000000000000000000000000002", AllowTOFU: true,
	})
	_ = clientPipe.Close()
	if err == nil || err.Error() != "enrollment result request ID does not match the session" {
		t.Fatalf("mismatched result request ID error = %v", err)
	}
	if serverErr := <-serverErrors; serverErr != nil {
		t.Fatal(serverErr)
	}
}

func exchange(t *testing.T, server *Server, device auth.DeviceIdentity, options ClientOptions) (ClientResult, error, error) {
	t.Helper()
	clientPipe, serverPipe := net.Pipe()
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Handle(context.Background(), serverPipe)
		_ = serverPipe.Close()
	}()
	result, clientErr := Enroll(context.Background(), clientPipe, device, options)
	_ = clientPipe.Close()
	serverErr := <-serverErrors
	if clientErr != nil && errors.Is(clientErr, ErrTrustConfirmationRequired) {
		// The server observes EOF because the client intentionally stops before proof.
		serverErr = nil
	}
	return result, serverErr, clientErr
}
