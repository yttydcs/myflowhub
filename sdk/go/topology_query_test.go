package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestManagementQueryTopologyUsesAttachedNodeAndExplicitDepth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	root, child, network := connectedSDKNodes(t, ctx, "sdk-topology-query")
	defer network.Close()
	defer root.Close()
	defer child.Close()
	requests := make(chan resource.OperationRequest, 4)
	provider := topologyQueryFixture(root.ID(), func(_ context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
		requests <- request
		var input protocol.ManagementTopologyQueryRequestV1
		if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
			return resource.OperationResult{}, err
		}
		return resource.OperationResult{Payload: []byte(topologyQueryJSON(input.Depth))}, nil
	})
	if err := root.Registry().Register(provider); err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(child)
	if err != nil {
		t.Fatal(err)
	}
	management, err := client.Management(root.ID())
	if err != nil {
		t.Fatal(err)
	}
	for _, depth := range []int{1, 0, 2, protocol.MaxItems} {
		t.Run(fmt.Sprintf("depth_%d", depth), func(t *testing.T) {
			value, err := management.QueryTopology(ctx, depth)
			if err != nil || value.RootNodeID != "1" || value.Depth != depth || len(value.Nodes) != 1 {
				t.Fatalf("unexpected topology: %+v (%v)", value, err)
			}
			request := <-requests
			capability, schema := protocol.CapabilitySubtree, protocol.SchemaManagementTopologyQueryRequestV1
			if depth == 1 {
				capability, schema = protocol.CapabilityChildren, protocol.SchemaManagementTopologyChildrenRequestV1
			}
			if request.Subject != child.ID() || request.Capability != capability || request.Schema != schema {
				t.Fatalf("request did not use attached identity and descriptor: %+v", request)
			}
			var wire map[string]json.RawMessage
			if err := json.Unmarshal(request.Payload, &wire); err != nil {
				t.Fatal(err)
			}
			if len(wire) != 2 || string(wire["version"]) != "1" || string(wire["depth"]) != fmt.Sprint(depth) {
				t.Fatalf("depth must be explicit on the wire, including zero: %s", request.Payload)
			}
		})
	}
}

func TestManagementQueryTopologyRejectsInvalidInputBeforeIO(t *testing.T) {
	runtime := newLocalNode(t, 1)
	defer runtime.Close()
	var calls atomic.Int32
	if err := runtime.Registry().Register(topologyQueryFixture(runtime.ID(), func(context.Context, resource.OperationRequest) (resource.OperationResult, error) {
		calls.Add(1)
		return resource.OperationResult{Payload: []byte(topologyQueryJSON(1))}, nil
	})); err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	management, err := client.Management(runtime.ID())
	if err != nil {
		t.Fatal(err)
	}
	for _, depth := range []int{-1, protocol.MaxItems + 1, int(^uint(0) >> 1)} {
		_, err := management.QueryTopology(context.Background(), depth)
		assertSDKError(t, err, protocol.CodeMalformed, false)
		if !strings.Contains(err.Error(), "depth") {
			t.Fatalf("invalid depth error lacks context: %v", err)
		}
	}
	if _, err := client.Management(0); err == nil {
		t.Fatal("zero topology owner was accepted")
	}
	if _, err := management.QueryTopology(nil, 1); err == nil {
		t.Fatal("nil context was accepted")
	}
	for _, missing := range []*ManagementClient{nil, {}} {
		if _, err := missing.QueryTopology(context.Background(), 1); err == nil {
			t.Fatal("missing management client was accepted")
		}
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("invalid input reached the provider %d times", got)
	}
	// An unattached client would fail at the runtime boundary if validation
	// were moved after I/O. Invalid depth must still be a typed payload error.
	unattached := &ManagementClient{client: &Client{}, owner: 1}
	_, err = unattached.QueryTopology(context.Background(), -1)
	assertSDKError(t, err, protocol.CodeMalformed, false)
}

