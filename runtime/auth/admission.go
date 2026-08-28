package auth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

var (
	ErrPermitInvalid  = errors.New("admission permit is invalid")
	ErrPermitConsumed = errors.New("admission permit was already consumed")
	ErrPermitRevoked  = errors.New("admission permit was revoked")
)

type PermitStatus string

const (
	PermitActive   PermitStatus = "active"
	PermitConsumed PermitStatus = "consumed"
	PermitRevoked  PermitStatus = "revoked"
)

type permitRecord struct {
	Permit     protocol.ProvisioningPermitV1 `json:"permit"`
	Status     PermitStatus                  `json:"status"`
	ConsumedAt int64                         `json:"consumed_at_unix_ms,omitempty"`
	RevokedAt  int64                         `json:"revoked_at_unix_ms,omitempty"`
}

type admissionState struct {
	Version    int            `json:"version"`
	Generation uint64         `json:"generation"`
	Records    []permitRecord `json:"records"`
}

type AdmissionConfig struct {
	Now        func() time.Time
	MaxTTL     time.Duration
	MaxRecords int
}

type Admission struct {
	mu         sync.Mutex
	identity   Identity
	store      *keystore.Store
	now        func() time.Time
	maxTTL     time.Duration
	maxRecords int
	generation uint64
	records    map[string]permitRecord
}

func LoadAdmission(identity Identity, store *keystore.Store, config AdmissionConfig) (*Admission, error) {
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("admission identity: %w", err)
	}
	if store == nil {
		return nil, errors.New("admission state store is required")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.MaxTTL <= 0 {
		config.MaxTTL = 24 * time.Hour
	}
	if config.MaxRecords <= 0 {
		config.MaxRecords = 4096
	}
	if config.MaxTTL < time.Second || config.MaxRecords < 1 {
		return nil, errors.New("admission limits are invalid")
	}
	state := admissionState{Version: stateVersion, Generation: 1, Records: []permitRecord{}}
	found, err := store.Load("admission.json", &state)
	if err != nil {
		return nil, fmt.Errorf("load admission state: %w", err)
	}
	if found && (state.Version != stateVersion || state.Generation == 0) {
		return nil, errors.New("load admission state: unsupported version or zero generation")
	}
	result := &Admission{identity: identity, store: store, now: config.Now, maxTTL: config.MaxTTL, maxRecords: config.MaxRecords, generation: state.Generation, records: make(map[string]permitRecord)}
	for _, record := range state.Records {
		if err := record.Permit.Validate(); err != nil {
			return nil, fmt.Errorf("load admission state: invalid permit: %w", err)
		}
		if record.Status != PermitActive && record.Status != PermitConsumed && record.Status != PermitRevoked {
			return nil, errors.New("load admission state: invalid permit status")
		}
		if _, exists := result.records[record.Permit.PermitID]; exists {
			return nil, errors.New("load admission state: duplicate permit ID")
		}
		result.records[record.Permit.PermitID] = record
	}
	if !found {
		if err := result.saveLocked(result.records, result.generation); err != nil {
			return nil, fmt.Errorf("initialize admission state: %w", err)
		}
	}
	return result, nil
}

func (a *Admission) Generation() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.generation
}

