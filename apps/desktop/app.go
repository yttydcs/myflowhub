package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/sdk/bindings"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

const (
	defaultRequestTimeoutMS = int64(15_000)
	defaultLeaseMS          = int64(60_000)
	maxDesktopLogs          = 500
)

type logEntry struct {
	TimeUnixMS int64  `json:"time_unix_ms"`
	Level      string `json:"level"`
	Message    string `json:"message"`
}

type publicIdentity struct {
	NodeID    string `json:"node_id"`
	PublicKey string `json:"public_key"`
}

type enrollmentStatus struct {
	Status             string `json:"status"`
	RequestID          string `json:"request_id"`
	DevicePublicKey    string `json:"device_public_key"`
	NodeID             string `json:"node_id"`
	EnrollmentID       string `json:"enrollment_id"`
	ParentNodeID       string `json:"parent_node_id"`
	ParentPublicKey    string `json:"parent_public_key"`
	AuthorityNodeID    string `json:"authority_node_id"`
	AuthorityPublicKey string `json:"authority_public_key"`
}

type preparedProfile struct {
	Profile  Profile        `json:"profile"`
	Identity publicIdentity `json:"identity"`
}

type profileState struct {
	ProfileID       string `json:"profile_id"`
	State           string `json:"state"`
	RequestID       string `json:"request_id,omitempty"`
	NodeID          string `json:"node_id,omitempty"`
	ParentNodeID    string `json:"parent_node_id,omitempty"`
	AuthorityNodeID string `json:"authority_node_id,omitempty"`
	Message         string `json:"message,omitempty"`
}

type CredentialStore interface {
	Open(Profile, string) (*profileRuntime, error)
	InspectEnrollment(string) (auth.EnrollmentClientSnapshot, bool, error)
	Remove(string) error
	Mode() string
}

type platformCredentialBackend interface {
	auth.IdentityStore
	auth.EnrollmentCredentialStore
}

// profileRuntime is the Desktop product's explicit ownership boundary. Every
// granted profile owns one NodeHost and an attached client. An authority
// profile may temporarily own a narrow pre-Grant Enrollment bootstrap.
type profileRuntime struct {
	profile   Profile
	host      *nodehost.Host
	client    *desktopbinding.Client
	bootstrap *bindings.EnrollmentBootstrap
	promote   func() (*nodehost.Host, *desktopbinding.Client, error)
}

func (r *profileRuntime) Close() error {
	if r == nil {
		return nil
	}
	var bootstrapErr, facadeErr, hostErr error
	if r.bootstrap != nil {
		bootstrapErr = r.bootstrap.Close()
		r.bootstrap = nil
	}
	if r.client != nil {
		facadeErr = r.client.Close()
	}
	if r.host != nil {
		hostErr = r.host.Close()
	}
	return errors.Join(bootstrapErr, facadeErr, hostErr)
}

func (r *profileRuntime) connect(permitJSON string, allowTOFU bool) error {
	if r == nil {
		return errors.New("desktop profile runtime is unavailable")
	}
	if r.bootstrap != nil {
		if err := enrollBootstrap(r.bootstrap, r.profile, permitJSON, allowTOFU); err != nil {
			return err
		}
		if err := r.bootstrap.Close(); err != nil {
			return fmt.Errorf("close Enrollment bootstrap before opening node host: %w", err)
		}
		r.bootstrap = nil
		if r.promote == nil {
			return errors.New("desktop Enrollment runtime cannot open a granted node host")
		}
		host, client, err := r.promote()
		if err != nil {
			return err
		}
		r.host = host
		r.client = client
		r.promote = nil
	}
	if r.host == nil || r.client == nil {
		return errors.New("desktop profile node host is unavailable")
	}
	switch r.host.Status().Lifecycle {
	case nodehost.LifecycleNew:
		if err := r.host.Start(); err != nil {
			return err
		}
	case nodehost.LifecycleRunning:
		// The Host-owned supervisor already reconnects; another Connect only
		// waits for that same path instead of creating a second runtime.
	default:
		return fmt.Errorf("desktop profile runtime cannot connect from lifecycle %q", r.host.Status().Lifecycle)
	}
	return r.client.WaitConnected(defaultRequestTimeoutMS)
}

func (r *profileRuntime) identityJSON() (string, error) {
	if r == nil {
		return "", errors.New("desktop profile runtime is unavailable")
	}
	if r.bootstrap != nil {
		statusJSON, err := r.bootstrap.EnrollmentStatusJSON()
		if err != nil {
			return "", err
		}
		var status enrollmentStatus
		if err := decodeStrictJSON([]byte(statusJSON), &status); err != nil {
			return "", err
		}
		return marshalJSON(publicIdentity{NodeID: status.NodeID, PublicKey: status.DevicePublicKey})
	}
	if r.client == nil {
		return "", errors.New("desktop profile client is unavailable")
	}
	return r.client.IdentityJSON()
}

