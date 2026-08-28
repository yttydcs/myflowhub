package resource

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func variableDescriptor(owner protocol.NodeID, name string, maxPayload int) Descriptor {
	return VariableDescriptor(protocol.ResourceID{Owner: owner, Name: name}, "application/octet-stream", "test.raw.v1", "test.read", maxPayload)
}

func streamDescriptor(owner protocol.NodeID, name string, maxPayload int) Descriptor {
	return StreamDescriptor(protocol.ResourceID{Owner: owner, Name: name}, "application/octet-stream", "test.raw.v1", "test.read", maxPayload)
}

func TestRegistryLifecycleAndOwnerBoundary(t *testing.T) {
	registry, _ := NewRegistry(1)
	variable, _ := NewVariable(variableDescriptor(1, "status", 16), []byte("ok"))
	if err := registry.Register(variable); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(variable); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate, got %v", err)
	}
	foreign, _ := NewStream(streamDescriptor(2, "events", 16))
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
	stream, _ := NewStream(streamDescriptor(1, "z/events", 32))
	variable, _ := NewVariable(variableDescriptor(1, "a/status", 32), []byte("ok"))
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
	var catalog protocol.ResourceCatalogV2
	if err := json.Unmarshal(current.Value, &catalog); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Validate(); err != nil {
		t.Fatalf("invalid catalog: %v", err)
	}
	want := []string{"a/status", protocol.BuiltinResourceCatalog, "z/events"}
	for index, name := range want {
		if catalog.Resources[index].ID.Name != name {
			t.Fatalf("catalog is not sorted: %#v", catalog.Resources)
		}
	}
	if err := registry.Remove(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinResourceCatalog}); !errors.Is(err, ErrReservedResource) {
		t.Fatalf("built-in catalog was removable: %v", err)
	}
}

func TestVariableRevisionAndConcurrentSets(t *testing.T) {
	variable, _ := NewVariable(variableDescriptor(1, "status", 16), []byte("a"))
	var wait sync.WaitGroup
	for range 20 {
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
	stream, _ := NewStream(streamDescriptor(1, "events", 16))
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
	command, _ := NewCommand(CommandDescriptor(protocol.ResourceID{Owner: 1, Name: "echo"}, "application/octet-stream", "test.raw.v1", "test.invoke", 16), func(_ context.Context, input []byte) ([]byte, error) {
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

func TestCommandDescriptorDeclaresDistinctInputAndOutputSchemas(t *testing.T) {
	descriptor := CommandDescriptorSchemas(
		protocol.ResourceID{Owner: 1, Name: "convert"}, "application/json",
		"test.request.v1", "test.response.v1", "test.convert", 64,
	)
	capability, ok := descriptor.Capability(protocol.CapabilityInvoke)
	if !ok || capability.InputSchema != "test.request.v1" || capability.OutputSchema != "test.response.v1" {
		t.Fatalf("unexpected invoke contract: %#v", capability)
	}
	if len(descriptor.Schemas) != 2 || descriptor.Schemas[0].ID != "test.request.v1" || descriptor.Schemas[1].ID != "test.response.v1" {
		t.Fatalf("unexpected command schemas: %#v", descriptor.Schemas)
	}
	if err := descriptor.Validate(); err != nil {
		t.Fatalf("distinct command descriptor is invalid: %v", err)
	}
}

func TestWritableVariableUsesConditionalRevision(t *testing.T) {
	descriptor := WritableVariableDescriptor(protocol.ResourceID{Owner: 1, Name: "settings"}, "application/json", "test.settings.v1", "test.read", "test.write", 256)
	variable, err := NewVariable(descriptor, []byte(`{"enabled":false}`))
	if err != nil {
		t.Fatal(err)
	}
	write, err := protocol.EncodeJSONPayload(&protocol.VariableWriteV2{Version: 2, ExpectedRevision: 1, Value: []byte(`{"enabled":true}`)}, 256)
	if err != nil {
		t.Fatal(err)
	}
	result, err := variable.Operate(context.Background(), OperationRequest{Capability: protocol.CapabilityWrite, Schema: protocol.SchemaVariableWriteV2, Payload: write})
	if err != nil || string(result.Payload) != `{"enabled":true}` || variable.Snapshot().Revision != 2 {
		t.Fatalf("conditional write failed: %#v %v", result, err)
	}
	if _, err := variable.Operate(context.Background(), OperationRequest{Capability: protocol.CapabilityWrite, Schema: protocol.SchemaVariableWriteV2, Payload: write}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale write did not conflict: %v", err)
	}
}

type customResource struct{ descriptor Descriptor }

func (c customResource) Descriptor() Descriptor { return c.descriptor }
func (c customResource) Operate(_ context.Context, request OperationRequest) (OperationResult, error) {
	return OperationResult{Schema: "example.pong.v1", Payload: append([]byte("pong:"), request.Payload...)}, nil
}

func TestCustomTypeRegistersWithoutCoreChanges(t *testing.T) {
	descriptor := Descriptor{
		ID: protocol.ResourceID{Owner: 1, Name: "custom/ping"}, Type: "example.ping", TypeVersion: 1,
		Capabilities: []protocol.CapabilityDescriptorV2{{Name: "ping", Permission: "custom.ping", InputSchema: "example.ping.v1", OutputSchema: "example.pong.v1", MaxPayloadBytes: 32}},
		Schemas:      []protocol.SchemaDescriptorV2{{ID: "example.ping.v1", ContentType: "text/plain"}, {ID: "example.pong.v1", ContentType: "text/plain"}},
		Limits:       protocol.ResourceLimitsV2{MaxPayloadBytes: 32},
	}
	registry, _ := NewRegistry(1)
	if err := registry.Register(customResource{descriptor: descriptor}); err != nil {
		t.Fatal(err)
	}
	result, err := registry.Operate(context.Background(), descriptor.ID, OperationRequest{Subject: 2, Capability: "ping", Schema: "example.ping.v1", Payload: []byte("ok")})
	if err != nil || string(result.Payload) != "pong:ok" {
		t.Fatalf("custom operation failed: %#v %v", result, err)
	}
}

func TestTopicSeparatesPublisherSequencesAndDoesNotReplay(t *testing.T) {
	topic, err := NewTopic(TopicDescriptor(protocol.ResourceID{Owner: 1, Name: "events/shared"}, "application/octet-stream", "test.raw.v1", "topic.publish", "topic.subscribe", 32), TopicConfig{MaxEventsPerSecond: 8})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := topic.Publish(2, []byte("before")); err != nil {
		t.Fatal(err)
	}
	events := make(chan TopicEvent, 2)
	cancel, err := topic.Watch(func(event TopicEvent) { events <- event })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	first, _ := topic.Publish(2, []byte("a"))
	second, _ := topic.Publish(3, []byte("b"))
	if first.PublisherSequence != 2 || second.PublisherSequence != 1 || first.Sequence+1 != second.Sequence {
		t.Fatalf("unexpected topic sequences: %#v %#v", first, second)
	}
	if got := <-events; string(got.Value) == "before" {
		t.Fatal("topic replayed an event published before subscription")
	}
}
