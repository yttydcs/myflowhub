package protocol

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	CurrentVersion       uint16 = 2
	DefaultMaxPayload           = 1 << 20
	MaxResourceNameBytes        = 255
	MaxCapabilityBytes          = 128
	MaxContentTypeBytes         = 127
	MaxSchemaBytes              = 255
)

var (
	ErrInvalidEnvelope = errors.New("invalid envelope")
	ErrPayloadTooLarge = errors.New("payload exceeds configured limit")
)

type NodeID uint64

func (id NodeID) Validate() error {
	if id == 0 {
		return fmt.Errorf("%w: node ID must be non-zero", ErrInvalidEnvelope)
	}
	return nil
}

type MessageID [16]byte

func NewMessageID() (MessageID, error) {
	var id MessageID
	if _, err := rand.Read(id[:]); err != nil {
		return MessageID{}, fmt.Errorf("generate message ID: %w", err)
	}
	return id, nil
}

func MustMessageID() MessageID {
	id, err := NewMessageID()
	if err != nil {
		panic(err)
	}
	return id
}

func ParseMessageID(value string) (MessageID, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != len(MessageID{}) || value != strings.ToLower(value) {
		return MessageID{}, errors.New("message ID must be 16 lowercase hexadecimal bytes")
	}
	var id MessageID
	copy(id[:], decoded)
	if id.IsZero() {
		return MessageID{}, errors.New("message ID must be non-zero")
	}
	return id, nil
}

func (id MessageID) IsZero() bool   { return id == MessageID{} }
func (id MessageID) String() string { return hex.EncodeToString(id[:]) }

type ResourceID struct {
	Owner NodeID `json:"owner"`
	Name  string `json:"name"`
}

func (id ResourceID) Validate() error {
	if err := id.Owner.Validate(); err != nil {
		return fmt.Errorf("resource owner: %w", err)
	}
	if id.Name == "" || len(id.Name) > MaxResourceNameBytes || !utf8.ValidString(id.Name) {
		return fmt.Errorf("%w: resource name must be valid UTF-8 between 1 and %d bytes", ErrInvalidEnvelope, MaxResourceNameBytes)
	}
	if strings.HasPrefix(id.Name, "/") || strings.HasSuffix(id.Name, "/") || strings.Contains(id.Name, "//") {
		return fmt.Errorf("%w: resource name must contain non-empty relative segments", ErrInvalidEnvelope)
	}
	for _, segment := range strings.Split(id.Name, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("%w: resource name cannot contain dot segments", ErrInvalidEnvelope)
		}
	}
	for _, r := range id.Name {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return fmt.Errorf("%w: resource name cannot contain whitespace or control characters", ErrInvalidEnvelope)
		}
	}
	return nil
}

type Phase uint8

const (
	PhaseUnknown Phase = iota
	PhaseRequest
	PhaseControl
	PhaseResponse
	PhaseEvent
)

func (p Phase) Valid() bool { return p >= PhaseRequest && p <= PhaseEvent }

type Operation uint8

const (
	OperationUnknown Operation = iota
	OperationJoin
	OperationJoinAck
	OperationRouteAnnounce
	OperationRouteWithdraw
	OperationSubscribe
	OperationUnsubscribe
	OperationSubscribeAck
	OperationResourceEvent
	OperationResourceGap
	OperationOperate
	OperationOperateResult
	OperationSessionOpen
	OperationSessionOpenResult
	OperationSessionData
	OperationSessionClose
	OperationError
	OperationHeartbeat
)

func (op Operation) Valid() bool { return op >= OperationJoin && op <= OperationHeartbeat }

func (op Operation) RequiresResource() bool {
	return op >= OperationSubscribe && op <= OperationSessionClose
}

func (op Operation) RequiresCorrelation() bool {
	switch op {
	case OperationJoinAck, OperationSubscribeAck, OperationResourceEvent, OperationResourceGap,
		OperationOperateResult, OperationSessionOpenResult, OperationSessionData, OperationSessionClose, OperationError:
		return true
	default:
		return false
	}
}

