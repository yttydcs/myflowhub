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
	known := make(map[string]struct{}, len(definitions))
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
	command := exec.Command("go", "run", "./cmd/mfh-bindgen", "-out", temporary, "-schemas-out", temporarySchemas)
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
