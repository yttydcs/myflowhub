package auth

import (
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestOpenStatePersistsIdentityTrustAndPolicy(t *testing.T) {
	directory := t.TempDir()
	state, err := OpenState(directory, 7)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := GenerateIdentity(8)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Trust.Add(peer.NodeID, peer.PublicKey); err != nil {
		t.Fatal(err)
	}
	request := Request{Subject: peer.NodeID, Action: ActionSubscribe, Resource: protocol.ResourceID{Owner: 7, Name: "state/test"}}
	if err := state.Policy.Grant(request); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenState(directory, 7)
	if err != nil {
		t.Fatal(err)
	}
	if string(reopened.Identity.PublicKey) != string(state.Identity.PublicKey) {
		t.Fatal("runtime identity was not durable")
	}
	if _, trusted := reopened.Trust.PublicKey(peer.NodeID); !trusted {
		t.Fatal("runtime trust was not durable")
	}
	if err := reopened.Policy.Authorize(nil, request); err != nil {
		t.Fatalf("runtime policy was not durable: %v", err)
	}
}
