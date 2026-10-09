package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

// Optional additions to the isolated TCP fixture. These are real owner resources
// used to inspect the production Desktop without its builtin product schemas.
func registerVariableGUIFixtures(t *testing.T, owner *node.Node, authority *hostconfig.Runtime) func() {
	t.Helper()
	if os.Getenv("MFH_VARIABLE_GUI_FIXTURES") != "1" {
		return func() {}
	}
	min, max := 0.0, 100.0
	fixtures := []struct {
		name       string
		definition protocol.DataSchemaDefinition
		value      string
		writable   bool
	}{
		{"qa/switch", protocol.DataSchemaDefinition{Type: "boolean", Title: "通用开关"}, "false", true},
		{"qa/level", protocol.DataSchemaDefinition{Type: "number", Title: "通用百分比", Unit: "%", Minimum: &min, Maximum: &max}, "37", true},
		{"qa/mode", protocol.DataSchemaDefinition{Type: "string", Title: "通用模式", Enum: []string{"auto", "quiet", "work"}}, `"auto"`, true},
		{"qa/object", protocol.DataSchemaDefinition{Type: "object", Title: "通用配置", Properties: []protocol.DataSchemaProperty{{Name: "enabled", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "boolean", Title: "启用"}}, {Name: "name", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "string", Title: "名称"}}}}, `{"enabled":false,"name":"陌生节点"}`, true},
		{"qa/array", protocol.DataSchemaDefinition{Type: "array", Title: "通用数组", Items: &protocol.DataSchemaDefinition{Type: "integer"}}, "[1,2,3]", true},
		{"qa/ticks", protocol.DataSchemaDefinition{Type: "integer", Title: "实时更新"}, "0", false},
	}
	var ticks *resource.Variable
	for _, fixture := range fixtures {
		fixture.definition.ID = "unregistered.demo." + fixture.name
		spec := resource.VariableSpec{Name: fixture.name, Definition: fixture.definition, ReadPermission: "qa.read", MaxPayloadBytes: 4096, Initial: resource.VariableState{Status: "ready", Value: json.RawMessage(fixture.value)}}
		capabilities := []protocol.CapabilityID{protocol.CapabilityRead, protocol.CapabilitySubscribe}
		if fixture.writable {
			spec.WritePermission = "qa.write"
			capabilities = append(capabilities, protocol.CapabilityWrite)
		}
		value, err := owner.Registry().Variable(spec)
		if err != nil {
			t.Fatal(err)
		}
		if fixture.name == "qa/ticks" {
			ticks = value
		}
		for _, capability := range capabilities {
			if err := authority.Policy.Grant(auth.Request{Subject: 7, Resource: value.Descriptor().ID, Capability: capability}); err != nil {
				t.Fatal(err)
			}
		}
	}
	last := time.Now()
	counter := 0
	return func() {
		if time.Since(last) < time.Second {
			return
		}
		last = time.Now()
		counter++
		raw, err := json.Marshal(counter)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ticks.Set(resource.VariableState{Status: "ready", Value: raw}); err != nil {
			t.Fatal(err)
		}
	}
}

// Prepare only this fixture's state for a separately launched production Metrics
// process. Its exact grants let Desktop inspect real samples without changing
// any existing identity, policy or running product process.
func prepareVariableGUIMetrics(t *testing.T, worktree, configRoot, endpoint string, parent *hostconfig.Runtime) {
	t.Helper()
	if os.Getenv("MFH_VARIABLE_GUI_FIXTURES") != "1" {
		return
	}
	stateDir := topologyGUICheckPath(t, worktree, filepath.Join(configRoot, "metrics-state"))
	state, err := auth.OpenState(stateDir, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := state.Policy.Close(); err != nil {
			t.Error(err)
		}
	}()
	names := []string{protocol.BuiltinResourceCatalog}
	for _, metric := range []metrics.Name{metrics.BatteryCharging, metrics.BatteryOnAC, metrics.BatteryPercent, metrics.BrightnessPercent, metrics.CPUPercent, metrics.MemoryPercent, metrics.NetworkOnline, metrics.NetworkType, metrics.VolumeMuted, metrics.VolumePercent} {
		names = append(names, metrics.ResourceName(metric))
	}
	for _, name := range names {
		for _, capability := range []protocol.CapabilityID{protocol.CapabilityRead, protocol.CapabilitySubscribe} {
			request := auth.Request{Subject: 7, Resource: protocol.ResourceID{Owner: 8, Name: name}, Capability: capability}
			if err := state.Policy.Grant(request); err != nil {
				t.Fatal(err)
			}
			// The shared parent also adjudicates this cross-child transit.
			if err := parent.Policy.Grant(request); err != nil {
				t.Fatal(err)
			}
		}
	}
	permit, err := parent.Admission.Issue(8, state.Identity.PublicKey, "variable-gui-metrics", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	permitPayload, err := protocol.EncodeJSONPayload(&permit, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	permitPath := filepath.Join(configRoot, "metrics-permit.json")
	if err := os.WriteFile(permitPath, permitPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := json.MarshalIndent(map[string]string{"state": stateDir, "endpoint": endpoint, "parent_key": base64.RawStdEncoding.EncodeToString(parent.Identity.PublicKey), "permit_file": permitPath}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configRoot, "metrics-launch.json"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
}
