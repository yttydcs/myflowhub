package nodehost

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

// Config describes one complete Go node runtime. Parent and Listeners determine
// the node's topology role; product names do not.
type Config struct {
	StateDirectory   string
	NodeID           protocol.NodeID
	IdentityStore    auth.IdentityStore
	CredentialSource auth.NodeCredentialSource
	Runtime          RuntimeConfig
	Parent           *ParentConfig
	Listeners        []ListenerConfig
}

// RuntimeConfig exposes only the non-owning limits and mechanics from
// node.Config. NodeHost always supplies identity, trust, policy, admission, and
// the optional parent permit from its persistent state and ParentConfig.
type RuntimeConfig struct {
	Session             link.SessionConfig
	Subscriptions       subscription.Config
	Commands            command.Config
	JoinTimeout         time.Duration
	MaxHandshakes       int
	MaxPending          int
	MaxResourceSessions int
}

type ParentConfig struct {
	NodeID     protocol.NodeID
	PublicKey  ed25519.PublicKey
	Permit     *protocol.ProvisioningPermitV1
	Driver     link.Driver
	Endpoint   link.Endpoint
	Supervisor node.SupervisorConfig
}

type ListenerConfig struct {
	Driver   link.Driver
	Endpoint link.Endpoint
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.StateDirectory) == "" {
		return errors.New("node host state directory is required")
	}
	if config.CredentialSource == nil {
		if err := config.NodeID.Validate(); err != nil {
			return fmt.Errorf("node host node ID: %w", err)
		}
	} else {
		if config.IdentityStore != nil {
			return errors.New("node host credential source and identity store are mutually exclusive")
		}
		if config.NodeID != 0 {
			if err := config.NodeID.Validate(); err != nil {
				return fmt.Errorf("node host node ID: %w", err)
			}
		}
		if config.Parent == nil {
			return errors.New("node host credential source requires a parent configuration")
		}
	}
	if config.NodeID != 0 && config.Runtime.Session.LocalNode != 0 && config.Runtime.Session.LocalNode != config.NodeID {
		return fmt.Errorf("node host runtime session local node %d conflicts with node ID %d", config.Runtime.Session.LocalNode, config.NodeID)
	}
	if config.Runtime.Commands.Authorizer != nil {
		return errors.New("node host runtime command authorizer is owned by persistent policy")
	}
	for index, listener := range config.Listeners {
		if err := validateListener(listener); err != nil {
			return fmt.Errorf("node host listener %d: %w", index, err)
		}
	}
	if config.Parent != nil {
		if err := validateParentShape(*config.Parent, config.CredentialSource != nil); err != nil {
			return fmt.Errorf("node host parent: %w", err)
		}
		if config.NodeID != 0 && config.Parent.NodeID != 0 && config.Parent.NodeID == config.NodeID {
			return errors.New("node host parent cannot be the local node")
		}
	}
	return nil
}

func validateListener(config ListenerConfig) error {
	if err := link.ValidateDriver(config.Driver); err != nil {
		return err
	}
	if err := config.Endpoint.Validate(); err != nil {
		return err
	}
	return nil
}

func validateParentShape(config ParentConfig, allowCredentialDefaults bool) error {
	if config.NodeID == 0 && !allowCredentialDefaults {
		return errors.New("node ID must be configured")
	}
	if config.NodeID != 0 {
		if err := config.NodeID.Validate(); err != nil {
			return fmt.Errorf("node ID: %w", err)
		}
	}
	if err := link.ValidateDriver(config.Driver); err != nil {
		return err
	}
	if err := config.Endpoint.Validate(); err != nil {
		return err
	}
	if len(config.PublicKey) != 0 && len(config.PublicKey) != ed25519.PublicKeySize {
		return errors.New("public key must be an Ed25519 public key")
	}
	if err := node.ValidateSupervisorConfig(config.Supervisor); err != nil {
		return fmt.Errorf("supervisor: %w", err)
	}
	if config.Permit != nil {
		if err := config.Permit.Validate(); err != nil {
			return fmt.Errorf("permit: %w", err)
		}
	}
	return nil
}

