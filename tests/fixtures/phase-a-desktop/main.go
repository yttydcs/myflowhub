// This process adapter runs the real Desktop Go binding against a Hub/Metrics
// pair. It does not replace the Wails UI or perform physical audio acceptance.
package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/sdk/bindings"
	desktop "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

func main() {
	var directory, endpoint, parentKey, permitFile string
	var identityOnly bool
	var expectDenied, serve bool
	flag.StringVar(&directory, "state", "", "independent test state directory")
	flag.StringVar(&endpoint, "endpoint", "", "Hub endpoint")
	flag.StringVar(&parentKey, "parent-key", "", "pinned Hub public key")
	flag.StringVar(&permitFile, "permit", "", "Legacy Join Permit file")
	flag.BoolVar(&identityOnly, "identity", false, "prepare public identity only")
	flag.BoolVar(&expectDenied, "expect-control-denied", false, "verify the latest persisted control revocation")
	flag.BoolVar(&serve, "serve", false, "accept bounded explicit probe/stop lines while keeping this test consumer alive")
	flag.Parse()
	if err := run(directory, endpoint, parentKey, permitFile, identityOnly, expectDenied, serve); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(directory, endpoint, parentKey, permitFile string, identityOnly, expectDenied, serve bool) error {
	if directory == "" {
		return errors.New("test state is required")
	}
	if identityOnly {
		state, err := auth.OpenState(directory, 2)
		if err != nil {
			return err
		}
		defer state.Policy.Close()
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"public_key": base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey)})
	}
	timeout := 15 * time.Second
	if serve {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	key, err := base64.RawStdEncoding.DecodeString(parentKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("invalid Hub key")
	}
	encoded, err := os.ReadFile(permitFile)
	if err != nil {
		return err
	}
	var permit protocol.ProvisioningPermitV1
	if err := protocol.DecodeJSONPayload(encoded, protocol.DefaultMaxPayload, &permit); err != nil {
		return err
	}
	host, err := nodehost.New(ctx, nodehost.Config{StateDirectory: directory, NodeID: 2, Parent: &nodehost.ParentConfig{NodeID: 1, PublicKey: key, Permit: &permit, Driver: tcp.Driver{}, Endpoint: link.Endpoint(endpoint)}})
	if err != nil {
		return err
	}
	defer host.Close()
	if err := host.Start(); err != nil {
		return err
	}
	status, ok := host.ParentStatus()
	if !ok {
		return errors.New("missing parent status")
	}
	core, err := bindings.NewAttachedClient(host.Client(), bindings.PublicIdentity{NodeID: 2, PublicKey: host.State().Identity.PublicKey}, status)
	if err != nil {
		return err
	}
	client, err := desktop.NewAttachedClient(core)
	if err != nil {
		return err
	}
	defer client.Close()
	if !serve {
		return probeConfiguration(client, expectDenied)
	}
	// Only the fixture harness submits probes. Keeping the same process and
	// NodeHost alive lets lifecycle checks detect accidental sibling restarts.
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64), 256)
	for scanner.Scan() {
		switch scanner.Text() {
		case "probe":
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := probeConfiguration(client, expectDenied); err != nil {
				return err
			}
		case "stop":
			return nil
		default:
			return errors.New("fixture accepts only probe or stop")
		}
	}
	return scanner.Err()
}

func probeConfiguration(client *desktop.Client, expectDenied bool) error {
	started := time.Now()
	if err := client.WaitConnected(12000); err != nil {
		return err
	}
	catalog, err := client.CatalogJSON(3, 5000)
	if err != nil {
		return err
	}
	var entries struct {
		Resources []json.RawMessage `json:"resources"`
	}
	if err := json.Unmarshal([]byte(catalog), &entries); err != nil {
		return err
	}
	beforeJSON, err := client.SnapshotJSON(3, metrics.ResourceConfig, 5000)
	if err != nil {
		return err
	}
	var before protocol.VariableValueV1
	if err := json.Unmarshal([]byte(beforeJSON), &before); err != nil {
		return err
	}
	var config metrics.ConfigV1
	if err := protocol.DecodeVariablePayload(metrics.ConfigDefinition(), before.Value, protocol.DefaultMaxPayload, &config); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	subscription, err := client.Subscribe(3, metrics.ResourceConfig, 60000)
	if err != nil {
		return err
	}
	defer client.CancelSubscription(subscription)
	if err := pollEvent(client, subscription); err != nil {
		return err
	}
	config.Settings[0].IntervalMS += 250
	request := metrics.ConfigUpdateV1{Version: 1, ExpectedRevision: config.Revision, Settings: config.Settings, NotificationChannels: config.NotificationChannels}
	input, err := json.Marshal(request)
	if err != nil {
		return err
	}
	_, operationError := client.OperateJSON(3, metrics.ResourceConfigUpdate, "invoke", metrics.SchemaConfigUpdateV1, string(input), 5000)
	if expectDenied {
		if operationError == nil || !strings.Contains(strings.ToLower(operationError.Error()), "forbidden") {
			return fmt.Errorf("latest revocation was not enforced: %v", operationError)
		}
	} else if operationError != nil {
		return operationError
	}
	afterJSON, err := client.SnapshotJSON(3, metrics.ResourceConfig, 5000)
	if err != nil {
		return err
	}
	var after protocol.VariableValueV1
	if err := json.Unmarshal([]byte(afterJSON), &after); err != nil {
		return err
	}
	var saved metrics.ConfigV1
	if err := protocol.DecodeVariablePayload(metrics.ConfigDefinition(), after.Value, protocol.DefaultMaxPayload, &saved); err != nil {
		return err
	}
	if expectDenied {
		if saved.Revision != config.Revision || saved.Settings[0].IntervalMS != config.Settings[0].IntervalMS-250 {
			return errors.New("rejected control modified saved configuration")
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"read": true, "subscribe": true, "configuration_control_denied": true, "elapsed_ms": time.Since(started).Milliseconds(), "wails_ui": false, "physical_audio": false})
	}
	if saved.Revision != config.Revision+1 || saved.Settings[0].IntervalMS != config.Settings[0].IntervalMS {
		return errors.New("configuration control was not independently read back")
	}
	if err := pollEvent(client, subscription); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"catalog_resources": len(entries.Resources), "read": true, "subscribe": true, "configuration_control": true, "elapsed_ms": time.Since(started).Milliseconds(), "wails_ui": false, "physical_audio": false})
}

func pollEvent(client *desktop.Client, id int64) error {
	data, err := client.PollSubscription(id, 5000)
	if err != nil {
		return err
	}
	var item struct {
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return err
	}
	if item.Kind != "event" {
		return fmt.Errorf("expected subscription event, got %s: %s", item.Kind, item.Payload)
	}
	return nil
}
