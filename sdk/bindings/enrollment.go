package bindings

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/enrollment"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

// EnrollmentBootstrap owns only the pre-Node Enrollment handshake and its
// credential mutation. It cannot start an ordinary Node runtime or perform
// Resource operations.
type EnrollmentBootstrap struct {
	mu         sync.Mutex
	enrollment *auth.EnrollmentClientState
	runCtx     context.Context
	cancel     context.CancelFunc
	closed     bool
	enrolling  bool
	wg         sync.WaitGroup
}

func NewEnrollmentBootstrap(stateDirectory string) (*EnrollmentBootstrap, error) {
	if stateDirectory == "" {
		return nil, errors.New("binding state directory is required")
	}
	store, err := keystore.New(filepath.Join(stateDirectory, "state"))
	if err != nil {
		return nil, err
	}
	state, err := auth.LoadOrCreateEnrollmentClientState(store)
	if err != nil {
		return nil, err
	}
	return newEnrollmentBootstrap(state), nil
}

func NewEnrollmentBootstrapWithCredentialStore(stateDirectory string, store auth.EnrollmentCredentialStore) (*EnrollmentBootstrap, error) {
	if stateDirectory == "" {
		return nil, errors.New("binding state directory is required")
	}
	if store == nil {
		return nil, errors.New("binding protected Enrollment credential store is required")
	}
	state, err := auth.LoadOrCreateEnrollmentClientStateWithStore(store)
	if err != nil {
		return nil, err
	}
	return newEnrollmentBootstrap(state), nil
}

func newEnrollmentBootstrap(state *auth.EnrollmentClientState) *EnrollmentBootstrap {
	runCtx, cancel := context.WithCancel(context.Background())
	return &EnrollmentBootstrap{enrollment: state, runCtx: runCtx, cancel: cancel}
}

func (bootstrap *EnrollmentBootstrap) EnrollmentStatusJSON() (string, error) {
	if bootstrap == nil {
		return "", errors.New("binding Enrollment bootstrap is closed")
	}
	bootstrap.mu.Lock()
	defer bootstrap.mu.Unlock()
	if bootstrap.closed || bootstrap.enrollment == nil {
		return "", errors.New("binding Enrollment bootstrap is closed")
	}
	return enrollmentStatusJSON(bootstrap.enrollment.Snapshot())
}

func (bootstrap *EnrollmentBootstrap) EnrollTCP(endpoint, permitJSON string, allowTOFU bool, expectedParentID int64, expectedParentKey, expectedAuthorityKey string, timeoutMS int64) (string, error) {
	if bootstrap == nil {
		return "", errors.New("binding Enrollment bootstrap is closed")
	}
	address := link.Endpoint(endpoint)
	if err := address.Validate(); err != nil {
		return "", err
	}
	timeout, err := bindingDuration(timeoutMS)
	if err != nil {
		return "", fmt.Errorf("binding timeout: %w", err)
	}
	bootstrap.mu.Lock()
	if bootstrap.closed || bootstrap.enrollment == nil {
		bootstrap.mu.Unlock()
		return "", errors.New("binding Enrollment bootstrap is closed")
	}
	if bootstrap.enrolling {
		bootstrap.mu.Unlock()
		return "", errors.New("binding Enrollment bootstrap is already enrolling")
	}
	bootstrap.enrolling = true
	bootstrap.wg.Add(1)
	state := bootstrap.enrollment
	runCtx := bootstrap.runCtx
	bootstrap.mu.Unlock()
	defer func() {
		bootstrap.mu.Lock()
		bootstrap.enrolling = false
		bootstrap.mu.Unlock()
		bootstrap.wg.Done()
	}()
	ctx, cancel := context.WithTimeout(runCtx, timeout)
	defer cancel()
	result, _, err := enrollStateTCP(ctx, state, address, permitJSON, allowTOFU, expectedParentID, expectedParentKey, expectedAuthorityKey)
	return result, err
}

func (bootstrap *EnrollmentBootstrap) Close() error {
	if bootstrap == nil {
		return nil
	}
	bootstrap.mu.Lock()
	if !bootstrap.closed {
		bootstrap.closed = true
		bootstrap.cancel()
	}
	bootstrap.mu.Unlock()
	bootstrap.wg.Wait()
	return nil
}

