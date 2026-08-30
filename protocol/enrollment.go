package protocol

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	EnrollmentProtocolVersion uint16 = 1
	EnrollmentMaxPayload             = 64 << 10
	enrollmentHeaderSize             = 12
	EnrollmentMagic                  = "MFHE"

	SchemaEnrollmentClientInitV1 = "mfh.enrollment.client-init.v1"
	SchemaEnrollmentChallengeV1  = "mfh.enrollment.challenge.v1"
	SchemaEnrollmentProofV1      = "mfh.enrollment.proof.v1"
	SchemaEnrollmentPermitV1     = "mfh.enrollment.permit.v1"
	SchemaEnrollmentGrantV1      = "mfh.enrollment.grant.v1"
	SchemaEnrollmentResultV1     = "mfh.enrollment.result.v1"
)

type EnrollmentMessageType uint8

const (
	EnrollmentMessageUnknown EnrollmentMessageType = iota
	EnrollmentMessageClientInit
	EnrollmentMessageChallenge
	EnrollmentMessageProof
	EnrollmentMessageResult
)

func (messageType EnrollmentMessageType) Valid() bool {
	return messageType >= EnrollmentMessageClientInit && messageType <= EnrollmentMessageResult
}

type EnrollmentFrame struct {
	Version uint16
	Type    EnrollmentMessageType
	Payload []byte
}

func (frame EnrollmentFrame) Validate(maxPayload int) error {
	if frame.Version != EnrollmentProtocolVersion {
		return fmt.Errorf("unsupported enrollment version %d", frame.Version)
	}
	if !frame.Type.Valid() {
		return fmt.Errorf("unknown enrollment message type %d", frame.Type)
	}
	if maxPayload <= 0 {
		maxPayload = EnrollmentMaxPayload
	}
	if len(frame.Payload) == 0 {
		return errors.New("enrollment payload is required")
	}
	if len(frame.Payload) > maxPayload {
		return fmt.Errorf("%w: got %d, max %d", ErrPayloadTooLarge, len(frame.Payload), maxPayload)
	}
	return nil
}

type EnrollmentCodec struct {
	MaxPayload int
}

func (codec EnrollmentCodec) payloadLimit() int {
	if codec.MaxPayload <= 0 {
		return EnrollmentMaxPayload
	}
	return codec.MaxPayload
}

func (codec EnrollmentCodec) Encode(writer io.Writer, frame EnrollmentFrame) error {
	if writer == nil {
		return errors.New("encode enrollment frame: nil writer")
	}
	if err := frame.Validate(codec.payloadLimit()); err != nil {
		return fmt.Errorf("encode enrollment frame: %w", err)
	}
	if uint64(len(frame.Payload)) > math.MaxUint32 {
		return fmt.Errorf("encode enrollment frame: %w: wire length exceeds uint32", ErrPayloadTooLarge)
	}
	header := make([]byte, enrollmentHeaderSize)
	copy(header[:4], EnrollmentMagic)
	binary.BigEndian.PutUint16(header[4:6], frame.Version)
	header[6] = byte(frame.Type)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(frame.Payload)))
	if err := writeAll(writer, header); err != nil {
		return fmt.Errorf("encode enrollment frame header: %w", err)
	}
	if err := writeAll(writer, frame.Payload); err != nil {
		return fmt.Errorf("encode enrollment frame payload: %w", err)
	}
	return nil
}

