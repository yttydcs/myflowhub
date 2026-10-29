package protocol

import (
	"errors"
	"fmt"
)

const (
	SchemaCatalogListV1     = "mfh.catalog-list.v1"
	SchemaCatalogPageV1     = "mfh.catalog-page.v1"
	DefaultCatalogPageItems = 128
	MaxCatalogPageItems     = 256
	MaxCatalogPageBytes     = 512 << 10
	MaxCatalogCursorBytes   = 512
)

type CatalogListV1 struct {
	Version int    `json:"version"`
	Limit   int    `json:"limit,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
}

func (r CatalogListV1) Validate() error {
	if err := validateVersion(r.Version); err != nil {
		return err
	}
	if r.Limit < 0 || r.Limit > MaxCatalogPageItems {
		return fmt.Errorf("catalog limit must be between 0 and %d", MaxCatalogPageItems)
	}
	return validateText("catalog cursor", r.Cursor, MaxCatalogCursorBytes, false)
}

type CatalogPageV1 struct {
	Version    int                    `json:"version"`
	Owner      NodeID                 `json:"owner"`
	InstanceID string                 `json:"instance_id"`
	Revision   string                 `json:"revision"`
	Offset     int                    `json:"offset"`
	Total      int                    `json:"total"`
	Resources  []ResourceDescriptorV2 `json:"resources"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

func (p CatalogPageV1) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	if err := p.Owner.Validate(); err != nil {
		return err
	}
	if err := validateHexID("catalog instance_id", p.InstanceID, 16); err != nil {
		return err
	}
	if _, err := ParseVariableRevision(p.Revision); err != nil {
		return err
	}
	if p.Total < 1 || p.Total > MaxItems || p.Offset < 0 || p.Offset >= p.Total {
		return errors.New("catalog page position is invalid")
	}
	if len(p.Resources) < 1 || len(p.Resources) > MaxCatalogPageItems || len(p.Resources) > p.Total-p.Offset {
		return errors.New("catalog page resource count is invalid")
	}
	if (p.Offset+len(p.Resources) < p.Total) != (p.NextCursor != "") {
		return errors.New("catalog continuation does not match remaining resources")
	}
	if err := validateText("catalog next_cursor", p.NextCursor, MaxCatalogCursorBytes, false); err != nil {
		return err
	}
	for i, descriptor := range p.Resources {
		if err := descriptor.Validate(); err != nil {
			return fmt.Errorf("catalog resources[%d]: %w", i, err)
		}
		if descriptor.ID.Owner != p.Owner {
			return errors.New("catalog page contains a foreign owner")
		}
		if i > 0 && compareResourceID(p.Resources[i-1].ID, descriptor.ID) >= 0 {
			return errors.New("catalog page resources must be strictly sorted")
		}
	}
	return nil
}
