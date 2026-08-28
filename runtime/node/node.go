package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

type Config struct {
	Identity            auth.Identity
	Trust               *auth.TrustStore
	Policy              auth.Policy
	Admission           *auth.Admission
	JoinPermit          *protocol.ProvisioningPermitV1
	Session             link.SessionConfig
	Subscriptions       subscription.Config
	Commands            command.Config
	JoinTimeout         time.Duration
	MaxHandshakes       int
	MaxPending          int
	MaxResourceSessions int
}

type peerSession struct {
	peer    protocol.NodeID
	role    link.Role
	epoch   uint64
	linkID  string
	session *link.Session
}

type Node struct {
	ctx    context.Context
	cancel context.CancelFunc
	config Config

	identity      auth.Identity
	trust         *auth.TrustStore
	policy        auth.Policy
	admission     *auth.Admission
	tree          *tree.State
	registry      *resource.Registry
	subscriptions *subscription.Manager
	commands      *command.Dispatcher

	mu                     sync.RWMutex
	sessions               map[protocol.NodeID]*peerSession
	listeners              []link.Listener
	pending                map[protocol.MessageID]*pendingEntry
	forwardedSubscriptions map[protocol.MessageID]forwardedSubscription
	resourceSessions       map[protocol.MessageID]*ownedResourceSession
	closed                 bool
	parentGeneration       uint64
	parentChanged          chan struct{}

	handshakes        chan struct{}
	diagnostics       chan error
	wg                sync.WaitGroup
	closeOnce         sync.Once
	policyWatchCancel func()
}

type OperationalStats struct {
	ActiveLinks            int
	ActiveSubscriptions    int
	ActiveResourceSessions int
}

func New(parent context.Context, config Config) (*Node, error) {
	if parent == nil {
		return nil, errors.New("node context is required")
	}
	if err := config.Identity.Validate(); err != nil {
		return nil, fmt.Errorf("node identity: %w", err)
	}
	if config.Trust == nil {
		return nil, errors.New("node trust store is required")
	}
	if config.Policy == nil {
		config.Policy = auth.NewStaticPolicy()
	}
	if config.JoinTimeout <= 0 {
		config.JoinTimeout = 5 * time.Second
	}
	if config.MaxHandshakes <= 0 {
		config.MaxHandshakes = 32
	}
	if config.MaxPending <= 0 {
		config.MaxPending = 1024
	}
	if config.MaxResourceSessions <= 0 {
		config.MaxResourceSessions = 256
	}
	if config.JoinTimeout < 10*time.Millisecond || config.MaxHandshakes < 1 || config.MaxPending < 1 || config.MaxResourceSessions < 1 {
		return nil, errors.New("node limits are invalid")
	}
	state, err := tree.New(config.Identity.NodeID)
	if err != nil {
		return nil, err
	}
	registry, err := resource.NewRegistry(config.Identity.NodeID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	manager, err := subscription.NewManager(ctx, registry, config.Subscriptions)
	if err != nil {
		cancel()
		return nil, err
	}
	config.Commands.Authorizer = func(ctx context.Context, call command.Call) error {
		return config.Policy.Authorize(ctx, auth.Request{Subject: call.Source, Action: auth.Action(call.Capability), Capability: call.Capability, Resource: call.Resource})
	}
	dispatcher, err := command.NewDispatcher(registry, config.Commands)
	if err != nil {
		_ = manager.Close()
		cancel()
		return nil, err
	}
	config.Session.LocalNode = config.Identity.NodeID
	value := &Node{
		ctx: ctx, cancel: cancel, config: config, identity: config.Identity, trust: config.Trust, policy: config.Policy, admission: config.Admission,
		tree: state, registry: registry, subscriptions: manager, commands: dispatcher,
		sessions: make(map[protocol.NodeID]*peerSession), pending: make(map[protocol.MessageID]*pendingEntry),
		forwardedSubscriptions: make(map[protocol.MessageID]forwardedSubscription),
		resourceSessions:       make(map[protocol.MessageID]*ownedResourceSession),
		handshakes:             make(chan struct{}, config.MaxHandshakes), diagnostics: make(chan error, 64),
		parentChanged: make(chan struct{}),
	}
	if generated, ok := config.Policy.(interface {
		WatchGeneration(func(uint64)) (uint64, func(), error)
	}); ok {
		_, stop, err := generated.WatchGeneration(func(generation uint64) {
			manager.CleanupPolicyGeneration(generation)
			value.expireForwardedSubscriptions(generation)
			value.cleanupResourceSessionsPolicy(generation)
		})
		if err != nil {
			_ = manager.Close()
			cancel()
			return nil, fmt.Errorf("watch policy generation: %w", err)
		}
		value.policyWatchCancel = stop
	}
	return value, nil
}

func (n *Node) ID() protocol.NodeID          { return n.identity.NodeID }
func (n *Node) Registry() *resource.Registry { return n.registry }
func (n *Node) Tree() *tree.State            { return n.tree }
func (n *Node) Errors() <-chan error         { return n.diagnostics }
func (n *Node) Done() <-chan struct{}        { return n.ctx.Done() }

func (n *Node) Stats() OperationalStats {
	n.mu.RLock()
	links := len(n.sessions)
	forwarded := len(n.forwardedSubscriptions)
	resourceSessions := len(n.resourceSessions)
	n.mu.RUnlock()
	return OperationalStats{ActiveLinks: links, ActiveSubscriptions: n.subscriptions.Count() + forwarded, ActiveResourceSessions: resourceSessions}
}

func (n *Node) DisconnectPeer(peer protocol.NodeID) error {
	if err := peer.Validate(); err != nil {
		return err
	}
	n.mu.RLock()
	current := n.sessions[peer]
	n.mu.RUnlock()
	if current == nil {
		return nil
	}
	return current.session.Close()
}

func (n *Node) emit(err error) {
	if err == nil {
		return
	}
	select {
	case n.diagnostics <- err:
	default:
	}
}

func (n *Node) Listen(driver link.Driver, endpoint link.Endpoint) (link.Endpoint, error) {
	if err := link.ValidateDriver(driver); err != nil {
		return "", err
	}
	listener, err := driver.Listen(n.ctx, endpoint)
	if err != nil {
		return "", err
	}
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		_ = listener.Close()
		return "", errors.New("node is closed")
	}
	n.listeners = append(n.listeners, listener)
	n.wg.Add(1)
	n.mu.Unlock()
	go n.acceptLoop(listener)
	return listener.Addr(), nil
}

