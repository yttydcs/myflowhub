package auth

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"sort"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

const stateVersion = 1

type identityState struct {
	Version    int                `json:"version"`
	NodeID     protocol.NodeID    `json:"node_id"`
	PublicKey  ed25519.PublicKey  `json:"public_key"`
	PrivateKey ed25519.PrivateKey `json:"private_key"`
}

func LoadOrCreateIdentity(store *keystore.Store, nodeID protocol.NodeID) (Identity, error) {
	if store == nil {
		return Identity{}, errors.New("identity store is required")
	}
	if err := nodeID.Validate(); err != nil {
		return Identity{}, err
	}
	var state identityState
	found, err := store.Load("identity.json", &state)
	if err != nil {
		return Identity{}, fmt.Errorf("load identity: %w", err)
	}
	if found {
		if state.Version != stateVersion {
			return Identity{}, fmt.Errorf("load identity: unsupported state version %d", state.Version)
		}
		identity := Identity{NodeID: state.NodeID, PublicKey: append(ed25519.PublicKey(nil), state.PublicKey...), PrivateKey: append(ed25519.PrivateKey(nil), state.PrivateKey...)}
		if identity.NodeID != nodeID {
			return Identity{}, fmt.Errorf("load identity: configured NodeID %d does not match stored NodeID %d", nodeID, identity.NodeID)
		}
		if err := identity.Validate(); err != nil {
			return Identity{}, fmt.Errorf("load identity: %w", err)
		}
		return identity, nil
	}
	identity, err := GenerateIdentity(nodeID)
	if err != nil {
		return Identity{}, err
	}
	state = identityState{Version: stateVersion, NodeID: identity.NodeID, PublicKey: identity.PublicKey, PrivateKey: identity.PrivateKey}
	if err := store.Save("identity.json", state); err != nil {
		return Identity{}, fmt.Errorf("persist new identity: %w", err)
	}
	return identity, nil
}

type trustRecord struct {
	NodeID    protocol.NodeID   `json:"node_id"`
	PublicKey ed25519.PublicKey `json:"public_key"`
}

type trustState struct {
	Version    int           `json:"version"`
	Generation uint64        `json:"generation"`
	Records    []trustRecord `json:"records"`
}

func LoadTrustStore(store *keystore.Store) (*TrustStore, error) {
	if store == nil {
		return nil, errors.New("trust state store is required")
	}
	state := trustState{Version: stateVersion, Generation: 1}
	found, err := store.Load("trust.json", &state)
	if err != nil {
		return nil, fmt.Errorf("load trust state: %w", err)
	}
	if found && (state.Version != stateVersion || state.Generation == 0) {
		return nil, errors.New("load trust state: unsupported version or zero generation")
	}
	result := &TrustStore{keys: make(map[protocol.NodeID]ed25519.PublicKey), generation: state.Generation}
	for _, record := range state.Records {
		if err := record.NodeID.Validate(); err != nil || len(record.PublicKey) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("load trust state: invalid record for node %d", record.NodeID)
		}
		if _, exists := result.keys[record.NodeID]; exists {
			return nil, fmt.Errorf("load trust state: duplicate node %d", record.NodeID)
		}
		result.keys[record.NodeID] = append(ed25519.PublicKey(nil), record.PublicKey...)
	}
	result.persist = func(next trustState) error { return store.Save("trust.json", next) }
	if !found {
		if err := result.persist(result.snapshotLocked()); err != nil {
			return nil, fmt.Errorf("initialize trust state: %w", err)
		}
	}
	return result, nil
}

func (s *TrustStore) Generation() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.generation
}

func (s *TrustStore) Revoke(nodeID protocol.NodeID) error {
	if err := nodeID.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.keys[nodeID]; !exists {
		return ErrUntrustedIdentity
	}
	keys := cloneKeys(s.keys)
	delete(keys, nodeID)
	if s.generation == ^uint64(0) {
		return errors.New("trust generation exhausted")
	}
	next := trustState{Version: stateVersion, Generation: s.generation + 1, Records: recordsFromKeys(keys)}
	if s.persist != nil {
		if err := s.persist(next); err != nil {
			return fmt.Errorf("persist trust revocation: %w", err)
		}
	}
	s.keys = keys
	s.generation = next.Generation
	return nil
}

func (s *TrustStore) snapshotLocked() trustState {
	return trustState{Version: stateVersion, Generation: s.generation, Records: recordsFromKeys(s.keys)}
}

func cloneKeys(source map[protocol.NodeID]ed25519.PublicKey) map[protocol.NodeID]ed25519.PublicKey {
	result := make(map[protocol.NodeID]ed25519.PublicKey, len(source))
	for id, key := range source {
		result[id] = append(ed25519.PublicKey(nil), key...)
	}
	return result
}

func recordsFromKeys(keys map[protocol.NodeID]ed25519.PublicKey) []trustRecord {
	ids := make([]protocol.NodeID, 0, len(keys))
	for id := range keys {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	records := make([]trustRecord, 0, len(ids))
	for _, id := range ids {
		records = append(records, trustRecord{NodeID: id, PublicKey: append(ed25519.PublicKey(nil), keys[id]...)})
	}
	return records
}
