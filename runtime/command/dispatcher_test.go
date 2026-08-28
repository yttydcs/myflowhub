package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func commandRegistry(t *testing.T, handler resource.CommandHandler) (*resource.Registry, protocol.ResourceID) {
	t.Helper()
	id := protocol.ResourceID{Owner: 1, Name: "do"}
	registry, _ := resource.NewRegistry(1)
	command, err := resource.NewCommand(resource.CommandDescriptor(id, "application/octet-stream", "test.raw.v1", "test.invoke", 64), handler)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(command); err != nil {
		t.Fatal(err)
	}
	return registry, id
}

func call(id protocol.ResourceID) Call {
	return Call{MessageID: protocol.MustMessageID(), Source: 2, Resource: id, Capability: protocol.CapabilityInvoke, Schema: "test.raw.v1", Input: []byte("input"), Deadline: time.Now().Add(time.Second), Origin: OriginAdjudicated}
}

func TestAdjudicatedAllowDenyAndParentControl(t *testing.T) {
	registry, id := commandRegistry(t, func(_ context.Context, input []byte) ([]byte, error) { return input, nil })
	dispatcher, _ := NewDispatcher(registry, Config{Authorizer: func(_ context.Context, value Call) error {
		if value.Source == 2 {
			return errors.New("denied")
		}
		return nil
	}})
	denied, err := dispatcher.Invoke(context.Background(), call(id))
	if err == nil || denied.Failure == nil || denied.Failure.Code != protocol.CodeForbidden {
		t.Fatalf("expected deny, got %#v %v", denied, err)
	}
	parent := call(id)
	parent.Origin = OriginParentControl
	result, err := dispatcher.Invoke(context.Background(), parent)
	if err != nil || string(result.Output) != "input" {
		t.Fatalf("parent control failed: %#v %v", result, err)
	}
}

func TestTimeoutDropsLateResult(t *testing.T) {
	registry, id := commandRegistry(t, func(_ context.Context, _ []byte) ([]byte, error) {
		time.Sleep(40 * time.Millisecond)
		return []byte("late"), nil
	})
	late := make(chan Result, 1)
	dispatcher, _ := NewDispatcher(registry, Config{Authorizer: func(context.Context, Call) error { return nil }, OnLateResult: func(_ Call, result Result) { late <- result }})
	value := call(id)
	value.Deadline = time.Now().Add(5 * time.Millisecond)
	result, err := dispatcher.Invoke(context.Background(), value)
	if !errors.Is(err, context.DeadlineExceeded) || result.Failure == nil || result.Failure.Code != protocol.CodeTimeout {
		t.Fatalf("expected timeout, got %#v %v", result, err)
	}
	select {
	case observed := <-late:
		if string(observed.Output) != "late" {
			t.Fatalf("unexpected late result: %#v", observed)
		}
	case <-time.After(time.Second):
		t.Fatal("late result was not observed")
	}
}

func TestDuplicateAndPanicIsolation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	registry, id := commandRegistry(t, func(_ context.Context, _ []byte) ([]byte, error) {
		close(started)
		<-release
		panic("boom")
	})
	dispatcher, _ := NewDispatcher(registry, Config{Authorizer: func(context.Context, Call) error { return nil }})
	value := call(id)
	first := make(chan Result, 1)
	go func() {
		result, _ := dispatcher.Invoke(context.Background(), value)
		first <- result
	}()
	<-started
	duplicate, err := dispatcher.Invoke(context.Background(), value)
	if !errors.Is(err, ErrDuplicate) || duplicate.Failure == nil || duplicate.Failure.Code != protocol.CodeConflict {
		t.Fatalf("expected duplicate, got %#v %v", duplicate, err)
	}
	close(release)
	result := <-first
	if result.Failure == nil || result.Failure.Code != protocol.CodeInternal {
		t.Fatalf("panic escaped isolation: %#v", result)
	}
	if _, err := dispatcher.Invoke(context.Background(), value); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("completed duplicate was accepted: %v", err)
	}
}
