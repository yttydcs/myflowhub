package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
)

func TestAdmissionPermitIsBoundSingleUseAndDurable(t *testing.T) {
	parent, _ := GenerateIdentity(1)
	child, _ := GenerateIdentity(2)
	clock := time.Unix(1000, 0)
	store, _ := keystore.New(t.TempDir())
	admission, err := LoadAdmission(parent, store, AdmissionConfig{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := admission.Issue(child.NodeID, child.PublicKey, "device", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := admission.Consume(child.NodeID, child.PublicKey, permit); err != nil {
		t.Fatal(err)
	}
	restarted, err := LoadAdmission(parent, store, AdmissionConfig{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Consume(child.NodeID, child.PublicKey, permit); !errors.Is(err, ErrPermitConsumed) {
		t.Fatalf("replayed permit error = %v", err)
	}
	otherChild, _ := GenerateIdentity(3)
	if err := restarted.Consume(otherChild.NodeID, otherChild.PublicKey, permit); !errors.Is(err, ErrPermitInvalid) {
		t.Fatalf("child binding error = %v", err)
	}
}

func TestAdmissionRejectsExpiryRevocationAndFakeParent(t *testing.T) {
	parent, _ := GenerateIdentity(1)
	child, _ := GenerateIdentity(2)
	clock := time.Unix(2000, 0)
	store, _ := keystore.New(t.TempDir())
	admission, _ := LoadAdmission(parent, store, AdmissionConfig{Now: func() time.Time { return clock }})
	expired, _ := admission.Issue(child.NodeID, child.PublicKey, "device", time.Second)
	clock = clock.Add(2 * time.Second)
	if err := admission.Consume(child.NodeID, child.PublicKey, expired); !errors.Is(err, ErrPermitInvalid) {
		t.Fatalf("expired permit error = %v", err)
	}
	clock = clock.Add(time.Second)
	revoked, _ := admission.Issue(child.NodeID, child.PublicKey, "device", time.Hour)
	if err := admission.Revoke(revoked.PermitID); err != nil {
		t.Fatal(err)
	}
	if err := admission.Consume(child.NodeID, child.PublicKey, revoked); !errors.Is(err, ErrPermitRevoked) {
		t.Fatalf("revoked permit error = %v", err)
	}
	fakeParent, _ := GenerateIdentity(9)
	fakeStore, _ := keystore.New(t.TempDir())
	fakeAdmission, _ := LoadAdmission(fakeParent, fakeStore, AdmissionConfig{Now: func() time.Time { return clock }})
	if err := fakeAdmission.Consume(child.NodeID, child.PublicKey, revoked); !errors.Is(err, ErrPermitInvalid) {
		t.Fatalf("fake parent error = %v", err)
	}
}
