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
	ErrDuplicate          = errors.New("resource already registered")
	ErrNotFound           = errors.New("resource not found")
	ErrRevisionRegression = errors.New("resource revision or sequence regression")
	ErrValueTooLarge      = errors.New("resource value exceeds configured limit")
	ErrResourceLimit      = errors.New("resource registry limit reached")
	ErrReservedResource   = errors.New("built-in resource cannot be replaced or removed")
)

type Kind uint8

const (
	KindVariable Kind = iota + 1
	KindStream
	KindCommand
)

func (k Kind) Valid() bool { return k >= KindVariable && k <= KindCommand }

type Descriptor struct {
	ID            protocol.ResourceID
	Kind          Kind
	ContentType   string
	Schema        string
	Permission    string
	MaxValueBytes int
}

func (d Descriptor) Validate() error {
	if err := d.ID.Validate(); err != nil {
		return err
	}
	if !d.Kind.Valid() {
		return errors.New("resource kind is invalid")
	}
	if len(d.ContentType) > protocol.MaxContentTypeBytes || len(d.Schema) > protocol.MaxSchemaBytes {
		return errors.New("resource content type or schema is too long")
	}
	if d.MaxValueBytes <= 0 || d.MaxValueBytes > protocol.DefaultMaxPayload {
		return fmt.Errorf("resource max value must be between 1 and %d bytes", protocol.DefaultMaxPayload)
	}
	return nil
}

func (d Descriptor) validateValue(value []byte) error {
	if len(value) > d.MaxValueBytes {
		return fmt.Errorf("%w: got %d, max %d", ErrValueTooLarge, len(value), d.MaxValueBytes)
	}
	return nil
}

type Resource interface {
	Descriptor() Descriptor
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
	descriptor := normalizeDescriptor(Descriptor{
		ID: protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog}, Kind: KindVariable,
		ContentType: "application/json", Schema: protocol.SchemaResourceCatalogV1, Permission: "resource.catalog.read", MaxValueBytes: protocol.DefaultMaxPayload,
	})
	payload, err := protocol.EncodeJSONPayload(&protocol.ResourceCatalogV1{Version: 1, Revision: 1, Resources: []protocol.ResourceDescriptorV1{catalogDescriptor(descriptor)}}, protocol.DefaultMaxPayload)
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
	descriptor := value.Descriptor()
	descriptor = normalizeDescriptor(descriptor)
	if err := descriptor.Validate(); err != nil {
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
		descriptors = append(descriptors, value.Descriptor())
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
	descriptors := make([]protocol.ResourceDescriptorV1, 0, len(r.resources))
	for _, value := range r.resources {
		descriptors = append(descriptors, catalogDescriptor(normalizeDescriptor(value.Descriptor())))
	}
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].Name < descriptors[j].Name })
	revision := r.catalogRevision + 1
	payload, err := protocol.EncodeJSONPayload(&protocol.ResourceCatalogV1{Version: 1, Revision: revision, Resources: descriptors}, protocol.DefaultMaxPayload)
	return payload, revision, err
}

func normalizeDescriptor(descriptor Descriptor) Descriptor {
	if descriptor.ContentType == "" {
		descriptor.ContentType = "application/octet-stream"
	}
	if descriptor.Schema == "" {
		descriptor.Schema = "mfh.raw.v1"
	}
	if descriptor.Permission == "" {
		if descriptor.Kind == KindCommand {
			descriptor.Permission = "resource.invoke"
		} else {
			descriptor.Permission = "resource.subscribe"
		}
	}
	return descriptor
}

func catalogDescriptor(descriptor Descriptor) protocol.ResourceDescriptorV1 {
	kind := protocol.ResourceKindVariable
	switch descriptor.Kind {
	case KindStream:
		kind = protocol.ResourceKindStream
	case KindCommand:
		kind = protocol.ResourceKindCommand
	}
	return protocol.ResourceDescriptorV1{
		Name: descriptor.ID.Name, Kind: kind, ContentType: descriptor.ContentType, Schema: descriptor.Schema,
		Permission: descriptor.Permission, MaxValueBytes: descriptor.MaxValueBytes,
	}
}

type CommandHandler func(context.Context, []byte) ([]byte, error)

type Command struct {
	descriptor Descriptor
	handler    CommandHandler
}

func NewCommand(descriptor Descriptor, handler CommandHandler) (*Command, error) {
	descriptor = normalizeDescriptor(descriptor)
	if descriptor.Kind != KindCommand {
		return nil, errors.New("command descriptor must use command kind")
	}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, errors.New("command handler is required")
	}
	return &Command{descriptor: descriptor, handler: handler}, nil
}

func (c *Command) Descriptor() Descriptor { return c.descriptor }

func (c *Command) Invoke(ctx context.Context, input []byte) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("command context is required")
	}
	if err := c.descriptor.validateValue(input); err != nil {
		return nil, err
	}
	output, err := c.handler(ctx, append([]byte(nil), input...))
	if err != nil {
		return nil, err
	}
	if err := c.descriptor.validateValue(output); err != nil {
		return nil, err
	}
	return append([]byte(nil), output...), nil
}
