package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type options struct {
	id             uint64
	address        string
	stateDirectory string
	identityOnly   bool
	issueNodeID    uint64
	issuePublicKey string
	issueRole      string
	permitTTL      time.Duration
	policy         string
	subject        uint64
	action         string
	resourceNode   uint64
	resource       string
}

func main() {
	var opts options
	flag.Uint64Var(&opts.id, "id", 1, "non-zero Hub node ID")
	flag.StringVar(&opts.address, "listen", "127.0.0.1:7331", "TCP listen address")
	flag.StringVar(&opts.stateDirectory, "state", "state/hub", "durable Hub state directory")
	flag.BoolVar(&opts.identityOnly, "identity", false, "print the durable Hub identity and exit (Hub must be stopped)")
	flag.Uint64Var(&opts.issueNodeID, "issue-node-id", 0, "child node ID for an offline admission permit")
	flag.StringVar(&opts.issuePublicKey, "issue-public-key", "", "child raw-base64 Ed25519 public key")
	flag.StringVar(&opts.issueRole, "issue-role", "device", "role bound to an offline admission permit")
	flag.DurationVar(&opts.permitTTL, "permit-ttl", time.Hour, "offline admission permit lifetime")
	flag.StringVar(&opts.policy, "policy", "", "offline policy mutation: grant or revoke")
	flag.Uint64Var(&opts.subject, "subject", 0, "policy subject node ID")
	flag.StringVar(&opts.action, "action", "", "policy action: subscribe or invoke")
	flag.Uint64Var(&opts.resourceNode, "resource-node", 0, "policy resource owner node ID")
	flag.StringVar(&opts.resource, "resource", "", "policy resource name")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, opts, os.Stdout); err != nil {
		fatal(err)
	}
}

func run(ctx context.Context, opts options, output io.Writer) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if output == nil {
		return errors.New("output writer is required")
	}
	nodeID := protocol.NodeID(opts.id)
	if err := nodeID.Validate(); err != nil {
		return fmt.Errorf("Hub identity: %w", err)
	}
	if opts.stateDirectory == "" {
		return errors.New("Hub state directory is required")
	}

	offlineModes := 0
	if opts.identityOnly {
		offlineModes++
	}
	if opts.issueNodeID != 0 || opts.issuePublicKey != "" {
		offlineModes++
	}
	if opts.policy != "" {
		offlineModes++
	}
	if offlineModes > 1 {
		return errors.New("identity, permit issuance, and policy mutation are mutually exclusive")
	}
	if offlineModes == 1 {
		state, err := hostconfig.Open(opts.stateDirectory, nodeID)
		if err != nil {
			return fmt.Errorf("open Hub state: %w", err)
		}
		switch {
		case opts.identityOnly:
			return json.NewEncoder(output).Encode(struct {
				NodeID    protocol.NodeID `json:"node_id"`
				PublicKey string          `json:"public_key"`
			}{NodeID: state.Identity.NodeID, PublicKey: base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey)})
		case opts.policy != "":
			return mutatePolicy(state.Policy, opts, output)
		default:
			return issuePermit(state.Admission, opts, output)
		}
	}

	if opts.subject != 0 || opts.action != "" || opts.resourceNode != 0 || opts.resource != "" {
		return errors.New("policy fields require -policy grant or -policy revoke")
	}
	if opts.address == "" {
		return errors.New("Hub listen address is required")
	}
	runtime, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: opts.stateDirectory,
		NodeID:         nodeID,
		Listeners:      []hub.ListenerConfig{{Driver: tcp.Driver{}, Endpoint: link.Endpoint(opts.address)}},
	})
	if err != nil {
		return err
	}
	defer runtime.Close()
	if err := json.NewEncoder(output).Encode(map[string]any{
		"node_id":   nodeID,
		"endpoints": runtime.Endpoints,
		"state":     opts.stateDirectory,
	}); err != nil {
		return fmt.Errorf("write Hub startup result: %w", err)
	}
	<-ctx.Done()
	return nil
}

func issuePermit(admission *auth.Admission, opts options, output io.Writer) error {
	childID := protocol.NodeID(opts.issueNodeID)
	if err := childID.Validate(); err != nil {
		return fmt.Errorf("permit child identity: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(opts.issuePublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("issue-public-key must be a raw-base64 Ed25519 public key")
	}
	permit, err := admission.Issue(childID, ed25519.PublicKey(key), opts.issueRole, opts.permitTTL)
	if err != nil {
		return fmt.Errorf("issue admission permit: %w", err)
	}
	if err := json.NewEncoder(output).Encode(permit); err != nil {
		return fmt.Errorf("write admission permit: %w", err)
	}
	return nil
}

func mutatePolicy(policy *auth.PolicyState, opts options, output io.Writer) error {
	if opts.policy != "grant" && opts.policy != "revoke" {
		return errors.New("policy must be grant or revoke")
	}
	subject := protocol.NodeID(opts.subject)
	if err := subject.Validate(); err != nil {
		return fmt.Errorf("policy subject: %w", err)
	}
	owner := protocol.NodeID(opts.resourceNode)
	if err := owner.Validate(); err != nil {
		return fmt.Errorf("policy resource owner: %w", err)
	}
	action := auth.Action(opts.action)
	request := auth.Request{
		Subject: subject,
		Action:  action,
		Resource: protocol.ResourceID{
			Owner: owner,
			Name:  opts.resource,
		},
	}
	var err error
	if opts.policy == "grant" {
		err = policy.Grant(request)
	} else {
		err = policy.Revoke(request)
	}
	if err != nil {
		return fmt.Errorf("%s policy: %w", opts.policy, err)
	}
	if err := json.NewEncoder(output).Encode(protocol.ManagementResultV1{Version: 1, Status: "ok"}); err != nil {
		return fmt.Errorf("write policy result: %w", err)
	}
	return nil
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
