package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestAdminResourceOwnerRoutesAdmissionToAuthority(t *testing.T) {
	options := options{resource: protocol.BuiltinAdmissionListPermits}
	if got := adminResourceOwner(options, 9, 1); got != 1 {
		t.Fatalf("Admission owner = %d, want Authority 1", got)
	}
	options.resource = protocol.BuiltinResourceCatalog
	if got := adminResourceOwner(options, 9, 1); got != 9 {
		t.Fatalf("ordinary resource owner = %d, want parent 9", got)
	}
	options.ownerID = 7
	if got := adminResourceOwner(options, 9, 1); got != 7 {
		t.Fatalf("explicit owner = %d, want 7", got)
	}
}

func TestJSONOperateValidatesCapabilityInputAndPayloadBounds(t *testing.T) {
	for _, value := range []options{
		{capability: ""}, {capability: "bad capability"},
		{capability: "create", schema: "test.v1", request: "{"},
		{capability: "create", request: "{}", requestFile: "also.json"},
		{capability: "create", request: strings.Repeat(" ", protocol.DefaultMaxPayload+1)},
	} {
		if _, err := operationRequest(value); err == nil {
			t.Fatalf("invalid operate accepted: %+v", value.capability)
		}
	}
	if payload, err := operationRequest(options{capability: "read"}); err != nil || len(payload) != 0 {
		t.Fatalf("empty read: %s %v", payload, err)
	}
	if payload, err := operationRequest(options{capability: "create", schema: "mfh.policy.definition-put.v1", request: `{ "version":1 }`}); err != nil || !strings.Contains(string(payload), "version") {
		t.Fatalf("typed operation: %s %v", payload, err)
	}
	path := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(path, []byte(strings.Repeat(" ", protocol.DefaultMaxPayload+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := operationRequest(options{capability: "create", requestFile: path}); err == nil {
		t.Fatal("oversized file accepted")
	}
}
