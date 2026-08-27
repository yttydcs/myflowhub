package tree

import (
	"errors"
	"sync"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestRoutesAndSourceProof(t *testing.T) {
	state, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	parentEpoch, err := state.AttachParent(9)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.AttachChild(2, 4); err != nil {
		t.Fatal(err)
	}
	if err := state.Announce(2, 3, 4); err != nil {
		t.Fatal(err)
	}
	down, err := state.RouteTo(3)
	if err != nil || down.Kind != DirectionDown || down.NextHop != 2 || down.Epoch != 4 {
		t.Fatalf("unexpected down route: %#v, %v", down, err)
	}
	up, err := state.RouteTo(99)
	if err != nil || up.Kind != DirectionUp || up.NextHop != 9 || up.Epoch != parentEpoch {
		t.Fatalf("unexpected up route: %#v, %v", up, err)
	}
	if err := state.ValidateSource(2, 3, 4); err != nil {
		t.Fatal(err)
	}
	if err := state.ValidateSource(2, 4, 4); !errors.Is(err, ErrForgedSource) {
		t.Fatalf("expected forged source, got %v", err)
	}
}

func TestReparentInvalidatesOldControl(t *testing.T) {
	state, _ := New(1)
	oldEpoch, _ := state.AttachParent(9)
	if err := state.ValidateParentControl(9, oldEpoch); err != nil {
		t.Fatal(err)
	}
	newEpoch, err := state.Reparent(8)
	if err != nil {
		t.Fatal(err)
	}
	if newEpoch <= oldEpoch {
		t.Fatalf("epoch did not advance: old %d new %d", oldEpoch, newEpoch)
	}
	if err := state.ValidateParentControl(9, oldEpoch); !errors.Is(err, ErrNotReachable) {
		t.Fatalf("old parent retained authority: %v", err)
	}
	if err := state.ValidateParentControl(8, oldEpoch); !errors.Is(err, ErrStaleEpoch) {
		t.Fatalf("old epoch retained authority: %v", err)
	}
}

func TestSignedParentEpochSurvivesOldSessionCleanup(t *testing.T) {
	state, _ := New(2)
	if _, err := state.AttachParent(1); err != nil {
		t.Fatal(err)
	}
	if !state.DetachParentEpoch(1, 1) {
		t.Fatal("old parent edge was not detached")
	}
	if err := state.ActivateParent(1, 2); err != nil {
		t.Fatalf("activate signed epoch after old cleanup: %v", err)
	}
	if state.DetachParentEpoch(1, 1) {
		t.Fatal("stale session detached the active parent edge")
	}
	parent, ok := state.Parent()
	if !ok || parent.Node != 1 || parent.Epoch != 2 {
		t.Fatalf("unexpected active parent: %#v", parent)
	}
}

func TestStaleChildSessionCannotWithdrawReattachedEdge(t *testing.T) {
	state, _ := New(1)
	if err := state.AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	if err := state.ReattachChild(2, 2); err != nil {
		t.Fatal(err)
	}
	if removed := state.WithdrawChildEpoch(2, 1); len(removed) != 0 {
		t.Fatalf("stale session removed routes: %#v", removed)
	}
	if epoch, ok := state.ChildEpoch(2); !ok || epoch != 2 {
		t.Fatalf("reattached edge was lost: epoch=%d ok=%v", epoch, ok)
	}
}

func TestConcurrentReparentMaintainsOneParent(t *testing.T) {
	state, _ := New(1)
	_, _ = state.AttachParent(2)
	var wait sync.WaitGroup
	for _, parent := range []protocol.NodeID{3, 4, 5, 6} {
		wait.Add(1)
		go func(parent protocol.NodeID) {
			defer wait.Done()
			_, _ = state.Reparent(parent)
		}(parent)
	}
	wait.Wait()
	parent, ok := state.Parent()
	if !ok || parent.Node < 3 || parent.Node > 6 {
		t.Fatalf("invalid final parent: %#v", parent)
	}
}

func TestCycleAndRouteConflict(t *testing.T) {
	state, _ := New(1)
	_, _ = state.AttachParent(9)
	if err := state.AttachChild(1, 1); !errors.Is(err, ErrCycle) {
		t.Fatalf("expected self cycle, got %v", err)
	}
	_ = state.AttachChild(2, 1)
	_ = state.AttachChild(3, 1)
	_ = state.Announce(2, 4, 1)
	if err := state.Announce(3, 4, 1); !errors.Is(err, ErrRouteConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestRelationsPreserveImmediateParentsAndRejectForgedParent(t *testing.T) {
	state, _ := New(1)
	if err := state.AttachChild(2, 7); err != nil {
		t.Fatal(err)
	}
	if err := state.AnnounceWithParent(2, 3, 2, 7); err != nil {
		t.Fatal(err)
	}
	if err := state.AnnounceWithParent(2, 4, 3, 7); err != nil {
		t.Fatal(err)
	}
	if err := state.AnnounceWithParent(2, 5, 99, 7); !errors.Is(err, ErrForgedSource) {
		t.Fatalf("expected forged parent rejection, got %v", err)
	}
	want := []Relation{{Node: 2, Parent: 1}, {Node: 3, Parent: 2}, {Node: 4, Parent: 3}}
	got := state.Relations()
	if len(got) != len(want) {
		t.Fatalf("unexpected relations: %#v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("relation %d: want %#v, got %#v", index, want[index], got[index])
		}
	}
	if parent, ok := state.ParentOf(4); !ok || parent != 3 {
		t.Fatalf("unexpected parent for node 4: %d, %v", parent, ok)
	}
}
