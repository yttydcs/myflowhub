package bindings

import (
	"encoding/json"
	"fmt"

	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

type bindingConnection struct {
	State          string `json:"state"`
	ParentNodeID   string `json:"parent_node_id,omitempty"`
	Endpoint       string `json:"endpoint,omitempty"`
	Attempt        uint64 `json:"attempt,omitempty"`
	LinkGeneration uint64 `json:"link_generation,omitempty"`
	LastError      string `json:"last_error,omitempty"`
	NextRetryUnix  int64  `json:"next_retry_unix_ms,omitempty"`
	Generation     uint64 `json:"generation,omitempty"`
}

type bindingEvent struct {
	Kind         string `json:"kind"`
	OwnerNodeID  string `json:"owner_node_id"`
	ResourceName string `json:"resource_name"`
	Revision     uint64 `json:"revision,omitempty"`
	Sequence     uint64 `json:"sequence,omitempty"`
	GapFrom      uint64 `json:"gap_from,omitempty"`
	GapTo        uint64 `json:"gap_to,omitempty"`
	Value        []byte `json:"value,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type bindingError struct {
	Code      string            `json:"code,omitempty"`
	Message   string            `json:"message"`
	Retryable bool              `json:"retryable,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

func connectionJSON(value sdk.ConnectionSnapshot) bindingConnection {
	nextRetry := int64(0)
	if !value.NextRetry.IsZero() {
		nextRetry = value.NextRetry.UnixMilli()
	}
	parent := ""
	if value.Parent != 0 {
		parent = fmt.Sprintf("%d", value.Parent)
	}
	return bindingConnection{
		State: string(value.State), ParentNodeID: parent, Endpoint: value.Endpoint, Attempt: value.Attempt,
		LinkGeneration: value.LinkGeneration, LastError: value.LastError, NextRetryUnix: nextRetry, Generation: value.Generation,
	}
}

func bindingErrorJSON(err error) string {
	value := bindingError{Message: err.Error()}
	if sdkError, ok := err.(*sdk.Error); ok {
		value.Code = string(sdkError.Code)
		value.Retryable = sdkError.Retryable
		value.Details = sdkError.Details
	}
	data, encodeErr := json.Marshal(value)
	if encodeErr != nil {
		return `{"message":"encode binding error"}`
	}
	return string(data)
}

func callListenerEvent(listener Listener, payload string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("binding event listener panic: %v", recovered)
		}
	}()
	listener.OnEvent(payload)
	return nil
}

func callListenerError(listener Listener, payload string) {
	defer func() { _ = recover() }()
	listener.OnError(payload)
}
