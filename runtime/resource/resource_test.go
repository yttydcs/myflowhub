package resource

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func descriptor(owner protocol.NodeID, name string, kind Kind) Descriptor {
	return Descriptor{ID: protocol.ResourceID{Owner: owner, Name: name}, Kind: kind, ContentType: "application/octet-stream", MaxValueBytes: 16}
}

func TestRegistryLifecycleAndOwnerBoundary(t *testing.T) {
	registry, _ := NewRegistry(1)
	variable, _ := NewVariable(descriptor(1, "status", KindVariable), []byte("ok"))
	if err := registry.Register(variable); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(variable); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate, got %v", err)
	}
	foreign, _ := NewStream(descriptor(2, "events", KindStream))
	if err := registry.Register(foreign); err == nil {
		t.Fatal("foreign owner was registered")
	}
	if _, ok := registry.Resolve(variable.Descriptor().ID); !ok {
		t.Fatal("resource not resolved")
	}
	if err := registry.Remove(variable.Descriptor().ID); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryPublishesSortedVersionedCatalog(t *testing.T) {
	registry, err := NewRegistry(1)
	if err != nil {
		t.Fatal(err)
	}
	initial := registry.Catalog().Snapshot()
	stream, _ := NewStream(Descriptor{ID: protocol.ResourceID{Owner: 1, Name: "z/events"}, Kind: KindStream, MaxValueBytes: 32})
	variable, _ := NewVariable(Descriptor{ID: protocol.ResourceID{Owner: 1, Name: "a/status"}, Kind: KindVariable, MaxValueBytes: 32}, []byte("ok"))
	if err := registry.Register(stream); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(variable); err != nil {
		t.Fatal(err)
	}
	current := registry.Catalog().Snapshot()
	if current.Revision != initial.Revision+2 {
		t.Fatalf("catalog variable revision did not follow registrations: %d -> %d", initial.Revision, current.Revision)
	}
	var catalog protocol.ResourceCatalogV1
	if err := json.Unmarshal(current.Value, &catalog); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Validate(); err != nil {
		t.Fatalf("invalid catalog: %v", err)
	}
	want := []string{"a/status", protocol.BuiltinResourceCatalog, "z/events"}
	for index, name := range want {
		if catalog.Resources[index].Name != name {
			t.Fatalf("catalog is not sorted: %#v", catalog.Resources)
		}
	}
	if err := registry.Remove(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinResourceCatalog}); !errors.Is(err, ErrReservedResource) {
		t.Fatalf("built-in catalog was removable: %v", err)
	}
}

func TestVariableRevisionAndConcurrentSets(t *testing.T) {
	variable, _ := NewVariable(descriptor(1, "status", KindVariable), []byte("a"))
	var wait sync.WaitGroup
	for i := 0; i < 20; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := variable.Set([]byte("b")); err != nil {
				t.Error(err)
			}
		}()
	}
	wait.Wait()
	if got := variable.Snapshot().Revision; got != 21 {
		t.Fatalf("want revision 21, got %d", got)
	}
	if _, err := variable.Apply(20, []byte("old")); !errors.Is(err, ErrRevisionRegression) {
		t.Fatalf("expected regression, got %v", err)
	}
}

func TestStreamSequenceAndLimits(t *testing.T) {
	stream, _ := NewStream(descriptor(1, "events", KindStream))
	first, err := stream.Publish([]byte("one"))
	if err != nil || first.Sequence != 1 {
		t.Fatalf("unexpected publish: %#v %v", first, err)
	}
	if _, err := stream.Apply(1, []byte("duplicate")); !errors.Is(err, ErrRevisionRegression) {
		t.Fatalf("expected regression, got %v", err)
	}
	if _, err := stream.Publish(make([]byte, 17)); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("expected limit, got %v", err)
	}
}

func TestCommandCopiesAndValidatesPayloads(t *testing.T) {
	command, _ := NewCommand(descriptor(1, "echo", KindCommand), func(_ context.Context, input []byte) ([]byte, error) {
		input[0] = 'E'
		return input, nil
	})
	input := []byte("echo")
	output, err := command.Invoke(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if string(input) != "echo" || string(output) != "Echo" {
		t.Fatalf("unexpected copy behavior: input %q output %q", input, output)
	}
}
