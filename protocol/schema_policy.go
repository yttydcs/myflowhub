package protocol

import (
	"errors"
	"fmt"
	"sort"
)

const (
	SchemaPolicyDefinitionV1       = "mfh.policy.definition.v1"
	SchemaPolicyDefinitionPutV1    = "mfh.policy.definition-put.v1"
	SchemaPolicyDefinitionDeleteV1 = "mfh.policy.definition-delete.v1"
	SchemaPolicyBindingV1          = "mfh.policy.binding.v1"
	SchemaPolicyBindingCreateV1    = "mfh.policy.binding-create.v1"
	SchemaPolicyBindingRevokeV1    = "mfh.policy.binding-revoke.v1"
	SchemaPolicyGrantV1            = "mfh.policy.grant.v1"
	SchemaPolicyEvaluateRequestV1  = "mfh.policy.evaluate-request.v1"
	SchemaPolicyEvaluationV1       = "mfh.policy.evaluation.v1"

	BuiltinPolicyDefinitions = "system/policy/definitions"
	BuiltinPolicyBindings    = "system/policy/bindings"
	BuiltinPolicyGrants      = "system/policy/grants"

	BuiltinPolicySuperadmin = "superadmin"

	PolicySelectorExact  = "exact"
	PolicySelectorPrefix = "prefix"
	PolicySelectorAll    = "all"

	PolicyScopeOwner           = "owner"
	PolicyScopeSubtree         = "subtree"
	PolicyScopeAuthorityDomain = "authority-domain"

	PolicyEvaluationNone       = "none"
	PolicyEvaluationExactGrant = "exact-grant"
	PolicyEvaluationBinding    = "binding"

	MaxPolicyRulesPerDefinition = 128
)

type PolicyResourceSelectorV1 struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

func (s PolicyResourceSelectorV1) Validate() error {
	switch s.Kind {
	case PolicySelectorAll:
		if s.Value != "" {
			return errors.New("all resource selector must not contain value")
		}
		return nil
	case PolicySelectorExact, PolicySelectorPrefix:
		if err := (ResourceID{Owner: 1, Name: s.Value}).Validate(); err != nil {
			return fmt.Errorf("resource selector value: %w", err)
		}
		return nil
	default:
		return errors.New("resource selector kind must be exact, prefix, or all")
	}
}

type PolicyCapabilitySelectorV1 struct {
	Kind   string         `json:"kind"`
	Values []CapabilityID `json:"values,omitempty"`
}

func (s PolicyCapabilitySelectorV1) Validate() error {
	switch s.Kind {
	case PolicySelectorAll:
		if len(s.Values) != 0 {
			return errors.New("all capability selector must not contain values")
		}
		return nil
	case PolicySelectorExact:
		if len(s.Values) == 0 || len(s.Values) > MaxCapabilities {
			return fmt.Errorf("exact capability selector requires 1..%d values", MaxCapabilities)
		}
		for index, value := range s.Values {
			if err := value.Validate(); err != nil {
				return fmt.Errorf("capability selector values[%d]: %w", index, err)
			}
			if index > 0 && s.Values[index-1] >= value {
				return errors.New("capability selector values must be strictly sorted and unique")
			}
		}
		return nil
	default:
		return errors.New("capability selector kind must be exact or all")
	}
}

type PolicyRuleV1 struct {
	Resource   PolicyResourceSelectorV1   `json:"resource"`
	Capability PolicyCapabilitySelectorV1 `json:"capability"`
}

func (r PolicyRuleV1) Validate() error {
	if err := r.Resource.Validate(); err != nil {
		return err
	}
	return r.Capability.Validate()
}

type PolicyDefinitionV1 struct {
	Version   int            `json:"version"`
	ID        string         `json:"id"`
	Label     string         `json:"label"`
	Revision  uint64         `json:"revision"`
	Immutable bool           `json:"immutable,omitempty"`
	Rules     []PolicyRuleV1 `json:"rules"`
}

func (d PolicyDefinitionV1) Validate() error {
	if err := validateVersion(d.Version); err != nil {
		return err
	}
	if err := validateExtensibleIdentifier("policy definition id", d.ID); err != nil {
		return err
	}
	if err := validateText("policy definition label", d.Label, MaxLabelBytes, true); err != nil {
		return err
	}
	if d.Revision == 0 || d.Revision > MaxCollectionRevision {
		return fmt.Errorf("policy definition revision must be between 1 and %d", MaxCollectionRevision)
	}
	return validatePolicyRules(d.Rules)
}

type PolicyDefinitionPutV1 struct {
	Version          int            `json:"version"`
	ID               string         `json:"id"`
	Label            string         `json:"label"`
	ExpectedRevision uint64         `json:"expected_revision,omitempty"`
	Rules            []PolicyRuleV1 `json:"rules"`
}

