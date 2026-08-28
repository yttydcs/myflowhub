package protocol

import (
	"errors"
	"fmt"
	"time"
)

const (
	SchemaSessionGrantV2 = "mfh.session.grant.v2"
	SchemaSessionDataV2  = "mfh.session.data.v2"
	SchemaSessionCloseV2 = "mfh.session.close.v2"
	MaxSessionChunkBytes = 256 << 10
)

type SessionGrantV2 struct {
	Version         int          `json:"version"`
	SessionID       string       `json:"session_id"`
	Capability      CapabilityID `json:"capability"`
	MaxChunkBytes   int          `json:"max_chunk_bytes"`
	MaxTotalBytes   int64        `json:"max_total_bytes"`
	ExpiresAtUnixMS int64        `json:"expires_at_unix_ms"`
}

func (g SessionGrantV2) Validate() error {
	if g.Version != SchemaVersionV2 {
		return fmt.Errorf("version must be %d", SchemaVersionV2)
	}
	if err := validateHexID("session_id", g.SessionID, len(MessageID{})); err != nil {
		return err
	}
	if err := g.Capability.Validate(); err != nil {
		return err
	}
	if g.MaxChunkBytes <= 0 || g.MaxChunkBytes > MaxSessionChunkBytes {
		return fmt.Errorf("max_chunk_bytes must be between 1 and %d", MaxSessionChunkBytes)
	}
	if g.MaxTotalBytes < 0 || g.ExpiresAtUnixMS <= time.Now().UnixMilli() {
		return errors.New("session grant size or expiry is invalid")
	}
	return nil
}

type SessionDataV2 struct {
	Version  int    `json:"version"`
	Offset   int64  `json:"offset"`
	Checksum string `json:"checksum"`
	Data     []byte `json:"data"`
}

func (d SessionDataV2) Validate() error {
	if d.Version != SchemaVersionV2 {
		return fmt.Errorf("version must be %d", SchemaVersionV2)
	}
	if d.Offset < 0 || len(d.Data) > MaxSessionChunkBytes {
		return errors.New("session data offset or size is invalid")
	}
	if err := validateHexID("checksum", d.Checksum, 32); err != nil {
		return err
	}
	return nil
}

type SessionCloseV2 struct {
	Version int    `json:"version"`
	Commit  bool   `json:"commit"`
	Schema  string `json:"schema,omitempty"`
	Payload []byte `json:"payload,omitempty"`
}

func (c SessionCloseV2) Validate() error {
	if c.Version != SchemaVersionV2 {
		return fmt.Errorf("version must be %d", SchemaVersionV2)
	}
	if err := validateText("session close schema", c.Schema, MaxSchemaBytes, false); err != nil {
		return err
	}
	if len(c.Payload) > DefaultMaxPayload {
		return ErrPayloadTooLarge
	}
	return nil
}
