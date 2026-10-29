package node

import (
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestForwardedSubscriptionWithoutReadyNextHopFailsAndReleasesInterest(t *testing.T) {
	ctx, root, child, _, _ := newSubscriptionFailureNodes(t)
	// Join makes the route visible before the corresponding session is installed.
	// Keep that intermediate state stable so the regression is deterministic.
	if err := root.Tree().AttachChild(3, root.Tree().NextEpoch()); err != nil {
		t.Fatal(err)
	}
	remote, err := child.Subscribe(ctx, protocol.ResourceID{Owner: 3, Name: "joining/value"}, time.Minute, 4)
	if remote != nil {
		remote.Cancel()
		t.Fatal("unavailable next hop returned a live subscription")
	}
	var failure protocol.ErrorPayload
	if !errors.As(err, &failure) || failure.Code != protocol.CodeStaleEpoch {
		t.Fatalf("expected prompt stale-epoch response, got %v", err)
	}
	root.mu.RLock()
	retained := len(root.forwardedSubscriptions)
	root.mu.RUnlock()
	if retained != 0 || child.Stats().PendingRequests != 0 {
		t.Fatalf("failed setup retained state: forwarded=%d, client=%+v", retained, child.Stats())
	}
}
