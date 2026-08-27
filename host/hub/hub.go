package hub

import (
	"context"
	"errors"

	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
)

type Config struct {
	Node     node.Config
	Driver   link.Driver
	Endpoint link.Endpoint
}

type Hub struct {
	Node     *node.Node
	Endpoint link.Endpoint
}

func Start(ctx context.Context, config Config) (*Hub, error) {
	if ctx == nil {
		return nil, errors.New("hub context is required")
	}
	if err := link.ValidateDriver(config.Driver); err != nil {
		return nil, err
	}
	runtime, err := node.New(ctx, config.Node)
	if err != nil {
		return nil, err
	}
	endpoint, err := runtime.Listen(config.Driver, config.Endpoint)
	if err != nil {
		_ = runtime.Close()
		return nil, err
	}
	return &Hub{Node: runtime, Endpoint: endpoint}, nil
}

func (h *Hub) Close() error {
	if h == nil || h.Node == nil {
		return nil
	}
	return h.Node.Close()
}
