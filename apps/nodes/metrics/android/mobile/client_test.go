package metricsmobile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func TestClientIdentityIsDurable(t *testing.T) {
	client := NewClient()
	request, _ := json.Marshal(identityRequestV1{Version: 1, StateDirectory: filepath.Join(t.TempDir(), "state"), NodeID: "81"})
	first, err := client.Identity(string(request))
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Identity(string(request))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("identity changed: %s != %s", first, second)
	}
}

func TestClientIdleGuards(t *testing.T) {
	client := NewClient()
	status, err := client.Status()
	if err != nil || status == "" {
		t.Fatalf("idle status: %q %v", status, err)
	}
	if _, err := client.NextAction(); err == nil {
		t.Fatal("expected idle action error")
	}
	if _, err := client.NextNotification(); err == nil {
		t.Fatal("expected idle notification error")
	}
	if err := client.UpdateMetric("memory_percent", "20", ""); err == nil {
		t.Fatal("expected idle update error")
	}
}

func TestClientRejectsUnknownJSONFields(t *testing.T) {
	client := NewClient()
	if _, err := client.Identity(`{"version":1,"state_directory":"x","node_id":"1","unknown":true}`); err == nil {
		t.Fatal("expected strict boundary error")
	}
}

func TestClientStartStopRestartUsesMetricsNodeHostAndFreshLiveState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	parent, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: t.TempDir(), NodeID: 91,
		Listeners: []hub.ListenerConfig{{Driver: tcp.Driver{}, Endpoint: "127.0.0.1:0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()

	directory := filepath.Join(t.TempDir(), "state")
	child, err := auth.OpenState(directory, 92)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := parent.Runtime.Admission.Issue(92, child.Identity.PublicKey, "metrics", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	request, err := json.Marshal(startRequestV1{
		Version: 1, StateDirectory: directory, NodeID: "92", ParentNodeID: "91",
		Endpoint:        string(parent.Endpoint),
		ParentPublicKey: base64.RawStdEncoding.EncodeToString(parent.Runtime.Identity.PublicKey),
		Permit:          &permit,
	})
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient()
	started, err := client.Start(string(request))
	if err != nil {
		t.Fatal(err)
	}
	var initial mobileStatusV1
	if err := json.Unmarshal([]byte(started), &initial); err != nil || !initial.Running {
		t.Fatalf("invalid initial mobile status %q: %v", started, err)
	}
	if client.runtime == nil || client.runtime.Host == nil || client.runtime.SDK != client.runtime.Host.Client() {
		t.Fatal("mobile runtime did not reuse the Metrics NodeHost client")
	}
	waitMobileConnected(t, client)
	if err := client.UpdateMetric("memory_percent", "37", ""); err != nil {
		t.Fatal(err)
	}
	statusJSON, err := client.Status()
	if err != nil {
		t.Fatal(err)
	}
	var status mobileStatusV1
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil || !hasMobileSample(status.Samples, "memory_percent", "37") {
		t.Fatalf("mobile metric operation did not reach the Host registry: %v (%s)", err, statusJSON)
	}
	if err := client.Stop(); err != nil {
		t.Fatal(err)
	}
	stoppedJSON, err := client.Status()
	if err != nil {
		t.Fatal(err)
	}
	var stopped mobileStatusV1
	if err := json.Unmarshal([]byte(stoppedJSON), &stopped); err != nil || stopped.Running || stopped.State != "stopped" {
		t.Fatalf("mobile stop retained live session state: %v (%s)", err, stoppedJSON)
	}

	// A new gomobile object models a new Android process. It must start from a
	// stopped/disconnected live state even though identity and desired settings
	// remain durable outside this object.
	restarted := NewClient()
	freshJSON, err := restarted.Status()
	if err != nil {
		t.Fatal(err)
	}
	var fresh mobileStatusV1
	if err := json.Unmarshal([]byte(freshJSON), &fresh); err != nil || fresh.Running || fresh.State != "stopped" {
		t.Fatalf("new mobile process fabricated a live session: %v (%s)", err, freshJSON)
	}
	if _, err := restarted.Start(string(request)); err != nil {
		t.Fatalf("restart could not reacquire the NodeHost state directory: %v", err)
	}
	waitMobileConnected(t, restarted)
	if err := restarted.Stop(); err != nil {
		t.Fatal(err)
	}
}

func waitMobileConnected(t *testing.T, client *Client) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := client.Status()
		if err != nil {
			t.Fatal(err)
		}
		var status mobileStatusV1
		if err := json.Unmarshal([]byte(raw), &status); err != nil {
			t.Fatal(err)
		}
		if status.State == "connected" {
			return
		}
		if status.State == "failed" {
			t.Fatalf("mobile connection failed: %s", status.LastError)
		}
		time.Sleep(10 * time.Millisecond)
	}
	raw, _ := client.Status()
	t.Fatalf("mobile connection timed out: %s", raw)
}

func hasMobileSample(samples []metrics.SampleV1, metric, value string) bool {
	for _, sample := range samples {
		if sample.Metric == metric && sample.Value == value {
			return true
		}
	}
	return false
}
