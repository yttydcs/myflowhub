package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

func variableTestPointer[T any](value T) *T { return &value }

func variableTestEnvelope() VariableValueV1 {
	return VariableValueV1{
		Version: 1, InstanceID: "4c2d8b7e398043e59b2943e96512c641", Revision: "9007199254740993",
		Definition: &DataSchemaDefinition{Type: "boolean"}, Value: json.RawMessage(`false`),
		Status: "ready", ObservedAtUnixMS: 0,
	}
}

func TestVariableSharedValidationVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/variable_validation_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		Version     int `json:"version"`
		Definitions []struct {
			Name       string
			Definition json.RawMessage
			Valid      bool
		} `json:"definitions"`
		Values []struct {
			Name       string
			Definition json.RawMessage
			Value      json.RawMessage
			Valid      bool
		} `json:"values"`
		Revisions []struct {
			Revision string
			Valid    bool
		} `json:"revisions"`
		Wire []struct {
			Name  string
			Kind  string
			JSON  string
			Valid bool
		} `json:"wire"`
	}
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	if vectors.Version != 1 || len(vectors.Definitions) == 0 || len(vectors.Values) == 0 || len(vectors.Wire) == 0 {
		t.Fatal("incomplete validation vectors")
	}
	for _, vector := range vectors.Definitions {
		t.Run("definition/"+vector.Name, func(t *testing.T) {
			var definition DataSchemaDefinition
			err := json.Unmarshal(vector.Definition, &definition)
			if err == nil {
				err = ValidateVariableDefinition(definition)
			}
			if (err == nil) != vector.Valid {
				t.Fatalf("valid=%v, error=%v", vector.Valid, err)
			}
		})
	}
	for _, vector := range vectors.Values {
		t.Run("value/"+vector.Name, func(t *testing.T) {
			var definition DataSchemaDefinition
			err := json.Unmarshal(vector.Definition, &definition)
			if err == nil {
				err = ValidateVariableValue(definition, vector.Value)
			}
			if (err == nil) != vector.Valid {
				t.Fatalf("valid=%v, error=%v", vector.Valid, err)
			}
		})
	}
	for _, vector := range vectors.Revisions {
		t.Run("revision/"+vector.Revision, func(t *testing.T) {
			_, err := ParseVariableRevision(vector.Revision)
			if (err == nil) != vector.Valid {
				t.Fatalf("valid=%v, error=%v", vector.Valid, err)
			}
		})
	}
	for _, vector := range vectors.Wire {
		t.Run("wire/"+vector.Name, func(t *testing.T) {
			var target ValidatedPayload
			switch vector.Kind {
			case "value":
				target = &VariableValueV1{}
			case "write":
				target = &VariableWriteV3{}
			default:
				t.Fatalf("unknown wire kind %q", vector.Kind)
			}
			err := DecodeJSONPayload([]byte(vector.JSON), DefaultMaxPayload, target)
			if (err == nil) != vector.Valid {
				t.Fatalf("valid=%v, error=%v", vector.Valid, err)
			}
			if err == nil {
				if _, err := EncodeJSONPayload(target, DefaultMaxPayload); err != nil {
					t.Fatalf("accepted wire cannot encode: %v", err)
				}
			}
		})
	}
}

func TestVariableRoundTripPreservesPresenceAndRevision(t *testing.T) {
	for _, test := range []struct {
		name, kind, raw string
		nullable        bool
	}{
		{"false", "boolean", `false`, false}, {"zero", "integer", `0`, false},
		{"empty", "string", `""`, false}, {"null", "string", `null`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := variableTestEnvelope()
			value.Definition = &DataSchemaDefinition{Type: test.kind, Nullable: test.nullable}
			value.Value = json.RawMessage(test.raw)
			value.Revision = "18446744073709551615"
			value.SourceTimeUnixMS = variableTestPointer(int64(0))
			data, err := EncodeJSONPayload(value, DefaultMaxPayload)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields["value"]) != test.raw || string(fields["source_time_unix_ms"]) != "0" || string(fields["observed_at_unix_ms"]) != "0" {
				t.Fatalf("presence was lost: %s", data)
			}
			var decoded VariableValueV1
			if err := DecodeJSONPayload(data, DefaultMaxPayload, &decoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(value, decoded) {
				t.Fatalf("round-trip changed value: %#v", decoded)
			}
			if revision, err := ParseVariableRevision(decoded.Revision); err != nil || revision != math.MaxUint64 {
				t.Fatalf("revision lost precision: %v, %v", revision, err)
			}
		})
	}
	value := variableTestEnvelope()
	value.Status, value.Value = "unavailable", nil
	data, err := EncodeJSONPayload(value, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(`"value":`)) {
		t.Fatalf("unavailable emitted value: %s", data)
	}
}

