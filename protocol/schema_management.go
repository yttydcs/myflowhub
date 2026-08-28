package protocol

import (
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	SchemaManagementTopologyV1     = "mfh.management.topology.v1"
	SchemaManagementHealthV1       = "mfh.management.health.v1"
	SchemaManagementConfigV1       = "mfh.management.config.v1"
	SchemaManagementAdmitV1        = "mfh.management.admit.v1"
	SchemaManagementRevokeV1       = "mfh.management.revoke.v1"
	SchemaManagementIssuePermitV1  = "mfh.management.issue-permit.v1"
	SchemaManagementRevokePermitV1 = "mfh.management.revoke-permit.v1"
	SchemaManagementConfigUpdateV1 = "mfh.management.config-update.v1"
	SchemaManagementPolicyRuleV1   = "mfh.management.policy-rule.v1"
	SchemaManagementAuditV1        = "mfh.management.audit.v1"
	SchemaManagementResultV1       = "mfh.management.result.v1"

	BuiltinManagementTopology     = "system/topology"
	BuiltinManagementHealth       = "system/health"
	BuiltinManagementConfig       = "system/config"
	BuiltinManagementAudit        = "system/audit"
	BuiltinManagementIssuePermit  = "system/admission/issue"
	BuiltinManagementRevokePermit = "system/admission/revoke"
	BuiltinManagementRevokeNode   = "system/node/revoke"
	BuiltinManagementConfigUpdate = "system/config/update"
	BuiltinManagementPolicyGrant  = "system/policy/grant"
	BuiltinManagementPolicyRevoke = "system/policy/revoke"
)

