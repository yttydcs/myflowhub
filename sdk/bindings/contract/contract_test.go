package contract

import (
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
