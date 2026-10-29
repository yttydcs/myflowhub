package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/examples/variables"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/sdk/bindings"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

type phaseALoadScope struct{ fixture *phaseAFixture }

func (s phaseALoadScope) MatchOwnerScope(scope protocol.PolicyOwnerScopeV1, owner protocol.NodeID) (bool, uint64, error) {
	if scope.Kind != "owner" {
		return false, 0, errors.New("local load fixture only supports exact owner scopes")
	}
	reachable, epoch, err := s.fixture.nodes[0].Tree().ScopeContains(owner, owner)
	return reachable && scope.NodeID == strconv.FormatUint(uint64(owner), 10), epoch, err
}

// Opt-in local real-TCP traffic exercises the production Desktop binding and
// renewals. Synthetic controls, one test process and no Wails UI mean this is
// deliberately not the formal two-device/private-commit/audio endurance pass.
func TestPhaseAReferenceTraffic(t *testing.T) {
	raw := os.Getenv("MFH_PHASE_A_LOCAL_LOAD_SECONDS")
	if raw == "" {
		t.Skip("set MFH_PHASE_A_LOCAL_LOAD_SECONDS=60 or longer for local load evidence")
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 60 || seconds > 86400 {
		t.Fatal("MFH_PHASE_A_LOCAL_LOAD_SECONDS must be in 60..86400")
	}
	duration := time.Duration(seconds) * time.Second
	var warmup time.Duration
	if raw := os.Getenv("MFH_PHASE_A_LOCAL_WARMUP_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds < 0 || seconds > 3600 {
			t.Fatal("MFH_PHASE_A_LOCAL_WARMUP_SECONDS must be in 0..3600")
		}
		warmup = time.Duration(seconds) * time.Second
	}
	f := newPhaseAFixture(t, warmup+duration+30*time.Second)
	owner := f.nodes[2]
	provider, err := variables.Register(owner.Registry(), owner.ID(), variables.ReferenceCount)
	if err != nil {
		t.Fatal(err)
	}
	extra, err := resource.NewVariable(resource.VariableDescriptor(protocol.ResourceID{Owner: 3, Name: "example/foundation/number-control"}, "test.read", 4096), protocol.DataSchemaDefinition{Type: "number"}, resource.VariableState{Status: "ready", Value: json.RawMessage(`20`)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.Registry().Register(extra); err != nil {
		t.Fatal(err)
	}
	if err := f.policy.BindScopeResolver(phaseALoadScope{f}); err != nil {
		t.Fatal(err)
	}
	_, err = f.policy.PutDefinition(protocol.PolicyDefinitionPutV1{Version: 1, ID: "test.phase-a-load", Label: "Synthetic local load", Rules: []protocol.PolicyRuleV1{{Resource: protocol.PolicyResourceSelectorV1{Kind: "prefix", Value: "example/foundation"}, Capability: protocol.PolicyCapabilitySelectorV1{Kind: "exact", Values: []protocol.CapabilityID{"read", "subscribe", "write"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.policy.CreateBinding(protocol.PolicyBindingCreateV1{Version: 1, BindingID: "aabbccddeeff00112233445566778899", Subject: "2", DefinitionID: "test.phase-a-load", Scope: protocol.PolicyOwnerScopeV1{Kind: "owner", NodeID: "3"}}, 1)
	if err != nil {
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
	resources := append(append([]*resource.Variable{}, provider.Variables...), f.variable, extra)
	ids := make([]int64, len(resources))
	var next atomic.Int64
	failures := make(chan error, 32)
	var setup sync.WaitGroup
	for range 32 {
		setup.Add(1)
		go func() {
			defer setup.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= len(resources) {
					return
				}
				id, err := facade.Subscribe(3, resources[i].Descriptor().ID.Name, 60000)
				if err != nil {
					failures <- err
					return
				}
				ids[i] = id
			}
		}()
	}
	setup.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	waitSubscriptionCount(t, owner, 2000)
	rawIDs, _ := json.Marshal(ids)
	sampleCtx, stopSamples := context.WithCancel(f.ctx)
	var sampleDone sync.WaitGroup
	var published atomic.Int64
	var cycles atomic.Int64
	sampleErrors := make(chan error, 1)
	sampleDone.Add(1)
	go func() {
		defer sampleDone.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		var tick uint64
		for {
			select {
			case <-sampleCtx.Done():
				return
			case <-ticker.C:
				tick++
				if err := provider.Sample(tick); err != nil {
					sampleErrors <- err
					return
				}
				for _, v := range []*resource.Variable{f.variable, extra} {
					before := v.Snapshot()
					if _, err := v.Apply(before.Revision+1, resource.VariableState{Status: "ready", Value: before.State.Value}); err != nil {
						sampleErrors <- err
						return
					}
				}
				cycles.Add(1)
				published.Store(time.Now().UnixNano())
			}
		}
	}()
	defer func() { stopSamples(); sampleDone.Wait() }()
	started := time.Now()
	warmupDeadline := started.Add(warmup)
	healthStarted := started
	healthPhase := warmup == 0
	deadline := warmupDeadline.Add(duration)
	nextCheckpoint := started.Add(time.Minute)
	t.Logf("LOAD_START unix_nano=%d resources=2000 warmup=%s measurement=%s; synthetic controls, real TCP, Desktop SDK; not Windows device acceptance", started.UnixNano(), warmup, duration)
	if healthPhase {
		t.Logf("HEALTH_START unix_nano=%d elapsed=%s", healthStarted.UnixNano(), time.Since(started))
	}
	counts := make(map[int64]int, len(ids))
	revisions := make(map[int64]string, len(ids))
	writes := 0
	var maxWrite time.Duration
	for time.Now().Before(deadline) {
		now := time.Now()
		if !healthPhase && !now.Before(warmupDeadline) {
			healthPhase = true
			healthStarted = now
			deadline = healthStarted.Add(duration)
			t.Logf("HEALTH_START unix_nano=%d elapsed=%s cycles=%d pending=%d", now.UnixNano(), now.Sub(started), cycles.Load(), f.nodes[1].Stats().PendingRequests)
		}
		select {
		case err := <-sampleErrors:
			t.Fatal(err)
		default:
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
			if item.Kind != "event" {
				t.Fatalf("unexpected binding result: %s %s", item.Kind, item.Payload)
			}
			var event struct {
				Kind  string                   `json:"kind"`
				Value protocol.VariableValueV1 `json:"value"`
			}
			if err := json.Unmarshal(item.Payload, &event); err != nil {
				t.Fatal(err)
			}
			if event.Kind == "gap" || event.Kind == "expired" {
				for index, current := range f.nodes {
					for range 32 {
						select {
						case err := <-current.Errors():
							t.Logf("node=%d diagnostic=%v", index+1, err)
						default:
						}
					}
				}
				consumerParent, _ := f.nodes[1].ParentStatus()
				ownerParent, _ := owner.ParentStatus()
				t.Fatalf("healthy run lost subscription continuity: payload=%s hub=%+v consumer=%+v owner=%+v consumer_parent=%+v owner_parent=%+v", item.Payload, f.nodes[0].Stats(), f.nodes[1].Stats(), owner.Stats(), consumerParent, ownerParent)
			}
			if event.Value.Revision != revisions[item.SubscriptionID] {
				counts[item.SubscriptionID]++
				revisions[item.SubscriptionID] = event.Value.Revision
			}
		}
		// Submit during a known post-sampling window, using a fresh read. The
		// producer keeps running; a conflict is reported, never silently retried.
		age := time.Since(time.Unix(0, published.Load()))
		if healthPhase && writes < 100 && time.Since(healthStarted) >= time.Duration(writes+1)*duration/102 && age > 100*time.Millisecond && age < 500*time.Millisecond {
			begin := time.Now()
			base, err := f.client.Snapshot(f.ctx, f.variable.Descriptor().ID)
			if err != nil {
				t.Fatal(err)
			}
			value := json.RawMessage(`true`)
			if writes%2 != 0 {
				value = json.RawMessage(`false`)
			}
			ack, err := f.client.WriteVariable(f.ctx, f.variable.Descriptor().ID, base.InstanceID, base.Revision, value)
			if err != nil {
				t.Fatalf("write %d: %v", writes+1, err)
			}
			actual, err := f.client.Snapshot(f.ctx, f.variable.Descriptor().ID)
			if err != nil || string(actual.Value) != string(value) || actual.InstanceID != ack.InstanceID {
				t.Fatalf("write %d readback: %s %v", writes+1, actual.Value, err)
			}
			elapsed := time.Since(begin)
			if elapsed > 2*time.Second {
				t.Fatalf("write %d took %s", writes+1, elapsed)
			}
			if elapsed > maxWrite {
				maxWrite = elapsed
			}
			writes++
			t.Logf("WRITE number=%d health_elapsed=%s read_write_read=%s", writes, time.Since(healthStarted), elapsed)
		}
		if now := time.Now(); !now.Before(nextCheckpoint) {
			phase := "warmup"
			var healthElapsed time.Duration
			if healthPhase {
				phase = "healthy"
				healthElapsed = now.Sub(healthStarted)
			}
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			samples, superseded := provider.Counts()
			t.Logf("CHECKPOINT phase=%s elapsed=%s health_elapsed=%s cycles=%d writes=%d provider_samples=%d superseded=%d owner_subscriptions=%d consumer_pending=%d goroutines=%d go_heap_bytes=%d go_sys_bytes=%d", phase, now.Sub(started), healthElapsed, cycles.Load(), writes, samples, superseded, owner.Stats().ActiveSubscriptions, f.nodes[1].Stats().PendingRequests, runtime.NumGoroutine(), memory.HeapAlloc, memory.Sys)
			nextCheckpoint = now.Add(time.Minute)
		}
	}
	stopSamples()
	sampleDone.Wait()
	pendingBefore := f.nodes[1].Stats().PendingRequests
	if pendingBefore != 2000 {
		t.Fatalf("consumer correlations accumulated during renewals: %d, want 2000", pendingBefore)
	}
	if writes != 100 {
		t.Fatalf("only %d/100 dispersed writes completed", writes)
	}
	minimum := int(cycles.Load()) - 2
	for _, id := range ids {
		if counts[id] < minimum {
			t.Fatalf("subscription %d only received %d revisions across %d cycles", id, counts[id], cycles.Load())
		}
	}
	if err := facade.Close(); err != nil {
		t.Fatal(err)
	}
	waitSubscriptionCount(t, f.nodes[0], 0)
	waitSubscriptionCount(t, owner, 0)
	if f.nodes[1].Stats().PendingRequests != 0 {
		t.Fatalf("pending correlations leaked after close: %d", f.nodes[1].Stats().PendingRequests)
	}
	t.Logf("LOCAL ONLY: resources=2000 duration=%s warmup=%s health_duration=%s cycles=%d writes=%d max_read_write_read=%s active_after_close=0", time.Since(started), warmup, time.Since(healthStarted), cycles.Load(), writes, maxWrite)
	t.Logf("provider_samples=%d; six-type owner resources and two synthetic controls; physical audio/private committed memory NOT measured", func() uint64 { n, _ := provider.Counts(); return n }())
	t.Logf("consumer_pending_before_close=%d after_close=%d", pendingBefore, f.nodes[1].Stats().PendingRequests)
}
