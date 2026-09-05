package bindings

//go:generate go run ./cmd/mfh-bindgen -out ./generated/contracts.json -schemas-out ../../apps/desktop/frontend/src/generated/resource-schemas.generated.json

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

	"github.com/yttydcs/myflowhub/protocol"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

const maxBindingTimeout = 10 * time.Minute

type Listener interface {
	OnEvent(eventJSON string)
	OnError(errorJSON string)
}

// PublicIdentity is the public portion of a Node identity exposed to bindings.
// It intentionally cannot carry private key material.
type PublicIdentity struct {
	NodeID    protocol.NodeID
	PublicKey ed25519.PublicKey
}

type Client struct {
	mu            sync.Mutex
	identity      PublicIdentity
	sdk           *sdk.Client
	connection    sdk.ConnectionStatus
	closed        bool
	nextID        int64
	subscriptions map[int64]context.CancelFunc
	wg            sync.WaitGroup
}

// NewAttachedClient creates a JSON/callback facade over an existing operation
// client and optional Host-owned parent status. It does not create or close
// auth state, a node, a parent supervisor, or any listener. The facade may be
// created before Host.Start for local operations, but WaitConnected and durable
// remote subscriptions require the Host to be started first.
func NewAttachedClient(client *sdk.Client, identity PublicIdentity, connection sdk.ConnectionStatus) (*Client, error) {
	if client == nil {
		return nil, errors.New("attached binding requires an SDK client")
	}
	if err := identity.NodeID.Validate(); err != nil {
		return nil, fmt.Errorf("attached binding NodeID: %w", err)
	}
	if len(identity.PublicKey) != ed25519.PublicKeySize {
		return nil, errors.New("attached binding public key must be Ed25519")
	}
	clientNodeID, err := client.NodeID()
	if err != nil {
		return nil, fmt.Errorf("attached binding SDK client: %w", err)
	}
	if clientNodeID != identity.NodeID {
		return nil, errors.New("attached binding identity does not match SDK client")
	}
	return &Client{
		identity:      PublicIdentity{NodeID: identity.NodeID, PublicKey: append(ed25519.PublicKey(nil), identity.PublicKey...)},
		sdk:           client,
		connection:    connection,
		subscriptions: make(map[int64]context.CancelFunc),
	}, nil
}

func (c *Client) IdentityJSON() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.sdk == nil {
		return "", errors.New("binding client is closed")
	}
	return encodeJSON(struct {
		NodeID    string `json:"node_id"`
		PublicKey string `json:"public_key"`
	}{
		NodeID:    strconv.FormatUint(uint64(c.identity.NodeID), 10),
		PublicKey: base64.RawStdEncoding.EncodeToString(c.identity.PublicKey),
	})
}

func (c *Client) StatusJSON() (string, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return "", errors.New("binding client is closed")
	}
	connection := c.connection
	c.mu.Unlock()
	if connection == nil {
		return encodeJSON(bindingConnection{State: string(sdk.ConnectionDisconnected)})
	}
	return encodeJSON(connectionJSON(connection.Snapshot()))
}

func (c *Client) WaitConnected(timeoutMS int64) error {
	ctx, cancel, err := bindingContext(timeoutMS)
	if err != nil {
		return err
	}
	defer cancel()
	connection, err := c.parentConnection()
	if err != nil {
		return err
	}
	for {
		snapshot := connection.Snapshot()
		switch snapshot.State {
		case sdk.ConnectionConnected:
			return nil
		case sdk.ConnectionFailed, sdk.ConnectionStopped:
			return fmt.Errorf("binding connection %s: %s", snapshot.State, snapshot.LastError)
		}
		if _, err := connection.WaitChange(ctx, snapshot.Generation); err != nil {
			return connectionWaitError(err, connection.Snapshot())
		}
	}
}

func connectionWaitError(cause error, snapshot sdk.ConnectionSnapshot) error {
	if snapshot.LastError != "" {
		return fmt.Errorf("wait for binding connection: %w; last connection error: %s", cause, snapshot.LastError)
	}
	return fmt.Errorf("wait for binding connection: %w", cause)
}

func (c *Client) CatalogJSON(ownerID, timeoutMS int64) (string, error) {
	owner, err := bindingNodeID(ownerID)
	if err != nil {
		return "", err
	}
	ctx, cancel, err := bindingContext(timeoutMS)
	if err != nil {
		return "", err
	}
	defer cancel()
	client, err := c.operationClient()
	if err != nil {
		return "", err
	}
	catalog, err := client.Catalog(ctx, owner)
	if err != nil {
		return "", err
	}
	return encodeJSON(catalogJSON(catalog))
}

