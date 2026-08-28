package bindings

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/yttydcs/myflowhub/protocol"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

type bindingCatalog struct {
	Version   int                         `json:"version"`
	Revision  uint64                      `json:"revision"`
	Resources []bindingResourceDescriptor `json:"resources"`
}

type bindingResourceDescriptor struct {
	ID struct {
		OwnerNodeID string `json:"owner_node_id"`
		Name        string `json:"name"`
	} `json:"id"`
	Type         protocol.ResourceTypeID           `json:"type"`
	TypeVersion  uint32                            `json:"type_version"`
	Capabilities []protocol.CapabilityDescriptorV2 `json:"capabilities"`
	Schemas      []protocol.SchemaDescriptorV2     `json:"schemas,omitempty"`
	Limits       protocol.ResourceLimitsV2         `json:"limits"`
	Presentation protocol.PresentationHintV2       `json:"presentation,omitempty"`
}

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
	Kind              string `json:"kind"`
	OwnerNodeID       string `json:"owner_node_id"`
	ResourceName      string `json:"resource_name"`
	Capability        string `json:"capability"`
	Schema            string `json:"schema,omitempty"`
	Revision          uint64 `json:"revision,omitempty"`
	Sequence          uint64 `json:"sequence,omitempty"`
	PublisherNodeID   string `json:"publisher_node_id,omitempty"`
	PublisherSequence uint64 `json:"publisher_sequence,omitempty"`
	GapFrom           uint64 `json:"gap_from,omitempty"`
	GapTo             uint64 `json:"gap_to,omitempty"`
	Value             []byte `json:"value,omitempty"`
	Reason            string `json:"reason,omitempty"`
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

func catalogJSON(value protocol.ResourceCatalogV2) bindingCatalog {
	result := bindingCatalog{Version: value.Version, Revision: value.Revision, Resources: make([]bindingResourceDescriptor, len(value.Resources))}
	for index, descriptor := range value.Resources {
		converted := bindingResourceDescriptor{
			Type: descriptor.Type, TypeVersion: descriptor.TypeVersion,
			Capabilities: append([]protocol.CapabilityDescriptorV2(nil), descriptor.Capabilities...),
			Schemas:      append([]protocol.SchemaDescriptorV2(nil), descriptor.Schemas...), Limits: descriptor.Limits, Presentation: descriptor.Presentation,
		}
		converted.ID.OwnerNodeID = strconv.FormatUint(uint64(descriptor.ID.Owner), 10)
		converted.ID.Name = descriptor.ID.Name
		result.Resources[index] = converted
	}
	return result
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
