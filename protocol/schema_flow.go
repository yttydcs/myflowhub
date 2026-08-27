package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	SchemaFlowDefinitionV1  = "mfh.flow.definition.v1"
	SchemaFlowRunV1         = "mfh.flow.run.v1"
	SchemaFlowRunSummaryV1  = "mfh.flow.run-summary.v1"
	SchemaFlowCancelV1      = "mfh.flow.cancel.v1"
	SchemaFlowEventV1       = "mfh.flow.event.v1"
	SchemaFlowDefinitionsV1 = "mfh.flow.definitions.v1"
	SchemaFlowRunsV1        = "mfh.flow.runs.v1"
	SchemaFlowArchiveV1     = "mfh.flow.archive.v1"
	MaxFlowNodes            = 256
	MaxFlowEdges            = 1024
	MaxFlowNodeConfigBytes  = 16 << 10
	BuiltinFlowDefinitions  = "flow/definitions"
	BuiltinFlowRuns         = "flow/runs"
	BuiltinFlowEvents       = "flow/events"
	BuiltinFlowCreate       = "flow/create"
	BuiltinFlowUpdate       = "flow/update"
	BuiltinFlowRun          = "flow/run"
	BuiltinFlowCancel       = "flow/cancel"
	BuiltinFlowArchive      = "flow/archive"
)

type FlowResourceRefV1 struct {
	OwnerNodeID string `json:"owner_node_id"`
	Name        string `json:"name"`
}

func (r FlowResourceRefV1) Validate() error {
	if err := validateNodeIDText("owner_node_id", r.OwnerNodeID); err != nil {
		return err
	}
	return (ResourceID{Owner: 1, Name: r.Name}).Validate()
}

type FlowNodeV1 struct {
	ID             string            `json:"id"`
	Kind           string            `json:"kind"`
	Resource       FlowResourceRefV1 `json:"resource"`
	Config         json.RawMessage   `json:"config,omitempty"`
	MaxAttempts    int               `json:"max_attempts,omitempty"`
	RetryBackoffMS int64             `json:"retry_backoff_ms,omitempty"`
}

func (n FlowNodeV1) Validate() error {
	if err := validateText("flow node id", n.ID, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if n.Kind != "variable-read" && n.Kind != "command-call" && n.Kind != "transform" {
		return errors.New("flow node kind is invalid")
	}
	if n.Kind != "transform" {
		if err := n.Resource.Validate(); err != nil {
			return err
		}
	} else if n.Resource != (FlowResourceRefV1{}) {
		return errors.New("transform node must not reference a resource")
	}
	if len(n.Config) > MaxFlowNodeConfigBytes {
		return fmt.Errorf("flow node config exceeds %d bytes", MaxFlowNodeConfigBytes)
	}
	if len(n.Config) > 0 && !json.Valid(n.Config) {
		return errors.New("flow node config is not valid JSON")
	}
	if n.MaxAttempts < 0 || n.MaxAttempts > 10 {
		return errors.New("flow node max_attempts must be between 0 and 10")
	}
	if n.MaxAttempts <= 1 && n.RetryBackoffMS != 0 {
		return errors.New("flow node retry_backoff_ms requires max_attempts greater than 1")
	}
	if n.MaxAttempts > 1 && (n.RetryBackoffMS < 10 || n.RetryBackoffMS > 60_000) {
		return errors.New("flow node retry_backoff_ms must be between 10 and 60000")
	}
	return nil
}

type FlowEdgeV1 struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type FlowDefinitionV1 struct {
	Version  int               `json:"version"`
	FlowID   string            `json:"flow_id"`
	Revision uint64            `json:"revision"`
	Name     string            `json:"name"`
	Nodes    []FlowNodeV1      `json:"nodes"`
	Edges    []FlowEdgeV1      `json:"edges"`
	Inputs   map[string]string `json:"inputs,omitempty"`
}

func (d FlowDefinitionV1) Validate() error {
	if err := validateVersion(d.Version); err != nil {
		return err
	}
	if err := validateHexID("flow_id", d.FlowID, 16); err != nil {
		return err
	}
	if d.Revision == 0 {
		return errors.New("flow revision must be non-zero")
	}
	if err := validateText("flow name", d.Name, MaxLabelBytes, true); err != nil {
		return err
	}
	if len(d.Nodes) == 0 || len(d.Nodes) > MaxFlowNodes || len(d.Edges) > MaxFlowEdges {
		return fmt.Errorf("flow must contain 1..%d nodes and at most %d edges", MaxFlowNodes, MaxFlowEdges)
	}
	if err := validateAttributes(d.Inputs); err != nil {
		return err
	}
	indegree := make(map[string]int, len(d.Nodes))
	graph := make(map[string][]string, len(d.Nodes))
	for index, node := range d.Nodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("nodes[%d]: %w", index, err)
		}
		if _, exists := indegree[node.ID]; exists {
			return fmt.Errorf("duplicate flow node %q", node.ID)
		}
		indegree[node.ID] = 0
	}
	edges := make(map[string]struct{}, len(d.Edges))
	for _, edge := range d.Edges {
		if _, exists := indegree[edge.From]; !exists {
			return fmt.Errorf("edge has unknown source %q", edge.From)
		}
		if _, exists := indegree[edge.To]; !exists {
			return fmt.Errorf("edge has unknown target %q", edge.To)
		}
		key := edge.From + "\x00" + edge.To
		if _, exists := edges[key]; exists {
			return fmt.Errorf("duplicate edge %q -> %q", edge.From, edge.To)
		}
		edges[key] = struct{}{}
		graph[edge.From] = append(graph[edge.From], edge.To)
		indegree[edge.To]++
	}
	queue := make([]string, 0, len(indegree))
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, next := range graph[id] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if visited != len(d.Nodes) {
		return errors.New("flow graph contains a cycle")
	}
	return nil
}

