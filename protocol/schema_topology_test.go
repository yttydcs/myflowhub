package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func topologyQueryFixture() ManagementTopologyQueryV1 {
	return ManagementTopologyQueryV1{
		Version: 1, RootNodeID: "41", Depth: 2,
		InstanceID: "0123456789abcdef0123456789abcdef", Revision: 1,
		Nodes: []TopologyQueryNodeV1{
			{NodeID: "43", ParentID: "42", Role: "node", Generation: 1},
			{NodeID: "42", ParentID: "41", Role: "relay", Generation: 2, HasChildren: true},
			{NodeID: "41", DisplayName: "Query root", Role: "relay", Generation: 3, HasChildren: true},
		},
	}
}

func topologyChain(count, depth int) ManagementTopologyQueryV1 {
	value := topologyQueryFixture()
	value.RootNodeID, value.Depth = "1", depth
	value.Nodes = make([]TopologyQueryNodeV1, count)
	for index := range value.Nodes {
		node := &value.Nodes[index]
		node.NodeID, node.Role, node.Generation = strconv.Itoa(index+1), "node", 1
		node.HasChildren = index < count-1
		if index > 0 {
			node.ParentID = strconv.Itoa(index)
		}
	}
	return value
}

func TestTopologyRequestDepthRoundTrip(t *testing.T) {
	for _, depth := range []int{0, 1, 2, MaxItems} {
		t.Run(strconv.Itoa(depth), func(t *testing.T) {
			request := ManagementTopologyQueryRequestV1{Version: 1, Depth: depth}
			payload, err := EncodeJSONPayload(request, DefaultMaxPayload)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintf(`{"version":1,"depth":%d}`, depth)
			if string(payload) != want {
				t.Fatalf("request = %s, want %s", payload, want)
			}
			var decoded ManagementTopologyQueryRequestV1
			if err := DecodeJSONPayload(payload, DefaultMaxPayload, &decoded); err != nil || decoded != request {
				t.Fatalf("round trip = %#v, %v", decoded, err)
			}
			children := ManagementTopologyChildrenRequestV1{Version: 1, Depth: depth}
			_, err = EncodeJSONPayload(children, DefaultMaxPayload)
			if (err == nil) != (depth == 1) {
				t.Fatalf("children typed depth %d: %v", depth, err)
			}
			var decodedChildren ManagementTopologyChildrenRequestV1
			err = DecodeJSONPayload(payload, DefaultMaxPayload, &decodedChildren)
			if (err == nil) != (depth == 1) {
				t.Fatalf("children wire depth %d: %v", depth, err)
			}
		})
	}
	for _, capability := range []CapabilityID{CapabilityChildren, CapabilitySubtree} {
		if err := capability.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTopologyRequestsRejectInvalidWire(t *testing.T) {
	tests := []string{
		`{}`, `null`, `[]`, `true`, `1`, `"request"`,
		`{"version":1}`, `{"version":1,"depth":null}`,
		`{"version":1,"depth":-1}`, `{"version":1,"depth":1.5}`,
		`{"version":1,"depth":1.0}`, `{"version":1,"depth":1e0}`,
		`{"version":1,"depth":"1"}`, `{"version":1,"depth":true}`,
		`{"version":1,"depth":[]}`, `{"version":1,"depth":{}}`,
		`{"version":1,"depth":4097}`, `{"version":1,"depth":9007199254740992}`,
		`{"version":1,"depth":18446744073709551616}`,
		`{"version":1,"depth":1,"root_node_id":"42"}`,
		`{"version":1,"depth":1,"unknown":true}`,
		`{"version":1,"Depth":1}`, `{"Version":1,"depth":1}`,
		`{"version":1,"depth":1,"depth":0}`,
		`{"version":1,"depth":0,"depth":1}`,
		`{"version":1,"depth":1,"depth":null}`,
		`{"version":2,"version":1,"depth":1}`,
		`{"depth":1}`, `{"version":null,"depth":1}`,
		`{"version":0,"depth":1}`, `{"version":2,"depth":1}`,
		`{"version":1,"depth":1} {}`, `{"version":1,"depth":1`,
	}
	for _, payload := range tests {
		t.Run(payload, func(t *testing.T) {
			for _, target := range []ValidatedPayload{
				&ManagementTopologyChildrenRequestV1{Version: 1, Depth: 1},
				&ManagementTopologyQueryRequestV1{Version: 1, Depth: 1},
			} {
				if err := DecodeJSONPayload([]byte(payload), DefaultMaxPayload, target); !errors.Is(err, ErrInvalidPayload) {
					t.Fatalf("%T accepted invalid request or lost error category: %v", target, err)
				}
			}
		})
	}
	for _, depth := range []int{-1, MaxItems + 1} {
		if _, err := EncodeJSONPayload(ManagementTopologyQueryRequestV1{Version: 1, Depth: depth}, DefaultMaxPayload); !errors.Is(err, ErrInvalidPayload) {
			t.Fatalf("invalid typed depth %d: %v", depth, err)
		}
	}
}

func TestTopologyQueryRoundTripAndBoundaries(t *testing.T) {
	for _, depth := range []int{0, 1, 2, MaxItems} {
		value := topologyQueryFixture()
		value.Depth, value.Revision = depth, MaxTopologyRevision
		if depth == 1 {
			value.Nodes = value.Nodes[1:]
		}
		payload, err := EncodeJSONPayload(value, DefaultMaxPayload)
		if err != nil {
			t.Fatalf("depth %d: %v", depth, err)
		}
		var decoded ManagementTopologyQueryV1
		if err := DecodeJSONPayload(payload, DefaultMaxPayload, &decoded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(decoded, value) {
			t.Fatalf("round trip mismatch: %#v", decoded)
		}
		if depth == 0 && !bytes.Contains(payload, []byte(`"depth":0`)) {
			t.Fatal("response omitted depth zero")
		}
		// A boundary may report an unreturned next layer, or a confirmed leaf.
		if depth == 1 {
			value.Nodes[0].HasChildren = false
			if err := value.Validate(); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, depth := range []int{0, 1, 2, MaxItems} {
		if err := topologyChain(1, depth).Validate(); err != nil {
			t.Fatalf("single root at depth %d: %v", depth, err)
		}
	}
	for _, depth := range []int{0, MaxItems - 1, MaxItems} {
		value := topologyChain(MaxItems, depth)
		payload, err := EncodeJSONPayload(value, DefaultMaxPayload)
		if err != nil {
			t.Fatal(err)
		}
		var decoded ManagementTopologyQueryV1
		if err := DecodeJSONPayload(payload, DefaultMaxPayload, &decoded); err != nil {
			t.Fatalf("maximum chain at depth %d: %v", depth, err)
		}
	}
	wide := topologyChain(MaxItems, 1)
	for index := 1; index < len(wide.Nodes); index++ {
		wide.Nodes[index].ParentID, wide.Nodes[index].HasChildren = "1", false
	}
	if err := wide.Validate(); err != nil {
		t.Fatalf("maximum breadth: %v", err)
	}
}

func TestTopologyQueryRejectsInvalidMetadataAndGraph(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ManagementTopologyQueryV1)
	}{
		{"version", func(v *ManagementTopologyQueryV1) { v.Version = 2 }},
		{"negative-depth", func(v *ManagementTopologyQueryV1) { v.Depth = -1 }},
		{"large-depth", func(v *ManagementTopologyQueryV1) { v.Depth = MaxItems + 1 }},
		{"zero-revision", func(v *ManagementTopologyQueryV1) { v.Revision = 0 }},
		{"unsafe-revision", func(v *ManagementTopologyQueryV1) { v.Revision = MaxTopologyRevision + 1 }},
		{"uint64-revision", func(v *ManagementTopologyQueryV1) { v.Revision = ^uint64(0) }},
		{"empty-instance", func(v *ManagementTopologyQueryV1) { v.InstanceID = "" }},
		{"short-instance", func(v *ManagementTopologyQueryV1) { v.InstanceID = v.InstanceID[:31] }},
		{"long-instance", func(v *ManagementTopologyQueryV1) { v.InstanceID += "0" }},
		{"uppercase-instance", func(v *ManagementTopologyQueryV1) { v.InstanceID = strings.ToUpper(v.InstanceID) }},
		{"nonhex-instance", func(v *ManagementTopologyQueryV1) { v.InstanceID = strings.Repeat("g", 32) }},
		{"empty-nodes", func(v *ManagementTopologyQueryV1) { v.Nodes = nil }},
		{"too-many-nodes", func(v *ManagementTopologyQueryV1) { *v = topologyChain(MaxItems+1, 0) }},
		{"missing-root", func(v *ManagementTopologyQueryV1) { v.RootNodeID = "44" }},
		{"wrong-root", func(v *ManagementTopologyQueryV1) { v.RootNodeID = "42" }},
		{"invalid-root-id", func(v *ManagementTopologyQueryV1) { v.RootNodeID = "+41" }},
		{"root-parent", func(v *ManagementTopologyQueryV1) { v.Nodes[2].ParentID = "40" }},
		{"root-cycle", func(v *ManagementTopologyQueryV1) { v.Nodes[2].ParentID = "43" }},
		{"multiple-roots", func(v *ManagementTopologyQueryV1) { v.Nodes[0].ParentID = "" }},
		{"no-root", func(v *ManagementTopologyQueryV1) { v.Nodes = v.Nodes[:2] }},
		{"absent-root-in-cycle", func(v *ManagementTopologyQueryV1) {
			v.Nodes = v.Nodes[:2]
			v.Nodes[1].ParentID = "43"
		}},
		{"duplicate-id", func(v *ManagementTopologyQueryV1) { v.Nodes = append(v.Nodes, v.Nodes[0]) }},
		{"external-parent", func(v *ManagementTopologyQueryV1) { v.Nodes[0].ParentID = "99" }},
		{"self-cycle", func(v *ManagementTopologyQueryV1) { v.Nodes[0].ParentID = "43" }},
		{"unreachable-cycle", func(v *ManagementTopologyQueryV1) {
			v.Nodes[1].ParentID = "43"
			v.Nodes[0].HasChildren, v.Nodes[2].HasChildren = true, false
		}},
		{"depth-exceeded", func(v *ManagementTopologyQueryV1) { v.Depth = 1 }},
		{"root-hidden-children", func(v *ManagementTopologyQueryV1) { v.Nodes[2].HasChildren = false }},
		{"internal-hidden-children", func(v *ManagementTopologyQueryV1) { v.Nodes[1].HasChildren = false }},
		{"internal-missing-children", func(v *ManagementTopologyQueryV1) { v.Depth = 3; v.Nodes[0].HasChildren = true }},
		{"unlimited-missing-children", func(v *ManagementTopologyQueryV1) { v.Depth = 0; v.Nodes[0].HasChildren = true }},
		{"zero-node-id", func(v *ManagementTopologyQueryV1) { v.Nodes[0].NodeID = "0" }},
		{"noncanonical-node-id", func(v *ManagementTopologyQueryV1) { v.Nodes[0].NodeID = "043" }},
		{"overflow-node-id", func(v *ManagementTopologyQueryV1) { v.Nodes[0].NodeID = "18446744073709551616" }},
		{"noncanonical-parent-id", func(v *ManagementTopologyQueryV1) { v.Nodes[0].ParentID = "042" }},
		{"empty-role", func(v *ManagementTopologyQueryV1) { v.Nodes[0].Role = "" }},
		{"large-role", func(v *ManagementTopologyQueryV1) { v.Nodes[0].Role = strings.Repeat("r", MaxIdentifierBytes+1) }},
		{"large-name", func(v *ManagementTopologyQueryV1) { v.Nodes[0].DisplayName = strings.Repeat("n", MaxLabelBytes+1) }},
		{"invalid-utf8-name", func(v *ManagementTopologyQueryV1) { v.Nodes[0].DisplayName = string([]byte{0xff}) }},
		{"zero-generation", func(v *ManagementTopologyQueryV1) { v.Nodes[0].Generation = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := topologyQueryFixture()
			test.mutate(&value)
			if _, err := EncodeJSONPayload(value, DefaultMaxPayload); !errors.Is(err, ErrInvalidPayload) {
				t.Fatalf("invalid response accepted or lost error category: %v", err)
			}
		})
	}
}

func TestTopologyQueryWireRequiresNonNullFields(t *testing.T) {
	payload, err := json.Marshal(topologyQueryFixture())
	if err != nil {
		t.Fatal(err)
	}
	var original map[string]json.RawMessage
	if err := json.Unmarshal(payload, &original); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"version", "root_node_id", "depth", "instance_id", "revision", "nodes"} {
		for _, null := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/null=%t", field, null), func(t *testing.T) {
				object := make(map[string]json.RawMessage, len(original))
				for name, raw := range original {
					object[name] = raw
				}
				if null {
					object[field] = json.RawMessage(`null`)
				} else {
					delete(object, field)
				}
				data, err := json.Marshal(object)
				if err != nil {
					t.Fatal(err)
				}
				// A reused destination must not supply a missing wire field.
				value := topologyQueryFixture()
				if err := DecodeJSONPayload(data, DefaultMaxPayload, &value); !errors.Is(err, ErrInvalidPayload) {
					t.Fatalf("accepted missing/null %s: %v", field, err)
				}
			})
		}
	}
	for _, field := range []string{"node_id", "role", "generation", "has_children", "parent_id", "display_name"} {
		for _, null := range []bool{false, true} {
			if !null && (field == "parent_id" || field == "display_name") {
				continue
			}
			t.Run(fmt.Sprintf("node/%s/null=%t", field, null), func(t *testing.T) {
				var node map[string]json.RawMessage
				if err := json.Unmarshal([]byte(`{"node_id":"41","role":"relay","generation":1,"has_children":false}`), &node); err != nil {
					t.Fatal(err)
				}
				if null {
					node[field] = json.RawMessage(`null`)
				} else {
					delete(node, field)
				}
				data, err := json.Marshal(node)
				if err != nil {
					t.Fatal(err)
				}
				response := fmt.Sprintf(`{"version":1,"root_node_id":"41","depth":0,"instance_id":"%s","revision":1,"nodes":[%s]}`, topologyQueryFixture().InstanceID, data)
				var decoded ManagementTopologyQueryV1
				if err := DecodeJSONPayload([]byte(response), DefaultMaxPayload, &decoded); !errors.Is(err, ErrInvalidPayload) {
					t.Fatalf("accepted invalid node field %s: %v", field, err)
				}
			})
		}
	}
}

