package management_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

func topologyQuery(t *testing.T, value fixture, depth int) (protocol.ManagementTopologyQueryV1, []byte) {
	t.Helper()
	capability := protocol.CapabilitySubtree
	if depth == 1 {
		capability = protocol.CapabilityChildren
	}
	result, err := value.node.Registry().Operate(context.Background(), protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinManagementTopology}, resource.OperationRequest{
		Capability: capability, Payload: []byte(fmt.Sprintf(`{"version":1,"depth":%d}`, depth)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Schema != protocol.SchemaManagementTopologyQueryV1 {
		t.Fatalf("wrong query schema: %s", result.Schema)
	}
	var decoded protocol.ManagementTopologyQueryV1
	decode(t, result.Payload, &decoded)
	return decoded, result.Payload
}

func TestTopologyDepthQueriesAndRefreshInvalidation(t *testing.T) {
	value := newFixture(t)
	tree := value.node.Tree()
	if err := tree.AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	if err := tree.AnnounceWithParent(2, 3, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err := tree.AnnounceWithParent(2, 4, 3, 1); err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	for depth, count := range map[int]int{0: 4, 1: 2, 2: 3, 3: 4, 4096: 4} {
		query, payload := topologyQuery(t, value, depth)
		if query.Depth != depth || query.RootNodeID != "1" || len(query.Nodes) != count || query.Nodes[0].ParentID != "" {
			t.Fatalf("wrong depth %d result: %+v", depth, query)
		}
		_, cached := topologyQuery(t, value, depth)
		if !bytes.Equal(payload, cached) {
			t.Fatal("unchanged query changed bytes")
		}
	}
	first, _ := topologyQuery(t, value, 1)
	if !first.Nodes[1].HasChildren {
		t.Fatal("boundary node lost its children hint")
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	unchanged, _ := topologyQuery(t, value, 1)
	if first.Revision != unchanged.Revision || first.InstanceID != unchanged.InstanceID {
		t.Fatal("unchanged refresh advanced snapshot")
	}
	oldEpoch := tree.Epoch()
	if err := tree.AnnounceWithParent(2, 5, 4, 1); err != nil {
		t.Fatal(err)
	}
	if tree.Epoch() != oldEpoch {
		t.Fatal("fixture must exercise descendant change without Epoch change")
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	changed, _ := topologyQuery(t, value, 0)
	if changed.Revision <= first.Revision || len(changed.Nodes) != 5 {
		t.Fatal("membership change reused old snapshot")
	}
	settings := value.state.Settings.Snapshot()
	if _, err := value.state.Settings.Replace(protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: settings.Revision, DisplayName: "renamed", Values: settings.Values}); err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	renamed, _ := topologyQuery(t, value, 1)
	if renamed.Revision <= changed.Revision || renamed.Nodes[0].DisplayName != "renamed" {
		t.Fatal("display name failed to invalidate snapshot")
	}
	if _, err := tree.AttachParent(9); err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	relay, _ := topologyQuery(t, value, 0)
	if relay.Nodes[0].Role != "node" || relay.Nodes[0].ParentID != "" {
		t.Fatal("query escaped local root boundary")
	}
	tree.WithdrawChild(2)
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	removed, _ := topologyQuery(t, value, 1)
	if len(removed.Nodes) != 1 || removed.Nodes[0].HasChildren {
		t.Fatal("removed branch remained cached")
	}
	other := newFixture(t)
	restarted, _ := topologyQuery(t, other, 1)
	if restarted.InstanceID == first.InstanceID {
		t.Fatal("new provider reused instance")
	}
}

func TestTopologyChildrenRejectsForgedDepthAndMissingInput(t *testing.T) {
	value := newFixture(t)
	for _, payload := range []string{`{"version":1,"depth":0}`, `{"version":1,"depth":2}`, `{"version":1}`, `{"version":1,"depth":null}`, `{"version":1,"depth":1,"root_node_id":"2"}`} {
		_, err := value.node.Registry().Operate(context.Background(), protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology}, resource.OperationRequest{Capability: protocol.CapabilityChildren, Payload: []byte(payload)})
		if !errors.Is(err, protocol.ErrInvalidPayload) {
			t.Fatalf("forged request accepted: %s (%v)", payload, err)
		}
	}
}

func TestTopologyOverflowDoesNotDisableShallowQuery(t *testing.T) {
	value := newFixture(t)
	if err := value.node.Tree().AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	for id := protocol.NodeID(3); id <= protocol.MaxItems+1; id++ {
		if err := value.node.Tree().AnnounceWithParent(2, id, 2, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := value.controller.Refresh(); !errors.Is(err, protocol.ErrPayloadTooLarge) {
		t.Fatalf("missing overflow: %v", err)
	}
	shallow, _ := topologyQuery(t, value, 1)
	if len(shallow.Nodes) != 2 || !shallow.Nodes[1].HasChildren {
		t.Fatal("full overflow disabled shallow discovery")
	}
	for _, capability := range []protocol.CapabilityID{protocol.CapabilitySubtree, protocol.CapabilityRead} {
		_, err := value.node.Registry().Operate(context.Background(), protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology}, resource.OperationRequest{Capability: capability, Payload: []byte(`{"version":1,"depth":0}`)})
		if !errors.Is(err, protocol.ErrPayloadTooLarge) {
			t.Fatalf("%s silently returned partial/stale data: %v", capability, err)
		}
	}
	registered, _ := value.node.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology})
	if _, cancel, err := registered.(resource.Observable).Observe(func(resource.Observation) {}); !errors.Is(err, protocol.ErrPayloadTooLarge) {
		if cancel != nil {
			cancel()
		}
		t.Fatalf("new legacy observer received stale full snapshot: %v", err)
	}
	var health protocol.ManagementHealthV1
	decodeVariable(t, value.node, protocol.BuiltinManagementHealth, &health)
	if health.LastError == "" {
		t.Fatal("refresh overflow was not visible in health")
	}
	value.node.Tree().WithdrawChild(2)
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	var legacy protocol.ManagementTopologyV1
	decodeVariable(t, value.node, protocol.BuiltinManagementTopology, &legacy)
	if len(legacy.Nodes) != 1 {
		t.Fatal("legacy snapshot did not recover")
	}
	health = protocol.ManagementHealthV1{}
	decodeVariable(t, value.node, protocol.BuiltinManagementHealth, &health)
	if health.LastError != "" {
		t.Fatalf("recovered health retains error: %s", health.LastError)
	}
}

func TestTopologyIncompleteWithdrawalFailsUntilRelationsRecover(t *testing.T) {
	value := newFixture(t)
	tree := value.node.Tree()
	if err := tree.AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	if err := tree.AnnounceWithParent(2, 3, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err := tree.AnnounceWithParent(2, 4, 3, 1); err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	if err := tree.WithdrawRoute(2, 3, 1); err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err == nil {
		t.Fatal("orphan relationship accepted")
	}
	_, err := value.node.Registry().Operate(context.Background(), protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology}, resource.OperationRequest{Capability: protocol.CapabilityChildren, Payload: []byte(`{"version":1,"depth":1}`)})
	if err == nil {
		t.Fatal("inconsistent tree silently returned an old snapshot")
	}
	if err := tree.WithdrawRoute(2, 4, 1); err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	query, _ := topologyQuery(t, value, 1)
	if len(query.Nodes) != 2 || query.Nodes[1].HasChildren {
		t.Fatal("recovered relation failed to publish")
	}
}

func TestTopologyObserveAllowsReentrantRefreshAndConcurrentQueries(t *testing.T) {
	value := newFixture(t)
	registered, _ := value.node.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology})
	observed := make(chan error, 8)
	initial, cancel, err := registered.(resource.Observable).Observe(func(resource.Observation) { observed <- value.controller.Refresh() })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if initial == nil || !initial.Snapshot {
		t.Fatal("legacy initial snapshot missing")
	}
	if err := value.node.Tree().AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- value.controller.Refresh() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("topology callback held controller lock")
	}
	select {
	case err := <-observed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("legacy observer missed refresh")
	}
	var workers sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for iteration := 0; iteration < 30; iteration++ {
				topologyQuery(t, value, iteration%5)
				if err := value.controller.Refresh(); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	workers.Wait()
}

