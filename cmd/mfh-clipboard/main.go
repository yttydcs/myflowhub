package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
	clipboardbridge "github.com/yttydcs/myflowhub/apps/nodes/clipboard/bridge"
	clipboardwindows "github.com/yttydcs/myflowhub/apps/nodes/clipboard/platform/windows"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

type options struct {
	stateDirectory string
	nodeID         uint64
	parentID       uint64
	endpoint       string
	parentKey      string
	permitFile     string
	identityOnly   bool
	bridgeMode     bool
	connectTimeout time.Duration
}

func main() {
	var value options
	flag.StringVar(&value.stateDirectory, "state", "state/clipboard", "durable ClipboardNode state directory")
	flag.Uint64Var(&value.nodeID, "id", 11, "non-zero ClipboardNode ID")
	flag.Uint64Var(&value.parentID, "parent-id", 1, "parent Hub node ID")
	flag.StringVar(&value.endpoint, "endpoint", "127.0.0.1:7331", "parent Hub TCP endpoint")
	flag.StringVar(&value.parentKey, "parent-key", "", "parent raw-base64 Ed25519 public key (required on first use)")
	flag.StringVar(&value.permitFile, "permit", "", "optional provisioning permit JSON file")
	flag.BoolVar(&value.identityOnly, "identity", false, "print durable node identity and exit")
	flag.BoolVar(&value.bridgeMode, "bridge", false, "serve the bounded local Flutter bridge over stdin/stdout")
	flag.DurationVar(&value.connectTimeout, "connect-timeout", 15*time.Second, "initial connection timeout")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, value); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, options options) error {
	if ctx == nil {
		return errors.New("clipboard command context is required")
	}
	if options.connectTimeout <= 0 {
		return errors.New("connect-timeout must be positive")
	}
	if options.bridgeMode {
		if runtime.GOOS != "windows" {
			return errors.New("mfh-clipboard bridge currently requires Windows")
		}
		server, err := clipboardbridge.New(func() clipboard.Adapter { return clipboardwindows.New() })
		if err != nil {
			return err
		}
		return server.Serve(ctx, os.Stdin, os.Stdout)
	}
	nodeID := protocol.NodeID(options.nodeID)
	state, err := auth.OpenState(options.stateDirectory, nodeID)
	if err != nil {
		return err
	}
	if options.identityOnly {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"node_id": strconv.FormatUint(uint64(nodeID), 10), "public_key": base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey),
		})
	}
	if runtime.GOOS != "windows" {
		return errors.New("mfh-clipboard currently requires Windows; Android and Web use the Flutter binding")
	}
	parentID := protocol.NodeID(options.parentID)
	if err := parentID.Validate(); err != nil {
		return err
	}
	var parentKey ed25519.PublicKey
	if options.parentKey != "" {
		decoded, err := base64.RawStdEncoding.DecodeString(options.parentKey)
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			return errors.New("parent-key must be a raw-base64 Ed25519 public key")
		}
		parentKey = ed25519.PublicKey(decoded)
	}
	permit, err := readPermit(options.permitFile)
	if err != nil {
		return err
	}
	platform := clipboardwindows.New()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	service, err := clipboard.Start(ctx, clipboard.RuntimeConfig{
		StateDirectory: options.stateDirectory, NodeID: nodeID, ParentID: parentID, ParentKey: parentKey,
		Permit: permit, Endpoint: link.Endpoint(options.endpoint), Adapter: platform,
	})
	if err != nil {
		_ = platform.Close()
		return err
	}
	defer service.Close()
	connectCtx, cancel := context.WithTimeout(ctx, options.connectTimeout)
	defer cancel()
	if err := waitConnected(connectCtx, service.Connection); err != nil {
		return err
	}
	logger.Info("ClipboardNode connected", "node_id", nodeID, "parent_id", parentID, "endpoint", options.endpoint)
	<-ctx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	return ctx.Err()
}

func readPermit(path string) (*protocol.ProvisioningPermitV1, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read permit: %w", err)
	}
	var permit protocol.ProvisioningPermitV1
	if err := protocol.DecodeJSONPayload(data, protocol.DefaultMaxPayload, &permit); err != nil {
		return nil, fmt.Errorf("decode permit: %w", err)
	}
	return &permit, nil
}

func waitConnected(ctx context.Context, connection sdk.ConnectionStatus) error {
	for {
		snapshot := connection.Snapshot()
		switch snapshot.State {
		case sdk.ConnectionConnected:
			return nil
		case sdk.ConnectionFailed, sdk.ConnectionStopped:
			return fmt.Errorf("clipboard parent connection %s: %s", snapshot.State, snapshot.LastError)
		}
		if _, err := connection.WaitChange(ctx, snapshot.Generation); err != nil {
			return fmt.Errorf("wait for clipboard parent: %w", err)
		}
	}
}
