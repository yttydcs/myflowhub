package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

var ErrUnsupported = errors.New("metric is unsupported on this platform")
var ErrOutcomeUnknown = errors.New("metric write outcome is unknown; wait for the next sample")

// MetricState is a product display projection, never a second state store.
type MetricState struct {
	Metric string                   `json:"metric"`
	State  protocol.VariableValueV1 `json:"state"`
}

type metricSlot struct {
	// io serializes collection, external writes, and configuration transitions.
	// Never acquire a Variable mutation gate while holding io or Controller.mu.
	io       sync.Mutex
	variable *resource.Variable
	resample chan struct{}
}

type Collector interface {
	Collect(context.Context, Name) (any, error)
}

type ApplyResult struct {
	Applied bool
	Value   any
}

type Actuator interface {
	Set(context.Context, Name, any) (ApplyResult, error)
}

type ControllerConfig struct {
	Node      *node.Node
	Resources *resource.Registry
	Store     *keystore.Store
	Platform  string
	Collector Collector
	Actuator  Actuator
	Logger    *slog.Logger
	Now       func() time.Time
}

type Controller struct {
	node      *node.Node
	resources *resource.Registry
	store     *keystore.Store
	collector Collector
	actuator  Actuator
	logger    *slog.Logger
	now       func() time.Time

	configMu       sync.Mutex
	mu             sync.RWMutex
	config         ConfigV1
	configVariable *resource.Variable
	variables      map[Name]*metricSlot
	configUpdating bool
	changed        chan struct{}
	closed         bool
	registered     []protocol.ResourceID

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func Register(config ControllerConfig) (*Controller, error) {
	if config.Node == nil || config.Store == nil {
		return nil, errors.New("metrics controller requires a node and durable store")
	}
	if config.Resources == nil {
		config.Resources = config.Node.Registry()
	}
	if config.Resources != config.Node.Registry() {
		return nil, errors.New("metrics controller resources must belong to its node")
	}
	if !validPlatform(config.Platform) {
		return nil, errors.New("metrics controller platform is invalid")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	current, err := loadConfig(config.Store, config.Platform)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	value := &Controller{
		node: config.Node, resources: config.Resources, store: config.Store, collector: config.Collector, actuator: config.Actuator, logger: config.Logger, now: config.Now,
		config: current, variables: make(map[Name]*metricSlot), changed: make(chan struct{}),
		ctx: ctx, cancel: cancel,
	}
	if err := value.registerResources(); err != nil {
		cancel()
		value.removeRegistered()
		return nil, err
	}
	value.wg.Add(1)
	go func() {
		defer value.wg.Done()
		select {
		case <-config.Node.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	if config.Collector != nil {
		for _, definition := range Definitions(config.Platform) {
			value.wg.Add(1)
			go value.runCollector(definition.Name)
		}
	}
	return value, nil
}

func (c *Controller) Config() ConfigV1 {
	if c == nil {
		return ConfigV1{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return cloneConfig(c.config)
}

func (c *Controller) State(name Name) (protocol.VariableValueV1, bool) {
	if c == nil {
		return protocol.VariableValueV1{}, false
	}
	slot := c.variables[name]
	if slot == nil {
		return protocol.VariableValueV1{}, false
	}
	return slot.variable.Snapshot().State, true
}

func (c *Controller) States() []MetricState {
	if c == nil {
		return nil
	}
	result := make([]MetricState, 0, len(c.variables))
	for name, slot := range c.variables {
		result = append(result, MetricState{Metric: string(name), State: slot.variable.Snapshot().State})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Metric < result[j].Metric })
	return result
}

func (c *Controller) ApplyConfig(ctx context.Context, request ConfigUpdateV1) (ConfigV1, error) {
	if ctx == nil {
		return ConfigV1{}, errors.New("metrics config update context is required")
	}
	input, err := protocol.EncodeJSONPayload(&request, protocol.DefaultMaxPayload)
	if err != nil {
		return ConfigV1{}, err
	}
	output, err := c.updateConfig(ctx, input)
	if err != nil {
		return ConfigV1{}, err
	}
	var result ConfigV1
	if err := protocol.DecodeJSONPayload(output, protocol.DefaultMaxPayload, &result); err != nil {
		return ConfigV1{}, err
	}
	return result, nil
}

// Update publishes an owner-provided typed sample. A sample superseded by a
// concurrent write/configuration transition is discarded instead of replayed.
func (c *Controller) Update(name Name, value any, collectErr error) error {
	if c == nil {
		return errors.New("metrics controller is closed")
	}
	slot := c.variables[name]
	if slot == nil {
		return ErrUnsupported
	}
	slot.io.Lock()
	setting, _, active := c.ioSetting(name)
	if !active {
		slot.io.Unlock()
		return errors.New("metrics controller is closed or updating configuration")
	}
	if !setting.Enabled {
		collectErr = errors.New("metric is disabled")
	}
	previous := slot.variable.Snapshot()
	next, err := c.sampleState(name, previous.State, value, collectErr)
	slot.io.Unlock()
	if err != nil {
		return err
	}
	return publishSample(slot.variable, previous.Revision, next)
}

func (c *Controller) Close() error {
	if c == nil {
		return nil
	}
	var closeErr error
	c.closeOnce.Do(func() {
		c.configMu.Lock()
		c.mu.Lock()
		c.closed = true
		close(c.changed)
		c.mu.Unlock()
		c.configMu.Unlock()
		c.cancel()
		c.wg.Wait()
		closeErr = c.removeRegistered()
	})
	return closeErr
}

func (c *Controller) registerResources() error {
	owner := c.node.ID()
	configPayload, err := protocol.EncodeVariablePayload(ConfigDefinition(), &c.config, protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	c.configVariable, err = c.resources.Variable(resource.VariableSpec{
		Name: ResourceConfig, Definition: ConfigDefinition(),
		ReadPermission: "metrics.config.read", MaxPayloadBytes: protocol.DefaultMaxPayload, Initial: resource.VariableState{Value: configPayload, Status: "ready"},
	})
	if err != nil {
		return err
	}
	c.registered = append(c.registered, c.configVariable.Descriptor().ID)
	update, err := resource.NewCommand(resource.CommandDescriptorSchemas(
		protocol.ResourceID{Owner: owner, Name: ResourceConfigUpdate}, "application/json", SchemaConfigUpdateV1, SchemaConfigV1,
		"metrics.config.write", protocol.DefaultMaxPayload,
	), c.updateConfig)
	if err != nil {
		return err
	}
	if err := c.register(update); err != nil {
		return err
	}
	for _, definition := range Definitions(c.config.Platform) {
		name := definition.Name
		valueDefinition, err := ValueDefinition(name)
		if err != nil {
			return err
		}
		slot := &metricSlot{resample: make(chan struct{}, 1)}
		spec := resource.VariableSpec{
			Name: ResourceName(name), Definition: valueDefinition,
			ReadPermission: "metrics.read", MaxPayloadBytes: protocol.DefaultMaxPayload,
			Initial: resource.VariableState{Status: "unavailable", Error: &protocol.VariableError{Code: "no_sample", Message: "metric has not produced a sample"}},
		}
		if definition.Controllable {
			spec.WritePermission = "metrics.write"
			spec.WriteHandler = func(ctx context.Context, value any) (resource.VariableState, error) {
				return c.write(ctx, name, slot, value)
			}
		}
		slot.variable, err = c.resources.Variable(spec)
		if err != nil {
			return err
		}
		c.registered = append(c.registered, slot.variable.Descriptor().ID)
		c.variables[name] = slot
	}
	return nil
}

func (c *Controller) register(value resource.Resource) error {
	if err := c.resources.Register(value); err != nil {
		return err
	}
	c.registered = append(c.registered, value.Descriptor().ID)
	return nil
}

func (c *Controller) removeRegistered() error {
	var result error
	for index := len(c.registered) - 1; index >= 0; index-- {
		if err := c.resources.Remove(c.registered[index]); err != nil && !errors.Is(err, resource.ErrNotFound) && result == nil {
			result = err
		}
	}
	c.registered = nil
	return result
}

func (c *Controller) sampleState(name Name, previous protocol.VariableValueV1, value any, collectErr error) (resource.VariableState, error) {
	if collectErr != nil {
		return failedSample(previous, "collection_failed", collectErr), nil
	}
	if err := ValidateValue(name, value); err != nil {
		return resource.VariableState{}, err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return resource.VariableState{}, err
	}
	now := c.now().UTC().UnixMilli()
	return resource.VariableState{Value: payload, Status: "ready", SourceTimeUnixMS: &now}, nil
}

func failedSample(previous protocol.VariableValueV1, code string, err error) resource.VariableState {
	next := resource.VariableState{Status: "unavailable", Error: &protocol.VariableError{Code: code, Message: truncateError(err.Error())}}
	if len(previous.Value) != 0 {
		next.Value, next.SourceTimeUnixMS, next.Status = previous.Value, previous.SourceTimeUnixMS, "stale"
	}
	return next
}

func publishSample(variable *resource.Variable, revision uint64, state resource.VariableState) error {
	if revision == math.MaxUint64 {
		return errors.New("metric revision exhausted")
	}
	_, err := variable.Apply(revision+1, state)
	if errors.Is(err, resource.ErrRevisionRegression) {
		// Another confirmed state won. Reapplying this older observation would
		// overwrite a write or resurrect a sample disabled by configuration.
		return nil
	}
	return err
}

func (c *Controller) write(ctx context.Context, name Name, slot *metricSlot, value any) (resource.VariableState, error) {
	// Runtime owns the mutation gate and has already validated value and CAS.
	// Return actual state; never call Set/Apply from this handler.
	slot.io.Lock()
	defer slot.io.Unlock()
	setting, _, active := c.ioSetting(name)
	if !active {
		return resource.VariableState{}, errors.New("metrics controller is closed or updating configuration")
	}
	if !setting.Enabled || !setting.Writable {
		return resource.VariableState{}, errors.New("metric write is disabled by configuration")
	}
	if c.actuator == nil {
		return resource.VariableState{}, ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		return resource.VariableState{}, err
	}
	result, err := c.actuator.Set(ctx, name, value)
	if err != nil || !result.Applied || ctx.Err() != nil {
		requestSample(slot)
		return resource.VariableState{}, ErrOutcomeUnknown
	}
	state, err := c.sampleState(name, protocol.VariableValueV1{}, result.Value, nil)
	if err != nil {
		requestSample(slot)
		return resource.VariableState{}, fmt.Errorf("%w: actuator returned invalid actual value", ErrOutcomeUnknown)
	}
	return state, nil
}

// Coalesce read requests only. An uncertain external write is never replayed.
func requestSample(slot *metricSlot) {
	select {
	case slot.resample <- struct{}{}:
	default:
	}
}

func (c *Controller) updateConfig(ctx context.Context, input []byte) ([]byte, error) {
	var request ConfigUpdateV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.configMu.Lock()
	defer c.configMu.Unlock()
	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()
	if closed {
		return nil, errors.New("metrics controller is closed")
	}
	if request.ExpectedRevision != c.config.Revision {
		return nil, fmt.Errorf("metrics config revision conflict: current %d", c.config.Revision)
	}
	if c.config.Revision == math.MaxUint64 {
		return nil, errors.New("metrics config revision exhausted")
	}
	settings := append([]SettingV1(nil), request.Settings...)
	sort.Slice(settings, func(i, j int) bool { return settings[i].Metric < settings[j].Metric })
	channels := append([]string{}, request.NotificationChannels...)
	sort.Strings(channels)
	if len(settings) != len(Definitions(c.config.Platform)) {
		return nil, errors.New("metrics config update must include every platform metric")
	}
	next := ConfigV1{
		Version: 1, Revision: c.config.Revision + 1, Platform: c.config.Platform,
		Settings: settings, NotificationChannels: channels,
	}
	if err := next.Validate(); err != nil {
		return nil, err
	}
	payload, err := protocol.EncodeJSONPayload(&next, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, err
	}
	variablePayload, err := protocol.EncodeVariablePayload(ConfigDefinition(), &next, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, err
	}
	previous := cloneConfig(c.config)
	// Drain every metric I/O before changing policy. The updating flag rejects
	// new I/O while persistence and Variable.Set run with I/O and state locks free.
	unlock := c.lockMetricIO()
	c.mu.Lock()
	c.configUpdating = true
	c.mu.Unlock()
	unlock()
	defer func() {
		c.mu.Lock()
		c.configUpdating = false
		close(c.changed)
		c.changed = make(chan struct{})
		c.mu.Unlock()
	}()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.store.Save("metrics.json", next); err != nil {
		return nil, fmt.Errorf("persist metrics config: %w", err)
	}
	if _, err := c.configVariable.Set(resource.VariableState{Value: variablePayload, Status: "ready"}); err != nil {
		if rollbackErr := c.store.Save("metrics.json", previous); rollbackErr != nil {
			return nil, fmt.Errorf("publish metrics config: %v; rollback persistence: %w", err, rollbackErr)
		}
		return nil, fmt.Errorf("publish metrics config: %w", err)
	}
	c.mu.Lock()
	c.config = next
	c.mu.Unlock()
	for _, setting := range settings {
		if !setting.Enabled {
			variable := c.variables[Name(setting.Metric)].variable
			for {
				snapshot := variable.Snapshot()
				if snapshot.Revision == math.MaxUint64 {
					return nil, errors.New("disabled metric revision exhausted")
				}
				_, err := variable.Apply(snapshot.Revision+1, failedSample(snapshot.State, "disabled", errors.New("metric is disabled")))
				if errors.Is(err, resource.ErrRevisionRegression) {
					continue
				}
				if err != nil {
					return nil, fmt.Errorf("publish disabled metric %s: %w", setting.Metric, err)
				}
				break
			}
		}
	}
	return payload, nil
}

func (c *Controller) lockMetricIO() func() {
	names := make([]Name, 0, len(c.variables))
	for name := range c.variables {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	for _, name := range names {
		c.variables[name].io.Lock()
	}
	return func() {
		for i := len(names) - 1; i >= 0; i-- {
			c.variables[names[i]].io.Unlock()
		}
	}
}

func (c *Controller) collectOnce(ctx context.Context, name Name) error {
	slot := c.variables[name]
	slot.io.Lock()
	setting, _, active := c.ioSetting(name)
	if !active || !setting.Enabled {
		slot.io.Unlock()
		return nil
	}
	select {
	case <-slot.resample:
	default:
	}
	previous := slot.variable.Snapshot()
	value, collectErr := c.collector.Collect(ctx, name)
	if collectErr == nil {
		collectErr = ctx.Err()
	}
	next, err := c.sampleState(name, previous.State, value, collectErr)
	if err != nil {
		next = failedSample(previous.State, "invalid_sample", err)
	}
	slot.io.Unlock()
	return publishSample(slot.variable, previous.Revision, next)
}

func (c *Controller) runCollector(name Name) {
	defer c.wg.Done()
	for {
		setting, changed, ok := c.workerSetting(name)
		if !ok {
			return
		}
		if !setting.Enabled {
			select {
			case <-changed:
				continue
			case <-c.ctx.Done():
				return
			}
		}
		timeout := time.Duration(setting.IntervalMS) * time.Millisecond / 2
		if timeout < 250*time.Millisecond {
			timeout = 250 * time.Millisecond
		}
		if timeout > 10*time.Second {
			timeout = 10 * time.Second
		}
		collectCtx, cancel := context.WithTimeout(c.ctx, timeout)
		updateErr := c.collectOnce(collectCtx, name)
		cancel()
		if updateErr != nil && !errors.Is(updateErr, context.Canceled) && c.logger != nil {
			c.logger.Warn("publish metric sample failed", "metric", name, "error", updateErr.Error())
		}
		timer := time.NewTimer(time.Duration(setting.IntervalMS) * time.Millisecond)
		select {
		case <-timer.C:
		case <-changed:
		case <-c.variables[name].resample:
		case <-c.ctx.Done():
			timer.Stop()
			return
		}
		timer.Stop()
	}
}

func (c *Controller) workerSetting(name Name) (SettingV1, <-chan struct{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return SettingV1{}, nil, false
	}
	setting, ok := settingFor(c.config, name)
	if c.configUpdating {
		setting.Enabled = false
	}
	return setting, c.changed, ok
}

func (c *Controller) ioSetting(name Name) (SettingV1, <-chan struct{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	setting, ok := settingFor(c.config, name)
	return setting, c.changed, ok && !c.closed && !c.configUpdating
}

func loadConfig(store *keystore.Store, platform string) (ConfigV1, error) {
	value, err := DefaultConfig(platform)
	if err != nil {
		return ConfigV1{}, err
	}
	found, err := store.Load("metrics.json", &value)
	if err != nil {
		return ConfigV1{}, fmt.Errorf("load metrics config: %w", err)
	}
	if value.Platform != platform {
		return ConfigV1{}, fmt.Errorf("load metrics config: stored platform %q does not match %q", value.Platform, platform)
	}
	if err := value.Validate(); err != nil {
		return ConfigV1{}, fmt.Errorf("load metrics config: %w", err)
	}
	if len(value.Settings) != len(Definitions(platform)) {
		return ConfigV1{}, errors.New("load metrics config: settings do not cover every platform metric")
	}
	if !found {
		if err := store.Save("metrics.json", value); err != nil {
			return ConfigV1{}, fmt.Errorf("initialize metrics config: %w", err)
		}
	}
	return value, nil
}

func settingFor(config ConfigV1, name Name) (SettingV1, bool) {
	for _, setting := range config.Settings {
		if setting.Metric == string(name) {
			return setting, true
		}
	}
	return SettingV1{}, false
}

func cloneConfig(value ConfigV1) ConfigV1 {
	value.Settings = append([]SettingV1(nil), value.Settings...)
	value.NotificationChannels = append([]string{}, value.NotificationChannels...)
	return value
}

func truncateError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "metric collection failed"
	}
	if len(value) > 1024 {
		value = value[:1024]
		for !utf8.ValidString(value) {
			value = value[:len(value)-1]
		}
	}
	return value
}
