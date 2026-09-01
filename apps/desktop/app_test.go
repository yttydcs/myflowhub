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
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/sdk/bindings"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
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

func TestAppConnectUsesOneLeafHostAndItsManagedReconnectPath(t *testing.T) {
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
	app.mu.Lock()
	started := app.active
	app.mu.Unlock()
	if started == nil || started.host == nil || started.client == nil {
		t.Fatal("login did not install a complete profile runtime")
	}
	if started.host.Role() != nodehost.RoleLeaf || len(started.host.Endpoints()) != 0 {
		t.Fatalf("desktop runtime is not a parent-only leaf: role=%s endpoints=%v", started.host.Role(), started.host.Endpoints())
	}
	hostClient := started.host.Client()
	if err := app.Connect(); err != nil {
		t.Fatalf("connect did not reuse the Host-owned supervisor: %v", err)
	}
	app.mu.Lock()
	reused := app.active
	app.mu.Unlock()
	if reused != started || reused.host.Client() != hostClient {
		t.Fatal("Connect replaced the Host or created a second SDK client")
	}
	status, err := app.StatusJSON()
	if err != nil || !strings.Contains(status, `"state":"connected"`) {
		t.Fatalf("unexpected reconnect status: %v (%s)", err, status)
	}

	// A different Profile whose persisted parent trust conflicts with its new
	// configuration must not replace the active runtime.
	rejected := testProfile("rejected", "3")
	rejected.Endpoint = string(root.Endpoint)
	rejected.ParentPublicKey = profile.ParentPublicKey
	preparedRejected, _ := json.Marshal(rejected)
	if _, err := app.PrepareProfileJSON(string(preparedRejected)); err != nil {
		t.Fatal(err)
	}
	rejected.ParentPublicKey = base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
	rejectedJSON, _ := json.Marshal(rejected)
	if _, err := app.SaveProfileJSON(string(rejectedJSON)); err == nil {
		t.Fatal("conflicting parent identity was unexpectedly saved")
	}
	app.mu.Lock()
	restored := app.active
	app.mu.Unlock()
	if restored != started || restored.host.Client() != hostClient {
		t.Fatal("failed candidate replaced the active Host")
	}

	// A same-Profile replacement cannot coexist with the old Host because the
	// state directory is exclusive. If opening the replacement fails, Desktop
	// must reopen the previous configuration and keep Connect retryable.
	conflicting := profile
	conflicting.ParentPublicKey = base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
	conflictingJSON, _ := json.Marshal(conflicting)
	if _, err := app.SaveProfileJSON(string(conflictingJSON)); err == nil {
		t.Fatal("conflicting parent identity was unexpectedly saved")
	}
	app.mu.Lock()
	restored = app.active
	app.mu.Unlock()
	if restored == nil || restored.host == nil || restored.host.Role() != nodehost.RoleLeaf {
		t.Fatalf("same-profile failure did not restore the active leaf runtime: %+v", restored)
	}
	if restored.host.Status().Lifecycle != nodehost.LifecycleNew {
		t.Fatalf("restored profile runtime lifecycle = %q, want new", restored.host.Status().Lifecycle)
	}
}