func TestVariableDefinitionNativeBoundsAndCycles(t *testing.T) {
	for _, invalid := range []DataSchemaDefinition{
		{Type: "number", Minimum: variableTestPointer(math.NaN())},
		{Type: "number", Maximum: variableTestPointer(math.Inf(1))},
		{Type: "number", MultipleOf: variableTestPointer(math.Inf(-1))},
		{Type: "number", Maximum: variableTestPointer(float64(MaxVariableSafeInteger + 1))},
		{Type: "boolean", ID: strings.Repeat("x", MaxSchemaBytes+1)},
	} {
		if err := ValidateVariableDefinition(invalid); err == nil {
			t.Fatalf("accepted invalid definition %#v", invalid)
		}
	}
	cycle := DataSchemaDefinition{Type: "array"}
	cycle.Items = &cycle
	if err := ValidateVariableDefinition(cycle); err == nil || !strings.Contains(err.Error(), "depth") {
		t.Fatalf("cyclic definition did not terminate at depth limit: %v", err)
	}
	object := DataSchemaDefinition{Type: "object"}
	object.AdditionalProperties = &object
	if err := ValidateVariableDefinition(object); err == nil {
		t.Fatal("accepted recursive map")
	}
	if _, err := dataSchemaFromType(reflect.TypeOf(VariableValueV1{}), 1); err == nil {
		t.Fatal("recursive envelope must not silently flatten through reflection")
	}
	for _, definition := range []DataSchemaDefinition{{Type: "boolean"}, {Type: "string", Enum: []string{"", "on"}}} {
		if err := ValidateVariableDefinition(definition); err != nil {
			t.Fatal(err)
		}
		if err := definition.Validate(); err == nil {
			t.Fatal("non-variable registration must still require ID")
		}
	}
	// The property bound is global, including properties beneath items/maps.
	definition := DataSchemaDefinition{Type: "object"}
	for index := range 2 {
		child := &DataSchemaDefinition{Type: "object"}
		for nested := range 64 {
			child.Properties = append(child.Properties, DataSchemaProperty{Name: fmt.Sprintf("p%d", nested), Schema: &DataSchemaDefinition{Type: "boolean"}})
		}
		definition.Properties = append(definition.Properties, DataSchemaProperty{Name: fmt.Sprintf("group%d", index), Schema: child})
	}
	if err := ValidateVariableDefinition(definition); err == nil {
		t.Fatal("accepted globally more than 128 properties")
	}
}

