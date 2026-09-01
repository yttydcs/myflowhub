package auth

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

type recordingEnrollmentCredentialStore struct {
	mu         sync.Mutex
	credential EnrollmentCredential
	found      bool
	loadErr    error
	saves      int
}

func (s *recordingEnrollmentCredentialStore) LoadEnrollmentCredential() (EnrollmentCredential, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneEnrollmentCredential(s.credential), s.found, s.loadErr
}

func (s *recordingEnrollmentCredentialStore) SaveEnrollmentCredential(credential EnrollmentCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credential = cloneEnrollmentCredential(credential)
	s.found = true
	s.saves++
	return nil
}

func TestInspectEnrollmentClientStateIsReadOnly(t *testing.T) {
	store := &recordingEnrollmentCredentialStore{}
	if snapshot, found, err := InspectEnrollmentClientState(store); err != nil || found || snapshot.Status != "" {
		t.Fatalf("missing inspection = %#v, %v, %v", snapshot, found, err)
	}
	if store.saves != 0 {
		t.Fatalf("inspection persisted %d credentials", store.saves)
	}

	device, err := GenerateDeviceIdentity()
	if err != nil {
		t.Fatal(err)
	}
	requestID, err := protocol.NewMessageID()
	if err != nil {
		t.Fatal(err)
	}
	store.credential = EnrollmentCredential{
		Version: enrollmentClientStateVersion, Status: "device", RequestID: requestID.String(),
		DevicePublicKey: device.PublicKey, DevicePrivateKey: device.PrivateKey,
	}
	store.found = true
	snapshot, found, err := InspectEnrollmentClientState(store)
	if err != nil || !found || snapshot.Status != "device" || snapshot.RequestID != requestID.String() {
		t.Fatalf("existing inspection = %#v, %v, %v", snapshot, found, err)
	}
	snapshot.DevicePublicKey[0] ^= 0xff
	if bytes.Equal(snapshot.DevicePublicKey, store.credential.DevicePublicKey) {
		t.Fatal("inspection returned aliased public key data")
	}
	if store.saves != 0 {
		t.Fatalf("inspection persisted %d credentials", store.saves)
	}
}

func TestInspectEnrollmentClientStateRejectsInvalidOrUnreadableState(t *testing.T) {
	store := &recordingEnrollmentCredentialStore{found: true, credential: EnrollmentCredential{Version: 99}}
	if _, found, err := InspectEnrollmentClientState(store); err == nil || found {
		t.Fatalf("invalid inspection unexpectedly succeeded: found=%v err=%v", found, err)
	}
	store.loadErr = errors.New("protected store unavailable")
	if _, found, err := InspectEnrollmentClientState(store); err == nil || found {
		t.Fatalf("failed inspection unexpectedly succeeded: found=%v err=%v", found, err)
	}
	if store.saves != 0 {
		t.Fatalf("failed inspections persisted %d credentials", store.saves)
	}
}

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

