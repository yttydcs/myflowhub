package auth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

const enrollmentAuthorityStateVersion = 1

var (
	ErrEnrollmentConflict = errors.New("enrollment state conflicts with the request")
	ErrEnrollmentPending  = errors.New("enrollment is pending approval")
	ErrEnrollmentRejected = errors.New("enrollment was rejected")
	ErrEnrollmentExpired  = errors.New("enrollment request expired")
	ErrEnrollmentRevoked  = errors.New("enrollment was revoked")
)

type EnrollmentAuthorityConfig struct {
	Now                 func() time.Time
	Random              io.Reader
	MaxPermitTTL        time.Duration
	PendingTTL          time.Duration
	MaxPermits          int
	MaxRequests         int
	MaxPending          int
	MaxPendingPerParent int
	MaxEnrollments      int
	IsTargetDescendant  func(target, candidate protocol.NodeID) bool
}

type EnrollmentSubmission struct {
	RequestID        string
	DevicePublicKey  ed25519.PublicKey
	ParentNodeID     protocol.NodeID
	ParentPublicKey  ed25519.PublicKey
	Permit           *protocol.EnrollmentPermitV1
	TranscriptDigest [32]byte
}

type EnrollmentOutcome struct {
	Status    string
	RequestID string
	Grant     *protocol.EnrollmentGrantV1
	Reason    string
}

type EnrollmentPermitRecord struct {
	Permit              protocol.EnrollmentPermitV1
	Status              string
	IssueRequestID      string
	ConsumedByRequestID string
	ConsumedAtUnixMS    int64
	RevokedAtUnixMS     int64
	Reason              string
}

type EnrollmentRequestRecord struct {
	RequestID                  string
	DevicePublicKey            ed25519.PublicKey
	DevicePublicKeyFingerprint string
	ParentNodeID               protocol.NodeID
	ParentPublicKey            ed25519.PublicKey
	TranscriptDigest           string
	Status                     string
	AdmissionProfile           string
	EnrollmentID               string
	CreatedAtUnixMS            int64
	UpdatedAtUnixMS            int64
	ExpiresAtUnixMS            int64
	Reason                     string
}

type EnrollmentRecord struct {
	Grant                      protocol.EnrollmentGrantV1
	DevicePublicKeyFingerprint string
	RequestID                  string
	Status                     string
	RevokedAtUnixMS            int64
	Reason                     string
}

type enrollmentAuthorityPermitState struct {
	Permit              protocol.EnrollmentPermitV1 `json:"permit"`
	Status              string                      `json:"status"`
	IssueRequestID      string                      `json:"issue_request_id"`
	ConsumedByRequestID string                      `json:"consumed_by_request_id,omitempty"`
	ConsumedAtUnixMS    int64                       `json:"consumed_at_unix_ms,omitempty"`
	RevokedAtUnixMS     int64                       `json:"revoked_at_unix_ms,omitempty"`
	Reason              string                      `json:"reason,omitempty"`
}

type enrollmentAuthorityRequestState struct {
	RequestID                  string            `json:"request_id"`
	DevicePublicKey            ed25519.PublicKey `json:"device_public_key"`
	DevicePublicKeyFingerprint string            `json:"device_public_key_fingerprint"`
	ParentNodeID               protocol.NodeID   `json:"parent_node_id"`
	ParentPublicKey            ed25519.PublicKey `json:"parent_public_key"`
	TranscriptDigest           string            `json:"transcript_digest"`
	Status                     string            `json:"status"`
	AdmissionProfile           string            `json:"admission_profile,omitempty"`
	EnrollmentID               string            `json:"enrollment_id,omitempty"`
	CreatedAtUnixMS            int64             `json:"created_at_unix_ms"`
	UpdatedAtUnixMS            int64             `json:"updated_at_unix_ms"`
	ExpiresAtUnixMS            int64             `json:"expires_at_unix_ms"`
	Reason                     string            `json:"reason,omitempty"`
}

type enrollmentAuthorityEnrollmentState struct {
	Grant                      protocol.EnrollmentGrantV1 `json:"grant"`
	DevicePublicKeyFingerprint string                     `json:"device_public_key_fingerprint"`
	RequestID                  string                     `json:"request_id"`
	Status                     string                     `json:"status"`
	RevokedAtUnixMS            int64                      `json:"revoked_at_unix_ms,omitempty"`
	Reason                     string                     `json:"reason,omitempty"`
}

type enrollmentAuthorityState struct {
	Version            int                                  `json:"version"`
	Generation         uint64                               `json:"generation"`
	AuthorityNodeID    protocol.NodeID                      `json:"authority_node_id"`
	AuthorityPublicKey ed25519.PublicKey                    `json:"authority_public_key"`
	Permits            []enrollmentAuthorityPermitState     `json:"permits"`
	Requests           []enrollmentAuthorityRequestState    `json:"requests"`
	Enrollments        []enrollmentAuthorityEnrollmentState `json:"enrollments"`
	UsedNodeIDs        []protocol.NodeID                    `json:"used_node_ids"`
}

type enrollmentAuthorityData struct {
	permits     map[string]enrollmentAuthorityPermitState
	requests    map[string]enrollmentAuthorityRequestState
	enrollments map[string]enrollmentAuthorityEnrollmentState
	usedNodeIDs map[protocol.NodeID]struct{}
}

type EnrollmentAuthority struct {
	mu                  sync.Mutex
	identity            Identity
	store               *keystore.Store
	now                 func() time.Time
	random              io.Reader
	maxPermitTTL        time.Duration
	pendingTTL          time.Duration
	maxPermits          int
	maxRequests         int
	maxPending          int
	maxPendingPerParent int
	maxEnrollments      int
	isTargetDescendant  func(protocol.NodeID, protocol.NodeID) bool
	generation          uint64
	data                enrollmentAuthorityData
}

