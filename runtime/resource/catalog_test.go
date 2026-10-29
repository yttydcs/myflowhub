package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func catalogPage(t *testing.T, registry *Registry, cursor string, limit int) protocol.CatalogPageV1 {
	t.Helper()
	request, err := protocol.EncodeJSONPayload(&protocol.CatalogListV1{Version: 1, Cursor: cursor, Limit: limit}, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	result, err := registry.Operate(context.Background(), protocol.ResourceID{Owner: registry.owner, Name: protocol.BuiltinResourceCatalog}, OperationRequest{Capability: protocol.CapabilityList, Payload: request})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Payload) > protocol.MaxCatalogPageBytes {
		t.Fatalf("page exceeded budget: %d", len(result.Payload))
	}
	var page protocol.CatalogPageV1
	if err := protocol.DecodeJSONPayload(result.Payload, protocol.MaxCatalogPageBytes, &page); err != nil {
		t.Fatal(err)
	}
	return page
}

func TestCatalogPagesLargeBatchWithoutTruncatingLegacySnapshot(t *testing.T) {
	registry, err := NewRegistry(1)
	if err != nil {
		t.Fatal(err)
	}
	values := make([]Resource, 2000)
	for i := range values {
		descriptor := streamDescriptor(1, fmt.Sprintf("load/%04d", i), 1024)
		descriptor.Presentation.Description = strings.Repeat("x", 1000)
		values[i], err = NewStream(descriptor)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := registry.RegisterBatch(values); err != nil {
		t.Fatal(err)
	}
	legacy := registry.Catalog().Snapshot()
	if legacy.State.Status != "unavailable" || len(legacy.State.Value) != 0 || legacy.State.Error == nil || legacy.State.Error.Code != "overflow" {
		t.Fatalf("legacy catalog falsely appears complete: %+v", legacy.State)
	}
	var names []string
	var cursor, instance, revision string
	for {
		page := catalogPage(t, registry, cursor, 128)
		if page.Offset != len(names) || page.Total != 2001 {
			t.Fatalf("invalid position: %+v", page)
		}
		if instance != "" && (page.InstanceID != instance || page.Revision != revision) {
			t.Fatal("mixed generations")
		}
		instance, revision = page.InstanceID, page.Revision
		for _, descriptor := range page.Resources {
			names = append(names, descriptor.ID.Name)
		}
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	if len(names) != 2001 || names[0] != "load/0000" || names[1999] != "load/1999" || names[2000] != protocol.BuiltinResourceCatalog {
		t.Fatalf("incomplete catalog: %d", len(names))
	}
}

func TestCatalogBatchFailureIsAtomicAndCursorRejectsNewGeneration(t *testing.T) {
	registry, _ := NewRegistry(1)
	one, _ := NewStream(streamDescriptor(1, "one", 32))
	two, _ := NewStream(streamDescriptor(1, "two", 32))
	if err := registry.RegisterBatch([]Resource{one, one}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate batch: %v", err)
	}
	if len(registry.List()) != 1 || registry.Catalog().Snapshot().Revision != 1 {
		t.Fatal("failed batch changed registry")
	}
	if err := registry.RegisterBatch([]Resource{one, two}); err != nil {
		t.Fatal(err)
	}
	page := catalogPage(t, registry, "", 1)
	if page.NextCursor == "" {
		t.Fatal("missing continuation")
	}
	if _, err := one.Publish([]byte("sample")); err != nil {
		t.Fatal(err)
	}
	_ = catalogPage(t, registry, page.NextCursor, 1)
	if err := registry.Remove(two.Descriptor().ID); err != nil {
		t.Fatal(err)
	}
	request, _ := protocol.EncodeJSONPayload(&protocol.CatalogListV1{Version: 1, Cursor: page.NextCursor, Limit: 1}, 4096)
	_, err := registry.Operate(context.Background(), registry.Catalog().Descriptor().ID, OperationRequest{Capability: protocol.CapabilityList, Payload: request})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale cursor: %v", err)
	}
	if err := registry.RegisterBatch([]Resource{two, nil}); err == nil {
		t.Fatal("nil member accepted")
	}
	if len(registry.List()) != 2 {
		t.Fatal("failed batch partially committed")
	}
	var missing *Stream
	if err := registry.RegisterBatch([]Resource{missing}); err == nil {
		t.Fatal("typed nil resource accepted")
	}
}

func TestCatalogPagesRespectEncodedByteBudgetAndOwner(t *testing.T) {
	registry, _ := NewRegistry(1)
	values := make([]Resource, 128)
	for i := range values {
		d := streamDescriptor(1, fmt.Sprintf("large/%03d", i), 1024)
		for j := 1; j < protocol.MaxCapabilities; j++ {
			d.Capabilities = append(d.Capabilities, protocol.CapabilityDescriptorV2{Name: protocol.CapabilityID(fmt.Sprintf("custom%d", j)), Permission: strings.Repeat("p", 128), MaxPayloadBytes: 1024})
		}
		value, err := NewStream(d)
		if err != nil {
			t.Fatal(err)
		}
		values[i] = value
	}
	if err := registry.RegisterBatch(values); err != nil {
		t.Fatal(err)
	}
	page := catalogPage(t, registry, "", 256)
	if len(page.Resources) >= 128 || page.NextCursor == "" {
		t.Fatal("item limit ignored the encoded byte budget")
	}
	other, _ := NewRegistry(2)
	request, _ := protocol.EncodeJSONPayload(&protocol.CatalogListV1{Version: 1, Cursor: page.NextCursor}, 4096)
	_, err := other.Operate(context.Background(), other.Catalog().Descriptor().ID, OperationRequest{Capability: protocol.CapabilityList, Payload: request})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("foreign cursor accepted: %v", err)
	}
	request, _ = protocol.EncodeJSONPayload(&protocol.CatalogListV1{Version: 1, Cursor: "not-a-json-cursor"}, 4096)
	_, err = registry.Operate(context.Background(), registry.Catalog().Descriptor().ID, OperationRequest{Capability: protocol.CapabilityList, Payload: request})
	if !errors.Is(err, protocol.ErrInvalidPayload) {
		t.Fatalf("malformed cursor: %v", err)
	}
}
