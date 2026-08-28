package androidbinding

import (
	"errors"

	"github.com/yttydcs/myflowhub/sdk/bindings"
)

type Client struct {
	core *bindings.Client
}

type Listener interface {
	OnEvent(eventJSON string)
	OnError(errorJSON string)
}

func NewClient(stateDirectory string, nodeID int64) (*Client, error) {
	core, err := bindings.NewClient(stateDirectory, nodeID)
	if err != nil {
		return nil, err
	}
	return &Client{core: core}, nil
}

func (c *Client) IdentityJSON() (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.IdentityJSON()
}

func (c *Client) TrustParent(parentID int64, rawPublicKey string) error {
	core, err := c.current()
	if err != nil {
		return err
	}
	return core.TrustParent(parentID, rawPublicKey)
}

func (c *Client) StartTCP(endpoint string, parentID int64, permitJSON string) error {
	core, err := c.current()
	if err != nil {
		return err
	}
	return core.StartTCP(endpoint, parentID, permitJSON)
}

func (c *Client) StartRFCOMM(endpoint string, parentID int64, permitJSON string) error {
	core, err := c.current()
	if err != nil {
		return err
	}
	return core.StartRFCOMM(endpoint, parentID, permitJSON)
}

func (c *Client) StatusJSON() (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.StatusJSON()
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
	if c != nil && c.core != nil {
		c.core.CancelSubscription(subscriptionID)
	}
}

func (c *Client) Close() error {
	if c == nil || c.core == nil {
		return nil
	}
	core := c.core
	c.core = nil
	return core.Close()
}

func (c *Client) current() (*bindings.Client, error) {
	if c == nil || c.core == nil {
		return nil, errors.New("Android client is closed")
	}
	return c.core, nil
}
