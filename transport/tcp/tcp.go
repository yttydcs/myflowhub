package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/yttydcs/myflowhub/runtime/link"
)

type Driver struct {
	Dialer       net.Dialer
	ListenConfig net.ListenConfig
}

func (d Driver) Dial(ctx context.Context, endpoint link.Endpoint) (link.Pipe, error) {
	if ctx == nil {
		return nil, errors.New("tcp dial context is required")
	}
	if err := endpoint.Validate(); err != nil {
		return nil, err
	}
	connection, err := d.Dialer.DialContext(ctx, "tcp", string(endpoint))
	if err != nil {
		return nil, fmt.Errorf("tcp dial %q: %w", endpoint, err)
	}
	return connection, nil
}

func (d Driver) Listen(ctx context.Context, endpoint link.Endpoint) (link.Listener, error) {
	if ctx == nil {
		return nil, errors.New("tcp listen context is required")
	}
	if err := endpoint.Validate(); err != nil {
		return nil, err
	}
	value, err := d.ListenConfig.Listen(ctx, "tcp", string(endpoint))
	if err != nil {
		return nil, fmt.Errorf("tcp listen %q: %w", endpoint, err)
	}
	return &listener{Listener: value}, nil
}

type listener struct{ net.Listener }

func (l *listener) Addr() link.Endpoint { return link.Endpoint(l.Listener.Addr().String()) }

func (l *listener) Accept(ctx context.Context) (link.Pipe, error) {
	if ctx == nil {
		return nil, errors.New("tcp accept context is required")
	}
	tcpListener, ok := l.Listener.(*net.TCPListener)
	if !ok {
		return nil, errors.New("tcp listener has unexpected implementation")
	}
	for {
		if err := tcpListener.SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			return nil, err
		}
		connection, err := tcpListener.AcceptTCP()
		if err == nil {
			_ = tcpListener.SetDeadline(time.Time{})
			return connection, nil
		}
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				continue
			}
		}
		return nil, fmt.Errorf("tcp accept: %w", err)
	}
}
