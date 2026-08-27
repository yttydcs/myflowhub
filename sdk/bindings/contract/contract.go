package contract

import (
	"errors"
	"sort"

	"github.com/yttydcs/myflowhub/protocol"
)

const APIVersion = "mfh.bindings.v1"

type Manifest struct {
	Version        int        `json:"version"`
	API            string     `json:"api"`
	Methods        []string   `json:"methods"`
	DesktopMethods []string   `json:"desktop_methods"`
	Resources      []Resource `json:"resources"`
}

type Resource struct {
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	SnapshotSchema string `json:"snapshot_schema,omitempty"`
	EventSchema    string `json:"event_schema,omitempty"`
	RequestSchema  string `json:"request_schema,omitempty"`
	ResponseSchema string `json:"response_schema,omitempty"`
}

func Canonical() (Manifest, error) {
	manifest := Manifest{
		Version: 1,
		API:     APIVersion,
		Methods: []string{
			"CancelSubscription", "CatalogJSON", "Close", "IdentityJSON", "InvokeJSON", "SnapshotJSON",
			"StartTCP", "StatusJSON", "Subscribe", "TrustParent", "WaitConnected",
		},
		DesktopMethods: []string{
			"CancelSubscription", "CatalogJSON", "Close", "IdentityJSON", "InvokeJSON", "Open", "PollSubscription",
			"SnapshotJSON", "StartTCP", "StatusJSON", "Subscribe", "TrustParent", "WaitConnected",
		},
		Resources: []Resource{
			variable(protocol.BuiltinResourceCatalog, protocol.SchemaResourceCatalogV1),
			variable(protocol.BuiltinManagementTopology, protocol.SchemaManagementTopologyV1),
			variable(protocol.BuiltinManagementHealth, protocol.SchemaManagementHealthV1),
			variable(protocol.BuiltinManagementConfig, protocol.SchemaManagementConfigV1),
			stream(protocol.BuiltinManagementAudit, protocol.SchemaManagementAuditV1),
			command(protocol.BuiltinManagementIssuePermit, protocol.SchemaManagementIssuePermitV1, protocol.SchemaProvisioningPermitV1),
			command(protocol.BuiltinManagementRevokePermit, protocol.SchemaManagementRevokePermitV1, protocol.SchemaManagementResultV1),
			command(protocol.BuiltinManagementRevokeNode, protocol.SchemaManagementRevokeV1, protocol.SchemaManagementResultV1),
			command(protocol.BuiltinManagementConfigUpdate, protocol.SchemaManagementConfigUpdateV1, protocol.SchemaManagementConfigV1),
			command(protocol.BuiltinManagementPolicyGrant, protocol.SchemaManagementPolicyRuleV1, protocol.SchemaManagementResultV1),
			command(protocol.BuiltinManagementPolicyRevoke, protocol.SchemaManagementPolicyRuleV1, protocol.SchemaManagementResultV1),
			stream(protocol.BuiltinNotificationEvents, protocol.SchemaNotificationEventV1),
			command(protocol.BuiltinNotificationPublish, protocol.SchemaNotificationPublishV1, protocol.SchemaNotificationEventV1),
			variable(protocol.BuiltinFileTransfers, protocol.SchemaFileTransfersV1),
			stream(protocol.BuiltinFileProgress, protocol.SchemaFileProgressV1),
			command(protocol.BuiltinFileOffer, protocol.SchemaFileOfferV1, protocol.SchemaFileProgressV1),
			command(protocol.BuiltinFileChunk, protocol.SchemaFileChunkV1, protocol.SchemaFileProgressV1),
			command(protocol.BuiltinFileComplete, protocol.SchemaFileCompleteV1, protocol.SchemaFileProgressV1),
			command(protocol.BuiltinFileCancel, protocol.SchemaFileCancelV1, protocol.SchemaFileProgressV1),
			variable(protocol.BuiltinFlowDefinitions, protocol.SchemaFlowDefinitionsV1),
			variable(protocol.BuiltinFlowRuns, protocol.SchemaFlowRunsV1),
			stream(protocol.BuiltinFlowEvents, protocol.SchemaFlowEventV1),
			command(protocol.BuiltinFlowCreate, protocol.SchemaFlowDefinitionV1, protocol.SchemaFlowDefinitionV1),
			command(protocol.BuiltinFlowUpdate, protocol.SchemaFlowDefinitionV1, protocol.SchemaFlowDefinitionV1),
			command(protocol.BuiltinFlowRun, protocol.SchemaFlowRunV1, protocol.SchemaFlowRunSummaryV1),
			command(protocol.BuiltinFlowCancel, protocol.SchemaFlowCancelV1, protocol.SchemaFlowRunSummaryV1),
			command(protocol.BuiltinFlowArchive, protocol.SchemaFlowArchiveV1, protocol.SchemaFlowArchiveV1),
		},
	}
	sort.Strings(manifest.Methods)
	sort.Strings(manifest.DesktopMethods)
	sort.Slice(manifest.Resources, func(i, j int) bool { return manifest.Resources[i].Name < manifest.Resources[j].Name })
	for index, resource := range manifest.Resources {
		if resource.Name == "" || resource.Kind == "" {
			return Manifest{}, errors.New("binding contract contains an empty resource")
		}
		if index > 0 && manifest.Resources[index-1].Name == resource.Name {
			return Manifest{}, errors.New("binding contract contains a duplicate resource")
		}
	}
	return manifest, nil
}

func variable(name, schema string) Resource {
	return Resource{Name: name, Kind: "variable", SnapshotSchema: schema}
}

func stream(name, schema string) Resource {
	return Resource{Name: name, Kind: "stream", EventSchema: schema}
}

func command(name, request, response string) Resource {
	return Resource{Name: name, Kind: "command", RequestSchema: request, ResponseSchema: response}
}
