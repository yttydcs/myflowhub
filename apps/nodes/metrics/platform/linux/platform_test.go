package linux

import (
	"context"
	"errors"
	"math"
	"net"
	"testing"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
)

func TestCPUUsesDeltaAndExcludesGuestDoubleCounting(t *testing.T) {
	before, err := parseCPU([]byte("cpu 100 10 20 400 10 2 3 5 90 8\ncpu0 ignored"))
	if err != nil {
		t.Fatal(err)
	}
	after, err := parseCPU([]byte("cpu 120 10 30 450 20 2 3 5 999 888\n"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := cpuPercent(before, after)
	if err != nil || math.Abs(value-100.0/3) > 0.00001 {
		t.Fatalf("got %v, %v", value, err)
	}
	for _, input := range []string{"", "cpu0 1 2 3 4", "cpu 1 2 -3 4", "cpu 1 x 3 4", "cpu 1 2"} {
		if _, err := parseCPU([]byte(input)); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
	for _, pair := range [][2]cpuTimes{
		{before, before},
		{before, {1, 0, 0, 0}},
		{{}, {math.MaxUint64, 1}},
	} {
		if _, err := cpuPercent(pair[0], pair[1]); err == nil {
			t.Fatal("accepted invalid CPU delta")
		}
	}
}

func TestCPUFirstSampleAndFailedReadNeverFabricateZero(t *testing.T) {
	p := New()
	step := 0
	p.readFile = func(string) ([]byte, error) {
		step++
		switch step {
		case 1, 4:
			return []byte("cpu 10 0 0 90 0 0 0 0"), nil
		case 2:
			return []byte("cpu 20 0 0 180 0 0 0 0"), nil
		default:
			return nil, errors.New("proc read failed")
		}
	}
	if value, err := p.Collect(context.Background(), metrics.CPUPercent); value != nil || err == nil {
		t.Fatalf("baseline produced %v, %v", value, err)
	}
	if value, err := p.Collect(context.Background(), metrics.CPUPercent); err != nil || value.(float64) != 10 {
		t.Fatalf("sample %v, %v", value, err)
	}
	if value, err := p.Collect(context.Background(), metrics.CPUPercent); value != nil || err == nil {
		t.Fatal("failed read produced sample")
	}
	if value, err := p.Collect(context.Background(), metrics.CPUPercent); value != nil || err == nil {
		t.Fatal("recovery reused stale baseline")
	}
}

func TestMemoryUsesAvailableNotFreeAndRejectsInvalidInput(t *testing.T) {
	value, err := memoryPercent([]byte("MemTotal: 1000 kB\nMemFree: 100 kB\nMemAvailable: 600 kB\n"))
	if err != nil || value != 40 {
		t.Fatalf("got %v, %v", value, err)
	}
	for _, input := range []string{
		"MemTotal: 100 kB", "MemAvailable: 10 kB", "MemTotal: 0 kB\nMemAvailable: 0 kB",
		"MemTotal: 100 kB\nMemAvailable: 101 kB", "MemTotal: -1 kB\nMemAvailable: 1 kB",
		"MemTotal: 100 MB\nMemAvailable: 50 kB", "MemTotal: 100 kB\nMemAvailable: x kB",
		"MemTotal: 100 kB\nMemTotal: 100 kB\nMemAvailable: 50 kB",
	} {
		if _, err := memoryPercent([]byte(input)); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
	p := New()
	p.readFile = func(string) ([]byte, error) { return nil, errors.New("denied") }
	if value, err := p.Collect(context.Background(), metrics.MemoryPercent); value != nil || err == nil {
		t.Fatal("read failure produced a sample")
	}
}

func TestNetworkReportsLocalAvailabilityAndErrors(t *testing.T) {
	p := New()
	p.interfaces = func() ([]net.Interface, error) {
		return []net.Interface{
			{Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
			{Name: "down", Flags: 0},
			{Name: "eth0", Flags: net.FlagUp, HardwareAddr: net.HardwareAddr{1, 2, 3, 4, 5, 6}},
		}, nil
	}
	p.addresses = func(net.Interface) ([]net.Addr, error) {
		return []net.Addr{&net.IPNet{IP: net.ParseIP("10.0.0.1")}}, nil
	}
	p.wireless = func(string) (bool, error) { return false, nil }
	if value, err := p.Collect(context.Background(), metrics.NetworkOnline); err != nil || value != true {
		t.Fatalf("online %v %v", value, err)
	}
	if value, err := p.Collect(context.Background(), metrics.NetworkType); err != nil || value != "ethernet" {
		t.Fatalf("kind %v %v", value, err)
	}
	p.wireless = func(string) (bool, error) { return true, nil }
	if value, err := p.Collect(context.Background(), metrics.NetworkType); err != nil || value != "wifi" {
		t.Fatalf("wifi %v %v", value, err)
	}
	p.addresses = func(net.Interface) ([]net.Addr, error) {
		return []net.Addr{&net.IPNet{IP: net.ParseIP("169.254.1.2")}}, nil
	}
	if value, err := p.Collect(context.Background(), metrics.NetworkOnline); err != nil || value != false {
		t.Fatalf("link-local %v %v", value, err)
	}
	p.addresses = func(net.Interface) ([]net.Addr, error) { return nil, errors.New("interface read failed") }
	if value, err := p.Collect(context.Background(), metrics.NetworkOnline); value != nil || err == nil {
		t.Fatal("network read failure fabricated offline")
	}
	p.interfaces = func() ([]net.Interface, error) { return nil, errors.New("interface list failed") }
	if value, err := p.Collect(context.Background(), metrics.NetworkType); value != nil || err == nil {
		t.Fatal("interface list failure fabricated kind")
	}
}

func TestContextAndUnsupportedMetrics(t *testing.T) {
	p := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Collect(ctx, metrics.CPUPercent); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := p.Collect(nil, metrics.MemoryPercent); err == nil {
		t.Fatal("accepted nil context")
	}
	if _, err := p.Collect(context.Background(), metrics.VolumePercent); !errors.Is(err, metrics.ErrUnsupported) {
		t.Fatal(err)
	}
}
