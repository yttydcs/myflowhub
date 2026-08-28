package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/yttydcs/myflowhub/protocol"
)

type routePayload struct {
	Node   protocol.NodeID `json:"node"`
	Parent protocol.NodeID `json:"parent"`
}

type subscribePayload struct {
	Version int   `json:"version"`
	LeaseMS int64 `json:"lease_ms"`
	Queue   int   `json:"queue"`
}

type resourceEventPayload struct {
	Version           int             `json:"version"`
	Snapshot          bool            `json:"snapshot,omitempty"`
	Revision          uint64          `json:"revision,omitempty"`
	Sequence          uint64          `json:"sequence,omitempty"`
	Publisher         protocol.NodeID `json:"publisher,omitempty"`
	PublisherSequence uint64          `json:"publisher_sequence,omitempty"`
	GapFrom           uint64          `json:"gap_from,omitempty"`
	GapTo             uint64          `json:"gap_to,omitempty"`
	Reason            string          `json:"reason,omitempty"`
	Schema            string          `json:"schema,omitempty"`
	Value             []byte          `json:"value,omitempty"`
}

func encodeJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode node payload: %w", err)
	}
	return data, nil
}

func decodeJSON(data []byte, target any) error {
	if len(data) == 0 {
		return errors.New("node payload is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode node payload: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("decode node payload: trailing data")
	}
	return nil
}
