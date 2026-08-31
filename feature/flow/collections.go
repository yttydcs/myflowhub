package flow

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

const (
	definitionsCursorKind = "definitions"
	runsCursorKind        = "runs"
)

type collectionCursor struct {
	Version  int    `json:"v"`
	Kind     string `json:"k"`
	Revision uint64 `json:"r"`
	After    string `json:"a"`
}

// observableHandlerResource keeps operation dispatch and event observation on
// one Resource without adding a second registered endpoint or event queue.
type observableHandlerResource struct {
	*resource.HandlerResource
	events *resource.Stream
}

func (r *observableHandlerResource) Observe(observer func(resource.Observation)) (*resource.Observation, func(), error) {
	return r.events.Observe(observer)
}

func (c *Controller) buildResources() (resource.Resource, resource.Resource, error) {
	owner := c.node.ID()
	definitionsDescriptor := collectionDescriptor(
		protocol.ResourceID{Owner: owner, Name: protocol.BuiltinFlowDefinitions},
		"Flow definitions",
		[]protocol.CapabilityDescriptorV2{
			flowCapability(protocol.CapabilityList, "flow.read", protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, ""),
			flowCapability(protocol.CapabilityGet, "flow.read", protocol.SchemaCollectionMemberRequestV1, protocol.SchemaFlowDefinitionV1, ""),
			flowCapability(protocol.CapabilityFlowCreate, "flow.write", protocol.SchemaFlowDefinitionV1, protocol.SchemaFlowDefinitionV1, ""),
			flowCapability(protocol.CapabilityFlowUpdate, "flow.write", protocol.SchemaFlowDefinitionV1, protocol.SchemaFlowDefinitionV1, ""),
			flowCapability(protocol.CapabilityFlowArchive, "flow.write", protocol.SchemaFlowArchiveV1, protocol.SchemaFlowArchiveV1, ""),
			flowCapability(protocol.CapabilityFlowRun, "flow.run", protocol.SchemaFlowRunV1, protocol.SchemaFlowRunSummaryV1, ""),
		},
	)
	definitions, err := resource.NewHandlerResource(definitionsDescriptor, map[protocol.CapabilityID]resource.Handler{
		protocol.CapabilityList:        c.listDefinitions,
		protocol.CapabilityGet:         c.getDefinition,
		protocol.CapabilityFlowCreate:  commandHandler(c.create),
		protocol.CapabilityFlowUpdate:  commandHandler(c.update),
		protocol.CapabilityFlowArchive: commandHandler(c.archive),
		protocol.CapabilityFlowRun:     commandHandler(c.run),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create flow definitions collection: %w", err)
	}

	runsDescriptor := collectionDescriptor(
		protocol.ResourceID{Owner: owner, Name: protocol.BuiltinFlowRuns},
		"Flow runs",
		[]protocol.CapabilityDescriptorV2{
			flowCapability(protocol.CapabilityList, "flow.read", protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, ""),
			flowCapability(protocol.CapabilityGet, "flow.read", protocol.SchemaCollectionMemberRequestV1, protocol.SchemaFlowRunSummaryV1, ""),
			flowCapability(protocol.CapabilitySubscribe, "flow.read", "", "", protocol.SchemaFlowEventV1),
			flowCapability(protocol.CapabilityFlowCancel, "flow.cancel", protocol.SchemaFlowCancelV1, protocol.SchemaFlowRunSummaryV1, ""),
		},
	)
	runsHandler, err := resource.NewHandlerResource(runsDescriptor, map[protocol.CapabilityID]resource.Handler{
		protocol.CapabilityList:       c.listRuns,
		protocol.CapabilityGet:        c.getRun,
		protocol.CapabilityFlowCancel: commandHandler(c.cancelRun),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create flow runs collection: %w", err)
	}
	events, err := resource.NewStream(resource.StreamDescriptor(
		protocol.ResourceID{Owner: owner, Name: protocol.BuiltinFlowRuns},
		"application/json", protocol.SchemaFlowEventV1, "flow.read", protocol.DefaultMaxPayload,
	))
	if err != nil {
		return nil, nil, fmt.Errorf("create flow runs event source: %w", err)
	}
	c.events = events
	runs := &observableHandlerResource{HandlerResource: runsHandler, events: events}
	return definitions, runs, nil
}

func collectionDescriptor(id protocol.ResourceID, label string, capabilities []protocol.CapabilityDescriptorV2) protocol.ResourceDescriptorV2 {
	schemas := make(map[string]struct{})
	for _, capability := range capabilities {
		for _, schema := range []string{capability.InputSchema, capability.OutputSchema, capability.EventSchema} {
			if schema != "" {
				schemas[schema] = struct{}{}
			}
		}
	}
	descriptor := protocol.ResourceDescriptorV2{
		ID: id, Type: protocol.ResourceTypeCollection, TypeVersion: 1,
		Capabilities: capabilities,
		Limits: protocol.ResourceLimitsV2{
			MaxPayloadBytes: protocol.DefaultMaxPayload,
			MaxSubscribers:  1024,
			MaxQueue:        256,
		},
		Presentation: protocol.PresentationHintV2{Renderer: string(protocol.ResourceTypeCollection), Label: label},
	}
	for schema := range schemas {
		descriptor.Schemas = append(descriptor.Schemas, protocol.SchemaDescriptorV2{ID: schema, ContentType: "application/json"})
	}
	descriptor.Sort()
	return descriptor
}

func flowCapability(name protocol.CapabilityID, permission, inputSchema, outputSchema, eventSchema string) protocol.CapabilityDescriptorV2 {
	return protocol.CapabilityDescriptorV2{
		Name: name, Permission: permission, InputSchema: inputSchema, OutputSchema: outputSchema,
		EventSchema: eventSchema, MaxPayloadBytes: protocol.DefaultMaxPayload,
	}
}

func commandHandler(handler resource.CommandHandler) resource.Handler {
	return func(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
		payload, err := handler(ctx, request.Payload)
		return resource.OperationResult{Payload: payload}, err
	}
}

func (c *Controller) listDefinitions(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	var list protocol.CollectionListRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &list); err != nil {
		return resource.OperationResult{}, err
	}
	if list.Parent != "" {
		return resource.OperationResult{}, errors.New("flow definitions is a flat collection and does not support parent")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.definitions))
	for key := range c.definitions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	start, err := collectionStart(list.Cursor, definitionsCursorKind, c.revision, keys)
	if err != nil {
		return resource.OperationResult{}, err
	}
	end := min(start+list.Limit, len(keys))
	members := make([]protocol.CollectionMemberV1, 0, end-start)
	for _, key := range keys[start:end] {
		definition := c.definitions[key]
		members = append(members, protocol.CollectionMemberV1{
			Key: key, Kind: "flow-definition", Label: definition.Name,
			ContentType: "application/json", Schema: protocol.SchemaFlowDefinitionV1,
			Capabilities: []protocol.CapabilityID{
				protocol.CapabilityFlowArchive, protocol.CapabilityGet, protocol.CapabilityFlowRun, protocol.CapabilityFlowUpdate,
			},
			Attributes: map[string]string{
				"edge_count": strconv.Itoa(len(definition.Edges)),
				"node_count": strconv.Itoa(len(definition.Nodes)),
				"revision":   strconv.FormatUint(definition.Revision, 10),
			},
		})
	}
	page := protocol.CollectionPageV1{Version: 1, Revision: c.revision, Members: members}
	if end < len(keys) {
		page.NextCursor, err = encodeCollectionCursor(definitionsCursorKind, c.revision, keys[end-1])
		if err != nil {
			return resource.OperationResult{}, err
		}
	}
	return c.encodePage(page, protocol.BuiltinFlowDefinitions)
}

