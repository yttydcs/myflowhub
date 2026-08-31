package flow_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	flowfeature "github.com/yttydcs/myflowhub/feature/flow"
	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/transport/memory"
)

const (
	flowOne = "00112233445566778899aabbccddeeff"
	runOne  = "10112233445566778899aabbccddeeff"
	runTwo  = "20112233445566778899aabbccddeeff"
)

type environment struct {
	node       *node.Node
	store      *keystore.Store
	controller *flowfeature.Controller
}

func TestDefinitionLifecycleDedupeAndPersistence(t *testing.T) {
	env := newEnvironment(t, auth.NewStaticPolicy(), flowfeature.Config{})
	definition := transformDefinition(flowOne, 1, json.RawMessage(`{"answer":42}`))
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})
	request := protocol.FlowRunV1{Version: 1, RunID: runOne, FlowID: flowOne, FlowRevision: 1, DedupeKey: "once", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &request, &protocol.FlowRunSummaryV1{})
	run := waitRun(t, env.node, runOne, "succeeded")
	if run.Error != "" {
		t.Fatalf("transform flow failed: %#v", run)
	}
	request.RunID = runTwo
	deduped := operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &request, &protocol.FlowRunSummaryV1{}).(*protocol.FlowRunSummaryV1)
	if deduped.RunID != runOne {
		t.Fatalf("dedupe created another run: %#v", deduped)
	}
	updated := transformDefinition(flowOne, 2, json.RawMessage(`{"answer":43}`))
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowUpdate, &updated, &protocol.FlowDefinitionV1{})
	if err := env.controller.Close(); err != nil {
		t.Fatal(err)
	}
	_ = env.node.Close()

	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, identity.PublicKey)
	restartedNode, _ := node.New(context.Background(), node.Config{Identity: identity, Trust: trust})
	defer restartedNode.Close()
	restarted, err := flowfeature.Register(flowfeature.Config{Node: restartedNode, Store: env.store})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	definitions := definitionsSnapshot(t, restartedNode)
	if len(definitions.Members) != 1 || definitions.Members[0].Attributes["revision"] != "2" {
		t.Fatalf("flow definition did not survive restart: %#v", definitions)
	}
	archive := protocol.FlowArchiveV1{Version: 1, FlowID: flowOne}
	operate(t, restartedNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowArchive, &archive, &protocol.FlowArchiveV1{})
	if got := definitionsSnapshot(t, restartedNode); len(got.Members) != 0 {
		t.Fatalf("archived flow remained visible: %#v", got)
	}
}

func TestRunConcurrencyCancelAndOutputLimits(t *testing.T) {
	policy := auth.NewStaticPolicy()
	env := newEnvironment(t, policy, flowfeature.Config{MaxActive: 1, MaxActivePerFlow: 1, MaxRuns: 4, MaxNodeOutput: 10, MaxTotalOutput: 20})
	blockID := protocol.ResourceID{Owner: 1, Name: "test/block"}
	block, _ := resource.NewCommand(resource.CommandDescriptor(blockID, "application/octet-stream", "test.raw.v1", "test.invoke", 128), func(ctx context.Context, _ []byte) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err := env.node.Registry().Register(block); err != nil {
		t.Fatal(err)
	}
	definition := protocol.FlowDefinitionV1{Version: 1, FlowID: flowOne, Revision: 1, Name: "blocking", Nodes: []protocol.FlowNodeV1{{ID: "block", Kind: "command-call", Resource: protocol.FlowResourceRefV1{OwnerNodeID: "1", Name: blockID.Name}, Config: json.RawMessage(`{}`)}}}
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})
	first := protocol.FlowRunV1{Version: 1, RunID: runOne, FlowID: flowOne, FlowRevision: 1, DedupeKey: "a", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &first, &protocol.FlowRunSummaryV1{})
	waitRun(t, env.node, runOne, "running")
	second := protocol.FlowRunV1{Version: 1, RunID: runTwo, FlowID: flowOne, FlowRevision: 1, DedupeKey: "b", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	if _, err := operateError(env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &second); err == nil {
		t.Fatal("flow concurrency limit was bypassed")
	}
	archive := protocol.FlowArchiveV1{Version: 1, FlowID: flowOne}
	if _, err := operateError(env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowArchive, &archive); err == nil {
		t.Fatal("active flow definition was archived")
	}
	cancel := protocol.FlowCancelV1{Version: 1, RunID: runOne, Reason: "test"}
	result := operate(t, env.node, protocol.BuiltinFlowRuns, protocol.CapabilityFlowCancel, &cancel, &protocol.FlowRunSummaryV1{}).(*protocol.FlowRunSummaryV1)
	if result.State != "cancelled" {
		t.Fatalf("run was not cancelled: %#v", result)
	}
	idempotent := operate(t, env.node, protocol.BuiltinFlowRuns, protocol.CapabilityFlowCancel, &cancel, &protocol.FlowRunSummaryV1{}).(*protocol.FlowRunSummaryV1)
	if !reflect.DeepEqual(idempotent, result) {
		t.Fatalf("terminal cancel was not idempotent: first=%#v second=%#v", result, idempotent)
	}

	overflowID := "30112233445566778899aabbccddeeff"
	overflow := transformDefinition(overflowID, 1, json.RawMessage(`{"value":"this is too large"}`))
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &overflow, &protocol.FlowDefinitionV1{})
	overflowRun := protocol.FlowRunV1{Version: 1, RunID: runTwo, FlowID: overflowID, FlowRevision: 1, DedupeKey: "overflow", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &overflowRun, &protocol.FlowRunSummaryV1{})
	failed := waitRun(t, env.node, runTwo, "failed")
	if failed.Error == "" {
		t.Fatal("output overflow did not retain an actionable error")
	}
}

