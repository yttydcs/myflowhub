package protocol

import (
	"errors"
	"fmt"
)

const (
	SchemaCollectionListRequestV1   = "mfh.collection.list-request.v1"
	SchemaCollectionMemberRequestV1 = "mfh.collection.member-request.v1"
	SchemaCollectionMemberV1        = "mfh.collection.member.v1"
	SchemaCollectionPageV1          = "mfh.collection.page.v1"

	MaxCollectionPageMembers     = 256
	MaxCollectionParentBytes     = MaxPathBytes
	MaxCollectionCursorBytes     = 1024
	MaxCollectionMemberKeyBytes  = MaxPathBytes
	MaxCollectionMemberKindBytes = MaxIdentifierBytes
	// MaxCollectionRevision is the largest integer that survives an exact
	// JSON round trip through JavaScript and other IEEE-754 based clients.
	MaxCollectionRevision uint64 = 1<<53 - 1
)

// CollectionListRequestV1 asks a provider for one bounded page. Parent and
// Cursor are provider-scoped opaque values and must not be interpreted by the
// generic runtime.
type CollectionListRequestV1 struct {
	Version int    `json:"version"`
	Parent  string `json:"parent,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
	Limit   int    `json:"limit"`
}

func (r CollectionListRequestV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if err := validateText("collection parent", r.Parent, MaxCollectionParentBytes, false); err != nil {
		return err
	}
	if err := validateText("collection cursor", r.Cursor, MaxCollectionCursorBytes, false); err != nil {
		return err
	}
	if r.Limit <= 0 || r.Limit > MaxCollectionPageMembers {
		return fmt.Errorf("collection limit must be between 1 and %d", MaxCollectionPageMembers)
	}
	return nil
}

type CollectionMemberRequestV1 struct {
	Version int    `json:"version"`
	Key     string `json:"key"`
}

func (r CollectionMemberRequestV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	return validateText("collection member key", r.Key, MaxCollectionMemberKeyBytes, true)
}

// CollectionMemberV1 identifies a member only within its provider-owned
// Collection. It intentionally contains no owner or global ResourceID.
type CollectionMemberV1 struct {
	Key          string            `json:"key"`
	Kind         string            `json:"kind"`
	Label        string            `json:"label"`
	ContentType  string            `json:"content_type,omitempty"`
	Schema       string            `json:"schema,omitempty"`
	Capabilities []CapabilityID    `json:"capabilities,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

func (m CollectionMemberV1) Validate() error {
	if err := validateText("collection member key", m.Key, MaxCollectionMemberKeyBytes, true); err != nil {
		return err
	}
	if err := validateText("collection member kind", m.Kind, MaxCollectionMemberKindBytes, true); err != nil {
		return err
	}
	if err := validateText("collection member label", m.Label, MaxLabelBytes, true); err != nil {
		return err
	}
	if err := validateText("collection member content_type", m.ContentType, MaxContentTypeBytes, false); err != nil {
		return err
	}
	if err := validateText("collection member schema", m.Schema, MaxSchemaBytes, false); err != nil {
		return err
	}
	if len(m.Capabilities) > MaxCapabilities {
		return fmt.Errorf("collection member capabilities exceed %d entries", MaxCapabilities)
	}
	for index, capability := range m.Capabilities {
		if err := capability.Validate(); err != nil {
			return fmt.Errorf("collection member capabilities[%d]: %w", index, err)
		}
		if index > 0 && m.Capabilities[index-1] >= capability {
			return errors.New("collection member capabilities must be strictly sorted and unique")
		}
	}
	return validateAttributes(m.Attributes)
}

type CollectionPageV1 struct {
	Version    int                  `json:"version"`
	Revision   uint64               `json:"revision"`
	Parent     string               `json:"parent,omitempty"`
	Members    []CollectionMemberV1 `json:"members"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

func (p CollectionPageV1) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	if p.Revision == 0 || p.Revision > MaxCollectionRevision {
		return fmt.Errorf("collection page revision must be between 1 and %d", MaxCollectionRevision)
	}
	if err := validateText("collection parent", p.Parent, MaxCollectionParentBytes, false); err != nil {
		return err
	}
	if err := validateText("collection next_cursor", p.NextCursor, MaxCollectionCursorBytes, false); err != nil {
		return err
	}
	if len(p.Members) > MaxCollectionPageMembers {
		return fmt.Errorf("collection page members exceed %d entries", MaxCollectionPageMembers)
	}
	for index, member := range p.Members {
		if err := member.Validate(); err != nil {
			return fmt.Errorf("collection page members[%d]: %w", index, err)
		}
		if index > 0 && p.Members[index-1].Key >= member.Key {
			return errors.New("collection page members must be strictly sorted and unique by key")
		}
	}
	return nil
}

// ValidateForDescriptor is the provider/runtime seam that prevents member
// metadata from advertising capabilities absent from the containing Resource.
// Network authorization remains exact ResourceID + CapabilityID.
func (p CollectionPageV1) ValidateForDescriptor(descriptor ResourceDescriptorV2) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if err := descriptor.Validate(); err != nil {
		return fmt.Errorf("collection descriptor: %w", err)
	}
	if descriptor.Type != ResourceTypeCollection {
		return errors.New("collection page descriptor must use mfh.collection type")
	}
	declared := make(map[CapabilityID]struct{}, len(descriptor.Capabilities))
	for _, capability := range descriptor.Capabilities {
		declared[capability.Name] = struct{}{}
	}
	for memberIndex, member := range p.Members {
		for _, capability := range member.Capabilities {
			if _, ok := declared[capability]; !ok {
				return fmt.Errorf("collection page members[%d] capability %q is not declared by the collection", memberIndex, capability)
			}
		}
	}
	return nil
}
