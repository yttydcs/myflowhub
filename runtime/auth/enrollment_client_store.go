package auth

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

const enrollmentClientStateVersion = 1

var (
	ErrEnrollmentCredentialMissing = errors.New("enrollment credential is missing")
	ErrEnrollmentNotGranted        = errors.New("enrollment credential is not granted")
)

type EnrollmentClientSnapshot struct {
	Status             string
	RequestID          string
	DevicePublicKey    ed25519.PublicKey
	Grant              *protocol.EnrollmentGrantV1
	ParentNodeID       protocol.NodeID
	ParentPublicKey    ed25519.PublicKey
	AuthorityNodeID    protocol.NodeID
	AuthorityPublicKey ed25519.PublicKey
}

type EnrollmentCredential struct {
	Version            int                         `json:"version"`
	Status             string                      `json:"status"`
	RequestID          string                      `json:"request_id"`
	DevicePublicKey    ed25519.PublicKey           `json:"device_public_key"`
	DevicePrivateKey   ed25519.PrivateKey          `json:"device_private_key"`
	Grant              *protocol.EnrollmentGrantV1 `json:"grant,omitempty"`
	ParentNodeID       protocol.NodeID             `json:"parent_node_id,omitempty"`
	ParentPublicKey    ed25519.PublicKey           `json:"parent_public_key,omitempty"`
	AuthorityNodeID    protocol.NodeID             `json:"authority_node_id,omitempty"`
	AuthorityPublicKey ed25519.PublicKey           `json:"authority_public_key,omitempty"`
}

type EnrollmentCredentialStore interface {
	LoadEnrollmentCredential() (EnrollmentCredential, bool, error)
	SaveEnrollmentCredential(EnrollmentCredential) error
}

// NodeCredential is the complete, already-enrolled identity material required
// to open an ordinary Node runtime. Callers must treat every byte slice as
// sensitive and must not persist a second identity copy.
type NodeCredential struct {
	Identity           Identity
	ParentNodeID       protocol.NodeID
	ParentPublicKey    ed25519.PublicKey
	AuthorityNodeID    protocol.NodeID
	AuthorityPublicKey ed25519.PublicKey
	EnrollmentID       string
}

// NodeCredentialSource resolves an existing Node identity without creating or
// mutating enrollment state.
type NodeCredentialSource interface {
	LoadNodeCredential() (NodeCredential, error)
}

// EnrollmentCredentialSource adapts a protected Enrollment credential store
// to the read-only NodeHost credential contract.
type EnrollmentCredentialSource struct {
	store EnrollmentCredentialStore
}

func NewEnrollmentCredentialSource(store EnrollmentCredentialStore) (*EnrollmentCredentialSource, error) {
	if store == nil {
		return nil, errors.New("enrollment credential store is required")
	}
	return &EnrollmentCredentialSource{store: store}, nil
}

func (source *EnrollmentCredentialSource) LoadNodeCredential() (NodeCredential, error) {
	if source == nil || source.store == nil {
		return NodeCredential{}, errors.New("enrollment credential source is required")
	}
	state, found, err := source.store.LoadEnrollmentCredential()
	if err != nil {
		return NodeCredential{}, fmt.Errorf("load enrolled Node credential: %w", err)
	}
	if !found {
		return NodeCredential{}, ErrEnrollmentCredentialMissing
	}
	if err := validateEnrollmentCredential(state); err != nil {
		return NodeCredential{}, fmt.Errorf("load enrolled Node credential: %w", err)
	}
	if state.Status != "enrolled" || state.Grant == nil {
		return NodeCredential{}, fmt.Errorf("%w: status %q", ErrEnrollmentNotGranted, state.Status)
	}
	nodeID, err := parseEnrollmentNodeID(state.Grant.NodeID)
	if err != nil {
		return NodeCredential{}, fmt.Errorf("load enrolled Node credential Node ID: %w", err)
	}
	identity, err := (DeviceIdentity{
		PublicKey:  state.DevicePublicKey,
		PrivateKey: state.DevicePrivateKey,
	}).Enroll(nodeID)
	if err != nil {
		return NodeCredential{}, fmt.Errorf("load enrolled Node credential identity: %w", err)
	}
	return cloneNodeCredential(NodeCredential{
		Identity:           identity,
		ParentNodeID:       state.ParentNodeID,
		ParentPublicKey:    state.ParentPublicKey,
		AuthorityNodeID:    state.AuthorityNodeID,
		AuthorityPublicKey: state.AuthorityPublicKey,
		EnrollmentID:       state.Grant.EnrollmentID,
	}), nil
}

