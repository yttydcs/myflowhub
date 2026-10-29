package integration_test

import (
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
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func TestPhaseAHubRestartRetains2000DurableSubscriptions(t *testing.T) {
	f := newPhaseAFixture(t, 150*time.Second)
	owner := f.nodes[2]
	if err := f.policy.BindScopeResolver(phaseALoadScope{f}); err != nil {
		t.Fatal(err)
	}
	_, err := f.policy.PutDefinition(protocol.PolicyDefinitionPutV1{Version: 1, ID: "test.phase-a-restart", Label: "Restart load", Rules: []protocol.PolicyRuleV1{{Resource: protocol.PolicyResourceSelectorV1{Kind: "prefix", Value: "load"}, Capability: protocol.PolicyCapabilitySelectorV1{Kind: "exact", Values: []protocol.CapabilityID{"read", "subscribe"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.policy.CreateBinding(protocol.PolicyBindingCreateV1{Version: 1, BindingID: "aabbccddeeff00112233445566778899", Subject: "2", DefinitionID: "test.phase-a-restart", Scope: protocol.PolicyOwnerScopeV1{Kind: "owner", NodeID: "3"}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	variables := make([]*resource.Variable, 2000)
	registered := make([]resource.Resource, len(variables))
	for i := range variables {
		variables[i], err = resource.NewVariable(resource.VariableDescriptor(protocol.ResourceID{Owner: 3, Name: fmt.Sprintf("load/%04d", i)}, "test.read", 4096), protocol.DataSchemaDefinition{Type: "integer"}, resource.VariableState{Status: "ready", Value: json.RawMessage(`0`)}, nil)
		if err != nil {
			t.Fatal(err)
		}
		registered[i] = variables[i]
	}
	if err := owner.Registry().RegisterBatch(registered); err != nil {
		t.Fatal(err)
	}
	core, err := bindings.NewAttachedClient(f.client, bindings.PublicIdentity{NodeID: 2, PublicKey: f.identities[1].PublicKey}, capacityConnection{f.parents[0]})
	if err != nil {
		t.Fatal(err)
	}
	facade, err := desktopbinding.NewAttachedClient(core)
	if err != nil {
		t.Fatal(err)
	}
	defer facade.Close()
	ids := make([]int64, len(variables))
	var next atomic.Int64
	var workers sync.WaitGroup
	failures := make(chan error, 32)
	for range 32 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= len(ids) {
					return
				}
				id, err := facade.Subscribe(3, variables[i].Descriptor().ID.Name, 60000)
				if err != nil {
					failures <- err
					return
				}
				ids[i] = id
			}
		}()
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	rawIDs, _ := json.Marshal(ids)
	drain := func(round int) {
		t.Helper()
		seen := make(map[int64]bool, len(ids))
		deadline := time.Now().Add(15 * time.Second)
		for len(seen) < len(ids) {
			if time.Now().After(deadline) {
				t.Fatalf("round %d recovered %d/%d; root=%+v owner=%+v consumer=%+v", round, len(seen), len(ids), f.nodes[0].Stats(), owner.Stats(), f.nodes[1].Stats())
			}
			raw, err := facade.PollSubscriptions(string(rawIDs), 100)
			if err != nil {
				t.Fatal(err)
			}
			var items []desktopbinding.SubscriptionPoll
			if err := json.Unmarshal([]byte(raw), &items); err != nil {
				t.Fatal(err)
			}
			for _, item := range items {
				if item.Kind != "event" {
					t.Fatalf("round %d subscription %d terminal %s: %s", round, item.SubscriptionID, item.Kind, item.Payload)
				}
				var event struct {
					Kind  string                   `json:"kind"`
					Value protocol.VariableValueV1 `json:"value"`
				}
				if err := json.Unmarshal(item.Payload, &event); err != nil {
					t.Fatal(err)
				}
				if event.Kind == "snapshot" && string(event.Value.Value) == fmt.Sprint(round) {
					seen[item.SubscriptionID] = true
				}
			}
		}
		t.Logf("round %d: all 2000 current snapshots received", round)
	}
	drain(0)
	endpoint := f.parents[0].Snapshot().Endpoint
	trust := auth.NewTrustStore()
	for _, identity := range f.identities {
		if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
			t.Fatal(err)
		}
	}
	mechanics, err := nodehost.ProductRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	for round := 1; round <= 10; round++ {
		if err := f.nodes[0].Close(); err != nil {
			t.Fatal(err)
		}
		for _, variable := range variables {
			if _, err := variable.Set(resource.VariableState{Status: "ready", Value: json.RawMessage(fmt.Sprint(round))}); err != nil {
				t.Fatal(err)
			}
		}
		f.nodes[0], err = node.New(f.ctx, node.Config{Identity: f.identities[0], Trust: trust, Policy: f.policy, Session: mechanics.Session, Subscriptions: mechanics.Subscriptions, MaxPending: mechanics.MaxPending})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.nodes[0].Listen(tcp.Driver{}, endpoint); err != nil {
			t.Fatal(err)
		}
		drain(round)
	}
}
