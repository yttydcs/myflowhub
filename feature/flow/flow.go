package flow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

type Config struct {
	Node             *node.Node
	Store            *keystore.Store
	MaxDefinitions   int
	MaxRuns          int
	MaxActive        int
	MaxActivePerFlow int
	MaxNodeOutput    int
	MaxTotalOutput   int
	Now              func() time.Time
}

type persistedState struct {
	Version     int                         `json:"version"`
	Revision    uint64                      `json:"revision"`
	Definitions []protocol.FlowDefinitionV1 `json:"definitions"`
	Runs        []protocol.FlowRunSummaryV1 `json:"runs"`
}

type runControl struct {
	cancel     context.CancelFunc
	sequence   uint64
	delegation command.Delegation
}

type Controller struct {
	mu               sync.Mutex
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	node             *node.Node
	store            *keystore.Store
	maxDefinitions   int
	maxRuns          int
	maxActive        int
	maxActivePerFlow int
	maxNodeOutput    int
	maxTotalOutput   int
	now              func() time.Time
	revision         uint64
	definitions      map[string]protocol.FlowDefinitionV1
	runs             map[string]protocol.FlowRunSummaryV1
	dedupe           map[string]string
	controls         map[string]*runControl
	definitionFeed   *resource.Variable
	runFeed          *resource.Variable
	events           *resource.Stream
}

