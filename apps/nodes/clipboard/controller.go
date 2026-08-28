package clipboard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

type ControllerConfig struct {
	Node    *node.Node
	Store   *keystore.Store
	Adapter Adapter
	Now     func() time.Time
}

type Controller struct {
	node    *node.Node
	store   *keystore.Store
	adapter Adapter
	now     func() time.Time

	mu             sync.Mutex
	config         ConfigV1
	history        *historyStore
	pending        map[string]TextEventV1
	pendingOrder   []string
	recentIDs      *recentSet
	recentHashes   *recentSet
	suppressed     *recentSet
	events         *resource.Stream
	status         StatusV1
	statusVariable *resource.Variable
	configVariable *resource.Variable
	changed        chan struct{}
	registered     []protocol.ResourceID
	closed         bool

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func Register(config ControllerConfig) (*Controller, error) {
	if config.Node == nil || config.Store == nil {
		return nil, errors.New("clipboard controller requires a node and durable store")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	current, err := loadConfig(config.Store)
	if err != nil {
		return nil, err
	}
	history, err := loadHistory(config.Store, current, config.Now())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	value := &Controller{
		node: config.Node, store: config.Store, adapter: config.Adapter, now: config.Now,
		config: current, history: history, pending: make(map[string]TextEventV1),
		recentIDs: newRecentSet(2048, 10*time.Minute), recentHashes: newRecentSet(2048, 10*time.Minute),
		suppressed: newRecentSet(128, 10*time.Second), changed: make(chan struct{}), ctx: ctx, cancel: cancel,
	}
	value.status = value.statusLocked()
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
	if config.Adapter != nil {
		value.wg.Add(1)
		go value.runWatcher()
	}
	return value, nil
}

func (c *Controller) Config() ConfigV1 {
	if c == nil {
		return ConfigV1{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return cloneConfig(c.config)
}

func (c *Controller) Status() StatusV1 {
	if c == nil {
		return StatusV1{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

func (c *Controller) History() []HistoryEntry {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.history.snapshot()
}

func (c *Controller) ApplyConfig(ctx context.Context, request ConfigUpdateV1) (ConfigV1, error) {
	if ctx == nil {
		return ConfigV1{}, errors.New("clipboard config update context is required")
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

func (c *Controller) SendText(ctx context.Context, text string) (DecisionV1, error) {
	if ctx == nil {
		return DecisionV1{}, errors.New("clipboard send context is required")
	}
	request := SendV1{Version: 1, Text: text}
	if err := request.Validate(); err != nil {
		return DecisionV1{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return DecisionV1{}, errors.New("clipboard controller is closed")
	}
	if !c.config.Enabled {
		return c.ignoredLocked("clipboard sync is disabled", "", "", len([]byte(text))), nil
	}
	if len([]byte(text)) > c.config.MaxInlineBytes {
		return DecisionV1{}, fmt.Errorf("clipboard text exceeds configured inline limit of %d bytes", c.config.MaxInlineBytes)
	}
	now := c.now().UTC()
	hash := textHash(text)
	if c.suppressed.Consume(hash, now) {
		return c.ignoredLocked("local observation matches a recently applied remote event", "", hash, len([]byte(text))), nil
	}
	if c.recentHashes.Contains(hash, now) {
		return c.ignoredLocked("clipboard content is a recent duplicate", "", hash, len([]byte(text))), nil
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		return DecisionV1{}, err
	}
	event := TextEventV1{
		Version: 1, EventID: id.String(), OriginNodeID: formatNodeID(c.node.ID()), CreatedAtUnixMS: now.UnixMilli(),
		ContentType: "text/plain; charset=utf-8", Text: text, SizeBytes: len([]byte(text)), SHA256: hash,
	}
	payload, err := protocol.EncodeJSONPayload(&event, protocol.DefaultMaxPayload)
	if err != nil {
		return DecisionV1{}, err
	}
	if _, err := c.events.Publish(payload); err != nil {
		return DecisionV1{}, err
	}
	c.recentIDs.Add(event.EventID, now)
	c.recentHashes.Add(event.SHA256, now)
	decision := c.decisionLocked("published", event, "")
	if err := c.history.add(c.config, event, "published", now); err != nil {
		c.setErrorLocked(err)
		return decision, err
	}
	if err := c.publishStatusLocked(); err != nil {
		return decision, err
	}
	return decision, nil
}

func (c *Controller) Receive(ctx context.Context, peer protocol.NodeID, event TextEventV1) (DecisionV1, error) {
	if ctx == nil {
		return DecisionV1{}, errors.New("clipboard receive context is required")
	}
	if err := peer.Validate(); err != nil {
		return DecisionV1{}, err
	}
	if err := event.Validate(); err != nil {
		return DecisionV1{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return DecisionV1{}, errors.New("clipboard controller is closed")
	}
	if !c.config.Enabled {
		return c.ignoredLocked("clipboard sync is disabled", event.EventID, event.SHA256, event.SizeBytes), nil
	}
	if event.OriginNodeID != formatNodeID(peer) || peer == c.node.ID() || !c.receivesFromLocked(peer) {
		return c.ignoredLocked("clipboard event source is not an enabled peer", event.EventID, event.SHA256, event.SizeBytes), nil
	}
	if event.SizeBytes > c.config.MaxInlineBytes {
		return DecisionV1{}, fmt.Errorf("clipboard event exceeds configured inline limit of %d bytes", c.config.MaxInlineBytes)
	}
	now := c.now().UTC()
	if c.recentIDs.Contains(event.EventID, now) || c.recentHashes.Contains(event.SHA256, now) {
		return c.ignoredLocked("clipboard event is a recent duplicate", event.EventID, event.SHA256, event.SizeBytes), nil
	}
	if c.config.AutoApply {
		if c.adapter == nil {
			return DecisionV1{}, errors.New("clipboard auto-apply requires a platform adapter")
		}
		if err := c.adapter.WriteText(ctx, event.Text); err != nil {
			safe := redactClipboardError(err, event.Text)
			c.setErrorLocked(safe)
			_ = c.publishStatusLocked()
			return DecisionV1{}, safe
		}
		c.suppressed.Add(event.SHA256, now)
		c.recentIDs.Add(event.EventID, now)
		c.recentHashes.Add(event.SHA256, now)
		decision := c.decisionLocked("applied", event, "")
		if err := c.history.add(c.config, event, "applied", now); err != nil {
			c.setErrorLocked(err)
			return decision, err
		}
		return decision, c.publishStatusLocked()
	}
	if len(c.pendingOrder) == MaxPendingEvents {
		oldest := c.pendingOrder[0]
		c.pendingOrder = c.pendingOrder[1:]
		delete(c.pending, oldest)
	}
	c.pending[event.EventID] = event
	c.pendingOrder = append(c.pendingOrder, event.EventID)
	c.recentIDs.Add(event.EventID, now)
	c.recentHashes.Add(event.SHA256, now)
	decision := c.decisionLocked("pending", event, "")
	return decision, c.publishStatusLocked()
}

func (c *Controller) ApplyPending(ctx context.Context, eventID string) (DecisionV1, error) {
	if ctx == nil {
		return DecisionV1{}, errors.New("clipboard apply context is required")
	}
	request := ApplyV1{Version: 1, EventID: eventID}
	if err := request.Validate(); err != nil {
		return DecisionV1{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	event, ok := c.pending[eventID]
	if !ok {
		return DecisionV1{}, errors.New("clipboard pending event was not found")
	}
	if c.adapter == nil {
		return DecisionV1{}, errors.New("clipboard apply requires a platform adapter")
	}
	if err := c.adapter.WriteText(ctx, event.Text); err != nil {
		safe := redactClipboardError(err, event.Text)
		c.setErrorLocked(safe)
		_ = c.publishStatusLocked()
		return DecisionV1{}, safe
	}
	now := c.now().UTC()
	delete(c.pending, eventID)
	c.removePendingOrderLocked(eventID)
	c.suppressed.Add(event.SHA256, now)
	decision := c.decisionLocked("applied", event, "")
	if err := c.history.add(c.config, event, "applied", now); err != nil {
		c.setErrorLocked(err)
		return decision, err
	}
	return decision, c.publishStatusLocked()
}

func (c *Controller) ClearHistory() (ClearResultV1, error) {
	if c == nil {
		return ClearResultV1{}, errors.New("clipboard controller is unavailable")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ClearResultV1{}, errors.New("clipboard controller is closed")
	}
	removed, err := c.history.clear()
	if err != nil {
		return ClearResultV1{}, err
	}
	c.status.LastAction = "history_cleared"
	if err := c.publishStatusLocked(); err != nil {
		return ClearResultV1{}, err
	}
	return ClearResultV1{Version: 1, Removed: removed}, nil
}

func (c *Controller) RecordError(err error) {
	if c == nil || err == nil {
		return
	}
	c.mu.Lock()
	c.setErrorLocked(err)
	_ = c.publishStatusLocked()
	c.mu.Unlock()
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
		if c.adapter != nil {
			closeErr = c.adapter.Close()
		}
		if err := c.removeRegistered(); err != nil && closeErr == nil {
			closeErr = err
		}
	})
	return closeErr
}

func (c *Controller) registerResources() error {
	owner := c.node.ID()
	var err error
	c.events, err = resource.NewStream(resource.StreamDescriptor(
		protocol.ResourceID{Owner: owner, Name: ResourceEvents}, "application/json", SchemaTextEventV1,
		"clipboard.events.read", protocol.DefaultMaxPayload,
	))
	if err != nil {
		return err
	}
	if err := c.register(c.events); err != nil {
		return err
	}
	statusPayload, err := protocol.EncodeJSONPayload(&c.status, protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	c.statusVariable, err = resource.NewVariable(resource.VariableDescriptor(
		protocol.ResourceID{Owner: owner, Name: ResourceStatus}, "application/json", SchemaStatusV1,
		"clipboard.status.read", protocol.DefaultMaxPayload,
	), statusPayload)
	if err != nil {
		return err
	}
	if err := c.register(c.statusVariable); err != nil {
		return err
	}
	configPayload, err := protocol.EncodeJSONPayload(&c.config, protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	c.configVariable, err = resource.NewVariable(resource.VariableDescriptor(
		protocol.ResourceID{Owner: owner, Name: ResourceConfig}, "application/json", SchemaConfigV1,
		"clipboard.config.read", protocol.DefaultMaxPayload,
	), configPayload)
	if err != nil {
		return err
	}
	if err := c.register(c.configVariable); err != nil {
		return err
	}
	commands := []struct {
		name, inputSchema, outputSchema string
		permission                      string
		handler                         resource.CommandHandler
	}{
		{ResourceConfigUpdate, SchemaConfigUpdateV1, SchemaConfigV1, "clipboard.config.write", c.updateConfig},
		{CommandSend, SchemaSendV1, SchemaDecisionV1, "clipboard.send", c.sendCommand},
		{CommandApply, SchemaApplyV1, SchemaDecisionV1, "clipboard.apply", c.applyCommand},
		{CommandHistoryClear, SchemaClearV1, SchemaClearResultV1, "clipboard.history.clear", c.clearCommand},
	}
	for _, definition := range commands {
		command, err := resource.NewCommand(resource.CommandDescriptorSchemas(
			protocol.ResourceID{Owner: owner, Name: definition.name}, "application/json", definition.inputSchema, definition.outputSchema,
			definition.permission, protocol.DefaultMaxPayload,
		), definition.handler)
		if err != nil {
			return err
		}
		if err := c.register(command); err != nil {
			return err
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

func (c *Controller) sendCommand(ctx context.Context, input []byte) ([]byte, error) {
	var request SendV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	decision, err := c.SendText(ctx, request.Text)
	if err != nil {
		return nil, err
	}
	return protocol.EncodeJSONPayload(&decision, protocol.DefaultMaxPayload)
}

func (c *Controller) applyCommand(ctx context.Context, input []byte) ([]byte, error) {
	var request ApplyV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	decision, err := c.ApplyPending(ctx, request.EventID)
	if err != nil {
		return nil, err
	}
	return protocol.EncodeJSONPayload(&decision, protocol.DefaultMaxPayload)
}

func (c *Controller) clearCommand(_ context.Context, input []byte) ([]byte, error) {
	var request ClearV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	result, err := c.ClearHistory()
	if err != nil {
		return nil, err
	}
	return protocol.EncodeJSONPayload(&result, protocol.DefaultMaxPayload)
}

func (c *Controller) updateConfig(_ context.Context, input []byte) ([]byte, error) {
	var request ConfigUpdateV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errors.New("clipboard controller is closed")
	}
	if request.ExpectedRevision != c.config.Revision {
		return nil, fmt.Errorf("clipboard config revision conflict: current %d", c.config.Revision)
	}
	if c.config.Revision == ^uint64(0) {
		return nil, errors.New("clipboard config revision exhausted")
	}
	next := configFromUpdate(c.config.Revision+1, request)
	if err := next.Validate(); err != nil {
		return nil, err
	}
	payload, err := protocol.EncodeJSONPayload(&next, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, err
	}
	previousConfig := cloneConfig(c.config)
	previousHistory := c.history.snapshot()
	if err := c.history.normalize(next, c.now()); err != nil {
		return nil, err
	}
	if err := c.store.Save("clipboard.json", next); err != nil {
		c.history.entries = previousHistory
		_ = c.history.persist()
		return nil, fmt.Errorf("persist clipboard config: %w", err)
	}
	if _, err := c.configVariable.Set(payload); err != nil {
		_ = c.store.Save("clipboard.json", previousConfig)
		c.history.entries = previousHistory
		_ = c.history.persist()
		return nil, fmt.Errorf("publish clipboard config: %w", err)
	}
	c.config = next
	close(c.changed)
	c.changed = make(chan struct{})
	c.status.LastAction = "config_updated"
	if err := c.publishStatusLocked(); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Controller) runWatcher() {
	defer c.wg.Done()
	for {
		c.mu.Lock()
		config := cloneConfig(c.config)
		changed := c.changed
		closed := c.closed
		c.mu.Unlock()
		if closed {
			return
		}
		if !config.Enabled || !config.AutoWatch {
			select {
			case <-changed:
				continue
			case <-c.ctx.Done():
				return
			}
		}
		watchCtx, cancel := context.WithCancel(c.ctx)
		events, errorsOut, err := c.adapter.WatchText(watchCtx)
		if err != nil {
			cancel()
			c.RecordError(errors.New("clipboard platform watcher could not start"))
			if !waitController(c.ctx, time.Second) {
				return
			}
			continue
		}
		watching := true
		for watching {
			select {
			case observation, ok := <-events:
				if !ok {
					watching = false
					continue
				}
				if _, err := c.SendText(c.ctx, observation.Text); err != nil {
					c.RecordError(fmt.Errorf("publish clipboard observation: %w", redactClipboardError(err, observation.Text)))
				}
			case err, ok := <-errorsOut:
				if ok && err != nil {
					c.RecordError(errors.New("clipboard platform watcher failed"))
				}
				watching = false
			case <-changed:
				watching = false
			case <-c.ctx.Done():
				cancel()
				return
			}
		}
		cancel()
	}
}

func (c *Controller) decisionLocked(action string, event TextEventV1, reason string) DecisionV1 {
	decision := DecisionV1{
		Version: 1, Action: action, EventID: event.EventID, OriginNode: event.OriginNodeID,
		SizeBytes: event.SizeBytes, HashPrefix: hashPrefix(event.SHA256), Reason: reason, TimestampMS: c.now().UTC().UnixMilli(),
	}
	c.status.LastAction = action
	c.status.LastEventID = event.EventID
	c.status.LastSizeBytes = event.SizeBytes
	c.status.LastHashPrefix = hashPrefix(event.SHA256)
	c.status.LastError = ""
	return decision
}

func (c *Controller) ignoredLocked(reason, eventID, hash string, size int) DecisionV1 {
	event := TextEventV1{EventID: eventID, SHA256: hash, SizeBytes: size}
	decision := c.decisionLocked("ignored", event, reason)
	_ = c.publishStatusLocked()
	return decision
}

func (c *Controller) publishStatusLocked() error {
	c.status = c.statusLocked()
	payload, err := protocol.EncodeJSONPayload(&c.status, protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	if c.statusVariable != nil {
		_, err = c.statusVariable.Set(payload)
	}
	return err
}

func (c *Controller) statusLocked() StatusV1 {
	historyCount, historyBytes := c.history.stats()
	status := c.status
	status.Version = 1
	status.ConfigRevision = c.config.Revision
	status.Enabled = c.config.Enabled
	status.AutoWatch = c.config.AutoWatch
	status.AutoApply = c.config.AutoApply
	status.PeerCount = len(c.config.Peers)
	status.PendingCount = len(c.pending)
	status.HistoryCount = historyCount
	status.HistoryBodyBytes = historyBytes
	status.ObservedAtUnixMS = c.now().UTC().UnixMilli()
	status.PendingEventID = ""
	status.PendingSizeBytes = 0
	status.PendingHash = ""
	if len(c.pendingOrder) > 0 {
		pending := c.pending[c.pendingOrder[len(c.pendingOrder)-1]]
		status.PendingEventID = pending.EventID
		status.PendingSizeBytes = pending.SizeBytes
		status.PendingHash = hashPrefix(pending.SHA256)
	}
	return status
}

func (c *Controller) receivesFromLocked(peer protocol.NodeID) bool {
	wanted := formatNodeID(peer)
	for _, configured := range c.config.Peers {
		if configured.NodeID == wanted {
			return configured.Receive
		}
	}
	return false
}

func (c *Controller) removePendingOrderLocked(eventID string) {
	for index, current := range c.pendingOrder {
		if current == eventID {
			c.pendingOrder = append(c.pendingOrder[:index], c.pendingOrder[index+1:]...)
			return
		}
	}
}

func (c *Controller) setErrorLocked(err error) {
	if err == nil {
		return
	}
	message := strings.Join(strings.Fields(err.Error()), " ")
	if len(message) > 1024 {
		message = message[:1024]
	}
	c.status.LastError = message
}

func redactClipboardError(err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	for _, secret := range secrets {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[clipboard text redacted]")
		}
	}
	return errors.New(message)
}

func loadConfig(store *keystore.Store) (ConfigV1, error) {
	value := DefaultConfig()
	found, err := store.Load("clipboard.json", &value)
	if err != nil {
		return ConfigV1{}, fmt.Errorf("load clipboard config: %w", err)
	}
	if err := value.Validate(); err != nil {
		return ConfigV1{}, fmt.Errorf("load clipboard config: %w", err)
	}
	if !found {
		if err := store.Save("clipboard.json", value); err != nil {
			return ConfigV1{}, fmt.Errorf("initialize clipboard config: %w", err)
		}
	}
	return value, nil
}

func waitController(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
