package bindings

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/memory"
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
	state, err := auth.OpenState(stateDirectory, 2)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := root.Runtime.Admission.Issue(2, state.Identity.PublicKey, "binding-test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Policy.Close(); err != nil {
		t.Fatal(err)
	}
	host, err := nodehost.New(ctx, nodehost.Config{
		StateDirectory: stateDirectory, NodeID: 2,
		Parent: &nodehost.ParentConfig{NodeID: 1, PublicKey: root.Runtime.Identity.PublicKey,
			Driver: tcp.Driver{}, Endpoint: root.Endpoint, Permit: &permit},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	connection, ok := host.ParentStatus()
	if !ok {
		t.Fatal("missing parent status")
	}
	client, err := NewAttachedClient(host.Client(), PublicIdentity{NodeID: host.ID(), PublicKey: host.PublicKey()}, connection)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	identityJSON, err := client.IdentityJSON()
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
	if err := host.Start(); err != nil {
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

	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	reopenedHost, err := nodehost.New(ctx, nodehost.Config{StateDirectory: stateDirectory, NodeID: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedHost.Close()
	reopened, err := NewAttachedClient(reopenedHost.Client(), PublicIdentity{NodeID: reopenedHost.ID(), PublicKey: reopenedHost.PublicKey()}, nil)
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
	host, err := nodehost.New(context.Background(), nodehost.Config{StateDirectory: t.TempDir(), NodeID: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	client, err := NewAttachedClient(host.Client(), PublicIdentity{NodeID: host.ID(), PublicKey: host.PublicKey()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.CatalogJSON(-1, 1000); err == nil {
		t.Fatal("negative NodeID accepted")
	}
	if _, err := client.CatalogJSON(1, -1); err == nil {
		t.Fatal("negative timeout accepted")
	}
	if _, err := client.InvokeJSON(1, "test/command", "not-json", 1_000); err == nil {
		t.Fatal("invalid command JSON was accepted")
	}
}

func TestAttachedBindingUsesHostClientAndOnlyClosesLocalFacade(t *testing.T) {
	host, err := nodehost.New(context.Background(), nodehost.Config{StateDirectory: t.TempDir(), NodeID: 72})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	variableID := protocol.ResourceID{Owner: host.ID(), Name: "test/status"}
	variable, err := resource.NewVariable(resource.VariableDescriptor(variableID, "application/json", "test.status.v1", "test.read", 64), []byte(`{"state":"ready"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Resources().Register(variable); err != nil {
		t.Fatal(err)
	}
	if err := host.Start(); err != nil {
		t.Fatal(err)
	}
	facade, err := NewAttachedClient(host.Client(), PublicIdentity{
		NodeID: host.ID(), PublicKey: host.PublicKey(),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	identityJSON, err := facade.IdentityJSON()
	if err != nil || !strings.Contains(identityJSON, `"node_id":"72"`) {
		t.Fatalf("unexpected attached identity %q: %v", identityJSON, err)
	}
	snapshot, err := facade.SnapshotJSON(72, variableID.Name, 1_000)
	if err != nil || snapshot != `{"state":"ready"}` {
		t.Fatalf("unexpected attached snapshot %q: %v", snapshot, err)
	}
	for _, method := range []string{"TrustParent", "StartTCP", "StartRFCOMM", "EnrollTCP", "StartEnrolledTCP"} {
		if _, ok := reflect.TypeOf(facade).MethodByName(method); ok {
			t.Fatalf("attached facade exposes runtime method %s", method)
		}
	}
	if err := facade.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-host.Node().Done():
		t.Fatal("attached binding Close closed the host node")
	default:
	}
	event, err := host.Client().Snapshot(context.Background(), variableID)
	if err != nil || string(event.Value) != `{"state":"ready"}` {
		t.Fatalf("host client was affected by attached facade Close: event=%+v err=%v", event, err)
	}
}

func TestAttachedBindingRejectsMismatchedPublicIdentity(t *testing.T) {
	host, err := nodehost.New(context.Background(), nodehost.Config{StateDirectory: t.TempDir(), NodeID: 73})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	if _, err := NewAttachedClient(host.Client(), PublicIdentity{NodeID: 74, PublicKey: host.PublicKey()}, nil); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched attached identity returned %v", err)
	}
}

func TestAttachedBindingBeforeHostStartAllowsLocalOperationsButNotParentWait(t *testing.T) {
	parent, err := auth.GenerateIdentity(76)
	if err != nil {
		t.Fatal(err)
	}
	network := memory.NewNetwork()
	defer network.Close()
	host, err := nodehost.New(context.Background(), nodehost.Config{
		StateDirectory: t.TempDir(), NodeID: 75,
		Parent: &nodehost.ParentConfig{NodeID: parent.NodeID, PublicKey: parent.PublicKey, Driver: network, Endpoint: "not-started"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	statusID := protocol.ResourceID{Owner: host.ID(), Name: "test/pre-start"}
	variable, err := resource.NewVariable(resource.VariableDescriptor(statusID, "application/json", "test.status.v1", "test.read", 64), []byte(`{"ready":false}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Resources().Register(variable); err != nil {
		t.Fatal(err)
	}
	connection, ok := host.ParentStatus()
	if !ok {
		t.Fatal("configured parent status was not available")
	}
	facade, err := NewAttachedClient(host.Client(), PublicIdentity{NodeID: host.ID(), PublicKey: host.PublicKey()}, connection)
	if err != nil {
		t.Fatal(err)
	}
	defer facade.Close()
	if snapshot, err := facade.SnapshotJSON(75, statusID.Name, 1_000); err != nil || snapshot != `{"ready":false}` {
		t.Fatalf("pre-start local operation failed: %q %v", snapshot, err)
	}
	if err := facade.WaitConnected(100); !errors.Is(err, nodehost.ErrNotStarted) {
		t.Fatalf("pre-start parent wait returned %v", err)
	}
}

func TestAttachedBindingSubscriptionCancellationDoesNotAffectHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	network := memory.NewNetwork()
	defer network.Close()
	root, err := nodehost.New(ctx, nodehost.Config{
		StateDirectory: t.TempDir(), NodeID: 81,
		Listeners: []nodehost.ListenerConfig{{Driver: network, Endpoint: "attached-root"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	statusID := protocol.ResourceID{Owner: root.ID(), Name: "test/streaming-status"}
	status, err := resource.NewVariable(resource.VariableDescriptor(statusID, "application/json", "test.status.v1", "test.subscribe", 64), []byte(`{"value":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Resources().Register(status); err != nil {
		t.Fatal(err)
	}
	leaf, err := nodehost.New(ctx, nodehost.Config{
		StateDirectory: t.TempDir(), NodeID: 82,
		Parent: &nodehost.ParentConfig{
			NodeID: root.ID(), PublicKey: root.State().Identity.PublicKey,
			Driver: network, Endpoint: "attached-root",
			Supervisor: node.SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer leaf.Close()
	if err := root.State().Trust.Add(leaf.ID(), leaf.State().Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	if err := root.State().Policy.Grant(auth.Request{Subject: leaf.ID(), Action: auth.ActionSubscribe, Resource: statusID}); err != nil {
		t.Fatal(err)
	}
	if err := root.Start(); err != nil {
		t.Fatal(err)
	}
	if err := leaf.Start(); err != nil {
		t.Fatal(err)
	}
	connection, ok := leaf.ParentStatus()
	if !ok {
		t.Fatal("leaf did not expose parent status")
	}
	facade, err := NewAttachedClient(leaf.Client(), PublicIdentity{
		NodeID: leaf.ID(), PublicKey: leaf.PublicKey(),
	}, connection)
	if err != nil {
		t.Fatal(err)
	}
	defer facade.Close()
	if err := facade.WaitConnected(2_000); err != nil {
		t.Fatal(err)
	}
	listener := &testListener{events: make(chan string, 2), errors: make(chan string, 1)}
	subscriptionID, err := facade.Subscribe(int64(root.ID()), statusID.Name, 60_000, listener)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-listener.events:
	case failure := <-listener.errors:
		t.Fatalf("attached subscription failed: %s", failure)
	case <-time.After(2 * time.Second):
		t.Fatal("attached subscription did not deliver its snapshot")
	}
	facade.CancelSubscription(subscriptionID)
	deadline := time.Now().Add(time.Second)
	for root.Node().Stats().ActiveSubscriptions != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if active := root.Node().Stats().ActiveSubscriptions; active != 0 {
		t.Fatalf("attached subscription was not cancelled: %d active", active)
	}
	if _, err := status.Set([]byte(`{"value":2}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-listener.events:
		t.Fatalf("cancelled attached subscription delivered %s", event)
	case <-time.After(50 * time.Millisecond):
	}
	if err := facade.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-leaf.Node().Done():
		t.Fatal("attached subscription facade Close stopped the leaf Host")
	default:
	}
	if current := connection.Snapshot(); current.State != sdk.ConnectionConnected {
		t.Fatalf("attached subscription cancellation changed Host connection: %+v", current)
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
	client, err := NewEnrollmentBootstrap(stateDirectory)
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
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewEnrollmentBootstrap(stateDirectory)
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

func TestEnrollmentBootstrapHasNoOrdinaryRuntimeSurface(t *testing.T) {
	bootstrap, err := NewEnrollmentBootstrap(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	statusJSON, err := bootstrap.EnrollmentStatusJSON()
	if err != nil || !strings.Contains(statusJSON, `"status":"device"`) {
		t.Fatalf("unexpected bootstrap status: %v (%s)", err, statusJSON)
	}
	typeOf := reflect.TypeOf(bootstrap)
	for _, forbidden := range []string{"StartEnrolledTCP", "StartTCP", "CatalogJSON", "OperateJSON", "Subscribe"} {
		if _, exists := typeOf.MethodByName(forbidden); exists {
			t.Fatalf("Enrollment bootstrap exposes ordinary runtime method %s", forbidden)
		}
	}
	if err := bootstrap.Close(); err != nil {
		t.Fatal(err)
	}
	if err := bootstrap.Close(); err != nil {
		t.Fatalf("repeated bootstrap Close returned %v", err)
	}
	if _, err := bootstrap.EnrollmentStatusJSON(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("closed bootstrap status returned %v", err)
	}
}

func TestEnrollmentBootstrapCloseCancelsActiveHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	release := make(chan struct{})
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			close(accepted)
			return
		}
		close(accepted)
		<-release
		_ = connection.Close()
	}()
	bootstrap, err := NewEnrollmentBootstrap(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := bootstrap.EnrollTCP(listener.Addr().String(), "", true, 0, "", "", 5_000)
		done <- err
	}()
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("Enrollment bootstrap did not enter the handshake")
	}
	started := time.Now()
	if err := bootstrap.Close(); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("bootstrap Close took %s", elapsed)
	}
	close(release)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled Enrollment returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled Enrollment did not return")
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
