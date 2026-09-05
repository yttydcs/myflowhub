package resource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
)

var (
	ErrDuplicate             = errors.New("resource already registered")
	ErrNotFound              = errors.New("resource not found")
	ErrRevisionRegression    = errors.New("resource revision or sequence regression")
	ErrRevisionConflict      = errors.New("resource revision conflict")
	ErrValueTooLarge         = errors.New("resource value exceeds configured limit")
	ErrResourceLimit         = errors.New("resource registry limit reached")
	ErrReservedResource      = errors.New("built-in resource cannot be replaced or removed")
	ErrUnsupportedCapability = errors.New("resource capability is not supported")
	ErrInvalidSchema         = errors.New("resource operation schema does not match capability")
)

type Descriptor = protocol.ResourceDescriptorV2

type OperationRequest struct {
	Subject    protocol.NodeID
	Capability protocol.CapabilityID
	Schema     string
	Payload    []byte
}

type OperationResult struct {
	Schema  string
	Payload []byte
}

type Observation struct {
	Snapshot          bool
	Revision          uint64
	Sequence          uint64
	Publisher         protocol.NodeID
	PublisherSequence uint64
	Schema            string
	Value             []byte
	// Failure terminates the observation; no value is delivered with it.
	Failure *protocol.ErrorPayload
}

type Resource interface {
	Descriptor() Descriptor
	Operate(context.Context, OperationRequest) (OperationResult, error)
}

type Observable interface {
	Resource
	Observe(func(Observation)) (*Observation, func(), error)
}

type SessionOpenRequest struct {
	Subject    protocol.NodeID
	Capability protocol.CapabilityID
	Schema     string
	Payload    []byte
}

type SessionGrant struct {
	Capability      protocol.CapabilityID
	MaxChunkBytes   int
	MaxTotalBytes   int64
	ExpiresAtUnixMS int64
}

type SessionData struct {
	Offset   int64
	Payload  []byte
	Checksum string
}

type SessionCloseRequest struct {
	Commit  bool
	Schema  string
	Payload []byte
}

type ActiveSession interface {
	Grant() SessionGrant
	Write(context.Context, SessionData) (OperationResult, error)
	Close(context.Context, SessionCloseRequest) (OperationResult, error)
	Abort(error)
}

type SessionResource interface {
	Resource
	OpenSession(context.Context, SessionOpenRequest) (ActiveSession, error)
}

type Registry struct {
	mu              sync.RWMutex
	catalogUpdateMu sync.Mutex
	owner           protocol.NodeID
	resources       map[string]Resource
	catalog         *Variable
	catalogRevision uint64
}

func NewRegistry(owner protocol.NodeID) (*Registry, error) {
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	descriptor := VariableDescriptor(
		protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog},
		"application/json", protocol.SchemaResourceCatalogV2, "resource.catalog.read", protocol.DefaultMaxPayload,
	)
	catalogValue := protocol.ResourceCatalogV2{Version: protocol.SchemaVersionV2, Revision: 1, Resources: []protocol.ResourceDescriptorV2{descriptor}}
	catalogValue.Sort()
	payload, err := protocol.EncodeJSONPayload(&catalogValue, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, fmt.Errorf("create resource catalog: %w", err)
	}
	catalog, err := NewVariable(descriptor, payload)
	if err != nil {
		return nil, fmt.Errorf("create resource catalog: %w", err)
	}
	return &Registry{owner: owner, resources: map[string]Resource{protocol.BuiltinResourceCatalog: catalog}, catalog: catalog, catalogRevision: 1}, nil
}