func (codec EnrollmentCodec) Decode(reader io.Reader) (EnrollmentFrame, error) {
	if reader == nil {
		return EnrollmentFrame{}, errors.New("decode enrollment frame: nil reader")
	}
	header := make([]byte, enrollmentHeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return EnrollmentFrame{}, fmt.Errorf("decode enrollment frame header: %w", err)
	}
	if string(header[:4]) != EnrollmentMagic {
		return EnrollmentFrame{}, errors.New("decode enrollment frame: invalid magic")
	}
	if header[7] != 0 {
		return EnrollmentFrame{}, errors.New("decode enrollment frame: non-zero reserved flags")
	}
	payloadLength := binary.BigEndian.Uint32(header[8:12])
	if uint64(payloadLength) > uint64(codec.payloadLimit()) {
		return EnrollmentFrame{}, fmt.Errorf("decode enrollment frame: %w: got %d, max %d", ErrPayloadTooLarge, payloadLength, codec.payloadLimit())
	}
	frame := EnrollmentFrame{
		Version: binary.BigEndian.Uint16(header[4:6]),
		Type:    EnrollmentMessageType(header[6]),
		Payload: make([]byte, int(payloadLength)),
	}
	if _, err := io.ReadFull(reader, frame.Payload); err != nil {
		return EnrollmentFrame{}, fmt.Errorf("decode enrollment frame payload: %w", err)
	}
	if err := frame.Validate(codec.payloadLimit()); err != nil {
		return EnrollmentFrame{}, fmt.Errorf("decode enrollment frame: %w", err)
	}
	return frame, nil
}

type EnrollmentClientInitV1 struct {
	Version                 int    `json:"version"`
	RequestID               string `json:"request_id"`
	DevicePublicKey         string `json:"device_public_key"`
	Permit                  string `json:"permit,omitempty"`
	ExpectedParentNodeID    string `json:"expected_parent_node_id,omitempty"`
	ExpectedParentPublicKey string `json:"expected_parent_public_key,omitempty"`
}

func (message EnrollmentClientInitV1) Validate() error {
	if err := validateVersion(message.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", message.RequestID, 16); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("device_public_key", message.DevicePublicKey, true); err != nil {
		return err
	}
	if err := validateText("permit", message.Permit, EnrollmentMaxPayload/2, false); err != nil {
		return err
	}
	if message.ExpectedParentNodeID != "" {
		if err := validateNodeIDText("expected_parent_node_id", message.ExpectedParentNodeID); err != nil {
			return err
		}
	}
	return validateEd25519PublicKey("expected_parent_public_key", message.ExpectedParentPublicKey, false)
}

type EnrollmentChallengeV1 struct {
	Version            int    `json:"version"`
	RequestID          string `json:"request_id"`
	ParentNodeID       string `json:"parent_node_id"`
	ParentPublicKey    string `json:"parent_public_key"`
	AuthorityNodeID    string `json:"authority_node_id"`
	AuthorityPublicKey string `json:"authority_public_key,omitempty"`
	Nonce              string `json:"nonce"`
	ExpiresAtUnixMS    int64  `json:"expires_at_unix_ms"`
	Signature          string `json:"signature"`
}

func (message EnrollmentChallengeV1) Validate() error {
	if err := validateVersion(message.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", message.RequestID, 16); err != nil {
		return err
	}
	if err := validateNodeIDText("parent_node_id", message.ParentNodeID); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("parent_public_key", message.ParentPublicKey, true); err != nil {
		return err
	}
	if err := validateNodeIDText("authority_node_id", message.AuthorityNodeID); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("authority_public_key", message.AuthorityPublicKey, true); err != nil {
		return err
	}
	if err := validateRawBase64("nonce", message.Nonce, 32, true); err != nil {
		return err
	}
	if message.ExpiresAtUnixMS <= 0 {
		return errors.New("expires_at_unix_ms must be positive")
	}
	return validateRawBase64("signature", message.Signature, 64, true)
}