func applyNodeCredential(config *Config, credential auth.NodeCredential) error {
	if config == nil || config.CredentialSource == nil {
		return errors.New("node host credential source is required")
	}
	if err := credential.Identity.Validate(); err != nil {
		return fmt.Errorf("node host credential identity: %w", err)
	}
	if err := credential.ParentNodeID.Validate(); err != nil {
		return fmt.Errorf("node host credential parent: %w", err)
	}
	if len(credential.ParentPublicKey) != ed25519.PublicKeySize {
		return errors.New("node host credential parent public key must be Ed25519")
	}
	if err := credential.AuthorityNodeID.Validate(); err != nil {
		return fmt.Errorf("node host credential Authority: %w", err)
	}
	if len(credential.AuthorityPublicKey) != ed25519.PublicKeySize {
		return errors.New("node host credential Authority public key must be Ed25519")
	}
	if strings.TrimSpace(credential.EnrollmentID) == "" {
		return errors.New("node host credential Enrollment ID is required")
	}
	if config.NodeID != 0 && config.NodeID != credential.Identity.NodeID {
		return fmt.Errorf("node host configured NodeID %d conflicts with credential NodeID %d", config.NodeID, credential.Identity.NodeID)
	}
	if config.Parent == nil {
		return errors.New("node host credential source requires a parent configuration")
	}
	if config.Parent.NodeID != 0 && config.Parent.NodeID != credential.ParentNodeID {
		return fmt.Errorf("node host configured parent NodeID %d conflicts with credential parent NodeID %d", config.Parent.NodeID, credential.ParentNodeID)
	}
	if len(config.Parent.PublicKey) != 0 && !bytes.Equal(config.Parent.PublicKey, credential.ParentPublicKey) {
		return errors.New("node host configured parent public key conflicts with credential parent public key")
	}
	config.NodeID = credential.Identity.NodeID
	config.Parent.NodeID = credential.ParentNodeID
	config.Parent.PublicKey = append(ed25519.PublicKey(nil), credential.ParentPublicKey...)
	if config.Runtime.Session.LocalNode != 0 && config.Runtime.Session.LocalNode != config.NodeID {
		return fmt.Errorf("node host runtime session local node %d conflicts with credential NodeID %d", config.Runtime.Session.LocalNode, config.NodeID)
	}
	if config.Parent.NodeID == config.NodeID {
		return errors.New("node host credential parent cannot be the local node")
	}
	return nil
}

func validateParentState(config ParentConfig, state *auth.State) error {
	if state == nil {
		return errors.New("persistent auth state is required")
	}
	if config.Permit != nil {
		permit := *config.Permit
		parentID := strconv.FormatUint(uint64(config.NodeID), 10)
		childID := strconv.FormatUint(uint64(state.Identity.NodeID), 10)
		if permit.ParentNodeID != parentID || permit.ChildNodeID != childID {
			return errors.New("parent permit identity binding does not match configured parent and local node")
		}
		childKey := base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey)
		if permit.ChildPublicKey != childKey {
			return errors.New("parent permit public key binding does not match local identity")
		}
		now := time.Now().UTC().UnixMilli()
		if now < permit.IssuedAtUnixMS || now >= permit.ExpiresAtUnixMS {
			return errors.New("parent permit is expired or not yet valid")
		}
	}
	if len(config.PublicKey) != 0 {
		if err := state.Trust.Add(config.NodeID, config.PublicKey); err != nil {
			return fmt.Errorf("trust parent identity: %w", err)
		}
	}
	if _, ok := state.Trust.PublicKey(config.NodeID); !ok {
		return fmt.Errorf("parent identity %d is not trusted; provide its public key or provision trust first", config.NodeID)
	}
	return nil
}

func (config RuntimeConfig) nodeConfig(state *auth.State, permit *protocol.ProvisioningPermitV1) node.Config {
	return node.Config{
		Identity:            state.Identity,
		Trust:               state.Trust,
		Policy:              state.Policy,
		Admission:           state.Admission,
		JoinPermit:          permit,
		Session:             config.Session,
		Subscriptions:       config.Subscriptions,
		Commands:            config.Commands,
		JoinTimeout:         config.JoinTimeout,
		MaxHandshakes:       config.MaxHandshakes,
		MaxPending:          config.MaxPending,
		MaxResourceSessions: config.MaxResourceSessions,
	}
}

func cloneConfig(config Config) Config {
	result := config
	result.Listeners = append([]ListenerConfig(nil), config.Listeners...)
	if config.Parent != nil {
		parent := *config.Parent
		parent.PublicKey = append(ed25519.PublicKey(nil), config.Parent.PublicKey...)
		if config.Parent.Permit != nil {
			permit := *config.Parent.Permit
			parent.Permit = &permit
		}
		result.Parent = &parent
	}
	return result
}
