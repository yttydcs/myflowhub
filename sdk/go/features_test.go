package sdk

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/memory"
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
	const runID = "10112233445566778899aabbccddeeff"
	definition := protocol.FlowDefinitionV1{
		Version: 1, FlowID: flowID, Revision: 1, Name: "SDK transform",
		Nodes: []protocol.FlowNodeV1{{ID: "result", Kind: "transform", Config: json.RawMessage(`{"ok":true}`)}},
	}
	created, err := flows.Create(ctx, definition)
	if err != nil || created.FlowID != flowID {
		t.Fatalf("unexpected flow create result: %+v (%v)", created, err)
	}
	run, err := flows.Run(ctx, protocol.FlowRunV1{
		Version: 1, RunID: runID, FlowID: flowID, FlowRevision: 1, DedupeKey: "sdk", DeadlineUnixMS: time.Now().Add(5 * time.Second).UnixMilli(),
	})
	if err != nil || run.RunID != runID {
		t.Fatalf("unexpected flow run result: %+v (%v)", run, err)
	}
	completed := waitSDKFlowRun(t, ctx, flows, runID)
	if completed.State != "succeeded" || completed.Error != "" {
		t.Fatalf("flow did not succeed: %+v", completed)
	}
	if err := flows.Archive(ctx, protocol.FlowArchiveV1{Version: 1, FlowID: flowID}); err != nil {
		t.Fatal(err)
	}
	definitions, err := flows.Definitions(ctx)
	if err != nil || len(definitions.Definitions) != 0 {
		t.Fatalf("flow archive was not observable: %+v (%v)", definitions, err)
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
		resource(auth.ActionRead, protocol.BuiltinFlowDefinitions),
		resource(auth.ActionRead, protocol.BuiltinFlowRuns),
		resource(auth.ActionInvoke, protocol.BuiltinFlowCreate),
		resource(auth.ActionInvoke, protocol.BuiltinFlowRun),
		resource(auth.ActionInvoke, protocol.BuiltinFlowArchive),
	}
}

func waitSDKFlowRun(t *testing.T, ctx context.Context, flows *FlowClient, runID string) protocol.FlowRunSummaryV1 {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		runs, err := flows.Runs(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, run := range runs.Runs {
			if run.RunID == runID && run.FinishedUnixMS > 0 {
				return run
			}
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatalf("flow run did not finish: %v", ctx.Err())
		}
	}
}