func LoadEnrollmentAuthority(identity Identity, store *keystore.Store, config EnrollmentAuthorityConfig) (*EnrollmentAuthority, error) {
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("enrollment authority identity: %w", err)
	}
	if store == nil {
		return nil, errors.New("enrollment authority state store is required")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Random == nil {
		config.Random = rand.Reader
	}
	if config.MaxPermitTTL <= 0 {
		config.MaxPermitTTL = 24 * time.Hour
	}
	if config.PendingTTL <= 0 {
		config.PendingTTL = 24 * time.Hour
	}
	if config.MaxPermits <= 0 {
		config.MaxPermits = 10_000
	}
	if config.MaxRequests <= 0 {
		config.MaxRequests = 100_000
	}
	if config.MaxPending <= 0 {
		config.MaxPending = 1_024
		if config.MaxPending > config.MaxRequests {
			config.MaxPending = config.MaxRequests
		}
	}
	if config.MaxPendingPerParent <= 0 {
		config.MaxPendingPerParent = 128
		if config.MaxPendingPerParent > config.MaxPending {
			config.MaxPendingPerParent = config.MaxPending
		}
	}
	if config.MaxEnrollments <= 0 {
		config.MaxEnrollments = 100_000
	}
	if config.MaxPermitTTL < time.Second || config.PendingTTL < time.Second || config.MaxPermits < 1 || config.MaxRequests < 1 || config.MaxPending < 1 || config.MaxPendingPerParent < 1 || config.MaxPendingPerParent > config.MaxPending || config.MaxPending > config.MaxRequests || config.MaxEnrollments < 1 {
		return nil, errors.New("enrollment authority limits are invalid")
	}
	state := enrollmentAuthorityState{
		Version:            enrollmentAuthorityStateVersion,
		Generation:         1,
		AuthorityNodeID:    identity.NodeID,
		AuthorityPublicKey: append(ed25519.PublicKey(nil), identity.PublicKey...),
		Permits:            []enrollmentAuthorityPermitState{},
		Requests:           []enrollmentAuthorityRequestState{},
		Enrollments:        []enrollmentAuthorityEnrollmentState{},
		UsedNodeIDs:        []protocol.NodeID{identity.NodeID},
	}
	found, err := store.Load("enrollment-authority.json", &state)
	if err != nil {
		return nil, fmt.Errorf("load enrollment authority state: %w", err)
	}
	if state.Version != enrollmentAuthorityStateVersion || state.Generation == 0 {
		return nil, errors.New("load enrollment authority state: unsupported version or zero generation")
	}
	if state.AuthorityNodeID != identity.NodeID || !bytes.Equal(state.AuthorityPublicKey, identity.PublicKey) {
		return nil, errors.New("load enrollment authority state: authority identity does not match persisted state")
	}
	data, err := enrollmentAuthorityDataFromState(state, identity.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("load enrollment authority state: %w", err)
	}
	result := &EnrollmentAuthority{
		identity: identity, store: store, now: config.Now, random: config.Random,
		maxPermitTTL: config.MaxPermitTTL, pendingTTL: config.PendingTTL, maxPermits: config.MaxPermits, maxRequests: config.MaxRequests, maxPending: config.MaxPending,
		maxPendingPerParent: config.MaxPendingPerParent,
		maxEnrollments:      config.MaxEnrollments, isTargetDescendant: config.IsTargetDescendant,
		generation: state.Generation, data: data,
	}
	if !found {
		if err := result.saveLocked(data, state.Generation); err != nil {
			return nil, fmt.Errorf("initialize enrollment authority state: %w", err)
		}
	}
	return result, nil
}

func (authority *EnrollmentAuthority) Generation() uint64 {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	return authority.generation
}

func (authority *EnrollmentAuthority) PublicKey() ed25519.PublicKey {
	return append(ed25519.PublicKey(nil), authority.identity.PublicKey...)
}

func (authority *EnrollmentAuthority) NodeID() protocol.NodeID {
	return authority.identity.NodeID
}

func (authority *EnrollmentAuthority) IssuePermit(issueRequestID, deviceFingerprint string, targetNodeID protocol.NodeID, allowDescendants bool, admissionProfile string, ttl time.Duration) (protocol.EnrollmentPermitV1, error) {
	if err := validateEnrollmentHexID("issue request ID", issueRequestID, 16); err != nil {
		return protocol.EnrollmentPermitV1{}, err
	}
	if err := validateEnrollmentHexID("device public key fingerprint", deviceFingerprint, sha256.Size); err != nil {
		return protocol.EnrollmentPermitV1{}, err
	}
	if err := targetNodeID.Validate(); err != nil {
		return protocol.EnrollmentPermitV1{}, err
	}
	if admissionProfile == "" || len(admissionProfile) > protocol.MaxIdentifierBytes {
		return protocol.EnrollmentPermitV1{}, errors.New("admission profile is invalid")
	}
	if ttl < time.Second || ttl > authority.maxPermitTTL {
		return protocol.EnrollmentPermitV1{}, fmt.Errorf("permit TTL must be between one second and %s", authority.maxPermitTTL)
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	for _, record := range authority.data.permits {
		if record.IssueRequestID != issueRequestID {
			continue
		}
		if record.Permit.DevicePublicKeyFingerprint != deviceFingerprint || record.Permit.TargetNodeID != strconv.FormatUint(uint64(targetNodeID), 10) || record.Permit.AllowDescendants != allowDescendants || record.Permit.AdmissionProfile != admissionProfile {
			return protocol.EnrollmentPermitV1{}, ErrEnrollmentConflict
		}
		return record.Permit, nil
	}
	if len(authority.data.permits) >= authority.maxPermits {
		return protocol.EnrollmentPermitV1{}, errors.New("enrollment permit record limit reached")
	}
	next := cloneEnrollmentAuthorityData(authority.data)
	permitID, err := authority.newUniqueIDLocked(func(id string) bool {
		_, exists := next.permits[id]
		return exists
	})
	if err != nil {
		return protocol.EnrollmentPermitV1{}, err
	}
	now := authority.now().UTC()
	permit := protocol.EnrollmentPermitV1{
		Version: protocol.SchemaVersionV1, PermitID: permitID,
		AuthorityNodeID:            strconv.FormatUint(uint64(authority.identity.NodeID), 10),
		AuthorityPublicKey:         base64.RawStdEncoding.EncodeToString(authority.identity.PublicKey),
		DevicePublicKeyFingerprint: deviceFingerprint,
		TargetNodeID:               strconv.FormatUint(uint64(targetNodeID), 10), AllowDescendants: allowDescendants,
		AdmissionProfile: admissionProfile, IssuedAtUnixMS: now.UnixMilli(), ExpiresAtUnixMS: now.Add(ttl).UnixMilli(),
		AuthorityEpoch: authority.generation + 1,
	}
	permit.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(authority.identity.PrivateKey, enrollmentPermitMessage(permit)))
	if err := permit.Validate(); err != nil {
		return protocol.EnrollmentPermitV1{}, fmt.Errorf("create enrollment permit: %w", err)
	}
	next.permits[permit.PermitID] = enrollmentAuthorityPermitState{Permit: permit, Status: "active", IssueRequestID: issueRequestID}
	if err := authority.commitLocked(next); err != nil {
		return protocol.EnrollmentPermitV1{}, err
	}
	return permit, nil
}

