package config

import (
	"errors"
	"fmt"
	"sync"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

type Settings struct {
	mu    sync.RWMutex
	store *keystore.Store
	value protocol.ManagementConfigV1
}

func loadSettings(store *keystore.Store) (*Settings, error) {
	value := protocol.ManagementConfigV1{Version: 1, Revision: 1, Values: map[string]string{}}
	found, err := store.Load("host.json", &value)
	if err != nil {
		return nil, fmt.Errorf("load host settings: %w", err)
	}
	if err := value.Validate(); err != nil {
		return nil, fmt.Errorf("load host settings: %w", err)
	}
	result := &Settings{store: store, value: cloneManagementConfig(value)}
	if !found {
		if err := store.Save("host.json", value); err != nil {
			return nil, fmt.Errorf("initialize host settings: %w", err)
		}
	}
	return result, nil
}

func (s *Settings) Snapshot() protocol.ManagementConfigV1 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneManagementConfig(s.value)
}

func (s *Settings) Replace(update protocol.ManagementConfigUpdateV1) (protocol.ManagementConfigV1, error) {
	if err := update.Validate(); err != nil {
		return protocol.ManagementConfigV1{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if update.ExpectedRevision != s.value.Revision {
		return protocol.ManagementConfigV1{}, fmt.Errorf("%w: expected revision %d, current %d", errors.New("host settings revision conflict"), update.ExpectedRevision, s.value.Revision)
	}
	if s.value.Revision == ^uint64(0) {
		return protocol.ManagementConfigV1{}, errors.New("host settings revision exhausted")
	}
	next := protocol.ManagementConfigV1{Version: 1, Revision: s.value.Revision + 1, DisplayName: update.DisplayName, Values: cloneStrings(update.Values)}
	if err := next.Validate(); err != nil {
		return protocol.ManagementConfigV1{}, err
	}
	if err := s.store.Save("host.json", next); err != nil {
		return protocol.ManagementConfigV1{}, fmt.Errorf("persist host settings: %w", err)
	}
	s.value = next
	return cloneManagementConfig(next), nil
}

func cloneManagementConfig(value protocol.ManagementConfigV1) protocol.ManagementConfigV1 {
	value.Values = cloneStrings(value.Values)
	return value
}

func cloneStrings(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
