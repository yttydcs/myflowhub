//go:build windows

package windows

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/moutend/go-wca/pkg/wca"
	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"golang.org/x/sys/windows"
)

type Platform struct {
	mu         sync.Mutex
	previous   systemTimes
	hasCPUBase bool
}

func New() *Platform { return &Platform{} }

func (p *Platform) Collect(ctx context.Context, name metrics.Name) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	switch name {
	case metrics.BatteryPercent:
		status, err := readPowerStatus()
		if err != nil {
			return "", err
		}
		if status.BatteryFlag == 128 || status.BatteryLifePercent == 255 {
			return "", errors.New("system battery is unavailable")
		}
		if status.BatteryLifePercent > 100 {
			return "", errors.New("battery percentage is out of range")
		}
		return strconv.Itoa(int(status.BatteryLifePercent)), nil
	case metrics.BatteryCharging:
		status, err := readPowerStatus()
		if err != nil {
			return "", err
		}
		if status.BatteryFlag == 128 {
			return "", errors.New("system battery is unavailable")
		}
		return strconv.FormatBool(status.BatteryFlag&8 != 0), nil
	case metrics.BatteryOnAC:
		status, err := readPowerStatus()
		if err != nil {
			return "", err
		}
		if status.ACLineStatus > 1 {
			return "", errors.New("AC line status is unknown")
		}
		return strconv.FormatBool(status.ACLineStatus == 1), nil
	case metrics.CPUPercent:
		current, err := readSystemTimes()
		if err != nil {
			return "", err
		}
		if !p.hasCPUBase {
			p.previous = current
			p.hasCPUBase = true
			return "", errors.New("CPU sampling baseline initialized")
		}
		previous := p.previous
		p.previous = current
		if current.idle < previous.idle || current.kernel < previous.kernel || current.user < previous.user {
			return "", errors.New("CPU counters regressed")
		}
		total := current.kernel - previous.kernel + current.user - previous.user
		idle := current.idle - previous.idle
		if total == 0 || idle > total {
			return "", errors.New("CPU sampling interval has no valid delta")
		}
		percent := math.Round(float64(total-idle) * 100 / float64(total))
		return strconv.FormatFloat(percent, 'f', 0, 64), nil
	case metrics.MemoryPercent:
		return readMemoryPercent()
	case metrics.NetworkOnline, metrics.NetworkType:
		online, networkType, err := readNetwork()
		if err != nil {
			return "", err
		}
		if name == metrics.NetworkOnline {
			return strconv.FormatBool(online), nil
		}
		return networkType, nil
	case metrics.VolumePercent, metrics.VolumeMuted:
		var percent int
		var muted bool
		err := withCOM(func() error {
			endpoint, release, err := openDefaultEndpointVolume()
			if err != nil {
				return err
			}
			defer release()
			percent, muted, err = readEndpointVolume(endpoint)
			return err
		})
		if err != nil {
			return "", err
		}
		if name == metrics.VolumePercent {
			return strconv.Itoa(percent), nil
		}
		return strconv.FormatBool(muted), nil
	case metrics.BrightnessPercent:
		return readBrightnessPercent()
	default:
		return "", metrics.ErrUnsupported
	}
}

func (p *Platform) Set(ctx context.Context, name metrics.Name, value string) (metrics.ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return metrics.ApplyResult{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	switch name {
	case metrics.VolumePercent:
		percent, _ := strconv.Atoi(value)
		err := withCOM(func() error {
			endpoint, release, err := openDefaultEndpointVolume()
			if err != nil {
				return err
			}
			defer release()
			return endpoint.SetMasterVolumeLevelScalar(float32(percent)/100, nil)
		})
		if err != nil {
			return metrics.ApplyResult{}, err
		}
		return metrics.ApplyResult{Applied: true, Value: strconv.Itoa(percent)}, nil
	case metrics.VolumeMuted:
		muted, _ := strconv.ParseBool(value)
		err := withCOM(func() error {
			endpoint, release, err := openDefaultEndpointVolume()
			if err != nil {
				return err
			}
			defer release()
			return endpoint.SetMute(muted, nil)
		})
		if err != nil {
			return metrics.ApplyResult{}, err
		}
		return metrics.ApplyResult{Applied: true, Value: strconv.FormatBool(muted)}, nil
	case metrics.BrightnessPercent:
		percent, _ := strconv.Atoi(value)
		if err := setBrightnessPercent(percent); err != nil {
			return metrics.ApplyResult{}, err
		}
		return metrics.ApplyResult{Applied: true, Value: strconv.Itoa(percent)}, nil
	default:
		return metrics.ApplyResult{}, metrics.ErrUnsupported
	}
}

type powerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

type fileTime struct {
	Low  uint32
	High uint32
}

func (t fileTime) value() uint64 { return uint64(t.High)<<32 | uint64(t.Low) }

type systemTimes struct{ idle, kernel, user uint64 }

type memoryStatus struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type physicalMonitor struct {
	Handle syscall.Handle
	Desc   [128]uint16
}

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getSystemPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
	getSystemTimes       = kernel32.NewProc("GetSystemTimes")
	globalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	user32               = syscall.NewLazyDLL("user32.dll")
	getDesktopWindow     = user32.NewProc("GetDesktopWindow")
	monitorFromWindow    = user32.NewProc("MonitorFromWindow")
	dxva2                = syscall.NewLazyDLL("dxva2.dll")
	physicalMonitorCount = dxva2.NewProc("GetNumberOfPhysicalMonitorsFromHMONITOR")
	physicalMonitors     = dxva2.NewProc("GetPhysicalMonitorsFromHMONITOR")
	destroyMonitor       = dxva2.NewProc("DestroyPhysicalMonitor")
	getMonitorBrightness = dxva2.NewProc("GetMonitorBrightness")
	setMonitorBrightness = dxva2.NewProc("SetMonitorBrightness")
)

