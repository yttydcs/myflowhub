package hub

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	filefeature "github.com/yttydcs/myflowhub/feature/file"
	flowfeature "github.com/yttydcs/myflowhub/feature/flow"
	"github.com/yttydcs/myflowhub/feature/management"
	"github.com/yttydcs/myflowhub/feature/notification"
	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/enrollment"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
)

type ListenerConfig struct {
	Driver   link.Driver
	Endpoint link.Endpoint
}

type Config struct {
	Node      node.Config
	Driver    link.Driver
	Endpoint  link.Endpoint
	Listeners []ListenerConfig
}

type PersistentConfig struct {
	StateDirectory              string
	NodeID                      protocol.NodeID
	Node                        node.Config
	Listeners                   []ListenerConfig
	RefreshInterval             time.Duration
	FileRoot                    string
	File                        filefeature.Config
	Flow                        flowfeature.Config
	AdmissionAuthorityNodeID    protocol.NodeID
	AdmissionAuthorityPublicKey ed25519.PublicKey
}

type Hub struct {
	Node                *node.Node
	Endpoint            link.Endpoint
	Endpoints           []link.Endpoint
	Runtime             *hostconfig.Runtime
	Management          *management.Controller
	Notification        *notification.Controller
	File                *filefeature.Controller
	Flow                *flowfeature.Controller
	EnrollmentAuthority *auth.EnrollmentAuthority

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func Start(ctx context.Context, config Config) (*Hub, error) {
	if ctx == nil {
		return nil, errors.New("hub context is required")
	}
	listeners, err := listenerConfigs(config)
	if err != nil {
		return nil, err
	}
	runtime, err := node.New(ctx, config.Node)
	if err != nil {
		return nil, err
	}
	result, err := listen(runtime, listeners)
	if err != nil {
		_ = runtime.Close()
		return nil, err
	}
	return &Hub{Node: runtime, Endpoint: result[0], Endpoints: result}, nil
}

func StartPersistent(ctx context.Context, config PersistentConfig) (*Hub, error) {
	if ctx == nil {
		return nil, errors.New("hub context is required")
	}
	if config.StateDirectory == "" {
		return nil, errors.New("hub state directory is required")
	}
	if err := config.NodeID.Validate(); err != nil {
		return nil, err
	}
	if len(config.Listeners) == 0 {
		return nil, errors.New("persistent hub requires at least one listener")
	}
	for index, listener := range config.Listeners {
		if err := validateListener(listener); err != nil {
			return nil, fmt.Errorf("listener %d: %w", index, err)
		}
	}
	state, err := hostconfig.Open(config.StateDirectory, config.NodeID)
	if err != nil {
		return nil, err
	}
	audit := management.NewAuditLog(nil, 256)
	var runtimeRef *node.Node
	var authority *auth.EnrollmentAuthority
	var broker enrollment.Broker
	var remoteBroker *enrollment.RemoteBroker
	authorityNodeID := config.AdmissionAuthorityNodeID
	if authorityNodeID == 0 || authorityNodeID == state.Identity.NodeID {
		authority, err = auth.LoadEnrollmentAuthority(state.Identity, state.Store, auth.EnrollmentAuthorityConfig{
			IsTargetDescendant: func(target, candidate protocol.NodeID) bool {
				return runtimeRef != nil && isDescendant(runtimeRef, target, candidate)
			},
		})
		if err != nil {
			return nil, err
		}
		authorityNodeID = state.Identity.NodeID
		broker = enrollment.LocalBroker{Authority: authority}
	} else {
		if err := authorityNodeID.Validate(); err != nil {
			return nil, fmt.Errorf("Admission Authority Node ID: %w", err)
		}
		if len(config.AdmissionAuthorityPublicKey) != ed25519.PublicKeySize {
			return nil, errors.New("remote Admission Authority requires an Ed25519 public key")
		}
		remoteBroker = &enrollment.RemoteBroker{
			AuthorityNodeID: authorityNodeID, AuthorityPublicKey: append(ed25519.PublicKey(nil), config.AdmissionAuthorityPublicKey...),
		}
		broker = remoteBroker
	}
	enrollmentServer, err := enrollment.NewServer(enrollment.ServerConfig{
		Parent: state.Identity, Trust: state.Trust, Broker: broker,
	})
	if err != nil {
		return nil, err
	}
	nodeConfig := config.Node
	nodeConfig.Identity = state.Identity
	nodeConfig.Trust = state.Trust
	nodeConfig.Admission = state.Admission
	nodeConfig.Policy = management.AuditPolicy(state.Policy, audit)
	if nodeConfig.Enrollment == nil {
		nodeConfig.Enrollment = enrollmentServer
	}
	runtime, err := node.New(ctx, nodeConfig)
	if err != nil {
		return nil, err
	}
	runtimeRef = runtime
	if remoteBroker != nil {
		remoteBroker.Node = runtime
	}
	controller, err := management.Register(management.Config{
		Node: runtime, Admission: state.Admission, EnrollmentAuthority: authority, AuthorityNodeID: authorityNodeID, Trust: state.Trust, Policy: state.Policy,
		Settings: state.Settings, RevokeNode: state.RevokeNode, Audit: audit,
	})
	if err != nil {
		_ = runtime.Close()
		return nil, err
	}
	notificationController, err := notification.Register(notification.Config{Node: runtime})
	if err != nil {
		_ = runtime.Close()
		return nil, err
	}
	fileConfig := config.File
	fileConfig.Node = runtime
	if fileConfig.Root == "" {
		fileConfig.Root = config.FileRoot
	}
	if fileConfig.Root == "" {
		fileConfig.Root = filepath.Join(config.StateDirectory, "files")
	}
	fileController, err := filefeature.Register(fileConfig)
	if err != nil {
		_ = runtime.Close()
		return nil, err
	}
	flowConfig := config.Flow
	flowConfig.Node = runtime
	flowConfig.Store = state.Store
	flowController, err := flowfeature.Register(flowConfig)
	if err != nil {
		_ = fileController.Close()
		_ = runtime.Close()
		return nil, err
	}
	endpoints, err := listen(runtime, config.Listeners)
	if err != nil {
		_ = flowController.Close()
		_ = fileController.Close()
		_ = runtime.Close()
		return nil, err
	}
	refreshInterval := config.RefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = time.Second
	}
	if refreshInterval < 10*time.Millisecond {
		_ = flowController.Close()
		_ = fileController.Close()
		_ = runtime.Close()
		return nil, errors.New("hub management refresh interval must be at least 10ms")
	}
	refreshCtx, cancel := context.WithCancel(ctx)
	hub := &Hub{
		Node: runtime, Endpoint: endpoints[0], Endpoints: endpoints, Runtime: state, Management: controller,
		Notification: notificationController, File: fileController, Flow: flowController, EnrollmentAuthority: authority, cancel: cancel,
	}
	hub.wg.Add(1)
	go func() {
		defer hub.wg.Done()
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = controller.Refresh()
			case <-refreshCtx.Done():
				return
			case <-runtime.Done():
				return
			}
		}
	}()
	return hub, nil
}

