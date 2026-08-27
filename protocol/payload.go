package protocol

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	SchemaVersionV1        = 1
	MaxIdentifierBytes     = 128
	MaxLabelBytes          = 255
	MaxPathBytes           = 1024
	MaxItems               = 4096
	MaxAttributes          = 64
	MaxAttributeKeyBytes   = 64
	MaxAttributeValueBytes = 1024
)

var ErrInvalidPayload = errors.New("invalid protocol payload")

type ValidatedPayload interface {
	Validate() error
}

func EncodeJSONPayload(value ValidatedPayload, maxBytes int) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: value is required", ErrInvalidPayload)
	}
	if err := value.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode protocol payload: %w", err)
	}
	if err := validatePayloadSize(data, maxBytes); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeJSONPayload(data []byte, maxBytes int, target ValidatedPayload) error {
	if target == nil {
		return fmt.Errorf("%w: target is required", ErrInvalidPayload)
	}
	if err := validatePayloadSize(data, maxBytes); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: decode JSON: %v", ErrInvalidPayload, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: JSON payload contains trailing data", ErrInvalidPayload)
	}
	if err := target.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return nil
}

func validatePayloadSize(data []byte, maxBytes int) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxPayload
	}
	if len(data) == 0 {
		return fmt.Errorf("%w: payload is empty", ErrInvalidPayload)
	}
	if len(data) > maxBytes {
		return fmt.Errorf("%w: got %d, max %d", ErrPayloadTooLarge, len(data), maxBytes)
	}
	return nil
}

func validateVersion(version int) error {
	if version != SchemaVersionV1 {
		return fmt.Errorf("version must be %d", SchemaVersionV1)
	}
	return nil
}

func validateText(field, value string, maxBytes int, required bool) error {
	if required && value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if !utf8.ValidString(value) || len(value) > maxBytes {
		return fmt.Errorf("%s must be valid UTF-8 with at most %d bytes", field, maxBytes)
	}
	return nil
}

func validateNodeIDText(field, value string) error {
	if value == "" || strings.HasPrefix(value, "+") || (len(value) > 1 && strings.HasPrefix(value, "0")) {
		return fmt.Errorf("%s must be a canonical non-zero decimal NodeID", field)
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return fmt.Errorf("%s must be a canonical non-zero decimal NodeID", field)
	}
	return nil
}

func validateHexID(field, value string, bytes int) error {
	if len(value) != bytes*2 {
		return fmt.Errorf("%s must be %d lowercase hexadecimal bytes", field, bytes)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != bytes || value != strings.ToLower(value) {
		return fmt.Errorf("%s must be %d lowercase hexadecimal bytes", field, bytes)
	}
	return nil
}

func validateAttributes(attributes map[string]string) error {
	if len(attributes) > MaxAttributes {
		return fmt.Errorf("attributes exceeds %d entries", MaxAttributes)
	}
	for key, value := range attributes {
		if err := validateText("attribute key", key, MaxAttributeKeyBytes, true); err != nil {
			return err
		}
		if err := validateText("attribute value", value, MaxAttributeValueBytes, false); err != nil {
			return err
		}
	}
	return nil
}

func validateRelativePath(value string) error {
	if err := validateText("path", value, MaxPathBytes, true); err != nil {
		return err
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "\\") || strings.Contains(value, "\\") {
		return errors.New("path must use relative slash-separated segments")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return errors.New("path contains an empty or dot segment")
		}
		if strings.Contains(segment, ":") {
			return errors.New("path segment contains a platform prefix")
		}
	}
	return nil
}
