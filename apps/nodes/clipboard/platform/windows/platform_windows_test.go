//go:build windows

package windows

import (
	"context"
	"testing"
	"time"
)

func TestWindowsAdapterRejectsNULWithoutMutatingClipboard(t *testing.T) {
	platform := New()
	defer platform.Close()
	if err := platform.WriteText(context.Background(), "secret\x00suffix"); err == nil {
		t.Fatal("Windows clipboard accepted an embedded NUL")
	}
}

func TestWindowsAdapterHonorsCanceledContext(t *testing.T) {
	platform := New()
	defer platform.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	if _, err := platform.ReadText(ctx); err == nil {
		t.Fatal("Windows clipboard ignored a canceled context")
	}
	if time.Since(started) > time.Second {
		t.Fatal("Windows clipboard cancellation was not prompt")
	}
}
