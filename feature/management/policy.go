package management

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

const (
	policyDefinitionsCursor = "policy-definitions"
	policyBindingsCursor    = "policy-bindings"
	policyGrantsCursor      = "policy-grants"
)

type policyCursor struct {
	Version    int    `json:"v"`
	Collection string `json:"c"`
	Revision   uint64 `json:"r"`
	After      string `json:"a"`
}

func (c *Controller) buildPolicyResources() ([]resource.Resource, error) {
	definitionsDescriptor := policyCollectionDescriptor(
		protocol.ResourceID{Owner: c.node.ID(), Name: protocol.BuiltinPolicyDefinitions},
		"Policy definitions",
		[]protocol.CapabilityDescriptorV2{
			policyCapability(protocol.CapabilityCreate, protocol.SchemaPolicyDefinitionPutV1, protocol.SchemaPolicyDefinitionV1, true),
			policyCapability(protocol.CapabilityDelete, protocol.SchemaPolicyDefinitionDeleteV1, protocol.SchemaManagementResultV1, true),
			policyCapability(protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, protocol.SchemaPolicyDefinitionV1, false),
			policyCapability(protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, false),
			policyCapability(protocol.CapabilityUpdate, protocol.SchemaPolicyDefinitionPutV1, protocol.SchemaPolicyDefinitionV1, true),
		},
	)
	definitions, err := resource.NewHandlerResource(definitionsDescriptor, map[protocol.CapabilityID]resource.Handler{
		protocol.CapabilityCreate: c.createPolicyDefinition,
		protocol.CapabilityDelete: c.deletePolicyDefinition,
		protocol.CapabilityGet:    c.getPolicyDefinition,
		protocol.CapabilityList:   c.listPolicyDefinitions,
		protocol.CapabilityUpdate: c.updatePolicyDefinition,
	})
	if err != nil {
		return nil, fmt.Errorf("create policy definitions collection: %w", err)
	}

	bindingsDescriptor := policyCollectionDescriptor(
		protocol.ResourceID{Owner: c.node.ID(), Name: protocol.BuiltinPolicyBindings},
		"Policy bindings",
		[]protocol.CapabilityDescriptorV2{
			policyCapability(protocol.CapabilityCreate, protocol.SchemaPolicyBindingCreateV1, protocol.SchemaPolicyBindingV1, true),
			policyCapability(protocol.CapabilityEvaluate, protocol.SchemaPolicyEvaluateRequestV1, protocol.SchemaPolicyEvaluationV1, false),
			policyCapability(protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, protocol.SchemaPolicyBindingV1, false),
			policyCapability(protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, false),
			policyCapability(protocol.CapabilityRevoke, protocol.SchemaPolicyBindingRevokeV1, protocol.SchemaManagementResultV1, true),
		},
	)
	bindings, err := resource.NewHandlerResource(bindingsDescriptor, map[protocol.CapabilityID]resource.Handler{
		protocol.CapabilityCreate:   c.createPolicyBinding,
		protocol.CapabilityEvaluate: c.evaluatePolicy,
		protocol.CapabilityGet:      c.getPolicyBinding,
		protocol.CapabilityList:     c.listPolicyBindings,
		protocol.CapabilityRevoke:   c.revokePolicyBinding,
	})
	if err != nil {
		return nil, fmt.Errorf("create policy bindings collection: %w", err)
	}

	grantsDescriptor := policyCollectionDescriptor(
		protocol.ResourceID{Owner: c.node.ID(), Name: protocol.BuiltinPolicyGrants},
		"Exact policy grants",
		[]protocol.CapabilityDescriptorV2{
			policyCapability(protocol.CapabilityCreate, protocol.SchemaPolicyGrantV1, protocol.SchemaPolicyGrantV1, true),
			policyCapability(protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, protocol.SchemaPolicyGrantV1, false),
			policyCapability(protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, false),
			policyCapability(protocol.CapabilityRevoke, protocol.SchemaPolicyGrantV1, protocol.SchemaManagementResultV1, true),
		},
	)
	grants, err := resource.NewHandlerResource(grantsDescriptor, map[protocol.CapabilityID]resource.Handler{
		protocol.CapabilityCreate: c.createPolicyGrant,
		protocol.CapabilityGet:    c.getPolicyGrant,
		protocol.CapabilityList:   c.listPolicyGrants,
		protocol.CapabilityRevoke: c.revokeExactPolicyGrant,
	})
	if err != nil {
		return nil, fmt.Errorf("create policy grants collection: %w", err)
	}

	legacyGrant, err := c.legacyPolicyCommand(protocol.BuiltinManagementPolicyGrant, c.createLegacyPolicyGrant)
	if err != nil {
		return nil, err
	}
	legacyRevoke, err := c.legacyPolicyCommand(protocol.BuiltinManagementPolicyRevoke, c.revokeLegacyPolicyGrant)
	if err != nil {
		return nil, err
	}
	return []resource.Resource{definitions, bindings, grants, legacyGrant, legacyRevoke}, nil
}