type profileCredentialStore struct {
	settings *settingsStore
	mu       sync.Mutex
	stores   map[string]platformCredentialBackend
}

func newProfileCredentialStore(settings *settingsStore) *profileCredentialStore {
	return &profileCredentialStore{settings: settings, stores: make(map[string]platformCredentialBackend)}
}

func (s *profileCredentialStore) Open(profile Profile, permitJSON string) (*profileRuntime, error) {
	if s == nil || s.settings == nil {
		return nil, errors.New("desktop credential store is unavailable")
	}
	if err := validateProfile(profile); err != nil {
		return nil, err
	}
	directory, err := s.settings.stateDirectory(profile.ID)
	if err != nil {
		return nil, err
	}
	backend, err := s.backend(profile.ID, directory)
	if err != nil {
		return nil, err
	}
	if profile.EnrollmentMode == "authority" {
		snapshot, found, err := auth.InspectEnrollmentClientState(backend)
		if err != nil {
			return nil, err
		}
		if found && snapshot.Status == "enrolled" {
			host, client, err := openAuthorityProfileHost(profile, directory, backend)
			if err != nil {
				return nil, err
			}
			return &profileRuntime{profile: profile, host: host, client: client}, nil
		}
		bootstrap, err := bindings.NewEnrollmentBootstrapWithCredentialStore(directory, backend)
		if err != nil {
			return nil, fmt.Errorf("open profile Enrollment bootstrap: %w", err)
		}
		return &profileRuntime{
			profile: profile, bootstrap: bootstrap,
			promote: func() (*nodehost.Host, *desktopbinding.Client, error) {
				return openAuthorityProfileHost(profile, directory, backend)
			},
		}, nil
	}
	nodeID, err := parsePositiveInt64(profile.NodeID, "node_id")
	if err != nil {
		return nil, err
	}
	parentID, err := parsePositiveInt64(profile.ParentNodeID, "parent_node_id")
	if err != nil {
		return nil, err
	}
	parentKey, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(profile.ParentPublicKey))
	if err != nil || len(parentKey) != ed25519.PublicKeySize {
		return nil, errors.New("parent_public_key must be a raw-base64 Ed25519 key")
	}
	var permit *protocol.ProvisioningPermitV1
	if strings.TrimSpace(permitJSON) != "" {
		var value protocol.ProvisioningPermitV1
		if err := protocol.DecodeJSONPayload([]byte(permitJSON), protocol.DefaultMaxPayload, &value); err != nil {
			return nil, fmt.Errorf("decode provisioning permit: %w", err)
		}
		permit = &value
	}
	host, err := nodehost.New(context.Background(), nodehost.Config{
		StateDirectory: directory,
		NodeID:         protocol.NodeID(nodeID),
		IdentityStore:  backend,
		Parent: &nodehost.ParentConfig{
			NodeID: protocol.NodeID(parentID), PublicKey: ed25519.PublicKey(parentKey), Permit: permit,
			Driver: tcp.Driver{}, Endpoint: link.Endpoint(strings.TrimSpace(profile.Endpoint)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open profile node host: %w", err)
	}
	client, err := attachDesktopClient(host)
	if err != nil {
		_ = host.Close()
		return nil, err
	}
	return &profileRuntime{profile: profile, host: host, client: client}, nil
}

func openAuthorityProfileHost(profile Profile, directory string, backend platformCredentialBackend) (*nodehost.Host, *desktopbinding.Client, error) {
	snapshot, found, err := auth.InspectEnrollmentClientState(backend)
	if err != nil {
		return nil, nil, err
	}
	if !found || snapshot.Status != "enrolled" || snapshot.Grant == nil {
		return nil, nil, auth.ErrEnrollmentNotGranted
	}
	if mismatch := enrollmentProfileMismatch(profile, snapshot); mismatch != "" {
		return nil, nil, fmt.Errorf("Profile 与受保护的 Enrollment Grant %s 不一致", mismatch)
	}
	source, err := auth.NewEnrollmentCredentialSource(backend)
	if err != nil {
		return nil, nil, err
	}
	var configuredNodeID protocol.NodeID
	if profile.NodeID != "" {
		value, err := parsePositiveInt64(profile.NodeID, "node_id")
		if err != nil {
			return nil, nil, err
		}
		configuredNodeID = protocol.NodeID(value)
	}
	parent := &nodehost.ParentConfig{Driver: tcp.Driver{}, Endpoint: link.Endpoint(strings.TrimSpace(profile.Endpoint))}
	if profile.ParentNodeID != "" {
		value, err := parsePositiveInt64(profile.ParentNodeID, "parent_node_id")
		if err != nil {
			return nil, nil, err
		}
		parent.NodeID = protocol.NodeID(value)
	}
	if strings.TrimSpace(profile.ParentPublicKey) != "" {
		key, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(profile.ParentPublicKey))
		if err != nil || len(key) != ed25519.PublicKeySize {
			return nil, nil, errors.New("parent_public_key must be a raw-base64 Ed25519 key")
		}
		parent.PublicKey = ed25519.PublicKey(key)
	}
	host, err := nodehost.New(context.Background(), nodehost.Config{
		StateDirectory: directory, NodeID: configuredNodeID, CredentialSource: source, Parent: parent,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("open granted profile node host: %w", err)
	}
	client, err := attachDesktopClient(host)
	if err != nil {
		_ = host.Close()
		return nil, nil, err
	}
	return host, client, nil
}

func attachDesktopClient(host *nodehost.Host) (*desktopbinding.Client, error) {
	status, ok := host.ParentStatus()
	if !ok {
		return nil, errors.New("desktop profile node host has no parent")
	}
	core, err := bindings.NewAttachedClient(host.Client(), bindings.PublicIdentity{
		NodeID: host.ID(), PublicKey: host.PublicKey(),
	}, status)
	if err != nil {
		return nil, fmt.Errorf("attach desktop binding: %w", err)
	}
	client, err := desktopbinding.NewAttachedClient(core)
	if err != nil {
		_ = core.Close()
		return nil, fmt.Errorf("open attached desktop binding: %w", err)
	}
	return client, nil
}

func (s *profileCredentialStore) InspectEnrollment(profileID string) (auth.EnrollmentClientSnapshot, bool, error) {
	if s == nil || s.settings == nil {
		return auth.EnrollmentClientSnapshot{}, false, errors.New("desktop credential store is unavailable")
	}
	directory, err := s.settings.stateDirectory(profileID)
	if err != nil {
		return auth.EnrollmentClientSnapshot{}, false, err
	}
	store, err := s.backend(profileID, directory)
	if err != nil {
		return auth.EnrollmentClientSnapshot{}, false, err
	}
	return auth.InspectEnrollmentClientState(store)
}

func (s *profileCredentialStore) backend(profileID, directory string) (platformCredentialBackend, error) {
	s.mu.Lock()
	store := s.stores[profileID]
	if store == nil {
		var err error
		store, err = newPlatformIdentityStore(directory)
		if err == nil {
			s.stores[profileID] = store
		}
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
	}
	s.mu.Unlock()
	return store, nil
}

func (s *profileCredentialStore) Remove(profileID string) error {
	directory, err := s.settings.stateDirectory(profileID)
	if err != nil {
		return err
	}
	root, err := filepath.Abs(filepath.Join(s.settings.root, "profiles"))
	if err != nil {
		return err
	}
	target, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	if filepath.Dir(target) != root {
		return errors.New("refusing to remove profile outside the credential root")
	}
	s.mu.Lock()
	delete(s.stores, profileID)
	s.mu.Unlock()
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove profile credentials: %w", err)
	}
	return nil
}

func (*profileCredentialStore) Mode() string {
	return platformCredentialMode() + "; permits are session-only"
}

type App struct {
	mu          sync.Mutex
	lifecycleMu sync.Mutex
	ctx         context.Context
	store       *settingsStore
	credentials CredentialStore
	settings    Settings
	active      *profileRuntime
	logs        []logEntry
}

func NewApp(configRoot string) (*App, error) {
	store, err := newSettingsStore(configRoot)
	if err != nil {
		return nil, err
	}
	settings, err := store.load()
	if err != nil {
		return nil, err
	}
	app := &App{
		store: store, credentials: newProfileCredentialStore(store), settings: settings,
		logs: make([]logEntry, 0, maxDesktopLogs),
	}
	if settings.ActiveProfileID != "" {
		profile, _ := findProfile(settings, settings.ActiveProfileID)
		if err := app.openProfileLocked(profile); err != nil {
			return nil, err
		}
	}
	app.appendLog("info", "desktop resource workspace opened")
	return app, nil
}

func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	profile, active := findProfile(a.settings, a.settings.ActiveProfileID)
	a.mu.Unlock()
	if active && profile.AutoConnect {
		go func() {
			if err := a.Connect(); err != nil {
				a.fail("auto-connect", err)
			}
		}()
	}
}

