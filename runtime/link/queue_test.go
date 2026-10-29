package link

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestEnvelopeQueueByteBudgetAndRollback(t *testing.T) {
	frame := testEnvelope(protocol.OperationResourceEvent, protocol.PhaseEvent)
	frame.Payload = []byte(`{"value":true}`)
	queue := newEnvelopeQueue(1, 2*envelopeBytes(frame))
	if !queue.push(frame) || queue.push(frame) {
		t.Fatal("frame count limit not enforced")
	}
	if queue.bytes.Load() != envelopeBytes(frame) {
		t.Fatal("failed enqueue leaked byte reservation")
	}
	queue.release(<-queue.frames)
	if queue.bytes.Load() != 0 || !queue.push(frame) {
		t.Fatal("dequeue did not free byte reservation")
	}
	tooLarge := frame
	tooLarge.Payload = make([]byte, queue.maxBytes)
	if queue.push(tooLarge) {
		t.Fatal("oversized frame bypassed byte limit")
	}
}

func TestSessionByteBudgetIncludesBlockedWriterAndReservesControl(t *testing.T) {
	pipe := newBlockingPipe()
	event := testEnvelope(protocol.OperationResourceEvent, protocol.PhaseEvent)
	session, err := NewSession(context.Background(), pipe, SessionConfig{LocalNode: 1, DataQueue: 4096, DataQueueBytes: envelopeBytes(event)})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Activate(2, RoleChild, 1); err != nil {
		t.Fatal(err)
	}
	if err := session.Send(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	select {
	case <-pipe.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("writer did not start")
	}
	if err := session.Send(context.Background(), event); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("in-flight payload escaped budget: %v", err)
	}
	if err := session.Send(context.Background(), testEnvelope(protocol.OperationOperate, protocol.PhaseControl)); err != nil {
		t.Fatalf("data consumed control budget: %v", err)
	}
}

func TestSessionInboundByteBudgetClosesOverflowWithoutConsumer(t *testing.T) {
	left, right := pipePair()
	defer right.Close()
	frame := testEnvelope(protocol.OperationOperate, protocol.PhaseControl)
	session, err := NewSession(context.Background(), left, SessionConfig{LocalNode: 1, InboundQueue: 4096, InboundQueueBytes: envelopeBytes(frame)})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Activate(2, RoleChild, 1); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := (protocol.Codec{}).Encode(right, frame); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-session.Done():
		if !errors.Is(session.Err(), ErrQueueFull) {
			t.Fatalf("overflow: %v", session.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("inbound exceeded byte budget")
	}
}
