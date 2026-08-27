package clipboard_test

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
	"strings"
	"testing"
	"time"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func TestClipboardCommandProcessPublishesWithoutLoggingBody(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the standalone ClipboardNode command is the Windows product entry")
	}
	rootDirectory, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	temporary := t.TempDir()
	binary := filepath.Join(temporary, "mfh-clipboard.exe")
	build := exec.Command("go", "build", "-o", binary, "./cmd/mfh-clipboard")
	build.Dir = rootDirectory
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build ClipboardNode command: %v\n%s", err, output)
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
	childDirectory := filepath.Join(temporary, "clipboard")
	identityCommand := exec.Command(binary, "-identity", "-state", childDirectory, "-id", "43")
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
	state, err := auth.OpenState(childDirectory, 43)
	if err != nil {
		t.Fatal(err)
	}
	config := clipboard.DefaultConfig()
	config.Enabled = true
	if err := state.Store.Save("clipboard.json", config); err != nil {
		t.Fatal(err)
	}
	permit, err := parent.Runtime.Admission.Issue(43, ed25519.PublicKey(childKey), "clipboard", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	permitPayload, _ := protocol.EncodeJSONPayload(&permit, protocol.DefaultMaxPayload)
	permitPath := filepath.Join(temporary, "permit.json")
	if err := os.WriteFile(permitPath, permitPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command := exec.Command(binary,
		"-state", childDirectory, "-id", "43", "-parent-id", "1", "-endpoint", string(parent.Endpoint),
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
	resourceID := protocol.ResourceID{Owner: 43, Name: clipboard.ResourceEvents}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		attemptCtx, attemptCancel := context.WithTimeout(ctx, 500*time.Millisecond)
		current, subscribeErr := parent.Node.Subscribe(attemptCtx, resourceID, time.Minute, 4)
		if subscribeErr == nil {
			secret := "process clipboard private body"
			request, _ := protocol.EncodeJSONPayload(&clipboard.SendV1{Version: 1, Text: secret}, protocol.DefaultMaxPayload)
			if _, invokeErr := parent.Node.Invoke(attemptCtx, protocol.ResourceID{Owner: 43, Name: clipboard.CommandSend}, request); invokeErr == nil {
				select {
				case event := <-current.Events:
					current.Cancel()
					attemptCancel()
					var payload clipboard.TextEventV1
					if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &payload); err != nil || payload.Text != secret {
						t.Fatalf("unexpected process clipboard event: %+v (%v)", payload, err)
					}
					if strings.Contains(stderr.String(), secret) {
						t.Fatalf("clipboard body leaked to process log: %s", stderr.String())
					}
					return
				case <-attemptCtx.Done():
				}
			}
			current.Cancel()
		}
		attemptCancel()
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("ClipboardNode process did not publish through Hub: %s", stderr.String())
}
