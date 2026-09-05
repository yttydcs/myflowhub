package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	SchemaManagementTopologyChildrenRequestV1 = "mfh.management.topology-children-request.v1"
	SchemaManagementTopologyQueryRequestV1    = "mfh.management.topology-query-request.v1"
	SchemaManagementTopologyQueryV1           = "mfh.management.topology-query.v1"

	CapabilityChildren CapabilityID = "children"
	CapabilitySubtree  CapabilityID = "subtree"

	// MaxTopologyRevision survives exact JSON round trips through JavaScript.
	MaxTopologyRevision uint64 = 1<<53 - 1
)

type ManagementTopologyChildrenRequestV1 struct {
	Version int `json:"version"`
	Depth   int `json:"depth"`
}

func (r ManagementTopologyChildrenRequestV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if r.Depth != 1 {
		return errors.New("topology children depth must be 1")
	}
	return nil
}

func (r *ManagementTopologyChildrenRequestV1) UnmarshalJSON(data []byte) error {
	var value ManagementTopologyChildrenRequestV1
	if err := decodeTopologyObject(data, map[string]topologyJSONField{
		"version": {&value.Version, true},
		"depth":   {&value.Depth, true},
	}); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	*r = value
	return nil
}

// Depth zero selects all descendants, while positive depths bound the number
// of edges from the resource owner. Depth is deliberately not omitempty.
type ManagementTopologyQueryRequestV1 struct {
	Version int `json:"version"`
	Depth   int `json:"depth"`
}

func (r ManagementTopologyQueryRequestV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	return validateTopologyDepth(r.Depth)
}

func (r *ManagementTopologyQueryRequestV1) UnmarshalJSON(data []byte) error {
	var value ManagementTopologyQueryRequestV1
	if err := decodeTopologyObject(data, map[string]topologyJSONField{
		"version": {&value.Version, true},
		"depth":   {&value.Depth, true},
	}); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	*r = value
	return nil
}

