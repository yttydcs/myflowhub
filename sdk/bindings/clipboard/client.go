package clipboardmobile

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
)

const mobilePayloadLimit = 512 * 1024

type Client struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	runtime *clipboard.Runtime
	adapter *mobileAdapter
}

type identityRequestV1 struct {
	Version        int    `json:"version"`
	StateDirectory string `json:"state_directory"`
	NodeID         string `json:"node_id"`
}

func (r identityRequestV1) Validate() error {
	if r.Version != 1 || r.StateDirectory == "" {
		return errors.New("identity request version or state directory is invalid")
	}
	_, err := parseNodeID(r.NodeID)
	return err
}

type startRequestV1 struct {
	Version         int                            `json:"version"`
	StateDirectory  string                         `json:"state_directory"`
	NodeID          string                         `json:"node_id"`
	ParentNodeID    string                         `json:"parent_node_id"`
	Endpoint        string                         `json:"endpoint"`
	ParentPublicKey string                         `json:"parent_public_key"`
	Permit          *protocol.ProvisioningPermitV1 `json:"permit,omitempty"`
}

func (r startRequestV1) Validate() error {
	_, _, _, err := r.values()
	return err
}

func (r startRequestV1) values() (protocol.NodeID, protocol.NodeID, ed25519.PublicKey, error) {
	if r.Version != 1 || r.StateDirectory == "" {
		return 0, 0, nil, errors.New("start request version or state directory is invalid")
	}
	nodeID, err := parseNodeID(r.NodeID)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("node_id: %w", err)
	}
	parentID, err := parseNodeID(r.ParentNodeID)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("parent_node_id: %w", err)
	}
	if err := link.Endpoint(r.Endpoint).Validate(); err != nil {
		return 0, 0, nil, fmt.Errorf("endpoint: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(r.ParentPublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return 0, 0, nil, errors.New("parent_public_key must be a raw-base64 Ed25519 public key")
	}
	if r.Permit != nil {
		if err := r.Permit.Validate(); err != nil {
			return 0, 0, nil, fmt.Errorf("permit: %w", err)
		}
	}
	return nodeID, parentID, ed25519.PublicKey(key), nil
}

type mobileStatusV1 struct {
	Version      int                 `json:"version"`
	Running      bool                `json:"running"`
	State        string              `json:"state"`
	ParentNodeID string              `json:"parent_node_id,omitempty"`
	Endpoint     string              `json:"endpoint,omitempty"`
	Attempt      uint64              `json:"attempt,omitempty"`
	LastError    string              `json:"last_error,omitempty"`
	Clipboard    *clipboard.StatusV1 `json:"clipboard,omitempty"`
}

func NewClient() *Client { return &Client{} }

func (c *Client) Identity(requestJSON string) (string, error) {
	var request identityRequestV1
	if err := protocol.DecodeJSONPayload([]byte(requestJSON), mobilePayloadLimit, &request); err != nil {
		return "", err
	}
	nodeID, _ := parseNodeID(request.NodeID)
	state, err := auth.OpenState(request.StateDirectory, nodeID)
	if err != nil {
		return "", err
	}
	return encodeMobile(map[string]any{
		"version": 1, "node_id": formatNodeID(state.Identity.NodeID),
		"public_key": base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey),
	})
}

func (c *Client) Start(requestJSON string) (string, error) {
	var request startRequestV1
	if err := protocol.DecodeJSONPayload([]byte(requestJSON), mobilePayloadLimit, &request); err != nil {
		return "", err
	}
	nodeID, parentID, parentKey, err := request.values()
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime != nil {
		return "", errors.New("clipboard runtime is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	adapter := newMobileAdapter()
	value, err := clipboard.Start(ctx, clipboard.RuntimeConfig{
		StateDirectory: request.StateDirectory, NodeID: nodeID, ParentID: parentID, ParentKey: parentKey,
		Permit: request.Permit, Endpoint: link.Endpoint(request.Endpoint), Adapter: adapter,
	})
	if err != nil {
		_ = adapter.Close()
		cancel()
		return "", err
	}
	c.cancel = cancel
	c.runtime = value
	c.adapter = adapter
	return encodeMobile(c.statusLocked())
}

func (c *Client) Stop() error {
	c.mu.Lock()
	value := c.runtime
	adapter := c.adapter
	cancel := c.cancel
	c.runtime = nil
	c.adapter = nil
	c.cancel = nil
	c.mu.Unlock()
	if adapter != nil {
		_ = adapter.Close()
	}
	if cancel != nil {
		cancel()
	}
	if value != nil {
		return value.Close()
	}
	return nil
}

func (c *Client) Status() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return encodeMobile(c.statusLocked())
}

func (c *Client) Configuration() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil {
		return "", errors.New("clipboard runtime is not running")
	}
	return encodeMobile(c.runtime.Clipboard.Config())
}