// InspectEnrollmentClientState validates and snapshots an existing credential
// without creating or persisting Enrollment state.
func InspectEnrollmentClientState(store EnrollmentCredentialStore) (EnrollmentClientSnapshot, bool, error) {
	if store == nil {
		return EnrollmentClientSnapshot{}, false, errors.New("enrollment credential store is required")
	}
	state, found, err := store.LoadEnrollmentCredential()
	if err != nil {
		return EnrollmentClientSnapshot{}, false, fmt.Errorf("load enrollment client state: %w", err)
	}
	if !found {
		return EnrollmentClientSnapshot{}, false, nil
	}
	if err := validateEnrollmentCredential(state); err != nil {
		return EnrollmentClientSnapshot{}, false, fmt.Errorf("load enrollment client state: %w", err)
	}
	return enrollmentClientSnapshot(state), true, nil
}

type keystoreEnrollmentCredentialStore struct {
	store *keystore.Store
}

func (store keystoreEnrollmentCredentialStore) LoadEnrollmentCredential() (EnrollmentCredential, bool, error) {
	var credential EnrollmentCredential
	found, err := store.store.Load("enrollment-client.json", &credential)
	return credential, found, err
}

func (store keystoreEnrollmentCredentialStore) SaveEnrollmentCredential(credential EnrollmentCredential) error {
	return store.store.Save("enrollment-client.json", credential)
}

type EnrollmentClientState struct {
	mu    sync.Mutex
	store EnrollmentCredentialStore
	state EnrollmentCredential
}

func LoadOrCreateEnrollmentClientState(store *keystore.Store) (*EnrollmentClientState, error) {
	if store == nil {
		return nil, errors.New("enrollment client state store is required")
	}
	return LoadOrCreateEnrollmentClientStateWithStore(keystoreEnrollmentCredentialStore{store: store})
}

func LoadOrCreateEnrollmentClientStateWithStore(store EnrollmentCredentialStore) (*EnrollmentClientState, error) {
	if store == nil {
		return nil, errors.New("enrollment credential store is required")
	}
	state, found, err := store.LoadEnrollmentCredential()
	if err != nil {
		return nil, fmt.Errorf("load enrollment client state: %w", err)
	}
	if !found {
		device, err := GenerateDeviceIdentity()
		if err != nil {
			return nil, err
		}
		requestID, err := protocol.NewMessageID()
		if err != nil {
			return nil, err
		}
		state = EnrollmentCredential{
			Version: enrollmentClientStateVersion, Status: "device", RequestID: requestID.String(),
			DevicePublicKey: device.PublicKey, DevicePrivateKey: device.PrivateKey,
		}
		if err := store.SaveEnrollmentCredential(state); err != nil {
			return nil, fmt.Errorf("initialize enrollment client state: %w", err)
		}
	}
	if err := validateEnrollmentCredential(state); err != nil {
		return nil, fmt.Errorf("load enrollment client state: %w", err)
	}
	return &EnrollmentClientState{store: store, state: cloneEnrollmentCredential(state)}, nil
}

func (state *EnrollmentClientState) DeviceIdentity() DeviceIdentity {
	state.mu.Lock()
	defer state.mu.Unlock()
	return DeviceIdentity{
		PublicKey:  append(ed25519.PublicKey(nil), state.state.DevicePublicKey...),
		PrivateKey: append(ed25519.PrivateKey(nil), state.state.DevicePrivateKey...),
	}
}

func (state *EnrollmentClientState) Snapshot() EnrollmentClientSnapshot {
	state.mu.Lock()
	defer state.mu.Unlock()
	return enrollmentClientSnapshot(state.state)
}

