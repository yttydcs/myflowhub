package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yttydcs/myflowhub/protocol"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

const (
	defaultRequestTimeoutMS = int64(15_000)
	defaultLeaseMS          = int64(60_000)
	maxDesktopLogs          = 500
)

type logEntry struct {
	TimeUnixMS int64  `json:"time_unix_ms"`
	Level      string `json:"level"`
	Message    string `json:"message"`
}

type App struct {
	mu       sync.Mutex
	ctx      context.Context
	store    *settingsStore
	settings Settings
	client   *desktopbinding.Client
	logs     []logEntry
}

func NewApp(configRoot string) (*App, error) {
	store, err := newSettingsStore(configRoot)
	if err != nil {
		return nil, err
	}
	settings, err := store.load()
	if err != nil {
		return nil, err
	}
	app := &App{store: store, settings: settings, logs: make([]logEntry, 0, maxDesktopLogs)}
	if err := app.openLocked(); err != nil {
		return nil, err
	}
	app.appendLog("info", "desktop profile opened")
	return app, nil
}

func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
}

func (a *App) Shutdown(context.Context) { _ = a.Close() }

func (a *App) SettingsJSON() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return marshalJSON(a.settings)
}

func (a *App) SaveSettingsJSON(raw string) (string, error) {
	var value Settings
	if err := decodeBoundedJSON(raw, &value); err != nil {
		return "", err
	}
	value.Version = desktopSettingsVersion
	if err := validateSettings(value); err != nil {
		return "", err
	}
	a.mu.Lock()
	previous := a.settings
	if err := a.store.save(value); err != nil {
		a.mu.Unlock()
		return "", err
	}
	if a.client != nil {
		_ = a.client.Close()
		a.client = nil
	}
	a.settings = value
	if err := a.openLocked(); err != nil {
		a.settings = previous
		_ = a.store.save(previous)
		_ = a.openLocked()
		a.mu.Unlock()
		return "", fmt.Errorf("apply desktop settings: %w", err)
	}
	result, err := marshalJSON(a.settings)
	a.mu.Unlock()
	a.appendLog("info", "desktop settings saved; connection reset")
	return result, err
}

func (a *App) ResetStorage(confirm string) (string, error) {
	a.mu.Lock()
	if a.client != nil {
		_ = a.client.Close()
		a.client = nil
	}
	settings, err := a.store.reset(confirm)
	if err == nil {
		a.settings = settings
		err = a.openLocked()
	}
	result, encodeErr := marshalJSON(a.settings)
	a.mu.Unlock()
	if err != nil {
		return "", err
	}
	if encodeErr != nil {
		return "", encodeErr
	}
	a.appendLog("warn", "desktop settings reset explicitly; identity state was retained")
	return result, nil
}

func (a *App) IdentityJSON() (string, error) {
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.IdentityJSON()
}

func (a *App) Connect() error {
	a.mu.Lock()
	settings := a.settings
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return errors.New("desktop client is not open")
	}
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint == "" {
		return errors.New("endpoint is required")
	}
	parentID, err := parsePositiveInt64(settings.ParentNodeID, "parent_node_id")
	if err != nil {
		return err
	}
	if strings.TrimSpace(settings.ParentPublicKey) == "" {
		return errors.New("parent_public_key is required")
	}
	if err := client.TrustParent(parentID, settings.ParentPublicKey); err != nil {
		return a.fail("trust parent", err)
	}
	if err := client.StartTCP(endpoint, parentID, settings.PermitJSON); err != nil {
		return a.fail("connect", err)
	}
	a.appendLog("info", "managed TCP connection started")
	return nil
}

func (a *App) Disconnect() error {
	a.mu.Lock()
	if a.client != nil {
		_ = a.client.Close()
		a.client = nil
	}
	err := a.openLocked()
	a.mu.Unlock()
	if err != nil {
		return err
	}
	a.appendLog("info", "desktop connection stopped")
	return nil
}

func (a *App) WaitConnected(timeoutMS int64) error {
	client, err := a.currentClient()
	if err != nil {
		return err
	}
	return client.WaitConnected(timeoutMS)
}

