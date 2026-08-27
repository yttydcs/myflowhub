package migrationtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

type item struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	Disposition string `json:"disposition"`
	State       string `json:"state"`
	Reason      string `json:"reason"`
}

type inventory struct {
	SchemaVersion int    `json:"schema_version"`
	Capabilities  []item `json:"capabilities"`
	BuildEntries  []item `json:"build_entries"`
}

type dirtyPath struct {
	Status string `json:"status"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type source struct {
	Name      string      `json:"name"`
	Commit    string      `json:"commit"`
	Tree      string      `json:"tree"`
	FileCount int         `json:"file_count"`
	Dirty     []dirtyPath `json:"dirty"`
}

type sourceAudit struct {
	SchemaVersion int      `json:"schema_version"`
	Repositories  []source `json:"repositories"`
}

func root(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate migration test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readJSON(t *testing.T, name string, target any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(t), "migration", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
}

func TestInventoryHasUniqueExplicitDispositions(t *testing.T) {
	var manifest inventory
	readJSON(t, "inventory.json", &manifest)
	if manifest.SchemaVersion != 1 || len(manifest.Capabilities) == 0 || len(manifest.BuildEntries) == 0 {
		t.Fatal("inventory must use schema v1 and contain capabilities and build entries")
	}
	seen := make(map[string]struct{})
	allowedDisposition := map[string]bool{"migrate": true, "replace": true, "drop": true}
	allowedState := map[string]bool{"pending": true, "implemented": true, "verified": true}
	for _, entry := range append(manifest.Capabilities, manifest.BuildEntries...) {
		if entry.ID == "" || entry.Source == "" || entry.Target == "" {
			t.Errorf("inventory entry has empty identity/source/target: %+v", entry)
		}
		if _, exists := seen[entry.ID]; exists {
			t.Errorf("duplicate inventory ID %q", entry.ID)
		}
		seen[entry.ID] = struct{}{}
		if !allowedDisposition[entry.Disposition] || !allowedState[entry.State] {
			t.Errorf("%s has invalid disposition/state %q/%q", entry.ID, entry.Disposition, entry.State)
		}
		if entry.Disposition == "drop" && strings.TrimSpace(entry.Reason) == "" {
			t.Errorf("%s is dropped without a reason", entry.ID)
		}
	}
}

func TestSourceAuditMatchesSourceManifest(t *testing.T) {
	var audit sourceAudit
	readJSON(t, "source-audit.json", &audit)
	if audit.SchemaVersion != 1 || len(audit.Repositories) != 10 {
		t.Fatalf("source audit must describe exactly 10 schema-v1 repositories, got %d", len(audit.Repositories))
	}
	data, err := os.ReadFile(filepath.Join(root(t), "migration", "sources.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(data)
	hex40 := regexp.MustCompile(`^[0-9a-f]{40}$`)
	seen := make(map[string]struct{})
	for _, repository := range audit.Repositories {
		if _, exists := seen[repository.Name]; exists {
			t.Errorf("duplicate source %q", repository.Name)
		}
		seen[repository.Name] = struct{}{}
		if !hex40.MatchString(repository.Commit) || !hex40.MatchString(repository.Tree) || repository.FileCount <= 0 {
			t.Errorf("invalid source audit entry for %s", repository.Name)
		}
		if !strings.Contains(yaml, "repository: "+repository.Name) || !strings.Contains(yaml, "commit: "+repository.Commit) {
			t.Errorf("sources.yaml does not pin %s@%s", repository.Name, repository.Commit)
		}
	}
	metrics, ok := seen["MyFlowHub-MetricsNode"]
	_ = metrics
	if !ok {
		t.Fatal("MetricsNode source is missing")
	}
	for _, repository := range audit.Repositories {
		if repository.Name == "MyFlowHub-MetricsNode" && len(repository.Dirty) != 29 {
			t.Fatalf("MetricsNode dirty exclusion must contain 29 exact paths, got %d", len(repository.Dirty))
		}
		if repository.Name != "MyFlowHub-MetricsNode" && len(repository.Dirty) != 0 {
			t.Errorf("unexpected dirty paths for %s", repository.Name)
		}
		for _, dirty := range repository.Dirty {
			if dirty.Path == "" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(dirty.SHA256) {
				t.Errorf("invalid dirty path record in %s: %+v", repository.Name, dirty)
			}
		}
	}
}

func TestEveryPinnedSourceHasAnInventoryDecision(t *testing.T) {
	var audit sourceAudit
	var manifest inventory
	readJSON(t, "source-audit.json", &audit)
	readJSON(t, "inventory.json", &manifest)
	entries := append(manifest.Capabilities, manifest.BuildEntries...)
	for _, repository := range audit.Repositories {
		covered := false
		for _, entry := range entries {
			if strings.Contains(entry.Source, repository.Name) {
				covered = true
				break
			}
		}
		if !covered {
			t.Errorf("%s has no capability or build disposition", repository.Name)
		}
	}
}

func TestFinalInventoryHasNoUnresolvedItems(t *testing.T) {
	var manifest inventory
	readJSON(t, "inventory.json", &manifest)
	allowedImplemented := map[string]bool{
		"embedded.esp32":   true,
		"embedded.esp-idf": true,
	}
	for _, entry := range append(manifest.Capabilities, manifest.BuildEntries...) {
		switch entry.State {
		case "verified":
		case "implemented":
			if !allowedImplemented[entry.ID] {
				t.Errorf("%s remains implemented without an explicit unavailable hardware gate", entry.ID)
			}
		default:
			t.Errorf("%s remains unresolved in final inventory state %q", entry.ID, entry.State)
		}
	}
}
