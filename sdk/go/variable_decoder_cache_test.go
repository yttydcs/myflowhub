package sdk

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestVariableDecoderCachedDataKeepsStrictWireValidation(t *testing.T) {
	definition := protocol.DataSchemaDefinition{Type: "boolean"}
	valid := variableEvent(t, EventData, testVariableEnvelope(nil, `true`, 2))
	raw := string(valid.Value)
	for name, invalid := range map[string]string{
		"duplicate":         strings.Replace(raw, `"revision":"2"`, `"revision":"2","revision":"2"`, 1),
		"escaped duplicate": strings.Replace(raw, `"revision":"2"`, `"revision":"2","revisio\u006e":"2"`, 1),
		"wrong case":        strings.Replace(raw, `"status"`, `"Status"`, 1),
		"missing":           strings.Replace(raw, `"version":1,`, ``, 1),
		"wrong value type":  strings.Replace(raw, `"value":true`, `"value":20`, 1),
		"null status":       strings.Replace(raw, `"status":"ready"`, `"status":null`, 1),
		"unknown field":     strings.Replace(raw, `"version":1`, `"version":1,"unknown":true`, 1),
		"trailing data":     raw + ` null`,
		"invalid encoding":  strings.Replace(raw, `"status":"ready"`, `"status":"\ud800"`, 1),
		"wire overflow":     raw + strings.Repeat(" ", protocol.DefaultMaxPayload),
	} {
		t.Run(name, func(t *testing.T) {
			if invalid == raw {
				t.Fatal("invalid fixture did not change the wire")
			}
			var decoder VariableEventDecoder
			if _, err := decoder.Decode(variableEvent(t, EventSnapshot, testVariableEnvelope(&definition, `false`, 1))); err != nil {
				t.Fatal(err)
			}
			event := valid
			event.Value = json.RawMessage(invalid)
			if _, err := decoder.Decode(event); err == nil {
				t.Fatal("cached definition bypassed strict wire validation")
			}
			if decoder.revision != 1 {
				t.Fatal("invalid wire advanced the session revision")
			}
			if _, err := decoder.Decode(valid); err != nil {
				t.Fatalf("valid event after rejection: %v", err)
			}
		})
	}
}

func TestVariableDecoderCacheCannotBeMutatedThroughReturnedDefinitions(t *testing.T) {
	maximum := 10.0
	definition := protocol.DataSchemaDefinition{Type: "object", Properties: []protocol.DataSchemaProperty{
		{Name: "level", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "integer", Maximum: &maximum}},
		{Name: "mode", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "string", Enum: []string{"a", "b"}}},
	}}
	var decoder VariableEventDecoder
	state, err := decoder.Decode(variableEvent(t, EventSnapshot, testVariableEnvelope(&definition, `{"level":1,"mode":"a"}`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	state.Definition.Properties[0].Name = "tampered"
	*state.Definition.Properties[0].Schema.Maximum = 0
	state.Definition.Properties[1].Schema.Enum[0] = "tampered"
	state, err = decoder.Decode(variableEvent(t, EventData, testVariableEnvelope(nil, `{"level":5,"mode":"a"}`, 2)))
	if err != nil {
		t.Fatalf("snapshot caller changed trusted cache: %v", err)
	}
	*state.Definition.Properties[0].Schema.Maximum = 0
	state.Definition.Properties[1].Schema.Enum[1] = "tampered"
	state, err = decoder.Decode(variableEvent(t, EventData, testVariableEnvelope(nil, `{"level":6,"mode":"b"}`, 3)))
	if err != nil {
		t.Fatalf("data caller changed trusted cache: %v", err)
	}
	*state.Definition.Properties[0].Schema.Maximum = 1000
	if _, err := decoder.Decode(variableEvent(t, EventData, testVariableEnvelope(nil, `{"level":11,"mode":"b"}`, 4))); err == nil {
		t.Fatal("caller mutation bypassed cached numeric validation")
	}
	if decoder.revision != 3 {
		t.Fatal("rejected data changed decoder revision")
	}
}
