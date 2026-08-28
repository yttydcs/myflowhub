package protocol

import (
	"encoding/json"
	"fmt"
)

type ErrorCode string

const (
	CodeMalformed       ErrorCode = "malformed"
	CodeUnsupported     ErrorCode = "unsupported"
	CodeUnauthenticated ErrorCode = "unauthenticated"
	CodeForbidden       ErrorCode = "forbidden"
	CodeNotFound        ErrorCode = "not_found"
	CodeConflict        ErrorCode = "conflict"
	CodeStaleEpoch      ErrorCode = "stale_epoch"
	CodeExpired         ErrorCode = "expired"
	CodeOverflow        ErrorCode = "overflow"
	CodeTimeout         ErrorCode = "timeout"
	CodeGone            ErrorCode = "gone"
	CodeGap             ErrorCode = "gap"
	CodeRateLimited     ErrorCode = "rate_limited"
	CodeInternal        ErrorCode = "internal"
)

type ErrorPayload struct {
	Code      ErrorCode         `json:"code"`
	Message   string            `json:"message"`
	Retryable bool              `json:"retryable,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

func (e ErrorPayload) Error() string {
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func EncodeErrorPayload(value ErrorPayload) ([]byte, error) {
	if value.Code == "" || value.Message == "" {
		return nil, fmt.Errorf("error code and message are required")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode wire error: %w", err)
	}
	return data, nil
}

func DecodeErrorPayload(data []byte) (ErrorPayload, error) {
	var value ErrorPayload
	if err := json.Unmarshal(data, &value); err != nil {
		return ErrorPayload{}, fmt.Errorf("decode wire error: %w", err)
	}
	if value.Code == "" || value.Message == "" {
		return ErrorPayload{}, fmt.Errorf("decode wire error: code and message are required")
	}
	return value, nil
}
