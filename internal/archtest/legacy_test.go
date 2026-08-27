package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionInputsDoNotReferenceLegacyArchitecture(t *testing.T) {
	root := repositoryRoot(t)
	productionRoots := []string{
		"cmd", "apps", "embedded", "feature", "host", "protocol", "runtime",
		"sdk", "transport", "scripts", ".github",
	}
	forbidden := []string{
		"github.com/yttydcs/myflowhub-",
		"repo/MyFlowHub-",
		`repo\MyFlowHub-`,
		"cmd/hub_server",
	}
	skippedDirectories := map[string]bool{
		".dart_tool":   true,
		".git":         true,
		".gradle":      true,
		"build":        true,
		"node_modules": true,
		"out":          true,
	}
	textExtensions := map[string]bool{
		".c": true, ".cmake": true, ".dart": true, ".go": true, ".h": true,
		".json": true, ".kt": true, ".kts": true, ".ps1": true, ".ts": true,
		".tsx": true, ".vue": true, ".yaml": true, ".yml": true,
	}
	for _, relativeRoot := range productionRoots {
		walkRoot := filepath.Join(root, relativeRoot)
		err := filepath.WalkDir(walkRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if path != walkRoot && skippedDirectories[entry.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if !textExtensions[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			for _, fragment := range forbidden {
				if strings.Contains(content, fragment) {
					t.Errorf("%s references forbidden legacy input %q", filepath.ToSlash(relative), fragment)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestDevelopmentEntrypointUsesCanonicalProducts(t *testing.T) {
	script := readBuildFile(t, repositoryRoot(t), "scripts/run-dev.ps1")
	for _, fragment := range []string{
		"$env:GOWORK = 'off'",
		"cmd/mfh-hub",
		"apps/desktop",
		"apps/nodes/metrics/windows",
		"-policy grant",
	} {
		if !strings.Contains(script, fragment) {
			t.Errorf("canonical development entry is missing %q", fragment)
		}
	}
}
