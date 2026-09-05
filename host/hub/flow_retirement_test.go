package hub

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestPersistentHubRetiresFlowWithoutTouchingLegacyState(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "fresh state"
		if legacy {
			name = "legacy state"
		}
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			flowPath := filepath.Join(directory, "flow.json")
			// Incompatible data must not prevent startup or be rewritten by the Hub.
			original := []byte("legacy Flow state: intentionally not valid JSON\n")
			if legacy {
				if err := os.WriteFile(flowPath, original, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			network := memory.NewNetwork()
			defer network.Close()
			value, err := StartPersistent(ctx, PersistentConfig{
				StateDirectory: directory, NodeID: 1,
				Listeners: []ListenerConfig{{Driver: network, Endpoint: "retired-flow"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer value.Close()
			var catalog protocol.ResourceCatalogV2
			if err := protocol.DecodeJSONPayload(value.Node.Registry().Catalog().Snapshot().Value, protocol.DefaultMaxPayload, &catalog); err != nil {
				t.Fatal(err)
			}
			present := make(map[string]bool)
			for _, descriptor := range catalog.Resources {
				present[descriptor.ID.Name] = true
				if strings.HasPrefix(descriptor.ID.Name, "flow/") {
					t.Errorf("retired Flow resource registered: %s", descriptor.ID.Name)
				}
			}
			for _, required := range []string{protocol.BuiltinManagementHealth, protocol.BuiltinFileUpload, protocol.BuiltinNotificationPublish, protocol.BuiltinPolicyDefinitions} {
				if !present[required] {
					t.Errorf("active resource missing: %s", required)
				}
			}
			_, err = value.Node.Operate(ctx, protocol.ResourceID{Owner: 1, Name: "flow/definitions"}, protocol.CapabilityList,
				protocol.SchemaCollectionListRequestV1, []byte(`{"version":1,"limit":10}`))
			var failure protocol.ErrorPayload
			if !errors.As(err, &failure) || failure.Code != protocol.CodeNotFound {
				t.Fatalf("retired Flow operation error = %v, want NotFound", err)
			}
			if err := value.Close(); err != nil {
				t.Fatal(err)
			}
			stored, err := os.ReadFile(flowPath)
			if legacy {
				if err != nil || !bytes.Equal(stored, original) {
					t.Fatalf("legacy Flow state changed: %q (%v)", stored, err)
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("fresh Hub created Flow state: %v", err)
			}
		})
	}
}
