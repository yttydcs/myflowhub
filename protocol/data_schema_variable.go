package protocol

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const MaxVariableDefinitionBytes = 64 << 10

// A fixed ASCII MIME syntax shared with display clients; this is not a
// provider-supplied expression. Parameters are annotations, not executable data.
var variableContentTypePattern = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+/[!#$%&'*+.^_`|~0-9A-Za-z-]+(?:[ \\t]*;[ \\t]*[!#$%&'*+.^_`|~0-9A-Za-z-]+=(?:[!#$%&'*+.^_`|~0-9A-Za-z-]+|\"[^\"\\r\\n]*\"))*$")

// DataSchemaByID returns an independent definition for a canonical provider's
// domain value. Variable envelopes are deliberately not reflected: their
// definition is recursive and their value depends on that definition.
func DataSchemaByID(id string) (DataSchemaDefinition, error) {
	for _, item := range builtinDataSchemaTypes() {
		if item.id == id {
			definition, err := dataSchemaFromTypeMode(item.typeOf, 1, true)
			if err != nil {
				return DataSchemaDefinition{}, fmt.Errorf("build variable schema %s: %w", id, err)
			}
			definition.ID, definition.Title = item.id, item.title
			annotateBuiltinDataSchema(&definition)
			if err := ValidateVariableDefinition(definition); err != nil {
				return DataSchemaDefinition{}, fmt.Errorf("variable schema %s: %w", id, err)
			}
			return definition, nil
		}
	}
	return DataSchemaDefinition{}, fmt.Errorf("unknown data schema %q", id)
}

// ValidateVariableDefinition accepts an optional semantic ID, unlike the
// registered non-variable schema's Validate method. Structure is checked
// before encoding so cyclic definitions terminate at the depth bound.
func ValidateVariableDefinition(definition DataSchemaDefinition) error {
	properties := 0
	if err := definition.validateNode(1, &properties, true); err != nil {
		return err
	}
	if err := validateVariableDefinitionNode(definition); err != nil {
		return err
	}
	data, err := json.Marshal(definition)
	if err != nil {
		return fmt.Errorf("encode variable definition: %w", err)
	}
	if len(data) > MaxVariableDefinitionBytes {
		return fmt.Errorf("variable definition exceeds %d encoded bytes", MaxVariableDefinitionBytes)
	}
	return nil
}

func validateVariableDefinitionNode(s DataSchemaDefinition) error {
	if err := validateText("data schema id", s.ID, MaxSchemaBytes, false); err != nil {
		return err
	}
	numeric := s.Type == "integer" || s.Type == "number"
	if !numeric && (s.Minimum != nil || s.Maximum != nil || s.MultipleOf != nil || s.Precision != nil || s.Unit != "") {
		return errors.New("numeric constraints require integer or number type")
	}
	for _, bound := range []*float64{s.Minimum, s.Maximum, s.MultipleOf} {
		if bound != nil && (math.IsNaN(*bound) || math.IsInf(*bound, 0) || math.Abs(*bound) > MaxVariableSafeInteger) {
			return errors.New("numeric constraints must be finite JS-safe numbers")
		}
	}
	if s.Type != "string" && (s.MinLength != nil || s.MaxLength != nil || s.Enum != nil) {
		return errors.New("length and enum constraints require string type")
	}
	if s.Type != "array" && (s.MinItems != nil || s.MaxItems != nil) {
		return errors.New("item bounds require array type")
	}
	if s.Type != "object" && s.Properties != nil {
		return errors.New("properties require object type")
	}
	for _, bound := range []*int{s.MinLength, s.MaxLength, s.MinItems, s.MaxItems} {
		if bound != nil && uint64(*bound) > MaxVariableSafeInteger {
			return errors.New("length/item bounds must be JS-safe integers")
		}
	}
	if s.ReadOnly && s.WriteOnly {
		return errors.New("data schema cannot be both read_only and write_only")
	}
	switch s.Format {
	case "":
	case "unix-ms":
		if s.Type != "integer" {
			return errors.New("unix-ms format requires integer type")
		}
	case "int64", "uint64", "node-id", "base64":
		if s.Type != "string" {
			return errors.New("decimal/base64 formats require string type")
		}
	default:
		return fmt.Errorf("unsupported variable format %q", s.Format)
	}
	// These are the existing platform patterns. Do not execute arbitrary
	// provider regexes: Go and JavaScript have different regex semantics.
	switch s.Pattern {
	case "", "^[0-9a-f]{32}$", "^[1-9][0-9]*$":
	default:
		return fmt.Errorf("unsupported variable pattern %q", s.Pattern)
	}
	if s.ContentType != "" {
		if s.Format != "base64" {
			return errors.New("content_type requires base64 format")
		}
		if err := validateText("content_type", s.ContentType, MaxContentTypeBytes, false); err != nil {
			return err
		}
		if !variableContentTypePattern.MatchString(s.ContentType) {
			return errors.New("invalid content_type syntax")
		}
	}
	for _, value := range s.Enum {
		if err := validateVariableString(s, value, false); err != nil {
			return fmt.Errorf("enum value %q: %w", value, err)
		}
	}
	for _, property := range s.Properties {
		if err := validateVariableDefinitionNode(*property.Schema); err != nil {
			return fmt.Errorf("property %q: %w", property.Name, err)
		}
	}
	if s.Items != nil {
		if err := validateVariableDefinitionNode(*s.Items); err != nil {
			return fmt.Errorf("items: %w", err)
		}
	}
	if s.AdditionalProperties != nil {
		if err := validateVariableDefinitionNode(*s.AdditionalProperties); err != nil {
			return fmt.Errorf("additional_properties: %w", err)
		}
	}
	return nil
}

func ValidateVariableValue(definition DataSchemaDefinition, value json.RawMessage) error {
	if err := ValidateVariableDefinition(definition); err != nil {
		return fmt.Errorf("variable definition: %w", err)
	}
	parsed, err := parseVariableJSON(value)
	if err != nil {
		return err
	}
	return validateVariableValueNode(definition, parsed)
}

func validateVariableValueNode(s DataSchemaDefinition, value any) error {
	if value == nil {
		if s.Nullable || s.Type == "null" {
			return nil
		}
		return errors.New("null is not allowed by definition")
	}
	switch s.Type {
	case "boolean":
		if _, ok := value.(bool); ok {
			return nil
		}
	case "integer", "number":
		if number, ok := value.(json.Number); ok {
			return validateVariableNumber(s, number)
		}
	case "string":
		if text, ok := value.(string); ok {
			return validateVariableString(s, text, true)
		}
	case "array":
		if items, ok := value.([]any); ok {
			if s.MinItems != nil && len(items) < *s.MinItems || s.MaxItems != nil && len(items) > *s.MaxItems {
				return errors.New("array length is outside item bounds")
			}
			for index, item := range items {
				if err := validateVariableValueNode(*s.Items, item); err != nil {
					return fmt.Errorf("items[%d]: %w", index, err)
				}
			}
			return nil
		}
	case "object":
		if object, ok := value.(map[string]any); ok {
			properties := make(map[string]*DataSchemaDefinition, len(s.Properties))
			for _, property := range s.Properties {
				properties[property.Name] = property.Schema
				if _, exists := object[property.Name]; property.Required && !exists {
					return fmt.Errorf("required property %q is absent", property.Name)
				}
			}
			for key, item := range object {
				if err := validateText("object key", key, MaxIdentifierBytes, true); err != nil {
					return err
				}
				child, known := properties[key]
				if !known {
					child = s.AdditionalProperties
				}
				if child == nil {
					return fmt.Errorf("unknown property %q", key)
				}
				if err := validateVariableValueNode(*child, item); err != nil {
					return fmt.Errorf("property %q: %w", key, err)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("value must have type %s", s.Type)
}

func validateVariableNumber(s DataSchemaDefinition, number json.Number) error {
	f, err := strconv.ParseFloat(string(number), 64)
	if err != nil || math.IsInf(f, 0) || math.Abs(f) > MaxVariableSafeInteger {
		return errors.New("number must be finite and within JS-safe numeric bounds")
	}
	// Use exact decimal arithmetic for bounds and steps, before any float64
	// rounding. Bound exponent length before constructing arbitrary precision
	// rationals, including values that ParseFloat would underflow to zero.
	text := string(number)
	if index := strings.IndexAny(text, "eE"); index >= 0 {
		exponent, err := strconv.ParseInt(text[index+1:], 10, 32)
		if err != nil || exponent < -324 || exponent > 308 {
			return errors.New("number exponent is outside supported range")
		}
	}
	rational, ok := new(big.Rat).SetString(text)
	if !ok {
		return errors.New("invalid decimal number")
	}
	if s.Type == "integer" && !rational.IsInt() {
		return errors.New("integer value must not have a fractional part")
	}
	if new(big.Rat).Abs(rational).Cmp(new(big.Rat).SetInt64(MaxVariableSafeInteger)) > 0 {
		return errors.New("number exceeds JS-safe bounds")
	}
	if f == 0 && rational.Sign() != 0 {
		return errors.New("number underflows JavaScript precision")
	}
	for _, bound := range []struct {
		value   *float64
		minimum bool
	}{{s.Minimum, true}, {s.Maximum, false}} {
		if bound.value == nil {
			continue
		}
		limit := variableDecimal(*bound.value)
		comparison := rational.Cmp(limit)
		if bound.minimum && comparison < 0 || !bound.minimum && comparison > 0 {
			return errors.New("number is outside definition bounds")
		}
	}
	if s.MultipleOf != nil && !new(big.Rat).Quo(rational, variableDecimal(*s.MultipleOf)).IsInt() {
		return errors.New("number is not a multiple_of the declared step")
	}
	if s.Precision != nil {
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(*s.Precision)), nil)
		if !new(big.Rat).Mul(rational, new(big.Rat).SetInt(scale)).IsInt() {
			return errors.New("number exceeds declared precision")
		}
	}
	return nil
}

func variableDecimal(value float64) *big.Rat {
	result, _ := new(big.Rat).SetString(strconv.FormatFloat(value, 'g', -1, 64))
	return result
}

func validateVariableString(s DataSchemaDefinition, value string, checkEnum bool) error {
	if !utf8.ValidString(value) {
		return errors.New("string must be valid UTF-8")
	}
	// Lengths are UTF-8 bytes, matching protocol text limits on every platform.
	if s.MinLength != nil && len(value) < *s.MinLength || s.MaxLength != nil && len(value) > *s.MaxLength {
		return errors.New("string length is outside UTF-8 byte bounds")
	}
	if checkEnum && len(s.Enum) > 0 {
		found := false
		for _, member := range s.Enum {
			if member == value {
				found = true
				break
			}
		}
		if !found {
			return errors.New("string is outside declared enum")
		}
	}
	switch s.Format {
	case "int64":
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil || strconv.FormatInt(number, 10) != value {
			return errors.New("int64 string must be canonical and in range")
		}
	case "uint64":
		number, err := strconv.ParseUint(value, 10, 64)
		if err != nil || strconv.FormatUint(number, 10) != value {
			return errors.New("uint64 string must be canonical and in range")
		}
	case "node-id":
		if _, err := ParseVariableRevision(value); err != nil {
			return fmt.Errorf("node-id: %w", err)
		}
	case "base64":
		decoded, err := base64.StdEncoding.Strict().DecodeString(value)
		if err != nil || base64.StdEncoding.EncodeToString(decoded) != value {
			return errors.New("base64 string must use canonical padded standard encoding")
		}
	}
	switch s.Pattern {
	case "^[0-9a-f]{32}$":
		if err := validateHexID("string", value, 16); err != nil {
			return err
		}
	case "^[1-9][0-9]*$":
		if value == "" || value[0] < '1' || value[0] > '9' {
			return errors.New("string does not match decimal pattern")
		}
		for _, digit := range value {
			if digit < '0' || digit > '9' {
				return errors.New("string does not match decimal pattern")
			}
		}
	}
	return nil
}