func TestTopologyQueryRejectsInvalidWireShapes(t *testing.T) {
	payload, err := json.Marshal(topologyQueryFixture())
	if err != nil {
		t.Fatal(err)
	}
	valid := string(payload)
	tests := []string{
		`null`, `[]`, valid + ` {}`,
		strings.Replace(valid, `"depth":2`, `"depth":2.5`, 1),
		strings.Replace(valid, `"depth":2`, `"depth":"2"`, 1),
		strings.Replace(valid, `"depth":2`, `"depth":true`, 1),
		strings.Replace(valid, `"depth":2`, `"depth":4097`, 1),
		strings.Replace(valid, `"revision":1`, `"revision":9007199254740992`, 1),
		strings.Replace(valid, `"revision":1`, `"revision":1.0`, 1),
		strings.Replace(valid, `"revision":1`, `"revision":-1`, 1),
		strings.Replace(valid, `"revision":1`, `"revision":"1"`, 1),
		strings.Replace(valid, `"node_id":"43"`, `"node_id":43`, 1),
		strings.Replace(valid, `"has_children":false`, `"has_children":0`, 1),
		strings.Replace(valid, `"has_children":false`, `"has_children":"false"`, 1),
		strings.Replace(valid, `"has_children":false`, `"has_children":false,"has_children":true`, 1),
		strings.Replace(valid, `"has_children":false`, `"Has_Children":false`, 1),
		strings.Replace(valid, `"node_id":"41"`, `"node_id":"41","parent_id":""`, 1),
		strings.Replace(valid, `"version":1`, `"version":1,"unknown":true`, 1),
		strings.Replace(valid, `"depth":2`, `"depth":2,"depth":0`, 1),
		strings.Replace(valid, `"node_id":"43"`, `"node_id":"43","unknown":true`, 1),
	}
	for index, data := range tests {
		var decoded ManagementTopologyQueryV1
		if err := DecodeJSONPayload([]byte(data), DefaultMaxPayload, &decoded); !errors.Is(err, ErrInvalidPayload) {
			t.Fatalf("invalid wire %d accepted or lost error category: %v", index, err)
		}
	}
}

