package metrics

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type RuntimeConfig struct {
	StateDirectory       string
	NodeID               protocol.NodeID
	ParentID             protocol.NodeID
	ParentKey            ed25519.PublicKey
	Permit               *protocol.ProvisioningPermitV1
	Driver               link.Driver
	Endpoint             link.Endpoint
	Platform             string
	Collector            Collector
	Actuator             Actuator
	Logger               *slog.Logger
	Supervisor           node.SupervisorConfig
	NotificationCapacity int
}

type Runtime struct {
	State         *auth.State
	Node          *node.Node
	SDK           *sdk.Client
	Connection    *sdk.Connection
	Metrics       *Controller
	Notifications *NotificationInbox

	cancel    context.CancelFunc
	closeOnce sync.Once
}

func Start(ctx context.Context, config RuntimeConfig) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("metrics runtime context is required")
	}
	if config.StateDirectory == "" {
		return nil, errors.New("metrics runtime state directory is required")
	}
	if err := config.NodeID.Validate(); err != nil {
		return nil, err
	}
	if err := config.ParentID.Validate(); err != nil {
		return nil, err
	}
	if !validPlatform(config.Platform) {
		return nil, errors.New("metrics runtime platform is invalid")
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
			return nil, errors.New("metrics runtime parent key must be Ed25519")
		}
		if err := state.Trust.Add(config.ParentID, config.ParentKey); err != nil {
			return nil, fmt.Errorf("trust metrics parent: %w", err)
		}
	}
	if _, trusted := state.Trust.PublicKey(config.ParentID); !trusted {
		return nil, errors.New("metrics runtime parent identity is not trusted")
	}
	runCtx, cancel := context.WithCancel(ctx)
	runtime, err := node.New(runCtx, node.Config{
		Identity: state.Identity, Trust: state.Trust, Policy: state.Policy, JoinPermit: config.Permit,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	controller, err := Register(ControllerConfig{
		Node: runtime, Store: state.Store, Platform: config.Platform, Collector: config.Collector,
		Actuator: config.Actuator, Logger: config.Logger,
	})
	if err != nil {
		_ = runtime.Close()
		cancel()
		return nil, err
	}
	client, err := sdk.NewClient(runtime)
	if err != nil {
		_ = controller.Close()
		_ = runtime.Close()
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
	notifications, err := startNotificationInbox(runCtx, client, connection, config.ParentID, controller, config.NotificationCapacity)
	if err != nil {
		connection.Stop()
		_ = controller.Close()
		_ = client.Close()
		cancel()
		return nil, err
	}
	return &Runtime{
		State: state, Node: runtime, SDK: client, Connection: connection, Metrics: controller, Notifications: notifications, cancel: cancel,
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	var closeErr error
	r.closeOnce.Do(func() {
		if r.Notifications != nil {
			r.Notifications.Close()
		}
		if r.Connection != nil {
			r.Connection.Stop()
		}
		if r.cancel != nil {
			r.cancel()
		}
		if r.Metrics != nil {
			closeErr = r.Metrics.Close()
		}
		if r.SDK != nil {
			if err := r.SDK.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
	})
	return closeErr
}
