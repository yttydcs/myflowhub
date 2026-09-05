package management

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

const topologyCacheEntries = 8
const topologyCacheBytes = topologyCacheEntries * protocol.DefaultMaxPayload
const maxTopologyRevision = 1<<53 - 1

type topologyResult struct {
	payload []byte
	err     error
}

// The tree-derived fields are immutable after publication. Only the bounded
// cache of less common depths is mutable, and never holds a runtime tree lock.
type topologySnapshot struct {
	version     tree.TopologyVersion
	displayName string
	instanceID  string
	revision    uint64
	nodes       []protocol.TopologyQueryNodeV1
	levels      []int
	shallow     topologyResult
	full        topologyResult
	legacy      []byte
	legacyErr   error
	mu          sync.Mutex
	cache       map[int]topologyResult
	order       []int
	cacheBytes  int
}

type topologyResource struct {
	base         *resource.Variable
	instanceID   string
	current      atomic.Pointer[topologySnapshot]
	failure      atomic.Pointer[topologyFailure]
	publishMu    sync.Mutex
	publishing   bool
	pending      *topologySnapshot
	published    uint64
	observersMu  sync.Mutex
	nextObserver uint64
	observers    map[uint64]topologyObserver
}

type topologyFailure struct{ err error }
type topologyObserver struct {
	callback func(resource.Observation)
	cancel   func()
}

func newTopologyResource(base *resource.Variable) (*topologyResource, error) {
	var instance [16]byte
	if _, err := rand.Read(instance[:]); err != nil {
		return nil, fmt.Errorf("topology instance: %w", err)
	}
	return &topologyResource{base: base, instanceID: hex.EncodeToString(instance[:]), observers: make(map[uint64]topologyObserver)}, nil
}

func (r *topologyResource) Descriptor() resource.Descriptor {
	descriptor := r.base.Descriptor()
	descriptor.Capabilities = append(descriptor.Capabilities,
		protocol.CapabilityDescriptorV2{Name: protocol.CapabilityChildren, Permission: "management.topology.children", InputSchema: protocol.SchemaManagementTopologyChildrenRequestV1, OutputSchema: protocol.SchemaManagementTopologyQueryV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
		protocol.CapabilityDescriptorV2{Name: protocol.CapabilitySubtree, Permission: "management.topology.subtree", InputSchema: protocol.SchemaManagementTopologyQueryRequestV1, OutputSchema: protocol.SchemaManagementTopologyQueryV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
	)
	for _, schema := range []string{protocol.SchemaManagementTopologyChildrenRequestV1, protocol.SchemaManagementTopologyQueryRequestV1, protocol.SchemaManagementTopologyQueryV1} {
		descriptor.Schemas = append(descriptor.Schemas, protocol.SchemaDescriptorV2{ID: schema, ContentType: contentTypeJSON})
	}
	descriptor.Sort()
	return descriptor
}

func (r *topologyResource) Operate(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if ctx == nil {
		return resource.OperationResult{}, errors.New("topology context is required")
	}
	if err := ctx.Err(); err != nil {
		return resource.OperationResult{}, err
	}
	if failure := r.failure.Load(); failure != nil {
		return resource.OperationResult{}, failure.err
	}
	depth := 1
	switch request.Capability {
	case protocol.CapabilityChildren:
		var input protocol.ManagementTopologyChildrenRequestV1
		if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
			return resource.OperationResult{}, err
		}
		depth = input.Depth
	case protocol.CapabilitySubtree:
		var input protocol.ManagementTopologyQueryRequestV1
		if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
			return resource.OperationResult{}, err
		}
		depth = input.Depth
	default:
		if current := r.current.Load(); current != nil && request.Capability == protocol.CapabilityRead {
			if current.legacyErr != nil {
				return resource.OperationResult{}, current.legacyErr
			}
			return resource.OperationResult{Schema: protocol.SchemaManagementTopologyV1, Payload: append([]byte(nil), current.legacy...)}, nil
		}
		return r.base.Operate(ctx, request)
	}
	current := r.current.Load()
	if current == nil {
		return resource.OperationResult{}, errors.New("topology snapshot is not ready")
	}
	result := current.query(depth)
	return resource.OperationResult{Schema: protocol.SchemaManagementTopologyQueryV1, Payload: append([]byte(nil), result.payload...)}, result.err
}

