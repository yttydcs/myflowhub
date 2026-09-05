package metrics

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestControllerPublishesFreshStaleAndTypedCatalog(t *testing.T) {
	controller, runtime, _ := newTestController(t, nil, nil)
	defer runtime.Close()
	defer controller.Close()
	if err := controller.Update(CPUPercent, "37.5", nil); err != nil {
		t.Fatal(err)
	}
	fresh := sampleSnapshot(t, runtime, CPUPercent)
	if fresh.Status != "fresh" || fresh.Value != "37.5" || fresh.Unit != "percent" || fresh.SampledAtUnixMS == 0 {
		t.Fatalf("unexpected fresh sample: %+v", fresh)
	}
	if err := controller.Update(CPUPercent, "", errors.New("collector offline")); err != nil {
		t.Fatal(err)
	}
	stale := sampleSnapshot(t, runtime, CPUPercent)
	if stale.Status != "stale" || stale.Value != fresh.Value || stale.SampledAtUnixMS != fresh.SampledAtUnixMS || stale.Error != "collector offline" {
		t.Fatalf("unexpected stale sample: %+v", stale)
	}
	if err := controller.Update(CPUPercent, "101", nil); err == nil {
		t.Fatal("out-of-range percent was accepted")
	}
	catalogPayload := runtime.Registry().Catalog().Snapshot().Value
	var catalog protocol.ResourceCatalogV2
	if err := protocol.DecodeJSONPayload(catalogPayload, protocol.DefaultMaxPayload, &catalog); err != nil {
		t.Fatal(err)
	}
	if !catalogContains(catalog, ResourceName(CPUPercent), protocol.ResourceTypeVariable, SchemaSampleV1) ||
		!catalogContains(catalog, CommandName(BrightnessPercent), protocol.ResourceTypeCommand, SchemaControlV1) {
		t.Fatalf("metrics catalog is incomplete: %+v", catalog.Resources)
	}
	assertVariableDescriptor(t, runtime, ResourceConfig, SchemaConfigV1, "metrics.config.read")
	assertVariableDescriptor(t, runtime, ResourceName(CPUPercent), SchemaSampleV1, "metrics.read")
}

func assertVariableDescriptor(t *testing.T, runtime *node.Node, name, schema, permission string) {
	t.Helper()
	id := protocol.ResourceID{Owner: runtime.ID(), Name: name}
	got, ok := runtime.Registry().Descriptor(id)
	if !ok {
		t.Fatalf("variable %q was not registered", name)
	}
	want := resource.VariableDescriptor(id, "application/json", schema, permission, protocol.DefaultMaxPayload)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("variable %q descriptor changed:\n got: %+v\nwant: %+v", name, got, want)
	}
}

