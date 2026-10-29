package sdk

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func BenchmarkVariableEventDecoderData(b *testing.B) {
	definition := protocol.DataSchemaDefinition{Type: "object", Properties: []protocol.DataSchemaProperty{
		{Name: "enabled", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "boolean"}},
		{Name: "level", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "integer"}},
	}}
	state := protocol.VariableValueV1{Version: 1, InstanceID: "0123456789abcdef0123456789abcdef", Revision: "1", Definition: &definition,
		Value: json.RawMessage(`{"enabled":true,"level":40}`), Status: "ready", ObservedAtUnixMS: 1}
	raw, err := json.Marshal(state)
	if err != nil {
		b.Fatal(err)
	}
	event := Event{Kind: EventSnapshot, Resource: protocol.ResourceID{Owner: 2, Name: "test/value"}, Schema: protocol.SchemaVariableValueV1, Revision: 1, Value: raw}
	var decoder VariableEventDecoder
	if _, err := decoder.Decode(event); err != nil {
		b.Fatal(err)
	}
	state.Definition = nil
	event.Kind = EventData
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.Revision = uint64(i) + 2
		state.Revision = strconv.FormatUint(event.Revision, 10)
		event.Value, err = json.Marshal(state)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := decoder.Decode(event); err != nil {
			b.Fatal(err)
		}
	}
}