func (authority *EnrollmentAuthority) Submit(submission EnrollmentSubmission) (EnrollmentOutcome, error) {
	if err := validateEnrollmentSubmission(submission); err != nil {
		return EnrollmentOutcome{}, err
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	fingerprint := DevicePublicKeyFingerprint(submission.DevicePublicKey)
	transcript := hex.EncodeToString(submission.TranscriptDigest[:])
	now := authority.now().UTC().UnixMilli()
	next := cloneEnrollmentAuthorityData(authority.data)
	expiredChanged := expirePendingRequests(next.requests, now)
	if existing, ok := next.requests[submission.RequestID]; ok {
		if existing.DevicePublicKeyFingerprint != fingerprint || existing.ParentNodeID != submission.ParentNodeID || !bytes.Equal(existing.ParentPublicKey, submission.ParentPublicKey) {
			return EnrollmentOutcome{}, ErrEnrollmentConflict
		}
		if submission.Permit != nil && existing.Status != "approved" {
			grant, err := authority.grantSubmissionWithPermitLocked(next, submission, existing, fingerprint, transcript, now)
			if err != nil {
				return EnrollmentOutcome{}, err
			}
			if err := authority.commitLocked(next); err != nil {
				return EnrollmentOutcome{}, err
			}
			return EnrollmentOutcome{Status: "granted", RequestID: submission.RequestID, Grant: &grant}, nil
		}
		if expiredChanged {
			if err := authority.commitLocked(next); err != nil {
				return EnrollmentOutcome{}, err
			}
		}
		return authority.outcomeForRequestLocked(existing)
	}
	for _, existing := range next.requests {
		if existing.DevicePublicKeyFingerprint != fingerprint || existing.ParentNodeID != submission.ParentNodeID || !bytes.Equal(existing.ParentPublicKey, submission.ParentPublicKey) {
			continue
		}
		if existing.Status != "pending" && existing.Status != "approved" {
			continue
		}
		if submission.Permit != nil && existing.Status == "pending" {
			grant, err := authority.grantSubmissionWithPermitLocked(next, submission, existing, fingerprint, transcript, now)
			if err != nil {
				return EnrollmentOutcome{}, err
			}
			if err := authority.commitLocked(next); err != nil {
				return EnrollmentOutcome{}, err
			}
			return EnrollmentOutcome{Status: "granted", RequestID: submission.RequestID, Grant: &grant}, nil
		}
		if expiredChanged {
			if err := authority.commitLocked(next); err != nil {
				return EnrollmentOutcome{}, err
			}
		}
		outcome, err := authority.outcomeForRequestLocked(existing)
		outcome.RequestID = submission.RequestID
		return outcome, err
	}
	request := enrollmentAuthorityRequestState{
		RequestID: submission.RequestID, DevicePublicKey: append(ed25519.PublicKey(nil), submission.DevicePublicKey...),
		DevicePublicKeyFingerprint: fingerprint, ParentNodeID: submission.ParentNodeID, TranscriptDigest: transcript,
		ParentPublicKey: append(ed25519.PublicKey(nil), submission.ParentPublicKey...),
		Status:          "pending", CreatedAtUnixMS: now, UpdatedAtUnixMS: now, ExpiresAtUnixMS: time.UnixMilli(now).Add(authority.pendingTTL).UnixMilli(),
	}
	if len(next.requests) >= authority.maxRequests {
		return EnrollmentOutcome{}, errors.New("enrollment request record limit reached")
	}
	if submission.Permit == nil {
		if countPendingRequests(next.requests) >= authority.maxPending {
			return EnrollmentOutcome{}, errors.New("pending enrollment request limit reached")
		}
		if countPendingRequestsForParent(next.requests, submission.ParentNodeID) >= authority.maxPendingPerParent {
			return EnrollmentOutcome{}, errors.New("pending enrollment request limit reached for parent")
		}
		next.requests[submission.RequestID] = request
		if err := authority.commitLocked(next); err != nil {
			return EnrollmentOutcome{}, err
		}
		return EnrollmentOutcome{Status: "pending", RequestID: submission.RequestID}, nil
	}
	grant, err := authority.grantSubmissionWithPermitLocked(next, submission, request, fingerprint, transcript, now)
	if err != nil {
		return EnrollmentOutcome{}, err
	}
	if err := authority.commitLocked(next); err != nil {
		return EnrollmentOutcome{}, err
	}
	return EnrollmentOutcome{Status: "granted", RequestID: submission.RequestID, Grant: &grant}, nil
}

func (authority *EnrollmentAuthority) grantSubmissionWithPermitLocked(next enrollmentAuthorityData, submission EnrollmentSubmission, request enrollmentAuthorityRequestState, fingerprint, transcript string, now int64) (protocol.EnrollmentGrantV1, error) {
	permitRecord, err := authority.consumePermitLocked(next, submission, fingerprint, now)
	if err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	request.TranscriptDigest = transcript
	grant, err := authority.grantLocked(next, request, permitRecord.Permit.AdmissionProfile, now)
	if err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	permitRecord.Status = "consumed"
	permitRecord.ConsumedByRequestID = submission.RequestID
	permitRecord.ConsumedAtUnixMS = now
	next.permits[permitRecord.Permit.PermitID] = permitRecord
	return grant, nil
}

func (authority *EnrollmentAuthority) Approve(commandRequestID, enrollmentRequestID, admissionProfile string) (protocol.EnrollmentGrantV1, error) {
	if err := validateEnrollmentHexID("command request ID", commandRequestID, 16); err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	if err := validateEnrollmentHexID("enrollment request ID", enrollmentRequestID, 16); err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	if admissionProfile == "" || len(admissionProfile) > protocol.MaxIdentifierBytes {
		return protocol.EnrollmentGrantV1{}, errors.New("admission profile is invalid")
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	request, exists := authority.data.requests[enrollmentRequestID]
	if !exists {
		return protocol.EnrollmentGrantV1{}, errors.New("enrollment request was not found")
	}
	if request.Status == "approved" {
		record, ok := authority.data.enrollments[request.EnrollmentID]
		if !ok || record.Status != "active" {
			return protocol.EnrollmentGrantV1{}, ErrEnrollmentRevoked
		}
		return record.Grant, nil
	}
	if request.Status == "rejected" {
		return protocol.EnrollmentGrantV1{}, ErrEnrollmentRejected
	}
	if request.Status == "expired" || (request.Status == "pending" && authority.now().UTC().UnixMilli() >= request.ExpiresAtUnixMS) {
		if request.Status == "pending" {
			next := cloneEnrollmentAuthorityData(authority.data)
			expirePendingRequest(&request)
			next.requests[enrollmentRequestID] = request
			if err := authority.commitLocked(next); err != nil {
				return protocol.EnrollmentGrantV1{}, err
			}
		}
		return protocol.EnrollmentGrantV1{}, ErrEnrollmentExpired
	}
	if request.Status != "pending" {
		return protocol.EnrollmentGrantV1{}, ErrEnrollmentConflict
	}
	next := cloneEnrollmentAuthorityData(authority.data)
	grant, err := authority.grantLocked(next, request, admissionProfile, authority.now().UTC().UnixMilli())
	if err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	if err := authority.commitLocked(next); err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	return grant, nil
}

func (authority *EnrollmentAuthority) Reject(commandRequestID, enrollmentRequestID, reason string) error {
	if err := validateEnrollmentHexID("command request ID", commandRequestID, 16); err != nil {
		return err
	}
	if err := validateEnrollmentHexID("enrollment request ID", enrollmentRequestID, 16); err != nil {
		return err
	}
	if reason == "" || len(reason) > protocol.MaxLabelBytes {
		return errors.New("rejection reason is invalid")
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	request, exists := authority.data.requests[enrollmentRequestID]
	if !exists {
		return errors.New("enrollment request was not found")
	}
	if request.Status == "rejected" {
		return nil
	}
	if request.Status == "expired" || (request.Status == "pending" && authority.now().UTC().UnixMilli() >= request.ExpiresAtUnixMS) {
		if request.Status == "pending" {
			next := cloneEnrollmentAuthorityData(authority.data)
			expirePendingRequest(&request)
			next.requests[enrollmentRequestID] = request
			if err := authority.commitLocked(next); err != nil {
				return err
			}
		}
		return ErrEnrollmentExpired
	}
	if request.Status != "pending" {
		return ErrEnrollmentConflict
	}
	next := cloneEnrollmentAuthorityData(authority.data)
	request.Status = "rejected"
	request.Reason = reason
	request.UpdatedAtUnixMS = authority.now().UTC().UnixMilli()
	next.requests[enrollmentRequestID] = request
	return authority.commitLocked(next)
}

func (authority *EnrollmentAuthority) RevokePermit(commandRequestID, permitID, reason string) error {
	if err := validateEnrollmentHexID("command request ID", commandRequestID, 16); err != nil {
		return err
	}
	if err := validateEnrollmentHexID("permit ID", permitID, 16); err != nil {
		return err
	}
	if reason == "" || len(reason) > protocol.MaxLabelBytes {
		return errors.New("permit revocation reason is invalid")
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	record, exists := authority.data.permits[permitID]
	if !exists {
		return errors.New("enrollment permit was not found")
	}
	if record.Status == "revoked" {
		return nil
	}
	if record.Status == "consumed" {
		return ErrPermitConsumed
	}
	next := cloneEnrollmentAuthorityData(authority.data)
	record.Status = "revoked"
	record.Reason = reason
	record.RevokedAtUnixMS = authority.now().UTC().UnixMilli()
	next.permits[permitID] = record
	return authority.commitLocked(next)
}

func (authority *EnrollmentAuthority) RevokeEnrollment(commandRequestID, enrollmentID, reason string) (protocol.NodeID, error) {
	if err := validateEnrollmentHexID("command request ID", commandRequestID, 16); err != nil {
		return 0, err
	}
	if err := validateEnrollmentHexID("enrollment ID", enrollmentID, 16); err != nil {
		return 0, err
	}
	if reason == "" || len(reason) > protocol.MaxLabelBytes {
		return 0, errors.New("enrollment revocation reason is invalid")
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	record, exists := authority.data.enrollments[enrollmentID]
	if !exists {
		return 0, errors.New("enrollment was not found")
	}
	nodeID, _ := parseEnrollmentNodeID(record.Grant.NodeID)
	if record.Status == "revoked" {
		return nodeID, nil
	}
	next := cloneEnrollmentAuthorityData(authority.data)
	record.Status = "revoked"
	record.Reason = reason
	record.RevokedAtUnixMS = authority.now().UTC().UnixMilli()
	next.enrollments[enrollmentID] = record
	if err := authority.commitLocked(next); err != nil {
		return 0, err
	}
	return nodeID, nil
}

func (authority *EnrollmentAuthority) Permits() []EnrollmentPermitRecord {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	result := make([]EnrollmentPermitRecord, 0, len(authority.data.permits))
	for _, record := range authority.data.permits {
		result = append(result, permitRecordFromState(record))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Permit.PermitID < result[j].Permit.PermitID })
	return result
}

func (authority *EnrollmentAuthority) Requests() []EnrollmentRequestRecord {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	now := authority.now().UTC().UnixMilli()
	result := make([]EnrollmentRequestRecord, 0, len(authority.data.requests))
	for _, record := range authority.data.requests {
		if record.Status == "pending" && now >= record.ExpiresAtUnixMS {
			expirePendingRequest(&record)
		}
		result = append(result, requestRecordFromState(record))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RequestID < result[j].RequestID })
	return result
}

func (authority *EnrollmentAuthority) Enrollments() []EnrollmentRecord {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	result := make([]EnrollmentRecord, 0, len(authority.data.enrollments))
	for _, record := range authority.data.enrollments {
		result = append(result, enrollmentRecordFromState(record))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Grant.EnrollmentID < result[j].Grant.EnrollmentID })
	return result
}

func (authority *EnrollmentAuthority) GrantForRequest(requestID string) (protocol.EnrollmentGrantV1, error) {
	authority.mu.Lock()
	defer authority.mu.Unlock()
	request, exists := authority.data.requests[requestID]
	if !exists {
		return protocol.EnrollmentGrantV1{}, errors.New("enrollment request was not found")
	}
	if request.Status == "pending" && authority.now().UTC().UnixMilli() >= request.ExpiresAtUnixMS {
		return protocol.EnrollmentGrantV1{}, ErrEnrollmentExpired
	}
	outcome, err := authority.outcomeForRequestLocked(request)
	if err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	if outcome.Grant == nil {
		return protocol.EnrollmentGrantV1{}, ErrEnrollmentPending
	}
	return *outcome.Grant, nil
}

func DevicePublicKeyFingerprint(publicKey ed25519.PublicKey) string {
	digest := sha256.Sum256(publicKey)
	return hex.EncodeToString(digest[:])
}

func VerifyEnrollmentPermit(publicKey ed25519.PublicKey, permit protocol.EnrollmentPermitV1) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New("authority public key must be Ed25519")
	}
	if err := permit.Validate(); err != nil {
		return fmt.Errorf("invalid enrollment permit: %w", err)
	}
	if permit.AuthorityPublicKey != base64.RawStdEncoding.EncodeToString(publicKey) {
		return errors.New("enrollment permit authority key does not match verifier")
	}
	signature, _ := base64.RawStdEncoding.DecodeString(permit.Signature)
	if !ed25519.Verify(publicKey, enrollmentPermitMessage(permit), signature) {
		return errors.New("invalid enrollment permit signature")
	}
	return nil
}

