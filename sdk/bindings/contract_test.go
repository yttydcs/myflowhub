package bindings

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/sdk/bindings/contract"
)

func TestBuiltinSchemasCoverCanonicalResourceContract(t *testing.T) {
	manifest, err := contract.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	definitions, err := protocol.BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	// Dynamic variable envelopes have canonical recursive Go/TS wire types;
	// they cannot be represented by a finite builtin business-value schema.
	known := map[string]struct{}{protocol.SchemaVariableValueV1: {}, protocol.SchemaVariableWriteV3: {}}
	for _, definition := range definitions {
		known[definition.ID] = struct{}{}
	}
	for _, resource := range manifest.Resources {
		for _, capability := range resource.Capabilities {
			for _, schemaID := range []string{capability.InputSchema, capability.OutputSchema, capability.EventSchema} {
				if schemaID == "" {
					continue
				}
				if _, exists := known[schemaID]; !exists {
					t.Errorf("resource %s capability %s references uncovered schema %s", resource.Name, capability.Name, schemaID)
				}
			}
		}
	}
}

func TestGeneratedContractIsCurrent(t *testing.T) {
	temporaryRoot := t.TempDir()
	temporary := filepath.Join(temporaryRoot, "contracts.json")
	temporarySchemas := filepath.Join(temporaryRoot, "resource-schemas.generated.json")
	temporaryTypes := filepath.Join(temporaryRoot, "variable-contract.generated.ts")
	command := exec.Command("go", "run", "./cmd/mfh-bindgen", "-out", temporary, "-schemas-out", temporarySchemas, "-types-out", temporaryTypes)
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("regenerate binding contract: %v\n%s", err, output)
	}
	want, err := os.ReadFile(filepath.Join("generated", "contracts.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(temporary)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("generated binding contract is stale; run go generate ./sdk/bindings")
	}
	for _, path := range []string{
		filepath.Join("generated", "variable-contract.generated.ts"),
		filepath.Join("..", "..", "apps", "desktop", "frontend", "src", "generated", "variable-contract.generated.ts"),
		filepath.Join("..", "..", "apps", "nodes", "metrics", "windows", "frontend", "src", "generated", "variable-contract.generated.ts"),
	} {
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(temporaryTypes)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("generated variable types %s are stale; run go generate ./sdk/bindings", path)
		}
	}
	wantSchemas, err := os.ReadFile(filepath.Join("..", "..", "apps", "desktop", "frontend", "src", "generated", "resource-schemas.generated.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotSchemas, err := os.ReadFile(temporarySchemas)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotSchemas, wantSchemas) {
		t.Fatal("generated Desktop schemas are stale; run go generate ./sdk/bindings")
	}
}
