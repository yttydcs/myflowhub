package link

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
)

type SessionState uint8

const (
	StateUnauthenticated SessionState = iota + 1
	StateActive
	StateClosing
	StateClosed
)

type Role uint8

const (
	RoleUnknown Role = iota
	RoleParent
	RoleChild
)

type SessionConfig struct {
	LocalNode         protocol.NodeID
	Codec             protocol.Codec
	ControlQueue      int
	DataQueue         int
	InboundQueue      int
	ControlQueueBytes int64
	DataQueueBytes    int64
	InboundQueueBytes int64
	HeartbeatInterval time.Duration
	IdleTimeout       time.Duration
	// IdleAfterPeerHeartbeat preserves peers that predate periodic heartbeats.
	// Once an authenticated peer sends a heartbeat, idle monitoring stays enabled
	// for this session; subsequent silence never switches it back off.
	IdleAfterPeerHeartbeat bool
}

func (c SessionConfig) normalized() (SessionConfig, error) {
	if err := c.LocalNode.Validate(); err != nil {
		return SessionConfig{}, fmt.Errorf("link session local node: %w", err)
	}
	if c.ControlQueue <= 0 {
		c.ControlQueue = 32
	}
	if c.DataQueue <= 0 {
		c.DataQueue = 128
	}
	if c.InboundQueue <= 0 {
		c.InboundQueue = 128
	}
	if c.ControlQueueBytes < 0 || c.DataQueueBytes < 0 || c.InboundQueueBytes < 0 {
		return SessionConfig{}, errors.New("link session queue byte budgets cannot be negative")
	}
	if c.HeartbeatInterval < 0 || c.IdleTimeout < 0 {
		return SessionConfig{}, errors.New("link session durations cannot be negative")
	}
	if c.HeartbeatInterval > 0 && c.IdleTimeout > 0 && c.IdleTimeout <= c.HeartbeatInterval {
		return SessionConfig{}, errors.New("idle timeout must exceed heartbeat interval")
	}
	return c, nil
}

type Session struct {
	config SessionConfig
	pipe   Pipe

	ctx    context.Context
	cancel context.CancelFunc

	mu    sync.RWMutex
	state SessionState
	peer  protocol.NodeID
	role  Role
	epoch uint64
	err   error

	control *envelopeQueue
	data    *envelopeQueue
	input   *envelopeQueue
	inbound chan protocol.Envelope
	done    chan struct{}
	once    sync.Once
	wg      sync.WaitGroup

	lastRead      atomic.Int64
	peerHeartbeat atomic.Bool
}

func NewSession(parent context.Context, pipe Pipe, config SessionConfig) (*Session, error) {
	if parent == nil {
		return nil, errors.New("link session context is required")
	}
	if pipe == nil {
		return nil, errors.New("link session pipe is required")
	}
	normalized, err := config.normalized()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	session := &Session{
		config:  normalized,
		pipe:    pipe,
		ctx:     ctx,
		cancel:  cancel,
		state:   StateUnauthenticated,
		control: newEnvelopeQueue(normalized.ControlQueue, normalized.ControlQueueBytes),
		data:    newEnvelopeQueue(normalized.DataQueue, normalized.DataQueueBytes),
		input:   newEnvelopeQueue(normalized.InboundQueue, normalized.InboundQueueBytes),
		inbound: make(chan protocol.Envelope),
		done:    make(chan struct{}),
	}
	session.lastRead.Store(time.Now().UnixNano())
	session.wg.Add(4)
	go session.readLoop()
	go session.dispatchLoop()
	go session.writeLoop()
	go session.watchdog()
	go func() {
		session.wg.Wait()
		session.mu.Lock()
		session.state = StateClosed
		session.mu.Unlock()
		close(session.inbound)
		close(session.done)
	}()
	return session, nil
}

func (s *Session) Activate(peer protocol.NodeID, role Role, epoch uint64) error {
	if err := peer.Validate(); err != nil {
		return fmt.Errorf("activate peer: %w", err)
	}
	if role != RoleParent && role != RoleChild {
		return errors.New("activate link: parent or child role is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateUnauthenticated {
		return fmt.Errorf("activate link: state is %d", s.state)
	}
	s.peer = peer
	s.role = role
	s.epoch = epoch
	s.state = StateActive
	return nil
}

func (s *Session) State() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Session) Peer() protocol.NodeID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.peer
}

func (s *Session) Role() Role {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.role
}

func (s *Session) Epoch() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.epoch
}

func (s *Session) Inbound() <-chan protocol.Envelope { return s.inbound }
func (s *Session) Done() <-chan struct{}             { return s.done }

func (s *Session) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

// IdleMonitoringActive distinguishes a legacy peer from one whose periodic
// heartbeats have enabled the configured liveness deadline.
func (s *Session) IdleMonitoringActive() bool {
	return s.config.IdleTimeout > 0 && (!s.config.IdleAfterPeerHeartbeat || s.peerHeartbeat.Load())
}

func (s *Session) Send(ctx context.Context, envelope protocol.Envelope) error {
	if ctx == nil {
		return errors.New("send context is required")
	}
	if err := envelope.Validate(s.config.Codec.MaxPayload); err != nil {
		return err
	}
	state := s.State()
	if state == StateUnauthenticated && envelope.Operation != protocol.OperationJoin && envelope.Operation != protocol.OperationJoinAck {
		return errors.New("send: unauthenticated session only accepts join frames")
	}
	if state == StateClosing || state == StateClosed {
		return ErrClosed
	}
	queue := s.control
	if isData(envelope) {
		queue = s.data
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ctx.Done():
		return ErrClosed
	default:
	}
	if !queue.push(envelope) {
		return ErrQueueFull
	}
	return nil
}