func (r *Registry) Register(value Resource) error {
	if value == nil {
		return errors.New("register resource: value is required")
	}
	descriptor, err := normalizeDescriptor(value.Descriptor())
	if err != nil {
		return fmt.Errorf("register resource: %w", err)
	}
	if descriptor.ID.Owner != r.owner {
		return fmt.Errorf("register resource: owner %d does not match local node %d", descriptor.ID.Owner, r.owner)
	}
	if descriptor.ID.Name == protocol.BuiltinResourceCatalog {
		return fmt.Errorf("%w: %s", ErrReservedResource, descriptor.ID.Name)
	}
	r.catalogUpdateMu.Lock()
	defer r.catalogUpdateMu.Unlock()
	r.mu.Lock()
	if _, exists := r.resources[descriptor.ID.Name]; exists {
		r.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrDuplicate, descriptor.ID.Name)
	}
	if len(r.resources) >= protocol.MaxItems {
		r.mu.Unlock()
		return ErrResourceLimit
	}
	r.resources[descriptor.ID.Name] = value
	payload, revision, err := r.buildCatalogLocked()
	if err != nil {
		delete(r.resources, descriptor.ID.Name)
		r.mu.Unlock()
		return fmt.Errorf("register resource catalog update: %w", err)
	}
	r.mu.Unlock()
	if _, err := r.catalog.Set(payload); err != nil {
		r.mu.Lock()
		delete(r.resources, descriptor.ID.Name)
		r.mu.Unlock()
		return fmt.Errorf("register resource catalog publish: %w", err)
	}
	r.catalogRevision = revision
	return nil
}

func (r *Registry) Resolve(id protocol.ResourceID) (Resource, bool) {
	if id.Owner != r.owner {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.resources[id.Name]
	return value, ok
}

func (r *Registry) Descriptor(id protocol.ResourceID) (Descriptor, bool) {
	value, ok := r.Resolve(id)
	if !ok {
		return Descriptor{}, false
	}
	return cloneDescriptor(value.Descriptor()), true
}

func (r *Registry) Operate(ctx context.Context, id protocol.ResourceID, request OperationRequest) (OperationResult, error) {
	if ctx == nil {
		return OperationResult{}, errors.New("resource operation context is required")
	}
	value, ok := r.Resolve(id)
	if !ok {
		return OperationResult{}, ErrNotFound
	}
	descriptor := value.Descriptor()
	capability, ok := descriptor.Capability(request.Capability)
	if !ok {
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
	if len(request.Payload) > capability.MaxPayloadBytes || len(request.Payload) > descriptor.Limits.MaxPayloadBytes {
		return OperationResult{}, fmt.Errorf("%w: got %d", ErrValueTooLarge, len(request.Payload))
	}
	if request.Schema == "" {
		request.Schema = capability.InputSchema
	} else if capability.InputSchema != "" && request.Schema != capability.InputSchema {
		return OperationResult{}, fmt.Errorf("%w: got %q, want %q", ErrInvalidSchema, request.Schema, capability.InputSchema)
	}
	request.Payload = append([]byte(nil), request.Payload...)
	result, err := value.Operate(ctx, request)
	if err != nil {
		return OperationResult{}, err
	}
	if len(result.Payload) > capability.MaxPayloadBytes || len(result.Payload) > descriptor.Limits.MaxPayloadBytes {
		return OperationResult{}, fmt.Errorf("%w: operation result got %d", ErrValueTooLarge, len(result.Payload))
	}
	if result.Schema == "" {
		result.Schema = capability.OutputSchema
	} else if capability.OutputSchema != "" && result.Schema != capability.OutputSchema {
		return OperationResult{}, fmt.Errorf("%w: result got %q, want %q", ErrInvalidSchema, result.Schema, capability.OutputSchema)
	}
	result.Payload = append([]byte(nil), result.Payload...)
	return result, nil
}

func (r *Registry) OpenSession(ctx context.Context, id protocol.ResourceID, request SessionOpenRequest) (ActiveSession, error) {
	if ctx == nil {
		return nil, errors.New("resource session context is required")
	}
	value, ok := r.Resolve(id)
	if !ok {
		return nil, ErrNotFound
	}
	descriptor := value.Descriptor()
	capability, ok := descriptor.Capability(request.Capability)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
	if len(request.Payload) > capability.MaxPayloadBytes || len(request.Payload) > descriptor.Limits.MaxPayloadBytes {
		return nil, fmt.Errorf("%w: session open got %d", ErrValueTooLarge, len(request.Payload))
	}
	if request.Schema == "" {
		request.Schema = capability.InputSchema
	} else if capability.InputSchema != "" && request.Schema != capability.InputSchema {
		return nil, fmt.Errorf("%w: got %q, want %q", ErrInvalidSchema, request.Schema, capability.InputSchema)
	}
	sessionResource, ok := value.(SessionResource)
	if !ok {
		return nil, fmt.Errorf("%w: %s is not session-oriented", ErrUnsupportedCapability, id.Name)
	}
	request.Payload = append([]byte(nil), request.Payload...)
	return sessionResource.OpenSession(ctx, request)
}

func (r *Registry) Remove(id protocol.ResourceID) error {
	if id.Owner != r.owner {
		return ErrNotFound
	}
	if id.Name == protocol.BuiltinResourceCatalog {
		return fmt.Errorf("%w: %s", ErrReservedResource, id.Name)
	}
	r.catalogUpdateMu.Lock()
	defer r.catalogUpdateMu.Unlock()
	r.mu.Lock()
	if _, exists := r.resources[id.Name]; !exists {
		r.mu.Unlock()
		return ErrNotFound
	}
	removed := r.resources[id.Name]
	delete(r.resources, id.Name)
	payload, revision, err := r.buildCatalogLocked()
	if err != nil {
		r.resources[id.Name] = removed
		r.mu.Unlock()
		return fmt.Errorf("remove resource catalog update: %w", err)
	}
	r.mu.Unlock()
	if _, err := r.catalog.Set(payload); err != nil {
		r.mu.Lock()
		r.resources[id.Name] = removed
		r.mu.Unlock()
		return fmt.Errorf("remove resource catalog publish: %w", err)
	}
	r.catalogRevision = revision
	return nil
}

func (r *Registry) List() []Descriptor {
	r.mu.RLock()
	descriptors := make([]Descriptor, 0, len(r.resources))
	for _, value := range r.resources {
		descriptors = append(descriptors, cloneDescriptor(value.Descriptor()))
	}
	r.mu.RUnlock()
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].ID.Name < descriptors[j].ID.Name })
	return descriptors
}

