package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/yttydcs/myflowhub/examples/variables"
	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func main() {
	var stateDirectory, endpoint, parentKey, permitFile string
	var id, parentID uint64
	var count int
	var identityOnly bool
	flag.StringVar(&stateDirectory, "state", "state/variable-example", "independent persistent state directory")
	flag.StringVar(&endpoint, "endpoint", "127.0.0.1:7331", "parent TCP endpoint")
	flag.StringVar(&parentKey, "parent-key", "", "parent raw-base64 Ed25519 public key")
	flag.StringVar(&permitFile, "permit", "", "provisioning permit JSON file")
	flag.Uint64Var(&id, "id", 20, "independent non-zero example Node ID")
	flag.Uint64Var(&parentID, "parent-id", 1, "parent Hub Node ID")
	flag.IntVar(&count, "count", 6, "six-type variable count; use 1998 for reference load")
	flag.BoolVar(&identityOnly, "identity", false, "print persistent public identity and exit")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, stateDirectory, protocol.NodeID(id), protocol.NodeID(parentID), endpoint, parentKey, permitFile, count, identityOnly); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, directory string, id, parentID protocol.NodeID, endpoint, parentKey, permitFile string, count int, identityOnly bool) error {
	if identityOnly {
		state, err := auth.OpenState(directory, id)
		if err != nil {
			return err
		}
		defer state.Policy.Close()
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"node_id": strconv.FormatUint(uint64(id), 10), "public_key": base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey)})
	}
	key, err := base64.RawStdEncoding.DecodeString(parentKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("parent-key must be a raw-base64 Ed25519 public key")
	}
	var permit *protocol.ProvisioningPermitV1
	if permitFile != "" {
		data, err := os.ReadFile(permitFile)
		if err != nil {
			return fmt.Errorf("read example permit: %w", err)
		}
		permit = &protocol.ProvisioningPermitV1{}
		if err := protocol.DecodeJSONPayload(data, protocol.DefaultMaxPayload, permit); err != nil {
			return err
		}
	}
	mechanics, err := nodehost.ProductRuntimeConfig()
	if err != nil {
		return err
	}
	host, err := nodehost.New(ctx, nodehost.Config{StateDirectory: directory, NodeID: id, Runtime: mechanics,
		Parent: &nodehost.ParentConfig{NodeID: parentID, PublicKey: key, Permit: permit, Driver: tcp.Driver{}, Endpoint: link.Endpoint(endpoint), Supervisor: nodehost.ProductSupervisorConfig(node.SupervisorConfig{})},
	})
	if err != nil {
		return err
	}
	defer host.Close()
	provider, err := variables.Register(host.Resources(), id, count)
	if err != nil {
		return err
	}
	if err := host.Start(); err != nil {
		return err
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"node_id": strconv.FormatUint(uint64(id), 10), "synthetic": true, "variables": count, "sample_interval_ms": 1000}); err != nil {
		return err
	}
	return provider.Run(ctx)
}