func TestManagementQueryTopologyRejectsMalformedResponses(t *testing.T) {
	valid := topologyQueryJSON(1)
	for _, test := range []struct {
		name    string
		payload string
		schema  string
		message string
	}{
		{"invalid_json", "{", "", "decode"},
		{"wrong_owner", strings.ReplaceAll(valid, `"1"`, `"2"`), "", "owner"},
		{"wrong_depth", topologyQueryJSON(2), "", "requested depth"},
		{"wrong_version", strings.Replace(valid, `"version":1`, `"version":2`, 1), "", "version"},
		{"missing_depth", strings.Replace(valid, `"depth":1,`, "", 1), "", "depth"},
		{"null_depth", strings.Replace(valid, `"depth":1`, `"depth":null`, 1), "", "depth"},
		{"unknown_field", strings.Replace(valid, `"depth":1`, `"depth":1,"extra":true`, 1), "", "extra"},
		{"missing_children_hint", strings.Replace(valid, `,"has_children":false`, "", 1), "", "has_children"},
		{"invalid_revision", strings.Replace(valid, `"revision":1`, `"revision":0`, 1), "", "revision"},
		{"invalid_instance", strings.Replace(valid, strings.Repeat("a", 32), "bad", 1), "", "instance_id"},
		{"absent_root", strings.Replace(valid, `"node_id":"1"`, `"node_id":"2","parent_id":"1"`, 1), "", "root_node_id"},
		{"wrong_output_schema", valid, protocol.SchemaManagementTopologyV1, "schema"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runtime := newLocalNode(t, 1)
			defer runtime.Close()
			if err := runtime.Registry().Register(topologyQueryFixture(runtime.ID(), func(context.Context, resource.OperationRequest) (resource.OperationResult, error) {
				return resource.OperationResult{Schema: test.schema, Payload: []byte(test.payload)}, nil
			})); err != nil {
				t.Fatal(err)
			}
			client, err := NewAttachedClient(runtime)
			if err != nil {
				t.Fatal(err)
			}
			management, err := client.Management(runtime.ID())
			if err != nil {
				t.Fatal(err)
			}
			value, err := management.QueryTopology(context.Background(), 1)
			assertSDKError(t, err, protocol.CodeMalformed, false)
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error lacks %q: %v", test.message, err)
			}
			if !reflect.DeepEqual(value, protocol.ManagementTopologyQueryV1{}) {
				t.Fatalf("malformed response leaked a usable partial result: %+v", value)
			}
		})
	}
}

func TestManagementQueryTopologyDoesNotFallBackToLegacyRead(t *testing.T) {
	runtime := newLocalNode(t, 1)
	defer runtime.Close()
	legacy, err := resource.NewVariable(resource.VariableDescriptor(
		protocol.ResourceID{Owner: runtime.ID(), Name: protocol.BuiltinManagementTopology},
		"application/json", protocol.SchemaManagementTopologyV1, "management.topology.read", protocol.DefaultMaxPayload,
	), []byte(`{"version":1,"epoch":1,"nodes":[{"node_id":"1","role":"root","generation":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Registry().Register(legacy); err != nil {
		t.Fatal(err)
	}
	client, err := NewAttachedClient(runtime)
	if err != nil {
		t.Fatal(err)
	}
	management, err := client.Management(runtime.ID())
	if err != nil {
		t.Fatal(err)
	}
	for _, depth := range []int{1, 0, 2} {
		_, err := management.QueryTopology(context.Background(), depth)
		assertSDKError(t, err, protocol.CodeUnsupported, false)
	}
	if value, err := management.Topology(context.Background()); err != nil || len(value.Nodes) != 1 || value.Nodes[0].NodeID != "1" {
		t.Fatalf("legacy snapshot helper changed: %+v (%v)", value, err)
	}
	if err := runtime.Registry().Remove(legacy.Descriptor().ID); err != nil {
		t.Fatal(err)
	}
	_, err = management.QueryTopology(context.Background(), 1)
	assertSDKError(t, err, protocol.CodeNotFound, false)
}

// This fixture deliberately leaves validation to the real Registry and SDK,
// allowing malformed provider results to exercise both boundaries.
type topologyQueryTestResource struct {
	descriptor resource.Descriptor
	handler    resource.Handler
}

func (r topologyQueryTestResource) Descriptor() resource.Descriptor { return r.descriptor }

func (r topologyQueryTestResource) Operate(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	return r.handler(ctx, request)
}

func topologyQueryFixture(owner protocol.NodeID, handler resource.Handler) topologyQueryTestResource {
	descriptor := resource.Descriptor{
		ID: protocol.ResourceID{Owner: owner, Name: protocol.BuiltinManagementTopology}, Type: protocol.ResourceTypeVariable, TypeVersion: 1,
		Capabilities: []protocol.CapabilityDescriptorV2{
			{Name: protocol.CapabilityChildren, Permission: "management.topology.children", InputSchema: protocol.SchemaManagementTopologyChildrenRequestV1, OutputSchema: protocol.SchemaManagementTopologyQueryV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
			{Name: protocol.CapabilitySubtree, Permission: "management.topology.subtree", InputSchema: protocol.SchemaManagementTopologyQueryRequestV1, OutputSchema: protocol.SchemaManagementTopologyQueryV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
		},
		Schemas: []protocol.SchemaDescriptorV2{
			{ID: protocol.SchemaManagementTopologyChildrenRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaManagementTopologyQueryRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaManagementTopologyQueryV1, ContentType: "application/json"},
		},
		Limits: protocol.ResourceLimitsV2{MaxPayloadBytes: protocol.DefaultMaxPayload},
	}
	descriptor.Sort()
	return topologyQueryTestResource{descriptor: descriptor, handler: handler}
}

func topologyQueryJSON(depth int) string {
	return fmt.Sprintf(`{"version":1,"root_node_id":"1","depth":%d,"instance_id":"%s","revision":1,"nodes":[{"node_id":"1","role":"root","generation":1,"has_children":false}]}`, depth, strings.Repeat("a", 32))
}
