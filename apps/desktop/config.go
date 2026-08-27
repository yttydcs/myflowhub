package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const desktopSettingsVersion = 1

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type Settings struct {
	Version         int    `json:"version"`
	Profile         string `json:"profile"`
	NodeID          string `json:"node_id"`
	Endpoint        string `json:"endpoint,omitempty"`
	ParentNodeID    string `json:"parent_node_id,omitempty"`
	ParentPublicKey string `json:"parent_public_key,omitempty"`
	PermitJSON      string `json:"permit_json,omitempty"`
	UpdatedAtUnixMS int64  `json:"updated_at_unix_ms"`
}

type settingsStore struct {
	root string
}

func newSettingsStore(root string) (*settingsStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("locate desktop config directory: %w", err)
		}
		root = filepath.Join(base, "MyFlowHub", "desktop-vnext")
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
	if err := json.Unmarshal(data, &value); err != nil {
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
	if err := validateSettings(value); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode desktop settings: %w", err)
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(s.root, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create desktop settings transaction: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect desktop settings transaction: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write desktop settings transaction: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync desktop settings transaction: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close desktop settings transaction: %w", err)
	}
	if err := os.Rename(temporaryName, s.path()); err != nil {
		return fmt.Errorf("commit desktop settings: %w", err)
	}
	return nil
}

func (s *settingsStore) stateDirectory(profile string) (string, error) {
	if !profileNamePattern.MatchString(profile) {
		return "", errors.New("desktop profile name is invalid")
	}
	return filepath.Join(s.root, "profiles", profile), nil
}

func (s *settingsStore) reset(confirm string) (Settings, error) {
	if confirm != "RESET DESKTOP V1" {
		return Settings{}, errors.New("explicit reset requires confirmation text RESET DESKTOP V1")
	}
	value := defaultSettings()
	if err := s.save(value); err != nil {
		return Settings{}, err
	}
	return value, nil
}

func (s *settingsStore) path() string { return filepath.Join(s.root, "settings.json") }

func defaultSettings() Settings {
	return Settings{Version: desktopSettingsVersion, Profile: "default", NodeID: "2", UpdatedAtUnixMS: time.Now().UTC().UnixMilli()}
}

func validateSettings(value Settings) error {
	if value.Version != desktopSettingsVersion {
		return fmt.Errorf("settings version must be %d", desktopSettingsVersion)
	}
	if !profileNamePattern.MatchString(value.Profile) {
		return errors.New("profile must contain only letters, numbers, dot, underscore, or dash")
	}
	nodeID, err := strconv.ParseUint(value.NodeID, 10, 64)
	if err != nil || nodeID == 0 || nodeID > uint64(^uint64(0)>>1) {
		return errors.New("node_id must be a positive signed 64-bit integer")
	}
	if value.ParentNodeID != "" {
		parentID, err := strconv.ParseUint(value.ParentNodeID, 10, 64)
		if err != nil || parentID == 0 || parentID > uint64(^uint64(0)>>1) {
			return errors.New("parent_node_id must be a positive signed 64-bit integer")
		}
	}
	if len(value.Endpoint) > 512 || len(value.ParentPublicKey) > 512 || len(value.PermitJSON) > 1<<20 {
		return errors.New("desktop setting exceeds its size limit")
	}
	if value.PermitJSON != "" && !json.Valid([]byte(value.PermitJSON)) {
		return errors.New("permit_json must be valid JSON")
	}
	return nil
}
