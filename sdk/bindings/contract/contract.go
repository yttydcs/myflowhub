package contract

import (
	"errors"
	"sort"

	"github.com/yttydcs/myflowhub/protocol"
)

const APIVersion = "mfh.bindings.v2"

type Manifest struct {
	Version        int        `json:"version"`
	API            string     `json:"api"`
	Methods        []string   `json:"methods"`
	DesktopMethods []string   `json:"desktop_methods"`
	Resources      []Resource `json:"resources"`
}

type Resource struct {
	Name         string       `json:"name"`
	Type         string       `json:"type"`
	Capabilities []Capability `json:"capabilities"`
}

type Capability struct {
	Name         string `json:"name"`
	InputSchema  string `json:"input_schema,omitempty"`
	OutputSchema string `json:"output_schema,omitempty"`
	EventSchema  string `json:"event_schema,omitempty"`
}

func Canonical() (Manifest, error) {
	manifest := Manifest{
		Version: 2,
		API:     APIVersion,
		Methods: []string{
			"CancelSubscription", "CatalogJSON", "Close", "EnrollmentStatusJSON", "EnrollTCP", "IdentityJSON",
			"InvokeJSON", "OperateJSON", "SnapshotJSON", "StartEnrolledTCP", "StartRFCOMM", "StartTCP",
			"StatusJSON", "Subscribe", "SubscribeCapability", "TrustParent", "UploadFile", "WaitConnected",
		},
		DesktopMethods: []string{
			"CancelSubscription", "CatalogJSON", "Close", "EnrollmentStatusJSON", "EnrollTCP", "IdentityJSON",
			"InvokeJSON", "Open", "OpenEnrollment", "OperateJSON", "PollSubscription", "SelectUploadFile", "SnapshotJSON",
			"StartEnrolledTCP", "StartTCP", "StatusJSON", "Subscribe", "SubscribeCapability", "TrustParent",
			"UploadFile", "WaitConnected",
		},
		Resources: []Resource{
			variable(protocol.BuiltinResourceCatalog, protocol.SchemaResourceCatalogV2),
			variable(protocol.BuiltinManagementTopology, protocol.SchemaManagementTopologyV1),
			variable(protocol.BuiltinManagementHealth, protocol.SchemaManagementHealthV1),
			variable(protocol.BuiltinManagementConfig, protocol.SchemaManagementConfigV1),
			stream(protocol.BuiltinManagementAudit, protocol.SchemaManagementAuditV1),
			command(protocol.BuiltinManagementIssuePermit, protocol.SchemaAdmissionIssuePermitV1, protocol.SchemaEnrollmentPermitV1),
			command(protocol.BuiltinManagementRevokePermit, protocol.SchemaManagementRevokePermitV1, protocol.SchemaManagementResultV1),
			variable(protocol.BuiltinAdmissionStatus, protocol.SchemaAdmissionStatusV1),
			command(protocol.BuiltinAdmissionListPermits, protocol.SchemaAdmissionListV1, protocol.SchemaAdmissionPermitListV1),
			command(protocol.BuiltinAdmissionRevokePermit, protocol.SchemaAdmissionRevokePermitV1, protocol.SchemaManagementResultV1),
			command(protocol.BuiltinAdmissionListRequests, protocol.SchemaAdmissionListV1, protocol.SchemaAdmissionRequestListV1),
			command(protocol.BuiltinAdmissionApprove, protocol.SchemaAdmissionDecisionV1, protocol.SchemaEnrollmentGrantV1),
			command(protocol.BuiltinAdmissionReject, protocol.SchemaAdmissionDecisionV1, protocol.SchemaManagementResultV1),
			command(protocol.BuiltinAdmissionListEnrollments, protocol.SchemaAdmissionListV1, protocol.SchemaAdmissionEnrollmentListV1),
			command(protocol.BuiltinAdmissionRevokeEnrollment, protocol.SchemaAdmissionRevokeEnrollmentV1, protocol.SchemaManagementResultV1),
			command(protocol.BuiltinManagementRevokeNode, protocol.SchemaManagementRevokeV1, protocol.SchemaManagementResultV1),
			command(protocol.BuiltinManagementConfigUpdate, protocol.SchemaManagementConfigUpdateV1, protocol.SchemaManagementConfigV1),
			command(protocol.BuiltinManagementPolicyGrant, protocol.SchemaManagementPolicyRuleV1, protocol.SchemaManagementResultV1),
			command(protocol.BuiltinManagementPolicyRevoke, protocol.SchemaManagementPolicyRuleV1, protocol.SchemaManagementResultV1),
			collection(protocol.BuiltinPolicyDefinitions,
				capability(protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, ""),
				capability(protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, protocol.SchemaPolicyDefinitionV1, ""),
				capability(protocol.CapabilityCreate, protocol.SchemaPolicyDefinitionPutV1, protocol.SchemaPolicyDefinitionV1, ""),
				capability(protocol.CapabilityUpdate, protocol.SchemaPolicyDefinitionPutV1, protocol.SchemaPolicyDefinitionV1, ""),
				capability(protocol.CapabilityDelete, protocol.SchemaPolicyDefinitionDeleteV1, protocol.SchemaManagementResultV1, ""),
			),
			collection(protocol.BuiltinPolicyBindings,
				capability(protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, ""),
				capability(protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, protocol.SchemaPolicyBindingV1, ""),
				capability(protocol.CapabilityCreate, protocol.SchemaPolicyBindingCreateV1, protocol.SchemaPolicyBindingV1, ""),
				capability(protocol.CapabilityRevoke, protocol.SchemaPolicyBindingRevokeV1, protocol.SchemaManagementResultV1, ""),
				capability(protocol.CapabilityEvaluate, protocol.SchemaPolicyEvaluateRequestV1, protocol.SchemaPolicyEvaluationV1, ""),
			),
			collection(protocol.BuiltinPolicyGrants,
				capability(protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, protocol.SchemaCollectionPageV1, ""),
				capability(protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, protocol.SchemaPolicyGrantV1, ""),
				capability(protocol.CapabilityCreate, protocol.SchemaPolicyGrantV1, protocol.SchemaPolicyGrantV1, ""),
				capability(protocol.CapabilityRevoke, protocol.SchemaPolicyGrantV1, protocol.SchemaManagementResultV1, ""),
			),
			stream(protocol.BuiltinNotificationEvents, protocol.SchemaNotificationEventV1),
			command(protocol.BuiltinNotificationPublish, protocol.SchemaNotificationPublishV1, protocol.SchemaNotificationEventV1),
			variable(protocol.BuiltinFileTransfers, protocol.SchemaFileTransfersV1),
			stream(protocol.BuiltinFileProgress, protocol.SchemaFileProgressV1),
			session(protocol.BuiltinFileUpload, protocol.SchemaFileOfferV1, protocol.SchemaFileProgressV1),
		},
	}
	sort.Strings(manifest.Methods)
	sort.Strings(manifest.DesktopMethods)
	sort.Slice(manifest.Resources, func(i, j int) bool { return manifest.Resources[i].Name < manifest.Resources[j].Name })
	for index := range manifest.Resources {
		current := &manifest.Resources[index]
		sort.Slice(current.Capabilities, func(i, j int) bool { return current.Capabilities[i].Name < current.Capabilities[j].Name })
		if current.Name == "" || current.Type == "" || len(current.Capabilities) == 0 {
			return Manifest{}, errors.New("binding contract contains an incomplete resource descriptor")
		}
		if index > 0 && manifest.Resources[index-1].Name == current.Name {
			return Manifest{}, errors.New("binding contract contains a duplicate resource")
		}
	}
	return manifest, nil
}

