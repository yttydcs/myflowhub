//go:build windows

package windows

import (
	"context"
	"testing"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"golang.org/x/sys/windows"
)

func TestWindowsCollectorSmoke(t *testing.T) {
	platform := New()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, name := range []metrics.Name{metrics.MemoryPercent, metrics.NetworkOnline, metrics.NetworkType} {
		value, err := platform.Collect(ctx, name)
		if err != nil {
			t.Fatalf("collect %s: %v", name, err)
		}
		if err := metrics.ValidateValue(name, value); err != nil {
			t.Fatalf("collect %s returned %q: %v", name, value, err)
		}
	}
	if _, err := platform.Collect(ctx, metrics.CPUPercent); err == nil {
		t.Fatal("first CPU collection did not establish an explicit baseline")
	}
	time.Sleep(50 * time.Millisecond)
	value, err := platform.Collect(ctx, metrics.CPUPercent)
	if err != nil {
		t.Fatalf("collect CPU after baseline: %v", err)
	}
	if err := metrics.ValidateValue(metrics.CPUPercent, value); err != nil {
		t.Fatalf("CPU collector returned %q: %v", value, err)
	}
}

func TestWindowsNetworkTypeAndVariantConversion(t *testing.T) {
	if value, rank := networkType(windows.IF_TYPE_ETHERNET_CSMACD); value != "ethernet" || rank != 3 {
		t.Fatalf("unexpected Ethernet mapping: %q %d", value, rank)
	}
	if value, rank := networkType(windows.IF_TYPE_IEEE80211); value != "wifi" || rank != 2 {
		t.Fatalf("unexpected Wi-Fi mapping: %q %d", value, rank)
	}
	if value, ok := numericVariant(uint8(75)); !ok || value != 75 {
		t.Fatalf("unexpected WMI variant conversion: %d %v", value, ok)
	}
}
