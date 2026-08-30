package androidbinding

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/sdk/bindings"
	"github.com/yttydcs/myflowhub/transport/rfcomm"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

// Client is the gomobile operation facade for an Android leaf node. It owns
// exactly one NodeHost after StartTCP or StartRFCOMM; its bindings.Client is
// attached to that Host and cannot create or close another node or parent.
type Client struct {
	mu        sync.Mutex
	directory string
	nodeID    protocol.NodeID
	state     *auth.State
	leaf      *androidLeafRuntime
	closed    bool
}

type Listener interface {
	OnEvent(eventJSON string)
	OnError(errorJSON string)
}

func NewClient(stateDirectory string, nodeID int64) (*Client, error) {
	if strings.TrimSpace(stateDirectory) == "" {
		return nil, errors.New("Android client state directory is required")
	}
	id, err := androidClientNodeID(nodeID)
	if err != nil {
		return nil, err
	}
	state, err := auth.OpenState(stateDirectory, id)
	if err != nil {
		return nil, fmt.Errorf("open Android client state: %w", err)
	}
	return &Client{directory: stateDirectory, nodeID: id, state: state}, nil
}

func (c *Client) IdentityJSON() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.state == nil {
		return "", errors.New("Android client is closed")
	}
	return androidClientJSON(map[string]any{
		"node_id":    fmt.Sprintf("%d", c.state.Identity.NodeID),
		"public_key": base64.RawStdEncoding.EncodeToString(c.state.Identity.PublicKey),
	})
}

func (c *Client) TrustParent(parentID int64, rawPublicKey string) error {
	parent, err := androidClientNodeID(parentID)
	if err != nil {
		return err
	}
	key, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(rawPublicKey))
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("parent public key must be a raw-base64 Ed25519 key")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.state == nil {
		return errors.New("Android client is closed")
	}
	if c.leaf != nil {
		return errors.New("parent trust cannot change after the Android client starts")
	}
	if err := c.state.Trust.Add(parent, ed25519.PublicKey(key)); err != nil {
		return fmt.Errorf("trust Android client parent: %w", err)
	}
	return nil
}

func (c *Client) StartTCP(endpoint string, parentID int64, permitJSON string) error {
	return c.start(tcp.Driver{}, endpoint, parentID, permitJSON)
}

func (c *Client) StartRFCOMM(endpoint string, parentID int64, permitJSON string) error {
	driver, err := rfcomm.New(0)
	if err != nil {
		return err
	}
	return c.start(driver, endpoint, parentID, permitJSON)
}

func (c *Client) start(driver link.Driver, endpoint string, parentID int64, permitJSON string) error {
	parent, err := androidClientNodeID(parentID)
	if err != nil {
		return err
	}
	address := link.Endpoint(strings.TrimSpace(endpoint))
	if err := address.Validate(); err != nil {
		return err
	}
	var permit *protocol.ProvisioningPermitV1
	if strings.TrimSpace(permitJSON) != "" {
		var value protocol.ProvisioningPermitV1
		if err := protocol.DecodeJSONPayload([]byte(permitJSON), protocol.DefaultMaxPayload, &value); err != nil {
			return fmt.Errorf("decode Android client provisioning permit: %w", err)
		}
		permit = &value
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.state == nil {
		return errors.New("Android client is closed")
	}
	if c.leaf != nil {
		return errors.New("Android client is already started")
	}
	parentKey, trusted := c.state.Trust.PublicKey(parent)
	if !trusted {
		return errors.New("parent identity is not trusted")
	}
	leaf, err := newAndroidLeafRuntime(androidLeafConfig{
		StateDirectory: c.directory, NodeID: c.nodeID, ParentID: parent,
		ParentKey: parentKey, Permit: permit, Driver: driver, Endpoint: address,
	})
	if err != nil {
		return err
	}
	c.leaf = leaf
	return nil
}

func (c *Client) StatusJSON() (string, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return "", errors.New("Android client is closed")
	}
	leaf := c.leaf
	c.mu.Unlock()
	if leaf == nil {
		return androidClientJSON(map[string]any{"state": "disconnected"})
	}
	return leaf.facade.StatusJSON()
}

func (c *Client) WaitConnected(timeoutMS int64) error {
	core, err := c.current()
	if err != nil {
		return err
	}
	return core.WaitConnected(timeoutMS)
}

func (c *Client) CatalogJSON(ownerID, timeoutMS int64) (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.CatalogJSON(ownerID, timeoutMS)
}

func (c *Client) SnapshotJSON(ownerID int64, name string, timeoutMS int64) (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.SnapshotJSON(ownerID, name, timeoutMS)
}

func (c *Client) InvokeJSON(ownerID int64, name, requestJSON string, timeoutMS int64) (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.InvokeJSON(ownerID, name, requestJSON, timeoutMS)
}

func (c *Client) OperateJSON(ownerID int64, name, capability, schema, requestJSON string, timeoutMS int64) (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.OperateJSON(ownerID, name, capability, schema, requestJSON, timeoutMS)
}

func (c *Client) UploadFile(ownerID int64, sourcePath, destination, contentType string, timeoutMS int64) (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.UploadFile(ownerID, sourcePath, destination, contentType, timeoutMS)
}

func (c *Client) Subscribe(ownerID int64, name string, leaseMS int64, listener Listener) (int64, error) {
	return c.SubscribeCapability(ownerID, name, "subscribe", leaseMS, listener)
}

func (c *Client) SubscribeCapability(ownerID int64, name, capability string, leaseMS int64, listener Listener) (int64, error) {
	core, err := c.current()
	if err != nil {
		return 0, err
	}
	if listener == nil {
		return 0, errors.New("Android subscription listener is required")
	}
	return core.SubscribeCapability(ownerID, name, capability, leaseMS, listener)
}

func (c *Client) CancelSubscription(subscriptionID int64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	leaf := c.leaf
	c.mu.Unlock()
	if leaf != nil {
		leaf.facade.CancelSubscription(subscriptionID)
	}
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	leaf := c.leaf
	c.leaf = nil
	c.state = nil
	c.mu.Unlock()
	if leaf != nil {
		return leaf.Close()
	}
	return nil
}

func (c *Client) current() (*bindings.Client, error) {
	if c == nil {
		return nil, errors.New("Android client is closed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errors.New("Android client is closed")
	}
	if c.leaf == nil {
		return nil, errors.New("Android client is not started")
	}
	return c.leaf.facade, nil
}

func androidClientNodeID(value int64) (protocol.NodeID, error) {
	if value <= 0 {
		return 0, errors.New("Android NodeID must be positive")
	}
	id := protocol.NodeID(value)
	return id, id.Validate()
}

func androidClientJSON(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode Android client JSON: %w", err)
	}
	return string(payload), nil
}
