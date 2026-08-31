package flow_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	flowfeature "github.com/yttydcs/myflowhub/feature/flow"
	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/transport/memory"
)

const (
	flowTwo   = "40112233445566778899aabbccddeeff"
	flowThree = "50112233445566778899aabbccddeeff"
	flowFour  = "60112233445566778899aabbccddeeff"
	runThree  = "70112233445566778899aabbccddeeff"
)

func TestFlowCatalogContainsOnlyCollections(t *testing.T) {
	env := newEnvironment(t, auth.NewStaticPolicy(), flowfeature.Config{})
	var flowDescriptors []protocol.ResourceDescriptorV2
	for _, descriptor := range env.node.Registry().List() {
		if strings.HasPrefix(descriptor.ID.Name, "flow/") {
			flowDescriptors = append(flowDescriptors, descriptor)
		}
	}
	if len(flowDescriptors) != 2 {
		t.Fatalf("catalog has unexpected Flow resources: %#v", flowDescriptors)
	}
	if flowDescriptors[0].ID.Name != protocol.BuiltinFlowDefinitions || flowDescriptors[1].ID.Name != protocol.BuiltinFlowRuns {
		t.Fatalf("catalog Flow resource names are not the clean-break contract: %#v", flowDescriptors)
	}
	for _, descriptor := range flowDescriptors {
		if descriptor.Type != protocol.ResourceTypeCollection {
			t.Fatalf("%s is not a collection: %s", descriptor.ID.Name, descriptor.Type)
		}
	}
	wantDefinitions := []protocol.CapabilityID{
		protocol.CapabilityFlowArchive, protocol.CapabilityFlowCreate, protocol.CapabilityGet,
		protocol.CapabilityList, protocol.CapabilityFlowRun, protocol.CapabilityFlowUpdate,
	}
	wantRuns := []protocol.CapabilityID{
		protocol.CapabilityFlowCancel, protocol.CapabilityGet, protocol.CapabilityList, protocol.CapabilitySubscribe,
	}
	if got := capabilityNames(flowDescriptors[0]); !reflect.DeepEqual(got, wantDefinitions) {
		t.Fatalf("definitions capabilities = %v, want %v", got, wantDefinitions)
	}
	if got := capabilityNames(flowDescriptors[1]); !reflect.DeepEqual(got, wantRuns) {
		t.Fatalf("runs capabilities = %v, want %v", got, wantRuns)
	}
	for _, oldName := range []string{"flow/events", "flow/create", "flow/update", "flow/run", "flow/cancel", "flow/archive"} {
		if _, ok := env.node.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: oldName}); ok {
			t.Fatalf("legacy Flow resource %q is still registered", oldName)
		}
	}
}

func TestFlowDataSchemasDropAggregateSnapshots(t *testing.T) {
	schemas, err := protocol.BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, schema := range schemas {
		if schema.ID == "mfh.flow.definitions.v1" || schema.ID == "mfh.flow.runs.v1" {
			t.Fatalf("obsolete Flow aggregate schema remains generated: %s", schema.ID)
		}
	}
}

func TestDefinitionsCollectionPaginationAndCursorValidation(t *testing.T) {
	env := newEnvironment(t, auth.NewStaticPolicy(), flowfeature.Config{})
	empty := listCollection(t, env.node, protocol.BuiltinFlowDefinitions, 2, "", "")
	if len(empty.Members) != 0 || empty.NextCursor != "" || empty.Revision == 0 {
		t.Fatalf("empty page is invalid: %#v", empty)
	}

	for _, definition := range []protocol.FlowDefinitionV1{
		transformDefinition(flowThree, 1, json.RawMessage(`{"order":3}`)),
		transformDefinition(flowOne, 1, json.RawMessage(`{"order":1}`)),
		transformDefinition(flowTwo, 1, json.RawMessage(`{"order":2}`)),
	} {
		operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})
	}
	first := listCollection(t, env.node, protocol.BuiltinFlowDefinitions, 2, "", "")
	if got := memberKeys(first); !reflect.DeepEqual(got, []string{flowOne, flowTwo}) || first.NextCursor == "" {
		t.Fatalf("first page is not stable and bounded: keys=%v cursor=%q", got, first.NextCursor)
	}
	if err := first.ValidateForDescriptor(mustDescriptor(t, env.node, protocol.BuiltinFlowDefinitions)); err != nil {
		t.Fatalf("page advertises invalid member capabilities: %v", err)
	}
	second := listCollection(t, env.node, protocol.BuiltinFlowDefinitions, 2, "", first.NextCursor)
	if got := memberKeys(second); !reflect.DeepEqual(got, []string{flowThree}) || second.NextCursor != "" {
		t.Fatalf("final page is invalid: keys=%v cursor=%q", got, second.NextCursor)
	}
	request := protocol.CollectionMemberRequestV1{Version: 1, Key: flowTwo}
	got := operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityGet, &request, &protocol.FlowDefinitionV1{}).(*protocol.FlowDefinitionV1)
	if got.FlowID != flowTwo || got.Revision != 1 {
		t.Fatalf("get returned wrong definition: %#v", got)
	}

	badParent := protocol.CollectionListRequestV1{Version: 1, Parent: "nested", Limit: 1}
	if _, err := operateError(env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityList, &badParent); err == nil || !strings.Contains(err.Error(), "does not support parent") {
		t.Fatalf("flat collection accepted parent: %v", err)
	}
	malformed := protocol.CollectionListRequestV1{Version: 1, Cursor: "not-a-cursor", Limit: 1}
	if _, err := operateError(env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityList, &malformed); err == nil || !strings.Contains(err.Error(), "invalid flow collection cursor") {
		t.Fatalf("malformed cursor was accepted: %v", err)
	}

	stale := first.NextCursor
	definition := transformDefinition(flowFour, 1, json.RawMessage(`{"order":4}`))
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})
	staleRequest := protocol.CollectionListRequestV1{Version: 1, Cursor: stale, Limit: 2}
	if _, err := operateError(env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityList, &staleRequest); err == nil || !strings.Contains(err.Error(), "stale revision") {
		t.Fatalf("stale cursor was accepted: %v", err)
	}
}

