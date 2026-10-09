package resource

import (
	"errors"
	"fmt"
	"github.com/yttydcs/myflowhub/protocol"
)

// VariableSpec declares a platform variable owned by this Registry. An explicit
// WritePermission is required to expose remote writes.
type VariableSpec struct {
	Name            string
	Definition      protocol.DataSchemaDefinition
	ReadPermission  string
	WritePermission string
	MaxPayloadBytes int
	Initial         VariableState
	WriteHandler    VariableWriteHandler
}

func (r *Registry) Variable(spec VariableSpec) (*Variable, error) {
	if r == nil {
		return nil, errors.New("declare variable: registry is required")
	}
	if spec.Name == "" {
		return nil, errors.New("declare variable: name is required")
	}
	if spec.ReadPermission == "" {
		return nil, errors.New("declare variable: read permission is required")
	}
	if spec.MaxPayloadBytes <= 0 {
		return nil, errors.New("declare variable: max payload bytes must be positive")
	}
	id := protocol.ResourceID{Owner: r.owner, Name: spec.Name}
	descriptor := VariableDescriptor(id, spec.ReadPermission, spec.MaxPayloadBytes)
	if spec.WritePermission != "" {
		descriptor = WritableVariableDescriptor(id, spec.ReadPermission, spec.WritePermission, spec.MaxPayloadBytes)
	}
	value, err := NewVariable(descriptor, spec.Definition, spec.Initial, spec.WriteHandler)
	if err != nil {
		return nil, fmt.Errorf("declare variable %q: %w", spec.Name, err)
	}
	if err := r.Register(value); err != nil {
		return nil, fmt.Errorf("declare variable %q: %w", spec.Name, err)
	}
	return value, nil
}
