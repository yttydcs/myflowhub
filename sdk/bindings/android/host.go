package androidbinding

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/rfcomm"
	"github.com/yttydcs/myflowhub/transport/tcp"
)

type Host struct {
	mu         sync.Mutex
	directory  string
	nodeID     protocol.NodeID
	state      *auth.State
	joinPermit *protocol.ProvisioningPermitV1
	runtime    *hub.Hub
	client     *sdk.Client
	connection *sdk.Connection
	cancel     context.CancelFunc
}

func NewHost(stateDirectory string, nodeID int64) (*Host, error) {
	if strings.TrimSpace(stateDirectory) == "" {
		return nil, errors.New("Android Hub state directory is required")
	}
	id, err := androidNodeID(nodeID)
	if err != nil {
		return nil, err
	}
	state, err := auth.OpenState(stateDirectory, id)
	if err != nil {
		return nil, err
	}
	return &Host{directory: stateDirectory, nodeID: id, state: state}, nil
}

func (h *Host) IdentityJSON() (string, error) {
	if h == nil || h.state == nil {
		return "", errors.New("Android Hub host is closed")
	}
	return androidJSON(map[string]any{
		"node_id":    strconv.FormatUint(uint64(h.state.Identity.NodeID), 10),
		"public_key": base64.RawStdEncoding.EncodeToString(h.state.Identity.PublicKey),
	})
}

func (h *Host) TrustParent(parentID int64, rawPublicKey string) error {
	if h == nil || h.state == nil {
		return errors.New("Android Hub host is closed")
	}
	parent, err := androidNodeID(parentID)
	if err != nil {
		return err
	}
	key, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(rawPublicKey))
	if err != nil || len(key) != 32 {
		return errors.New("parent public key must be raw-base64 Ed25519")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.runtime != nil {
		return errors.New("parent trust must be configured before the Android Hub starts")
	}
	return h.state.Trust.Add(parent, key)
}

func (h *Host) SetJoinPermit(permitJSON string) error {
	if h == nil || h.state == nil {
		return errors.New("Android Hub host is closed")
	}
	var permit *protocol.ProvisioningPermitV1
	if strings.TrimSpace(permitJSON) != "" {
		var value protocol.ProvisioningPermitV1
		if err := protocol.DecodeJSONPayload([]byte(permitJSON), protocol.DefaultMaxPayload, &value); err != nil {
			return fmt.Errorf("decode Android Hub join permit: %w", err)
		}
		permit = &value
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.runtime != nil {
		return errors.New("join permit must be configured before the Android Hub starts")
	}
	h.joinPermit = permit
	return nil
}

func (h *Host) Start(tcpEndpoint, rfcommEndpoint string) error {
	if h == nil || h.state == nil {
		return errors.New("Android Hub host is closed")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.runtime != nil {
		return errors.New("Android Hub host is already started")
	}
	listeners := make([]hub.ListenerConfig, 0, 2)
	if endpoint := strings.TrimSpace(tcpEndpoint); endpoint != "" {
		listeners = append(listeners, hub.ListenerConfig{Driver: tcp.Driver{}, Endpoint: link.Endpoint(endpoint)})
	}
	if endpoint := strings.TrimSpace(rfcommEndpoint); endpoint != "" {
		driver, err := rfcomm.New(0)
		if err != nil {
			return err
		}
		listeners = append(listeners, hub.ListenerConfig{Driver: driver, Endpoint: link.Endpoint(endpoint)})
	}
	if len(listeners) == 0 {
		return errors.New("Android Hub requires at least one TCP or RFCOMM listener")
	}
	ctx, cancel := context.WithCancel(context.Background())
	runtime, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: h.directory,
		NodeID:         h.nodeID,
		Node:           node.Config{JoinPermit: h.joinPermit},
		Listeners:      listeners,
	})
	if err != nil {
		cancel()
		return err
	}
	client, err := sdk.NewClient(runtime.Node)
	if err != nil {
		_ = runtime.Close()
		cancel()
		return err
	}
	h.runtime = runtime
	h.client = client
	h.cancel = cancel
	return nil
}

func (h *Host) ConnectParentTCP(endpoint string, parentID int64) error {
	return h.connectParent(tcp.Driver{}, endpoint, parentID)
}

func (h *Host) ConnectParentRFCOMM(endpoint string, parentID int64) error {
	driver, err := rfcomm.New(0)
	if err != nil {
		return err
	}
	return h.connectParent(driver, endpoint, parentID)
}

func (h *Host) connectParent(driver link.Driver, endpoint string, parentID int64) error {
	parent, err := androidNodeID(parentID)
	if err != nil {
		return err
	}
	address := link.Endpoint(strings.TrimSpace(endpoint))
	if err := address.Validate(); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.runtime == nil || h.client == nil {
		return errors.New("Android Hub host is not started")
	}
	if h.connection != nil {
		return errors.New("Android Hub parent connection is already started")
	}
	connection, err := h.client.ConnectManaged(context.Background(), driver, address, parent, node.SupervisorConfig{})
	if err != nil {
		return err
	}
	h.connection = connection
	return nil
}

func (h *Host) DisconnectParent() {
	if h == nil {
		return
	}
	h.mu.Lock()
	connection := h.connection
	h.connection = nil
	h.mu.Unlock()
	if connection != nil {
		connection.Stop()
	}
}

func (h *Host) StatusJSON() (string, error) {
	if h == nil {
		return "", errors.New("Android Hub host is closed")
	}
	h.mu.Lock()
	runtime := h.runtime
	connection := h.connection
	h.mu.Unlock()
	endpoints := []string{}
	if runtime != nil {
		for _, endpoint := range runtime.Endpoints {
			endpoints = append(endpoints, string(endpoint))
		}
	}
	parent := map[string]any{"state": string(sdk.ConnectionDisconnected)}
	if connection != nil {
		snapshot := connection.Snapshot()
		parent = map[string]any{
			"state": string(snapshot.State), "parent_node_id": strconv.FormatUint(uint64(snapshot.Parent), 10),
			"endpoint": snapshot.Endpoint, "attempt": snapshot.Attempt, "link_generation": snapshot.LinkGeneration,
			"last_error": snapshot.LastError, "generation": snapshot.Generation,
		}
	}
	return androidJSON(map[string]any{"running": runtime != nil, "node_id": strconv.FormatUint(uint64(h.nodeID), 10), "endpoints": endpoints, "parent": parent})
}

func (h *Host) Stop() error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	connection := h.connection
	runtime := h.runtime
	cancel := h.cancel
	h.connection = nil
	h.runtime = nil
	h.client = nil
	h.cancel = nil
	h.mu.Unlock()
	if connection != nil {
		connection.Stop()
	}
	if cancel != nil {
		cancel()
	}
	if runtime != nil {
		return runtime.Close()
	}
	return nil
}

func androidNodeID(value int64) (protocol.NodeID, error) {
	if value <= 0 {
		return 0, errors.New("Android NodeID must be positive")
	}
	id := protocol.NodeID(value)
	return id, id.Validate()
}

func androidJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
