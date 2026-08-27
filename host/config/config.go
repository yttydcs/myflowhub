package config

import (
	"errors"
	"fmt"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
)

type Runtime struct {
	Directory string
	Store     *keystore.Store
	Identity  auth.Identity
	Trust     *auth.TrustStore
	Policy    *auth.PolicyState
	Admission *auth.Admission
	Settings  *Settings
}

func Open(directory string, nodeID protocol.NodeID) (*Runtime, error) {
	state, err := auth.OpenState(directory, nodeID)
	if err != nil {
		return nil, err
	}
	settings, err := loadSettings(state.Store)
	if err != nil {
		return nil, err
	}
	return &Runtime{
		Directory: state.Directory, Store: state.Store, Identity: state.Identity, Trust: state.Trust,
		Policy: state.Policy, Admission: state.Admission, Settings: settings,
	}, nil
}

func (r *Runtime) RevokeNode(nodeID protocol.NodeID) error {
	if r == nil || r.Trust == nil || r.Policy == nil {
		return errors.New("host runtime config is not initialized")
	}
	if nodeID == r.Identity.NodeID {
		return errors.New("local host identity cannot be revoked")
	}
	if err := r.Trust.Revoke(nodeID); err != nil {
		return err
	}
	if err := r.Policy.RevokeSubject(nodeID); err != nil {
		return fmt.Errorf("remove revoked node policy: %w", err)
	}
	return nil
}