func TestTopologyReentrantMutationPublishesInOrderToEveryObserver(t *testing.T) {
	value := newFixture(t)
	registered, _ := value.node.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology})
	var changed atomic.Bool
	var mu sync.Mutex
	seen := [2][]uint64{}
	for index := range seen {
		_, cancel, err := registered.(resource.Observable).Observe(func(observation resource.Observation) {
			mu.Lock()
			seen[index] = append(seen[index], observation.Revision)
			mu.Unlock()
			if changed.CompareAndSwap(false, true) {
				if err := value.node.Tree().AttachChild(3, 1); err != nil {
					t.Error(err)
					return
				}
				if err := value.controller.Refresh(); err != nil {
					t.Error(err)
				}
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		defer cancel()
	}
	if err := value.node.Tree().AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- value.controller.Refresh() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reentrant publication deadlocked")
	}
	mu.Lock()
	defer mu.Unlock()
	for index, revisions := range seen {
		if len(revisions) != 2 || revisions[1] <= revisions[0] {
			t.Fatalf("observer %d saw out-of-order revisions: %v", index, revisions)
		}
	}
}

func TestTopologyExistingSubscriptionFailsOnOverflowAndCanRecover(t *testing.T) {
	value := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := sdk.NewAttachedClient(value.node)
	if err != nil {
		t.Fatal(err)
	}
	id := protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology}
	stream, err := client.Subscribe(ctx, id, time.Minute, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Cancel()
	select {
	case <-stream.Events:
	case <-ctx.Done():
		t.Fatal("missing initial subscription snapshot")
	}
	if err := value.node.Tree().AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	for id := protocol.NodeID(3); id <= protocol.MaxItems+1; id++ {
		if err := value.node.Tree().AnnounceWithParent(2, id, 2, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := value.controller.Refresh(); !errors.Is(err, protocol.ErrPayloadTooLarge) {
		t.Fatalf("missing overflow: %v", err)
	}
	select {
	case err := <-stream.Errors:
		var failure *sdk.Error
		if !errors.As(err, &failure) || failure.Code != protocol.CodeOverflow {
			t.Fatalf("expected terminal overflow, got %v", err)
		}
	case <-ctx.Done():
		t.Fatal("existing subscription silently retained stale full snapshot")
	}
	value.node.Tree().WithdrawChild(2)
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	recovered, err := client.Subscribe(ctx, id, time.Minute, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Cancel()
	select {
	case event := <-recovered.Events:
		var topology protocol.ManagementTopologyV1
		if err := sdk.DecodeEvent(event, &topology); err != nil {
			t.Fatal(err)
		}
		if len(topology.Nodes) != 1 {
			t.Fatal("recovered subscriber saw stale nodes")
		}
	case <-ctx.Done():
		t.Fatal("recovered subscriber did not receive a snapshot")
	}
}
