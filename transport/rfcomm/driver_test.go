package rfcomm

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/yttydcs/myflowhub/internal/transporttest"
	"github.com/yttydcs/myflowhub/runtime/link"
)

func TestTransportContract(t *testing.T) {
	provider := &memoryProvider{}
	driver, err := NewWithProvider(provider, 64)
	if err != nil {
		t.Fatal(err)
	}
	transporttest.Run(t, driver, link.Endpoint("bt+rfcomm://AA:BB:CC:DD:EE:FF?channel=7"))
}

func TestChunkedPipeCompletesShortWritesWithinMTU(t *testing.T) {
	underlying := &shortPipe{maxWrite: 3}
	pipe := &chunkedPipe{Pipe: underlying, mtu: 8}
	data := []byte("0123456789abcdef")
	written, err := pipe.Write(data)
	if err != nil || written != len(data) || string(underlying.data) != string(data) {
		t.Fatalf("written=%d data=%q err=%v", written, underlying.data, err)
	}
	for _, size := range underlying.writes {
		if size > 8 {
			t.Fatalf("provider write exceeded MTU: %d", size)
		}
	}
}

func TestEndpointRejectsReservedNameAndUnsafeChannel(t *testing.T) {
	if _, err := ParseEndpoint("bt+rfcomm://AA:BB:CC:DD:EE:FF?name=nearby"); !errors.Is(err, ErrEndpointNameReserved) {
		t.Fatalf("reserved name error = %v", err)
	}
	if _, err := ParseEndpoint("bt+rfcomm://AA:BB:CC:DD:EE:FF?channel=31"); !errors.Is(err, ErrEndpointChannelInvalid) {
		t.Fatalf("channel error = %v", err)
	}
}

type memoryProvider struct {
	mu       sync.Mutex
	listener *memoryProviderListener
}

func (p *memoryProvider) Listen(Options) (ProviderListener, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener != nil {
		return nil, errors.New("already listening")
	}
	p.listener = &memoryProviderListener{pending: make(chan link.Pipe), done: make(chan struct{})}
	return p.listener, nil
}

func (p *memoryProvider) Dial(ctx context.Context, _ DialOptions) (link.Pipe, error) {
	p.mu.Lock()
	listener := p.listener
	p.mu.Unlock()
	if listener == nil {
		return nil, errors.New("not listening")
	}
	client, server := net.Pipe()
	select {
	case listener.pending <- server:
		return client, nil
	case <-ctx.Done():
		_ = client.Close()
		_ = server.Close()
		return nil, ctx.Err()
	case <-listener.done:
		_ = client.Close()
		_ = server.Close()
		return nil, link.ErrClosed
	}
}

type memoryProviderListener struct {
	pending chan link.Pipe
	done    chan struct{}
	once    sync.Once
}

func (l *memoryProviderListener) Accept() (link.Pipe, net.Addr, net.Addr, error) {
	select {
	case pipe := <-l.pending:
		return pipe, testAddr("local"), testAddr("remote"), nil
	case <-l.done:
		return nil, nil, nil, link.ErrClosed
	}
}

func (l *memoryProviderListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *memoryProviderListener) Addr() net.Addr { return testAddr("listener") }

type testAddr string

func (a testAddr) Network() string { return "rfcomm-test" }
func (a testAddr) String() string  { return string(a) }

type shortPipe struct {
	data     []byte
	writes   []int
	maxWrite int
}

func (p *shortPipe) Read([]byte) (int, error) { return 0, io.EOF }
func (p *shortPipe) Write(data []byte) (int, error) {
	if len(data) > p.maxWrite {
		data = data[:p.maxWrite]
	}
	p.writes = append(p.writes, len(data))
	p.data = append(p.data, data...)
	return len(data), nil
}
func (p *shortPipe) Close() error { return nil }