func readPowerStatus() (powerStatus, error) {
	var value powerStatus
	result, _, callErr := getSystemPowerStatus.Call(uintptr(unsafe.Pointer(&value)))
	if result == 0 {
		return powerStatus{}, callError("GetSystemPowerStatus", callErr)
	}
	return value, nil
}

func readSystemTimes() (systemTimes, error) {
	var idle, kernel, user fileTime
	result, _, callErr := getSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)),
	)
	if result == 0 {
		return systemTimes{}, callError("GetSystemTimes", callErr)
	}
	return systemTimes{idle: idle.value(), kernel: kernel.value(), user: user.value()}, nil
}

func readMemoryPercent() (string, error) {
	var value memoryStatus
	value.Length = uint32(unsafe.Sizeof(value))
	result, _, callErr := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&value)))
	if result == 0 {
		return "", callError("GlobalMemoryStatusEx", callErr)
	}
	if value.MemoryLoad > 100 {
		return "", errors.New("memory percentage is out of range")
	}
	return strconv.Itoa(int(value.MemoryLoad)), nil
}

const (
	ifTypeWWANPP  = 243
	ifTypeWWANPP2 = 244
)

func readNetwork() (bool, string, error) {
	flags := uint32(windows.GAA_FLAG_SKIP_ANYCAST | windows.GAA_FLAG_SKIP_MULTICAST | windows.GAA_FLAG_SKIP_DNS_SERVER | windows.GAA_FLAG_SKIP_FRIENDLY_NAME)
	size := uint32(15 * 1024)
	for attempt := 0; attempt < 3; attempt++ {
		buffer := make([]byte, size)
		addresses := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buffer[0]))
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, flags, 0, addresses, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return false, "", err
		}
		bestRank := -1
		bestType := "none"
		for current := addresses; current != nil; current = current.Next {
			if current.OperStatus != windows.IfOperStatusUp || current.IfType == windows.IF_TYPE_SOFTWARE_LOOPBACK || current.IfType == windows.IF_TYPE_TUNNEL || current.FirstUnicastAddress == nil {
				continue
			}
			value, rank := networkType(current.IfType)
			if rank > bestRank {
				bestType, bestRank = value, rank
			}
		}
		return bestRank >= 0, bestType, nil
	}
	return false, "", errors.New("GetAdaptersAddresses buffer overflow")
}

func networkType(value uint32) (string, int) {
	switch value {
	case windows.IF_TYPE_ETHERNET_CSMACD:
		return "ethernet", 3
	case windows.IF_TYPE_IEEE80211:
		return "wifi", 2
	case ifTypeWWANPP, ifTypeWWANPP2:
		return "cellular", 1
	default:
		return "unknown", 0
	}
}

func openDefaultEndpointVolume() (*wca.IAudioEndpointVolume, func(), error) {
	var enumerator *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &enumerator); err != nil {
		return nil, nil, err
	}
	if enumerator == nil {
		return nil, nil, errors.New("audio device enumerator is nil")
	}
	var device *wca.IMMDevice
	if err := enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &device); err != nil {
		enumerator.Release()
		return nil, nil, err
	}
	if device == nil {
		enumerator.Release()
		return nil, nil, errors.New("default audio device is nil")
	}
	var endpoint *wca.IAudioEndpointVolume
	if err := device.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &endpoint); err != nil {
		device.Release()
		enumerator.Release()
		return nil, nil, err
	}
	if endpoint == nil {
		device.Release()
		enumerator.Release()
		return nil, nil, errors.New("audio endpoint volume is nil")
	}
	return endpoint, func() { endpoint.Release(); device.Release(); enumerator.Release() }, nil
}