func VerifyEnrollmentGrant(publicKey ed25519.PublicKey, grant protocol.EnrollmentGrantV1) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New("authority public key must be Ed25519")
	}
	if err := grant.Validate(); err != nil {
		return fmt.Errorf("invalid enrollment grant: %w", err)
	}
	if grant.AuthorityPublicKey != base64.RawStdEncoding.EncodeToString(publicKey) {
		return errors.New("enrollment grant authority key does not match verifier")
	}
	signature, _ := base64.RawStdEncoding.DecodeString(grant.Signature)
	if !ed25519.Verify(publicKey, enrollmentGrantMessage(grant), signature) {
		return errors.New("invalid enrollment grant signature")
	}
	return nil
}

func (authority *EnrollmentAuthority) consumePermitLocked(next enrollmentAuthorityData, submission EnrollmentSubmission, fingerprint string, now int64) (enrollmentAuthorityPermitState, error) {
	permit := *submission.Permit
	if err := VerifyEnrollmentPermit(authority.identity.PublicKey, permit); err != nil {
		return enrollmentAuthorityPermitState{}, err
	}
	record, exists := next.permits[permit.PermitID]
	if !exists || !sameEnrollmentPermit(record.Permit, permit) {
		return enrollmentAuthorityPermitState{}, fmt.Errorf("%w: unknown permit", ErrPermitInvalid)
	}
	if record.Status == "consumed" {
		return enrollmentAuthorityPermitState{}, ErrPermitConsumed
	}
	if record.Status == "revoked" {
		return enrollmentAuthorityPermitState{}, ErrPermitRevoked
	}
	if now < permit.IssuedAtUnixMS || now >= permit.ExpiresAtUnixMS {
		return enrollmentAuthorityPermitState{}, fmt.Errorf("%w: expired or not yet valid", ErrPermitInvalid)
	}
	if permit.DevicePublicKeyFingerprint != fingerprint {
		return enrollmentAuthorityPermitState{}, fmt.Errorf("%w: device key binding mismatch", ErrPermitInvalid)
	}
	target, err := parseEnrollmentNodeID(permit.TargetNodeID)
	if err != nil {
		return enrollmentAuthorityPermitState{}, fmt.Errorf("%w: target binding is invalid", ErrPermitInvalid)
	}
	allowed := target == submission.ParentNodeID
	if !allowed && permit.AllowDescendants && authority.isTargetDescendant != nil {
		allowed = authority.isTargetDescendant(target, submission.ParentNodeID)
	}
	if !allowed {
		return enrollmentAuthorityPermitState{}, fmt.Errorf("%w: target parent binding mismatch", ErrPermitInvalid)
	}
	return record, nil
}