func (c *Controller) getDefinition(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	var member protocol.CollectionMemberRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &member); err != nil {
		return resource.OperationResult{}, err
	}
	c.mu.Lock()
	definition, ok := c.definitions[member.Key]
	c.mu.Unlock()
	if !ok {
		return resource.OperationResult{}, fmt.Errorf("flow definition %q not found", member.Key)
	}
	payload, err := protocol.EncodeJSONPayload(&definition, protocol.DefaultMaxPayload)
	return resource.OperationResult{Payload: payload}, err
}

func (c *Controller) listRuns(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	var list protocol.CollectionListRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &list); err != nil {
		return resource.OperationResult{}, err
	}
	if list.Parent != "" {
		return resource.OperationResult{}, errors.New("flow runs is a flat collection and does not support parent")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.runs))
	for key := range c.runs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	start, err := collectionStart(list.Cursor, runsCursorKind, c.revision, keys)
	if err != nil {
		return resource.OperationResult{}, err
	}
	end := min(start+list.Limit, len(keys))
	members := make([]protocol.CollectionMemberV1, 0, end-start)
	for _, key := range keys[start:end] {
		run := c.runs[key]
		attributes := map[string]string{
			"flow_id":         run.FlowID,
			"flow_revision":   strconv.FormatUint(run.FlowRevision, 10),
			"initiator":       run.Initiator,
			"started_unix_ms": strconv.FormatInt(run.StartedUnixMS, 10),
			"state":           run.State,
		}
		if run.FinishedUnixMS > 0 {
			attributes["finished_unix_ms"] = strconv.FormatInt(run.FinishedUnixMS, 10)
		}
		members = append(members, protocol.CollectionMemberV1{
			Key: key, Kind: "flow-run", Label: key,
			ContentType: "application/json", Schema: protocol.SchemaFlowRunSummaryV1,
			Capabilities: []protocol.CapabilityID{protocol.CapabilityFlowCancel, protocol.CapabilityGet},
			Attributes:   attributes,
		})
	}
	page := protocol.CollectionPageV1{Version: 1, Revision: c.revision, Members: members}
	if end < len(keys) {
		page.NextCursor, err = encodeCollectionCursor(runsCursorKind, c.revision, keys[end-1])
		if err != nil {
			return resource.OperationResult{}, err
		}
	}
	return c.encodePage(page, protocol.BuiltinFlowRuns)
}