func Register(config Config) (*Controller, error) {
	if config.Node == nil || config.Store == nil {
		return nil, errors.New("flow node and durable store are required")
	}
	if config.MaxDefinitions <= 0 {
		config.MaxDefinitions = 256
	}
	if config.MaxRuns <= 0 {
		config.MaxRuns = 256
	}
	if config.MaxActive <= 0 {
		config.MaxActive = 16
	}
	if config.MaxActivePerFlow <= 0 {
		config.MaxActivePerFlow = 4
	}
	if config.MaxNodeOutput <= 0 {
		config.MaxNodeOutput = 256 << 10
	}
	if config.MaxTotalOutput <= 0 {
		config.MaxTotalOutput = 1 << 20
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.MaxDefinitions > protocol.MaxItems || config.MaxRuns > protocol.MaxItems || config.MaxActive > config.MaxRuns || config.MaxActivePerFlow > config.MaxActive || config.MaxNodeOutput > protocol.DefaultMaxPayload || config.MaxTotalOutput < config.MaxNodeOutput {
		return nil, errors.New("flow limits are invalid")
	}
	ctx, cancel := context.WithCancel(context.Background())
	value := &Controller{
		ctx: ctx, cancel: cancel, node: config.Node, store: config.Store, maxDefinitions: config.MaxDefinitions, maxRuns: config.MaxRuns,
		maxActive: config.MaxActive, maxActivePerFlow: config.MaxActivePerFlow, maxNodeOutput: config.MaxNodeOutput, maxTotalOutput: config.MaxTotalOutput,
		now: config.Now, definitions: make(map[string]protocol.FlowDefinitionV1), runs: make(map[string]protocol.FlowRunSummaryV1),
		dedupe: make(map[string]string), controls: make(map[string]*runControl),
	}
	if err := value.load(); err != nil {
		cancel()
		return nil, err
	}
	definitionsPayload, err := value.definitionsPayloadLocked()
	if err != nil {
		cancel()
		return nil, err
	}
	runsPayload, err := value.runsPayloadLocked()
	if err != nil {
		cancel()
		return nil, err
	}
	owner := config.Node.ID()
	value.definitionFeed, err = resource.NewVariable(resource.VariableDescriptor(protocol.ResourceID{Owner: owner, Name: protocol.BuiltinFlowDefinitions}, "application/json", protocol.SchemaFlowDefinitionsV1, "flow.read", protocol.DefaultMaxPayload), definitionsPayload)
	if err != nil {
		cancel()
		return nil, err
	}
	value.runFeed, err = resource.NewVariable(resource.VariableDescriptor(protocol.ResourceID{Owner: owner, Name: protocol.BuiltinFlowRuns}, "application/json", protocol.SchemaFlowRunsV1, "flow.read", protocol.DefaultMaxPayload), runsPayload)
	if err != nil {
		cancel()
		return nil, err
	}
	value.events, err = resource.NewStream(resource.StreamDescriptor(protocol.ResourceID{Owner: owner, Name: protocol.BuiltinFlowEvents}, "application/json", protocol.SchemaFlowEventV1, "flow.read", protocol.DefaultMaxPayload))
	if err != nil {
		cancel()
		return nil, err
	}
	commands := []struct {
		name, inputSchema, outputSchema, permission string
		handler                                     resource.CommandHandler
	}{
		{protocol.BuiltinFlowCreate, protocol.SchemaFlowDefinitionV1, protocol.SchemaFlowDefinitionV1, "flow.write", value.create},
		{protocol.BuiltinFlowUpdate, protocol.SchemaFlowDefinitionV1, protocol.SchemaFlowDefinitionV1, "flow.write", value.update},
		{protocol.BuiltinFlowRun, protocol.SchemaFlowRunV1, protocol.SchemaFlowRunSummaryV1, "flow.run", value.run},
		{protocol.BuiltinFlowCancel, protocol.SchemaFlowCancelV1, protocol.SchemaFlowRunSummaryV1, "flow.cancel", value.cancelRun},
		{protocol.BuiltinFlowArchive, protocol.SchemaFlowArchiveV1, protocol.SchemaFlowArchiveV1, "flow.write", value.archive},
	}
	resources := []resource.Resource{value.definitionFeed, value.runFeed, value.events}
	for _, definition := range commands {
		current, err := resource.NewCommand(resource.CommandDescriptorSchemas(protocol.ResourceID{Owner: owner, Name: definition.name}, "application/json", definition.inputSchema, definition.outputSchema, definition.permission, protocol.DefaultMaxPayload), definition.handler)
		if err != nil {
			cancel()
			return nil, err
		}
		resources = append(resources, current)
	}
	registered := make([]protocol.ResourceID, 0, len(resources))
	for _, current := range resources {
		if err := config.Node.Registry().Register(current); err != nil {
			for index := len(registered) - 1; index >= 0; index-- {
				_ = config.Node.Registry().Remove(registered[index])
			}
			cancel()
			return nil, fmt.Errorf("register flow resource %s: %w", current.Descriptor().ID.Name, err)
		}
		registered = append(registered, current.Descriptor().ID)
	}
	value.wg.Add(1)
	go func() {
		defer value.wg.Done()
		select {
		case <-config.Node.Done():
			value.cancel()
		case <-value.ctx.Done():
		}
	}()
	return value, nil
}

func (c *Controller) Close() error {
	if c == nil {
		return nil
	}
	c.cancel()
	c.mu.Lock()
	controls := make([]*runControl, 0, len(c.controls))
	for id, control := range c.controls {
		controls = append(controls, control)
		c.finishRunLocked(id, "interrupted", "flow controller stopped")
	}
	c.mu.Unlock()
	for _, control := range controls {
		control.cancel()
	}
	c.wg.Wait()
	return nil
}

func (c *Controller) create(_ context.Context, input []byte) ([]byte, error) {
	var definition protocol.FlowDefinitionV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &definition); err != nil {
		return nil, err
	}
	if definition.Revision != 1 {
		return nil, errors.New("new flow definition revision must be 1")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.definitions[definition.FlowID]; exists {
		return nil, errors.New("flow definition already exists")
	}
	if len(c.definitions) >= c.maxDefinitions {
		return nil, errors.New("flow definition limit reached")
	}
	c.definitions[definition.FlowID] = definition
	if err := c.commitLocked(); err != nil {
		delete(c.definitions, definition.FlowID)
		return nil, err
	}
	return protocol.EncodeJSONPayload(&definition, protocol.DefaultMaxPayload)
}

func (c *Controller) update(_ context.Context, input []byte) ([]byte, error) {
	var definition protocol.FlowDefinitionV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &definition); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	current, exists := c.definitions[definition.FlowID]
	if !exists {
		return nil, errors.New("flow definition not found")
	}
	if definition.Revision != current.Revision+1 {
		return nil, fmt.Errorf("flow revision conflict: current %d", current.Revision)
	}
	if c.activeForFlowLocked(definition.FlowID) > 0 {
		return nil, errors.New("flow definition cannot change while runs are active")
	}
	c.definitions[definition.FlowID] = definition
	if err := c.commitLocked(); err != nil {
		c.definitions[definition.FlowID] = current
		return nil, err
	}
	return protocol.EncodeJSONPayload(&definition, protocol.DefaultMaxPayload)
}

