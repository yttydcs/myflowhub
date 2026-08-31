package protocol

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const (
	SchemaResourceCatalogV2 = "mfh.catalog.v2"
	BuiltinResourceCatalog  = "system/catalog"
	MaxCapabilities         = 64
	MaxSchemas              = 64
)

type ResourceTypeID string

const (
	ResourceTypeVariable   ResourceTypeID = "mfh.variable"
	ResourceTypeStream     ResourceTypeID = "mfh.stream"
	ResourceTypeTopic      ResourceTypeID = "mfh.topic"
	ResourceTypeCommand    ResourceTypeID = "mfh.command"
	ResourceTypeFile       ResourceTypeID = "mfh.file"
	ResourceTypeCollection ResourceTypeID = "mfh.collection"
)

func (id ResourceTypeID) Validate() error {
	return validateExtensibleIdentifier("resource type", string(id))
}

type CapabilityID string

const (
	CapabilityRead      CapabilityID = "read"
	CapabilityWrite     CapabilityID = "write"
	CapabilitySubscribe CapabilityID = "subscribe"
	CapabilityPublish   CapabilityID = "publish"
	CapabilityInvoke    CapabilityID = "invoke"
	CapabilityOpen      CapabilityID = "open"
	CapabilityList      CapabilityID = "list"
	CapabilityGet       CapabilityID = "get"
)

func (id CapabilityID) Validate() error {
	return validateExtensibleIdentifier("capability", string(id))
}

type SchemaDescriptorV2 struct {
	ID          string `json:"id"`
	ContentType string `json:"content_type"`
}

func (s SchemaDescriptorV2) Validate() error {
	if err := validateText("schema id", s.ID, MaxSchemaBytes, true); err != nil {
		return err
	}
	return validateText("schema content_type", s.ContentType, MaxContentTypeBytes, true)
}

type CapabilityDescriptorV2 struct {
	Name            CapabilityID `json:"name"`
	Permission      string       `json:"permission"`
	InputSchema     string       `json:"input_schema,omitempty"`
	OutputSchema    string       `json:"output_schema,omitempty"`
	EventSchema     string       `json:"event_schema,omitempty"`
	MaxPayloadBytes int          `json:"max_payload_bytes"`
}

func (c CapabilityDescriptorV2) Validate() error {
	if err := c.Name.Validate(); err != nil {
		return err
	}
	if err := validateText("capability permission", c.Permission, MaxIdentifierBytes, true); err != nil {
		return err
	}
	for field, value := range map[string]string{
		"input_schema": c.InputSchema, "output_schema": c.OutputSchema, "event_schema": c.EventSchema,
	} {
		if err := validateText(field, value, MaxSchemaBytes, false); err != nil {
			return err
		}
	}
	if c.MaxPayloadBytes <= 0 || c.MaxPayloadBytes > DefaultMaxPayload {
		return fmt.Errorf("capability max_payload_bytes must be between 1 and %d", DefaultMaxPayload)
	}
	return nil
}

type ResourceLimitsV2 struct {
	MaxPayloadBytes int `json:"max_payload_bytes"`
	MaxSubscribers  int `json:"max_subscribers,omitempty"`
	MaxSessions     int `json:"max_sessions,omitempty"`
	MaxQueue        int `json:"max_queue,omitempty"`
}

func (l ResourceLimitsV2) Validate() error {
	if l.MaxPayloadBytes <= 0 || l.MaxPayloadBytes > DefaultMaxPayload {
		return fmt.Errorf("resource max_payload_bytes must be between 1 and %d", DefaultMaxPayload)
	}
	for name, value := range map[string]int{
		"max_subscribers": l.MaxSubscribers, "max_sessions": l.MaxSessions, "max_queue": l.MaxQueue,
	} {
		if value < 0 || value > MaxItems {
			return fmt.Errorf("%s must be between 0 and %d", name, MaxItems)
		}
	}
	return nil
}

