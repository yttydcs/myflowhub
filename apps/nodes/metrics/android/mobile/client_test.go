package metricsmobile

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestClientIdentityIsDurable(t *testing.T) {
	client := NewClient()
	request, _ := json.Marshal(identityRequestV1{Version: 1, StateDirectory: filepath.Join(t.TempDir(), "state"), NodeID: "81"})
	first, err := client.Identity(string(request))
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Identity(string(request))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("identity changed: %s != %s", first, second)
	}
}

func TestClientIdleGuards(t *testing.T) {
	client := NewClient()
	status, err := client.Status()
	if err != nil || status == "" {
		t.Fatalf("idle status: %q %v", status, err)
	}
	if _, err := client.NextAction(); err == nil {
		t.Fatal("expected idle action error")
	}
	if _, err := client.NextNotification(); err == nil {
		t.Fatal("expected idle notification error")
	}
	if err := client.UpdateMetric("memory_percent", "20", ""); err == nil {
		t.Fatal("expected idle update error")
	}
}

func TestClientRejectsUnknownJSONFields(t *testing.T) {
	client := NewClient()
	if _, err := client.Identity(`{"version":1,"state_directory":"x","node_id":"1","unknown":true}`); err == nil {
		t.Fatal("expected strict boundary error")
	}
}
