package protocol

import (
	"bytes"
	"encoding/base64"
	"reflect"
	"regexp"
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

func TestBuiltinTopologyQueryDataSchemas(t *testing.T) {
	definitions, err := BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]DataSchemaDefinition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}
	assertFields := func(schema *DataSchemaDefinition, required, optional []string) {
		t.Helper()
		if schema.Type != "object" || schema.AdditionalProperties != nil || len(schema.Properties) != len(required)+len(optional) {
			t.Fatalf("unexpected object shape: %#v", schema)
		}
		want := make(map[string]bool, len(required)+len(optional))
		for _, name := range required {
			want[name] = true
		}
		for _, name := range optional {
			want[name] = false
		}
		for _, property := range schema.Properties {
			required, ok := want[property.Name]
			if !ok || property.Required != required || property.Schema == nil || property.Schema.Type == "null" {
				t.Fatalf("unexpected field or presence/nullability: %#v", property)
			}
			delete(want, property.Name)
		}
		if len(want) != 0 {
			t.Fatalf("missing fields: %v", want)
		}
	}
	assertBounds := func(schema *DataSchemaDefinition, path string, minimum, maximum float64) {
		t.Helper()
		field := dataSchemaAtPath(schema, path)
		if field == nil || field.Type != "integer" || field.MultipleOf == nil || *field.MultipleOf != 1 ||
			field.Minimum == nil || *field.Minimum != minimum || field.Maximum == nil || *field.Maximum != maximum {
			t.Fatalf("%s bounds = %#v, want integer %v..%v", path, field, minimum, maximum)
		}
	}
	for _, id := range []string{SchemaManagementTopologyChildrenRequestV1, SchemaManagementTopologyQueryRequestV1} {
		schema, ok := byID[id]
		if !ok {
			t.Fatalf("missing topology request schema %s", id)
		}
		assertFields(&schema, []string{"version", "depth"}, nil)
		assertBounds(&schema, "version", 1, 1)
		if id == SchemaManagementTopologyChildrenRequestV1 {
			assertBounds(&schema, "depth", 1, 1)
		} else {
			assertBounds(&schema, "depth", 0, MaxItems)
		}
	}
	response, ok := byID[SchemaManagementTopologyQueryV1]
	if !ok {
		t.Fatal("missing topology response schema")
	}
	assertFields(&response, []string{"version", "root_node_id", "depth", "instance_id", "revision", "nodes"}, nil)
	assertBounds(&response, "version", 1, 1)
	assertBounds(&response, "depth", 0, MaxItems)
	assertBounds(&response, "revision", 1, float64(MaxTopologyRevision))
	instance := dataSchemaAtPath(&response, "instance_id")
	if instance == nil || instance.Type != "string" || instance.MinLength == nil || *instance.MinLength != 32 || instance.MaxLength == nil || *instance.MaxLength != 32 {
		t.Fatalf("instance_id shape: %#v", instance)
	}
	pattern, err := regexp.Compile(instance.Pattern)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{topologyQueryFixture().InstanceID, strings.Repeat("0", 32), strings.Repeat("f", 32)} {
		if !pattern.MatchString(value) {
			t.Fatalf("instance pattern rejected %q", value)
		}
	}
	for _, value := range []string{"", strings.Repeat("0", 31), strings.Repeat("0", 33), strings.Repeat("F", 32), strings.Repeat("g", 32)} {
		if pattern.MatchString(value) {
			t.Fatalf("instance pattern accepted %q", value)
		}
	}
	nodes := dataSchemaAtPath(&response, "nodes")
	if nodes == nil || nodes.Type != "array" || nodes.MinItems == nil || *nodes.MinItems != 1 || nodes.MaxItems == nil || *nodes.MaxItems != MaxItems || nodes.Items == nil {
		t.Fatalf("nodes bounds: %#v", nodes)
	}
	assertFields(nodes.Items, []string{"node_id", "role", "generation", "has_children"}, []string{"parent_id", "display_name"})
	if field := dataSchemaAtPath(nodes.Items, "has_children"); field.Type != "boolean" {
		t.Fatalf("has_children type: %#v", field)
	}
	if field := dataSchemaAtPath(nodes.Items, "generation"); field.Minimum == nil || *field.Minimum != 1 {
		t.Fatalf("generation minimum: %#v", field)
	}
	for _, field := range []*DataSchemaDefinition{
		dataSchemaAtPath(&response, "root_node_id"), dataSchemaAtPath(nodes.Items, "node_id"), dataSchemaAtPath(nodes.Items, "parent_id"),
	} {
		if field == nil || field.Type != "string" || field.Format != "node-id" || field.MinLength == nil || *field.MinLength != 1 || field.MaxLength == nil || *field.MaxLength != 20 {
			t.Fatalf("Node ID bounds: %#v", field)
		}
	}
	for path, maximum := range map[string]int{"display_name": MaxLabelBytes, "role": MaxIdentifierBytes} {
		field := dataSchemaAtPath(nodes.Items, path)
		if field == nil || field.MaxLength == nil || *field.MaxLength != maximum {
			t.Fatalf("%s max length: %#v", path, field)
		}
	}
	legacy, ok := byID[SchemaManagementTopologyV1]
	if !ok {
		t.Fatal("legacy topology schema disappeared")
	}
	assertFields(&legacy, []string{"version", "epoch", "nodes"}, nil)
	assertFields(dataSchemaAtPath(&legacy, "nodes").Items, []string{"node_id", "role", "generation"}, []string{"parent_id", "display_name"})
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
	for _, schemaID := range []string{SchemaCollectionPageV1, SchemaPolicyDefinitionV1, SchemaPolicyBindingV1} {
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
