// Package variables is a standard Variable provider with no consumer-specific
// schemas, renderers or core extensions. Its values are explicitly synthetic.
package variables

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

const ReferenceCount = 1998

var kinds = []string{"bool", "number", "enum", "text", "object", "array"}

type Provider struct {
	Variables  []*resource.Variable
	samples    atomic.Uint64
	superseded atomic.Uint64
}

func Register(registry *resource.Registry, owner protocol.NodeID, count int) (*Provider, error) {
	if registry == nil {
		return nil, errors.New("example registry is required")
	}
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	if count < 6 || count > 3996 || count%6 != 0 {
		return nil, errors.New("example count must be a multiple of six in 6..3996")
	}
	provider := &Provider{Variables: make([]*resource.Variable, count)}
	batch := make([]resource.Resource, count)
	for i := range batch {
		kind := i % 6
		id := protocol.ResourceID{Owner: owner, Name: fmt.Sprintf("example/foundation/%s/%04d", kinds[kind], i/6)}
		descriptor := resource.VariableDescriptor(id, "example.read", 4096)
		definition := definition(kind)
		var handler resource.VariableWriteHandler
		if kind < 2 {
			descriptor = resource.WritableVariableDescriptor(id, "example.read", "example.write", 4096)
			handler = func(ctx context.Context, value any) (resource.VariableState, error) {
				if err := ctx.Err(); err != nil {
					return resource.VariableState{}, err
				}
				encoded, err := json.Marshal(value)
				return resource.VariableState{Status: "ready", Value: encoded}, err
			}
		}
		descriptor.Presentation.Label = fmt.Sprintf("示例负载 %s %04d", kinds[kind], i/6)
		descriptor.Presentation.Description = "Synthetic reference load; not a physical device measurement."
		variable, err := resource.NewVariable(descriptor, definition, resource.VariableState{Status: "ready", Value: sampleValue(kind, i, 0)}, handler)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", id.Name, err)
		}
		provider.Variables[i] = variable
		batch[i] = variable
	}
	if err := registry.RegisterBatch(batch); err != nil {
		return nil, err
	}
	return provider, nil
}

func definition(kind int) protocol.DataSchemaDefinition {
	maxText, maxArray := 128, 8
	switch kind {
	case 0:
		return protocol.DataSchemaDefinition{Type: "boolean", Title: "示例负载 · 布尔"}
	case 1:
		low, high := 0.0, 100.0
		return protocol.DataSchemaDefinition{Type: "number", Minimum: &low, Maximum: &high, Title: "示例负载 · 数字"}
	case 2:
		return protocol.DataSchemaDefinition{Type: "string", Enum: []string{"busy", "normal", "quiet"}, ReadOnly: true, Title: "示例负载 · 枚举"}
	case 3:
		return protocol.DataSchemaDefinition{Type: "string", MaxLength: &maxText, ReadOnly: true, Title: "示例负载 · 文本"}
	case 4:
		return protocol.DataSchemaDefinition{Type: "object", ReadOnly: true, Title: "示例负载 · 对象", Properties: []protocol.DataSchemaProperty{
			{Name: "sample", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "integer"}},
			{Name: "tick", Required: true, Schema: &protocol.DataSchemaDefinition{Type: "integer"}},
		}}
	default:
		return protocol.DataSchemaDefinition{Type: "array", MaxItems: &maxArray, Items: &protocol.DataSchemaDefinition{Type: "number"}, ReadOnly: true, Title: "示例负载 · 数组"}
	}
}

func sampleValue(kind, index int, tick uint64) json.RawMessage {
	var value any
	switch kind {
	case 0:
		value = index%2 == 0
	case 1:
		value = index % 101
	case 2:
		value = []string{"quiet", "normal", "busy"}[(uint64(index)+tick)%3]
	case 3:
		value = fmt.Sprintf("synthetic sample %d tick %d", index, tick)
	case 4:
		value = map[string]any{"tick": tick % 1000000, "sample": index}
	default:
		value = []int{index % 100, int(tick % 100), (index + int(tick%100)) % 100}
	}
	encoded, _ := json.Marshal(value) // Only the bounded, serializable values above.
	return encoded
}

func (p *Provider) Sample(tick uint64) error {
	for i, variable := range p.Variables {
		before := variable.Snapshot()
		value := before.State.Value
		if i%6 >= 2 {
			value = sampleValue(i%6, i, tick)
		}
		// A concurrent write may win after Snapshot. A fixed next revision then
		// rejects this sample; Set(oldValue) would overwrite the user's write.
		_, err := variable.Apply(before.Revision+1, resource.VariableState{Status: "ready", Value: value})
		if errors.Is(err, resource.ErrRevisionRegression) {
			p.superseded.Add(1)
			continue
		}
		if err != nil {
			return fmt.Errorf("sample %s: %w", variable.Descriptor().ID.Name, err)
		}
		p.samples.Add(1)
	}
	return nil
}

func (p *Provider) Counts() (samples, superseded uint64) {
	return p.samples.Load(), p.superseded.Load()
}

func (p *Provider) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("example context is required")
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var tick uint64
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			tick++
			if err := p.Sample(tick); err != nil {
				return err
			}
		}
	}
}