func (authority *EnrollmentAuthority) grantLocked(next enrollmentAuthorityData, request enrollmentAuthorityRequestState, profile string, now int64) (protocol.EnrollmentGrantV1, error) {
	if len(next.enrollments) >= authority.maxEnrollments {
		return protocol.EnrollmentGrantV1{}, errors.New("enrollment record limit reached")
	}
	nodeID, err := authority.allocateNodeIDLocked(next.usedNodeIDs)
	if err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	enrollmentID, err := authority.newUniqueIDLocked(func(id string) bool {
		_, exists := next.enrollments[id]
		return exists
	})
	if err != nil {
		return protocol.EnrollmentGrantV1{}, err
	}
	grant := protocol.EnrollmentGrantV1{
		Version: protocol.SchemaVersionV1, EnrollmentID: enrollmentID,
		AuthorityNodeID:    strconv.FormatUint(uint64(authority.identity.NodeID), 10),
		AuthorityPublicKey: base64.RawStdEncoding.EncodeToString(authority.identity.PublicKey),
		NodeID:             strconv.FormatUint(uint64(nodeID), 10), DevicePublicKey: base64.RawStdEncoding.EncodeToString(request.DevicePublicKey),
		ParentNodeID: strconv.FormatUint(uint64(request.ParentNodeID), 10), ParentPublicKey: base64.RawStdEncoding.EncodeToString(request.ParentPublicKey), AdmissionProfile: profile,
		AuthorityEpoch: authority.generation + 1, IssuedAtUnixMS: now,
	}
	grant.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(authority.identity.PrivateKey, enrollmentGrantMessage(grant)))
	if err := grant.Validate(); err != nil {
		return protocol.EnrollmentGrantV1{}, fmt.Errorf("create enrollment grant: %w", err)
	}
	request.Status = "approved"
	request.AdmissionProfile = profile
	request.EnrollmentID = enrollmentID
	request.Reason = ""
	request.UpdatedAtUnixMS = now
	next.requests[request.RequestID] = request
	next.enrollments[enrollmentID] = enrollmentAuthorityEnrollmentState{
		Grant: grant, DevicePublicKeyFingerprint: request.DevicePublicKeyFingerprint, RequestID: request.RequestID, Status: "active",
	}
	next.usedNodeIDs[nodeID] = struct{}{}
	return grant, nil
}

