package resource

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
)

type VariableUpdate struct {
	Revision uint64
	Value    []byte
}

type Variable struct {
	mu         sync.RWMutex
	descriptor Descriptor
	revision   uint64
	value      []byte
	nextWatch  uint64
	watchers   map[uint64]func(VariableUpdate)
}

func NewVariable(descriptor Descriptor, initial []byte) (*Variable, error) {
	var err error
	descriptor, err = normalizeDescriptor(descriptor)
	if err != nil {
		return nil, err
	}
	if descriptor.Type != protocol.ResourceTypeVariable {
		return nil, errors.New("variable descriptor must use mfh.variable type")
	}
	if _, ok := descriptor.Capability(protocol.CapabilityRead); !ok {
		return nil, errors.New("variable descriptor requires read capability")
	}
	if _, ok := descriptor.Capability(protocol.CapabilitySubscribe); !ok {
		return nil, errors.New("variable descriptor requires subscribe capability")
	}
	if err := validatePayload(descriptor, initial); err != nil {
		return nil, err
	}
	return &Variable{descriptor: descriptor, revision: 1, value: append([]byte(nil), initial...), watchers: make(map[uint64]func(VariableUpdate))}, nil
}

func (v *Variable) Descriptor() Descriptor { return cloneDescriptor(v.descriptor) }

func (v *Variable) Operate(ctx context.Context, request OperationRequest) (OperationResult, error) {
	if ctx == nil {
		return OperationResult{}, errors.New("variable context is required")
	}
	switch request.Capability {
	case protocol.CapabilityRead:
		value := v.Snapshot()
		capability, _ := v.descriptor.Capability(protocol.CapabilityRead)
		return OperationResult{Schema: capability.OutputSchema, Payload: value.Value}, nil
	case protocol.CapabilityWrite:
		if _, ok := v.descriptor.Capability(protocol.CapabilityWrite); !ok {
			return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
		}
		var write protocol.VariableWriteV2
		if err := protocol.DecodeJSONPayload(request.Payload, v.descriptor.Limits.MaxPayloadBytes, &write); err != nil {
			return OperationResult{}, err
		}
		value, err := v.CompareAndSet(write.ExpectedRevision, write.Value)
		if err != nil {
			return OperationResult{}, err
		}
		capability, _ := v.descriptor.Capability(protocol.CapabilityWrite)
		return OperationResult{Schema: capability.OutputSchema, Payload: value.Value}, nil
	default:
		return OperationResult{}, fmt.Errorf("%w: %s", ErrUnsupportedCapability, request.Capability)
	}
}

func (v *Variable) CompareAndSet(expectedRevision uint64, value []byte) (VariableUpdate, error) {
	if expectedRevision == 0 {
		return VariableUpdate{}, errors.New("variable expected revision must be non-zero")
	}
	if err := validatePayload(v.descriptor, value); err != nil {
		return VariableUpdate{}, err
	}
	v.mu.Lock()
	if v.revision != expectedRevision {
		current := v.revision
		v.mu.Unlock()
		return VariableUpdate{}, fmt.Errorf("%w: got %d, current %d", ErrRevisionConflict, expectedRevision, current)
	}
	if v.revision == math.MaxUint64 {
		v.mu.Unlock()
		return VariableUpdate{}, errors.New("variable revision exhausted")
	}
	v.revision++
	v.value = append(v.value[:0], value...)
	update := VariableUpdate{Revision: v.revision, Value: append([]byte(nil), v.value...)}
	watchers := v.copyWatchersLocked()
	v.mu.Unlock()
	notifyVariable(watchers, update)
	return update, nil
}

func (v *Variable) Observe(observer func(Observation)) (*Observation, func(), error) {
	snapshot, cancel, err := v.Watch(func(update VariableUpdate) {
		observer(Observation{Revision: update.Revision, Schema: v.eventSchema(), Value: update.Value})
	})
	if err != nil {
		return nil, nil, err
	}
	initial := &Observation{Snapshot: true, Revision: snapshot.Revision, Schema: v.eventSchema(), Value: snapshot.Value}
	return initial, cancel, nil
}

func (v *Variable) Snapshot() VariableUpdate {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return VariableUpdate{Revision: v.revision, Value: append([]byte(nil), v.value...)}
}

func (v *Variable) Set(value []byte) (VariableUpdate, error) {
	if err := validatePayload(v.descriptor, value); err != nil {
		return VariableUpdate{}, err
	}
	v.mu.Lock()
	if v.revision == math.MaxUint64 {
		v.mu.Unlock()
		return VariableUpdate{}, errors.New("variable revision exhausted")
	}
	v.revision++
	v.value = append(v.value[:0], value...)
	update := VariableUpdate{Revision: v.revision, Value: append([]byte(nil), v.value...)}
	watchers := v.copyWatchersLocked()
	v.mu.Unlock()
	notifyVariable(watchers, update)
	return update, nil
}

func (v *Variable) Apply(revision uint64, value []byte) (VariableUpdate, error) {
	if err := validatePayload(v.descriptor, value); err != nil {
		return VariableUpdate{}, err
	}
	v.mu.Lock()
	if revision <= v.revision {
		v.mu.Unlock()
		return VariableUpdate{}, ErrRevisionRegression
	}
	v.revision = revision
	v.value = append(v.value[:0], value...)
	update := VariableUpdate{Revision: revision, Value: append([]byte(nil), v.value...)}
	watchers := v.copyWatchersLocked()
	v.mu.Unlock()
	notifyVariable(watchers, update)
	return update, nil
}

func (v *Variable) eventSchema() string {
	capability, _ := v.descriptor.Capability(protocol.CapabilitySubscribe)
	return capability.EventSchema
}

func (v *Variable) Watch(observer func(VariableUpdate)) (VariableUpdate, func(), error) {
	if observer == nil {
		return VariableUpdate{}, nil, errors.New("variable observer is required")
	}
	v.mu.Lock()
	v.nextWatch++
	id := v.nextWatch
	v.watchers[id] = observer
	snapshot := VariableUpdate{Revision: v.revision, Value: append([]byte(nil), v.value...)}
	v.mu.Unlock()
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			v.mu.Lock()
			delete(v.watchers, id)
			v.mu.Unlock()
		})
	}
	return snapshot, cancel, nil
}

func (v *Variable) copyWatchersLocked() []func(VariableUpdate) {
	result := make([]func(VariableUpdate), 0, len(v.watchers))
	for _, watcher := range v.watchers {
		result = append(result, watcher)
	}
	return result
}

func notifyVariable(watchers []func(VariableUpdate), update VariableUpdate) {
	for _, watcher := range watchers {
		watcher(VariableUpdate{Revision: update.Revision, Value: append([]byte(nil), update.Value...)})
	}
}
