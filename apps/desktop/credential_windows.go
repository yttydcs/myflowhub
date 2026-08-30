//go:build windows

package main

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
)

const protectedIdentityVersion = 1
const protectedEnrollmentVersion = 1

type protectedIdentityRecord struct {
	Version    int                `json:"version"`
	NodeID     uint64             `json:"node_id"`
	PublicKey  ed25519.PublicKey  `json:"public_key"`
	PrivateKey ed25519.PrivateKey `json:"private_key"`
}

type dpapiIdentityStore struct {
	directory      string
	path           string
	enrollmentPath string
}

func newPlatformIdentityStore(directory string) (platformCredentialBackend, error) {
	if directory == "" {
		return nil, errors.New("profile identity directory is required")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create profile identity directory: %w", err)
	}
	return &dpapiIdentityStore{
		directory: directory, path: filepath.Join(directory, "identity.dpapi"),
		enrollmentPath: filepath.Join(directory, "enrollment.dpapi"),
	}, nil
}

type protectedEnrollmentRecord struct {
	Version    int                       `json:"version"`
	Credential auth.EnrollmentCredential `json:"credential"`
}

func (s *dpapiIdentityStore) LoadEnrollmentCredential() (auth.EnrollmentCredential, bool, error) {
	protected, err := os.ReadFile(s.enrollmentPath)
	if errors.Is(err, os.ErrNotExist) {
		return auth.EnrollmentCredential{}, false, nil
	}
	if err != nil {
		return auth.EnrollmentCredential{}, false, fmt.Errorf("read DPAPI Enrollment credential: %w", err)
	}
	plain, err := unprotectDPAPI(protected)
	if err != nil {
		return auth.EnrollmentCredential{}, false, fmt.Errorf("unprotect DPAPI Enrollment credential: %w", err)
	}
	var record protectedEnrollmentRecord
	if err := decodeStrictJSON(plain, &record); err != nil {
		return auth.EnrollmentCredential{}, false, fmt.Errorf("decode DPAPI Enrollment credential: %w", err)
	}
	if record.Version != protectedEnrollmentVersion {
		return auth.EnrollmentCredential{}, false, fmt.Errorf("unsupported DPAPI Enrollment credential version %d", record.Version)
	}
	return record.Credential, true, nil
}

func (s *dpapiIdentityStore) SaveEnrollmentCredential(credential auth.EnrollmentCredential) error {
	plain, err := json.Marshal(protectedEnrollmentRecord{Version: protectedEnrollmentVersion, Credential: credential})
	if err != nil {
		return fmt.Errorf("encode DPAPI Enrollment credential: %w", err)
	}
	protected, err := protectDPAPI(plain)
	if err != nil {
		return fmt.Errorf("protect DPAPI Enrollment credential: %w", err)
	}
	return writeAtomicDesktop(s.directory, s.enrollmentPath, protected)
}

func platformCredentialMode() string { return "windows-dpapi-user" }

func (s *dpapiIdentityStore) LoadIdentity() (auth.Identity, bool, error) {
	protected, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return auth.Identity{}, false, nil
	}
	if err != nil {
		return auth.Identity{}, false, fmt.Errorf("read DPAPI identity: %w", err)
	}
	plain, err := unprotectDPAPI(protected)
	if err != nil {
		return auth.Identity{}, false, fmt.Errorf("unprotect DPAPI identity: %w", err)
	}
	var record protectedIdentityRecord
	if err := decodeStrictJSON(plain, &record); err != nil {
		return auth.Identity{}, false, fmt.Errorf("decode DPAPI identity: %w", err)
	}
	if record.Version != protectedIdentityVersion {
		return auth.Identity{}, false, fmt.Errorf("unsupported DPAPI identity version %d", record.Version)
	}
	return auth.Identity{NodeID: protocol.NodeID(record.NodeID), PublicKey: record.PublicKey, PrivateKey: record.PrivateKey}, true, nil
}

func (s *dpapiIdentityStore) SaveIdentity(identity auth.Identity) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	plain, err := json.Marshal(protectedIdentityRecord{
		Version: protectedIdentityVersion, NodeID: uint64(identity.NodeID),
		PublicKey: identity.PublicKey, PrivateKey: identity.PrivateKey,
	})
	if err != nil {
		return fmt.Errorf("encode DPAPI identity: %w", err)
	}
	protected, err := protectDPAPI(plain)
	if err != nil {
		return fmt.Errorf("protect DPAPI identity: %w", err)
	}
	return writeAtomicDesktop(s.directory, s.path, protected)
}

func protectDPAPI(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("DPAPI input is empty")
	}
	input := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var output windows.DataBlob
	if err := windows.CryptProtectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return append([]byte(nil), unsafe.Slice(output.Data, output.Size)...), nil
}

func unprotectDPAPI(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("DPAPI payload is empty")
	}
	input := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var output windows.DataBlob
	if err := windows.CryptUnprotectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return append([]byte(nil), unsafe.Slice(output.Data, output.Size)...), nil
}
