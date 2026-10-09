package file

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestSummaryEnvelopeOverflowPreservesRevisionAndEvents(t *testing.T) {
	definition, err := protocol.DataSchemaByID(protocol.SchemaFileTransfersV1)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := protocol.EncodeVariablePayload(definition, &protocol.FileTransfersV1{Version: 1, Revision: 1, Transfers: []protocol.FileTransferSummaryV1{}}, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	id := protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFileTransfers}
	probe, err := resource.NewVariable(resource.VariableDescriptor(id, "file.read", protocol.DefaultMaxPayload), definition, resource.VariableState{Status: "ready", Value: initial}, nil)
	if err != nil {
		t.Fatal(err)
	}
	state := probe.Snapshot().State
	envelope, err := protocol.EncodeJSONPayload(&state, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	variable, err := resource.NewVariable(resource.VariableDescriptor(id, "file.read", len(envelope)+32), definition, resource.VariableState{Status: "ready", Value: initial}, nil)
	if err != nil {
		t.Fatal(err)
	}
	controller := &Controller{summaries: variable, revision: 1, transfers: map[string]*transfer{
		"00112233445566778899aabbccddeeff": {owner: 1, state: "offered", offer: protocol.FileOfferV1{Path: "inbox.txt", Size: 1, ExpiresAtUnixMS: time.Now().Add(time.Minute).UnixMilli()}},
	}}
	events := 0
	_, cancel, err := variable.Observe(func(resource.Observation) { events++ })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	before := variable.Snapshot()
	if err := controller.updateSummariesLocked(); !errors.Is(err, resource.ErrValueTooLarge) {
		t.Fatalf("expected complete-envelope overflow, got %v", err)
	}
	after := variable.Snapshot()
	if controller.revision != 1 || after.Revision != before.Revision || !bytes.Equal(after.State.Value, before.State.Value) || events != 0 {
		t.Fatal("failed publication changed revision/value or emitted an update")
	}
	controller.transfers = map[string]*transfer{}
	if err := controller.updateSummariesLocked(); err != nil {
		t.Fatal(err)
	}
	if controller.revision != 2 || variable.Snapshot().Revision != 2 || events != 1 {
		t.Fatal("valid publication did not recover with one revision and event")
	}
}
