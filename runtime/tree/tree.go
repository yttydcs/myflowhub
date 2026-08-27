package tree

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
)

var (
	ErrParentExists  = errors.New("parent already exists")
	ErrChildExists   = errors.New("child already exists")
	ErrCycle         = errors.New("tree cycle")
	ErrRouteConflict = errors.New("route conflict")
	ErrNotReachable  = errors.New("node is not reachable")
	ErrStaleEpoch    = errors.New("stale topology epoch")
	ErrForgedSource  = errors.New("source is not represented by child link")
)

type Direction uint8

const (
	DirectionLocal Direction = iota + 1
	DirectionUp
	DirectionDown
)

type Edge struct {
	Node  protocol.NodeID
	Epoch uint64
}

type Route struct {
	Target  protocol.NodeID
	NextHop protocol.NodeID
	Epoch   uint64
	Kind    Direction
}

type State struct {
	mu       sync.RWMutex
	local    protocol.NodeID
	parent   *Edge
	children map[protocol.NodeID]Edge
	routes   map[protocol.NodeID]protocol.NodeID
	parents  map[protocol.NodeID]protocol.NodeID
	epoch    uint64
}

func New(local protocol.NodeID) (*State, error) {
	if err := local.Validate(); err != nil {
		return nil, fmt.Errorf("create tree: %w", err)
	}
	return &State{
		local:    local,
		children: make(map[protocol.NodeID]Edge),
		routes:   make(map[protocol.NodeID]protocol.NodeID),
		parents:  make(map[protocol.NodeID]protocol.NodeID),
	}, nil
}

func (s *State) Local() protocol.NodeID { return s.local }

func (s *State) Epoch() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.epoch
}

func (s *State) NextEpoch() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.epoch + 1
}

func (s *State) Parent() (Edge, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.parent == nil {
		return Edge{}, false
	}
	return *s.parent, true
}

