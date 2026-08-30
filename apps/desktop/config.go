package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const desktopSettingsVersion = 2

var profileIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

type Settings struct {
	Version         int       `json:"version"`
	ActiveProfileID string    `json:"active_profile_id,omitempty"`
	Profiles        []Profile `json:"profiles"`
	UpdatedAtUnixMS int64     `json:"updated_at_unix_ms"`
}

type Profile struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	EnrollmentMode     string `json:"enrollment_mode,omitempty"`
	NodeID             string `json:"node_id"`
	Endpoint           string `json:"endpoint"`
	ParentNodeID       string `json:"parent_node_id"`
	ParentPublicKey    string `json:"parent_public_key"`
	AuthorityNodeID    string `json:"authority_node_id,omitempty"`
	AuthorityPublicKey string `json:"authority_public_key,omitempty"`
	AutoConnect        bool   `json:"auto_connect"`
	CreatedAtUnixMS    int64  `json:"created_at_unix_ms"`
	UpdatedAtUnixMS    int64  `json:"updated_at_unix_ms"`
}

type LoginRequest struct {
	Profile    Profile `json:"profile"`
	PermitJSON string  `json:"permit_json,omitempty"`
	AllowTOFU  bool    `json:"allow_tofu,omitempty"`
}

type settingsStore struct{ root string }

func newSettingsStore(root string) (*settingsStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("locate desktop config directory: %w", err)
		}
		root = filepath.Join(base, "MyFlowHub", "desktop-resource-workspace")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve desktop config directory: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create desktop config directory: %w", err)
	}
	return &settingsStore{root: abs}, nil
}

func ResetSettings(root, confirmation string) error {
	store, err := newSettingsStore(root)
	if err != nil {
		return err
	}
	_, err = store.reset(confirmation)
	return err
}

func (s *settingsStore) load() (Settings, error) {
	data, err := os.ReadFile(s.path())
	if errors.Is(err, os.ErrNotExist) {
		value := defaultSettings()
		if err := s.save(value); err != nil {
			return Settings{}, err
		}
		return value, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read desktop settings: %w", err)
	}
	var value Settings
	if err := decodeStrictJSON(data, &value); err != nil {
		return Settings{}, fmt.Errorf("desktop settings are invalid; use explicit reset: %w", err)
	}
	if value.Version != desktopSettingsVersion {
		return Settings{}, fmt.Errorf("desktop settings version %d is unsupported; use explicit reset to version %d", value.Version, desktopSettingsVersion)
	}
	if err := validateSettings(value); err != nil {
		return Settings{}, fmt.Errorf("desktop settings are invalid; use explicit reset: %w", err)
	}
	return value, nil
}

func (s *settingsStore) save(value Settings) error {
	value.Version = desktopSettingsVersion
	value.UpdatedAtUnixMS = time.Now().UTC().UnixMilli()
	sort.Slice(value.Profiles, func(i, j int) bool { return value.Profiles[i].ID < value.Profiles[j].ID })
	if err := validateSettings(value); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode desktop settings: %w", err)
	}
	return writeAtomicDesktop(s.root, s.path(), append(data, '\n'))
}

func (s *settingsStore) stateDirectory(profileID string) (string, error) {
	if !profileIDPattern.MatchString(profileID) {
		return "", errors.New("desktop profile ID is invalid")
	}
	return filepath.Join(s.root, "profiles", profileID), nil
}

func (s *settingsStore) reset(confirm string) (Settings, error) {
	if confirm != "RESET DESKTOP V2" {
		return Settings{}, errors.New("explicit reset requires confirmation text RESET DESKTOP V2")
	}
	value := defaultSettings()
	if err := s.save(value); err != nil {
		return Settings{}, err
	}
	return value, nil
}

func (s *settingsStore) path() string { return filepath.Join(s.root, "settings.json") }

func defaultSettings() Settings {
	return Settings{Version: desktopSettingsVersion, Profiles: []Profile{}, UpdatedAtUnixMS: time.Now().UTC().UnixMilli()}
}

func validateSettings(value Settings) error {
	if value.Version != desktopSettingsVersion {
		return fmt.Errorf("settings version must be %d", desktopSettingsVersion)
	}
	if len(value.Profiles) > 64 {
		return errors.New("desktop supports at most 64 profiles")
	}
	seen := make(map[string]struct{}, len(value.Profiles))
	activeFound := value.ActiveProfileID == ""
	for index, profile := range value.Profiles {
		if err := validateProfile(profile); err != nil {
			return fmt.Errorf("profiles[%d]: %w", index, err)
		}
		if _, exists := seen[profile.ID]; exists {
			return fmt.Errorf("duplicate profile %q", profile.ID)
		}
		seen[profile.ID] = struct{}{}
		activeFound = activeFound || profile.ID == value.ActiveProfileID
	}
	if !activeFound {
		return errors.New("active_profile_id does not reference a profile")
	}
	return nil
}

