package file_test

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	filefeature "github.com/yttydcs/myflowhub/feature/file"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

func TestTransferSummariesReadAndObserveStandardContract(t *testing.T) {
	for _, owner := range []protocol.NodeID{1, 9007199254740993, ^protocol.NodeID(0)} {
		t.Run(strconv.FormatUint(uint64(owner), 10), func(t *testing.T) { testTransferVariableContract(t, owner) })
	}
}

func testTransferVariableContract(t *testing.T, owner protocol.NodeID) {
	t.Helper()
	runtime, root := localNodeWithID(t, owner)
	controller, err := filefeature.Register(filefeature.Config{Node: runtime, Root: root, MaxFileBytes: 1024, MaxTotalBytes: 2048})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	id := protocol.ResourceID{Owner: runtime.ID(), Name: protocol.BuiltinFileTransfers}
	registered, _ := runtime.Registry().Resolve(id)
	descriptor := registered.Descriptor()
	read, _ := descriptor.Capability(protocol.CapabilityRead)
	subscribe, _ := descriptor.Capability(protocol.CapabilitySubscribe)
	if descriptor.TypeVersion != 2 || read.OutputSchema != protocol.SchemaVariableValueV1 || subscribe.EventSchema != protocol.SchemaVariableValueV1 || read.Permission != "file.read" {
		t.Fatalf("invalid transfers descriptor: %+v", descriptor)
	}
	events := make(chan resource.Observation, 8)
	initial, cancel, err := registered.(resource.Observable).Observe(func(event resource.Observation) { events <- event })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	var state protocol.VariableValueV1
	if err := protocol.DecodeJSONPayload(initial.Value, protocol.DefaultMaxPayload, &state); err != nil {
		t.Fatal(err)
	}
	if !initial.Snapshot || state.Definition == nil || state.Definition.ID != protocol.SchemaFileTransfersV1 || state.Status != "ready" {
		t.Fatal("incomplete transfer snapshot")
	}
	result, err := runtime.Registry().Operate(context.Background(), id, resource.OperationRequest{Capability: protocol.CapabilityRead})
	if err != nil || result.Schema != protocol.SchemaVariableValueV1 {
		t.Fatalf("read/Observe disagree: %v", err)
	}
	var readState protocol.VariableValueV1
	if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &readState); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, readState) {
		t.Fatal("read/Observe state mismatch")
	}
	var decoder sdk.VariableEventDecoder
	if _, err := decoder.Decode(sdk.Event{Kind: sdk.EventSnapshot, Resource: id, Schema: initial.Schema, Revision: initial.Revision, Value: initial.Value}); err != nil {
		t.Fatal(err)
	}
	var transfers protocol.FileTransfersV1
	if err := protocol.DecodeVariablePayload(*state.Definition, state.Value, protocol.DefaultMaxPayload, &transfers); err != nil {
		t.Fatal(err)
	}
	if len(transfers.Transfers) != 0 {
		t.Fatal("initial transfers not empty")
	}
	content := []byte("new contract")
	offer := protocol.FileOfferV1{Version: 1, TransferID: transferOne, Path: "summary.txt", Size: int64(len(content)), SHA256: sum(content), ChunkSize: len(content), ExpiresAtUnixMS: time.Now().Add(time.Minute).UnixMilli()}
	session := openSessionAtOwner(t, runtime, owner, offer)
	defer session.Close(context.Background(), false, "", nil)
	select {
	case event := <-events:
		var update protocol.VariableValueV1
		if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &update); err != nil {
			t.Fatal(err)
		}
		if update.Definition != nil || update.InstanceID != state.InstanceID || update.Revision != strconv.FormatUint(event.Revision, 10) || event.Revision <= initial.Revision {
			t.Fatalf("invalid summary event: %+v", update)
		}
		if err := protocol.ValidateVariableValue(*state.Definition, update.Value); err != nil {
			t.Fatal(err)
		}
		decoded, err := decoder.Decode(sdk.Event{Kind: sdk.EventData, Resource: id, Schema: event.Schema, Revision: event.Revision, Value: event.Value})
		if err != nil {
			t.Fatal(err)
		}
		if err := sdk.DecodeVariableEvent(decoded, &transfers); err != nil {
			t.Fatal(err)
		}
		if len(transfers.Transfers) != 1 || transfers.Transfers[0].TransferID != transferOne || transfers.Transfers[0].State != "offered" || transfers.Transfers[0].OwnerNodeID != strconv.FormatUint(uint64(owner), 10) {
			t.Fatalf("wrong summary domain object: %+v", transfers)
		}
	case <-time.After(time.Second):
		t.Fatal("summary update missing")
	}
	client, err := sdk.NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	files, err := client.Files(runtime.ID())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := files.Transfers(context.Background())
	if err != nil || len(decoded.Transfers) != 1 || decoded.Transfers[0].OwnerNodeID != strconv.FormatUint(uint64(owner), 10) {
		t.Fatalf("domain SDK did not unwrap standard snapshot: %+v, %v", decoded, err)
	}
}
