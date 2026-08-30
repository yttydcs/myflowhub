// Package nodehost composes persistent auth state, one node runtime, an
// attached SDK client, and optional topology links into a single lifecycle.
package nodehost

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

var (
	ErrAlreadyStarted = errors.New("node host has already started")
	ErrNotStarted     = errors.New("node host has not started")
	ErrStartFailed    = errors.New("node host cannot restart after a failed start")
	ErrClosed         = errors.New("node host is closed")
)

type LifecycleState string

const (
	LifecycleNew      LifecycleState = "new"
	LifecycleStarting LifecycleState = "starting"
	LifecycleRunning  LifecycleState = "running"
	LifecycleFailed   LifecycleState = "failed"
	LifecycleStopping LifecycleState = "stopping"
	LifecycleStopped  LifecycleState = "stopped"
)

type Role string

const (
	RoleOffline Role = "offline"
	RoleLeaf    Role = "leaf"
	RoleRelay   Role = "relay"
	RoleRoot    Role = "root"
)

type Status struct {
	Lifecycle LifecycleState
	Role      Role
	Endpoints []link.Endpoint
	Parent    node.ConnectionSnapshot
	HasParent bool
}

type Host struct {
	config Config
	state  *auth.State
	node   *node.Node
	client *sdk.Client
	role   Role

	runCtx context.Context
	cancel context.CancelFunc

	lifecycleMu sync.Mutex
	lifecycle   LifecycleState
	listeners   []*managedListener
	endpoints   []link.Endpoint
	parent      *node.ParentSupervisor

	closeOnce sync.Once
	closeDone chan struct{}
	closeErr  error

	releaseStateDirectory func()
}

