package protocol

import (
	"errors"
	"fmt"
)

const (
	SchemaAdmissionStatusV1           = "mfh.admission.status.v1"
	SchemaAdmissionListV1             = "mfh.admission.list.v1"
	SchemaAdmissionIssuePermitV1      = "mfh.admission.issue-permit.v1"
	SchemaAdmissionRevokePermitV1     = "mfh.admission.revoke-permit.v1"
	SchemaAdmissionDecisionV1         = "mfh.admission.decision.v1"
	SchemaAdmissionRevokeEnrollmentV1 = "mfh.admission.revoke-enrollment.v1"
	SchemaAdmissionSubmitV1           = "mfh.admission.submit.v1"
	SchemaAdmissionPermitRecordV1     = "mfh.admission.permit-record.v1"
	SchemaAdmissionRequestRecordV1    = "mfh.admission.request-record.v1"
	SchemaAdmissionEnrollmentRecordV1 = "mfh.admission.enrollment-record.v1"
	SchemaAdmissionPermitListV1       = "mfh.admission.permit-list.v1"
	SchemaAdmissionRequestListV1      = "mfh.admission.request-list.v1"
	SchemaAdmissionEnrollmentListV1   = "mfh.admission.enrollment-list.v1"

	BuiltinAdmissionStatus           = "system/admission/status"
	BuiltinAdmissionListPermits      = "system/admission/list-permits"
	BuiltinAdmissionIssuePermit      = BuiltinManagementIssuePermit
	BuiltinAdmissionRevokePermit     = "system/admission/revoke-permit"
	BuiltinAdmissionListRequests     = "system/admission/list-requests"
	BuiltinAdmissionApprove          = "system/admission/approve"
	BuiltinAdmissionReject           = "system/admission/reject"
	BuiltinAdmissionListEnrollments  = "system/admission/list-enrollments"
	BuiltinAdmissionRevokeEnrollment = "system/admission/revoke-enrollment"
	BuiltinAdmissionSubmitEnrollment = "system/admission/submit-enrollment"
	BuiltinAdmissionApplyRevocation  = "system/admission/apply-revocation"
)

type AdmissionStatusV1 struct {
	Version         int    `json:"version"`
	AuthorityNodeID string `json:"authority_node_id"`
	AuthorityEpoch  uint64 `json:"authority_epoch"`
	Permits         int    `json:"permits"`
	PendingRequests int    `json:"pending_requests"`
	Enrollments     int    `json:"enrollments"`
	Revocations     int    `json:"revocations"`
}

func (status AdmissionStatusV1) Validate() error {
	if err := validateVersion(status.Version); err != nil {
		return err
	}
	if err := validateNodeIDText("authority_node_id", status.AuthorityNodeID); err != nil {
		return err
	}
	if status.AuthorityEpoch == 0 {
		return errors.New("authority_epoch must be non-zero")
	}
	if status.Permits < 0 || status.PendingRequests < 0 || status.Enrollments < 0 || status.Revocations < 0 {
		return errors.New("admission status counts cannot be negative")
	}
	return nil
}