func TestVariableEncodedDefinitionAndEnvelopeBounds(t *testing.T) {
	definition := DataSchemaDefinition{Type: "object"}
	for index := range 64 {
		definition.Properties = append(definition.Properties, DataSchemaProperty{Name: fmt.Sprintf("p%d", index), Schema: &DataSchemaDefinition{Type: "boolean", Description: "x"}})
	}
	encoded, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	remaining := MaxVariableDefinitionBytes - len(encoded)
	for _, property := range definition.Properties {
		added := min(remaining, 1023)
		property.Schema.Description += strings.Repeat("x", added)
		remaining -= added
	}
	if remaining != 0 {
		t.Fatal("fixture cannot reach definition limit")
	}
	encoded, err = json.Marshal(definition)
	if err != nil || len(encoded) != MaxVariableDefinitionBytes {
		t.Fatalf("wrong boundary fixture: %d, %v", len(encoded), err)
	}
	if err := ValidateVariableDefinition(definition); err != nil {
		t.Fatalf("64KiB should pass: %v", err)
	}
	definition.Properties[len(definition.Properties)-1].Schema.Description += "x"
	if err := ValidateVariableDefinition(definition); err == nil {
		t.Fatal("accepted 64KiB+1 definition")
	}

	value := variableTestEnvelope()
	value.Definition = &DataSchemaDefinition{Type: "string"}
	value.Value = json.RawMessage(`""`)
	encoded, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	value.Value = json.RawMessage(`"` + strings.Repeat("x", DefaultMaxPayload-len(encoded)) + `"`)
	encoded, err = EncodeJSONPayload(value, DefaultMaxPayload)
	if err != nil || len(encoded) != DefaultMaxPayload {
		t.Fatalf("full envelope boundary failed: %d, %v", len(encoded), err)
	}
	value.Value = append(value.Value[:len(value.Value)-1], 'x', '"')
	if err := value.Validate(); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("envelope overhead was not bounded: %v", err)
	}
	write := VariableWriteV3{Version: 3, ExpectedInstanceID: value.InstanceID, ExpectedRevision: "1", Value: value.Value}
	write.Value = json.RawMessage(`"` + strings.Repeat("x", DefaultMaxPayload-2) + `"`)
	if err := write.Validate(); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("write overhead was not bounded: %v", err)
	}
}

func TestVariableWireRejectsMalformedPresenceAndJSON(t *testing.T) {
	value := variableTestEnvelope()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"version", "instance_id", "revision", "status", "observed_at_unix_ms"} {
		for _, missing := range []bool{true, false} {
			var object map[string]json.RawMessage
			if err := json.Unmarshal(data, &object); err != nil {
				t.Fatal(err)
			}
			if missing {
				delete(object, field)
			} else {
				object[field] = json.RawMessage(`null`)
			}
			bad, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			var next VariableValueV1
			if err := DecodeJSONPayload(bad, DefaultMaxPayload, &next); !errors.Is(err, ErrInvalidPayload) {
				t.Fatalf("field %s missing=%v accepted: %v", field, missing, err)
			}
		}
	}
	for _, raw := range []string{``, ` `, `true false`, `{"a":1,"a":2}`, `"\ud800"`, `"\udfff"`, `"\ud800\u0041"`, "\"\xff\"", `NaN`, `Infinity`, `1e999999`} {
		definition := DataSchemaDefinition{Type: "number"}
		if err := ValidateVariableValue(definition, json.RawMessage(raw)); err == nil {
			t.Fatalf("accepted malformed value %q", raw)
		}
	}
	for _, raw := range []string{`"\ud83d\ude00"`, `"\\ud800"`, `"\ufffd"`} {
		if err := ValidateVariableValue(DataSchemaDefinition{Type: "string"}, json.RawMessage(raw)); err != nil {
			t.Fatalf("rejected valid Unicode %s: %v", raw, err)
		}
	}
	for _, raw := range []string{`9007199254740991.1`, `9007199254740993`, `-9007199254740991.1`, `1e-400`, `1e400`} {
		if err := ValidateVariableValue(DataSchemaDefinition{Type: "number"}, json.RawMessage(raw)); err == nil {
			t.Fatalf("unsafe number was rounded into validity: %s", raw)
		}
	}
	for _, raw := range []string{`{"type":"number","maximum":9007199254740991.1}`, `{"type":"string","enum":[null]}`, `{"type":"object","properties":[null]}`} {
		var definition DataSchemaDefinition
		err := json.Unmarshal([]byte(raw), &definition)
		if err == nil {
			err = ValidateVariableDefinition(definition)
		}
		if err == nil {
			t.Fatalf("invalid definition decoded: %s", raw)
		}
	}
	for _, depth := range []int{12, 13} {
		definition := DataSchemaDefinition{Type: "boolean"}
		raw := `false`
		for range depth - 1 {
			child := definition
			definition = DataSchemaDefinition{Type: "array", Items: &child}
			raw = `[` + raw + `]`
		}
		// Build a separate copy at each level; using the root's address would
		// create a cycle rather than a nested array.
		if err := ValidateVariableValue(definition, json.RawMessage(raw)); (err == nil) != (depth == 12) {
			t.Fatalf("depth %d: %v", depth, err)
		}
	}
}

