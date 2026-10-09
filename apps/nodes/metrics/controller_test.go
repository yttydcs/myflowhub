package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestAllTenMetricsAndConfigUseStandardVariableCatalog(t *testing.T) {
	c, n, _ := newTestController(t, nil, nil)
	defer n.Close()
	defer c.Close()
	values := map[Name]any{BatteryPercent: 0, BatteryCharging: false, BatteryOnAC: false, NetworkOnline: false, NetworkType: "none", CPUPercent: 37.5, MemoryPercent: 0, VolumePercent: 0, VolumeMuted: false, BrightnessPercent: 0}
	if len(c.States()) != 10 {
		t.Fatal("expected exactly ten metrics")
	}
	for name, value := range values {
		t.Run(string(name), func(t *testing.T) {
			if err := c.Update(name, value, nil); err != nil {
				t.Fatal(err)
			}
			state := sampleSnapshot(t, n, name)
			raw, _ := json.Marshal(value)
			if state.Status != "ready" || string(state.Value) != string(raw) || state.SourceTimeUnixMS == nil || state.Definition == nil || state.InstanceID == "" || state.Revision != "2" {
				t.Fatalf("bad standard state: %+v", state)
			}
			if err := protocol.ValidateVariableValue(*state.Definition, state.Value); err != nil {
				t.Fatal(err)
			}
			variable := c.variables[name].variable
			observed := make(chan resource.Observation, 1)
			initial, cancel, err := variable.Observe(func(event resource.Observation) { observed <- event })
			if err != nil {
				t.Fatal(err)
			}
			defer cancel()
			var snapshot protocol.VariableValueV1
			if err := protocol.DecodeJSONPayload(initial.Value, protocol.DefaultMaxPayload, &snapshot); err != nil || snapshot.Definition == nil {
				t.Fatalf("snapshot missing definition: %v", err)
			}
			if err := c.Update(name, value, nil); err != nil {
				t.Fatal(err)
			}
			event := <-observed
			var data map[string]json.RawMessage
			if err := json.Unmarshal(event.Value, &data); err != nil {
				t.Fatal(err)
			}
			if _, present := data["definition"]; present {
				t.Fatal("data repeats definition")
			}
			if string(data["value"]) != string(raw) || string(data["status"]) != `"ready"` {
				t.Fatalf("nonstandard data %s", event.Value)
			}
			id := protocol.ResourceID{Owner: n.ID(), Name: ResourceName(name)}
			definition, _ := DefinitionFor(name)
			want := resource.VariableDescriptor(id, "metrics.read", protocol.DefaultMaxPayload)
			if definition.Controllable {
				want = resource.WritableVariableDescriptor(id, "metrics.read", "metrics.write", protocol.DefaultMaxPayload)
			}
			if got := variable.Descriptor(); !reflect.DeepEqual(got, want) {
				t.Fatalf("descriptor: %+v, want %+v", got, want)
			}
		})
	}
	var catalog protocol.ResourceCatalogV2
	catalogState := n.Registry().Catalog().Snapshot().State
	if err := protocol.DecodeVariablePayload(*catalogState.Definition, catalogState.Value, protocol.DefaultMaxPayload, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Resources) != 13 {
		t.Fatalf("expected catalog + config + update + ten metrics, got %d", len(catalog.Resources))
	}
	for _, d := range catalog.Resources {
		if strings.HasSuffix(d.ID.Name, "/set") {
			t.Fatalf("retired control resource: %s", d.ID.Name)
		}
		if d.Type == protocol.ResourceTypeVariable {
			read, ok := d.Capability(protocol.CapabilityRead)
			if !ok || d.TypeVersion != 2 || read.OutputSchema != protocol.SchemaVariableValueV1 {
				t.Fatalf("nonstandard descriptor: %+v", d)
			}
		}
	}
	config := c.configVariable.Snapshot().State
	if config.Status != "ready" || config.Definition.Type != "object" {
		t.Fatalf("config is not a standard object: %+v", config)
	}
	if err := protocol.ValidateVariableValue(*config.Definition, config.Value); err != nil {
		t.Fatal(err)
	}
	for _, name := range []Name{VolumePercent, VolumeMuted, BrightnessPercent} {
		if _, err := n.Invoke(context.Background(), protocol.ResourceID{Owner: n.ID(), Name: ResourceName(name) + "/set"}, []byte(`{}`)); err == nil {
			t.Fatal("retired /set remains callable")
		}
	}
}

