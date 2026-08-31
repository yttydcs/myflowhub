package contract

import (
	"reflect"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestCanonicalFlowContractUsesOnlyCollections(t *testing.T) {
	manifest, err := Canonical()
	if err != nil {
		t.Fatal(err)
	}
	flows := make(map[string]Resource)
	for _, resource := range manifest.Resources {
		if resource.Name == protocol.BuiltinFlowDefinitions || resource.Name == protocol.BuiltinFlowRuns {
			flows[resource.Name] = resource
		}
		if resource.Name == "flow/events" || resource.Name == "flow/create" || resource.Name == "flow/update" ||
			resource.Name == "flow/run" || resource.Name == "flow/cancel" || resource.Name == "flow/archive" {
			t.Fatalf("legacy Flow resource remains in canonical contract: %s", resource.Name)
		}
	}
	if len(flows) != 2 {
		t.Fatalf("canonical Flow resources = %v, want definitions and runs", flows)
	}
	wantDefinitions := []Capability{
		capability(protocol.CapabilityFlowArchive, protocol.SchemaFlowArchiveV1, protocol.SchemaFlowArchiveV1, ""),
		capability(protocol.CapabilityFlowCreate, protocol.SchemaFlowDefinitionV1, protocol.SchemaFlowDefinitionV1, ""),
		capability(protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, protocol.SchemaFlowDefinitionV1, ""),
		capability(protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, ""),
		capability(protocol.CapabilityFlowRun, protocol.SchemaFlowRunV1, protocol.SchemaFlowRunSummaryV1, ""),
		capability(protocol.CapabilityFlowUpdate, protocol.SchemaFlowDefinitionV1, protocol.SchemaFlowDefinitionV1, ""),
	}
	wantRuns := []Capability{
		capability(protocol.CapabilityFlowCancel, protocol.SchemaFlowCancelV1, protocol.SchemaFlowRunSummaryV1, ""),
		capability(protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, protocol.SchemaFlowRunSummaryV1, ""),
		capability(protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, ""),
		capability(protocol.CapabilitySubscribe, "", "", protocol.SchemaFlowEventV1),
	}
	for name, want := range map[string][]Capability{
		protocol.BuiltinFlowDefinitions: wantDefinitions,
		protocol.BuiltinFlowRuns:        wantRuns,
	} {
		got := flows[name]
		if got.Type != string(protocol.ResourceTypeCollection) || !reflect.DeepEqual(got.Capabilities, want) {
			t.Fatalf("canonical Flow resource %s = %#v, want capabilities %#v", name, got, want)
		}
	}
}

func TestCanonicalKeepsGenericOperationAndSubscriptionMethods(t *testing.T) {
	manifest, err := Canonical()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"OperateJSON", "SubscribeCapability"} {
		if !containsString(manifest.Methods, required) || !containsString(manifest.DesktopMethods, required) {
			t.Fatalf("canonical bindings dropped generic method %s", required)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
