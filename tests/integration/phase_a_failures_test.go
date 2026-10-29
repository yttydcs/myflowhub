package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type faultTCP struct{ drop atomic.Bool }
type faultPipe struct {
	link.Pipe
	drop *atomic.Bool
}

func (p faultPipe) Read(data []byte) (int, error) {
	for {
		n, err := p.Pipe.Read(data)
		if err != nil || !p.drop.Load() {
			return n, err
		}
	}
}
func (p faultPipe) Write(data []byte) (int, error) {
	if p.drop.Load() {
		return len(data), nil
	}
	return p.Pipe.Write(data)
}
func (d *faultTCP) Dial(ctx context.Context, endpoint link.Endpoint) (link.Pipe, error) {
	pipe, err := (tcp.Driver{}).Dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return faultPipe{Pipe: pipe, drop: &d.drop}, nil
}
func (d *faultTCP) Listen(ctx context.Context, endpoint link.Endpoint) (link.Listener, error) {
	return (tcp.Driver{}).Listen(ctx, endpoint)
}

type phaseAFixture struct {
	ctx        context.Context
	nodes      [3]*node.Node
	parents    [2]*node.ParentSupervisor
	drivers    [2]*faultTCP
	client     *sdk.Client
	policy     *auth.PolicyState
	variable   *resource.Variable
	effects    atomic.Int64
	identities []auth.Identity
}

