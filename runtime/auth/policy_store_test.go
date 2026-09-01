package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

type policyScopeResolverFunc func(protocol.PolicyOwnerScopeV1, protocol.NodeID) (bool, uint64, error)

func (f policyScopeResolverFunc) MatchOwnerScope(scope protocol.PolicyOwnerScopeV1, owner protocol.NodeID) (bool, uint64, error) {
	return f(scope, owner)
}

func TestPolicyStateFreshHasUnboundSuperadmin(t *testing.T) {
	state, err := OpenState(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Policy.Close()
	definition, ok := state.Policy.Definition(protocol.BuiltinPolicySuperadmin)
	if !ok || !definition.Immutable || len(definition.Rules) != 1 {
		t.Fatalf("unexpected built-in superadmin definition: %+v", definition)
	}
	if bindings := state.Policy.Bindings(); len(bindings) != 0 {
		t.Fatalf("fresh policy unexpectedly grants bindings: %+v", bindings)
	}
	request := Request{Subject: 41, Capability: protocol.CapabilityRead, Resource: protocol.ResourceID{Owner: 1, Name: "system/health"}}
	if err := state.Policy.Authorize(context.Background(), request); !errors.Is(err, ErrForbidden) {
		t.Fatalf("fresh unbound policy must deny, got %v", err)
	}
}

func TestPolicyStateMigratesExactGrantsWithoutBinding(t *testing.T) {
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	grant := Request{Subject: 41, Capability: protocol.CapabilityRead, Action: ActionRead, Resource: protocol.ResourceID{Owner: 42, Name: "metrics/cpu_percent"}}
	if err := store.Save("policy.json", policyState{Version: legacyPolicyStateVersion, Generation: 7, Grants: []Request{grant}}); err != nil {
		t.Fatal(err)
	}
	policy, err := LoadPolicyState(store)
	if err != nil {
		t.Fatal(err)
	}
	defer policy.Close()
	if policy.Generation() != 8 {
		t.Fatalf("migration generation = %d, want 8", policy.Generation())
	}
	if err := policy.Authorize(context.Background(), grant); err != nil {
		t.Fatalf("migrated exact grant was lost: %v", err)
	}
	if len(policy.Bindings()) != 0 {
		t.Fatal("migration must not create a privileged binding")
	}
}

func TestPolicyStateBindingUsesSegmentSafePrefixAndCurrentScope(t *testing.T) {
	state, err := OpenState(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Policy.Close()
	inside := map[protocol.NodeID]bool{42: true}
	if err := state.Policy.BindScopeResolver(policyScopeResolverFunc(func(scope protocol.PolicyOwnerScopeV1, owner protocol.NodeID) (bool, uint64, error) {
		return scope.NodeID == "1" && inside[owner], 9, nil
	})); err != nil {
		t.Fatal(err)
	}
	definition, err := state.Policy.PutDefinition(protocol.PolicyDefinitionPutV1{
		Version: 1, ID: "metrics-reader", Label: "Metrics reader",
		Rules: []protocol.PolicyRuleV1{{
			Resource:   protocol.PolicyResourceSelectorV1{Kind: protocol.PolicySelectorPrefix, Value: "metrics"},
			Capability: protocol.PolicyCapabilitySelectorV1{Kind: protocol.PolicySelectorExact, Values: []protocol.CapabilityID{protocol.CapabilityRead}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	bindingID := strings.Repeat("a", 32)
	if _, err := state.Policy.CreateBinding(protocol.PolicyBindingCreateV1{
		Version: 1, BindingID: bindingID, Subject: "41", DefinitionID: definition.ID,
		Scope: protocol.PolicyOwnerScopeV1{Kind: protocol.PolicyScopeAuthorityDomain, NodeID: "1"},
	}, 1); err != nil {
		t.Fatal(err)
	}
	allowed := Request{Subject: 41, Capability: protocol.CapabilityRead, Resource: protocol.ResourceID{Owner: 42, Name: "metrics/cpu_percent"}}
	decision, err := state.Policy.Evaluate(allowed)
	if err != nil || !decision.Allowed || decision.BindingID != bindingID || decision.TopologyEpoch != 9 {
		t.Fatalf("unexpected scoped decision: %+v, %v", decision, err)
	}
	boundary := allowed
	boundary.Resource.Name = "metrics-evil/cpu_percent"
	if decision, err := state.Policy.Evaluate(boundary); err != nil || decision.Allowed {
		t.Fatalf("prefix crossed a segment boundary: %+v, %v", decision, err)
	}
	inside[42] = false
	if decision, err := state.Policy.Evaluate(allowed); err != nil || decision.Allowed {
		t.Fatalf("detached owner retained access: %+v, %v", decision, err)
	}
}

func TestPolicyStateMutationSuperadminRequiresAuthorityDomainBinding(t *testing.T) {
	state, err := OpenState(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Policy.Close()
	if err := state.Policy.BindScopeResolver(policyScopeResolverFunc(func(scope protocol.PolicyOwnerScopeV1, owner protocol.NodeID) (bool, uint64, error) {
		return scope.NodeID == "1" && owner == 1, 3, nil
	})); err != nil {
		t.Fatal(err)
	}
	ownerBinding := protocol.PolicyBindingCreateV1{
		Version: 1, BindingID: strings.Repeat("b", 32), Subject: "41", DefinitionID: protocol.BuiltinPolicySuperadmin,
		Scope: protocol.PolicyOwnerScopeV1{Kind: protocol.PolicyScopeOwner, NodeID: "1"},
	}
	if _, err := state.Policy.CreateBinding(ownerBinding, 1); err != nil {
		t.Fatal(err)
	}
	if allowed, _, err := state.Policy.HasAuthoritySuperadmin(41, 1); err != nil || allowed {
		t.Fatalf("owner-only superadmin must not mutate Authority policy: allowed=%v err=%v", allowed, err)
	}
	domainBinding := ownerBinding
	domainBinding.BindingID = strings.Repeat("c", 32)
	domainBinding.Scope.Kind = protocol.PolicyScopeAuthorityDomain
	if _, err := state.Policy.CreateBinding(domainBinding, 1); err != nil {
		t.Fatal(err)
	}
	if allowed, epoch, err := state.Policy.HasAuthoritySuperadmin(41, 1); err != nil || !allowed || epoch != 3 {
		t.Fatalf("Authority-domain superadmin not recognized: allowed=%v epoch=%d err=%v", allowed, epoch, err)
	}
}

func TestPolicyBindingExpiryPersistsGenerationChange(t *testing.T) {
	state, err := OpenState(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Policy.Close()
	if err := state.Policy.BindScopeResolver(policyScopeResolverFunc(func(scope protocol.PolicyOwnerScopeV1, owner protocol.NodeID) (bool, uint64, error) {
		return scope.NodeID == "1" && owner == 1, 4, nil
	})); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(100 * time.Millisecond).UnixMilli()
	if _, err := state.Policy.CreateBinding(protocol.PolicyBindingCreateV1{
		Version: 1, BindingID: strings.Repeat("f", 32), Subject: "41", DefinitionID: protocol.BuiltinPolicySuperadmin,
		Scope: protocol.PolicyOwnerScopeV1{Kind: protocol.PolicyScopeAuthorityDomain, NodeID: "1"}, ExpiresAtUnixMS: expires,
	}, 1); err != nil {
		t.Fatal(err)
	}
	createdGeneration := state.Policy.Generation()
	changes := make(chan uint64, 1)
	_, cancel, err := state.Policy.WatchGeneration(func(generation uint64) { changes <- generation })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	select {
	case generation := <-changes:
		if generation <= createdGeneration || len(state.Policy.Bindings()) != 0 {
			t.Fatalf("expiry generation=%d after %d, bindings=%d", generation, createdGeneration, len(state.Policy.Bindings()))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Binding expiry generation")
	}
}
