package archtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalBuildEntrypointAndPinnedToolchain(t *testing.T) {
	root := repositoryRoot(t)
	toolchainData, err := os.ReadFile(filepath.Join(root, "build", "toolchain.json"))
	if err != nil {
		t.Fatal(err)
	}
	var toolchain struct {
		Go      string `json:"go"`
		Node    string `json:"node"`
		Wails   string `json:"wails"`
		Flutter string `json:"flutter"`
	}
	if err := json.Unmarshal(toolchainData, &toolchain); err != nil {
		t.Fatalf("parse build/toolchain.json: %v", err)
	}
	if toolchain.Go != "1.24.x" || toolchain.Node != "22.x" ||
		toolchain.Wails != "v2.11.0" || toolchain.Flutter != "3.47.1" {
		t.Errorf("unexpected pinned toolchain: %+v", toolchain)
	}

	entry := readBuildFile(t, root, "scripts/mfh.ps1")
	for _, fragment := range []string{
		"$env:GOWORK = 'off'", "MFH_FLUTTER_HOME",
		"'core'", "'hub'", "'desktop'", "'metrics'", "'clipboard'", "'generated'",
	} {
		if !strings.Contains(entry, fragment) {
			t.Errorf("canonical build entry is missing %q", fragment)
		}
	}
}

func TestCIHasEveryProductGateAndNoPathFilter(t *testing.T) {
	workflow := readBuildFile(t, repositoryRoot(t), ".github/workflows/ci.yml")
	for _, job := range []string{
		"  core:", "  generated:", "  desktop-windows:",
		"  metrics-windows:", "  clipboard:",
	} {
		if !strings.Contains(workflow, job) {
			t.Errorf("canonical CI is missing job %q", strings.TrimSpace(job))
		}
	}
	if strings.Contains(workflow, "    paths:") || strings.Contains(workflow, "    paths-ignore:") {
		t.Fatal("canonical CI must not use path filters because shared runtime changes affect every product")
	}
}

func TestRetiredPlatformsAreNotBuildInputs(t *testing.T) {
	root := repositoryRoot(t)
	for _, path := range []string{
		"apps/android/app/build.gradle.kts", "apps/nodes/metrics/android/app/build.gradle.kts",
		"apps/nodes/clipboard/app/android/app/build.gradle.kts", "internal/tools/mobile.go",
		"transport/rfcomm/android_provider.go", "embedded/c/CMakeLists.txt",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Errorf("retired build input still exists or cannot be inspected: %s (%v)", path, err)
		}
	}
	for path, fragments := range map[string][]string{
		"scripts/mfh.ps1":          {"'android'", "'embedded'", "gomobile", "gradlew", "MFH_JAVA_HOME"},
		".github/workflows/ci.yml": {"  android:", "  metrics-android:", "  embedded-host:", "  esp32-idf6:", "setup-android", "gomobile"},
		"go.mod":                   {"golang.org/x/mobile"},
	} {
		content := readBuildFile(t, root, path)
		for _, fragment := range fragments {
			if strings.Contains(content, fragment) {
				t.Errorf("%s still requires retired platform input %q", path, fragment)
			}
		}
	}
}

func readBuildFile(t *testing.T, root, relative string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(data)
}
