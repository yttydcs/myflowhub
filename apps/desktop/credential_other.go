//go:build !windows

package main

import (
	"errors"
	"sync"

	"github.com/yttydcs/myflowhub/runtime/auth"
)

type sessionIdentityStore struct {
	mu       sync.Mutex
	identity *auth.Identity
}

func newPlatformIdentityStore(string) (auth.IdentityStore, error) {
	return &sessionIdentityStore{}, nil
}

func platformCredentialMode() string { return "session-only-identity" }

func (s *sessionIdentityStore) LoadIdentity() (auth.Identity, bool, error) {
	if s == nil {
		return auth.Identity{}, false, errors.New("session identity store is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.identity == nil {
		return auth.Identity{}, false, nil
	}
	return cloneSessionIdentity(*s.identity), true, nil
}

func (s *sessionIdentityStore) SaveIdentity(identity auth.Identity) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	cloned := cloneSessionIdentity(identity)
	s.identity = &cloned
	s.mu.Unlock()
	return nil
}

func cloneSessionIdentity(identity auth.Identity) auth.Identity {
	identity.PublicKey = append([]byte(nil), identity.PublicKey...)
	identity.PrivateKey = append([]byte(nil), identity.PrivateKey...)
	return identity
}