func (d PolicyDefinitionPutV1) Validate() error {
	if err := validateVersion(d.Version); err != nil {
		return err
	}
	if err := validateExtensibleIdentifier("policy definition id", d.ID); err != nil {
		return err
	}
	if err := validateText("policy definition label", d.Label, MaxLabelBytes, true); err != nil {
		return err
	}
	if d.ExpectedRevision > MaxCollectionRevision {
		return fmt.Errorf("expected_revision exceeds %d", MaxCollectionRevision)
	}
	return validatePolicyRules(d.Rules)
}

type PolicyDefinitionDeleteV1 struct {
	Version          int    `json:"version"`
	ID               string `json:"id"`
	ExpectedRevision uint64 `json:"expected_revision"`
}

func (d PolicyDefinitionDeleteV1) Validate() error {
	if err := validateVersion(d.Version); err != nil {
		return err
	}
	if err := validateExtensibleIdentifier("policy definition id", d.ID); err != nil {
		return err
	}
	if d.ExpectedRevision == 0 || d.ExpectedRevision > MaxCollectionRevision {
		return fmt.Errorf("expected_revision must be between 1 and %d", MaxCollectionRevision)
	}
	return nil
}

func validatePolicyRules(rules []PolicyRuleV1) error {
	if len(rules) == 0 || len(rules) > MaxPolicyRulesPerDefinition {
		return fmt.Errorf("policy definition requires 1..%d rules", MaxPolicyRulesPerDefinition)
	}
	seen := make(map[string]struct{}, len(rules))
	for index, rule := range rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("policy rules[%d]: %w", index, err)
		}
		key := fmt.Sprintf("%s\x00%s\x00%s\x00%v", rule.Resource.Kind, rule.Resource.Value, rule.Capability.Kind, rule.Capability.Values)
		if _, exists := seen[key]; exists {
			return errors.New("policy rules must be unique")
		}
		seen[key] = struct{}{}
	}
	return nil
}

type PolicyOwnerScopeV1 struct {
	Kind   string `json:"kind"`
	NodeID string `json:"node_id"`
}

func (s PolicyOwnerScopeV1) Validate() error {
	if s.Kind != PolicyScopeOwner && s.Kind != PolicyScopeSubtree && s.Kind != PolicyScopeAuthorityDomain {
		return errors.New("policy owner scope kind must be owner, subtree, or authority-domain")
	}
	return validateNodeIDText("policy owner scope node_id", s.NodeID)
}

type PolicyBindingV1 struct {
	Version         int                `json:"version"`
	BindingID       string             `json:"binding_id"`
	Subject         string             `json:"subject"`
	DefinitionID    string             `json:"definition_id"`
	Scope           PolicyOwnerScopeV1 `json:"scope"`
	CreatedBy       string             `json:"created_by"`
	CreatedAtUnixMS int64              `json:"created_at_unix_ms"`
	ExpiresAtUnixMS int64              `json:"expires_at_unix_ms,omitempty"`
}

func (b PolicyBindingV1) Validate() error {
	if err := validateVersion(b.Version); err != nil {
		return err
	}
	if err := validateHexID("binding_id", b.BindingID, 16); err != nil {
		return err
	}
	if err := validateNodeIDText("subject", b.Subject); err != nil {
		return err
	}
	if err := validateExtensibleIdentifier("definition_id", b.DefinitionID); err != nil {
		return err
	}
	if err := b.Scope.Validate(); err != nil {
		return err
	}
	if err := validateNodeIDText("created_by", b.CreatedBy); err != nil {
		return err
	}
	if b.CreatedAtUnixMS <= 0 {
		return errors.New("created_at_unix_ms must be positive")
	}
	if b.ExpiresAtUnixMS < 0 || b.ExpiresAtUnixMS != 0 && b.ExpiresAtUnixMS <= b.CreatedAtUnixMS {
		return errors.New("expires_at_unix_ms must be zero or later than creation")
	}
	return nil
}

type PolicyBindingCreateV1 struct {
	Version         int                `json:"version"`
	BindingID       string             `json:"binding_id"`
	Subject         string             `json:"subject"`
	DefinitionID    string             `json:"definition_id"`
	Scope           PolicyOwnerScopeV1 `json:"scope"`
	ExpiresAtUnixMS int64              `json:"expires_at_unix_ms,omitempty"`
}

func (b PolicyBindingCreateV1) Validate() error {
	if err := validateVersion(b.Version); err != nil {
		return err
	}
	if err := validateHexID("binding_id", b.BindingID, 16); err != nil {
		return err
	}
	if err := validateNodeIDText("subject", b.Subject); err != nil {
		return err
	}
	if err := validateExtensibleIdentifier("definition_id", b.DefinitionID); err != nil {
		return err
	}
	if err := b.Scope.Validate(); err != nil {
		return err
	}
	if b.ExpiresAtUnixMS < 0 {
		return errors.New("expires_at_unix_ms must not be negative")
	}
	return nil
}

