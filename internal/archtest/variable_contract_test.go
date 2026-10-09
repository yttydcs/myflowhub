package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVariableContractHasNoProductionLegacyPath(t *testing.T) {
	root := repositoryRoot(t)
	for _, directory := range []string{"protocol", "runtime", "sdk", "apps", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				switch entry.Name() {
				case "node_modules", "dist", "build", "out", ".dart_tool", ".git":
					return filepath.SkipDir
				}
				return nil
			}
			name := entry.Name()
			extension := filepath.Ext(name)
			if extension != ".go" && extension != ".ts" && extension != ".tsx" && extension != ".json" {
				return nil
			}
			if strings.HasSuffix(name, "_test.go") || strings.Contains(name, ".test.") || strings.Contains(name, "test-support") || strings.Contains(name, ".spec.") || strings.HasPrefix(name, "package") || strings.HasPrefix(name, "tsconfig") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(data)
			banned := []string{"VariableWriteV2", "mfh.variable-write.v2", "mfh.metrics.sample.v1", "mfh.metrics.control.v1", "mfh.metrics.control-result.v1"}
			for _, token := range banned {
				if strings.Contains(text, token) {
					t.Errorf("superseded variable contract %q in production source %s", token, path)
				}
			}
			if strings.Contains(filepath.ToSlash(path), "apps/nodes/metrics/") {
				for _, token := range []string{"type SampleV1 ", "type ControlV1 ", "type ControlResultV1 ", "func ControlResourceName("} {
					if strings.Contains(text, token) {
						t.Errorf("superseded Metrics API %q in %s", token, path)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
