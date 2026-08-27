package sdk

import (
	"context"
	"errors"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
)

type Client struct {
	runtime *node.Node
}

func NewClient(runtime *node.Node) (*Client, error) {
	if runtime == nil {
		return nil, errors.New("SDK client requires a node runtime")
	}
	return &Client{runtime: runtime}, nil
}

func (c *Client) Connect(ctx context.Context, driver link.Driver, endpoint link.Endpoint, parent protocol.NodeID) error {
	return c.runtime.ConnectParent(ctx, driver, endpoint, parent)
}

func (c *Client) Subscribe(ctx context.Context, resource protocol.ResourceID, lease time.Duration, queue int) (*node.RemoteSubscription, error) {
	return c.runtime.Subscribe(ctx, resource, lease, queue)
}

func (c *Client) Invoke(ctx context.Context, resource protocol.ResourceID, input []byte) ([]byte, error) {
	return c.runtime.Invoke(ctx, resource, input)
}

func (c *Client) Close() error { return c.runtime.Close() }
