package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

var ErrReparented = errors.New("parent supervisor stopped after reparent")

type ConnectionState uint8

const (
	ConnectionDisconnected ConnectionState = iota + 1
	ConnectionConnecting
	ConnectionConnected
	ConnectionFailed
	ConnectionStopped
)

type ConnectionSnapshot struct {
	State          ConnectionState
	Parent         protocol.NodeID
	Endpoint       link.Endpoint
	Attempt        uint64
	LinkGeneration uint64
	LastError      string
	NextRetry      time.Time
	Generation     uint64
}

type SupervisorConfig struct {
	MinBackoff time.Duration
	MaxBackoff time.Duration
	Multiplier float64
	Sleep      func(context.Context, time.Duration) error
}

// ValidateSupervisorConfig validates the effective parent reconnection policy
// without starting a supervisor or creating external work.
func ValidateSupervisorConfig(config SupervisorConfig) error {
	_, err := config.normalized()
	return err
}

func (c SupervisorConfig) normalized() (SupervisorConfig, error) {
	if c.MinBackoff <= 0 {
		c.MinBackoff = 100 * time.Millisecond
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = 30 * time.Second
	}
	if c.Multiplier == 0 {
		c.Multiplier = 2
	}
	if c.MinBackoff > c.MaxBackoff || c.Multiplier < 1 || math.IsNaN(c.Multiplier) || math.IsInf(c.Multiplier, 0) {
		return SupervisorConfig{}, errors.New("parent supervisor backoff is invalid")
	}
	if c.Sleep == nil {
		c.Sleep = sleepContext
	}
	return c, nil
}

type ParentSupervisor struct {
	node     *Node
	driver   link.Driver
	endpoint link.Endpoint
	parent   protocol.NodeID
	config   SupervisorConfig
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}

	mu      sync.Mutex
	current ConnectionSnapshot
	changed chan struct{}
}

func (n *Node) SuperviseParent(ctx context.Context, driver link.Driver, endpoint link.Endpoint, parent protocol.NodeID, config SupervisorConfig) (*ParentSupervisor, error) {
	if ctx == nil {
		return nil, errors.New("parent supervisor context is required")
	}
	if err := link.ValidateDriver(driver); err != nil {
		return nil, err
	}
	if err := endpoint.Validate(); err != nil {
		return nil, err
	}
	if err := parent.Validate(); err != nil {
		return nil, err
	}
	normalized, err := config.normalized()
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	value := &ParentSupervisor{
		node: n, driver: driver, endpoint: endpoint, parent: parent, config: normalized,
		ctx: runCtx, cancel: cancel, done: make(chan struct{}), changed: make(chan struct{}),
		current: ConnectionSnapshot{State: ConnectionDisconnected, Parent: parent, Endpoint: endpoint, Generation: 1},
	}
	go value.run()
	return value, nil
}

func (s *ParentSupervisor) Snapshot() ConnectionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *ParentSupervisor) WaitChange(ctx context.Context, after uint64) (ConnectionSnapshot, error) {
	if ctx == nil {
		return ConnectionSnapshot{}, errors.New("wait context is required")
	}
	s.mu.Lock()
	if s.current.Generation > after {
		current := s.current
		s.mu.Unlock()
		return current, nil
	}
	changed := s.changed
	s.mu.Unlock()
	select {
	case <-changed:
		return s.Snapshot(), nil
	case <-ctx.Done():
		return ConnectionSnapshot{}, ctx.Err()
	}
}

func (s *ParentSupervisor) Done() <-chan struct{} { return s.done }

func (s *ParentSupervisor) Stop() {
	if s != nil {
		s.cancel()
		<-s.done
	}
}