func isDescendant(runtime *node.Node, target, candidate protocol.NodeID) bool {
	if target == candidate {
		return true
	}
	parents := make(map[protocol.NodeID]protocol.NodeID)
	for _, relation := range runtime.Tree().Relations() {
		parents[relation.Node] = relation.Parent
	}
	visited := make(map[protocol.NodeID]struct{})
	for current := candidate; current != 0; current = parents[current] {
		if current == target {
			return true
		}
		if _, exists := visited[current]; exists {
			return false
		}
		visited[current] = struct{}{}
	}
	return false
}

func (h *Hub) Close() error {
	if h == nil || h.Node == nil {
		return nil
	}
	var closeErr error
	h.closeOnce.Do(func() {
		if h.cancel != nil {
			h.cancel()
		}
		h.wg.Wait()
		if h.Flow != nil {
			if err := h.Flow.Close(); err != nil {
				closeErr = err
			}
		}
		if h.File != nil {
			if err := h.File.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
		if err := h.Node.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	})
	return closeErr
}

func listenerConfigs(config Config) ([]ListenerConfig, error) {
	listeners := append([]ListenerConfig(nil), config.Listeners...)
	if len(listeners) == 0 {
		listeners = []ListenerConfig{{Driver: config.Driver, Endpoint: config.Endpoint}}
	}
	for index, listener := range listeners {
		if err := validateListener(listener); err != nil {
			return nil, fmt.Errorf("listener %d: %w", index, err)
		}
	}
	return listeners, nil
}

func validateListener(listener ListenerConfig) error {
	if err := link.ValidateDriver(listener.Driver); err != nil {
		return err
	}
	return listener.Endpoint.Validate()
}

func listen(runtime *node.Node, listeners []ListenerConfig) ([]link.Endpoint, error) {
	endpoints := make([]link.Endpoint, 0, len(listeners))
	for index, listener := range listeners {
		endpoint, err := runtime.Listen(listener.Driver, listener.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("start listener %d: %w", index, err)
		}
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, nil
}