func policyCollectionDescriptor(id protocol.ResourceID, label string, capabilities []protocol.CapabilityDescriptorV2) protocol.ResourceDescriptorV2 {
	schemas := make(map[string]struct{})
	for _, capability := range capabilities {
		if capability.InputSchema != "" {
			schemas[capability.InputSchema] = struct{}{}
		}
		if capability.OutputSchema != "" {
			schemas[capability.OutputSchema] = struct{}{}
		}
	}
	descriptor := protocol.ResourceDescriptorV2{
		ID: id, Type: protocol.ResourceTypeCollection, TypeVersion: 1, Capabilities: capabilities,
		Limits:       protocol.ResourceLimitsV2{MaxPayloadBytes: protocol.DefaultMaxPayload, MaxQueue: 256},
		Presentation: protocol.PresentationHintV2{Renderer: string(protocol.ResourceTypeCollection), Label: label},
	}
	for schema := range schemas {
		descriptor.Schemas = append(descriptor.Schemas, protocol.SchemaDescriptorV2{ID: schema, ContentType: contentTypeJSON})
	}
	descriptor.Sort()
	return descriptor
}

func policyCapability(name protocol.CapabilityID, input, output string, mutation bool) protocol.CapabilityDescriptorV2 {
	permission := "management.policy.read"
	if mutation {
		permission = "management.policy.write"
	} else if name == protocol.CapabilityEvaluate {
		permission = "management.policy.evaluate"
	}
	return protocol.CapabilityDescriptorV2{Name: name, Permission: permission, InputSchema: input, OutputSchema: output, MaxPayloadBytes: protocol.DefaultMaxPayload}
}

func (c *Controller) legacyPolicyCommand(name string, handler resource.Handler) (resource.Resource, error) {
	descriptor := resource.CommandDescriptorSchemas(
		protocol.ResourceID{Owner: c.node.ID(), Name: name}, contentTypeJSON,
		protocol.SchemaManagementPolicyRuleV1, protocol.SchemaManagementResultV1,
		"management.policy.write", protocol.DefaultMaxPayload,
	)
	value, err := resource.NewHandlerResource(descriptor, map[protocol.CapabilityID]resource.Handler{protocol.CapabilityInvoke: handler})
	if err != nil {
		return nil, fmt.Errorf("create legacy policy command %s: %w", name, err)
	}
	return value, nil
}

func (c *Controller) requirePolicySuperadmin(request resource.OperationRequest, resourceName string) error {
	allowed, _, err := c.policy.HasAuthoritySuperadmin(request.Subject, c.node.ID())
	if err != nil {
		err = fmt.Errorf("%w: verify Authority-domain superadmin: %v", auth.ErrForbidden, err)
	} else if !allowed {
		err = fmt.Errorf("%w: Authority-domain superadmin binding is required", auth.ErrForbidden)
	}
	if err != nil {
		c.audit.RecordDecision(policyAuditRequest(request, c.node.ID(), resourceName), err)
	}
	return err
}

func policyAuditRequest(request resource.OperationRequest, owner protocol.NodeID, resourceName string) auth.Request {
	return auth.Request{Subject: request.Subject, Action: auth.Action(request.Capability), Capability: request.Capability, Resource: protocol.ResourceID{Owner: owner, Name: resourceName}}
}

func (c *Controller) recordPolicyMutation(request resource.OperationRequest, resourceName, target, status string) {
	c.audit.RecordOutcome(policyAuditRequest(request, c.node.ID(), resourceName), target, status)
}

