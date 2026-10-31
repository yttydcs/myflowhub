//go:build linux

package linux

import (
	"context"
	"testing"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
)

func TestNativeLinuxReadOnlyCollection(t *testing.T) {
	p := New()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if value, err := p.Collect(ctx, metrics.CPUPercent); value != nil || err == nil {
		t.Fatalf("first CPU sample: %v, %v", value, err)
	}
	time.Sleep(100 * time.Millisecond)
	for _, name := range []metrics.Name{metrics.CPUPercent, metrics.MemoryPercent, metrics.NetworkOnline, metrics.NetworkType} {
		value, err := p.Collect(ctx, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := metrics.ValidateValue(name, value); err != nil {
			t.Fatalf("%s value: %v", name, err)
		}
		t.Logf("%s=%v", name, value)
	}
}
