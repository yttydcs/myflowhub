package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

func TestServerListsCanonicalToolsAndGatesWrites(t *testing.T) {
	client := &desktopbinding.Client{}
	if err := client.Open(t.TempDir(), 91); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := New(client, false)
	if err != nil {
		t.Fatal(err)
	}
	input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}\n" +
		"{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"mfh_command_invoke\",\"arguments\":{\"owner_node_id\":1,\"name\":\"system/config/update\",\"request\":{\"version\":1}}}}\n")
	var output bytes.Buffer
	if err := server.Serve(input, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "mfh_resource_catalog") || !strings.Contains(lines[1], "--allow-write") {
		t.Fatalf("unexpected MCP output: %s", output.String())
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatalf("invalid JSON-RPC response: %s", line)
		}
	}
}

func TestServerRejectsOversizedInput(t *testing.T) {
	client := &desktopbinding.Client{}
	if err := client.Open(t.TempDir(), 92); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, _ := New(client, false)
	err := server.Serve(strings.NewReader(strings.Repeat("x", maxMessageBytes+1)), &bytes.Buffer{})
	if err == nil {
		t.Fatal("oversized MCP input was accepted")
	}
}
