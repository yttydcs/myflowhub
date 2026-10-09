//go:build windows

package windows

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
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

func TestEndpointMuteUsesWin32BOOLAndSuccessHRESULT(t *testing.T) {
	var muteValue int32 = 1
	var calls int
	var result uintptr = 1 // S_FALSE: already requested mute state.
	table := &wca.IAudioEndpointVolumeVtbl{}
	table.GetMute = syscall.NewCallback(func(_ uintptr, output *int32) uintptr { *output = muteValue; return 0 })
	table.GetMasterVolumeLevelScalar = syscall.NewCallback(func(_ uintptr, output *float32) uintptr { *output = 0.38; return 0 })
	table.SetMute = syscall.NewCallback(func(_ uintptr, value int32, _ uintptr) uintptr { calls++; muteValue = value; return result })
	endpoint := &wca.IAudioEndpointVolume{IUnknown: ole.IUnknown{RawVTable: (*interface{})(unsafe.Pointer(table))}}
	for _, expected := range []bool{true, false, true} {
		if err := setEndpointMute(endpoint, expected); err != nil {
			t.Fatalf("successful S_FALSE was rejected: %v", err)
		}
		percent, muted, err := readEndpointVolume(endpoint)
		if err != nil || percent != 38 || muted != expected {
			t.Fatalf("BOOL output corrupted adjacent volume: percent=%d muted=%t err=%v", percent, muted, err)
		}
	}
	if calls != 3 {
		t.Fatalf("mute calls=%d, want 3", calls)
	}
	result = 0x80004005
	if err := setEndpointMute(endpoint, false); err == nil {
		t.Fatal("failed HRESULT was accepted")
	}
	if calls != 4 {
		t.Fatal("failed mute was retried")
	}
}
