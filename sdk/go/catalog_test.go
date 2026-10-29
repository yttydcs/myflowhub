package sdk

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestClientAggregatesLargeCatalogWithoutLegacyTruncation(t *testing.T) {
	runtime := newLocalNode(t, 91)
	defer runtime.Close()
	values := make([]resource.Resource, 2000)
	for i := range values {
		descriptor := resource.StreamDescriptor(protocol.ResourceID{Owner: 91, Name: fmt.Sprintf("load/%04d", i)}, "application/octet-stream", "test.bytes.v1", "test.read", 1024)
		descriptor.Presentation.Description = strings.Repeat("x", 1000)
		value, err := resource.NewStream(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		values[i] = value
	}
	if err := runtime.Registry().RegisterBatch(values); err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := client.Catalog(context.Background(), 91)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Resources) != 2001 || catalog.Resources[0].ID.Name != "load/0000" || catalog.Resources[2000].ID.Name != protocol.BuiltinResourceCatalog {
		t.Fatalf("catalog missing resources: %d", len(catalog.Resources))
	}
	if err := catalog.Validate(); err != nil {
		t.Fatal(err)
	}
}