func TestDelegatedRunUsesInitiatorPermission(t *testing.T) {
	rootIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, rootIdentity.PublicKey)
	_ = trust.Add(2, childIdentity.PublicKey)
	policy := auth.NewStaticPolicy()
	flowRunResource := protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFlowDefinitions}
	policy.Allow(auth.Request{Subject: 2, Action: auth.Action(protocol.CapabilityFlowRun), Resource: flowRunResource})
	rootNode, _ := node.New(context.Background(), node.Config{Identity: rootIdentity, Trust: trust, Policy: policy})
	defer rootNode.Close()
	store, _ := keystore.New(t.TempDir())
	controller, err := flowfeature.Register(flowfeature.Config{Node: rootNode, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	called := make(chan struct{}, 1)
	targetID := protocol.ResourceID{Owner: 1, Name: "test/protected"}
	target, _ := resource.NewCommand(resource.CommandDescriptor(targetID, "application/octet-stream", "test.raw.v1", "test.invoke", 128), func(context.Context, []byte) ([]byte, error) {
		called <- struct{}{}
		return []byte(`{"ok":true}`), nil
	})
	if err := rootNode.Registry().Register(target); err != nil {
		t.Fatal(err)
	}
	definition := protocol.FlowDefinitionV1{Version: 1, FlowID: flowOne, Revision: 1, Name: "delegated", Nodes: []protocol.FlowNodeV1{{ID: "call", Kind: "command-call", Resource: protocol.FlowResourceRefV1{OwnerNodeID: "1", Name: targetID.Name}, Config: json.RawMessage(`{}`)}}}
	operate(t, rootNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})
	network := memory.NewNetwork()
	defer network.Close()
	if _, err := rootNode.Listen(network, "root"); err != nil {
		t.Fatal(err)
	}
	childTrust := auth.NewTrustStore()
	_ = childTrust.Add(1, rootIdentity.PublicKey)
	childNode, _ := node.New(context.Background(), node.Config{Identity: childIdentity, Trust: childTrust})
	defer childNode.Close()
	if err := childNode.ConnectParent(context.Background(), network, "root", 1); err != nil {
		t.Fatal(err)
	}
	deniedRun := protocol.FlowRunV1{Version: 1, RunID: runOne, FlowID: flowOne, FlowRevision: 1, DedupeKey: "denied", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, childNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &deniedRun, &protocol.FlowRunSummaryV1{})
	failed := waitRun(t, rootNode, runOne, "failed")
	if failed.Error == "" {
		t.Fatal("delegated target denial was not recorded")
	}
	select {
	case <-called:
		t.Fatal("Flow host authority bypassed the initiator policy")
	default:
	}
	policy.Allow(auth.Request{Subject: 2, Action: auth.ActionInvoke, Resource: targetID})
	allowedRun := protocol.FlowRunV1{Version: 1, RunID: runTwo, FlowID: flowOne, FlowRevision: 1, DedupeKey: "allowed", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, childNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &allowedRun, &protocol.FlowRunSummaryV1{})
	waitRun(t, rootNode, runTwo, "succeeded")
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("allowed delegated target was not invoked")
	}
}

