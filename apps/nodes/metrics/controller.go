package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

var ErrUnsupported = errors.New("metric is unsupported on this platform")

type Collector interface {
	Collect(context.Context, Name) (string, error)
}

type ApplyResult struct {
	Applied bool
	Value   string
}

type Actuator interface {
	Set(context.Context, Name, string) (ApplyResult, error)
}

type ControllerConfig struct {
	Node      *node.Node
	Store     *keystore.Store
	Platform  string
	Collector Collector
	Actuator  Actuator
	Logger    *slog.Logger
	Now       func() time.Time
}

type Controller struct {
	node      *node.Node
	store     *keystore.Store
	collector Collector
	actuator  Actuator
	logger    *slog.Logger
	now       func() time.Time

	mu             sync.RWMutex
	config         ConfigV1
	configVariable *resource.Variable
	variables      map[Name]*resource.Variable
	samples        map[Name]SampleV1
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
		node: config.Node, store: config.Store, collector: config.Collector, actuator: config.Actuator, logger: config.Logger, now: config.Now,
		config: current, variables: make(map[Name]*resource.Variable), samples: make(map[Name]SampleV1), changed: make(chan struct{}),
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

func (c *Controller) Sample(name Name) (SampleV1, bool) {
	if c == nil {
		return SampleV1{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.samples[name]
	return value, ok
}

func (c *Controller) Samples() []SampleV1 {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	result := make([]SampleV1, 0, len(c.samples))
	for _, sample := range c.samples {
		result = append(result, sample)
	}
	c.mu.RUnlock()
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

func (c *Controller) Update(name Name, value string, collectErr error) error {
	if c == nil {
		return errors.New("metrics controller is closed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("metrics controller is closed")
	}
	return c.updateLocked(name, value, collectErr)
}

func (c *Controller) Close() error {
	if c == nil {
		return nil
	}
	var closeErr error
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		close(c.changed)
		c.mu.Unlock()
		c.cancel()
		c.wg.Wait()
		closeErr = c.removeRegistered()
	})
	return closeErr
}

func (c *Controller) registerResources() error {
	owner := c.node.ID()
	configPayload, err := protocol.EncodeJSONPayload(&c.config, protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	c.configVariable, err = resource.NewVariable(resource.VariableDescriptor(
		protocol.ResourceID{Owner: owner, Name: ResourceConfig}, "application/json", SchemaConfigV1,
		"metrics.config.read", protocol.DefaultMaxPayload,
	), configPayload)
	if err != nil {
		return err
	}
	if err := c.register(c.configVariable); err != nil {
		return err
	}
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
		now := c.now().UTC().UnixMilli()
		sample := SampleV1{
			Version: 1, Metric: string(definition.Name), Unit: definition.Unit, Status: "unavailable",
			ObservedAtUnixMS: now, Error: "metric has not produced a sample",
		}
		payload, err := protocol.EncodeJSONPayload(&sample, protocol.DefaultMaxPayload)
		if err != nil {
			return err
		}
		variable, err := resource.NewVariable(resource.VariableDescriptor(
			protocol.ResourceID{Owner: owner, Name: ResourceName(definition.Name)}, "application/json", SchemaSampleV1,
			"metrics.read", protocol.DefaultMaxPayload,
		), payload)
		if err != nil {
			return err
		}
		if err := c.register(variable); err != nil {
			return err
		}
		c.variables[definition.Name] = variable
		c.samples[definition.Name] = sample
		if definition.Controllable {
			name := definition.Name
			command, err := resource.NewCommand(resource.CommandDescriptorSchemas(
				protocol.ResourceID{Owner: owner, Name: CommandName(name)}, "application/json", SchemaControlV1, SchemaControlResultV1,
				"metrics.control", protocol.DefaultMaxPayload,
			), func(ctx context.Context, input []byte) ([]byte, error) { return c.control(ctx, name, input) })
			if err != nil {
				return err
			}
			if err := c.register(command); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) register(value resource.Resource) error {
	if err := c.node.Registry().Register(value); err != nil {
		return err
	}
	c.registered = append(c.registered, value.Descriptor().ID)
	return nil
}

func (c *Controller) removeRegistered() error {
	var result error
	for index := len(c.registered) - 1; index >= 0; index-- {
		if err := c.node.Registry().Remove(c.registered[index]); err != nil && !errors.Is(err, resource.ErrNotFound) && result == nil {
			result = err
		}
	}
	c.registered = nil
	return result
}

func (c *Controller) updateLocked(name Name, value string, collectErr error) error {
	definition, ok := DefinitionFor(name)
	variable := c.variables[name]
	if !ok || variable == nil || !definition.Platforms[c.config.Platform] {
		return ErrUnsupported
	}
	previous := c.samples[name]
	now := c.now().UTC().UnixMilli()
	next := SampleV1{Version: 1, Metric: string(name), Unit: definition.Unit, ObservedAtUnixMS: now}
	if collectErr == nil {
		value = strings.TrimSpace(value)
		if err := ValidateValue(name, value); err != nil {
			return err
		}
		next.Value = value
		next.Status = "fresh"
		next.SampledAtUnixMS = now
	} else if previous.SampledAtUnixMS > 0 {
		next.Value = previous.Value
		next.Status = "stale"
		next.SampledAtUnixMS = previous.SampledAtUnixMS
		next.Error = truncateError(collectErr.Error())
	} else {
		next.Status = "unavailable"
		next.Error = truncateError(collectErr.Error())
	}
	payload, err := protocol.EncodeJSONPayload(&next, protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	if _, err := variable.Set(payload); err != nil {
		return err
	}
	c.samples[name] = next
	return nil
}

func (c *Controller) control(ctx context.Context, name Name, input []byte) ([]byte, error) {
	var request ControlV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	if err := ValidateValue(name, request.Value); err != nil {
		return nil, err
	}
	c.mu.RLock()
	setting, ok := settingFor(c.config, name)
	actuator := c.actuator
	closed := c.closed
	c.mu.RUnlock()
	if closed {
		return nil, errors.New("metrics controller is closed")
	}
	if !ok || !setting.Enabled || !setting.Writable {
		return nil, errors.New("metric control is disabled by configuration")
	}
	if actuator == nil {
		return nil, ErrUnsupported
	}
	result, err := actuator.Set(ctx, name, strings.TrimSpace(request.Value))
	if err != nil {
		return nil, err
	}
	response := ControlResultV1{
		Version: 1, Metric: string(name), RequestedValue: strings.TrimSpace(request.Value), Accepted: true,
		Applied: result.Applied, AppliedValue: strings.TrimSpace(result.Value),
	}
	if response.Applied {
		if err := c.Update(name, response.AppliedValue, nil); err != nil {
			return nil, fmt.Errorf("publish applied metric state: %w", err)
		}
	}
	return protocol.EncodeJSONPayload(&response, protocol.DefaultMaxPayload)
}

func (c *Controller) updateConfig(_ context.Context, input []byte) ([]byte, error) {
	var request ConfigUpdateV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errors.New("metrics controller is closed")
	}
	if request.ExpectedRevision != c.config.Revision {
		return nil, fmt.Errorf("metrics config revision conflict: current %d", c.config.Revision)
	}
	if c.config.Revision == ^uint64(0) {
		return nil, errors.New("metrics config revision exhausted")
	}
	settings := append([]SettingV1(nil), request.Settings...)
	sort.Slice(settings, func(i, j int) bool { return settings[i].Metric < settings[j].Metric })
	channels := append([]string(nil), request.NotificationChannels...)
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
	previous := cloneConfig(c.config)
	if err := c.store.Save("metrics.json", next); err != nil {
		return nil, fmt.Errorf("persist metrics config: %w", err)
	}
	if _, err := c.configVariable.Set(payload); err != nil {
		if rollbackErr := c.store.Save("metrics.json", previous); rollbackErr != nil {
			return nil, fmt.Errorf("publish metrics config: %v; rollback persistence: %w", err, rollbackErr)
		}
		return nil, fmt.Errorf("publish metrics config: %w", err)
	}
	c.config = next
	close(c.changed)
	c.changed = make(chan struct{})
	for _, setting := range settings {
		if !setting.Enabled {
			if err := c.updateLocked(Name(setting.Metric), "", errors.New("metric is disabled")); err != nil {
				return nil, fmt.Errorf("publish disabled metric %s: %w", setting.Metric, err)
			}
		}
	}
	return payload, nil
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
		value, err := c.collector.Collect(collectCtx, name)
		cancel()
		if updateErr := c.Update(name, value, err); updateErr != nil && !errors.Is(updateErr, context.Canceled) && c.logger != nil {
			c.logger.Warn("publish metric sample failed", "metric", name, "error", updateErr.Error())
		}
		timer := time.NewTimer(time.Duration(setting.IntervalMS) * time.Millisecond)
		select {
		case <-timer.C:
		case <-changed:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-c.ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

func (c *Controller) workerSetting(name Name) (SettingV1, <-chan struct{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return SettingV1{}, nil, false
	}
	setting, ok := settingFor(c.config, name)
	return setting, c.changed, ok
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
	value.NotificationChannels = append([]string(nil), value.NotificationChannels...)
	return value
}

func truncateError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "metric collection failed"
	}
	if len(value) > 1024 {
		value = value[:1024]
	}
	return value
}
