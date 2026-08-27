package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/yttydcs/myflowhub/runtime/auth"
)

func TestOpenPreservesIdentityTrustAndPolicyAcrossRestart(t *testing.T) {
	directory := t.TempDir()
	first, err := Open(directory, 1)
	if err != nil {
		t.Fatal(err)
	}
	child, _ := auth.GenerateIdentity(2)
	if err := first.Trust.Add(child.NodeID, child.PublicKey); err != nil {
		t.Fatal(err)
	}
	request := auth.Request{Subject: 2, Action: auth.ActionSubscribe, Resource: resourceID(1, "system/catalog")}
	if err := first.Policy.Grant(request); err != nil {
		t.Fatal(err)
	}
	second, err := Open(directory, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Identity.PrivateKey, second.Identity.PrivateKey) {
		t.Fatal("identity changed across restart")
	}
	if key, ok := second.Trust.PublicKey(2); !ok || !bytes.Equal(key, child.PublicKey) {
		t.Fatal("trusted child did not survive restart")
	}
	if err := second.Policy.Authorize(testContext(), request); err != nil {
		t.Fatalf("policy did not survive restart: %v", err)
	}
}

func TestOpenFailsOnCorruptIdentityWithoutReplacingIt(t *testing.T) {
	directory := t.TempDir()
	if _, err := Open(directory, 1); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "state", "identity.json")
	corrupt := []byte(`{"version":1,"node_id":1,"public_key":"broken"}`)
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(directory, 1); err == nil {
		t.Fatal("corrupt identity was silently replaced")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, corrupt) {
		t.Fatalf("corrupt identity was modified: %v", err)
	}
}
