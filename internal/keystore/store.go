package keystore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

var fileName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*\.json$`)

type Store struct {
	root string
	mu   sync.Mutex
}

func New(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("keystore root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve keystore root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create keystore root: %w", err)
	}
	if err := os.Chmod(abs, 0o700); err != nil {
		return nil, fmt.Errorf("protect keystore root: %w", err)
	}
	return &Store{root: abs}, nil
}

func (s *Store) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *Store) Load(name string, target any) (bool, error) {
	if s == nil {
		return false, errors.New("keystore is required")
	}
	if target == nil {
		return false, errors.New("keystore load target is required")
	}
	path, err := s.path(name)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", name, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false, fmt.Errorf("decode %s: %w", name, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return false, fmt.Errorf("decode %s: %w", name, err)
	}
	return true, nil
}

func (s *Store) Save(name string, value any) error {
	if s == nil {
		return errors.New("keystore is required")
	}
	if value == nil {
		return errors.New("keystore value is required")
	}
	path, err := s.path(name)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	data = append(data, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeAtomic(path, data)
}

func (s *Store) path(name string) (string, error) {
	if !fileName.MatchString(name) {
		return "", fmt.Errorf("invalid keystore file name %q", name)
	}
	return filepath.Join(s.root, name), nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values are forbidden")
	}
	return err
}

func writeAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)
	temp, err := os.CreateTemp(directory, ".mfh-state-*")
	if err != nil {
		return fmt.Errorf("create temporary state: %w", err)
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()
	if err := temp.Chmod(0o600); err != nil {
		return fmt.Errorf("protect temporary state: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("write temporary state: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary state: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary state: %w", err)
	}

	backup := path + ".bak"
	hadCurrent := false
	if _, err := os.Lstat(path); err == nil {
		hadCurrent = true
		if err := os.Remove(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale state backup: %w", err)
		}
		if err := os.Rename(path, backup); err != nil {
			return fmt.Errorf("backup current state: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect current state: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if hadCurrent {
			_ = os.Rename(backup, path)
		}
		return fmt.Errorf("activate new state: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("protect state file: %w", err)
	}
	return nil
}
