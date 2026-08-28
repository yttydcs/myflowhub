package management_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/feature/management"
	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/transport/memory"
)

type fixture struct {
	state      *hostconfig.Runtime
	node       *node.Node
	controller *management.Controller
	audit      *management.AuditLog
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	state, err := hostconfig.Open(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	audit := management.NewAuditLog(nil, 8)
	runtime, err := node.New(context.Background(), node.Config{Identity: state.Identity, Trust: state.Trust, Admission: state.Admission, Policy: management.AuditPolicy(state.Policy, audit)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	controller, err := management.Register(management.Config{Node: runtime, Admission: state.Admission, Trust: state.Trust, Policy: state.Policy, Settings: state.Settings, RevokeNode: state.RevokeNode, Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	return fixture{state: state, node: runtime, controller: controller, audit: audit}
}

func TestManagementCatalogAndTopologyUseCanonicalTree(t *testing.T) {
	value := newFixture(t)
	catalogSnapshot := value.node.Registry().Catalog().Snapshot()
	var catalog protocol.ResourceCatalogV2
	decode(t, catalogSnapshot.Value, &catalog)
	for _, name := range []string{
		protocol.BuiltinManagementIssuePermit,
		protocol.BuiltinManagementAudit,
		protocol.BuiltinManagementConfig,
		protocol.BuiltinManagementConfigUpdate,
		protocol.BuiltinManagementHealth,
		protocol.BuiltinManagementPolicyGrant,
		protocol.BuiltinManagementPolicyRevoke,
		protocol.BuiltinManagementRevokeNode,
		protocol.BuiltinManagementRevokePermit,
		protocol.BuiltinManagementTopology,
	} {
		if !catalogHas(catalog, name) {
			t.Fatalf("management resource %q missing from catalog", name)
		}
	}
	if err := value.node.Tree().AttachChild(2, 3); err != nil {
		t.Fatal(err)
	}
	if err := value.node.Tree().AnnounceWithParent(2, 3, 2, 3); err != nil {
		t.Fatal(err)
	}
	if err := value.controller.Refresh(); err != nil {
		t.Fatal(err)
	}
	var topology protocol.ManagementTopologyV1
	decodeVariable(t, value.node, protocol.BuiltinManagementTopology, &topology)
	if len(topology.Nodes) != 3 || topology.Nodes[1].NodeID != "2" || topology.Nodes[1].ParentID != "1" || topology.Nodes[2].NodeID != "3" || topology.Nodes[2].ParentID != "2" {
		t.Fatalf("topology did not preserve immediate parents: %#v", topology.Nodes)
	}
}

func TestManagementPermitConfigAndRevocationCommands(t *testing.T) {
	value := newFixture(t)
	child, _ := auth.GenerateIdentity(2)
	issue := protocol.ManagementIssuePermitV1{Version: 1, ChildNodeID: "2", ChildPublicKey: base64.RawStdEncoding.EncodeToString(child.PublicKey), Role: "device", TTLMS: 60_000}
	permitData := invoke(t, value.node, protocol.BuiltinManagementIssuePermit, encode(t, &issue))
	var permit protocol.ProvisioningPermitV1
	decode(t, permitData, &permit)
	if permit.ChildNodeID != "2" || len(permit.Signature) == 0 {
		t.Fatalf("invalid issued permit: %#v", permit)
	}
	revokePermit := protocol.ManagementRevokePermitV1{Version: 1, PermitID: permit.PermitID}
	invoke(t, value.node, protocol.BuiltinManagementRevokePermit, encode(t, &revokePermit))
	if err := value.state.Admission.Consume(child.NodeID, child.PublicKey, permit); !errors.Is(err, auth.ErrPermitRevoked) {
		t.Fatalf("revoked permit remained usable: %v", err)
	}

	current := value.state.Settings.Snapshot()
	update := protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: current.Revision, DisplayName: "root", Values: map[string]string{"zone": "lab"}}
	updatedData := invoke(t, value.node, protocol.BuiltinManagementConfigUpdate, encode(t, &update))
	var updated protocol.ManagementConfigV1
	decode(t, updatedData, &updated)
	if updated.Revision != current.Revision+1 || updated.DisplayName != "root" || updated.Values["zone"] != "lab" {
		t.Fatalf("unexpected config update: %#v", updated)
	}

	if err := value.state.Trust.Add(child.NodeID, child.PublicKey); err != nil {
		t.Fatal(err)
	}
	grant := auth.Request{Subject: child.NodeID, Action: auth.ActionSubscribe, Resource: protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinManagementHealth}}
	if err := value.state.Policy.Grant(grant); err != nil {
		t.Fatal(err)
	}
	revokeNode := protocol.ManagementRevokeV1{Version: 1, NodeID: "2", Reason: "retired"}
	invoke(t, value.node, protocol.BuiltinManagementRevokeNode, encode(t, &revokeNode))
	if _, trusted := value.state.Trust.PublicKey(child.NodeID); trusted {
		t.Fatal("revoked node remained trusted")
	}
	if err := value.state.Policy.Authorize(context.Background(), grant); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("revoked node retained policy grant: %v", err)
	}
}

func TestManagementPolicyGrantAndRevokeCommands(t *testing.T) {
	value := newFixture(t)
	rule := protocol.ManagementPolicyRuleV1{
		Version: 1, Subject: "2", Action: "subscribe", ResourceNode: "3", ResourceName: "metrics/cpu_percent",
	}
	request := auth.Request{Subject: 2, Action: auth.ActionSubscribe, Resource: protocol.ResourceID{Owner: 3, Name: "metrics/cpu_percent"}}
	invoke(t, value.node, protocol.BuiltinManagementPolicyGrant, encode(t, &rule))
	if err := value.state.Policy.Authorize(context.Background(), request); err != nil {
		t.Fatalf("policy grant command did not persist: %v", err)
	}
	invoke(t, value.node, protocol.BuiltinManagementPolicyRevoke, encode(t, &rule))
	if err := value.state.Policy.Authorize(context.Background(), request); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("policy revoke command did not persist: %v", err)
	}
}

func TestAuditPolicyRecordsDenyAndAllowWithoutPayload(t *testing.T) {
	value := newFixture(t)
	auditResource, ok := value.node.Registry().Resolve(protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinManagementAudit})
	if !ok {
		t.Fatal("audit stream missing")
	}
	events := make(chan resource.StreamEvent, 2)
	cancel, err := auditResource.(*resource.Stream).Watch(func(event resource.StreamEvent) { events <- event })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	request := auth.Request{Subject: 2, Action: auth.ActionInvoke, Resource: protocol.ResourceID{Owner: 1, Name: protocol.BuiltinManagementConfigUpdate}}
	policy := management.AuditPolicy(value.state.Policy, value.audit)
	if err := policy.Authorize(context.Background(), request); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("request was not denied: %v", err)
	}
	if err := value.state.Policy.Grant(request); err != nil {
		t.Fatal(err)
	}
	if err := policy.Authorize(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	decisions := []string{"deny", "allow"}
	for _, want := range decisions {
		select {
		case event := <-events:
			var audit protocol.ManagementAuditV1
			decode(t, event.Value, &audit)
			if audit.Decision != want || audit.Subject != "2" || audit.ResourceName != protocol.BuiltinManagementConfigUpdate {
				t.Fatalf("unexpected audit event: %#v", audit)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s audit event", want)
		}
	}
}

func TestRemotePolicyDenyAndAllowAreAudited(t *testing.T) {
	value := newFixture(t)
	network := memory.NewNetwork()
	defer network.Close()
	if _, err := value.node.Listen(network, "root"); err != nil {
		t.Fatal(err)
	}
	childIdentity, _ := auth.GenerateIdentity(2)
	if err := value.state.Trust.Add(childIdentity.NodeID, childIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	childTrust := auth.NewTrustStore()
	if err := childTrust.Add(value.node.ID(), value.state.Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	child, err := node.New(context.Background(), node.Config{Identity: childIdentity, Trust: childTrust})
	if err != nil {
		t.Fatal(err)
	}
	defer child.Close()
	if err := child.ConnectParent(context.Background(), network, "root", value.node.ID()); err != nil {
		t.Fatal(err)
	}
	auditResource, _ := value.node.Registry().Resolve(protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinManagementAudit})
	events := make(chan resource.StreamEvent, 2)
	cancel, _ := auditResource.(*resource.Stream).Watch(func(event resource.StreamEvent) { events <- event })
	defer cancel()
	resourceID := protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinManagementConfigUpdate}
	input := encode(t, &protocol.ManagementConfigUpdateV1{Version: 1, ExpectedRevision: 1, DisplayName: "remote"})
	if _, err := child.Invoke(context.Background(), resourceID, input); err == nil {
		t.Fatal("remote management command bypassed default-deny policy")
	}
	if err := value.state.Policy.Grant(auth.Request{Subject: child.ID(), Action: auth.ActionInvoke, Resource: resourceID}); err != nil {
		t.Fatal(err)
	}
	if _, err := child.Invoke(context.Background(), resourceID, input); err != nil {
		t.Fatalf("allowed remote management command failed: %v", err)
	}
	for _, want := range []string{"deny", "allow"} {
		select {
		case event := <-events:
			var audit protocol.ManagementAuditV1
			decode(t, event.Value, &audit)
			if audit.Decision != want || audit.Subject != "2" {
				t.Fatalf("unexpected remote audit event: %#v", audit)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for remote %s audit event", want)
		}
	}
}

func invoke(t *testing.T, runtime *node.Node, name string, input []byte) []byte {
	t.Helper()
	value, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: name})
	if !ok {
		t.Fatalf("command %s missing", name)
	}
	output, err := value.(*resource.Command).Invoke(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func decodeVariable(t *testing.T, runtime *node.Node, name string, target protocol.ValidatedPayload) {
	t.Helper()
	value, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: name})
	if !ok {
		t.Fatalf("variable %s missing", name)
	}
	decode(t, value.(*resource.Variable).Snapshot().Value, target)
}

func encode(t *testing.T, value protocol.ValidatedPayload) []byte {
	t.Helper()
	payload, err := protocol.EncodeJSONPayload(value, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func decode(t *testing.T, data []byte, target protocol.ValidatedPayload) {
	t.Helper()
	if err := protocol.DecodeJSONPayload(data, protocol.DefaultMaxPayload, target); err != nil {
		t.Fatal(err)
	}
}

func catalogHas(catalog protocol.ResourceCatalogV2, name string) bool {
	for _, descriptor := range catalog.Resources {
		if descriptor.ID.Name == name {
			return true
		}
	}
	return false
}
