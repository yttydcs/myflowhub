package protocol

import (
	"strings"
	"testing"
)

func TestCatalogListRejectsInvalidExternalLimitsAndCursor(t *testing.T) {
	for _, value := range []string{`{"version":2}`, `{"version":1,"limit":-1}`, `{"version":1,"limit":257}`, `{"version":1,"unexpected":true}`, `{"version":1} {"version":1}`} {
		var request CatalogListV1
		if err := DecodeJSONPayload([]byte(value), 4096, &request); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	if err := (CatalogListV1{Version: 1, Cursor: strings.Repeat("x", MaxCatalogCursorBytes+1)}).Validate(); err == nil {
		t.Fatal("unbounded cursor")
	}
	for _, limit := range []int{0, 1, 128, 256} {
		if err := (CatalogListV1{Version: 1, Limit: limit}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCatalogPageRequiresConsistentOwnerAndContinuation(t *testing.T) {
	descriptor := ResourceDescriptorV2{ID: ResourceID{Owner: 1, Name: BuiltinResourceCatalog}, Type: ResourceTypeVariable, TypeVersion: 2, Capabilities: []CapabilityDescriptorV2{{Name: CapabilityRead, Permission: "read", MaxPayloadBytes: 1024}}, Limits: ResourceLimitsV2{MaxPayloadBytes: 1024}}
	page := CatalogPageV1{Version: 1, Owner: 1, InstanceID: strings.Repeat("a", 32), Revision: "18446744073709551615", Total: 1, Resources: []ResourceDescriptorV2{descriptor}}
	if err := page.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*CatalogPageV1){
		func(p *CatalogPageV1) { p.Revision = "01" },
		func(p *CatalogPageV1) { p.Owner = 2 },
		func(p *CatalogPageV1) { p.Total = 2 },
		func(p *CatalogPageV1) { p.NextCursor = "unexpected" },
		func(p *CatalogPageV1) { p.Offset = -1 },
		func(p *CatalogPageV1) { p.Resources = nil },
	} {
		bad := page
		mutate(&bad)
		if err := bad.Validate(); err == nil {
			t.Fatalf("accepted invalid page %+v", bad)
		}
	}
}