type FlowRunV1 struct {
	Version        int               `json:"version"`
	RunID          string            `json:"run_id"`
	FlowID         string            `json:"flow_id"`
	FlowRevision   uint64            `json:"flow_revision"`
	DedupeKey      string            `json:"dedupe_key"`
	DeadlineUnixMS int64             `json:"deadline_unix_ms"`
	Inputs         map[string]string `json:"inputs,omitempty"`
}

func (r FlowRunV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if err := validateHexID("run_id", r.RunID, 16); err != nil {
		return err
	}
	if err := validateHexID("flow_id", r.FlowID, 16); err != nil {
		return err
	}
	if r.FlowRevision == 0 || r.DeadlineUnixMS <= 0 {
		return errors.New("flow revision and deadline must be positive")
	}
	if err := validateText("dedupe_key", r.DedupeKey, MaxIdentifierBytes, true); err != nil {
		return err
	}
	return validateAttributes(r.Inputs)
}

type FlowCancelV1 struct {
	Version int    `json:"version"`
	RunID   string `json:"run_id"`
	Reason  string `json:"reason"`
}

func (c FlowCancelV1) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}
	if err := validateHexID("run_id", c.RunID, 16); err != nil {
		return err
	}
	return validateText("reason", c.Reason, MaxLabelBytes, true)
}

type FlowEventV1 struct {
	Version          int               `json:"version"`
	RunID            string            `json:"run_id"`
	Sequence         uint64            `json:"sequence"`
	State            string            `json:"state"`
	NodeID           string            `json:"node_id,omitempty"`
	OccurredAtUnixMS int64             `json:"occurred_at_unix_ms"`
	Summary          map[string]string `json:"summary,omitempty"`
}

func (e FlowEventV1) Validate() error {
	if err := validateVersion(e.Version); err != nil {
		return err
	}
	if err := validateHexID("run_id", e.RunID, 16); err != nil {
		return err
	}
	if e.Sequence == 0 || e.OccurredAtUnixMS <= 0 {
		return errors.New("flow event sequence and timestamp must be positive")
	}
	if e.State != "queued" && e.State != "running" && e.State != "succeeded" && e.State != "failed" && e.State != "cancelled" && e.State != "interrupted" {
		return errors.New("flow event state is invalid")
	}
	if err := validateText("node_id", e.NodeID, MaxIdentifierBytes, false); err != nil {
		return err
	}
	return validateAttributes(e.Summary)
}

