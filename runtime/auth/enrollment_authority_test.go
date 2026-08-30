package auth

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

func TestDeviceIdentityHasNoNodeIDUntilGrant(t *testing.T) {
	device, err := GenerateDeviceIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := device.Enroll(0); err == nil {
		t.Fatal("zero Node ID must not create an enrolled identity")
	}
	enrolled, err := device.Enroll(42)
	if err != nil {
		t.Fatal(err)
	}
	if enrolled.NodeID != 42 || !bytes.Equal(enrolled.PublicKey, device.PublicKey) {
		t.Fatalf("unexpected enrolled identity: %#v", enrolled)
	}
}

func TestEnrollmentAuthorityPermitGrantIsIdempotentAndDurable(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	parentID := protocol.NodeID(7)
	parent, _ := GenerateIdentity(parentID)
	device, _ := GenerateDeviceIdentity()
	clock := time.Unix(1000, 0)
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := authority.IssuePermit(
		"00000000000000000000000000000001",
		DevicePublicKeyFingerprint(device.PublicKey),
		parentID,
		false,
		"leaf",
		time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	digest := [32]byte{1}
	submission := EnrollmentSubmission{
		RequestID: "00000000000000000000000000000002", DevicePublicKey: device.PublicKey,
		ParentNodeID: parentID, ParentPublicKey: parent.PublicKey, Permit: &permit, TranscriptDigest: digest,
	}
	outcome, err := authority.Submit(submission)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != "granted" || outcome.Grant == nil || outcome.Grant.NodeID == "" {
		t.Fatalf("unexpected outcome: %#v", outcome)
	}
	if err := VerifyEnrollmentGrant(authority.PublicKey(), *outcome.Grant); err != nil {
		t.Fatal(err)
	}
	restarted, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := restarted.Submit(submission)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Grant == nil || replayed.Grant.EnrollmentID != outcome.Grant.EnrollmentID || replayed.Grant.NodeID != outcome.Grant.NodeID {
		t.Fatalf("idempotent replay allocated a different grant: first %#v, replay %#v", outcome.Grant, replayed.Grant)
	}
	otherSubmission := submission
	otherSubmission.RequestID = "00000000000000000000000000000003"
	otherSubmission.TranscriptDigest[0] = 2
	deduplicated, err := restarted.Submit(otherSubmission)
	if err != nil || deduplicated.Grant == nil || deduplicated.Grant.NodeID != outcome.Grant.NodeID {
		t.Fatalf("same device and parent did not deduplicate to the durable Grant: %#v, %v", deduplicated, err)
	}
	otherDevice, _ := GenerateDeviceIdentity()
	otherSubmission.DevicePublicKey = otherDevice.PublicKey
	otherSubmission.RequestID = "00000000000000000000000000000004"
	if _, err := restarted.Submit(otherSubmission); !errors.Is(err, ErrPermitConsumed) && !errors.Is(err, ErrPermitInvalid) {
		t.Fatalf("consumed Permit was accepted for another device: %v", err)
	}
}

func TestEnrollmentAuthorityPendingApprovalIsConcurrentAndIdempotent(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	device, _ := GenerateDeviceIdentity()
	parent, _ := GenerateIdentity(9)
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	submission := EnrollmentSubmission{
		RequestID: "10000000000000000000000000000001", DevicePublicKey: device.PublicKey,
		ParentNodeID: 9, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{3},
	}
	const workers = 12
	results := make(chan EnrollmentOutcome, workers)
	errorsSeen := make(chan error, workers)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			outcome, submitErr := authority.Submit(submission)
			if submitErr != nil {
				errorsSeen <- submitErr
				return
			}
			results <- outcome
		}()
	}
	wait.Wait()
	close(results)
	close(errorsSeen)
	for submitErr := range errorsSeen {
		t.Fatal(submitErr)
	}
	for outcome := range results {
		if outcome.Status != "pending" || outcome.Grant != nil {
			t.Fatalf("unexpected pending result: %#v", outcome)
		}
	}
	if got := len(authority.Requests()); got != 1 {
		t.Fatalf("pending request count = %d, want 1", got)
	}
	grant, err := authority.Approve("10000000000000000000000000000002", submission.RequestID, "operator-approved")
	if err != nil {
		t.Fatal(err)
	}
	again, err := authority.Approve("10000000000000000000000000000003", submission.RequestID, "operator-approved")
	if err != nil {
		t.Fatal(err)
	}
	if grant.NodeID != again.NodeID || grant.EnrollmentID != again.EnrollmentID || len(authority.Enrollments()) != 1 {
		t.Fatal("repeated approval was not idempotent")
	}
}

