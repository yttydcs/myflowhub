package auth

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

const (
	legacyPolicyStateVersion = 1
	policyStateVersion       = 2
)

var (
	ErrPolicyConflict         = errors.New("policy conflict")
	ErrPolicyNotFound         = errors.New("policy record not found")
	ErrPolicyImmutable        = errors.New("policy definition is immutable")
	ErrPolicyScopeUnavailable = errors.New("policy owner scope resolver is unavailable")
)

type policyState struct {
	Version     int                           `json:"version"`
	Generation  uint64                        `json:"generation"`
	Grants      []Request                     `json:"grants"`
	Definitions []protocol.PolicyDefinitionV1 `json:"definitions,omitempty"`
	Bindings    []protocol.PolicyBindingV1    `json:"bindings,omitempty"`
}

// OwnerScopeResolver answers from the current authoritative topology. It must
// not maintain an independent permission or route tree.
type OwnerScopeResolver interface {
	MatchOwnerScope(protocol.PolicyOwnerScopeV1, protocol.NodeID) (matched bool, topologyEpoch uint64, err error)
}

type PolicyState struct {
	mu          sync.RWMutex
	generation  uint64
	grants      map[Request]struct{}
	definitions map[string]protocol.PolicyDefinitionV1
	bindings    map[string]protocol.PolicyBindingV1
	bySubject   map[protocol.NodeID][]string
	persist     func(policyState) error
	resolver    OwnerScopeResolver
	nextWatch   uint64
	watchers    map[uint64]func(uint64)
	expiryTimer *time.Timer
	closed      bool
}

func LoadPolicyState(store *keystore.Store) (*PolicyState, error) {
	if store == nil {
		return nil, errors.New("policy state store is required")
	}
	state := policyState{}
	found, err := store.Load("policy.json", &state)
	if err != nil {
		return nil, fmt.Errorf("load policy state: %w", err)
	}
	if !found {
		state = policyState{
			Version:     policyStateVersion,
			Generation:  1,
			Grants:      []Request{},
			Definitions: []protocol.PolicyDefinitionV1{builtinSuperadminDefinition()},
			Bindings:    []protocol.PolicyBindingV1{},
		}
	}
	if state.Generation == 0 {
		return nil, errors.New("load policy state: zero generation")
	}
	if state.Generation > protocol.MaxCollectionRevision {
		return nil, fmt.Errorf("load policy state: generation exceeds %d", protocol.MaxCollectionRevision)
	}
	if len(state.Grants) > protocol.MaxItems || len(state.Definitions) > protocol.MaxItems || len(state.Bindings) > protocol.MaxItems {
		return nil, fmt.Errorf("load policy state: grants, definitions, and bindings must each contain at most %d items", protocol.MaxItems)
	}
	migrated := false
	if found {
		switch state.Version {
		case legacyPolicyStateVersion:
			if state.Generation >= protocol.MaxCollectionRevision {
				return nil, errors.New("load policy state: generation exhausted during migration")
			}
			state.Version = policyStateVersion
			state.Generation++
			state.Definitions = []protocol.PolicyDefinitionV1{builtinSuperadminDefinition()}
			state.Bindings = []protocol.PolicyBindingV1{}
			migrated = true
		case policyStateVersion:
		default:
			return nil, fmt.Errorf("load policy state: unsupported version %d", state.Version)
		}
	}
	result := &PolicyState{
		generation: state.Generation, grants: make(map[Request]struct{}), definitions: make(map[string]protocol.PolicyDefinitionV1),
		bindings: make(map[string]protocol.PolicyBindingV1), watchers: make(map[uint64]func(uint64)),
	}
	for _, grant := range state.Grants {
		grant, err = normalizeRequest(grant)
		if err != nil {
			return nil, fmt.Errorf("load policy state: %w", err)
		}
		if _, exists := result.grants[grant]; exists {
			return nil, fmt.Errorf("load policy state: duplicate grant %+v", grant)
		}
		result.grants[grant] = struct{}{}
	}
	for _, definition := range state.Definitions {
		if err := definition.Validate(); err != nil {
			return nil, fmt.Errorf("load policy state definition %q: %w", definition.ID, err)
		}
		if _, exists := result.definitions[definition.ID]; exists {
			return nil, fmt.Errorf("load policy state: duplicate definition %q", definition.ID)
		}
		result.definitions[definition.ID] = cloneDefinition(definition)
	}
	if !sameDefinition(result.definitions[protocol.BuiltinPolicySuperadmin], builtinSuperadminDefinition()) {
		return nil, errors.New("load policy state: canonical superadmin definition is missing or modified")
	}
	now := time.Now().UTC().UnixMilli()
	prunedExpired := false
	for _, binding := range state.Bindings {
		if err := binding.Validate(); err != nil {
			return nil, fmt.Errorf("load policy state binding %q: %w", binding.BindingID, err)
		}
		if _, exists := result.definitions[binding.DefinitionID]; !exists {
			return nil, fmt.Errorf("load policy state binding %q: unknown definition %q", binding.BindingID, binding.DefinitionID)
		}
		if _, exists := result.bindings[binding.BindingID]; exists {
			return nil, fmt.Errorf("load policy state: duplicate binding %q", binding.BindingID)
		}
		if binding.ExpiresAtUnixMS != 0 && binding.ExpiresAtUnixMS <= now {
			prunedExpired = true
			continue
		}
		result.bindings[binding.BindingID] = cloneBinding(binding)
	}
	if prunedExpired {
		if result.generation >= protocol.MaxCollectionRevision {
			return nil, errors.New("load policy state: generation exhausted while pruning expired bindings")
		}
		result.generation++
		migrated = true
	}
	result.rebuildBindingIndexLocked()
	result.persist = func(next policyState) error { return store.Save("policy.json", next) }
	if !found || migrated {
		if err := result.persist(result.snapshotLocked()); err != nil {
			return nil, fmt.Errorf("initialize policy state: %w", err)
		}
	}
	result.scheduleExpiryLocked()
	return result, nil
}