func TestFailedProfileRuntimeClosesHostAndAllowsRetry(t *testing.T) {
	parent, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	open := func() (*profileRuntime, error) {
		return openTestProfileRuntime(directory, testProfile("retry", "2"), parent, permanentFailureDriver{})
	}

	failed, err := open()
	if err != nil {
		t.Fatal(err)
	}
	if err := connectRuntime(failed, "", false); err == nil || !strings.Contains(err.Error(), auth.ErrUntrustedIdentity.Error()) {
		t.Fatalf("permanent parent failure was not surfaced: %v", err)
	}
	if err := failed.Close(); err != nil {
		t.Fatal(err)
	}
	retry, err := open()
	if err != nil {
		t.Fatalf("failed runtime retained the state directory: %v", err)
	}
	if retry.host.Role() != nodehost.RoleLeaf || len(retry.host.Endpoints()) != 0 {
		t.Fatalf("retry runtime is not a listener-free leaf: role=%s endpoints=%v", retry.host.Role(), retry.host.Endpoints())
	}
	if err := retry.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAppConnectReplacesTerminalParentRuntimeBeforeRetry(t *testing.T) {
	parent, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	store, err := newSettingsStore(root)
	if err != nil {
		t.Fatal(err)
	}
	profile := testProfile("terminal-retry", "2")
	profile.Endpoint = "parent"
	profile.ParentPublicKey = base64.RawStdEncoding.EncodeToString(parent.PublicKey)
	settings := upsertProfile(defaultSettings(), profile)
	if err := store.save(settings); err != nil {
		t.Fatal(err)
	}
	firstDriver := &recordingPermanentFailureDriver{}
	secondDriver := &recordingPermanentFailureDriver{}
	credentials := &sequenceCredentialStore{
		directory: filepath.Join(root, "profiles", profile.ID),
		parent:    parent,
		drivers:   []link.Driver{firstDriver, secondDriver},
	}
	first, err := credentials.Open(profile, "")
	if err != nil {
		t.Fatal(err)
	}
	app := &App{store: store, credentials: credentials, settings: settings, active: first}
	defer app.Close()

	if err := app.Connect(); err == nil || !strings.Contains(err.Error(), auth.ErrUntrustedIdentity.Error()) {
		t.Fatalf("first terminal connection failure was not surfaced: %v", err)
	}
	app.mu.Lock()
	activeAfterFirst := app.active
	app.mu.Unlock()
	if activeAfterFirst != nil || first.host.Status().Lifecycle != nodehost.LifecycleStopped {
		t.Fatalf("terminal runtime was retained: active=%p lifecycle=%q", activeAfterFirst, first.host.Status().Lifecycle)
	}
	if firstDriver.Dials() != 1 {
		t.Fatalf("first runtime dial count = %d, want 1", firstDriver.Dials())
	}

	if err := app.Connect(); err == nil || !strings.Contains(err.Error(), auth.ErrUntrustedIdentity.Error()) {
		t.Fatalf("second terminal connection failure was not surfaced: %v", err)
	}
	if credentials.Opens() != 2 || secondDriver.Dials() != 1 {
		t.Fatalf("retry did not create and dial a new runtime: opens=%d second_dials=%d", credentials.Opens(), secondDriver.Dials())
	}
}

type permanentFailureDriver struct{}

func (permanentFailureDriver) Dial(context.Context, link.Endpoint) (link.Pipe, error) {
	return nil, auth.ErrUntrustedIdentity
}

func (permanentFailureDriver) Listen(context.Context, link.Endpoint) (link.Listener, error) {
	return nil, errors.New("permanent failure driver does not listen")
}

type recordingPermanentFailureDriver struct {
	mu    sync.Mutex
	dials int
}

func (d *recordingPermanentFailureDriver) Dial(context.Context, link.Endpoint) (link.Pipe, error) {
	d.mu.Lock()
	d.dials++
	d.mu.Unlock()
	return nil, auth.ErrUntrustedIdentity
}

func (*recordingPermanentFailureDriver) Listen(context.Context, link.Endpoint) (link.Listener, error) {
	return nil, errors.New("recording failure driver does not listen")
}

func (d *recordingPermanentFailureDriver) Dials() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.dials
}

type sequenceCredentialStore struct {
	mu        sync.Mutex
	directory string
	parent    auth.Identity
	drivers   []link.Driver
	opens     int
}

func (s *sequenceCredentialStore) Open(profile Profile, _ string) (*profileRuntime, error) {
	s.mu.Lock()
	if s.opens >= len(s.drivers) {
		s.mu.Unlock()
		return nil, errors.New("test credential store exhausted")
	}
	driver := s.drivers[s.opens]
	s.opens++
	s.mu.Unlock()
	return openTestProfileRuntime(s.directory, profile, s.parent, driver)
}

