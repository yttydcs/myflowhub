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
	id               uint64
	address          string
	stateDirectory   string
	identityOnly     bool
	issueNodeID      uint64
	issueTargetID    uint64
	issuePublicKey   string
	issueRole        string
	issueRequestID   string
	issueDescendants bool
	authorityID      uint64
	authorityKey     string
	permitTTL        time.Duration
	policy           string
	subject          uint64
	action           string
	resourceNode     uint64
	resource         string
}

func main() {
	var opts options
	flag.Uint64Var(&opts.id, "id", 1, "non-zero Hub node ID")
	flag.StringVar(&opts.address, "listen", "127.0.0.1:7331", "TCP listen address")
	flag.StringVar(&opts.stateDirectory, "state", "state/hub", "durable Hub state directory")
	flag.BoolVar(&opts.identityOnly, "identity", false, "print the durable Hub identity and exit (Hub must be stopped)")
	flag.Uint64Var(&opts.issueNodeID, "issue-node-id", 0, "legacy child Node ID; omit to issue a new Enrollment Permit without a client-selected Node ID")
	flag.Uint64Var(&opts.issueTargetID, "issue-target-id", 0, "target parent/subtree Node ID for a new Enrollment Permit (defaults to this Hub)")
	flag.StringVar(&opts.issuePublicKey, "issue-public-key", "", "device raw-base64 Ed25519 public key")
	flag.StringVar(&opts.issueRole, "issue-role", "device", "legacy role or new Enrollment admission profile")
	flag.StringVar(&opts.issueRequestID, "issue-request-id", "", "optional 16-byte lowercase hex idempotency ID for new Permit issuance")
	flag.BoolVar(&opts.issueDescendants, "issue-allow-descendants", false, "allow the new Enrollment Permit below the target subtree")
	flag.Uint64Var(&opts.authorityID, "admission-authority-id", 0, "Admission Authority Node ID; zero or this Hub uses the local Authority")
	flag.StringVar(&opts.authorityKey, "admission-authority-key", "", "remote Admission Authority raw-base64 Ed25519 public key")
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
	if opts.issueNodeID != 0 || opts.issueTargetID != 0 || opts.issuePublicKey != "" || opts.issueRequestID != "" || opts.issueDescendants {
		offlineModes++
	}
	if opts.policy != "" {
		offlineModes++
	}
	if offlineModes > 1 {
		return errors.New("identity, permit issuance, and policy mutation are mutually exclusive")
	}
	if offlineModes == 1 {
		if opts.authorityID != 0 || opts.authorityKey != "" {
			return errors.New("Admission Authority routing options apply only while running the Hub")
		}
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
			return issuePermit(state, opts, output)
		}
	}

	if opts.subject != 0 || opts.action != "" || opts.resourceNode != 0 || opts.resource != "" {
		return errors.New("policy fields require -policy grant or -policy revoke")
	}
	if opts.address == "" {
		return errors.New("Hub listen address is required")
	}
	authorityNodeID := protocol.NodeID(opts.authorityID)
	var authorityPublicKey ed25519.PublicKey
	if authorityNodeID != 0 && authorityNodeID != nodeID {
		decoded, err := base64.RawStdEncoding.DecodeString(opts.authorityKey)
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			return errors.New("remote admission-authority-key must be a raw-base64 Ed25519 public key")
		}
		authorityPublicKey = ed25519.PublicKey(decoded)
	} else if opts.authorityKey != "" {
		return errors.New("admission-authority-key is only valid for a remote Admission Authority")
	}
	runtime, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory:              opts.stateDirectory,
		NodeID:                      nodeID,
		Listeners:                   []hub.ListenerConfig{{Driver: tcp.Driver{}, Endpoint: link.Endpoint(opts.address)}},
		AdmissionAuthorityNodeID:    authorityNodeID,
		AdmissionAuthorityPublicKey: authorityPublicKey,
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

func issuePermit(state *hostconfig.Runtime, opts options, output io.Writer) error {
	if state == nil {
		return errors.New("Hub state is required")
	}
	key, err := base64.RawStdEncoding.DecodeString(opts.issuePublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("issue-public-key must be a raw-base64 Ed25519 public key")
	}
	if opts.issueNodeID != 0 {
		if opts.issueTargetID != 0 || opts.issueRequestID != "" || opts.issueDescendants {
			return errors.New("legacy issue-node-id cannot be combined with new Enrollment Permit scope options")
		}
		childID := protocol.NodeID(opts.issueNodeID)
		if err := childID.Validate(); err != nil {
			return fmt.Errorf("permit child identity: %w", err)
		}
		permit, err := state.Admission.Issue(childID, ed25519.PublicKey(key), opts.issueRole, opts.permitTTL)
		if err != nil {
			return fmt.Errorf("issue legacy admission permit: %w", err)
		}
		if err := json.NewEncoder(output).Encode(permit); err != nil {
			return fmt.Errorf("write legacy admission permit: %w", err)
		}
		return nil
	}
	targetID := protocol.NodeID(opts.issueTargetID)
	if targetID == 0 {
		targetID = state.Identity.NodeID
	}
	if err := targetID.Validate(); err != nil {
		return fmt.Errorf("Enrollment Permit target: %w", err)
	}
	requestID := opts.issueRequestID
	if requestID == "" {
		generated, err := protocol.NewMessageID()
		if err != nil {
			return err
		}
		requestID = generated.String()
	}
	authority, err := auth.LoadEnrollmentAuthority(state.Identity, state.Store, auth.EnrollmentAuthorityConfig{})
	if err != nil {
		return err
	}
	permit, err := authority.IssuePermit(requestID, auth.DevicePublicKeyFingerprint(ed25519.PublicKey(key)), targetID, opts.issueDescendants, opts.issueRole, opts.permitTTL)
	if err != nil {
		return fmt.Errorf("issue Enrollment Permit: %w", err)
	}
	if err := json.NewEncoder(output).Encode(permit); err != nil {
		return fmt.Errorf("write Enrollment Permit: %w", err)
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