func (s *ParentSupervisor) run() {
	defer close(s.done)
	backoff := s.config.MinBackoff
	var attempt uint64
	for {
		if s.ctx.Err() != nil || s.node.ctx.Err() != nil {
			s.update(ConnectionSnapshot{State: ConnectionStopped, Parent: s.parent, Endpoint: s.endpoint, Attempt: attempt})
			return
		}
		attempt++
		s.update(ConnectionSnapshot{State: ConnectionConnecting, Parent: s.parent, Endpoint: s.endpoint, Attempt: attempt})
		err := s.node.ConnectParent(s.ctx, s.driver, s.endpoint, s.parent)
		if err != nil {
			if s.ctx.Err() != nil || s.node.ctx.Err() != nil {
				continue
			}
			if classifyConnectionError(err) == errorPermanent {
				s.update(ConnectionSnapshot{State: ConnectionFailed, Parent: s.parent, Endpoint: s.endpoint, Attempt: attempt, LastError: err.Error()})
				return
			}
			nextRetry := time.Now().Add(backoff)
			s.update(ConnectionSnapshot{State: ConnectionDisconnected, Parent: s.parent, Endpoint: s.endpoint, Attempt: attempt, LastError: err.Error(), NextRetry: nextRetry})
			if err := s.config.Sleep(s.ctx, backoff); err != nil {
				continue
			}
			backoff = nextBackoff(backoff, s.config.MaxBackoff, s.config.Multiplier)
			continue
		}
		status, generation := s.node.ParentStatus()
		if !status.Connected || status.Parent != s.parent {
			s.update(ConnectionSnapshot{State: ConnectionFailed, Parent: s.parent, Endpoint: s.endpoint, Attempt: attempt, LastError: ErrReparented.Error()})
			return
		}
		backoff = s.config.MinBackoff
		s.update(ConnectionSnapshot{State: ConnectionConnected, Parent: s.parent, Endpoint: s.endpoint, Attempt: attempt, LinkGeneration: status.LinkGeneration})
		if _, err := s.node.WaitParentChange(s.ctx, generation); err != nil {
			continue
		}
		current, _ := s.node.ParentStatus()
		if current.Connected && current.Parent != s.parent {
			s.update(ConnectionSnapshot{State: ConnectionFailed, Parent: s.parent, Endpoint: s.endpoint, Attempt: attempt, LastError: ErrReparented.Error()})
			return
		}
		s.update(ConnectionSnapshot{State: ConnectionDisconnected, Parent: s.parent, Endpoint: s.endpoint, Attempt: attempt, LastError: "parent link disconnected"})
	}
}

func (s *ParentSupervisor) update(next ConnectionSnapshot) {
	s.mu.Lock()
	next.Generation = s.current.Generation + 1
	s.current = next
	close(s.changed)
	s.changed = make(chan struct{})
	s.mu.Unlock()
}

type ParentStatus struct {
	Connected      bool
	Parent         protocol.NodeID
	Epoch          uint64
	LinkGeneration uint64
}

func (n *Node) ParentStatus() (ParentStatus, uint64) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, current := range n.sessions {
		if current.role == link.RoleParent && current.session.State() == link.StateActive {
			return ParentStatus{Connected: true, Parent: current.peer, Epoch: current.epoch, LinkGeneration: n.parentGeneration}, n.parentGeneration
		}
	}
	return ParentStatus{}, n.parentGeneration
}

func (n *Node) WaitParentChange(ctx context.Context, after uint64) (ParentStatus, error) {
	n.mu.RLock()
	if n.parentGeneration > after {
		n.mu.RUnlock()
		status, _ := n.ParentStatus()
		return status, nil
	}
	changed := n.parentChanged
	n.mu.RUnlock()
	select {
	case <-changed:
		status, _ := n.ParentStatus()
		return status, nil
	case <-ctx.Done():
		return ParentStatus{}, ctx.Err()
	case <-n.ctx.Done():
		return ParentStatus{}, errors.New("node closed")
	}
}

func (n *Node) signalParentChange() {
	n.mu.Lock()
	n.parentGeneration++
	close(n.parentChanged)
	n.parentChanged = make(chan struct{})
	n.mu.Unlock()
}

type connectionErrorClass uint8

const (
	errorTemporary connectionErrorClass = iota + 1
	errorPermanent
)

func classifyConnectionError(err error) connectionErrorClass {
	if errors.Is(err, auth.ErrUntrustedIdentity) || errors.Is(err, auth.ErrPermitInvalid) || errors.Is(err, auth.ErrPermitConsumed) || errors.Is(err, auth.ErrPermitRevoked) || errors.Is(err, auth.ErrForbidden) || errors.Is(err, protocol.ErrInvalidEnvelope) || errors.Is(err, tree.ErrCycle) {
		return errorPermanent
	}
	var networkError net.Error
	if errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary()) {
		return errorTemporary
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, link.ErrClosed) || errors.Is(err, link.ErrIdle) || errors.Is(err, context.DeadlineExceeded) {
		return errorTemporary
	}
	return errorTemporary
}

func nextBackoff(current, maximum time.Duration, multiplier float64) time.Duration {
	if current >= maximum {
		return maximum
	}
	next := time.Duration(float64(current) * multiplier)
	if next <= current || next > maximum {
		return maximum
	}
	return next
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("parent supervisor sleep: %w", ctx.Err())
	}
}
