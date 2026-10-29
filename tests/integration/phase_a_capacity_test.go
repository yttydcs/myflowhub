package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/sdk/bindings"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type capacityConnection struct{ parent *node.ParentSupervisor }

func (s capacityConnection) Snapshot() sdk.ConnectionSnapshot {
	return sdk.ConnectionSnapshotFromRuntime(s.parent.Snapshot())
}
func (s capacityConnection) WaitChange(ctx context.Context, after uint64) (sdk.ConnectionSnapshot, error) {
	value, err := s.parent.WaitChange(ctx, after)
	return sdk.ConnectionSnapshotFromRuntime(value), err
}

// Local TCP proves real runtime/SDK/binding capacity, not the separate two-PC,
// two-hour or physical Windows audio acceptance criteria.
func TestPhaseACapacity2000AcrossTCPAndDesktopBatchBinding(t *testing.T) {
	t.Setenv("MFH_MAX_SUBSCRIPTIONS", "4096")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	mechanics, err := nodehost.ProductRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	identities := make([]auth.Identity, 3)
	for i := range identities {
		identities[i], err = auth.GenerateIdentity(protocol.NodeID(i + 1))
		if err != nil {
			t.Fatal(err)
		}
		if err := trust.Add(identities[i].NodeID, identities[i].PublicKey); err != nil {
			t.Fatal(err)
		}
	}
	nodes := make([]*node.Node, 3)
	for i := range nodes {
		nodes[i], err = node.New(ctx, node.Config{Identity: identities[i], Trust: trust, Policy: auth.AllowAll{}, Subscriptions: mechanics.Subscriptions, MaxPending: mechanics.MaxPending, Session: mechanics.Session})
		if err != nil {
			t.Fatal(err)
		}
		defer nodes[i].Close()
	}
	root, consumer, owner := nodes[0], nodes[1], nodes[2]
	endpoint, err := root.Listen(tcp.Driver{}, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.ConnectParent(ctx, tcp.Driver{}, endpoint, root.ID()); err != nil {
		t.Fatal(err)
	}
	parent, err := consumer.SuperviseParent(ctx, tcp.Driver{}, endpoint, root.ID(), node.SupervisorConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Stop()
	client, err := sdk.NewAttachedClient(consumer)
	if err != nil {
		t.Fatal(err)
	}
	core, err := bindings.NewAttachedClient(client, bindings.PublicIdentity{NodeID: consumer.ID(), PublicKey: identities[1].PublicKey}, capacityConnection{parent})
	if err != nil {
		t.Fatal(err)
	}
	facade, err := desktopbinding.NewAttachedClient(core)
	if err != nil {
		t.Fatal(err)
	}
	defer facade.Close()
	if err := facade.WaitConnected(5000); err != nil {
		t.Fatal(err)
	}
	variables := make([]*resource.Variable, 2000)
	values := make([]resource.Resource, len(variables))
	for i := range variables {
		variables[i], err = resource.NewVariable(resource.VariableDescriptor(protocol.ResourceID{Owner: owner.ID(), Name: fmt.Sprintf("load/%04d", i)}, "test.read", 4096), protocol.DataSchemaDefinition{Type: "boolean"}, resource.VariableState{Status: "ready", Value: json.RawMessage(`false`)}, nil)
		if err != nil {
			t.Fatal(err)
		}
		values[i] = variables[i]
	}
	if err := owner.Registry().RegisterBatch(values); err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, len(variables))
	var next atomic.Int64
	var failed atomic.Bool
	failures := make(chan error, 32)
	var workers sync.WaitGroup
	started := time.Now()
	for range 32 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for !failed.Load() {
				index := int(next.Add(1) - 1)
				if index >= len(variables) {
					return
				}
				id, err := facade.Subscribe(3, variables[index].Descriptor().ID.Name, 60_000)
				if err != nil {
					failed.Store(true)
					failures <- fmt.Errorf("subscription %d: %w; root=%+v consumer=%+v owner=%+v", index, err, root.Stats(), consumer.Stats(), owner.Stats())
					return
				}
				ids[index] = id
			}
		}()
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		for _, current := range nodes {
			for range 16 {
				select {
				case diagnostic := <-current.Errors():
					t.Logf("node %d: %v", current.ID(), diagnostic)
				default:
				}
			}
		}
		t.Fatal(err)
	}
	if owner.Stats().ActiveSubscriptions != 2000 {
		t.Fatalf("only %d owner subscriptions are active", owner.Stats().ActiveSubscriptions)
	}
	if consumer.Stats().PendingRequests != 2000 {
		t.Fatalf("consumer correlations: %d, want 2000", consumer.Stats().PendingRequests)
	}
	t.Logf("all subscriptions ready after %s", time.Since(started))
	rawIDs, _ := json.Marshal(ids)
	drain := func(want string) {
		t.Helper()
		seen := make(map[int64]bool)
		logged := false
		for len(seen) < 2000 {
			if err := ctx.Err(); err != nil {
				t.Fatalf("received %d/2000 values: %v", len(seen), err)
			}
			data, err := facade.PollSubscriptions(string(rawIDs), 100)
			if err != nil {
				t.Fatal(err)
			}
			var items []desktopbinding.SubscriptionPoll
			if err := json.Unmarshal([]byte(data), &items); err != nil {
				t.Fatal(err)
			}
			for _, item := range items {
				if !logged {
					t.Logf("first %s packet: %.1000s", want, item.Payload)
					logged = true
				}
				if item.Kind != "event" {
					t.Fatalf("received %d/2000 %s values; unexpected subscription result %s: %s", len(seen), want, item.Kind, item.Payload)
				}
				var event struct {
					Value protocol.VariableValueV1 `json:"value"`
				}
				if err := json.Unmarshal(item.Payload, &event); err != nil {
					t.Fatal(err)
				}
				if string(event.Value.Value) == want {
					seen[item.SubscriptionID] = true
				}
			}
		}
		t.Logf("all %s values drained after %s", want, time.Since(started))
	}
	drain("false")
	for _, variable := range variables {
		if _, err := variable.Set(resource.VariableState{Status: "ready", Value: json.RawMessage(`true`)}); err != nil {
			t.Fatal(err)
		}
	}
	drain("true")
	t.Logf("2000 TCP subscriptions and all initial/updated values drained in %s", time.Since(started))
	if err := facade.Close(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for owner.Stats().ActiveSubscriptions != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if owner.Stats().ActiveSubscriptions != 0 {
		t.Fatalf("subscriptions leaked after client close: %d", owner.Stats().ActiveSubscriptions)
	}
	if consumer.Stats().PendingRequests != 0 {
		t.Fatalf("consumer correlations leaked after close: %d", consumer.Stats().PendingRequests)
	}
}