func (a *App) Shutdown(context.Context) { _ = a.Close() }

func (a *App) SettingsJSON() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return marshalJSON(struct {
		Settings
		CredentialMode string `json:"credential_mode"`
	}{Settings: a.settings, CredentialMode: a.credentials.Mode()})
}

func (a *App) ProfileStatesJSON() (string, error) {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.mu.Lock()
	profiles := append([]Profile(nil), a.settings.Profiles...)
	a.mu.Unlock()

	states := make([]profileState, 0, len(profiles))
	for _, profile := range profiles {
		state := profileState{ProfileID: profile.ID}
		if profile.EnrollmentMode != "authority" {
			state.State = "legacy"
			state.NodeID = profile.NodeID
			state.ParentNodeID = profile.ParentNodeID
			states = append(states, state)
			continue
		}
		snapshot, found, err := a.credentials.InspectEnrollment(profile.ID)
		if err != nil {
			state.State = "error"
			state.Message = "受保护的 Enrollment 凭据不可用；请确认当前系统用户，或删除此 Profile 后重新连接。"
			states = append(states, state)
			continue
		}
		if !found {
			state.State = "missing"
			states = append(states, state)
			continue
		}
		state.State = snapshot.Status
		state.RequestID = snapshot.RequestID
		state.ParentNodeID = optionalNodeID(snapshot.ParentNodeID)
		state.AuthorityNodeID = optionalNodeID(snapshot.AuthorityNodeID)
		if snapshot.Grant != nil {
			state.NodeID = snapshot.Grant.NodeID
		}
		if mismatch := enrollmentProfileMismatch(profile, snapshot); mismatch != "" {
			state.State = "error"
			state.Message = "Profile " + mismatch + " 与受保护的 Enrollment Grant 不一致；请勿继续连接。"
		}
		states = append(states, state)
	}
	return marshalJSON(states)
}

