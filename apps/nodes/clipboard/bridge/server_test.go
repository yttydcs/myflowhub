package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
)

type unavailableAdapter struct{}

func (unavailableAdapter) ReadText(context.Context) (string, error) { return "", nil }
func (unavailableAdapter) WriteText(context.Context, string) error  { return nil }
func (unavailableAdapter) WatchText(context.Context) (<-chan clipboard.TextObservation, <-chan error, error) {
	return make(chan clipboard.TextObservation), make(chan error), nil
}
func (unavailableAdapter) Close() error { return nil }

func TestBridgeIdentityAndStrictErrors(t *testing.T) {
	server, err := New(func() clipboard.Adapter { return unavailableAdapter{} })
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	identity := requestLine(t, "one", "identity", map[string]any{"version": 1, "state_directory": state, "node_id": "7"})
	unknown := requestLine(t, "two", "unknown", map[string]any{"version": 1})
	malformed := `{"version":1,"id":"three","op":"status","payload":{"version":1},"extra":true}`
	input := strings.NewReader(identity + "\n" + unknown + "\n" + malformed + "\n")
	var output bytes.Buffer
	if err := server.Serve(context.Background(), input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("unexpected bridge responses: %s", output.String())
	}
	var first, second, third responseV1
	_ = json.Unmarshal([]byte(lines[0]), &first)
	_ = json.Unmarshal([]byte(lines[1]), &second)
	_ = json.Unmarshal([]byte(lines[2]), &third)
	if !first.OK || first.ID != "one" || !bytes.Contains(first.Result, []byte(`"public_key"`)) {
		t.Fatalf("unexpected identity response: %+v", first)
	}
	if second.OK || !strings.Contains(second.Error, "unsupported") {
		t.Fatalf("unknown operation was accepted: %+v", second)
	}
	if third.OK || third.Error == "" {
		t.Fatalf("unknown request field was accepted: %+v", third)
	}
}

func TestBridgeRejectsOversizeLine(t *testing.T) {
	server, _ := New(func() clipboard.Adapter { return unavailableAdapter{} })
	input := strings.NewReader(strings.Repeat("x", MaxMessageBytes+2))
	if err := server.Serve(context.Background(), input, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize bridge line was not rejected: %v", err)
	}
}

func requestLine(t *testing.T, id, op string, payload any) string {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{"version": 1, "id": id, "op": op, "payload": payload})
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