func (c *Controller) archive(_ context.Context, input []byte) ([]byte, error) {
	var request protocol.FlowArchiveV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	current, exists := c.definitions[request.FlowID]
	if !exists {
		return protocol.EncodeJSONPayload(&request, protocol.DefaultMaxPayload)
	}
	if c.activeForFlowLocked(request.FlowID) > 0 {
		return nil, errors.New("active flow cannot be archived")
	}
	delete(c.definitions, request.FlowID)
	if err := c.commitLocked(); err != nil {
		c.definitions[request.FlowID] = current
		return nil, err
	}
	return protocol.EncodeJSONPayload(&request, protocol.DefaultMaxPayload)
}

func (c *Controller) run(ctx context.Context, input []byte) ([]byte, error) {
	delegation, ok := command.DelegationFromContext(ctx)
	if !ok {
		return nil, errors.New("flow run requires an authenticated command context")
	}
	initiator, ok := delegation.Subject()
	if !ok {
		return nil, errors.New("flow initiator is unavailable")
	}
	var request protocol.FlowRunV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	now := c.now().UTC()
	if request.DeadlineUnixMS <= now.UnixMilli() {
		return nil, errors.New("flow run deadline has expired")
	}
	c.mu.Lock()
	definition, exists := c.definitions[request.FlowID]
	if !exists || definition.Revision != request.FlowRevision {
		c.mu.Unlock()
		return nil, errors.New("flow definition or requested revision not found")
	}
	dedupeKey := strconv.FormatUint(uint64(initiator), 10) + "\x00" + request.FlowID + "\x00" + request.DedupeKey
	if existingID := c.dedupe[dedupeKey]; existingID != "" {
		existing := c.runs[existingID]
		c.mu.Unlock()
		return protocol.EncodeJSONPayload(&existing, protocol.DefaultMaxPayload)
	}
	if _, exists := c.runs[request.RunID]; exists {
		c.mu.Unlock()
		return nil, errors.New("flow run ID already exists")
	}
	if c.activeCountLocked() >= c.maxActive || c.activeForFlowLocked(request.FlowID) >= c.maxActivePerFlow {
		c.mu.Unlock()
		return nil, errors.New("flow active run limit reached")
	}
	if err := c.makeRunCapacityLocked(); err != nil {
		c.mu.Unlock()
		return nil, err
	}
	run := protocol.FlowRunSummaryV1{RunID: request.RunID, FlowID: request.FlowID, FlowRevision: request.FlowRevision, Initiator: strconv.FormatUint(uint64(initiator), 10), DedupeKey: request.DedupeKey, State: "queued", StartedUnixMS: now.UnixMilli()}
	runCtx, cancel := context.WithDeadline(c.ctx, time.UnixMilli(request.DeadlineUnixMS))
	control := &runControl{cancel: cancel, delegation: delegation}
	c.runs[request.RunID] = run
	c.dedupe[dedupeKey] = request.RunID
	c.controls[request.RunID] = control
	if err := c.commitLocked(); err != nil {
		delete(c.runs, request.RunID)
		delete(c.dedupe, dedupeKey)
		delete(c.controls, request.RunID)
		cancel()
		c.mu.Unlock()
		return nil, err
	}
	c.emitLocked(request.RunID, "queued", "", nil)
	c.wg.Add(1)
	c.mu.Unlock()
	go func() {
		defer c.wg.Done()
		defer cancel()
		c.execute(runCtx, request, definition, control)
	}()
	return protocol.EncodeJSONPayload(&run, protocol.DefaultMaxPayload)
}

