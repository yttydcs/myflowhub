package link

import (
	"sync/atomic"

	"github.com/yttydcs/myflowhub/protocol"
)

// A count limit protects against tiny-frame floods; the byte budget also bounds
// retained large payloads. A frame in the writer/dispatcher still owns its bytes.
type envelopeQueue struct {
	frames   chan protocol.Envelope
	bytes    atomic.Int64
	maxBytes int64
}

func newEnvelopeQueue(count int, maxBytes int64) *envelopeQueue {
	return &envelopeQueue{frames: make(chan protocol.Envelope, count), maxBytes: maxBytes}
}

func envelopeBytes(envelope protocol.Envelope) int64 {
	return int64(512 + len(envelope.Payload) + len(envelope.ContentType) + len(envelope.Schema) + len(envelope.Capability) + len(envelope.Resource.Name))
}

func (q *envelopeQueue) push(envelope protocol.Envelope) bool {
	size := envelopeBytes(envelope)
	for {
		before := q.bytes.Load()
		if q.maxBytes > 0 && size > q.maxBytes-before {
			return false
		}
		if q.bytes.CompareAndSwap(before, before+size) {
			break
		}
	}
	select {
	case q.frames <- envelope:
		return true
	default:
		q.release(envelope)
		return false
	}
}

func (q *envelopeQueue) release(envelope protocol.Envelope) {
	q.bytes.Add(-envelopeBytes(envelope))
}