func enrollmentProfileMismatch(profile Profile, snapshot auth.EnrollmentClientSnapshot) string {
	if snapshot.Status != "enrolled" || snapshot.Grant == nil {
		return ""
	}
	checks := []struct {
		name       string
		configured string
		canonical  string
	}{
		{name: "Node ID", configured: profile.NodeID, canonical: snapshot.Grant.NodeID},
		{name: "父 Node ID", configured: profile.ParentNodeID, canonical: optionalNodeID(snapshot.ParentNodeID)},
		{name: "父公钥", configured: profile.ParentPublicKey, canonical: base64.RawStdEncoding.EncodeToString(snapshot.ParentPublicKey)},
		{name: "Authority Node ID", configured: profile.AuthorityNodeID, canonical: optionalNodeID(snapshot.AuthorityNodeID)},
		{name: "Authority 公钥", configured: profile.AuthorityPublicKey, canonical: base64.RawStdEncoding.EncodeToString(snapshot.AuthorityPublicKey)},
	}
	for _, check := range checks {
		configured := strings.TrimSpace(check.configured)
		if configured != "" && configured != check.canonical {
			return check.name
		}
	}
	return ""
}

func optionalNodeID(value protocol.NodeID) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(value), 10)
}

func (a *App) SaveProfileJSON(raw string) (string, error) {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	var profile Profile
	if err := decodeBoundedJSON(raw, &profile); err != nil {
		return "", err
	}
	if err := validateProfile(profile); err != nil {
		return "", err
	}
	a.mu.Lock()
	previous := a.settings
	old := a.active
	a.mu.Unlock()
	next := upsertProfile(previous, profile)
	candidate, oldClosed, err := a.openCandidate(profile, "", old)
	if err != nil {
		return "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = candidate.Close()
		}
	}()
	if err := a.store.save(next); err != nil {
		_ = candidate.Close()
		candidate = nil
		if restoreErr := a.restoreClosedRuntime(previous, old, oldClosed); restoreErr != nil {
			return "", errors.Join(err, restoreErr)
		}
		return "", err
	}
	a.mu.Lock()
	a.active = candidate
	a.settings = next
	a.mu.Unlock()
	committed = true
	if old != nil && !oldClosed {
		_ = old.Close()
	}
	return marshalJSON(profile)
}

func (a *App) PrepareProfileJSON(raw string) (string, error) {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	var profile Profile
	if err := decodeBoundedJSON(raw, &profile); err != nil {
		return "", err
	}
	if err := validateProfile(profile); err != nil {
		return "", err
	}

	a.mu.Lock()
	activeProfileID := a.settings.ActiveProfileID
	a.mu.Unlock()
	if activeProfileID == profile.ID {
		return "", errors.New("active profile identity cannot be prepared; save it from Settings instead")
	}

	candidate, err := a.credentials.Open(profile, "")
	if err != nil {
		return "", err
	}
	defer candidate.Close()
	identityJSON, err := candidate.identityJSON()
	if err != nil {
		return "", fmt.Errorf("read prepared profile identity: %w", err)
	}
	var identity publicIdentity
	if err := decodeStrictJSON([]byte(identityJSON), &identity); err != nil {
		return "", fmt.Errorf("decode prepared profile identity: %w", err)
	}

	a.mu.Lock()
	next := upsertInactiveProfile(a.settings, profile)
	if err := a.store.save(next); err != nil {
		a.mu.Unlock()
		return "", err
	}
	a.settings = next
	a.mu.Unlock()
	a.appendLog("info", "profile identity prepared: "+profile.Name)
	return marshalJSON(preparedProfile{Profile: profile, Identity: identity})
}

