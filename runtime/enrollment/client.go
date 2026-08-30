package enrollment

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
)

var ErrTrustConfirmationRequired = errors.New("enrollment requires explicit parent and authority trust confirmation")

type TrustObservation struct {
	ParentNodeID       protocol.NodeID
	ParentPublicKey    ed25519.PublicKey
	AuthorityNodeID    protocol.NodeID
	AuthorityPublicKey ed25519.PublicKey
}

type ClientOptions struct {
	RequestID                  string
	Permit                     *protocol.EnrollmentPermitV1
	ExpectedParentNodeID       protocol.NodeID
	ExpectedParentPublicKey    ed25519.PublicKey
	ExpectedAuthorityNodeID    protocol.NodeID
	ExpectedAuthorityPublicKey ed25519.PublicKey
	AllowTOFU                  bool
	ConfirmTOFU                func(TrustObservation) error
	Codec                      protocol.EnrollmentCodec
	Now                        func() time.Time
}

type ClientResult struct {
	Outcome            protocol.EnrollmentResultV1
	Identity           *auth.Identity
	ParentNodeID       protocol.NodeID
	ParentPublicKey    ed25519.PublicKey
	AuthorityNodeID    protocol.NodeID
	AuthorityPublicKey ed25519.PublicKey
}

func Enroll(ctx context.Context, pipe link.Pipe, device auth.DeviceIdentity, options ClientOptions) (ClientResult, error) {
	if ctx == nil {
		return ClientResult{}, errors.New("enrollment context is required")
	}
	if pipe == nil {
		return ClientResult{}, errors.New("enrollment pipe is required")
	}
	if err := device.Validate(); err != nil {
		return ClientResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ClientResult{}, err
	}
	stopClose := context.AfterFunc(ctx, func() { _ = pipe.Close() })
	defer stopClose()
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.RequestID == "" {
		requestID, err := protocol.NewMessageID()
		if err != nil {
			return ClientResult{}, err
		}
		options.RequestID = requestID.String()
	}
	init := protocol.EnrollmentClientInitV1{
		Version: protocol.SchemaVersionV1, RequestID: options.RequestID,
		DevicePublicKey: base64.RawStdEncoding.EncodeToString(device.PublicKey),
	}
	if options.ExpectedParentNodeID != 0 {
		if err := options.ExpectedParentNodeID.Validate(); err != nil {
			return ClientResult{}, err
		}
		init.ExpectedParentNodeID = strconv.FormatUint(uint64(options.ExpectedParentNodeID), 10)
	}
	if len(options.ExpectedParentPublicKey) > 0 {
		if len(options.ExpectedParentPublicKey) != ed25519.PublicKeySize {
			return ClientResult{}, errors.New("expected parent public key must be Ed25519")
		}
		init.ExpectedParentPublicKey = base64.RawStdEncoding.EncodeToString(options.ExpectedParentPublicKey)
	}
	if options.ExpectedAuthorityNodeID != 0 {
		if err := options.ExpectedAuthorityNodeID.Validate(); err != nil {
			return ClientResult{}, err
		}
	}
	anchoredAuthorityKey := append(ed25519.PublicKey(nil), options.ExpectedAuthorityPublicKey...)
	if len(anchoredAuthorityKey) != 0 && len(anchoredAuthorityKey) != ed25519.PublicKeySize {
		return ClientResult{}, errors.New("expected authority public key must be Ed25519")
	}
	if options.Permit != nil {
		permitKey, err := base64.RawStdEncoding.DecodeString(options.Permit.AuthorityPublicKey)
		if err != nil || len(permitKey) != ed25519.PublicKeySize {
			return ClientResult{}, errors.New("Permit authority public key is invalid")
		}
		if err := auth.VerifyEnrollmentPermit(ed25519.PublicKey(permitKey), *options.Permit); err != nil {
			return ClientResult{}, err
		}
		if options.Permit.DevicePublicKeyFingerprint != auth.DevicePublicKeyFingerprint(device.PublicKey) {
			return ClientResult{}, errors.New("Permit is bound to a different device key")
		}
		if len(anchoredAuthorityKey) > 0 && !bytes.Equal(anchoredAuthorityKey, permitKey) {
			return ClientResult{}, errors.New("Permit authority key conflicts with the configured authority key")
		}
		anchoredAuthorityKey = append(ed25519.PublicKey(nil), permitKey...)
		permitPayload, err := protocol.EncodeJSONPayload(*options.Permit, protocol.EnrollmentMaxPayload)
		if err != nil {
			return ClientResult{}, err
		}
		init.Permit = string(permitPayload)
	}
	initPayload, err := protocol.EncodeJSONPayload(init, protocol.EnrollmentMaxPayload)
	if err != nil {
		return ClientResult{}, err
	}
	if err := options.Codec.Encode(pipe, protocol.EnrollmentFrame{Version: protocol.EnrollmentProtocolVersion, Type: protocol.EnrollmentMessageClientInit, Payload: initPayload}); err != nil {
		return ClientResult{}, enrollmentContextError(ctx, err)
	}
	challengeFrame, err := options.Codec.Decode(pipe)
	if err != nil {
		return ClientResult{}, enrollmentContextError(ctx, err)
	}
	if challengeFrame.Type != protocol.EnrollmentMessageChallenge {
		return ClientResult{}, errors.New("enrollment server did not return a challenge")
	}
	var challenge protocol.EnrollmentChallengeV1
	if err := protocol.DecodeJSONPayload(challengeFrame.Payload, protocol.EnrollmentMaxPayload, &challenge); err != nil {
		return ClientResult{}, err
	}
	if challenge.RequestID != init.RequestID || options.Now().UTC().UnixMilli() >= challenge.ExpiresAtUnixMS {
		return ClientResult{}, errors.New("enrollment challenge is mismatched or expired")
	}
	observation, err := decodeTrustObservation(challenge)
	if err != nil {
		return ClientResult{}, err
	}
	unsigned := unsignedChallenge{
		Version: challenge.Version, RequestID: challenge.RequestID, ParentNodeID: challenge.ParentNodeID,
		ParentPublicKey: challenge.ParentPublicKey, AuthorityNodeID: challenge.AuthorityNodeID,
		AuthorityPublicKey: challenge.AuthorityPublicKey, Nonce: challenge.Nonce, ExpiresAtUnixMS: challenge.ExpiresAtUnixMS,
	}
	challengeSignature, _ := base64.RawStdEncoding.DecodeString(challenge.Signature)
	if !verifyChallenge(observation.ParentPublicKey, unsigned, challengeSignature) {
		return ClientResult{}, errors.New("parent challenge signature is invalid")
	}
	if options.ExpectedParentNodeID != 0 && observation.ParentNodeID != options.ExpectedParentNodeID {
		return ClientResult{}, errors.New("parent Node ID does not match the configured parent")
	}
	if len(options.ExpectedParentPublicKey) > 0 && !bytes.Equal(options.ExpectedParentPublicKey, observation.ParentPublicKey) {
		return ClientResult{}, errors.New("parent public key does not match the configured parent")
	}
	if len(anchoredAuthorityKey) > 0 && !bytes.Equal(anchoredAuthorityKey, observation.AuthorityPublicKey) {
		return ClientResult{}, errors.New("authority public key does not match the trusted authority")
	}
	if options.ExpectedAuthorityNodeID != 0 && options.ExpectedAuthorityNodeID != observation.AuthorityNodeID {
		return ClientResult{}, errors.New("authority Node ID does not match the trusted authority")
	}
	if options.Permit != nil {
		if options.Permit.AuthorityNodeID != challenge.AuthorityNodeID {
			return ClientResult{}, errors.New("Permit authority Node ID does not match the enrollment Authority")
		}
		if !options.Permit.AllowDescendants && options.Permit.TargetNodeID != challenge.ParentNodeID {
			return ClientResult{}, errors.New("Permit target does not match the connected parent")
		}
	}
	// An Authority key authenticates grants, but by itself does not identify the
	// parent at this endpoint. A Permit is also a parent anchor because the
	// Authority verifies its target scope against the authenticated submitter.
	parentAnchored := len(options.ExpectedParentPublicKey) > 0 || options.Permit != nil
	if !parentAnchored {
		if !options.AllowTOFU {
			return ClientResult{}, ErrTrustConfirmationRequired
		}
		if options.ConfirmTOFU != nil {
			if err := options.ConfirmTOFU(observation); err != nil {
				return ClientResult{}, fmt.Errorf("confirm enrollment trust: %w", err)
			}
		}
	}
	digest := transcriptDigest(initPayload, challengeFrame.Payload)
	proof := protocol.EnrollmentProofV1{
		Version: protocol.SchemaVersionV1, RequestID: init.RequestID, Nonce: challenge.Nonce,
		Signature: base64.RawStdEncoding.EncodeToString(ed25519.Sign(device.PrivateKey, digest[:])),
	}
	proofPayload, err := protocol.EncodeJSONPayload(proof, protocol.EnrollmentMaxPayload)
	if err != nil {
		return ClientResult{}, err
	}
	if err := options.Codec.Encode(pipe, protocol.EnrollmentFrame{Version: protocol.EnrollmentProtocolVersion, Type: protocol.EnrollmentMessageProof, Payload: proofPayload}); err != nil {
		return ClientResult{}, enrollmentContextError(ctx, err)
	}
	resultFrame, err := options.Codec.Decode(pipe)
	if err != nil {
		return ClientResult{}, enrollmentContextError(ctx, err)
	}
	if resultFrame.Type != protocol.EnrollmentMessageResult {
		return ClientResult{}, errors.New("enrollment server did not return a result")
	}
	var result protocol.EnrollmentResultV1
	if err := protocol.DecodeJSONPayload(resultFrame.Payload, protocol.EnrollmentMaxPayload, &result); err != nil {
		return ClientResult{}, err
	}
	if result.RequestID != init.RequestID {
		return ClientResult{}, errors.New("enrollment result request ID does not match the session")
	}
	clientResult := ClientResult{
		Outcome: result, ParentNodeID: observation.ParentNodeID, ParentPublicKey: observation.ParentPublicKey,
		AuthorityNodeID: observation.AuthorityNodeID, AuthorityPublicKey: observation.AuthorityPublicKey,
	}
	if result.Status == "error" || result.Status == "rejected" {
		return clientResult, result.Error
	}
	if result.Status == "pending" {
		return clientResult, nil
	}
	grant := result.Grant
	if grant == nil {
		return ClientResult{}, errors.New("granted enrollment result has no Grant")
	}
	if err := auth.VerifyEnrollmentGrant(observation.AuthorityPublicKey, *grant); err != nil {
		return ClientResult{}, err
	}
	if grant.DevicePublicKey != init.DevicePublicKey || grant.ParentNodeID != challenge.ParentNodeID || grant.ParentPublicKey != challenge.ParentPublicKey || grant.AuthorityNodeID != challenge.AuthorityNodeID || grant.AuthorityPublicKey != challenge.AuthorityPublicKey {
		return ClientResult{}, errors.New("enrollment Grant does not match the authenticated session")
	}
	nodeID, err := strconv.ParseUint(grant.NodeID, 10, 64)
	if err != nil || nodeID == 0 {
		return ClientResult{}, errors.New("enrollment Grant contains an invalid Node ID")
	}
	identity, err := device.Enroll(protocol.NodeID(nodeID))
	if err != nil {
		return ClientResult{}, err
	}
	clientResult.Identity = &identity
	return clientResult, nil
}

func enrollmentContextError(ctx context.Context, err error) error {
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}
	return err
}

func decodeTrustObservation(challenge protocol.EnrollmentChallengeV1) (TrustObservation, error) {
	parentNodeID, err := strconv.ParseUint(challenge.ParentNodeID, 10, 64)
	if err != nil || parentNodeID == 0 {
		return TrustObservation{}, errors.New("challenge parent Node ID is invalid")
	}
	authorityNodeID, err := strconv.ParseUint(challenge.AuthorityNodeID, 10, 64)
	if err != nil || authorityNodeID == 0 {
		return TrustObservation{}, errors.New("challenge authority Node ID is invalid")
	}
	parentKey, _ := base64.RawStdEncoding.DecodeString(challenge.ParentPublicKey)
	authorityKey, _ := base64.RawStdEncoding.DecodeString(challenge.AuthorityPublicKey)
	return TrustObservation{
		ParentNodeID: protocol.NodeID(parentNodeID), ParentPublicKey: ed25519.PublicKey(parentKey),
		AuthorityNodeID: protocol.NodeID(authorityNodeID), AuthorityPublicKey: ed25519.PublicKey(authorityKey),
	}, nil
}