func (*sequenceCredentialStore) InspectEnrollment(string) (auth.EnrollmentClientSnapshot, bool, error) {
	return auth.EnrollmentClientSnapshot{}, false, nil
}

func (*sequenceCredentialStore) Remove(string) error { return nil }
func (*sequenceCredentialStore) Mode() string        { return "test" }

func (s *sequenceCredentialStore) Opens() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opens
}

func openTestProfileRuntime(directory string, profile Profile, parent auth.Identity, driver link.Driver) (*profileRuntime, error) {
	host, err := nodehost.New(context.Background(), nodehost.Config{
		StateDirectory: directory, NodeID: 2,
		Parent: &nodehost.ParentConfig{
			NodeID: 1, PublicKey: parent.PublicKey, Driver: driver, Endpoint: "parent",
		},
	})
	if err != nil {
		return nil, err
	}
	status, ok := host.ParentStatus()
	if !ok {
		_ = host.Close()
		return nil, errors.New("test host has no parent status")
	}
	core, err := bindings.NewAttachedClient(host.Client(), bindings.PublicIdentity{
		NodeID: host.ID(), PublicKey: host.PublicKey(),
	}, status)
	if err != nil {
		_ = host.Close()
		return nil, err
	}
	client, err := desktopbinding.NewAttachedClient(core)
	if err != nil {
		_ = core.Close()
		_ = host.Close()
		return nil, err
	}
	return &profileRuntime{profile: profile, host: host, client: client}, nil
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

func TestProfileStatesAreExactSanitizedAndReadOnly(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	authority := Profile{ID: "authority-missing", Name: "Authority Missing", EnrollmentMode: "authority", Endpoint: "127.0.0.1:7331"}
	legacy := testProfile("legacy-ready", "52")
	app.mu.Lock()
	next := upsertInactiveProfile(app.settings, authority)
	next = upsertInactiveProfile(next, legacy)
	if err := app.store.save(next); err != nil {
		app.mu.Unlock()
		t.Fatal(err)
	}
	app.settings = next
	app.mu.Unlock()

	raw, err := app.ProfileStatesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var states []profileState
	if err := json.Unmarshal([]byte(raw), &states); err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 || states[0].ProfileID != authority.ID || states[0].State != "missing" || states[1].State != "legacy" {
		t.Fatalf("unexpected initial profile states: %s", raw)
	}
	if strings.Contains(raw, "public_key") || strings.Contains(raw, "private") || strings.Contains(raw, "grant") || strings.Contains(raw, "permit") {
		t.Fatalf("profile states exposed protected material: %s", raw)
	}
	if _, err := os.Stat(filepath.Join(root, "profiles", authority.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only inspection created a Profile directory: %v", err)
	}

	encoded, _ := json.Marshal(authority)
	preparedRaw, err := app.PrepareProfileJSON(string(encoded))
	if err != nil {
		t.Fatal(err)
	}
	var prepared preparedProfile
	if err := json.Unmarshal([]byte(preparedRaw), &prepared); err != nil {
		t.Fatal(err)
	}
	raw, err = app.ProfileStatesJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw), &states); err != nil {
		t.Fatal(err)
	}
	if states[0].State != "device" || states[0].RequestID == "" {
		t.Fatalf("prepared authority state was not projected exactly: %s", raw)
	}
	if strings.Contains(raw, prepared.Identity.PublicKey) {
		t.Fatalf("profile state projection exposed the device public key: %s", raw)
	}
}

