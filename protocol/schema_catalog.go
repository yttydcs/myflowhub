package protocol

import (
	"errors"
	"fmt"
	"sort"
)

const (
	SchemaResourceCatalogV1 = "mfh.catalog.v1"
	BuiltinResourceCatalog  = "system/catalog"
)

type ResourceKind string

const (
	ResourceKindVariable ResourceKind = "variable"
	ResourceKindStream   ResourceKind = "stream"
	ResourceKindCommand  ResourceKind = "command"
)

type ResourceDescriptorV1 struct {
	Name          string       `json:"name"`
	Kind          ResourceKind `json:"kind"`
	ContentType   string       `json:"content_type"`
	Schema        string       `json:"schema"`
	Permission    string       `json:"permission"`
	MaxValueBytes int          `json:"max_value_bytes"`
}

func (d ResourceDescriptorV1) Validate() error {
	if err := (ResourceID{Owner: 1, Name: d.Name}).Validate(); err != nil {
		return err
	}
	if d.Kind != ResourceKindVariable && d.Kind != ResourceKindStream && d.Kind != ResourceKindCommand {
		return errors.New("resource kind is invalid")
	}
	if err := validateText("content_type", d.ContentType, MaxContentTypeBytes, true); err != nil {
		return err
	}
	if err := validateText("schema", d.Schema, MaxSchemaBytes, true); err != nil {
		return err
	}
	if err := validateText("permission", d.Permission, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if d.MaxValueBytes <= 0 || d.MaxValueBytes > DefaultMaxPayload {
		return fmt.Errorf("max_value_bytes must be between 1 and %d", DefaultMaxPayload)
	}
	return nil
}

type ResourceCatalogV1 struct {
	Version   int                    `json:"version"`
	Revision  uint64                 `json:"revision"`
	Resources []ResourceDescriptorV1 `json:"resources"`
}

func (c ResourceCatalogV1) Validate() error {
	if err := validateVersion(c.Version); err != nil {
		return err
	}
	if c.Revision == 0 {
		return errors.New("catalog revision must be non-zero")
	}
	if len(c.Resources) == 0 || len(c.Resources) > MaxItems {
		return fmt.Errorf("catalog resources must contain between 1 and %d entries", MaxItems)
	}
	seen := make(map[string]struct{}, len(c.Resources))
	for index, descriptor := range c.Resources {
		if err := descriptor.Validate(); err != nil {
			return fmt.Errorf("resources[%d]: %w", index, err)
		}
		if _, exists := seen[descriptor.Name]; exists {
			return fmt.Errorf("duplicate resource %q", descriptor.Name)
		}
		seen[descriptor.Name] = struct{}{}
		if index > 0 && c.Resources[index-1].Name >= descriptor.Name {
			return errors.New("catalog resources must be strictly sorted by name")
		}
	}
	if _, exists := seen[BuiltinResourceCatalog]; !exists {
		return fmt.Errorf("catalog must describe %s", BuiltinResourceCatalog)
	}
	return nil
}

func (c *ResourceCatalogV1) Sort() {
	sort.Slice(c.Resources, func(i, j int) bool { return c.Resources[i].Name < c.Resources[j].Name })
}
