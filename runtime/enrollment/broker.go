package enrollment

import (
	"context"
	"crypto/ed25519"
	"errors"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
)

type Broker interface {
	AuthorityIdentity(context.Context) (protocol.NodeID, ed25519.PublicKey, error)
	Submit(context.Context, auth.EnrollmentSubmission) (auth.EnrollmentOutcome, error)
}

type LocalBroker struct {
	Authority *auth.EnrollmentAuthority
}

func (broker LocalBroker) AuthorityIdentity(context.Context) (protocol.NodeID, ed25519.PublicKey, error) {
	if broker.Authority == nil {
		return 0, nil, errors.New("local enrollment authority is required")
	}
	return broker.Authority.NodeID(), broker.Authority.PublicKey(), nil
}

func (broker LocalBroker) Submit(_ context.Context, submission auth.EnrollmentSubmission) (auth.EnrollmentOutcome, error) {
	if broker.Authority == nil {
		return auth.EnrollmentOutcome{}, errors.New("local enrollment authority is required")
	}
	return broker.Authority.Submit(submission)
}
