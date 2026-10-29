package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

// Run the same embedded production UI with opt-in, local-only Go profiling.
// The trigger file is written only after the UI has established its workload.
func TestMain(m *testing.M) {
	if os.Getenv("MFH_PHASE_A_NATIVE_PHASE") == "gui-profile" {
		if err := runPhaseANativeGUIProfile(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runPhaseANativeGUIProfile() (profileErr error) {
	root, err := filepath.Abs(os.Getenv("MFH_PHASE_A_NATIVE_ROOT"))
	if err != nil {
		return err
	}
	temporary, err := filepath.Abs(os.TempDir())
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(temporary, root)
	if err != nil || strings.HasPrefix(relative, "..") || !strings.HasPrefix(filepath.Base(root), "mfh-phase-a-native-") {
		return errors.New("GUI profiling requires the isolated native fixture under TEMP")
	}
	if err := os.Setenv("MFH_DESKTOP_CONFIG_DIR", filepath.Join(root, "desktop")); err != nil {
		return err
	}
	stopped := make(chan struct{})
	finished := make(chan struct{})
	failures := make(chan error, 1)
	report := func(errs ...error) { failures <- errors.Join(errs...) }
	defer func() {
		close(stopped)
		<-finished
		select {
		case profileErr = <-failures:
		default:
		}
	}()
	go func() {
		defer close(finished)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopped:
				return
			case <-ticker.C:
				if _, err := os.Stat(filepath.Join(root, "desktop-profile.start")); os.IsNotExist(err) {
					continue
				} else if err != nil {
					report(err)
					return
				}
			}
			cpu, err := os.OpenFile(filepath.Join(root, "desktop-cpu.pprof"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				report(err)
				return
			}
			if err := pprof.StartCPUProfile(cpu); err != nil {
				cpu.Close()
				report(err)
				return
			}
			select {
			case <-time.After(20 * time.Second):
			case <-stopped:
			}
			pprof.StopCPUProfile()
			if err := cpu.Close(); err != nil {
				report(err)
				return
			}
			for _, name := range []string{"allocs", "goroutine"} {
				file, err := os.OpenFile(filepath.Join(root, "desktop-"+name+".pprof"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
				if err != nil {
					report(err)
					return
				}
				debug := 0
				if name == "goroutine" {
					debug = 1
				}
				err = pprof.Lookup(name).WriteTo(file, debug)
				closeErr := file.Close()
				if err != nil || closeErr != nil {
					report(err, closeErr)
					return
				}
			}
			if err := os.WriteFile(filepath.Join(root, "desktop-profile.done"), []byte("profile captured"), 0o600); err != nil {
				report(err)
			}
			return
		}
	}()
	app, err := NewApp(filepath.Join(root, "desktop"))
	if err != nil {
		return err
	}
	trace, err := os.OpenFile(filepath.Join(root, "desktop-poll-trace.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	tracer := &nativePollTrace{encoder: json.NewEncoder(trace)}
	traceDone := make(chan struct{})
	traceStop := make(chan struct{})
	go func() {
		defer close(traceDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		wrapped := make(map[subscriptionClient]*nativeTracedClient)
		for {
			select {
			case <-traceStop:
				return
			case <-ticker.C:
				app.subscriptions.mu.Lock()
				for id, entry := range app.subscriptions.entries {
					if _, ok := entry.client.(*nativeTracedClient); ok {
						continue
					}
					client := wrapped[entry.client]
					if client == nil {
						client = &nativeTracedClient{subscriptionClient: entry.client, trace: tracer}
						wrapped[entry.client] = client
					}
					entry.client = client
					app.subscriptions.entries[id] = entry
				}
				app.subscriptions.mu.Unlock()
			}
		}
	}()
	err = runDesktop(app)
	close(traceStop)
	<-traceDone
	tracer.mu.Lock()
	defer tracer.mu.Unlock()
	tracer.closed = true
	return errors.Join(err, trace.Close(), tracer.err)
}

// Passive diagnostics only in the opt-in test binary. All inputs still come
// through the production native UI, and event ordering/payloads are unchanged.
type nativePollTrace struct {
	mu      sync.Mutex
	encoder *json.Encoder
	err     error
	closed  bool
}
type nativeTracedClient struct {
	subscriptionClient
	trace *nativePollTrace
}

func (c *nativeTracedClient) PollSubscriptions(ids string, timeout int64) (string, error) {
	start := time.Now()
	result, err := c.subscriptionClient.(batchSubscriptionClient).PollSubscriptions(ids, timeout)
	finished := time.Now()
	var items []desktopbinding.SubscriptionPoll
	decodeErr := json.Unmarshal([]byte(result), &items)
	var observed int64
	if len(items) > 0 {
		var event struct {
			Value struct {
				Observed int64 `json:"observed_at_unix_ms"`
			} `json:"value"`
		}
		if json.Unmarshal(items[len(items)-1].Payload, &event) == nil {
			observed = event.Value.Observed
		}
	}
	c.trace.mu.Lock()
	defer c.trace.mu.Unlock()
	if !c.trace.closed && c.trace.err == nil {
		c.trace.err = c.trace.encoder.Encode(map[string]any{"start_ms": start.UnixMilli(), "end_ms": finished.UnixMilli(), "events": len(items), "bytes": len(result), "observed_ms": observed, "failed": err != nil || decodeErr != nil})
	}
	return result, err
}

// Provision an isolated native acceptance fixture through the same public App
// methods used by deployment. It never opens the user's default Desktop state.
func TestPhaseANativeFixtureProvision(t *testing.T) {
	root := phaseANativeRoot(t)
	if os.Getenv("MFH_PHASE_A_NATIVE_PHASE") == "bootstrap" {
		bootstrapPhaseANativeFixture(t, root)
		return
	}
	provisionPhaseANativeProfile(t, root)
}

// The isolated fixture's maintenance identity is provisioned with its Hub
// stopped. Runtime revocations then use the normal management resource API.
func TestPhaseANativeAdminProvision(t *testing.T) {
	if os.Getenv("MFH_PHASE_A_NATIVE_PHASE") != "admin-prepare" {
		t.Skip("opt-in fixture maintenance identity")
	}
	root := phaseANativeRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "native.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	host, _, err := net.SplitHostPort(config.Endpoint)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		t.Fatal("native fixture endpoint must be loopback")
	}
	connection, err := net.DialTimeout("tcp", config.Endpoint, 100*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatal("stop the isolated Hub before provisioning its maintenance identity")
	}
	hub, err := auth.OpenState(filepath.Join(root, "hub"), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Policy.Close()
	admin, err := auth.OpenState(filepath.Join(root, "admin"), 5)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Policy.Close()
	permit, err := hub.Admission.Issue(5, admin.Identity.PublicKey, "native-acceptance-maintenance", 12*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(permit)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "admin-permit.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Policy.CreateBinding(protocol.PolicyBindingCreateV1{Version: 1, BindingID: "303132333435363738393a3b3c3d3e3f", Subject: "5", DefinitionID: "superadmin", Scope: protocol.PolicyOwnerScopeV1{Kind: protocol.PolicyScopeAuthorityDomain, NodeID: "1"}}, 1); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"system/policy/definitions", "system/policy/bindings"} {
		for _, capability := range []protocol.CapabilityID{"read", "list", "create", "update", "delete"} {
			if err := hub.Policy.Grant(auth.Request{Subject: 5, Action: auth.Action(capability), Capability: capability, Resource: protocol.ResourceID{Owner: 1, Name: name}}); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, capability := range []protocol.CapabilityID{"read", "write"} {
		if err := hub.Policy.Grant(auth.Request{Subject: 5, Action: auth.Action(capability), Capability: capability, Resource: protocol.ResourceID{Owner: 3, Name: "example/foundation/number/0000"}}); err != nil {
			t.Fatal(err)
		}
	}
}

func phaseANativeRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("MFH_PHASE_A_NATIVE_ROOT")
	if root == "" {
		t.Skip("set MFH_PHASE_A_NATIVE_ROOT and MFH_PHASE_A_NATIVE_PHASE for native fixture provisioning")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	temporary, err := filepath.Abs(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(temporary, root)
	if err != nil || strings.HasPrefix(relative, "..") || !strings.HasPrefix(filepath.Base(root), "mfh-phase-a-native-") {
		t.Fatal("native fixture root must be a dedicated mfh-phase-a-native-* directory under TEMP")
	}
	return root
}

func provisionPhaseANativeProfile(t *testing.T, root string) {
	t.Helper()
	var configuration struct {
		Endpoint        string `json:"endpoint"`
		ParentPublicKey string `json:"parent_public_key"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "native.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &configuration); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp(filepath.Join(root, "desktop"))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	profile := Profile{ID: "phase-a-native", Name: "A 原生验证", NodeID: "2", Endpoint: configuration.Endpoint,
		ParentNodeID: "1", ParentPublicKey: configuration.ParentPublicKey, AutoConnect: true}
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	switch os.Getenv("MFH_PHASE_A_NATIVE_PHASE") {
	case "prepare":
		result, err = app.PrepareProfileJSON(string(profileJSON))
	case "login":
		permit, readErr := os.ReadFile(filepath.Join(root, "desktop-permit.json"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		request, encodeErr := json.Marshal(LoginRequest{Profile: profile, PermitJSON: string(permit)})
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		result, err = app.LoginJSON(string(request))
	default:
		t.Fatal("native fixture phase must be prepare or login")
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "native-"+os.Getenv("MFH_PHASE_A_NATIVE_PHASE")+".json"), []byte(result), 0o600); err != nil {
		t.Fatal(err)
	}
}

func bootstrapPhaseANativeFixture(t *testing.T, root string) {
	t.Helper()
	write := func(name string, value any) {
		t.Helper()
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	hub, err := auth.OpenState(filepath.Join(root, "hub"), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Policy.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	parentKey := base64.RawStdEncoding.EncodeToString(hub.Identity.PublicKey)
	write("native.json", map[string]string{"endpoint": endpoint, "parent_public_key": parentKey})
	app, err := NewApp(filepath.Join(root, "desktop"))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	profile := Profile{ID: "phase-a-native", Name: "A 原生验证", NodeID: "2", Endpoint: endpoint,
		ParentNodeID: "1", ParentPublicKey: parentKey, AutoConnect: true}
	rawProfile, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := app.PrepareProfileJSON(string(rawProfile))
	if err != nil {
		t.Fatal(err)
	}
	var prepared preparedProfile
	if err := json.Unmarshal([]byte(raw), &prepared); err != nil {
		t.Fatal(err)
	}
	key, err := base64.RawStdEncoding.DecodeString(prepared.Identity.PublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		t.Fatal("prepared Desktop public identity is invalid")
	}
	permit, err := hub.Admission.Issue(2, key, "native-acceptance-desktop", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	write("desktop-permit.json", permit)
	for index, name := range []string{"example", "metrics"} {
		id := protocol.NodeID(index + 3)
		state, err := auth.OpenState(filepath.Join(root, name), id)
		if err != nil {
			t.Fatal(err)
		}
		permit, err := hub.Admission.Issue(id, state.Identity.PublicKey, "native-acceptance-"+name, time.Hour)
		if err != nil {
			state.Policy.Close()
			t.Fatal(err)
		}
		write(name+"-permit.json", permit)
		if name == "metrics" {
			configuration, err := metrics.DefaultConfig("windows")
			if err != nil {
				t.Fatal(err)
			}
			for i := range configuration.Settings {
				setting := &configuration.Settings[i]
				setting.Enabled = setting.Metric == string(metrics.VolumePercent) || setting.Metric == string(metrics.VolumeMuted)
				if setting.Enabled {
					setting.IntervalMS = 1000
				}
			}
			if err := state.Store.Save("metrics.json", configuration); err != nil {
				t.Fatal(err)
			}
		}
		if err := state.Policy.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for _, owner := range []protocol.NodeID{3, 4} {
		id := "test.native.example"
		rules := []protocol.PolicyRuleV1{{Resource: protocol.PolicyResourceSelectorV1{Kind: "exact", Value: "system/catalog"}, Capability: protocol.PolicyCapabilitySelectorV1{Kind: "exact", Values: []protocol.CapabilityID{"list", "read"}}}}
		if owner == 3 {
			rules = append(rules, protocol.PolicyRuleV1{Resource: protocol.PolicyResourceSelectorV1{Kind: "prefix", Value: "example/foundation"}, Capability: protocol.PolicyCapabilitySelectorV1{Kind: "exact", Values: []protocol.CapabilityID{"read", "subscribe", "write"}}})
		} else {
			id = "test.native.audio"
			for _, name := range []string{"metrics/volume_percent", "metrics/volume_muted"} {
				rules = append(rules, protocol.PolicyRuleV1{Resource: protocol.PolicyResourceSelectorV1{Kind: "exact", Value: name}, Capability: protocol.PolicyCapabilitySelectorV1{Kind: "exact", Values: []protocol.CapabilityID{"read", "subscribe", "write"}}})
			}
		}
		if _, err := hub.Policy.PutDefinition(protocol.PolicyDefinitionPutV1{Version: 1, ID: id, Label: id, Rules: rules}); err != nil {
			t.Fatal(err)
		}
		bindingID := "00112233445566778899aabbccddeeff"
		ownerID := "3"
		if owner == 4 {
			bindingID, ownerID = "10112233445566778899aabbccddeeff", "4"
		}
		if _, err := hub.Policy.CreateBinding(protocol.PolicyBindingCreateV1{Version: 1, BindingID: bindingID, Subject: "2", DefinitionID: id, Scope: protocol.PolicyOwnerScopeV1{Kind: "owner", NodeID: ownerID}}, 1); err != nil {
			t.Fatal(err)
		}
	}
	for _, capability := range []protocol.CapabilityID{"children", "subtree"} {
		if err := hub.Policy.Grant(auth.Request{Subject: 2, Action: auth.Action(capability), Capability: capability, Resource: protocol.ResourceID{Owner: 1, Name: "system/topology"}}); err != nil {
			t.Fatal(err)
		}
	}
}