func builtinSuperadminDefinition() protocol.PolicyDefinitionV1 {
	return protocol.PolicyDefinitionV1{
		Version: 1, ID: protocol.BuiltinPolicySuperadmin, Label: "Super administrator", Revision: 1, Immutable: true,
		Rules: []protocol.PolicyRuleV1{{
			Resource:   protocol.PolicyResourceSelectorV1{Kind: protocol.PolicySelectorAll},
			Capability: protocol.PolicyCapabilitySelectorV1{Kind: protocol.PolicySelectorAll},
		}},
	}
}

func (p *PolicyState) BindScopeResolver(resolver OwnerScopeResolver) error {
	if resolver == nil {
		return errors.New("policy owner scope resolver is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errors.New("policy state is closed")
	}
	if p.resolver != nil {
		return errors.New("policy owner scope resolver is already bound")
	}
	p.resolver = resolver
	return nil
}

func (p *PolicyState) Authorize(_ context.Context, request Request) error {
	decision, err := p.Evaluate(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	}
	if !decision.Allowed {
		request, normalizeErr := normalizeRequest(request)
		if normalizeErr != nil {
			return fmt.Errorf("%w: %v", ErrForbidden, normalizeErr)
		}
		return fmt.Errorf("%w: subject %d cannot %s %s", ErrForbidden, request.Subject, request.Action, request.Resource.Name)
	}
	return nil
}

func (p *PolicyState) Evaluate(request Request) (protocol.PolicyEvaluationV1, error) {
	request, err := normalizeRequest(request)
	if err != nil {
		return protocol.PolicyEvaluationV1{}, err
	}
	p.mu.RLock()
	generation := p.generation
	if _, allowed := p.grants[request]; allowed {
		p.mu.RUnlock()
		return protocol.PolicyEvaluationV1{Version: 1, Allowed: true, Source: protocol.PolicyEvaluationExactGrant, PolicyGeneration: generation}, nil
	}
	bindingIDs := append([]string(nil), p.bySubject[request.Subject]...)
	bindings := make([]protocol.PolicyBindingV1, 0, len(bindingIDs))
	definitions := make(map[string]protocol.PolicyDefinitionV1, len(bindingIDs))
	for _, id := range bindingIDs {
		binding := p.bindings[id]
		bindings = append(bindings, cloneBinding(binding))
		if _, exists := definitions[binding.DefinitionID]; !exists {
			definitions[binding.DefinitionID] = cloneDefinition(p.definitions[binding.DefinitionID])
		}
	}
	resolver := p.resolver
	p.mu.RUnlock()
	now := time.Now().UTC().UnixMilli()
	for _, binding := range bindings {
		if binding.ExpiresAtUnixMS != 0 && binding.ExpiresAtUnixMS <= now {
			continue
		}
		if resolver == nil {
			return protocol.PolicyEvaluationV1{}, ErrPolicyScopeUnavailable
		}
		matched, epoch, err := resolver.MatchOwnerScope(binding.Scope, request.Resource.Owner)
		if err != nil {
			return protocol.PolicyEvaluationV1{}, fmt.Errorf("resolve binding %s scope: %w", binding.BindingID, err)
		}
		if !matched {
			continue
		}
		definition := definitions[binding.DefinitionID]
		for index, rule := range definition.Rules {
			if matchPolicyRule(rule, request.Resource.Name, request.Capability) {
				return protocol.PolicyEvaluationV1{
					Version: 1, Allowed: true, Source: protocol.PolicyEvaluationBinding, PolicyGeneration: generation,
					TopologyEpoch: epoch, BindingID: binding.BindingID, DefinitionID: definition.ID,
					DefinitionRevision: definition.Revision, RuleIndex: index, Scope: policyScopePointer(binding.Scope),
				}, nil
			}
		}
	}
	return protocol.PolicyEvaluationV1{Version: 1, Allowed: false, Source: protocol.PolicyEvaluationNone, PolicyGeneration: generation}, nil
}

func matchPolicyRule(rule protocol.PolicyRuleV1, resourceName string, capability protocol.CapabilityID) bool {
	resourceMatched := false
	switch rule.Resource.Kind {
	case protocol.PolicySelectorAll:
		resourceMatched = true
	case protocol.PolicySelectorExact:
		resourceMatched = resourceName == rule.Resource.Value
	case protocol.PolicySelectorPrefix:
		resourceMatched = resourceName == rule.Resource.Value || strings.HasPrefix(resourceName, rule.Resource.Value+"/")
	}
	if !resourceMatched {
		return false
	}
	if rule.Capability.Kind == protocol.PolicySelectorAll {
		return true
	}
	index := sort.Search(len(rule.Capability.Values), func(i int) bool { return rule.Capability.Values[i] >= capability })
	return index < len(rule.Capability.Values) && rule.Capability.Values[index] == capability
}

func (p *PolicyState) Generation() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.generation
}

