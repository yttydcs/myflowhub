package resource

import (
	"errors"
	"fmt"

	"github.com/yttydcs/myflowhub/protocol"
)

// VariableSpec describes a variable owned by the Registry's local node.
// WritePermission is optional; omitting it keeps the variable remotely read-only.
type VariableSpec struct {
	Name            string
	ContentType     string
	Schema          string
	ReadPermission  string
	WritePermission string
	MaxPayloadBytes int
	Initial         []byte
}

// Variable creates and registers a variable owned by the Registry's local node.
// It uses the same descriptor normalization, payload validation, duplicate checks,
// and catalog publication path as NewVariable followed by Register.
func (r *Registry) Variable(spec VariableSpec) (*Variable, error) {
	if r == nil {
		return nil, errors.New("declare variable: registry is required")
	}
	if err := spec.validate(); err != nil {
		return nil, fmt.Errorf("declare variable: %w", err)
	}

	id := protocol.ResourceID{Owner: r.owner, Name: spec.Name}
	descriptor := VariableDescriptor(
		id,
		spec.ContentType,
		spec.Schema,
		spec.ReadPermission,
		spec.MaxPayloadBytes,
	)
	if spec.WritePermission != "" {
		descriptor = WritableVariableDescriptor(
			id,
			spec.ContentType,
			spec.Schema,
			spec.ReadPermission,
			spec.WritePermission,
			spec.MaxPayloadBytes,
		)
	}

	value, err := NewVariable(descriptor, spec.Initial)
	if err != nil {
		return nil, fmt.Errorf("declare variable %q: %w", spec.Name, err)
	}
	if err := r.Register(value); err != nil {
		return nil, fmt.Errorf("declare variable %q: %w", spec.Name, err)
	}
	return value, nil
}

func (s VariableSpec) validate() error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.ContentType == "" {
		return errors.New("content type is required")
	}
	if s.Schema == "" {
		return errors.New("schema is required")
	}
	if s.ReadPermission == "" {
		return errors.New("read permission is required")
	}
	if s.MaxPayloadBytes <= 0 {
		return errors.New("max payload bytes must be positive")
	}
	return nil
}
