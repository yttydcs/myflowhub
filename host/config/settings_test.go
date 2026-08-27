package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestSettingsPersistAndEnforceRevision(t *testing.T) {
	directory := t.TempDir()
	first, err := Open(directory, 1)
	if err != nil {
		t.Fatal(err)
	}
	initial := first.Settings.Snapshot()
	updated, err := first.Settings.Replace(protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: initial.Revision, DisplayName: "hub", Values: map[string]string{"site": "home"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Settings.Replace(protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: initial.Revision}); err == nil {
		t.Fatal("stale settings update was accepted")
	}
	second, err := Open(directory, 1)
	if err != nil {
		t.Fatal(err)
	}
	got := second.Settings.Snapshot()
	if got.Revision != updated.Revision || got.DisplayName != "hub" || got.Values["site"] != "home" {
		t.Fatalf("settings did not survive restart: %#v", got)
	}
}

func TestOpenFailsOnCorruptSettingsWithoutReplacingIt(t *testing.T) {
	directory := t.TempDir()
	if _, err := Open(directory, 1); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "state", "host.json")
	corrupt := []byte(`{"version":1,"revision":0}`)
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(directory, 1); err == nil {
		t.Fatal("corrupt settings were accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, corrupt) {
		t.Fatalf("corrupt settings were modified: %v", err)
	}
}
