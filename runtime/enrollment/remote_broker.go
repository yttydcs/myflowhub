package enrollment

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
)

type CommandInvoker interface {
	Invoke(context.Context, protocol.ResourceID, []byte) ([]byte, error)
}

type RemoteBroker struct {
	Node               CommandInvoker
	AuthorityNodeID    protocol.NodeID
	AuthorityPublicKey ed25519.PublicKey
}

func (broker RemoteBroker) AuthorityIdentity(context.Context) (protocol.NodeID, ed25519.PublicKey, error) {
	if broker.Node == nil {
		return 0, nil, errors.New("remote enrollment broker node is required")
	}
	if err := broker.AuthorityNodeID.Validate(); err != nil {
		return 0, nil, err
	}
	if len(broker.AuthorityPublicKey) != ed25519.PublicKeySize {
		return 0, nil, errors.New("remote enrollment Authority public key must be Ed25519")
	}
	return broker.AuthorityNodeID, append(ed25519.PublicKey(nil), broker.AuthorityPublicKey...), nil
}

func (broker RemoteBroker) Submit(ctx context.Context, submission auth.EnrollmentSubmission) (auth.EnrollmentOutcome, error) {
	if _, _, err := broker.AuthorityIdentity(ctx); err != nil {
		return auth.EnrollmentOutcome{}, err
	}
	request := protocol.AdmissionSubmitV1{
		Version: protocol.SchemaVersionV1, RequestID: submission.RequestID,
		DevicePublicKey:  base64.RawStdEncoding.EncodeToString(submission.DevicePublicKey),
		ParentNodeID:     strconv.FormatUint(uint64(submission.ParentNodeID), 10),
		ParentPublicKey:  base64.RawStdEncoding.EncodeToString(submission.ParentPublicKey),
		TranscriptDigest: hex.EncodeToString(submission.TranscriptDigest[:]),
	}
	if submission.Permit != nil {
		permit, err := protocol.EncodeJSONPayload(*submission.Permit, protocol.EnrollmentMaxPayload)
		if err != nil {
			return auth.EnrollmentOutcome{}, err
		}
		request.Permit = string(permit)
	}
	payload, err := protocol.EncodeJSONPayload(request, protocol.EnrollmentMaxPayload)
	if err != nil {
		return auth.EnrollmentOutcome{}, err
	}
	output, err := broker.Node.Invoke(ctx, protocol.ResourceID{Owner: broker.AuthorityNodeID, Name: protocol.BuiltinAdmissionSubmitEnrollment}, payload)
	if err != nil {
		return auth.EnrollmentOutcome{}, err
	}
	var result protocol.EnrollmentResultV1
	if err := protocol.DecodeJSONPayload(output, protocol.EnrollmentMaxPayload, &result); err != nil {
		return auth.EnrollmentOutcome{}, err
	}
	outcome := auth.EnrollmentOutcome{Status: result.Status, RequestID: result.RequestID, Grant: result.Grant}
	if result.Error != nil {
		outcome.Reason = result.Error.Message
		if result.Status == "error" {
			return outcome, result.Error
		}
	}
	return outcome, nil
}
