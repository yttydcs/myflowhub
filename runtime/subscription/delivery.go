package subscription

import (
	"context"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
)

type EventKind uint8

const (
	EventVariableSnapshot EventKind = iota + 1
	EventVariableUpdate
	EventStream
	EventStreamGap
	EventExpired
)

type Event struct {
	Kind     EventKind
	Resource protocol.ResourceID
	Revision uint64
	Sequence uint64
	GapFrom  uint64
	GapTo    uint64
	Value    []byte
	Reason   string
}

func cloneEvent(event Event) Event {
	event.Value = append([]byte(nil), event.Value...)
	return event
}

type delivery struct {
	ctx    context.Context
	cancel context.CancelFunc
	out    chan Event
	wake   chan struct{}

	mu             sync.Mutex
	resource       protocol.ResourceID
	capacity       int
	pending        []Event
	latestVariable *Event
	gapFrom        uint64
	gapTo          uint64
	gapReason      string
	lastStream     uint64
	deferredStream *Event
	terminal       *Event
}

func newDelivery(parent context.Context, resource protocol.ResourceID, queue int) *delivery {
	ctx, cancel := context.WithCancel(parent)
	return &delivery{
		ctx:      ctx,
		cancel:   cancel,
		out:      make(chan Event),
		wake:     make(chan struct{}, 1),
		resource: resource,
		capacity: queue,
		pending:  make([]Event, 0, queue),
	}
}

func (d *delivery) start(onDone func()) {
	go func() {
		defer close(d.out)
		defer onDone()
		for {
			event, ok, terminal := d.next()
			if !ok {
				select {
				case <-d.wake:
					continue
				case <-d.ctx.Done():
					return
				}
			}
			select {
			case d.out <- event:
				if terminal {
					return
				}
			case <-d.ctx.Done():
				return
			}
		}
	}()
}

func (d *delivery) next() (Event, bool, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.gapFrom == 0 && d.deferredStream != nil {
		event := *d.deferredStream
		d.deferredStream = nil
		return event, true, false
	}
	if len(d.pending) > 0 {
		event := d.pending[0]
		copy(d.pending, d.pending[1:])
		d.pending = d.pending[:len(d.pending)-1]
		return event, true, false
	}
	if d.gapFrom != 0 {
		event := Event{Kind: EventStreamGap, Resource: d.resource, GapFrom: d.gapFrom, GapTo: d.gapTo, Reason: d.gapReason}
		d.gapFrom, d.gapTo = 0, 0
		d.gapReason = ""
		return event, true, false
	}
	if d.latestVariable != nil {
		event := *d.latestVariable
		d.latestVariable = nil
		return event, true, false
	}
	if d.terminal != nil {
		event := *d.terminal
		d.terminal = nil
		return event, true, true
	}
	return Event{}, false, false
}

func (d *delivery) enqueueSnapshot(event Event) {
	d.mu.Lock()
	d.pending = append(d.pending, cloneEvent(event))
	d.mu.Unlock()
	d.signal()
}

func (d *delivery) enqueueVariable(event Event) {
	event = cloneEvent(event)
	d.mu.Lock()
	d.latestVariable = &event
	d.mu.Unlock()
	d.signal()
}

func (d *delivery) enqueueStream(event Event) {
	event = cloneEvent(event)
	d.mu.Lock()
	if d.lastStream != 0 && event.Sequence > d.lastStream+1 && d.gapFrom == 0 {
		d.gapFrom, d.gapTo = d.lastStream+1, event.Sequence-1
		d.gapReason = "source_gap"
		d.deferredStream = &event
		d.lastStream = event.Sequence
	} else if d.gapFrom != 0 {
		d.gapTo = event.Sequence
		d.deferredStream = nil
		d.lastStream = event.Sequence
	} else if len(d.pending) >= d.capacity {
		d.gapFrom, d.gapTo = event.Sequence, event.Sequence
		d.gapReason = "slow_consumer"
		d.lastStream = event.Sequence
	} else {
		d.pending = append(d.pending, event)
		d.lastStream = event.Sequence
	}
	d.mu.Unlock()
	d.signal()
}

func (d *delivery) expire(reason string) {
	event := Event{Kind: EventExpired, Resource: d.resource, Reason: reason}
	d.mu.Lock()
	if d.terminal == nil {
		d.terminal = &event
	}
	d.mu.Unlock()
	d.signal()
}

func (d *delivery) signal() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

func (d *delivery) stop() { d.cancel() }
