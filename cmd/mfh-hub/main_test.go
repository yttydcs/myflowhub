package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
)

func TestOfflineIdentityPermitAndPolicyBootstrap(t *testing.T) {
	directory := t.TempDir()
	base := options{id: 1, stateDirectory: directory}

	var identityOutput bytes.Buffer
	identityOptions := base
	identityOptions.identityOnly = true
	if err := run(context.Background(), identityOptions, &identityOutput); err != nil {
		t.Fatal(err)
	}
	var identity struct {
		NodeID    protocol.NodeID `json:"node_id"`
		PublicKey string          `json:"public_key"`
	}
	if err := json.Unmarshal(identityOutput.Bytes(), &identity); err != nil {
		t.Fatal(err)
	}
	if identity.NodeID != 1 {
		t.Fatalf("identity node ID = %d, want 1", identity.NodeID)
	}
	if key, err := base64.RawStdEncoding.DecodeString(identity.PublicKey); err != nil || len(key) != 32 {
		t.Fatalf("identity public key is invalid: %q", identity.PublicKey)
	}

	child, err := auth.GenerateIdentity(2)
	if err != nil {
		t.Fatal(err)
	}
	var permitOutput bytes.Buffer
	permitOptions := base
	permitOptions.issueNodeID = 2
	permitOptions.issuePublicKey = base64.RawStdEncoding.EncodeToString(child.PublicKey)
	permitOptions.issueRole = "device"
	permitOptions.permitTTL = time.Hour
	if err := run(context.Background(), permitOptions, &permitOutput); err != nil {
		t.Fatal(err)
	}
	var permit protocol.ProvisioningPermitV1
	if err := json.Unmarshal(permitOutput.Bytes(), &permit); err != nil {
		t.Fatal(err)
	}
	if err := permit.Validate(); err != nil {
		t.Fatalf("issued permit is invalid: %v", err)
	}
	if permit.ChildNodeID != "2" || permit.Role != "device" {
		t.Fatalf("unexpected permit: %+v", permit)
	}

	request := auth.Request{
		Subject: 2,
		Action:  auth.ActionSubscribe,
		Resource: protocol.ResourceID{
			Owner: 1,
			Name:  protocol.BuiltinManagementHealth,
		},
	}
	grantOptions := base
	grantOptions.policy = "grant"
	grantOptions.subject = uint64(request.Subject)
	grantOptions.action = string(request.Action)
	grantOptions.resourceNode = uint64(request.Resource.Owner)
	grantOptions.resource = request.Resource.Name
	if err := run(context.Background(), grantOptions, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	stateAfterGrant, err := hostconfig.Open(directory, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := stateAfterGrant.Policy.Authorize(context.Background(), request); err != nil {
		t.Fatalf("granted policy was not persisted: %v", err)
	}

	revokeOptions := grantOptions
	revokeOptions.policy = "revoke"
	if err := run(context.Background(), revokeOptions, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	stateAfterRevoke, err := hostconfig.Open(directory, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := stateAfterRevoke.Policy.Authorize(context.Background(), request); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("revoked policy authorization error = %v, want forbidden", err)
	}
}

func TestOfflineModesAreMutuallyExclusive(t *testing.T) {
	err := run(context.Background(), options{
		id:             1,
		stateDirectory: t.TempDir(),
		identityOnly:   true,
		policy:         "grant",
	}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected mutually exclusive mode error")
	}
}