func TestTopologyPayloadLimits(t *testing.T) {
	request := ManagementTopologyQueryRequestV1{Version: 1, Depth: 0}
	payload, err := EncodeJSONPayload(request, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeJSONPayload(request, len(payload)-1); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("small encode limit: %v", err)
	}
	var decodedRequest ManagementTopologyQueryRequestV1
	if err := DecodeJSONPayload(payload, len(payload)-1, &decodedRequest); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("small decode limit: %v", err)
	}
	boundary := append(payload, bytes.Repeat([]byte(" "), DefaultMaxPayload-len(payload))...)
	if err := DecodeJSONPayload(boundary, 0, &decodedRequest); err != nil {
		t.Fatalf("default payload boundary: %v", err)
	}
	if err := DecodeJSONPayload(append(boundary, ' '), 0, &decodedRequest); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("default payload overflow: %v", err)
	}
	large := topologyChain(MaxItems, 0)
	for index := range large.Nodes {
		large.Nodes[index].DisplayName = strings.Repeat("n", MaxLabelBytes)
	}
	if err := large.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeJSONPayload(large, DefaultMaxPayload); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("oversized response encode: %v", err)
	}
	data, err := json.Marshal(large)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ManagementTopologyQueryV1
	if err := DecodeJSONPayload(data, DefaultMaxPayload, &decoded); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("oversized response decode: %v", err)
	}
	if err := json.Unmarshal(data, &decoded); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("direct unmarshal bypassed response cap: %v", err)
	}
}

func TestTopologyLegacySnapshotRemainsCompatible(t *testing.T) {
	const payload = `{"version":1,"epoch":1,"nodes":[{"node_id":"41","role":"relay","generation":1}]}`
	var value ManagementTopologyV1
	if err := DecodeJSONPayload([]byte(payload), DefaultMaxPayload, &value); err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeJSONPayload(value, DefaultMaxPayload)
	if err != nil || string(encoded) != payload {
		t.Fatalf("legacy wire changed: %s, %v", encoded, err)
	}
}

func BenchmarkTopologyQueryValidateChain(b *testing.B) {
	for _, count := range []int{256, 1024, MaxItems} {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			value := topologyChain(count, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := value.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
