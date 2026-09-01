package protocol

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const (
	MaxDataSchemaDepth      = 12
	MaxDataSchemaProperties = 128
	MaxDataSchemaEnumValues = 64
)

// DataSchemaDefinition is the bounded, declarative shape exported to display
// clients. Runtime payload validation remains authoritative in each payload's
// Validate method; this contract exists so clients do not need to infer data
// semantics from resource names or current values.
type DataSchemaDefinition struct {
	ID                   string                `json:"id,omitempty"`
	Type                 string                `json:"type"`
	Title                string                `json:"title,omitempty"`
	Description          string                `json:"description,omitempty"`
	Format               string                `json:"format,omitempty"`
	Unit                 string                `json:"unit,omitempty"`
	Precision            *int                  `json:"precision,omitempty"`
	Minimum              *float64              `json:"minimum,omitempty"`
	Maximum              *float64              `json:"maximum,omitempty"`
	MultipleOf           *float64              `json:"multiple_of,omitempty"`
	MinLength            *int                  `json:"min_length,omitempty"`
	MaxLength            *int                  `json:"max_length,omitempty"`
	Pattern              string                `json:"pattern,omitempty"`
	MinItems             *int                  `json:"min_items,omitempty"`
	MaxItems             *int                  `json:"max_items,omitempty"`
	Enum                 []string              `json:"enum,omitempty"`
	ReadOnly             bool                  `json:"read_only,omitempty"`
	WriteOnly            bool                  `json:"write_only,omitempty"`
	Sensitive            bool                  `json:"sensitive,omitempty"`
	Properties           []DataSchemaProperty  `json:"properties,omitempty"`
	Items                *DataSchemaDefinition `json:"items,omitempty"`
	AdditionalProperties *DataSchemaDefinition `json:"additional_properties,omitempty"`
}

type DataSchemaProperty struct {
	Name     string                `json:"name"`
	Required bool                  `json:"required,omitempty"`
	Schema   *DataSchemaDefinition `json:"schema"`
}

func (s DataSchemaDefinition) Validate() error {
	if err := validateText("data schema id", s.ID, MaxSchemaBytes, true); err != nil {
		return err
	}
	properties := 0
	return s.validateNode(1, &properties)
}

func (s DataSchemaDefinition) validateNode(depth int, properties *int) error {
	if depth > MaxDataSchemaDepth {
		return fmt.Errorf("data schema exceeds depth %d", MaxDataSchemaDepth)
	}
	if s.Type != "null" && s.Type != "boolean" && s.Type != "integer" && s.Type != "number" && s.Type != "string" && s.Type != "object" && s.Type != "array" {
		return fmt.Errorf("unsupported data schema type %q", s.Type)
	}
	for name, value := range map[string]string{
		"title": s.Title, "description": s.Description, "format": s.Format, "unit": s.Unit, "pattern": s.Pattern,
	} {
		if err := validateText("data schema "+name, value, MaxAttributeValueBytes, false); err != nil {
			return err
		}
	}
	if s.Precision != nil && (*s.Precision < 0 || *s.Precision > 12) {
		return errors.New("data schema precision must be between 0 and 12")
	}
	if s.Minimum != nil && s.Maximum != nil && *s.Minimum > *s.Maximum {
		return errors.New("data schema minimum exceeds maximum")
	}
	if s.MultipleOf != nil && *s.MultipleOf <= 0 {
		return errors.New("data schema multiple_of must be positive")
	}
	if err := validateOptionalBounds("length", s.MinLength, s.MaxLength); err != nil {
		return err
	}
	if s.Pattern != "" {
		if s.Type != "string" {
			return errors.New("data schema pattern is only valid for strings")
		}
		if _, err := regexp.Compile(s.Pattern); err != nil {
			return fmt.Errorf("data schema pattern is invalid: %w", err)
		}
	}
	if err := validateOptionalBounds("items", s.MinItems, s.MaxItems); err != nil {
		return err
	}
	if len(s.Enum) > MaxDataSchemaEnumValues {
		return fmt.Errorf("data schema enum exceeds %d entries", MaxDataSchemaEnumValues)
	}
	for index, value := range s.Enum {
		if err := validateText("data schema enum", value, MaxAttributeValueBytes, true); err != nil {
			return fmt.Errorf("enum[%d]: %w", index, err)
		}
		if index > 0 && s.Enum[index-1] >= value {
			return errors.New("data schema enum must be unique and sorted")
		}
	}
	if s.Sensitive && !s.WriteOnly {
		return errors.New("sensitive data schema fields must be write_only")
	}
	switch s.Type {
	case "object":
		if s.Items != nil {
			return errors.New("object data schema must not declare items")
		}
		seen := make(map[string]struct{}, len(s.Properties))
		for index, property := range s.Properties {
			if err := validateText("data schema property name", property.Name, MaxIdentifierBytes, true); err != nil {
				return fmt.Errorf("properties[%d]: %w", index, err)
			}
			if property.Schema == nil {
				return fmt.Errorf("properties[%d] schema is required", index)
			}
			if _, exists := seen[property.Name]; exists {
				return fmt.Errorf("duplicate data schema property %q", property.Name)
			}
			seen[property.Name] = struct{}{}
			(*properties)++
			if *properties > MaxDataSchemaProperties {
				return fmt.Errorf("data schema exceeds %d properties", MaxDataSchemaProperties)
			}
			if err := property.Schema.validateNode(depth+1, properties); err != nil {
				return fmt.Errorf("property %q: %w", property.Name, err)
			}
		}
		if s.AdditionalProperties != nil {
			if err := s.AdditionalProperties.validateNode(depth+1, properties); err != nil {
				return fmt.Errorf("additional_properties: %w", err)
			}
		}
	case "array":
		if s.Items == nil {
			return errors.New("array data schema requires items")
		}
		if len(s.Properties) != 0 || s.AdditionalProperties != nil {
			return errors.New("array data schema must not declare object properties")
		}
		if err := s.Items.validateNode(depth+1, properties); err != nil {
			return fmt.Errorf("items: %w", err)
		}
	default:
		if len(s.Properties) != 0 || s.Items != nil || s.AdditionalProperties != nil {
			return errors.New("scalar data schema contains structural fields")
		}
	}
	return nil
}

