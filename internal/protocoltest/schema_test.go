package protocoltest

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func fixtures(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate protocol test source")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "tests", "fixtures", "protocol")
}

func TestCrossLanguageGoldenPayloads(t *testing.T) {
	tests := []struct {
		name  string
		value protocol.ValidatedPayload
		fresh func() protocol.ValidatedPayload
	}{
		{
			name:  "provisioning-permit-v1.json",
			value: &protocol.ProvisioningPermitV1{Version: 1, PermitID: "00112233445566778899aabbccddeeff", ParentNodeID: "1", ChildNodeID: "2", ChildPublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", Role: "device", IssuedAtUnixMS: 1000, ExpiresAtUnixMS: 2000, MaxUses: 1, Signature: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
			fresh: func() protocol.ValidatedPayload { return &protocol.ProvisioningPermitV1{} },
		},
		{
			name:  "resource-catalog-v1.json",
			value: &protocol.ResourceCatalogV1{Version: 1, Revision: 3, Resources: []protocol.ResourceDescriptorV1{{Name: "metrics/cpu", Kind: protocol.ResourceKindVariable, ContentType: "application/json", Schema: "mfh.metrics.cpu.v1", Permission: "metrics.read", MaxValueBytes: 4096}, {Name: protocol.BuiltinResourceCatalog, Kind: protocol.ResourceKindVariable, ContentType: "application/json", Schema: protocol.SchemaResourceCatalogV1, Permission: "resource.catalog.read", MaxValueBytes: protocol.DefaultMaxPayload}}},
			fresh: func() protocol.ValidatedPayload { return &protocol.ResourceCatalogV1{} },
		},
		{
			name:  "management-health-v1.json",
			value: &protocol.ManagementHealthV1{Version: 1, State: "running", StartedAtUnixMS: 1000, TopologyEpoch: 7, ActiveLinks: 2, ActiveSubscriptions: 3, Components: map[string]string{"identity": "ready", "tcp": "ready"}},
			fresh: func() protocol.ValidatedPayload { return &protocol.ManagementHealthV1{} },
		},
		{
			name:  "notification-event-v1.json",
			value: &protocol.NotificationEventV1{Version: 1, EventID: "00112233445566778899aabbccddeeff", Channel: "system", SourceNodeID: "2", CreatedAtUnixMS: 1000, ContentType: "text/plain", Body: []byte("hello"), Attributes: map[string]string{"severity": "info"}},
			fresh: func() protocol.ValidatedPayload { return &protocol.NotificationEventV1{} },
		},
		{
			name:  "file-chunk-v1.json",
			value: &protocol.FileChunkV1{Version: 1, TransferID: "00112233445566778899aabbccddeeff", Offset: 0, Data: []byte("abc"), SHA256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
			fresh: func() protocol.ValidatedPayload { return &protocol.FileChunkV1{} },
		},
		{
			name:  "flow-definition-v1.json",
			value: &protocol.FlowDefinitionV1{Version: 1, FlowID: "00112233445566778899aabbccddeeff", Revision: 1, Name: "sample", Nodes: []protocol.FlowNodeV1{{ID: "read", Kind: "variable-read", Resource: protocol.FlowResourceRefV1{OwnerNodeID: "2", Name: "metrics/cpu"}}, {ID: "notify", Kind: "command-call", Resource: protocol.FlowResourceRefV1{OwnerNodeID: "1", Name: "notifications/publish"}, Config: json.RawMessage(`{"channel":"system"}`)}}, Edges: []protocol.FlowEdgeV1{{From: "read", To: "notify"}}, Inputs: map[string]string{"threshold": "80"}},
			fresh: func() protocol.ValidatedPayload { return &protocol.FlowDefinitionV1{} },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			golden, err := os.ReadFile(filepath.Join(fixtures(t), test.name))
			if err != nil {
				t.Fatal(err)
			}
			golden = bytes.TrimSpace(golden)
			encoded, err := protocol.EncodeJSONPayload(test.value, protocol.DefaultMaxPayload)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, golden) {
				t.Fatalf("golden drift\nwant: %s\n got: %s", golden, encoded)
			}
			decoded := test.fresh()
			if err := protocol.DecodeJSONPayload(golden, protocol.DefaultMaxPayload, decoded); err != nil {
				t.Fatal(err)
			}
			roundTrip, err := protocol.EncodeJSONPayload(decoded, protocol.DefaultMaxPayload)
			if err != nil || !bytes.Equal(roundTrip, golden) {
				t.Fatalf("round-trip drift: %v\n%s", err, roundTrip)
			}
		})
	}
}

func TestSchemaValidationRejectsUnsafeInputs(t *testing.T) {
	badCatalog := &protocol.ResourceCatalogV1{Version: 2, Revision: 1}
	if err := badCatalog.Validate(); err == nil {
		t.Fatal("unknown schema version accepted")
	}
	badChunk := &protocol.FileChunkV1{Version: 1, TransferID: "00112233445566778899aabbccddeeff", Data: make([]byte, protocol.MaxFileChunkBytes+1), SHA256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"}
	if err := badChunk.Validate(); err == nil {
		t.Fatal("oversize chunk accepted")
	}
	badOffer := &protocol.FileOfferV1{Version: 1, TransferID: "00112233445566778899aabbccddeeff", Path: "../escape", SHA256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", ChunkSize: 1, ExpiresAtUnixMS: 1}
	if err := badOffer.Validate(); err == nil {
		t.Fatal("path escape accepted")
	}
	cycle := &protocol.FlowDefinitionV1{Version: 1, FlowID: "00112233445566778899aabbccddeeff", Revision: 1, Name: "cycle", Nodes: []protocol.FlowNodeV1{{ID: "a", Kind: "transform"}, {ID: "b", Kind: "transform"}}, Edges: []protocol.FlowEdgeV1{{From: "a", To: "b"}, {From: "b", To: "a"}}}
	if err := cycle.Validate(); err == nil {
		t.Fatal("flow cycle accepted")
	}
	badPolicy := &protocol.ManagementPolicyRuleV1{Version: 1, Subject: "2", Action: "admin", ResourceNode: "1", ResourceName: "system/health"}
	if err := badPolicy.Validate(); err == nil {
		t.Fatal("unknown policy action accepted")
	}
	var target protocol.NotificationPublishV1
	err := protocol.DecodeJSONPayload(make([]byte, 10), 5, &target)
	if !errors.Is(err, protocol.ErrPayloadTooLarge) {
		t.Fatalf("expected payload size error, got %v", err)
	}
}

func FuzzCatalogSchema(f *testing.F) {
	f.Add([]byte(`{"version":1,"revision":1,"resources":[]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > protocol.DefaultMaxPayload {
			t.Skip()
		}
		var value protocol.ResourceCatalogV1
		_ = protocol.DecodeJSONPayload(data, protocol.DefaultMaxPayload, &value)
	})
}

func FuzzFileChunkSchema(f *testing.F) {
	f.Add([]byte(`{"version":1}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > protocol.DefaultMaxPayload {
			t.Skip()
		}
		var value protocol.FileChunkV1
		_ = protocol.DecodeJSONPayload(data, protocol.DefaultMaxPayload, &value)
	})
}