func (c *Controller) listPolicyDefinitions(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	values := c.policy.Definitions()
	members := make([]protocol.CollectionMemberV1, 0, len(values))
	for _, value := range values {
		members = append(members, protocol.CollectionMemberV1{
			Key: value.ID, Kind: "policy-definition", Label: value.Label, ContentType: contentTypeJSON, Schema: protocol.SchemaPolicyDefinitionV1,
			Capabilities: []protocol.CapabilityID{protocol.CapabilityDelete, protocol.CapabilityGet, protocol.CapabilityUpdate},
			Attributes:   map[string]string{"immutable": strconv.FormatBool(value.Immutable), "revision": strconv.FormatUint(value.Revision, 10), "rules": strconv.Itoa(len(value.Rules))},
		})
	}
	return c.policyPage(request, policyDefinitionsCursor, protocol.BuiltinPolicyDefinitions, members)
}

func (c *Controller) getPolicyDefinition(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	member, err := decodePolicyMemberRequest(request.Payload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	value, ok := c.policy.Definition(member.Key)
	if !ok {
		return resource.OperationResult{}, fmt.Errorf("%w: policy definition %q", auth.ErrPolicyNotFound, member.Key)
	}
	return encodePolicyResult(&value)
}

func (c *Controller) createPolicyDefinition(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinPolicyDefinitions); err != nil {
		return resource.OperationResult{}, err
	}
	var input protocol.PolicyDefinitionPutV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if input.ExpectedRevision != 0 {
		return resource.OperationResult{}, errors.New("create policy definition requires expected_revision 0")
	}
	value, err := c.policy.PutDefinition(input)
	if err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinPolicyDefinitions, value.ID, "created")
	return encodePolicyResult(&value)
}

func (c *Controller) updatePolicyDefinition(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinPolicyDefinitions); err != nil {
		return resource.OperationResult{}, err
	}
	var input protocol.PolicyDefinitionPutV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if input.ExpectedRevision == 0 {
		return resource.OperationResult{}, errors.New("update policy definition requires expected_revision")
	}
	value, err := c.policy.PutDefinition(input)
	if err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinPolicyDefinitions, value.ID, "updated")
	return encodePolicyResult(&value)
}

func (c *Controller) deletePolicyDefinition(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinPolicyDefinitions); err != nil {
		return resource.OperationResult{}, err
	}
	var input protocol.PolicyDefinitionDeleteV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if err := c.policy.DeleteDefinition(input); err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinPolicyDefinitions, input.ID, "deleted")
	return encodePolicyResult(&protocol.ManagementResultV1{Version: 1, Status: "ok"})
}

func (c *Controller) listPolicyBindings(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	values := c.policy.Bindings()
	members := make([]protocol.CollectionMemberV1, 0, len(values))
	for _, value := range values {
		members = append(members, protocol.CollectionMemberV1{
			Key: value.BindingID, Kind: "policy-binding", Label: value.Subject + " → " + value.DefinitionID,
			ContentType: contentTypeJSON, Schema: protocol.SchemaPolicyBindingV1,
			Capabilities: []protocol.CapabilityID{protocol.CapabilityGet, protocol.CapabilityRevoke},
			Attributes:   map[string]string{"definition_id": value.DefinitionID, "scope_kind": value.Scope.Kind, "scope_node": value.Scope.NodeID, "subject": value.Subject},
		})
	}
	return c.policyPage(request, policyBindingsCursor, protocol.BuiltinPolicyBindings, members)
}

func (c *Controller) getPolicyBinding(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	member, err := decodePolicyMemberRequest(request.Payload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	value, ok := c.policy.Binding(member.Key)
	if !ok {
		return resource.OperationResult{}, fmt.Errorf("%w: policy binding %q", auth.ErrPolicyNotFound, member.Key)
	}
	return encodePolicyResult(&value)
}

func (c *Controller) createPolicyBinding(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinPolicyBindings); err != nil {
		return resource.OperationResult{}, err
	}
	var input protocol.PolicyBindingCreateV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if err := c.policy.ValidateOwnerScope(input.Scope); err != nil {
		return resource.OperationResult{}, fmt.Errorf("validate policy binding scope: %w", err)
	}
	value, err := c.policy.CreateBinding(input, request.Subject)
	if err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinPolicyBindings, value.BindingID, "active")
	return encodePolicyResult(&value)
}

func (c *Controller) revokePolicyBinding(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinPolicyBindings); err != nil {
		return resource.OperationResult{}, err
	}
	var input protocol.PolicyBindingRevokeV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if err := c.policy.RevokeBinding(input.BindingID); err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinPolicyBindings, input.BindingID, "revoked")
	return encodePolicyResult(&protocol.ManagementResultV1{Version: 1, Status: "ok"})
}

