package memory

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/yttydcs/myflowhub/runtime/link"
)

type Network struct {
	mu        sync.Mutex
	listeners map[link.Endpoint]*listener
	closed    bool
}

func NewNetwork() *Network {
	return &Network{listeners: make(map[link.Endpoint]*listener)}
}

func (n *Network) Dial(ctx context.Context, endpoint link.Endpoint) (link.Pipe, error) {
	if ctx == nil {
		return nil, errors.New("memory dial context is required")
	}
	if err := endpoint.Validate(); err != nil {
		return nil, err
	}
	n.mu.Lock()
	current := n.listeners[endpoint]
	closed := n.closed
	n.mu.Unlock()
	if closed {
		return nil, link.ErrClosed
	}
	if current == nil {
		return nil, fmt.Errorf("memory endpoint %q is not listening", endpoint)
	}
	client, server := net.Pipe()
	select {
	case current.pending <- server:
		return client, nil
	case <-ctx.Done():
		_ = client.Close()
		_ = server.Close()
		return nil, ctx.Err()
	case <-current.done:
		_ = client.Close()
		_ = server.Close()
		return nil, link.ErrClosed
	}
}

func (n *Network) Listen(ctx context.Context, endpoint link.Endpoint) (link.Listener, error) {
	if ctx == nil {
		return nil, errors.New("memory listen context is required")
	}
	if err := endpoint.Validate(); err != nil {
		return nil, err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return nil, link.ErrClosed
	}
	if _, exists := n.listeners[endpoint]; exists {
		return nil, fmt.Errorf("memory endpoint %q already listening", endpoint)
	}
	value := &listener{network: n, endpoint: endpoint, pending: make(chan link.Pipe), done: make(chan struct{})}
	n.listeners[endpoint] = value
	go func() {
		select {
		case <-ctx.Done():
			_ = value.Close()
		case <-value.done:
		}
	}()
	return value, nil
}

func (n *Network) Close() error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return nil
	}
	n.closed = true
	listeners := make([]*listener, 0, len(n.listeners))
	for _, value := range n.listeners {
		listeners = append(listeners, value)
	}
	n.mu.Unlock()
	for _, value := range listeners {
		_ = value.Close()
	}
	return nil
}

type listener struct {
	network  *Network
	endpoint link.Endpoint
	pending  chan link.Pipe
	done     chan struct{}
	once     sync.Once
}

func (l *listener) Accept(ctx context.Context) (link.Pipe, error) {
	if ctx == nil {
		return nil, errors.New("memory accept context is required")
	}
	select {
	case pipe := <-l.pending:
		return pipe, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.done:
		return nil, link.ErrClosed
	}
}

func (l *listener) Addr() link.Endpoint { return l.endpoint }

func (l *listener) Close() error {
	l.once.Do(func() {
		l.network.mu.Lock()
		if l.network.listeners[l.endpoint] == l {
			delete(l.network.listeners, l.endpoint)
		}
		l.network.mu.Unlock()
		close(l.done)
	})
	return nil
}