func TestVariablePrevalidatedStateAndBuiltinProviders(t *testing.T) {
	value := variableTestEnvelope()
	definition := *value.Definition
	if err := ValidateVariableState(definition, value); err != nil {
		t.Fatal(err)
	}
	value.Value = json.RawMessage(`"false"`)
	if err := ValidateVariableState(definition, value); err == nil {
		t.Fatal("prevalidated definition bypassed value checks")
	}
	value.Value, value.Definition = json.RawMessage(`false`), nil
	if err := ValidateVariableState(definition, value); err != nil {
		t.Fatal(err)
	}
	value.Value = json.RawMessage(`"cached-definition-required"`)
	if err := value.Validate(); err != nil {
		t.Fatalf("data without definition should only validate structure: %v", err)
	}
	if err := ValidateVariableState(definition, value); err == nil {
		t.Fatal("cached definition did not enforce type")
	}

	definitions, err := BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, builtin := range definitions {
		if builtin.ID == "mfh.variable-write.v2" || builtin.ID == SchemaVariableValueV1 || builtin.ID == SchemaVariableWriteV3 {
			t.Fatalf("dynamic variable or old schema reflected: %s", builtin.ID)
		}
		if _, err := DataSchemaByID(builtin.ID); err != nil {
			t.Fatalf("builtin cannot be used as a variable definition: %s: %v", builtin.ID, err)
		}
	}
	for _, id := range []string{"missing", "mfh.variable-write.v2", SchemaVariableValueV1} {
		if _, err := DataSchemaByID(id); err == nil {
			t.Fatalf("unknown schema %q accepted", id)
		}
	}
	for _, id := range []string{SchemaResourceCatalogV2, SchemaManagementConfigV1, SchemaManagementHealthV1, SchemaManagementTopologyV1, SchemaFileTransfersV1, SchemaAdmissionStatusV1} {
		t.Run(id, func(t *testing.T) {
			definition, err := DataSchemaByID(id)
			if err != nil {
				t.Fatal(err)
			}
			var inspect func(DataSchemaDefinition)
			inspect = func(node DataSchemaDefinition) {
				if node.Type == "integer" && (node.Minimum == nil || node.Maximum == nil || *node.Minimum < -MaxVariableSafeInteger || *node.Maximum > MaxVariableSafeInteger) {
					t.Fatalf("native integer has no safe bound: %#v", node)
				}
				for _, property := range node.Properties {
					inspect(*property.Schema)
				}
				if node.Items != nil {
					inspect(*node.Items)
				}
				if node.AdditionalProperties != nil {
					inspect(*node.AdditionalProperties)
				}
			}
			inspect(definition)
		})
	}
	config, err := DataSchemaByID(SchemaManagementConfigV1)
	if err != nil {
		t.Fatal(err)
	}
	for _, revision := range []uint64{MaxVariableSafeInteger, MaxVariableSafeInteger + 1} {
		payload, err := json.Marshal(ManagementConfigV1{Version: 1, Revision: revision})
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateVariableValue(config, payload); (err == nil) != (revision == MaxVariableSafeInteger) {
			t.Fatalf("native uint64 precision: %d: %v", revision, err)
		}
	}
	fileDefinition, err := DataSchemaByID(SchemaFileTransfersV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateVariableValue(fileDefinition, json.RawMessage(`{"version":1,"revision":1,"transfers":null}`)); err != nil {
		t.Fatalf("native nil slice must be explicitly nullable: %v", err)
	}
	mutated, err := DataSchemaByID(SchemaManagementConfigV1)
	if err != nil {
		t.Fatal(err)
	}
	mutated.Properties[0].Schema.Type = "null"
	fresh, err := DataSchemaByID(SchemaManagementConfigV1)
	if err != nil || fresh.Properties[0].Schema.Type == "null" {
		t.Fatal("builtin lookup leaked mutable shared definitions")
	}
}
