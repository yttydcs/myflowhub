package management

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func (c *Controller) buildEnrollmentAuthorityResources() ([]resource.Resource, error) {
	resources := make([]resource.Resource, 0, 10)
	if c.authorityNodeID != 0 {
		apply, err := c.command(
			protocol.BuiltinAdmissionApplyRevocation,
			protocol.SchemaManagementRevokeV1,
			protocol.SchemaManagementResultV1,
			"management.admission.apply-revocation",
			c.applyEnrollmentRevocation,
		)
		if err != nil {
			return nil, err
		}
		resources = append(resources, apply)
	}
	if c.authority == nil {
		return resources, nil
	}
	status, err := newJSONVariable(
		c.node.ID(), protocol.BuiltinAdmissionStatus, protocol.SchemaAdmissionStatusV1,
		"management.admission.read", c.admissionStatusSnapshot(),
	)
	if err != nil {
		return nil, err
	}
	c.admissionStatus = status
	resources = append(resources, status)
	definitions := []struct {
		name, input, output, permission string
		handler                         resource.CommandHandler
	}{
		{protocol.BuiltinAdmissionListPermits, protocol.SchemaAdmissionListV1, protocol.SchemaAdmissionPermitListV1, "management.admission.read", c.listEnrollmentPermits},
		{protocol.BuiltinAdmissionRevokePermit, protocol.SchemaAdmissionRevokePermitV1, protocol.SchemaManagementResultV1, "management.admission.revoke-permit", c.revokeEnrollmentPermit},
		{protocol.BuiltinAdmissionListRequests, protocol.SchemaAdmissionListV1, protocol.SchemaAdmissionRequestListV1, "management.admission.read", c.listEnrollmentRequests},
		{protocol.BuiltinAdmissionApprove, protocol.SchemaAdmissionDecisionV1, protocol.SchemaEnrollmentGrantV1, "management.admission.approve", c.approveEnrollment},
		{protocol.BuiltinAdmissionReject, protocol.SchemaAdmissionDecisionV1, protocol.SchemaManagementResultV1, "management.admission.reject", c.rejectEnrollment},
		{protocol.BuiltinAdmissionListEnrollments, protocol.SchemaAdmissionListV1, protocol.SchemaAdmissionEnrollmentListV1, "management.admission.read", c.listEnrollments},
		{protocol.BuiltinAdmissionRevokeEnrollment, protocol.SchemaAdmissionRevokeEnrollmentV1, protocol.SchemaManagementResultV1, "management.admission.revoke-enrollment", c.revokeEnrollment},
		{protocol.BuiltinAdmissionSubmitEnrollment, protocol.SchemaAdmissionSubmitV1, protocol.SchemaEnrollmentResultV1, "management.admission.submit", c.submitEnrollment},
	}
	for _, definition := range definitions {
		value, commandErr := c.command(definition.name, definition.input, definition.output, definition.permission, definition.handler)
		if commandErr != nil {
			return nil, commandErr
		}
		resources = append(resources, value)
	}
	return resources, nil
}