func TestRetryableCommandUsesBoundedBackoff(t *testing.T) {
	env := newEnvironment(t, auth.NewStaticPolicy(), flowfeature.Config{})
	var attempts atomic.Int32
	targetID := protocol.ResourceID{Owner: 1, Name: "test/flaky"}
	target, _ := resource.NewCommand(resource.CommandDescriptor(targetID, "application/octet-stream", "test.raw.v1", "test.invoke", 128), func(context.Context, []byte) ([]byte, error) {
		if attempts.Add(1) < 3 {
			return nil, command.Retryable(errors.New("temporary"))
		}
		return []byte(`{"ok":true}`), nil
	})
	if err := env.node.Registry().Register(target); err != nil {
		t.Fatal(err)
	}
	definition := protocol.FlowDefinitionV1{Version: 1, FlowID: flowOne, Revision: 1, Name: "retry", Nodes: []protocol.FlowNodeV1{{ID: "flaky", Kind: "command-call", Resource: protocol.FlowResourceRefV1{OwnerNodeID: "1", Name: targetID.Name}, Config: json.RawMessage(`{}`), MaxAttempts: 3, RetryBackoffMS: 10}}}
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})
	run := protocol.FlowRunV1{Version: 1, RunID: runOne, FlowID: flowOne, FlowRevision: 1, DedupeKey: "retry", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &run, &protocol.FlowRunSummaryV1{})
	waitRun(t, env.node, runOne, "succeeded")
	if attempts.Load() != 3 {
		t.Fatalf("unexpected retry count: %d", attempts.Load())
	}
}

func TestRestartMarksPersistedActiveRunInterrupted(t *testing.T) {
	store, _ := keystore.New(t.TempDir())
	definition := transformDefinition(flowOne, 1, json.RawMessage(`{}`))
	run := protocol.FlowRunSummaryV1{RunID: runOne, FlowID: flowOne, FlowRevision: 1, Initiator: "1", DedupeKey: "restart", State: "running", StartedUnixMS: time.Now().Add(-time.Minute).UnixMilli()}
	state := struct {
		Version     int                         `json:"version"`
		Revision    uint64                      `json:"revision"`
		Definitions []protocol.FlowDefinitionV1 `json:"definitions"`
		Runs        []protocol.FlowRunSummaryV1 `json:"runs"`
	}{1, 2, []protocol.FlowDefinitionV1{definition}, []protocol.FlowRunSummaryV1{run}}
	if err := store.Save("flow.json", state); err != nil {
		t.Fatal(err)
	}
	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, identity.PublicKey)
	runtime, _ := node.New(context.Background(), node.Config{Identity: identity, Trust: trust})
	defer runtime.Close()
	controller, err := flowfeature.Register(flowfeature.Config{Node: runtime, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	interrupted := waitRun(t, runtime, runOne, "interrupted")
	if interrupted.FinishedUnixMS == 0 || interrupted.Error == "" {
		t.Fatalf("active run was not safely interrupted on restart: %#v", interrupted)
	}
}

func TestCollectionRevisionRejectsUnsafeStateAndExhaustsAtJSONLimit(t *testing.T) {
	newRuntime := func(t *testing.T) (*node.Node, *keystore.Store) {
		t.Helper()
		store, err := keystore.New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		identity, err := auth.GenerateIdentity(1)
		if err != nil {
			t.Fatal(err)
		}
		trust := auth.NewTrustStore()
		if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
			t.Fatal(err)
		}
		runtimeNode, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = runtimeNode.Close() })
		return runtimeNode, store
	}
	saveState := func(t *testing.T, store *keystore.Store, revision uint64) {
		t.Helper()
		state := struct {
			Version     int                         `json:"version"`
			Revision    uint64                      `json:"revision"`
			Definitions []protocol.FlowDefinitionV1 `json:"definitions"`
			Runs        []protocol.FlowRunSummaryV1 `json:"runs"`
		}{1, revision, []protocol.FlowDefinitionV1{}, []protocol.FlowRunSummaryV1{}}
		if err := store.Save("flow.json", state); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("unsafe-persisted-revision", func(t *testing.T) {
		runtimeNode, store := newRuntime(t)
		saveState(t, store, protocol.MaxCollectionRevision+1)
		if controller, err := flowfeature.Register(flowfeature.Config{Node: runtimeNode, Store: store}); err == nil {
			_ = controller.Close()
			t.Fatal("flow accepted a collection revision that cannot round-trip through JavaScript")
		}
	})

	t.Run("safe-limit-does-not-wrap", func(t *testing.T) {
		runtimeNode, store := newRuntime(t)
		saveState(t, store, protocol.MaxCollectionRevision)
		controller, err := flowfeature.Register(flowfeature.Config{Node: runtimeNode, Store: store})
		if err != nil {
			t.Fatal(err)
		}
		defer controller.Close()
		if page := definitionsSnapshot(t, runtimeNode); page.Revision != protocol.MaxCollectionRevision {
			t.Fatalf("collection page revision = %d", page.Revision)
		}
		definition := transformDefinition(flowOne, 1, json.RawMessage(`{}`))
		if _, err := operateError(runtimeNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition); err == nil {
			t.Fatal("flow collection revision wrapped past the JSON-safe limit")
		}
		if page := definitionsSnapshot(t, runtimeNode); page.Revision != protocol.MaxCollectionRevision || len(page.Members) != 0 {
			t.Fatalf("failed mutation changed exhausted flow state: %#v", page)
		}
	})
}

