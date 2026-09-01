package sdk

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	flowfeature "github.com/yttydcs/myflowhub/feature/flow"
	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/memory"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func TestFeatureClientsFileFlowAndNotification(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	network := memory.NewNetwork()
	defer network.Close()
	fileRoot := t.TempDir()
	root, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: t.TempDir(), NodeID: 1, FileRoot: fileRoot,
		Listeners: []hub.ListenerConfig{{Driver: network, Endpoint: "sdk-features"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	childState, err := hostconfig.Open(t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := childState.Trust.Add(1, root.Runtime.Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	permit, err := root.Runtime.Admission.Issue(2, childState.Identity.PublicKey, "sdk-test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, grant := range featureClientGrants() {
		if err := root.Runtime.Policy.Grant(grant); err != nil {
			t.Fatal(err)
		}
	}
	child, err := node.New(ctx, node.Config{Identity: childState.Identity, Trust: childState.Trust, Policy: childState.Policy, JoinPermit: &permit})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	client, err := NewClient(child)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Connect(ctx, network, root.Endpoint, 1); err != nil {
		t.Fatal(err)
	}

	notifications, err := client.Notifications(1)
	if err != nil {
		t.Fatal(err)
	}
	notification, err := notifications.Publish(ctx, protocol.NotificationPublishV1{
		Version: 1, Channel: "sdk", ContentType: "text/plain", Body: []byte("feature client"),
	})
	if err != nil || string(notification.Body) != "feature client" || notification.SourceNodeID != "2" {
		t.Fatalf("unexpected notification result: %+v (%v)", notification, err)
	}

	files, err := client.Files(1)
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(t.TempDir(), "payload.bin")
	content := []byte("complete SDK file upload")
	if err := os.WriteFile(sourcePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	progress, err := files.UploadFile(ctx, sourcePath, "incoming/payload.bin", "application/octet-stream", 7, time.Minute)
	if err != nil || progress.State != "completed" || progress.ReceivedBytes != int64(len(content)) {
		t.Fatalf("unexpected upload result: %+v (%v)", progress, err)
	}
	stored, err := os.ReadFile(filepath.Join(fileRoot, "incoming", "payload.bin"))
	if err != nil || string(stored) != string(content) {
		t.Fatalf("unexpected uploaded file: %q (%v)", stored, err)
	}
	transfers, err := files.Transfers(ctx)
	if err != nil || len(transfers.Transfers) != 1 || transfers.Transfers[0].State != "completed" {
		t.Fatalf("unexpected transfer snapshot: %+v (%v)", transfers, err)
	}

	flows, err := client.Flows(1)
	if err != nil {
		t.Fatal(err)
	}
	const flowID = "00112233445566778899aabbccddeeff"
	const secondFlowID = "20112233445566778899aabbccddeeff"
	const runID = "10112233445566778899aabbccddeeff"
	definition := protocol.FlowDefinitionV1{
		Version: 1, FlowID: flowID, Revision: 1, Name: "SDK transform",
		Nodes: []protocol.FlowNodeV1{{ID: "result", Kind: "transform", Config: json.RawMessage(`{"ok":true}`)}},
	}
	created, err := flows.Create(ctx, definition)
	if err != nil || created.FlowID != flowID {
		t.Fatalf("unexpected flow create result: %+v (%v)", created, err)
	}
	_, err = flows.GetDefinition(ctx, protocol.CollectionMemberRequestV1{Version: 1, Key: flowID})
	assertSDKError(t, err, protocol.CodeForbidden, false)
	if err := root.Runtime.Policy.Grant(auth.Request{
		Subject: 2, Action: auth.Action(protocol.CapabilityGet),
		Resource: protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFlowDefinitions},
	}); err != nil {
		t.Fatal(err)
	}
	secondDefinition := definition
	secondDefinition.FlowID = secondFlowID
	secondDefinition.Name = "SDK second transform"
	if _, err := flows.Create(ctx, secondDefinition); err != nil {
		t.Fatal(err)
	}
	firstPage, err := flows.ListDefinitions(ctx, protocol.CollectionListRequestV1{Version: 1, Limit: 1})
	if err != nil || len(firstPage.Members) != 1 || firstPage.NextCursor == "" {
		t.Fatalf("unexpected first definitions page: %+v (%v)", firstPage, err)
	}
	secondPage, err := flows.ListDefinitions(ctx, protocol.CollectionListRequestV1{Version: 1, Cursor: firstPage.NextCursor, Limit: 1})
	if err != nil || len(secondPage.Members) != 1 || secondPage.NextCursor != "" {
		t.Fatalf("unexpected final definitions page: %+v (%v)", secondPage, err)
	}
	gotDefinition, err := flows.GetDefinition(ctx, protocol.CollectionMemberRequestV1{Version: 1, Key: flowID})
	if err != nil || gotDefinition.FlowID != flowID {
		t.Fatalf("unexpected typed definition: %+v (%v)", gotDefinition, err)
	}
	definition.Revision = 2
	definition.Name = "SDK transform updated"
	updated, err := flows.Update(ctx, definition)
	if err != nil || updated.Revision != 2 {
		t.Fatalf("unexpected flow update result: %+v (%v)", updated, err)
	}
	events, err := flows.Events(ctx, time.Minute, 16)
	if err != nil {
		t.Fatal(err)
	}
	defer events.Cancel()
	run, err := flows.Run(ctx, protocol.FlowRunV1{
		Version: 1, RunID: runID, FlowID: flowID, FlowRevision: 2, DedupeKey: "sdk", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli(),
	})
	if err != nil || run.RunID != runID {
		t.Fatalf("unexpected flow run result: %+v (%v)", run, err)
	}
	event := receiveEvent(t, events)
	var flowEvent protocol.FlowEventV1
	decodeErr := DecodeEvent(event, &flowEvent)
	if event.Resource.Name != protocol.BuiltinFlowRuns || event.Capability != protocol.CapabilitySubscribe || decodeErr != nil || flowEvent.RunID != runID {
		t.Fatalf("unexpected typed Flow event: %+v payload=%+v (%v)", event, flowEvent, decodeErr)
	}
	completed := waitSDKFlowRun(t, ctx, flows, runID)
	if completed.State != "succeeded" || completed.Error != "" {
		t.Fatalf("flow did not succeed: %+v", completed)
	}
	runs, err := flows.ListRuns(ctx, protocol.CollectionListRequestV1{Version: 1, Limit: 1})
	if err != nil || len(runs.Members) != 1 || runs.Members[0].Key != runID {
		t.Fatalf("unexpected typed runs page: %+v (%v)", runs, err)
	}
	cancelled, err := flows.Cancel(ctx, protocol.FlowCancelV1{Version: 1, RunID: runID, Reason: "SDK idempotent cancel"})
	if err != nil || cancelled.RunID != runID || cancelled.State != "succeeded" {
		t.Fatalf("unexpected terminal cancel result: %+v (%v)", cancelled, err)
	}
	if err := flows.Archive(ctx, protocol.FlowArchiveV1{Version: 1, FlowID: flowID}); err != nil {
		t.Fatal(err)
	}
	if err := flows.Archive(ctx, protocol.FlowArchiveV1{Version: 1, FlowID: secondFlowID}); err != nil {
		t.Fatal(err)
	}
	definitions, err := flows.ListDefinitions(ctx, protocol.CollectionListRequestV1{Version: 1, Limit: 10})
	if err != nil || len(definitions.Members) != 0 {
		t.Fatalf("flow archive was not observable: %+v (%v)", definitions, err)
	}
}

func TestFlowClientTypedTCPContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rootIdentity, _ := auth.GenerateIdentity(31)
	childIdentity, _ := auth.GenerateIdentity(32)
	trust := auth.NewTrustStore()
	if err := trust.Add(rootIdentity.NodeID, rootIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	if err := trust.Add(childIdentity.NodeID, childIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	policy := auth.NewStaticPolicy()
	definitionsID := protocol.ResourceID{Owner: rootIdentity.NodeID, Name: protocol.BuiltinFlowDefinitions}
	policy.Allow(auth.Request{Subject: childIdentity.NodeID, Action: auth.Action(protocol.CapabilityFlowCreate), Resource: definitionsID})
	policy.Allow(auth.Request{Subject: childIdentity.NodeID, Action: auth.Action(protocol.CapabilityList), Resource: definitionsID})
	root, err := node.New(ctx, node.Config{Identity: rootIdentity, Trust: trust, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	controller, err := flowfeature.Register(flowfeature.Config{Node: root, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	endpoint, err := root.Listen(tcp.Driver{}, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	child, err := node.New(ctx, node.Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	client, err := NewClient(child)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Connect(ctx, tcp.Driver{}, endpoint, rootIdentity.NodeID); err != nil {
		t.Fatal(err)
	}
	flows, err := client.Flows(rootIdentity.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	definition := protocol.FlowDefinitionV1{
		Version: 1, FlowID: "30112233445566778899aabbccddeeff", Revision: 1, Name: "TCP typed Flow",
		Nodes: []protocol.FlowNodeV1{{ID: "result", Kind: "transform", Config: json.RawMessage(`{"transport":"tcp"}`)}},
	}
	if _, err := flows.Create(ctx, definition); err != nil {
		t.Fatal(err)
	}
	page, err := flows.ListDefinitions(ctx, protocol.CollectionListRequestV1{Version: 1, Limit: 1})
	if err != nil || len(page.Members) != 1 || page.Members[0].Key != definition.FlowID {
		t.Fatalf("unexpected TCP typed page: %+v (%v)", page, err)
	}
}

func TestPolicyClientUsesAttachedNodePath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	directory := t.TempDir()
	bootstrap, err := hostconfig.Open(directory, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrap.Policy.CreateBinding(protocol.PolicyBindingCreateV1{
		Version: 1, BindingID: strings.Repeat("d", 32), Subject: "2", DefinitionID: protocol.BuiltinPolicySuperadmin,
		Scope: protocol.PolicyOwnerScopeV1{Kind: protocol.PolicyScopeAuthorityDomain, NodeID: "1"},
	}, 1); err != nil {
		t.Fatal(err)
	}
	if err := bootstrap.Policy.Close(); err != nil {
		t.Fatal(err)
	}
	network := memory.NewNetwork()
	defer network.Close()
	root, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: directory, NodeID: 1,
		Listeners: []hub.ListenerConfig{{Driver: network, Endpoint: "sdk-policy"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	childState, err := hostconfig.Open(t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer childState.Policy.Close()
	if err := childState.Trust.Add(1, root.Runtime.Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	permit, err := root.Runtime.Admission.Issue(2, childState.Identity.PublicKey, "sdk-policy", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	child, err := node.New(ctx, node.Config{Identity: childState.Identity, Trust: childState.Trust, Policy: childState.Policy, JoinPermit: &permit})
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(child)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Connect(ctx, network, root.Endpoint, 1); err != nil {
		t.Fatal(err)
	}
	policies, err := client.Policies(1)
	if err != nil {
		t.Fatal(err)
	}
	page, err := policies.ListDefinitions(ctx, protocol.CollectionListRequestV1{Version: 1, Limit: 10})
	if err != nil || len(page.Members) != 1 || page.Members[0].Key != protocol.BuiltinPolicySuperadmin {
		t.Fatalf("unexpected initial policy page: %+v (%v)", page, err)
	}
	created, err := policies.CreateDefinition(ctx, protocol.PolicyDefinitionPutV1{
		Version: 1, ID: "health-reader", Label: "Health reader",
		Rules: []protocol.PolicyRuleV1{{
			Resource:   protocol.PolicyResourceSelectorV1{Kind: protocol.PolicySelectorExact, Value: protocol.BuiltinManagementHealth},
			Capability: protocol.PolicyCapabilitySelectorV1{Kind: protocol.PolicySelectorExact, Values: []protocol.CapabilityID{protocol.CapabilityRead}},
		}},
	})
	if err != nil || created.Revision != 1 {
		t.Fatalf("typed create definition: %+v (%v)", created, err)
	}
	decision, err := policies.Evaluate(ctx, protocol.PolicyEvaluateRequestV1{
		Version: 1, Subject: "2", Capability: "read", ResourceNode: "1", ResourceName: protocol.BuiltinManagementHealth,
	})
	if err != nil || !decision.Allowed || decision.DefinitionID != protocol.BuiltinPolicySuperadmin {
		t.Fatalf("typed policy evaluation: %+v (%v)", decision, err)
	}
}

func featureClientGrants() []auth.Request {
	resource := func(action auth.Action, name string) auth.Request {
		return auth.Request{Subject: 2, Action: action, Resource: protocol.ResourceID{Owner: 1, Name: name}}
	}
	return []auth.Request{
		resource(auth.ActionInvoke, protocol.BuiltinNotificationPublish),
		resource(auth.ActionRead, protocol.BuiltinFileTransfers),
		resource(auth.ActionOpen, protocol.BuiltinFileUpload),
		resource(auth.Action(protocol.CapabilityList), protocol.BuiltinFlowDefinitions),
		resource(auth.Action(protocol.CapabilityFlowCreate), protocol.BuiltinFlowDefinitions),
		resource(auth.Action(protocol.CapabilityFlowUpdate), protocol.BuiltinFlowDefinitions),
		resource(auth.Action(protocol.CapabilityFlowRun), protocol.BuiltinFlowDefinitions),
		resource(auth.Action(protocol.CapabilityFlowArchive), protocol.BuiltinFlowDefinitions),
		resource(auth.Action(protocol.CapabilityList), protocol.BuiltinFlowRuns),
		resource(auth.Action(protocol.CapabilityGet), protocol.BuiltinFlowRuns),
		resource(auth.Action(protocol.CapabilitySubscribe), protocol.BuiltinFlowRuns),
		resource(auth.Action(protocol.CapabilityFlowCancel), protocol.BuiltinFlowRuns),
	}
}

func waitSDKFlowRun(t *testing.T, ctx context.Context, flows *FlowClient, runID string) protocol.FlowRunSummaryV1 {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		run, err := flows.GetRun(ctx, protocol.CollectionMemberRequestV1{Version: 1, Key: runID})
		if err != nil {
			t.Fatal(err)
		}
		if run.RunID == runID && run.FinishedUnixMS > 0 {
			return run
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("flow run did not finish: %v", ctx.Err())
		}
	}
}