func (r *Registry) Catalog() *Variable { return r.catalog }

func (r *Registry) buildCatalogLocked() ([]byte, uint64, error) {
	if r.catalogRevision == ^uint64(0) {
		return nil, 0, errors.New("resource catalog revision exhausted")
	}
	descriptors := make([]protocol.ResourceDescriptorV2, 0, len(r.resources))
	for _, value := range r.resources {
		descriptor, err := normalizeDescriptor(value.Descriptor())
		if err != nil {
			return nil, 0, err
		}
		descriptors = append(descriptors, descriptor)
	}
	revision := r.catalogRevision + 1
	catalog := protocol.ResourceCatalogV2{Version: protocol.SchemaVersionV2, Revision: revision, Resources: descriptors}
	catalog.Sort()
	payload, err := protocol.EncodeJSONPayload(&catalog, protocol.DefaultMaxPayload)
	return payload, revision, err
}

func VariableDescriptor(id protocol.ResourceID, contentType, schema, permission string, maxPayload int) Descriptor {
	return descriptorFor(id, protocol.ResourceTypeVariable, contentType, schema, permission, maxPayload,
		protocol.CapabilityDescriptorV2{Name: protocol.CapabilityRead, OutputSchema: schema},
		protocol.CapabilityDescriptorV2{Name: protocol.CapabilitySubscribe, EventSchema: schema})
}

func WritableVariableDescriptor(id protocol.ResourceID, contentType, schema, readPermission, writePermission string, maxPayload int) Descriptor {
	descriptor := VariableDescriptor(id, contentType, schema, readPermission, maxPayload)
	if writePermission == "" {
		writePermission = "resource.write"
	}
	descriptor.Capabilities = append(descriptor.Capabilities, protocol.CapabilityDescriptorV2{
		Name: protocol.CapabilityWrite, Permission: writePermission, InputSchema: protocol.SchemaVariableWriteV2,
		OutputSchema: schema, MaxPayloadBytes: descriptor.Limits.MaxPayloadBytes,
	})
	descriptor.Schemas = append(descriptor.Schemas, protocol.SchemaDescriptorV2{ID: protocol.SchemaVariableWriteV2, ContentType: "application/json"})
	descriptor.Sort()
	return descriptor
}

