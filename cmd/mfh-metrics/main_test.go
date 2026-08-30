package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

func TestWaitConnectedUsesConnectionStatusChanges(t *testing.T) {
	connection := &scriptedConnection{snapshots: []sdk.ConnectionSnapshot{
		{State: sdk.ConnectionConnecting, Generation: 1},
		{State: sdk.ConnectionConnected, Generation: 2},
	}}
	if err := waitConnected(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	if connection.waits != 1 {
		t.Fatalf("waited %d times, want 1", connection.waits)
	}
}

func TestWaitConnectedReportsTerminalAndContextErrors(t *testing.T) {
	failed := &scriptedConnection{snapshots: []sdk.ConnectionSnapshot{{
		State: sdk.ConnectionFailed, LastError: "authentication rejected", Generation: 1,
	}}}
	if err := waitConnected(context.Background(), failed); err == nil || !strings.Contains(err.Error(), "authentication rejected") {
		t.Fatalf("unexpected terminal error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := waitConnected(ctx, blockingConnection{}); err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected context error: %v", err)
	}
}

type scriptedConnection struct {
	snapshots []sdk.ConnectionSnapshot
	index     int
	waits     int
}

func (c *scriptedConnection) Snapshot() sdk.ConnectionSnapshot { return c.snapshots[c.index] }

func (c *scriptedConnection) WaitChange(context.Context, uint64) (sdk.ConnectionSnapshot, error) {
	c.waits++
	if c.index+1 >= len(c.snapshots) {
		return sdk.ConnectionSnapshot{}, errors.New("scripted connection exhausted")
	}
	c.index++
	return c.snapshots[c.index], nil
}

type blockingConnection struct{}

func (blockingConnection) Snapshot() sdk.ConnectionSnapshot {
	return sdk.ConnectionSnapshot{State: sdk.ConnectionConnecting, Generation: 1}
}

func (blockingConnection) WaitChange(ctx context.Context, _ uint64) (sdk.ConnectionSnapshot, error) {
	<-ctx.Done()
	return sdk.ConnectionSnapshot{}, ctx.Err()
}
