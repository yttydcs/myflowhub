package quic

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	quicgo "github.com/quic-go/quic-go"
	"github.com/yttydcs/myflowhub/runtime/link"
)

const (
	EndpointScheme = "quic"
	DefaultALPN    = "mfh/3"
)

var streamPreface = []byte{'M', 'F', 'H', '3', 'Q'}

type Config struct {
	ServerTLS           *tls.Config
	ClientTLS           *tls.Config
	QUIC                *quicgo.Config
	AcceptStreamTimeout time.Duration
}

type Driver struct{ config Config }

func New(config Config) (*Driver, error) {
	if config.ServerTLS == nil && config.ClientTLS == nil {
		return nil, errors.New("QUIC requires a server or client TLS configuration")
	}
	if config.AcceptStreamTimeout <= 0 {
		config.AcceptStreamTimeout = 10 * time.Second
	}
	if config.AcceptStreamTimeout < time.Millisecond {
		return nil, errors.New("QUIC stream accept timeout is too small")
	}
	if config.ServerTLS != nil {
		config.ServerTLS = normalizedTLS(config.ServerTLS)
		if len(config.ServerTLS.Certificates) == 0 && config.ServerTLS.GetCertificate == nil {
			return nil, errors.New("QUIC server TLS requires a certificate")
		}
	}
	if config.ClientTLS != nil {
		config.ClientTLS = normalizedTLS(config.ClientTLS)
	}
	if config.QUIC == nil {
		config.QUIC = &quicgo.Config{KeepAlivePeriod: 15 * time.Second, MaxIdleTimeout: 60 * time.Second}
	} else {
		config.QUIC = config.QUIC.Clone()
	}
	return &Driver{config: config}, nil
}

func normalizedTLS(source *tls.Config) *tls.Config {
	result := source.Clone()
	if result.MinVersion == 0 || result.MinVersion < tls.VersionTLS13 {
		result.MinVersion = tls.VersionTLS13
	}
	if len(result.NextProtos) == 0 {
		result.NextProtos = []string{DefaultALPN}
	}
	return result
}

func (d *Driver) Dial(ctx context.Context, endpoint link.Endpoint) (link.Pipe, error) {
	if ctx == nil {
		return nil, errors.New("QUIC dial context is required")
	}
	if d.config.ClientTLS == nil {
		return nil, errors.New("QUIC driver has no client TLS configuration")
	}
	parsed, err := parseEndpoint(endpoint, false)
	if err != nil {
		return nil, err
	}
	tlsConfig := d.config.ClientTLS.Clone()
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = parsed.serverName
	}
	if !tlsConfig.InsecureSkipVerify && tlsConfig.ServerName == "" {
		return nil, errors.New("QUIC verified TLS requires server_name for an IP or wildcard endpoint")
	}
	connection, err := quicgo.DialAddr(ctx, parsed.address, tlsConfig, d.config.QUIC.Clone())
	if err != nil {
		return nil, fmt.Errorf("dial QUIC: %w", err)
	}
	stream, err := connection.OpenStreamSync(ctx)
	if err != nil {
		_ = connection.CloseWithError(0, "open stream failed")
		return nil, fmt.Errorf("open QUIC stream: %w", err)
	}
	if _, err := stream.Write(streamPreface); err != nil {
		stream.CancelWrite(0)
		_ = connection.CloseWithError(0, "stream preface failed")
		return nil, fmt.Errorf("write QUIC stream preface: %w", err)
	}
	return &pipe{connection: connection, stream: stream}, nil
}

func (d *Driver) Listen(ctx context.Context, endpoint link.Endpoint) (link.Listener, error) {
	if ctx == nil {
		return nil, errors.New("QUIC listen context is required")
	}
	if d.config.ServerTLS == nil {
		return nil, errors.New("QUIC driver has no server TLS configuration")
	}
	parsed, err := parseEndpoint(endpoint, true)
	if err != nil {
		return nil, err
	}
	underlying, err := quicgo.ListenAddr(parsed.address, d.config.ServerTLS.Clone(), d.config.QUIC.Clone())
	if err != nil {
		return nil, fmt.Errorf("listen QUIC: %w", err)
	}
	actual := link.Endpoint((&url.URL{Scheme: EndpointScheme, Host: underlying.Addr().String()}).String())
	value := &listener{underlying: underlying, endpoint: actual, streamTimeout: d.config.AcceptStreamTimeout, done: make(chan struct{})}
	go func() {
		select {
		case <-ctx.Done():
			_ = value.Close()
		case <-value.done:
		}
	}()
	return value, nil
}

