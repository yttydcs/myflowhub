package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/yttydcs/myflowhub"

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate architecture test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestRepositoryHasOneModuleAndNoNestedGit(t *testing.T) {
	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.Name() == ".git" && entry.IsDir() {
			t.Errorf("nested Git repository is forbidden: %s", rel)
			return filepath.SkipDir
		}
		if entry.Name() == "go.mod" && rel != "go.mod" {
			t.Errorf("nested Go module is forbidden: %s", rel)
		}
		if entry.Name() == "go.work" || entry.Name() == "go.work.sum" {
			t.Errorf("legacy workspace file is forbidden: %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestImportBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parsed, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range parsed.Imports {
			checkImport(t, filepath.ToSlash(rel), importPath(t, spec))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func importPath(t *testing.T, spec *ast.ImportSpec) string {
	t.Helper()
	value, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		t.Fatalf("invalid import literal %q: %v", spec.Path.Value, err)
	}
	return value
}

func checkImport(t *testing.T, file, imported string) {
	t.Helper()
	if strings.Contains(imported, "github.com/yttydcs/myflowhub-") {
		t.Errorf("%s imports legacy module %q", file, imported)
	}
	if strings.HasSuffix(file, "_test.go") {
		return
	}
	if !strings.HasPrefix(imported, modulePath+"/") {
		return
	}
	internal := strings.TrimPrefix(imported, modulePath+"/")
	switch {
	case strings.HasPrefix(file, "protocol/"):
		for _, forbidden := range []string{"runtime/", "transport/", "host/", "sdk/", "cmd/"} {
			if strings.HasPrefix(internal, forbidden) {
				t.Errorf("%s: protocol cannot import %q", file, imported)
			}
		}
	case strings.HasPrefix(file, "runtime/"):
		for _, forbidden := range []string{"transport/", "host/", "sdk/", "cmd/"} {
			if strings.HasPrefix(internal, forbidden) {
				t.Errorf("%s: runtime cannot import %q", file, imported)
			}
		}
	case strings.HasPrefix(file, "transport/"):
		for _, forbidden := range []string{"runtime/tree", "runtime/auth", "runtime/resource", "runtime/subscription", "runtime/command", "runtime/node", "host/", "sdk/", "cmd/"} {
			if strings.HasPrefix(internal, forbidden) {
				t.Errorf("%s: transport adapter cannot import %q", file, imported)
			}
		}
	case strings.HasPrefix(file, "sdk/"):
		for _, forbidden := range []string{"host/", "cmd/"} {
			if strings.HasPrefix(internal, forbidden) {
				if file == "sdk/bindings/android/host.go" && internal == "host/hub" {
					continue
				}
				t.Errorf("%s: SDK cannot import %q", file, imported)
			}
		}
	}
}
