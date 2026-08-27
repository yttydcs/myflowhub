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
		Go                string `json:"go"`
		Node              string `json:"node"`
		Java              string `json:"java"`
		AndroidCompileSDK int    `json:"android_compile_sdk"`
		AndroidMinSDK     int    `json:"android_min_sdk"`
		Wails             string `json:"wails"`
		Flutter           string `json:"flutter"`
	}
	if err := json.Unmarshal(toolchainData, &toolchain); err != nil {
		t.Fatalf("parse build/toolchain.json: %v", err)
	}
	if toolchain.Go != "1.24.x" || toolchain.Node != "22.x" || toolchain.Java != "17" ||
		toolchain.AndroidCompileSDK != 34 || toolchain.AndroidMinSDK != 26 ||
		toolchain.Wails != "v2.11.0" || toolchain.Flutter != "3.47.1" {
		t.Errorf("unexpected pinned toolchain: %+v", toolchain)
	}

	entry := readBuildFile(t, root, "scripts/mfh.ps1")
	for _, fragment := range []string{
		"$env:GOWORK = 'off'", "MFH_JAVA_HOME", "MFH_FLUTTER_HOME", "MFH_SHORT_TEMP",
		"'core'", "'hub'", "'desktop'", "'android'", "'metrics'", "'clipboard'", "'embedded'", "'generated'",
	} {
		if !strings.Contains(entry, fragment) {
			t.Errorf("canonical build entry is missing %q", fragment)
		}
	}
}

func TestCIHasEveryProductGateAndNoPathFilter(t *testing.T) {
	workflow := readBuildFile(t, repositoryRoot(t), ".github/workflows/ci.yml")
	for _, job := range []string{
		"  core:", "  generated:", "  desktop-windows:", "  android:",
		"  metrics-windows:", "  metrics-android:", "  clipboard:",
		"  embedded-host:", "  esp32-idf6:",
	} {
		if !strings.Contains(workflow, job) {
			t.Errorf("canonical CI is missing job %q", strings.TrimSpace(job))
		}
	}
	if strings.Contains(workflow, "    paths:") || strings.Contains(workflow, "    paths-ignore:") {
		t.Fatal("canonical CI must not use path filters because shared runtime changes affect every product")
	}
}

func TestAndroidBindingsAreRequiredBuildInputs(t *testing.T) {
	root := repositoryRoot(t)
	for path, fragment := range map[string]string{
		"apps/android/app/build.gradle.kts":                     "myflowhub.aar is required",
		"apps/nodes/metrics/android/app/build.gradle.kts":       "metricsmobile.aar is required",
		"apps/nodes/clipboard/app/android/app/build.gradle.kts": "libs/clipboardmobile.aar",
	} {
		if !strings.Contains(readBuildFile(t, root, path), fragment) {
			t.Errorf("%s does not require canonical generated binding %q", path, fragment)
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
