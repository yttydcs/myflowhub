//go:build windows

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopIdentityIsProtectedWithDPAPI(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	profile := testProfile("protected", "52")
	encoded, err := marshalJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.SaveProfileJSON(encoded); err != nil {
		t.Fatal(err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
	protected, err := os.ReadFile(filepath.Join(root, "profiles", profile.ID, "identity.dpapi"))
	if err != nil {
		t.Fatal(err)
	}
	if len(protected) == 0 || bytes.Contains(protected, []byte("private_key")) || jsonLike(protected) {
		t.Fatal("persistent identity was not stored as a DPAPI-protected payload")
	}
	reopened, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	identity, err := reopened.IdentityJSON()
	if err != nil || !strings.Contains(identity, `"node_id":"52"`) {
		t.Fatalf("DPAPI identity did not reopen: %v (%s)", err, identity)
	}
}

func TestDesktopEnrollmentCredentialIsProtectedSeparatelyWithDPAPI(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	profile := Profile{ID: "enrollment-protected", Name: "Enrollment Protected", EnrollmentMode: "authority", Endpoint: "127.0.0.1:7331"}
	encoded, err := marshalJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.PrepareProfileJSON(encoded); err != nil {
		t.Fatal(err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
	protected, err := os.ReadFile(filepath.Join(root, "profiles", profile.ID, "enrollment.dpapi"))
	if err != nil {
		t.Fatal(err)
	}
	if len(protected) == 0 || bytes.Contains(protected, []byte("private_key")) || jsonLike(protected) {
		t.Fatal("Enrollment credential was not stored as a separate DPAPI-protected payload")
	}
	if _, err := os.Stat(filepath.Join(root, "profiles", profile.ID, "identity.dpapi")); !os.IsNotExist(err) {
		t.Fatalf("Authority Profile unexpectedly wrote the legacy identity file: %v", err)
	}
}

func jsonLike(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}
