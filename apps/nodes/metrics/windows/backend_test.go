package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestIdentityIsDurableAndUsesStringNodeID(t *testing.T) {
	app := NewApp()
	request := `{"version":1,"state_directory":` + quote(filepath.Join(t.TempDir(), "state")) + `,"node_id":"42"}`
	first, err := app.Identity(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.Identity(request)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("identity changed: %s != %s", first, second)
	}
	var value identityResponseV1
	if err := json.Unmarshal([]byte(first), &value); err != nil {
		t.Fatal(err)
	}
	if value.NodeID != "42" || value.PublicKey == "" {
		t.Fatalf("unexpected identity: %+v", value)
	}
}

func TestStartRejectsUnknownAndInvalidBoundaryFields(t *testing.T) {
	app := NewApp()
	if _, err := app.Start(`{"version":1,"unknown":true}`); err == nil {
		t.Fatal("expected strict JSON boundary error")
	}
	request := startRequestV1{
		Version: 1, StateDirectory: t.TempDir(), NodeID: "1", ParentNodeID: "2",
		Endpoint: "not an endpoint", ParentPublicKey: "invalid",
	}
	payload, err := json.Marshal(&request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Start(string(payload)); err == nil {
		t.Fatal("expected invalid start request error")
	}
}

func TestIdleStatusAndConfigurationGuard(t *testing.T) {
	app := NewApp()
	statusJSON, err := app.Status()
	if err != nil {
		t.Fatal(err)
	}
	var status statusV1
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		t.Fatal(err)
	}
	if status.Version != 1 || status.Running || status.Samples == nil {
		t.Fatalf("unexpected status: %+v", status)
	}
	if _, err := app.Configuration(); err == nil {
		t.Fatal("expected idle configuration error")
	}
}

func quote(value string) string {
	payload, _ := json.Marshal(value)
	return string(payload)
}