func TestSamplesRetainTrueZeroAndFalseOnFailure(t *testing.T) {
	c, n, _ := newTestController(t, nil, nil)
	defer n.Close()
	defer c.Close()
	for name, value := range map[Name]any{CPUPercent: 0, VolumeMuted: false} {
		if err := c.Update(name, nil, errors.New("offline")); err != nil {
			t.Fatal(err)
		}
		absent, _ := c.State(name)
		if absent.Status != "unavailable" || len(absent.Value) != 0 || absent.SourceTimeUnixMS != nil {
			t.Fatalf("fabricated value: %+v", absent)
		}
		if err := c.Update(name, value, nil); err != nil {
			t.Fatal(err)
		}
		fresh, _ := c.State(name)
		if err := c.Update(name, nil, errors.New("offline")); err != nil {
			t.Fatal(err)
		}
		stale, _ := c.State(name)
		if stale.Status != "stale" || string(stale.Value) != string(fresh.Value) || *stale.SourceTimeUnixMS != *fresh.SourceTimeUnixMS || stale.Error.Message != "offline" {
			t.Fatalf("lost true value: %+v", stale)
		}
		if err := c.Update(name, value, nil); err != nil {
			t.Fatal(err)
		}
		restored, _ := c.State(name)
		if restored.Status != "ready" || restored.Error != nil {
			t.Fatalf("not recovered: %+v", restored)
		}
	}
	before, _ := c.State(CPUPercent)
	if err := c.Update(CPUPercent, "37", nil); err == nil {
		t.Fatal("accepted numeric string sample")
	}
	after, _ := c.State(CPUPercent)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("invalid input changed state")
	}
}

