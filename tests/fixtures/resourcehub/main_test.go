package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestParseOptionsRejectsUnsafeOrAmbiguousConfiguration(t *testing.T) {
	roots := testRoots(t)
	state := filepath.Join(t.TempDir(), "hub-state")
	valid := []string{"-state", state, "-allowed-root", roots[0], "-forbidden-root", roots[1], "-misc-root", roots[2]}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "state required", args: valid[2:], want: "state directory is required"},
		{name: "state absolute", args: replaceArgument(valid, "-state", "relative-state"), want: "state directory must be absolute"},
		{name: "node identity", args: append(append([]string(nil), valid...), "-id", "0"), want: "Hub identity"},
		{name: "loopback only", args: append(append([]string(nil), valid...), "-listen", "0.0.0.0:0"), want: "loopback"},
		{name: "root required", args: replaceArgument(valid, "-misc-root", ""), want: "mount 2 root is required"},
		{name: "root absolute", args: replaceArgument(valid, "-allowed-root", "relative-root"), want: "mount 0 root must be absolute"},
		{name: "duplicate name", args: append(append([]string(nil), valid...), "-forbidden-name", "storage/allowed"), want: "duplicates Resource name"},
		{name: "unexpected argument", args: append(append([]string(nil), valid...), "extra"), want: "unexpected fixture arguments"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseOptions(test.args, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("parse error = %v, want containing %q", err, test.want)
			}
		})
	}
	if _, err := parseOptions(valid, &bytes.Buffer{}); err != nil {
		t.Fatalf("valid fixture options failed: %v", err)
	}
}