func (state *EnrollmentClientState) RecordObservation(parentNodeID protocol.NodeID, parentPublicKey ed25519.PublicKey, authorityNodeID protocol.NodeID, authorityPublicKey ed25519.PublicKey) error {
	if err := validateEnrollmentObservation(parentNodeID, parentPublicKey, authorityNodeID, authorityPublicKey); err != nil {
		return err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.state.Status == "enrolled" {
		if state.state.ParentNodeID != parentNodeID || state.state.AuthorityNodeID != authorityNodeID || !bytes.Equal(state.state.ParentPublicKey, parentPublicKey) || !bytes.Equal(state.state.AuthorityPublicKey, authorityPublicKey) {
			return ErrEnrollmentConflict
		}
		return nil
	}
	if state.state.ParentNodeID != 0 && (state.state.ParentNodeID != parentNodeID || state.state.AuthorityNodeID != authorityNodeID || !bytes.Equal(state.state.ParentPublicKey, parentPublicKey) || !bytes.Equal(state.state.AuthorityPublicKey, authorityPublicKey)) {
		return ErrEnrollmentConflict
	}
	next := cloneEnrollmentCredential(state.state)
	next.Status = "pending"
	next.ParentNodeID = parentNodeID
	next.ParentPublicKey = append(ed25519.PublicKey(nil), parentPublicKey...)
	next.AuthorityNodeID = authorityNodeID
	next.AuthorityPublicKey = append(ed25519.PublicKey(nil), authorityPublicKey...)
	return state.commitLocked(next)
}

func (state *EnrollmentClientState) RecordGrant(grant protocol.EnrollmentGrantV1) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	authorityKey, err := base64.RawStdEncoding.DecodeString(grant.AuthorityPublicKey)
	if err != nil || len(authorityKey) != ed25519.PublicKeySize {
		return errors.New("Enrollment Grant Authority key is invalid")
	}
	if err := VerifyEnrollmentGrant(ed25519.PublicKey(authorityKey), grant); err != nil {
		return err
	}
	if grant.DevicePublicKey != base64.RawStdEncoding.EncodeToString(state.state.DevicePublicKey) {
		return ErrEnrollmentConflict
	}
	parentNodeID, err := parseEnrollmentNodeID(grant.ParentNodeID)
	if err != nil {
		return err
	}
	authorityNodeID, err := parseEnrollmentNodeID(grant.AuthorityNodeID)
	if err != nil {
		return err
	}
	parentKey, _ := base64.RawStdEncoding.DecodeString(grant.ParentPublicKey)
	if err := validateEnrollmentObservation(parentNodeID, ed25519.PublicKey(parentKey), authorityNodeID, ed25519.PublicKey(authorityKey)); err != nil {
		return err
	}
	if state.state.ParentNodeID != 0 && (state.state.ParentNodeID != parentNodeID || state.state.AuthorityNodeID != authorityNodeID || !bytes.Equal(state.state.ParentPublicKey, parentKey) || !bytes.Equal(state.state.AuthorityPublicKey, authorityKey)) {
		return ErrEnrollmentConflict
	}
	next := cloneEnrollmentCredential(state.state)
	next.Status = "enrolled"
	next.Grant = cloneEnrollmentGrant(&grant)
	next.ParentNodeID = parentNodeID
	next.ParentPublicKey = append(ed25519.PublicKey(nil), parentKey...)
	next.AuthorityNodeID = authorityNodeID
	next.AuthorityPublicKey = append(ed25519.PublicKey(nil), authorityKey...)
	return state.commitLocked(next)
}

func (state *EnrollmentClientState) Identity() (Identity, bool, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.state.Status != "enrolled" || state.state.Grant == nil {
		return Identity{}, false, nil
	}
	nodeIDValue, err := strconv.ParseUint(state.state.Grant.NodeID, 10, 64)
	if err != nil || nodeIDValue == 0 {
		return Identity{}, false, errors.New("persisted Enrollment Grant Node ID is invalid")
	}
	identity, err := (DeviceIdentity{PublicKey: state.state.DevicePublicKey, PrivateKey: state.state.DevicePrivateKey}).Enroll(protocol.NodeID(nodeIDValue))
	if err != nil {
		return Identity{}, false, err
	}
	return identity, true, nil
}

func (state *EnrollmentClientState) commitLocked(next EnrollmentCredential) error {
	if err := validateEnrollmentCredential(next); err != nil {
		return err
	}
	if err := state.store.SaveEnrollmentCredential(next); err != nil {
		return fmt.Errorf("persist enrollment client state: %w", err)
	}
	state.state = cloneEnrollmentCredential(next)
	return nil
}