func NewEnrollmentClient(stateDirectory string) (*Client, error) {
	if stateDirectory == "" {
		return nil, errors.New("binding state directory is required")
	}
	store, err := keystore.New(filepath.Join(stateDirectory, "state"))
	if err != nil {
		return nil, err
	}
	state, err := auth.LoadOrCreateEnrollmentClientState(store)
	if err != nil {
		return nil, err
	}
	return newEnrollmentBindingClient(state)
}

func NewEnrollmentClientWithCredentialStore(stateDirectory string, store auth.EnrollmentCredentialStore) (*Client, error) {
	if stateDirectory == "" {
		return nil, errors.New("binding state directory is required")
	}
	if store == nil {
		return nil, errors.New("binding protected Enrollment credential store is required")
	}
	state, err := auth.LoadOrCreateEnrollmentClientStateWithStore(store)
	if err != nil {
		return nil, err
	}
	return newEnrollmentBindingClient(state)
}

func newEnrollmentBindingClient(enrollmentState *auth.EnrollmentClientState) (*Client, error) {
	client := &Client{enrollment: enrollmentState, ownsRuntime: true, subscriptions: make(map[int64]context.CancelFunc)}
	if err := client.installEnrolledState(); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) EnrollmentStatusJSON() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.enrollment == nil {
		return "", errors.New("binding Enrollment client is closed")
	}
	return enrollmentStatusJSON(c.enrollment.Snapshot())
}

func enrollmentStatusJSON(snapshot auth.EnrollmentClientSnapshot) (string, error) {
	nodeID := ""
	enrollmentID := ""
	if snapshot.Grant != nil {
		nodeID = snapshot.Grant.NodeID
		enrollmentID = snapshot.Grant.EnrollmentID
	}
	return encodeJSON(struct {
		Status             string `json:"status"`
		RequestID          string `json:"request_id"`
		DevicePublicKey    string `json:"device_public_key"`
		NodeID             string `json:"node_id,omitempty"`
		EnrollmentID       string `json:"enrollment_id,omitempty"`
		ParentNodeID       string `json:"parent_node_id,omitempty"`
		ParentPublicKey    string `json:"parent_public_key,omitempty"`
		AuthorityNodeID    string `json:"authority_node_id,omitempty"`
		AuthorityPublicKey string `json:"authority_public_key,omitempty"`
	}{
		Status: snapshot.Status, RequestID: snapshot.RequestID,
		DevicePublicKey: base64.RawStdEncoding.EncodeToString(snapshot.DevicePublicKey),
		NodeID:          nodeID, EnrollmentID: enrollmentID,
		ParentNodeID:       bindingOptionalNodeID(snapshot.ParentNodeID),
		ParentPublicKey:    base64.RawStdEncoding.EncodeToString(snapshot.ParentPublicKey),
		AuthorityNodeID:    bindingOptionalNodeID(snapshot.AuthorityNodeID),
		AuthorityPublicKey: base64.RawStdEncoding.EncodeToString(snapshot.AuthorityPublicKey),
	})
}

func (c *Client) EnrollTCP(endpoint, permitJSON string, allowTOFU bool, expectedParentID int64, expectedParentKey, expectedAuthorityKey string, timeoutMS int64) (string, error) {
	address := link.Endpoint(endpoint)
	if err := address.Validate(); err != nil {
		return "", err
	}
	ctx, cancel, err := bindingContext(timeoutMS)
	if err != nil {
		return "", err
	}
	defer cancel()
	c.mu.Lock()
	if c.closed || c.enrollment == nil {
		c.mu.Unlock()
		return "", errors.New("binding Enrollment client is closed")
	}
	if c.runtime != nil {
		c.mu.Unlock()
		return "", errors.New("binding client must stop before Enrollment")
	}
	enrollmentState := c.enrollment
	c.mu.Unlock()
	resultJSON, granted, err := enrollStateTCP(ctx, enrollmentState, address, permitJSON, allowTOFU, expectedParentID, expectedParentKey, expectedAuthorityKey)
	if err != nil {
		return "", err
	}
	if granted {
		c.mu.Lock()
		err = c.installEnrolledStateLocked()
		c.mu.Unlock()
		if err != nil {
			return "", err
		}
	}
	return resultJSON, nil
}

