package variables

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestReferenceLoadUsesStandardDefinitionsAndPreservesWrittenValues(t *testing.T) {
	registry, err := resource.NewRegistry(7)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := Register(registry, 7, ReferenceCount)
	if err != nil {
		t.Fatal(err)
	}
	if len(provider.Variables) != 1998 {
		t.Fatal("wrong reference count")
	}
	for i, variable := range provider.Variables {
		before := variable.Snapshot()
		if err := protocol.ValidateVariableState(*before.State.Definition, before.State); err != nil {
			t.Fatalf("type %d: %v", i%6, err)
		}
		if i%6 < 2 {
			input := json.RawMessage(`false`)
			if i%6 == 1 {
				input = json.RawMessage(`40`)
			}
			if _, err := variable.CompareAndSet(context.Background(), before.State.InstanceID, before.Revision, input); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := provider.Sample(1); err != nil {
		t.Fatal(err)
	}
	for i, variable := range provider.Variables {
		after := variable.Snapshot()
		if i%6 == 0 && string(after.State.Value) != "false" {
			t.Fatal("sampling undid a boolean write")
		}
		if i%6 == 1 && string(after.State.Value) != "40" {
			t.Fatal("sampling undid a number write")
		}
		if len(after.State.Value) > 256 {
			t.Fatal("reference sample exceeds its specified size")
		}
	}
	if samples, superseded := provider.Counts(); samples != 1998 || superseded != 0 {
		t.Fatalf("counts: %d %d", samples, superseded)
	}
}

func TestExampleRejectsInvalidCountsBeforeRegistration(t *testing.T) {
	registry, _ := resource.NewRegistry(7)
	for _, count := range []int{0, 5, 7, 4002} {
		if _, err := Register(registry, 7, count); err == nil {
			t.Fatalf("accepted %d", count)
		}
	}
	if len(registry.List()) != 1 {
		t.Fatal("failed example changed the registry")
	}
}