type TopologyNodeV1 struct {
	NodeID      string `json:"node_id"`
	ParentID    string `json:"parent_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Role        string `json:"role"`
	Generation  uint64 `json:"generation"`
}

func (n TopologyNodeV1) Validate(root bool) error {
	if err := validateNodeIDText("node_id", n.NodeID); err != nil {
		return err
	}
	if root {
		if n.ParentID != "" {
			return errors.New("root topology node must not have parent_id")
		}
	} else if err := validateNodeIDText("parent_id", n.ParentID); err != nil {
		return err
	}
	if err := validateText("display_name", n.DisplayName, MaxLabelBytes, false); err != nil {
		return err
	}
	if err := validateText("role", n.Role, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if n.Generation == 0 {
		return errors.New("node generation must be non-zero")
	}
	return nil
}

type ManagementTopologyV1 struct {
	Version int              `json:"version"`
	Epoch   uint64           `json:"epoch"`
	Nodes   []TopologyNodeV1 `json:"nodes"`
}

func (t ManagementTopologyV1) Validate() error {
	if err := validateVersion(t.Version); err != nil {
		return err
	}
	if t.Epoch == 0 || len(t.Nodes) == 0 || len(t.Nodes) > MaxItems {
		return fmt.Errorf("topology requires a non-zero epoch and 1..%d nodes", MaxItems)
	}
	seen := make(map[string]struct{}, len(t.Nodes))
	parents := make(map[string]string, len(t.Nodes))
	roots := 0
	for index, node := range t.Nodes {
		if err := node.Validate(node.ParentID == ""); err != nil {
			return fmt.Errorf("nodes[%d]: %w", index, err)
		}
		if _, exists := seen[node.NodeID]; exists {
			return fmt.Errorf("duplicate topology node %s", node.NodeID)
		}
		seen[node.NodeID] = struct{}{}
		parents[node.NodeID] = node.ParentID
		if node.ParentID == "" {
			roots++
		}
	}
	if roots != 1 {
		return errors.New("topology must contain exactly one root")
	}
	for _, node := range t.Nodes {
		if node.ParentID != "" {
			if _, exists := seen[node.ParentID]; !exists {
				return fmt.Errorf("node %s has unknown parent %s", node.NodeID, node.ParentID)
			}
		}
		visited := make(map[string]struct{})
		current := node.NodeID
		for current != "" {
			if _, exists := visited[current]; exists {
				return fmt.Errorf("topology contains a cycle at node %s", current)
			}
			visited[current] = struct{}{}
			current = parents[current]
		}
	}
	return nil
}

type ManagementHealthV1 struct {
	Version             int               `json:"version"`
	State               string            `json:"state"`
	StartedAtUnixMS     int64             `json:"started_at_unix_ms"`
	TopologyEpoch       uint64            `json:"topology_epoch"`
	ActiveLinks         int               `json:"active_links"`
	ActiveSubscriptions int               `json:"active_subscriptions"`
	LastError           string            `json:"last_error,omitempty"`
	Components          map[string]string `json:"components,omitempty"`
}

func (h ManagementHealthV1) Validate() error {
	if err := validateVersion(h.Version); err != nil {
		return err
	}
	if h.State != "starting" && h.State != "running" && h.State != "degraded" && h.State != "stopping" {
		return errors.New("health state is invalid")
	}
	if h.StartedAtUnixMS <= 0 || h.TopologyEpoch == 0 || h.ActiveLinks < 0 || h.ActiveSubscriptions < 0 {
		return errors.New("health counters or start time are invalid")
	}
	if err := validateText("last_error", h.LastError, 2048, false); err != nil {
		return err
	}
	if len(h.Components) > MaxAttributes {
		return errors.New("too many health components")
	}
	for name, state := range h.Components {
		if err := validateText("component name", name, MaxIdentifierBytes, true); err != nil {
			return err
		}
		if err := validateText("component state", state, MaxLabelBytes, true); err != nil {
			return err
		}
	}
	return nil
}

type ManagementConfigV1 struct {
	Version     int               `json:"version"`
	Revision    uint64            `json:"revision"`
	DisplayName string            `json:"display_name,omitempty"`
	Values      map[string]string `json:"values,omitempty"`
}

func (c ManagementConfigV1) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}
	if c.Revision == 0 {
		return errors.New("config revision must be non-zero")
	}
	if err := validateText("display_name", c.DisplayName, MaxLabelBytes, false); err != nil {
		return err
	}
	return validateAttributes(c.Values)
}

type ManagementAdmitV1 struct {
	Version int    `json:"version"`
	Permit  string `json:"permit"`
}

type ManagementIssuePermitV1 struct {
	Version        int    `json:"version"`
	ChildNodeID    string `json:"child_node_id"`
	ChildPublicKey string `json:"child_public_key"`
	Role           string `json:"role"`
	TTLMS          int64  `json:"ttl_ms"`
}

func (p ManagementIssuePermitV1) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	if err := validateNodeIDText("child_node_id", p.ChildNodeID); err != nil {
		return err
	}
	key, err := base64.RawStdEncoding.DecodeString(p.ChildPublicKey)
	if err != nil || len(key) != 32 {
		return errors.New("child_public_key must be a raw-base64 Ed25519 public key")
	}
	if err := validateText("role", p.Role, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if p.TTLMS <= 0 {
		return errors.New("ttl_ms must be positive")
	}
	return nil
}

type ManagementRevokePermitV1 struct {
	Version  int    `json:"version"`
	PermitID string `json:"permit_id"`
}

func (p ManagementRevokePermitV1) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	return validateHexID("permit_id", p.PermitID, 16)
}

type ManagementConfigUpdateV1 struct {
	Version          int               `json:"version"`
	ExpectedRevision uint64            `json:"expected_revision"`
	DisplayName      string            `json:"display_name,omitempty"`
	Values           map[string]string `json:"values,omitempty"`
}

func (c ManagementConfigUpdateV1) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}
	if c.ExpectedRevision == 0 {
		return errors.New("expected_revision must be non-zero")
	}
	if err := validateText("display_name", c.DisplayName, MaxLabelBytes, false); err != nil {
		return err
	}
	return validateAttributes(c.Values)
}

func (a ManagementAdmitV1) Validate() error {
	if err := validateVersion(a.Version); err != nil {
		return err
	}
	return validateText("permit", a.Permit, DefaultMaxPayload/2, true)
}

type ManagementRevokeV1 struct {
	Version int    `json:"version"`
	NodeID  string `json:"node_id"`
	Reason  string `json:"reason"`
}

type ManagementPolicyRuleV1 struct {
	Version      int    `json:"version"`
	Subject      string `json:"subject"`
	Action       string `json:"action"`
	ResourceNode string `json:"resource_node"`
	ResourceName string `json:"resource_name"`
}

func (r ManagementPolicyRuleV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if err := validateNodeIDText("subject", r.Subject); err != nil {
		return err
	}
	if err := CapabilityID(r.Action).Validate(); err != nil {
		return fmt.Errorf("policy action: %w", err)
	}
	if err := validateNodeIDText("resource_node", r.ResourceNode); err != nil {
		return err
	}
	return (ResourceID{Owner: 1, Name: r.ResourceName}).Validate()
}

func (r ManagementRevokeV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if err := validateNodeIDText("node_id", r.NodeID); err != nil {
		return err
	}
	return validateText("reason", r.Reason, MaxLabelBytes, true)
}

type ManagementAuditV1 struct {
	Version      int    `json:"version"`
	TimeUnixMS   int64  `json:"time_unix_ms"`
	Subject      string `json:"subject"`
	Action       string `json:"action"`
	ResourceNode string `json:"resource_node"`
	ResourceName string `json:"resource_name"`
	Decision     string `json:"decision"`
	Detail       string `json:"detail,omitempty"`
}

func (a ManagementAuditV1) Validate() error {
	if err := validateVersion(a.Version); err != nil {
		return err
	}
	if a.TimeUnixMS <= 0 {
		return errors.New("audit timestamp must be positive")
	}
	if err := validateNodeIDText("subject", a.Subject); err != nil {
		return err
	}
	if err := CapabilityID(a.Action).Validate(); err != nil {
		return fmt.Errorf("audit action: %w", err)
	}
	if err := validateNodeIDText("resource_node", a.ResourceNode); err != nil {
		return err
	}
	if err := (ResourceID{Owner: 1, Name: a.ResourceName}).Validate(); err != nil {
		return err
	}
	if a.Decision != "allow" && a.Decision != "deny" {
		return errors.New("audit decision is invalid")
	}
	return validateText("detail", a.Detail, 2048, false)
}

type ManagementResultV1 struct {
	Version int    `json:"version"`
	Status  string `json:"status"`
}

func (r ManagementResultV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if r.Status != "ok" {
		return errors.New("management result status is invalid")
	}
	return nil
}