func enrollStateTCP(ctx context.Context, enrollmentState *auth.EnrollmentClientState, address link.Endpoint, permitJSON string, allowTOFU bool, expectedParentID int64, expectedParentKey, expectedAuthorityKey string) (string, bool, error) {
	snapshot := enrollmentState.Snapshot()
	options := enrollment.ClientOptions{RequestID: snapshot.RequestID, AllowTOFU: allowTOFU}
	if snapshot.ParentNodeID != 0 {
		options.ExpectedParentNodeID = snapshot.ParentNodeID
		options.ExpectedParentPublicKey = snapshot.ParentPublicKey
		options.ExpectedAuthorityNodeID = snapshot.AuthorityNodeID
		options.ExpectedAuthorityPublicKey = snapshot.AuthorityPublicKey
	} else {
		if expectedParentID < 0 {
			return "", false, errors.New("expected parent Node ID cannot be negative")
		}
		if expectedParentID > 0 {
			options.ExpectedParentNodeID = protocol.NodeID(expectedParentID)
		}
		if expectedParentKey != "" {
			key, err := bindingPublicKey("expected parent", expectedParentKey)
			if err != nil {
				return "", false, err
			}
			options.ExpectedParentPublicKey = key
		}
		if expectedAuthorityKey != "" {
			key, err := bindingPublicKey("expected Admission Authority", expectedAuthorityKey)
			if err != nil {
				return "", false, err
			}
			options.ExpectedAuthorityPublicKey = key
		}
	}
	if permitJSON != "" {
		var permit protocol.EnrollmentPermitV1
		if err := protocol.DecodeJSONPayload([]byte(permitJSON), protocol.EnrollmentMaxPayload, &permit); err != nil {
			return "", false, fmt.Errorf("decode Enrollment Permit: %w", err)
		}
		options.Permit = &permit
	}
	pipe, err := (tcp.Driver{}).Dial(ctx, address)
	if err != nil {
		return "", false, err
	}
	result, err := enrollment.Enroll(ctx, pipe, enrollmentState.DeviceIdentity(), options)
	_ = pipe.Close()
	if err != nil {
		return "", false, err
	}
	if err := enrollmentState.RecordObservation(result.ParentNodeID, result.ParentPublicKey, result.AuthorityNodeID, result.AuthorityPublicKey); err != nil {
		return "", false, err
	}
	if result.Outcome.Status == "granted" {
		if result.Outcome.Grant == nil {
			return "", false, errors.New("Enrollment result is granted without a Grant")
		}
		if err := enrollmentState.RecordGrant(*result.Outcome.Grant); err != nil {
			return "", false, err
		}
	}
	encoded, err := encodeJSON(result.Outcome)
	return encoded, result.Outcome.Status == "granted", err
}

func (c *Client) StartEnrolledTCP(endpoint string) error {
	c.mu.Lock()
	if c.closed || c.enrollment == nil {
		c.mu.Unlock()
		return errors.New("binding Enrollment client is closed")
	}
	snapshot := c.enrollment.Snapshot()
	c.mu.Unlock()
	if snapshot.Status != "enrolled" || snapshot.ParentNodeID == 0 {
		return errors.New("binding client has no granted Node ID; complete Enrollment first")
	}
	return c.start(tcp.Driver{}, endpoint, int64(snapshot.ParentNodeID), "")
}

func (c *Client) installEnrolledState() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.installEnrolledStateLocked()
}

func (c *Client) installEnrolledStateLocked() error {
	if c.enrollment == nil {
		return nil
	}
	identity, enrolled, err := c.enrollment.Identity()
	if err != nil || !enrolled {
		return err
	}
	snapshot := c.enrollment.Snapshot()
	trust := auth.NewTrustStore()
	if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
		return err
	}
	if err := trust.Add(snapshot.ParentNodeID, snapshot.ParentPublicKey); err != nil {
		return err
	}
	c.state = &auth.State{Identity: identity, Trust: trust}
	c.identity = PublicIdentity{NodeID: identity.NodeID, PublicKey: append(ed25519.PublicKey(nil), identity.PublicKey...)}
	return nil
}

func bindingPublicKey(label, value string) (ed25519.PublicKey, error) {
	decoded, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%s public key must be raw-base64 Ed25519", label)
	}
	return ed25519.PublicKey(decoded), nil
}

func bindingOptionalNodeID(value protocol.NodeID) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%d", value)
}