type PolicyBindingRevokeV1 struct {
	Version   int    `json:"version"`
	BindingID string `json:"binding_id"`
}

func (b PolicyBindingRevokeV1) Validate() error {
	if err := validateVersion(b.Version); err != nil {
		return err
	}
	return validateHexID("binding_id", b.BindingID, 16)
}

type PolicyGrantV1 struct {
	Version      int    `json:"version"`
	Subject      string `json:"subject"`
	Capability   string `json:"capability"`
	ResourceNode string `json:"resource_node"`
	ResourceName string `json:"resource_name"`
}

func (g PolicyGrantV1) Validate() error {
	if err := validateVersion(g.Version); err != nil {
		return err
	}
	if err := validateNodeIDText("subject", g.Subject); err != nil {
		return err
	}
	if err := CapabilityID(g.Capability).Validate(); err != nil {
		return err
	}
	if err := validateNodeIDText("resource_node", g.ResourceNode); err != nil {
		return err
	}
	return (ResourceID{Owner: 1, Name: g.ResourceName}).Validate()
}

type PolicyEvaluateRequestV1 struct {
	Version      int    `json:"version"`
	Subject      string `json:"subject"`
	Capability   string `json:"capability"`
	ResourceNode string `json:"resource_node"`
	ResourceName string `json:"resource_name"`
}

func (r PolicyEvaluateRequestV1) Validate() error {
	return PolicyGrantV1{Version: r.Version, Subject: r.Subject, Capability: r.Capability, ResourceNode: r.ResourceNode, ResourceName: r.ResourceName}.Validate()
}

type PolicyEvaluationV1 struct {
	Version            int                 `json:"version"`
	Allowed            bool                `json:"allowed"`
	Source             string              `json:"source"`
	PolicyGeneration   uint64              `json:"policy_generation"`
	TopologyEpoch      uint64              `json:"topology_epoch,omitempty"`
	BindingID          string              `json:"binding_id,omitempty"`
	DefinitionID       string              `json:"definition_id,omitempty"`
	DefinitionRevision uint64              `json:"definition_revision,omitempty"`
	RuleIndex          int                 `json:"rule_index"`
	Scope              *PolicyOwnerScopeV1 `json:"scope,omitempty"`
}

func (e PolicyEvaluationV1) Validate() error {
	if err := validateVersion(e.Version); err != nil {
		return err
	}
	if e.PolicyGeneration == 0 {
		return errors.New("policy_generation must be non-zero")
	}
	if e.Source != PolicyEvaluationNone && e.Source != PolicyEvaluationExactGrant && e.Source != PolicyEvaluationBinding {
		return errors.New("policy evaluation source is invalid")
	}
	if e.Allowed && e.Source == PolicyEvaluationNone {
		return errors.New("allowed evaluation must identify an authorization source")
	}
	if !e.Allowed && e.Source != PolicyEvaluationNone {
		return errors.New("denied evaluation must use none source")
	}
	if e.Source == PolicyEvaluationBinding {
		if err := validateHexID("binding_id", e.BindingID, 16); err != nil {
			return err
		}
		if err := validateExtensibleIdentifier("definition_id", e.DefinitionID); err != nil {
			return err
		}
		if e.DefinitionRevision == 0 || e.TopologyEpoch == 0 || e.RuleIndex < 0 {
			return errors.New("binding evaluation metadata is incomplete")
		}
		if e.Scope == nil {
			return errors.New("binding evaluation scope is required")
		}
		return e.Scope.Validate()
	}
	if e.TopologyEpoch != 0 || e.BindingID != "" || e.DefinitionID != "" || e.DefinitionRevision != 0 || e.RuleIndex != 0 || e.Scope != nil {
		return errors.New("non-binding evaluation must not contain binding metadata")
	}
	return nil
}

func SortPolicyRules(rules []PolicyRuleV1) {
	for index := range rules {
		sort.Slice(rules[index].Capability.Values, func(i, j int) bool { return rules[index].Capability.Values[i] < rules[index].Capability.Values[j] })
	}
	sort.Slice(rules, func(i, j int) bool {
		left, right := rules[i], rules[j]
		if left.Resource.Kind != right.Resource.Kind {
			return left.Resource.Kind < right.Resource.Kind
		}
		if left.Resource.Value != right.Resource.Value {
			return left.Resource.Value < right.Resource.Value
		}
		if left.Capability.Kind != right.Capability.Kind {
			return left.Capability.Kind < right.Capability.Kind
		}
		return fmt.Sprint(left.Capability.Values) < fmt.Sprint(right.Capability.Values)
	})
}
