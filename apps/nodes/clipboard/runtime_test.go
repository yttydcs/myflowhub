package clipboard

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/subscription"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestTwoNodeRuntimeSynchronizesThroughParentWithoutTopics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rootIdentity, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	rootStore, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	admission, err := auth.LoadAdmission(rootIdentity, rootStore, auth.AdmissionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	rootTrust := auth.NewTrustStore()
	_ = rootTrust.Add(1, rootIdentity.PublicKey)
	root, err := node.New(ctx, node.Config{
		Identity: rootIdentity, Trust: rootTrust, Admission: admission, Policy: auth.AllowAll{},
		Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := root.Listen(network, "clipboard-two-node")
	if err != nil {
		t.Fatal(err)
	}

	firstState, _ := auth.OpenState(t.TempDir(), 2)
	secondState, _ := auth.OpenState(t.TempDir(), 3)
	firstPermit, err := admission.Issue(2, firstState.Identity.PublicKey, "clipboard-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	secondPermit, err := admission.Issue(3, secondState.Identity.PublicKey, "clipboard-b", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Start(ctx, RuntimeConfig{
		StateDirectory: firstState.Directory, NodeID: 2, ParentID: 1, ParentKey: rootIdentity.PublicKey,
		Permit: &firstPermit, Driver: network, Endpoint: endpoint,
		Supervisor: node.SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	secondAdapter := newFakeAdapter()
	second, err := Start(ctx, RuntimeConfig{
		StateDirectory: secondState.Directory, NodeID: 3, ParentID: 1, ParentKey: rootIdentity.PublicKey,
		Permit: &secondPermit, Driver: network, Endpoint: endpoint, Adapter: secondAdapter,
		Supervisor: node.SupervisorConfig{MinBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	waitClipboardConnected(t, first)
	waitClipboardConnected(t, second)
	enableClipboard(t, first.Clipboard, false, false, nil)
	enableClipboard(t, second.Clipboard, false, true, []PeerV1{{NodeID: "2", Receive: true}})

	deadline := time.Now().Add(3 * time.Second)
	for attempt := 0; time.Now().Before(deadline); attempt++ {
		text := fmt.Sprintf("two-node-private-%d", attempt)
		if _, err := first.Clipboard.SendText(ctx, text); err != nil {
			t.Fatal(err)
		}
		until := time.Now().Add(150 * time.Millisecond)
		for time.Now().Before(until) {
			if secondAdapter.text() == text {
				status := second.Clipboard.Status()
				if status.LastAction != "applied" || status.LastSizeBytes != len([]byte(text)) || stringsContainBody(status, text) {
					t.Fatalf("unexpected receiving status: %+v", status)
				}
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatalf("clipboard nodes did not synchronize, receiver status: %+v", second.Clipboard.Status())
}

func waitClipboardConnected(t *testing.T, runtime *Runtime) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.Connection.Snapshot().State == "connected" {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("clipboard runtime did not connect: %+v", runtime.Connection.Snapshot())
}

func stringsContainBody(status StatusV1, body string) bool {
	return status.LastError == body || status.LastAction == body || status.PendingHash == body || status.LastHashPrefix == body
}
