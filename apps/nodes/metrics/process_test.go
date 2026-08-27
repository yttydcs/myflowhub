package metrics_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func TestMetricsCommandProcessPublishesToPersistentHub(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the standalone MetricsNode command is the Windows product entry")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	temporary := t.TempDir()
	binary := filepath.Join(temporary, "mfh-metrics.exe")
	build := exec.Command("go", "build", "-o", binary, "./cmd/mfh-metrics")
	build.Dir = root
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build MetricsNode command: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	parent, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: filepath.Join(temporary, "hub"), NodeID: 1,
		Listeners:       []hub.ListenerConfig{{Driver: tcp.Driver{}, Endpoint: link.Endpoint("127.0.0.1:0")}},
		RefreshInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()

	childState := filepath.Join(temporary, "metrics")
	identityCommand := exec.Command(binary, "-identity", "-state", childState, "-id", "42")
	identityOutput, err := identityCommand.Output()
	if err != nil {
		t.Fatal(err)
	}
	var identity struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(identityOutput, &identity); err != nil {
		t.Fatal(err)
	}
	childKey, err := base64.RawStdEncoding.DecodeString(identity.PublicKey)
	if err != nil || len(childKey) != ed25519.PublicKeySize {
		t.Fatalf("invalid child identity output: %s", identityOutput)
	}
	permit, err := parent.Runtime.Admission.Issue(42, ed25519.PublicKey(childKey), "metrics", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	permitPayload, err := protocol.EncodeJSONPayload(&permit, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	permitPath := filepath.Join(temporary, "permit.json")
	if err := os.WriteFile(permitPath, permitPayload, 0o600); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	command := exec.Command(binary,
		"-state", childState, "-id", "42", "-parent-id", "1", "-endpoint", string(parent.Endpoint),
		"-parent-key", base64.RawStdEncoding.EncodeToString(parent.Runtime.Identity.PublicKey), "-permit", permitPath,
		"-connect-timeout", "5s",
	)
	command.Stdout = io.Discard
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	}()

	resourceID := protocol.ResourceID{Owner: 42, Name: metrics.ResourceName(metrics.CPUPercent)}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		attemptCtx, attemptCancel := context.WithTimeout(ctx, 500*time.Millisecond)
		subscription, subscribeErr := parent.Node.Subscribe(attemptCtx, resourceID, time.Minute, 4)
		if subscribeErr == nil {
			select {
			case event, ok := <-subscription.Events:
				subscription.Cancel()
				attemptCancel()
				if !ok {
					continue
				}
				var sample metrics.SampleV1
				if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &sample); err != nil {
					t.Fatal(err)
				}
				if sample.Metric != string(metrics.CPUPercent) || sample.Value == "-1" {
					t.Fatalf("unexpected process metric: %+v", sample)
				}
				return
			case <-attemptCtx.Done():
				subscription.Cancel()
			}
		}
		attemptCancel()
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("MetricsNode process did not publish to Hub: %s", stderr.String())
}
