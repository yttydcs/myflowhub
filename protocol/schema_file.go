package protocol

import (
	"errors"
	"fmt"
)

const (
	SchemaFileOfferV1     = "mfh.file.offer.v1"
	SchemaFileChunkV1     = "mfh.file.chunk.v1"
	SchemaFileCompleteV1  = "mfh.file.complete.v1"
	SchemaFileCancelV1    = "mfh.file.cancel.v1"
	SchemaFileProgressV1  = "mfh.file.progress.v1"
	MaxFileChunkBytes     = 64 << 10
	SchemaFileTransfersV1 = "mfh.file.transfers.v1"
	BuiltinFileTransfers  = "file/transfers"
	BuiltinFileProgress   = "file/progress"
	BuiltinFileOffer      = "file/offer"
	BuiltinFileChunk      = "file/chunk"
	BuiltinFileComplete   = "file/complete"
	BuiltinFileCancel     = "file/cancel"
)

type FileOfferV1 struct {
	Version         int    `json:"version"`
	TransferID      string `json:"transfer_id"`
	Path            string `json:"path"`
	Size            int64  `json:"size"`
	SHA256          string `json:"sha256"`
	ChunkSize       int    `json:"chunk_size"`
	ContentType     string `json:"content_type,omitempty"`
	ExpiresAtUnixMS int64  `json:"expires_at_unix_ms"`
}

func (o FileOfferV1) Validate() error {
	if err := validateVersion(o.Version); err != nil {
		return err
	}
	if err := validateHexID("transfer_id", o.TransferID, 16); err != nil {
		return err
	}
	if err := validateRelativePath(o.Path); err != nil {
		return err
	}
	if o.Size < 0 || o.ChunkSize <= 0 || o.ChunkSize > MaxFileChunkBytes || o.ExpiresAtUnixMS <= 0 {
		return errors.New("file size, chunk size, or expiry is invalid")
	}
	if err := validateHexID("sha256", o.SHA256, 32); err != nil {
		return err
	}
	return validateText("content_type", o.ContentType, MaxContentTypeBytes, false)
}

type FileChunkV1 struct {
	Version    int    `json:"version"`
	TransferID string `json:"transfer_id"`
	Offset     int64  `json:"offset"`
	Data       []byte `json:"data"`
	SHA256     string `json:"sha256"`
}

func (c FileChunkV1) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}
	if err := validateHexID("transfer_id", c.TransferID, 16); err != nil {
		return err
	}
	if c.Offset < 0 || len(c.Data) == 0 || len(c.Data) > MaxFileChunkBytes {
		return fmt.Errorf("chunk offset/data must be non-negative and contain 1..%d bytes", MaxFileChunkBytes)
	}
	return validateHexID("sha256", c.SHA256, 32)
}

type FileCompleteV1 struct {
	Version    int    `json:"version"`
	TransferID string `json:"transfer_id"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
}

func (c FileCompleteV1) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}
	if err := validateHexID("transfer_id", c.TransferID, 16); err != nil {
		return err
	}
	if c.Size < 0 {
		return errors.New("file size must not be negative")
	}
	return validateHexID("sha256", c.SHA256, 32)
}

type FileCancelV1 struct {
	Version    int    `json:"version"`
	TransferID string `json:"transfer_id"`
	Reason     string `json:"reason"`
}

func (c FileCancelV1) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}
	if err := validateHexID("transfer_id", c.TransferID, 16); err != nil {
		return err
	}
	return validateText("reason", c.Reason, MaxLabelBytes, true)
}

type FileProgressV1 struct {
	Version       int    `json:"version"`
	TransferID    string `json:"transfer_id"`
	ReceivedBytes int64  `json:"received_bytes"`
	TotalBytes    int64  `json:"total_bytes"`
	State         string `json:"state"`
	Error         string `json:"error,omitempty"`
}

func (p FileProgressV1) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	if err := validateHexID("transfer_id", p.TransferID, 16); err != nil {
		return err
	}
	if p.ReceivedBytes < 0 || p.TotalBytes < 0 || p.ReceivedBytes > p.TotalBytes {
		return errors.New("file progress byte counts are invalid")
	}
	if p.State != "offered" && p.State != "receiving" && p.State != "completed" && p.State != "cancelled" && p.State != "failed" {
		return errors.New("file progress state is invalid")
	}
	return validateText("error", p.Error, 2048, false)
}

type FileTransferSummaryV1 struct {
	TransferID      string `json:"transfer_id"`
	OwnerNodeID     string `json:"owner_node_id"`
	Path            string `json:"path"`
	ReceivedBytes   int64  `json:"received_bytes"`
	TotalBytes      int64  `json:"total_bytes"`
	State           string `json:"state"`
	ExpiresAtUnixMS int64  `json:"expires_at_unix_ms"`
	Error           string `json:"error,omitempty"`
}

func (s FileTransferSummaryV1) Validate() error {
	if err := validateHexID("transfer_id", s.TransferID, 16); err != nil {
		return err
	}
	if err := validateNodeIDText("owner_node_id", s.OwnerNodeID); err != nil {
		return err
	}
	if err := validateRelativePath(s.Path); err != nil {
		return err
	}
	if s.ReceivedBytes < 0 || s.TotalBytes < 0 || s.ReceivedBytes > s.TotalBytes || s.ExpiresAtUnixMS <= 0 {
		return errors.New("file transfer summary counters or expiry are invalid")
	}
	if s.State != "offered" && s.State != "receiving" && s.State != "completed" && s.State != "cancelled" && s.State != "failed" {
		return errors.New("file transfer summary state is invalid")
	}
	return validateText("error", s.Error, 2048, false)
}

type FileTransfersV1 struct {
	Version   int                     `json:"version"`
	Revision  uint64                  `json:"revision"`
	Transfers []FileTransferSummaryV1 `json:"transfers"`
}

func (t FileTransfersV1) Validate() error {
	if err := validateVersion(t.Version); err != nil {
		return err
	}
	if t.Revision == 0 || len(t.Transfers) > MaxItems {
		return fmt.Errorf("file transfers revision must be non-zero and contain at most %d entries", MaxItems)
	}
	seen := make(map[string]struct{}, len(t.Transfers))
	for index, transfer := range t.Transfers {
		if err := transfer.Validate(); err != nil {
			return fmt.Errorf("transfers[%d]: %w", index, err)
		}
		if _, exists := seen[transfer.TransferID]; exists {
			return fmt.Errorf("duplicate transfer %s", transfer.TransferID)
		}
		seen[transfer.TransferID] = struct{}{}
		if index > 0 && t.Transfers[index-1].TransferID >= transfer.TransferID {
			return errors.New("file transfers must be strictly sorted by transfer_id")
		}
	}
	return nil
}
