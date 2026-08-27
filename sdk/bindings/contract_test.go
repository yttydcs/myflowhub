package bindings

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGeneratedContractIsCurrent(t *testing.T) {
	temporary := filepath.Join(t.TempDir(), "contracts.json")
	command := exec.Command("go", "run", "./cmd/mfh-bindgen", "-out", temporary)
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
}