type TopologyQueryNodeV1 struct {
	NodeID      string `json:"node_id"`
	ParentID    string `json:"parent_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Role        string `json:"role"`
	Generation  uint64 `json:"generation"`
	HasChildren bool   `json:"has_children"`
}

func (n TopologyQueryNodeV1) Validate(root bool) error {
	return (TopologyNodeV1{
		NodeID: n.NodeID, ParentID: n.ParentID, DisplayName: n.DisplayName,
		Role: n.Role, Generation: n.Generation,
	}).Validate(root)
}

func (n *TopologyQueryNodeV1) UnmarshalJSON(data []byte) error {
	var value TopologyQueryNodeV1
	var parent *string
	if err := decodeTopologyObject(data, map[string]topologyJSONField{
		"node_id":      {&value.NodeID, true},
		"parent_id":    {&parent, false},
		"display_name": {&value.DisplayName, false},
		"role":         {&value.Role, true},
		"generation":   {&value.Generation, true},
		"has_children": {&value.HasChildren, true},
	}); err != nil {
		return err
	}
	if parent != nil {
		if err := validateNodeIDText("parent_id", *parent); err != nil {
			return err
		}
		value.ParentID = *parent
	}
	if err := value.Validate(parent == nil); err != nil {
		return err
	}
	*n = value
	return nil
}

type ManagementTopologyQueryV1 struct {
	Version    int                   `json:"version"`
	RootNodeID string                `json:"root_node_id"`
	Depth      int                   `json:"depth"`
	InstanceID string                `json:"instance_id"`
	Revision   uint64                `json:"revision"`
	Nodes      []TopologyQueryNodeV1 `json:"nodes"`
}

func (t *ManagementTopologyQueryV1) UnmarshalJSON(data []byte) error {
	var value ManagementTopologyQueryV1
	if err := decodeTopologyObject(data, map[string]topologyJSONField{
		"version":      {&value.Version, true},
		"root_node_id": {&value.RootNodeID, true},
		"depth":        {&value.Depth, true},
		"instance_id":  {&value.InstanceID, true},
		"revision":     {&value.Revision, true},
		"nodes":        {&value.Nodes, true},
	}); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	*t = value
	return nil
}

func (t ManagementTopologyQueryV1) Validate() error {
	if err := validateVersion(t.Version); err != nil {
		return err
	}
	if err := validateNodeIDText("root_node_id", t.RootNodeID); err != nil {
		return err
	}
	if err := validateTopologyDepth(t.Depth); err != nil {
		return err
	}
	if err := validateHexID("instance_id", t.InstanceID, 16); err != nil {
		return err
	}
	if t.Revision == 0 || t.Revision > MaxTopologyRevision {
		return fmt.Errorf("topology revision must be between 1 and %d", MaxTopologyRevision)
	}
	if len(t.Nodes) == 0 || len(t.Nodes) > MaxItems {
		return fmt.Errorf("topology query requires 1..%d nodes", MaxItems)
	}
	byID := make(map[string]int, len(t.Nodes))
	for index, node := range t.Nodes {
		if err := node.Validate(node.NodeID == t.RootNodeID); err != nil {
			return fmt.Errorf("nodes[%d]: %w", index, err)
		}
		if _, exists := byID[node.NodeID]; exists {
			return fmt.Errorf("duplicate topology node %s", node.NodeID)
		}
		byID[node.NodeID] = index
	}
	root, exists := byID[t.RootNodeID]
	if !exists {
		return errors.New("topology query root_node_id is absent from nodes")
	}
	children := make([][]int, len(t.Nodes))
	for index, node := range t.Nodes {
		if index == root {
			continue
		}
		parent, exists := byID[node.ParentID]
		if !exists {
			return fmt.Errorf("node %s has unknown parent %s", node.NodeID, node.ParentID)
		}
		children[parent] = append(children[parent], index)
	}
	// Each non-root node has exactly one parent. A root traversal therefore
	// cannot revisit a node; any unreached component must contain a cycle.
	// This uses O(nodes) time and memory, including for a MaxItems-long chain.
	depths := make([]int, len(t.Nodes))
	queue := make([]int, 1, len(t.Nodes))
	queue[0] = root
	for head := 0; head < len(queue); head++ {
		index := queue[head]
		node, depth := t.Nodes[index], depths[index]
		if t.Depth > 0 && depth > t.Depth {
			return fmt.Errorf("node %s exceeds topology depth %d", node.NodeID, t.Depth)
		}
		hasReturnedChildren := len(children[index]) != 0
		if hasReturnedChildren && !node.HasChildren || (t.Depth == 0 || depth < t.Depth) && node.HasChildren != hasReturnedChildren {
			return fmt.Errorf("node %s has_children is inconsistent with returned children", node.NodeID)
		}
		for _, child := range children[index] {
			depths[child] = depth + 1
			queue = append(queue, child)
		}
	}
	if len(queue) != len(t.Nodes) {
		return errors.New("topology query contains a cycle or nodes unreachable from root_node_id")
	}
	return nil
}

func validateTopologyDepth(depth int) error {
	if depth < 0 || depth > MaxItems {
		return fmt.Errorf("topology depth must be between 0 and %d", MaxItems)
	}
	return nil
}

type topologyJSONField struct {
	target   any
	required bool
}

// Scope stricter wire decoding to the new topology contract. In particular,
// missing/null depth must never become an unbounded query, and missing/null
// has_children must never masquerade as a known leaf. Exact keys and duplicate
// rejection prevent consumers from interpreting the same object differently.
func decodeTopologyObject(data []byte, fields map[string]topologyJSONField) error {
	if err := validatePayloadSize(data, DefaultMaxPayload); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return errors.New("topology JSON must be an object")
	}
	seen := make(map[string]bool, len(fields))
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		name, ok := token.(string)
		if !ok {
			return errors.New("topology JSON field name must be a string")
		}
		field, ok := fields[name]
		if !ok {
			return fmt.Errorf("unknown topology field %q", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate topology field %q", name)
		}
		seen[name] = true
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%s must not be null", name)
		}
		if err := json.Unmarshal(raw, field.target); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("topology JSON contains trailing data")
	}
	for name, field := range fields {
		if field.required && !seen[name] {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}
