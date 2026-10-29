package resource

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/yttydcs/myflowhub/protocol"
)

// catalogResource preserves the standard Variable snapshot and observation
// path while adding bounded discovery for registries larger than one frame.
type catalogResource struct {
	*Variable
	registry *Registry
}

func catalogDescriptor(owner protocol.NodeID) Descriptor {
	d := VariableDescriptor(protocol.ResourceID{Owner: owner, Name: protocol.BuiltinResourceCatalog}, "resource.catalog.read", protocol.DefaultMaxPayload)
	d.Capabilities = append(d.Capabilities, protocol.CapabilityDescriptorV2{Name: protocol.CapabilityList, Permission: "resource.catalog.list", InputSchema: protocol.SchemaCatalogListV1, OutputSchema: protocol.SchemaCatalogPageV1, MaxPayloadBytes: protocol.MaxCatalogPageBytes})
	d.Schemas = append(d.Schemas, protocol.SchemaDescriptorV2{ID: protocol.SchemaCatalogListV1, ContentType: "application/json"}, protocol.SchemaDescriptorV2{ID: protocol.SchemaCatalogPageV1, ContentType: "application/json"})
	d.Sort()
	return d
}

func (c *catalogResource) Operate(ctx context.Context, request OperationRequest) (OperationResult, error) {
	if request.Capability != protocol.CapabilityList {
		return c.Variable.Operate(ctx, request)
	}
	if ctx == nil {
		return OperationResult{}, errors.New("catalog context is required")
	}
	if err := ctx.Err(); err != nil {
		return OperationResult{}, err
	}
	if request.Schema != "" && request.Schema != protocol.SchemaCatalogListV1 {
		return OperationResult{}, ErrInvalidSchema
	}
	var input protocol.CatalogListV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.MaxCatalogPageBytes, &input); err != nil {
		return OperationResult{}, err
	}
	return c.registry.catalogPage(input)
}

type catalogCursor struct {
	Version    int             `json:"version"`
	Owner      protocol.NodeID `json:"owner"`
	InstanceID string          `json:"instance_id"`
	Revision   string          `json:"revision"`
	Offset     int             `json:"offset"`
}

func (c catalogCursor) Validate() error {
	if c.Version != 1 || c.Owner.Validate() != nil || c.Offset < 1 || c.Offset >= protocol.MaxItems {
		return errors.New("invalid catalog cursor")
	}
	// Reuse the variable identity/revision validator without inventing another
	// spelling for the catalog's standard Variable generation.
	return (protocol.VariableValueV1{Version: 1, InstanceID: c.InstanceID, Revision: c.Revision, Status: "unavailable"}).Validate()
}

func encodeCatalogCursor(cursor catalogCursor) (string, error) {
	data, err := protocol.EncodeJSONPayload(&cursor, protocol.MaxCatalogCursorBytes*3/4)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func (r *Registry) catalogPage(input protocol.CatalogListV1) (OperationResult, error) {
	limit := input.Limit
	if limit == 0 {
		limit = protocol.DefaultCatalogPageItems
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	page := protocol.CatalogPageV1{Version: 1, Owner: r.owner, InstanceID: r.catalog.instanceID, Revision: strconv.FormatUint(r.catalogRevision, 10), Total: len(r.catalogDescriptors)}
	if input.Cursor != "" {
		data, err := base64.RawURLEncoding.Strict().DecodeString(input.Cursor)
		if err != nil {
			return OperationResult{}, fmt.Errorf("%w: invalid catalog cursor encoding", protocol.ErrInvalidPayload)
		}
		var cursor catalogCursor
		if err := protocol.DecodeJSONPayload(data, protocol.MaxCatalogCursorBytes, &cursor); err != nil {
			return OperationResult{}, err
		}
		if cursor.Owner != r.owner || cursor.InstanceID != page.InstanceID || cursor.Revision != page.Revision {
			return OperationResult{}, fmt.Errorf("%w: catalog changed; restart discovery", ErrRevisionConflict)
		}
		if cursor.Offset >= page.Total {
			return OperationResult{}, fmt.Errorf("%w: catalog cursor is outside the member set", protocol.ErrInvalidPayload)
		}
		page.Offset = cursor.Offset
	}
	end := min(page.Offset+limit, page.Total)
	for end > page.Offset {
		page.Resources = r.catalogDescriptors[page.Offset:end]
		page.NextCursor = ""
		if end < page.Total {
			var err error
			page.NextCursor, err = encodeCatalogCursor(catalogCursor{Version: 1, Owner: r.owner, InstanceID: page.InstanceID, Revision: page.Revision, Offset: end})
			if err != nil {
				return OperationResult{}, err
			}
		}
		payload, err := protocol.EncodeJSONPayload(&page, protocol.MaxCatalogPageBytes)
		if err == nil {
			return OperationResult{Schema: protocol.SchemaCatalogPageV1, Payload: payload}, nil
		}
		if !errors.Is(err, protocol.ErrPayloadTooLarge) {
			return OperationResult{}, err
		}
		// Byte-bounded pages may contain fewer items than requested. Binary
		// reduction avoids repeatedly encoding an oversized near-full page.
		if end-page.Offset == 1 {
			return OperationResult{}, fmt.Errorf("%w: a catalog descriptor exceeds the page budget", protocol.ErrPayloadTooLarge)
		}
		end = page.Offset + (end-page.Offset)/2
	}
	return OperationResult{}, fmt.Errorf("%w: empty catalog page", protocol.ErrInvalidPayload)
}

func (r *Registry) prepareCatalogLocked() (preparedVariable, uint64, []Descriptor, error) {
	if r.catalogRevision == ^uint64(0) {
		return preparedVariable{}, 0, nil, errors.New("resource catalog revision exhausted")
	}
	descriptors := make([]Descriptor, 0, len(r.resources))
	for _, value := range r.resources {
		descriptor, err := normalizeDescriptor(value.Descriptor())
		if err != nil {
			return preparedVariable{}, 0, nil, err
		}
		descriptors = append(descriptors, descriptor)
	}
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].ID.Name < descriptors[j].ID.Name })
	revision := r.catalogRevision + 1
	catalog := protocol.ResourceCatalogV2{Version: protocol.SchemaVersionV2, Revision: revision, Resources: descriptors}
	payload, err := protocol.EncodeVariablePayload(r.catalog.definition, &catalog, protocol.DefaultMaxPayload)
	var prepared preparedVariable
	if err == nil {
		prepared, err = r.catalog.prepareNext(0, VariableState{Value: payload, Status: "ready"})
	}
	if err != nil && r.catalog.payloadLimit() == protocol.DefaultMaxPayload && (errors.Is(err, protocol.ErrPayloadTooLarge) || errors.Is(err, ErrValueTooLarge)) {
		prepared, err = r.catalog.prepareNext(0, VariableState{Status: "unavailable", Error: &protocol.VariableError{Code: "overflow", Message: "Complete catalog exceeds one frame; use the authorized catalog list capability"}})
	}
	return prepared, revision, descriptors, err
}