func newEnvironment(t *testing.T, policy auth.Policy, override flowfeature.Config) environment {
	t.Helper()
	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, identity.PublicKey)
	runtime, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	override.Node = runtime
	override.Store = store
	controller, err := flowfeature.Register(override)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = controller.Close()
		_ = runtime.Close()
	})
	return environment{node: runtime, store: store, controller: controller}
}

func transformDefinition(id string, revision uint64, config json.RawMessage) protocol.FlowDefinitionV1 {
	return protocol.FlowDefinitionV1{Version: 1, FlowID: id, Revision: revision, Name: "transform", Nodes: []protocol.FlowNodeV1{{ID: "transform", Kind: "transform", Config: config}}}
}

func operate(t *testing.T, runtime *node.Node, name string, capability protocol.CapabilityID, request protocol.ValidatedPayload, response protocol.ValidatedPayload) protocol.ValidatedPayload {
	t.Helper()
	data, err := operateError(runtime, name, capability, request)
	if err != nil {
		t.Fatal(err)
	}
	if err := protocol.DecodeJSONPayload(data, protocol.DefaultMaxPayload, response); err != nil {
		t.Fatal(err)
	}
	return response
}

func operateError(runtime *node.Node, name string, capability protocol.CapabilityID, request protocol.ValidatedPayload) ([]byte, error) {
	payload, err := protocol.EncodeJSONPayload(request, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := runtime.Operate(ctx, protocol.ResourceID{Owner: 1, Name: name}, capability, "", payload)
	return result.Payload, err
}

func waitRun(t *testing.T, runtime *node.Node, runID, state string) protocol.FlowRunSummaryV1 {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var last protocol.FlowRunSummaryV1
	for time.Now().Before(deadline) {
		request := protocol.CollectionMemberRequestV1{Version: 1, Key: runID}
		payload, err := operateError(runtime, protocol.BuiltinFlowRuns, protocol.CapabilityGet, &request)
		if err == nil {
			if err := protocol.DecodeJSONPayload(payload, protocol.DefaultMaxPayload, &last); err != nil {
				t.Fatal(err)
			}
			if last.State == state {
				return last
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("run %s did not reach %s; last state: %#v", runID, state, last)
	return protocol.FlowRunSummaryV1{}
}

func definitionsSnapshot(t *testing.T, runtime *node.Node) protocol.CollectionPageV1 {
	t.Helper()
	request := protocol.CollectionListRequestV1{Version: 1, Limit: protocol.MaxCollectionPageMembers}
	return *operate(t, runtime, protocol.BuiltinFlowDefinitions, protocol.CapabilityList, &request, &protocol.CollectionPageV1{}).(*protocol.CollectionPageV1)
}
