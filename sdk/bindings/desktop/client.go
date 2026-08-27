package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/sdk/bindings"
)

const maxPollTimeout = 10 * time.Minute

type Client struct {
	mu            sync.Mutex
	core          *bindings.Client
	subscriptions map[int64]*subscription
}

func (c *Client) Open(stateDirectory string, nodeID int64) error {
	if c == nil {
		return errors.New("desktop binding client is required")
	}
	core, err := bindings.NewClient(stateDirectory, nodeID)
	if err != nil {
		return err
	}
	c.mu.Lock()
	if c.core != nil {
		c.mu.Unlock()
		_ = core.Close()
		return errors.New("desktop binding client is already open")
	}
	c.core = core
	c.subscriptions = make(map[int64]*subscription)
	c.mu.Unlock()
	return nil
}

func (c *Client) IdentityJSON() (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.IdentityJSON()
}

func (c *Client) TrustParent(parentID int64, rawPublicKey string) error {
	core, err := c.current()
	if err != nil {
		return err
	}
	return core.TrustParent(parentID, rawPublicKey)
}

func (c *Client) StartTCP(endpoint string, parentID int64, permitJSON string) error {
	core, err := c.current()
	if err != nil {
		return err
	}
	return core.StartTCP(endpoint, parentID, permitJSON)
}

func (c *Client) StatusJSON() (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.StatusJSON()
}

func (c *Client) WaitConnected(timeoutMS int64) error {
	core, err := c.current()
	if err != nil {
		return err
	}
	return core.WaitConnected(timeoutMS)
}

func (c *Client) CatalogJSON(ownerID, timeoutMS int64) (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.CatalogJSON(ownerID, timeoutMS)
}

func (c *Client) SnapshotJSON(ownerID int64, name string, timeoutMS int64) (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.SnapshotJSON(ownerID, name, timeoutMS)
}

func (c *Client) InvokeJSON(ownerID int64, name, requestJSON string, timeoutMS int64) (string, error) {
	core, err := c.current()
	if err != nil {
		return "", err
	}
	return core.InvokeJSON(ownerID, name, requestJSON, timeoutMS)
}

func (c *Client) Subscribe(ownerID int64, name string, leaseMS int64) (int64, error) {
	core, err := c.current()
	if err != nil {
		return 0, err
	}
	listener := newSubscription()
	id, err := core.Subscribe(ownerID, name, leaseMS, listener)
	if err != nil {
		listener.stop()
		return 0, err
	}
	c.mu.Lock()
	if c.core != core {
		c.mu.Unlock()
		core.CancelSubscription(id)
		listener.stop()
		return 0, errors.New("desktop binding client closed while subscribing")
	}
	c.subscriptions[id] = listener
	c.mu.Unlock()
	return id, nil
}

func (c *Client) PollSubscription(subscriptionID, timeoutMS int64) (string, error) {
	if subscriptionID <= 0 {
		return "", errors.New("desktop subscription ID must be positive")
	}
	timeout, err := pollDuration(timeoutMS)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	current := c.subscriptions[subscriptionID]
	c.mu.Unlock()
	if current == nil {
		return "", errors.New("desktop subscription was not found")
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case event := <-current.events:
		return pollResult("event", json.RawMessage(event))
	case failure := <-current.errors:
		return pollResult("error", json.RawMessage(failure))
	case <-current.done:
		return pollResult("closed", nil)
	case <-timer.C:
		return pollResult("timeout", nil)
	}
}

func (c *Client) CancelSubscription(subscriptionID int64) {
	if c == nil || subscriptionID <= 0 {
		return
	}
	c.mu.Lock()
	current := c.subscriptions[subscriptionID]
	delete(c.subscriptions, subscriptionID)
	core := c.core
	c.mu.Unlock()
	if current != nil {
		current.stop()
	}
	if core != nil {
		core.CancelSubscription(subscriptionID)
	}
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	core := c.core
	c.core = nil
	subscriptions := c.subscriptions
	c.subscriptions = nil
	c.mu.Unlock()
	for _, current := range subscriptions {
		current.stop()
	}
	if core == nil {
		return nil
	}
	return core.Close()
}

func (c *Client) current() (*bindings.Client, error) {
	if c == nil {
		return nil, errors.New("desktop binding client is not open")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.core == nil {
		return nil, errors.New("desktop binding client is not open")
	}
	return c.core, nil
}

type subscription struct {
	events chan string
	errors chan string
	done   chan struct{}
	once   sync.Once
}

func newSubscription() *subscription {
	return &subscription{events: make(chan string, 128), errors: make(chan string, 16), done: make(chan struct{})}
}

func (s *subscription) OnEvent(eventJSON string) {
	select {
	case s.events <- eventJSON:
	case <-s.done:
	}
}

func (s *subscription) OnError(errorJSON string) {
	select {
	case s.errors <- errorJSON:
	case <-s.done:
	}
}

func (s *subscription) stop() {
	s.once.Do(func() { close(s.done) })
}

func pollDuration(milliseconds int64) (time.Duration, error) {
	if milliseconds <= 0 {
		return 0, errors.New("desktop poll timeout must be positive")
	}
	duration := time.Duration(milliseconds) * time.Millisecond
	if duration <= 0 || duration > maxPollTimeout {
		return 0, fmt.Errorf("desktop poll timeout must not exceed %s", maxPollTimeout)
	}
	return duration, nil
}

func pollResult(kind string, payload json.RawMessage) (string, error) {
	value := struct {
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload,omitempty"`
	}{Kind: kind, Payload: payload}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode desktop subscription result: %w", err)
	}
	return string(data), nil
}
