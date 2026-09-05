package nodehost

import (
	"context"
	"crypto/ed25519"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestOfflineHostOwnsOneNodeClientAndRegistry(t *testing.T) {
	identity, err := auth.GenerateIdentity(41)
	if err != nil {
		t.Fatal(err)
	}
	identityStore := &memoryIdentityStore{identity: identity, found: true}
	directory := t.TempDir()
	host, err := New(context.Background(), Config{
		StateDirectory: directory,
		NodeID:         identity.NodeID,
		IdentityStore:  identityStore,
	})
	if err != nil {
		t.Fatal(err)
	}
	if host.ID() != identity.NodeID || host.Node() == nil || host.State() == nil {
		t.Fatalf("host accessors do not expose the configured runtime: id=%d node=%p state=%p", host.ID(), host.Node(), host.State())
	}
	if host.Client() == nil || host.Client() != host.Client() {
		t.Fatal("host did not retain one attached SDK client")
	}
	if host.Resources() != host.Node().Registry() {
		t.Fatal("host resources do not reference the node registry")
	}
	if host.Role() != RoleOffline {
		t.Fatalf("unexpected offline role %q", host.Role())
	}

	variableID := protocol.ResourceID{Owner: host.ID(), Name: "device/status"}
	variable, err := resource.NewVariable(resource.VariableDescriptor(variableID, "application/octet-stream", "test.status.v1", "test.status.read", 64), []byte("ready"))
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Resources().Register(variable); err != nil {
		t.Fatal(err)
	}
	if err := host.Start(); err != nil {
		t.Fatal(err)
	}
	event, err := host.Client().Snapshot(context.Background(), variableID)
	if err != nil || string(event.Value) != "ready" {
		t.Fatalf("attached client did not use the host node: event=%+v err=%v", event, err)
	}
	if err := host.Start(); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("repeated Start returned %v", err)
	}
	if _, ok := host.Parent(); ok {
		t.Fatal("offline host unexpectedly reports a parent")
	}
	if endpoints := host.Endpoints(); len(endpoints) != 0 {
		t.Fatalf("offline host has endpoints: %v", endpoints)
	}

	if _, err := New(context.Background(), Config{StateDirectory: directory, NodeID: identity.NodeID, IdentityStore: identityStore}); err == nil {
		t.Fatal("concurrent host was allowed to own the same state directory")
	}
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	if err := host.Close(); err != nil {
		t.Fatalf("repeated Close returned %v", err)
	}
	if err := host.Start(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Start after Close returned %v", err)
	}
	reopened, err := New(context.Background(), Config{StateDirectory: directory, NodeID: identity.NodeID, IdentityStore: identityStore})
	if err != nil {
		t.Fatalf("state directory was not released by Close: %v", err)
	}
	_ = reopened.Close()
}