func TestStandardWritesConfirmActualAndRejectBeforeActuator(t *testing.T) {
	for _, name := range []Name{VolumePercent, VolumeMuted, BrightnessPercent} {
		t.Run(string(name), func(t *testing.T) {
			requested, actual := json.RawMessage(`55`), any(54)
			invalid := []string{`101`, `-1`, `1.5`, `"55"`, `false`, `null`}
			if name == VolumeMuted {
				requested, actual = json.RawMessage(`true`), false
				invalid = []string{`"false"`, `0`, `null`, `{}`}
			}
			a := &fakeActuator{result: ApplyResult{Applied: true, Value: actual}}
			c, n, _ := newTestController(t, nil, a)
			defer n.Close()
			defer c.Close()
			v := c.variables[name].variable
			base := v.Snapshot()
			events := atomic.Int32{}
			_, cancel, err := v.Watch(func(resource.VariableUpdate) { events.Add(1) })
			if err != nil {
				t.Fatal(err)
			}
			defer cancel()
			for _, raw := range invalid {
				if _, err := v.CompareAndSet(context.Background(), base.State.InstanceID, base.Revision, json.RawMessage(raw)); err == nil {
					t.Fatalf("accepted %s", raw)
				}
			}
			if _, err := v.CompareAndSet(context.Background(), strings.Repeat("0", 32), base.Revision, requested); err == nil {
				t.Fatal("accepted wrong instance")
			}
			if _, err := v.CompareAndSet(context.Background(), base.State.InstanceID, base.Revision+1, requested); err == nil {
				t.Fatal("accepted wrong revision")
			}
			for _, retired := range []string{`{"version":1,"value":"55"}`, `{"version":2,"expected_revision":1,"value":"NTU="}`} {
				if _, err := v.Operate(context.Background(), resource.OperationRequest{Capability: protocol.CapabilityWrite, Schema: protocol.SchemaVariableWriteV3, Payload: []byte(retired)}); err == nil {
					t.Fatal("retired write format accepted")
				}
			}
			if a.calls.Load() != 0 || events.Load() != 0 || !reflect.DeepEqual(base, v.Snapshot()) {
				t.Fatal("invalid/conflicting write produced side effects")
			}
			request := protocol.VariableWriteV3{Version: 3, ExpectedInstanceID: base.State.InstanceID, ExpectedRevision: base.State.Revision, Value: requested}
			payload, err := protocol.EncodeJSONPayload(&request, protocol.DefaultMaxPayload)
			if err != nil {
				t.Fatal(err)
			}
			response, err := n.Operate(context.Background(), v.Descriptor().ID, protocol.CapabilityWrite, protocol.SchemaVariableWriteV3, payload)
			if err != nil {
				t.Fatal(err)
			}
			var result protocol.VariableValueV1
			if err := protocol.DecodeJSONPayload(response.Payload, protocol.DefaultMaxPayload, &result); err != nil {
				t.Fatal(err)
			}
			rawActual, _ := json.Marshal(actual)
			if a.calls.Load() != 1 || events.Load() != 1 || string(result.Value) != string(rawActual) || result.Status != "ready" || result.Definition == nil {
				t.Fatalf("not confirmed actual: %+v, calls %d", result, a.calls.Load())
			}
			if name == VolumeMuted {
				if _, ok := a.value.(bool); !ok {
					t.Fatalf("handler received %T", a.value)
				}
			} else {
				if _, ok := a.value.(json.Number); !ok {
					t.Fatalf("handler received %T", a.value)
				}
			}
			if _, err := n.Operate(context.Background(), v.Descriptor().ID, protocol.CapabilityWrite, protocol.SchemaVariableWriteV3, payload); err == nil {
				t.Fatal("accepted stale CAS")
			}
			if a.calls.Load() != 1 {
				t.Fatal("CAS retried actuator")
			}
		})
	}
}

func TestUnknownExternalOutcomeKeepsConfirmedStateAndNextSampleCorrects(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		result ApplyResult
		err    error
	}{
		{name: "unconfirmed", result: ApplyResult{Value: 80}},
		{name: "deadline", err: context.DeadlineExceeded},
		{name: "failure", err: errors.New("device failed after mutation")},
		{name: "invalid_actual", result: ApplyResult{Applied: true, Value: "80"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			a := &fakeActuator{result: scenario.result, err: scenario.err}
			c, n, _ := newTestController(t, nil, a)
			defer n.Close()
			defer c.Close()
			if err := c.Update(VolumePercent, 20, nil); err != nil {
				t.Fatal(err)
			}
			v := c.variables[VolumePercent].variable
			base := v.Snapshot()
			events := atomic.Int32{}
			_, stop, _ := v.Watch(func(resource.VariableUpdate) { events.Add(1) })
			defer stop()
			_, err := v.CompareAndSet(context.Background(), base.State.InstanceID, base.Revision, json.RawMessage(`80`))
			if !errors.Is(err, ErrOutcomeUnknown) {
				t.Fatalf("unknown outcome was not explicit: %v", err)
			}
			if a.calls.Load() != 1 || events.Load() != 0 || !reflect.DeepEqual(base, v.Snapshot()) {
				t.Fatal("unconfirmed state was committed/retried")
			}
			c.collector = collectorFunc(func(context.Context, Name) (any, error) { return 80, nil })
			if err := c.collectOnce(context.Background(), VolumePercent); err != nil {
				t.Fatal(err)
			}
			if state := v.Snapshot().State; state.Status != "ready" || string(state.Value) != "80" || a.calls.Load() != 1 {
				t.Fatalf("sample did not correct: %+v", state)
			}
		})
	}
}