type Envelope struct {
	Version        uint16
	Phase          Phase
	Operation      Operation
	MessageID      MessageID
	CorrelationID  MessageID
	Source         NodeID
	Principal      NodeID
	Target         NodeID
	Resource       ResourceID
	TopologyEpoch  uint64
	DeadlineUnixMS int64
	ContentType    string
	Schema         string
	Capability     CapabilityID
	Payload        []byte
}

func (e Envelope) Validate(maxPayload int) error {
	if e.Version != CurrentVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrInvalidEnvelope, e.Version)
	}
	if !e.Phase.Valid() || !e.Operation.Valid() {
		return fmt.Errorf("%w: unknown phase %d or operation %d", ErrInvalidEnvelope, e.Phase, e.Operation)
	}
	if !phaseAllows(e.Phase, e.Operation) {
		return fmt.Errorf("%w: phase %d does not allow operation %d", ErrInvalidEnvelope, e.Phase, e.Operation)
	}
	if e.MessageID.IsZero() {
		return fmt.Errorf("%w: message ID is required", ErrInvalidEnvelope)
	}
	if err := e.Source.Validate(); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if e.Principal != 0 {
		if err := e.Principal.Validate(); err != nil {
			return fmt.Errorf("principal: %w", err)
		}
	}
	if err := e.Target.Validate(); err != nil {
		return fmt.Errorf("target: %w", err)
	}
	if e.Operation.RequiresResource() {
		if err := e.Resource.Validate(); err != nil {
			return err
		}
	} else if e.Resource != (ResourceID{}) {
		return fmt.Errorf("%w: operation %d must not carry a resource", ErrInvalidEnvelope, e.Operation)
	}
	if e.Operation.RequiresCorrelation() && e.CorrelationID.IsZero() {
		return fmt.Errorf("%w: correlation ID is required", ErrInvalidEnvelope)
	}
	if e.DeadlineUnixMS < 0 {
		return fmt.Errorf("%w: deadline cannot be negative", ErrInvalidEnvelope)
	}
	if len(e.ContentType) > MaxContentTypeBytes || len(e.Schema) > MaxSchemaBytes || len(e.Capability) > MaxCapabilityBytes {
		return fmt.Errorf("%w: content type, schema, or capability metadata is too long", ErrInvalidEnvelope)
	}
	if e.Operation.RequiresResource() {
		if err := e.Capability.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
		}
	} else if e.Capability != "" {
		return fmt.Errorf("%w: operation %d must not carry a capability", ErrInvalidEnvelope, e.Operation)
	}
	if maxPayload <= 0 {
		maxPayload = DefaultMaxPayload
	}
	if len(e.Payload) > maxPayload {
		return fmt.Errorf("%w: got %d, max %d", ErrPayloadTooLarge, len(e.Payload), maxPayload)
	}
	return nil
}

func (e Envelope) Subject() NodeID {
	if e.Principal != 0 {
		return e.Principal
	}
	return e.Source
}

func phaseAllows(phase Phase, op Operation) bool {
	switch op {
	case OperationJoin:
		return phase == PhaseRequest
	case OperationJoinAck, OperationSubscribeAck, OperationOperateResult, OperationSessionOpenResult, OperationError:
		return phase == PhaseResponse
	case OperationRouteAnnounce, OperationRouteWithdraw, OperationResourceGap, OperationHeartbeat:
		return phase == PhaseEvent
	case OperationResourceEvent:
		return phase == PhaseResponse || phase == PhaseEvent
	case OperationSessionData, OperationSessionClose:
		return phase == PhaseRequest || phase == PhaseControl || phase == PhaseResponse
	case OperationSubscribe, OperationUnsubscribe, OperationOperate, OperationSessionOpen:
		return phase == PhaseRequest || phase == PhaseControl
	default:
		return false
	}
}
