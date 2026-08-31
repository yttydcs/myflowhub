package protocol

import (
	"encoding/base64"
	"errors"
	"fmt"
	"unicode/utf8"
)

const (
	SchemaFilesystemReadRequestV1 = "mfh.filesystem.read-request.v1"
	SchemaFilesystemContentV1     = "mfh.filesystem.content.v1"

	FilesystemEncodingUTF8   = "utf-8"
	FilesystemEncodingBase64 = "base64"

	// MaxFilesystemReadBytes leaves enough room for JSON escaping in the
	// one-megabyte control payload. Large downloads belong to a data session.
	MaxFilesystemReadBytes     = 128 << 10
	MaxFilesystemRevisionBytes = MaxIdentifierBytes
)

// FilesystemReadRequestV1 requests one complete, bounded regular file. Key is
// opaque outside the containing filesystem Collection.
type FilesystemReadRequestV1 struct {
	Version          int    `json:"version"`
	Key              string `json:"key"`
	MaxBytes         int    `json:"max_bytes"`
	ExpectedRevision string `json:"expected_revision,omitempty"`
}

func (r FilesystemReadRequestV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if err := validateText("filesystem member key", r.Key, MaxCollectionMemberKeyBytes, true); err != nil {
		return err
	}
	if r.MaxBytes <= 0 || r.MaxBytes > MaxFilesystemReadBytes {
		return fmt.Errorf("filesystem max_bytes must be between 1 and %d", MaxFilesystemReadBytes)
	}
	return validateText("filesystem expected_revision", r.ExpectedRevision, MaxFilesystemRevisionBytes, false)
}

// FilesystemContentV1 contains a complete bounded file and stable metadata for
// stale-read feedback. It intentionally contains no physical root or host path.
type FilesystemContentV1 struct {
	Version        int    `json:"version"`
	Key            string `json:"key"`
	ContentType    string `json:"content_type"`
	Encoding       string `json:"encoding"`
	Data           string `json:"data"`
	Size           int64  `json:"size"`
	ModifiedUnixMS int64  `json:"modified_unix_ms"`
	Revision       string `json:"revision"`
}

func (c FilesystemContentV1) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}
	if err := validateText("filesystem member key", c.Key, MaxCollectionMemberKeyBytes, true); err != nil {
		return err
	}
	if err := validateText("filesystem content_type", c.ContentType, MaxContentTypeBytes, true); err != nil {
		return err
	}
	if err := validateText("filesystem revision", c.Revision, MaxFilesystemRevisionBytes, true); err != nil {
		return err
	}
	if c.Size < 0 || c.Size > MaxFilesystemReadBytes {
		return fmt.Errorf("filesystem size must be between 0 and %d", MaxFilesystemReadBytes)
	}
	if c.ModifiedUnixMS < 0 {
		return errors.New("filesystem modified_unix_ms must be non-negative")
	}
	switch c.Encoding {
	case FilesystemEncodingUTF8:
		if !utf8.ValidString(c.Data) {
			return errors.New("filesystem utf-8 data is invalid")
		}
		if int64(len(c.Data)) != c.Size {
			return errors.New("filesystem utf-8 data length does not match size")
		}
	case FilesystemEncodingBase64:
		decoded, err := base64.StdEncoding.Strict().DecodeString(c.Data)
		if err != nil {
			return errors.New("filesystem base64 data is invalid")
		}
		if int64(len(decoded)) != c.Size {
			return errors.New("filesystem base64 data length does not match size")
		}
	default:
		return fmt.Errorf("filesystem encoding must be %q or %q", FilesystemEncodingUTF8, FilesystemEncodingBase64)
	}
	return nil
}