func TestFlowCapabilitiesUseExactCrossNodeAuthorizationAndOldGrantIsInert(t *testing.T) {
	rootIdentity, _ := auth.GenerateIdentity(1)
	childIdentity, _ := auth.GenerateIdentity(2)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, rootIdentity.PublicKey)
	_ = trust.Add(2, childIdentity.PublicKey)
	policy := auth.NewStaticPolicy()
	definitionsID := protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFlowDefinitions}
	policy.Allow(auth.Request{Subject: 2, Action: auth.Action(protocol.CapabilityList), Resource: definitionsID})
	policy.Allow(auth.Request{Subject: 2, Action: auth.ActionInvoke, Resource: protocol.ResourceID{Owner: 1, Name: "flow/run"}})
	rootNode, err := node.New(context.Background(), node.Config{Identity: rootIdentity, Trust: trust, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	defer rootNode.Close()
	store, _ := keystore.New(t.TempDir())
	controller, err := flowfeature.Register(flowfeature.Config{Node: rootNode, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	definition := transformDefinition(flowOne, 1, json.RawMessage(`{"ok":true}`))
	operate(t, rootNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})

	network := memory.NewNetwork()
	defer network.Close()
	if _, err := rootNode.Listen(network, "root"); err != nil {
		t.Fatal(err)
	}
	childTrust := auth.NewTrustStore()
	_ = childTrust.Add(1, rootIdentity.PublicKey)
	childNode, err := node.New(context.Background(), node.Config{Identity: childIdentity, Trust: childTrust})
	if err != nil {
		t.Fatal(err)
	}
	defer childNode.Close()
	if err := childNode.ConnectParent(context.Background(), network, "root", 1); err != nil {
		t.Fatal(err)
	}

	listCollection(t, childNode, protocol.BuiltinFlowDefinitions, 1, "", "")
	member := protocol.CollectionMemberRequestV1{Version: 1, Key: flowOne}
	if _, err := operateError(childNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityGet, &member); err == nil || !strings.Contains(err.Error(), auth.ErrForbidden.Error()) {
		t.Fatalf("list grant incorrectly authorized get: %v", err)
	}
	run := protocol.FlowRunV1{Version: 1, RunID: runOne, FlowID: flowOne, FlowRevision: 1, DedupeKey: "old-grant", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	if _, err := operateError(childNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &run); err == nil || !strings.Contains(err.Error(), auth.ErrForbidden.Error()) {
		t.Fatalf("legacy flow/run invoke grant authorized the new capability: %v", err)
	}
	policy.Allow(auth.Request{Subject: 2, Action: auth.Action(protocol.CapabilityFlowRun), Resource: definitionsID})
	operate(t, childNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &run, &protocol.FlowRunSummaryV1{})
	waitRun(t, rootNode, runOne, "succeeded")
}

func TestNonOwnerCannotCancelRun(t *testing.T) {
	rootIdentity, _ := auth.GenerateIdentity(1)
	ownerIdentity, _ := auth.GenerateIdentity(2)
	otherIdentity, _ := auth.GenerateIdentity(3)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, rootIdentity.PublicKey)
	_ = trust.Add(2, ownerIdentity.PublicKey)
	_ = trust.Add(3, otherIdentity.PublicKey)
	rootNode, err := node.New(context.Background(), node.Config{Identity: rootIdentity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer rootNode.Close()
	store, _ := keystore.New(t.TempDir())
	controller, err := flowfeature.Register(flowfeature.Config{Node: rootNode, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	blockID := protocol.ResourceID{Owner: 1, Name: "test/cancel-owner"}
	block, err := resource.NewCommand(resource.CommandDescriptor(blockID, "application/json", "test.cancel-owner.v1", "test.invoke", 128), func(ctx context.Context, _ []byte) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rootNode.Registry().Register(block); err != nil {
		t.Fatal(err)
	}
	definition := protocol.FlowDefinitionV1{
		Version: 1, FlowID: flowOne, Revision: 1, Name: "cancel owner",
		Nodes: []protocol.FlowNodeV1{{ID: "block", Kind: "command-call", Resource: protocol.FlowResourceRefV1{OwnerNodeID: "1", Name: blockID.Name}, Config: json.RawMessage(`{}`)}},
	}
	operate(t, rootNode, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})

	network := memory.NewNetwork()
	defer network.Close()
	if _, err := rootNode.Listen(network, "root"); err != nil {
		t.Fatal(err)
	}
	connectChild := func(identity *auth.Identity) *node.Node {
		t.Helper()
		childTrust := auth.NewTrustStore()
		_ = childTrust.Add(1, rootIdentity.PublicKey)
		child, err := node.New(context.Background(), node.Config{Identity: *identity, Trust: childTrust})
		if err != nil {
			t.Fatal(err)
		}
		if err := child.ConnectParent(context.Background(), network, "root", 1); err != nil {
			t.Fatal(err)
		}
		return child
	}
	owner := connectChild(&ownerIdentity)
	defer owner.Close()
	other := connectChild(&otherIdentity)
	defer other.Close()
	run := protocol.FlowRunV1{Version: 1, RunID: runOne, FlowID: flowOne, FlowRevision: 1, DedupeKey: "owner", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, owner, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &run, &protocol.FlowRunSummaryV1{})
	waitRun(t, rootNode, runOne, "running")
	cancelRequest := protocol.FlowCancelV1{Version: 1, RunID: runOne, Reason: "not mine"}
	if _, err := operateError(other, protocol.BuiltinFlowRuns, protocol.CapabilityFlowCancel, &cancelRequest); err == nil || !strings.Contains(err.Error(), "not found for caller") {
		t.Fatalf("non-owner cancel did not preserve existence privacy: %v", err)
	}
	cancelRequest.Reason = "mine"
	cancelled := operate(t, owner, protocol.BuiltinFlowRuns, protocol.CapabilityFlowCancel, &cancelRequest, &protocol.FlowRunSummaryV1{}).(*protocol.FlowRunSummaryV1)
	if cancelled.State != "cancelled" {
		t.Fatalf("run owner could not cancel: %#v", cancelled)
	}
}

func TestRunsSubscriptionPreservesSequenceAndSlowConsumerGap(t *testing.T) {
	env := newEnvironment(t, auth.NewStaticPolicy(), flowfeature.Config{})
	runsID := protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFlowRuns}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	current, err := env.node.Subscribe(ctx, runsID, 4*time.Second, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer current.Cancel()
	definition := manyNodeDefinition(flowOne, 8)
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &definition, &protocol.FlowDefinitionV1{})
	run := protocol.FlowRunV1{Version: 1, RunID: runOne, FlowID: flowOne, FlowRevision: 1, DedupeKey: "events", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &run, &protocol.FlowRunSummaryV1{})
	var lastTransportSequence uint64
	var lastFlowSequence uint64
	for lastState := ""; lastState != "succeeded"; {
		select {
		case event := <-current.Events:
			if event.Kind != subscription.EventData || event.Schema != protocol.SchemaFlowEventV1 {
				t.Fatalf("unexpected Flow subscription event: %#v", event)
			}
			if event.Sequence <= lastTransportSequence {
				t.Fatalf("transport sequence regressed: %d after %d", event.Sequence, lastTransportSequence)
			}
			lastTransportSequence = event.Sequence
			var flowEvent protocol.FlowEventV1
			if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &flowEvent); err != nil {
				t.Fatal(err)
			}
			if flowEvent.Sequence != lastFlowSequence+1 {
				t.Fatalf("Flow event sequence = %d after %d", flowEvent.Sequence, lastFlowSequence)
			}
			lastFlowSequence = flowEvent.Sequence
			lastState = flowEvent.State
		case err := <-current.Errors:
			t.Fatalf("subscription failed before terminal event: %v", err)
		case <-ctx.Done():
			t.Fatal("timed out waiting for Flow events")
		}
	}

	slow, err := env.node.Subscribe(ctx, runsID, 4*time.Second, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer slow.Cancel()
	overflowDefinition := manyNodeDefinition(flowTwo, 64)
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowCreate, &overflowDefinition, &protocol.FlowDefinitionV1{})
	overflowRun := protocol.FlowRunV1{Version: 1, RunID: runTwo, FlowID: flowTwo, FlowRevision: 1, DedupeKey: "overflow", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli()}
	operate(t, env.node, protocol.BuiltinFlowDefinitions, protocol.CapabilityFlowRun, &overflowRun, &protocol.FlowRunSummaryV1{})
	waitRun(t, env.node, runTwo, "succeeded")
	runsPage := listCollection(t, env.node, protocol.BuiltinFlowRuns, 1, "", "")
	if got := memberKeys(runsPage); !reflect.DeepEqual(got, []string{runOne}) || runsPage.NextCursor == "" {
		t.Fatalf("runs first page is invalid: keys=%v cursor=%q", got, runsPage.NextCursor)
	}
	if err := runsPage.ValidateForDescriptor(mustDescriptor(t, env.node, protocol.BuiltinFlowRuns)); err != nil {
		t.Fatalf("runs page advertises invalid member capabilities: %v", err)
	}
	finalRunsPage := listCollection(t, env.node, protocol.BuiltinFlowRuns, 1, "", runsPage.NextCursor)
	if got := memberKeys(finalRunsPage); !reflect.DeepEqual(got, []string{runTwo}) || finalRunsPage.NextCursor != "" {
		t.Fatalf("runs final page is invalid: keys=%v cursor=%q", got, finalRunsPage.NextCursor)
	}
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event, ok := <-slow.Events:
			if !ok {
				t.Fatal("slow subscription closed without a gap or overflow error")
			}
			if event.Kind == subscription.EventGap && event.Reason == "slow_consumer" {
				return
			}
		case err := <-slow.Errors:
			if err != nil && (strings.Contains(err.Error(), "overflow") || strings.Contains(err.Error(), "queue")) {
				return
			}
		case <-deadline:
			t.Fatal("slow subscription did not surface a gap or overflow")
		}
	}
}

func TestFlowRegistrationRollsBackDefinitionsOnRunsConflict(t *testing.T) {
	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, identity.PublicKey)
	runtime, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	conflict, err := resource.NewVariable(resource.VariableDescriptor(
		protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFlowRuns}, "application/json", "test.flow-runs.v1", "test.read", protocol.DefaultMaxPayload,
	), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Registry().Register(conflict); err != nil {
		t.Fatal(err)
	}
	store, _ := keystore.New(t.TempDir())
	if _, err := flowfeature.Register(flowfeature.Config{Node: runtime, Store: store}); err == nil {
		t.Fatal("Flow registration unexpectedly succeeded with a runs conflict")
	}
	if _, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFlowDefinitions}); ok {
		t.Fatal("failed Flow registration left definitions registered")
	}
	resolved, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFlowRuns})
	if !ok || resolved != conflict {
		t.Fatal("registration rollback replaced or removed the pre-existing conflict")
	}
}

