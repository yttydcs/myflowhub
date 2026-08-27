package rfcomm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/yttydcs/myflowhub/runtime/link"
)

const DefaultMTU = 4096

var ErrUnsupported = errors.New("RFCOMM is not supported by the current platform provider")

type Options struct {
	UUID     string
	Channel  int
	Adapter  string
	Insecure bool
	Logger   *slog.Logger
}

func (o *Options) setDefaults() {
	if strings.TrimSpace(o.UUID) == "" {
		o.UUID = DefaultRFCOMMUUID
	}
	if strings.TrimSpace(o.Adapter) == "" {
		o.Adapter = "hci0"
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
}

func (o Options) Validate() error {
	if !isUUIDLike(o.UUID) {
		return ErrEndpointUUIDInvalid
	}
	if o.Channel != 0 && (o.Channel < 1 || o.Channel > 30) {
		return ErrEndpointChannelInvalid
	}
	if strings.TrimSpace(o.Adapter) == "" {
		return errors.New("RFCOMM adapter is required")
	}
	return nil
}

type DialOptions struct {
	BDAddr   string
	UUID     string
	Channel  int
	Adapter  string
	Insecure bool
}

func (o *DialOptions) setDefaults() {
	if strings.TrimSpace(o.UUID) == "" {
		o.UUID = DefaultRFCOMMUUID
	}
	if strings.TrimSpace(o.Adapter) == "" {
		o.Adapter = "hci0"
	}
}

func (o DialOptions) Validate() error {
	if _, err := normalizeBDAddr(o.BDAddr); err != nil {
		return fmt.Errorf("RFCOMM remote address: %w", err)
	}
	return (Options{UUID: o.UUID, Channel: o.Channel, Adapter: o.Adapter, Insecure: o.Insecure}).Validate()
}

type ProviderListener interface {
	Accept() (link.Pipe, net.Addr, net.Addr, error)
	Close() error
	Addr() net.Addr
}

type nativeListener = ProviderListener

type Provider interface {
	Dial(context.Context, DialOptions) (link.Pipe, error)
	Listen(Options) (ProviderListener, error)
}

type SystemProvider struct{}

func (SystemProvider) Dial(ctx context.Context, options DialOptions) (link.Pipe, error) {
	pipe, _, _, err := dialNative(ctx, options)
	return pipe, err
}

func (SystemProvider) Listen(options Options) (ProviderListener, error) {
	return listenNative(options)
}

type Driver struct {
	provider Provider
	mtu      int
}

func New(mtu int) (*Driver, error) { return NewWithProvider(SystemProvider{}, mtu) }

func NewWithProvider(provider Provider, mtu int) (*Driver, error) {
	if provider == nil {
		return nil, errors.New("RFCOMM provider is required")
	}
	if mtu == 0 {
		mtu = DefaultMTU
	}
	if mtu < 64 || mtu > protocolFrameLimit() {
		return nil, fmt.Errorf("RFCOMM MTU must be between 64 and %d", protocolFrameLimit())
	}
	return &Driver{provider: provider, mtu: mtu}, nil
}

func (d *Driver) Dial(ctx context.Context, endpoint link.Endpoint) (link.Pipe, error) {
	if ctx == nil {
		return nil, errors.New("RFCOMM dial context is required")
	}
	parsed, err := ParseEndpoint(string(endpoint))
	if err != nil {
		return nil, err
	}
	options := DialOptions{BDAddr: parsed.BDAddr, UUID: parsed.UUID, Channel: parsed.Channel, Adapter: parsed.Adapter, Insecure: parsed.Insecure}
	pipe, err := d.provider.Dial(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("dial RFCOMM: %w", err)
	}
	if pipe == nil {
		return nil, errors.New("dial RFCOMM: provider returned nil pipe")
	}
	return &chunkedPipe{Pipe: pipe, mtu: d.mtu}, nil
}

func (d *Driver) Listen(ctx context.Context, endpoint link.Endpoint) (link.Listener, error) {
	if ctx == nil {
		return nil, errors.New("RFCOMM listen context is required")
	}
	options, err := parseListenEndpoint(string(endpoint))
	if err != nil {
		return nil, err
	}
	native, err := d.provider.Listen(options)
	if err != nil {
		return nil, fmt.Errorf("listen RFCOMM: %w", err)
	}
	if native == nil {
		return nil, errors.New("listen RFCOMM: provider returned nil listener")
	}
	value := &listener{native: native, endpoint: endpoint, mtu: d.mtu, done: make(chan struct{})}
	go func() {
		select {
		case <-ctx.Done():
			_ = value.Close()
		case <-value.done:
		}
	}()
	return value, nil
}

type acceptResult struct {
	pipe link.Pipe
	err  error
}

type listener struct {
	native   ProviderListener
	endpoint link.Endpoint
	mtu      int
	done     chan struct{}
	once     sync.Once
}

func (l *listener) Accept(ctx context.Context) (link.Pipe, error) {
	if ctx == nil {
		return nil, errors.New("RFCOMM accept context is required")
	}
	result := make(chan acceptResult, 1)
	go func() {
		pipe, _, _, err := l.native.Accept()
		select {
		case result <- acceptResult{pipe: pipe, err: err}:
		case <-ctx.Done():
			if pipe != nil {
				_ = pipe.Close()
			}
		case <-l.done:
			if pipe != nil {
				_ = pipe.Close()
			}
		}
	}()
	select {
	case accepted := <-result:
		if accepted.err != nil {
			return nil, accepted.err
		}
		if accepted.pipe == nil {
			return nil, errors.New("RFCOMM provider accepted a nil pipe")
		}
		return &chunkedPipe{Pipe: accepted.pipe, mtu: l.mtu}, nil
	case <-ctx.Done():
		_ = l.Close()
		return nil, ctx.Err()
	case <-l.done:
		return nil, link.ErrClosed
	}
}

func (l *listener) Addr() link.Endpoint { return l.endpoint }

func (l *listener) Close() error {
	var closeErr error
	l.once.Do(func() {
		close(l.done)
		closeErr = l.native.Close()
	})
	return closeErr
}

type chunkedPipe struct {
	link.Pipe
	mtu int
}

func (p *chunkedPipe) Write(data []byte) (int, error) {
	total := 0
	for total < len(data) {
		end := total + p.mtu
		if end > len(data) {
			end = len(data)
		}
		n, err := p.Pipe.Write(data[total:end])
		if n < 0 || n > end-total {
			return total, errors.New("RFCOMM provider returned an invalid write count")
		}
		total += n
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, io.ErrShortWrite
		}
	}
	return total, nil
}

