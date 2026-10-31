package metrics

import (
	"context"
	"testing"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
)

func TestLinuxConfigAndResourcesArePlatformScopedAndReadOnly(t *testing.T) {
	runtime := newMetricsNode(t)
	defer runtime.Close()
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	controller, err := Register(ControllerConfig{Node: runtime, Store: store, Platform: "linux"})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	if len(controller.Config().Settings) != 4 || len(controller.States()) != 4 {
		t.Fatal("Linux must expose exactly four read-only metrics")
	}
	for _, definition := range Definitions("linux") {
		if definition.Controllable {
			t.Fatalf("Linux metric %s is controllable", definition.Name)
		}
		variable := controller.variables[definition.Name].variable
		if _, ok := variable.Descriptor().Capability(protocol.CapabilityWrite); ok {
			t.Fatal("Linux metric exposes write")
		}
	}
	state := controller.configVariable.Snapshot().State
	if err := protocol.ValidateVariableValue(*state.Definition, state.Value); err != nil {
		t.Fatal(err)
	}
	var decoded ConfigV1
	if err := protocol.DecodeVariablePayload(*state.Definition, state.Value, protocol.DefaultMaxPayload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Platform != "linux" || len(decoded.Settings) != 4 {
		t.Fatal("wrong Linux config schema")
	}
	settings := append([]SettingV1(nil), decoded.Settings...)
	settings[0].IntervalMS = 3000
	next, err := controller.ApplyConfig(context.Background(), ConfigUpdateV1{Version: 1, ExpectedRevision: decoded.Revision, Settings: settings})
	if err != nil || next.Revision != 2 {
		t.Fatalf("config update: %+v %v", next, err)
	}
	loaded, err := loadConfig(store, "linux")
	if err != nil || loaded.Revision != 2 {
		t.Fatalf("persisted config: %+v %v", loaded, err)
	}
	if _, err := loadConfig(store, "windows"); err == nil {
		t.Fatal("cross-platform state was accepted")
	}
	bad := next
	bad.Settings = append([]SettingV1(nil), next.Settings...)
	bad.Settings[0].Writable = true
	if err := bad.Validate(); err == nil {
		t.Fatal("Linux write configuration accepted")
	}
	if len(Definitions("windows")) != 10 {
		t.Fatal("Windows metrics changed")
	}
}
