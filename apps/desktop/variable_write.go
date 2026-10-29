package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

type variableWriteClient interface {
	SnapshotJSON(int64, string, int64) (string, error)
	OperateJSON(int64, string, string, string, string, int64) (string, error)
}

type VariableWriteAttempt struct {
	Sent             bool   `json:"sent"`
	ExpectedRevision string `json:"expected_revision,omitempty"`
	ResponseJSON     string `json:"response_json,omitempty"`
	Error            string `json:"error,omitempty"`
}

// Read and dispatch share one native call, so WebView round trips cannot age
// the refreshed CAS revision. There is still exactly one conditional write.
func (a *App) WriteVariableFromSnapshot(ownerNodeID, name, baseJSON, requestJSON string) VariableWriteAttempt {
	owner, err := parsePositiveInt64(ownerNodeID, "owner_node_id")
	if err != nil {
		return VariableWriteAttempt{Error: err.Error()}
	}
	client, err := a.currentClient()
	if err != nil {
		return VariableWriteAttempt{Error: err.Error()}
	}
	attempt := writeVariableFromSnapshot(client, owner, name, baseJSON, requestJSON, time.Now)
	if attempt.Error != "" {
		a.appendLog("error", "variable write: "+name+": "+attempt.Error)
	} else {
		a.appendLog("info", "resource operation completed: "+name+"/write")
	}
	return attempt
}

func writeVariableFromSnapshot(client variableWriteClient, owner int64, name, baseJSON, requestJSON string, now func() time.Time) (attempt VariableWriteAttempt) {
	deadline := now().Add(5 * time.Second)
	fail := func(err error) VariableWriteAttempt { attempt.Error = err.Error(); return attempt }
	if owner <= 0 {
		return fail(errors.New("owner_node_id must be positive"))
	}
	if err := (protocol.ResourceID{Owner: protocol.NodeID(owner), Name: name}).Validate(); err != nil {
		return fail(err)
	}
	var base protocol.VariableValueV1
	if err := protocol.DecodeJSONPayload([]byte(baseJSON), protocol.DefaultMaxPayload, &base); err != nil {
		return fail(fmt.Errorf("invalid base snapshot: %w", err))
	}
	var request protocol.VariableWriteV3
	if err := protocol.DecodeJSONPayload([]byte(requestJSON), protocol.DefaultMaxPayload, &request); err != nil {
		return fail(fmt.Errorf("invalid write request: %w", err))
	}
	if base.Definition == nil || base.Status == "unavailable" || request.ExpectedInstanceID != base.InstanceID || request.ExpectedRevision != base.Revision {
		return fail(errors.New("conflict: write does not match its available base snapshot"))
	}
	if base.Definition.ReadOnly {
		return fail(errors.New("unsupported: variable definition is read-only"))
	}
	if err := protocol.ValidateVariableValue(*base.Definition, request.Value); err != nil {
		return fail(fmt.Errorf("invalid draft value: %w", err))
	}
	remaining := func() int64 { return deadline.Sub(now()).Milliseconds() }
	budget := remaining()
	if budget <= 0 {
		return fail(errors.New("write preparation deadline exceeded"))
	}
	raw, err := client.SnapshotJSON(owner, name, budget)
	if err != nil {
		return fail(err)
	}
	var current protocol.VariableValueV1
	if err := protocol.DecodeJSONPayload([]byte(raw), protocol.DefaultMaxPayload, &current); err != nil {
		return fail(fmt.Errorf("invalid fresh snapshot: %w", err))
	}
	baseRevision, _ := protocol.ParseVariableRevision(base.Revision)
	currentRevision, _ := protocol.ParseVariableRevision(current.Revision)
	if current.InstanceID != base.InstanceID || currentRevision < baseRevision || current.Status == "unavailable" || !reflect.DeepEqual(current.Definition, base.Definition) {
		return fail(errors.New("conflict: variable instance, definition or available base changed"))
	}
	var previousValue, currentValue any
	if err := json.Unmarshal(base.Value, &previousValue); err != nil {
		return fail(err)
	}
	if err := json.Unmarshal(current.Value, &currentValue); err != nil {
		return fail(err)
	}
	if !sameJSONVariableValue(previousValue, currentValue) {
		return fail(errors.New("conflict: variable value changed since editing began"))
	}
	request.ExpectedRevision = current.Revision
	encoded, err := json.Marshal(request)
	if err != nil {
		return fail(err)
	}
	budget = remaining()
	if budget <= 0 {
		return fail(errors.New("write preparation deadline exceeded"))
	}
	attempt.Sent = true
	attempt.ExpectedRevision = current.Revision
	attempt.ResponseJSON, err = client.OperateJSON(owner, name, string(protocol.CapabilityWrite), protocol.SchemaVariableWriteV3, string(encoded), budget)
	if err != nil {
		return fail(err)
	}
	return attempt
}

func sameJSONVariableValue(left, right any) bool {
	switch value := left.(type) {
	case float64:
		other, ok := right.(float64)
		return ok && value == other && (value != 0 || math.Signbit(value) == math.Signbit(other))
	case []any:
		other, ok := right.([]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for i := range value {
			if !sameJSONVariableValue(value[i], other[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		other, ok := right.(map[string]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for key, child := range value {
			next, exists := other[key]
			if !exists || !sameJSONVariableValue(child, next) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(left, right)
	}
}