func (r *topologyResource) Observe(observer func(resource.Observation)) (*resource.Observation, func(), error) {
	if observer == nil {
		return nil, nil, errors.New("topology observer is required")
	}
	r.observersMu.Lock()
	defer r.observersMu.Unlock()
	if failure := r.failure.Load(); failure != nil {
		return nil, nil, failure.err
	}
	if current := r.current.Load(); current != nil && current.legacyErr != nil {
		return nil, nil, current.legacyErr
	}
	snapshot, cancel, err := r.base.Observe(observer)
	if err != nil {
		return nil, nil, err
	}
	r.nextObserver++
	id := r.nextObserver
	r.observers[id] = topologyObserver{callback: observer, cancel: cancel}
	return snapshot, func() {
		r.observersMu.Lock()
		delete(r.observers, id)
		r.observersMu.Unlock()
		cancel()
	}, nil
}

// A single drainer preserves callback order even when a callback changes the
// tree and re-enters Refresh. Concurrent refreshes coalesce the pending latest
// Variable value. No observer runs while a publication or controller lock is held.
func (r *topologyResource) publish(snapshot *topologySnapshot) error {
	r.publishMu.Lock()
	if snapshot.revision <= r.published || r.pending != nil && snapshot.revision <= r.pending.revision {
		r.publishMu.Unlock()
		return nil
	}
	r.pending = snapshot
	if r.publishing {
		r.publishMu.Unlock()
		return nil
	}
	r.publishing = true
	var result error
	for {
		current := r.pending
		r.pending = nil
		r.published = current.revision
		r.publishMu.Unlock()
		if current.legacyErr != nil {
			r.failObservers(current.legacyErr)
		} else {
			_, err := r.base.Apply(current.revision+1, current.legacy)
			if err != nil && !errors.Is(err, resource.ErrRevisionRegression) {
				result = errors.Join(result, err)
			}
		}
		r.publishMu.Lock()
		if r.pending == nil {
			r.publishing = false
			r.publishMu.Unlock()
			return result
		}
	}
}

func (r *topologyResource) failObservers(err error) {
	r.observersMu.Lock()
	observers := r.observers
	r.observers = make(map[uint64]topologyObserver)
	r.observersMu.Unlock()
	code := protocol.CodeInternal
	if errors.Is(err, protocol.ErrPayloadTooLarge) {
		code = protocol.CodeOverflow
	}
	for _, observer := range observers {
		observer.cancel()
		observer.callback(resource.Observation{Failure: &protocol.ErrorPayload{Code: code, Message: err.Error()}})
	}
}

func (c *Controller) refreshTopologyLocked() (*topologySnapshot, error) {
	previous := c.topologyQuery.current.Load()
	var version tree.TopologyVersion
	if previous != nil {
		version = previous.version
	}
	displayName := c.settings.Snapshot().DisplayName
	if previous != nil && previous.displayName != displayName {
		version = tree.TopologyVersion{}
	}
	state, changed := c.node.Tree().TopologySnapshotIfChanged(version)
	if !changed {
		return nil, previous.legacyErr
	}
	revision := uint64(1)
	if previous != nil {
		if previous.revision == maxTopologyRevision {
			err := errors.New("topology revision exhausted; restart provider")
			c.topologyQuery.failure.Store(&topologyFailure{err: err})
			return nil, err
		}
		revision = previous.revision + 1
	}
	current, err := buildTopologySnapshot(state, displayName, c.topologyQuery.instanceID, revision)
	if err != nil {
		c.topologyQuery.failure.Store(&topologyFailure{err: err})
		return nil, err
	}
	c.topologyQuery.current.Store(current)
	c.topologyQuery.failure.Store(nil)
	return current, current.legacyErr
}