func TestEnrollmentCredentialSourceFailsClosedAndDoesNotWrite(t *testing.T) {
	missing := &recordingEnrollmentCredentialStore{}
	source, err := NewEnrollmentCredentialSource(missing)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.LoadNodeCredential(); !errors.Is(err, ErrEnrollmentCredentialMissing) {
		t.Fatalf("missing credential returned %v", err)
	}
	if missing.saves != 0 {
		t.Fatalf("missing credential source wrote %d times", missing.saves)
	}

	device, err := LoadOrCreateEnrollmentClientStateWithStore(missing)
	if err != nil {
		t.Fatal(err)
	}
	writes := missing.saves
	if _, err := source.LoadNodeCredential(); !errors.Is(err, ErrEnrollmentNotGranted) {
		t.Fatalf("device credential returned %v", err)
	}
	if missing.saves != writes {
		t.Fatal("credential source mutated device state")
	}

	parent, _ := GenerateIdentity(2)
	authorityIdentity, _ := GenerateIdentity(1)
	if err := device.RecordObservation(parent.NodeID, parent.PublicKey, authorityIdentity.NodeID, authorityIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	writes = missing.saves
	if _, err := source.LoadNodeCredential(); !errors.Is(err, ErrEnrollmentNotGranted) {
		t.Fatalf("pending credential returned %v", err)
	}
	if missing.saves != writes {
		t.Fatal("credential source mutated pending state")
	}
}

func TestEnrollmentCredentialSourceReturnsClonedGrantedIdentity(t *testing.T) {
	store := &recordingEnrollmentCredentialStore{}
	client, err := LoadOrCreateEnrollmentClientStateWithStore(store)
	if err != nil {
		t.Fatal(err)
	}
	parent, _ := GenerateIdentity(2)
	authorityIdentity, _ := GenerateIdentity(1)
	authority, err := LoadEnrollmentAuthority(authorityIdentity, storeForAuthority(t), EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := client.Snapshot()
	if _, err := authority.Submit(EnrollmentSubmission{
		RequestID: snapshot.RequestID, DevicePublicKey: snapshot.DevicePublicKey,
		ParentNodeID: parent.NodeID, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{9},
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.RecordObservation(parent.NodeID, parent.PublicKey, authorityIdentity.NodeID, authorityIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	grant, err := authority.Approve("90000000000000000000000000000001", snapshot.RequestID, "desktop")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.RecordGrant(grant); err != nil {
		t.Fatal(err)
	}
	source, _ := NewEnrollmentCredentialSource(store)
	writes := store.saves
	credential, err := source.LoadNodeCredential()
	if err != nil {
		t.Fatal(err)
	}
	if credential.Identity.NodeID == 0 || credential.ParentNodeID != parent.NodeID || credential.AuthorityNodeID != authorityIdentity.NodeID || credential.EnrollmentID != grant.EnrollmentID {
		t.Fatalf("unexpected Node credential: %#v", credential)
	}
	credential.Identity.PrivateKey[0] ^= 0xff
	credential.ParentPublicKey[0] ^= 0xff
	again, err := source.LoadNodeCredential()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(credential.Identity.PrivateKey, again.Identity.PrivateKey) || bytes.Equal(credential.ParentPublicKey, again.ParentPublicKey) {
		t.Fatal("Node credential source returned aliased key material")
	}
	if store.saves != writes {
		t.Fatal("Node credential source persisted during read")
	}
	original, _, _ := store.LoadEnrollmentCredential()
	otherParent, _ := GenerateIdentity(3)
	otherAuthority, _ := GenerateIdentity(4)
	for _, test := range []struct {
		name   string
		mutate func(*EnrollmentCredential)
	}{
		{name: "Grant signature", mutate: func(value *EnrollmentCredential) { value.Grant.Signature = "invalid" }},
		{name: "parent Node ID", mutate: func(value *EnrollmentCredential) { value.ParentNodeID = otherParent.NodeID }},
		{name: "parent public key", mutate: func(value *EnrollmentCredential) { value.ParentPublicKey = otherParent.PublicKey }},
		{name: "Authority Node ID", mutate: func(value *EnrollmentCredential) { value.AuthorityNodeID = otherAuthority.NodeID }},
		{name: "Authority public key", mutate: func(value *EnrollmentCredential) { value.AuthorityPublicKey = otherAuthority.PublicKey }},
	} {
		t.Run(test.name, func(t *testing.T) {
			corrupt := cloneEnrollmentCredential(original)
			test.mutate(&corrupt)
			corruptStore := &recordingEnrollmentCredentialStore{credential: corrupt, found: true}
			corruptSource, _ := NewEnrollmentCredentialSource(corruptStore)
			if _, err := corruptSource.LoadNodeCredential(); err == nil {
				t.Fatal("corrupt enrolled credential was accepted")
			}
			if corruptStore.saves != 0 {
				t.Fatal("corrupt credential read wrote state")
			}
		})
	}

	directory := t.TempDir()
	state, err := OpenStateWithIdentity(directory, again.Identity)
	if err != nil {
		t.Fatal(err)
	}
	if state.Identity.NodeID != again.Identity.NodeID {
		t.Fatal("resolved state changed the Node identity")
	}
	if _, err := os.Stat(filepath.Join(directory, "state", "identity.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resolved identity path created identity.json: %v", err)
	}

	var wg sync.WaitGroup
	errorsSeen := make(chan error, 16)
	for index := 0; index < 16; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := source.LoadNodeCredential()
			errorsSeen <- err
		}()
	}
	wg.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal(err)
		}
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
