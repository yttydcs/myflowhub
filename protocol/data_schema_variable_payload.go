package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// EncodeVariablePayload adapts a validated Go domain value to its declared
// platform value definition. Explicit decimal string formats preserve full-width
// identifiers without changing persistence or non-variable operation codecs.
func EncodeVariablePayload(definition DataSchemaDefinition, value ValidatedPayload, maxBytes int) (json.RawMessage, error) {
	if value == nil || (reflect.ValueOf(value).Kind() == reflect.Pointer && reflect.ValueOf(value).IsNil()) {
		return nil, fmt.Errorf("%w: variable domain value is required", ErrInvalidPayload)
	}
	if err := ValidateVariableDefinition(definition); err != nil {
		return nil, fmt.Errorf("variable definition: %w", err)
	}
	data, err := EncodeJSONPayload(value, maxBytes)
	if err != nil {
		return nil, err
	}
	domain, err := parseVariableJSON(data)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid Go domain JSON: %v", ErrInvalidPayload, err)
	}
	converted, err := encodeVariableDomainNode(definition, domain)
	if err != nil {
		return nil, err
	}
	if err := validateVariableValueNode(definition, converted); err != nil {
		return nil, fmt.Errorf("%w: variable domain value: %v", ErrInvalidPayload, err)
	}
	result, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("encode variable domain value: %w", err)
	}
	if err := validatePayloadSize(result, maxBytes); err != nil {
		return nil, err
	}
	return result, nil
}

// DecodeVariablePayload accepts only the current standard value representation.
// Validation precedes Go-type adaptation: numeric JSON is never accepted in a
// field whose definition requires a decimal string, including small NodeIDs.
func DecodeVariablePayload(definition DataSchemaDefinition, data json.RawMessage, maxBytes int, target ValidatedPayload) error {
	if target == nil {
		return errors.New("variable domain target is required")
	}
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Pointer || reflect.ValueOf(target).IsNil() {
		return errors.New("variable domain target must be a non-nil pointer")
	}
	if err := validatePayloadSize(data, maxBytes); err != nil {
		return err
	}
	if err := ValidateVariableDefinition(definition); err != nil {
		return fmt.Errorf("variable definition: %w", err)
	}
	value, err := parseVariableJSON(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if err := validateVariableValueNode(definition, value); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	converted, err := decodeVariableDomainNode(definition, value, targetType)
	if err != nil {
		return fmt.Errorf("variable domain target: %w", err)
	}
	encoded, err := json.Marshal(converted)
	if err != nil {
		return fmt.Errorf("encode Go domain value: %w", err)
	}
	// Decode a fresh destination. Invalid business objects must not partially
	// overwrite the caller's existing state before their Validate method fails.
	candidate := reflect.New(targetType.Elem())
	payload, ok := candidate.Interface().(ValidatedPayload)
	if !ok {
		return errors.New("variable domain target must implement Validate")
	}
	if err := DecodeJSONPayload(encoded, maxBytes, payload); err != nil {
		return err
	}
	reflect.ValueOf(target).Elem().Set(candidate.Elem())
	return nil
}

func decimalVariableFormat(definition DataSchemaDefinition) bool {
	return definition.Type == "string" && (definition.Format == "node-id" || definition.Format == "uint64" || definition.Format == "int64")
}
func encodeVariableDomainNode(definition DataSchemaDefinition, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	if decimalVariableFormat(definition) {
		if number, ok := value.(json.Number); ok {
			return string(number), nil
		}
		return value, nil
	}
	switch current := value.(type) {
	case map[string]any:
		if definition.Type != "object" {
			return value, nil
		}
		for name, item := range current {
			child := variableDomainProperty(definition, name)
			if child == nil {
				continue
			}
			converted, err := encodeVariableDomainNode(*child, item)
			if err != nil {
				return nil, fmt.Errorf("property %q: %w", name, err)
			}
			current[name] = converted
		}
	case []any:
		if definition.Type != "array" || definition.Items == nil {
			return value, nil
		}
		for index, item := range current {
			converted, err := encodeVariableDomainNode(*definition.Items, item)
			if err != nil {
				return nil, fmt.Errorf("item %d: %w", index, err)
			}
			current[index] = converted
		}
	}
	return value, nil
}
func decodeVariableDomainNode(definition DataSchemaDefinition, value any, target reflect.Type) (any, error) {
	for target != nil && target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if value == nil {
		return nil, nil
	}
	if decimalVariableFormat(definition) && target != nil {
		switch target.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			text, ok := value.(string)
			if !ok {
				return nil, errors.New("decimal variable field requires a string")
			}
			return json.Number(text), nil
		}
	}
	switch current := value.(type) {
	case map[string]any:
		if definition.Type != "object" {
			return value, nil
		}
		for name, item := range current {
			child := variableDomainProperty(definition, name)
			if child == nil {
				continue
			}
			childType := variableDomainFieldType(target, name)
			converted, err := decodeVariableDomainNode(*child, item, childType)
			if err != nil {
				return nil, fmt.Errorf("property %q: %w", name, err)
			}
			current[name] = converted
		}
	case []any:
		if definition.Type != "array" || definition.Items == nil {
			return value, nil
		}
		var itemType reflect.Type
		if target != nil && (target.Kind() == reflect.Array || target.Kind() == reflect.Slice) {
			itemType = target.Elem()
		}
		for index, item := range current {
			converted, err := decodeVariableDomainNode(*definition.Items, item, itemType)
			if err != nil {
				return nil, fmt.Errorf("item %d: %w", index, err)
			}
			current[index] = converted
		}
	}
	return value, nil
}
func variableDomainProperty(definition DataSchemaDefinition, name string) *DataSchemaDefinition {
	for _, property := range definition.Properties {
		if property.Name == name {
			return property.Schema
		}
	}
	return definition.AdditionalProperties
}
func variableDomainFieldType(target reflect.Type, name string) reflect.Type {
	if target == nil {
		return nil
	}
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if target.Kind() == reflect.Map {
		return target.Elem()
	}
	if target.Kind() != reflect.Struct {
		return nil
	}
	for index := 0; index < target.NumField(); index++ {
		field := target.Field(index)
		if !field.IsExported() {
			continue
		}
		jsonName, options, _ := strings.Cut(field.Tag.Get("json"), ",")
		if jsonName == "-" {
			continue
		}
		if jsonName == "" {
			jsonName = field.Name
		}
		if jsonName == name {
			// A Go field already tagged ,string must receive the canonical string.
			for _, option := range strings.Split(options, ",") {
				if option == "string" {
					return nil
				}
			}
			return field.Type
		}
	}
	return nil
}