func (c *Controller) cancelRun(ctx context.Context, input []byte) ([]byte, error) {
	delegation, ok := command.DelegationFromContext(ctx)
	if !ok {
		return nil, errors.New("flow cancel requires an authenticated command context")
	}
	initiator, _ := delegation.Subject()
	var request protocol.FlowCancelV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	c.mu.Lock()
	run, exists := c.runs[request.RunID]
	caller := strconv.FormatUint(uint64(initiator), 10)
	if !exists || (run.Initiator != caller && initiator != c.node.ID()) {
		c.mu.Unlock()
		return nil, errors.New("flow run not found for caller")
	}
	if terminal(run.State) {
		c.mu.Unlock()
		return protocol.EncodeJSONPayload(&run, protocol.DefaultMaxPayload)
	}
	control := c.controls[request.RunID]
	c.finishRunLocked(request.RunID, "cancelled", request.Reason)
	run = c.runs[request.RunID]
	c.mu.Unlock()
	if control != nil {
		control.cancel()
	}
	return protocol.EncodeJSONPayload(&run, protocol.DefaultMaxPayload)
}

func (c *Controller) execute(ctx context.Context, request protocol.FlowRunV1, definition protocol.FlowDefinitionV1, control *runControl) {
	c.mu.Lock()
	run := c.runs[request.RunID]
	if run.State != "queued" {
		c.mu.Unlock()
		return
	}
	run.State = "running"
	c.runs[request.RunID] = run
	if err := c.commitLocked(); err != nil {
		c.finishRunLocked(request.RunID, "failed", err.Error())
		c.mu.Unlock()
		return
	}
	c.emitLocked(request.RunID, "running", "", nil)
	c.mu.Unlock()

	outputs := make(map[string][]byte, len(definition.Nodes))
	totalOutput := 0
	predecessors := predecessorMap(definition)
	for _, definitionNode := range topologicalNodes(definition) {
		if err := ctx.Err(); err != nil {
			c.finishFromContext(request.RunID, err)
			return
		}
		c.mu.Lock()
		c.emitLocked(request.RunID, "running", definitionNode.ID, map[string]string{"status": "started"})
		c.mu.Unlock()
		output, err := c.executeNodeWithRetry(ctx, control.delegation, request, definitionNode, predecessors[definitionNode.ID], outputs)
		if err != nil {
			c.finishRun(request.RunID, "failed", fmt.Sprintf("node %s: %v", definitionNode.ID, err))
			return
		}
		if len(output) > c.maxNodeOutput || totalOutput > c.maxTotalOutput-len(output) {
			c.finishRun(request.RunID, "failed", fmt.Sprintf("node %s output exceeds flow limits", definitionNode.ID))
			return
		}
		outputs[definitionNode.ID] = append([]byte(nil), output...)
		totalOutput += len(output)
		c.mu.Lock()
		c.emitLocked(request.RunID, "running", definitionNode.ID, map[string]string{"status": "succeeded", "output_bytes": strconv.Itoa(len(output))})
		c.mu.Unlock()
	}
	c.finishRun(request.RunID, "succeeded", "")
}

func (c *Controller) executeNodeWithRetry(ctx context.Context, delegation command.Delegation, run protocol.FlowRunV1, definitionNode protocol.FlowNodeV1, predecessors []string, outputs map[string][]byte) ([]byte, error) {
	attempts := definitionNode.MaxAttempts
	if attempts == 0 {
		attempts = 1
	}
	backoff := time.Duration(definitionNode.RetryBackoffMS) * time.Millisecond
	for attempt := 1; ; attempt++ {
		output, err := c.executeNode(ctx, delegation, run, definitionNode, predecessors, outputs)
		if err == nil || attempt >= attempts || !retryable(err) {
			return output, err
		}
		c.mu.Lock()
		c.emitLocked(run.RunID, "running", definitionNode.ID, map[string]string{"status": "retrying", "attempt": strconv.Itoa(attempt + 1)})
		c.mu.Unlock()
		delay := backoff
		for step := 1; step < attempt && delay < 30*time.Second; step++ {
			delay *= 2
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, ctx.Err()
		}
	}
}