func validateProfile(value Profile) error {
	if !profileIDPattern.MatchString(value.ID) {
		return errors.New("profile id must use lowercase letters, numbers, dot, underscore, or dash")
	}
	if name := strings.TrimSpace(value.Name); name == "" || len(name) > 80 {
		return errors.New("profile name must contain between 1 and 80 bytes")
	}
	mode := value.EnrollmentMode
	if mode == "" {
		mode = "legacy"
	}
	if mode != "legacy" && mode != "authority" {
		return errors.New("enrollment_mode must be legacy or authority")
	}
	if value.AuthorityNodeID != "" {
		if _, err := parsePositiveInt64(value.AuthorityNodeID, "authority_node_id"); err != nil {
			return err
		}
	}
	if len(value.AuthorityPublicKey) > 512 {
		return errors.New("authority_public_key must contain at most 512 bytes")
	}
	if mode == "legacy" {
		if _, err := parsePositiveInt64(value.NodeID, "node_id"); err != nil {
			return err
		}
		if _, err := parsePositiveInt64(value.ParentNodeID, "parent_node_id"); err != nil {
			return err
		}
		if key := strings.TrimSpace(value.ParentPublicKey); key == "" || len(key) > 512 {
			return errors.New("parent_public_key must contain between 1 and 512 bytes")
		}
	} else {
		if value.NodeID != "" {
			if _, err := parsePositiveInt64(value.NodeID, "node_id"); err != nil {
				return err
			}
		}
		if value.ParentNodeID != "" {
			if _, err := parsePositiveInt64(value.ParentNodeID, "parent_node_id"); err != nil {
				return err
			}
		}
		if len(value.ParentPublicKey) > 512 {
			return errors.New("pinned public keys must contain at most 512 bytes")
		}
	}
	if endpoint := strings.TrimSpace(value.Endpoint); endpoint == "" || len(endpoint) > 512 {
		return errors.New("endpoint must contain between 1 and 512 bytes")
	}
	return nil
}

func upsertProfile(settings Settings, profile Profile) Settings {
	return upsertProfileWithActivation(settings, profile, true)
}

func upsertInactiveProfile(settings Settings, profile Profile) Settings {
	return upsertProfileWithActivation(settings, profile, false)
}

func upsertProfileWithActivation(settings Settings, profile Profile, activate bool) Settings {
	now := time.Now().UTC().UnixMilli()
	profile.UpdatedAtUnixMS = now
	updated := false
	for index := range settings.Profiles {
		if settings.Profiles[index].ID != profile.ID {
			continue
		}
		profile.CreatedAtUnixMS = settings.Profiles[index].CreatedAtUnixMS
		settings.Profiles[index] = profile
		updated = true
		break
	}
	if !updated {
		profile.CreatedAtUnixMS = now
		settings.Profiles = append(settings.Profiles, profile)
	}
	if activate {
		settings.ActiveProfileID = profile.ID
	}
	return settings
}

func findProfile(settings Settings, id string) (Profile, bool) {
	for _, profile := range settings.Profiles {
		if profile.ID == id {
			return profile, true
		}
	}
	return Profile{}, false
}

func parsePositiveInt64(raw, name string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive signed 64-bit integer", name)
	}
	return value, nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are forbidden")
		}
		return err
	}
	return nil
}

func writeAtomicDesktop(directory, path string, data []byte) error {
	temporary, err := os.CreateTemp(directory, ".desktop-*.tmp")
	if err != nil {
		return fmt.Errorf("create desktop transaction: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect desktop transaction: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write desktop transaction: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync desktop transaction: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close desktop transaction: %w", err)
	}
	backup := path + ".bak"
	if err := os.Remove(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale desktop backup: %w", err)
	}
	_, statErr := os.Stat(path)
	hadPrevious := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect desktop transaction target: %w", statErr)
	}
	if hadPrevious {
		if err := os.Rename(path, backup); err != nil {
			return fmt.Errorf("protect previous desktop state: %w", err)
		}
	}
	if err := os.Rename(temporaryName, path); err != nil {
		if hadPrevious {
			if restoreErr := os.Rename(backup, path); restoreErr != nil {
				return fmt.Errorf("commit desktop transaction: %v; restore previous state: %w", err, restoreErr)
			}
		}
		return fmt.Errorf("commit desktop transaction: %w", err)
	}
	return nil
}