func (c *Controller) evaluatePolicy(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	var input protocol.PolicyEvaluateRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	policyRequest, err := policyRequestFromGrant(protocol.PolicyGrantV1{Version: input.Version, Subject: input.Subject, Capability: input.Capability, ResourceNode: input.ResourceNode, ResourceName: input.ResourceName})
	if err != nil {
		return resource.OperationResult{}, err
	}
	value, err := c.policy.Evaluate(policyRequest)
	if err != nil {
		return resource.OperationResult{}, err
	}
	return encodePolicyResult(&value)
}

func (c *Controller) listPolicyGrants(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	values := c.policy.Grants()
	members := make([]protocol.CollectionMemberV1, 0, len(values))
	for _, value := range values {
		grant := policyGrantFromRequest(value)
		members = append(members, protocol.CollectionMemberV1{
			Key: policyGrantKey(grant), Kind: "policy-grant", Label: grant.Subject + " · " + grant.Capability,
			ContentType: contentTypeJSON, Schema: protocol.SchemaPolicyGrantV1,
			Capabilities: []protocol.CapabilityID{protocol.CapabilityGet, protocol.CapabilityRevoke},
			Attributes:   map[string]string{"capability": grant.Capability, "resource_name": grant.ResourceName, "resource_node": grant.ResourceNode, "subject": grant.Subject},
		})
	}
	return c.policyPage(request, policyGrantsCursor, protocol.BuiltinPolicyGrants, members)
}

func (c *Controller) getPolicyGrant(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	member, err := decodePolicyMemberRequest(request.Payload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	for _, value := range c.policy.Grants() {
		grant := policyGrantFromRequest(value)
		if policyGrantKey(grant) == member.Key {
			return encodePolicyResult(&grant)
		}
	}
	return resource.OperationResult{}, fmt.Errorf("%w: exact policy grant %q", auth.ErrPolicyNotFound, member.Key)
}

func (c *Controller) createPolicyGrant(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinPolicyGrants); err != nil {
		return resource.OperationResult{}, err
	}
	var input protocol.PolicyGrantV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	value, err := policyRequestFromGrant(input)
	if err != nil {
		return resource.OperationResult{}, err
	}
	if err := c.policy.Grant(value); err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinPolicyGrants, policyGrantKey(input), "active")
	return encodePolicyResult(&input)
}

func (c *Controller) revokeExactPolicyGrant(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinPolicyGrants); err != nil {
		return resource.OperationResult{}, err
	}
	var input protocol.PolicyGrantV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	value, err := policyRequestFromGrant(input)
	if err != nil {
		return resource.OperationResult{}, err
	}
	if err := c.policy.Revoke(value); err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinPolicyGrants, policyGrantKey(input), "revoked")
	return encodePolicyResult(&protocol.ManagementResultV1{Version: 1, Status: "ok"})
}

func (c *Controller) createLegacyPolicyGrant(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinManagementPolicyGrant); err != nil {
		return resource.OperationResult{}, err
	}
	value, err := decodePolicyRule(request.Payload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	if err := c.policy.Grant(value); err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinManagementPolicyGrant, policyGrantKey(policyGrantFromRequest(value)), "active")
	return encodePolicyResult(&protocol.ManagementResultV1{Version: 1, Status: "ok"})
}

func (c *Controller) revokeLegacyPolicyGrant(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	if err := c.requirePolicySuperadmin(request, protocol.BuiltinManagementPolicyRevoke); err != nil {
		return resource.OperationResult{}, err
	}
	value, err := decodePolicyRule(request.Payload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	if err := c.policy.Revoke(value); err != nil {
		return resource.OperationResult{}, err
	}
	c.recordPolicyMutation(request, protocol.BuiltinManagementPolicyRevoke, policyGrantKey(policyGrantFromRequest(value)), "revoked")
	return encodePolicyResult(&protocol.ManagementResultV1{Version: 1, Status: "ok"})
}

func decodePolicyMemberRequest(payload []byte) (protocol.CollectionMemberRequestV1, error) {
	var input protocol.CollectionMemberRequestV1
	err := protocol.DecodeJSONPayload(payload, protocol.DefaultMaxPayload, &input)
	return input, err
}

func (c *Controller) policyPage(request resource.OperationRequest, cursorKind, resourceName string, members []protocol.CollectionMemberV1) (resource.OperationResult, error) {
	var input protocol.CollectionListRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if input.Parent != "" {
		return resource.OperationResult{}, errors.New("policy collections are flat and do not support parent")
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Key < members[j].Key })
	revision := c.policy.Generation()
	start, err := policyCollectionStart(input.Cursor, cursorKind, revision, members)
	if err != nil {
		return resource.OperationResult{}, err
	}
	end := start + input.Limit
	if end > len(members) {
		end = len(members)
	}
	page := protocol.CollectionPageV1{Version: 1, Revision: revision, Members: members[start:end]}
	if end < len(members) {
		page.NextCursor, err = encodePolicyCursor(cursorKind, revision, members[end-1].Key)
		if err != nil {
			return resource.OperationResult{}, err
		}
	}
	registered, ok := c.node.Registry().Resolve(protocol.ResourceID{Owner: c.node.ID(), Name: resourceName})
	if ok {
		err = page.ValidateForDescriptor(registered.Descriptor())
	} else {
		err = page.Validate()
	}
	if err != nil {
		return resource.OperationResult{}, fmt.Errorf("validate policy collection page: %w", err)
	}
	return encodePolicyResult(&page)
}

