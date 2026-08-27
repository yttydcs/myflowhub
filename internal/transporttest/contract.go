package transporttest

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/runtime/link"
)

func Run(t *testing.T, driver link.Driver, endpoint link.Endpoint) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	listener, err := driver.Listen(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan link.Pipe, 1)
	acceptErr := make(chan error, 1)
	go func() {
		pipe, err := listener.Accept(ctx)
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- pipe
	}()
	client, err := driver.Dial(ctx, listener.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var server link.Pipe
	select {
	case server = <-accepted:
	case err := <-acceptErr:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	defer server.Close()
	want := []byte("transport-contract")
	go func() { _, _ = client.Write(want) }()
	got := make([]byte, len(want))
	if _, err := io.ReadFull(server, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("want %q, got %q", want, got)
	}
}