func (s *State) AttachParent(parent protocol.NodeID) (uint64, error) {
	if err := parent.Validate(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if parent == s.local || s.routes[parent] != 0 {
		return 0, ErrCycle
	}
	if s.parent != nil {
		return 0, ErrParentExists
	}
	s.epoch++
	s.parent = &Edge{Node: parent, Epoch: s.epoch}
	return s.epoch, nil
}

func (s *State) Reparent(parent protocol.NodeID) (uint64, error) {
	if err := parent.Validate(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if parent == s.local || s.routes[parent] != 0 {
		return 0, ErrCycle
	}
	s.epoch++
	s.parent = &Edge{Node: parent, Epoch: s.epoch}
	return s.epoch, nil
}

func (s *State) ActivateParent(parent protocol.NodeID, epoch uint64) error {
	if err := parent.Validate(); err != nil {
		return err
	}
	if epoch == 0 {
		return errors.New("parent epoch must be non-zero")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if parent == s.local || s.routes[parent] != 0 {
		return ErrCycle
	}
	if epoch < s.epoch || (epoch == s.epoch && s.parent != nil) {
		return ErrStaleEpoch
	}
	s.epoch = epoch
	s.parent = &Edge{Node: parent, Epoch: epoch}
	return nil
}

func (s *State) ReattachChild(child protocol.NodeID, epoch uint64) error {
	if epoch == 0 {
		return errors.New("child epoch must be non-zero")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.children[child]
	if !ok {
		return ErrNotReachable
	}
	if epoch <= current.Epoch {
		return ErrStaleEpoch
	}
	s.children[child] = Edge{Node: child, Epoch: epoch}
	return nil
}

func (s *State) DetachParent(parent protocol.NodeID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.parent == nil || s.parent.Node != parent {
		return false
	}
	s.epoch++
	s.parent = nil
	return true
}

func (s *State) DetachParentEpoch(parent protocol.NodeID, epoch uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.parent == nil || s.parent.Node != parent || s.parent.Epoch != epoch {
		return false
	}
	s.epoch++
	s.parent = nil
	return true
}

func (s *State) AttachChild(child protocol.NodeID, epoch uint64) error {
	if err := child.Validate(); err != nil {
		return err
	}
	if epoch == 0 {
		return errors.New("child epoch must be non-zero")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if child == s.local || (s.parent != nil && child == s.parent.Node) {
		return ErrCycle
	}
	if _, exists := s.children[child]; exists {
		return ErrChildExists
	}
	if via := s.routes[child]; via != 0 {
		return fmt.Errorf("%w: %d is already reachable via %d", ErrRouteConflict, child, via)
	}
	s.children[child] = Edge{Node: child, Epoch: epoch}
	s.routes[child] = child
	s.parents[child] = s.local
	return nil
}

func (s *State) ChildEpoch(child protocol.NodeID) (uint64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	edge, ok := s.children[child]
	return edge.Epoch, ok
}

func (s *State) Announce(via, descendant protocol.NodeID, edgeEpoch uint64) error {
	return s.AnnounceWithParent(via, descendant, via, edgeEpoch)
}

func (s *State) AnnounceWithParent(via, descendant, parent protocol.NodeID, edgeEpoch uint64) error {
	if err := descendant.Validate(); err != nil {
		return err
	}
	if err := parent.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	edge, ok := s.children[via]
	if !ok {
		return ErrNotReachable
	}
	if edge.Epoch != edgeEpoch {
		return ErrStaleEpoch
	}
	if descendant == s.local || (s.parent != nil && descendant == s.parent.Node) {
		return ErrCycle
	}
	if parent == descendant || (parent != via && s.routes[parent] != via) {
		return fmt.Errorf("%w: announced parent %d is not in child %d subtree", ErrForgedSource, parent, via)
	}
	if child, direct := s.children[descendant]; direct && child.Node != via {
		return ErrRouteConflict
	}
	if current := s.routes[descendant]; current != 0 && current != via {
		return fmt.Errorf("%w: %d already routes via %d", ErrRouteConflict, descendant, current)
	}
	s.routes[descendant] = via
	s.parents[descendant] = parent
	return nil
}

func (s *State) WithdrawChild(child protocol.NodeID) []protocol.NodeID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withdrawChildLocked(child)
}

func (s *State) WithdrawChildEpoch(child protocol.NodeID, epoch uint64) []protocol.NodeID {
	s.mu.Lock()
	defer s.mu.Unlock()
	edge, ok := s.children[child]
	if !ok || edge.Epoch != epoch {
		return nil
	}
	return s.withdrawChildLocked(child)
}

func (s *State) withdrawChildLocked(child protocol.NodeID) []protocol.NodeID {
	if _, ok := s.children[child]; !ok {
		return nil
	}
	delete(s.children, child)
	delete(s.parents, child)
	removed := make([]protocol.NodeID, 0)
	for target, via := range s.routes {
		if via == child {
			removed = append(removed, target)
			delete(s.routes, target)
			delete(s.parents, target)
		}
	}
	return removed
}

func (s *State) WithdrawRoute(via, descendant protocol.NodeID, edgeEpoch uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	edge, ok := s.children[via]
	if !ok {
		return ErrNotReachable
	}
	if edge.Epoch != edgeEpoch {
		return ErrStaleEpoch
	}
	if descendant == via {
		return errors.New("direct child must be withdrawn with WithdrawChild")
	}
	if s.routes[descendant] != via {
		return ErrNotReachable
	}
	delete(s.routes, descendant)
	delete(s.parents, descendant)
	return nil
}

func (s *State) ValidateSource(via, source protocol.NodeID, edgeEpoch uint64) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	edge, ok := s.children[via]
	if !ok {
		return ErrNotReachable
	}
	if edge.Epoch != edgeEpoch {
		return ErrStaleEpoch
	}
	if source == via || s.routes[source] == via {
		return nil
	}
	return ErrForgedSource
}

func (s *State) ValidateParentControl(parent protocol.NodeID, epoch uint64) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.parent == nil || s.parent.Node != parent {
		return ErrNotReachable
	}
	if s.parent.Epoch != epoch {
		return ErrStaleEpoch
	}
	return nil
}

func (s *State) RouteTo(target protocol.NodeID) (Route, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if target == s.local {
		return Route{Target: target, Kind: DirectionLocal, Epoch: s.epoch}, nil
	}
	if child := s.routes[target]; child != 0 {
		edge := s.children[child]
		return Route{Target: target, NextHop: child, Epoch: edge.Epoch, Kind: DirectionDown}, nil
	}
	if s.parent != nil {
		return Route{Target: target, NextHop: s.parent.Node, Epoch: s.parent.Epoch, Kind: DirectionUp}, nil
	}
	return Route{}, ErrNotReachable
}

func (s *State) Descendants() map[protocol.NodeID]protocol.NodeID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[protocol.NodeID]protocol.NodeID, len(s.routes))
	for target, via := range s.routes {
		result[target] = via
	}
	return result
}

type Relation struct {
	Node   protocol.NodeID
	Parent protocol.NodeID
}

func (s *State) Relations() []Relation {
	s.mu.RLock()
	relations := make([]Relation, 0, len(s.parents))
	parents := make(map[protocol.NodeID]protocol.NodeID, len(s.parents))
	for node, parent := range s.parents {
		relations = append(relations, Relation{Node: node, Parent: parent})
		parents[node] = parent
	}
	local := s.local
	s.mu.RUnlock()
	depth := func(node protocol.NodeID) int {
		value := 0
		seen := make(map[protocol.NodeID]struct{})
		for node != local {
			if _, exists := seen[node]; exists {
				return int(^uint(0) >> 1)
			}
			seen[node] = struct{}{}
			parent := parents[node]
			if parent == 0 {
				return int(^uint(0) >> 1)
			}
			node = parent
			value++
		}
		return value
	}
	sort.Slice(relations, func(i, j int) bool {
		left, right := depth(relations[i].Node), depth(relations[j].Node)
		if left != right {
			return left < right
		}
		return relations[i].Node < relations[j].Node
	})
	return relations
}

func (s *State) ParentOf(node protocol.NodeID) (protocol.NodeID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if node == s.local && s.parent != nil {
		return s.parent.Node, true
	}
	parent, ok := s.parents[node]
	return parent, ok
}
