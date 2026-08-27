package metricsmobile

import (
	"context"
	"errors"
	"sync"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"github.com/yttydcs/myflowhub/protocol"
)

const actionQueueCapacity = 64

type actionV1 struct {
	Version  int    `json:"version"`
	ActionID string `json:"action_id"`
	Metric   string `json:"metric"`
	Value    string `json:"value"`
}

func (a actionV1) Validate() error {
	if a.Version != 1 || len(a.ActionID) != 32 {
		return errors.New("mobile metric action version or ID is invalid")
	}
	definition, ok := metrics.DefinitionFor(metrics.Name(a.Metric))
	if !ok || !definition.Controllable || !definition.Platforms["android"] {
		return errors.New("mobile metric action is unsupported")
	}
	return metrics.ValidateValue(definition.Name, a.Value)
}

type actionQueue struct {
	mu          sync.Mutex
	queue       []actionV1
	outstanding map[string]actionV1
	closed      bool
}

func newActionQueue() *actionQueue {
	return &actionQueue{outstanding: make(map[string]actionV1)}
}

func (q *actionQueue) Set(ctx context.Context, name metrics.Name, value string) (metrics.ApplyResult, error) {
	if ctx == nil {
		return metrics.ApplyResult{}, errors.New("mobile metric action context is required")
	}
	select {
	case <-ctx.Done():
		return metrics.ApplyResult{}, ctx.Err()
	default:
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		return metrics.ApplyResult{}, err
	}
	action := actionV1{Version: 1, ActionID: id.String(), Metric: string(name), Value: value}
	if err := action.Validate(); err != nil {
		return metrics.ApplyResult{}, err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return metrics.ApplyResult{}, errors.New("mobile metric action queue is closed")
	}
	if len(q.outstanding) >= actionQueueCapacity {
		return metrics.ApplyResult{}, errors.New("mobile metric action queue is full")
	}
	q.outstanding[action.ActionID] = action
	q.queue = append(q.queue, action)
	return metrics.ApplyResult{Applied: false}, nil
}

func (q *actionQueue) next() (actionV1, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.queue) == 0 {
		return actionV1{}, false
	}
	action := q.queue[0]
	copy(q.queue, q.queue[1:])
	q.queue = q.queue[:len(q.queue)-1]
	return action, true
}

func (q *actionQueue) complete(actionID string) (actionV1, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	action, ok := q.outstanding[actionID]
	if !ok {
		return actionV1{}, errors.New("mobile metric action is unknown or already completed")
	}
	delete(q.outstanding, actionID)
	return action, nil
}

func (q *actionQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.queue = nil
	q.outstanding = make(map[string]actionV1)
	q.mu.Unlock()
}