func New(ctx context.Context, config Config) (*Host, error) {
	if ctx == nil {
		return nil, errors.New("node host context is required")
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	config = cloneConfig(config)
	release, err := reserveStateDirectory(config.StateDirectory)
	if err != nil {
		return nil, err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			release()
		}
	}()

	var state *auth.State
	if config.IdentityStore == nil {
		state, err = auth.OpenState(config.StateDirectory, config.NodeID)
	} else {
		state, err = auth.OpenStateWithIdentityStore(config.StateDirectory, config.NodeID, config.IdentityStore)
	}
	if err != nil {
		return nil, fmt.Errorf("open node host state: %w", err)
	}
	if config.Parent != nil {
		if err := validateParentState(*config.Parent, state); err != nil {
			return nil, fmt.Errorf("validate node host parent: %w", err)
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	var permit *protocol.ProvisioningPermitV1
	if config.Parent != nil {
		permit = config.Parent.Permit
	}
	runtime, err := node.New(runCtx, config.Runtime.nodeConfig(state, permit))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create node host runtime: %w", err)
	}
	client, err := sdk.NewAttachedClient(runtime)
	if err != nil {
		_ = runtime.Close()
		cancel()
		return nil, fmt.Errorf("create node host client: %w", err)
	}
	host := &Host{
		config: config, state: state, node: runtime, client: client, role: roleFor(config),
		runCtx: runCtx, cancel: cancel, lifecycle: LifecycleNew,
		closeDone: make(chan struct{}), releaseStateDirectory: release,
	}
	succeeded = true
	return host, nil
}

func (h *Host) Start() error {
	if h == nil {
		return ErrClosed
	}
	h.lifecycleMu.Lock()
	defer h.lifecycleMu.Unlock()
	switch h.lifecycle {
	case LifecycleNew:
		h.lifecycle = LifecycleStarting
	case LifecycleStarting, LifecycleRunning:
		return ErrAlreadyStarted
	case LifecycleFailed:
		return ErrStartFailed
	case LifecycleStopping, LifecycleStopped:
		return ErrClosed
	default:
		return fmt.Errorf("node host has invalid lifecycle state %q", h.lifecycle)
	}

	for index, config := range h.config.Listeners {
		if err := h.runCtx.Err(); err != nil {
			return h.failStart(fmt.Errorf("start node host listener %d (%q): host is stopping: %w", index, config.Endpoint, err))
		}
		listener, endpoint, err := h.startListener(config)
		if err != nil {
			return h.failStart(fmt.Errorf("start node host listener %d (%q): %w", index, config.Endpoint, err))
		}
		h.listeners = append(h.listeners, listener)
		h.endpoints = append(h.endpoints, endpoint)
	}
	if err := h.runCtx.Err(); err != nil {
		return h.failStart(fmt.Errorf("start node host: host is stopping: %w", err))
	}
	if h.config.Parent != nil {
		config := h.config.Parent
		supervisor, err := h.node.SuperviseParent(h.runCtx, config.Driver, config.Endpoint, config.NodeID, config.Supervisor)
		if err != nil {
			return h.failStart(fmt.Errorf("start node host parent %d (%q): %w", config.NodeID, config.Endpoint, err))
		}
		h.parent = supervisor
	}
	h.lifecycle = LifecycleRunning
	return nil
}

func (h *Host) Close() error {
	if h == nil {
		return nil
	}
	h.closeOnce.Do(func() {
		// Cancellation is intentionally signalled before taking lifecycleMu so a
		// concurrent Start blocked in Driver.Listen can unwind. The ordered
		// ownership cleanup still happens below while lifecycle transitions are
		// serialized.
		h.cancel()
		h.lifecycleMu.Lock()
		if h.lifecycle != LifecycleFailed && h.lifecycle != LifecycleStopped {
			h.lifecycle = LifecycleStopping
		}
		var failures []error
		if h.parent != nil {
			h.parent.Stop()
			h.parent = nil
		}
		failures = append(failures, closeListenersReverse(h.listeners)...)
		h.listeners = nil
		h.endpoints = nil
		h.cancel()
		if err := h.node.Close(); err != nil {
			failures = append(failures, fmt.Errorf("close node host runtime: %w", err))
		}
		h.lifecycle = LifecycleStopped
		h.closeErr = errors.Join(failures...)
		h.releaseStateDirectory()
		h.lifecycleMu.Unlock()
		close(h.closeDone)
	})
	<-h.closeDone
	return h.closeErr
}

func (h *Host) Node() *node.Node {
	if h == nil {
		return nil
	}
	return h.node
}

func (h *Host) Client() *sdk.Client {
	if h == nil {
		return nil
	}
	return h.client
}

func (h *Host) Resources() *resource.Registry {
	if h == nil || h.node == nil {
		return nil
	}
	return h.node.Registry()
}

func (h *Host) State() *auth.State {
	if h == nil {
		return nil
	}
	return h.state
}

func (h *Host) ID() protocol.NodeID {
	if h == nil || h.node == nil {
		return 0
	}
	return h.node.ID()
}

// PublicKey returns a copy of the Host identity's public key without exposing
// its private key to operation or binding facades.
func (h *Host) PublicKey() ed25519.PublicKey {
	if h == nil || h.state == nil {
		return nil
	}
	return append(ed25519.PublicKey(nil), h.state.Identity.PublicKey...)
}

func (h *Host) Role() Role {
	if h == nil {
		return ""
	}
	return h.role
}

func (h *Host) Endpoints() []link.Endpoint {
	if h == nil {
		return nil
	}
	h.lifecycleMu.Lock()
	defer h.lifecycleMu.Unlock()
	return append([]link.Endpoint(nil), h.endpoints...)
}

func (h *Host) Parent() (node.ConnectionSnapshot, bool) {
	if h == nil {
		return node.ConnectionSnapshot{}, false
	}
	h.lifecycleMu.Lock()
	supervisor := h.parent
	configured := h.config.Parent
	lifecycle := h.lifecycle
	h.lifecycleMu.Unlock()
	if supervisor != nil {
		return supervisor.Snapshot(), true
	}
	if configured == nil {
		return node.ConnectionSnapshot{}, false
	}
	state := node.ConnectionDisconnected
	if lifecycle == LifecycleFailed || lifecycle == LifecycleStopping || lifecycle == LifecycleStopped {
		state = node.ConnectionStopped
	}
	return node.ConnectionSnapshot{State: state, Parent: configured.NodeID, Endpoint: configured.Endpoint}, true
}

// ParentStatus returns a non-owning SDK view of this Host's configured parent
// connection. The view remains stable across reconnects and never owns the
// ParentSupervisor. Before Start, Snapshot reports disconnected and WaitChange
// returns ErrNotStarted; callers must start the Host before waiting.
func (h *Host) ParentStatus() (sdk.ConnectionStatus, bool) {
	if h == nil || h.config.Parent == nil {
		return nil, false
	}
	return parentStatus{host: h}, true
}

func (h *Host) WaitParentChange(ctx context.Context, after uint64) (node.ConnectionSnapshot, error) {
	if ctx == nil {
		return node.ConnectionSnapshot{}, errors.New("wait for node host parent context is required")
	}
	if h == nil {
		return node.ConnectionSnapshot{}, ErrClosed
	}
	h.lifecycleMu.Lock()
	supervisor := h.parent
	lifecycle := h.lifecycle
	h.lifecycleMu.Unlock()
	if supervisor == nil {
		if lifecycle == LifecycleNew || lifecycle == LifecycleStarting {
			return node.ConnectionSnapshot{}, ErrNotStarted
		}
		return node.ConnectionSnapshot{}, ErrClosed
	}
	return supervisor.WaitChange(ctx, after)
}

type parentStatus struct {
	host *Host
}

func (s parentStatus) Snapshot() sdk.ConnectionSnapshot {
	value, _ := s.host.Parent()
	return sdk.ConnectionSnapshotFromRuntime(value)
}

func (s parentStatus) WaitChange(ctx context.Context, after uint64) (sdk.ConnectionSnapshot, error) {
	value, err := s.host.WaitParentChange(ctx, after)
	if err != nil {
		return sdk.ConnectionSnapshot{}, err
	}
	return sdk.ConnectionSnapshotFromRuntime(value), nil
}

func (h *Host) Status() Status {
	if h == nil {
		return Status{Lifecycle: LifecycleStopped}
	}
	h.lifecycleMu.Lock()
	status := Status{
		Lifecycle: h.lifecycle,
		Role:      h.role,
		Endpoints: append([]link.Endpoint(nil), h.endpoints...),
		HasParent: h.config.Parent != nil,
	}
	if h.parent != nil {
		status.Parent = h.parent.Snapshot()
	} else if h.config.Parent != nil {
		status.Parent = node.ConnectionSnapshot{State: node.ConnectionDisconnected, Parent: h.config.Parent.NodeID, Endpoint: h.config.Parent.Endpoint}
	}
	h.lifecycleMu.Unlock()
	return status
}

func (h *Host) failStart(startErr error) error {
	var failures []error
	if h.parent != nil {
		h.parent.Stop()
		h.parent = nil
	}
	failures = append(failures, closeListenersReverse(h.listeners)...)
	h.listeners = nil
	h.endpoints = nil
	h.cancel()
	if err := h.node.Close(); err != nil {
		failures = append(failures, fmt.Errorf("rollback node host runtime: %w", err))
	}
	h.lifecycle = LifecycleFailed
	h.releaseStateDirectory()
	return errors.Join(append([]error{startErr}, failures...)...)
}

func roleFor(config Config) Role {
	hasParent := config.Parent != nil
	hasListeners := len(config.Listeners) != 0
	switch {
	case hasParent && hasListeners:
		return RoleRelay
	case hasParent:
		return RoleLeaf
	case hasListeners:
		return RoleRoot
	default:
		return RoleOffline
	}
}

var stateDirectories = struct {
	sync.Mutex
	active map[string]struct{}
}{active: make(map[string]struct{})}

func reserveStateDirectory(directory string) (func(), error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve node host state directory: %w", err)
	}
	key := filepath.Clean(absolute)
	stateDirectories.Lock()
	if _, exists := stateDirectories.active[key]; exists {
		stateDirectories.Unlock()
		return nil, fmt.Errorf("node host state directory %q is already in use", key)
	}
	stateDirectories.active[key] = struct{}{}
	stateDirectories.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			stateDirectories.Lock()
			delete(stateDirectories.active, key)
			stateDirectories.Unlock()
		})
	}, nil
}