func (authority *EnrollmentAuthority) outcomeForRequestLocked(request enrollmentAuthorityRequestState) (EnrollmentOutcome, error) {
	switch request.Status {
	case "pending":
		return EnrollmentOutcome{Status: "pending", RequestID: request.RequestID}, nil
	case "rejected":
		return EnrollmentOutcome{Status: "rejected", RequestID: request.RequestID, Reason: request.Reason}, nil
	case "expired":
		return EnrollmentOutcome{Status: "rejected", RequestID: request.RequestID, Reason: "enrollment request expired"}, nil
	case "approved":
		record, ok := authority.data.enrollments[request.EnrollmentID]
		if !ok {
			return EnrollmentOutcome{}, errors.New("approved enrollment request has no enrollment record")
		}
		if record.Status == "revoked" {
			return EnrollmentOutcome{}, ErrEnrollmentRevoked
		}
		grant := record.Grant
		return EnrollmentOutcome{Status: "granted", RequestID: request.RequestID, Grant: &grant}, nil
	default:
		return EnrollmentOutcome{}, errors.New("enrollment request has an invalid status")
	}
}

func (authority *EnrollmentAuthority) allocateNodeIDLocked(used map[protocol.NodeID]struct{}) (protocol.NodeID, error) {
	var raw [8]byte
	for attempt := 0; attempt < 128; attempt++ {
		if _, err := io.ReadFull(authority.random, raw[:]); err != nil {
			return 0, fmt.Errorf("generate Node ID: %w", err)
		}
		candidate := protocol.NodeID(binary.BigEndian.Uint64(raw[:]) & uint64(^uint64(0)>>1))
		if candidate == 0 {
			continue
		}
		if _, exists := used[candidate]; !exists {
			return candidate, nil
		}
	}
	return 0, errors.New("generate Node ID: collision retry limit reached")
}

func (authority *EnrollmentAuthority) newUniqueIDLocked(exists func(string) bool) (string, error) {
	var raw [16]byte
	for attempt := 0; attempt < 128; attempt++ {
		if _, err := io.ReadFull(authority.random, raw[:]); err != nil {
			return "", fmt.Errorf("generate enrollment object ID: %w", err)
		}
		id := hex.EncodeToString(raw[:])
		if id != strings.Repeat("0", len(id)) && !exists(id) {
			return id, nil
		}
	}
	return "", errors.New("generate enrollment object ID: collision retry limit reached")
}

func (authority *EnrollmentAuthority) commitLocked(next enrollmentAuthorityData) error {
	if authority.generation == ^uint64(0) {
		return errors.New("enrollment authority generation exhausted")
	}
	nextGeneration := authority.generation + 1
	if err := authority.saveLocked(next, nextGeneration); err != nil {
		return fmt.Errorf("persist enrollment authority state: %w", err)
	}
	authority.data = next
	authority.generation = nextGeneration
	return nil
}

func (authority *EnrollmentAuthority) saveLocked(data enrollmentAuthorityData, generation uint64) error {
	state := enrollmentAuthorityState{
		Version: enrollmentAuthorityStateVersion, Generation: generation, AuthorityNodeID: authority.identity.NodeID,
		AuthorityPublicKey: append(ed25519.PublicKey(nil), authority.identity.PublicKey...),
	}
	for _, id := range sortedStringKeys(data.permits) {
		state.Permits = append(state.Permits, data.permits[id])
	}
	for _, id := range sortedStringKeys(data.requests) {
		state.Requests = append(state.Requests, data.requests[id])
	}
	for _, id := range sortedStringKeys(data.enrollments) {
		state.Enrollments = append(state.Enrollments, data.enrollments[id])
	}
	for nodeID := range data.usedNodeIDs {
		state.UsedNodeIDs = append(state.UsedNodeIDs, nodeID)
	}
	sort.Slice(state.UsedNodeIDs, func(i, j int) bool { return state.UsedNodeIDs[i] < state.UsedNodeIDs[j] })
	return authority.store.Save("enrollment-authority.json", state)
}