func (c *Controller) executeNode(ctx context.Context, delegation command.Delegation, run protocol.FlowRunV1, definitionNode protocol.FlowNodeV1, predecessors []string, outputs map[string][]byte) ([]byte, error) {
	resourceID := func() (protocol.ResourceID, error) {
		owner, err := strconv.ParseUint(definitionNode.Resource.OwnerNodeID, 10, 64)
		if err != nil {
			return protocol.ResourceID{}, err
		}
		return protocol.ResourceID{Owner: protocol.NodeID(owner), Name: definitionNode.Resource.Name}, nil
	}
	switch definitionNode.Kind {
	case "variable-read":
		id, err := resourceID()
		if err != nil {
			return nil, err
		}
		lease := time.Until(time.UnixMilli(run.DeadlineUnixMS))
		if lease < time.Second {
			lease = time.Second
		}
		current, err := c.node.SubscribeDelegated(ctx, delegation, id, lease, 4)
		if err != nil {
			return nil, err
		}
		defer current.Cancel()
		select {
		case event, ok := <-current.Events:
			if !ok || event.Kind != subscription.EventSnapshot {
				return nil, errors.New("variable resource did not return a snapshot")
			}
			return event.Value, nil
		case err := <-current.Errors:
			if err == nil {
				return nil, errors.New("variable subscription closed")
			}
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case "command-call":
		id, err := resourceID()
		if err != nil {
			return nil, err
		}
		input, err := nodeInput(definitionNode.Config, predecessors, outputs, run.Inputs)
		if err != nil {
			return nil, err
		}
		return c.node.InvokeDelegated(ctx, delegation, id, input)
	case "transform":
		return nodeInput(definitionNode.Config, predecessors, outputs, run.Inputs)
	default:
		return nil, errors.New("unsupported flow node kind")
	}
}

func (c *Controller) finishFromContext(runID string, err error) {
	state := "failed"
	if errors.Is(err, context.Canceled) {
		state = "cancelled"
	}
	c.finishRun(runID, state, err.Error())
}

func (c *Controller) finishRun(runID, state, message string) {
	c.mu.Lock()
	c.finishRunLocked(runID, state, message)
	c.mu.Unlock()
}

func (c *Controller) finishRunLocked(runID, state, message string) {
	run, exists := c.runs[runID]
	if !exists || terminal(run.State) {
		return
	}
	run.State = state
	run.FinishedUnixMS = c.now().UTC().UnixMilli()
	run.Error = truncate(message, 2048)
	c.runs[runID] = run
	_ = c.commitLocked()
	c.emitLocked(runID, state, "", map[string]string{"error": run.Error})
	delete(c.controls, runID)
}

func (c *Controller) emitLocked(runID, state, nodeID string, summary map[string]string) {
	control := c.controls[runID]
	if control == nil {
		return
	}
	control.sequence++
	if c.controls[runID] != nil {
		c.controls[runID].sequence = control.sequence
	}
	event := protocol.FlowEventV1{Version: 1, RunID: runID, Sequence: control.sequence, State: state, NodeID: nodeID, OccurredAtUnixMS: c.now().UTC().UnixMilli(), Summary: summary}
	payload, err := protocol.EncodeJSONPayload(&event, protocol.DefaultMaxPayload)
	if err == nil {
		_, _ = c.events.Publish(payload)
	}
}

func (c *Controller) load() error {
	state := persistedState{Version: 1, Revision: 1, Definitions: []protocol.FlowDefinitionV1{}, Runs: []protocol.FlowRunSummaryV1{}}
	found, err := c.store.Load("flow.json", &state)
	if err != nil {
		return fmt.Errorf("load flow state: %w", err)
	}
	if state.Version != 1 || state.Revision == 0 || len(state.Definitions) > c.maxDefinitions || len(state.Runs) > c.maxRuns {
		return errors.New("load flow state: unsupported version, revision, or count")
	}
	for _, definition := range state.Definitions {
		if err := definition.Validate(); err != nil {
			return fmt.Errorf("load flow definition: %w", err)
		}
		if _, exists := c.definitions[definition.FlowID]; exists {
			return errors.New("load flow state: duplicate definition")
		}
		c.definitions[definition.FlowID] = definition
	}
	changed := false
	now := c.now().UTC().UnixMilli()
	for _, run := range state.Runs {
		if err := run.Validate(); err != nil {
			return fmt.Errorf("load flow run: %w", err)
		}
		if _, exists := c.runs[run.RunID]; exists {
			return errors.New("load flow state: duplicate run")
		}
		if run.State == "queued" || run.State == "running" {
			run.State = "interrupted"
			run.FinishedUnixMS = now
			run.Error = "host restarted while run was active"
			changed = true
		}
		c.runs[run.RunID] = run
		c.dedupe[run.Initiator+"\x00"+run.FlowID+"\x00"+run.DedupeKey] = run.RunID
	}
	c.revision = state.Revision
	if !found || changed {
		if changed {
			c.revision++
		}
		if err := c.saveLocked(); err != nil {
			return fmt.Errorf("initialize flow state: %w", err)
		}
	}
	return nil
}

func (c *Controller) commitLocked() error {
	if c.revision == ^uint64(0) {
		return errors.New("flow state revision exhausted")
	}
	c.revision++
	if err := c.saveLocked(); err != nil {
		c.revision--
		return fmt.Errorf("persist flow state: %w", err)
	}
	definitions, err := c.definitionsPayloadLocked()
	if err != nil {
		return err
	}
	runs, err := c.runsPayloadLocked()
	if err != nil {
		return err
	}
	if c.definitionFeed != nil {
		if _, err := c.definitionFeed.Set(definitions); err != nil {
			return err
		}
	}
	if c.runFeed != nil {
		if _, err := c.runFeed.Set(runs); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) saveLocked() error {
	state := persistedState{Version: 1, Revision: c.revision, Definitions: make([]protocol.FlowDefinitionV1, 0, len(c.definitions)), Runs: make([]protocol.FlowRunSummaryV1, 0, len(c.runs))}
	for _, definition := range c.definitions {
		state.Definitions = append(state.Definitions, definition)
	}
	for _, run := range c.runs {
		state.Runs = append(state.Runs, run)
	}
	sort.Slice(state.Definitions, func(i, j int) bool { return state.Definitions[i].FlowID < state.Definitions[j].FlowID })
	sort.Slice(state.Runs, func(i, j int) bool { return state.Runs[i].RunID < state.Runs[j].RunID })
	return c.store.Save("flow.json", state)
}

func (c *Controller) definitionsPayloadLocked() ([]byte, error) {
	definitions := make([]protocol.FlowDefinitionSummaryV1, 0, len(c.definitions))
	for _, definition := range c.definitions {
		definitions = append(definitions, protocol.FlowDefinitionSummaryV1{FlowID: definition.FlowID, Revision: definition.Revision, Name: definition.Name, NodeCount: len(definition.Nodes), EdgeCount: len(definition.Edges)})
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].FlowID < definitions[j].FlowID })
	return protocol.EncodeJSONPayload(&protocol.FlowDefinitionsV1{Version: 1, Revision: c.revision, Definitions: definitions}, protocol.DefaultMaxPayload)
}

