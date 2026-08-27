package metricsmobile

import (
	"context"
	"testing"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
)

func TestActionQueueRequiresExplicitCompletion(t *testing.T) {
	queue := newActionQueue()
	result, err := queue.Set(context.Background(), metrics.VolumePercent, "42")
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied {
		t.Fatal("mobile action must not claim platform application before Kotlin confirms it")
	}
	action, ok := queue.next()
	if !ok || action.Metric != string(metrics.VolumePercent) || action.Value != "42" {
		t.Fatalf("unexpected action: %+v", action)
	}
	completed, err := queue.complete(action.ActionID)
	if err != nil || completed.ActionID != action.ActionID {
		t.Fatalf("complete action: %+v %v", completed, err)
	}
	if _, err := queue.complete(action.ActionID); err == nil {
		t.Fatal("expected duplicate completion error")
	}
}

func TestActionQueueIsBounded(t *testing.T) {
	queue := newActionQueue()
	for index := 0; index < actionQueueCapacity; index++ {
		if _, err := queue.Set(context.Background(), metrics.VolumeMuted, "true"); err != nil {
			t.Fatalf("enqueue %d: %v", index, err)
		}
	}
	if _, err := queue.Set(context.Background(), metrics.VolumeMuted, "false"); err == nil {
		t.Fatal("expected queue capacity error")
	}
}
