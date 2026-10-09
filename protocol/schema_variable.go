package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

const (
	SchemaVariableValueV1        = "mfh.variable-value.v1"
	SchemaVariableWriteV3        = "mfh.variable-write.v3"
	MaxVariableSafeInteger       = 1<<53 - 1
	MaxVariableErrorMessageBytes = 1024
)

type VariableError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type VariableValueV1 struct {
	Version          int                   `json:"version"`
	InstanceID       string                `json:"instance_id"`
	Revision         string                `json:"revision"`
	Definition       *DataSchemaDefinition `json:"definition,omitempty"`
	Value            json.RawMessage       `json:"value,omitempty"`
	Status           string                `json:"status"`
	SourceTimeUnixMS *int64                `json:"source_time_unix_ms,omitempty"`
	ObservedAtUnixMS int64                 `json:"observed_at_unix_ms"`
	Error            *VariableError        `json:"error,omitempty"`
}

type VariableWriteV3 struct {
	Version            int             `json:"version"`
	ExpectedInstanceID string          `json:"expected_instance_id"`
	ExpectedRevision   string          `json:"expected_revision"`
	Value              json.RawMessage `json:"value"`
}

// ParseVariableRevision accepts only the canonical, nonzero uint64 spelling.
func ParseVariableRevision(value string) (uint64, error) {
	if value == "" || value[0] < '1' || value[0] > '9' || len(value) > 20 {
		return 0, errors.New("variable revision must be a canonical decimal string in 1..18446744073709551615")
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, errors.New("variable revision must contain only ASCII decimal digits")
		}
	}
	revision, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("variable revision out of range: %w", err)
	}
	return revision, nil
}

func (v VariableValueV1) Validate() error {
	if v.Definition != nil {
		if err := ValidateVariableDefinition(*v.Definition); err != nil {
			return fmt.Errorf("variable definition: %w", err)
		}
	}
	if err := validateVariableState(v, v.Definition); err != nil {
		return err
	}
	return validateVariableEncodedSize(v)
}

// ValidateVariableState validates fields and value using a definition already
// validated at registration. It does not validate or encode value.Definition.
// The owner must bind that definition to this instance and check the complete
// encoded envelope against its resource limit before committing any mutation.
func ValidateVariableState(definition DataSchemaDefinition, value VariableValueV1) error {
	return validateVariableState(value, &definition)
}

func validateVariableState(v VariableValueV1, definition *DataSchemaDefinition) error {
	if v.Version != 1 {
		return errors.New("variable value version must be 1")
	}
	if err := validateHexID("instance_id", v.InstanceID, 16); err != nil {
		return err
	}
	if _, err := ParseVariableRevision(v.Revision); err != nil {
		return err
	}
	if v.ObservedAtUnixMS < -MaxVariableSafeInteger || v.ObservedAtUnixMS > MaxVariableSafeInteger {
		return errors.New("observed_at_unix_ms must be a JS-safe integer")
	}
	if v.SourceTimeUnixMS != nil && (*v.SourceTimeUnixMS < -MaxVariableSafeInteger || *v.SourceTimeUnixMS > MaxVariableSafeInteger) {
		return errors.New("source_time_unix_ms must be a JS-safe integer")
	}
	switch v.Status {
	case "ready", "stale":
		if len(v.Value) == 0 {
			return errors.New("ready/stale variable value is required")
		}
	case "unavailable":
		if len(v.Value) != 0 {
			return errors.New("unavailable variable must omit value")
		}
	default:
		return errors.New("variable status must be ready, stale or unavailable")
	}
	if v.Error != nil {
		if v.Status == "ready" {
			return errors.New("ready variable must not contain error")
		}
		if err := validateText("variable error code", v.Error.Code, MaxIdentifierBytes, true); err != nil {
			return err
		}
		if err := validateText("variable error message", v.Error.Message, MaxVariableErrorMessageBytes, false); err != nil {
			return err
		}
	}
	if len(v.Value) != 0 {
		parsed, err := parseVariableJSON(v.Value)
		if err != nil {
			return fmt.Errorf("variable value: %w", err)
		}
		// A data event may omit the definition. Its consumer must then validate
		// against the cached definition for this exact registration instance.
		if definition != nil {
			if err := validateVariableValueNode(*definition, parsed); err != nil {
				return fmt.Errorf("variable value: %w", err)
			}
		}
	}
	return nil
}

func (w VariableWriteV3) Validate() error {
	if w.Version != 3 {
		return errors.New("variable write version must be 3")
	}
	if err := validateHexID("expected_instance_id", w.ExpectedInstanceID, 16); err != nil {
		return err
	}
	if _, err := ParseVariableRevision(w.ExpectedRevision); err != nil {
		return fmt.Errorf("expected_revision: %w", err)
	}
	if _, err := parseVariableJSON(w.Value); err != nil {
		return fmt.Errorf("variable write value: %w", err)
	}
	return validateVariableEncodedSize(w)
}

func validateVariableEncodedSize(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode variable: %w", err)
	}
	return validatePayloadSize(data, DefaultMaxPayload)
}

func (v *VariableValueV1) UnmarshalJSON(data []byte) error {
	var next VariableValueV1
	err := decodeVariableObject(data, map[string]variableJSONField{
		"version":             {&next.Version, true, false},
		"instance_id":         {&next.InstanceID, true, false},
		"revision":            {&next.Revision, true, false},
		"definition":          {&next.Definition, false, false},
		"value":               {&next.Value, false, true},
		"status":              {&next.Status, true, false},
		"source_time_unix_ms": {&next.SourceTimeUnixMS, false, false},
		"observed_at_unix_ms": {&next.ObservedAtUnixMS, true, false},
		"error":               {&next.Error, false, false},
	})
	if err != nil {
		return err
	}
	*v = next
	return nil
}

func (w *VariableWriteV3) UnmarshalJSON(data []byte) error {
	var next VariableWriteV3
	err := decodeVariableObject(data, map[string]variableJSONField{
		"version":              {&next.Version, true, false},
		"expected_instance_id": {&next.ExpectedInstanceID, true, false},
		"expected_revision":    {&next.ExpectedRevision, true, false},
		"value":                {&next.Value, true, true},
	})
	if err != nil {
		return err
	}
	*w = next
	return nil
}

func (e *VariableError) UnmarshalJSON(data []byte) error {
	var next VariableError
	if err := decodeVariableObject(data, map[string]variableJSONField{
		"code": {&next.Code, true, false}, "message": {&next.Message, true, false},
	}); err != nil {
		return err
	}
	*e = next
	return nil
}
