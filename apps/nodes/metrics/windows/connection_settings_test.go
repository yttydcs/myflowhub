package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func connectionTestRequest(t *testing.T) startRequestV1 {
	t.Helper()
	key, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return startRequestV1{Version: 1, StateDirectory: t.TempDir(), NodeID: "42", ParentNodeID: "1",
		Endpoint: "127.0.0.1:1", ParentPublicKey: base64.RawStdEncoding.EncodeToString(key)}
}

func TestGUIConnectionRestoresWithoutPermitAndExplicitStopPersists(t *testing.T) {
	directory := t.TempDir()
	request := connectionTestRequest(t)
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{configDirectory: directory}
	if _, err := app.Start(string(raw)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.shutdown(context.Background()) })
	identityPath := filepath.Join(request.StateDirectory, "state", "identity.json")
	identity, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatal(err)
	}
	app.shutdown(context.Background())
	saved, err := os.ReadFile(filepath.Join(directory, "connection.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved), "permit") || !strings.Contains(string(saved), `"active": true`) {
		t.Fatalf("unexpected saved connection: %s", saved)
	}
	restarted := &App{configDirectory: directory}
	restarted.startup(context.Background())
	t.Cleanup(func() { restarted.shutdown(context.Background()) })
	if restarted.runtime == nil || restarted.startupError != "" {
		t.Fatalf("restore failed: %s", restarted.startupError)
	}
	after, err := os.ReadFile(identityPath)
	if err != nil || string(identity) != string(after) {
		t.Fatal("restart changed identity")
	}
	if err := restarted.Stop(); err != nil {
		t.Fatal(err)
	}
	stopped := &App{configDirectory: directory}
	stopped.startup(context.Background())
	if stopped.runtime != nil || stopped.startupError != "" || stopped.savedConnection == nil || stopped.savedConnection.Active {
		t.Fatal("explicit stop was not retained")
	}
}

func TestGUIConnectionMissingIdentityDoesNotCreateReplacement(t *testing.T) {
	app := &App{configDirectory: t.TempDir()}
	request := connectionTestRequest(t)
	if err := app.saveConnection(savedConnectionV1{Version: 1, Active: true, Connection: request}); err != nil {
		t.Fatal(err)
	}
	app.startup(context.Background())
	if app.runtime != nil || !strings.Contains(app.startupError, "identity is unavailable") {
		t.Fatalf("missing identity: %s", app.startupError)
	}
	if _, err := os.Stat(filepath.Join(request.StateDirectory, "state")); !os.IsNotExist(err) {
		t.Fatal("automatic recovery created state")
	}
}

func TestGUIConnectionMalformedStateFailsVisibly(t *testing.T) {
	for _, content := range []string{`{`, `{"version":2}`, `{"version":1,"unexpected":true}`} {
		t.Run(content, func(t *testing.T) {
			app := &App{configDirectory: t.TempDir()}
			if err := os.WriteFile(filepath.Join(app.configDirectory, "connection.json"), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			app.startup(context.Background())
			if app.runtime != nil || app.startupError == "" {
				t.Fatal("invalid saved state was silently accepted")
			}
		})
	}
}

func TestGUIConnectionSaveFailureStopsRuntime(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	app := &App{configDirectory: blocker}
	request := connectionTestRequest(t)
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Start(string(raw)); err == nil {
		t.Fatal("save failure was ignored")
	}
	if app.runtime != nil {
		t.Fatal("untracked runtime remained after failure")
	}
}
