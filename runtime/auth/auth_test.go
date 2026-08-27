package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

func TestSignedJoinAndAcknowledgement(t *testing.T) {
	parent, _ := GenerateIdentity(1)
	child, _ := GenerateIdentity(2)
	store := NewTrustStore()
	_ = store.Add(parent.NodeID, parent.PublicKey)
	_ = store.Add(child.NodeID, child.PublicKey)
	claim, err := NewJoinClaim(child, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.VerifyJoin(claim); err != nil {
		t.Fatal(err)
	}
	ack, err := SignJoinAck(parent, child.NodeID, claim.Nonce, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.VerifyJoinAck(ack, parent.NodeID, child.NodeID, claim.Nonce); err != nil {
		t.Fatal(err)
	}
	claim.Nonce[0]++
	if err := store.VerifyJoin(claim); err == nil {
		t.Fatal("tampered claim was accepted")
	}
}

func TestInboundAuthorityValidation(t *testing.T) {
	state, _ := tree.New(2)
	epoch, _ := state.AttachParent(1)
	_ = state.AttachChild(3, 7)
	_ = state.Announce(3, 4, 7)
	resource := protocol.ResourceID{Owner: 2, Name: "state"}
	control := protocol.Envelope{Phase: protocol.PhaseControl, Source: 1, Resource: resource, TopologyEpoch: epoch}
	if err := ValidateInboundParent(state, 1, control); err != nil {
		t.Fatal(err)
	}
	control.TopologyEpoch = epoch + 1
	if err := ValidateInboundParent(state, 1, control); !errors.Is(err, tree.ErrStaleEpoch) {
		t.Fatalf("expected stale epoch, got %v", err)
	}
	fromChild := protocol.Envelope{Phase: protocol.PhaseRequest, Source: 4, Resource: resource}
	if err := ValidateInboundChild(state, 3, 7, fromChild); err != nil {
		t.Fatal(err)
	}
	fromChild.Source = 5
	if err := ValidateInboundChild(state, 3, 7, fromChild); !errors.Is(err, tree.ErrForgedSource) {
		t.Fatalf("expected forged source, got %v", err)
	}
	fromChild.Source = 3
	fromChild.Phase = protocol.PhaseControl
	if err := ValidateInboundChild(state, 3, 7, fromChild); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden control, got %v", err)
	}
}

func TestStaticPolicyDefaultsToDeny(t *testing.T) {
	resource := protocol.ResourceID{Owner: 2, Name: "restart"}
	request := Request{Subject: 3, Action: ActionInvoke, Resource: resource}
	policy := NewStaticPolicy()
	if err := policy.Authorize(context.Background(), request); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected default deny, got %v", err)
	}
	policy.Allow(request)
	if err := policy.Authorize(context.Background(), request); err != nil {
		t.Fatal(err)
	}
}
