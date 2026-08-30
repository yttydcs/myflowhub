package auth

import (
	"bytes"
	"testing"

	"github.com/yttydcs/myflowhub/internal/keystore"
)

func TestEnrollmentClientStatePersistsOneDeviceAndGrantAtomically(t *testing.T) {
	store, _ := keystore.New(t.TempDir())
	client, err := LoadOrCreateEnrollmentClientState(store)
	if err != nil {
		t.Fatal(err)
	}
	initial := client.Snapshot()
	if initial.Status != "device" || initial.RequestID == "" || len(initial.DevicePublicKey) == 0 {
		t.Fatalf("unexpected initial client state: %#v", initial)
	}
	authorityIdentity, _ := GenerateIdentity(1)
	parentIdentity, _ := GenerateIdentity(2)
	authority, _ := LoadEnrollmentAuthority(authorityIdentity, storeForAuthority(t), EnrollmentAuthorityConfig{})
	device := client.DeviceIdentity()
	_, err = authority.Submit(EnrollmentSubmission{
		RequestID: initial.RequestID, DevicePublicKey: device.PublicKey,
		ParentNodeID: parentIdentity.NodeID, ParentPublicKey: parentIdentity.PublicKey,
		TranscriptDigest: [32]byte{1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.RecordObservation(parentIdentity.NodeID, parentIdentity.PublicKey, authorityIdentity.NodeID, authorityIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	grant, err := authority.Approve("80000000000000000000000000000001", initial.RequestID, "desktop")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.RecordGrant(grant); err != nil {
		t.Fatal(err)
	}
	restarted, err := LoadOrCreateEnrollmentClientState(store)
	if err != nil {
		t.Fatal(err)
	}
	reopened := restarted.Snapshot()
	if reopened.Status != "enrolled" || reopened.Grant == nil || reopened.Grant.NodeID != grant.NodeID || reopened.RequestID != initial.RequestID || !bytes.Equal(reopened.DevicePublicKey, initial.DevicePublicKey) {
		t.Fatalf("Enrollment client state was not durable: %#v", reopened)
	}
	identity, enrolled, err := restarted.Identity()
	if err != nil || !enrolled || identity.NodeID == 0 || !bytes.Equal(identity.PublicKey, initial.DevicePublicKey) {
		t.Fatalf("durable Grant did not create the enrolled identity: %#v, %v, %v", identity, enrolled, err)
	}
}

func TestEnrollmentClientStateRejectsTrustReplacement(t *testing.T) {
	store, _ := keystore.New(t.TempDir())
	client, _ := LoadOrCreateEnrollmentClientState(store)
	firstParent, _ := GenerateIdentity(2)
	secondParent, _ := GenerateIdentity(3)
	authority, _ := GenerateIdentity(1)
	if err := client.RecordObservation(firstParent.NodeID, firstParent.PublicKey, authority.NodeID, authority.PublicKey); err != nil {
		t.Fatal(err)
	}
	if err := client.RecordObservation(secondParent.NodeID, secondParent.PublicKey, authority.NodeID, authority.PublicKey); err == nil {
		t.Fatal("persisted TOFU observation was silently replaced")
	}
}

func storeForAuthority(t *testing.T) *keystore.Store {
	t.Helper()
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}