func TestHostTopologyRolesAndParentObservation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	network := memory.NewNetwork()
	defer network.Close()

	root, err := New(ctx, Config{
		StateDirectory: t.TempDir(), NodeID: 1,
		Listeners: []ListenerConfig{{Driver: network, Endpoint: "root"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if root.Role() != RoleRoot {
		t.Fatalf("unexpected root role %q", root.Role())
	}
	if err := root.Start(); err != nil {
		t.Fatal(err)
	}
	if got := root.Endpoints(); !reflect.DeepEqual(got, []link.Endpoint{"root"}) {
		t.Fatalf("unexpected root endpoints: %v", got)
	}

	leaf, err := New(ctx, Config{
		StateDirectory: t.TempDir(), NodeID: 2,
		Parent: &ParentConfig{
			NodeID: 1, PublicKey: root.State().Identity.PublicKey, Driver: network, Endpoint: "root",
			Supervisor: node.SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer leaf.Close()
	if err := root.State().Trust.Add(leaf.ID(), leaf.State().Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	if leaf.Role() != RoleLeaf {
		t.Fatalf("unexpected leaf role %q", leaf.Role())
	}
	before, ok := leaf.Parent()
	if !ok || before.State != node.ConnectionDisconnected || before.Parent != root.ID() {
		t.Fatalf("unexpected pre-start parent snapshot: %+v, %t", before, ok)
	}
	if _, err := leaf.WaitParentChange(ctx, 0); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("pre-start parent wait returned %v", err)
	}
	parentStatus, ok := leaf.ParentStatus()
	if !ok || parentStatus.Snapshot().State != sdk.ConnectionDisconnected {
		t.Fatalf("unexpected SDK parent status before Start: %+v, %t", parentStatus, ok)
	}
	if _, err := parentStatus.WaitChange(ctx, 0); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("pre-start SDK parent wait returned %v", err)
	}
	if err := leaf.Start(); err != nil {
		t.Fatal(err)
	}
	connected := waitParentState(t, leaf, node.ConnectionConnected, 2*time.Second)
	if current := parentStatus.Snapshot(); current.State != sdk.ConnectionConnected || current.Parent != root.ID() || current.Generation != connected.Generation {
		t.Fatalf("SDK parent status did not mirror Host: %+v", current)
	}
	statusWaitCtx, statusWaitCancel := context.WithTimeout(ctx, time.Second)
	defer statusWaitCancel()
	statusChanged, err := parentStatus.WaitChange(statusWaitCtx, connected.Generation-1)
	if err != nil || statusChanged.Generation < connected.Generation {
		t.Fatalf("SDK parent status wait did not expose Host generation: %+v err=%v", statusChanged, err)
	}
	waitCtx, waitCancel := context.WithTimeout(ctx, time.Second)
	defer waitCancel()
	changed, err := leaf.WaitParentChange(waitCtx, connected.Generation-1)
	if err != nil || changed.Generation < connected.Generation {
		t.Fatalf("parent wait did not expose a generation change: %+v err=%v", changed, err)
	}

	relay, err := New(ctx, Config{
		StateDirectory: t.TempDir(), NodeID: 3,
		Parent: &ParentConfig{
			NodeID: 1, PublicKey: root.State().Identity.PublicKey, Driver: network, Endpoint: "root",
			Supervisor: node.SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond},
		},
		Listeners: []ListenerConfig{{Driver: network, Endpoint: "relay"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	if err := root.State().Trust.Add(relay.ID(), relay.State().Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	if relay.Role() != RoleRelay {
		t.Fatalf("unexpected relay role %q", relay.Role())
	}
	if err := relay.Start(); err != nil {
		t.Fatal(err)
	}
	waitParentState(t, relay, node.ConnectionConnected, 2*time.Second)
}

func TestParentSupervisorReconnectsAndPersistsTrust(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	network := memory.NewNetwork()
	defer network.Close()
	rootDirectory := t.TempDir()
	rootConfig := Config{
		StateDirectory: rootDirectory, NodeID: 11,
		Listeners: []ListenerConfig{{Driver: network, Endpoint: "reconnect-root"}},
	}
	root, err := New(ctx, rootConfig)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := New(ctx, Config{
		StateDirectory: t.TempDir(), NodeID: 12,
		Parent: &ParentConfig{
			NodeID: root.ID(), PublicKey: root.State().Identity.PublicKey, Driver: network, Endpoint: "reconnect-root",
			Supervisor: node.SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer leaf.Close()
	if err := root.State().Trust.Add(leaf.ID(), leaf.State().Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	if err := root.Start(); err != nil {
		t.Fatal(err)
	}
	if err := leaf.Start(); err != nil {
		t.Fatal(err)
	}
	first := waitParentState(t, leaf, node.ConnectionConnected, 2*time.Second)
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	waitParentAttempt(t, leaf, first.Attempt+1, 2*time.Second)

	root, err = New(ctx, rootConfig)
	if err != nil {
		t.Fatalf("reopen parent host: %v", err)
	}
	defer root.Close()
	if _, trusted := root.State().Trust.PublicKey(leaf.ID()); !trusted {
		t.Fatal("parent trust was not persisted across host recreation")
	}
	if err := root.Start(); err != nil {
		t.Fatal(err)
	}
	reconnected := waitParentState(t, leaf, node.ConnectionConnected, 2*time.Second)
	if reconnected.Attempt <= first.Attempt {
		t.Fatalf("parent did not reconnect with a later attempt: first=%+v next=%+v", first, reconnected)
	}
}

func TestListenerStartFailureRollsBackInReverseAndCannotRestart(t *testing.T) {
	driver := &recordingDriver{failEndpoint: "third"}
	directory := t.TempDir()
	host, err := New(context.Background(), Config{
		StateDirectory: directory,
		NodeID:         21,
		Listeners: []ListenerConfig{
			{Driver: driver, Endpoint: "first"},
			{Driver: driver, Endpoint: "second"},
			{Driver: driver, Endpoint: "third"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = host.Start()
	if err == nil || !containsAll(err.Error(), "listener 2", "third", "synthetic listen failure") {
		t.Fatalf("unexpected listener start error: %v", err)
	}
	if got := driver.closedEndpoints(); !reflect.DeepEqual(got, []link.Endpoint{"second", "first"}) {
		t.Fatalf("listeners closed in wrong order: %v", got)
	}
	if status := host.Status(); status.Lifecycle != LifecycleFailed || len(status.Endpoints) != 0 {
		t.Fatalf("unexpected failed status: %+v", status)
	}
	if err := host.Start(); !errors.Is(err, ErrStartFailed) {
		t.Fatalf("failed host restarted: %v", err)
	}
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	replacement, err := New(context.Background(), Config{StateDirectory: directory, NodeID: 21})
	if err != nil {
		t.Fatalf("failed Start did not release state ownership: %v", err)
	}
	_ = replacement.Close()
}

func TestCloseUnblocksConcurrentStart(t *testing.T) {
	driver := &blockingDriver{entered: make(chan struct{})}
	host, err := New(context.Background(), Config{
		StateDirectory: t.TempDir(), NodeID: 31,
		Listeners: []ListenerConfig{{Driver: driver, Endpoint: "blocked"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	startDone := make(chan error, 1)
	go func() { startDone <- host.Start() }()
	select {
	case <-driver.entered:
	case <-time.After(time.Second):
		t.Fatal("Start did not enter the blocking listener")
	}
	closeDone := make(chan error, 1)
	go func() { closeDone <- host.Close() }()
	select {
	case err := <-startDone:
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("concurrent Start returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not unblock concurrent Start")
	}
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("concurrent Close did not finish")
	}
	if status := host.Status(); status.Lifecycle != LifecycleStopped {
		t.Fatalf("unexpected final lifecycle: %+v", status)
	}
}

func TestParentConfigRejectsInvalidTrustPermitAndRuntimeOwnership(t *testing.T) {
	identity, err := auth.GenerateIdentity(51)
	if err != nil {
		t.Fatal(err)
	}
	badRuntime := Config{
		StateDirectory: t.TempDir(), NodeID: identity.NodeID,
		Runtime: RuntimeConfig{Session: link.SessionConfig{LocalNode: 99}},
	}
	if _, err := New(context.Background(), badRuntime); err == nil || !containsAll(err.Error(), "local node", "conflicts") {
		t.Fatalf("conflicting runtime owner returned %v", err)
	}

	network := memory.NewNetwork()
	defer network.Close()
	missingTrust := Config{
		StateDirectory: t.TempDir(), NodeID: identity.NodeID,
		Parent: &ParentConfig{NodeID: 52, Driver: network, Endpoint: "parent"},
	}
	if _, err := New(context.Background(), missingTrust); err == nil || !containsAll(err.Error(), "not trusted") {
		t.Fatalf("missing parent trust returned %v", err)
	}

	firstParent, _ := auth.GenerateIdentity(52)
	secondParent, _ := auth.GenerateIdentity(52)
	directory := t.TempDir()
	trusted, err := New(context.Background(), Config{
		StateDirectory: directory, NodeID: identity.NodeID,
		Parent: &ParentConfig{NodeID: 52, PublicKey: firstParent.PublicKey, Driver: network, Endpoint: "parent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = trusted.Close()
	if _, err := New(context.Background(), Config{
		StateDirectory: directory, NodeID: identity.NodeID,
		Parent: &ParentConfig{NodeID: 52, PublicKey: secondParent.PublicKey, Driver: network, Endpoint: "parent"},
	}); err == nil || !containsAll(err.Error(), "different key") {
		t.Fatalf("conflicting parent key returned %v", err)
	}
}

func TestCredentialBackedHostUsesGrantedIdentityWithoutPersistingAnotherIdentity(t *testing.T) {
	network := memory.NewNetwork()
	defer network.Close()
	identity, _ := auth.GenerateIdentity(61)
	parent, _ := auth.GenerateIdentity(60)
	authority, _ := auth.GenerateIdentity(1)
	source := &staticCredentialSource{credential: auth.NodeCredential{
		Identity: identity, ParentNodeID: parent.NodeID, ParentPublicKey: parent.PublicKey,
		AuthorityNodeID: authority.NodeID, AuthorityPublicKey: authority.PublicKey, EnrollmentID: "enrollment-test",
	}}
	directory := t.TempDir()
	host, err := New(context.Background(), Config{
		StateDirectory: directory, CredentialSource: source,
		Parent: &ParentConfig{Driver: network, Endpoint: "credential-parent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	if host.ID() != identity.NodeID || host.Role() != RoleLeaf || len(host.Endpoints()) != 0 {
		t.Fatalf("unexpected credential-backed Host: id=%d role=%q endpoints=%v", host.ID(), host.Role(), host.Endpoints())
	}
	if source.Calls() != 1 {
		t.Fatalf("credential source calls = %d, want 1", source.Calls())
	}
	if key, ok := host.State().Trust.PublicKey(parent.NodeID); !ok || !reflect.DeepEqual(key, parent.PublicKey) {
		t.Fatal("credential parent was not installed into Host trust")
	}
	if _, err := os.Stat(filepath.Join(directory, "state", "identity.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("credential-backed Host created identity.json: %v", err)
	}
}

func TestCredentialBackedHostRejectsAmbiguousOrConflictingConfiguration(t *testing.T) {
	network := memory.NewNetwork()
	defer network.Close()
	identity, _ := auth.GenerateIdentity(71)
	parent, _ := auth.GenerateIdentity(70)
	authority, _ := auth.GenerateIdentity(1)
	credential := auth.NodeCredential{
		Identity: identity, ParentNodeID: parent.NodeID, ParentPublicKey: parent.PublicKey,
		AuthorityNodeID: authority.NodeID, AuthorityPublicKey: authority.PublicKey, EnrollmentID: "enrollment-test",
	}
	base := func() Config {
		return Config{
			StateDirectory: t.TempDir(), CredentialSource: &staticCredentialSource{credential: credential},
			Parent: &ParentConfig{Driver: network, Endpoint: "credential-parent"},
		}
	}

	ambiguous := base()
	ambiguous.IdentityStore = &memoryIdentityStore{}
	if _, err := New(context.Background(), ambiguous); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("ambiguous credential configuration returned %v", err)
	}
	nodeMismatch := base()
	nodeMismatch.NodeID = 72
	if _, err := New(context.Background(), nodeMismatch); err == nil || !strings.Contains(err.Error(), "conflicts with credential NodeID") {
		t.Fatalf("NodeID mismatch returned %v", err)
	}
	parentMismatch := base()
	parentMismatch.Parent.NodeID = 72
	if _, err := New(context.Background(), parentMismatch); err == nil || !strings.Contains(err.Error(), "conflicts with credential parent NodeID") {
		t.Fatalf("parent NodeID mismatch returned %v", err)
	}
	keyMismatch := base()
	otherParent, _ := auth.GenerateIdentity(parent.NodeID)
	keyMismatch.Parent.PublicKey = otherParent.PublicKey
	if _, err := New(context.Background(), keyMismatch); err == nil || !strings.Contains(err.Error(), "configured parent public key conflicts") {
		t.Fatalf("parent key mismatch returned %v", err)
	}
	missingParent := base()
	missingParent.Parent = nil
	if _, err := New(context.Background(), missingParent); err == nil || !strings.Contains(err.Error(), "requires a parent") {
		t.Fatalf("missing credential parent returned %v", err)
	}
	loadFailure := base()
	loadFailure.CredentialSource = &staticCredentialSource{err: errors.New("protected store unavailable")}
	if _, err := New(context.Background(), loadFailure); err == nil || !strings.Contains(err.Error(), "protected store unavailable") {
		t.Fatalf("credential load failure returned %v", err)
	}
}

func waitParentState(t *testing.T, host *Host, state node.ConnectionState, timeout time.Duration) node.ConnectionSnapshot {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		current, ok := host.Parent()
		if ok && current.State == state {
			return current
		}
		time.Sleep(time.Millisecond)
	}
	current, _ := host.Parent()
	t.Fatalf("parent did not reach state %d: %+v", state, current)
	return node.ConnectionSnapshot{}
}

func waitParentAttempt(t *testing.T, host *Host, attempt uint64, timeout time.Duration) node.ConnectionSnapshot {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		current, ok := host.Parent()
		if ok && current.Attempt >= attempt {
			return current
		}
		time.Sleep(time.Millisecond)
	}
	current, _ := host.Parent()
	t.Fatalf("parent did not reach attempt %d: %+v", attempt, current)
	return node.ConnectionSnapshot{}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}

type memoryIdentityStore struct {
	mu       sync.Mutex
	identity auth.Identity
	found    bool
}

type staticCredentialSource struct {
	mu         sync.Mutex
	credential auth.NodeCredential
	err        error
	calls      int
}

func (source *staticCredentialSource) LoadNodeCredential() (auth.NodeCredential, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	source.calls++
	credential := source.credential
	credential.Identity.PublicKey = append(ed25519.PublicKey(nil), credential.Identity.PublicKey...)
	credential.Identity.PrivateKey = append(ed25519.PrivateKey(nil), credential.Identity.PrivateKey...)
	credential.ParentPublicKey = append(ed25519.PublicKey(nil), credential.ParentPublicKey...)
	credential.AuthorityPublicKey = append(ed25519.PublicKey(nil), credential.AuthorityPublicKey...)
	return credential, source.err
}

func (source *staticCredentialSource) Calls() int {
	source.mu.Lock()
	defer source.mu.Unlock()
	return source.calls
}

func (s *memoryIdentityStore) LoadIdentity() (auth.Identity, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneIdentity(s.identity), s.found, nil
}

func (s *memoryIdentityStore) SaveIdentity(identity auth.Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identity = cloneIdentity(identity)
	s.found = true
	return nil
}

func cloneIdentity(identity auth.Identity) auth.Identity {
	identity.PublicKey = append(ed25519.PublicKey(nil), identity.PublicKey...)
	identity.PrivateKey = append(ed25519.PrivateKey(nil), identity.PrivateKey...)
	return identity
}

type recordingDriver struct {
	mu           sync.Mutex
	failEndpoint link.Endpoint
	closed       []link.Endpoint
}

func (d *recordingDriver) Dial(context.Context, link.Endpoint) (link.Pipe, error) {
	return nil, errors.New("recording driver does not dial")
}

func (d *recordingDriver) Listen(_ context.Context, endpoint link.Endpoint) (link.Listener, error) {
	if endpoint == d.failEndpoint {
		return nil, errors.New("synthetic listen failure")
	}
	return &recordingListener{driver: d, endpoint: endpoint, done: make(chan struct{})}, nil
}

func (d *recordingDriver) closedEndpoints() []link.Endpoint {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]link.Endpoint(nil), d.closed...)
}

type recordingListener struct {
	driver   *recordingDriver
	endpoint link.Endpoint
	done     chan struct{}
	once     sync.Once
}

func (l *recordingListener) Accept(ctx context.Context) (link.Pipe, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.done:
		return nil, link.ErrClosed
	}
}

func (l *recordingListener) Addr() link.Endpoint { return l.endpoint }

func (l *recordingListener) Close() error {
	l.once.Do(func() {
		l.driver.mu.Lock()
		l.driver.closed = append(l.driver.closed, l.endpoint)
		l.driver.mu.Unlock()
		close(l.done)
	})
	return nil
}

type blockingDriver struct {
	entered chan struct{}
	once    sync.Once
}

func (d *blockingDriver) Dial(context.Context, link.Endpoint) (link.Pipe, error) {
	return nil, net.ErrClosed
}

func (d *blockingDriver) Listen(ctx context.Context, _ link.Endpoint) (link.Listener, error) {
	d.once.Do(func() { close(d.entered) })
	<-ctx.Done()
	return nil, ctx.Err()
}

var _ link.Driver = (*recordingDriver)(nil)
var _ link.Driver = (*blockingDriver)(nil)
