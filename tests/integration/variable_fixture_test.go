package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

func variableEventString(t *testing.T, event subscription.Event) string {
	t.Helper()
	var state protocol.VariableValueV1
	if event.Schema != protocol.SchemaVariableValueV1 {
		t.Fatalf("unexpected variable event schema: %q", event.Schema)
	}
	if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &state); err != nil {
		t.Fatal(err)
	}
	revision, err := protocol.ParseVariableRevision(state.Revision)
	if err != nil || revision != event.Revision || state.Status != "ready" {
		t.Fatalf("invalid variable event state: %+v (%v)", state, err)
	}
	if (event.Kind == subscription.EventSnapshot) != (state.Definition != nil) {
		t.Fatalf("definition must be present only on the snapshot: %+v", state)
	}
	var value string
	if err := json.Unmarshal(state.Value, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