func (a *App) StatusJSON() (string, error) {
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.StatusJSON()
}

func (a *App) CatalogJSON(ownerID int64) (string, error) {
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.CatalogJSON(ownerID, defaultRequestTimeoutMS)
}

func (a *App) SnapshotJSON(ownerID int64, name string) (string, error) {
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.SnapshotJSON(ownerID, name, defaultRequestTimeoutMS)
}

func (a *App) InvokeJSON(ownerID int64, name, requestJSON string) (string, error) {
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	result, err := client.InvokeJSON(ownerID, name, requestJSON, defaultRequestTimeoutMS)
	if err != nil {
		return "", a.fail("invoke "+name, err)
	}
	a.appendLog("info", "command completed: "+name)
	return result, nil
}

func (a *App) Subscribe(ownerID int64, name string, leaseMS int64) (int64, error) {
	if leaseMS == 0 {
		leaseMS = defaultLeaseMS
	}
	client, err := a.currentClient()
	if err != nil {
		return 0, err
	}
	id, err := client.Subscribe(ownerID, name, leaseMS)
	if err != nil {
		return 0, a.fail("subscribe "+name, err)
	}
	a.appendLog("info", "subscription ready: "+name)
	return id, nil
}

func (a *App) PollSubscription(subscriptionID, timeoutMS int64) (string, error) {
	client, err := a.currentClient()
	if err != nil {
		return "", err
	}
	return client.PollSubscription(subscriptionID, timeoutMS)
}

func (a *App) CancelSubscription(subscriptionID int64) {
	client, err := a.currentClient()
	if err == nil {
		client.CancelSubscription(subscriptionID)
	}
}

func (a *App) TopologyJSON(ownerID int64) (string, error) {
	return a.SnapshotJSON(ownerID, protocol.BuiltinManagementTopology)
}

func (a *App) HealthJSON(ownerID int64) (string, error) {
	return a.SnapshotJSON(ownerID, protocol.BuiltinManagementHealth)
}

func (a *App) ManagementConfigJSON(ownerID int64) (string, error) {
	return a.SnapshotJSON(ownerID, protocol.BuiltinManagementConfig)
}

func (a *App) IssuePermitJSON(ownerID int64, requestJSON string) (string, error) {
	var request protocol.ManagementIssuePermitV1
	if err := decodeBoundedJSON(requestJSON, &request); err != nil {
		return "", err
	}
	if err := request.Validate(); err != nil {
		return "", err
	}
	return a.InvokeJSON(ownerID, protocol.BuiltinManagementIssuePermit, requestJSON)
}

func (a *App) RevokeNodeJSON(ownerID int64, requestJSON string) (string, error) {
	var request protocol.ManagementRevokeV1
	if err := decodeBoundedJSON(requestJSON, &request); err != nil {
		return "", err
	}
	if err := request.Validate(); err != nil {
		return "", err
	}
	return a.InvokeJSON(ownerID, protocol.BuiltinManagementRevokeNode, requestJSON)
}

func (a *App) GrantPolicyJSON(ownerID int64, requestJSON string) (string, error) {
	var request protocol.ManagementPolicyRuleV1
	if err := decodeBoundedJSON(requestJSON, &request); err != nil {
		return "", err
	}
	if err := request.Validate(); err != nil {
		return "", err
	}
	return a.InvokeJSON(ownerID, protocol.BuiltinManagementPolicyGrant, requestJSON)
}

func (a *App) RevokePolicyJSON(ownerID int64, requestJSON string) (string, error) {
	var request protocol.ManagementPolicyRuleV1
	if err := decodeBoundedJSON(requestJSON, &request); err != nil {
		return "", err
	}
	if err := request.Validate(); err != nil {
		return "", err
	}
	return a.InvokeJSON(ownerID, protocol.BuiltinManagementPolicyRevoke, requestJSON)
}

func (a *App) FileTransfersJSON(ownerID int64) (string, error) {
	return a.SnapshotJSON(ownerID, protocol.BuiltinFileTransfers)
}

func (a *App) FlowDefinitionsJSON(ownerID int64) (string, error) {
	return a.SnapshotJSON(ownerID, protocol.BuiltinFlowDefinitions)
}