func TestUnknownWriteWakesCollectionWithoutRetry(t *testing.T) {
	var reads atomic.Int32
	collector := collectorFunc(func(_ context.Context, name Name) (any, error) {
		if name != VolumePercent {
			return nil, ErrUnsupported
		}
		if reads.Add(1) == 1 {
			return 20, nil
		}
		return 80, nil
	})
	actuator := &fakeActuator{err: errors.New("outcome unavailable")}
	c, n, _ := newTestController(t, collector, actuator)
	defer n.Close()
	defer c.Close()
	deadline := time.Now().Add(time.Second)
	for {
		state, _ := c.State(VolumePercent)
		if state.Status == "ready" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("initial collection did not complete")
		}
		time.Sleep(time.Millisecond)
	}
	// A one-day interval makes immediate correction depend on the read wakeup.
	config := c.Config()
	for i := range config.Settings {
		config.Settings[i].IntervalMS = 86_400_000
	}
	if _, err := c.ApplyConfig(context.Background(), ConfigUpdateV1{Version: 1, ExpectedRevision: config.Revision, Settings: config.Settings, NotificationChannels: []string{}}); err != nil {
		t.Fatal(err)
	}
	// Let the configuration wakeup finish before recording the unknown write.
	deadline = time.Now().Add(time.Second)
	for {
		state, _ := c.State(VolumePercent)
		if reads.Load() >= 2 && string(state.Value) == "80" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("config did not wake collector")
		}
		time.Sleep(time.Millisecond)
	}
	v := c.variables[VolumePercent].variable
	before := v.Snapshot()
	readBefore := reads.Load()
	_, err := v.CompareAndSet(context.Background(), before.State.InstanceID, before.Revision, json.RawMessage(`80`))
	if !errors.Is(err, ErrOutcomeUnknown) {
		t.Fatalf("unexpected write result: %v", err)
	}
	deadline = time.Now().Add(500 * time.Millisecond)
	for {
		if reads.Load() > readBefore && v.Snapshot().Revision > before.Revision {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("unknown write did not wake next collection")
		}
		time.Sleep(time.Millisecond)
	}
	if actuator.calls.Load() != 1 {
		t.Fatal("unknown actuator write was retried")
	}
}

