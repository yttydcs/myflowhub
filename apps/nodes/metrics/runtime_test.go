package metrics

import (
	"context"
	"testing"
	"time"

	featurenotification "github.com/yttydcs/myflowhub/feature/notification"
	"github.com/yttydcs/myflowhub/host/nodehost"
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
	t.Cleanup(func() { _ = runtime.Close() })
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
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if status := runtime.Host.Status(); status.Lifecycle != nodehost.LifecycleStopped {
		t.Fatalf("host did not stop with metrics runtime: %+v", status)
	}
}

func TestRuntimeUsesSingleLeafHostAndCompletePreStartCatalog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parentIdentity, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	network := memory.NewNetwork()
	defer network.Close()
	runtime, err := prepareRuntime(ctx, RuntimeConfig{
		StateDirectory: t.TempDir(), NodeID: 2, ParentID: 1, ParentKey: parentIdentity.PublicKey,
		Driver: network, Endpoint: "metrics-prestart", Platform: "windows",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if runtime.Host == nil || runtime.Node != runtime.Host.Node() || runtime.SDK != runtime.Host.Client() {
		t.Fatal("metrics runtime did not expose the single Host-owned Node and Client")
	}
	if runtime.State != runtime.Host.State() || runtime.Host.Resources() != runtime.Node.Registry() {
		t.Fatal("metrics runtime did not expose the Host-owned state and Registry")
	}
	status := runtime.Host.Status()
	if status.Lifecycle != nodehost.LifecycleNew || status.Role != nodehost.RoleLeaf || len(status.Endpoints) != 0 {
		t.Fatalf("unexpected prepared metrics host: %+v", status)
	}

	descriptors := runtime.Host.Resources().List()
	expected := 3 + len(Definitions("windows")) // catalog, config, config update, and every metric variable
	for _, definition := range Definitions("windows") {
		if definition.Controllable {
			expected++
		}
	}
	if len(descriptors) != expected {
		t.Fatalf("pre-start catalog has %d resources, want %d: %+v", len(descriptors), expected, descriptors)
	}
	for _, descriptor := range descriptors {
		if descriptor.ID.Owner != runtime.Host.ID() {
			t.Fatalf("catalog resource %q owner %d, want %d", descriptor.ID.Name, descriptor.ID.Owner, runtime.Host.ID())
		}
	}
	var catalog protocol.ResourceCatalogV2
	if err := protocol.DecodeJSONPayload(
		runtime.Host.Resources().Catalog().Snapshot().Value, protocol.DefaultMaxPayload, &catalog,
	); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Resources) != expected {
		t.Fatalf("published pre-start catalog has %d resources, want %d", len(catalog.Resources), expected)
	}
}

func TestRuntimePreparationFailureReleasesHostStateDirectory(t *testing.T) {
	parentIdentity, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	stateDirectory := t.TempDir()
	state, err := auth.OpenState(stateDirectory, 2)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := DefaultConfig("android")
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Store.Save("metrics.json", stored); err != nil {
		t.Fatal(err)
	}
	network := memory.NewNetwork()
	defer network.Close()
	if _, err := Start(context.Background(), RuntimeConfig{
		StateDirectory: stateDirectory, NodeID: 2, ParentID: 1, ParentKey: parentIdentity.PublicKey,
		Driver: network, Endpoint: "metrics-rollback", Platform: "windows",
	}); err == nil {
		t.Fatal("expected stored platform mismatch")
	}
	host, err := nodehost.New(context.Background(), nodehost.Config{StateDirectory: stateDirectory, NodeID: 2})
	if err != nil {
		t.Fatalf("failed metrics preparation retained the Host state reservation: %v", err)
	}
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
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
