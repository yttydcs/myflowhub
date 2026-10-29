package sdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/yttydcs/myflowhub/protocol"
)

// Catalog retains the one-read bootstrap contract for existing small catalogs.
// Only an explicit owner overflow state requires separately authorized paging.
func (c *Client) Catalog(ctx context.Context, owner protocol.NodeID) (protocol.ResourceCatalogV2, error) {
	state, err := c.Snapshot(ctx, protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog})
	if err != nil {
		return protocol.ResourceCatalogV2{}, err
	}
	if state.Status == "unavailable" && state.Error != nil && state.Error.Code == string(protocol.CodeOverflow) {
		return c.CatalogPages(ctx, owner)
	}
	if len(state.Value) == 0 {
		return protocol.ResourceCatalogV2{}, &Error{Message: "catalog variable value is unavailable"}
	}
	var catalog protocol.ResourceCatalogV2
	if err := protocol.DecodeVariablePayload(*state.Definition, state.Value, protocol.DefaultMaxPayload, &catalog); err != nil {
		return protocol.ResourceCatalogV2{}, &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("decode catalog snapshot: %v", err), cause: err}
	}
	if err := validateCatalogOwner(catalog, owner); err != nil {
		return protocol.ResourceCatalogV2{}, err
	}
	return catalog, nil
}

// CatalogPages explicitly requests list permission and can be used without
// read permission. Unsupported legacy providers alone may use complete read.
func (c *Client) CatalogPages(ctx context.Context, owner protocol.NodeID) (protocol.ResourceCatalogV2, error) {
	if ctx == nil {
		return protocol.ResourceCatalogV2{}, errors.New("catalog context is required")
	}
	if err := owner.Validate(); err != nil {
		return protocol.ResourceCatalogV2{}, err
	}
	id := protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog}
	for attempt := 0; attempt < 3; attempt++ {
		var catalog protocol.ResourceCatalogV2
		var instance, cursor string
		total := 0
		for {
			if err := ctx.Err(); err != nil {
				return protocol.ResourceCatalogV2{}, err
			}
			input, err := protocol.EncodeJSONPayload(&protocol.CatalogListV1{Version: 1, Limit: protocol.DefaultCatalogPageItems, Cursor: cursor}, protocol.MaxCatalogPageBytes)
			if err != nil {
				return protocol.ResourceCatalogV2{}, err
			}
			result, err := c.Operate(ctx, id, protocol.CapabilityList, protocol.SchemaCatalogListV1, input)
			if err != nil {
				var failure *Error
				if errors.As(err, &failure) {
					if failure.Code == protocol.CodeUnsupported && attempt == 0 && cursor == "" {
						// Only an old provider explicitly lacking list may use the
						// legacy path. Authorization errors never broaden access.
						var legacy protocol.ResourceCatalogV2
						if err := c.DecodeSnapshot(ctx, id, &legacy); err != nil {
							return protocol.ResourceCatalogV2{}, err
						}
						if err := validateCatalogOwner(legacy, owner); err != nil {
							return protocol.ResourceCatalogV2{}, err
						}
						return legacy, nil
					}
					if failure.Code == protocol.CodeConflict {
						break
					}
				}
				return protocol.ResourceCatalogV2{}, err
			}
			malformed := func(message string) (protocol.ResourceCatalogV2, error) {
				return protocol.ResourceCatalogV2{}, &Error{Code: protocol.CodeMalformed, Message: message}
			}
			if result.Schema != protocol.SchemaCatalogPageV1 {
				return malformed("catalog list returned an unexpected schema")
			}
			var page protocol.CatalogPageV1
			if err := protocol.DecodeJSONPayload(result.Payload, protocol.MaxCatalogPageBytes, &page); err != nil {
				return malformed(fmt.Sprintf("decode catalog page: %v", err))
			}
			if page.Owner != owner || page.Offset != len(catalog.Resources) {
				return malformed("catalog page owner or offset does not match the request")
			}
			revision, _ := protocol.ParseVariableRevision(page.Revision) // validated by DecodeJSONPayload
			if cursor == "" {
				instance, total = page.InstanceID, page.Total
				catalog.Version, catalog.Revision = protocol.SchemaVersionV2, revision
			} else if page.InstanceID != instance || revision != catalog.Revision || page.Total != total {
				return malformed("catalog pages mixed different member generations")
			}
			if len(catalog.Resources) > 0 && catalog.Resources[len(catalog.Resources)-1].ID.Name >= page.Resources[0].ID.Name {
				return malformed("catalog pages contain duplicate or out-of-order resources")
			}
			catalog.Resources = append(catalog.Resources, page.Resources...)
			if page.NextCursor == "" {
				if err := validateCatalogOwner(catalog, owner); err != nil {
					return protocol.ResourceCatalogV2{}, err
				}
				return catalog, nil
			}
			if page.NextCursor == cursor {
				return malformed("catalog continuation did not advance")
			}
			cursor = page.NextCursor
		}
	}
	return protocol.ResourceCatalogV2{}, &Error{Code: protocol.CodeConflict, Message: "catalog membership changed during three discovery attempts"}
}

func validateCatalogOwner(catalog protocol.ResourceCatalogV2, owner protocol.NodeID) error {
	if err := catalog.Validate(); err != nil {
		return &Error{Code: protocol.CodeMalformed, Message: fmt.Sprintf("invalid complete catalog: %v", err)}
	}
	for _, descriptor := range catalog.Resources {
		if descriptor.ID.Owner != owner {
			return &Error{Code: protocol.CodeMalformed, Message: "catalog contains a foreign owner"}
		}
	}
	return nil
}