func (n *Node) acceptLoop(listener link.Listener) {
	defer n.wg.Done()
	for {
		pipe, err := listener.Accept(n.ctx)
		if err != nil {
			if n.ctx.Err() == nil && !errors.Is(err, link.ErrClosed) {
				n.emit(fmt.Errorf("accept child link: %w", err))
			}
			return
		}
		select {
		case n.handshakes <- struct{}{}:
			if !n.launch(func() {
				defer func() { <-n.handshakes }()
				if err := n.acceptChild(pipe); err != nil && n.ctx.Err() == nil {
					n.emit(err)
				}
			}) {
				<-n.handshakes
				_ = pipe.Close()
				return
			}
		default:
			_ = pipe.Close()
			n.emit(errors.New("reject child link: handshake limit reached"))
		}
	}
}

func (n *Node) launch(run func()) bool {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return false
	}
	n.wg.Add(1)
	n.mu.Unlock()
	go func() {
		defer n.wg.Done()
		run()
	}()
	return true
}

func (n *Node) Close() error {
	n.closeOnce.Do(func() {
		n.mu.Lock()
		n.closed = true
		listeners := append([]link.Listener(nil), n.listeners...)
		sessions := make([]*link.Session, 0, len(n.sessions))
		for _, current := range n.sessions {
			sessions = append(sessions, current.session)
		}
		pending := make([]*pendingEntry, 0, len(n.pending))
		for _, current := range n.pending {
			pending = append(pending, current)
		}
		n.pending = make(map[protocol.MessageID]*pendingEntry)
		n.forwardedSubscriptions = make(map[protocol.MessageID]forwardedSubscription)
		resourceSessions := make([]*ownedResourceSession, 0, len(n.resourceSessions))
		for _, current := range n.resourceSessions {
			resourceSessions = append(resourceSessions, current)
		}
		n.resourceSessions = make(map[protocol.MessageID]*ownedResourceSession)
		n.mu.Unlock()
		n.cancel()
		if n.policyWatchCancel != nil {
			n.policyWatchCancel()
		}
		for _, listener := range listeners {
			_ = listener.Close()
		}
		for _, session := range sessions {
			_ = session.Close()
		}
		for _, current := range pending {
			current.fail(errors.New("node closed"))
		}
		for _, current := range resourceSessions {
			current.session.Abort(errors.New("node closed"))
		}
		_ = n.subscriptions.Close()
		n.wg.Wait()
		close(n.diagnostics)
	})
	return nil
}

func (n *Node) registerPending(id protocol.MessageID, queue int) (*pendingEntry, error) {
	value := newPending(queue)
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return nil, errors.New("node is closed")
	}
	if len(n.pending) >= n.config.MaxPending {
		return nil, errors.New("node pending request limit reached")
	}
	if _, exists := n.pending[id]; exists {
		return nil, errors.New("pending request ID already exists")
	}
	n.pending[id] = value
	return value, nil
}

func (n *Node) unregisterPending(id protocol.MessageID, failure error) {
	n.mu.Lock()
	value, ok := n.pending[id]
	if ok {
		delete(n.pending, id)
	}
	n.mu.Unlock()
	if ok {
		value.fail(failure)
	}
}

func (n *Node) deliverPending(envelope protocol.Envelope) bool {
	if envelope.CorrelationID.IsZero() {
		return false
	}
	n.mu.RLock()
	value := n.pending[envelope.CorrelationID]
	n.mu.RUnlock()
	if value == nil {
		return false
	}
	value.deliver(envelope)
	return true
}
