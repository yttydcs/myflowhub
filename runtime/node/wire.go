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
	Node protocol.NodeID `json:"node"`
}

type subscribePayload struct {
	LeaseMS int64 `json:"lease_ms"`
	Queue   int   `json:"queue"`
}

type variablePayload struct {
	Revision uint64 `json:"revision"`
	Value    []byte `json:"value"`
}

type streamPayload struct {
	Sequence uint64 `json:"sequence,omitempty"`
	GapFrom  uint64 `json:"gap_from,omitempty"`
	GapTo    uint64 `json:"gap_to,omitempty"`
	Value    []byte `json:"value,omitempty"`
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