func StreamDescriptor(id protocol.ResourceID, contentType, schema, permission string, maxPayload int) Descriptor {
	return descriptorFor(id, protocol.ResourceTypeStream, contentType, schema, permission, maxPayload,
		protocol.CapabilityDescriptorV2{Name: protocol.CapabilitySubscribe, EventSchema: schema})
}

func CommandDescriptor(id protocol.ResourceID, contentType, schema, permission string, maxPayload int) Descriptor {
	return CommandDescriptorSchemas(id, contentType, schema, schema, permission, maxPayload)
}

func CommandDescriptorSchemas(id protocol.ResourceID, contentType, inputSchema, outputSchema, permission string, maxPayload int) Descriptor {
	if outputSchema == "" {
		outputSchema = inputSchema
	}
	descriptor := descriptorFor(id, protocol.ResourceTypeCommand, contentType, inputSchema, permission, maxPayload,
		protocol.CapabilityDescriptorV2{Name: protocol.CapabilityInvoke, InputSchema: inputSchema, OutputSchema: outputSchema})
	if outputSchema != inputSchema {
		descriptor.Schemas = append(descriptor.Schemas, protocol.SchemaDescriptorV2{ID: outputSchema, ContentType: descriptor.Schemas[0].ContentType})
		descriptor.Sort()
	}
	return descriptor
}

func TopicDescriptor(id protocol.ResourceID, contentType, schema, publishPermission, subscribePermission string, maxPayload int) Descriptor {
	if publishPermission == "" {
		publishPermission = "resource.publish"
	}
	if subscribePermission == "" {
		subscribePermission = "resource.subscribe"
	}
	descriptor := descriptorFor(id, protocol.ResourceTypeTopic, contentType, schema, subscribePermission, maxPayload,
		protocol.CapabilityDescriptorV2{Name: protocol.CapabilityPublish, Permission: publishPermission, InputSchema: schema},
		protocol.CapabilityDescriptorV2{Name: protocol.CapabilitySubscribe, Permission: subscribePermission, EventSchema: schema})
	return descriptor
}

