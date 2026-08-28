package auth

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

type policyState struct {
	Version    int       `json:"version"`
	Generation uint64    `json:"generation"`
	Grants     []Request `json:"grants"`
}

type PolicyState struct {
	mu         sync.RWMutex
	generation uint64
	grants     map[Request]struct{}
	persist    func(policyState) error
	nextWatch  uint64
	watchers   map[uint64]func(uint64)
}

func LoadPolicyState(store *keystore.Store) (*PolicyState, error) {
	if store == nil {
		return nil, errors.New("policy state store is required")
	}
	state := policyState{Version: stateVersion, Generation: 1, Grants: []Request{}}
	found, err := store.Load("policy.json", &state)
	if err != nil {
		return nil, fmt.Errorf("load policy state: %w", err)
	}
	if found && (state.Version != stateVersion || state.Generation == 0) {
		return nil, errors.New("load policy state: unsupported version or zero generation")
	}
	result := &PolicyState{generation: state.Generation, grants: make(map[Request]struct{}), watchers: make(map[uint64]func(uint64))}
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
	result.persist = func(next policyState) error { return store.Save("policy.json", next) }
	if !found {
		if err := result.persist(result.snapshotLocked()); err != nil {
			return nil, fmt.Errorf("initialize policy state: %w", err)
		}
	}
	return result, nil
}

func (p *PolicyState) Authorize(_ context.Context, request Request) error {
	request, err := normalizeRequest(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	}
	p.mu.RLock()
	_, allowed := p.grants[request]
	p.mu.RUnlock()
	if !allowed {
		return fmt.Errorf("%w: subject %d cannot %s %s", ErrForbidden, request.Subject, request.Action, request.Resource.Name)
	}
	return nil
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
	next := cloneGrants(p.grants)
	next[request] = struct{}{}
	return p.commitLocked(next)
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
	return p.commitLocked(next)
}

func (p *PolicyState) RevokeSubject(subject protocol.NodeID) error {
	if err := subject.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	next := cloneGrants(p.grants)
	changed := false
	for grant := range next {
		if grant.Subject == subject {
			delete(next, grant)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return p.commitLocked(next)
}

func (p *PolicyState) commitLocked(grants map[Request]struct{}) error {
	if p.generation == ^uint64(0) {
		return errors.New("policy generation exhausted")
	}
	next := policyState{Version: stateVersion, Generation: p.generation + 1, Grants: requestsFromGrants(grants)}
	if err := p.persist(next); err != nil {
		return fmt.Errorf("persist policy state: %w", err)
	}
	p.grants = grants
	p.generation = next.Generation
	for _, observer := range p.watchers {
		observer(p.generation)
	}
	return nil
}

func (p *PolicyState) snapshotLocked() policyState {
	return policyState{Version: stateVersion, Generation: p.generation, Grants: requestsFromGrants(p.grants)}
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
