package enrollment

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
)

type ServerConfig struct {
	Parent       auth.Identity
	Trust        *auth.TrustStore
	Broker       Broker
	Codec        protocol.EnrollmentCodec
	Now          func() time.Time
	Random       io.Reader
	ChallengeTTL time.Duration
}

type Server struct {
	parent       auth.Identity
	trust        *auth.TrustStore
	broker       Broker
	codec        protocol.EnrollmentCodec
	now          func() time.Time
	random       io.Reader
	challengeTTL time.Duration
}

func NewServer(config ServerConfig) (*Server, error) {
	if err := config.Parent.Validate(); err != nil {
		return nil, fmt.Errorf("enrollment parent identity: %w", err)
	}
	if config.Trust == nil {
		return nil, errors.New("enrollment parent trust store is required")
	}
	if config.Broker == nil {
		return nil, errors.New("enrollment broker is required")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Random == nil {
		config.Random = rand.Reader
	}
	if config.ChallengeTTL <= 0 {
		config.ChallengeTTL = 30 * time.Second
	}
	if config.ChallengeTTL < time.Second || config.ChallengeTTL > 5*time.Minute {
		return nil, errors.New("enrollment challenge TTL must be between one second and five minutes")
	}
	return &Server{
		parent: config.Parent, trust: config.Trust, broker: config.Broker, codec: config.Codec,
		now: config.Now, random: config.Random, challengeTTL: config.ChallengeTTL,
	}, nil
}

func (server *Server) Handle(ctx context.Context, pipe link.Pipe) error {
	if ctx == nil {
		return errors.New("enrollment context is required")
	}
	if pipe == nil {
		return errors.New("enrollment pipe is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	initFrame, err := server.codec.Decode(pipe)
	if err != nil {
		return err
	}
	if initFrame.Type != protocol.EnrollmentMessageClientInit {
		return errors.New("enrollment session must start with ClientInit")
	}
	var init protocol.EnrollmentClientInitV1
	if err := protocol.DecodeJSONPayload(initFrame.Payload, protocol.EnrollmentMaxPayload, &init); err != nil {
		return err
	}
	devicePublicKey, _ := base64.RawStdEncoding.DecodeString(init.DevicePublicKey)
	if init.ExpectedParentNodeID != "" && init.ExpectedParentNodeID != strconv.FormatUint(uint64(server.parent.NodeID), 10) {
		return server.sendFailure(pipe, init.RequestID, protocol.CodeConflict, "connected parent does not match expected parent", false)
	}
	if init.ExpectedParentPublicKey != "" && init.ExpectedParentPublicKey != base64.RawStdEncoding.EncodeToString(server.parent.PublicKey) {
		return server.sendFailure(pipe, init.RequestID, protocol.CodeConflict, "connected parent key does not match expected parent", false)
	}
	authorityNodeID, authorityPublicKey, err := server.broker.AuthorityIdentity(ctx)
	if err != nil {
		return server.sendFailure(pipe, init.RequestID, protocol.CodeInternal, "admission authority is unavailable", true)
	}
	if err := authorityNodeID.Validate(); err != nil || len(authorityPublicKey) != ed25519.PublicKeySize {
		return server.sendFailure(pipe, init.RequestID, protocol.CodeInternal, "admission authority identity is invalid", false)
	}
	var nonce [32]byte
	if _, err := io.ReadFull(server.random, nonce[:]); err != nil {
		return fmt.Errorf("generate enrollment challenge: %w", err)
	}
	unsigned := unsignedChallenge{
		Version: protocol.SchemaVersionV1, RequestID: init.RequestID,
		ParentNodeID:       strconv.FormatUint(uint64(server.parent.NodeID), 10),
		ParentPublicKey:    base64.RawStdEncoding.EncodeToString(server.parent.PublicKey),
		AuthorityNodeID:    strconv.FormatUint(uint64(authorityNodeID), 10),
		AuthorityPublicKey: base64.RawStdEncoding.EncodeToString(authorityPublicKey),
		Nonce:              base64.RawStdEncoding.EncodeToString(nonce[:]),
		ExpiresAtUnixMS:    server.now().UTC().Add(server.challengeTTL).UnixMilli(),
	}
	challenge := protocol.EnrollmentChallengeV1{
		Version: unsigned.Version, RequestID: unsigned.RequestID, ParentNodeID: unsigned.ParentNodeID,
		ParentPublicKey: unsigned.ParentPublicKey, AuthorityNodeID: unsigned.AuthorityNodeID,
		AuthorityPublicKey: unsigned.AuthorityPublicKey, Nonce: unsigned.Nonce, ExpiresAtUnixMS: unsigned.ExpiresAtUnixMS,
		Signature: base64.RawStdEncoding.EncodeToString(signChallenge(server.parent.PrivateKey, unsigned)),
	}
	challengePayload, err := protocol.EncodeJSONPayload(challenge, protocol.EnrollmentMaxPayload)
	if err != nil {
		return err
	}
	if err := server.codec.Encode(pipe, protocol.EnrollmentFrame{Version: protocol.EnrollmentProtocolVersion, Type: protocol.EnrollmentMessageChallenge, Payload: challengePayload}); err != nil {
		return err
	}
	proofFrame, err := server.codec.Decode(pipe)
	if err != nil {
		return err
	}
	if proofFrame.Type != protocol.EnrollmentMessageProof {
		return server.sendFailure(pipe, init.RequestID, protocol.CodeMalformed, "expected ClientProof", false)
	}
	var proof protocol.EnrollmentProofV1
	if err := protocol.DecodeJSONPayload(proofFrame.Payload, protocol.EnrollmentMaxPayload, &proof); err != nil {
		return server.sendFailure(pipe, init.RequestID, protocol.CodeMalformed, "invalid ClientProof", false)
	}
	if proof.RequestID != init.RequestID || proof.Nonce != challenge.Nonce || server.now().UTC().UnixMilli() >= challenge.ExpiresAtUnixMS {
		return server.sendFailure(pipe, init.RequestID, protocol.CodeExpired, "enrollment challenge is mismatched or expired", false)
	}
	digest := transcriptDigest(initFrame.Payload, challengePayload)
	proofSignature, _ := base64.RawStdEncoding.DecodeString(proof.Signature)
	if !ed25519.Verify(ed25519.PublicKey(devicePublicKey), digest[:], proofSignature) {
		return server.sendFailure(pipe, init.RequestID, protocol.CodeUnauthenticated, "device proof signature is invalid", false)
	}
	var permit *protocol.EnrollmentPermitV1
	if init.Permit != "" {
		var decoded protocol.EnrollmentPermitV1
		if err := protocol.DecodeJSONPayload([]byte(init.Permit), protocol.EnrollmentMaxPayload, &decoded); err != nil {
			return server.sendFailure(pipe, init.RequestID, protocol.CodeMalformed, "enrollment Permit is invalid", false)
		}
		permit = &decoded
	}
	outcome, err := server.broker.Submit(ctx, auth.EnrollmentSubmission{
		RequestID: init.RequestID, DevicePublicKey: ed25519.PublicKey(devicePublicKey),
		ParentNodeID: server.parent.NodeID, ParentPublicKey: server.parent.PublicKey,
		Permit: permit, TranscriptDigest: digest,
	})
	if err != nil {
		return server.sendFailure(pipe, init.RequestID, enrollmentErrorCode(err), err.Error(), false)
	}
	if outcome.Status == "granted" {
		if outcome.Grant == nil || outcome.Grant.DevicePublicKey != init.DevicePublicKey || outcome.Grant.ParentNodeID != unsigned.ParentNodeID || outcome.Grant.ParentPublicKey != unsigned.ParentPublicKey || outcome.Grant.AuthorityNodeID != unsigned.AuthorityNodeID || outcome.Grant.AuthorityPublicKey != unsigned.AuthorityPublicKey {
			return server.sendFailure(pipe, init.RequestID, protocol.CodeInternal, "admission authority returned a mismatched Grant", false)
		}
		nodeID, parseErr := strconv.ParseUint(outcome.Grant.NodeID, 10, 64)
		if parseErr != nil || nodeID == 0 {
			return server.sendFailure(pipe, init.RequestID, protocol.CodeInternal, "admission authority returned an invalid Node ID", false)
		}
		if err := server.trust.Add(protocol.NodeID(nodeID), ed25519.PublicKey(devicePublicKey)); err != nil {
			return server.sendFailure(pipe, init.RequestID, protocol.CodeInternal, "failed to persist the granted child trust", true)
		}
	}
	return server.sendOutcome(pipe, outcome)
}

func (server *Server) sendOutcome(pipe link.Pipe, outcome auth.EnrollmentOutcome) error {
	result := protocol.EnrollmentResultV1{Version: protocol.SchemaVersionV1, RequestID: outcome.RequestID, Status: outcome.Status, Grant: outcome.Grant}
	if outcome.Status == "pending" {
		result.RetryAfterMS = 2_000
	}
	if outcome.Status == "rejected" {
		result.Error = &protocol.ErrorPayload{Code: protocol.CodeForbidden, Message: outcome.Reason}
	}
	payload, err := protocol.EncodeJSONPayload(result, protocol.EnrollmentMaxPayload)
	if err != nil {
		return err
	}
	return server.codec.Encode(pipe, protocol.EnrollmentFrame{Version: protocol.EnrollmentProtocolVersion, Type: protocol.EnrollmentMessageResult, Payload: payload})
}

func (server *Server) sendFailure(pipe link.Pipe, requestID string, code protocol.ErrorCode, message string, retryable bool) error {
	result := protocol.EnrollmentResultV1{
		Version: protocol.SchemaVersionV1, RequestID: requestID, Status: "error",
		Error: &protocol.ErrorPayload{Code: code, Message: message, Retryable: retryable},
	}
	payload, err := protocol.EncodeJSONPayload(result, protocol.EnrollmentMaxPayload)
	if err != nil {
		return err
	}
	if err := server.codec.Encode(pipe, protocol.EnrollmentFrame{Version: protocol.EnrollmentProtocolVersion, Type: protocol.EnrollmentMessageResult, Payload: payload}); err != nil {
		return err
	}
	return result.Error
}

func enrollmentErrorCode(err error) protocol.ErrorCode {
	switch {
	case errors.Is(err, auth.ErrPermitConsumed), errors.Is(err, auth.ErrEnrollmentConflict):
		return protocol.CodeConflict
	case errors.Is(err, auth.ErrPermitRevoked), errors.Is(err, auth.ErrEnrollmentRevoked):
		return protocol.CodeGone
	case errors.Is(err, auth.ErrPermitInvalid):
		return protocol.CodeForbidden
	default:
		return protocol.CodeInternal
	}
}