func variable(name, schema string) Resource {
	return Resource{Name: name, Type: string(protocol.ResourceTypeVariable), Capabilities: []Capability{
		{Name: string(protocol.CapabilityRead), OutputSchema: schema},
		{Name: string(protocol.CapabilitySubscribe), EventSchema: schema},
	}}
}

func stream(name, schema string) Resource {
	return Resource{Name: name, Type: string(protocol.ResourceTypeStream), Capabilities: []Capability{{Name: string(protocol.CapabilitySubscribe), EventSchema: schema}}}
}

func command(name, request, response string) Resource {
	return Resource{Name: name, Type: string(protocol.ResourceTypeCommand), Capabilities: []Capability{{
		Name: string(protocol.CapabilityInvoke), InputSchema: request, OutputSchema: response,
	}}}
}

func session(name, request, response string) Resource {
	return Resource{Name: name, Type: string(protocol.ResourceTypeFile), Capabilities: []Capability{{
		Name: string(protocol.CapabilityOpen), InputSchema: request, OutputSchema: response,
	}}}
}

func collection(name string, capabilities ...Capability) Resource {
	return Resource{Name: name, Type: string(protocol.ResourceTypeCollection), Capabilities: capabilities}
}

func capability(name protocol.CapabilityID, input, output, event string) Capability {
	return Capability{Name: string(name), InputSchema: input, OutputSchema: output, EventSchema: event}
}
