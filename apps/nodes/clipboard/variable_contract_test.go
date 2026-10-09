package clipboard

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestClipboardVariablesReadObserveAndValidate(t *testing.T) {
	controller, runtime, _ := newClipboardController(t, 2, nil)
	defer runtime.Close()
	defer controller.Close()
	for _, name := range []string{ResourceConfig, ResourceStatus} {
		t.Run(name, func(t *testing.T) {
			registered, _ := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: name})
			variable := registered.(*resource.Variable)
			descriptor := variable.Descriptor()
			read, _ := descriptor.Capability(protocol.CapabilityRead)
			subscribe, _ := descriptor.Capability(protocol.CapabilitySubscribe)
			if descriptor.TypeVersion != 2 || read.OutputSchema != protocol.SchemaVariableValueV1 || subscribe.EventSchema != protocol.SchemaVariableValueV1 {
				t.Fatalf("nonstandard descriptor: %+v", descriptor)
			}
			if _, ok := descriptor.Capability(protocol.CapabilityWrite); ok {
				t.Fatal("config/status variable unexpectedly writable")
			}
			events := make(chan resource.Observation, 8)
			initial, cancel, err := variable.Observe(func(event resource.Observation) { events <- event })
			if err != nil {
				t.Fatal(err)
			}
			defer cancel()
			var state protocol.VariableValueV1
			if err := protocol.DecodeJSONPayload(initial.Value, protocol.DefaultMaxPayload, &state); err != nil {
				t.Fatal(err)
			}
			if !initial.Snapshot || state.Definition == nil || state.Status != "ready" {
				t.Fatal("missing complete standard snapshot")
			}
			result, err := runtime.Registry().Operate(context.Background(), descriptor.ID, resource.OperationRequest{Capability: protocol.CapabilityRead})
			if err != nil {
				t.Fatal(err)
			}
			var readState protocol.VariableValueV1
			if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &readState); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(state, readState) {
				t.Fatal("read and Observe initial state disagree")
			}
			if name == ResourceConfig {
				var config ConfigV1
				if err := protocol.DecodeJSONPayload(state.Value, protocol.DefaultMaxPayload, &config); err != nil {
					t.Fatal(err)
				}
				if config.Enabled || config.AutoWatch || config.AutoApply || !bytes.Contains(state.Value, []byte(`"peers":[]`)) {
					t.Fatalf("false/empty values lost: %s", state.Value)
				}
				enableClipboard(t, controller, false, false, nil)
			} else {
				controller.RecordError(context.DeadlineExceeded)
			}
			var event resource.Observation
			select {
			case event = <-events:
			case <-time.After(time.Second):
				t.Fatal("missing Clipboard variable update")
			}
			var update protocol.VariableValueV1
			if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &update); err != nil {
				t.Fatal(err)
			}
			if event.Snapshot || update.Definition != nil || update.InstanceID != state.InstanceID || update.Revision != strconv.FormatUint(event.Revision, 10) || event.Revision <= initial.Revision {
				t.Fatalf("invalid update: %+v", update)
			}
			if err := protocol.ValidateVariableValue(*state.Definition, update.Value); err != nil {
				t.Fatal(err)
			}
			var target protocol.ValidatedPayload = &ConfigV1{}
			if name == ResourceStatus {
				target = &StatusV1{}
			}
			if err := protocol.DecodeJSONPayload(update.Value, protocol.DefaultMaxPayload, target); err != nil {
				t.Fatal(err)
			}
			before := variable.Snapshot()
			for _, invalid := range []resource.VariableState{
				{Status: "ready", Value: json.RawMessage(`{"version":1}`)},
				{Status: "ready", Value: json.RawMessage(`false`)},
				{Status: "unknown", Value: before.State.Value},
				{Status: "ready", Value: json.RawMessage(`{"unknown":"` + strings.Repeat("x", protocol.DefaultMaxPayload) + `"}`)},
			} {
				if _, err := variable.Set(invalid); err == nil {
					t.Fatal("invalid state accepted")
				}
				after := variable.Snapshot()
				if after.Revision != before.Revision || !bytes.Equal(after.State.Value, before.State.Value) {
					t.Fatal("rejected update changed variable")
				}
				select {
				case event := <-events:
					t.Fatalf("rejected update emitted event: %+v", event)
				default:
				}
			}
		})
	}
}

func TestClipboardVariableDefinitionsMatchDomainConstraints(t *testing.T) {
	for _, definition := range []protocol.DataSchemaDefinition{configDefinition(), statusDefinition()} {
		if err := protocol.ValidateVariableDefinition(definition); err != nil {
			t.Fatal(err)
		}
	}
	config := DefaultConfig()
	data, err := protocol.EncodeJSONPayload(config, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range [][]byte{
		bytes.Replace(data, []byte(`"enabled":false`), []byte(`"enabled":"false"`), 1),
		bytes.Replace(data, []byte(`"peers":[]`), []byte(`"peers":null`), 1),
		bytes.Replace(data, []byte(`"history_retention":"body"`), []byte(`"history_retention":"unknown"`), 1),
		bytes.Replace(data, []byte(`"revision":1`), []byte(`"revision":9007199254740992`), 1),
	} {
		if err := protocol.ValidateVariableValue(configDefinition(), invalid); err == nil {
			t.Fatalf("invalid config accepted: %s", invalid)
		}
	}
}

func TestClipboardConfigPlatformValidationPrecedesPersistence(t *testing.T) {
	controller, runtime, _ := newClipboardController(t, 2, nil)
	defer runtime.Close()
	defer controller.Close()
	controller.config.Revision = 1<<53 - 1
	if err := controller.store.Save("clipboard.json", controller.config); err != nil {
		t.Fatal(err)
	}
	data, err := protocol.EncodeJSONPayload(controller.config, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.configVariable.Set(resource.VariableState{Status: "ready", Value: data}); err != nil {
		t.Fatal(err)
	}
	before := controller.configVariable.Snapshot()
	events := 0
	_, cancel, err := controller.configVariable.Observe(func(resource.Observation) { events++ })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	config := controller.Config()
	_, err = controller.ApplyConfig(context.Background(), ConfigUpdateV1{
		Version: 1, ExpectedRevision: config.Revision, Enabled: true, MaxInlineBytes: config.MaxInlineBytes,
		HistoryRetention: config.HistoryRetention, HistoryLimit: config.HistoryLimit,
		HistoryMaxBytes: config.HistoryMaxBytes, HistoryTTLMS: config.HistoryTTLMS, Peers: config.Peers,
	})
	if err == nil || !strings.Contains(err.Error(), "validate clipboard config variable") {
		t.Fatalf("unsafe integer update reached persistence: %v", err)
	}
	var stored ConfigV1
	if found, err := controller.store.Load("clipboard.json", &stored); err != nil || !found {
		t.Fatalf("read persisted config: %v", err)
	}
	after := controller.configVariable.Snapshot()
	if stored.Revision != config.Revision || stored.Enabled || controller.Config().Enabled || after.Revision != before.Revision || !bytes.Equal(after.State.Value, before.State.Value) || events != 0 {
		t.Fatal("platform rejection changed persistent, domain, or variable state")
	}
}
