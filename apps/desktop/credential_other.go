//go:build !windows

package main

import (
	"errors"
	"sync"

	"github.com/yttydcs/myflowhub/runtime/auth"
)

type sessionIdentityStore struct {
	mu         sync.Mutex
	identity   *auth.Identity
	enrollment *auth.EnrollmentCredential
}

func newPlatformIdentityStore(string) (platformCredentialBackend, error) {
	return &sessionIdentityStore{}, nil
}

func (s *sessionIdentityStore) LoadEnrollmentCredential() (auth.EnrollmentCredential, bool, error) {
	if s == nil {
		return auth.EnrollmentCredential{}, false, errors.New("session Enrollment credential store is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enrollment == nil {
		return auth.EnrollmentCredential{}, false, nil
	}
	return cloneSessionEnrollmentCredential(*s.enrollment), true, nil
}

func (s *sessionIdentityStore) SaveEnrollmentCredential(credential auth.EnrollmentCredential) error {
	if s == nil {
		return errors.New("session Enrollment credential store is unavailable")
	}
	s.mu.Lock()
	cloned := cloneSessionEnrollmentCredential(credential)
	s.enrollment = &cloned
	s.mu.Unlock()
	return nil
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

func cloneSessionEnrollmentCredential(credential auth.EnrollmentCredential) auth.EnrollmentCredential {
	credential.DevicePublicKey = append([]byte(nil), credential.DevicePublicKey...)
	credential.DevicePrivateKey = append([]byte(nil), credential.DevicePrivateKey...)
	credential.ParentPublicKey = append([]byte(nil), credential.ParentPublicKey...)
	credential.AuthorityPublicKey = append([]byte(nil), credential.AuthorityPublicKey...)
	if credential.Grant != nil {
		grant := *credential.Grant
		credential.Grant = &grant
	}
	return credential
}
