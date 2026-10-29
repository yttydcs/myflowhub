package link

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestHeartbeatCompatibilityDoesNotExpireLegacyPeer(t *testing.T) {
	left, right := pipePair()
	defer right.Close()
	session, err := NewSession(context.Background(), left, SessionConfig{LocalNode: 1, IdleTimeout: 40 * time.Millisecond, IdleAfterPeerHeartbeat: true})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Activate(2, RoleChild, 1); err != nil {
		t.Fatal(err)
	}
	// Ordinary traffic and another node's heartbeat must not opt this peer in.
	for _, op := range []protocol.Operation{protocol.OperationOperate, protocol.OperationHeartbeat} {
		envelope := testEnvelope(op, protocol.PhaseRequest)
		envelope.Source, envelope.Target = 2, 1
		if op == protocol.OperationHeartbeat {
			envelope.Phase = protocol.PhaseEvent
			envelope.Resource = protocol.ResourceID{}
			envelope.Capability = ""
			envelope.Source = 3
		}
		if err := (protocol.Codec{}).Encode(right, envelope); err != nil {
			t.Fatal(err)
		}
		select {
		case <-session.Inbound():
		case <-time.After(time.Second):
			t.Fatal("frame not delivered")
		}
	}
	select {
	case <-session.Done():
		t.Fatalf("healthy legacy peer was disconnected: %v", session.Err())
	case <-time.After(160 * time.Millisecond):
	}
	if session.IdleMonitoringActive() {
		t.Fatal("legacy peer unexpectedly enabled idle monitoring")
	}
}

func TestHeartbeatCompatibilityEnablesStickyIdleDeadline(t *testing.T) {
	left, right := pipePair()
	defer right.Close()
	session, err := NewSession(context.Background(), left, SessionConfig{LocalNode: 1, IdleTimeout: 60 * time.Millisecond, IdleAfterPeerHeartbeat: true})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Activate(2, RoleChild, 1); err != nil {
		t.Fatal(err)
	}
	if session.IdleMonitoringActive() {
		t.Fatal("peer has not advertised a heartbeat")
	}
	heartbeat := protocol.Envelope{Version: protocol.CurrentVersion, Phase: protocol.PhaseEvent,
		Operation: protocol.OperationHeartbeat, MessageID: protocol.MustMessageID(), Source: 2, Target: 1, TopologyEpoch: 1}
	if err := (protocol.Codec{}).Encode(right, heartbeat); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.Inbound():
	case <-time.After(time.Second):
		t.Fatal("heartbeat not delivered")
	}
	if !session.IdleMonitoringActive() {
		t.Fatal("matching peer heartbeat did not enable idle monitoring")
	}
	select {
	case <-session.Done():
		if !errors.Is(session.Err(), ErrIdle) {
			t.Fatalf("unexpected close reason: %v", session.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("silent current peer stayed connected")
	}
	if !session.IdleMonitoringActive() {
		t.Fatal("silence switched back to legacy mode")
	}
}
