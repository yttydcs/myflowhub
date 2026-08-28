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
	"time"

	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type options struct {
	stateDirectory string
	clientID       uint64
	parentID       uint64
	endpoint       string
	parentKey      string
	permitFile     string
	operation      string
	resource       string
	request        string
	requestFile    string
	timeout        time.Duration
}

func main() {
	var value options
	flag.StringVar(&value.stateDirectory, "state", "state/admin", "durable admin client state directory")
	flag.Uint64Var(&value.clientID, "id", 2, "non-zero admin client node ID")
	flag.Uint64Var(&value.parentID, "parent-id", 1, "Hub node ID")
	flag.StringVar(&value.endpoint, "endpoint", "127.0.0.1:7331", "Hub TCP endpoint")
	flag.StringVar(&value.parentKey, "parent-key", "", "Hub raw-base64 Ed25519 public key (required on first use)")
	flag.StringVar(&value.permitFile, "permit", "", "optional provisioning permit JSON file")
	flag.StringVar(&value.operation, "op", "snapshot", "operation: identity, snapshot, or invoke")
	flag.StringVar(&value.resource, "resource", protocol.BuiltinResourceCatalog, "target resource name")
	flag.StringVar(&value.request, "request", "", "inline command request payload")
	flag.StringVar(&value.requestFile, "request-file", "", "command request payload file")
	flag.DurationVar(&value.timeout, "timeout", 10*time.Second, "connect and operation timeout")
	flag.Parse()
	if err := run(value); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(options options) error {
	if options.timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	state, err := hostconfig.Open(options.stateDirectory, protocol.NodeID(options.clientID))
	if err != nil {
		return err
	}
	if options.operation == "identity" {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"node_id":    state.Identity.NodeID,
			"public_key": base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey),
		})
	}
	parentID := protocol.NodeID(options.parentID)
	if err := parentID.Validate(); err != nil {
		return err
	}
	if options.parentKey != "" {
		key, err := base64.RawStdEncoding.DecodeString(options.parentKey)
		if err != nil || len(key) != ed25519.PublicKeySize {
			return errors.New("parent-key must be a raw-base64 Ed25519 public key")
		}
		if err := state.Trust.Add(parentID, ed25519.PublicKey(key)); err != nil {
			return fmt.Errorf("trust Hub identity: %w", err)
		}
	}
	if _, trusted := state.Trust.PublicKey(parentID); !trusted {
		return errors.New("Hub identity is not trusted; provide parent-key on first use")
	}
	var permit *protocol.ProvisioningPermitV1
	if options.permitFile != "" {
		data, err := os.ReadFile(options.permitFile)
		if err != nil {
			return fmt.Errorf("read permit: %w", err)
		}
		var value protocol.ProvisioningPermitV1
		if err := protocol.DecodeJSONPayload(data, protocol.DefaultMaxPayload, &value); err != nil {
			return fmt.Errorf("decode permit: %w", err)
		}
		permit = &value
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
	defer cancel()
	runtime, err := node.New(ctx, node.Config{Identity: state.Identity, Trust: state.Trust, JoinPermit: permit})
	if err != nil {
		return err
	}
	defer runtime.Close()
	if err := runtime.ConnectParent(ctx, tcp.Driver{}, link.Endpoint(options.endpoint), parentID); err != nil {
		return fmt.Errorf("connect Hub: %w", err)
	}
	resourceID := protocol.ResourceID{Owner: parentID, Name: options.resource}
	if err := resourceID.Validate(); err != nil {
		return err
	}
	switch options.operation {
	case "snapshot":
		current, err := runtime.Subscribe(ctx, resourceID, options.timeout, 4)
		if err != nil {
			return err
		}
		defer current.Cancel()
		select {
		case event := <-current.Events:
			if event.Kind != subscription.EventSnapshot {
				return fmt.Errorf("resource %s did not return a Variable snapshot", options.resource)
			}
			_, err = os.Stdout.Write(append(event.Value, '\n'))
			return err
		case err := <-current.Errors:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case "invoke":
		request, err := commandRequest(options)
		if err != nil {
			return err
		}
		response, err := runtime.Invoke(ctx, resourceID, request)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(response, '\n'))
		return err
	default:
		return fmt.Errorf("unsupported operation %q", options.operation)
	}
}

func commandRequest(options options) ([]byte, error) {
	if options.request != "" && options.requestFile != "" {
		return nil, errors.New("request and request-file are mutually exclusive")
	}
	if options.requestFile != "" {
		data, err := os.ReadFile(options.requestFile)
		if err != nil {
			return nil, fmt.Errorf("read request file: %w", err)
		}
		return data, nil
	}
	if options.request == "" {
		return nil, errors.New("invoke requires request or request-file")
	}
	return []byte(options.request), nil
}