func TestFixtureRegistersRealFlowAndFilesystemCollectionsWithoutRootLeak(t *testing.T) {
	config := testOptions(t)
	if err := os.WriteFile(filepath.Join(config.mounts[0].root, "hello.txt"), []byte("fixture hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fixture, err := startFixture(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	node := fixture.hub.Node
	defer fixture.Close()

	snapshot := node.Registry().Catalog().Snapshot()
	var catalog protocol.ResourceCatalogV2
	if err := protocol.DecodeJSONPayload(snapshot.Value, protocol.DefaultMaxPayload, &catalog); err != nil {
		t.Fatal(err)
	}
	wantFilesystem := map[string]bool{
		config.mounts[0].name: false,
		config.mounts[1].name: false,
		config.mounts[2].name: false,
	}
	wantFlow := map[string]bool{
		protocol.BuiltinFlowDefinitions: false,
		protocol.BuiltinFlowRuns:        false,
	}
	for _, descriptor := range catalog.Resources {
		if _, exists := wantFilesystem[descriptor.ID.Name]; exists {
			if descriptor.Type != protocol.ResourceTypeCollection {
				t.Fatalf("filesystem descriptor %q type = %q", descriptor.ID.Name, descriptor.Type)
			}
			wantFilesystem[descriptor.ID.Name] = true
		}
		if _, exists := wantFlow[descriptor.ID.Name]; exists {
			if descriptor.Type != protocol.ResourceTypeCollection {
				t.Fatalf("Flow descriptor %q type = %q", descriptor.ID.Name, descriptor.Type)
			}
			wantFlow[descriptor.ID.Name] = true
		}
	}
	for name, found := range wantFilesystem {
		if !found {
			t.Errorf("filesystem Resource %q is missing from the catalog", name)
		}
	}
	for name, found := range wantFlow {
		if !found {
			t.Errorf("Flow Resource %q is missing from the catalog", name)
		}
	}
	for _, mount := range config.mounts {
		encoded, err := json.Marshal(mount.root)
		if err != nil {
			t.Fatal(err)
		}
		needle := encoded[1 : len(encoded)-1]
		if bytes.Contains(snapshot.Value, needle) || bytes.Contains(snapshot.Value, []byte(filepath.ToSlash(mount.root))) {
			t.Fatalf("catalog leaked physical root for Resource %q", mount.name)
		}
	}

	filesystemPage := listCollection(t, node.Registry(), protocol.ResourceID{Owner: node.ID(), Name: config.mounts[0].name})
	if len(filesystemPage.Members) != 1 || filesystemPage.Members[0].Key != "hello.txt" {
		t.Fatalf("unexpected filesystem fixture page: %#v", filesystemPage)
	}
	flowPage := listCollection(t, node.Registry(), protocol.ResourceID{Owner: node.ID(), Name: protocol.BuiltinFlowDefinitions})
	if len(flowPage.Members) != 0 {
		t.Fatalf("new fixture Flow store is not empty: %#v", flowPage.Members)
	}

	resourceIDs := make([]protocol.ResourceID, 0, len(config.mounts))
	for _, mount := range config.mounts {
		resourceIDs = append(resourceIDs, protocol.ResourceID{Owner: node.ID(), Name: mount.name})
	}
	if err := fixture.Close(); err != nil {
		t.Fatal(err)
	}
	for _, id := range resourceIDs {
		if _, exists := node.Registry().Resolve(id); exists {
			t.Errorf("filesystem Resource %q remained registered after close", id.Name)
		}
	}
	select {
	case <-node.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("fixture Hub node did not close")
	}
	if err := fixture.Close(); err != nil {
		t.Fatalf("fixture close is not idempotent: %v", err)
	}
}

func TestFixtureStateIsCompatibleWithCanonicalOfflinePolicy(t *testing.T) {
	config := testOptions(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fixture, err := startFixture(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.Close(); err != nil {
		t.Fatal(err)
	}
	state, err := hostconfig.Open(config.stateDirectory, protocol.NodeID(config.nodeID))
	if err != nil {
		t.Fatalf("canonical offline state open failed: %v", err)
	}
	grant := auth.Request{
		Subject: 2,
		Action:  auth.Action(protocol.CapabilityRead),
		Resource: protocol.ResourceID{
			Owner: protocol.NodeID(config.nodeID), Name: protocol.BuiltinResourceCatalog,
		},
	}
	if err := state.Policy.Grant(grant); err != nil {
		t.Fatalf("canonical offline policy grant failed: %v", err)
	}
	restarted, err := startFixture(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	if err := restarted.hub.Runtime.Policy.Authorize(context.Background(), grant); err != nil {
		t.Fatalf("fixture did not load canonical offline policy: %v", err)
	}
}

func TestStartFixtureReportsFilesystemRegistrationFailure(t *testing.T) {
	config := testOptions(t)
	config.mounts[1].root = config.mounts[0].root
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fixture, err := startFixture(ctx, config)
	if fixture != nil {
		_ = fixture.Close()
		t.Fatal("filesystem registration failure returned a running fixture")
	}
	if err == nil || !strings.Contains(err.Error(), "register filesystem fixture") || !strings.Contains(err.Error(), "duplicate canonical root") {
		t.Fatalf("registration error = %v", err)
	}
}

func listCollection(t *testing.T, registry *resource.Registry, id protocol.ResourceID) protocol.CollectionPageV1 {
	t.Helper()
	payload, err := protocol.EncodeJSONPayload(&protocol.CollectionListRequestV1{Version: 1, Limit: 16}, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	result, err := registry.Operate(context.Background(), id, resource.OperationRequest{
		Capability: protocol.CapabilityList,
		Schema:     protocol.SchemaCollectionListRequestV1,
		Payload:    payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	var page protocol.CollectionPageV1
	if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &page); err != nil {
		t.Fatal(err)
	}
	return page
}

func testOptions(t *testing.T) options {
	t.Helper()
	roots := testRoots(t)
	return options{
		stateDirectory: filepath.Join(t.TempDir(), "hub-state"),
		listen:         "127.0.0.1:0",
		nodeID:         1,
		mounts: [3]mountOption{
			{name: "storage/allowed", root: roots[0], label: "Allowed files"},
			{name: "storage/forbidden", root: roots[1], label: "Forbidden files"},
			{name: "storage/misc", root: roots[2], label: "Miscellaneous files"},
		},
	}
}

func testRoots(t *testing.T) [3]string {
	t.Helper()
	base := t.TempDir()
	var roots [3]string
	for index, name := range []string{"allowed", "forbidden", "misc"} {
		roots[index] = filepath.Join(base, name)
		if err := os.Mkdir(roots[index], 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return roots
}

func replaceArgument(args []string, name, value string) []string {
	result := append([]string(nil), args...)
	for index := 0; index+1 < len(result); index++ {
		if result[index] == name {
			result[index+1] = value
			return result
		}
	}
	return append(result, name, value)
}

func TestExecuteRejectsNilStreams(t *testing.T) {
	err := execute(context.Background(), nil, nil, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "output streams") {
		t.Fatalf("execute error = %v", err)
	}
}
