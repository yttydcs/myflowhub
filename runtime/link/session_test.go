package link

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestSessionRejectsApplicationFrameBeforeActivation(t *testing.T) {
	left, right := pipePair()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session, err := NewSession(ctx, left, SessionConfig{LocalNode: 1})
	if err != nil {
		t.Fatal(err)
	}
	envelope := testEnvelope(protocol.OperationCommandCall, protocol.PhaseRequest)
	if err := (protocol.Codec{}).Encode(right, envelope); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.Done():
		if session.Err() == nil {
			t.Fatal("expected authentication failure")
		}
	case <-time.After(time.Second):
		t.Fatal("session did not reject frame")
	}
	_ = right.Close()
}

func TestSessionBoundedDataQueue(t *testing.T) {
	pipe := newBlockingPipe()
	session, err := NewSession(context.Background(), pipe, SessionConfig{LocalNode: 1, ControlQueue: 1, DataQueue: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Activate(2, RoleChild, 1); err != nil {
		t.Fatal(err)
	}
	event := testEnvelope(protocol.OperationStreamEvent, protocol.PhaseEvent)
	if err := session.Send(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	select {
	case <-pipe.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("writer did not block")
	}
	if err := session.Send(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := session.Send(context.Background(), event); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected queue full, got %v", err)
	}
	control := testEnvelope(protocol.OperationCommandCall, protocol.PhaseControl)
	if err := session.Send(context.Background(), control); err != nil {
		t.Fatalf("control queue was starved by saturated data: %v", err)
	}
	_ = session.Close()
}

func TestSessionIdleTimeoutClosesInactiveLink(t *testing.T) {
	pipe := newBlockingPipe()
	session, err := NewSession(context.Background(), pipe, SessionConfig{LocalNode: 1, IdleTimeout: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Activate(2, RoleChild, 1); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.Done():
		if !errors.Is(session.Err(), ErrIdle) {
			t.Fatalf("expected idle timeout, got %v", session.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("inactive session did not close")
	}
}

func TestSessionParentCancellationClosesPipe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pipe := newBlockingPipe()
	session, err := NewSession(ctx, pipe, SessionConfig{LocalNode: 1})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-session.Done():
		if session.State() != StateClosed {
			t.Fatalf("unexpected final state %d", session.State())
		}
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not close session")
	}
}

func testEnvelope(operation protocol.Operation, phase protocol.Phase) protocol.Envelope {
	return protocol.Envelope{
		Version:   protocol.CurrentVersion,
		Phase:     phase,
		Operation: operation,
		MessageID: protocol.MustMessageID(),
		Source:    1,
		Target:    2,
		Resource:  protocol.ResourceID{Owner: 2, Name: "test"},
	}
}

type inMemoryPipe struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func (p *inMemoryPipe) Read(data []byte) (int, error)  { return p.reader.Read(data) }
func (p *inMemoryPipe) Write(data []byte) (int, error) { return p.writer.Write(data) }
func (p *inMemoryPipe) Close() error {
	_ = p.reader.Close()
	return p.writer.Close()
}

func pipePair() (*inMemoryPipe, *inMemoryPipe) {
	leftReader, rightWriter := io.Pipe()
	rightReader, leftWriter := io.Pipe()
	return &inMemoryPipe{reader: leftReader, writer: leftWriter}, &inMemoryPipe{reader: rightReader, writer: rightWriter}
}

type blockingPipe struct {
	writeStarted chan struct{}
	closed       chan struct{}
	once         sync.Once
}

func newBlockingPipe() *blockingPipe {
	return &blockingPipe{writeStarted: make(chan struct{}), closed: make(chan struct{})}
}

func (p *blockingPipe) Read([]byte) (int, error) {
	<-p.closed
	return 0, io.EOF
}

func (p *blockingPipe) Write(data []byte) (int, error) {
	p.once.Do(func() { close(p.writeStarted) })
	<-p.closed
	return 0, io.ErrClosedPipe
}

func (p *blockingPipe) Close() error {
	select {
	case <-p.closed:
	default:
		close(p.closed)
	}
	return nil
}