type endpointValue struct {
	address    string
	serverName string
}

func parseEndpoint(endpoint link.Endpoint, listen bool) (endpointValue, error) {
	raw := strings.TrimSpace(string(endpoint))
	if raw == "" {
		return endpointValue{}, errors.New("QUIC endpoint is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return endpointValue{}, fmt.Errorf("parse QUIC endpoint: %w", err)
	}
	if strings.ToLower(parsed.Scheme) != EndpointScheme {
		return endpointValue{}, fmt.Errorf("QUIC endpoint scheme must be %s", EndpointScheme)
	}
	if parsed.Host == "" {
		return endpointValue{}, errors.New("QUIC endpoint host:port is required")
	}
	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil || port == "" {
		return endpointValue{}, errors.New("QUIC endpoint must contain a valid host and port")
	}
	if !listen && (host == "" || net.ParseIP(host) != nil && net.ParseIP(host).IsUnspecified()) {
		return endpointValue{}, errors.New("QUIC dial endpoint must identify a remote host")
	}
	serverName := strings.TrimSpace(parsed.Query().Get("server_name"))
	if serverName == "" && net.ParseIP(strings.Trim(host, "[]")) == nil {
		serverName = strings.Trim(host, "[]")
	}
	return endpointValue{address: parsed.Host, serverName: serverName}, nil
}

type listener struct {
	underlying    *quicgo.Listener
	endpoint      link.Endpoint
	streamTimeout time.Duration
	done          chan struct{}
	once          sync.Once
}

func (l *listener) Accept(ctx context.Context) (link.Pipe, error) {
	if ctx == nil {
		return nil, errors.New("QUIC accept context is required")
	}
	connection, err := l.underlying.Accept(ctx)
	if err != nil {
		if errors.Is(err, quicgo.ErrServerClosed) {
			return nil, link.ErrClosed
		}
		return nil, err
	}
	streamCtx, cancel := context.WithTimeout(ctx, l.streamTimeout)
	defer cancel()
	stream, err := connection.AcceptStream(streamCtx)
	if err != nil {
		_ = connection.CloseWithError(0, "stream accept failed")
		return nil, fmt.Errorf("accept QUIC stream: %w", err)
	}
	preface := make([]byte, len(streamPreface))
	if _, err := io.ReadFull(stream, preface); err != nil || string(preface) != string(streamPreface) {
		stream.CancelRead(0)
		_ = connection.CloseWithError(0, "invalid stream preface")
		if err != nil {
			return nil, fmt.Errorf("read QUIC stream preface: %w", err)
		}
		return nil, errors.New("invalid QUIC stream preface")
	}
	return &pipe{connection: connection, stream: stream}, nil
}

func (l *listener) Addr() link.Endpoint { return l.endpoint }

func (l *listener) Close() error {
	var closeErr error
	l.once.Do(func() {
		close(l.done)
		closeErr = l.underlying.Close()
	})
	return closeErr
}

type pipe struct {
	connection *quicgo.Conn
	stream     *quicgo.Stream
	once       sync.Once
}

func (p *pipe) Read(data []byte) (int, error)  { return p.stream.Read(data) }
func (p *pipe) Write(data []byte) (int, error) { return p.stream.Write(data) }
func (p *pipe) SetDeadline(deadline time.Time) error {
	return p.stream.SetDeadline(deadline)
}
func (p *pipe) SetReadDeadline(deadline time.Time) error {
	return p.stream.SetReadDeadline(deadline)
}
func (p *pipe) SetWriteDeadline(deadline time.Time) error {
	return p.stream.SetWriteDeadline(deadline)
}
func (p *pipe) Close() error {
	var closeErr error
	p.once.Do(func() {
		p.stream.CancelRead(0)
		p.stream.CancelWrite(0)
		_ = p.stream.Close()
		closeErr = p.connection.CloseWithError(0, "closed")
	})
	return closeErr
}
