package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
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

func TestPrepareProfileCreatesStableInactivePublicIdentity(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	profile := testProfile("first-admission", "2")
	profileJSON, _ := json.Marshal(profile)
	firstJSON, err := app.PrepareProfileJSON(string(profileJSON))
	if err != nil {
		t.Fatal(err)
	}
	var first preparedProfile
	if err := json.Unmarshal([]byte(firstJSON), &first); err != nil {
		t.Fatal(err)
	}
	if first.Profile.ID != profile.ID || first.Identity.NodeID != profile.NodeID || first.Identity.PublicKey == "" {
		t.Fatalf("unexpected prepared profile: %+v", first)
	}
	if strings.Contains(firstJSON, "private") {
		t.Fatalf("prepared profile exposed private identity material: %s", firstJSON)
	}

	settingsJSON, err := app.SettingsJSON()
	if err != nil {
		t.Fatal(err)
	}
	var settings struct{ Settings }
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		t.Fatal(err)
	}
	if settings.ActiveProfileID != "" || len(settings.Profiles) != 1 || settings.Profiles[0].ID != profile.ID {
		t.Fatalf("prepared profile was activated or not persisted: %s", settingsJSON)
	}
	if _, err := app.IdentityJSON(); err == nil {
		t.Fatal("prepared profile unexpectedly installed an active client")
	}

	secondJSON, err := app.PrepareProfileJSON(string(profileJSON))
	if err != nil {
		t.Fatal(err)
	}
	var second preparedProfile
	if err := json.Unmarshal([]byte(secondJSON), &second); err != nil {
		t.Fatal(err)
	}
	if second.Identity != first.Identity {
		t.Fatalf("prepared identity changed: first=%+v second=%+v", first.Identity, second.Identity)
	}
}

