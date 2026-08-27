package androidbinding

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClientLifecycle(t *testing.T) {
	client, err := NewClient(t.TempDir(), 71)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := client.IdentityJSON()
	if err != nil || !json.Valid([]byte(identity)) {
		t.Fatalf("invalid identity: %v (%s)", err, identity)
	}
	status, err := client.StatusJSON()
	if err != nil || !strings.Contains(status, "disconnected") {
		t.Fatalf("invalid initial status: %v (%s)", err, status)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := client.IdentityJSON(); err == nil {
		t.Fatal("closed Android client remained usable")
	}
}