func TestConfigVariablePreservesFullWidthDomainRevision(t *testing.T) {
	n := newMetricsNode(t)
	defer n.Close()
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	config, err := DefaultConfig("windows")
	if err != nil {
		t.Fatal(err)
	}
	config.Revision = 9_007_199_254_740_993
	if err := store.Save("metrics.json", config); err != nil {
		t.Fatal(err)
	}
	c, err := Register(ControllerConfig{Node: n, Store: store, Platform: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	state := c.configVariable.Snapshot().State
	var object map[string]json.RawMessage
	if err := json.Unmarshal(state.Value, &object); err != nil {
		t.Fatal(err)
	}
	if string(object["revision"]) != `"9007199254740993"` {
		t.Fatalf("lost full-width config revision: %s", state.Value)
	}
	var decoded ConfigV1
	if err := protocol.DecodeVariablePayload(*state.Definition, state.Value, protocol.DefaultMaxPayload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Revision != config.Revision {
		t.Fatal("domain revision changed")
	}
	updated, err := c.ApplyConfig(context.Background(), ConfigUpdateV1{Version: 1, ExpectedRevision: config.Revision, Settings: config.Settings, NotificationChannels: config.NotificationChannels})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != config.Revision+1 {
		t.Fatal("domain command did not preserve revision")
	}
}

func TestConfigIsDurableAndBlocksWrites(t *testing.T) {
	a := &fakeActuator{result: ApplyResult{Applied: true, Value: 55}}
	c, n, store := newTestController(t, nil, a)
	defer n.Close()
	defer c.Close()
	config := c.Config()
	for i := range config.Settings {
		if config.Settings[i].Metric == string(BrightnessPercent) {
			config.Settings[i].Writable = false
			config.Settings[i].Enabled = false
		}
	}
	var updated ConfigV1
	invokeMetric(t, n, ResourceConfigUpdate, &ConfigUpdateV1{Version: 1, ExpectedRevision: config.Revision, Settings: config.Settings, NotificationChannels: []string{"alerts", "system"}}, &updated)
	v := c.variables[BrightnessPercent].variable
	base := v.Snapshot()
	if _, err := v.CompareAndSet(context.Background(), base.State.InstanceID, base.Revision, json.RawMessage(`55`)); err == nil {
		t.Fatal("disabled write accepted")
	}
	if a.calls.Load() != 0 {
		t.Fatal("disabled write invoked actuator")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	restartedNode := newMetricsNode(t)
	defer restartedNode.Close()
	restarted, err := Register(ControllerConfig{Node: restartedNode, Store: store, Platform: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	setting, _ := settingFor(restarted.Config(), BrightnessPercent)
	if setting.Writable || setting.Enabled || restarted.Config().Revision != updated.Revision || len(restarted.Config().NotificationChannels) != 2 {
		t.Fatal("configuration not durable")
	}
}

func TestSamplingWriteAndConfigConcurrentWithoutLockInversion(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var calls atomic.Int32
	a := actuatorFunc(func(ctx context.Context, _ Name, value any) (ApplyResult, error) {
		calls.Add(1)
		once.Do(func() { close(entered) })
		select {
		case <-release:
			return ApplyResult{Applied: true, Value: value}, nil
		case <-ctx.Done():
			return ApplyResult{}, ctx.Err()
		}
	})
	c, n, _ := newTestController(t, nil, a)
	defer n.Close()
	defer c.Close()
	if err := c.Update(VolumePercent, 10, nil); err != nil {
		t.Fatal(err)
	}
	c.collector = collectorFunc(func(context.Context, Name) (any, error) { return 63, nil })
	v := c.variables[VolumePercent].variable
	base := v.Snapshot()
	var eventMu sync.Mutex
	revisions := []uint64{}
	_, stop, _ := v.Watch(func(update resource.VariableUpdate) {
		eventMu.Lock()
		revisions = append(revisions, update.Revision)
		eventMu.Unlock()
		_ = c.Config()
	})
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	wrote, collected, configured := make(chan error, 1), make(chan error, 1), make(chan error, 1)
	go func() {
		_, err := v.CompareAndSet(ctx, base.State.InstanceID, base.Revision, json.RawMessage(`60`))
		wrote <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("actuator not entered")
	}
	go func() { collected <- c.collectOnce(ctx, VolumePercent) }()
	config := c.Config()
	for i := range config.Settings {
		if config.Settings[i].Metric == string(VolumePercent) {
			config.Settings[i].Enabled = false
		}
	}
	go func() {
		_, err := c.ApplyConfig(ctx, ConfigUpdateV1{Version: 1, ExpectedRevision: config.Revision, Settings: config.Settings, NotificationChannels: []string{}})
		configured <- err
	}()
	close(release)
	for _, done := range []chan error{wrote, collected, configured} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal("sampling/write/config deadlocked")
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("actuator called %d times", calls.Load())
	}
	final := v.Snapshot().State
	if final.Status != "stale" || (string(final.Value) != "60" && string(final.Value) != "63") {
		t.Fatalf("disabled state resurrected: %+v", final)
	}
	eventMu.Lock()
	defer eventMu.Unlock()
	for i := 1; i < len(revisions); i++ {
		if revisions[i] <= revisions[i-1] {
			t.Fatalf("events out of order: %v", revisions)
		}
	}
}

func TestOlderCollectedStateCannotOverwriteConfirmedWrite(t *testing.T) {
	a := &fakeActuator{result: ApplyResult{Applied: true, Value: 60}}
	c, n, _ := newTestController(t, nil, a)
	defer n.Close()
	defer c.Close()
	v := c.variables[VolumePercent].variable
	old := v.Snapshot()
	state, err := c.sampleState(VolumePercent, old.State, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.CompareAndSet(context.Background(), old.State.InstanceID, old.Revision, json.RawMessage(`60`)); err != nil {
		t.Fatal(err)
	}
	if err := publishSample(v, old.Revision, state); err != nil {
		t.Fatal(err)
	}
	if actual := v.Snapshot().State; string(actual.Value) != "60" {
		t.Fatalf("older sample overwrote write: %+v", actual)
	}
}

func TestExactWritePermissionDoesNotTranslateRetiredGrant(t *testing.T) {
	identity, err := auth.GenerateIdentity(10)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	_ = trust.Add(10, identity.PublicKey)
	policy := auth.NewStaticPolicy()
	id := protocol.ResourceID{Owner: 10, Name: ResourceName(VolumePercent)}
	policy.Allow(auth.Request{Subject: 11, Action: auth.ActionInvoke, Resource: protocol.ResourceID{Owner: 10, Name: id.Name + "/set"}})
	policy.Allow(auth.Request{Subject: 11, Action: auth.ActionRead, Resource: id})
	n, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &fakeActuator{result: ApplyResult{Applied: true, Value: 40}}
	c, err := Register(ControllerConfig{Node: n, Store: store, Platform: "windows", Actuator: a})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	childIdentity, err := auth.GenerateIdentity(11)
	if err != nil {
		t.Fatal(err)
	}
	if err := trust.Add(11, childIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	child, err := node.New(context.Background(), node.Config{Identity: childIdentity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := n.Listen(network, "metrics-exact-permission")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := child.ConnectParent(ctx, network, endpoint, n.ID()); err != nil {
		t.Fatal(err)
	}
	state, _ := c.State(VolumePercent)
	request, _ := protocol.EncodeJSONPayload(&protocol.VariableWriteV3{Version: 3, ExpectedInstanceID: state.InstanceID, ExpectedRevision: state.Revision, Value: json.RawMessage(`40`)}, protocol.DefaultMaxPayload)
	if _, err := child.Operate(context.Background(), id, protocol.CapabilityWrite, protocol.SchemaVariableWriteV3, request); err == nil {
		t.Fatal("retired grant implicitly authorized write")
	}
	if a.calls.Load() != 0 {
		t.Fatal("forbidden write reached actuator")
	}
	policy.Allow(auth.Request{Subject: 11, Action: auth.ActionWrite, Resource: id})
	if _, err := child.Operate(context.Background(), id, protocol.CapabilityWrite, protocol.SchemaVariableWriteV3, request); err != nil {
		t.Fatal(err)
	}
	if a.calls.Load() != 1 {
		t.Fatal("exact write did not invoke once")
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
	value  any
}

func (a *fakeActuator) Set(_ context.Context, _ Name, value any) (ApplyResult, error) {
	a.calls.Add(1)
	a.value = value
	return a.result, a.err
}

type collectorFunc func(context.Context, Name) (any, error)

func (f collectorFunc) Collect(ctx context.Context, name Name) (any, error) { return f(ctx, name) }

type actuatorFunc func(context.Context, Name, any) (ApplyResult, error)

func (f actuatorFunc) Set(ctx context.Context, name Name, value any) (ApplyResult, error) {
	return f(ctx, name, value)
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

func sampleSnapshot(t *testing.T, n *node.Node, name Name) protocol.VariableValueV1 {
	t.Helper()
	value, ok := n.Registry().Resolve(protocol.ResourceID{Owner: n.ID(), Name: ResourceName(name)})
	if !ok {
		t.Fatalf("metric %s not registered", name)
	}
	variable, ok := value.(*resource.Variable)
	if !ok {
		t.Fatal("metric is not a standard Variable")
	}
	return variable.Snapshot().State
}