func TestAuthorityProfileEnrollsWithoutClientAssignedNodeOrParentKey(t *testing.T) {
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

	desktopRoot := t.TempDir()
	app, err := NewApp(desktopRoot)
	if err != nil {
		t.Fatal(err)
	}
	profile := Profile{
		ID: "authority-device", Name: "Authority Device", EnrollmentMode: "authority",
		Endpoint: string(root.Endpoint), AutoConnect: true,
	}
	profileJSON, _ := json.Marshal(profile)
	preparedJSON, err := app.PrepareProfileJSON(string(profileJSON))
	if err != nil {
		t.Fatal(err)
	}
	var prepared preparedProfile
	if err := json.Unmarshal([]byte(preparedJSON), &prepared); err != nil {
		t.Fatal(err)
	}
	if prepared.Identity.NodeID != "" || prepared.Identity.PublicKey == "" {
		t.Fatalf("unregistered identity unexpectedly had a Node ID: %+v", prepared.Identity)
	}
	deviceKey, err := base64.RawStdEncoding.DecodeString(prepared.Identity.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := root.EnrollmentAuthority.IssuePermit(
		"00112233445566778899aabbccddeeff",
		auth.DevicePublicKeyFingerprint(ed25519.PublicKey(deviceKey)),
		root.Runtime.Identity.NodeID, false, "desktop", time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	permitJSON, err := protocol.EncodeJSONPayload(&permit, protocol.EnrollmentMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	requestJSON, _ := json.Marshal(LoginRequest{Profile: profile, PermitJSON: string(permitJSON)})
	savedJSON, err := app.LoginJSON(string(requestJSON))
	if err != nil {
		t.Fatal(err)
	}
	var saved Profile
	if err := json.Unmarshal([]byte(savedJSON), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.NodeID == "" || saved.ParentNodeID != "1" || saved.ParentPublicKey == "" || saved.AuthorityNodeID != "1" || saved.AuthorityPublicKey == "" {
		t.Fatalf("Authority binding was not hydrated into the Profile: %+v", saved)
	}
	if status, err := app.StatusJSON(); err != nil || !strings.Contains(status, `"state":"connected"`) {
		t.Fatalf("Authority-enrolled Profile did not enter ordinary Join: %v (%s)", err, status)
	}
	settingsData, err := os.ReadFile(filepath.Join(desktopRoot, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	logsJSON, err := app.LogsJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{permit.PermitID, permit.Signature} {
		if strings.Contains(string(settingsData), secret) || strings.Contains(logsJSON, secret) {
			t.Fatal("Enrollment Permit material was persisted in settings or logs")
		}
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewApp(desktopRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err := reopened.Connect(); err != nil {
		t.Fatalf("persisted Authority Grant did not reconnect without another enrollment: %v", err)
	}
}

func TestPrepareProfileRejectsActiveProfile(t *testing.T) {
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	profile := testProfile("active", "2")
	profileJSON, _ := json.Marshal(profile)
	if _, err := app.SaveProfileJSON(string(profileJSON)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.PrepareProfileJSON(string(profileJSON)); err == nil || !strings.Contains(err.Error(), "active profile") {
		t.Fatalf("expected active profile preparation rejection, got %v", err)
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
	view := ViewDefinition{
		ID: "dashboard", Name: "Dashboard",
		Widgets:    []ViewWidget{{ID: "cpu", OwnerNodeID: "3", ResourceName: "metrics/cpu", Renderer: "mfh.variable"}},
		LayoutRoot: &ViewLayoutNode{Kind: "leaf", WidgetID: "cpu"},
	}
	saved, err := store.saveView(view)
	if err != nil || saved.Revision != 1 {
		t.Fatalf("save view: %v (%+v)", err, saved)
	}
	if saved.LayoutRoot == nil || saved.LayoutRoot.WidgetID != "cpu" {
		t.Fatalf("view layout was not persisted exactly: %+v", saved.LayoutRoot)
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

func TestViewLayoutValidation(t *testing.T) {
	base := ViewDefinition{
		ID: "dashboard", Name: "Dashboard", Revision: 1,
		Widgets:    []ViewWidget{{ID: "cpu", OwnerNodeID: "3", ResourceName: "metrics/cpu", Renderer: "mfh.variable"}},
		LayoutRoot: &ViewLayoutNode{Kind: "leaf", WidgetID: "cpu"},
	}
	if err := validateView(base); err != nil {
		t.Fatalf("valid v3 view rejected: %v", err)
	}
	invalid := []*ViewLayoutNode{
		nil,
		{Kind: "diagonal"},
		{Kind: "leaf", WidgetID: "cpu", Children: []ViewLayoutNode{}},
		{Kind: "split", Axis: "horizontal", Children: []ViewLayoutNode{{Kind: "leaf", WidgetID: "cpu"}}, Weights: []float64{1}},
		{Kind: "split", Axis: "diagonal", Children: []ViewLayoutNode{{Kind: "leaf", WidgetID: "cpu"}, {Kind: "leaf", WidgetID: "cpu"}}, Weights: []float64{0.5, 0.5}},
		{Kind: "split", Axis: "horizontal", Children: []ViewLayoutNode{{Kind: "leaf", WidgetID: "cpu"}, {Kind: "leaf", WidgetID: "memory"}}, Weights: []float64{math.NaN(), 0.5}},
	}
	for _, layout := range invalid {
		candidate := base
		candidate.LayoutRoot = layout
		if err := validateView(candidate); err == nil {
			t.Fatalf("invalid layout accepted: %+v", layout)
		}
	}

	twoWidgets := base
	twoWidgets.Widgets = append(twoWidgets.Widgets, ViewWidget{ID: "memory", OwnerNodeID: "3", ResourceName: "metrics/memory", Renderer: "mfh.variable"})
	twoWidgets.LayoutRoot = &ViewLayoutNode{
		Kind: "split", Axis: "horizontal",
		Children: []ViewLayoutNode{{Kind: "leaf", WidgetID: "cpu"}, {Kind: "leaf", WidgetID: "memory"}},
		Weights:  []float64{0.4, 0.6},
	}
	if err := validateView(twoWidgets); err != nil {
		t.Fatalf("valid split rejected: %v", err)
	}
	redundant := twoWidgets
	redundant.LayoutRoot = &ViewLayoutNode{
		Kind: "split", Axis: "horizontal", Weights: []float64{0.5, 0.5},
		Children: []ViewLayoutNode{
			{Kind: "leaf", WidgetID: "cpu"},
			{Kind: "split", Axis: "horizontal", Weights: []float64{0.5, 0.5}, Children: []ViewLayoutNode{
				{Kind: "leaf", WidgetID: "memory"}, {Kind: "leaf", WidgetID: "extra"},
			}},
		},
	}
	redundant.Widgets = append(redundant.Widgets, ViewWidget{ID: "extra", OwnerNodeID: "3", ResourceName: "metrics/extra", Renderer: "mfh.variable"})
	if err := validateView(redundant); err == nil || !strings.Contains(err.Error(), "must be flattened") {
		t.Fatalf("redundant same-axis split accepted: %v", err)
	}
}

func TestLegacyViewDocumentMigratesBeforeSave(t *testing.T) {
	store, err := newViewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	legacy := `{"version":2,"views":[{"id":"dashboard","name":"Dashboard","revision":1,"widgets":[{"id":"cpu","owner_node_id":"3","resource_name":"metrics/cpu","renderer":"mfh.variable","x":0,"y":0,"w":7,"h":12},{"id":"memory","owner_node_id":"3","resource_name":"metrics/memory","renderer":"mfh.variable","x":7,"y":0,"w":5,"h":12},{"id":"events","owner_node_id":"3","resource_name":"metrics/events","renderer":"mfh.stream","x":0,"y":12,"w":12,"h":12}],"created_at_unix_ms":1,"updated_at_unix_ms":1}]}`
	legacyData := []byte(legacy)
	if err := os.WriteFile(store.path(), legacyData, 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := store.load()
	if err != nil {
		t.Fatalf("load legacy view: %v", err)
	}
	root := document.Views[0].LayoutRoot
	if document.Version != desktopViewsVersion || root == nil || root.Kind != "split" || root.Axis != "vertical" || len(root.Children) != 2 {
		t.Fatalf("legacy view was not migrated in memory: %+v", document)
	}
	if root.Children[0].Axis != "horizontal" || len(root.Children[0].Children) != 2 || root.Children[1].WidgetID != "events" {
		t.Fatalf("legacy grid topology changed unexpectedly: %+v", root)
	}
	saved, err := store.saveView(document.Views[0])
	if err != nil {
		t.Fatalf("save migrated view: %v", err)
	}
	if saved.Revision != 2 {
		t.Fatalf("migrated view revision = %d, want 2", saved.Revision)
	}
	data, err := os.ReadFile(store.path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"version": 3`) || strings.Contains(string(data), `"x":`) {
		t.Fatalf("migrated document was not persisted as canonical v3: %s", data)
	}
	snapshot, err := os.ReadFile(store.preV3SnapshotPath())
	if err != nil {
		t.Fatalf("read pre-v3 snapshot: %v", err)
	}
	if string(snapshot) != legacy {
		t.Fatalf("pre-v3 snapshot changed: %s", snapshot)
	}
	saved.Name = "Dashboard 2"
	if _, err := store.saveView(saved); err != nil {
		t.Fatalf("save v3 view: %v", err)
	}
	snapshotAgain, err := os.ReadFile(store.preV3SnapshotPath())
	if err != nil || string(snapshotAgain) != legacy {
		t.Fatalf("pre-v3 snapshot was overwritten: %v %s", err, snapshotAgain)
	}
}

func TestViewMigrationVariantsAndStrictFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		raw  string
		axis string
	}{
		{
			name: "v1 two widget grid",
			raw:  `{"version":1,"views":[{"id":"dashboard","name":"Dashboard","revision":1,"widgets":[{"id":"cpu","owner_node_id":"3","resource_name":"metrics/cpu","renderer":"mfh.variable","x":0,"y":0,"w":8,"h":24},{"id":"memory","owner_node_id":"3","resource_name":"metrics/memory","renderer":"mfh.variable","x":8,"y":0,"w":4,"h":24}],"created_at_unix_ms":1,"updated_at_unix_ms":1}]}`,
			axis: "horizontal",
		},
		{
			name: "v2 vertical layout",
			raw:  `{"version":2,"views":[{"id":"dashboard","name":"Dashboard","revision":1,"widgets":[{"id":"cpu","owner_node_id":"3","resource_name":"metrics/cpu","renderer":"mfh.variable","x":0,"y":0,"w":7,"h":24},{"id":"memory","owner_node_id":"3","resource_name":"metrics/memory","renderer":"mfh.variable","x":7,"y":0,"w":5,"h":24}],"layout":{"direction":"vertical","split_ratio":0.637},"created_at_unix_ms":1,"updated_at_unix_ms":1}]}`,
			axis: "vertical",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, source, err := decodeViewDocument([]byte(test.raw))
			if err != nil {
				t.Fatal(err)
			}
			if source >= desktopViewsVersion || document.Views[0].LayoutRoot == nil || document.Views[0].LayoutRoot.Axis != test.axis {
				t.Fatalf("unexpected migration: source=%d document=%+v", source, document)
			}
		})
	}

	for _, raw := range []string{
		`{"version":99,"views":[]}`,
		`{"version":1,"views":[],"unknown":true}`,
		`{"version":1,"views":[{"id":"dashboard","name":"Dashboard","revision":1,"widgets":[],"layout":{"direction":"horizontal","split_ratio":0.5},"created_at_unix_ms":1,"updated_at_unix_ms":1}]}`,
		`{"version":3,"views":[{"id":"dashboard","name":"Dashboard","revision":1,"widgets":[{"id":"cpu","owner_node_id":"3","resource_name":"metrics/cpu","renderer":"mfh.variable"}],"layout_root":{"kind":"leaf","widget_id":"missing"},"created_at_unix_ms":1,"updated_at_unix_ms":1}]}`,
	} {
		if _, _, err := decodeViewDocument([]byte(raw)); err == nil {
			t.Fatalf("invalid view document accepted: %s", raw)
		}
	}
}

func TestInvalidPreV3SnapshotBlocksMigrationWrite(t *testing.T) {
	store, err := newViewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	legacy := []byte(`{"version":1,"views":[{"id":"dashboard","name":"Dashboard","revision":1,"widgets":[{"id":"cpu","owner_node_id":"3","resource_name":"metrics/cpu","renderer":"mfh.variable","x":0,"y":0,"w":12,"h":24}],"created_at_unix_ms":1,"updated_at_unix_ms":1}]}`)
	if err := os.WriteFile(store.path(), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.preV3SnapshotPath(), []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := store.load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.saveView(document.Views[0]); err == nil || !strings.Contains(err.Error(), "snapshot is invalid") {
		t.Fatalf("invalid snapshot did not block migration: %v", err)
	}
	current, err := os.ReadFile(store.path())
	if err != nil || string(current) != string(legacy) {
		t.Fatalf("legacy file changed after failed migration: %v %s", err, current)
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
