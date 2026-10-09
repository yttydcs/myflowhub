package contract

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVariableTypescriptIsRecursiveAndPreservesJSON(t *testing.T) {
	content, err := VariableTypescript()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"export interface DataSchemaDefinition", "properties?: DataSchemaProperty[];", "items?: DataSchemaDefinition;",
		"additional_properties?: DataSchemaDefinition;", "schema: DataSchemaDefinition;", "nullable?: boolean;",
		"content_type?: string;", "definition?: DataSchemaDefinition;", "value?: JSONValue;", "value: JSONValue;",
		"revision: string;", "expected_revision: string;", "expected_instance_id: string;", "export interface VariableError",
	} {
		if !strings.Contains(string(content), expected) {
			t.Errorf("missing canonical wire field %q", expected)
		}
	}
	if bytes.Contains(content, []byte("value: string")) || bytes.Contains(content, []byte("VariableWriteV2")) {
		t.Fatal("old/base64 variable wire type generated")
	}
	if bytes.Count(content, []byte("export interface DataSchemaDefinition")) != 1 {
		t.Fatal("recursive type expanded or duplicated")
	}
	second, err := VariableTypescript()
	if err != nil || !bytes.Equal(content, second) {
		t.Fatal("variable TypeScript generation is not deterministic")
	}
	want, err := os.ReadFile(filepath.Join("..", "generated", "variable-contract.generated.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, want) {
		t.Fatal("SDK variable TypeScript is stale; run go generate ./sdk/bindings")
	}
}