func SessionDescriptor(id protocol.ResourceID, typeID protocol.ResourceTypeID, contentType, inputSchema, outputSchema, permission string, maxPayload int) Descriptor {
	if outputSchema == "" {
		outputSchema = inputSchema
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if permission == "" {
		permission = "resource.open"
	}
	if maxPayload <= 0 {
		maxPayload = protocol.DefaultMaxPayload
	}
	descriptor := Descriptor{
		ID: id, Type: typeID, TypeVersion: 1,
		Capabilities: []protocol.CapabilityDescriptorV2{{
			Name: protocol.CapabilityOpen, Permission: permission, InputSchema: inputSchema,
			OutputSchema: outputSchema, MaxPayloadBytes: maxPayload,
		}},
		Schemas: []protocol.SchemaDescriptorV2{
			{ID: inputSchema, ContentType: contentType},
			{ID: outputSchema, ContentType: contentType},
		},
		Limits:       protocol.ResourceLimitsV2{MaxPayloadBytes: maxPayload, MaxSessions: 64, MaxQueue: 256},
		Presentation: protocol.PresentationHintV2{Renderer: string(typeID)},
	}
	if inputSchema == outputSchema {
		descriptor.Schemas = descriptor.Schemas[:1]
	}
	descriptor.Sort()
	return descriptor
}

func descriptorFor(id protocol.ResourceID, typeID protocol.ResourceTypeID, contentType, schema, permission string, maxPayload int, capabilities ...protocol.CapabilityDescriptorV2) Descriptor {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if schema == "" {
		schema = "mfh.raw.v1"
	}
	if permission == "" {
		permission = "resource.access"
	}
	if maxPayload <= 0 {
		maxPayload = protocol.DefaultMaxPayload
	}
	for index := range capabilities {
		if capabilities[index].Permission == "" {
			capabilities[index].Permission = permission
		}
		if capabilities[index].MaxPayloadBytes <= 0 {
			capabilities[index].MaxPayloadBytes = maxPayload
		}
		if capabilities[index].InputSchema == "" && capabilities[index].Name == protocol.CapabilityPublish {
			capabilities[index].InputSchema = schema
		}
	}
	descriptor := Descriptor{
		ID: id, Type: typeID, TypeVersion: 1, Capabilities: capabilities,
		Schemas:      []protocol.SchemaDescriptorV2{{ID: schema, ContentType: contentType}},
		Limits:       protocol.ResourceLimitsV2{MaxPayloadBytes: maxPayload, MaxSubscribers: 1024, MaxSessions: 64, MaxQueue: 256},
		Presentation: protocol.PresentationHintV2{Renderer: string(typeID)},
	}
	descriptor.Sort()
	return descriptor
}

func normalizeDescriptor(descriptor Descriptor) (Descriptor, error) {
	descriptor = cloneDescriptor(descriptor)
	if descriptor.TypeVersion == 0 {
		descriptor.TypeVersion = 1
	}
	if descriptor.Limits.MaxPayloadBytes <= 0 {
		descriptor.Limits.MaxPayloadBytes = protocol.DefaultMaxPayload
	}
	for index := range descriptor.Capabilities {
		if descriptor.Capabilities[index].MaxPayloadBytes <= 0 {
			descriptor.Capabilities[index].MaxPayloadBytes = descriptor.Limits.MaxPayloadBytes
		}
	}
	descriptor.Sort()
	if err := descriptor.Validate(); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}

func cloneDescriptor(descriptor Descriptor) Descriptor {
	descriptor.Capabilities = append([]protocol.CapabilityDescriptorV2(nil), descriptor.Capabilities...)
	descriptor.Schemas = append([]protocol.SchemaDescriptorV2(nil), descriptor.Schemas...)
	return descriptor
}

func validatePayload(descriptor Descriptor, value []byte) error {
	if len(value) > descriptor.Limits.MaxPayloadBytes {
		return fmt.Errorf("%w: got %d, max %d", ErrValueTooLarge, len(value), descriptor.Limits.MaxPayloadBytes)
	}
	return nil
}

type CommandHandler func(context.Context, []byte) ([]byte, error)

type Command struct {
	descriptor Descriptor
	handler    CommandHandler
}

func NewCommand(descriptor Descriptor, handler CommandHandler) (*Command, error) {
	descriptor, err := normalizeDescriptor(descriptor)
	if err != nil {
		return nil, err
	}
	if descriptor.Type != protocol.ResourceTypeCommand {
		return nil, errors.New("command descriptor must use mfh.command type")
	}
	if _, ok := descriptor.Capability(protocol.CapabilityInvoke); !ok {
		return nil, errors.New("command descriptor requires invoke capability")
	}
	if handler == nil {
		return nil, errors.New("command handler is required")
	}
	return &Command{descriptor: descriptor, handler: handler}, nil
}

func (c *Command) Descriptor() Descriptor { return cloneDescriptor(c.descriptor) }

func (c *Command) Operate(ctx context.Context, request OperationRequest) (OperationResult, error) {
	if ctx == nil {
		return OperationResult{}, errors.New("command context is required")
	}
	if request.Capability != protocol.CapabilityInvoke {
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
	if err := validatePayload(c.descriptor, request.Payload); err != nil {
		return OperationResult{}, err
	}
	output, err := c.handler(ctx, append([]byte(nil), request.Payload...))
	if err != nil {
		return OperationResult{}, err
	}
	if err := validatePayload(c.descriptor, output); err != nil {
		return OperationResult{}, err
	}
	capability, _ := c.descriptor.Capability(protocol.CapabilityInvoke)
	return OperationResult{Schema: capability.OutputSchema, Payload: append([]byte(nil), output...)}, nil
}

func (c *Command) Invoke(ctx context.Context, input []byte) ([]byte, error) {
	result, err := c.Operate(ctx, OperationRequest{Capability: protocol.CapabilityInvoke, Payload: input})
	return result.Payload, err
}
