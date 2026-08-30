package metrics

import (
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
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
	Host          *nodehost.Host
	State         *auth.State
	Node          *node.Node
	SDK           *sdk.Client
	Connection    sdk.ConnectionStatus
	Metrics       *Controller
	Notifications *NotificationInbox

	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

func Start(ctx context.Context, config RuntimeConfig) (*Runtime, error) {
	runtime, err := prepareRuntime(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := runtime.Host.Start(); err != nil {
		return nil, errors.Join(err, runtime.Close())
	}
	notifications, err := startNotificationInbox(
		runtime.ctx, runtime.SDK, runtime.Connection, config.ParentID, runtime.Metrics, config.NotificationCapacity,
	)
	if err != nil {
		return nil, errors.Join(err, runtime.Close())
	}
	runtime.Notifications = notifications
	return runtime, nil
}

func prepareRuntime(ctx context.Context, config RuntimeConfig) (*Runtime, error) {
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
	if err := validateNotificationCapacity(config.NotificationCapacity); err != nil {
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
		StateDirectory: config.StateDirectory,
		NodeID:         config.NodeID,
		Parent: &nodehost.ParentConfig{
			NodeID:     config.ParentID,
			PublicKey:  append(ed25519.PublicKey(nil), config.ParentKey...),
			Permit:     config.Permit,
			Driver:     config.Driver,
			Endpoint:   config.Endpoint,
			Supervisor: config.Supervisor,
		},
	})
	if err != nil {
		cancel()
		return nil, err
	}
	runtime := &Runtime{
		Host: host, State: host.State(), Node: host.Node(), SDK: host.Client(),
		ctx: runCtx, cancel: cancel,
	}
	controller, err := Register(ControllerConfig{
		Node: host.Node(), Resources: host.Resources(), Store: host.State().Store, Platform: config.Platform, Collector: config.Collector,
		Actuator: config.Actuator, Logger: config.Logger,
	})
	if err != nil {
		return nil, errors.Join(err, runtime.Close())
	}
	runtime.Metrics = controller
	connection, ok := host.ParentStatus()
	if !ok {
		return nil, errors.Join(errors.New("metrics runtime parent status is unavailable"), runtime.Close())
	}
	runtime.Connection = connection
	return runtime, nil
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
		if r.Metrics != nil {
			closeErr = r.Metrics.Close()
		}
		if r.cancel != nil {
			r.cancel()
		}
		if r.Host != nil {
			closeErr = errors.Join(closeErr, r.Host.Close())
		}
	})
	return closeErr
}