func (c *Controller) runsPayloadLocked() ([]byte, error) {
	runs := make([]protocol.FlowRunSummaryV1, 0, len(c.runs))
	for _, run := range c.runs {
		runs = append(runs, run)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].RunID < runs[j].RunID })
	return protocol.EncodeJSONPayload(&protocol.FlowRunsV1{Version: 1, Revision: c.revision, Runs: runs}, protocol.DefaultMaxPayload)
}

func (c *Controller) makeRunCapacityLocked() error {
	if len(c.runs) < c.maxRuns {
		return nil
	}
	terminalRuns := make([]protocol.FlowRunSummaryV1, 0)
	for _, run := range c.runs {
		if terminal(run.State) {
			terminalRuns = append(terminalRuns, run)
		}
	}
	sort.Slice(terminalRuns, func(i, j int) bool {
		if terminalRuns[i].FinishedUnixMS != terminalRuns[j].FinishedUnixMS {
			return terminalRuns[i].FinishedUnixMS < terminalRuns[j].FinishedUnixMS
		}
		return terminalRuns[i].RunID < terminalRuns[j].RunID
	})
	for len(c.runs) >= c.maxRuns && len(terminalRuns) > 0 {
		run := terminalRuns[0]
		terminalRuns = terminalRuns[1:]
		delete(c.runs, run.RunID)
		delete(c.dedupe, run.Initiator+"\x00"+run.FlowID+"\x00"+run.DedupeKey)
	}
	if len(c.runs) >= c.maxRuns {
		return errors.New("flow retained run limit reached")
	}
	return nil
}