func (c *Client) SnapshotJSON(ownerID int64, name string, timeoutMS int64) (string, error) {
	resourceID, err := bindingResource(ownerID, name)
	if err != nil {
		return "", err
	}
	ctx, cancel, err := bindingContext(timeoutMS)
	if err != nil {
		return "", err
	}
	defer cancel()
	client, err := c.operationClient()
	if err != nil {
		return "", err
	}
	event, err := client.Snapshot(ctx, resourceID)
	if err != nil {
		return "", err
	}
	if !json.Valid(event.Value) {
		return "", errors.New("snapshot payload is not valid JSON")
	}
	return string(event.Value), nil
}

func (c *Client) InvokeJSON(ownerID int64, name, requestJSON string, timeoutMS int64) (string, error) {
	encoded, err := c.OperateJSON(ownerID, name, string(protocol.CapabilityInvoke), "", requestJSON, timeoutMS)
	if err != nil {
		return "", err
	}
	var result struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(encoded), &result); err != nil || !json.Valid(result.Payload) {
		return "", errors.New("invoke response is not valid JSON")
	}
	return string(result.Payload), nil
}

func (c *Client) OperateJSON(ownerID int64, name, capability, schema, requestJSON string, timeoutMS int64) (string, error) {
	resourceID, err := bindingResource(ownerID, name)
	if err != nil {
		return "", err
	}
	request := []byte(requestJSON)
	if len(request) == 0 || len(request) > protocol.DefaultMaxPayload || !json.Valid(request) {
		return "", errors.New("resource operation request must be valid JSON within the protocol payload limit")
	}
	capabilityID := protocol.CapabilityID(capability)
	if err := capabilityID.Validate(); err != nil {
		return "", err
	}
	ctx, cancel, err := bindingContext(timeoutMS)
	if err != nil {
		return "", err
	}
	defer cancel()
	client, err := c.operationClient()
	if err != nil {
		return "", err
	}
	response, err := client.Operate(ctx, resourceID, capabilityID, schema, request)
	if err != nil {
		return "", err
	}
	if !json.Valid(response.Payload) {
		return "", errors.New("resource operation response is not valid JSON")
	}
	return encodeJSON(struct {
		Schema  string          `json:"schema,omitempty"`
		Payload json.RawMessage `json:"payload"`
	}{Schema: response.Schema, Payload: response.Payload})
}

func (c *Client) UploadFile(ownerID int64, sourcePath, destination, contentType string, timeoutMS int64) (string, error) {
	owner, err := bindingNodeID(ownerID)
	if err != nil {
		return "", err
	}
	ctx, cancel, err := bindingContext(timeoutMS)
	if err != nil {
		return "", err
	}
	defer cancel()
	client, err := c.operationClient()
	if err != nil {
		return "", err
	}
	files, err := client.Files(owner)
	if err != nil {
		return "", err
	}
	result, err := files.UploadFile(ctx, sourcePath, destination, contentType, protocol.MaxFileChunkBytes, time.Hour)
	if err != nil {
		return "", err
	}
	return encodeJSON(result)
}

func (c *Client) Subscribe(ownerID int64, name string, leaseMS int64, listener Listener) (int64, error) {
	return c.SubscribeCapability(ownerID, name, string(protocol.CapabilitySubscribe), leaseMS, listener)
}

func (c *Client) SubscribeCapability(ownerID int64, name, capability string, leaseMS int64, listener Listener) (int64, error) {
	if listener == nil {
		return 0, errors.New("binding subscription listener is required")
	}
	resourceID, err := bindingResource(ownerID, name)
	if err != nil {
		return 0, err
	}
	capabilityID := protocol.CapabilityID(capability)
	if err := capabilityID.Validate(); err != nil {
		return 0, err
	}
	lease, err := bindingDuration(leaseMS)
	if err != nil {
		return 0, fmt.Errorf("binding subscription lease: %w", err)
	}
	client, err := c.operationClient()
	if err != nil {
		return 0, err
	}
	connection, err := c.parentConnection()
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	subscription, err := client.SubscribeDurableStatusCapability(ctx, connection, resourceID, capabilityID, lease, 64)
	if err != nil {
		cancel()
		return 0, err
	}
	readyTimer := time.NewTimer(10 * time.Second)
	defer readyTimer.Stop()
	ready := false
	for !ready {
		select {
		case <-subscription.Ready:
			ready = true
		case err, ok := <-subscription.Errors:
			if !ok {
				subscription.Cancel()
				cancel()
				return 0, errors.New("binding subscription stopped before becoming ready")
			}
			if terminalSubscriptionError(err) {
				subscription.Cancel()
				cancel()
				return 0, err
			}
		case <-readyTimer.C:
			subscription.Cancel()
			cancel()
			return 0, errors.New("binding subscription did not become ready within 10s")
		}
	}
	c.mu.Lock()
	if c.closed || c.sdk != client {
		c.mu.Unlock()
		subscription.Cancel()
		cancel()
		return 0, errors.New("binding client stopped while subscribing")
	}
	if c.nextID == int64(^uint64(0)>>1) {
		c.mu.Unlock()
		subscription.Cancel()
		cancel()
		return 0, errors.New("binding subscription identifiers are exhausted")
	}
	c.nextID++
	id := c.nextID
	c.subscriptions[id] = cancel
	c.wg.Add(1)
	c.mu.Unlock()
	go c.forwardSubscription(id, ctx, subscription, listener)
	return id, nil
}

