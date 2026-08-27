package metricsmobile

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
)

const mobilePayloadLimit = 256 * 1024

type Client struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	runtime *metrics.Runtime
	actions *actionQueue
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
	Version       int                `json:"version"`
	Running       bool               `json:"running"`
	State         string             `json:"state"`
	ParentNodeID  string             `json:"parent_node_id,omitempty"`
	Endpoint      string             `json:"endpoint,omitempty"`
	Attempt       uint64             `json:"attempt,omitempty"`
	LastError     string             `json:"last_error,omitempty"`
	Configuration *metrics.ConfigV1  `json:"configuration,omitempty"`
	Samples       []metrics.SampleV1 `json:"samples"`
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
		return "", errors.New("metrics runtime is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	actions := newActionQueue()
	value, err := metrics.Start(ctx, metrics.RuntimeConfig{
		StateDirectory: request.StateDirectory, NodeID: nodeID, ParentID: parentID, ParentKey: parentKey,
		Permit: request.Permit, Endpoint: link.Endpoint(request.Endpoint), Platform: "android", Actuator: actions,
	})
	if err != nil {
		actions.close()
		cancel()
		return "", err
	}
	c.cancel = cancel
	c.runtime = value
	c.actions = actions
	return encodeMobile(c.statusLocked())
}

func (c *Client) Stop() error {
	c.mu.Lock()
	value := c.runtime
	actions := c.actions
	cancel := c.cancel
	c.runtime = nil
	c.actions = nil
	c.cancel = nil
	c.mu.Unlock()
	if actions != nil {
		actions.close()
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
		return "", errors.New("metrics runtime is not running")
	}
	return encodeMobile(c.runtime.Metrics.Config())
}

func (c *Client) UpdateConfiguration(requestJSON string) (string, error) {
	var request metrics.ConfigUpdateV1
	if err := protocol.DecodeJSONPayload([]byte(requestJSON), mobilePayloadLimit, &request); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil {
		return "", errors.New("metrics runtime is not running")
	}
	value, err := c.runtime.Metrics.ApplyConfig(context.Background(), request)
	if err != nil {
		return "", err
	}
	return encodeMobile(value)
}

func (c *Client) UpdateMetric(metric, value, errorMessage string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil {
		return errors.New("metrics runtime is not running")
	}
	var collectErr error
	if errorMessage != "" {
		collectErr = errors.New(errorMessage)
	}
	return c.runtime.Metrics.Update(metrics.Name(metric), value, collectErr)
}

func (c *Client) NextAction() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil || c.actions == nil {
		return "", errors.New("metrics runtime is not running")
	}
	action, ok := c.actions.next()
	if !ok {
		return "", nil
	}
	return encodeMobile(action)
}

func (c *Client) NextNotification() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil || c.runtime.Notifications == nil {
		return "", errors.New("metrics runtime is not running")
	}
	event, ok := c.runtime.Notifications.Next()
	if !ok {
		return "", nil
	}
	return encodeMobile(event)
}

func (c *Client) CompleteAction(actionID, actualValue, errorMessage string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runtime == nil || c.actions == nil {
		return errors.New("metrics runtime is not running")
	}
	action, err := c.actions.complete(actionID)
	if err != nil {
		return err
	}
	if errorMessage != "" {
		return c.runtime.Metrics.Update(metrics.Name(action.Metric), "", errors.New(errorMessage))
	}
	if err := metrics.ValidateValue(metrics.Name(action.Metric), actualValue); err != nil {
		return err
	}
	return c.runtime.Metrics.Update(metrics.Name(action.Metric), actualValue, nil)
}

func (c *Client) statusLocked() mobileStatusV1 {
	status := mobileStatusV1{Version: 1, State: "stopped", Samples: []metrics.SampleV1{}}
	if c.runtime == nil {
		return status
	}
	snapshot := c.runtime.Connection.Snapshot()
	configuration := c.runtime.Metrics.Config()
	status.Running = true
	status.State = string(snapshot.State)
	status.ParentNodeID = formatNodeID(snapshot.Parent)
	status.Endpoint = snapshot.Endpoint
	status.Attempt = snapshot.Attempt
	status.LastError = snapshot.LastError
	status.Configuration = &configuration
	status.Samples = c.runtime.Metrics.Samples()
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
