// Package linux provides read-only host metrics from procfs and interface state.
// Parser tests are portable; New is selected only by the Linux product entry.
package linux

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
)

const maxProcBytes = 64 << 10

type cpuTimes [8]uint64 // guest/guest_nice are already included in user/nice.

type Platform struct {
	readFile   func(string) ([]byte, error)
	interfaces func() ([]net.Interface, error)
	addresses  func(net.Interface) ([]net.Addr, error)
	wireless   func(string) (bool, error)

	mu       sync.Mutex
	previous cpuTimes
	hasBase  bool
}

var _ metrics.Collector = (*Platform)(nil)

func New() *Platform {
	return &Platform{
		readFile:   readProc,
		interfaces: net.Interfaces,
		addresses:  func(iface net.Interface) ([]net.Addr, error) { return iface.Addrs() },
		wireless: func(name string) (bool, error) {
			_, err := os.Stat(filepath.Join("/sys/class/net", name, "wireless"))
			if errors.Is(err, os.ErrNotExist) {
				return false, nil
			}
			return err == nil, err
		},
	}
}

func readProc(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// Only the first line of stat is needed; large machines can have many CPUs.
	if path == "/proc/stat" {
		line, err := bufio.NewReaderSize(file, 4096).ReadSlice('\n')
		if err != nil {
			return nil, fmt.Errorf("read proc CPU header: %w", err)
		}
		return line[:len(line)-1], nil
	}
	data, err := io.ReadAll(io.LimitReader(file, maxProcBytes+1))
	if err == nil && len(data) > maxProcBytes {
		return nil, errors.New("proc input exceeds size limit")
	}
	return data, err
}

func (p *Platform) Collect(ctx context.Context, name metrics.Name) (any, error) {
	if ctx == nil {
		return nil, errors.New("metrics context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch name {
	case metrics.CPUPercent:
		p.mu.Lock()
		defer p.mu.Unlock()
		data, err := p.readFile("/proc/stat")
		if err != nil {
			p.hasBase = false
			return nil, err
		}
		current, err := parseCPU(data)
		if err != nil {
			p.hasBase = false
			return nil, err
		}
		previous, hadBase := p.previous, p.hasBase
		p.previous, p.hasBase = current, true
		if !hadBase {
			return nil, errors.New("CPU sampling baseline initialized; waiting for next sample")
		}
		return cpuPercent(previous, current)
	case metrics.MemoryPercent:
		data, err := p.readFile("/proc/meminfo")
		if err != nil {
			return nil, err
		}
		return memoryPercent(data)
	case metrics.NetworkOnline, metrics.NetworkType:
		online, kind, err := p.network(ctx)
		if err != nil {
			return nil, err
		}
		if name == metrics.NetworkOnline {
			return online, nil
		}
		return kind, nil
	default:
		return nil, metrics.ErrUnsupported
	}
}

func parseCPU(data []byte) (cpuTimes, error) {
	var result cpuTimes
	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return result, errors.New("aggregate proc CPU counters are missing")
	}
	for index := 1; index < len(fields) && index <= len(result); index++ {
		value, err := strconv.ParseUint(fields[index], 10, 64)
		if err != nil {
			return cpuTimes{}, fmt.Errorf("invalid CPU counter: %w", err)
		}
		result[index-1] = value
	}
	return result, nil
}

func cpuPercent(previous, current cpuTimes) (float64, error) {
	var total, idle uint64
	for index, value := range current {
		if value < previous[index] {
			return 0, errors.New("CPU counters regressed; baseline reset")
		}
		delta := value - previous[index]
		if math.MaxUint64-total < delta {
			return 0, errors.New("CPU counter delta overflow")
		}
		total += delta
		if index == 3 || index == 4 { // idle and iowait.
			idle += delta // Cannot exceed the checked total.
		}
	}
	if total == 0 {
		return 0, errors.New("CPU sampling interval has no valid delta")
	}
	return float64(total-idle) * 100 / float64(total), nil
}

func memoryPercent(data []byte) (float64, error) {
	values := make(map[string]uint64, 2)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || (fields[0] != "MemTotal:" && fields[0] != "MemAvailable:") {
			continue
		}
		if len(fields) != 3 || fields[2] != "kB" {
			return 0, errors.New("memory counter unit or shape is invalid")
		}
		if _, duplicate := values[fields[0]]; duplicate {
			return 0, errors.New("duplicate memory counter")
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid memory counter: %w", err)
		}
		values[fields[0]] = value
	}
	total, hasTotal := values["MemTotal:"]
	available, hasAvailable := values["MemAvailable:"]
	if !hasTotal || !hasAvailable || total == 0 || available > total {
		return 0, errors.New("MemTotal or MemAvailable is missing or out of range")
	}
	return float64(total-available) * 100 / float64(total), nil
}

// This reports local interface availability, not successful access to the
// Internet or to any target. Private addresses are valid; loopback/link-local
// interfaces alone never count as online.
func (p *Platform) network(ctx context.Context) (bool, string, error) {
	interfaces, err := p.interfaces()
	if err != nil {
		return false, "", err
	}
	var observationErr error
	for _, iface := range interfaces {
		if err := ctx.Err(); err != nil {
			return false, "", err
		}
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := p.addresses(iface)
		if err != nil {
			observationErr = err
			continue
		}
		for _, address := range addresses {
			var ip net.IP
			switch value := address.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if !ip.IsGlobalUnicast() {
				continue
			}
			wireless, err := p.wireless(iface.Name)
			if err != nil {
				return false, "", err
			}
			if wireless {
				return true, "wifi", nil
			}
			if len(iface.HardwareAddr) == 6 {
				return true, "ethernet", nil
			}
			return true, "unknown", nil
		}
	}
	if observationErr != nil {
		return false, "", observationErr
	}
	return false, "none", nil
}