func validateOptionalBounds(name string, minimum, maximum *int) error {
	if minimum != nil && *minimum < 0 || maximum != nil && *maximum < 0 {
		return fmt.Errorf("data schema %s bounds must not be negative", name)
	}
	if minimum != nil && maximum != nil && *minimum > *maximum {
		return fmt.Errorf("data schema minimum %s exceeds maximum", name)
	}
	return nil
}

type dataSchemaType struct {
	id     string
	title  string
	typeOf reflect.Type
}

// BuiltinDataSchemas returns the provider-owned descriptions for canonical
// protocol payloads. The slice is sorted by schema ID for deterministic output.
func BuiltinDataSchemas() ([]DataSchemaDefinition, error) {
	types := []dataSchemaType{
		{SchemaResourceCatalogV2, "Resource catalog", reflect.TypeOf(ResourceCatalogV2{})},
		{SchemaCollectionListRequestV1, "List collection members", reflect.TypeOf(CollectionListRequestV1{})},
		{SchemaCollectionMemberRequestV1, "Get collection member", reflect.TypeOf(CollectionMemberRequestV1{})},
		{SchemaCollectionMemberV1, "Collection member", reflect.TypeOf(CollectionMemberV1{})},
		{SchemaCollectionPageV1, "Collection page", reflect.TypeOf(CollectionPageV1{})},
		{SchemaFilesystemReadRequestV1, "Read filesystem member", reflect.TypeOf(FilesystemReadRequestV1{})},
		{SchemaFilesystemContentV1, "Filesystem member content", reflect.TypeOf(FilesystemContentV1{})},
		{SchemaAdmissionStatusV1, "Admission status", reflect.TypeOf(AdmissionStatusV1{})},
		{SchemaAdmissionListV1, "List admission records", reflect.TypeOf(AdmissionListV1{})},
		{SchemaAdmissionIssuePermitV1, "Issue admission permit", reflect.TypeOf(AdmissionIssuePermitV1{})},
		{SchemaAdmissionRevokePermitV1, "Revoke admission permit", reflect.TypeOf(AdmissionRevokePermitV1{})},
		{SchemaAdmissionDecisionV1, "Decide admission request", reflect.TypeOf(AdmissionDecisionV1{})},
		{SchemaAdmissionRevokeEnrollmentV1, "Revoke enrollment", reflect.TypeOf(AdmissionRevokeEnrollmentV1{})},
		{SchemaAdmissionSubmitV1, "Submit admission request", reflect.TypeOf(AdmissionSubmitV1{})},
		{SchemaAdmissionPermitRecordV1, "Admission permit", reflect.TypeOf(AdmissionPermitRecordV1{})},
		{SchemaAdmissionRequestRecordV1, "Admission request", reflect.TypeOf(AdmissionRequestRecordV1{})},
		{SchemaAdmissionEnrollmentRecordV1, "Admission enrollment", reflect.TypeOf(AdmissionEnrollmentRecordV1{})},
		{SchemaAdmissionPermitListV1, "Admission permits", reflect.TypeOf(AdmissionPermitListV1{})},
		{SchemaAdmissionRequestListV1, "Admission requests", reflect.TypeOf(AdmissionRequestListV1{})},
		{SchemaAdmissionEnrollmentListV1, "Admission enrollments", reflect.TypeOf(AdmissionEnrollmentListV1{})},
		{SchemaEnrollmentClientInitV1, "Enrollment client initialization", reflect.TypeOf(EnrollmentClientInitV1{})},
		{SchemaEnrollmentChallengeV1, "Enrollment challenge", reflect.TypeOf(EnrollmentChallengeV1{})},
		{SchemaEnrollmentProofV1, "Enrollment proof", reflect.TypeOf(EnrollmentProofV1{})},
		{SchemaEnrollmentPermitV1, "Enrollment permit", reflect.TypeOf(EnrollmentPermitV1{})},
		{SchemaEnrollmentGrantV1, "Enrollment grant", reflect.TypeOf(EnrollmentGrantV1{})},
		{SchemaEnrollmentResultV1, "Enrollment result", reflect.TypeOf(EnrollmentResultV1{})},
		{SchemaFileOfferV1, "File offer", reflect.TypeOf(FileOfferV1{})},
		{SchemaFileChunkV1, "File chunk", reflect.TypeOf(FileChunkV1{})},
		{SchemaFileCompleteV1, "File completion", reflect.TypeOf(FileCompleteV1{})},
		{SchemaFileCancelV1, "File cancellation", reflect.TypeOf(FileCancelV1{})},
		{SchemaFileProgressV1, "File progress", reflect.TypeOf(FileProgressV1{})},
		{SchemaFileTransfersV1, "File transfers", reflect.TypeOf(FileTransfersV1{})},
		{SchemaFlowDefinitionV1, "Flow definition", reflect.TypeOf(FlowDefinitionV1{})},
		{SchemaFlowRunV1, "Run flow", reflect.TypeOf(FlowRunV1{})},
		{SchemaFlowRunSummaryV1, "Flow run", reflect.TypeOf(FlowRunSummaryV1{})},
		{SchemaFlowCancelV1, "Cancel flow run", reflect.TypeOf(FlowCancelV1{})},
		{SchemaFlowEventV1, "Flow event", reflect.TypeOf(FlowEventV1{})},
		{SchemaFlowArchiveV1, "Flow archive", reflect.TypeOf(FlowArchiveV1{})},
		{SchemaManagementTopologyV1, "Node topology", reflect.TypeOf(ManagementTopologyV1{})},
		{SchemaManagementHealthV1, "Hub health", reflect.TypeOf(ManagementHealthV1{})},
		{SchemaManagementConfigV1, "Hub configuration", reflect.TypeOf(ManagementConfigV1{})},
		{SchemaManagementAdmitV1, "Admission", reflect.TypeOf(ManagementAdmitV1{})},
		{SchemaManagementRevokeV1, "Revoke node", reflect.TypeOf(ManagementRevokeV1{})},
		{SchemaManagementIssuePermitV1, "Issue permit", reflect.TypeOf(ManagementIssuePermitV1{})},
		{SchemaManagementRevokePermitV1, "Revoke permit", reflect.TypeOf(ManagementRevokePermitV1{})},
		{SchemaManagementConfigUpdateV1, "Update Hub configuration", reflect.TypeOf(ManagementConfigUpdateV1{})},
		{SchemaManagementPolicyRuleV1, "Policy rule", reflect.TypeOf(ManagementPolicyRuleV1{})},
		{SchemaPolicyDefinitionV1, "Policy definition", reflect.TypeOf(PolicyDefinitionV1{})},
		{SchemaPolicyDefinitionPutV1, "Put policy definition", reflect.TypeOf(PolicyDefinitionPutV1{})},
		{SchemaPolicyDefinitionDeleteV1, "Delete policy definition", reflect.TypeOf(PolicyDefinitionDeleteV1{})},
		{SchemaPolicyBindingV1, "Policy binding", reflect.TypeOf(PolicyBindingV1{})},
		{SchemaPolicyBindingCreateV1, "Create policy binding", reflect.TypeOf(PolicyBindingCreateV1{})},
		{SchemaPolicyBindingRevokeV1, "Revoke policy binding", reflect.TypeOf(PolicyBindingRevokeV1{})},
		{SchemaPolicyGrantV1, "Exact policy grant", reflect.TypeOf(PolicyGrantV1{})},
		{SchemaPolicyEvaluateRequestV1, "Evaluate policy", reflect.TypeOf(PolicyEvaluateRequestV1{})},
		{SchemaPolicyEvaluationV1, "Policy evaluation", reflect.TypeOf(PolicyEvaluationV1{})},
		{SchemaManagementAuditV1, "Audit event", reflect.TypeOf(ManagementAuditV1{})},
		{SchemaManagementResultV1, "Management result", reflect.TypeOf(ManagementResultV1{})},
		{SchemaNotificationEventV1, "Notification", reflect.TypeOf(NotificationEventV1{})},
		{SchemaNotificationPublishV1, "Publish notification", reflect.TypeOf(NotificationPublishV1{})},
		{SchemaProvisioningPermitV1, "Provisioning permit", reflect.TypeOf(ProvisioningPermitV1{})},
		{SchemaSessionGrantV2, "Session grant", reflect.TypeOf(SessionGrantV2{})},
		{SchemaSessionDataV2, "Session data", reflect.TypeOf(SessionDataV2{})},
		{SchemaSessionCloseV2, "Session close", reflect.TypeOf(SessionCloseV2{})},
		{SchemaVariableWriteV2, "Variable write", reflect.TypeOf(VariableWriteV2{})},
	}
	result := make([]DataSchemaDefinition, 0, len(types))
	for _, item := range types {
		schema, err := dataSchemaFromType(item.typeOf, 1)
		if err != nil {
			return nil, fmt.Errorf("build data schema %s: %w", item.id, err)
		}
		schema.ID = item.id
		schema.Title = item.title
		annotateBuiltinDataSchema(&schema)
		if err := schema.Validate(); err != nil {
			return nil, fmt.Errorf("validate data schema %s: %w", item.id, err)
		}
		result = append(result, schema)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func dataSchemaFromType(value reflect.Type, depth int) (DataSchemaDefinition, error) {
	if depth > MaxDataSchemaDepth {
		return DataSchemaDefinition{}, fmt.Errorf("Go payload exceeds data schema depth %d", MaxDataSchemaDepth)
	}
	for value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.PkgPath() == "encoding/json" && value.Name() == "RawMessage" {
		return DataSchemaDefinition{Type: "string", Format: "json"}, nil
	}
	switch value.Kind() {
	case reflect.Bool:
		return DataSchemaDefinition{Type: "boolean"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		step := 1.0
		return DataSchemaDefinition{Type: "integer", MultipleOf: &step}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		minimum, step := 0.0, 1.0
		return DataSchemaDefinition{Type: "integer", Minimum: &minimum, MultipleOf: &step}, nil
	case reflect.Float32, reflect.Float64:
		return DataSchemaDefinition{Type: "number"}, nil
	case reflect.String:
		return DataSchemaDefinition{Type: "string"}, nil
	case reflect.Slice, reflect.Array:
		if value.Elem().Kind() == reflect.Uint8 {
			return DataSchemaDefinition{Type: "string", Format: "base64"}, nil
		}
		items, err := dataSchemaFromType(value.Elem(), depth+1)
		if err != nil {
			return DataSchemaDefinition{}, err
		}
		maximum := MaxItems
		return DataSchemaDefinition{Type: "array", Items: &items, MaxItems: &maximum}, nil
	case reflect.Map:
		if value.Key().Kind() != reflect.String {
			return DataSchemaDefinition{}, errors.New("only string-keyed maps are supported")
		}
		additional, err := dataSchemaFromType(value.Elem(), depth+1)
		if err != nil {
			return DataSchemaDefinition{}, err
		}
		return DataSchemaDefinition{Type: "object", AdditionalProperties: &additional}, nil
	case reflect.Struct:
		schema := DataSchemaDefinition{Type: "object"}
		for index := 0; index < value.NumField(); index++ {
			field := value.Field(index)
			if !field.IsExported() {
				continue
			}
			name, options, _ := strings.Cut(field.Tag.Get("json"), ",")
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			child, err := dataSchemaFromType(field.Type, depth+1)
			if err != nil {
				return DataSchemaDefinition{}, fmt.Errorf("field %s: %w", field.Name, err)
			}
			child.Title = humanizeDataSchemaName(name)
			if child.Type == "integer" && strings.HasSuffix(name, "_at_unix_ms") {
				child.Format = "unix-ms"
			} else if (child.Type == "integer" || child.Type == "number") && strings.HasSuffix(name, "_ms") {
				child.Unit = "ms"
			}
			schema.Properties = append(schema.Properties, DataSchemaProperty{
				Name: name, Required: options != "omitempty" && !strings.Contains(options, "omitempty"), Schema: &child,
			})
		}
		return schema, nil
	default:
		return DataSchemaDefinition{}, fmt.Errorf("unsupported Go payload kind %s", value.Kind())
	}
}

func humanizeDataSchemaName(value string) string {
	parts := strings.Fields(strings.ReplaceAll(value, "_", " "))
	for index := range parts {
		parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
	}
	return strings.Join(parts, " ")
}

func annotateBuiltinDataSchema(schema *DataSchemaDefinition) {
	for _, path := range []string{"version"} {
		setDataSchemaMinimum(schema, path, 1)
	}
	for _, path := range []string{"revision", "expected_revision", "epoch", "generation", "authority_epoch"} {
		setDataSchemaMinimum(schema, path, 1)
	}
	switch schema.ID {
	case SchemaCollectionListRequestV1:
		setDataSchemaMinimum(schema, "limit", 1)
		setDataSchemaMaximum(schema, "limit", MaxCollectionPageMembers)
		setDataSchemaMaxLength(schema, "parent", MaxCollectionParentBytes)
		setDataSchemaMaxLength(schema, "cursor", MaxCollectionCursorBytes)
	case SchemaCollectionMemberRequestV1:
		setDataSchemaMaxLength(schema, "key", MaxCollectionMemberKeyBytes)
	case SchemaCollectionMemberV1:
		annotateCollectionMemberDataSchema(schema)
	case SchemaCollectionPageV1:
		setDataSchemaMaximum(schema, "revision", float64(MaxCollectionRevision))
		setDataSchemaMaxLength(schema, "parent", MaxCollectionParentBytes)
		setDataSchemaMaxLength(schema, "next_cursor", MaxCollectionCursorBytes)
		setDataSchemaMaxItems(schema, "members", MaxCollectionPageMembers)
		if members := dataSchemaAtPath(schema, "members"); members != nil && members.Items != nil {
			annotateCollectionMemberDataSchema(members.Items)
		}
	case SchemaFilesystemReadRequestV1:
		setDataSchemaMaxLength(schema, "key", MaxCollectionMemberKeyBytes)
		setDataSchemaMinimum(schema, "max_bytes", 1)
		setDataSchemaMaximum(schema, "max_bytes", MaxFilesystemReadBytes)
		setDataSchemaMaxLength(schema, "expected_revision", MaxFilesystemRevisionBytes)
		clearDataSchemaMinimum(schema, "expected_revision")
	case SchemaFilesystemContentV1:
		setDataSchemaMaxLength(schema, "key", MaxCollectionMemberKeyBytes)
		setDataSchemaMaxLength(schema, "content_type", MaxContentTypeBytes)
		setDataSchemaEnum(schema, "encoding", FilesystemEncodingUTF8, FilesystemEncodingBase64)
		setDataSchemaMaxLength(schema, "data", base64.StdEncoding.EncodedLen(MaxFilesystemReadBytes))
		setDataSchemaMinimum(schema, "size", 0)
		setDataSchemaMaximum(schema, "size", MaxFilesystemReadBytes)
		setDataSchemaMinimum(schema, "modified_unix_ms", 0)
		setDataSchemaMaxLength(schema, "revision", MaxFilesystemRevisionBytes)
		clearDataSchemaMinimum(schema, "revision")
	case SchemaAdmissionStatusV1:
		for _, path := range []string{"permits", "pending_requests", "enrollments", "revocations"} {
			setDataSchemaMinimum(schema, path, 0)
		}
	case SchemaAdmissionListV1:
		setDataSchemaMinimum(schema, "limit", 0)
		setDataSchemaMaximum(schema, "limit", MaxItems)
	case SchemaAdmissionPermitRecordV1:
		setDataSchemaEnum(schema, "status", "active", "consumed", "expired", "revoked")
	case SchemaAdmissionRequestRecordV1:
		setDataSchemaEnum(schema, "status", "approved", "expired", "pending", "rejected")
	case SchemaAdmissionEnrollmentRecordV1:
		setDataSchemaEnum(schema, "status", "active", "revoked")
	case SchemaEnrollmentResultV1:
		setDataSchemaEnum(schema, "status", "error", "granted", "pending", "rejected")
	case SchemaFileProgressV1:
		setDataSchemaEnum(schema, "state", "cancelled", "completed", "failed", "offered", "receiving")
	case SchemaManagementHealthV1:
		setDataSchemaEnum(schema, "state", "degraded", "running", "starting", "stopping")
	case SchemaManagementAuditV1:
		setDataSchemaEnum(schema, "decision", "allow", "deny")
	case SchemaManagementResultV1:
		setDataSchemaEnum(schema, "status", "ok")
	case SchemaNotificationEventV1, SchemaNotificationPublishV1:
		setDataSchemaMaxLength(schema, "body", MaxNotificationBodyBytes)
	case SchemaManagementAdmitV1, SchemaAdmissionSubmitV1, SchemaEnrollmentClientInitV1:
		setDataSchemaSensitive(schema, "permit")
	case SchemaVariableWriteV2:
		setDataSchemaFormat(schema, "value", "json-base64")
	}
}

func annotateCollectionMemberDataSchema(schema *DataSchemaDefinition) {
	setDataSchemaMaxLength(schema, "key", MaxCollectionMemberKeyBytes)
	setDataSchemaMaxLength(schema, "kind", MaxCollectionMemberKindBytes)
	setDataSchemaMaxLength(schema, "label", MaxLabelBytes)
	setDataSchemaMaxLength(schema, "content_type", MaxContentTypeBytes)
	setDataSchemaMaxLength(schema, "schema", MaxSchemaBytes)
	setDataSchemaMaxItems(schema, "capabilities", MaxCapabilities)
	if attributes := dataSchemaAtPath(schema, "attributes"); attributes != nil && attributes.AdditionalProperties != nil {
		maximum := MaxAttributeValueBytes
		attributes.AdditionalProperties.MaxLength = &maximum
	}
}

func dataSchemaAtPath(schema *DataSchemaDefinition, path string) *DataSchemaDefinition {
	current := schema
	for _, segment := range strings.Split(path, ".") {
		if current == nil || current.Type != "object" {
			return nil
		}
		var next *DataSchemaDefinition
		for index := range current.Properties {
			if current.Properties[index].Name == segment {
				next = current.Properties[index].Schema
				break
			}
		}
		current = next
	}
	return current
}

func setDataSchemaMinimum(schema *DataSchemaDefinition, path string, value float64) {
	if field := dataSchemaAtPath(schema, path); field != nil {
		field.Minimum = &value
	}
}

func clearDataSchemaMinimum(schema *DataSchemaDefinition, path string) {
	if field := dataSchemaAtPath(schema, path); field != nil {
		field.Minimum = nil
	}
}

func setDataSchemaMaximum(schema *DataSchemaDefinition, path string, value float64) {
	if field := dataSchemaAtPath(schema, path); field != nil {
		field.Maximum = &value
	}
}

func setDataSchemaMaxLength(schema *DataSchemaDefinition, path string, value int) {
	if field := dataSchemaAtPath(schema, path); field != nil {
		field.MaxLength = &value
	}
}

func setDataSchemaMaxItems(schema *DataSchemaDefinition, path string, value int) {
	if field := dataSchemaAtPath(schema, path); field != nil {
		field.MaxItems = &value
	}
}

func setDataSchemaEnum(schema *DataSchemaDefinition, path string, values ...string) {
	if field := dataSchemaAtPath(schema, path); field != nil {
		field.Enum = append([]string(nil), values...)
		sort.Strings(field.Enum)
	}
}

func setDataSchemaSensitive(schema *DataSchemaDefinition, path string) {
	if field := dataSchemaAtPath(schema, path); field != nil {
		field.WriteOnly = true
		field.Sensitive = true
	}
}

func setDataSchemaFormat(schema *DataSchemaDefinition, path, format string) {
	if field := dataSchemaAtPath(schema, path); field != nil {
		field.Format = format
	}
}

// MarshalBuiltinDataSchemas is shared by generators and freshness tests.
func MarshalBuiltinDataSchemas() ([]byte, error) {
	definitions, err := BuiltinDataSchemas()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(struct {
		Version int                    `json:"version"`
		Schemas []DataSchemaDefinition `json:"schemas"`
	}{Version: 1, Schemas: definitions}, "", "  ")
}