func (c *Controller) refreshAdmissionStatusLocked() error {
	if c.admissionStatus == nil {
		return nil
	}
	payload, err := protocol.EncodeJSONPayload(c.admissionStatusSnapshot(), protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	return setIfChanged(c.admissionStatus, payload)
}

func (c *Controller) admissionStatusSnapshot() *protocol.AdmissionStatusV1 {
	status := &protocol.AdmissionStatusV1{
		Version: protocol.SchemaVersionV1, AuthorityNodeID: strconv.FormatUint(uint64(c.authority.NodeID()), 10),
		AuthorityEpoch: c.authority.Generation(),
	}
	for _, permit := range c.authority.Permits() {
		if permit.Status == "active" && permit.Permit.ExpiresAtUnixMS > c.now().UTC().UnixMilli() {
			status.Permits++
		}
	}
	for _, request := range c.authority.Requests() {
		if request.Status == "pending" {
			status.PendingRequests++
		}
	}
	for _, enrollment := range c.authority.Enrollments() {
		if enrollment.Status == "active" {
			status.Enrollments++
		} else if enrollment.Status == "revoked" {
			status.Revocations++
		}
	}
	return status
}

func (c *Controller) listEnrollmentPermits(_ context.Context, input []byte) ([]byte, error) {
	query, err := decodeAdmissionListQuery(input)
	if err != nil {
		return nil, err
	}
	now := c.now().UTC().UnixMilli()
	items := make([]protocol.AdmissionPermitRecordV1, 0)
	nextCursor := ""
	for _, record := range c.authority.Permits() {
		if record.Permit.PermitID <= query.Cursor {
			continue
		}
		status := record.Status
		if status == "active" && record.Permit.ExpiresAtUnixMS <= now {
			status = "expired"
		}
		if query.Status != "" && query.Status != status {
			continue
		}
		if len(items) == query.Limit {
			nextCursor = items[len(items)-1].PermitID
			break
		}
		items = append(items, protocol.AdmissionPermitRecordV1{
			Version: protocol.SchemaVersionV1, PermitID: record.Permit.PermitID,
			DevicePublicKeyFingerprint: record.Permit.DevicePublicKeyFingerprint,
			TargetNodeID:               record.Permit.TargetNodeID, AllowDescendants: record.Permit.AllowDescendants,
			AdmissionProfile: record.Permit.AdmissionProfile, IssuedAtUnixMS: record.Permit.IssuedAtUnixMS,
			ExpiresAtUnixMS: record.Permit.ExpiresAtUnixMS, Status: status,
		})
	}
	return protocol.EncodeJSONPayload(&protocol.AdmissionPermitListV1{
		Version: protocol.SchemaVersionV1, AuthorityEpoch: c.authority.Generation(), Items: items, NextCursor: nextCursor,
	}, protocol.DefaultMaxPayload)
}

func (c *Controller) listEnrollmentRequests(_ context.Context, input []byte) ([]byte, error) {
	query, err := decodeAdmissionListQuery(input)
	if err != nil {
		return nil, err
	}
	items := make([]protocol.AdmissionRequestRecordV1, 0)
	nextCursor := ""
	for _, record := range c.authority.Requests() {
		if record.RequestID <= query.Cursor {
			continue
		}
		if query.Status != "" && query.Status != record.Status {
			continue
		}
		if len(items) == query.Limit {
			nextCursor = items[len(items)-1].RequestID
			break
		}
		items = append(items, protocol.AdmissionRequestRecordV1{
			Version: protocol.SchemaVersionV1, RequestID: record.RequestID,
			DevicePublicKeyFingerprint: record.DevicePublicKeyFingerprint,
			ParentNodeID:               strconv.FormatUint(uint64(record.ParentNodeID), 10),
			CreatedAtUnixMS:            record.CreatedAtUnixMS, UpdatedAtUnixMS: record.UpdatedAtUnixMS, ExpiresAtUnixMS: record.ExpiresAtUnixMS,
			Status: record.Status, Reason: record.Reason,
		})
	}
	return protocol.EncodeJSONPayload(&protocol.AdmissionRequestListV1{
		Version: protocol.SchemaVersionV1, AuthorityEpoch: c.authority.Generation(), Items: items, NextCursor: nextCursor,
	}, protocol.DefaultMaxPayload)
}

func (c *Controller) listEnrollments(_ context.Context, input []byte) ([]byte, error) {
	query, err := decodeAdmissionListQuery(input)
	if err != nil {
		return nil, err
	}
	items := make([]protocol.AdmissionEnrollmentRecordV1, 0)
	nextCursor := ""
	for _, record := range c.authority.Enrollments() {
		if record.Grant.EnrollmentID <= query.Cursor {
			continue
		}
		if query.Status != "" && query.Status != record.Status {
			continue
		}
		if len(items) == query.Limit {
			nextCursor = items[len(items)-1].EnrollmentID
			break
		}
		items = append(items, protocol.AdmissionEnrollmentRecordV1{
			Version: protocol.SchemaVersionV1, EnrollmentID: record.Grant.EnrollmentID,
			NodeID: record.Grant.NodeID, DevicePublicKeyFingerprint: record.DevicePublicKeyFingerprint,
			ParentNodeID: record.Grant.ParentNodeID, AdmissionProfile: record.Grant.AdmissionProfile,
			CreatedAtUnixMS: record.Grant.IssuedAtUnixMS, Status: record.Status,
			RevokedAtUnixMS: record.RevokedAtUnixMS, Reason: record.Reason,
		})
	}
	return protocol.EncodeJSONPayload(&protocol.AdmissionEnrollmentListV1{
		Version: protocol.SchemaVersionV1, AuthorityEpoch: c.authority.Generation(), Items: items, NextCursor: nextCursor,
	}, protocol.DefaultMaxPayload)
}

func (c *Controller) revokeEnrollmentPermit(ctx context.Context, input []byte) ([]byte, error) {
	var request protocol.AdmissionRevokePermitV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	if err := c.authority.RevokePermit(request.RequestID, request.PermitID, request.Reason); err != nil {
		return nil, err
	}
	c.auditAdmissionOutcome(ctx, "management.admission.revoke-permit", protocol.BuiltinAdmissionRevokePermit, request.PermitID, "revoked")
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	return managementOK()
}

func (c *Controller) approveEnrollment(ctx context.Context, input []byte) ([]byte, error) {
	var request protocol.AdmissionDecisionV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	if request.AdmissionProfile == "" {
		return nil, errors.New("admission_profile is required for approval")
	}
	grant, err := c.authority.Approve(request.RequestID, request.EnrollmentRequestID, request.AdmissionProfile)
	if err != nil {
		return nil, err
	}
	c.auditAdmissionOutcome(ctx, "management.admission.approve", protocol.BuiltinAdmissionApprove, request.EnrollmentRequestID, "approved")
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	return protocol.EncodeJSONPayload(&grant, protocol.DefaultMaxPayload)
}

func (c *Controller) rejectEnrollment(ctx context.Context, input []byte) ([]byte, error) {
	var request protocol.AdmissionDecisionV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	if request.Reason == "" {
		return nil, errors.New("reason is required for rejection")
	}
	if err := c.authority.Reject(request.RequestID, request.EnrollmentRequestID, request.Reason); err != nil {
		return nil, err
	}
	c.auditAdmissionOutcome(ctx, "management.admission.reject", protocol.BuiltinAdmissionReject, request.EnrollmentRequestID, "rejected")
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	return managementOK()
}

func (c *Controller) submitEnrollment(ctx context.Context, input []byte) ([]byte, error) {
	var request protocol.AdmissionSubmitV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	parentIDValue, _ := strconv.ParseUint(request.ParentNodeID, 10, 64)
	parentID := protocol.NodeID(parentIDValue)
	delegation, ok := command.DelegationFromContext(ctx)
	source, sourceOK := delegation.Subject()
	if !ok || !sourceOK || source != parentID {
		return nil, errors.New("enrollment submission source must be the authenticated direct parent")
	}
	deviceKey, _ := base64.RawStdEncoding.DecodeString(request.DevicePublicKey)
	parentKey, _ := base64.RawStdEncoding.DecodeString(request.ParentPublicKey)
	trustedParentKey, trusted := c.trust.PublicKey(parentID)
	if !trusted || !bytes.Equal(trustedParentKey, parentKey) {
		return nil, errors.New("enrollment submission parent key does not match the authenticated parent identity")
	}
	digestBytes, _ := hex.DecodeString(request.TranscriptDigest)
	var digest [32]byte
	copy(digest[:], digestBytes)
	var permit *protocol.EnrollmentPermitV1
	if request.Permit != "" {
		var value protocol.EnrollmentPermitV1
		if err := protocol.DecodeJSONPayload([]byte(request.Permit), protocol.EnrollmentMaxPayload, &value); err != nil {
			return nil, err
		}
		permit = &value
	}
	outcome, err := c.authority.Submit(auth.EnrollmentSubmission{
		RequestID: request.RequestID, DevicePublicKey: ed25519.PublicKey(deviceKey),
		ParentNodeID: parentID, ParentPublicKey: ed25519.PublicKey(parentKey), Permit: permit, TranscriptDigest: digest,
	})
	if err != nil {
		return nil, err
	}
	c.auditAdmissionOutcome(ctx, "management.admission.submit", protocol.BuiltinAdmissionSubmitEnrollment, outcome.RequestID, outcome.Status)
	result := protocol.EnrollmentResultV1{
		Version: protocol.SchemaVersionV1, RequestID: outcome.RequestID, Status: outcome.Status, Grant: outcome.Grant,
	}
	if outcome.Status == "pending" {
		result.RetryAfterMS = 2_000
	}
	if outcome.Status == "rejected" {
		result.Error = &protocol.ErrorPayload{Code: protocol.CodeForbidden, Message: outcome.Reason}
	}
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	return protocol.EncodeJSONPayload(&result, protocol.DefaultMaxPayload)
}

func (c *Controller) revokeEnrollment(ctx context.Context, input []byte) ([]byte, error) {
	var request protocol.AdmissionRevokeEnrollmentV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	var parentID protocol.NodeID
	found := false
	for _, record := range c.authority.Enrollments() {
		if record.Grant.EnrollmentID == request.EnrollmentID {
			parsed, _ := strconv.ParseUint(record.Grant.ParentNodeID, 10, 64)
			parentID = protocol.NodeID(parsed)
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("enrollment was not found")
	}
	nodeID, err := c.authority.RevokeEnrollment(request.RequestID, request.EnrollmentID, request.Reason)
	if err != nil {
		return nil, err
	}
	c.auditAdmissionOutcome(ctx, "management.admission.revoke-enrollment", protocol.BuiltinAdmissionRevokeEnrollment, request.EnrollmentID, "revoked")
	if parentID == c.node.ID() {
		if err := c.revoke(nodeID); err != nil && !errors.Is(err, auth.ErrUntrustedIdentity) {
			return nil, fmt.Errorf("enrollment revoked at Authority but local parent invalidation failed: %w", err)
		}
		_ = c.node.DisconnectPeer(nodeID)
	} else {
		payload, encodeErr := protocol.EncodeJSONPayload(&protocol.ManagementRevokeV1{
			Version: protocol.SchemaVersionV1, NodeID: strconv.FormatUint(uint64(nodeID), 10), Reason: request.Reason,
		}, protocol.DefaultMaxPayload)
		if encodeErr != nil {
			return nil, encodeErr
		}
		if _, invokeErr := c.node.Invoke(ctx, protocol.ResourceID{Owner: parentID, Name: protocol.BuiltinAdmissionApplyRevocation}, payload); invokeErr != nil {
			return nil, fmt.Errorf("enrollment revoked at Authority but parent %d invalidation failed: %w", parentID, invokeErr)
		}
	}
	if err := c.Refresh(); err != nil {
		return nil, err
	}
	return managementOK()
}

func (c *Controller) applyEnrollmentRevocation(ctx context.Context, input []byte) ([]byte, error) {
	delegation, ok := command.DelegationFromContext(ctx)
	source, sourceOK := delegation.Subject()
	if !ok || !sourceOK || source != c.authorityNodeID {
		return nil, errors.New("enrollment revocation must originate from the configured Admission Authority")
	}
	var request protocol.ManagementRevokeV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	parsed, _ := strconv.ParseUint(request.NodeID, 10, 64)
	nodeID := protocol.NodeID(parsed)
	if err := c.revoke(nodeID); err != nil && !errors.Is(err, auth.ErrUntrustedIdentity) {
		return nil, err
	}
	_ = c.node.DisconnectPeer(nodeID)
	return managementOK()
}

func decodeAdmissionListQuery(input []byte) (protocol.AdmissionListV1, error) {
	var query protocol.AdmissionListV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &query); err != nil {
		return protocol.AdmissionListV1{}, err
	}
	if query.Limit == 0 {
		query.Limit = 100
	}
	return query, nil
}

func admissionTTL(ttlMS int64) (time.Duration, error) {
	if ttlMS > math.MaxInt64/int64(time.Millisecond) {
		return 0, errors.New("ttl_ms overflows duration")
	}
	return time.Duration(ttlMS) * time.Millisecond, nil
}
