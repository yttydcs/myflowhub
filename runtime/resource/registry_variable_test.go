package resource

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func testVariableSpec(name string) VariableSpec {
	return VariableSpec{
		Name:            name,
		ContentType:     "application/json",
		Schema:          "test.status.v1",
		ReadPermission:  "test.status.read",
		MaxPayloadBytes: 64,
		Initial:         []byte(`{"online":true}`),
	}
}

func TestRegistryVariableBindsOwnerAndDefaultsToReadOnly(t *testing.T) {
	registry, err := NewRegistry(42)
	if err != nil {
		t.Fatal(err)
	}
	variable, err := registry.Variable(testVariableSpec("device/status"))
	if err != nil {
		t.Fatal(err)
	}

	descriptor := variable.Descriptor()
	if descriptor.ID != (protocol.ResourceID{Owner: 42, Name: "device/status"}) {
		t.Fatalf("variable was not bound to registry owner: %#v", descriptor.ID)
	}
	if descriptor.Type != protocol.ResourceTypeVariable {
		t.Fatalf("unexpected resource type: %q", descriptor.Type)
	}
	if _, ok := descriptor.Capability(protocol.CapabilityWrite); ok {
		t.Fatal("variable without an explicit write permission became remotely writable")
	}
	read, ok := descriptor.Capability(protocol.CapabilityRead)
	if !ok || read.Permission != "test.status.read" || read.OutputSchema != "test.status.v1" {
		t.Fatalf("unexpected read capability: %#v", read)
	}
	if descriptor.Limits.MaxPayloadBytes != 64 {
		t.Fatalf("unexpected payload limit: %d", descriptor.Limits.MaxPayloadBytes)
	}
	resolved, ok := registry.Resolve(descriptor.ID)
	if !ok || resolved != variable {
		t.Fatal("registered variable did not resolve to the returned instance")
	}
}

func TestRegistryVariableRequiresExplicitWritePermission(t *testing.T) {
	registry, _ := NewRegistry(7)
	spec := testVariableSpec("device/settings")
	spec.WritePermission = "test.settings.write"
	variable, err := registry.Variable(spec)
	if err != nil {
		t.Fatal(err)
	}
	write, ok := variable.Descriptor().Capability(protocol.CapabilityWrite)
	if !ok || write.Permission != spec.WritePermission {
		t.Fatalf("unexpected write capability: %#v", write)
	}
}

func TestRegistryVariableRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*VariableSpec)
		want   string
	}{
		{name: "name", mutate: func(spec *VariableSpec) { spec.Name = "" }, want: "name is required"},
		{name: "content type", mutate: func(spec *VariableSpec) { spec.ContentType = "" }, want: "content type is required"},
		{name: "schema", mutate: func(spec *VariableSpec) { spec.Schema = "" }, want: "schema is required"},
		{name: "read permission", mutate: func(spec *VariableSpec) { spec.ReadPermission = "" }, want: "read permission is required"},
		{name: "payload limit", mutate: func(spec *VariableSpec) { spec.MaxPayloadBytes = 0 }, want: "max payload bytes must be positive"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry, _ := NewRegistry(1)
			spec := testVariableSpec("status")
			test.mutate(&spec)
			if _, err := registry.Variable(spec); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q error, got %v", test.want, err)
			}
		})
	}
}

func TestRegistryVariableRejectsInvalidDescriptorFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*VariableSpec)
	}{
		{name: "name", mutate: func(spec *VariableSpec) { spec.Name = "/status" }},
		{name: "content type", mutate: func(spec *VariableSpec) { spec.ContentType = strings.Repeat("c", protocol.MaxContentTypeBytes+1) }},
		{name: "schema", mutate: func(spec *VariableSpec) { spec.Schema = strings.Repeat("s", protocol.MaxSchemaBytes+1) }},
		{name: "read permission", mutate: func(spec *VariableSpec) { spec.ReadPermission = strings.Repeat("r", protocol.MaxIdentifierBytes+1) }},
		{name: "write permission", mutate: func(spec *VariableSpec) { spec.WritePermission = strings.Repeat("w", protocol.MaxIdentifierBytes+1) }},
		{name: "payload limit", mutate: func(spec *VariableSpec) { spec.MaxPayloadBytes = protocol.DefaultMaxPayload + 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry, _ := NewRegistry(1)
			spec := testVariableSpec("status")
			test.mutate(&spec)
			if _, err := registry.Variable(spec); err == nil {
				t.Fatal("invalid variable spec was accepted")
			}
		})
	}
}

func TestRegistryVariableRejectsInitialPayloadOverLimit(t *testing.T) {
	registry, _ := NewRegistry(1)
	spec := testVariableSpec("status")
	spec.MaxPayloadBytes = 4
	spec.Initial = []byte("large")
	if _, err := registry.Variable(spec); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected value-too-large error, got %v", err)
	}
}

func TestRegistryVariableUsesAtomicRegistrationAndCatalogUpdate(t *testing.T) {
	registry, _ := NewRegistry(9)
	before := registry.Catalog().Snapshot()
	variable, err := registry.Variable(testVariableSpec("device/status"))
	if err != nil {
		t.Fatal(err)
	}
	after := registry.Catalog().Snapshot()
	if after.Revision != before.Revision+1 {
		t.Fatalf("catalog variable revision did not advance once: %d -> %d", before.Revision, after.Revision)
	}
	if _, err := registry.Variable(testVariableSpec("device/status")); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	if got := registry.Catalog().Snapshot().Revision; got != after.Revision {
		t.Fatalf("failed duplicate registration changed catalog revision: %d -> %d", after.Revision, got)
	}

	var catalog protocol.ResourceCatalogV2
	if err := json.Unmarshal(after.Value, &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.Revision != 2 {
		t.Fatalf("unexpected catalog payload revision: %d", catalog.Revision)
	}
	found := false
	for _, descriptor := range catalog.Resources {
		if descriptor.ID == variable.Descriptor().ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("registered variable missing from catalog: %#v", catalog.Resources)
	}
}

func TestRegistryVariableRetainsSetAndObserveSemantics(t *testing.T) {
	registry, _ := NewRegistry(1)
	variable, err := registry.Variable(testVariableSpec("status"))
	if err != nil {
		t.Fatal(err)
	}
	observed := make(chan Observation, 1)
	initial, cancel, err := variable.Observe(func(observation Observation) { observed <- observation })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if !initial.Snapshot || initial.Revision != 1 || string(initial.Value) != `{"online":true}` {
		t.Fatalf("unexpected initial observation: %#v", initial)
	}
	update, err := variable.Set([]byte(`{"online":false}`))
	if err != nil {
		t.Fatal(err)
	}
	if update.Revision != 2 {
		t.Fatalf("unexpected update revision: %d", update.Revision)
	}
	event := <-observed
	if event.Snapshot || event.Revision != update.Revision || string(event.Value) != `{"online":false}` {
		t.Fatalf("unexpected observation: %#v", event)
	}
}

func TestNilRegistryCannotDeclareVariable(t *testing.T) {
	var registry *Registry
	if _, err := registry.Variable(testVariableSpec("status")); err == nil {
		t.Fatal("nil registry accepted variable declaration")
	}
}
