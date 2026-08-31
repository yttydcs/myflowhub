package protocol

import (
	"bytes"
	"encoding/base64"
	"reflect"
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
	enrollmentInit := byID[SchemaEnrollmentClientInitV1]
	permit = dataSchemaAtPath(&enrollmentInit, "permit")
	if permit == nil || !permit.Sensitive || !permit.WriteOnly {
		t.Fatal("enrollment permit must be write-only sensitive data")
	}
	admissionList := byID[SchemaAdmissionListV1]
	limit := dataSchemaAtPath(&admissionList, "limit")
	if limit == nil || limit.Minimum == nil || *limit.Minimum != 0 || limit.Maximum == nil || *limit.Maximum != MaxItems {
		t.Fatal("admission list limit bounds are missing")
	}
	health := byID[SchemaManagementHealthV1]
	startedAt := dataSchemaAtPath(&health, "started_at_unix_ms")
	if startedAt == nil || startedAt.Format != "unix-ms" {
		t.Fatal("unix millisecond timestamp annotation is missing")
	}
}

func TestBuiltinFilesystemDataSchemaAnnotations(t *testing.T) {
	definitions, err := BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]DataSchemaDefinition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}
	request, ok := byID[SchemaFilesystemReadRequestV1]
	if !ok {
		t.Fatalf("missing filesystem request schema %q", SchemaFilesystemReadRequestV1)
	}
	content, ok := byID[SchemaFilesystemContentV1]
	if !ok {
		t.Fatalf("missing filesystem content schema %q", SchemaFilesystemContentV1)
	}
	for _, schemaID := range []string{SchemaCollectionPageV1, SchemaFlowDefinitionV1, SchemaFlowRunV1} {
		if _, ok := byID[schemaID]; !ok {
			t.Fatalf("existing schema %q regressed", schemaID)
		}
	}

	assertMaxLength := func(schema *DataSchemaDefinition, path string, want int) {
		t.Helper()
		field := dataSchemaAtPath(schema, path)
		if field == nil || field.MaxLength == nil || *field.MaxLength != want {
			t.Fatalf("%s max length = %#v, want %d", path, field, want)
		}
	}
	assertBounds := func(schema *DataSchemaDefinition, path string, minimum, maximum float64) {
		t.Helper()
		field := dataSchemaAtPath(schema, path)
		if field == nil || field.Minimum == nil || *field.Minimum != minimum || field.Maximum == nil || *field.Maximum != maximum {
			t.Fatalf("%s bounds = %#v, want %v..%v", path, field, minimum, maximum)
		}
	}

	assertMaxLength(&request, "key", MaxCollectionMemberKeyBytes)
	assertMaxLength(&request, "expected_revision", MaxFilesystemRevisionBytes)
	if field := dataSchemaAtPath(&request, "expected_revision"); field.Minimum != nil {
		t.Fatalf("string expected_revision has numeric minimum: %#v", field)
	}
	assertBounds(&request, "max_bytes", 1, MaxFilesystemReadBytes)

	assertMaxLength(&content, "key", MaxCollectionMemberKeyBytes)
	assertMaxLength(&content, "content_type", MaxContentTypeBytes)
	assertMaxLength(&content, "revision", MaxFilesystemRevisionBytes)
	if field := dataSchemaAtPath(&content, "revision"); field.Minimum != nil {
		t.Fatalf("string revision has numeric minimum: %#v", field)
	}
	encoding := dataSchemaAtPath(&content, "encoding")
	wantEncodings := []string{FilesystemEncodingBase64, FilesystemEncodingUTF8}
	if encoding == nil || !reflect.DeepEqual(encoding.Enum, wantEncodings) {
		t.Fatalf("filesystem encoding enum = %#v, want %v", encoding, wantEncodings)
	}
	data := dataSchemaAtPath(&content, "data")
	wantDataLength := base64.StdEncoding.EncodedLen(MaxFilesystemReadBytes)
	if data == nil || data.MaxLength == nil || *data.MaxLength < wantDataLength {
		t.Fatalf("filesystem data max length = %#v, want at least %d", data, wantDataLength)
	}
	assertBounds(&content, "size", 0, MaxFilesystemReadBytes)
	modified := dataSchemaAtPath(&content, "modified_unix_ms")
	if modified == nil || modified.Minimum == nil || *modified.Minimum != 0 {
		t.Fatalf("modified_unix_ms minimum = %#v, want 0", modified)
	}
	for _, schema := range []*DataSchemaDefinition{&request, &content} {
		for _, forbidden := range []string{"root", "physical_path", "mount_config"} {
			if dataSchemaAtPath(schema, forbidden) != nil {
				t.Fatalf("filesystem wire schema exposes local field %q", forbidden)
			}
		}
	}
}