func (c *Client) CancelSubscription(subscriptionID int64) {
	if c == nil || subscriptionID <= 0 {
		return
	}
	c.mu.Lock()
	cancel := c.subscriptions[subscriptionID]
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.connection = nil
	c.sdk = nil
	cancellations := make([]context.CancelFunc, 0, len(c.subscriptions))
	for _, cancel := range c.subscriptions {
		cancellations = append(cancellations, cancel)
	}
	c.mu.Unlock()
	for _, cancel := range cancellations {
		cancel()
	}
	c.wg.Wait()
	return nil
}

func (c *Client) operationClient() (*sdk.Client, error) {
	if c == nil {
		return nil, errors.New("binding client is closed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errors.New("binding client is closed")
	}
	if c.sdk == nil {
		return nil, errors.New("binding client is not started")
	}
	return c.sdk, nil
}

func (c *Client) parentConnection() (sdk.ConnectionStatus, error) {
	if c == nil {
		return nil, errors.New("binding client is closed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errors.New("binding client is closed")
	}
	if c.connection == nil {
		return nil, errors.New("binding client has no parent connection")
	}
	return c.connection, nil
}

func (c *Client) forwardSubscription(id int64, ctx context.Context, subscription *sdk.DurableSubscription, listener Listener) {
	defer c.wg.Done()
	defer subscription.Cancel()
	defer func() {
		c.mu.Lock()
		delete(c.subscriptions, id)
		c.mu.Unlock()
	}()
	for {
		select {
		case event, ok := <-subscription.Events:
			if !ok {
				return
			}
			publisher := ""
			if event.Publisher != 0 {
				publisher = strconv.FormatUint(uint64(event.Publisher), 10)
			}
			payload, err := encodeJSON(bindingEvent{
				Kind: string(event.Kind), OwnerNodeID: strconv.FormatUint(uint64(event.Resource.Owner), 10), ResourceName: event.Resource.Name,
				Capability: string(event.Capability), Schema: event.Schema, Revision: event.Revision, Sequence: event.Sequence,
				PublisherNodeID: publisher, PublisherSequence: event.PublisherSequence,
				GapFrom: event.GapFrom, GapTo: event.GapTo, Value: event.Value, Reason: event.Reason,
			})
			if err != nil {
				callListenerError(listener, bindingErrorJSON(err))
				return
			}
			if err := callListenerEvent(listener, payload); err != nil {
				callListenerError(listener, bindingErrorJSON(err))
				return
			}
		case err, ok := <-subscription.Errors:
			if !ok {
				return
			}
			if err != nil {
				callListenerError(listener, bindingErrorJSON(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func bindingNodeID(value int64) (protocol.NodeID, error) {
	if value <= 0 {
		return 0, errors.New("binding NodeID must be positive")
	}
	id := protocol.NodeID(value)
	if err := id.Validate(); err != nil {
		return 0, err
	}
	return id, nil
}

func bindingResource(ownerID int64, name string) (protocol.ResourceID, error) {
	owner, err := bindingNodeID(ownerID)
	if err != nil {
		return protocol.ResourceID{}, err
	}
	value := protocol.ResourceID{Owner: owner, Name: name}
	if err := value.Validate(); err != nil {
		return protocol.ResourceID{}, err
	}
	return value, nil
}

func bindingContext(timeoutMS int64) (context.Context, context.CancelFunc, error) {
	timeout, err := bindingDuration(timeoutMS)
	if err != nil {
		return nil, nil, fmt.Errorf("binding timeout: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return ctx, cancel, nil
}

func bindingDuration(milliseconds int64) (time.Duration, error) {
	if milliseconds <= 0 {
		return 0, errors.New("duration must be positive")
	}
	duration := time.Duration(milliseconds) * time.Millisecond
	if duration <= 0 || duration > maxBindingTimeout {
		return 0, fmt.Errorf("duration must not exceed %s", maxBindingTimeout)
	}
	return duration, nil
}

func terminalSubscriptionError(err error) bool {
	var value *sdk.Error
	if !errors.As(err, &value) {
		return false
	}
	switch value.Code {
	case protocol.CodeMalformed, protocol.CodeForbidden, protocol.CodeNotFound, protocol.CodeConflict:
		return true
	default:
		return false
	}
}

func encodeJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode binding JSON: %w", err)
	}
	return string(data), nil
}