func validateEnrollmentCredential(state EnrollmentCredential) error {
	if state.Version != enrollmentClientStateVersion {
		return fmt.Errorf("unsupported state version %d", state.Version)
	}
	if err := validateEnrollmentHexID("enrollment request ID", state.RequestID, 16); err != nil {
		return err
	}
	device := DeviceIdentity{PublicKey: state.DevicePublicKey, PrivateKey: state.DevicePrivateKey}
	if err := device.Validate(); err != nil {
		return err
	}
	switch state.Status {
	case "device":
		if state.Grant != nil || state.ParentNodeID != 0 || state.AuthorityNodeID != 0 || len(state.ParentPublicKey) != 0 || len(state.AuthorityPublicKey) != 0 {
			return errors.New("device state cannot contain Enrollment trust or Grant")
		}
	case "pending":
		if state.Grant != nil {
			return errors.New("pending state cannot contain a Grant")
		}
		if err := validateEnrollmentObservation(state.ParentNodeID, state.ParentPublicKey, state.AuthorityNodeID, state.AuthorityPublicKey); err != nil {
			return err
		}
	case "enrolled":
		if state.Grant == nil {
			return errors.New("enrolled state requires a Grant")
		}
		if err := validateEnrollmentObservation(state.ParentNodeID, state.ParentPublicKey, state.AuthorityNodeID, state.AuthorityPublicKey); err != nil {
			return err
		}
		if err := VerifyEnrollmentGrant(state.AuthorityPublicKey, *state.Grant); err != nil {
			return err
		}
		if state.Grant.DevicePublicKey != base64.RawStdEncoding.EncodeToString(state.DevicePublicKey) || state.Grant.ParentNodeID != strconv.FormatUint(uint64(state.ParentNodeID), 10) || state.Grant.ParentPublicKey != base64.RawStdEncoding.EncodeToString(state.ParentPublicKey) || state.Grant.AuthorityNodeID != strconv.FormatUint(uint64(state.AuthorityNodeID), 10) || state.Grant.AuthorityPublicKey != base64.RawStdEncoding.EncodeToString(state.AuthorityPublicKey) {
			return errors.New("persisted Enrollment Grant does not match client state")
		}
	default:
		return fmt.Errorf("invalid enrollment client status %q", state.Status)
	}
	return nil
}

func validateEnrollmentObservation(parentNodeID protocol.NodeID, parentPublicKey ed25519.PublicKey, authorityNodeID protocol.NodeID, authorityPublicKey ed25519.PublicKey) error {
	if err := parentNodeID.Validate(); err != nil {
		return fmt.Errorf("Enrollment parent: %w", err)
	}
	if len(parentPublicKey) != ed25519.PublicKeySize {
		return errors.New("Enrollment parent public key must be Ed25519")
	}
	if err := authorityNodeID.Validate(); err != nil {
		return fmt.Errorf("Enrollment Authority: %w", err)
	}
	if len(authorityPublicKey) != ed25519.PublicKeySize {
		return errors.New("Enrollment Authority public key must be Ed25519")
	}
	return nil
}

func cloneEnrollmentCredential(source EnrollmentCredential) EnrollmentCredential {
	source.DevicePublicKey = append(ed25519.PublicKey(nil), source.DevicePublicKey...)
	source.DevicePrivateKey = append(ed25519.PrivateKey(nil), source.DevicePrivateKey...)
	source.Grant = cloneEnrollmentGrant(source.Grant)
	source.ParentPublicKey = append(ed25519.PublicKey(nil), source.ParentPublicKey...)
	source.AuthorityPublicKey = append(ed25519.PublicKey(nil), source.AuthorityPublicKey...)
	return source
}

func cloneEnrollmentGrant(source *protocol.EnrollmentGrantV1) *protocol.EnrollmentGrantV1 {
	if source == nil {
		return nil
	}
	result := *source
	return &result
}

func enrollmentClientSnapshot(state EnrollmentCredential) EnrollmentClientSnapshot {
	return EnrollmentClientSnapshot{
		Status: state.Status, RequestID: state.RequestID, DevicePublicKey: append(ed25519.PublicKey(nil), state.DevicePublicKey...),
		Grant: cloneEnrollmentGrant(state.Grant), ParentNodeID: state.ParentNodeID,
		ParentPublicKey: append(ed25519.PublicKey(nil), state.ParentPublicKey...), AuthorityNodeID: state.AuthorityNodeID,
		AuthorityPublicKey: append(ed25519.PublicKey(nil), state.AuthorityPublicKey...),
	}
}

func cloneNodeCredential(source NodeCredential) NodeCredential {
	source.Identity.PublicKey = append(ed25519.PublicKey(nil), source.Identity.PublicKey...)
	source.Identity.PrivateKey = append(ed25519.PrivateKey(nil), source.Identity.PrivateKey...)
	source.ParentPublicKey = append(ed25519.PublicKey(nil), source.ParentPublicKey...)
	source.AuthorityPublicKey = append(ed25519.PublicKey(nil), source.AuthorityPublicKey...)
	return source
}
