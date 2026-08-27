package keystore

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fixture struct {
	Version int    `json:"version"`
	Value   string `json:"value"`
}

func TestStoreRoundTripBackupAndStrictDecode(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save("identity.json", fixture{Version: 1, Value: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save("identity.json", fixture{Version: 1, Value: "second"}); err != nil {
		t.Fatal(err)
	}
	var got fixture
	ok, err := store.Load("identity.json", &got)
	if err != nil || !ok || got.Value != "second" {
		t.Fatalf("load current state: ok=%v value=%+v err=%v", ok, got, err)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "identity.json.bak")); err != nil {
		t.Fatalf("versioned backup missing: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(store.Root(), "identity.json"))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("state mode = %v, err=%v", info.Mode().Perm(), err)
		}
	}
	if err := os.WriteFile(filepath.Join(store.Root(), "broken.json"), []byte(`{"version":1,"value":"x","unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load("broken.json", &fixture{}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("strict decode did not reject unknown field: %v", err)
	}
}

func TestStoreRejectsPathTraversal(t *testing.T) {
	store, _ := New(t.TempDir())
	if err := store.Save("../outside.json", fixture{}); err == nil {
		t.Fatal("path traversal accepted")
	}
}
