package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func main() {
	var id uint64
	var address string
	flag.Uint64Var(&id, "id", 1, "non-zero node ID")
	flag.StringVar(&address, "listen", "127.0.0.1:0", "TCP listen address")
	flag.Parse()
	identity, err := auth.GenerateIdentity(protocol.NodeID(id))
	if err != nil {
		fatal(err)
	}
	trust := auth.NewTrustStore()
	if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
		fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runtime, err := hub.Start(ctx, hub.Config{
		Node:   node.Config{Identity: identity, Trust: trust, Policy: auth.AllowAll{}},
		Driver: tcp.Driver{}, Endpoint: link.Endpoint(address),
	})
	if err != nil {
		fatal(err)
	}
	defer runtime.Close()
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"node_id": id, "endpoint": runtime.Endpoint, "public_key_hex": hex.EncodeToString(identity.PublicKey)})
	<-ctx.Done()
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