func newPhaseAFixture(t *testing.T, budgets ...time.Duration) *phaseAFixture {
	t.Helper()
	budget := 90 * time.Second
	if len(budgets) > 0 {
		budget = budgets[0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	t.Cleanup(cancel)
	f := &phaseAFixture{ctx: ctx}
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f.policy, err = auth.LoadPolicyState(store)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.policy.Close() })
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
	f.identities = identities
	mechanics, err := nodehost.ProductRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	for i := range f.nodes {
		var policy auth.Policy = auth.AllowAll{}
		if i == 0 {
			policy = f.policy
		}
		f.nodes[i], err = node.New(ctx, node.Config{Identity: identities[i], Trust: trust, Policy: policy, Session: mechanics.Session, Subscriptions: mechanics.Subscriptions, MaxPending: mechanics.MaxPending})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.nodes[i].Close() })
	}
	f.variable = f.newVariable(t, false)
	if err := f.nodes[2].Registry().Register(f.variable); err != nil {
		t.Fatal(err)
	}
	for _, capability := range []protocol.CapabilityID{protocol.CapabilityRead, protocol.CapabilitySubscribe, protocol.CapabilityWrite} {
		if err := f.policy.Grant(f.grant(capability)); err != nil {
			t.Fatal(err)
		}
	}
	endpoint, err := f.nodes[0].Listen(tcp.Driver{}, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	for i := range f.parents {
		f.drivers[i] = &faultTCP{}
		f.parents[i], err = f.nodes[i+1].SuperviseParent(ctx, f.drivers[i], endpoint, 1, nodehost.ProductSupervisorConfig(node.SupervisorConfig{}))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(f.parents[i].Stop)
	}
	f.client, err = sdk.NewAttachedClient(f.nodes[1])
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for f.parents[0].Snapshot().State != node.ConnectionConnected || f.parents[1].Snapshot().State != node.ConnectionConnected {
		if time.Now().After(deadline) {
			t.Fatal("fixture parents did not connect")
		}
		time.Sleep(time.Millisecond)
	}
	return f
}
func (f *phaseAFixture) grant(capability protocol.CapabilityID) auth.Request {
	return auth.Request{Subject: 2, Action: auth.Action(capability), Capability: capability, Resource: protocol.ResourceID{Owner: 3, Name: "test/phase-a/value"}}
}
func (f *phaseAFixture) newVariable(t *testing.T, value bool) *resource.Variable {
	t.Helper()
	encoded, _ := json.Marshal(value)
	variable, err := resource.NewVariable(resource.WritableVariableDescriptor(f.grant(protocol.CapabilityRead).Resource, "test.read", "test.write", 4096), protocol.DataSchemaDefinition{Type: "boolean"}, resource.VariableState{Status: "ready", Value: encoded}, func(ctx context.Context, input any) (resource.VariableState, error) {
		if err := ctx.Err(); err != nil {
			return resource.VariableState{}, err
		}
		f.effects.Add(1)
		encoded, err := json.Marshal(input)
		return resource.VariableState{Status: "ready", Value: encoded}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return variable
}
func durableKind(t *testing.T, subscription *sdk.DurableSubscription, kind sdk.EventKind, budget time.Duration) sdk.Event {
	t.Helper()
	timer := time.NewTimer(budget)
	defer timer.Stop()
	for {
		select {
		case event, ok := <-subscription.Events:
			if !ok {
				t.Fatal("durable events closed")
			}
			if event.Kind == kind {
				return event
			}
		case err := <-subscription.Errors:
			t.Fatalf("durable failed: %v", err)
		case <-timer.C:
			t.Fatalf("%s exceeded %s", kind, budget)
		}
	}
}

// Blackhole drops bytes while retaining real TCP sockets. It is deliberately
// separate from explicit peer close, and modifies no host firewall rules.
func TestPhaseANetworkPathsRecoverWithFiveFaultRepeats(t *testing.T) {
	for _, path := range []int{0, 1} {
		for _, blackhole := range []bool{false, true} {
			t.Run(fmt.Sprintf("path%d_blackhole%t", path, blackhole), func(t *testing.T) {
				f := newPhaseAFixture(t)
				remote, err := f.client.SubscribeDurable(f.ctx, f.parents[0], f.variable.Descriptor().ID, time.Minute, 8)
				if err != nil {
					t.Fatal(err)
				}
				defer remote.Cancel()
				durableKind(t, remote, sdk.EventSnapshot, 5*time.Second)
				for round := range 5 {
					beforeGeneration := f.parents[0].Snapshot().LinkGeneration
					started := time.Now()
					if blackhole {
						f.drivers[path].drop.Store(true)
					} else if err := f.nodes[0].DisconnectPeer(protocol.NodeID(path + 2)); err != nil {
						t.Fatal(err)
					}
					durableKind(t, remote, sdk.EventGap, 5*time.Second)
					detected := time.Since(started)
					f.drivers[path].drop.Store(false)
					reachable := time.Now()
					value := json.RawMessage(`false`)
					if round%2 == 0 {
						value = json.RawMessage(`true`)
					}
					if _, err := f.variable.Set(resource.VariableState{Status: "ready", Value: value}); err != nil {
						t.Fatal(err)
					}
					event := durableKind(t, remote, sdk.EventSnapshot, 10*time.Second)
					var snapshot protocol.VariableValueV1
					if err := json.Unmarshal(event.Value, &snapshot); err != nil || string(snapshot.Value) != string(value) {
						t.Fatalf("recovery did not deliver latest value: %s %v", event.Value, err)
					}
					if path == 1 && f.parents[0].Snapshot().LinkGeneration != beforeGeneration {
						t.Fatal("owner fault unexpectedly required consumer reconnect")
					}
					waitSubscriptionCount(t, f.nodes[0], 1)
					waitSubscriptionCount(t, f.nodes[2], 1)
					t.Logf("round %d detect=%s recover=%s", round+1, detected, time.Since(reachable))
				}
				remote.Cancel()
				waitSubscriptionCount(t, f.nodes[0], 0)
				waitSubscriptionCount(t, f.nodes[2], 0)
			})
		}
	}
}

func TestPhaseARevokeWritePreservesReadsThenStopsProtectedValues(t *testing.T) {
	f := newPhaseAFixture(t)
	id := f.variable.Descriptor().ID
	remote, err := f.client.Subscribe(f.ctx, id, time.Minute, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Cancel()
	receiveProductEvent(t, f.ctx, remote)
	if err := f.policy.Revoke(f.grant(protocol.CapabilityWrite)); err != nil {
		t.Fatal(err)
	}
	base := f.variable.Snapshot()
	if _, err := f.client.WriteVariable(f.ctx, id, base.State.InstanceID, base.State.Revision, json.RawMessage(`true`)); !isSDKCode(err, protocol.CodeForbidden) {
		t.Fatalf("revoked write: %v", err)
	}
	if f.effects.Load() != 0 {
		t.Fatal("rejected write had an effect")
	}
	if _, err := f.variable.Set(resource.VariableState{Status: "ready", Value: json.RawMessage(`true`)}); err != nil {
		t.Fatal(err)
	}
	if event := receiveProductEvent(t, f.ctx, remote); string(decodeVariableEvent(t, event, sdk.EventData).Value) != "true" {
		t.Fatal("write-only revoke interrupted subscription")
	}
	if _, err := f.client.Snapshot(f.ctx, id); err != nil {
		t.Fatal(err)
	}
	for _, capability := range []protocol.CapabilityID{protocol.CapabilityRead, protocol.CapabilitySubscribe} {
		if err := f.policy.Revoke(f.grant(capability)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.variable.Set(resource.VariableState{Status: "ready", Value: json.RawMessage(`false`)}); err != nil {
		t.Fatal(err)
	}
	if err := receiveProductError(t, f.ctx, remote); !isSDKCode(err, protocol.CodeExpired) {
		t.Fatalf("revoke did not invalidate subscription: %v", err)
	}
	for event := range remote.Events {
		if len(event.Value) > 0 {
			t.Fatalf("protected post-revoke value delivered: %s", event.Value)
		}
	}
	if _, err := f.client.Snapshot(f.ctx, id); !isSDKCode(err, protocol.CodeForbidden) {
		t.Fatalf("revoked read: %v", err)
	}
	waitSubscriptionCount(t, f.nodes[0], 0)
	waitSubscriptionCount(t, f.nodes[2], 0)
}

func TestPhaseAConcurrentCASAndReplacementTwentyRounds(t *testing.T) {
	f := newPhaseAFixture(t)
	id := f.variable.Descriptor().ID
	for round := range 20 {
		base := f.variable.Snapshot()
		effects := f.effects.Load()
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, value := range []string{"true", "false"} {
			go func(value string) {
				<-start
				_, err := f.client.WriteVariable(f.ctx, id, base.State.InstanceID, base.State.Revision, json.RawMessage(value))
				results <- err
			}(value)
		}
		close(start)
		success, conflict := 0, 0
		for range 2 {
			err := <-results
			if err == nil {
				success++
			} else if isSDKCode(err, protocol.CodeConflict) {
				conflict++
			} else {
				t.Fatal(err)
			}
		}
		if success != 1 || conflict != 1 || f.effects.Load()-effects != 1 {
			t.Fatalf("round %d: success=%d conflict=%d effects=%d", round, success, conflict, f.effects.Load()-effects)
		}
		old := f.variable.Snapshot()
		if err := f.nodes[2].Registry().Remove(id); err != nil {
			t.Fatal(err)
		}
		f.variable = f.newVariable(t, false)
		if err := f.nodes[2].Registry().Register(f.variable); err != nil {
			t.Fatal(err)
		}
		_, err := f.client.WriteVariable(f.ctx, id, old.State.InstanceID, old.State.Revision, json.RawMessage(`true`))
		var payload *sdk.Error
		if !errors.As(err, &payload) || payload.Code != protocol.CodeConflict || f.effects.Load()-effects != 1 {
			t.Fatalf("old-instance write crossed replacement: %v", err)
		}
	}
}
