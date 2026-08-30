package bindings

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func TestBindingClientTCPJSONAndSubscriptionContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: t.TempDir(), NodeID: 1,
		Listeners: []hub.ListenerConfig{{Driver: tcp.Driver{}, Endpoint: "127.0.0.1:0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	stateDirectory := t.TempDir()
	client, err := NewClient(stateDirectory, 2)
	if err != nil {
		t.Fatal(err)
	}
	identityJSON, err := client.IdentityJSON()
	if err != nil {
		t.Fatal(err)
	}
	var identity struct {
		NodeID    string `json:"node_id"`
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal([]byte(identityJSON), &identity); err != nil {
		t.Fatal(err)
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(identity.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || identity.NodeID != "2" {
		t.Fatalf("invalid binding identity: %s", identityJSON)
	}
	permit, err := root.Runtime.Admission.Issue(2, ed25519.PublicKey(publicKey), "binding-test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	resources := []auth.Request{
		{Subject: 2, Action: auth.ActionRead, Resource: protocol.ResourceID{Owner: 1, Name: protocol.BuiltinResourceCatalog}},
		{Subject: 2, Action: auth.ActionRead, Resource: protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementHealth}},
		{Subject: 2, Action: auth.ActionSubscribe, Resource: protocol.ResourceID{Owner: 1, Name: protocol.BuiltinNotificationEvents}},
		{Subject: 2, Action: auth.ActionInvoke, Resource: protocol.ResourceID{Owner: 1, Name: protocol.BuiltinNotificationPublish}},
	}
	for _, grant := range resources {
		if err := root.Runtime.Policy.Grant(grant); err != nil {
			t.Fatal(err)
		}
	}
	if err := client.TrustParent(1, base64.RawStdEncoding.EncodeToString(root.Runtime.Identity.PublicKey)); err != nil {
		t.Fatal(err)
	}
	permitPayload, err := protocol.EncodeJSONPayload(&permit, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.StartTCP(string(root.Endpoint), 1, string(permitPayload)); err != nil {
		t.Fatal(err)
	}
	if err := client.WaitConnected(3_000); err != nil {
		t.Fatal(err)
	}
	statusJSON, err := client.StatusJSON()
	if err != nil {
		t.Fatal(err)
	}
	var status bindingConnection
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil || status.State != string("connected") || status.ParentNodeID != "1" {
		t.Fatalf("unexpected binding status %s: %v", statusJSON, err)
	}
	catalogJSON, err := client.CatalogJSON(1, 3_000)
	if err != nil {
		t.Fatal(err)
	}
	var catalog protocol.ResourceCatalogV2
	if err := json.Unmarshal([]byte(catalogJSON), &catalog); err != nil || len(catalog.Resources) < 10 {
		t.Fatalf("unexpected binding catalog: %v (%s)", err, catalogJSON)
	}
	healthJSON, err := client.SnapshotJSON(1, protocol.BuiltinManagementHealth, 3_000)
	if err != nil {
		t.Fatal(err)
	}
	var health protocol.ManagementHealthV1
	if err := json.Unmarshal([]byte(healthJSON), &health); err != nil || health.Version != 1 {
		t.Fatalf("unexpected binding health: %v (%s)", err, healthJSON)
	}

	listener := &testListener{events: make(chan string, 2), errors: make(chan string, 1)}
	subscriptionID, err := client.Subscribe(1, protocol.BuiltinNotificationEvents, 60_000, listener)
	if err != nil {
		t.Fatal(err)
	}
	request, err := protocol.EncodeJSONPayload(&protocol.NotificationPublishV1{
		Version: 1, Channel: "sdk", ContentType: "text/plain", Body: []byte("hello"),
	}, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	responseJSON, err := client.InvokeJSON(1, protocol.BuiltinNotificationPublish, string(request), 3_000)
	if err != nil {
		t.Fatal(err)
	}
	var response protocol.NotificationEventV1
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil || string(response.Body) != "hello" {
		t.Fatalf("unexpected binding command response: %v (%s)", err, responseJSON)
	}
	select {
	case eventJSON := <-listener.events:
		var event bindingEvent
		if err := json.Unmarshal([]byte(eventJSON), &event); err != nil || event.Kind != "data" ||
			event.ResourceName != protocol.BuiltinNotificationEvents || event.Capability != string(protocol.CapabilitySubscribe) ||
			event.Schema != protocol.SchemaNotificationEventV1 {
			t.Fatalf("unexpected binding event: %v (%s)", err, eventJSON)
		}
		var notification protocol.NotificationEventV1
		if err := json.Unmarshal(event.Value, &notification); err != nil || notification.EventID != response.EventID {
			t.Fatalf("unexpected notification event payload: %v (%s)", err, event.Value)
		}
	case errorJSON := <-listener.errors:
		t.Fatalf("binding subscription failed: %s", errorJSON)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for binding subscription event")
	}
	client.CancelSubscription(subscriptionID)
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewClient(stateDirectory, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopenedIdentity, err := reopened.IdentityJSON()
	if err != nil || reopenedIdentity != identityJSON {
		t.Fatalf("binding identity was not durable: %v (%s != %s)", err, reopenedIdentity, identityJSON)
	}
}

func TestBindingRejectsInvalidBoundaryValues(t *testing.T) {
	if _, err := NewClient(t.TempDir(), -1); err == nil {
		t.Fatal("negative NodeID was accepted")
	}
	client, err := NewClient(t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.TrustParent(1, "invalid"); err == nil {
		t.Fatal("invalid parent key was accepted")
	}
	if err := client.StartTCP("", 1, ""); err == nil {
		t.Fatal("empty endpoint was accepted")
	}
	if _, err := client.InvokeJSON(1, "test/command", "not-json", 1_000); err == nil {
		t.Fatal("invalid command JSON was accepted")
	}
}

func TestBindingEnrollmentNeedsNoClientNodeIDOrParentKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: t.TempDir(), NodeID: 1,
		Listeners: []hub.ListenerConfig{{Driver: tcp.Driver{}, Endpoint: "127.0.0.1:0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	stateDirectory := t.TempDir()
	client, err := NewEnrollmentClient(stateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	statusJSON, err := client.EnrollmentStatusJSON()
	if err != nil {
		t.Fatal(err)
	}
	var status struct {
		Status          string `json:"status"`
		RequestID       string `json:"request_id"`
		DevicePublicKey string `json:"device_public_key"`
		NodeID          string `json:"node_id"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		t.Fatal(err)
	}
	if status.Status != "device" || status.NodeID != "" || status.DevicePublicKey == "" {
		t.Fatalf("new binding client already had a Node ID: %s", statusJSON)
	}
	deviceKey, _ := base64.RawStdEncoding.DecodeString(status.DevicePublicKey)
	permit, err := root.EnrollmentAuthority.IssuePermit(
		"a0000000000000000000000000000001",
		auth.DevicePublicKeyFingerprint(ed25519.PublicKey(deviceKey)),
		root.Node.ID(), false, "binding-enrollment", time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	permitPayload, _ := protocol.EncodeJSONPayload(&permit, protocol.EnrollmentMaxPayload)
	resultJSON, err := client.EnrollTCP(string(root.Endpoint), string(permitPayload), false, 0, "", "", 3_000)
	if err != nil {
		t.Fatal(err)
	}
	var result protocol.EnrollmentResultV1
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil || result.Status != "granted" || result.Grant == nil {
		t.Fatalf("unexpected binding Enrollment result: %v (%s)", err, resultJSON)
	}
	if err := client.StartEnrolledTCP(string(root.Endpoint)); err != nil {
		t.Fatal(err)
	}
	if err := client.WaitConnected(3_000); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewEnrollmentClient(stateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopenedStatus, err := reopened.EnrollmentStatusJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(reopenedStatus), &status); err != nil || status.Status != "enrolled" || status.NodeID != result.Grant.NodeID || status.RequestID == "" {
		t.Fatalf("binding Enrollment state was not durable: %v (%s)", err, reopenedStatus)
	}
}

func TestConnectionWaitErrorKeepsLatestDiagnostic(t *testing.T) {
	err := connectionWaitError(context.DeadlineExceeded, sdk.ConnectionSnapshot{LastError: "remote closed during admission"})
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "remote closed during admission") {
		t.Fatalf("latest connection diagnostic was lost: %v", err)
	}
}

type testListener struct {
	events chan string
	errors chan string
}

func (l *testListener) OnEvent(eventJSON string) { l.events <- eventJSON }
func (l *testListener) OnError(errorJSON string) { l.errors <- errorJSON }
