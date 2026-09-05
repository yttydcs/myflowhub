package management_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
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
	authority  *auth.EnrollmentAuthority
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
	authority, err := auth.LoadEnrollmentAuthority(state.Identity, state.Store, auth.EnrollmentAuthorityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := node.New(context.Background(), node.Config{Identity: state.Identity, Trust: state.Trust, Admission: state.Admission, Policy: management.AuditPolicy(state.Policy, audit)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	if err := state.Policy.BindScopeResolver(testPolicyScopeResolver{runtime: runtime}); err != nil {
		t.Fatal(err)
	}
	if _, err := state.Policy.CreateBinding(protocol.PolicyBindingCreateV1{
		Version: 1, BindingID: strings.Repeat("1", 32), Subject: "1", DefinitionID: protocol.BuiltinPolicySuperadmin,
		Scope: protocol.PolicyOwnerScopeV1{Kind: protocol.PolicyScopeAuthorityDomain, NodeID: "1"},
	}, 1); err != nil {
		t.Fatal(err)
	}
	controller, err := management.Register(management.Config{Node: runtime, Admission: state.Admission, EnrollmentAuthority: authority, Trust: state.Trust, Policy: state.Policy, Settings: state.Settings, RevokeNode: state.RevokeNode, Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	return fixture{state: state, authority: authority, node: runtime, controller: controller, audit: audit}
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
		protocol.BuiltinPolicyDefinitions,
		protocol.BuiltinPolicyBindings,
		protocol.BuiltinPolicyGrants,
		protocol.BuiltinManagementRevokeNode,
		protocol.BuiltinManagementRevokePermit,
		protocol.BuiltinManagementTopology,
		protocol.BuiltinAdmissionStatus,
		protocol.BuiltinAdmissionListPermits,
		protocol.BuiltinAdmissionRevokePermit,
		protocol.BuiltinAdmissionListRequests,
		protocol.BuiltinAdmissionApprove,
		protocol.BuiltinAdmissionReject,
		protocol.BuiltinAdmissionListEnrollments,
		protocol.BuiltinAdmissionRevokeEnrollment,
		protocol.BuiltinAdmissionSubmitEnrollment,
		protocol.BuiltinAdmissionApplyRevocation,
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

func TestCentralAdmissionManagementListsApprovesAndRevokes(t *testing.T) {
	value := newFixture(t)
	device, _ := auth.GenerateDeviceIdentity()
	issue := protocol.AdmissionIssuePermitV1{
		Version:                    protocol.SchemaVersionV1,
		RequestID:                  "50000000000000000000000000000001",
		DevicePublicKeyFingerprint: auth.DevicePublicKeyFingerprint(device.PublicKey),
		TargetNodeID:               "1", AdmissionProfile: "desktop", TTLMS: 60_000,
	}
	permitData := invoke(t, value.node, protocol.BuiltinAdmissionIssuePermit, encode(t, &issue))
	var permit protocol.EnrollmentPermitV1
	decode(t, permitData, &permit)
	permitListData := invoke(t, value.node, protocol.BuiltinAdmissionListPermits, encode(t, &protocol.AdmissionListV1{Version: 1, Limit: 10}))
	var permitList protocol.AdmissionPermitListV1
	decode(t, permitListData, &permitList)
	if len(permitList.Items) != 1 || permitList.Items[0].PermitID != permit.PermitID {
		t.Fatalf("unexpected Permit list: %#v", permitList)
	}
	revokePermit := protocol.AdmissionRevokePermitV1{
		Version: 1, RequestID: "50000000000000000000000000000002",
		PermitID: permit.PermitID, Reason: "replaced",
	}
	invoke(t, value.node, protocol.BuiltinAdmissionRevokePermit, encode(t, &revokePermit))

	pendingID := "50000000000000000000000000000003"
	if outcome, err := value.authority.Submit(auth.EnrollmentSubmission{
		RequestID: pendingID, DevicePublicKey: device.PublicKey,
		ParentNodeID: value.node.ID(), ParentPublicKey: value.state.Identity.PublicKey,
		TranscriptDigest: [32]byte{1},
	}); err != nil || outcome.Status != "pending" {
		t.Fatalf("create pending request: %#v, %v", outcome, err)
	}
	requestListData := invoke(t, value.node, protocol.BuiltinAdmissionListRequests, encode(t, &protocol.AdmissionListV1{Version: 1, Limit: 10, Status: "pending"}))
	var requestList protocol.AdmissionRequestListV1
	decode(t, requestListData, &requestList)
	if len(requestList.Items) != 1 || requestList.Items[0].RequestID != pendingID {
		t.Fatalf("unexpected Pending list: %#v", requestList)
	}
	approve := protocol.AdmissionDecisionV1{
		Version: 1, RequestID: "50000000000000000000000000000004",
		EnrollmentRequestID: pendingID, AdmissionProfile: "approved-desktop",
	}
	grantData := invoke(t, value.node, protocol.BuiltinAdmissionApprove, encode(t, &approve))
	var grant protocol.EnrollmentGrantV1
	decode(t, grantData, &grant)
	nodeIDValue, _ := strconv.ParseUint(grant.NodeID, 10, 64)
	if err := value.state.Trust.Add(protocol.NodeID(nodeIDValue), device.PublicKey); err != nil {
		t.Fatal(err)
	}
	enrollmentListData := invoke(t, value.node, protocol.BuiltinAdmissionListEnrollments, encode(t, &protocol.AdmissionListV1{Version: 1, Limit: 10}))
	var enrollmentList protocol.AdmissionEnrollmentListV1
	decode(t, enrollmentListData, &enrollmentList)
	if len(enrollmentList.Items) != 1 || enrollmentList.Items[0].NodeID != grant.NodeID {
		t.Fatalf("unexpected Enrollment list: %#v", enrollmentList)
	}
	revoke := protocol.AdmissionRevokeEnrollmentV1{
		Version: 1, RequestID: "50000000000000000000000000000005",
		EnrollmentID: grant.EnrollmentID, Reason: "retired",
	}
	invoke(t, value.node, protocol.BuiltinAdmissionRevokeEnrollment, encode(t, &revoke))
	if _, trusted := value.state.Trust.PublicKey(protocol.NodeID(nodeIDValue)); trusted {
		t.Fatal("revoked Enrollment remained in parent trust")
	}
	var status protocol.AdmissionStatusV1
	decodeVariable(t, value.node, protocol.BuiltinAdmissionStatus, &status)
	if status.Enrollments != 0 || status.Revocations != 1 || status.PendingRequests != 0 {
		t.Fatalf("unexpected admission status: %#v", status)
	}
}

func TestCentralAdmissionListsUseStableCursors(t *testing.T) {
	value := newFixture(t)
	for index := 1; index <= 3; index++ {
		device, _ := auth.GenerateDeviceIdentity()
		requestID := "5100000000000000000000000000000" + strconv.Itoa(index)
		issue := protocol.AdmissionIssuePermitV1{
			Version: 1, RequestID: requestID,
			DevicePublicKeyFingerprint: auth.DevicePublicKeyFingerprint(device.PublicKey),
			TargetNodeID:               "1", AdmissionProfile: "desktop", TTLMS: 60_000,
		}
		invoke(t, value.node, protocol.BuiltinAdmissionIssuePermit, encode(t, &issue))
	}
	seen := make(map[string]struct{})
	cursor := ""
	for page := 0; page < 4; page++ {
		data := invoke(t, value.node, protocol.BuiltinAdmissionListPermits, encode(t, &protocol.AdmissionListV1{Version: 1, Cursor: cursor, Limit: 1}))
		var result protocol.AdmissionPermitListV1
		decode(t, data, &result)
		if len(result.Items) != 1 {
			t.Fatalf("page %d returned %d items", page, len(result.Items))
		}
		if _, duplicate := seen[result.Items[0].PermitID]; duplicate {
			t.Fatalf("cursor repeated Permit %s", result.Items[0].PermitID)
		}
		seen[result.Items[0].PermitID] = struct{}{}
		cursor = result.NextCursor
		if cursor == "" {
			break
		}
	}
	if len(seen) != 3 || cursor != "" {
		t.Fatalf("pagination returned %d unique Permits and final cursor %q", len(seen), cursor)
	}
}

func TestCentralAdmissionMutationAuditNamesActorTargetAndStatusWithoutSecretMaterial(t *testing.T) {
	value := newFixture(t)
	auditResource, ok := value.node.Registry().Resolve(protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinManagementAudit})
	if !ok {
		t.Fatal("audit stream missing")
	}
	events := make(chan resource.StreamEvent, 1)
	cancel, err := auditResource.(*resource.Stream).Watch(func(event resource.StreamEvent) { events <- event })
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	device, _ := auth.GenerateDeviceIdentity()
	fingerprint := auth.DevicePublicKeyFingerprint(device.PublicKey)
	issue := protocol.AdmissionIssuePermitV1{
		Version: 1, RequestID: "52000000000000000000000000000001",
		DevicePublicKeyFingerprint: fingerprint, TargetNodeID: "1", AdmissionProfile: "desktop", TTLMS: 60_000,
	}
	permitData, err := value.node.Invoke(context.Background(), protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinAdmissionIssuePermit}, encode(t, &issue))
	if err != nil {
		t.Fatal(err)
	}
	var permit protocol.EnrollmentPermitV1
	decode(t, permitData, &permit)

	select {
	case event := <-events:
		var audit protocol.ManagementAuditV1
		decode(t, event.Value, &audit)
		if audit.Subject != "1" || audit.Action != "management.admission.issue" || audit.ResourceName != protocol.BuiltinAdmissionIssuePermit || audit.Target != permit.PermitID || audit.Status != "active" {
			t.Fatalf("unexpected admission audit event: %#v", audit)
		}
		if strings.Contains(string(event.Value), fingerprint) || strings.Contains(string(event.Value), string(permitData)) {
			t.Fatalf("admission audit leaked Permit input or body: %s", event.Value)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for admission audit event")
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

func TestPolicyCollectionsCreateListGetAndEvaluate(t *testing.T) {
	value := newFixture(t)
	definitionInput := protocol.PolicyDefinitionPutV1{
		Version: 1, ID: "metrics-reader", Label: "Metrics reader",
		Rules: []protocol.PolicyRuleV1{{
			Resource:   protocol.PolicyResourceSelectorV1{Kind: protocol.PolicySelectorPrefix, Value: "metrics"},
			Capability: protocol.PolicyCapabilitySelectorV1{Kind: protocol.PolicySelectorExact, Values: []protocol.CapabilityID{protocol.CapabilityRead}},
		}},
	}
	definitionResource, _ := value.node.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinPolicyDefinitions})
	created, err := definitionResource.Operate(context.Background(), resource.OperationRequest{
		Subject: 1, Capability: protocol.CapabilityCreate, Payload: encode(t, &definitionInput),
	})
	if err != nil {
		t.Fatal(err)
	}
	var definition protocol.PolicyDefinitionV1
	decode(t, created.Payload, &definition)
	if definition.ID != definitionInput.ID || definition.Revision != 1 {
		t.Fatalf("unexpected created definition: %+v", definition)
	}
	listed, err := definitionResource.Operate(context.Background(), resource.OperationRequest{
		Subject: 1, Capability: protocol.CapabilityList,
		Payload: encode(t, &protocol.CollectionListRequestV1{Version: 1, Limit: 10}),
	})
	if err != nil {
		t.Fatal(err)
	}
	var page protocol.CollectionPageV1
	decode(t, listed.Payload, &page)
	if len(page.Members) != 2 || page.Members[0].Key != "metrics-reader" || page.Members[1].Key != protocol.BuiltinPolicySuperadmin {
		t.Fatalf("unexpected policy definitions page: %+v", page.Members)
	}

	bindingResource, _ := value.node.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinPolicyBindings})
	evaluated, err := bindingResource.Operate(context.Background(), resource.OperationRequest{
		Subject: 1, Capability: protocol.CapabilityEvaluate,
		Payload: encode(t, &protocol.PolicyEvaluateRequestV1{Version: 1, Subject: "1", Capability: "read", ResourceNode: "1", ResourceName: "system/health"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	var decision protocol.PolicyEvaluationV1
	decode(t, evaluated.Payload, &decision)
	if !decision.Allowed || decision.DefinitionID != protocol.BuiltinPolicySuperadmin {
		t.Fatalf("unexpected effective decision: %+v", decision)
	}
}

func TestPolicyMutationGuardRejectsExactCapabilityWithoutSuperadminBinding(t *testing.T) {
	value := newFixture(t)
	resourceID := protocol.ResourceID{Owner: 1, Name: protocol.BuiltinPolicyDefinitions}
	if err := value.state.Policy.Grant(auth.Request{Subject: 2, Capability: protocol.CapabilityCreate, Resource: resourceID}); err != nil {
		t.Fatal(err)
	}
	policyResource, _ := value.node.Registry().Resolve(resourceID)
	input := protocol.PolicyDefinitionPutV1{
		Version: 1, ID: "escalation", Label: "Escalation",
		Rules: []protocol.PolicyRuleV1{{Resource: protocol.PolicyResourceSelectorV1{Kind: protocol.PolicySelectorAll}, Capability: protocol.PolicyCapabilitySelectorV1{Kind: protocol.PolicySelectorAll}}},
	}
	if _, err := policyResource.Operate(context.Background(), resource.OperationRequest{Subject: 2, Capability: protocol.CapabilityCreate, Payload: encode(t, &input)}); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("exact management capability bypassed superadmin guard: %v", err)
	}
	if _, ok := value.state.Policy.Definition(input.ID); ok {
		t.Fatal("rejected escalation still created a definition")
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

func TestCentralAdmissionResourcesRequireExplicitRemotePermission(t *testing.T) {
	value := newFixture(t)
	network := memory.NewNetwork()
	defer network.Close()
	if _, err := value.node.Listen(network, "admission-authority"); err != nil {
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
	if err := child.ConnectParent(context.Background(), network, "admission-authority", value.node.ID()); err != nil {
		t.Fatal(err)
	}
	resourceID := protocol.ResourceID{Owner: value.node.ID(), Name: protocol.BuiltinAdmissionListPermits}
	input := encode(t, &protocol.AdmissionListV1{Version: 1, Limit: 10})
	if _, err := child.Invoke(context.Background(), resourceID, input); err == nil {
		t.Fatal("remote actor read centralized Permit state without permission")
	}
	if err := value.state.Policy.Grant(auth.Request{Subject: child.ID(), Action: auth.ActionInvoke, Resource: resourceID}); err != nil {
		t.Fatal(err)
	}
	output, err := child.Invoke(context.Background(), resourceID, input)
	if err != nil {
		t.Fatalf("authorized remote admission read failed: %v", err)
	}
	var permits protocol.AdmissionPermitListV1
	decode(t, output, &permits)
}

func invoke(t *testing.T, runtime *node.Node, name string, input []byte) []byte {
	t.Helper()
	value, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: name})
	if !ok {
		t.Fatalf("command %s missing", name)
	}
	output, err := value.Operate(context.Background(), resource.OperationRequest{Subject: runtime.ID(), Capability: protocol.CapabilityInvoke, Payload: input})
	if err != nil {
		t.Fatal(err)
	}
	return output.Payload
}

type testPolicyScopeResolver struct{ runtime *node.Node }

func (r testPolicyScopeResolver) MatchOwnerScope(scope protocol.PolicyOwnerScopeV1, owner protocol.NodeID) (bool, uint64, error) {
	anchor, err := strconv.ParseUint(scope.NodeID, 10, 64)
	if err != nil {
		return false, 0, err
	}
	if scope.Kind == protocol.PolicyScopeOwner && protocol.NodeID(anchor) != owner {
		return false, 0, nil
	}
	if scope.Kind == protocol.PolicyScopeAuthorityDomain && protocol.NodeID(anchor) != r.runtime.ID() {
		return false, 0, nil
	}
	return r.runtime.Tree().ScopeContains(protocol.NodeID(anchor), owner)
}

func decodeVariable(t *testing.T, runtime *node.Node, name string, target protocol.ValidatedPayload) {
	t.Helper()
	value, ok := runtime.Registry().Resolve(protocol.ResourceID{Owner: runtime.ID(), Name: name})
	if !ok {
		t.Fatalf("variable %s missing", name)
	}
	result, err := value.Operate(context.Background(), resource.OperationRequest{Capability: protocol.CapabilityRead})
	if err != nil {
		t.Fatal(err)
	}
	decode(t, result.Payload, target)
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