func (a *App) LoginJSON(raw string) (string, error) {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	var request LoginRequest
	if err := decodeBoundedJSON(raw, &request); err != nil {
		return "", err
	}
	if err := validateProfile(request.Profile); err != nil {
		return "", err
	}
	if request.PermitJSON != "" && (!json.Valid([]byte(request.PermitJSON)) || len(request.PermitJSON) > protocol.DefaultMaxPayload) {
		return "", errors.New("permit_json must be valid bounded JSON")
	}
	a.mu.Lock()
	old := a.active
	previous := a.settings
	a.mu.Unlock()
	candidate, oldClosed, err := a.openCandidate(request.Profile, request.PermitJSON, old)
	if err != nil {
		return "", err
	}
	connected := false
	defer func() {
		if !connected {
			_ = candidate.Close()
		}
	}()
	if err := connectRuntime(candidate, request.PermitJSON, request.AllowTOFU); err != nil {
		if errors.Is(err, auth.ErrEnrollmentPending) {
			a.mu.Lock()
			next := upsertInactiveProfile(previous, request.Profile)
			saveErr := a.store.save(next)
			if saveErr == nil {
				a.settings = next
			}
			a.mu.Unlock()
			if saveErr != nil {
				err = errors.Join(err, saveErr)
			}
		}
		_ = candidate.Close()
		candidate = nil
		if restoreErr := a.restoreClosedRuntime(previous, old, oldClosed); restoreErr != nil {
			return "", a.fail("login", errors.Join(err, restoreErr))
		}
		return "", a.fail("login", err)
	}
	if request.Profile.EnrollmentMode == "authority" {
		snapshot, found, inspectErr := a.credentials.InspectEnrollment(request.Profile.ID)
		if inspectErr != nil {
			err = inspectErr
		} else {
			request.Profile, err = hydrateEnrollmentProfile(snapshot, found, request.Profile)
		}
		if err != nil {
			_ = candidate.Close()
			candidate = nil
			if restoreErr := a.restoreClosedRuntime(previous, old, oldClosed); restoreErr != nil {
				return "", a.fail("login", errors.Join(err, restoreErr))
			}
			return "", a.fail("login", err)
		}
		candidate.profile = request.Profile
	}

	a.mu.Lock()
	next := upsertProfile(previous, request.Profile)
	if err := a.store.save(next); err != nil {
		a.mu.Unlock()
		_ = candidate.Close()
		candidate = nil
		if restoreErr := a.restoreClosedRuntime(previous, old, oldClosed); restoreErr != nil {
			return "", errors.Join(err, restoreErr)
		}
		return "", err
	}
	a.active = candidate
	a.settings = next
	a.mu.Unlock()
	connected = true
	if old != nil && !oldClosed {
		_ = old.Close()
	}
	a.appendLog("info", "profile logged in: "+request.Profile.Name)
	return marshalJSON(request.Profile)
}

func (a *App) SwitchProfile(profileID string) error {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.mu.Lock()
	profile, exists := findProfile(a.settings, profileID)
	if !exists {
		a.mu.Unlock()
		return errors.New("profile not found")
	}
	if a.active != nil && a.active.profile.ID == profile.ID {
		a.mu.Unlock()
		return nil
	}
	old := a.active
	a.mu.Unlock()
	candidate, err := a.credentials.Open(profile, "")
	if err != nil {
		return err
	}
	a.mu.Lock()
	next := a.settings
	next.ActiveProfileID = profile.ID
	if err := a.store.save(next); err != nil {
		a.mu.Unlock()
		_ = candidate.Close()
		return err
	}
	a.active = candidate
	a.settings = next
	a.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	a.appendLog("info", "active profile switched: "+profile.Name)
	if profile.AutoConnect {
		go func() {
			if err := a.Connect(); err != nil {
				a.fail("profile auto-connect", err)
			}
		}()
	}
	return nil
}

func (a *App) DeactivateProfile() error {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()

	a.mu.Lock()
	if a.settings.ActiveProfileID == "" {
		old := a.active
		a.active = nil
		a.mu.Unlock()
		if old != nil {
			return old.Close()
		}
		return nil
	}
	next := a.settings
	next.ActiveProfileID = ""
	if err := a.store.save(next); err != nil {
		a.mu.Unlock()
		return fmt.Errorf("persist Profile deactivation: %w", err)
	}
	old := a.active
	a.active = nil
	a.settings = next
	a.mu.Unlock()

	a.appendLog("info", "returned to Profile selection")
	if old != nil {
		if err := old.Close(); err != nil {
			return fmt.Errorf("Profile was deactivated but the previous client did not close cleanly: %w", err)
		}
	}
	return nil
}

func (a *App) DeleteProfile(profileID, confirmation string) error {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	if confirmation != "DELETE "+profileID {
		return errors.New("profile deletion confirmation does not match")
	}
	a.mu.Lock()
	if _, exists := findProfile(a.settings, profileID); !exists {
		a.mu.Unlock()
		return errors.New("profile not found")
	}
	next := a.settings
	next.Profiles = append([]Profile(nil), next.Profiles...)
	for index := range next.Profiles {
		if next.Profiles[index].ID == profileID {
			next.Profiles = append(next.Profiles[:index], next.Profiles[index+1:]...)
			break
		}
	}
	active := next.ActiveProfileID == profileID
	if active {
		next.ActiveProfileID = ""
	}
	if err := a.store.save(next); err != nil {
		a.mu.Unlock()
		return err
	}
	var old *profileRuntime
	if active {
		old = a.active
		a.active = nil
	}
	a.settings = next
	a.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	if err := a.credentials.Remove(profileID); err != nil {
		return err
	}
	a.appendLog("warn", "profile and local identity removed: "+profileID)
	return nil
}

