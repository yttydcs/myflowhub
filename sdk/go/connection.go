package sdk

import (
	"context"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/node"
)

type ConnectionState string

const (
	ConnectionDisconnected ConnectionState = "disconnected"
	ConnectionConnecting   ConnectionState = "connecting"
	ConnectionConnected    ConnectionState = "connected"
	ConnectionFailed       ConnectionState = "failed"
	ConnectionStopped      ConnectionState = "stopped"
)

type ConnectionSnapshot struct {
	State          ConnectionState
	Parent         protocol.NodeID
	Endpoint       string
	Attempt        uint64
	LinkGeneration uint64
	LastError      string
	NextRetry      time.Time
	Generation     uint64
}

// ConnectionStatus is a read-only view of a supervised parent connection.
// Its owner manages the parent supervisor; SDK consumers only observe it.
type ConnectionStatus interface {
	Snapshot() ConnectionSnapshot
	WaitChange(context.Context, uint64) (ConnectionSnapshot, error)
}

// ConnectionSnapshotFromRuntime converts runtime connection state into the
// stable SDK view used by bindings and Host status adapters.
func ConnectionSnapshotFromRuntime(value node.ConnectionSnapshot) ConnectionSnapshot {
	return convertConnection(value)
}

func convertConnection(value node.ConnectionSnapshot) ConnectionSnapshot {
	state := ConnectionDisconnected
	switch value.State {
	case node.ConnectionConnecting:
		state = ConnectionConnecting
	case node.ConnectionConnected:
		state = ConnectionConnected
	case node.ConnectionFailed:
		state = ConnectionFailed
	case node.ConnectionStopped:
		state = ConnectionStopped
	}
	return ConnectionSnapshot{
		State: state, Parent: value.Parent, Endpoint: string(value.Endpoint), Attempt: value.Attempt,
		LinkGeneration: value.LinkGeneration, LastError: value.LastError, NextRetry: value.NextRetry, Generation: value.Generation,
	}
}
