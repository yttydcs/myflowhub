package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

type variableJSONField struct {
	target   any
	required bool
	nullable bool
}

// decodeVariableObject keeps wire presence distinct from Go zero values and
// accepts exact field names only. Decode into a fresh candidate before commit.
func decodeVariableObject(data []byte, fields map[string]variableJSONField) error {
	if err := validatePayloadSize(data, DefaultMaxPayload); err != nil {
		return err
	}
	if err := validateVariableJSONEncoding(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return errors.New("variable JSON must be an object")
	}
	seen := make(map[string]bool, len(fields))
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		name, ok := token.(string)
		if !ok {
			return errors.New("variable JSON field name must be a string")
		}
		field, ok := fields[name]
		if !ok {
			return fmt.Errorf("unknown variable field %q", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate variable field %q", name)
		}
		seen[name] = true
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return err
		}
		if !field.nullable && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%s must not be null", name)
		}
		if _, stringsField := field.target.(*[]string); stringsField {
			var members []json.RawMessage
			if err := json.Unmarshal(raw, &members); err != nil {
				return err
			}
			for _, member := range members {
				if bytes.Equal(bytes.TrimSpace(member), []byte("null")) {
					return fmt.Errorf("%s entries must not be null", name)
				}
			}
		}
		if _, numericConstraint := field.target.(**float64); numericConstraint {
			if err := validateVariableNumber(DataSchemaDefinition{Type: "number"}, json.Number(raw)); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		if err := json.Unmarshal(raw, field.target); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("variable JSON contains trailing data")
	}
	for name, field := range fields {
		if field.required && !seen[name] {
			return fmt.Errorf("required variable field %q is absent", name)
		}
	}
	return nil
}

func parseVariableJSON(data []byte) (any, error) {
	return parseVariableJSONDepth(data, MaxDataSchemaDepth)
}

func parseVariableJSONDepth(data []byte, maxDepth int) (any, error) {
	if err := validatePayloadSize(data, DefaultMaxPayload); err != nil {
		return nil, err
	}
	if err := validateVariableJSONEncoding(data); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := readVariableJSONValue(decoder, 1, maxDepth)
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("variable JSON contains trailing data")
	}
	return value, nil
}

func readVariableJSONValue(decoder *json.Decoder, depth, maxDepth int) (any, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("variable JSON exceeds depth %d", maxDepth)
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		object := make(map[string]any)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			name, ok := key.(string)
			if !ok {
				return nil, errors.New("object key must be a string")
			}
			if _, exists := object[name]; exists {
				return nil, fmt.Errorf("duplicate object key %q", name)
			}
			value, err := readVariableJSONValue(decoder, depth+1, maxDepth)
			if err != nil {
				return nil, err
			}
			object[name] = value
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return object, nil
	case json.Delim('['):
		items := make([]any, 0)
		for decoder.More() {
			value, err := readVariableJSONValue(decoder, depth+1, maxDepth)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return items, nil
	default:
		return token, nil
	}
}

// encoding/json replaces malformed UTF-8 and lone UTF-16 surrogates. Reject
// them instead so Go and JavaScript cannot disagree about keys or values.
func validateVariableJSONEncoding(data []byte) error {
	if !utf8.Valid(data) || !json.Valid(data) {
		return errors.New("variable value must be valid UTF-8 JSON")
	}
	inside := false
	for index := 0; index < len(data); index++ {
		if data[index] == '"' {
			inside = !inside
			continue
		}
		if !inside || data[index] != '\\' {
			continue
		}
		index++
		if data[index] != 'u' {
			continue
		}
		code, _ := strconv.ParseUint(string(data[index+1:index+5]), 16, 16)
		index += 4
		if code >= 0xdc00 && code <= 0xdfff {
			return errors.New("JSON string contains an unpaired low surrogate")
		}
		if code < 0xd800 || code > 0xdbff {
			continue
		}
		if index+6 >= len(data) || data[index+1] != '\\' || data[index+2] != 'u' {
			return errors.New("JSON string contains an unpaired high surrogate")
		}
		low, err := strconv.ParseUint(string(data[index+3:index+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return errors.New("JSON string contains an unpaired high surrogate")
		}
		index += 6
	}
	return nil
}

func (s *DataSchemaDefinition) UnmarshalJSON(data []byte) error {
	if len(data) > MaxVariableDefinitionBytes {
		return fmt.Errorf("definition exceeds %d encoded bytes", MaxVariableDefinitionBytes)
	}
	// The schema wire grammar adds properties[]/schema wrappers around each
	// semantic node. Bound that grammar before recursive custom decoding.
	if _, err := parseVariableJSONDepth(data, MaxDataSchemaDepth*3+4); err != nil {
		return err
	}
	var next DataSchemaDefinition
	if err := decodeVariableObject(data, map[string]variableJSONField{
		"id": {&next.ID, false, false}, "type": {&next.Type, true, false},
		"nullable": {&next.Nullable, false, false}, "title": {&next.Title, false, false},
		"description": {&next.Description, false, false}, "format": {&next.Format, false, false},
		"content_type": {&next.ContentType, false, false}, "unit": {&next.Unit, false, false},
		"precision": {&next.Precision, false, false}, "minimum": {&next.Minimum, false, false},
		"maximum": {&next.Maximum, false, false}, "multiple_of": {&next.MultipleOf, false, false},
		"min_length": {&next.MinLength, false, false}, "max_length": {&next.MaxLength, false, false},
		"pattern": {&next.Pattern, false, false}, "min_items": {&next.MinItems, false, false},
		"max_items": {&next.MaxItems, false, false}, "enum": {&next.Enum, false, false},
		"read_only": {&next.ReadOnly, false, false}, "write_only": {&next.WriteOnly, false, false},
		"sensitive": {&next.Sensitive, false, false}, "properties": {&next.Properties, false, false},
		"items": {&next.Items, false, false}, "additional_properties": {&next.AdditionalProperties, false, false},
	}); err != nil {
		return err
	}
	*s = next
	return nil
}

func (p *DataSchemaProperty) UnmarshalJSON(data []byte) error {
	var next DataSchemaProperty
	if err := decodeVariableObject(data, map[string]variableJSONField{
		"name": {&next.Name, true, false}, "required": {&next.Required, false, false}, "schema": {&next.Schema, true, false},
	}); err != nil {
		return err
	}
	*p = next
	return nil
}
