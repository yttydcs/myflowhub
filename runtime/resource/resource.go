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
	mu        sync.RWMutex
	owner     protocol.NodeID
	resources map[string]Resource
}

func NewRegistry(owner protocol.NodeID) (*Registry, error) {
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	return &Registry{owner: owner, resources: make(map[string]Resource)}, nil
}

func (r *Registry) Register(value Resource) error {
	if value == nil {
		return errors.New("register resource: value is required")
	}
	descriptor := value.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return fmt.Errorf("register resource: %w", err)
	}
	if descriptor.ID.Owner != r.owner {
		return fmt.Errorf("register resource: owner %d does not match local node %d", descriptor.ID.Owner, r.owner)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.resources[descriptor.ID.Name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicate, descriptor.ID.Name)
	}
	r.resources[descriptor.ID.Name] = value
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.resources[id.Name]; !exists {
		return ErrNotFound
	}
	delete(r.resources, id.Name)
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

type CommandHandler func(context.Context, []byte) ([]byte, error)

type Command struct {
	descriptor Descriptor
	handler    CommandHandler
}

func NewCommand(descriptor Descriptor, handler CommandHandler) (*Command, error) {
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
