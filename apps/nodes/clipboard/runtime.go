package clipboard

import (
	"context"
	"crypto/ed25519"
	"errors"
	"sync"

	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type RuntimeConfig struct {
	StateDirectory string
	NodeID         protocol.NodeID
	ParentID       protocol.NodeID
	ParentKey      ed25519.PublicKey
	Permit         *protocol.ProvisioningPermitV1
	Driver         link.Driver
	Endpoint       link.Endpoint
	Adapter        Adapter
	Supervisor     node.SupervisorConfig
}

type Runtime struct {
	Host       *nodehost.Host
	State      *auth.State
	Node       *node.Node
	SDK        *sdk.Client
	Connection sdk.ConnectionStatus
	Clipboard  *Controller
	Sync       *SyncEngine

	cancel    context.CancelFunc
	closeOnce sync.Once
	closeErr  error
}

func Start(ctx context.Context, config RuntimeConfig) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("clipboard runtime context is required")
	}
	if config.StateDirectory == "" {
		return nil, errors.New("clipboard runtime state directory is required")
	}
	if err := config.NodeID.Validate(); err != nil {
		return nil, err
	}
	if err := config.ParentID.Validate(); err != nil {
		return nil, err
	}
	if config.Driver == nil {
		config.Driver = tcp.Driver{}
	}
	if err := link.ValidateDriver(config.Driver); err != nil {
		return nil, err
	}
	if err := config.Endpoint.Validate(); err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	host, err := nodehost.New(runCtx, nodehost.Config{
		StateDirectory: config.StateDirectory, NodeID: config.NodeID,
		Parent: &nodehost.ParentConfig{
			NodeID: config.ParentID, PublicKey: append(ed25519.PublicKey(nil), config.ParentKey...),
			Permit: config.Permit, Driver: config.Driver, Endpoint: config.Endpoint, Supervisor: config.Supervisor,
		},
	})
	if err != nil {
		cancel()
		return nil, err
	}
	value := &Runtime{Host: host, State: host.State(), Node: host.Node(), SDK: host.Client(), cancel: cancel}
	controller, err := Register(ControllerConfig{Node: host.Node(), Store: host.State().Store, Adapter: config.Adapter})
	if err != nil {
		return nil, errors.Join(err, value.Close())
	}
	value.Clipboard = controller
	connection, ok := host.ParentStatus()
	if !ok {
		return nil, errors.Join(errors.New("clipboard runtime parent status is unavailable"), value.Close())
	}
	value.Connection = connection
	if err := host.Start(); err != nil {
		return nil, errors.Join(err, value.Close())
	}
	syncEngine, err := StartSync(runCtx, value.SDK, connection, controller)
	if err != nil {
		return nil, errors.Join(err, value.Close())
	}
	value.Sync = syncEngine
	return value, nil
}

func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		if r.Sync != nil {
			r.Sync.Close()
		}
		if r.Clipboard != nil {
			r.closeErr = r.Clipboard.Close()
		}
		if r.cancel != nil {
			r.cancel()
		}
		if r.Host != nil {
			r.closeErr = errors.Join(r.closeErr, r.Host.Close())
		}
	})
	return r.closeErr
}
