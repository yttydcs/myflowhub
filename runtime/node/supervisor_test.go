package node

import (
	"context"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestParentSupervisorReconnectsOneHundredTimes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	parentIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	_ = trust.Add(parentIdentity.NodeID, parentIdentity.PublicKey)
	_ = trust.Add(childIdentity.NodeID, childIdentity.PublicKey)
	child, err := New(ctx, Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	network := memory.NewNetwork()
	defer network.Close()
	startParent := func() *Node {
		parent, err := New(ctx, Config{Identity: parentIdentity, Trust: trust, Policy: auth.AllowAll{}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parent.Listen(network, "supervised-parent"); err != nil {
			t.Fatal(err)
		}
		return parent
	}
	parent := startParent()
	supervisor, err := child.SuperviseParent(ctx, network, "supervised-parent", parentIdentity.NodeID, SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Stop()
	waitConnectionState(t, supervisor, ConnectionConnected, 2*time.Second)
	for index := 0; index < 100; index++ {
		if err := parent.Close(); err != nil {
			t.Fatal(err)
		}
		parent = startParent()
		waitConnectionAttempt(t, supervisor, uint64(index+2), 2*time.Second)
	}
	_ = parent.Close()
}

func TestParentSupervisorStopsInsteadOfRestoringAfterReparent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	firstIdentity, _ := auth.GenerateIdentity(1)
	secondIdentity, _ := auth.GenerateIdentity(3)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	for _, identity := range []auth.Identity{firstIdentity, secondIdentity, childIdentity} {
		_ = trust.Add(identity.NodeID, identity.PublicKey)
	}
	network := memory.NewNetwork()
	defer network.Close()
	first, _ := New(ctx, Config{Identity: firstIdentity, Trust: trust, Policy: auth.AllowAll{}})
	defer first.Close()
	second, _ := New(ctx, Config{Identity: secondIdentity, Trust: trust, Policy: auth.AllowAll{}})
	defer second.Close()
	child, _ := New(ctx, Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}})
	defer child.Close()
	firstEndpoint, _ := first.Listen(network, "first-parent")
	secondEndpoint, _ := second.Listen(network, "second-parent")
	supervisor, err := child.SuperviseParent(ctx, network, firstEndpoint, first.ID(), SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Stop()
	waitConnectionState(t, supervisor, ConnectionConnected, time.Second)
	if err := child.ConnectParent(ctx, network, secondEndpoint, second.ID()); err != nil {
		t.Fatal(err)
	}
	failed := waitConnectionState(t, supervisor, ConnectionFailed, time.Second)
	if failed.LastError != ErrReparented.Error() {
		t.Fatalf("unexpected supervisor failure %q", failed.LastError)
	}
	status, _ := child.ParentStatus()
	if !status.Connected || status.Parent != second.ID() {
		t.Fatalf("reparented link was not retained: %+v", status)
	}
}

func TestBackoffIsBoundedAcrossOneHundredFailures(t *testing.T) {
	delay := time.Millisecond
	for range 100 {
		delay = nextBackoff(delay, time.Second, 2)
	}
	if delay != time.Second {
		t.Fatalf("backoff did not clamp: %s", delay)
	}
}

func waitConnectionAttempt(t *testing.T, supervisor *ParentSupervisor, minimum uint64, timeout time.Duration) ConnectionSnapshot {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		current := supervisor.Snapshot()
		if current.State == ConnectionConnected && current.Attempt >= minimum {
			return current
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("supervisor did not connect at attempt %d: %+v", minimum, supervisor.Snapshot())
	return ConnectionSnapshot{}
}

func waitConnectionState(t *testing.T, supervisor *ParentSupervisor, state ConnectionState, timeout time.Duration) ConnectionSnapshot {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		current := supervisor.Snapshot()
		if current.State == state {
			return current
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("supervisor did not reach state %d: %+v", state, supervisor.Snapshot())
	return ConnectionSnapshot{}
}
