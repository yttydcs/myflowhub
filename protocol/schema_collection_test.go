package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func validCollectionMember(key string) CollectionMemberV1 {
	return CollectionMemberV1{
		Key: key, Kind: "file", Label: "Member " + key, ContentType: "text/plain", Schema: "example.text.v1",
		Capabilities: []CapabilityID{CapabilityGet, CapabilityRead},
		Attributes:   map[string]string{"size": "4"},
	}
}

func TestCollectionPayloadsValidateAndEncodeCanonicalFields(t *testing.T) {
	request := CollectionListRequestV1{Version: 1, Parent: "folder", Cursor: "opaque", Limit: 25}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	memberRequest := CollectionMemberRequestV1{Version: 1, Key: "folder/item"}
	if err := memberRequest.Validate(); err != nil {
		t.Fatal(err)
	}
	page := CollectionPageV1{
		Version: 1, Revision: 2, Parent: "folder",
		Members:    []CollectionMemberV1{validCollectionMember("a"), validCollectionMember("b")},
		NextCursor: "next",
	}
	if err := page.Validate(); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(payload)
	for _, field := range []string{`"version":1`, `"revision":2`, `"parent":"folder"`, `"members":`, `"next_cursor":"next"`, `"content_type":"text/plain"`} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("collection JSON omitted canonical field %s: %s", field, encoded)
		}
	}
	if strings.Contains(encoded, `"owner"`) || strings.Contains(encoded, `"resource_id"`) {
		t.Fatalf("collection member leaked global identity: %s", encoded)
	}
}

func TestCollectionListAndMemberRequestsRejectInvalidBounds(t *testing.T) {
	tests := []struct {
		name  string
		value ValidatedPayload
	}{
		{"list-version", &CollectionListRequestV1{Version: 2, Limit: 1}},
		{"list-zero-limit", &CollectionListRequestV1{Version: 1}},
		{"list-large-limit", &CollectionListRequestV1{Version: 1, Limit: MaxCollectionPageMembers + 1}},
		{"list-parent", &CollectionListRequestV1{Version: 1, Parent: strings.Repeat("p", MaxCollectionParentBytes+1), Limit: 1}},
		{"list-cursor", &CollectionListRequestV1{Version: 1, Cursor: strings.Repeat("c", MaxCollectionCursorBytes+1), Limit: 1}},
		{"member-version", &CollectionMemberRequestV1{Version: 2, Key: "a"}},
		{"member-empty", &CollectionMemberRequestV1{Version: 1}},
		{"member-key", &CollectionMemberRequestV1{Version: 1, Key: strings.Repeat("k", MaxCollectionMemberKeyBytes+1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.value.Validate(); err == nil {
				t.Fatalf("invalid collection payload was accepted: %#v", test.value)
			}
		})
	}
}

func TestCollectionMemberRejectsInvalidFieldsCapabilitiesAndAttributes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CollectionMemberV1)
	}{
		{"key-empty", func(member *CollectionMemberV1) { member.Key = "" }},
		{"key-large", func(member *CollectionMemberV1) { member.Key = strings.Repeat("k", MaxCollectionMemberKeyBytes+1) }},
		{"kind-empty", func(member *CollectionMemberV1) { member.Kind = "" }},
		{"kind-large", func(member *CollectionMemberV1) { member.Kind = strings.Repeat("k", MaxCollectionMemberKindBytes+1) }},
		{"label-empty", func(member *CollectionMemberV1) { member.Label = "" }},
		{"label-large", func(member *CollectionMemberV1) { member.Label = strings.Repeat("l", MaxLabelBytes+1) }},
		{"content-type", func(member *CollectionMemberV1) { member.ContentType = strings.Repeat("c", MaxContentTypeBytes+1) }},
		{"schema", func(member *CollectionMemberV1) { member.Schema = strings.Repeat("s", MaxSchemaBytes+1) }},
		{"capability-unsorted", func(member *CollectionMemberV1) { member.Capabilities = []CapabilityID{CapabilityRead, CapabilityGet} }},
		{"capability-duplicate", func(member *CollectionMemberV1) { member.Capabilities = []CapabilityID{CapabilityGet, CapabilityGet} }},
		{"capability-invalid", func(member *CollectionMemberV1) { member.Capabilities = []CapabilityID{"_invalid"} }},
		{"capability-count", func(member *CollectionMemberV1) {
			member.Capabilities = make([]CapabilityID, MaxCapabilities+1)
		}},
		{"attributes-count", func(member *CollectionMemberV1) {
			member.Attributes = make(map[string]string, MaxAttributes+1)
			for index := range MaxAttributes + 1 {
				member.Attributes[string(rune('a'+index))] = "value"
			}
		}},
		{"attribute-key", func(member *CollectionMemberV1) {
			member.Attributes = map[string]string{strings.Repeat("k", MaxAttributeKeyBytes+1): "value"}
		}},
		{"attribute-value", func(member *CollectionMemberV1) {
			member.Attributes = map[string]string{"key": strings.Repeat("v", MaxAttributeValueBytes+1)}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			member := validCollectionMember("a")
			test.mutate(&member)
			if err := member.Validate(); err == nil {
				t.Fatalf("invalid member was accepted: %#v", member)
			}
		})
	}
}