func policyCollectionStart(encoded, collection string, revision uint64, members []protocol.CollectionMemberV1) (int, error) {
	if encoded == "" {
		return 0, nil
	}
	cursor, err := decodePolicyCursor(encoded)
	if err != nil {
		return 0, fmt.Errorf("invalid policy collection cursor: %w", err)
	}
	if cursor.Collection != collection {
		return 0, errors.New("invalid policy collection cursor: collection mismatch")
	}
	if cursor.Revision != revision {
		return 0, errors.New("invalid policy collection cursor: stale revision")
	}
	index := sort.Search(len(members), func(index int) bool { return members[index].Key >= cursor.After })
	if index >= len(members) || members[index].Key != cursor.After {
		return 0, errors.New("invalid policy collection cursor: member no longer exists")
	}
	return index + 1, nil
}

func encodePolicyCursor(collection string, revision uint64, after string) (string, error) {
	payload, err := json.Marshal(policyCursor{Version: 1, Collection: collection, Revision: revision, After: after})
	if err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(payload)
	if len(value) > protocol.MaxCollectionCursorBytes {
		return "", errors.New("policy collection cursor exceeds protocol limit")
	}
	return value, nil
}

func decodePolicyCursor(value string) (policyCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return policyCursor{}, errors.New("cursor encoding is malformed")
	}
	var cursor policyCursor
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return policyCursor{}, errors.New("cursor payload is malformed")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return policyCursor{}, errors.New("cursor payload has trailing data")
	}
	if cursor.Version != 1 || cursor.Collection == "" || cursor.Revision == 0 || cursor.After == "" {
		return policyCursor{}, errors.New("cursor fields are invalid")
	}
	return cursor, nil
}

func policyRequestFromGrant(value protocol.PolicyGrantV1) (auth.Request, error) {
	if err := value.Validate(); err != nil {
		return auth.Request{}, err
	}
	subject, _ := strconv.ParseUint(value.Subject, 10, 64)
	owner, _ := strconv.ParseUint(value.ResourceNode, 10, 64)
	return auth.Request{
		Subject: protocol.NodeID(subject), Capability: protocol.CapabilityID(value.Capability),
		Resource: protocol.ResourceID{Owner: protocol.NodeID(owner), Name: value.ResourceName},
	}, nil
}

func policyGrantFromRequest(value auth.Request) protocol.PolicyGrantV1 {
	capability := value.Capability
	if capability == "" {
		capability = protocol.CapabilityID(value.Action)
	}
	return protocol.PolicyGrantV1{
		Version: 1, Subject: strconv.FormatUint(uint64(value.Subject), 10), Capability: string(capability),
		ResourceNode: strconv.FormatUint(uint64(value.Resource.Owner), 10), ResourceName: value.Resource.Name,
	}
}

func policyGrantKey(value protocol.PolicyGrantV1) string {
	digest := sha256.Sum256([]byte(value.Subject + "\x00" + value.Capability + "\x00" + value.ResourceNode + "\x00" + value.ResourceName))
	return hex.EncodeToString(digest[:])
}

func encodePolicyResult(value protocol.ValidatedPayload) (resource.OperationResult, error) {
	payload, err := protocol.EncodeJSONPayload(value, protocol.DefaultMaxPayload)
	return resource.OperationResult{Payload: payload}, err
}