func (a *Admission) Issue(childID protocol.NodeID, childPublicKey ed25519.PublicKey, role string, ttl time.Duration) (protocol.ProvisioningPermitV1, error) {
	if err := childID.Validate(); err != nil {
		return protocol.ProvisioningPermitV1{}, err
	}
	if len(childPublicKey) != ed25519.PublicKeySize {
		return protocol.ProvisioningPermitV1{}, errors.New("admission child key must be Ed25519")
	}
	if role == "" || len(role) > protocol.MaxIdentifierBytes {
		return protocol.ProvisioningPermitV1{}, errors.New("admission role is invalid")
	}
	if ttl < time.Millisecond || ttl > a.maxTTL {
		return protocol.ProvisioningPermitV1{}, fmt.Errorf("admission TTL must be between 1ms and %s", a.maxTTL)
	}
	var rawID [16]byte
	if _, err := rand.Read(rawID[:]); err != nil {
		return protocol.ProvisioningPermitV1{}, fmt.Errorf("generate permit ID: %w", err)
	}
	now := a.now().UTC()
	permit := protocol.ProvisioningPermitV1{
		Version: 1, PermitID: hex.EncodeToString(rawID[:]), ParentNodeID: strconv.FormatUint(uint64(a.identity.NodeID), 10), ChildNodeID: strconv.FormatUint(uint64(childID), 10),
		ChildPublicKey: base64.RawStdEncoding.EncodeToString(childPublicKey), Role: role, IssuedAtUnixMS: now.UnixMilli(), ExpiresAtUnixMS: now.Add(ttl).UnixMilli(), MaxUses: 1,
	}
	permit.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(a.identity.PrivateKey, permitMessage(permit)))
	if err := permit.Validate(); err != nil {
		return protocol.ProvisioningPermitV1{}, fmt.Errorf("create permit: %w", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	next := clonePermitRecords(a.records)
	cleanupExpiredPermits(next, now.UnixMilli())
	if len(next) >= a.maxRecords {
		return protocol.ProvisioningPermitV1{}, errors.New("admission permit record limit reached")
	}
	next[permit.PermitID] = permitRecord{Permit: permit, Status: PermitActive}
	if err := a.commitLocked(next); err != nil {
		return protocol.ProvisioningPermitV1{}, err
	}
	return permit, nil
}

func (a *Admission) Consume(childID protocol.NodeID, childPublicKey ed25519.PublicKey, permit protocol.ProvisioningPermitV1) error {
	if err := a.verify(childID, childPublicKey, permit); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	record, exists := a.records[permit.PermitID]
	if !exists || !samePermit(record.Permit, permit) {
		return fmt.Errorf("%w: unknown permit", ErrPermitInvalid)
	}
	switch record.Status {
	case PermitConsumed:
		return ErrPermitConsumed
	case PermitRevoked:
		return ErrPermitRevoked
	case PermitActive:
	default:
		return fmt.Errorf("%w: invalid status", ErrPermitInvalid)
	}
	now := a.now().UTC().UnixMilli()
	if now < permit.IssuedAtUnixMS || now >= permit.ExpiresAtUnixMS {
		return fmt.Errorf("%w: expired or not yet valid", ErrPermitInvalid)
	}
	next := clonePermitRecords(a.records)
	record.Status = PermitConsumed
	record.ConsumedAt = now
	next[permit.PermitID] = record
	return a.commitLocked(next)
}

func (a *Admission) Revoke(permitID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	record, exists := a.records[permitID]
	if !exists {
		return fmt.Errorf("%w: unknown permit", ErrPermitInvalid)
	}
	if record.Status == PermitConsumed {
		return ErrPermitConsumed
	}
	if record.Status == PermitRevoked {
		return nil
	}
	next := clonePermitRecords(a.records)
	record.Status = PermitRevoked
	record.RevokedAt = a.now().UTC().UnixMilli()
	next[permitID] = record
	return a.commitLocked(next)
}

func (a *Admission) Active() []protocol.ProvisioningPermitV1 {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now().UTC().UnixMilli()
	result := make([]protocol.ProvisioningPermitV1, 0)
	for _, record := range a.records {
		if record.Status == PermitActive && now >= record.Permit.IssuedAtUnixMS && now < record.Permit.ExpiresAtUnixMS {
			result = append(result, record.Permit)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PermitID < result[j].PermitID })
	return result
}

func (a *Admission) verify(childID protocol.NodeID, childPublicKey ed25519.PublicKey, permit protocol.ProvisioningPermitV1) error {
	if err := permit.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrPermitInvalid, err)
	}
	if permit.ParentNodeID != strconv.FormatUint(uint64(a.identity.NodeID), 10) || permit.ChildNodeID != strconv.FormatUint(uint64(childID), 10) {
		return fmt.Errorf("%w: parent or child binding mismatch", ErrPermitInvalid)
	}
	encodedKey := base64.RawStdEncoding.EncodeToString(childPublicKey)
	if len(childPublicKey) != ed25519.PublicKeySize || permit.ChildPublicKey != encodedKey {
		return fmt.Errorf("%w: child key binding mismatch", ErrPermitInvalid)
	}
	signature, err := base64.RawStdEncoding.DecodeString(permit.Signature)
	if err != nil || !ed25519.Verify(a.identity.PublicKey, permitMessage(permit), signature) {
		return fmt.Errorf("%w: signature verification failed", ErrPermitInvalid)
	}
	return nil
}

func (a *Admission) commitLocked(records map[string]permitRecord) error {
	if a.generation == ^uint64(0) {
		return errors.New("admission generation exhausted")
	}
	nextGeneration := a.generation + 1
	if err := a.saveLocked(records, nextGeneration); err != nil {
		return fmt.Errorf("persist admission state: %w", err)
	}
	a.records = records
	a.generation = nextGeneration
	return nil
}

func (a *Admission) saveLocked(records map[string]permitRecord, generation uint64) error {
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	state := admissionState{Version: stateVersion, Generation: generation, Records: make([]permitRecord, 0, len(ids))}
	for _, id := range ids {
		state.Records = append(state.Records, records[id])
	}
	return a.store.Save("admission.json", state)
}

func permitMessage(permit protocol.ProvisioningPermitV1) []byte {
	unsigned := struct {
		Domain          string `json:"domain"`
		Version         int    `json:"version"`
		PermitID        string `json:"permit_id"`
		ParentNodeID    string `json:"parent_node_id"`
		ChildNodeID     string `json:"child_node_id"`
		ChildPublicKey  string `json:"child_public_key"`
		Role            string `json:"role"`
		IssuedAtUnixMS  int64  `json:"issued_at_unix_ms"`
		ExpiresAtUnixMS int64  `json:"expires_at_unix_ms"`
		MaxUses         int    `json:"max_uses"`
	}{"MFH4-PERMIT", permit.Version, permit.PermitID, permit.ParentNodeID, permit.ChildNodeID, permit.ChildPublicKey, permit.Role, permit.IssuedAtUnixMS, permit.ExpiresAtUnixMS, permit.MaxUses}
	data, _ := json.Marshal(unsigned)
	return data
}

func samePermit(left, right protocol.ProvisioningPermitV1) bool {
	return bytes.Equal(permitMessage(left), permitMessage(right)) && left.Signature == right.Signature
}

func clonePermitRecords(source map[string]permitRecord) map[string]permitRecord {
	result := make(map[string]permitRecord, len(source))
	for id, record := range source {
		result[id] = record
	}
	return result
}

func cleanupExpiredPermits(records map[string]permitRecord, now int64) {
	for id, record := range records {
		if record.Permit.ExpiresAtUnixMS <= now {
			delete(records, id)
		}
	}
}