func enrollmentAuthorityDataFromState(state enrollmentAuthorityState, authorityPublicKey ed25519.PublicKey) (enrollmentAuthorityData, error) {
	data := enrollmentAuthorityData{
		permits: make(map[string]enrollmentAuthorityPermitState), requests: make(map[string]enrollmentAuthorityRequestState),
		enrollments: make(map[string]enrollmentAuthorityEnrollmentState), usedNodeIDs: make(map[protocol.NodeID]struct{}),
	}
	for _, nodeID := range state.UsedNodeIDs {
		if err := nodeID.Validate(); err != nil {
			return enrollmentAuthorityData{}, fmt.Errorf("invalid used Node ID: %w", err)
		}
		if _, duplicate := data.usedNodeIDs[nodeID]; duplicate {
			return enrollmentAuthorityData{}, fmt.Errorf("duplicate used Node ID %d", nodeID)
		}
		data.usedNodeIDs[nodeID] = struct{}{}
	}
	if _, ok := data.usedNodeIDs[state.AuthorityNodeID]; !ok {
		return enrollmentAuthorityData{}, errors.New("authority Node ID is missing from used Node IDs")
	}
	for _, record := range state.Permits {
		if err := VerifyEnrollmentPermit(authorityPublicKey, record.Permit); err != nil {
			return enrollmentAuthorityData{}, err
		}
		if err := validateEnrollmentHexID("issue request ID", record.IssueRequestID, 16); err != nil {
			return enrollmentAuthorityData{}, err
		}
		if record.Status != "active" && record.Status != "consumed" && record.Status != "revoked" {
			return enrollmentAuthorityData{}, errors.New("invalid permit status")
		}
		switch record.Status {
		case "active":
			if record.ConsumedByRequestID != "" || record.ConsumedAtUnixMS != 0 || record.RevokedAtUnixMS != 0 || record.Reason != "" {
				return enrollmentAuthorityData{}, errors.New("active permit contains terminal state")
			}
		case "consumed":
			if err := validateEnrollmentHexID("consuming enrollment request ID", record.ConsumedByRequestID, 16); err != nil {
				return enrollmentAuthorityData{}, err
			}
			if record.ConsumedAtUnixMS <= 0 || record.RevokedAtUnixMS != 0 || record.Reason != "" {
				return enrollmentAuthorityData{}, errors.New("consumed permit state is invalid")
			}
		case "revoked":
			if record.RevokedAtUnixMS <= 0 || record.Reason == "" || record.ConsumedByRequestID != "" || record.ConsumedAtUnixMS != 0 {
				return enrollmentAuthorityData{}, errors.New("revoked permit state is invalid")
			}
		}
		if _, duplicate := data.permits[record.Permit.PermitID]; duplicate {
			return enrollmentAuthorityData{}, errors.New("duplicate permit ID")
		}
		data.permits[record.Permit.PermitID] = record
	}
	for _, record := range state.Requests {
		if err := validateEnrollmentRequestState(record); err != nil {
			return enrollmentAuthorityData{}, err
		}
		if _, duplicate := data.requests[record.RequestID]; duplicate {
			return enrollmentAuthorityData{}, errors.New("duplicate enrollment request ID")
		}
		data.requests[record.RequestID] = record
	}
	for _, record := range state.Enrollments {
		if err := VerifyEnrollmentGrant(authorityPublicKey, record.Grant); err != nil {
			return enrollmentAuthorityData{}, err
		}
		if record.Status != "active" && record.Status != "revoked" {
			return enrollmentAuthorityData{}, errors.New("invalid enrollment status")
		}
		if err := validateEnrollmentHexID("enrollment request ID", record.RequestID, 16); err != nil {
			return enrollmentAuthorityData{}, err
		}
		deviceKey, _ := base64.RawStdEncoding.DecodeString(record.Grant.DevicePublicKey)
		if record.DevicePublicKeyFingerprint != DevicePublicKeyFingerprint(ed25519.PublicKey(deviceKey)) {
			return enrollmentAuthorityData{}, errors.New("enrollment device fingerprint does not match its Grant")
		}
		if record.Status == "active" && (record.RevokedAtUnixMS != 0 || record.Reason != "") {
			return enrollmentAuthorityData{}, errors.New("active enrollment contains revocation state")
		}
		if record.Status == "revoked" && (record.RevokedAtUnixMS <= 0 || record.Reason == "") {
			return enrollmentAuthorityData{}, errors.New("revoked enrollment state is invalid")
		}
		if _, duplicate := data.enrollments[record.Grant.EnrollmentID]; duplicate {
			return enrollmentAuthorityData{}, errors.New("duplicate enrollment ID")
		}
		nodeID, _ := parseEnrollmentNodeID(record.Grant.NodeID)
		if _, ok := data.usedNodeIDs[nodeID]; !ok {
			return enrollmentAuthorityData{}, fmt.Errorf("enrollment Node ID %d is missing from used Node IDs", nodeID)
		}
		data.enrollments[record.Grant.EnrollmentID] = record
	}
	for enrollmentID, enrollment := range data.enrollments {
		request, ok := data.requests[enrollment.RequestID]
		if !ok || request.Status != "approved" || request.EnrollmentID != enrollmentID {
			return enrollmentAuthorityData{}, errors.New("Enrollment record does not match an approved request")
		}
	}
	for requestID, request := range data.requests {
		if request.Status != "approved" {
			continue
		}
		enrollment, ok := data.enrollments[request.EnrollmentID]
		if !ok || enrollment.RequestID != requestID || enrollment.DevicePublicKeyFingerprint != request.DevicePublicKeyFingerprint || enrollment.Grant.AdmissionProfile != request.AdmissionProfile || enrollment.Grant.ParentNodeID != strconv.FormatUint(uint64(request.ParentNodeID), 10) || enrollment.Grant.ParentPublicKey != base64.RawStdEncoding.EncodeToString(request.ParentPublicKey) {
			return enrollmentAuthorityData{}, errors.New("approved Enrollment request does not match its Enrollment record")
		}
	}
	return data, nil
}

func validateEnrollmentSubmission(submission EnrollmentSubmission) error {
	if err := validateEnrollmentHexID("enrollment request ID", submission.RequestID, 16); err != nil {
		return err
	}
	if len(submission.DevicePublicKey) != ed25519.PublicKeySize {
		return errors.New("device public key must be Ed25519")
	}
	if err := submission.ParentNodeID.Validate(); err != nil {
		return err
	}
	if len(submission.ParentPublicKey) != ed25519.PublicKeySize {
		return errors.New("parent public key must be Ed25519")
	}
	if submission.TranscriptDigest == [32]byte{} {
		return errors.New("enrollment transcript digest is required")
	}
	return nil
}

func validateEnrollmentRequestState(record enrollmentAuthorityRequestState) error {
	if err := validateEnrollmentHexID("enrollment request ID", record.RequestID, 16); err != nil {
		return err
	}
	if len(record.DevicePublicKey) != ed25519.PublicKeySize || DevicePublicKeyFingerprint(record.DevicePublicKey) != record.DevicePublicKeyFingerprint {
		return errors.New("enrollment request device key or fingerprint is invalid")
	}
	if err := record.ParentNodeID.Validate(); err != nil {
		return err
	}
	if len(record.ParentPublicKey) != ed25519.PublicKeySize {
		return errors.New("enrollment request parent key is invalid")
	}
	if err := validateEnrollmentHexID("transcript digest", record.TranscriptDigest, 32); err != nil {
		return err
	}
	if record.Status != "pending" && record.Status != "approved" && record.Status != "rejected" && record.Status != "expired" {
		return errors.New("invalid enrollment request status")
	}
	if record.CreatedAtUnixMS <= 0 || record.UpdatedAtUnixMS < record.CreatedAtUnixMS || record.ExpiresAtUnixMS <= record.CreatedAtUnixMS {
		return errors.New("invalid enrollment request timestamps")
	}
	switch record.Status {
	case "pending":
		if record.AdmissionProfile != "" || record.EnrollmentID != "" || record.Reason != "" {
			return errors.New("pending enrollment request contains terminal state")
		}
	case "approved":
		if record.AdmissionProfile == "" || len(record.AdmissionProfile) > protocol.MaxIdentifierBytes {
			return errors.New("approved enrollment request profile is invalid")
		}
		if err := validateEnrollmentHexID("enrollment ID", record.EnrollmentID, 16); err != nil {
			return err
		}
		if record.Reason != "" {
			return errors.New("approved enrollment request contains a rejection reason")
		}
	case "rejected", "expired":
		if record.AdmissionProfile != "" || record.EnrollmentID != "" || record.Reason == "" {
			return errors.New("rejected or expired enrollment request state is invalid")
		}
	}
	return nil
}