func (a *App) ResetStorage(confirm string) (string, error) {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.mu.Lock()
	old := a.active
	a.active = nil
	settings, err := a.store.reset(confirm)
	if err == nil {
		a.settings = settings
	}
	result, encodeErr := marshalJSON(a.settings)
	a.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	if err != nil {
		return "", err
	}
	if encodeErr != nil {
		return "", encodeErr
	}
	a.appendLog("warn", "desktop profile index reset; profile identities were retained")
	return result, nil
}

func (a *App) IdentityJSON() (string, error) {
	a.mu.Lock()
	current := a.active
	a.mu.Unlock()
	if current == nil {
		return "", errors.New("desktop is signed out")
	}
	return current.identityJSON()
}

func (a *App) Connect() error {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.mu.Lock()
	profile, exists := findProfile(a.settings, a.settings.ActiveProfileID)
	current := a.active
	a.mu.Unlock()
	if !exists {
		return errors.New("no active profile")
	}
	if current == nil {
		var err error
		current, err = a.credentials.Open(profile, "")
		if err != nil {
			return err
		}
		a.mu.Lock()
		a.active = current
		a.mu.Unlock()
	}
	if err := connectRuntime(current, "", false); err != nil {
		if profileRuntimeTerminal(current) {
			a.mu.Lock()
			if a.active == current {
				a.active = nil
			}
			a.mu.Unlock()
			_ = current.Close()
		}
		return err
	}
	a.appendLog("info", "managed TCP connection started")
	return nil
}

func enrollBootstrap(bootstrap *bindings.EnrollmentBootstrap, profile Profile, permitJSON string, allowTOFU bool) error {
	statusJSON, err := bootstrap.EnrollmentStatusJSON()
	if err != nil {
		return err
	}
	var status enrollmentStatus
	if err := decodeStrictJSON([]byte(statusJSON), &status); err != nil {
		return err
	}
	if status.Status != "enrolled" {
		expectedParentID := int64(0)
		if profile.ParentNodeID != "" {
			expectedParentID, err = parsePositiveInt64(profile.ParentNodeID, "parent_node_id")
			if err != nil {
				return err
			}
		}
		resultJSON, err := bootstrap.EnrollTCP(profile.Endpoint, permitJSON, allowTOFU, expectedParentID, profile.ParentPublicKey, profile.AuthorityPublicKey, defaultRequestTimeoutMS)
		if err != nil {
			return fmt.Errorf("enroll: %w", err)
		}
		var result protocol.EnrollmentResultV1
		if err := decodeStrictJSON([]byte(resultJSON), &result); err != nil {
			return err
		}
		if result.Status == "pending" {
			return fmt.Errorf("%w: request %s is waiting for approval", auth.ErrEnrollmentPending, result.RequestID)
		}
	}
	return nil
}

func profileRuntimeTerminal(current *profileRuntime) bool {
	if current == nil || current.host == nil {
		return true
	}
	status := current.host.Status()
	if status.Lifecycle == nodehost.LifecycleFailed || status.Lifecycle == nodehost.LifecycleStopped {
		return true
	}
	parent, ok := current.host.Parent()
	return ok && (parent.State == node.ConnectionFailed || parent.State == node.ConnectionStopped)
}

func connectRuntime(current *profileRuntime, permitJSON string, allowTOFU bool) error {
	if err := current.connect(permitJSON, allowTOFU); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return connectionTimeoutGuidance(err, permitJSON)
		}
		return fmt.Errorf("wait for connection: %w", err)
	}
	return nil
}

func hydrateEnrollmentProfile(snapshot auth.EnrollmentClientSnapshot, found bool, profile Profile) (Profile, error) {
	if !found || snapshot.Status != "enrolled" || snapshot.Grant == nil || snapshot.Grant.NodeID == "" {
		return Profile{}, errors.New("Enrollment did not persist a granted Node ID")
	}
	profile.NodeID = snapshot.Grant.NodeID
	profile.ParentNodeID = optionalNodeID(snapshot.ParentNodeID)
	profile.ParentPublicKey = base64.RawStdEncoding.EncodeToString(snapshot.ParentPublicKey)
	profile.AuthorityNodeID = optionalNodeID(snapshot.AuthorityNodeID)
	profile.AuthorityPublicKey = base64.RawStdEncoding.EncodeToString(snapshot.AuthorityPublicKey)
	return profile, nil
}

