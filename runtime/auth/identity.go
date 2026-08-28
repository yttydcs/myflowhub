package auth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
)

var ErrUntrustedIdentity = errors.New("untrusted identity")

type Identity struct {
	NodeID     protocol.NodeID
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

func GenerateIdentity(nodeID protocol.NodeID) (Identity, error) {
	if err := nodeID.Validate(); err != nil {
		return Identity{}, err
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Identity{}, fmt.Errorf("generate identity: %w", err)
	}
	return Identity{NodeID: nodeID, PublicKey: publicKey, PrivateKey: privateKey}, nil
}

func (i Identity) Validate() error {
	if err := i.NodeID.Validate(); err != nil {
		return err
	}
	if len(i.PublicKey) != ed25519.PublicKeySize || len(i.PrivateKey) != ed25519.PrivateKeySize {
		return errors.New("identity requires an Ed25519 key pair")
	}
	if !bytes.Equal(i.PrivateKey.Public().(ed25519.PublicKey), i.PublicKey) {
		return errors.New("identity public and private keys do not match")
	}
	return nil
}

type JoinClaim struct {
	NodeID        protocol.NodeID                `json:"node_id"`
	PublicKey     []byte                         `json:"public_key"`
	Nonce         [32]byte                       `json:"nonce"`
	TopologyEpoch uint64                         `json:"topology_epoch"`
	Permit        *protocol.ProvisioningPermitV1 `json:"permit,omitempty"`
	Signature     []byte                         `json:"signature"`
}

func NewJoinClaim(identity Identity, topologyEpoch uint64) (JoinClaim, error) {
	return newJoinClaim(identity, topologyEpoch, nil)
}

func NewJoinClaimWithPermit(identity Identity, topologyEpoch uint64, permit protocol.ProvisioningPermitV1) (JoinClaim, error) {
	if err := permit.Validate(); err != nil {
		return JoinClaim{}, fmt.Errorf("join permit: %w", err)
	}
	return newJoinClaim(identity, topologyEpoch, &permit)
}

func newJoinClaim(identity Identity, topologyEpoch uint64, permit *protocol.ProvisioningPermitV1) (JoinClaim, error) {
	if err := identity.Validate(); err != nil {
		return JoinClaim{}, err
	}
	if topologyEpoch == 0 {
		return JoinClaim{}, errors.New("join claim requires a topology epoch")
	}
	claim := JoinClaim{NodeID: identity.NodeID, PublicKey: append([]byte(nil), identity.PublicKey...), TopologyEpoch: topologyEpoch, Permit: permit}
	if _, err := rand.Read(claim.Nonce[:]); err != nil {
		return JoinClaim{}, fmt.Errorf("generate join nonce: %w", err)
	}
	claim.Signature = ed25519.Sign(identity.PrivateKey, joinMessage(claim.NodeID, claim.Nonce, claim.TopologyEpoch))
	return claim, nil
}

type TrustStore struct {
	mu         sync.RWMutex
	keys       map[protocol.NodeID]ed25519.PublicKey
	generation uint64
	persist    func(trustState) error
}

func NewTrustStore() *TrustStore {
	return &TrustStore{keys: make(map[protocol.NodeID]ed25519.PublicKey), generation: 1}
}

func (s *TrustStore) Add(nodeID protocol.NodeID, publicKey ed25519.PublicKey) error {
	if err := nodeID.Validate(); err != nil {
		return err
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New("trust store requires an Ed25519 public key")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, exists := s.keys[nodeID]; exists {
		if !bytes.Equal(current, publicKey) {
			return fmt.Errorf("identity %d already has a different key", nodeID)
		}
		return nil
	}
	keys := cloneKeys(s.keys)
	keys[nodeID] = append(ed25519.PublicKey(nil), publicKey...)
	if s.generation == ^uint64(0) {
		return errors.New("trust generation exhausted")
	}
	next := trustState{Version: stateVersion, Generation: s.generation + 1, Records: recordsFromKeys(keys)}
	if s.persist != nil {
		if err := s.persist(next); err != nil {
			return fmt.Errorf("persist trusted identity: %w", err)
		}
	}
	s.keys = keys
	s.generation = next.Generation
	return nil
}

func (s *TrustStore) PublicKey(nodeID protocol.NodeID) (ed25519.PublicKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, ok := s.keys[nodeID]
	return append(ed25519.PublicKey(nil), key...), ok
}

func (s *TrustStore) VerifyJoin(claim JoinClaim) error {
	key, ok := s.PublicKey(claim.NodeID)
	if !ok || !bytes.Equal(key, claim.PublicKey) {
		return ErrUntrustedIdentity
	}
	return VerifyJoinClaim(claim)
}

func VerifyJoinClaim(claim JoinClaim) error {
	if err := claim.NodeID.Validate(); err != nil {
		return err
	}
	if len(claim.PublicKey) != ed25519.PublicKeySize || claim.TopologyEpoch == 0 || len(claim.Signature) != ed25519.SignatureSize || !ed25519.Verify(ed25519.PublicKey(claim.PublicKey), joinMessage(claim.NodeID, claim.Nonce, claim.TopologyEpoch), claim.Signature) {
		return errors.New("invalid join signature")
	}
	return nil
}

func joinMessage(nodeID protocol.NodeID, nonce [32]byte, topologyEpoch uint64) []byte {
	message := make([]byte, len("MFH4-JOIN")+8+len(nonce)+8)
	copy(message, "MFH4-JOIN")
	binary.BigEndian.PutUint64(message[len("MFH4-JOIN"):], uint64(nodeID))
	copy(message[len("MFH4-JOIN")+8:], nonce[:])
	binary.BigEndian.PutUint64(message[len("MFH4-JOIN")+8+len(nonce):], topologyEpoch)
	return message
}

type JoinAck struct {
	NodeID        protocol.NodeID `json:"node_id"`
	ChildID       protocol.NodeID `json:"child_id"`
	ChildNonce    [32]byte        `json:"child_nonce"`
	TopologyEpoch uint64          `json:"topology_epoch"`
	Signature     []byte          `json:"signature"`
}

func SignJoinAck(parent Identity, childID protocol.NodeID, nonce [32]byte, epoch uint64) (JoinAck, error) {
	if err := parent.Validate(); err != nil {
		return JoinAck{}, err
	}
	if err := childID.Validate(); err != nil {
		return JoinAck{}, err
	}
	if epoch == 0 {
		return JoinAck{}, errors.New("join acknowledgement requires a topology epoch")
	}
	ack := JoinAck{NodeID: parent.NodeID, ChildID: childID, ChildNonce: nonce, TopologyEpoch: epoch}
	ack.Signature = ed25519.Sign(parent.PrivateKey, ackMessage(ack))
	return ack, nil
}

func (s *TrustStore) VerifyJoinAck(ack JoinAck, expectedParent, expectedChild protocol.NodeID, nonce [32]byte) error {
	if ack.NodeID != expectedParent || ack.ChildID != expectedChild || ack.ChildNonce != nonce || ack.TopologyEpoch == 0 {
		return errors.New("join acknowledgement does not match request")
	}
	key, ok := s.PublicKey(expectedParent)
	if !ok {
		return ErrUntrustedIdentity
	}
	if len(ack.Signature) != ed25519.SignatureSize || !ed25519.Verify(key, ackMessage(ack), ack.Signature) {
		return errors.New("invalid join acknowledgement signature")
	}
	return nil
}

func ackMessage(ack JoinAck) []byte {
	message := make([]byte, len("MFH4-ACK")+8+8+32+8)
	offset := copy(message, "MFH4-ACK")
	binary.BigEndian.PutUint64(message[offset:offset+8], uint64(ack.NodeID))
	offset += 8
	binary.BigEndian.PutUint64(message[offset:offset+8], uint64(ack.ChildID))
	offset += 8
	copy(message[offset:offset+32], ack.ChildNonce[:])
	offset += 32
	binary.BigEndian.PutUint64(message[offset:offset+8], ack.TopologyEpoch)
	return message
}
