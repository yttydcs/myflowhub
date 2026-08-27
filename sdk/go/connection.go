package sdk

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/link"
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

type Connection struct {
	supervisor *node.ParentSupervisor
	cancel     context.CancelFunc
	changes    chan ConnectionSnapshot
	done       chan struct{}
	stopOnce   sync.Once
}

func (c *Client) ConnectManaged(ctx context.Context, driver link.Driver, endpoint link.Endpoint, parent protocol.NodeID, config node.SupervisorConfig) (*Connection, error) {
	if ctx == nil {
		return nil, errors.New("SDK managed connection context is required")
	}
	runtime, err := c.runtimeNode()
	if err != nil {
		return nil, err
	}
	supervisor, err := runtime.SuperviseParent(ctx, driver, endpoint, parent, config)
	if err != nil {
		return nil, wrapError(err)
	}
	watchCtx, cancel := context.WithCancel(ctx)
	value := &Connection{supervisor: supervisor, cancel: cancel, changes: make(chan ConnectionSnapshot, 1), done: make(chan struct{})}
	go value.watch(watchCtx)
	return value, nil
}

func (c *Connection) Snapshot() ConnectionSnapshot {
	if c == nil || c.supervisor == nil {
		return ConnectionSnapshot{State: ConnectionStopped}
	}
	return convertConnection(c.supervisor.Snapshot())
}

func (c *Connection) Changes() <-chan ConnectionSnapshot {
	if c == nil {
		closed := make(chan ConnectionSnapshot)
		close(closed)
		return closed
	}
	return c.changes
}

func (c *Connection) Done() <-chan struct{} {
	if c == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return c.done
}

func (c *Connection) Stop() {
	if c == nil {
		return
	}
	c.stopOnce.Do(func() {
		c.cancel()
		c.supervisor.Stop()
		<-c.done
	})
}

func (c *Connection) watch(ctx context.Context) {
	defer close(c.done)
	defer close(c.changes)
	current := c.supervisor.Snapshot()
	c.publish(convertConnection(current))
	for {
		next, err := c.supervisor.WaitChange(ctx, current.Generation)
		if err != nil {
			return
		}
		current = next
		c.publish(convertConnection(next))
		if next.State == node.ConnectionFailed || next.State == node.ConnectionStopped {
			return
		}
	}
}

func (c *Connection) publish(snapshot ConnectionSnapshot) {
	select {
	case c.changes <- snapshot:
	default:
		select {
		case <-c.changes:
		default:
		}
		select {
		case c.changes <- snapshot:
		default:
		}
	}
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