func (p *PolicyState) WatchGeneration(observer func(uint64)) (uint64, func(), error) {
	if observer == nil {
		return 0, nil, errors.New("policy generation observer is required")
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, nil, errors.New("policy state is closed")
	}
	p.nextWatch++
	id := p.nextWatch
	p.watchers[id] = observer
	generation := p.generation
	p.mu.Unlock()
	var once sync.Once
	return generation, func() {
		once.Do(func() {
			p.mu.Lock()
			delete(p.watchers, id)
			p.mu.Unlock()
		})
	}, nil
}

func (p *PolicyState) Grant(request Request) error {
	request, err := normalizeRequest(request)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.grants[request]; exists {
		return nil
	}
	if len(p.grants) >= protocol.MaxItems {
		return fmt.Errorf("policy grants cannot exceed %d items", protocol.MaxItems)
	}
	next := cloneGrants(p.grants)
	next[request] = struct{}{}
	return p.commitLocked(next, cloneDefinitions(p.definitions), cloneBindings(p.bindings))
}

func (p *PolicyState) Revoke(request Request) error {
	request, err := normalizeRequest(request)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.grants[request]; !exists {
		return nil
	}
	next := cloneGrants(p.grants)
	delete(next, request)
	return p.commitLocked(next, cloneDefinitions(p.definitions), cloneBindings(p.bindings))
}