func readEndpointVolume(endpoint *wca.IAudioEndpointVolume) (int, bool, error) {
	var level float32
	if err := endpoint.GetMasterVolumeLevelScalar(&level); err != nil {
		return 0, false, err
	}
	level = max(0, min(1, level))
	var muted bool
	if err := endpoint.GetMute(&muted); err != nil {
		return 0, false, err
	}
	return int(math.Round(float64(level * 100))), muted, nil
}

func readBrightnessPercent() (string, error) {
	if percent, err := readBrightnessDXVA2(); err == nil {
		return strconv.Itoa(percent), nil
	} else {
		var percent int
		wmiErr := withCOM(func() error {
			value, found, err := readBrightnessWMI()
			percent = value
			if err == nil && !found {
				return errors.New("WMI brightness provider was not found")
			}
			return err
		})
		if wmiErr != nil {
			return "", fmt.Errorf("read brightness: DXVA2: %v; WMI: %w", err, wmiErr)
		}
		return strconv.Itoa(percent), nil
	}
}

func readBrightnessDXVA2() (int, error) {
	monitor, release, err := primaryPhysicalMonitor()
	if err != nil {
		return 0, err
	}
	defer release()
	var minimum, current, maximum uint32
	result, _, callErr := getMonitorBrightness.Call(
		uintptr(monitor), uintptr(unsafe.Pointer(&minimum)), uintptr(unsafe.Pointer(&current)), uintptr(unsafe.Pointer(&maximum)),
	)
	if result == 0 {
		return 0, callError("GetMonitorBrightness", callErr)
	}
	if maximum <= minimum {
		return 0, errors.New("monitor brightness range is invalid")
	}
	current = max(minimum, min(maximum, current))
	return int(uint64(current-minimum) * 100 / uint64(maximum-minimum)), nil
}

func setBrightnessPercent(percent int) error {
	if err := setBrightnessDXVA2(percent); err == nil {
		return nil
	} else if wmiErr := withCOM(func() error { return setBrightnessWMI(percent) }); wmiErr != nil {
		return fmt.Errorf("set brightness: DXVA2: %v; WMI: %w", err, wmiErr)
	}
	return nil
}

func setBrightnessDXVA2(percent int) error {
	monitor, release, err := primaryPhysicalMonitor()
	if err != nil {
		return err
	}
	defer release()
	var minimum, current, maximum uint32
	result, _, callErr := getMonitorBrightness.Call(
		uintptr(monitor), uintptr(unsafe.Pointer(&minimum)), uintptr(unsafe.Pointer(&current)), uintptr(unsafe.Pointer(&maximum)),
	)
	if result == 0 {
		return callError("GetMonitorBrightness", callErr)
	}
	if maximum <= minimum {
		return errors.New("monitor brightness range is invalid")
	}
	target := minimum + uint32((uint64(maximum-minimum)*uint64(percent)+50)/100)
	result, _, callErr = setMonitorBrightness.Call(uintptr(monitor), uintptr(target))
	if result == 0 {
		return callError("SetMonitorBrightness", callErr)
	}
	return nil
}

func primaryPhysicalMonitor() (syscall.Handle, func(), error) {
	window, _, callErr := getDesktopWindow.Call()
	if window == 0 {
		return 0, nil, callError("GetDesktopWindow", callErr)
	}
	monitor, _, callErr := monitorFromWindow.Call(window, 1)
	if monitor == 0 {
		return 0, nil, callError("MonitorFromWindow", callErr)
	}
	var count uint32
	result, _, callErr := physicalMonitorCount.Call(monitor, uintptr(unsafe.Pointer(&count)))
	if result == 0 {
		return 0, nil, callError("GetNumberOfPhysicalMonitorsFromHMONITOR", callErr)
	}
	if count == 0 {
		return 0, nil, errors.New("no physical monitors were found")
	}
	values := make([]physicalMonitor, count)
	result, _, callErr = physicalMonitors.Call(monitor, uintptr(count), uintptr(unsafe.Pointer(&values[0])))
	if result == 0 {
		return 0, nil, callError("GetPhysicalMonitorsFromHMONITOR", callErr)
	}
	selected := values[0].Handle
	release := func() {
		for _, value := range values {
			if value.Handle != 0 {
				_, _, _ = destroyMonitor.Call(uintptr(value.Handle))
			}
		}
	}
	if selected == 0 {
		release()
		return 0, nil, errors.New("primary physical monitor handle is zero")
	}
	return selected, release, nil
}