type AdmissionListV1 struct {
	Version int    `json:"version"`
	Cursor  string `json:"cursor,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Status  string `json:"status,omitempty"`
}

func (query AdmissionListV1) Validate() error {
	if err := validateVersion(query.Version); err != nil {
		return err
	}
	if query.Cursor != "" {
		if err := validateHexID("cursor", query.Cursor, 16); err != nil {
			return err
		}
	}
	if query.Limit < 0 || query.Limit > MaxItems {
		return fmt.Errorf("limit must be between 0 and %d", MaxItems)
	}
	return validateText("status", query.Status, MaxIdentifierBytes, false)
}

type AdmissionIssuePermitV1 struct {
	Version                    int    `json:"version"`
	RequestID                  string `json:"request_id"`
	DevicePublicKeyFingerprint string `json:"device_public_key_fingerprint"`
	TargetNodeID               string `json:"target_node_id"`
	AllowDescendants           bool   `json:"allow_descendants,omitempty"`
	AdmissionProfile           string `json:"admission_profile"`
	TTLMS                      int64  `json:"ttl_ms"`
}

func (request AdmissionIssuePermitV1) Validate() error {
	if err := validateVersion(request.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", request.RequestID, 16); err != nil {
		return err
	}
	if err := validateHexID("device_public_key_fingerprint", request.DevicePublicKeyFingerprint, 32); err != nil {
		return err
	}
	if err := validateNodeIDText("target_node_id", request.TargetNodeID); err != nil {
		return err
	}
	if err := validateText("admission_profile", request.AdmissionProfile, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if request.TTLMS <= 0 {
		return errors.New("ttl_ms must be positive")
	}
	return nil
}

type AdmissionRevokePermitV1 struct {
	Version   int    `json:"version"`
	RequestID string `json:"request_id"`
	PermitID  string `json:"permit_id"`
	Reason    string `json:"reason"`
}

func (request AdmissionRevokePermitV1) Validate() error {
	if err := validateVersion(request.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", request.RequestID, 16); err != nil {
		return err
	}
	if err := validateHexID("permit_id", request.PermitID, 16); err != nil {
		return err
	}
	return validateText("reason", request.Reason, MaxLabelBytes, true)
}

type AdmissionDecisionV1 struct {
	Version             int    `json:"version"`
	RequestID           string `json:"request_id"`
	EnrollmentRequestID string `json:"enrollment_request_id"`
	AdmissionProfile    string `json:"admission_profile,omitempty"`
	Reason              string `json:"reason,omitempty"`
}

func (request AdmissionDecisionV1) Validate() error {
	if err := validateVersion(request.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", request.RequestID, 16); err != nil {
		return err
	}
	if err := validateHexID("enrollment_request_id", request.EnrollmentRequestID, 16); err != nil {
		return err
	}
	if err := validateText("admission_profile", request.AdmissionProfile, MaxIdentifierBytes, false); err != nil {
		return err
	}
	return validateText("reason", request.Reason, MaxLabelBytes, false)
}

type AdmissionRevokeEnrollmentV1 struct {
	Version      int    `json:"version"`
	RequestID    string `json:"request_id"`
	EnrollmentID string `json:"enrollment_id"`
	Reason       string `json:"reason"`
}

func (request AdmissionRevokeEnrollmentV1) Validate() error {
	if err := validateVersion(request.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", request.RequestID, 16); err != nil {
		return err
	}
	if err := validateHexID("enrollment_id", request.EnrollmentID, 16); err != nil {
		return err
	}
	return validateText("reason", request.Reason, MaxLabelBytes, true)
}

type AdmissionSubmitV1 struct {
	Version          int    `json:"version"`
	RequestID        string `json:"request_id"`
	DevicePublicKey  string `json:"device_public_key"`
	ParentNodeID     string `json:"parent_node_id"`
	ParentPublicKey  string `json:"parent_public_key"`
	Permit           string `json:"permit,omitempty"`
	TranscriptDigest string `json:"transcript_digest"`
}

func (request AdmissionSubmitV1) Validate() error {
	if err := validateVersion(request.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", request.RequestID, 16); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("device_public_key", request.DevicePublicKey, true); err != nil {
		return err
	}
	if err := validateNodeIDText("parent_node_id", request.ParentNodeID); err != nil {
		return err
	}
	if err := validateEd25519PublicKey("parent_public_key", request.ParentPublicKey, true); err != nil {
		return err
	}
	if err := validateText("permit", request.Permit, EnrollmentMaxPayload/2, false); err != nil {
		return err
	}
	return validateHexID("transcript_digest", request.TranscriptDigest, 32)
}

type AdmissionPermitRecordV1 struct {
	Version                    int    `json:"version"`
	PermitID                   string `json:"permit_id"`
	DevicePublicKeyFingerprint string `json:"device_public_key_fingerprint"`
	TargetNodeID               string `json:"target_node_id"`
	AllowDescendants           bool   `json:"allow_descendants,omitempty"`
	AdmissionProfile           string `json:"admission_profile"`
	IssuedAtUnixMS             int64  `json:"issued_at_unix_ms"`
	ExpiresAtUnixMS            int64  `json:"expires_at_unix_ms"`
	Status                     string `json:"status"`
}

func (record AdmissionPermitRecordV1) Validate() error {
	if err := validateVersion(record.Version); err != nil {
		return err
	}
	if err := validateHexID("permit_id", record.PermitID, 16); err != nil {
		return err
	}
	if err := validateHexID("device_public_key_fingerprint", record.DevicePublicKeyFingerprint, 32); err != nil {
		return err
	}
	if err := validateNodeIDText("target_node_id", record.TargetNodeID); err != nil {
		return err
	}
	if err := validateText("admission_profile", record.AdmissionProfile, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if record.IssuedAtUnixMS <= 0 || record.ExpiresAtUnixMS <= record.IssuedAtUnixMS {
		return errors.New("permit record time range is invalid")
	}
	if err := validateAdmissionRecordStatus(record.Status, "active", "consumed", "revoked", "expired"); err != nil {
		return err
	}
	return nil
}

type AdmissionRequestRecordV1 struct {
	Version                    int    `json:"version"`
	RequestID                  string `json:"request_id"`
	DevicePublicKeyFingerprint string `json:"device_public_key_fingerprint"`
	ParentNodeID               string `json:"parent_node_id"`
	CreatedAtUnixMS            int64  `json:"created_at_unix_ms"`
	UpdatedAtUnixMS            int64  `json:"updated_at_unix_ms"`
	ExpiresAtUnixMS            int64  `json:"expires_at_unix_ms"`
	Status                     string `json:"status"`
	Reason                     string `json:"reason,omitempty"`
}

func (record AdmissionRequestRecordV1) Validate() error {
	if err := validateVersion(record.Version); err != nil {
		return err
	}
	if err := validateHexID("request_id", record.RequestID, 16); err != nil {
		return err
	}
	if err := validateHexID("device_public_key_fingerprint", record.DevicePublicKeyFingerprint, 32); err != nil {
		return err
	}
	if err := validateNodeIDText("parent_node_id", record.ParentNodeID); err != nil {
		return err
	}
	if record.CreatedAtUnixMS <= 0 || record.UpdatedAtUnixMS < record.CreatedAtUnixMS || record.ExpiresAtUnixMS <= record.CreatedAtUnixMS {
		return errors.New("request timestamps are invalid")
	}
	if err := validateAdmissionRecordStatus(record.Status, "pending", "approved", "rejected", "expired"); err != nil {
		return err
	}
	return validateText("reason", record.Reason, MaxLabelBytes, false)
}

type AdmissionEnrollmentRecordV1 struct {
	Version                    int    `json:"version"`
	EnrollmentID               string `json:"enrollment_id"`
	NodeID                     string `json:"node_id"`
	DevicePublicKeyFingerprint string `json:"device_public_key_fingerprint"`
	ParentNodeID               string `json:"parent_node_id"`
	AdmissionProfile           string `json:"admission_profile"`
	CreatedAtUnixMS            int64  `json:"created_at_unix_ms"`
	Status                     string `json:"status"`
	RevokedAtUnixMS            int64  `json:"revoked_at_unix_ms,omitempty"`
	Reason                     string `json:"reason,omitempty"`
}

func (record AdmissionEnrollmentRecordV1) Validate() error {
	if err := validateVersion(record.Version); err != nil {
		return err
	}
	if err := validateHexID("enrollment_id", record.EnrollmentID, 16); err != nil {
		return err
	}
	if err := validateNodeIDText("node_id", record.NodeID); err != nil {
		return err
	}
	if err := validateHexID("device_public_key_fingerprint", record.DevicePublicKeyFingerprint, 32); err != nil {
		return err
	}
	if err := validateNodeIDText("parent_node_id", record.ParentNodeID); err != nil {
		return err
	}
	if err := validateText("admission_profile", record.AdmissionProfile, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if record.CreatedAtUnixMS <= 0 || record.RevokedAtUnixMS < 0 {
		return errors.New("enrollment timestamps are invalid")
	}
	if err := validateAdmissionRecordStatus(record.Status, "active", "revoked"); err != nil {
		return err
	}
	if record.Status == "revoked" && record.RevokedAtUnixMS < record.CreatedAtUnixMS {
		return errors.New("revoked enrollment requires a valid revoked_at_unix_ms")
	}
	return validateText("reason", record.Reason, MaxLabelBytes, false)
}

func validateAdmissionRecordStatus(value string, allowed ...string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("invalid admission record status %q", value)
}

type AdmissionPermitListV1 struct {
	Version        int                       `json:"version"`
	AuthorityEpoch uint64                    `json:"authority_epoch"`
	Items          []AdmissionPermitRecordV1 `json:"items"`
	NextCursor     string                    `json:"next_cursor,omitempty"`
}

func (list AdmissionPermitListV1) Validate() error {
	if err := validateAdmissionListHeader(list.Version, list.AuthorityEpoch, len(list.Items)); err != nil {
		return err
	}
	for index, item := range list.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d]: %w", index, err)
		}
	}
	if list.NextCursor != "" {
		return validateHexID("next_cursor", list.NextCursor, 16)
	}
	return nil
}

type AdmissionRequestListV1 struct {
	Version        int                        `json:"version"`
	AuthorityEpoch uint64                     `json:"authority_epoch"`
	Items          []AdmissionRequestRecordV1 `json:"items"`
	NextCursor     string                     `json:"next_cursor,omitempty"`
}

func (list AdmissionRequestListV1) Validate() error {
	if err := validateAdmissionListHeader(list.Version, list.AuthorityEpoch, len(list.Items)); err != nil {
		return err
	}
	for index, item := range list.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d]: %w", index, err)
		}
	}
	if list.NextCursor != "" {
		return validateHexID("next_cursor", list.NextCursor, 16)
	}
	return nil
}

type AdmissionEnrollmentListV1 struct {
	Version        int                           `json:"version"`
	AuthorityEpoch uint64                        `json:"authority_epoch"`
	Items          []AdmissionEnrollmentRecordV1 `json:"items"`
	NextCursor     string                        `json:"next_cursor,omitempty"`
}

func (list AdmissionEnrollmentListV1) Validate() error {
	if err := validateAdmissionListHeader(list.Version, list.AuthorityEpoch, len(list.Items)); err != nil {
		return err
	}
	for index, item := range list.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d]: %w", index, err)
		}
	}
	if list.NextCursor != "" {
		return validateHexID("next_cursor", list.NextCursor, 16)
	}
	return nil
}

func validateAdmissionListHeader(version int, epoch uint64, items int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	if epoch == 0 {
		return errors.New("authority_epoch must be non-zero")
	}
	if items > MaxItems {
		return fmt.Errorf("admission list exceeds %d items", MaxItems)
	}
	return nil
}