func (p *PolicyState) RevokeSubject(subject protocol.NodeID) error {
	if err := subject.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	grants := cloneGrants(p.grants)
	bindings := cloneBindings(p.bindings)
	changed := false
	for grant := range grants {
		if grant.Subject == subject {
			delete(grants, grant)
			changed = true
		}
	}
	for id, binding := range bindings {
		parsed, _ := strconv.ParseUint(binding.Subject, 10, 64)
		if protocol.NodeID(parsed) == subject {
			delete(bindings, id)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return p.commitLocked(grants, cloneDefinitions(p.definitions), bindings)
}

func (p *PolicyState) PutDefinition(request protocol.PolicyDefinitionPutV1) (protocol.PolicyDefinitionV1, error) {
	if err := request.Validate(); err != nil {
		return protocol.PolicyDefinitionV1{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	existing, exists := p.definitions[request.ID]
	if request.ExpectedRevision == 0 {
		if exists {
			return protocol.PolicyDefinitionV1{}, fmt.Errorf("%w: definition %q already exists", ErrPolicyConflict, request.ID)
		}
		if len(p.definitions) >= protocol.MaxItems {
			return protocol.PolicyDefinitionV1{}, fmt.Errorf("policy definitions cannot exceed %d items", protocol.MaxItems)
		}
	} else {
		if !exists {
			return protocol.PolicyDefinitionV1{}, fmt.Errorf("%w: definition %q", ErrPolicyNotFound, request.ID)
		}
		if existing.Immutable {
			return protocol.PolicyDefinitionV1{}, fmt.Errorf("%w: %s", ErrPolicyImmutable, request.ID)
		}
		if existing.Revision != request.ExpectedRevision {
			return protocol.PolicyDefinitionV1{}, fmt.Errorf("%w: definition %q revision is %d, want %d", ErrPolicyConflict, request.ID, existing.Revision, request.ExpectedRevision)
		}
		if existing.Revision == protocol.MaxCollectionRevision {
			return protocol.PolicyDefinitionV1{}, errors.New("policy definition revision exhausted")
		}
	}
	revision := uint64(1)
	if exists {
		revision = existing.Revision + 1
	}
	definition := protocol.PolicyDefinitionV1{Version: 1, ID: request.ID, Label: request.Label, Revision: revision, Rules: cloneRules(request.Rules)}
	protocol.SortPolicyRules(definition.Rules)
	if err := definition.Validate(); err != nil {
		return protocol.PolicyDefinitionV1{}, err
	}
	definitions := cloneDefinitions(p.definitions)
	definitions[definition.ID] = definition
	if err := p.commitLocked(cloneGrants(p.grants), definitions, cloneBindings(p.bindings)); err != nil {
		return protocol.PolicyDefinitionV1{}, err
	}
	return cloneDefinition(definition), nil
}

func (p *PolicyState) DeleteDefinition(request protocol.PolicyDefinitionDeleteV1) error {
	if err := request.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	existing, exists := p.definitions[request.ID]
	if !exists {
		return fmt.Errorf("%w: definition %q", ErrPolicyNotFound, request.ID)
	}
	if existing.Immutable {
		return fmt.Errorf("%w: %s", ErrPolicyImmutable, request.ID)
	}
	if existing.Revision != request.ExpectedRevision {
		return fmt.Errorf("%w: definition %q revision is %d, want %d", ErrPolicyConflict, request.ID, existing.Revision, request.ExpectedRevision)
	}
	for _, binding := range p.bindings {
		if binding.DefinitionID == request.ID {
			return fmt.Errorf("%w: definition %q is referenced by binding %s", ErrPolicyConflict, request.ID, binding.BindingID)
		}
	}
	definitions := cloneDefinitions(p.definitions)
	delete(definitions, request.ID)
	return p.commitLocked(cloneGrants(p.grants), definitions, cloneBindings(p.bindings))
}

func (p *PolicyState) CreateBinding(request protocol.PolicyBindingCreateV1, actor protocol.NodeID) (protocol.PolicyBindingV1, error) {
	if err := request.Validate(); err != nil {
		return protocol.PolicyBindingV1{}, err
	}
	if err := actor.Validate(); err != nil {
		return protocol.PolicyBindingV1{}, fmt.Errorf("binding actor: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.definitions[request.DefinitionID]; !exists {
		return protocol.PolicyBindingV1{}, fmt.Errorf("%w: definition %q", ErrPolicyNotFound, request.DefinitionID)
	}
	if existing, exists := p.bindings[request.BindingID]; exists {
		if bindingMatchesCreate(existing, request) {
			return cloneBinding(existing), nil
		}
		return protocol.PolicyBindingV1{}, fmt.Errorf("%w: binding %s already exists with different content", ErrPolicyConflict, request.BindingID)
	}
	if len(p.bindings) >= protocol.MaxItems {
		return protocol.PolicyBindingV1{}, fmt.Errorf("policy bindings cannot exceed %d items", protocol.MaxItems)
	}
	now := time.Now().UTC().UnixMilli()
	if request.ExpiresAtUnixMS != 0 && request.ExpiresAtUnixMS <= now {
		return protocol.PolicyBindingV1{}, errors.New("binding expiry must be in the future")
	}
	binding := protocol.PolicyBindingV1{
		Version: 1, BindingID: request.BindingID, Subject: request.Subject, DefinitionID: request.DefinitionID,
		Scope: request.Scope, CreatedBy: strconv.FormatUint(uint64(actor), 10), CreatedAtUnixMS: now, ExpiresAtUnixMS: request.ExpiresAtUnixMS,
	}
	if err := binding.Validate(); err != nil {
		return protocol.PolicyBindingV1{}, err
	}
	bindings := cloneBindings(p.bindings)
	bindings[binding.BindingID] = binding
	if err := p.commitLocked(cloneGrants(p.grants), cloneDefinitions(p.definitions), bindings); err != nil {
		return protocol.PolicyBindingV1{}, err
	}
	return cloneBinding(binding), nil
}

func (p *PolicyState) RevokeBinding(bindingID string) error {
	request := protocol.PolicyBindingRevokeV1{Version: 1, BindingID: bindingID}
	if err := request.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.bindings[bindingID]; !exists {
		return nil
	}
	bindings := cloneBindings(p.bindings)
	delete(bindings, bindingID)
	return p.commitLocked(cloneGrants(p.grants), cloneDefinitions(p.definitions), bindings)
}

func (p *PolicyState) Definitions() []protocol.PolicyDefinitionV1 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return definitionsFromMap(p.definitions)
}

func (p *PolicyState) Definition(id string) (protocol.PolicyDefinitionV1, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	value, ok := p.definitions[id]
	return cloneDefinition(value), ok
}

func (p *PolicyState) Bindings() []protocol.PolicyBindingV1 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return bindingsFromMap(p.bindings)
}

func (p *PolicyState) Binding(id string) (protocol.PolicyBindingV1, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	value, ok := p.bindings[id]
	return cloneBinding(value), ok
}

func (p *PolicyState) Grants() []Request {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return requestsFromGrants(p.grants)
}

func (p *PolicyState) HasAuthoritySuperadmin(subject, authority protocol.NodeID) (bool, uint64, error) {
	if err := subject.Validate(); err != nil {
		return false, 0, err
	}
	if err := authority.Validate(); err != nil {
		return false, 0, err
	}
	p.mu.RLock()
	ids := append([]string(nil), p.bySubject[subject]...)
	bindings := make([]protocol.PolicyBindingV1, 0, len(ids))
	for _, id := range ids {
		if value := p.bindings[id]; value.DefinitionID == protocol.BuiltinPolicySuperadmin {
			bindings = append(bindings, cloneBinding(value))
		}
	}
	resolver := p.resolver
	p.mu.RUnlock()
	if len(bindings) == 0 {
		return false, 0, nil
	}
	if resolver == nil {
		return false, 0, ErrPolicyScopeUnavailable
	}
	now := time.Now().UTC().UnixMilli()
	for _, binding := range bindings {
		if binding.ExpiresAtUnixMS != 0 && binding.ExpiresAtUnixMS <= now {
			continue
		}
		if binding.Scope.Kind != protocol.PolicyScopeAuthorityDomain || binding.Scope.NodeID != strconv.FormatUint(uint64(authority), 10) {
			continue
		}
		matched, epoch, err := resolver.MatchOwnerScope(binding.Scope, authority)
		if err != nil {
			return false, 0, err
		}
		if matched {
			return true, epoch, nil
		}
	}
	return false, 0, nil
}

func (p *PolicyState) ValidateOwnerScope(scope protocol.PolicyOwnerScopeV1) error {
	if err := scope.Validate(); err != nil {
		return err
	}
	parsed, _ := strconv.ParseUint(scope.NodeID, 10, 64)
	p.mu.RLock()
	resolver := p.resolver
	p.mu.RUnlock()
	if resolver == nil {
		return ErrPolicyScopeUnavailable
	}
	matched, _, err := resolver.MatchOwnerScope(scope, protocol.NodeID(parsed))
	if err != nil {
		return err
	}
	if !matched {
		return errors.New("policy owner scope anchor is outside the current authority topology")
	}
	return nil
}

func (p *PolicyState) commitLocked(grants map[Request]struct{}, definitions map[string]protocol.PolicyDefinitionV1, bindings map[string]protocol.PolicyBindingV1) error {
	if p.closed {
		return errors.New("policy state is closed")
	}
	if p.generation >= protocol.MaxCollectionRevision {
		return errors.New("policy generation exhausted")
	}
	next := policyState{
		Version: policyStateVersion, Generation: p.generation + 1, Grants: requestsFromGrants(grants),
		Definitions: definitionsFromMap(definitions), Bindings: bindingsFromMap(bindings),
	}
	if err := p.persist(next); err != nil {
		return fmt.Errorf("persist policy state: %w", err)
	}
	p.grants, p.definitions, p.bindings = grants, definitions, bindings
	p.generation = next.Generation
	p.rebuildBindingIndexLocked()
	p.scheduleExpiryLocked()
	for _, observer := range p.watchers {
		observer(p.generation)
	}
	return nil
}

func (p *PolicyState) snapshotLocked() policyState {
	return policyState{
		Version: policyStateVersion, Generation: p.generation, Grants: requestsFromGrants(p.grants),
		Definitions: definitionsFromMap(p.definitions), Bindings: bindingsFromMap(p.bindings),
	}
}

func (p *PolicyState) scheduleExpiryLocked() {
	if p.expiryTimer != nil {
		p.expiryTimer.Stop()
		p.expiryTimer = nil
	}
	if p.closed {
		return
	}
	now := time.Now().UTC().UnixMilli()
	earliest := int64(0)
	for _, binding := range p.bindings {
		if binding.ExpiresAtUnixMS > now && (earliest == 0 || binding.ExpiresAtUnixMS < earliest) {
			earliest = binding.ExpiresAtUnixMS
		}
	}
	if earliest == 0 {
		return
	}
	delay := time.Until(time.UnixMilli(earliest))
	if delay < time.Millisecond {
		delay = time.Millisecond
	}
	p.expiryTimer = time.AfterFunc(delay, p.expireBindings)
}

func (p *PolicyState) expireBindings() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	now := time.Now().UTC().UnixMilli()
	bindings := cloneBindings(p.bindings)
	changed := false
	for id, binding := range bindings {
		if binding.ExpiresAtUnixMS != 0 && binding.ExpiresAtUnixMS <= now {
			delete(bindings, id)
			changed = true
		}
	}
	if !changed {
		p.scheduleExpiryLocked()
		return
	}
	if err := p.commitLocked(cloneGrants(p.grants), cloneDefinitions(p.definitions), bindings); err != nil {
		p.expiryTimer = time.AfterFunc(time.Second, p.expireBindings)
	}
}

func (p *PolicyState) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	if p.expiryTimer != nil {
		p.expiryTimer.Stop()
		p.expiryTimer = nil
	}
	p.watchers = make(map[uint64]func(uint64))
	return nil
}

func (p *PolicyState) rebuildBindingIndexLocked() {
	p.bySubject = make(map[protocol.NodeID][]string)
	for id, binding := range p.bindings {
		parsed, _ := strconv.ParseUint(binding.Subject, 10, 64)
		subject := protocol.NodeID(parsed)
		p.bySubject[subject] = append(p.bySubject[subject], id)
	}
	for subject := range p.bySubject {
		sort.Strings(p.bySubject[subject])
	}
}

func validateRequest(request Request) error {
	_, err := normalizeRequest(request)
	return err
}

func cloneGrants(source map[Request]struct{}) map[Request]struct{} {
	result := make(map[Request]struct{}, len(source))
	for grant := range source {
		result[grant] = struct{}{}
	}
	return result
}

func requestsFromGrants(grants map[Request]struct{}) []Request {
	result := make([]Request, 0, len(grants))
	for grant := range grants {
		result = append(result, grant)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Subject != result[j].Subject {
			return result[i].Subject < result[j].Subject
		}
		if result[i].Action != result[j].Action {
			return result[i].Action < result[j].Action
		}
		if result[i].Resource.Owner != result[j].Resource.Owner {
			return result[i].Resource.Owner < result[j].Resource.Owner
		}
		return result[i].Resource.Name < result[j].Resource.Name
	})
	return result
}

