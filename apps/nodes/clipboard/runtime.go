package clipboard

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"sync"

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
	State      *auth.State
	Node       *node.Node
	SDK        *sdk.Client
	Connection *sdk.Connection
	Clipboard  *Controller
	Sync       *SyncEngine

	cancel    context.CancelFunc
	closeOnce sync.Once
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
	state, err := auth.OpenState(config.StateDirectory, config.NodeID)
	if err != nil {
		return nil, err
	}
	if len(config.ParentKey) > 0 {
		if len(config.ParentKey) != ed25519.PublicKeySize {
			return nil, errors.New("clipboard runtime parent key must be Ed25519")
		}
		if err := state.Trust.Add(config.ParentID, config.ParentKey); err != nil {
			return nil, fmt.Errorf("trust clipboard parent: %w", err)
		}
	}
	if _, trusted := state.Trust.PublicKey(config.ParentID); !trusted {
		return nil, errors.New("clipboard runtime parent identity is not trusted")
	}
	runCtx, cancel := context.WithCancel(ctx)
	runtimeNode, err := node.New(runCtx, node.Config{
		Identity: state.Identity, Trust: state.Trust, Policy: state.Policy, JoinPermit: config.Permit,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	controller, err := Register(ControllerConfig{Node: runtimeNode, Store: state.Store, Adapter: config.Adapter})
	if err != nil {
		_ = runtimeNode.Close()
		cancel()
		return nil, err
	}
	client, err := sdk.NewClient(runtimeNode)
	if err != nil {
		_ = controller.Close()
		_ = runtimeNode.Close()
		cancel()
		return nil, err
	}
	connection, err := client.ConnectManaged(runCtx, config.Driver, config.Endpoint, config.ParentID, config.Supervisor)
	if err != nil {
		_ = controller.Close()
		_ = client.Close()
		cancel()
		return nil, err
	}
	syncEngine, err := StartSync(runCtx, client, connection, controller)
	if err != nil {
		connection.Stop()
		_ = controller.Close()
		_ = client.Close()
		cancel()
		return nil, err
	}
	return &Runtime{
		State: state, Node: runtimeNode, SDK: client, Connection: connection, Clipboard: controller, Sync: syncEngine, cancel: cancel,
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	var closeErr error
	r.closeOnce.Do(func() {
		if r.Sync != nil {
			r.Sync.Close()
		}
		if r.Connection != nil {
			r.Connection.Stop()
		}
		if r.cancel != nil {
			r.cancel()
		}
		if r.Clipboard != nil {
			closeErr = r.Clipboard.Close()
		}
		if r.SDK != nil {
			if err := r.SDK.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
	})
	return closeErr
}
