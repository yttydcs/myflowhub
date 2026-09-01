package protocol

import (
	"strings"
	"testing"
)

func TestPolicySelectorAndScopeValidation(t *testing.T) {
	valid := PolicyDefinitionPutV1{
		Version: 1, ID: "metrics-reader", Label: "Metrics reader",
		Rules: []PolicyRuleV1{{
			Resource:   PolicyResourceSelectorV1{Kind: PolicySelectorPrefix, Value: "metrics"},
			Capability: PolicyCapabilitySelectorV1{Kind: PolicySelectorExact, Values: []CapabilityID{CapabilityRead, CapabilitySubscribe}},
		}},
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.Rules = []PolicyRuleV1{{
		Resource:   PolicyResourceSelectorV1{Kind: PolicySelectorPrefix, Value: "metrics/"},
		Capability: PolicyCapabilitySelectorV1{Kind: PolicySelectorExact, Values: []CapabilityID{CapabilitySubscribe, CapabilityRead}},
	}}
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid prefix and unsorted capabilities were accepted")
	}
	if err := (PolicyOwnerScopeV1{Kind: "network", NodeID: "1"}).Validate(); err == nil {
		t.Fatal("unknown owner scope was accepted")
	}
}

func TestPolicyDefinitionBoundsAndEvaluationMetadata(t *testing.T) {
	rules := make([]PolicyRuleV1, MaxPolicyRulesPerDefinition+1)
	for index := range rules {
		rules[index] = PolicyRuleV1{
			Resource:   PolicyResourceSelectorV1{Kind: PolicySelectorExact, Value: "metrics/" + strings.Repeat("a", index+1)},
			Capability: PolicyCapabilitySelectorV1{Kind: PolicySelectorAll},
		}
	}
	if err := (PolicyDefinitionPutV1{Version: 1, ID: "too-large", Label: "Too large", Rules: rules}).Validate(); err == nil {
		t.Fatal("oversized policy rule set was accepted")
	}
	if err := (PolicyEvaluationV1{Version: 1, Allowed: true, Source: PolicyEvaluationNone, PolicyGeneration: 1}).Validate(); err == nil {
		t.Fatal("allowed evaluation without a source was accepted")
	}
	scope := PolicyOwnerScopeV1{Kind: PolicyScopeAuthorityDomain, NodeID: "1"}
	valid := PolicyEvaluationV1{
		Version: 1, Allowed: true, Source: PolicyEvaluationBinding, PolicyGeneration: 2, TopologyEpoch: 3,
		BindingID: strings.Repeat("a", 32), DefinitionID: BuiltinPolicySuperadmin, DefinitionRevision: 1, RuleIndex: 0, Scope: &scope,
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
}