func TestEnrollmentAuthorityPendingLimitDoesNotAllocateNodeIDs(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	parent, _ := GenerateIdentity(2)
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{MaxPending: 1})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := GenerateDeviceIdentity()
	second, _ := GenerateDeviceIdentity()
	if outcome, err := authority.Submit(EnrollmentSubmission{
		RequestID: "20000000000000000000000000000001", DevicePublicKey: first.PublicKey,
		ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{1},
	}); err != nil || outcome.Status != "pending" {
		t.Fatalf("first pending submission = %#v, %v", outcome, err)
	}
	if _, err := authority.Submit(EnrollmentSubmission{
		RequestID: "20000000000000000000000000000002", DevicePublicKey: second.PublicKey,
		ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{2},
	}); err == nil {
		t.Fatal("expected pending capacity error")
	}
	if got := len(authority.Enrollments()); got != 0 {
		t.Fatalf("unapproved requests allocated %d enrollments", got)
	}
}

func TestEnrollmentAuthorityBoundsRetainedRequestRecords(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	parent, _ := GenerateIdentity(2)
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{MaxRequests: 1, MaxPending: 1})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := GenerateDeviceIdentity()
	firstID := "20500000000000000000000000000001"
	if _, err := authority.Submit(EnrollmentSubmission{
		RequestID: firstID, DevicePublicKey: first.PublicKey,
		ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := authority.Reject("20500000000000000000000000000002", firstID, "denied"); err != nil {
		t.Fatal(err)
	}
	second, _ := GenerateDeviceIdentity()
	if _, err := authority.Submit(EnrollmentSubmission{
		RequestID: "20500000000000000000000000000003", DevicePublicKey: second.PublicKey,
		ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{2},
	}); err == nil {
		t.Fatal("retained Enrollment request records exceeded their configured bound")
	}
	if len(authority.Enrollments()) != 0 {
		t.Fatal("bounded request rejection allocated a Node ID")
	}
}

func TestEnrollmentAuthorityDeduplicatesPendingByDeviceAndParent(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	parent, _ := GenerateIdentity(2)
	device, _ := GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	firstID := "21000000000000000000000000000001"
	secondID := "21000000000000000000000000000002"
	first := EnrollmentSubmission{RequestID: firstID, DevicePublicKey: device.PublicKey, ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{1}}
	second := first
	second.RequestID = secondID
	second.TranscriptDigest = [32]byte{2}
	if outcome, err := authority.Submit(first); err != nil || outcome.Status != "pending" {
		t.Fatalf("first submission = %#v, %v", outcome, err)
	}
	if outcome, err := authority.Submit(second); err != nil || outcome.Status != "pending" || outcome.RequestID != secondID {
		t.Fatalf("deduplicated submission = %#v, %v", outcome, err)
	}
	if got := len(authority.Requests()); got != 1 {
		t.Fatalf("same device created %d Pending records", got)
	}
	grant, err := authority.Approve("21000000000000000000000000000003", firstID, "leaf")
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := authority.Submit(second)
	if err != nil || outcome.Grant == nil || outcome.Grant.NodeID != grant.NodeID || outcome.RequestID != secondID {
		t.Fatalf("deduplicated retry did not receive the approved Grant: %#v, %v", outcome, err)
	}
}

func TestEnrollmentAuthorityPermitUpgradesExistingPendingRequest(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	parent, _ := GenerateIdentity(2)
	device, _ := GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	requestID := "21500000000000000000000000000001"
	submission := EnrollmentSubmission{RequestID: requestID, DevicePublicKey: device.PublicKey, ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{1}}
	if outcome, err := authority.Submit(submission); err != nil || outcome.Status != "pending" {
		t.Fatalf("initial Pending = %#v, %v", outcome, err)
	}
	permit, err := authority.IssuePermit("21500000000000000000000000000002", DevicePublicKeyFingerprint(device.PublicKey), 2, false, "expedited", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	submission.Permit = &permit
	submission.TranscriptDigest = [32]byte{2}
	outcome, err := authority.Submit(submission)
	if err != nil || outcome.Status != "granted" || outcome.Grant == nil || outcome.Grant.AdmissionProfile != "expedited" {
		t.Fatalf("Permit did not upgrade Pending request to Grant: %#v, %v", outcome, err)
	}
	if len(authority.Requests()) != 1 || len(authority.Enrollments()) != 1 {
		t.Fatal("Pending upgrade created duplicate request or enrollment records")
	}
}

func TestEnrollmentAuthorityExpiresPendingWithoutAllocatingNodeID(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	parent, _ := GenerateIdentity(2)
	device, _ := GenerateDeviceIdentity()
	clock := time.Unix(1_000, 0)
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{
		Now: func() time.Time { return clock }, PendingTTL: time.Minute, MaxPending: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	requestID := "22000000000000000000000000000001"
	if _, err := authority.Submit(EnrollmentSubmission{RequestID: requestID, DevicePublicKey: device.PublicKey, ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{1}}); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(2 * time.Minute)
	requests := authority.Requests()
	if len(requests) != 1 || requests[0].Status != "expired" || requests[0].ExpiresAtUnixMS <= requests[0].CreatedAtUnixMS {
		t.Fatalf("expired request was not visible: %#v", requests)
	}
	if _, err := authority.Approve("22000000000000000000000000000002", requestID, "leaf"); !errors.Is(err, ErrEnrollmentExpired) {
		t.Fatalf("expired request approval error = %v", err)
	}
	newDevice, _ := GenerateDeviceIdentity()
	if outcome, err := authority.Submit(EnrollmentSubmission{RequestID: "22000000000000000000000000000003", DevicePublicKey: newDevice.PublicKey, ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{2}}); err != nil || outcome.Status != "pending" {
		t.Fatalf("expired request did not release bounded Pending capacity: %#v, %v", outcome, err)
	}
	if len(authority.Enrollments()) != 0 {
		t.Fatal("expired Pending request allocated a Node ID")
	}
}

func TestEnrollmentAuthorityAppliesPendingQuotaPerParent(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	firstParent, _ := GenerateIdentity(2)
	secondParent, _ := GenerateIdentity(3)
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{MaxPending: 2, MaxPendingPerParent: 1})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := GenerateDeviceIdentity()
	second, _ := GenerateDeviceIdentity()
	third, _ := GenerateDeviceIdentity()
	if _, err := authority.Submit(EnrollmentSubmission{RequestID: "23000000000000000000000000000001", DevicePublicKey: first.PublicKey, ParentNodeID: 2, ParentPublicKey: firstParent.PublicKey, TranscriptDigest: [32]byte{1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := authority.Submit(EnrollmentSubmission{RequestID: "23000000000000000000000000000002", DevicePublicKey: second.PublicKey, ParentNodeID: 2, ParentPublicKey: firstParent.PublicKey, TranscriptDigest: [32]byte{2}}); err == nil {
		t.Fatal("one parent exceeded its Pending quota")
	}
	if _, err := authority.Submit(EnrollmentSubmission{RequestID: "23000000000000000000000000000003", DevicePublicKey: third.PublicKey, ParentNodeID: 3, ParentPublicKey: secondParent.PublicKey, TranscriptDigest: [32]byte{3}}); err != nil {
		t.Fatalf("another parent could not use remaining global Pending capacity: %v", err)
	}
}

func TestEnrollmentAuthorityRetriesNodeIDCollisionAndKeepsTombstone(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	parent, _ := GenerateIdentity(2)
	store, _ := keystore.New(t.TempDir())
	random := &scriptedEnrollmentRandom{}
	// First approval: NodeID 55 and enrollment ID 0x11. Second approval first sees
	// 55 again, then receives 56; the revoked 55 remains in used_node_ids.
	random.addUint64(55)
	random.add(bytes.Repeat([]byte{0x11}, 16))
	random.addUint64(55)
	random.addUint64(56)
	random.add(bytes.Repeat([]byte{0x22}, 16))
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{Random: random})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := GenerateDeviceIdentity()
	firstRequest := "30000000000000000000000000000001"
	_, _ = authority.Submit(EnrollmentSubmission{RequestID: firstRequest, DevicePublicKey: first.PublicKey, ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{1}})
	firstGrant, err := authority.Approve("30000000000000000000000000000002", firstRequest, "leaf")
	if err != nil {
		t.Fatal(err)
	}
	if firstGrant.NodeID != "55" {
		t.Fatalf("first Node ID = %s, want 55", firstGrant.NodeID)
	}
	if _, err := authority.RevokeEnrollment("30000000000000000000000000000003", firstGrant.EnrollmentID, "retired"); err != nil {
		t.Fatal(err)
	}
	second, _ := GenerateDeviceIdentity()
	secondRequest := "30000000000000000000000000000004"
	_, _ = authority.Submit(EnrollmentSubmission{RequestID: secondRequest, DevicePublicKey: second.PublicKey, ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{2}})
	secondGrant, err := authority.Approve("30000000000000000000000000000005", secondRequest, "leaf")
	if err != nil {
		t.Fatal(err)
	}
	if secondGrant.NodeID != "56" {
		t.Fatalf("second Node ID = %s, want collision retry to produce 56", secondGrant.NodeID)
	}
}

func TestEnrollmentAuthorityRejectsPersistedIdentityMismatch(t *testing.T) {
	firstIdentity, _ := GenerateIdentity(1)
	store, _ := keystore.New(t.TempDir())
	if _, err := LoadEnrollmentAuthority(firstIdentity, store, EnrollmentAuthorityConfig{}); err != nil {
		t.Fatal(err)
	}
	otherIdentity, _ := GenerateIdentity(2)
	if _, err := LoadEnrollmentAuthority(otherIdentity, store, EnrollmentAuthorityConfig{}); err == nil {
		t.Fatal("expected persisted authority identity mismatch")
	}
}

func TestEnrollmentAuthorityRejectsInconsistentPersistedRelationships(t *testing.T) {
	authorityIdentity, _ := GenerateIdentity(1)
	parent, _ := GenerateIdentity(2)
	device, _ := GenerateDeviceIdentity()
	store, _ := keystore.New(t.TempDir())
	authority, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	requestID := "70000000000000000000000000000001"
	if _, err := authority.Submit(EnrollmentSubmission{
		RequestID: requestID, DevicePublicKey: device.PublicKey,
		ParentNodeID: 2, ParentPublicKey: parent.PublicKey, TranscriptDigest: [32]byte{1},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := authority.Approve("70000000000000000000000000000002", requestID, "leaf"); err != nil {
		t.Fatal(err)
	}
	var persisted enrollmentAuthorityState
	found, err := store.Load("enrollment-authority.json", &persisted)
	if err != nil || !found || len(persisted.Requests) != 1 {
		t.Fatalf("load persisted Authority state: found=%v err=%v", found, err)
	}
	persisted.Requests[0].EnrollmentID = "ffffffffffffffffffffffffffffffff"
	if err := store.Save("enrollment-authority.json", persisted); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEnrollmentAuthority(authorityIdentity, store, EnrollmentAuthorityConfig{}); err == nil {
		t.Fatal("Authority accepted an approved request linked to the wrong Enrollment")
	}
}

type scriptedEnrollmentRandom struct {
	data bytes.Buffer
}

func (random *scriptedEnrollmentRandom) add(data []byte) {
	_, _ = random.data.Write(data)
}

func (random *scriptedEnrollmentRandom) addUint64(value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	random.add(raw[:])
}

func (random *scriptedEnrollmentRandom) Read(target []byte) (int, error) {
	return random.data.Read(target)
}