type EnrollmentProofV1 struct {
	Version   int    `json:"version"`
	RequestID string `json:"request_id"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

func (message EnrollmentProofV1) Validate() error {
	if err := validateVersion(message.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", message.RequestID, 16); err != nil {
		return err
	}
	if err := validateRawBase64("nonce", message.Nonce, 32, true); err != nil {
		return err
	}
	return validateRawBase64("signature", message.Signature, 64, true)
}

type EnrollmentPermitV1 struct {
	Version                    int    `json:"version"`
	PermitID                   string `json:"permit_id"`
	AuthorityNodeID            string `json:"authority_node_id"`
	AuthorityPublicKey         string `json:"authority_public_key"`
	DevicePublicKeyFingerprint string `json:"device_public_key_fingerprint"`
	TargetNodeID               string `json:"target_node_id"`
	AllowDescendants           bool   `json:"allow_descendants,omitempty"`
	AdmissionProfile           string `json:"admission_profile"`
	IssuedAtUnixMS             int64  `json:"issued_at_unix_ms"`
	ExpiresAtUnixMS            int64  `json:"expires_at_unix_ms"`
	AuthorityEpoch             uint64 `json:"authority_epoch"`
	Signature                  string `json:"signature"`
}

func (permit EnrollmentPermitV1) Validate() error {
	if err := validateVersion(permit.Version); err != nil {
		return err
	}
	if err := validateHexID("permit_id", permit.PermitID, 16); err != nil {
		return err
	}
	if err := validateNodeIDText("authority_node_id", permit.AuthorityNodeID); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("authority_public_key", permit.AuthorityPublicKey, true); err != nil {
		return err
	}
	if err := validateHexID("device_public_key_fingerprint", permit.DevicePublicKeyFingerprint, 32); err != nil {
		return err
	}
	if err := validateNodeIDText("target_node_id", permit.TargetNodeID); err != nil {
		return err
	}
	if err := validateText("admission_profile", permit.AdmissionProfile, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if permit.IssuedAtUnixMS <= 0 || permit.ExpiresAtUnixMS <= permit.IssuedAtUnixMS {
		return errors.New("permit time range is invalid")
	}
	if permit.AuthorityEpoch == 0 {
		return errors.New("authority_epoch must be non-zero")
	}
	return validateRawBase64("signature", permit.Signature, 64, true)
}

type EnrollmentGrantV1 struct {
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
	Signature          string `json:"signature"`
}

func (grant EnrollmentGrantV1) Validate() error {
	if err := validateVersion(grant.Version); err != nil {
		return err
	}
	if err := validateHexID("enrollment_id", grant.EnrollmentID, 16); err != nil {
		return err
	}
	if err := validateNodeIDText("authority_node_id", grant.AuthorityNodeID); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("authority_public_key", grant.AuthorityPublicKey, true); err != nil {
		return err
	}
	if err := validateNodeIDText("node_id", grant.NodeID); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("device_public_key", grant.DevicePublicKey, true); err != nil {
		return err
	}
	if err := validateNodeIDText("parent_node_id", grant.ParentNodeID); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("parent_public_key", grant.ParentPublicKey, true); err != nil {
		return err
	}
	if err := validateText("admission_profile", grant.AdmissionProfile, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if grant.AuthorityEpoch == 0 || grant.IssuedAtUnixMS <= 0 {
		return errors.New("grant authority_epoch and issued_at_unix_ms must be positive")
	}
	return validateRawBase64("signature", grant.Signature, 64, true)
}

type EnrollmentResultV1 struct {
	Version      int                `json:"version"`
	RequestID    string             `json:"request_id"`
	Status       string             `json:"status"`
	Grant        *EnrollmentGrantV1 `json:"grant,omitempty"`
	Error        *ErrorPayload      `json:"error,omitempty"`
	RetryAfterMS int64              `json:"retry_after_ms,omitempty"`
}

func (result EnrollmentResultV1) Validate() error {
	if err := validateVersion(result.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", result.RequestID, 16); err != nil {
		return err
	}
	switch result.Status {
	case "granted":
		if result.Grant == nil || result.Error != nil {
			return errors.New("granted result requires grant and no error")
		}
		if err := result.Grant.Validate(); err != nil {
			return fmt.Errorf("grant: %w", err)
		}
	case "pending":
		if result.Grant != nil || result.Error != nil || result.RetryAfterMS < 0 {
			return errors.New("pending result cannot contain grant or error")
		}
	case "rejected", "error":
		if result.Grant != nil || result.Error == nil || result.Error.Code == "" || result.Error.Message == "" {
			return errors.New("rejected or error result requires an error and no grant")
		}
	default:
		return errors.New("enrollment result status is invalid")
	}
	return nil
}

func validateEd25519PublicKey(field, value string, required bool) error {
	return validateRawBase64(field, value, 32, required)
}

func validateRawBase64(field, value string, expectedBytes int, required bool) error {
	if value == "" && !required {
		return nil
	}
	decoded, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(decoded) != expectedBytes {
		return fmt.Errorf("%s must be a raw-base64 value containing %d bytes", field, expectedBytes)
	}
	return nil
}