func enrollmentPermitMessage(permit protocol.EnrollmentPermitV1) []byte {
	unsigned := struct {
		Domain                     string `json:"domain"`
		Version                    int    `json:"version"`
		PermitID                   string `json:"permit_id"`
		AuthorityNodeID            string `json:"authority_node_id"`
		AuthorityPublicKey         string `json:"authority_public_key"`
		DevicePublicKeyFingerprint string `json:"device_public_key_fingerprint"`
		TargetNodeID               string `json:"target_node_id"`
		AllowDescendants           bool   `json:"allow_descendants"`
		AdmissionProfile           string `json:"admission_profile"`
		IssuedAtUnixMS             int64  `json:"issued_at_unix_ms"`
		ExpiresAtUnixMS            int64  `json:"expires_at_unix_ms"`
		AuthorityEpoch             uint64 `json:"authority_epoch"`
	}{"MFHE-PERMIT", permit.Version, permit.PermitID, permit.AuthorityNodeID, permit.AuthorityPublicKey, permit.DevicePublicKeyFingerprint, permit.TargetNodeID, permit.AllowDescendants, permit.AdmissionProfile, permit.IssuedAtUnixMS, permit.ExpiresAtUnixMS, permit.AuthorityEpoch}
	data, _ := json.Marshal(unsigned)
	return data
}

func enrollmentGrantMessage(grant protocol.EnrollmentGrantV1) []byte {
	unsigned := struct {
		Domain             string `json:"domain"`
		Version            int    `json:"version"`
		EnrollmentID       string `json:"enrollment_id"`
		AuthorityNodeID    string `json:"authority_node_id"`
		AuthorityPublicKey string `json:"authority_public_key"`
		NodeID             string `json:"node_id"`
		DevicePublicKey    string `json:"device_public_key"`
		ParentNodeID       string `json:"parent_node_id"`
		ParentPublicKey    string `json:"parent_public_key"`
		AdmissionProfile   string `json:"admission_profile"`
		AuthorityEpoch     uint64 `json:"authority_epoch"`
		IssuedAtUnixMS     int64  `json:"issued_at_unix_ms"`
	}{"MFHE-GRANT", grant.Version, grant.EnrollmentID, grant.AuthorityNodeID, grant.AuthorityPublicKey, grant.NodeID, grant.DevicePublicKey, grant.ParentNodeID, grant.ParentPublicKey, grant.AdmissionProfile, grant.AuthorityEpoch, grant.IssuedAtUnixMS}
	data, _ := json.Marshal(unsigned)
	return data
}

func sameEnrollmentPermit(left, right protocol.EnrollmentPermitV1) bool {
	return bytes.Equal(enrollmentPermitMessage(left), enrollmentPermitMessage(right)) && left.Signature == right.Signature
}

func parseEnrollmentNodeID(value string) (protocol.NodeID, error) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("invalid enrollment Node ID")
	}
	return protocol.NodeID(parsed), nil
}

func validateEnrollmentHexID(field, value string, size int) error {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != size || value != strings.ToLower(value) {
		return fmt.Errorf("%s must be %d lowercase hexadecimal bytes", field, size)
	}
	return nil
}

func cloneEnrollmentAuthorityData(source enrollmentAuthorityData) enrollmentAuthorityData {
	result := enrollmentAuthorityData{
		permits:     make(map[string]enrollmentAuthorityPermitState, len(source.permits)),
		requests:    make(map[string]enrollmentAuthorityRequestState, len(source.requests)),
		enrollments: make(map[string]enrollmentAuthorityEnrollmentState, len(source.enrollments)),
		usedNodeIDs: make(map[protocol.NodeID]struct{}, len(source.usedNodeIDs)),
	}
	for id, record := range source.permits {
		result.permits[id] = record
	}
	for id, record := range source.requests {
		record.DevicePublicKey = append(ed25519.PublicKey(nil), record.DevicePublicKey...)
		record.ParentPublicKey = append(ed25519.PublicKey(nil), record.ParentPublicKey...)
		result.requests[id] = record
	}
	for id, record := range source.enrollments {
		result.enrollments[id] = record
	}
	for id := range source.usedNodeIDs {
		result.usedNodeIDs[id] = struct{}{}
	}
	return result
}

func countPendingRequests(records map[string]enrollmentAuthorityRequestState) int {
	count := 0
	for _, record := range records {
		if record.Status == "pending" {
			count++
		}
	}
	return count
}

func countPendingRequestsForParent(records map[string]enrollmentAuthorityRequestState, parentNodeID protocol.NodeID) int {
	count := 0
	for _, record := range records {
		if record.Status == "pending" && record.ParentNodeID == parentNodeID {
			count++
		}
	}
	return count
}

func expirePendingRequests(records map[string]enrollmentAuthorityRequestState, now int64) bool {
	changed := false
	for requestID, record := range records {
		if record.Status != "pending" || now < record.ExpiresAtUnixMS {
			continue
		}
		expirePendingRequest(&record)
		records[requestID] = record
		changed = true
	}
	return changed
}

func expirePendingRequest(record *enrollmentAuthorityRequestState) {
	record.Status = "expired"
	record.Reason = "approval window expired"
	record.UpdatedAtUnixMS = record.ExpiresAtUnixMS
}

func sortedStringKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func permitRecordFromState(record enrollmentAuthorityPermitState) EnrollmentPermitRecord {
	return EnrollmentPermitRecord{
		Permit: record.Permit, Status: record.Status, IssueRequestID: record.IssueRequestID,
		ConsumedByRequestID: record.ConsumedByRequestID, ConsumedAtUnixMS: record.ConsumedAtUnixMS,
		RevokedAtUnixMS: record.RevokedAtUnixMS, Reason: record.Reason,
	}
}

func requestRecordFromState(record enrollmentAuthorityRequestState) EnrollmentRequestRecord {
	return EnrollmentRequestRecord{
		RequestID: record.RequestID, DevicePublicKey: append(ed25519.PublicKey(nil), record.DevicePublicKey...),
		DevicePublicKeyFingerprint: record.DevicePublicKeyFingerprint, ParentNodeID: record.ParentNodeID,
		ParentPublicKey:  append(ed25519.PublicKey(nil), record.ParentPublicKey...),
		TranscriptDigest: record.TranscriptDigest, Status: record.Status, AdmissionProfile: record.AdmissionProfile,
		EnrollmentID: record.EnrollmentID, CreatedAtUnixMS: record.CreatedAtUnixMS, UpdatedAtUnixMS: record.UpdatedAtUnixMS, Reason: record.Reason,
		ExpiresAtUnixMS: record.ExpiresAtUnixMS,
	}
}

func enrollmentRecordFromState(record enrollmentAuthorityEnrollmentState) EnrollmentRecord {
	return EnrollmentRecord{
		Grant: record.Grant, DevicePublicKeyFingerprint: record.DevicePublicKeyFingerprint,
		RequestID: record.RequestID, Status: record.Status, RevokedAtUnixMS: record.RevokedAtUnixMS, Reason: record.Reason,
	}
}
