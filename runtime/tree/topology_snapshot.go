package tree

import "github.com/yttydcs/myflowhub/protocol"

// TopologyVersion identifies one State's topology. Epoch is the existing link
// generation; MembershipEpoch also tracks child edges and descendant relations.
type TopologyVersion struct {
	Epoch           uint64
	MembershipEpoch uint64
}

// TopologySnapshot is a consistent copy of one State's topology. Relations
// includes direct children and announced descendants, with their immediate
// parents; it excludes Local and Local's parent edge. Children contains only
// direct child edges. Parent is nil when detached. The parent edge and both
// unordered slices are owned by the caller.
type TopologySnapshot struct {
	Version   TopologyVersion
	Local     protocol.NodeID
	Parent    *Edge
	Children  []Edge
	Relations []Relation
}

// TopologySnapshotIfChanged compares both versions under the same read lock used
// to copy the topology. Pass Version from a prior snapshot of this State, or a
// version with MembershipEpoch == 0 to force a copy.
//
// When unchanged, it returns only the current Version and false without
// allocating or copying the topology. A saturated membership counter always
// forces a copy because subsequent mutations can no longer advance that counter.
func (s *State) TopologySnapshotIfChanged(version TopologyVersion) (TopologySnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := TopologySnapshot{
		Version: TopologyVersion{Epoch: s.epoch, MembershipEpoch: s.membershipEpoch},
	}
	if version.MembershipEpoch != 0 && s.membershipEpoch != ^uint64(0) && version == snapshot.Version {
		return snapshot, false
	}

	snapshot.Local = s.local
	if s.parent != nil {
		parent := *s.parent
		snapshot.Parent = &parent
	}
	snapshot.Relations = make([]Relation, 0, len(s.parents))
	for node, parent := range s.parents {
		snapshot.Relations = append(snapshot.Relations, Relation{Node: node, Parent: parent})
	}
	snapshot.Children = make([]Edge, 0, len(s.children))
	for _, child := range s.children {
		snapshot.Children = append(snapshot.Children, child)
	}
	return snapshot, true
}