func connectionTimeoutGuidance(err error, permitJSON string) error {
	if strings.TrimSpace(permitJSON) == "" {
		return fmt.Errorf("连接超时：首次接入需要粘贴父节点签发的一次性准入 Permit；如果此前已准入，请检查连接端点、父节点状态和网络。技术详情：%w", err)
	}
	return fmt.Errorf("连接超时：Permit 可能无效、已使用或已过期；请重新签发 Permit，并检查连接端点和父节点状态。技术详情：%w", err)
}

func (a *App) Disconnect() error {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.mu.Lock()
	profile, exists := findProfile(a.settings, a.settings.ActiveProfileID)
	old := a.active
	a.active = nil
	a.mu.Unlock()
	if old != nil {
		if err := old.Close(); err != nil {
			return err
		}
	}
	if exists {
		client, err := a.credentials.Open(profile, "")
		if err != nil {
			return err
		}
		a.mu.Lock()
		a.active = client
		a.mu.Unlock()
	}
	a.appendLog("info", "desktop connection stopped")
	return nil
}

func (a *App) WaitConnected(timeoutMS int64) error {
	client, err := a.currentClient()
	if err != nil {
		return err
	}
	return client.WaitConnected(timeoutMS)
}

func (a *App) StatusJSON() (string, error) {
	client, err := a.currentClient()
	if err != nil {
		return marshalJSON(map[string]string{"state": "signed_out"})
	}
	return client.StatusJSON()
}

func (a *App) CatalogJSON(ownerNodeID string) (string, error) {
	ownerID, err := parsePositiveInt64(ownerNodeID, "owner_node_id")
	if err != nil {
		return "", err
	}
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.CatalogJSON(ownerID, defaultRequestTimeoutMS)
}

func (a *App) SnapshotJSON(ownerNodeID, name string) (string, error) {
	ownerID, err := parsePositiveInt64(ownerNodeID, "owner_node_id")
	if err != nil {
		return "", err
	}
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.SnapshotJSON(ownerID, name, defaultRequestTimeoutMS)
}

func (a *App) OperateJSON(ownerNodeID, name, capability, schema, requestJSON string) (string, error) {
	ownerID, err := parsePositiveInt64(ownerNodeID, "owner_node_id")
	if err != nil {
		return "", err
	}
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	result, err := client.OperateJSON(ownerID, name, capability, schema, requestJSON, defaultRequestTimeoutMS)
	if err != nil {
		return "", a.fail("operate "+name+"/"+capability, err)
	}
	a.appendLog("info", "resource operation completed: "+name+"/"+capability)
	return result, nil
}

func (a *App) InvokeJSON(ownerNodeID, name, requestJSON string) (string, error) {
	ownerID, err := parsePositiveInt64(ownerNodeID, "owner_node_id")
	if err != nil {
		return "", err
	}
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.InvokeJSON(ownerID, name, requestJSON, defaultRequestTimeoutMS)
}

func (a *App) SubscribeCapability(ownerNodeID, name, capability string, leaseMS int64) (int64, error) {
	if leaseMS == 0 {
		leaseMS = defaultLeaseMS
	}
	ownerID, err := parsePositiveInt64(ownerNodeID, "owner_node_id")
	if err != nil {
		return 0, err
	}
	client, err := a.currentClient()
	if err != nil {
		return 0, err
	}
	id, err := client.SubscribeCapability(ownerID, name, capability, leaseMS)
	if err != nil {
		return 0, a.fail("subscribe "+name+"/"+capability, err)
	}
	return id, nil
}

func (a *App) Subscribe(ownerNodeID, name string, leaseMS int64) (int64, error) {
	return a.SubscribeCapability(ownerNodeID, name, string(protocol.CapabilitySubscribe), leaseMS)
}

func (a *App) PollSubscription(subscriptionID, timeoutMS int64) (string, error) {
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.PollSubscription(subscriptionID, timeoutMS)
}

func (a *App) CancelSubscription(subscriptionID int64) {
	client, err := a.currentClient()
	if err == nil {
		client.CancelSubscription(subscriptionID)
	}
}

func (a *App) TopologyJSON(ownerNodeID string) (string, error) {
	return a.SnapshotJSON(ownerNodeID, protocol.BuiltinManagementTopology)
}

func (a *App) UploadFile(ownerNodeID, sourcePath, destination, contentType string) (string, error) {
	ownerID, err := parsePositiveInt64(ownerNodeID, "owner_node_id")
	if err != nil {
		return "", err
	}
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.UploadFile(ownerID, sourcePath, destination, contentType, 10*60*1000)
}

// SelectUploadFile opens the platform-native picker and returns a validated
// regular file. Cancellation is represented by an empty path, not an error.
func (a *App) SelectUploadFile() (string, error) {
	a.mu.Lock()
	ctx := a.ctx
	a.mu.Unlock()
	if ctx == nil {
		return "", errors.New("desktop application is not started")
	}
	selected, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title:   "选择要上传的文件",
		Filters: []runtime.FileFilter{{DisplayName: "所有文件", Pattern: "*"}},
	})
	if err != nil {
		return "", fmt.Errorf("select upload file: %w", err)
	}
	if selected == "" {
		return "", nil
	}
	return validateSelectedUploadFile(selected)
}

func validateSelectedUploadFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(path) != path {
		return "", errors.New("selected upload path is invalid")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve selected upload file: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect selected upload file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("selected upload path must be a regular file")
	}
	return filepath.Clean(absolute), nil
}

func (a *App) ViewsJSON() (string, error) {
	store, err := a.activeViewStore()
	if err != nil {
		return "", err
	}
	document, err := store.load()
	if err != nil {
		return "", err
	}
	return marshalJSON(document)
}

func (a *App) SaveViewJSON(raw string) (string, error) {
	var view ViewDefinition
	if err := decodeBoundedJSON(raw, &view); err != nil {
		return "", err
	}
	store, err := a.activeViewStore()
	if err != nil {
		return "", err
	}
	saved, err := store.saveView(view)
	if err != nil {
		return "", err
	}
	return marshalJSON(saved)
}

func (a *App) DeleteView(viewID string, revision int64) error {
	if revision <= 0 {
		return errors.New("view revision must be positive")
	}
	store, err := a.activeViewStore()
	if err != nil {
		return err
	}
	return store.deleteView(viewID, uint64(revision))
}

func (a *App) activeViewStore() (*viewStore, error) {
	a.mu.Lock()
	profileID := a.settings.ActiveProfileID
	a.mu.Unlock()
	if profileID == "" {
		return nil, errors.New("no active profile")
	}
	directory, err := a.store.stateDirectory(profileID)
	if err != nil {
		return nil, err
	}
	return newViewStore(directory)
}

func (a *App) LogsJSON() (string, error) {
	a.mu.Lock()
	logs := append([]logEntry(nil), a.logs...)
	a.mu.Unlock()
	return marshalJSON(logs)
}

func (a *App) Close() error {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.mu.Lock()
	current := a.active
	a.active = nil
	a.mu.Unlock()
	if current == nil {
		return nil
	}
	return current.Close()
}

func (a *App) currentClient() (*desktopbinding.Client, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.active == nil || a.active.client == nil {
		return nil, errors.New("desktop is signed out")
	}
	return a.active.client, nil
}

func (a *App) openProfileLocked(profile Profile) error {
	current, err := a.credentials.Open(profile, "")
	if err != nil {
		return err
	}
	a.active = current
	return nil
}

// openCandidate preserves an existing runtime until the replacement Host has
// been created. The one exception is a same-Profile replacement because a
// state directory is exclusively owned by one Host; in that case failure is
// rolled back by reopening the previous Profile runtime.
func (a *App) openCandidate(profile Profile, permitJSON string, old *profileRuntime) (*profileRuntime, bool, error) {
	oldClosed := old != nil && old.profile.ID == profile.ID
	if oldClosed {
		if err := old.Close(); err != nil {
			return nil, false, fmt.Errorf("close previous desktop profile runtime: %w", err)
		}
		a.mu.Lock()
		if a.active == old {
			a.active = nil
		}
		a.mu.Unlock()
	}
	candidate, err := a.credentials.Open(profile, permitJSON)
	if err == nil {
		return candidate, oldClosed, nil
	}
	if !oldClosed {
		return nil, false, err
	}
	restoreErr := a.restoreProfileRuntime(old.profile)
	if restoreErr != nil {
		return nil, true, errors.Join(err, restoreErr)
	}
	return nil, true, err
}

func (a *App) restoreClosedRuntime(_ Settings, old *profileRuntime, oldClosed bool) error {
	if !oldClosed || old == nil {
		return nil
	}
	return a.restoreProfileRuntime(old.profile)
}

func (a *App) restoreProfileRuntime(profile Profile) error {
	restored, err := a.credentials.Open(profile, "")
	if err != nil {
		return fmt.Errorf("restore previous desktop profile runtime: %w", err)
	}
	a.mu.Lock()
	a.active = restored
	a.mu.Unlock()
	return nil
}

func (a *App) appendLog(level, message string) {
	entry := logEntry{TimeUnixMS: time.Now().UTC().UnixMilli(), Level: level, Message: message}
	a.mu.Lock()
	if len(a.logs) == maxDesktopLogs {
		copy(a.logs, a.logs[1:])
		a.logs[len(a.logs)-1] = entry
	} else {
		a.logs = append(a.logs, entry)
	}
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "desktop:log", entry)
	}
}

func (a *App) fail(action string, err error) error {
	a.appendLog("error", action+": "+err.Error())
	return err
}

func decodeBoundedJSON(raw string, target any) error {
	if len(raw) == 0 || len(raw) > protocol.DefaultMaxPayload || !json.Valid([]byte(raw)) {
		return errors.New("request must be valid JSON within the protocol payload limit")
	}
	if err := decodeStrictJSON([]byte(raw), target); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	return nil
}

func marshalJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
