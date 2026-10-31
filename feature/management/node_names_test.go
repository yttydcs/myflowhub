package management_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/feature/management"
	"github.com/yttydcs/myflowhub/protocol"
)

func TestSharedNodeNamesArePersistedAndOnlyProjectRealMembership(t *testing.T) {
	value := newFixture(t)
	if err := value.node.Tree().AttachChild(2, 1); err != nil {
		t.Fatal(err)
	}
	if err := value.node.Tree().AnnounceWithParent(2, 3, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	before, _ := topologyQuery(t, value, 0)
	epoch := value.node.Tree().Epoch()
	settings := value.state.Settings.Snapshot()
	names := map[string]string{"zone": "cloud", management.NodeDisplayNamePrefix + "1": "ignored local override", management.NodeDisplayNamePrefix + "2": "云服务器探针", management.NodeDisplayNamePrefix + "3": "Win11-Main", management.NodeDisplayNamePrefix + "999": "not a member"}
	update := protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: settings.Revision, DisplayName: "公网 Hub", Values: names}
	invoke(t, value.node, protocol.BuiltinManagementConfigUpdate, encode(t, &update))
	after, _ := topologyQuery(t, value, 0)
	if len(after.Nodes) != 3 || after.Nodes[0].DisplayName != "公网 Hub" || after.Nodes[1].DisplayName != "云服务器探针" || after.Nodes[2].DisplayName != "Win11-Main" {
		t.Fatalf("unexpected names or membership: %+v", after.Nodes)
	}
	if after.Revision <= before.Revision || value.node.Tree().Epoch() != epoch || after.Nodes[1].ParentID != "1" || after.Nodes[2].ParentID != "2" {
		t.Fatal("renaming changed authority edges or failed to invalidate the presentation snapshot")
	}
	shallow, _ := topologyQuery(t, value, 1)
	if len(shallow.Nodes) != 2 || shallow.Nodes[1].DisplayName != "云服务器探针" {
		t.Fatal("shallow discovery omitted the name")
	}
	var persisted protocol.ManagementConfigV1
	found, err := value.state.Store.Load("host.json", &persisted)
	if err != nil || !found || !reflect.DeepEqual(persisted.Values, names) || persisted.DisplayName != "公网 Hub" {
		t.Fatalf("name metadata was not persisted: %+v, %v", persisted, err)
	}
	// A name update must invalidate both cached depth results without an edge change.
	settings = value.state.Settings.Snapshot()
	names[management.NodeDisplayNamePrefix+"3"] = "工作站"
	update = protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: settings.Revision, DisplayName: settings.DisplayName, Values: names}
	invoke(t, value.node, protocol.BuiltinManagementConfigUpdate, encode(t, &update))
	changed, _ := topologyQuery(t, value, 0)
	if changed.Revision <= after.Revision || changed.Nodes[2].DisplayName != "工作站" {
		t.Fatal("descendant-only name update reused the old topology cache")
	}
	// Unrelated host attributes must not churn topology revisions.
	settings = value.state.Settings.Snapshot()
	names["zone"] = "other"
	update = protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: settings.Revision, DisplayName: settings.DisplayName, Values: names}
	invoke(t, value.node, protocol.BuiltinManagementConfigUpdate, encode(t, &update))
	unchanged, _ := topologyQuery(t, value, 0)
	if unchanged.Revision != changed.Revision {
		t.Fatal("unrelated settings invalidated topology")
	}
	// Explicit empty labels publish a nonempty fallback, clearing old client caches.
	settings = value.state.Settings.Snapshot()
	names[management.NodeDisplayNamePrefix+"3"] = ""
	update = protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: settings.Revision, DisplayName: settings.DisplayName, Values: names}
	invoke(t, value.node, protocol.BuiltinManagementConfigUpdate, encode(t, &update))
	cleared, _ := topologyQuery(t, value, 0)
	if cleared.Nodes[2].DisplayName != "Node 3" {
		t.Fatal("explicit reset did not publish the fallback")
	}
}

func TestNodeNameValidationRejectsBadMetadataBeforePersistence(t *testing.T) {
	for _, test := range []struct{ key, name string }{
		{management.NodeDisplayNamePrefix + "0", "bad"},
		{management.NodeDisplayNamePrefix + "01", "bad"},
		{management.NodeDisplayNamePrefix + "18446744073709551616", "bad"},
		{management.NodeDisplayNamePrefix + "2", strings.Repeat("字", 86)},
	} {
		t.Run(test.key, func(t *testing.T) {
			value := newFixture(t)
			current := value.state.Settings.Snapshot()
			update := protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: current.Revision, DisplayName: "unchanged", Values: map[string]string{test.key: test.name}}
			_, err := value.node.Invoke(context.Background(), protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementConfigUpdate}, encode(t, &update))
			if err == nil {
				t.Fatal("invalid node-name metadata was accepted")
			}
			if !reflect.DeepEqual(value.state.Settings.Snapshot(), current) {
				t.Fatal("failed validation changed durable settings")
			}
		})
	}
}
