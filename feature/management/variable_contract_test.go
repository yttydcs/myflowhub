package management_test

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

func TestManagementVariablesReadAndObserveStandardContract(t *testing.T) {
	for _, owner := range []protocol.NodeID{1, 9007199254740993, ^protocol.NodeID(0)} {
		t.Run(strconv.FormatUint(uint64(owner), 10), func(t *testing.T) { testManagementVariableContract(t, owner) })
	}
}

func testManagementVariableContract(t *testing.T, owner protocol.NodeID) {
	t.Helper()
	value := newFixtureWithNodeID(t, owner)
	type observedVariable struct {
		name    string
		initial protocol.VariableValueV1
		events  chan resource.Observation
		target  protocol.ValidatedPayload
		decoder *sdk.VariableEventDecoder
	}
	var watched []observedVariable
	for _, item := range []struct {
		name, schema, permission string
		target                   protocol.ValidatedPayload
	}{
		{protocol.BuiltinManagementTopology, protocol.SchemaManagementTopologyV1, "management.topology.read", &protocol.ManagementTopologyV1{}},
		{protocol.BuiltinManagementHealth, protocol.SchemaManagementHealthV1, "management.health.read", &protocol.ManagementHealthV1{}},
		{protocol.BuiltinManagementConfig, protocol.SchemaManagementConfigV1, "management.config.read", &protocol.ManagementConfigV1{}},
		{protocol.BuiltinAdmissionStatus, protocol.SchemaAdmissionStatusV1, "management.admission.read", &protocol.AdmissionStatusV1{}},
	} {
		id := protocol.ResourceID{Owner: value.node.ID(), Name: item.name}
		registered, ok := value.node.Registry().Resolve(id)
		if !ok {
			t.Fatalf("missing variable %s", item.name)
		}
		descriptor := registered.Descriptor()
		if descriptor.Type != protocol.ResourceTypeVariable || descriptor.TypeVersion != 2 {
			t.Fatalf("nonstandard variable descriptor: %+v", descriptor)
		}
		read, _ := descriptor.Capability(protocol.CapabilityRead)
		subscribe, _ := descriptor.Capability(protocol.CapabilitySubscribe)
		if read.OutputSchema != protocol.SchemaVariableValueV1 || subscribe.EventSchema != protocol.SchemaVariableValueV1 || read.Permission != item.permission || subscribe.Permission != item.permission {
			t.Fatalf("wrong capabilities for %s", item.name)
		}
		if _, writable := descriptor.Capability(protocol.CapabilityWrite); writable {
			t.Fatalf("read-only variable %s became writable", item.name)
		}
		result, err := value.node.Registry().Operate(context.Background(), id, resource.OperationRequest{Capability: protocol.CapabilityRead})
		if err != nil {
			t.Fatal(err)
		}
		if result.Schema != protocol.SchemaVariableValueV1 {
			t.Fatal("read returned private schema")
		}
		var state protocol.VariableValueV1
		decode(t, result.Payload, &state)
		if state.Definition == nil || state.Definition.ID != item.schema || state.Status != "ready" {
			t.Fatalf("incomplete read for %s: %+v", item.name, state)
		}
		decodeVariableDomain(t, state, item.target)
		events := make(chan resource.Observation, 8)
		initial, cancel, err := registered.(resource.Observable).Observe(func(event resource.Observation) { events <- event })
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(cancel)
		if initial == nil || !initial.Snapshot || initial.Schema != protocol.SchemaVariableValueV1 {
			t.Fatalf("read/Observe snapshot mismatch for %s", item.name)
		}
		var observedState protocol.VariableValueV1
		decode(t, initial.Value, &observedState)
		if !reflect.DeepEqual(state, observedState) {
			t.Fatalf("read/Observe state mismatch for %s", item.name)
		}
		decoder := new(sdk.VariableEventDecoder)
		if _, err := decoder.Decode(sdk.Event{Kind: sdk.EventSnapshot, Resource: id, Schema: initial.Schema, Revision: initial.Revision, Value: initial.Value}); err != nil {
			t.Fatal(err)
		}
		watched = append(watched, observedVariable{item.name, state, events, item.target, decoder})
	}
	if err := value.node.Tree().AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	child, err := auth.GenerateIdentity(2)
	if err != nil {
		t.Fatal(err)
	}
	if err := value.state.Trust.Add(child.NodeID, child.PublicKey); err != nil {
		t.Fatal(err)
	}
	config := value.state.Settings.Snapshot()
	if _, err := value.state.Settings.Replace(protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: config.Revision, DisplayName: "standard", Values: config.Values}); err != nil {
		t.Fatal(err)
	}
	device, err := auth.GenerateDeviceIdentity()
	if err != nil {
		t.Fatal(err)
	}
	_, err = value.authority.IssuePermit("60000000000000000000000000000001", auth.DevicePublicKeyFingerprint(device.PublicKey), owner, false, "test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	for _, item := range watched {
		select {
		case event := <-item.events:
			if event.Failure != nil || event.Snapshot || event.Schema != protocol.SchemaVariableValueV1 {
				t.Fatalf("invalid update for %s: %+v", item.name, event)
			}
			var state protocol.VariableValueV1
			decode(t, event.Value, &state)
			previous, _ := protocol.ParseVariableRevision(item.initial.Revision)
			if state.Definition != nil || state.InstanceID != item.initial.InstanceID || state.Revision != strconv.FormatUint(event.Revision, 10) || event.Revision <= previous {
				t.Fatalf("invalid update identity/definition for %s: %+v", item.name, state)
			}
			if err := protocol.ValidateVariableValue(*item.initial.Definition, state.Value); err != nil {
				t.Fatal(err)
			}
			decoded, err := item.decoder.Decode(sdk.Event{Kind: sdk.EventData, Resource: protocol.ResourceID{Owner: owner, Name: item.name}, Schema: event.Schema, Revision: event.Revision, Value: event.Value})
			if err != nil {
				t.Fatal(err)
			}
			if err := sdk.DecodeVariableEvent(decoded, item.target); err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatalf("missing provider update for %s", item.name)
		}
	}
	client, err := sdk.NewAttachedClient(value.node)
	if err != nil {
		t.Fatal(err)
	}
	management, err := client.Management(value.node.ID())
	if err != nil {
		t.Fatal(err)
	}
	ownerText := strconv.FormatUint(uint64(owner), 10)
	if topology, err := management.Topology(context.Background()); err != nil || len(topology.Nodes) != 2 || topology.Nodes[0].NodeID != ownerText || topology.Nodes[1].ParentID != ownerText {
		t.Fatalf("domain SDK topology: %+v, %v", topology, err)
	}
	var admission protocol.AdmissionStatusV1
	decodeVariable(t, value.node, protocol.BuiltinAdmissionStatus, &admission)
	if admission.AuthorityNodeID != ownerText {
		t.Fatalf("admission lost owner precision: %+v", admission)
	}
	catalogState, err := client.Snapshot(context.Background(), protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog})
	if err != nil {
		t.Fatal(err)
	}
	var catalog protocol.ResourceCatalogV2
	decodeVariableDomain(t, catalogState, &catalog)
	if !bytes.Contains(catalogState.Value, []byte(`"owner":"`+ownerText+`"`)) {
		t.Fatalf("catalog owner was not a decimal string: %s", catalogState.Value)
	}
	for _, descriptor := range catalog.Resources {
		if descriptor.ID.Owner != owner {
			t.Fatalf("catalog owner lost precision: %+v", descriptor.ID)
		}
	}
	ordinaryJSON, err := protocol.EncodeJSONPayload(&catalog, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(ordinaryJSON, []byte(`"owner":`+ownerText+`,`)) {
		t.Fatal("variable codec changed ordinary JSON owner encoding")
	}
	// The carried definition, including an unfamiliar semantic ID, is the
	// decoding authority; the helper must not replace it from a local catalog.
	catalogState.Definition.ID = "test.unregistered.catalog"
	if err := sdk.DecodeVariableEvent(catalogState, &catalog); err != nil {
		t.Fatal(err)
	}
	for _, descriptor := range catalog.Resources {
		if descriptor.ID.Owner != owner {
			t.Fatal("decoded-state helper lost full-width owner")
		}
	}
	catalogState.Definition = nil
	if err := sdk.DecodeVariableEvent(catalogState, &catalog); err == nil {
		t.Fatal("domain helper accepted missing session definition")
	}
	if health, err := management.Health(context.Background()); err != nil || health.State != "running" {
		t.Fatalf("domain SDK health: %+v, %v", health, err)
	}
	if config, err := management.Config(context.Background()); err != nil || config.DisplayName != "standard" {
		t.Fatalf("domain SDK config: %+v, %v", config, err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	for _, item := range watched {
		select {
		case event := <-item.events:
			t.Fatalf("unchanged %s emitted an update: %+v", item.name, event)
		default:
		}
	}
}

func TestTopologyRemovalRetiresWrappedVariable(t *testing.T) {
	value := newFixture(t)
	id := protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinManagementTopology}
	registered, _ := value.node.Registry().Resolve(id)
	events := make(chan resource.Observation, 2)
	_, cancel, err := registered.(resource.Observable).Observe(func(event resource.Observation) { events <- event })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if err := value.node.Registry().Remove(id); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Failure == nil || event.Failure.Code != protocol.CodeGone {
			t.Fatalf("missing terminal removal: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("wrapped variable subscription survived removal")
	}
	if _, err := registered.Operate(context.Background(), resource.OperationRequest{Capability: protocol.CapabilityRead}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("retired variable read: %v", err)
	}
	if _, cancel, err := registered.(resource.Observable).Observe(func(resource.Observation) {}); err == nil {
		cancel()
		t.Fatal("retired variable accepted observer")
	}
	if err := value.node.Registry().Register(registered); err == nil {
		t.Fatal("retired wrapper was registered again")
	}
}

func TestManagementDomainEventRejectsNonstandardVariableFormats(t *testing.T) {
	value := newFixture(t)
	id := protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinManagementTopology}
	client, err := sdk.NewAttachedClient(value.node)
	if err != nil {
		t.Fatal(err)
	}
	state, err := client.Snapshot(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	revision, _ := protocol.ParseVariableRevision(state.Revision)
	payload := encode(t, &state)
	for _, test := range []struct {
		name, schema string
		payload      []byte
		revision     uint64
	}{
		{"domain payload", protocol.SchemaVariableValueV1, state.Value, revision},
		{"private schema", protocol.SchemaManagementTopologyV1, state.Value, revision},
		{"unknown version", protocol.SchemaVariableValueV1, bytes.Replace(payload, []byte(`"version":1`), []byte(`"version":99`), 1), revision},
		{"unknown schema", "mfh.variable-value.v99", payload, revision},
		{"revision mismatch", protocol.SchemaVariableValueV1, payload, revision + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var topology protocol.ManagementTopologyV1
			var decoder sdk.VariableEventDecoder
			state, err := decoder.Decode(sdk.Event{Kind: sdk.EventSnapshot, Resource: id, Schema: test.schema, Revision: test.revision, Value: test.payload})
			if err == nil {
				err = sdk.DecodeVariableEvent(state, &topology)
			}
			if err == nil {
				t.Fatal("nonstandard variable event accepted")
			}
		})
	}
}
