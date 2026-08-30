package main

import (
	"context"
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
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
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
	Open(Profile) (*desktopbinding.Client, error)
	InspectEnrollment(string) (auth.EnrollmentClientSnapshot, bool, error)
	Remove(string) error
	Mode() string
}

type platformCredentialBackend interface {
	auth.IdentityStore
	auth.EnrollmentCredentialStore
}

type profileCredentialStore struct {
	settings *settingsStore
	mu       sync.Mutex
	stores   map[string]platformCredentialBackend
}

func newProfileCredentialStore(settings *settingsStore) *profileCredentialStore {
	return &profileCredentialStore{settings: settings, stores: make(map[string]platformCredentialBackend)}
}

func (s *profileCredentialStore) Open(profile Profile) (*desktopbinding.Client, error) {
	if s == nil || s.settings == nil {
		return nil, errors.New("desktop credential store is unavailable")
	}
	directory, err := s.settings.stateDirectory(profile.ID)
	if err != nil {
		return nil, err
	}
	identityStore, err := s.backend(profile.ID, directory)
	if err != nil {
		return nil, err
	}
	client := &desktopbinding.Client{}
	if profile.EnrollmentMode == "authority" {
		if err := client.OpenEnrollmentWithCredentialStore(directory, identityStore); err != nil {
			return nil, fmt.Errorf("open profile Enrollment credential: %w", err)
		}
	} else {
		nodeID, err := parsePositiveInt64(profile.NodeID, "node_id")
		if err != nil {
			return nil, err
		}
		if err := client.OpenWithIdentityStore(directory, nodeID, identityStore); err != nil {
			return nil, fmt.Errorf("open profile identity: %w", err)
		}
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
	client      *desktopbinding.Client
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
		if profile.NodeID != "" && state.NodeID != "" && profile.NodeID != state.NodeID {
			state.State = "error"
			state.Message = "Profile Node ID 与受保护的 Enrollment Grant 不一致；请勿继续连接。"
		}
		states = append(states, state)
	}
	return marshalJSON(states)
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
	defer a.mu.Unlock()
	previous := a.settings
	next := upsertProfile(a.settings, profile)
	if err := a.store.save(next); err != nil {
		return "", err
	}
	if a.client != nil {
		_ = a.client.Close()
		a.client = nil
	}
	a.settings = next
	if err := a.openProfileLocked(profile); err != nil {
		a.settings = previous
		_ = a.store.save(previous)
		return "", err
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

	candidate, err := a.credentials.Open(profile)
	if err != nil {
		return "", err
	}
	defer candidate.Close()
	identityJSON := ""
	if profile.EnrollmentMode == "authority" {
		identityJSON, err = candidate.EnrollmentStatusJSON()
	} else {
		identityJSON, err = candidate.IdentityJSON()
	}
	if err != nil {
		return "", fmt.Errorf("read prepared profile identity: %w", err)
	}
	var identity publicIdentity
	if profile.EnrollmentMode == "authority" {
		var status enrollmentStatus
		if err := decodeStrictJSON([]byte(identityJSON), &status); err != nil {
			return "", fmt.Errorf("decode prepared Enrollment identity: %w", err)
		}
		identity = publicIdentity{NodeID: status.NodeID, PublicKey: status.DevicePublicKey}
	} else if err := decodeStrictJSON([]byte(identityJSON), &identity); err != nil {
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
	candidate, err := a.credentials.Open(request.Profile)
	if err != nil {
		return "", err
	}
	connected := false
	defer func() {
		if !connected {
			_ = candidate.Close()
		}
	}()
	if err := connectClient(candidate, request.Profile, request.PermitJSON, request.AllowTOFU); err != nil {
		if errors.Is(err, auth.ErrEnrollmentPending) {
			a.mu.Lock()
			next := upsertInactiveProfile(a.settings, request.Profile)
			saveErr := a.store.save(next)
			if saveErr == nil {
				a.settings = next
			}
			a.mu.Unlock()
			if saveErr != nil {
				return "", saveErr
			}
		}
		return "", a.fail("login", err)
	}
	if request.Profile.EnrollmentMode == "authority" {
		request.Profile, err = hydrateEnrollmentProfile(candidate, request.Profile)
		if err != nil {
			return "", err
		}
	}

	a.mu.Lock()
	previous := a.settings
	next := upsertProfile(previous, request.Profile)
	if err := a.store.save(next); err != nil {
		a.mu.Unlock()
		return "", err
	}
	old := a.client
	a.client = candidate
	a.settings = next
	a.mu.Unlock()
	connected = true
	if old != nil {
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
	candidate, err := a.credentials.Open(profile)
	if err != nil {
		a.mu.Unlock()
		return err
	}
	next := a.settings
	next.ActiveProfileID = profile.ID
	if err := a.store.save(next); err != nil {
		a.mu.Unlock()
		_ = candidate.Close()
		return err
	}
	old := a.client
	a.client = candidate
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
		old := a.client
		a.client = nil
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
	old := a.client
	a.client = nil
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
	var old *desktopbinding.Client
	if active {
		old = a.client
		a.client = nil
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
	if a.client != nil {
		_ = a.client.Close()
		a.client = nil
	}
	settings, err := a.store.reset(confirm)
	if err == nil {
		a.settings = settings
	}
	result, encodeErr := marshalJSON(a.settings)
	a.mu.Unlock()
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
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	profile, _ := findProfile(a.settings, a.settings.ActiveProfileID)
	a.mu.Unlock()
	if profile.EnrollmentMode != "authority" {
		return client.IdentityJSON()
	}
	statusJSON, err := client.EnrollmentStatusJSON()
	if err != nil {
		return "", err
	}
	var status enrollmentStatus
	if err := decodeStrictJSON([]byte(statusJSON), &status); err != nil {
		return "", err
	}
	return marshalJSON(publicIdentity{NodeID: status.NodeID, PublicKey: status.DevicePublicKey})
}

func (a *App) Connect() error {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.mu.Lock()
	profile, exists := findProfile(a.settings, a.settings.ActiveProfileID)
	old := a.client
	a.client = nil
	a.mu.Unlock()
	if !exists {
		if old != nil {
			a.mu.Lock()
			a.client = old
			a.mu.Unlock()
		}
		return errors.New("no active profile")
	}
	if old != nil {
		_ = old.Close()
	}
	candidate, err := a.credentials.Open(profile)
	if err != nil {
		return err
	}
	a.mu.Lock()
	current, currentExists := findProfile(a.settings, a.settings.ActiveProfileID)
	if !currentExists || current.ID != profile.ID {
		a.mu.Unlock()
		_ = candidate.Close()
		return errors.New("active profile changed while reconnecting")
	}
	a.client = candidate
	a.mu.Unlock()
	if err := connectClient(candidate, profile, "", false); err != nil {
		return err
	}
	a.appendLog("info", "managed TCP connection started")
	return nil
}

func connectClient(client *desktopbinding.Client, profile Profile, permitJSON string, allowTOFU bool) error {
	if profile.EnrollmentMode == "authority" {
		statusJSON, err := client.EnrollmentStatusJSON()
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
			resultJSON, err := client.EnrollTCP(profile.Endpoint, permitJSON, allowTOFU, expectedParentID, profile.ParentPublicKey, profile.AuthorityPublicKey, defaultRequestTimeoutMS)
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
		if err := client.StartEnrolledTCP(profile.Endpoint); err != nil {
			return fmt.Errorf("connect enrolled parent: %w", err)
		}
		if err := client.WaitConnected(defaultRequestTimeoutMS); err != nil {
			return fmt.Errorf("wait for enrolled connection: %w", err)
		}
		return nil
	}
	parentID, err := parsePositiveInt64(profile.ParentNodeID, "parent_node_id")
	if err != nil {
		return err
	}
	if err := client.TrustParent(parentID, profile.ParentPublicKey); err != nil {
		return fmt.Errorf("trust parent: %w", err)
	}
	if err := client.StartTCP(profile.Endpoint, parentID, permitJSON); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	if err := client.WaitConnected(defaultRequestTimeoutMS); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return connectionTimeoutGuidance(err, permitJSON)
		}
		return fmt.Errorf("wait for connection: %w", err)
	}
	return nil
}

func hydrateEnrollmentProfile(client *desktopbinding.Client, profile Profile) (Profile, error) {
	statusJSON, err := client.EnrollmentStatusJSON()
	if err != nil {
		return Profile{}, err
	}
	var status enrollmentStatus
	if err := decodeStrictJSON([]byte(statusJSON), &status); err != nil {
		return Profile{}, err
	}
	if status.Status != "enrolled" || status.NodeID == "" {
		return Profile{}, errors.New("Enrollment did not persist a granted Node ID")
	}
	profile.NodeID = status.NodeID
	profile.ParentNodeID = status.ParentNodeID
	profile.ParentPublicKey = status.ParentPublicKey
	profile.AuthorityNodeID = status.AuthorityNodeID
	profile.AuthorityPublicKey = status.AuthorityPublicKey
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
	old := a.client
	a.client = nil
	if exists {
		client, err := a.credentials.Open(profile)
		if err != nil {
			a.client = old
			a.mu.Unlock()
			return err
		}
		a.client = client
	}
	a.mu.Unlock()
	if old != nil {
		_ = old.Close()
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
	client := a.client
	a.client = nil
	a.mu.Unlock()
	if client == nil {
		return nil
	}
	return client.Close()
}

func (a *App) currentClient() (*desktopbinding.Client, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		return nil, errors.New("desktop is signed out")
	}
	return a.client, nil
}

func (a *App) openProfileLocked(profile Profile) error {
	client, err := a.credentials.Open(profile)
	if err != nil {
		return err
	}
	a.client = client
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
