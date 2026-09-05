package integration_test

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/feature/management"
	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/memory"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

// The six-level chain is 1 -> 2 -> 3 -> 4 -> 5 -> 6. Node 7 queries
// across subtrees from root 1; node 8 is a leaf under relay 2 with old grants.
type topologyDiscoveryFixture struct {
	states map[protocol.NodeID]*hostconfig.Runtime
	nodes  map[protocol.NodeID]*node.Node
}

func TestTopologyDiscovery(t *testing.T) {
	for _, transport := range []string{"memory", "tcp"} {
		t.Run(transport, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)
			var driver link.Driver
			endpoint := func(string) link.Endpoint { return "127.0.0.1:0" }
			if transport == "memory" {
				network := memory.NewNetwork()
				driver = network
				endpoint = func(name string) link.Endpoint { return link.Endpoint("topology-discovery-" + name) }
				t.Cleanup(func() {
					if err := network.Close(); err != nil {
						t.Errorf("close memory network: %v", err)
					}
				})
			} else {
				driver = tcp.Driver{}
			}
			fixture := newTopologyDiscoveryFixture(t, ctx, driver, endpoint)
			client := fixture.client(t, 7)
			root := topologyDiscoveryManagement(t, client, 1)
			relay := topologyDiscoveryManagement(t, client, 2)

			t.Run("DepthBoundaries", func(t *testing.T) {
				for _, owner := range []protocol.NodeID{1, 2} {
					for _, capability := range []protocol.CapabilityID{protocol.CapabilityChildren, protocol.CapabilitySubtree} {
						fixture.grant(t, 7, owner, protocol.BuiltinManagementTopology, capability)
					}
				}
				fixture.grant(t, 7, 1, protocol.BuiltinManagementTopology, protocol.CapabilityRead)
				for _, test := range []struct {
					name  string
					owner protocol.NodeID
					depth int
					ids   []protocol.NodeID
				}{
					{"root_all", 1, 0, []protocol.NodeID{1, 2, 3, 4, 5, 6, 7, 8}},
					{"root_children", 1, 1, []protocol.NodeID{1, 2, 7}},
					{"root_two_levels", 1, 2, []protocol.NodeID{1, 2, 3, 7, 8}},
					{"relay_children", 2, 1, []protocol.NodeID{2, 3, 8}},
					{"relay_two_levels", 2, 2, []protocol.NodeID{2, 3, 4, 8}},
					{"relay_all", 2, 0, []protocol.NodeID{2, 3, 4, 5, 6, 8}},
				} {
					t.Run(test.name, func(t *testing.T) {
						managementClient := topologyDiscoveryManagement(t, client, test.owner)
						value, err := managementClient.QueryTopology(ctx, test.depth)
						if err != nil {
							t.Fatal(err)
						}
						assertTopologyDiscovery(t, value, test.owner, test.depth, test.ids...)
						repeated, err := managementClient.QueryTopology(ctx, test.depth)
						if err != nil || !reflect.DeepEqual(repeated, value) {
							t.Fatalf("unchanged query changed snapshot: first=%+v repeated=%+v error=%v", value, repeated, err)
						}
					})
				}
				shallow, err := root.QueryTopology(ctx, 1)
				if err != nil {
					t.Fatal(err)
				}
				legacy, err := root.Topology(ctx)
				if err != nil {
					t.Fatal(err)
				}
				shallowPayload, err := protocol.EncodeJSONPayload(&shallow, protocol.DefaultMaxPayload)
				if err != nil {
					t.Fatal(err)
				}
				legacyPayload, err := protocol.EncodeJSONPayload(&legacy, protocol.DefaultMaxPayload)
				if err != nil {
					t.Fatal(err)
				}
				if len(shallow.Nodes) >= len(legacy.Nodes) || len(shallowPayload) >= len(legacyPayload) {
					t.Fatal("depth 1 did not reduce nodes and payload on the six-level fixture")
				}
				t.Logf("root depth=1: nodes=%d bytes=%d; legacy full: nodes=%d bytes=%d", len(shallow.Nodes), len(shallowPayload), len(legacy.Nodes), len(legacyPayload))
			})

			t.Run("ChildrenOnlyExactGrant", func(t *testing.T) {
				grant := fixture.grant(t, 7, 2, protocol.BuiltinManagementTopology, protocol.CapabilityChildren)
				value, err := relay.QueryTopology(ctx, 1)
				if err != nil {
					t.Fatal(err)
				}
				assertTopologyDiscovery(t, value, 2, 1, 2, 3, 8)
				for _, depth := range []int{0, 2} {
					if _, err := relay.QueryTopology(ctx, depth); !isSDKCode(err, protocol.CodeForbidden) {
						t.Fatalf("children grant allowed subtree depth %d: %v", depth, err)
					}
				}
				if _, err := root.QueryTopology(ctx, 1); !isSDKCode(err, protocol.CodeForbidden) {
					t.Fatalf("relay grant escaped its owner: %v", err)
				}
				if _, err := relay.Topology(ctx); !isSDKCode(err, protocol.CodeForbidden) {
					t.Fatalf("children grant allowed legacy read: %v", err)
				}
				resourceID := protocol.ResourceID{Owner: 2, Name: protocol.BuiltinManagementTopology}
				denied, err := client.Subscribe(ctx, resourceID, time.Minute, 4)
				if denied != nil {
					denied.Cancel()
				}
				if !isSDKCode(err, protocol.CodeForbidden) {
					t.Fatalf("children grant allowed legacy subscribe: %v", err)
				}
				if _, err := client.Catalog(ctx, 2); !isSDKCode(err, protocol.CodeForbidden) {
					t.Fatalf("children grant allowed catalog read: %v", err)
				}
				for _, depth := range []int{0, 2} {
					// Bypass typed request validation to exercise the remote handler
					// and the dispatcher's protocol error mapping over both transports.
					payload := []byte(fmt.Sprintf(`{"version":1,"depth":%d}`, depth))
					if _, err := client.Operate(ctx, resourceID, protocol.CapabilityChildren, protocol.SchemaManagementTopologyChildrenRequestV1, payload); !isSDKCode(err, protocol.CodeMalformed) {
						t.Fatalf("forged children depth %d: want malformed, got %v", depth, err)
					}
				}
				if err := fixture.states[1].Policy.Revoke(grant); err != nil {
					t.Fatal(err)
				}
				if _, err := relay.QueryTopology(ctx, 1); !isSDKCode(err, protocol.CodeForbidden) {
					t.Fatalf("cached children snapshot bypassed revoked grant: %v", err)
				}
			})

			t.Run("SubtreeDoesNotGrantChildren", func(t *testing.T) {
				fixture.grant(t, 7, 2, protocol.BuiltinManagementTopology, protocol.CapabilitySubtree)
				value, err := relay.QueryTopology(ctx, 0)
				if err != nil {
					t.Fatal(err)
				}
				assertTopologyDiscovery(t, value, 2, 0, 2, 3, 4, 5, 6, 8)
				if _, err := relay.QueryTopology(ctx, 1); !isSDKCode(err, protocol.CodeForbidden) {
					t.Fatalf("subtree grant implicitly granted children: %v", err)
				}
			})

			t.Run("CatalogIndependentOfDiscovery", func(t *testing.T) {
				fixture.grant(t, 7, 2, protocol.BuiltinResourceCatalog, protocol.CapabilityRead)
				catalog, err := client.Catalog(ctx, 2)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, descriptor := range catalog.Resources {
					if descriptor.ID == (protocol.ResourceID{Owner: 2, Name: protocol.BuiltinManagementTopology}) {
						found = true
					}
				}
				if !found {
					t.Fatal("independently readable catalog omitted topology resource")
				}
				for _, depth := range []int{0, 1} {
					if _, err := relay.QueryTopology(ctx, depth); !isSDKCode(err, protocol.CodeForbidden) {
						t.Fatalf("catalog read granted discovery depth %d: %v", depth, err)
					}
				}
				if _, err := client.Catalog(ctx, 1); !isSDKCode(err, protocol.CodeForbidden) {
					t.Fatalf("catalog grant escaped its owner: %v", err)
				}
			})

			t.Run("ParentControlWithEmptyOwnerPolicy", func(t *testing.T) {
				for _, owner := range []protocol.NodeID{2, 3} {
					grant := fixture.grant(t, 7, owner, protocol.BuiltinManagementTopology, protocol.CapabilityChildren)
					policy := fixture.states[owner].Policy
					before := snapshotTopologyDiscoveryPolicy(policy)
					if len(before.grants) != 0 || len(before.bindings) != 0 || policy.Authorize(ctx, grant) == nil {
						t.Fatalf("node %d must have an empty, denying owner policy", owner)
					}
					value, err := topologyDiscoveryManagement(t, client, owner).QueryTopology(ctx, 1)
					if err != nil {
						t.Fatalf("root-adjudicated query to owner %d was re-denied: %v", owner, err)
					}
					if owner == 2 {
						assertTopologyDiscovery(t, value, 2, 1, 2, 3, 8)
					} else {
						assertTopologyDiscovery(t, value, 3, 1, 3, 4)
					}
					if !reflect.DeepEqual(before, snapshotTopologyDiscoveryPolicy(policy)) {
						t.Fatalf("query changed owner %d policy", owner)
					}
				}
			})

			t.Run("ExistingGrantsRemainUnchanged", func(t *testing.T) {
				legacyClient := fixture.client(t, 8)
				legacyRoot := topologyDiscoveryManagement(t, legacyClient, 1)
				policy := fixture.states[1].Policy
				before := snapshotTopologyDiscoveryPolicy(policy)
				if len(before.grants) != 2 || len(before.bindings) != 0 {
					t.Fatalf("expected only the two pre-existing grants: %+v", before)
				}
				value, err := legacyRoot.Topology(ctx)
				if err != nil {
					t.Fatal(err)
				}
				assertTopologyDiscoveryLegacy(t, value)
				remote, err := legacyClient.Subscribe(ctx, protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology}, time.Minute, 4)
				if err != nil {
					t.Fatal(err)
				}
				defer remote.Cancel()
				event := receiveProductEvent(t, ctx, remote)
				if event.Kind != sdk.EventSnapshot || event.Schema != protocol.SchemaManagementTopologyV1 {
					t.Fatalf("legacy subscription lost its snapshot contract: %+v", event)
				}
				var subscribed protocol.ManagementTopologyV1
				if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &subscribed); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(subscribed, value) {
					t.Fatalf("legacy read and subscribe diverged: read=%+v subscribe=%+v", value, subscribed)
				}
				for _, depth := range []int{0, 1, 2} {
					if _, err := legacyRoot.QueryTopology(ctx, depth); !isSDKCode(err, protocol.CodeForbidden) {
						t.Fatalf("legacy grants implicitly added depth %d query permission: %v", depth, err)
					}
				}
				if !reflect.DeepEqual(before, snapshotTopologyDiscoveryPolicy(policy)) {
					t.Fatal("legacy or denied discovery operations changed existing policy")
				}
				reloaded, err := auth.LoadPolicyState(fixture.states[1].Store)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := reloaded.Close(); err != nil {
						t.Errorf("close reloaded policy: %v", err)
					}
				})
				if !reflect.DeepEqual(before, snapshotTopologyDiscoveryPolicy(reloaded)) {
					t.Fatal("persisted grants changed on reload")
				}
			})
		})
	}
}