func buildTopologySnapshot(state tree.TopologySnapshot, displayName, instanceID string, revision uint64) (*topologySnapshot, error) {
	epoch := state.Version.Epoch
	if epoch == 0 {
		epoch = 1
	}
	children := make(map[protocol.NodeID][]protocol.NodeID)
	for _, relation := range state.Relations {
		children[relation.Parent] = append(children[relation.Parent], relation.Node)
	}
	for _, ids := range children {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}
	role := "node"
	if state.Parent == nil {
		role = "root"
	}
	idText := func(id protocol.NodeID) string { return strconv.FormatUint(uint64(id), 10) }
	current := &topologySnapshot{version: state.Version, displayName: displayName, instanceID: instanceID, revision: revision, cache: make(map[int]topologyResult)}
	current.nodes = []protocol.TopologyQueryNodeV1{{NodeID: idText(state.Local), DisplayName: displayName, Role: role, Generation: epoch, HasChildren: len(children[state.Local]) != 0}}
	current.levels = []int{0}
	queue := []protocol.NodeID{state.Local}
	seen := map[protocol.NodeID]bool{state.Local: true}
	for index := 0; index < len(queue); index++ {
		parent := queue[index]
		for _, id := range children[parent] {
			if seen[id] {
				return nil, fmt.Errorf("runtime topology contains duplicate or cyclic node %d", id)
			}
			seen[id] = true
			queue = append(queue, id)
			current.nodes = append(current.nodes, protocol.TopologyQueryNodeV1{NodeID: idText(id), ParentID: idText(parent), Role: "node", Generation: epoch, HasChildren: len(children[id]) != 0})
			current.levels = append(current.levels, current.levels[index]+1)
		}
	}
	if len(current.nodes) != len(state.Relations)+1 {
		return nil, errors.New("runtime topology contains unreachable relations")
	}
	// Preserve the legacy depth-then-numeric-ID ordering across sibling groups.
	order := make([]int, len(queue))
	for index := range order {
		order[index] = index
	}
	sort.Slice(order, func(i, j int) bool {
		left, right := order[i], order[j]
		if current.levels[left] != current.levels[right] {
			return current.levels[left] < current.levels[right]
		}
		return queue[left] < queue[right]
	})
	nodes := make([]protocol.TopologyQueryNodeV1, len(order))
	levels := make([]int, len(order))
	for index, original := range order {
		nodes[index], levels[index] = current.nodes[original], current.levels[original]
	}
	current.nodes, current.levels = nodes, levels
	current.shallow = current.encode(1)
	current.full = current.encode(0)
	if len(current.nodes) > protocol.MaxItems {
		current.legacyErr = topologyOverflow(len(current.nodes))
		return current, nil
	}
	legacy := protocol.ManagementTopologyV1{Version: 1, Epoch: epoch, Nodes: make([]protocol.TopologyNodeV1, len(current.nodes))}
	for index, item := range current.nodes {
		legacy.Nodes[index] = protocol.TopologyNodeV1{NodeID: item.NodeID, ParentID: item.ParentID, DisplayName: item.DisplayName, Role: item.Role, Generation: item.Generation}
	}
	// Relations were validated in one traversal above. Avoid the legacy
	// validator's repeated ancestor walks when publishing an already valid tree.
	current.legacy, current.legacyErr = json.Marshal(legacy)
	if len(current.legacy) > protocol.DefaultMaxPayload {
		current.legacy, current.legacyErr = nil, topologyOverflow(len(current.nodes))
	}
	return current, nil
}

func topologyOverflow(count int) error {
	return fmt.Errorf("%w: topology query exceeds %d nodes or %d bytes (nodes: %d); request a smaller depth", protocol.ErrPayloadTooLarge, protocol.MaxItems, protocol.DefaultMaxPayload, count)
}

func (s *topologySnapshot) encode(depth int) topologyResult {
	count := len(s.nodes)
	if depth > 0 {
		count = sort.Search(len(s.levels), func(i int) bool { return s.levels[i] > depth })
	}
	if count > protocol.MaxItems {
		return topologyResult{err: topologyOverflow(count)}
	}
	value := protocol.ManagementTopologyQueryV1{Version: 1, RootNodeID: s.nodes[0].NodeID, Depth: depth, InstanceID: s.instanceID, Revision: s.revision, Nodes: s.nodes[:count]}
	payload, err := protocol.EncodeJSONPayload(&value, protocol.DefaultMaxPayload)
	return topologyResult{payload: payload, err: err}
}

func (s *topologySnapshot) query(depth int) topologyResult {
	if depth == 1 {
		return s.shallow
	}
	if depth == 0 {
		return s.full
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if result, ok := s.cache[depth]; ok {
		return result
	}
	result := s.encode(depth)
	for len(s.order) >= topologyCacheEntries || s.cacheBytes+len(result.payload) > topologyCacheBytes {
		oldest := s.order[0]
		s.order = s.order[1:]
		s.cacheBytes -= len(s.cache[oldest].payload)
		delete(s.cache, oldest)
	}
	s.cache[depth] = result
	s.order = append(s.order, depth)
	s.cacheBytes += len(result.payload)
	return result
}
