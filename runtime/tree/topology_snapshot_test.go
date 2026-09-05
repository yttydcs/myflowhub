package tree

import (
	"cmp"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestTopologySnapshotTransitions(t *testing.T) {
	state, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	previous := TopologySnapshot{}
	check := func(want TopologySnapshot) {
		t.Helper()
		want.Local = 1
		want.Version.MembershipEpoch = previous.Version.MembershipEpoch + 1
		got, changed := state.TopologySnapshotIfChanged(previous.Version)
		if !changed {
			t.Fatal("topology change was missed")
		}
		assertTopologySnapshot(t, got, want)
		unchanged, changed := state.TopologySnapshotIfChanged(got.Version)
		if changed {
			t.Fatal("unchanged topology was copied")
		}
		assertTopologySnapshot(t, unchanged, TopologySnapshot{Version: got.Version})
		previous = got
	}
	check(TopologySnapshot{})
	if _, err := state.AttachParent(9); err != nil {
		t.Fatal(err)
	}
	want := TopologySnapshot{Version: TopologyVersion{Epoch: 1}, Parent: &Edge{Node: 9, Epoch: 1}}
	check(want)
	if err := state.AttachChild(2, 7); err != nil {
		t.Fatal(err)
	}
	want.Children = []Edge{{Node: 2, Epoch: 7}}
	want.Relations = []Relation{{Node: 2, Parent: 1}}
	check(want)
	if err := state.Announce(2, 3, 7); err != nil {
		t.Fatal(err)
	}
	want.Relations = append(want.Relations, Relation{Node: 3, Parent: 2})
	check(want)
	if err := state.AnnounceWithParent(2, 4, 3, 7); err != nil {
		t.Fatal(err)
	}
	want.Relations = append(want.Relations, Relation{Node: 4, Parent: 3})
	check(want)
	if err := state.AttachChild(5, 11); err != nil {
		t.Fatal(err)
	}
	want.Children = append(want.Children, Edge{Node: 5, Epoch: 11})
	want.Relations = append(want.Relations, Relation{Node: 5, Parent: 1})
	check(want)
	if err := state.ReattachChild(2, 8); err != nil {
		t.Fatal(err)
	}
	want.Children[0].Epoch = 8
	check(want)
	if err := state.AnnounceWithParent(2, 4, 2, 8); err != nil {
		t.Fatal(err)
	}
	want.Relations[2].Parent = 2
	check(want)
	if err := state.WithdrawRoute(2, 4, 8); err != nil {
		t.Fatal(err)
	}
	want.Relations = []Relation{{Node: 2, Parent: 1}, {Node: 3, Parent: 2}, {Node: 5, Parent: 1}}
	check(want)
	for _, parent := range []protocol.NodeID{8, 8} {
		if _, err := state.Reparent(parent); err != nil {
			t.Fatal(err)
		}
		want.Version.Epoch++
		want.Parent = &Edge{Node: parent, Epoch: want.Version.Epoch}
		check(want)
	}
	if removed := state.WithdrawChildEpoch(2, 8); len(removed) != 2 {
		t.Fatalf("withdraw child removed %v, want child and descendant", removed)
	}
	want.Children = []Edge{{Node: 5, Epoch: 11}}
	want.Relations = []Relation{{Node: 5, Parent: 1}}
	check(want)
	if !state.DetachParentEpoch(8, want.Version.Epoch) {
		t.Fatal("parent was not detached")
	}
	want.Version.Epoch++
	want.Parent = nil
	check(want)
	if err := state.ActivateParent(9, 20); err != nil {
		t.Fatal(err)
	}
	want.Version.Epoch = 20
	want.Parent = &Edge{Node: 9, Epoch: 20}
	check(want)
}

func TestTopologySnapshotCopiesAreIsolated(t *testing.T) {
	state, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.AttachParent(9); err != nil {
		t.Fatal(err)
	}
	if err := state.AttachChild(2, 7); err != nil {
		t.Fatal(err)
	}
	if err := state.Announce(2, 3, 7); err != nil {
		t.Fatal(err)
	}
	original, _ := state.TopologySnapshotIfChanged(TopologyVersion{})
	copy, changed := state.TopologySnapshotIfChanged(TopologyVersion{Epoch: original.Version.Epoch})
	if !changed {
		t.Fatal("zero membership version must force a copy")
	}
	copy.Local = 99
	copy.Parent.Node = 99
	copy.Children[0] = Edge{Node: 99, Epoch: 99}
	copy.Relations[0] = Relation{Node: 99, Parent: 99}
	fresh, _ := state.TopologySnapshotIfChanged(TopologyVersion{})
	assertTopologySnapshot(t, fresh, original)
	if _, err := state.Reparent(8); err != nil {
		t.Fatal(err)
	}
	if err := state.ReattachChild(2, 8); err != nil {
		t.Fatal(err)
	}
	state.WithdrawChild(2)
	assertTopologySnapshot(t, original, TopologySnapshot{
		Version: TopologyVersion{Epoch: 1, MembershipEpoch: 4}, Local: 1, Parent: &Edge{Node: 9, Epoch: 1},
		Children: []Edge{{Node: 2, Epoch: 7}}, Relations: []Relation{{Node: 2, Parent: 1}, {Node: 3, Parent: 2}},
	})
}

func TestTopologySnapshotUnchanged(t *testing.T) {
	state, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.AttachChild(2, 7); err != nil {
		t.Fatal(err)
	}
	if err := state.Announce(2, 3, 7); err != nil {
		t.Fatal(err)
	}
	before, _ := state.TopologySnapshotIfChanged(TopologyVersion{})
	if err := state.ReattachChild(2, 7); err != ErrStaleEpoch {
		t.Fatalf("stale reconnect: %v", err)
	}
	if removed := state.WithdrawChildEpoch(2, 6); len(removed) != 0 {
		t.Fatalf("stale withdrawal removed %v", removed)
	}
	if state.DetachParent(9) {
		t.Fatal("absent parent was detached")
	}
	if err := state.Announce(2, 4, 6); err != ErrStaleEpoch {
		t.Fatalf("stale announcement: %v", err)
	}
	allocations := testing.AllocsPerRun(100, func() {
		got, changed := state.TopologySnapshotIfChanged(before.Version)
		if changed || got.Version != before.Version ||
			got.Local != 0 || got.Parent != nil || got.Relations != nil || got.Children != nil {
			t.Fatalf("unchanged snapshot = %+v, changed = %v", got, changed)
		}
	})
	if allocations != 0 {
		t.Fatalf("unchanged snapshot allocated %g times, want zero", allocations)
	}
}

func TestTopologySnapshotEpochOnlyChange(t *testing.T) {
	state, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := state.TopologySnapshotIfChanged(TopologyVersion{})
	// Isolate the link generation to guard against consumers comparing only the
	// membership version, even if current public mutations advance both.
	state.mu.Lock()
	state.epoch++
	state.mu.Unlock()
	got, changed := state.TopologySnapshotIfChanged(before.Version)
	if !changed {
		t.Fatal("link generation change was missed")
	}
	assertTopologySnapshot(t, got, TopologySnapshot{Local: 1, Version: TopologyVersion{Epoch: 1, MembershipEpoch: before.Version.MembershipEpoch}})
}

func TestTopologySnapshotSaturatedMembership(t *testing.T) {
	state, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	state.mu.Lock()
	state.membershipEpoch = ^uint64(0)
	state.mu.Unlock()
	before, _ := state.TopologySnapshotIfChanged(TopologyVersion{})
	if err := state.AttachChild(2, 7); err != nil {
		t.Fatal(err)
	}
	got, changed := state.TopologySnapshotIfChanged(before.Version)
	if !changed {
		t.Fatal("saturated membership hid a topology change")
	}
	assertTopologySnapshot(t, got, TopologySnapshot{
		Local: 1, Version: TopologyVersion{MembershipEpoch: ^uint64(0)},
		Children: []Edge{{Node: 2, Epoch: 7}}, Relations: []Relation{{Node: 2, Parent: 1}},
	})
}

func TestTopologySnapshotConcurrentConsistency(t *testing.T) {
	const cycles = 100
	serial, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	want := make(map[uint64]TopologySnapshot)
	record := func() {
		snapshot, _ := serial.TopologySnapshotIfChanged(TopologyVersion{})
		want[snapshot.Version.MembershipEpoch] = snapshot
	}
	record()
	if err := mutateTopologySnapshots(serial, cycles, record); err != nil {
		t.Fatal(err)
	}
	state, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	var readers sync.WaitGroup
	var ready sync.WaitGroup
	done := make(chan struct{})
	for range 4 {
		readers.Add(1)
		ready.Add(1)
		go func() {
			defer readers.Done()
			previous, _ := state.TopologySnapshotIfChanged(TopologyVersion{})
			ready.Done()
			assertTopologySnapshot(t, previous, want[previous.Version.MembershipEpoch])
			for {
				got, changed := state.TopologySnapshotIfChanged(previous.Version)
				if changed {
					if got.Version.MembershipEpoch <= previous.Version.MembershipEpoch {
						t.Errorf("membership went backwards: %d -> %d", previous.Version.MembershipEpoch, got.Version.MembershipEpoch)
						return
					}
					assertTopologySnapshot(t, got, want[got.Version.MembershipEpoch])
					previous = got
				} else {
					assertTopologySnapshot(t, got, TopologySnapshot{Version: previous.Version})
				}
				select {
				case <-done:
					return
				default:
					runtime.Gosched()
				}
			}
		}()
	}
	ready.Wait()
	err = mutateTopologySnapshots(state, cycles, runtime.Gosched)
	close(done)
	readers.Wait()
	if err != nil {
		t.Fatal(err)
	}
	final, _ := state.TopologySnapshotIfChanged(TopologyVersion{})
	assertTopologySnapshot(t, final, want[final.Version.MembershipEpoch])
}

func mutateTopologySnapshots(state *State, cycles int, after func()) error {
	for i := range cycles {
		epoch := uint64(2*i + 1)
		parent := protocol.NodeID(8 + i%2)
		steps := []func() error{
			func() error { _, err := state.Reparent(parent); return err },
			func() error { return state.AttachChild(2, epoch) },
			func() error { return state.Announce(2, 3, epoch) },
			func() error { return state.AnnounceWithParent(2, 4, 3, epoch) },
			func() error { return state.ReattachChild(2, epoch+1) },
			func() error { return state.AnnounceWithParent(2, 4, 2, epoch+1) },
			func() error { return state.WithdrawRoute(2, 4, epoch+1) },
			func() error {
				if removed := state.WithdrawChildEpoch(2, epoch+1); len(removed) != 2 {
					return fmt.Errorf("withdraw child removed %v, want two nodes", removed)
				}
				return nil
			},
			func() error {
				if !state.DetachParent(parent) {
					return fmt.Errorf("parent %d was not detached", parent)
				}
				return nil
			},
		}
		for index, step := range steps {
			if err := step(); err != nil {
				return fmt.Errorf("cycle %d step %d: %w", i, index, err)
			}
			after()
		}
	}
	return nil
}

func assertTopologySnapshot(t *testing.T, got, want TopologySnapshot) {
	t.Helper()
	if got.Version != want.Version ||
		got.Local != want.Local || (got.Parent == nil) != (want.Parent == nil) || (got.Parent != nil && *got.Parent != *want.Parent) {
		t.Fatalf("snapshot metadata = %+v, want %+v", got, want)
	}
	gotRelations, wantRelations := slices.Clone(got.Relations), slices.Clone(want.Relations)
	sortRelations := func(a, b Relation) int { return cmp.Compare(a.Node, b.Node) }
	slices.SortFunc(gotRelations, sortRelations)
	slices.SortFunc(wantRelations, sortRelations)
	if !slices.Equal(gotRelations, wantRelations) {
		t.Fatalf("snapshot relations at %d = %v, want %v", got.Version.MembershipEpoch, gotRelations, wantRelations)
	}
	gotChildren, wantChildren := slices.Clone(got.Children), slices.Clone(want.Children)
	sortChildren := func(a, b Edge) int { return cmp.Compare(a.Node, b.Node) }
	slices.SortFunc(gotChildren, sortChildren)
	slices.SortFunc(wantChildren, sortChildren)
	if !slices.Equal(gotChildren, wantChildren) {
		t.Fatalf("snapshot children at %d = %v, want %v", got.Version.MembershipEpoch, gotChildren, wantChildren)
	}
}
