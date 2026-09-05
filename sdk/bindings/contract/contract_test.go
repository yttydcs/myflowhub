package contract

import (
	"reflect"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestCanonicalOmitsRetiredFlowContract(t *testing.T) {
	manifest, err := Canonical()
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range manifest.Resources {
		if strings.HasPrefix(resource.Name, "flow/") {
			t.Errorf("retired Flow resource remains in canonical contract: %s", resource.Name)
		}
		for _, capability := range resource.Capabilities {
			for _, schema := range []string{capability.InputSchema, capability.OutputSchema, capability.EventSchema} {
				if strings.HasPrefix(schema, "mfh.flow.") {
					t.Errorf("retired Flow schema referenced by %s: %s", resource.Name, schema)
				}
			}
		}
	}
	schemas, err := protocol.BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, schema := range schemas {
		if strings.HasPrefix(schema.ID, "mfh.flow.") {
			t.Errorf("retired Flow schema remains in builtin data schemas: %s", schema.ID)
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

func TestCanonicalTopologyKeepsVariableAndDeclaresDepthQueries(t *testing.T) {
	manifest, err := Canonical()
	if err != nil {
		t.Fatal(err)
	}
	want := Resource{
		Name: protocol.BuiltinManagementTopology, Type: string(protocol.ResourceTypeVariable),
		Capabilities: []Capability{
			{Name: string(protocol.CapabilityChildren), InputSchema: protocol.SchemaManagementTopologyChildrenRequestV1, OutputSchema: protocol.SchemaManagementTopologyQueryV1},
			{Name: string(protocol.CapabilityRead), OutputSchema: protocol.SchemaManagementTopologyV1},
			{Name: string(protocol.CapabilitySubscribe), EventSchema: protocol.SchemaManagementTopologyV1},
			{Name: string(protocol.CapabilitySubtree), InputSchema: protocol.SchemaManagementTopologyQueryRequestV1, OutputSchema: protocol.SchemaManagementTopologyQueryV1},
		},
	}
	for _, value := range manifest.Resources {
		if value.Name == want.Name {
			if !reflect.DeepEqual(value, want) {
				t.Fatalf("topology contract mismatch: got %+v, want %+v", value, want)
			}
			return
		}
	}
	t.Fatal("canonical contract is missing the topology Resource")
}

func TestCanonicalTopologyDoesNotAddFacadeOrLifecycleMethods(t *testing.T) {
	manifest, err := Canonical()
	if err != nil {
		t.Fatal(err)
	}
	portable := []string{
		"CancelSubscription", "CatalogJSON", "Close", "IdentityJSON", "InvokeJSON", "OperateJSON",
		"SnapshotJSON", "StatusJSON", "Subscribe", "SubscribeCapability", "UploadFile", "WaitConnected",
	}
	desktop := []string{
		"CancelSubscription", "CatalogJSON", "Close", "IdentityJSON", "InvokeJSON", "OperateJSON", "PollSubscription",
		"SnapshotJSON", "StatusJSON", "Subscribe", "SubscribeCapability", "UploadFile", "WaitConnected",
	}
	if !reflect.DeepEqual(manifest.Methods, portable) || !reflect.DeepEqual(manifest.DesktopMethods, desktop) {
		t.Fatalf("topology query changed attached binding APIs: portable=%v desktop=%v", manifest.Methods, manifest.DesktopMethods)
	}
}