func TestControlAndConfigAreValidatedAndDurable(t *testing.T) {
	actuator := &fakeActuator{result: ApplyResult{Applied: true, Value: "55"}}
	controller, runtime, store := newTestController(t, nil, actuator)
	defer runtime.Close()
	request := ControlV1{Version: 1, Value: "55"}
	var result ControlResultV1
	invokeMetric(t, runtime, CommandName(BrightnessPercent), &request, &result)
	if !result.Applied || result.AppliedValue != "55" || actuator.calls.Load() != 1 {
		t.Fatalf("unexpected control result: %+v", result)
	}
	if sample, _ := controller.Sample(BrightnessPercent); sample.Value != "55" || sample.Status != "fresh" {
		t.Fatalf("applied control did not publish state: %+v", sample)
	}
	invalid := ControlV1{Version: 1, Value: "200"}
	if _, err := invokeMetricError(runtime, CommandName(BrightnessPercent), &invalid); err == nil {
		t.Fatal("invalid control value was accepted")
	}

	config := controller.Config()
	settings := append([]SettingV1(nil), config.Settings...)
	for index := range settings {
		if settings[index].Metric == string(BrightnessPercent) {
			settings[index].Writable = false
		}
	}
	update := ConfigUpdateV1{
		Version: 1, ExpectedRevision: config.Revision, Settings: settings,
		NotificationChannels: []string{"alerts", "system"},
	}
	var updated ConfigV1
	invokeMetric(t, runtime, ResourceConfigUpdate, &update, &updated)
	if updated.Revision != config.Revision+1 {
		t.Fatalf("config revision did not advance: %+v", updated)
	}
	if _, err := invokeMetricError(runtime, CommandName(BrightnessPercent), &request); err == nil {
		t.Fatal("disabled metric control was accepted")
	}
	if err := controller.Close(); err != nil {
		t.Fatal(err)
	}

	restartedNode := newMetricsNode(t)
	defer restartedNode.Close()
	restarted, err := Register(ControllerConfig{Node: restartedNode, Store: store, Platform: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	persisted, ok := settingFor(restarted.Config(), BrightnessPercent)
	if !ok || persisted.Writable || restarted.Config().Revision != updated.Revision || len(restarted.Config().NotificationChannels) != 2 {
		t.Fatalf("metrics config was not durable: %+v", restarted.Config())
	}
}

func TestNotificationChannelsAreCanonicalAndBounded(t *testing.T) {
	config, err := DefaultConfig("windows")
	if err != nil {
		t.Fatal(err)
	}
	config.NotificationChannels = []string{"system", "alerts"}
	if err := config.Validate(); err == nil {
		t.Fatal("expected unsorted channels error")
	}
	config.NotificationChannels = []string{"alerts", "alerts"}
	if err := config.Validate(); err == nil {
		t.Fatal("expected duplicate channels error")
	}
}

func TestCollectorTransitionsFromFreshToStaleWithoutFakeValues(t *testing.T) {
	collector := &sequenceCollector{}
	controller, runtime, _ := newTestController(t, collector, nil)
	defer runtime.Close()
	defer controller.Close()
	waitMetric(t, controller, CPUPercent, func(sample SampleV1) bool { return sample.Status == "fresh" && sample.Value == "42" })
	waitMetric(t, controller, CPUPercent, func(sample SampleV1) bool {
		return sample.Status == "stale" && sample.Value == "42" && sample.Error == "sensor failed"
	})
}

func TestDefaultConfigIsCompleteSortedAndPlatformScoped(t *testing.T) {
	windows, err := DefaultConfig("windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(windows.Settings) != len(Definitions("windows")) {
		t.Fatal("incomplete Windows config")
	}
	if _, err := DefaultConfig("android"); err == nil {
		t.Fatal("retired platform was accepted")
	}
	if len(Definitions("android")) != 0 {
		t.Fatal("retired platform exposed metrics")
	}
}

func TestControllerRejectsRegistryFromAnotherNode(t *testing.T) {
	local := newMetricsNode(t)
	defer local.Close()
	foreign := newMetricsNode(t)
	defer foreign.Close()
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Register(ControllerConfig{
		Node: local, Resources: foreign.Registry(), Store: store, Platform: "windows",
	}); err == nil {
		t.Fatal("controller accepted a Registry owned by another Node")
	}
}

type fakeActuator struct {
	result ApplyResult
	err    error
	calls  atomic.Int32
}

func (a *fakeActuator) Set(context.Context, Name, string) (ApplyResult, error) {
	a.calls.Add(1)
	return a.result, a.err
}

type sequenceCollector struct{ cpuCalls atomic.Int32 }

func (c *sequenceCollector) Collect(_ context.Context, name Name) (string, error) {
	if name != CPUPercent {
		return "", ErrUnsupported
	}
	if c.cpuCalls.Add(1) == 1 {
		return "42", nil
	}
	return "", errors.New("sensor failed")
}

func newTestController(t *testing.T, collector Collector, actuator Actuator) (*Controller, *node.Node, *keystore.Store) {
	t.Helper()
	runtime := newMetricsNode(t)
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	controller, err := Register(ControllerConfig{
		Node: runtime, Store: store, Platform: "windows", Collector: collector, Actuator: actuator,
	})
	if err != nil {
		_ = runtime.Close()
		t.Fatal(err)
	}
	return controller, runtime, store
}

func newMetricsNode(t *testing.T) *node.Node {
	t.Helper()
	identity, err := auth.GenerateIdentity(10)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	_ = trust.Add(identity.NodeID, identity.PublicKey)
	runtime, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func sampleSnapshot(t *testing.T, runtime *node.Node, name Name) SampleV1 {
	t.Helper()
	value, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: ResourceName(name)})
	if !ok {
		t.Fatalf("metric %s was not registered", name)
	}
	variable, ok := value.(*resource.Variable)
	if !ok {
		t.Fatalf("metric %s is not a Variable", name)
	}
	var sample SampleV1
	if err := protocol.DecodeJSONPayload(variable.Snapshot().Value, protocol.DefaultMaxPayload, &sample); err != nil {
		t.Fatal(err)
	}
	return sample
}

func invokeMetric(t *testing.T, runtime *node.Node, name string, request, response protocol.ValidatedPayload) {
	t.Helper()
	output, err := invokeMetricError(runtime, name, request)
	if err != nil {
		t.Fatal(err)
	}
	if err := protocol.DecodeJSONPayload(output, protocol.DefaultMaxPayload, response); err != nil {
		t.Fatal(err)
	}
}

func invokeMetricError(runtime *node.Node, name string, request protocol.ValidatedPayload) ([]byte, error) {
	input, err := protocol.EncodeJSONPayload(request, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, err
	}
	value, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: name})
	if !ok {
		return nil, resource.ErrNotFound
	}
	command, ok := value.(*resource.Command)
	if !ok {
		return nil, errors.New("resource is not a command")
	}
	return command.Invoke(context.Background(), input)
}

func catalogContains(catalog protocol.ResourceCatalogV2, name string, typeID protocol.ResourceTypeID, schema string) bool {
	for _, descriptor := range catalog.Resources {
		if descriptor.ID.Name == name && descriptor.Type == typeID {
			for _, current := range descriptor.Schemas {
				if current.ID == schema {
					return true
				}
			}
		}
	}
	return false
}

func waitMetric(t *testing.T, controller *Controller, name Name, matches func(SampleV1) bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if sample, ok := controller.Sample(name); ok && matches(sample) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	sample, _ := controller.Sample(name)
	t.Fatalf("metric %s did not reach expected state: %+v", name, sample)
}