func (c *Controller) getRun(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	var member protocol.CollectionMemberRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &member); err != nil {
		return resource.OperationResult{}, err
	}
	c.mu.Lock()
	run, ok := c.runs[member.Key]
	c.mu.Unlock()
	if !ok {
		return resource.OperationResult{}, fmt.Errorf("flow run %q not found", member.Key)
	}
	payload, err := protocol.EncodeJSONPayload(&run, protocol.DefaultMaxPayload)
	return resource.OperationResult{Payload: payload}, err
}

func (c *Controller) encodePage(page protocol.CollectionPageV1, resourceName string) (resource.OperationResult, error) {
	value, ok := c.node.Registry().Resolve(protocol.ResourceID{Owner: c.node.ID(), Name: resourceName})
	if ok {
		if err := page.ValidateForDescriptor(value.Descriptor()); err != nil {
			return resource.OperationResult{}, fmt.Errorf("validate flow collection page: %w", err)
		}
	} else if err := page.Validate(); err != nil {
		return resource.OperationResult{}, fmt.Errorf("validate flow collection page: %w", err)
	}
	payload, err := protocol.EncodeJSONPayload(&page, protocol.DefaultMaxPayload)
	return resource.OperationResult{Payload: payload}, err
}

func collectionStart(encoded, kind string, revision uint64, keys []string) (int, error) {
	if encoded == "" {
		return 0, nil
	}
	cursor, err := decodeCollectionCursor(encoded)
	if err != nil {
		return 0, fmt.Errorf("invalid flow collection cursor: %w", err)
	}
	if cursor.Kind != kind {
		return 0, errors.New("invalid flow collection cursor: collection mismatch")
	}
	if cursor.Revision != revision {
		return 0, errors.New("invalid flow collection cursor: stale revision")
	}
	index := sort.SearchStrings(keys, cursor.After)
	if index >= len(keys) || keys[index] != cursor.After {
		return 0, errors.New("invalid flow collection cursor: member no longer exists")
	}
	return index + 1, nil
}

func encodeCollectionCursor(kind string, revision uint64, after string) (string, error) {
	payload, err := json.Marshal(collectionCursor{Version: 1, Kind: kind, Revision: revision, After: after})
	if err != nil {
		return "", fmt.Errorf("encode flow collection cursor: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	if len(encoded) > protocol.MaxCollectionCursorBytes {
		return "", errors.New("encode flow collection cursor: cursor exceeds protocol limit")
	}
	return encoded, nil
}

func decodeCollectionCursor(encoded string) (collectionCursor, error) {
	if len(encoded) > protocol.MaxCollectionCursorBytes {
		return collectionCursor{}, errors.New("cursor exceeds protocol limit")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return collectionCursor{}, errors.New("cursor encoding is malformed")
	}
	var cursor collectionCursor
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return collectionCursor{}, errors.New("cursor payload is malformed")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return collectionCursor{}, errors.New("cursor payload has trailing data")
	}
	if cursor.Version != 1 || cursor.Kind == "" || cursor.Revision == 0 || cursor.After == "" {
		return collectionCursor{}, errors.New("cursor fields are invalid")
	}
	return cursor, nil
}
