package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func TestConnectClientProvidesActionableAdmissionTimeout(t *testing.T) {
	err := connectionTimeoutGuidance(context.DeadlineExceeded, "")
	if err == nil || !strings.Contains(err.Error(), "一次性准入 Permit") || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("missing first-admission guidance: %v", err)
	}
	err = connectionTimeoutGuidance(context.DeadlineExceeded, `{}`)
	if err == nil || !strings.Contains(err.Error(), "无效、已使用或已过期") || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("missing permit-reissue guidance: %v", err)
	}
}

func TestAppReconnectReplacesAStartedBindingClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: t.TempDir(), NodeID: 1,
		Listeners: []hub.ListenerConfig{{Driver: tcp.Driver{}, Endpoint: "127.0.0.1:0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	profile := testProfile("reconnect", "2")
	profile.Endpoint = string(root.Endpoint)
	profile.ParentPublicKey = base64.RawStdEncoding.EncodeToString(root.Runtime.Identity.PublicKey)
	profileJSON, _ := json.Marshal(profile)
	if _, err := app.SaveProfileJSON(string(profileJSON)); err != nil {
		t.Fatal(err)
	}
	identityJSON, err := app.IdentityJSON()
	if err != nil {
		t.Fatal(err)
	}
	var identity struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal([]byte(identityJSON), &identity); err != nil {
		t.Fatal(err)
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(identity.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := root.Runtime.Admission.Issue(2, ed25519.PublicKey(publicKey), "desktop-reconnect", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	permitJSON, _ := protocol.EncodeJSONPayload(&permit, protocol.DefaultMaxPayload)
	requestJSON, _ := json.Marshal(LoginRequest{Profile: profile, PermitJSON: string(permitJSON)})
	if _, err := app.LoginJSON(string(requestJSON)); err != nil {
		t.Fatal(err)
	}
	if err := app.Disconnect(); err != nil {
		t.Fatal(err)
	}
	started, err := app.currentClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := started.TrustParent(1, profile.ParentPublicKey); err != nil {
		t.Fatal(err)
	}
	if err := started.StartTCP("127.0.0.1:1", 1, ""); err != nil {
		t.Fatal(err)
	}
	if err := app.Connect(); err != nil {
		t.Fatalf("reconnect did not replace the already-started binding client: %v", err)
	}
	status, err := app.StatusJSON()
	if err != nil || !strings.Contains(status, `"state":"connected"`) {
		t.Fatalf("unexpected reconnect status: %v (%s)", err, status)
	}
}

func testProfile(id, nodeID string) Profile {
	return Profile{
		ID: id, Name: "Profile " + id, NodeID: nodeID, Endpoint: "127.0.0.1:9540", ParentNodeID: "1",
		ParentPublicKey: base64.RawStdEncoding.EncodeToString(make([]byte, 32)), AutoConnect: true,
	}
}

func TestAppSupportsIsolatedPersistentProfiles(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	status, err := app.StatusJSON()
	if err != nil || !strings.Contains(status, `"signed_out"`) {
		t.Fatalf("new desktop did not start signed out: %v (%s)", err, status)
	}
	first := testProfile("operator-1", "42")
	data, _ := json.Marshal(first)
	if _, err := app.SaveProfileJSON(string(data)); err != nil {
		t.Fatal(err)
	}
	identity, err := app.IdentityJSON()
	if err != nil || !strings.Contains(identity, `"node_id":"42"`) {
		t.Fatalf("first profile identity mismatch: %v (%s)", err, identity)
	}
	second := testProfile("operator-2", "43")
	data, _ = json.Marshal(second)
	if _, err := app.SaveProfileJSON(string(data)); err != nil {
		t.Fatal(err)
	}
	identity, err = app.IdentityJSON()
	if err != nil || !strings.Contains(identity, `"node_id":"43"`) {
		t.Fatalf("second profile identity mismatch: %v (%s)", err, identity)
	}
	if err := app.SwitchProfile(first.ID); err != nil {
		t.Fatal(err)
	}
	identity, _ = app.IdentityJSON()
	if !strings.Contains(identity, `"node_id":"42"`) {
		t.Fatalf("switch did not restore isolated identity: %s", identity)
	}
	settingsJSON, _ := app.SettingsJSON()
	var settings struct {
		Settings
		CredentialMode string `json:"credential_mode"`
	}
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil || len(settings.Profiles) != 2 || settings.ActiveProfileID != first.ID || !strings.Contains(settings.CredentialMode, "session-only") {
		t.Fatalf("unexpected profile index: %v (%s)", err, settingsJSON)
	}
	if _, err := app.ResetStorage("wrong"); err == nil {
		t.Fatal("reset without explicit confirmation was accepted")
	}
	if _, err := app.ResetStorage("RESET DESKTOP V2"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "profiles", first.ID, "state")); err != nil {
		t.Fatalf("profile index reset unexpectedly deleted identity state: %v", err)
	}
}

func TestUnsupportedSettingsRequireExplicitReset(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte(`{"version":99,"profiles":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewApp(root); err == nil || !strings.Contains(err.Error(), "explicit reset") {
		t.Fatalf("unsupported settings were not rejected explicitly: %v", err)
	}
	if err := ResetSettings(root, "RESET DESKTOP V2"); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp(root)
	if err != nil {
		t.Fatalf("explicit reset did not recover settings: %v", err)
	}
	_ = app.Close()
}

func TestViewStoreRevisionAndCorruptionSafety(t *testing.T) {
	store, err := newViewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	view := ViewDefinition{ID: "dashboard", Name: "Dashboard", Widgets: []ViewWidget{{
		ID: "cpu", OwnerNodeID: "3", ResourceName: "metrics/cpu", Renderer: "mfh.variable", W: 6, H: 4,
	}}}
	saved, err := store.saveView(view)
	if err != nil || saved.Revision != 1 {
		t.Fatalf("save view: %v (%+v)", err, saved)
	}
	if _, err := store.saveView(view); err == nil || !strings.Contains(err.Error(), "revision conflict") {
		t.Fatalf("stale view update accepted: %v", err)
	}
	saved.Name = "Operations"
	updated, err := store.saveView(saved)
	if err != nil || updated.Revision != 2 {
		t.Fatalf("update view: %v (%+v)", err, updated)
	}
	if err := os.WriteFile(store.path(), []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.saveView(updated); err == nil || !strings.Contains(err.Error(), "was not overwritten") {
		t.Fatalf("corrupt view store was silently replaced: %v", err)
	}
}

func TestDesktopInputBoundaries(t *testing.T) {
	if _, err := parsePositiveInt64("0", "node_id"); err == nil {
		t.Fatal("zero node ID accepted")
	}
	profile := testProfile("../escape", "2")
	if err := validateProfile(profile); err == nil {
		t.Fatal("unsafe profile ID accepted")
	}
	var request Profile
	if err := decodeBoundedJSON(`{"id":"default","unknown":true}`, &request); err == nil {
		t.Fatal("unknown JSON field should be rejected")
	}
}