func cloneDefinition(value protocol.PolicyDefinitionV1) protocol.PolicyDefinitionV1 {
	value.Rules = cloneRules(value.Rules)
	return value
}

func cloneRules(values []protocol.PolicyRuleV1) []protocol.PolicyRuleV1 {
	result := append([]protocol.PolicyRuleV1(nil), values...)
	for index := range result {
		result[index].Capability.Values = append([]protocol.CapabilityID(nil), result[index].Capability.Values...)
	}
	return result
}

func cloneDefinitions(source map[string]protocol.PolicyDefinitionV1) map[string]protocol.PolicyDefinitionV1 {
	result := make(map[string]protocol.PolicyDefinitionV1, len(source))
	for id, value := range source {
		result[id] = cloneDefinition(value)
	}
	return result
}

func definitionsFromMap(source map[string]protocol.PolicyDefinitionV1) []protocol.PolicyDefinitionV1 {
	result := make([]protocol.PolicyDefinitionV1, 0, len(source))
	for _, value := range source {
		result = append(result, cloneDefinition(value))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func cloneBinding(value protocol.PolicyBindingV1) protocol.PolicyBindingV1 { return value }

func policyScopePointer(value protocol.PolicyOwnerScopeV1) *protocol.PolicyOwnerScopeV1 {
	return &value
}

func cloneBindings(source map[string]protocol.PolicyBindingV1) map[string]protocol.PolicyBindingV1 {
	result := make(map[string]protocol.PolicyBindingV1, len(source))
	for id, value := range source {
		result[id] = cloneBinding(value)
	}
	return result
}

func bindingsFromMap(source map[string]protocol.PolicyBindingV1) []protocol.PolicyBindingV1 {
	result := make([]protocol.PolicyBindingV1, 0, len(source))
	for _, value := range source {
		result = append(result, cloneBinding(value))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].BindingID < result[j].BindingID })
	return result
}

func sameDefinition(left, right protocol.PolicyDefinitionV1) bool {
	if left.Version != right.Version || left.ID != right.ID || left.Label != right.Label || left.Revision != right.Revision || left.Immutable != right.Immutable || len(left.Rules) != len(right.Rules) {
		return false
	}
	for index := range left.Rules {
		l, r := left.Rules[index], right.Rules[index]
		if l.Resource != r.Resource || l.Capability.Kind != r.Capability.Kind || fmt.Sprint(l.Capability.Values) != fmt.Sprint(r.Capability.Values) {
			return false
		}
	}
	return true
}

func bindingMatchesCreate(binding protocol.PolicyBindingV1, request protocol.PolicyBindingCreateV1) bool {
	return binding.BindingID == request.BindingID && binding.Subject == request.Subject && binding.DefinitionID == request.DefinitionID && binding.Scope == request.Scope && binding.ExpiresAtUnixMS == request.ExpiresAtUnixMS
}
