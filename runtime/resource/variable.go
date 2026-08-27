package resource

import (
	"errors"
	"math"
	"sync"
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
	descriptor = normalizeDescriptor(descriptor)
	if descriptor.Kind != KindVariable {
		return nil, errors.New("variable descriptor must use variable kind")
	}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	if err := descriptor.validateValue(initial); err != nil {
		return nil, err
	}
	return &Variable{descriptor: descriptor, revision: 1, value: append([]byte(nil), initial...), watchers: make(map[uint64]func(VariableUpdate))}, nil
}

func (v *Variable) Descriptor() Descriptor { return v.descriptor }

func (v *Variable) Snapshot() VariableUpdate {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return VariableUpdate{Revision: v.revision, Value: append([]byte(nil), v.value...)}
}

func (v *Variable) Set(value []byte) (VariableUpdate, error) {
	if err := v.descriptor.validateValue(value); err != nil {
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
	if err := v.descriptor.validateValue(value); err != nil {
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
