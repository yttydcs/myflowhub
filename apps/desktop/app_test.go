package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppProfileAndExplicitSettingsReset(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	identity, err := app.IdentityJSON()
	if err != nil || !json.Valid([]byte(identity)) {
		t.Fatalf("invalid identity: %v (%s)", err, identity)
	}
	settingsJSON, err := app.SettingsJSON()
	if err != nil {
		t.Fatal(err)
	}
	var settings Settings
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		t.Fatal(err)
	}
	settings.Profile = "operator-1"
	settings.NodeID = "42"
	data, _ := json.Marshal(settings)
	if _, err := app.SaveSettingsJSON(string(data)); err != nil {
		t.Fatal(err)
	}
	identity, err = app.IdentityJSON()
	if err != nil || !strings.Contains(identity, `"node_id":"42"`) {
		t.Fatalf("profile identity was not reopened: %v (%s)", err, identity)
	}
	if _, err := app.ResetStorage("wrong"); err == nil {
		t.Fatal("reset without explicit confirmation was accepted")
	}
	if _, err := app.ResetStorage("RESET DESKTOP V1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "profiles", "operator-1", "state")); err != nil {
		t.Fatalf("explicit settings reset unexpectedly deleted identity state: %v", err)
	}
}

func TestUnsupportedSettingsRequireExplicitReset(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte(`{"version":99,"profile":"default","node_id":"2"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewApp(root); err == nil || !strings.Contains(err.Error(), "explicit reset") {
		t.Fatalf("unsupported settings were not rejected explicitly: %v", err)
	}
	if err := ResetSettings(root, "RESET DESKTOP V1"); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp(root)
	if err != nil {
		t.Fatalf("explicit reset did not recover settings: %v", err)
	}
	_ = app.Close()
}

func TestDesktopInputBoundaries(t *testing.T) {
	if _, err := parsePositiveInt64("0", "node_id"); err == nil {
		t.Fatal("zero node ID accepted")
	}
	settings := defaultSettings()
	settings.Profile = "../escape"
	if err := validateSettings(settings); err == nil {
		t.Fatal("unsafe profile name accepted")
	}
	var request Settings
	if err := decodeBoundedJSON(`{"version":1,"profile":"default","node_id":"2","unknown":true}`, &request); err == nil {
		t.Fatal("unknown JSON field should be rejected")
	}
}
