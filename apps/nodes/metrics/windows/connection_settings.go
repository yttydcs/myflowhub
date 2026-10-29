package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yttydcs/myflowhub/internal/keystore"
)

// The GUI remembers public connection parameters, never a one-time Permit.
// Product identity and collection policy remain in the selected state directory.
type savedConnectionV1 struct {
	Version    int            `json:"version"`
	Active     bool           `json:"active"`
	Connection startRequestV1 `json:"connection"`
}

func (s savedConnectionV1) Validate() error {
	if s.Version != 1 || s.Connection.Permit != nil || !filepath.IsAbs(s.Connection.StateDirectory) {
		return errors.New("saved connection requires version 1, an absolute state directory and no Permit")
	}
	return s.Connection.Validate()
}

func (a *App) connectionStore() (*keystore.Store, error) {
	directory := a.configDirectory
	if directory == "" {
		directory = os.Getenv("MFH_METRICS_CONFIG_DIR")
	}
	if directory == "" {
		root, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		directory = filepath.Join(root, "MyFlowHub", "metrics")
	}
	if !filepath.IsAbs(directory) {
		return nil, errors.New("MFH_METRICS_CONFIG_DIR must be absolute")
	}
	return keystore.New(directory)
}

func (a *App) saveConnection(saved savedConnectionV1) error {
	if err := saved.Validate(); err != nil {
		return err
	}
	store, err := a.connectionStore()
	if err != nil {
		return err
	}
	return store.Save("connection.json", saved)
}

func (a *App) restoreConnection() error {
	store, err := a.connectionStore()
	if err != nil {
		return err
	}
	info, err := os.Stat(filepath.Join(store.Root(), "connection.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() > uiPayloadLimit {
		return errors.New("saved connection exceeds size limit")
	}
	var saved savedConnectionV1
	found, err := store.Load("connection.json", &saved)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("saved connection disappeared while loading")
	}
	if err := saved.Validate(); err != nil {
		return err
	}
	a.mu.Lock()
	a.savedConnection = &saved
	a.mu.Unlock()
	if !saved.Active {
		return nil
	}
	// Automatic restart must not silently recreate a deleted identity.
	if _, err := os.Stat(filepath.Join(saved.Connection.StateDirectory, "state", "identity.json")); err != nil {
		return fmt.Errorf("saved Metrics identity is unavailable; restore its state directory before reconnecting: %w", err)
	}
	raw, err := json.Marshal(saved.Connection)
	if err != nil {
		return err
	}
	_, err = a.Start(string(raw))
	return err
}