func listCollection(t *testing.T, runtime *node.Node, name string, limit int, parent, cursor string) protocol.CollectionPageV1 {
	t.Helper()
	request := protocol.CollectionListRequestV1{Version: 1, Parent: parent, Cursor: cursor, Limit: limit}
	return *operate(t, runtime, name, protocol.CapabilityList, &request, &protocol.CollectionPageV1{}).(*protocol.CollectionPageV1)
}

func capabilityNames(descriptor protocol.ResourceDescriptorV2) []protocol.CapabilityID {
	result := make([]protocol.CapabilityID, 0, len(descriptor.Capabilities))
	for _, capability := range descriptor.Capabilities {
		result = append(result, capability.Name)
	}
	return result
}

func memberKeys(page protocol.CollectionPageV1) []string {
	result := make([]string, 0, len(page.Members))
	for _, member := range page.Members {
		result = append(result, member.Key)
	}
	return result
}

func mustDescriptor(t *testing.T, runtime *node.Node, name string) protocol.ResourceDescriptorV2 {
	t.Helper()
	descriptor, ok := runtime.Registry().Descriptor(protocol.ResourceID{Owner: 1, Name: name})
	if !ok {
		t.Fatalf("resource %s is missing", name)
	}
	return descriptor
}

func manyNodeDefinition(flowID string, count int) protocol.FlowDefinitionV1 {
	nodes := make([]protocol.FlowNodeV1, 0, count)
	for index := 0; index < count; index++ {
		nodes = append(nodes, protocol.FlowNodeV1{
			ID:   "node-" + string(rune('a'+index%26)) + "-" + strings.Repeat("x", index/26),
			Kind: "transform", Config: json.RawMessage(`{"ok":true}`),
		})
	}
	return protocol.FlowDefinitionV1{Version: 1, FlowID: flowID, Revision: 1, Name: "events", Nodes: nodes}
}