func parseListenEndpoint(raw string) (Options, error) {
	raw = strings.TrimSpace(raw)
	prefix := EndpointSchemeRFCOMM + "://"
	if len(raw) < len(prefix) || strings.ToLower(raw[:len(prefix)]) != prefix {
		return Options{}, ErrEndpointSchemeInvalid
	}
	parts := strings.SplitN(raw[len(prefix):], "?", 2)
	query := url.Values{}
	var err error
	if len(parts) == 2 {
		query, err = url.ParseQuery(parts[1])
		if err != nil {
			return Options{}, fmt.Errorf("RFCOMM listen query: %w", err)
		}
	}
	options := Options{UUID: strings.ToLower(strings.TrimSpace(query.Get("uuid"))), Adapter: strings.TrimSpace(query.Get("adapter"))}
	options.setDefaults()
	if channel := strings.TrimSpace(query.Get("channel")); channel != "" {
		options.Channel, err = strconv.Atoi(channel)
		if err != nil {
			return Options{}, ErrEndpointChannelInvalid
		}
	}
	secure := true
	if rawSecure := strings.TrimSpace(query.Get("secure")); rawSecure != "" {
		secure, err = parseBool(rawSecure)
		if err != nil {
			return Options{}, err
		}
	}
	options.Insecure = !secure
	return options, options.Validate()
}

func protocolFrameLimit() int { return 1 << 20 }
