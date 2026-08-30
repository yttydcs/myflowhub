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
	"path/filepath"
	"strings"
	"time"

	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/enrollment"
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
	authorityKey   string
	allowTOFU      bool
	permitFile     string
	operation      string
	resource       string
	ownerID        uint64
	request        string
	requestFile    string
	timeout        time.Duration
}

func main() {
	var value options
	flag.StringVar(&value.stateDirectory, "state", "state/admin", "durable admin client state directory")
	flag.Uint64Var(&value.clientID, "id", 0, "legacy preassigned admin Node ID; zero uses Authority-managed Enrollment")
	flag.Uint64Var(&value.parentID, "parent-id", 0, "legacy Hub Node ID or optional expected parent during Enrollment")
	flag.StringVar(&value.endpoint, "endpoint", "127.0.0.1:7331", "Hub TCP endpoint")
	flag.StringVar(&value.parentKey, "parent-key", "", "legacy Hub key or optional pinned parent key during Enrollment")
	flag.StringVar(&value.authorityKey, "authority-key", "", "optional pinned Admission Authority raw-base64 Ed25519 public key")
	flag.BoolVar(&value.allowTOFU, "tofu", false, "explicitly accept and persist the first observed parent and Authority fingerprints")
	flag.StringVar(&value.permitFile, "permit", "", "optional legacy Join Permit or new Enrollment Permit JSON file")
	flag.StringVar(&value.operation, "op", "snapshot", "operation: identity, enroll, enrollment-status, snapshot, or invoke")
	flag.StringVar(&value.resource, "resource", protocol.BuiltinResourceCatalog, "target resource name")
	flag.Uint64Var(&value.ownerID, "owner-id", 0, "target resource owner Node ID; defaults to the direct parent or Enrollment Authority for system/admission resources")
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
	if options.clientID == 0 {
		return runEnrollmentManaged(options)
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
	resourceID := protocol.ResourceID{Owner: adminResourceOwner(options, parentID, 0), Name: options.resource}
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

func runEnrollmentManaged(options options) error {
	store, err := keystore.New(filepath.Join(options.stateDirectory, "state"))
	if err != nil {
		return err
	}
	clientState, err := auth.LoadOrCreateEnrollmentClientState(store)
	if err != nil {
		return err
	}
	snapshot := clientState.Snapshot()
	if options.operation == "identity" || options.operation == "enrollment-status" {
		result := map[string]any{
			"status": snapshot.Status, "request_id": snapshot.RequestID,
			"device_public_key": base64.RawStdEncoding.EncodeToString(snapshot.DevicePublicKey),
		}
		if snapshot.Grant != nil {
			result["node_id"] = snapshot.Grant.NodeID
			result["enrollment_id"] = snapshot.Grant.EnrollmentID
			result["parent_node_id"] = snapshot.Grant.ParentNodeID
			result["authority_node_id"] = snapshot.Grant.AuthorityNodeID
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	if options.operation == "enroll" {
		return enrollClient(options, clientState, snapshot)
	}
	identity, enrolled, err := clientState.Identity()
	if err != nil {
		return err
	}
	if !enrolled {
		return errors.New("client has no Node ID yet; run -op enroll and wait for approval")
	}
	trust := auth.NewTrustStore()
	if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
		return err
	}
	if err := trust.Add(snapshot.ParentNodeID, snapshot.ParentPublicKey); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
	defer cancel()
	runtime, err := node.New(ctx, node.Config{Identity: identity, Trust: trust})
	if err != nil {
		return err
	}
	defer runtime.Close()
	if err := runtime.ConnectParent(ctx, tcp.Driver{}, link.Endpoint(options.endpoint), snapshot.ParentNodeID); err != nil {
		return fmt.Errorf("connect enrolled parent: %w", err)
	}
	return operateAdmin(ctx, runtime, snapshot.ParentNodeID, snapshot.AuthorityNodeID, options)
}

func enrollClient(options options, clientState *auth.EnrollmentClientState, snapshot auth.EnrollmentClientSnapshot) error {
	clientOptions := enrollment.ClientOptions{RequestID: snapshot.RequestID, AllowTOFU: options.allowTOFU}
	if snapshot.ParentNodeID != 0 {
		clientOptions.ExpectedParentNodeID = snapshot.ParentNodeID
		clientOptions.ExpectedParentPublicKey = snapshot.ParentPublicKey
		clientOptions.ExpectedAuthorityNodeID = snapshot.AuthorityNodeID
		clientOptions.ExpectedAuthorityPublicKey = snapshot.AuthorityPublicKey
	} else {
		if options.parentID != 0 {
			clientOptions.ExpectedParentNodeID = protocol.NodeID(options.parentID)
		}
		if options.parentKey != "" {
			key, err := decodeEd25519Key("parent-key", options.parentKey)
			if err != nil {
				return err
			}
			clientOptions.ExpectedParentPublicKey = key
		}
		if options.authorityKey != "" {
			key, err := decodeEd25519Key("authority-key", options.authorityKey)
			if err != nil {
				return err
			}
			clientOptions.ExpectedAuthorityPublicKey = key
		}
	}
	if options.permitFile != "" {
		data, err := os.ReadFile(options.permitFile)
		if err != nil {
			return fmt.Errorf("read Enrollment Permit: %w", err)
		}
		var permit protocol.EnrollmentPermitV1
		if err := protocol.DecodeJSONPayload(data, protocol.EnrollmentMaxPayload, &permit); err != nil {
			return fmt.Errorf("decode Enrollment Permit: %w", err)
		}
		clientOptions.Permit = &permit
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
	defer cancel()
	pipe, err := (tcp.Driver{}).Dial(ctx, link.Endpoint(options.endpoint))
	if err != nil {
		return err
	}
	result, err := enrollment.Enroll(ctx, pipe, clientState.DeviceIdentity(), clientOptions)
	_ = pipe.Close()
	if err != nil {
		return err
	}
	if err := clientState.RecordObservation(result.ParentNodeID, result.ParentPublicKey, result.AuthorityNodeID, result.AuthorityPublicKey); err != nil {
		return err
	}
	if result.Outcome.Status == "granted" {
		if result.Outcome.Grant == nil {
			return errors.New("Enrollment result is granted without a Grant")
		}
		if err := clientState.RecordGrant(*result.Outcome.Grant); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(result.Outcome)
}

func operateAdmin(ctx context.Context, runtime *node.Node, parentID, authorityNodeID protocol.NodeID, options options) error {
	resourceID := protocol.ResourceID{Owner: adminResourceOwner(options, parentID, authorityNodeID), Name: options.resource}
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

func adminResourceOwner(options options, parentID, authorityNodeID protocol.NodeID) protocol.NodeID {
	if options.ownerID != 0 {
		return protocol.NodeID(options.ownerID)
	}
	if authorityNodeID != 0 && strings.HasPrefix(options.resource, "system/admission/") {
		return authorityNodeID
	}
	return parentID
}

func decodeEd25519Key(name, value string) (ed25519.PublicKey, error) {
	key, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%s must be a raw-base64 Ed25519 public key", name)
	}
	return ed25519.PublicKey(key), nil
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