type FlowDefinitionSummaryV1 struct {
	FlowID    string `json:"flow_id"`
	Revision  uint64 `json:"revision"`
	Name      string `json:"name"`
	NodeCount int    `json:"node_count"`
	EdgeCount int    `json:"edge_count"`
}

func (s FlowDefinitionSummaryV1) Validate() error {
	if err := validateHexID("flow_id", s.FlowID, 16); err != nil {
		return err
	}
	if s.Revision == 0 || s.NodeCount < 1 || s.NodeCount > MaxFlowNodes || s.EdgeCount < 0 || s.EdgeCount > MaxFlowEdges {
		return errors.New("flow definition summary counters are invalid")
	}
	return validateText("flow name", s.Name, MaxLabelBytes, true)
}

type FlowDefinitionsV1 struct {
	Version     int                       `json:"version"`
	Revision    uint64                    `json:"revision"`
	Definitions []FlowDefinitionSummaryV1 `json:"definitions"`
}

func (d FlowDefinitionsV1) Validate() error {
	if err := validateVersion(d.Version); err != nil {
		return err
	}
	if d.Revision == 0 || len(d.Definitions) > MaxItems {
		return errors.New("flow definitions revision or count is invalid")
	}
	for index, definition := range d.Definitions {
		if err := definition.Validate(); err != nil {
			return fmt.Errorf("definitions[%d]: %w", index, err)
		}
		if index > 0 && d.Definitions[index-1].FlowID >= definition.FlowID {
			return errors.New("flow definitions must be strictly sorted by flow_id")
		}
	}
	return nil
}

type FlowRunSummaryV1 struct {
	RunID          string `json:"run_id"`
	FlowID         string `json:"flow_id"`
	FlowRevision   uint64 `json:"flow_revision"`
	Initiator      string `json:"initiator"`
	DedupeKey      string `json:"dedupe_key"`
	State          string `json:"state"`
	StartedUnixMS  int64  `json:"started_unix_ms"`
	FinishedUnixMS int64  `json:"finished_unix_ms,omitempty"`
	Error          string `json:"error,omitempty"`
}

func (r FlowRunSummaryV1) Validate() error {
	if err := validateHexID("run_id", r.RunID, 16); err != nil {
		return err
	}
	if err := validateHexID("flow_id", r.FlowID, 16); err != nil {
		return err
	}
	if r.FlowRevision == 0 || r.StartedUnixMS <= 0 {
		return errors.New("flow run revision or start time is invalid")
	}
	if err := validateNodeIDText("initiator", r.Initiator); err != nil {
		return err
	}
	if err := validateText("dedupe_key", r.DedupeKey, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if r.State != "queued" && r.State != "running" && r.State != "succeeded" && r.State != "failed" && r.State != "cancelled" && r.State != "interrupted" {
		return errors.New("flow run state is invalid")
	}
	terminal := r.State == "succeeded" || r.State == "failed" || r.State == "cancelled" || r.State == "interrupted"
	if terminal != (r.FinishedUnixMS > 0) {
		return errors.New("flow run terminal state and finish time disagree")
	}
	return validateText("error", r.Error, 2048, false)
}

type FlowRunsV1 struct {
	Version  int                `json:"version"`
	Revision uint64             `json:"revision"`
	Runs     []FlowRunSummaryV1 `json:"runs"`
}

func (r FlowRunsV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if r.Revision == 0 || len(r.Runs) > MaxItems {
		return errors.New("flow runs revision or count is invalid")
	}
	for index, run := range r.Runs {
		if err := run.Validate(); err != nil {
			return fmt.Errorf("runs[%d]: %w", index, err)
		}
		if index > 0 && r.Runs[index-1].RunID >= run.RunID {
			return errors.New("flow runs must be strictly sorted by run_id")
		}
	}
	return nil
}

type FlowArchiveV1 struct {
	Version int    `json:"version"`
	FlowID  string `json:"flow_id"`
}

func (a FlowArchiveV1) Validate() error {
	if err := validateVersion(a.Version); err != nil {
		return err
	}
	return validateHexID("flow_id", a.FlowID, 16)
}