func (c *Controller) activeCountLocked() int {
	count := 0
	for _, run := range c.runs {
		if run.State == "queued" || run.State == "running" {
			count++
		}
	}
	return count
}

func (c *Controller) activeForFlowLocked(flowID string) int {
	count := 0
	for _, run := range c.runs {
		if run.FlowID == flowID && (run.State == "queued" || run.State == "running") {
			count++
		}
	}
	return count
}

func topologicalNodes(definition protocol.FlowDefinitionV1) []protocol.FlowNodeV1 {
	nodes := make(map[string]protocol.FlowNodeV1, len(definition.Nodes))
	indegree := make(map[string]int, len(definition.Nodes))
	graph := make(map[string][]string, len(definition.Nodes))
	for _, current := range definition.Nodes {
		nodes[current.ID] = current
		indegree[current.ID] = 0
	}
	for _, edge := range definition.Edges {
		graph[edge.From] = append(graph[edge.From], edge.To)
		indegree[edge.To]++
	}
	ready := make([]string, 0)
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	result := make([]protocol.FlowNodeV1, 0, len(nodes))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		result = append(result, nodes[id])
		for _, next := range graph[id] {
			indegree[next]--
			if indegree[next] == 0 {
				ready = append(ready, next)
				sort.Strings(ready)
			}
		}
	}
	return result
}

func predecessorMap(definition protocol.FlowDefinitionV1) map[string][]string {
	result := make(map[string][]string, len(definition.Nodes))
	for _, edge := range definition.Edges {
		result[edge.To] = append(result[edge.To], edge.From)
	}
	for id := range result {
		sort.Strings(result[id])
	}
	return result
}

func nodeInput(config json.RawMessage, predecessors []string, outputs map[string][]byte, inputs map[string]string) ([]byte, error) {
	if len(config) > 0 {
		return append([]byte(nil), config...), nil
	}
	if len(predecessors) == 1 {
		return append([]byte(nil), outputs[predecessors[0]]...), nil
	}
	if len(predecessors) > 1 {
		values := make(map[string][]byte, len(predecessors))
		for _, id := range predecessors {
			values[id] = outputs[id]
		}
		return json.Marshal(values)
	}
	return json.Marshal(inputs)
}

func terminal(state string) bool {
	return state == "succeeded" || state == "failed" || state == "cancelled" || state == "interrupted"
}

func retryable(err error) bool {
	var failure protocol.ErrorPayload
	if errors.As(err, &failure) {
		return failure.Retryable
	}
	var temporary interface{ Temporary() bool }
	return errors.As(err, &temporary) && temporary.Temporary()
}

func truncate(value string, maxBytes int) string {
	for len(value) > maxBytes {
		_, size := utf8.DecodeLastRuneInString(value)
		value = value[:len(value)-size]
	}
	return value
}
