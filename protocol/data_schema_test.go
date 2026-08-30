package protocol

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuiltinDataSchemasAreValidSortedAndDeterministic(t *testing.T) {
	definitions, err := BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) < 30 {
		t.Fatalf("expected broad first-party schema coverage, got %d", len(definitions))
	}
	seen := make(map[string]struct{}, len(definitions))
	for index, definition := range definitions {
		if err := definition.Validate(); err != nil {
			t.Fatalf("schema %s: %v", definition.ID, err)
		}
		if _, exists := seen[definition.ID]; exists {
			t.Fatalf("duplicate schema %s", definition.ID)
		}
		seen[definition.ID] = struct{}{}
		if index > 0 && definitions[index-1].ID >= definition.ID {
			t.Fatalf("schemas are not strictly sorted at %s", definition.ID)
		}
	}
	first, err := MarshalBuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalBuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("schema generation is not deterministic")
	}
}

func TestDataSchemaDefinitionRejectsInvalidOrUnsafeShapes(t *testing.T) {
	invalidPattern := DataSchemaDefinition{ID: "example.invalid-pattern.v1", Type: "string", Pattern: "["}
	if err := invalidPattern.Validate(); err == nil || !strings.Contains(err.Error(), "pattern") {
		t.Fatalf("invalid pattern was not rejected: %v", err)
	}
	deep := DataSchemaDefinition{ID: "example.deep.v1", Type: "array"}
	current := &deep
	for range MaxDataSchemaDepth {
		current.Items = &DataSchemaDefinition{Type: "array"}
		current = current.Items
	}
	current.Items = &DataSchemaDefinition{Type: "string"}
	if err := deep.Validate(); err == nil || !strings.Contains(err.Error(), "depth") {
		t.Fatalf("excessive depth was not rejected: %v", err)
	}
}

func TestBuiltinDataSchemaAnnotations(t *testing.T) {
	definitions, err := BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]DataSchemaDefinition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}
	progress := byID[SchemaFileProgressV1]
	state := dataSchemaAtPath(&progress, "state")
	if state == nil || len(state.Enum) != 5 {
		t.Fatal("file progress state enum is missing")
	}
	admit := byID[SchemaManagementAdmitV1]
	permit := dataSchemaAtPath(&admit, "permit")
	if permit == nil || !permit.Sensitive || !permit.WriteOnly {
		t.Fatal("admission permit must be write-only sensitive data")
	}
	health := byID[SchemaManagementHealthV1]
	startedAt := dataSchemaAtPath(&health, "started_at_unix_ms")
	if startedAt == nil || startedAt.Format != "unix-ms" {
		t.Fatal("unix millisecond timestamp annotation is missing")
	}
}