func TestCollectionPageRejectsInvalidOrderAndBounds(t *testing.T) {
	valid := CollectionPageV1{Version: 1, Revision: 1, Members: []CollectionMemberV1{validCollectionMember("a")}}
	tests := []struct {
		name   string
		mutate func(*CollectionPageV1)
	}{
		{"version", func(page *CollectionPageV1) { page.Version = 2 }},
		{"revision", func(page *CollectionPageV1) { page.Revision = 0 }},
		{"revision-not-json-safe", func(page *CollectionPageV1) { page.Revision = MaxCollectionRevision + 1 }},
		{"parent", func(page *CollectionPageV1) { page.Parent = strings.Repeat("p", MaxCollectionParentBytes+1) }},
		{"cursor", func(page *CollectionPageV1) { page.NextCursor = strings.Repeat("c", MaxCollectionCursorBytes+1) }},
		{"unsorted", func(page *CollectionPageV1) {
			page.Members = []CollectionMemberV1{validCollectionMember("b"), validCollectionMember("a")}
		}},
		{"duplicate", func(page *CollectionPageV1) {
			page.Members = []CollectionMemberV1{validCollectionMember("a"), validCollectionMember("a")}
		}},
		{"member-count", func(page *CollectionPageV1) {
			page.Members = make([]CollectionMemberV1, MaxCollectionPageMembers+1)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page := valid
			test.mutate(&page)
			if err := page.Validate(); err == nil {
				t.Fatalf("invalid page was accepted: %#v", page)
			}
		})
	}
}

func TestCollectionPageCapabilitiesMustBeDescriptorSubset(t *testing.T) {
	descriptor := ResourceDescriptorV2{
		ID: ResourceID{Owner: 1, Name: "items"}, Type: ResourceTypeCollection, TypeVersion: 1,
		Capabilities: []CapabilityDescriptorV2{
			{Name: CapabilityGet, Permission: "items.get", InputSchema: SchemaCollectionMemberRequestV1, OutputSchema: SchemaCollectionMemberV1, MaxPayloadBytes: 4096},
			{Name: CapabilityList, Permission: "items.list", InputSchema: SchemaCollectionListRequestV1, OutputSchema: SchemaCollectionPageV1, MaxPayloadBytes: 4096},
		},
		Schemas: []SchemaDescriptorV2{
			{ID: SchemaCollectionListRequestV1, ContentType: "application/json"},
			{ID: SchemaCollectionMemberRequestV1, ContentType: "application/json"},
			{ID: SchemaCollectionMemberV1, ContentType: "application/json"},
			{ID: SchemaCollectionPageV1, ContentType: "application/json"},
		},
		Limits: ResourceLimitsV2{MaxPayloadBytes: 4096},
	}
	descriptor.Sort()
	page := CollectionPageV1{Version: 1, Revision: 1, Members: []CollectionMemberV1{{
		Key: "a", Kind: "item", Label: "A", Capabilities: []CapabilityID{CapabilityGet},
	}}}
	if err := page.ValidateForDescriptor(descriptor); err != nil {
		t.Fatal(err)
	}
	page.Members[0].Capabilities = []CapabilityID{CapabilityWrite}
	if err := page.ValidateForDescriptor(descriptor); err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("undeclared member capability was accepted: %v", err)
	}
}

func TestCollectionBuiltinDataSchemasExposeHardBounds(t *testing.T) {
	definitions, err := BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]DataSchemaDefinition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}
	for _, id := range []string{SchemaCollectionListRequestV1, SchemaCollectionMemberRequestV1, SchemaCollectionMemberV1, SchemaCollectionPageV1} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("collection data schema %s is missing", id)
		}
	}
	list := byID[SchemaCollectionListRequestV1]
	limit := dataSchemaAtPath(&list, "limit")
	if limit == nil || limit.Minimum == nil || *limit.Minimum != 1 || limit.Maximum == nil || *limit.Maximum != MaxCollectionPageMembers {
		t.Fatalf("collection list limit bounds are missing: %#v", limit)
	}
	page := byID[SchemaCollectionPageV1]
	revision := dataSchemaAtPath(&page, "revision")
	if revision == nil || revision.Minimum == nil || *revision.Minimum != 1 || revision.Maximum == nil || *revision.Maximum != float64(MaxCollectionRevision) {
		t.Fatalf("collection page revision bounds are missing: %#v", revision)
	}
	members := dataSchemaAtPath(&page, "members")
	if members == nil || members.MaxItems == nil || *members.MaxItems != MaxCollectionPageMembers || members.Items == nil {
		t.Fatalf("collection page member bound is missing: %#v", members)
	}
	capabilities := dataSchemaAtPath(members.Items, "capabilities")
	if capabilities == nil || capabilities.MaxItems == nil || *capabilities.MaxItems != MaxCapabilities {
		t.Fatalf("collection member capability bound is missing: %#v", capabilities)
	}
}
