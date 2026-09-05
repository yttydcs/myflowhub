package sdk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestFeatureClientsFileAndNotification(t *testing.T) {
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
	}
}