func (c *Client) UpdateConfiguration(requestJSON string) (string, error) {
	var request clipboard.ConfigUpdateV1
	if err := protocol.DecodeJSONPayload([]byte(requestJSON), mobilePayloadLimit, &request); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil {
		return "", errors.New("clipboard runtime is not running")
	}
	value, err := c.runtime.Clipboard.ApplyConfig(context.Background(), request)
	if err != nil {
		return "", err
	}
	return encodeMobile(value)
}

func (c *Client) ObserveText(text string) error {
	c.mu.Lock()
	adapter := c.adapter
	c.mu.Unlock()
	if adapter == nil {
		return errors.New("clipboard runtime is not running")
	}
	return adapter.observe(text)
}

func (c *Client) SendText(text string) (string, error) {
	c.mu.Lock()
	runtime := c.runtime
	c.mu.Unlock()
	if runtime == nil {
		return "", errors.New("clipboard runtime is not running")
	}
	value, err := runtime.Clipboard.SendText(context.Background(), text)
	if err != nil {
		return "", err
	}
	return encodeMobile(value)
}

func (c *Client) NextWrite() (string, error) {
	c.mu.Lock()
	adapter := c.adapter
	c.mu.Unlock()
	if adapter == nil {
		return "", errors.New("clipboard runtime is not running")
	}
	action, ok := adapter.next()
	if !ok {
		return "", nil
	}
	return encodeMobile(action)
}

func (c *Client) CompleteWrite(actionID, errorMessage string) error {
	c.mu.Lock()
	adapter := c.adapter
	c.mu.Unlock()
	if adapter == nil {
		return errors.New("clipboard runtime is not running")
	}
	return adapter.complete(actionID, errorMessage)
}

func (c *Client) ApplyPending(eventID string) (string, error) {
	c.mu.Lock()
	runtime := c.runtime
	c.mu.Unlock()
	if runtime == nil {
		return "", errors.New("clipboard runtime is not running")
	}
	value, err := runtime.Clipboard.ApplyPending(context.Background(), eventID)
	if err != nil {
		return "", err
	}
	return encodeMobile(value)
}

func (c *Client) History() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil {
		return "", errors.New("clipboard runtime is not running")
	}
	return encodeMobile(c.runtime.Clipboard.History())
}

func (c *Client) ClearHistory() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil {
		return "", errors.New("clipboard runtime is not running")
	}
	value, err := c.runtime.Clipboard.ClearHistory()
	if err != nil {
		return "", err
	}
	return encodeMobile(value)
}

func (c *Client) statusLocked() mobileStatusV1 {
	status := mobileStatusV1{Version: 1, State: "stopped"}
	if c.runtime == nil {
		return status
	}
	snapshot := c.runtime.Connection.Snapshot()
	clipboardStatus := c.runtime.Clipboard.Status()
	status.Running = true
	status.State = string(snapshot.State)
	status.ParentNodeID = formatNodeID(snapshot.Parent)
	status.Endpoint = snapshot.Endpoint
	status.Attempt = snapshot.Attempt
	status.LastError = snapshot.LastError
	status.Clipboard = &clipboardStatus
	return status
}

func encodeMobile(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode mobile response: %w", err)
	}
	if len(payload) > mobilePayloadLimit {
		return "", fmt.Errorf("encode mobile response: %w", protocol.ErrPayloadTooLarge)
	}
	return string(payload), nil
}

func parseNodeID(value string) (protocol.NodeID, error) {
	if value == "" || value[0] == '+' || (len(value) > 1 && value[0] == '0') {
		return 0, errors.New("node ID must be a canonical non-zero decimal value")
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("node ID must be a canonical non-zero decimal value")
	}
	return protocol.NodeID(parsed), nil
}

func formatNodeID(value protocol.NodeID) string { return strconv.FormatUint(uint64(value), 10) }
