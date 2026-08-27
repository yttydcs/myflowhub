package androidbinding

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
	defer client.Close()
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
}
