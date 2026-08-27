package auth

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

type State struct {
	Directory string
	Store     *keystore.Store
	Identity  Identity
	Trust     *TrustStore
	Policy    *PolicyState
	Admission *Admission
}

func OpenState(directory string, nodeID protocol.NodeID) (*State, error) {
	if directory == "" {
		return nil, errors.New("runtime state directory is required")
	}
	if err := nodeID.Validate(); err != nil {
		return nil, err
	}
	store, err := keystore.New(filepath.Join(directory, "state"))
	if err != nil {
		return nil, err
	}
	identity, err := LoadOrCreateIdentity(store, nodeID)
	if err != nil {
		return nil, err
	}
	trust, err := LoadTrustStore(store)
	if err != nil {
		return nil, err
	}
	if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
		return nil, fmt.Errorf("trust local identity: %w", err)
	}
	policy, err := LoadPolicyState(store)
	if err != nil {
		return nil, err
	}
	admission, err := LoadAdmission(identity, store, AdmissionConfig{})
	if err != nil {
		return nil, err
	}
	return &State{
		Directory: directory, Store: store, Identity: identity, Trust: trust, Policy: policy, Admission: admission,
	}, nil
}
