package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/feature/management"
	hostconfig "github.com/yttydcs/myflowhub/host/config"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/tree"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

// TestTopologyGUIFixture is an opt-in, real TCP fixture for a separate Wails
// process. Use a fresh absolute directory below this worktree's out/qa-topology/gui:
//
//	$env:MFH_TOPOLOGY_GUI_DIR = '<worktree>/out/qa-topology/gui/<fresh-run>'
//	$env:GOWORK = 'off'
//	go test ./apps/desktop -run '^TestTopologyGUIFixture$' -count=1 -v -timeout=11m
//
// Once fixture.json appears, launch Wails with MFH_DESKTOP_CONFIG_DIR set to
// that same directory. Creating STOP there ends the fixture; it also expires
// after ten minutes. Optionally set MFH_TOPOLOGY_GUI_DENY_CHILDREN_NODE=4 for a
// Forbidden expansion while keeping subtree and catalog independently readable.
// Node state is temporary; Desktop state and the public manifest are retained.
func TestTopologyGUIFixture(t *testing.T) {
	rawRoot := os.Getenv("MFH_TOPOLOGY_GUI_DIR")
	if rawRoot == "" {
		t.Skip("set MFH_TOPOLOGY_GUI_DIR to opt into the isolated TCP GUI fixture")
	}
	_, source, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(source) {
		t.Fatal("cannot locate worktree; run go test without -trimpath")
	}
	worktree, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(source), "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	configRoot := topologyGUICheckPath(t, worktree, rawRoot)
	guiRoot := filepath.Join(worktree, "out", "qa-topology", "gui")
	if relative, err := filepath.Rel(guiRoot, configRoot); err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("MFH_TOPOLOGY_GUI_DIR must be a fresh directory strictly inside this worktree's out/qa-topology/gui")
	}
	deniedNode := 0
	if raw := os.Getenv("MFH_TOPOLOGY_GUI_DENY_CHILDREN_NODE"); raw != "" {
		deniedNode, err = strconv.Atoi(raw)
		if err != nil || deniedNode < 1 || deniedNode > 6 {
			t.Fatal("MFH_TOPOLOGY_GUI_DENY_CHILDREN_NODE must be an integer from 1 to 6")
		}
	}
	if err := os.MkdirAll(configRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(configRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("fixture Desktop directory must be empty; choose a fresh run directory, do not reset existing profiles")
	}
	qaRoot := topologyGUICheckPath(t, worktree, filepath.Join(worktree, "out", "qa-topology"))
	stateRoot, err := os.MkdirTemp(qaRoot, "gui-node-state-")
	if err != nil {
		t.Fatal(err)
	}
	stateRoot = topologyGUICheckPath(t, worktree, stateRoot)
	t.Cleanup(func() {
		// Only the newly allocated node-state directory is removed, after all
		// node and policy cleanup callbacks. Desktop credentials are retained.
		topologyGUICheckPath(t, worktree, stateRoot)
		if err := os.RemoveAll(stateRoot); err != nil {
			t.Errorf("remove isolated node state: %v", err)
		}
	})
	expires := time.Now().Add(10 * time.Minute)
	if deadline, ok := t.Deadline(); ok && deadline.Add(-5*time.Second).Before(expires) {
		expires = deadline.Add(-5 * time.Second)
	}
	ctx, cancel := context.WithDeadline(context.Background(), expires)
	defer cancel()
	states := make(map[protocol.NodeID]*hostconfig.Runtime)
	nodes := make(map[protocol.NodeID]*node.Node)
	endpoints := make(map[protocol.NodeID]link.Endpoint)
	for id := protocol.NodeID(1); id <= 6; id++ {
		directory := topologyGUICheckPath(t, worktree, filepath.Join(stateRoot, fmt.Sprintf("node-%d", id)))
		state, err := hostconfig.Open(directory, id)
		if err != nil {
			t.Fatalf("open node %d: %v", id, err)
		}
		states[id] = state
		t.Cleanup(func() {
			if err := state.Policy.Close(); err != nil {
				t.Errorf("close node %d policy: %v", id, err)
			}
		})
		if _, err := state.Settings.Replace(protocol.ManagementConfigUpdateV1{
			Version: 1, ExpectedRevision: state.Settings.Snapshot().Revision,
			DisplayName: fmt.Sprintf("GUI TCP Node %d", id), Values: map[string]string{},
		}); err != nil {
			t.Fatal(err)
		}
	}
	for id := protocol.NodeID(1); id <= 6; id++ {
		state := states[id]
		for peer := protocol.NodeID(1); peer <= 6; peer++ {
			if err := state.Trust.Add(peer, states[peer].Identity.PublicKey); err != nil {
				t.Fatal(err)
			}
		}
		current, err := node.New(ctx, node.Config{
			Identity: state.Identity, Trust: state.Trust, Policy: state.Policy, Admission: state.Admission,
		})
		if err != nil {
			t.Fatalf("create node %d: %v", id, err)
		}
		nodes[id] = current
		t.Cleanup(func() {
			if err := current.Close(); err != nil {
				t.Errorf("close node %d: %v", id, err)
			}
		})
		address, err := current.Listen(tcp.Driver{}, "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen node %d: %v", id, err)
		}
		endpoints[id] = address
	}
	for id := protocol.NodeID(2); id <= 6; id++ {
		if err := nodes[id].ConnectParent(ctx, tcp.Driver{}, endpoints[id-1], id-1); err != nil {
			t.Fatalf("connect node %d to %d: %v", id, id-1, err)
		}
	}
	topologyGUIWait(t, ctx, "six-layer route propagation", func() bool {
		for id := protocol.NodeID(1); id < 6; id++ {
			route, err := nodes[id].Tree().RouteTo(6)
			if err != nil || route.Kind != tree.DirectionDown {
				return false
			}
		}
		return true
	})
	controllers := make([]*management.Controller, 0, 6)
	for id := protocol.NodeID(1); id <= 6; id++ {
		state := states[id]
		controller, err := management.Register(management.Config{
			Node: nodes[id], Admission: state.Admission, Trust: state.Trust, Policy: state.Policy,
			Settings: state.Settings, RevokeNode: state.RevokeNode,
		})
		if err != nil {
			t.Fatalf("register management %d: %v", id, err)
		}
		controllers = append(controllers, controller)
		for _, permission := range []struct {
			name       string
			capability protocol.CapabilityID
		}{
			{protocol.BuiltinManagementTopology, protocol.CapabilityChildren},
			{protocol.BuiltinManagementTopology, protocol.CapabilitySubtree},
			{protocol.BuiltinResourceCatalog, protocol.CapabilityRead},
		} {
			if int(id) == deniedNode && permission.capability == protocol.CapabilityChildren {
				continue
			}
			if err := states[1].Policy.Grant(auth.Request{
				Subject: 7, Resource: protocol.ResourceID{Owner: id, Name: permission.name}, Capability: permission.capability,
			}); err != nil {
				t.Fatalf("grant Node7 %s/%s on %d: %v", permission.name, permission.capability, id, err)
			}
		}
	}
	refresh := func() {
		for index, controller := range controllers {
			if err := controller.Refresh(); err != nil {
				t.Fatalf("refresh management %d: %v", index+1, err)
			}
		}
	}

	app, err := NewApp(configRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.Close(); err != nil {
			t.Errorf("close preparation App: %v", err)
		}
	})
	profile := Profile{
		ID: "topology-gui", Name: "Isolated six-layer TCP topology", EnrollmentMode: "legacy",
		NodeID: "7", ParentNodeID: "1", Endpoint: string(endpoints[1]), AutoConnect: true,
		ParentPublicKey: base64.RawStdEncoding.EncodeToString(states[1].Identity.PublicKey),
	}
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	preparedJSON, err := app.PrepareProfileJSON(string(profileJSON))
	if err != nil {
		t.Fatalf("prepare public Desktop profile: %v", err)
	}
	var prepared preparedProfile
	if err := json.Unmarshal([]byte(preparedJSON), &prepared); err != nil {
		t.Fatal(err)
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(prepared.Identity.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || prepared.Identity.NodeID != "7" {
		t.Fatal("prepared profile did not expose a valid Node7 public identity")
	}
	permit, err := states[1].Admission.Issue(7, ed25519.PublicKey(publicKey), "topology-gui", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	permitJSON, err := protocol.EncodeJSONPayload(&permit, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	loginJSON, err := json.Marshal(LoginRequest{Profile: profile, PermitJSON: string(permitJSON)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.LoginJSON(string(loginJSON)); err != nil {
		t.Fatalf("admit isolated Desktop Node7: %v", err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}

	// Exercise the same public startup hook used by Wails, without a Wails
	// event bridge. Startup(nil) still performs real asynchronous auto-connect.
	restored, err := NewApp(configRoot)
	if err != nil {
		t.Fatalf("restore Desktop configuration: %v", err)
	}
	t.Cleanup(func() {
		if err := restored.Close(); err != nil {
			t.Errorf("close restored App: %v", err)
		}
	})
	restored.Startup(nil)
	topologyGUIWait(t, ctx, "Desktop auto-connect without another permit", func() bool {
		raw, err := restored.StatusJSON()
		if err != nil {
			t.Fatal(err)
		}
		var status struct {
			State string `json:"state"`
		}
		if err := json.Unmarshal([]byte(raw), &status); err != nil {
			t.Fatal(err)
		}
		return status.State == "connected"
	})
	refresh()
	for owner := 1; owner <= 6; owner++ {
		id := strconv.Itoa(owner)
		if _, err := restored.CatalogJSON(id); err != nil {
			t.Fatalf("restored catalog %s: %v", id, err)
		}
		_, err := restored.OperateJSON(id, protocol.BuiltinManagementTopology, string(protocol.CapabilityChildren),
			protocol.SchemaManagementTopologyChildrenRequestV1, `{"version":1,"depth":1}`)
		if owner == deniedNode {
			var failure *sdk.Error
			if !errors.As(err, &failure) || failure.Code != protocol.CodeForbidden {
				t.Fatalf("expected explicit Forbidden for node %d children", owner)
			}
		} else if err != nil {
			t.Fatalf("restored children %s: %v", id, err)
		}
	}
	fullJSON, err := restored.OperateJSON("1", protocol.BuiltinManagementTopology, string(protocol.CapabilitySubtree),
		protocol.SchemaManagementTopologyQueryRequestV1, `{"version":1,"depth":0}`)
	if err != nil {
		t.Fatalf("restored full subtree: %v", err)
	}
	var full struct {
		Schema  string                             `json:"schema"`
		Payload protocol.ManagementTopologyQueryV1 `json:"payload"`
	}
	if err := json.Unmarshal([]byte(fullJSON), &full); err != nil {
		t.Fatal(err)
	}
	if full.Schema != protocol.SchemaManagementTopologyQueryV1 || full.Payload.RootNodeID != "1" || full.Payload.Depth != 0 || len(full.Payload.Nodes) != 7 {
		t.Fatal("restored Desktop did not read the complete six-layer tree plus Node7")
	}
	for _, current := range full.Payload.Nodes {
		id, err := strconv.Atoi(current.NodeID)
		if err != nil || id < 1 || id > 7 {
			t.Fatal("unexpected node in isolated topology")
		}
		parent := ""
		if id > 1 {
			parent = strconv.Itoa(id - 1)
		}
		if id == 7 {
			parent = "1"
		}
		if current.ParentID != parent {
			t.Fatalf("node %d parent is %s, expected %s", id, current.ParentID, parent)
		}
	}
	if err := restored.Close(); err != nil {
		t.Fatal(err)
	}

	stopFile := topologyGUICheckPath(t, worktree, filepath.Join(configRoot, "STOP"))
	manifestPath := topologyGUICheckPath(t, worktree, filepath.Join(configRoot, "fixture.json"))
	manifest := struct {
		ConfigDirectory    string            `json:"config_directory"`
		DesktopNodeID      int               `json:"desktop_node_id"`
		DesktopParentID    int               `json:"desktop_parent_id"`
		Endpoints          map[string]string `json:"endpoints"`
		DeniedChildrenNode int               `json:"denied_children_node,omitempty"`
		StopFile           string            `json:"stop_file"`
		ExpiresAt          time.Time         `json:"expires_at"`
		AutoConnectChecked bool              `json:"auto_connect_checked"`
	}{configRoot, 7, 1, make(map[string]string), deniedNode, stopFile, expires.UTC(), true}
	for id, address := range endpoints {
		manifest.Endpoints[strconv.FormatUint(uint64(id), 10)] = string(address)
	}
	publicJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(publicJSON, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("MFH_TOPOLOGY_GUI_READY manifest=%s config=%s auto_connect_checked=true", manifestPath, configRoot)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Log("GUI fixture reached its bounded deadline")
			return
		case <-ticker.C:
			if _, err := os.Lstat(stopFile); err == nil {
				t.Log("GUI fixture stopped by STOP file")
				return
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			refresh()
		}
	}
}

func topologyGUICheckPath(t *testing.T, worktree, target string) string {
	t.Helper()
	if !filepath.IsAbs(target) {
		t.Fatal("GUI fixture paths must be absolute")
	}
	target = filepath.Clean(target)
	qaRoot := filepath.Join(worktree, "out", "qa-topology")
	relative, err := filepath.Rel(qaRoot, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("refusing GUI fixture path outside this worktree's out/qa-topology")
	}
	// Reject pre-existing symlinks/junctions along the write path rather than
	// allowing an apparent workspace path to redirect into user configuration.
	for current := target; current != worktree; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("GUI fixture paths must not contain symlinks or junctions")
		}
		if filepath.Dir(current) == current {
			t.Fatal("GUI fixture path did not resolve beneath its worktree")
		}
	}
	return target
}

func topologyGUIWait(t *testing.T, ctx context.Context, description string, ready func() bool) {
	t.Helper()
	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for !ready() {
		select {
		case <-ctx.Done():
			t.Fatalf("%s: fixture deadline reached", description)
		case <-timeout.C:
			t.Fatalf("%s: timed out", description)
		case <-ticker.C:
		}
	}
}