type PresentationHintV2 struct {
	Renderer    string `json:"renderer,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

func (p PresentationHintV2) Validate() error {
	if p.Renderer != "" {
		if err := validateExtensibleIdentifier("renderer", p.Renderer); err != nil {
			return err
		}
	}
	if err := validateText("presentation label", p.Label, MaxLabelBytes, false); err != nil {
		return err
	}
	if err := validateText("presentation description", p.Description, MaxAttributeValueBytes, false); err != nil {
		return err
	}
	return validateText("presentation icon", p.Icon, MaxIdentifierBytes, false)
}

type ResourceDescriptorV2 struct {
	ID           ResourceID               `json:"id"`
	Type         ResourceTypeID           `json:"type"`
	TypeVersion  uint32                   `json:"type_version"`
	Capabilities []CapabilityDescriptorV2 `json:"capabilities"`
	Schemas      []SchemaDescriptorV2     `json:"schemas,omitempty"`
	Limits       ResourceLimitsV2         `json:"limits"`
	Presentation PresentationHintV2       `json:"presentation,omitempty"`
}

func (d ResourceDescriptorV2) Validate() error {
	if err := d.ID.Validate(); err != nil {
		return err
	}
	if err := d.Type.Validate(); err != nil {
		return err
	}
	if d.TypeVersion == 0 {
		return errors.New("resource type_version must be non-zero")
	}
	if len(d.Capabilities) == 0 || len(d.Capabilities) > MaxCapabilities {
		return fmt.Errorf("resource capabilities must contain between 1 and %d entries", MaxCapabilities)
	}
	if len(d.Schemas) > MaxSchemas {
		return fmt.Errorf("resource schemas exceeds %d entries", MaxSchemas)
	}
	if err := d.Limits.Validate(); err != nil {
		return err
	}
	if err := d.Presentation.Validate(); err != nil {
		return err
	}
	schemas := make(map[string]struct{}, len(d.Schemas))
	for index, schema := range d.Schemas {
		if err := schema.Validate(); err != nil {
			return fmt.Errorf("schemas[%d]: %w", index, err)
		}
		if _, exists := schemas[schema.ID]; exists {
			return fmt.Errorf("duplicate schema %q", schema.ID)
		}
		schemas[schema.ID] = struct{}{}
		if index > 0 && d.Schemas[index-1].ID >= schema.ID {
			return errors.New("resource schemas must be strictly sorted by id")
		}
	}
	capabilities := make(map[CapabilityID]struct{}, len(d.Capabilities))
	for index, capability := range d.Capabilities {
		if err := capability.Validate(); err != nil {
			return fmt.Errorf("capabilities[%d]: %w", index, err)
		}
		if _, exists := capabilities[capability.Name]; exists {
			return fmt.Errorf("duplicate capability %q", capability.Name)
		}
		capabilities[capability.Name] = struct{}{}
		if index > 0 && d.Capabilities[index-1].Name >= capability.Name {
			return errors.New("resource capabilities must be strictly sorted by name")
		}
		for field, schema := range map[string]string{
			"input_schema": capability.InputSchema, "output_schema": capability.OutputSchema, "event_schema": capability.EventSchema,
		} {
			if schema != "" {
				if _, exists := schemas[schema]; !exists {
					return fmt.Errorf("capability %q %s references unknown schema %q", capability.Name, field, schema)
				}
			}
		}
	}
	return nil
}

func (d ResourceDescriptorV2) Capability(name CapabilityID) (CapabilityDescriptorV2, bool) {
	for _, capability := range d.Capabilities {
		if capability.Name == name {
			return capability, true
		}
	}
	return CapabilityDescriptorV2{}, false
}

func (d *ResourceDescriptorV2) Sort() {
	sort.Slice(d.Capabilities, func(i, j int) bool { return d.Capabilities[i].Name < d.Capabilities[j].Name })
	sort.Slice(d.Schemas, func(i, j int) bool { return d.Schemas[i].ID < d.Schemas[j].ID })
}

type ResourceCatalogV2 struct {
	Version   int                    `json:"version"`
	Revision  uint64                 `json:"revision"`
	Resources []ResourceDescriptorV2 `json:"resources"`
}

func (c ResourceCatalogV2) Validate() error {
	if c.Version != SchemaVersionV2 {
		return fmt.Errorf("version must be %d", SchemaVersionV2)
	}
	if c.Revision == 0 {
		return errors.New("catalog revision must be non-zero")
	}
	if len(c.Resources) == 0 || len(c.Resources) > MaxItems {
		return fmt.Errorf("catalog resources must contain between 1 and %d entries", MaxItems)
	}
	seen := make(map[ResourceID]struct{}, len(c.Resources))
	hasCatalog := false
	for index, descriptor := range c.Resources {
		if err := descriptor.Validate(); err != nil {
			return fmt.Errorf("resources[%d]: %w", index, err)
		}
		if _, exists := seen[descriptor.ID]; exists {
			return fmt.Errorf("duplicate resource %d/%q", descriptor.ID.Owner, descriptor.ID.Name)
		}
		seen[descriptor.ID] = struct{}{}
		if descriptor.ID.Name == BuiltinResourceCatalog {
			hasCatalog = true
		}
		if index > 0 && compareResourceID(c.Resources[index-1].ID, descriptor.ID) >= 0 {
			return errors.New("catalog resources must be strictly sorted by owner and name")
		}
	}
	if !hasCatalog {
		return fmt.Errorf("catalog must describe %s", BuiltinResourceCatalog)
	}
	return nil
}

func (c *ResourceCatalogV2) Sort() {
	for index := range c.Resources {
		c.Resources[index].Sort()
	}
	sort.Slice(c.Resources, func(i, j int) bool { return compareResourceID(c.Resources[i].ID, c.Resources[j].ID) < 0 })
}

func compareResourceID(left, right ResourceID) int {
	if left.Owner < right.Owner {
		return -1
	}
	if left.Owner > right.Owner {
		return 1
	}
	return strings.Compare(left.Name, right.Name)
}

func validateExtensibleIdentifier(field, value string) error {
	if err := validateText(field, value, MaxIdentifierBytes, true); err != nil {
		return err
	}
	for index, character := range value {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '.' || character == '-' || character == '_' {
			if index == 0 && (character == '.' || character == '-' || character == '_') {
				return fmt.Errorf("%s must start with a letter or digit", field)
			}
			continue
		}
		return fmt.Errorf("%s contains unsupported character %q", field, character)
	}
	return nil
}