func newTopologyDiscoveryFixture(t *testing.T, ctx context.Context, driver link.Driver, endpoint func(string) link.Endpoint) topologyDiscoveryFixture {
	t.Helper()
	fixture := topologyDiscoveryFixture{states: make(map[protocol.NodeID]*hostconfig.Runtime), nodes: make(map[protocol.NodeID]*node.Node)}
	trust := auth.NewTrustStore()
	for id := protocol.NodeID(1); id <= 8; id++ {
		state, err := hostconfig.Open(t.TempDir(), id)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := state.Policy.Close(); err != nil {
				t.Errorf("close node %d policy: %v", state.Identity.NodeID, err)
			}
		})
		fixture.states[id] = state
		if err := trust.Add(id, state.Identity.PublicKey); err != nil {
			t.Fatal(err)
		}
	}
	for id := protocol.NodeID(1); id <= 8; id++ {
		state := fixture.states[id]
		current, err := node.New(ctx, node.Config{Identity: state.Identity, Trust: trust, Policy: state.Policy})
		if err != nil {
			t.Fatal(err)
		}
		fixture.nodes[id] = current
		t.Cleanup(func() {
			if err := current.Close(); err != nil {
				t.Errorf("close node %d: %v", current.ID(), err)
			}
		})
	}
	addresses := make(map[protocol.NodeID]link.Endpoint)
	for id := protocol.NodeID(1); id <= 5; id++ {
		address, err := fixture.nodes[id].Listen(driver, endpoint(strconv.FormatUint(uint64(id), 10)))
		if err != nil {
			t.Fatal(err)
		}
		addresses[id] = address
	}
	for _, edge := range [][2]protocol.NodeID{{2, 1}, {3, 2}, {4, 3}, {5, 4}, {6, 5}, {7, 1}, {8, 2}} {
		if err := fixture.nodes[edge[0]].ConnectParent(ctx, driver, addresses[edge[1]], edge[1]); err != nil {
			t.Fatalf("connect %d to %d: %v", edge[0], edge[1], err)
		}
	}
	for _, relation := range [][2]protocol.NodeID{{1, 2}, {1, 3}, {1, 4}, {1, 5}, {1, 6}, {1, 7}, {1, 8}, {2, 3}, {2, 4}, {2, 5}, {2, 6}, {2, 8}, {3, 4}, {3, 5}, {3, 6}} {
		waitDownRoute(t, fixture.nodes[relation[0]].Tree(), relation[1])
	}
	// Install old-style exact grants before registering the new provider.
	for _, action := range []auth.Action{auth.ActionRead, auth.ActionSubscribe} {
		if err := fixture.states[1].Policy.Grant(auth.Request{Subject: 8, Action: action, Resource: protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementTopology}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, owner := range []protocol.NodeID{1, 2, 3} {
		state := fixture.states[owner]
		before := snapshotTopologyDiscoveryPolicy(state.Policy)
		if _, err := management.Register(management.Config{
			Node: fixture.nodes[owner], Admission: state.Admission, Trust: trust, Policy: state.Policy,
			Settings: state.Settings, RevokeNode: state.RevokeNode,
		}); err != nil {
			t.Fatalf("register provider %d: %v", owner, err)
		}
		if !reflect.DeepEqual(before, snapshotTopologyDiscoveryPolicy(state.Policy)) {
			t.Fatalf("provider registration changed owner %d policy", owner)
		}
	}
	return fixture
}

func (f topologyDiscoveryFixture) client(t *testing.T, subject protocol.NodeID) *sdk.Client {
	t.Helper()
	client, err := sdk.NewAttachedClient(f.nodes[subject])
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func (f topologyDiscoveryFixture) grant(t *testing.T, subject, owner protocol.NodeID, name string, capability protocol.CapabilityID) auth.Request {
	t.Helper()
	grant := auth.Request{Subject: subject, Capability: capability, Resource: protocol.ResourceID{Owner: owner, Name: name}}
	policy := f.states[1].Policy
	if err := policy.Grant(grant); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := policy.Revoke(grant); err != nil {
			t.Errorf("revoke fixture grant: %v", err)
		}
	})
	return grant
}

