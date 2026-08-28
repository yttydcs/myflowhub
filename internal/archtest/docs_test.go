package archtest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var markdownLink = regexp.MustCompile(`\[[^]]+\]\(([^)]+)\)`)

func TestAcceptedArchitectureDocsHaveValidRelativeLinks(t *testing.T) {
	root := repositoryRoot(t)
	files := []string{
		"README.md", "docs/README.md", "docs/intake/README.md", "docs/requirements/README.md", "docs/features/README.md", "docs/specs/README.md", "docs/decisions/README.md",
		"docs/change/README.md", "docs/plan/README.md", "docs/lessons/README.md",
		"docs/intake/2026-08-27_node-tree-subscription-command-redesign.md",
		"docs/requirements/unified-node-runtime.md",
		"docs/features/hub.md", "docs/features/desktop.md", "docs/features/android.md", "docs/features/metrics-node.md", "docs/features/clipboard-node.md",
		"docs/features/file-transfer.md", "docs/features/flow.md",
		"docs/decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md",
		"docs/decisions/2026-08-27_single-canonical-monorepo.md",
		"docs/specs/node-tree-link-resource-architecture.md",
		"docs/specs/repository-and-module-boundaries.md",
		"docs/specs/wire-protocol-vnext.md", "docs/specs/resource-model-vnext.md", "docs/specs/subscription-vnext.md", "docs/specs/command-vnext.md",
		"docs/specs/operational-lifecycle.md", "docs/specs/resource-catalog.md", "docs/specs/notification-vnext.md", "docs/specs/file-transfer-vnext.md", "docs/specs/flow-vnext.md",
		"docs/specs/build-and-ci.md", "docs/features/embedded.md",
		"docs/change/2026-08-27_canonical-monorepo-unified-node-runtime.md",
		"docs/plan/plan_archive_2026-08-27_canonical-monorepo-unified-node-runtime.md",
		"docs/lessons/session-replacement-generation-cleanup.md",
		"docs/change/2026-08-29_extensible-resource-platform-desktop-workspace.md",
		"docs/plan/plan_archive_2026-08-29_extensible-resource-platform-desktop-workspace.md",
		"docs/lessons/desktop-binding-reconnect-and-admission-diagnostics.md",
	}
	for _, relative := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", relative, err)
			continue
		}
		for _, match := range markdownLink.FindAllStringSubmatch(string(data), -1) {
			target := strings.SplitN(match[1], "#", 2)[0]
			if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			nativeTarget := filepath.FromSlash(target)
			if filepath.IsAbs(nativeTarget) {
				continue
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), nativeTarget))
			if _, err := os.Stat(resolved); err != nil {
				t.Errorf("%s has broken link %q: %v", relative, match[1], err)
			}
		}
	}
}

func TestMigrationManifestPinsEveryLegacySource(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "migration", "sources.yaml")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	repositories := 0
	commits := 0
	dirtyPaths := 0
	sha := regexp.MustCompile(`^[0-9a-f]{40}$`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "- repository: MyFlowHub-"):
			repositories++
		case strings.HasPrefix(line, "commit: "):
			value := strings.TrimSpace(strings.TrimPrefix(line, "commit: "))
			if !sha.MatchString(value) {
				t.Errorf("invalid pinned commit %q", value)
			}
			commits++
		case strings.HasPrefix(line, "- windows/frontend/wailsjs/go/"):
			dirtyPaths++
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if repositories != 10 || commits != 10 || dirtyPaths != 16 {
		t.Fatal(fmt.Sprintf("manifest counts: repositories=%d commits=%d preserved_dirty_paths=%d", repositories, commits, dirtyPaths))
	}
}
