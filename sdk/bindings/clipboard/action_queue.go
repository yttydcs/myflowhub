package clipboardmobile

import (
	"context"
	"errors"
	"sync"
	"time"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
	"github.com/yttydcs/myflowhub/protocol"
)

const actionQueueCapacity = 64

type writeActionV1 struct {
	Version  int    `json:"version"`
	ActionID string `json:"action_id"`
	Text     string `json:"text"`
}

func (a writeActionV1) Validate() error {
	if a.Version != 1 || len(a.ActionID) != 32 || a.Text == "" || len([]byte(a.Text)) > clipboard.MaxTextBytes {
		return errors.New("mobile clipboard write action is invalid")
	}
	return nil
}

type writeResult struct{ err error }

type queuedWrite struct {
	action writeActionV1
	result chan writeResult
}

type mobileAdapter struct {
	mu          sync.Mutex
	queue       []*queuedWrite
	outstanding map[string]*queuedWrite
	observed    chan clipboard.TextObservation
	errors      chan error
	closed      bool
}

func newMobileAdapter() *mobileAdapter {
	return &mobileAdapter{
		outstanding: make(map[string]*queuedWrite), observed: make(chan clipboard.TextObservation, actionQueueCapacity),
		errors: make(chan error, 1),
	}
}

func (a *mobileAdapter) ReadText(context.Context) (string, error) {
	return "", errors.New("mobile clipboard reads are supplied as observations")
}

func (a *mobileAdapter) WriteText(ctx context.Context, text string) error {
	if ctx == nil {
		return errors.New("mobile clipboard write context is required")
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		return err
	}
	value := &queuedWrite{action: writeActionV1{Version: 1, ActionID: id.String(), Text: text}, result: make(chan writeResult, 1)}
	if err := value.action.Validate(); err != nil {
		return err
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return errors.New("mobile clipboard adapter is closed")
	}
	if len(a.outstanding) >= actionQueueCapacity {
		a.mu.Unlock()
		return errors.New("mobile clipboard write queue is full")
	}
	a.outstanding[value.action.ActionID] = value
	a.queue = append(a.queue, value)
	a.mu.Unlock()
	select {
	case result := <-value.result:
		return result.err
	case <-ctx.Done():
		a.remove(value.action.ActionID)
		return ctx.Err()
	}
}

func (a *mobileAdapter) WatchText(ctx context.Context) (<-chan clipboard.TextObservation, <-chan error, error) {
	if ctx == nil {
		return nil, nil, errors.New("mobile clipboard watch context is required")
	}
	return a.observed, a.errors, nil
}

func (a *mobileAdapter) Close() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.closed = true
	pending := make([]*queuedWrite, 0, len(a.outstanding))
	for _, value := range a.outstanding {
		pending = append(pending, value)
	}
	a.queue = nil
	a.outstanding = make(map[string]*queuedWrite)
	a.mu.Unlock()
	for _, value := range pending {
		value.result <- writeResult{err: errors.New("mobile clipboard adapter is closed")}
	}
	return nil
}

func (a *mobileAdapter) observe(text string) error {
	if text == "" || len([]byte(text)) > clipboard.MaxTextBytes {
		return errors.New("mobile clipboard observation is invalid")
	}
	a.mu.Lock()
	closed := a.closed
	a.mu.Unlock()
	if closed {
		return errors.New("mobile clipboard adapter is closed")
	}
	select {
	case a.observed <- clipboard.TextObservation{Text: text, ObservedAt: time.Now().UTC()}:
		return nil
	default:
		return errors.New("mobile clipboard observation queue is full")
	}
}

func (a *mobileAdapter) next() (writeActionV1, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.queue) == 0 {
		return writeActionV1{}, false
	}
	value := a.queue[0]
	a.queue = a.queue[1:]
	return value.action, true
}

func (a *mobileAdapter) complete(actionID, errorMessage string) error {
	a.mu.Lock()
	value, ok := a.outstanding[actionID]
	if ok {
		delete(a.outstanding, actionID)
	}
	a.mu.Unlock()
	if !ok {
		return errors.New("mobile clipboard write action is unknown or already completed")
	}
	var err error
	if errorMessage != "" {
		err = errors.New("mobile platform clipboard write failed")
	}
	value.result <- writeResult{err: err}
	return nil
}

func (a *mobileAdapter) remove(actionID string) {
	a.mu.Lock()
	delete(a.outstanding, actionID)
	for index, value := range a.queue {
		if value.action.ActionID == actionID {
			a.queue = append(a.queue[:index], a.queue[index+1:]...)
			break
		}
	}
	a.mu.Unlock()
}

var _ clipboard.Adapter = (*mobileAdapter)(nil)