func topologyDiscoveryManagement(t *testing.T, client *sdk.Client, owner protocol.NodeID) *sdk.ManagementClient {
	t.Helper()
	value, err := client.Management(owner)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func assertTopologyDiscovery(t *testing.T, value protocol.ManagementTopologyQueryV1, owner protocol.NodeID, depth int, ids ...protocol.NodeID) {
	t.Helper()
	if err := value.Validate(); err != nil {
		t.Fatalf("invalid topology response: %v", err)
	}
	if value.RootNodeID != strconv.FormatUint(uint64(owner), 10) || value.Depth != depth || len(value.Nodes) != len(ids) {
		t.Fatalf("unexpected owner/depth/node count: %+v; want owner=%d depth=%d nodes=%v", value, owner, depth, ids)
	}
	parents := map[protocol.NodeID]protocol.NodeID{2: 1, 3: 2, 4: 3, 5: 4, 6: 5, 7: 1, 8: 2}
	byID := make(map[string]protocol.TopologyQueryNodeV1, len(value.Nodes))
	for _, current := range value.Nodes {
		byID[current.NodeID] = current
	}
	for _, id := range ids {
		current, ok := byID[strconv.FormatUint(uint64(id), 10)]
		if !ok {
			t.Fatalf("node %d missing from %+v", id, value)
		}
		parent := ""
		if id != owner {
			parent = strconv.FormatUint(uint64(parents[id]), 10)
		}
		if current.ParentID != parent || current.HasChildren != (id >= 1 && id <= 5) {
			t.Fatalf("wrong relation/has_children at owner=%d depth=%d: %+v, want parent=%q has_children=%t", owner, depth, current, parent, id >= 1 && id <= 5)
		}
		if id == owner && ((id == 1 && current.Role != "root") || (id != 1 && current.Role != "node")) {
			t.Fatalf("query boundary was confused with network root: %+v", current)
		}
	}
}

func assertTopologyDiscoveryLegacy(t *testing.T, value protocol.ManagementTopologyV1) {
	t.Helper()
	if err := value.Validate(); err != nil {
		t.Fatal(err)
	}
	parents := map[string]string{"1": "", "2": "1", "3": "2", "4": "3", "5": "4", "6": "5", "7": "1", "8": "2"}
	if len(value.Nodes) != len(parents) {
		t.Fatalf("legacy grants no longer expose the full snapshot: %+v", value)
	}
	for _, current := range value.Nodes {
		parent, ok := parents[current.NodeID]
		if !ok || current.ParentID != parent {
			t.Fatalf("legacy snapshot changed relation: %+v", current)
		}
	}
}

type topologyDiscoveryPolicySnapshot struct {
	generation  uint64
	grants      []auth.Request
	bindings    []protocol.PolicyBindingV1
	definitions []protocol.PolicyDefinitionV1
}

func snapshotTopologyDiscoveryPolicy(policy *auth.PolicyState) topologyDiscoveryPolicySnapshot {
	return topologyDiscoveryPolicySnapshot{policy.Generation(), policy.Grants(), policy.Bindings(), policy.Definitions()}
}
