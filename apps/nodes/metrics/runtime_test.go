package metrics

import (
	"context"
	"testing"
	"time"

	featurenotification "github.com/yttydcs/myflowhub/feature/notification"
	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestRuntimeConnectsAndParentControlsMetricsResources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rootIdentity, _ := auth.GenerateIdentity(1)
	rootTrust := auth.NewTrustStore()
	_ = rootTrust.Add(1, rootIdentity.PublicKey)
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	admission, err := auth.LoadAdmission(rootIdentity, store, auth.AdmissionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	root, err := node.New(ctx, node.Config{
		Identity: rootIdentity, Trust: rootTrust, Admission: admission, Policy: auth.AllowAll{},
		Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := featurenotification.Register(featurenotification.Config{Node: root}); err != nil {
		t.Fatal(err)
	}
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := root.Listen(network, "metrics-runtime")
	if err != nil {
		t.Fatal(err)
	}
	childState, err := auth.OpenState(t.TempDir(), 2)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := admission.Issue(2, childState.Identity.PublicKey, "metrics", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	actuator := &fakeActuator{result: ApplyResult{Applied: true, Value: "40"}}
	runtime, err := Start(ctx, RuntimeConfig{
		StateDirectory: childState.Directory, NodeID: 2, ParentID: 1, ParentKey: rootIdentity.PublicKey,
		Permit: &permit, Driver: network, Endpoint: endpoint, Platform: "windows", Actuator: actuator,
		Supervisor: node.SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	waitRuntimeConnected(t, runtime)
	if err := runtime.Metrics.Update(CPUPercent, "73", nil); err != nil {
		t.Fatal(err)
	}
	remote, err := root.Subscribe(ctx, protocol.ResourceID{Owner: 2, Name: ResourceName(CPUPercent)}, time.Minute, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Cancel()
	event := <-remote.Events
	var sample SampleV1
	if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &sample); err != nil || sample.Value != "73" {
		t.Fatalf("unexpected parent snapshot: %+v (%v)", sample, err)
	}
	request, _ := protocol.EncodeJSONPayload(&ControlV1{Version: 1, Value: "40"}, protocol.DefaultMaxPayload)
	response, err := root.Invoke(ctx, protocol.ResourceID{Owner: 2, Name: CommandName(BrightnessPercent)}, request)
	if err != nil {
		t.Fatal(err)
	}
	var result ControlResultV1
	if err := protocol.DecodeJSONPayload(response, protocol.DefaultMaxPayload, &result); err != nil || !result.Applied {
		t.Fatalf("unexpected parent control response: %+v (%v)", result, err)
	}

	config := runtime.Metrics.Config()
	if _, err := runtime.Metrics.ApplyConfig(ctx, ConfigUpdateV1{
		Version: 1, ExpectedRevision: config.Revision, Settings: config.Settings,
		NotificationChannels: []string{"system"},
	}); err != nil {
		t.Fatal(err)
	}
	waitNotificationFromParent(t, ctx, root, runtime)
}

func waitNotificationFromParent(t *testing.T, ctx context.Context, root *node.Node, runtime *Runtime) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		request := protocol.NotificationPublishV1{
			Version: 1, Channel: "system", ContentType: "text/plain", Body: []byte("hello metrics"),
			Attributes: map[string]string{"title": "Runtime test"},
		}
		input, err := protocol.EncodeJSONPayload(&request, protocol.DefaultMaxPayload)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := root.Invoke(ctx, protocol.ResourceID{Owner: 1, Name: protocol.BuiltinNotificationPublish}, input); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
		events := runtime.Notifications.Dequeue()
		if len(events) > 0 {
			if events[0].Channel != "system" || string(events[0].Body) != "hello metrics" {
				t.Fatalf("unexpected notification: %+v", events[0])
			}
			return
		}
	}
	t.Fatalf("metrics runtime did not receive notification: %+v", runtime.Notifications.Status())
}

func waitRuntimeConnected(t *testing.T, runtime *Runtime) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.Connection.Snapshot().State == "connected" {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("metrics runtime did not connect: %+v", runtime.Connection.Snapshot())
}
