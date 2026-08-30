package androidbinding

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/runtime/auth"
)

func TestHostPersistentTCPRuntime(t *testing.T) {
	host, err := NewHost(t.TempDir(), 81)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := host.IdentityJSON()
	if err != nil || !json.Valid([]byte(identity)) {
		t.Fatalf("invalid host identity: %v (%s)", err, identity)
	}
	if err := host.Start("127.0.0.1:0", ""); err != nil {
		t.Fatal(err)
	}
	status, err := host.StatusJSON()
	if err != nil || !strings.Contains(status, `"running":true`) || !strings.Contains(status, "127.0.0.1:") {
		t.Fatalf("invalid running status: %v (%s)", err, status)
	}
	if err := host.Start("127.0.0.1:0", ""); err == nil {
		t.Fatal("host started twice")
	}
	if err := host.Stop(); err != nil {
		t.Fatal(err)
	}
	status, err = host.StatusJSON()
	if err != nil || !strings.Contains(status, `"running":false`) {
		t.Fatalf("invalid stopped status: %v (%s)", err, status)
	}
}

func TestHostRequiresListenerAndParentTrustBeforeStart(t *testing.T) {
	host, err := NewHost(t.TempDir(), 82)
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Start("", ""); err == nil {
		t.Fatal("host without listener was accepted")
	}
	if err := host.TrustParent(1, "invalid"); err == nil {
		t.Fatal("invalid parent key was accepted")
	}
}

func TestAndroidClientJoinsHostedTCPHub(t *testing.T) {
	host, err := NewHost(t.TempDir(), 83)
	if err != nil {
		t.Fatal(err)
	}
	childDirectory := t.TempDir()
	childState, err := auth.OpenState(childDirectory, 84)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := host.state.Admission.Issue(84, childState.Identity.PublicKey, "member", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	permitJSON, _ := json.Marshal(permit)
	if err := host.Start("127.0.0.1:0", ""); err != nil {
		t.Fatal(err)
	}
	defer host.Stop()
	client, err := NewClient(childDirectory, 84)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.TrustParent(83, base64.RawStdEncoding.EncodeToString(host.state.Identity.PublicKey)); err != nil {
		t.Fatal(err)
	}
	endpoint := string(host.runtime.Endpoints[0])
	if err := client.StartTCP(endpoint, 83, string(permitJSON)); err != nil {
		t.Fatal(err)
	}
	if err := client.WaitConnected(5_000); err != nil {
		t.Fatalf("Android client did not join hosted Hub at %s: %v", endpoint, err)
	}
	status, err := client.StatusJSON()
	if err != nil || !strings.Contains(status, `"state":"connected"`) || !strings.Contains(status, `"parent_node_id":"83"`) {
		t.Fatalf("unexpected joined status: %v (%s)", err, status)
	}
	if client.leaf == nil || client.leaf.host.Client() == nil || client.leaf.facade == nil {
		t.Fatal("Android client did not compose a NodeHost and attached facade")
	}
	if err := client.leaf.facade.Close(); err != nil {
		t.Fatalf("close attached facade: %v", err)
	}
	if _, err := client.leaf.host.Client().NodeID(); err != nil {
		t.Fatalf("closing attached facade closed its Host: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := nodehost.New(context.Background(), nodehost.Config{StateDirectory: childDirectory, NodeID: 84})
	if err != nil {
		t.Fatalf("terminal Android client close did not release its Host state directory: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAndroidClientFailedStartReleasesNodeHostReservation(t *testing.T) {
	parent, err := NewHost(t.TempDir(), 85)
	if err != nil {
		t.Fatal(err)
	}
	if err := parent.Start("127.0.0.1:0", ""); err != nil {
		t.Fatal(err)
	}
	defer parent.Stop()

	directory := t.TempDir()
	client, err := NewClient(directory, 86)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.TrustParent(85, base64.RawStdEncoding.EncodeToString(parent.state.Identity.PublicKey)); err != nil {
		t.Fatal(err)
	}
	other, err := auth.OpenState(t.TempDir(), 87)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := parent.state.Admission.Issue(87, other.Identity.PublicKey, "other-child", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	permitJSON, err := json.Marshal(permit)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.StartTCP(string(parent.runtime.Endpoints[0]), 85, string(permitJSON)); err == nil {
		t.Fatal("Android client accepted a permit bound to another child")
	}
	probe, err := nodehost.New(context.Background(), nodehost.Config{StateDirectory: directory, NodeID: 86})
	if err != nil {
		t.Fatalf("failed Android start leaked the NodeHost state-directory reservation: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
}