func (a *App) FlowRunsJSON(ownerID int64) (string, error) {
	return a.SnapshotJSON(ownerID, protocol.BuiltinFlowRuns)
}

func (a *App) UploadFile(ownerID int64, sourcePath, destination, contentType string) (string, error) {
	info, err := os.Stat(sourcePath)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("upload source must be a readable regular file")
	}
	if info.Size() > 64<<20 {
		return "", errors.New("desktop upload is limited to 64 MiB")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read upload source: %w", err)
	}
	transferID, digest := fileIdentity(data)
	offer := protocol.FileOfferV1{Version: 1, TransferID: transferID, Path: destination, Size: int64(len(data)), SHA256: digest, ChunkSize: protocol.MaxFileChunkBytes, ContentType: contentType, ExpiresAtUnixMS: time.Now().Add(time.Hour).UnixMilli()}
	if err := offer.Validate(); err != nil {
		return "", err
	}
	offerJSON, _ := marshalJSON(offer)
	_, err = a.InvokeJSON(ownerID, protocol.BuiltinFileOffer, offerJSON)
	if err != nil {
		return "", err
	}
	for offset := 0; offset < len(data); offset += protocol.MaxFileChunkBytes {
		end := min(offset+protocol.MaxFileChunkBytes, len(data))
		chunk := protocol.FileChunkV1{Version: 1, TransferID: transferID, Offset: int64(offset), Data: data[offset:end], SHA256: sha256Hex(data[offset:end])}
		chunkJSON, _ := marshalJSON(chunk)
		_, err = a.InvokeJSON(ownerID, protocol.BuiltinFileChunk, chunkJSON)
		if err != nil {
			cancel, _ := marshalJSON(protocol.FileCancelV1{Version: 1, TransferID: transferID, Reason: "desktop upload failed"})
			_, _ = a.InvokeJSON(ownerID, protocol.BuiltinFileCancel, cancel)
			return "", err
		}
	}
	complete, _ := marshalJSON(protocol.FileCompleteV1{Version: 1, TransferID: transferID, Size: int64(len(data)), SHA256: digest})
	return a.InvokeJSON(ownerID, protocol.BuiltinFileComplete, complete)
}

func (a *App) LogsJSON() (string, error) {
	a.mu.Lock()
	logs := append([]logEntry(nil), a.logs...)
	a.mu.Unlock()
	return marshalJSON(logs)
}

func (a *App) Close() error {
	a.mu.Lock()
	client := a.client
	a.client = nil
	a.mu.Unlock()
	if client == nil {
		return nil
	}
	return client.Close()
}

func (a *App) currentClient() (*desktopbinding.Client, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		return nil, errors.New("desktop client is not open")
	}
	return a.client, nil
}

func (a *App) openLocked() error {
	directory, err := a.store.stateDirectory(a.settings.Profile)
	if err != nil {
		return err
	}
	nodeID, err := parsePositiveInt64(a.settings.NodeID, "node_id")
	if err != nil {
		return err
	}
	client := &desktopbinding.Client{}
	if err := client.Open(directory, nodeID); err != nil {
		return err
	}
	a.client = client
	return nil
}

func (a *App) appendLog(level, message string) {
	entry := logEntry{TimeUnixMS: time.Now().UTC().UnixMilli(), Level: level, Message: message}
	a.mu.Lock()
	if len(a.logs) == maxDesktopLogs {
		copy(a.logs, a.logs[1:])
		a.logs[len(a.logs)-1] = entry
	} else {
		a.logs = append(a.logs, entry)
	}
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "desktop:log", entry)
	}
}

func (a *App) fail(action string, err error) error {
	a.appendLog("error", action+": "+err.Error())
	return err
}

func parsePositiveInt64(raw, name string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive signed 64-bit integer", name)
	}
	return value, nil
}

func decodeBoundedJSON(raw string, target any) error {
	if len(raw) == 0 || len(raw) > protocol.DefaultMaxPayload || !json.Valid([]byte(raw)) {
		return errors.New("request must be valid JSON within the protocol payload limit")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	return nil
}

func marshalJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
