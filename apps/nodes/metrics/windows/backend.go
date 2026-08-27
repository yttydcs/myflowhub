package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	metricswindows "github.com/yttydcs/myflowhub/apps/nodes/metrics/platform/windows"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
)

const uiPayloadLimit = 256 * 1024

type App struct {
	mu             sync.Mutex
	ctx            context.Context
	runtime        *metrics.Runtime
	presentCancel  context.CancelFunc
	presentWG      sync.WaitGroup
	presenterError string
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

type identityResponseV1 struct {
	Version   int    `json:"version"`
	NodeID    string `json:"node_id"`
	PublicKey string `json:"public_key"`
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
	endpoint := link.Endpoint(r.Endpoint)
	if err := endpoint.Validate(); err != nil {
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

type statusV1 struct {
	Version        int                        `json:"version"`
	Running        bool                       `json:"running"`
	Connection     *connectionStatusV1        `json:"connection,omitempty"`
	Configuration  *metrics.ConfigV1          `json:"configuration,omitempty"`
	Samples        []metrics.SampleV1         `json:"samples"`
	Notifications  metrics.NotificationStatus `json:"notifications"`
	PresenterError string                     `json:"presenter_error,omitempty"`
}

type connectionStatusV1 struct {
	State          string `json:"state"`
	ParentNodeID   string `json:"parent_node_id"`
	Endpoint       string `json:"endpoint"`
	Attempt        uint64 `json:"attempt"`
	LinkGeneration uint64 `json:"link_generation"`
	LastError      string `json:"last_error,omitempty"`
	NextRetry      string `json:"next_retry,omitempty"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
}

func (a *App) shutdown(context.Context) { _ = a.Stop() }

func (a *App) Identity(requestJSON string) (string, error) {
	var request identityRequestV1
	if err := protocol.DecodeJSONPayload([]byte(requestJSON), uiPayloadLimit, &request); err != nil {
		return "", err
	}
	nodeID, _ := parseNodeID(request.NodeID)
	state, err := auth.OpenState(request.StateDirectory, nodeID)
	if err != nil {
		return "", err
	}
	return encodeUI(identityResponseV1{
		Version: 1, NodeID: formatNodeID(state.Identity.NodeID),
		PublicKey: base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey),
	})
}

func (a *App) Start(requestJSON string) (string, error) {
	var request startRequestV1
	if err := protocol.DecodeJSONPayload([]byte(requestJSON), uiPayloadLimit, &request); err != nil {
		return "", err
	}
	nodeID, parentID, parentKey, err := request.values()
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runtime != nil {
		return "", errors.New("metrics runtime is already running")
	}
	if a.ctx == nil {
		a.ctx = context.Background()
	}
	platform := metricswindows.New()
	value, err := metrics.Start(a.ctx, metrics.RuntimeConfig{
		StateDirectory: request.StateDirectory,
		NodeID:         nodeID,
		ParentID:       parentID,
		ParentKey:      parentKey,
		Permit:         request.Permit,
		Endpoint:       link.Endpoint(request.Endpoint),
		Platform:       "windows",
		Collector:      platform,
		Actuator:       platform,
	})
	if err != nil {
		return "", err
	}
	a.runtime = value
	presentCtx, presentCancel := context.WithCancel(a.ctx)
	a.presentCancel = presentCancel
	a.presenterError = ""
	a.presentWG.Add(1)
	go a.presentNotifications(presentCtx, value)
	return encodeUI(a.statusLocked())
}

func (a *App) Stop() error {
	a.mu.Lock()
	value := a.runtime
	presentCancel := a.presentCancel
	a.runtime = nil
	a.presentCancel = nil
	a.mu.Unlock()
	if presentCancel != nil {
		presentCancel()
		a.presentWG.Wait()
	}
	if value == nil {
		return nil
	}
	return value.Close()
}

func (a *App) Status() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return encodeUI(a.statusLocked())
}

func (a *App) Configuration() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runtime == nil {
		return "", errors.New("metrics runtime is not running")
	}
	return encodeUI(a.runtime.Metrics.Config())
}

func (a *App) UpdateConfiguration(requestJSON string) (string, error) {
	var request metrics.ConfigUpdateV1
	if err := protocol.DecodeJSONPayload([]byte(requestJSON), uiPayloadLimit, &request); err != nil {
		return "", err
	}
	if err := request.Validate(); err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runtime == nil {
		return "", errors.New("metrics runtime is not running")
	}
	result, err := a.runtime.Metrics.ApplyConfig(a.ctx, request)
	if err != nil {
		return "", err
	}
	return encodeUI(result)
}

func (a *App) Definitions() (string, error) {
	return encodeUI(metrics.Definitions("windows"))
}

func (a *App) statusLocked() statusV1 {
	status := statusV1{Version: 1, Running: a.runtime != nil, Samples: []metrics.SampleV1{}}
	if a.runtime == nil {
		return status
	}
	snapshot := a.runtime.Connection.Snapshot()
	connection := &connectionStatusV1{
		State: string(snapshot.State), ParentNodeID: formatNodeID(snapshot.Parent), Endpoint: snapshot.Endpoint,
		Attempt: snapshot.Attempt, LinkGeneration: snapshot.LinkGeneration, LastError: snapshot.LastError,
	}
	if !snapshot.NextRetry.IsZero() {
		connection.NextRetry = snapshot.NextRetry.UTC().Format(time.RFC3339Nano)
	}
	configuration := a.runtime.Metrics.Config()
	status.Connection = connection
	status.Configuration = &configuration
	status.Samples = a.runtime.Metrics.Samples()
	status.Notifications = a.runtime.Notifications.Status()
	status.PresenterError = a.presenterError
	return status
}

func (a *App) presentNotifications(ctx context.Context, value *metrics.Runtime) {
	defer a.presentWG.Done()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, event := range value.Notifications.Dequeue() {
				if err := showSystemNotification(ctx, event); err != nil && ctx.Err() == nil {
					a.mu.Lock()
					a.presenterError = truncatePresenterError(err.Error())
					a.mu.Unlock()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func truncatePresenterError(value string) string {
	if len(value) > 1024 {
		return value[:1024]
	}
	return value
}

func encodeUI(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode UI response: %w", err)
	}
	if len(payload) > uiPayloadLimit {
		return "", fmt.Errorf("encode UI response: %w: got %d, max %d", protocol.ErrPayloadTooLarge, len(payload), uiPayloadLimit)
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