var errWMIFound = errors.New("WMI value found")

func readBrightnessWMI() (int, bool, error) {
	service, err := connectWMI()
	if err != nil {
		return 0, false, err
	}
	defer service.Release()
	setValue, err := oleutil.CallMethod(service, "ExecQuery", "SELECT CurrentBrightness FROM WmiMonitorBrightness WHERE Active=TRUE")
	if err != nil {
		return 0, false, err
	}
	if setValue == nil {
		return 0, false, errors.New("WMI brightness query returned nil")
	}
	defer setValue.Clear()
	set := setValue.ToIDispatch()
	if set == nil {
		return 0, false, errors.New("WMI brightness query returned nil")
	}
	var result int
	found := false
	err = oleutil.ForEach(set, func(value *ole.VARIANT) error {
		defer value.Clear()
		item := value.ToIDispatch()
		if item == nil {
			return nil
		}
		current, err := oleutil.GetProperty(item, "CurrentBrightness")
		if err != nil || current == nil {
			return err
		}
		defer current.Clear()
		number, ok := numericVariant(current.Value())
		if ok {
			result, found = max(0, min(100, number)), true
			return errWMIFound
		}
		return nil
	})
	if err != nil && !errors.Is(err, errWMIFound) {
		return 0, false, err
	}
	return result, found, nil
}

func setBrightnessWMI(percent int) error {
	service, err := connectWMI()
	if err != nil {
		return err
	}
	defer service.Release()
	setValue, err := oleutil.CallMethod(service, "ExecQuery", "SELECT * FROM WmiMonitorBrightnessMethods")
	if err != nil {
		return err
	}
	if setValue == nil {
		return errors.New("WMI brightness methods query returned nil")
	}
	defer setValue.Clear()
	set := setValue.ToIDispatch()
	if set == nil {
		return errors.New("WMI brightness methods query returned nil")
	}
	called := false
	var lastErr error
	err = oleutil.ForEach(set, func(value *ole.VARIANT) error {
		defer value.Clear()
		item := value.ToIDispatch()
		if item == nil {
			return nil
		}
		attempts := [][2]any{{uint32(0), uint8(percent)}, {int32(0), int32(percent)}, {int(0), int(percent)}}
		for _, arguments := range attempts {
			response, err := oleutil.CallMethod(item, "WmiSetBrightness", arguments[0], arguments[1])
			if response != nil {
				_ = response.Clear()
			}
			if err == nil {
				called = true
				return errWMIFound
			}
			lastErr = err
		}
		return nil
	})
	if err != nil && !errors.Is(err, errWMIFound) {
		return err
	}
	if !called {
		if lastErr != nil {
			return lastErr
		}
		return errors.New("WMI brightness methods were not found")
	}
	return nil
}

func connectWMI() (*ole.IDispatch, error) {
	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return nil, err
	}
	defer unknown.Release()
	locator, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}
	defer locator.Release()
	serviceValue, err := oleutil.CallMethod(locator, "ConnectServer", nil, "root\\wmi")
	if err != nil {
		serviceValue, err = oleutil.CallMethod(locator, "ConnectServer", ".", "root\\wmi")
		if err != nil {
			return nil, err
		}
	}
	if serviceValue == nil {
		return nil, errors.New("WMI service is nil")
	}
	service := serviceValue.ToIDispatch()
	if service == nil {
		_ = serviceValue.Clear()
		return nil, errors.New("WMI service is nil")
	}
	service.AddRef()
	_ = serviceValue.Clear()
	return service, nil
}

func withCOM(operation func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := ole.CoInitialize(0)
	if err != nil {
		value, ok := err.(*ole.OleError)
		if !ok || value.Code() != 1 {
			return err
		}
	}
	defer ole.CoUninitialize()
	return operation()
}

func numericVariant(value any) (int, bool) {
	switch number := value.(type) {
	case int8:
		return int(number), true
	case uint8:
		return int(number), true
	case int16:
		return int(number), true
	case uint16:
		return int(number), true
	case int32:
		return int(number), true
	case uint32:
		return int(number), true
	case int64:
		return int(number), true
	case uint64:
		if number <= uint64(^uint(0)>>1) {
			return int(number), true
		}
	case int:
		return number, true
	case uint:
		if number <= ^uint(0)>>1 {
			return int(number), true
		}
	}
	return 0, false
}

func callError(operation string, err error) error {
	if err != nil && !errors.Is(err, syscall.Errno(0)) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s failed", operation)
}