func TestDeactivateProfilePreservesProfileStateAndSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	profile := testProfile("deactivate", "61")
	encoded, _ := json.Marshal(profile)
	if _, err := app.SaveProfileJSON(string(encoded)); err != nil {
		t.Fatal(err)
	}
	profileDirectory, err := app.store.stateDirectory(profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(profileDirectory, "preserved-view-marker")
	if err := os.WriteFile(marker, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := app.DeactivateProfile(); err != nil {
		t.Fatal(err)
	}
	settingsRaw, err := app.SettingsJSON()
	if err != nil || strings.Contains(settingsRaw, `"active_profile_id"`) || !strings.Contains(settingsRaw, profile.ID) {
		t.Fatalf("deactivation did not preserve the inactive Profile: %v (%s)", err, settingsRaw)
	}
	statusRaw, err := app.StatusJSON()
	if err != nil || !strings.Contains(statusRaw, `"state":"signed_out"`) {
		t.Fatalf("deactivation did not sign out: %v (%s)", err, statusRaw)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("deactivation removed Profile state: %v", err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err := reopened.currentClient(); err == nil {
		t.Fatal("deactivated Profile was reopened across restart")
	}
	settingsRaw, _ = reopened.SettingsJSON()
	if !strings.Contains(settingsRaw, profile.ID) || strings.Contains(settingsRaw, `"active_profile_id"`) {
		t.Fatalf("restarted settings lost deactivation truth: %s", settingsRaw)
	}
	if err := reopened.SwitchProfile(profile.ID); err != nil {
		t.Fatalf("preserved Profile could not be selected again: %v", err)
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
	app.mu.Lock()
	authorityRuntime := app.active
	app.mu.Unlock()
	if authorityRuntime == nil || authorityRuntime.host == nil || authorityRuntime.client == nil || authorityRuntime.bootstrap != nil {
		t.Fatalf("Authority Profile did not hand off to NodeHost: %+v", authorityRuntime)
	}
	if authorityRuntime.host.Role() != nodehost.RoleLeaf || len(authorityRuntime.host.Endpoints()) != 0 {
		t.Fatalf("Authority Profile Host is not a listener-free leaf: role=%q endpoints=%v", authorityRuntime.host.Role(), authorityRuntime.host.Endpoints())
	}
	hostClient := authorityRuntime.host.Client()
	if err := app.Connect(); err != nil {
		t.Fatalf("repeated Authority Connect did not reuse the Host supervisor: %v", err)
	}
	app.mu.Lock()
	reusedAuthorityRuntime := app.active
	app.mu.Unlock()
	if reusedAuthorityRuntime != authorityRuntime || reusedAuthorityRuntime.host.Client() != hostClient {
		t.Fatal("repeated Authority Connect replaced the Host or SDK client")
	}
	if _, err := os.Stat(filepath.Join(desktopRoot, "profiles", profile.ID, "identity.dpapi")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Authority Profile created a duplicate Legacy identity: %v", err)
	}
	statesJSON, err := app.ProfileStatesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var states []profileState
	if err := json.Unmarshal([]byte(statesJSON), &states); err != nil || len(states) != 1 || states[0].State != "enrolled" || states[0].NodeID != saved.NodeID {
		t.Fatalf("enrolled credential was not projected independently of Profile hydration: %v (%s)", err, statesJSON)
	}
	snapshot, found, err := app.credentials.InspectEnrollment(profile.ID)
	if err != nil || !found {
		t.Fatalf("inspect enrolled Profile: found=%v err=%v", found, err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*Profile)
	}{
		{name: "Node ID", mutate: func(value *Profile) { value.NodeID = "999" }},
		{name: "父 Node ID", mutate: func(value *Profile) { value.ParentNodeID = "999" }},
		{name: "父公钥", mutate: func(value *Profile) {
			value.ParentPublicKey = base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
		}},
		{name: "Authority Node ID", mutate: func(value *Profile) { value.AuthorityNodeID = "999" }},
		{name: "Authority 公钥", mutate: func(value *Profile) {
			value.AuthorityPublicKey = base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
		}},
	} {
		conflicting := saved
		test.mutate(&conflicting)
		if mismatch := enrollmentProfileMismatch(conflicting, snapshot); mismatch != test.name {
			t.Fatalf("%s mismatch classified as %q", test.name, mismatch)
		}
	}
	app.mu.Lock()
	app.settings.Profiles[0].NodeID = "999"
	app.mu.Unlock()
	statesJSON, err = app.ProfileStatesJSON()
	if err != nil || !strings.Contains(statesJSON, `"state":"error"`) || !strings.Contains(statesJSON, "不一致") {
		t.Fatalf("Profile/Grant identity conflict was not explicit: %v (%s)", err, statesJSON)
	}
	app.mu.Lock()
	app.settings.Profiles[0].NodeID = saved.NodeID
	app.mu.Unlock()
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
	reopened.mu.Lock()
	restartedRuntime := reopened.active
	reopened.mu.Unlock()
	if restartedRuntime == nil || restartedRuntime.host == nil || restartedRuntime.bootstrap != nil {
		t.Fatalf("enrolled restart did not open NodeHost directly: %+v", restartedRuntime)
	}
	if err := reopened.Connect(); err != nil {
		t.Fatalf("persisted Authority Grant did not reconnect without another enrollment: %v", err)
	}
}

func TestAuthorityPendingRetryCreatesNoNodeUntilGrant(t *testing.T) {
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
	defer app.Close()
	profile := Profile{
		ID: "authority-pending", Name: "Authority Pending", EnrollmentMode: "authority",
		Endpoint: string(root.Endpoint), AutoConnect: false,
	}
	requestJSON, _ := json.Marshal(LoginRequest{Profile: profile, AllowTOFU: true})
	if _, err := app.LoginJSON(string(requestJSON)); !errors.Is(err, auth.ErrEnrollmentPending) {
		t.Fatalf("first authority login did not become pending: %v", err)
	}
	app.mu.Lock()
	active := app.active
	app.mu.Unlock()
	if active != nil {
		t.Fatal("pending authority Profile created an active ordinary runtime")
	}
	profileDirectory := filepath.Join(desktopRoot, "profiles", profile.ID)
	for _, forbidden := range []string{
		filepath.Join(profileDirectory, "identity.dpapi"),
		filepath.Join(profileDirectory, "state", "trust.json"),
	} {
		if _, err := os.Stat(forbidden); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("pending authority Profile created ordinary Node state %s: %v", forbidden, err)
		}
	}
	statesJSON, err := app.ProfileStatesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var states []profileState
	if err := json.Unmarshal([]byte(statesJSON), &states); err != nil || len(states) != 1 || states[0].State != "pending" || states[0].RequestID == "" {
		t.Fatalf("unexpected pending state: %v (%s)", err, statesJSON)
	}
	if _, err := root.EnrollmentAuthority.Approve("00112233445566778899aabbccddeeff", states[0].RequestID, "desktop"); err != nil {
		t.Fatal(err)
	}
	retryJSON, _ := json.Marshal(LoginRequest{Profile: profile})
	if _, err := app.LoginJSON(string(retryJSON)); err != nil {
		t.Fatalf("approved pending Profile did not retry through bootstrap and Host: %v", err)
	}
	app.mu.Lock()
	granted := app.active
	app.mu.Unlock()
	if granted == nil || granted.host == nil || granted.client == nil || granted.bootstrap != nil {
		t.Fatalf("approved pending Profile did not hand off to NodeHost: %+v", granted)
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
	first.AutoConnect = false
	data, _ := json.Marshal(first)
	if _, err := app.SaveProfileJSON(string(data)); err != nil {
		t.Fatal(err)
	}
	identity, err := app.IdentityJSON()
	if err != nil || !strings.Contains(identity, `"node_id":"42"`) {
		t.Fatalf("first profile identity mismatch: %v (%s)", err, identity)
	}
	second := testProfile("operator-2", "43")
	second.AutoConnect = false
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

func TestValidateSelectedUploadFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "upload.txt")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	selected, err := validateSelectedUploadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if selected != filepath.Clean(path) {
		t.Fatalf("selected path = %q, want %q", selected, filepath.Clean(path))
	}
	if _, err := validateSelectedUploadFile(" " + path); err == nil {
		t.Fatal("path with surrounding whitespace accepted")
	}
	if _, err := validateSelectedUploadFile(directory); err == nil {
		t.Fatal("directory accepted as upload file")
	}
}