func (s *Session) Close() error {
	s.closeWithError(nil)
	<-s.done
	return nil
}

func (s *Session) closeWithError(err error) {
	s.once.Do(func() {
		s.mu.Lock()
		if err != nil {
			s.err = err
		}
		s.state = StateClosing
		s.mu.Unlock()
		s.cancel()
		_ = s.pipe.Close()
	})
}

func (s *Session) readLoop() {
	defer s.wg.Done()
	for {
		envelope, err := s.config.Codec.Decode(s.pipe)
		if err != nil {
			if s.ctx.Err() == nil {
				s.closeWithError(fmt.Errorf("read link frame: %w", err))
			}
			return
		}
		s.lastRead.Store(time.Now().UnixNano())
		if s.State() == StateUnauthenticated && envelope.Operation != protocol.OperationJoin && envelope.Operation != protocol.OperationJoinAck {
			s.closeWithError(errors.New("read link frame: operation received before authentication"))
			return
		}
		if envelope.Operation == protocol.OperationHeartbeat && s.State() == StateActive &&
			envelope.Source == s.Peer() && envelope.Target == s.config.LocalNode && envelope.TopologyEpoch == s.Epoch() {
			if first := !s.peerHeartbeat.Swap(true); first {
				if err := s.SendHeartbeat(s.ctx); err != nil {
					s.closeWithError(fmt.Errorf("reply to first heartbeat: %w", err))
					return
				}
			}
		}
		if !s.input.push(envelope) {
			s.closeWithError(fmt.Errorf("read link frame: %w", ErrQueueFull))
			return
		}
	}
}

func (s *Session) dispatchLoop() {
	defer s.wg.Done()
	for {
		select {
		case envelope := <-s.input.frames:
			select {
			case s.inbound <- envelope:
				s.input.release(envelope)
			case <-s.ctx.Done():
				s.input.release(envelope)
				return
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Session) writeLoop() {
	defer s.wg.Done()
	var heartbeat <-chan time.Time
	var ticker *time.Ticker
	if s.config.HeartbeatInterval > 0 {
		ticker = time.NewTicker(s.config.HeartbeatInterval)
		heartbeat = ticker.C
		defer ticker.Stop()
	}
	const maxControlBurst = 8
	controlBudget := maxControlBurst
	for {
		if controlBudget == 0 {
			select {
			case envelope := <-s.data.frames:
				if !s.writeQueued(s.data, envelope) {
					return
				}
				controlBudget = maxControlBurst
				continue
			default:
				controlBudget = maxControlBurst
			}
		}
		select {
		case envelope := <-s.control.frames:
			if !s.writeQueued(s.control, envelope) {
				return
			}
			controlBudget--
			continue
		default:
		}
		select {
		case envelope := <-s.control.frames:
			if !s.writeQueued(s.control, envelope) {
				return
			}
			controlBudget--
		case envelope := <-s.data.frames:
			if !s.writeQueued(s.data, envelope) {
				return
			}
			controlBudget = maxControlBurst
		case <-heartbeat:
			if s.State() == StateActive {
				envelope, err := s.heartbeat()
				if err != nil || !s.write(envelope) {
					if err != nil {
						s.closeWithError(err)
					}
					return
				}
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Session) writeQueued(queue *envelopeQueue, envelope protocol.Envelope) bool {
	defer queue.release(envelope)
	return s.write(envelope)
}

func (s *Session) write(envelope protocol.Envelope) bool {
	if err := s.config.Codec.Encode(s.pipe, envelope); err != nil {
		if s.ctx.Err() == nil {
			s.closeWithError(fmt.Errorf("write link frame: %w", err))
		}
		return false
	}
	return true
}

func (s *Session) heartbeat() (protocol.Envelope, error) {
	id, err := protocol.NewMessageID()
	if err != nil {
		return protocol.Envelope{}, err
	}
	return protocol.Envelope{
		Version:       protocol.CurrentVersion,
		Phase:         protocol.PhaseEvent,
		Operation:     protocol.OperationHeartbeat,
		MessageID:     id,
		Source:        s.config.LocalNode,
		Target:        s.Peer(),
		TopologyEpoch: s.Epoch(),
	}, nil
}

// SendHeartbeat starts liveness discovery after the authenticated child has
// received its join acknowledgement. Legacy configurations remain silent.
func (s *Session) SendHeartbeat(ctx context.Context) error {
	if s.config.HeartbeatInterval == 0 {
		return nil
	}
	envelope, err := s.heartbeat()
	if err != nil {
		return err
	}
	return s.Send(ctx, envelope)
}

func (s *Session) watchdog() {
	defer s.wg.Done()
	if s.config.IdleTimeout == 0 {
		<-s.ctx.Done()
		_ = s.pipe.Close()
		return
	}
	interval := s.config.IdleTimeout / 2
	if interval <= 0 {
		interval = time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if !s.IdleMonitoringActive() {
				continue
			}
			if time.Since(time.Unix(0, s.lastRead.Load())) > s.config.IdleTimeout {
				s.closeWithError(ErrIdle)
				return
			}
		case <-s.ctx.Done():
			_ = s.pipe.Close()
			return
		}
	}
}

func isData(envelope protocol.Envelope) bool {
	return envelope.Operation == protocol.OperationResourceEvent || envelope.Operation == protocol.OperationResourceGap || envelope.Operation == protocol.OperationSessionData
}
