//go:build windows

package windows

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
	"golang.org/x/sys/windows"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard     = user32.NewProc("OpenClipboard")
	procCloseClipboard    = user32.NewProc("CloseClipboard")
	procEmptyClipboard    = user32.NewProc("EmptyClipboard")
	procIsFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardData  = user32.NewProc("GetClipboardData")
	procSetClipboardData  = user32.NewProc("SetClipboardData")
	procGlobalAlloc       = kernel32.NewProc("GlobalAlloc")
	procGlobalFree        = kernel32.NewProc("GlobalFree")
	procGlobalLock        = kernel32.NewProc("GlobalLock")
	procGlobalUnlock      = kernel32.NewProc("GlobalUnlock")
	procGlobalSize        = kernel32.NewProc("GlobalSize")
)

type Platform struct {
	done      chan struct{}
	closeOnce sync.Once
}

func New() *Platform { return &Platform{done: make(chan struct{})} }

func (p *Platform) ReadText(ctx context.Context) (string, error) {
	if p == nil {
		return "", errors.New("Windows clipboard adapter is nil")
	}
	if err := p.open(ctx); err != nil {
		return "", err
	}
	defer procCloseClipboard.Call()
	available, _, callErr := procIsFormatAvailable.Call(cfUnicodeText)
	if available == 0 {
		return "", windowsCallError("check Unicode clipboard format", callErr)
	}
	handle, _, callErr := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", windowsCallError("get Unicode clipboard data", callErr)
	}
	pointer, _, callErr := procGlobalLock.Call(handle)
	if pointer == 0 {
		return "", windowsCallError("lock Unicode clipboard data", callErr)
	}
	defer procGlobalUnlock.Call(handle)
	size, _, callErr := procGlobalSize.Call(handle)
	if size == 0 {
		return "", windowsCallError("size Unicode clipboard data", callErr)
	}
	if size > uintptr((clipboard.MaxTextBytes+1)*2) {
		return "", errors.New("Windows clipboard text exceeds the supported size")
	}
	encoded := make([]byte, int(size))
	process, err := windows.GetCurrentProcess()
	if err != nil {
		return "", fmt.Errorf("open current process for clipboard read: %w", err)
	}
	var read uintptr
	if err := windows.ReadProcessMemory(process, pointer, &encoded[0], size, &read); err != nil {
		return "", fmt.Errorf("read Unicode clipboard memory: %w", err)
	}
	if read != size {
		return "", errors.New("read Unicode clipboard memory returned a short result")
	}
	units := make([]uint16, 0, len(encoded)/2)
	for index := 0; index+1 < len(encoded); index += 2 {
		unit := binary.LittleEndian.Uint16(encoded[index : index+2])
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return windows.UTF16ToString(units), nil
}

func (p *Platform) WriteText(ctx context.Context, text string) error {
	if p == nil {
		return errors.New("Windows clipboard adapter is nil")
	}
	if text == "" || strings.ContainsRune(text, '\x00') {
		return errors.New("Windows clipboard text must be non-empty and contain no NUL")
	}
	encoded, err := windows.UTF16FromString(text)
	if err != nil {
		return errors.New("encode Windows clipboard text as UTF-16")
	}
	byteCount := uintptr(len(encoded) * 2)
	handle, _, callErr := procGlobalAlloc.Call(gmemMoveable, byteCount)
	if handle == 0 {
		return windowsCallError("allocate Windows clipboard memory", callErr)
	}
	owned := true
	defer func() {
		if owned {
			procGlobalFree.Call(handle)
		}
	}()
	pointer, _, callErr := procGlobalLock.Call(handle)
	if pointer == 0 {
		return windowsCallError("lock Windows clipboard memory", callErr)
	}
	bytes := make([]byte, byteCount)
	for index, unit := range encoded {
		binary.LittleEndian.PutUint16(bytes[index*2:index*2+2], unit)
	}
	process, err := windows.GetCurrentProcess()
	if err != nil {
		procGlobalUnlock.Call(handle)
		return fmt.Errorf("open current process for clipboard write: %w", err)
	}
	var written uintptr
	if err := windows.WriteProcessMemory(process, pointer, &bytes[0], byteCount, &written); err != nil {
		procGlobalUnlock.Call(handle)
		return fmt.Errorf("write Unicode clipboard memory: %w", err)
	}
	if written != byteCount {
		procGlobalUnlock.Call(handle)
		return errors.New("write Unicode clipboard memory returned a short result")
	}
	procGlobalUnlock.Call(handle)
	if err := p.open(ctx); err != nil {
		return err
	}
	defer procCloseClipboard.Call()
	if result, _, callErr := procEmptyClipboard.Call(); result == 0 {
		return windowsCallError("empty Windows clipboard", callErr)
	}
	if result, _, callErr := procSetClipboardData.Call(cfUnicodeText, handle); result == 0 {
		return windowsCallError("set Windows clipboard text", callErr)
	}
	owned = false
	return nil
}

func (p *Platform) WatchText(ctx context.Context) (<-chan clipboard.TextObservation, <-chan error, error) {
	if ctx == nil {
		return nil, nil, errors.New("Windows clipboard watch context is required")
	}
	if p == nil {
		return nil, nil, errors.New("Windows clipboard adapter is nil")
	}
	events := make(chan clipboard.TextObservation, 1)
	errorsOut := make(chan error, 1)
	initial, _ := p.ReadText(ctx)
	previous := clipboardHash(initial)
	go func() {
		defer close(events)
		defer close(errorsOut)
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				text, err := p.ReadText(ctx)
				if err != nil {
					continue
				}
				hash := clipboardHash(text)
				if text == "" || hash == previous {
					continue
				}
				previous = hash
				select {
				case events <- clipboard.TextObservation{Text: text, ObservedAt: time.Now().UTC()}:
				case <-ctx.Done():
					return
				case <-p.done:
					return
				}
			case <-ctx.Done():
				return
			case <-p.done:
				return
			}
		}
	}()
	return events, errorsOut, nil
}

func (p *Platform) Close() error {
	if p != nil {
		p.closeOnce.Do(func() { close(p.done) })
	}
	return nil
}

func (p *Platform) open(ctx context.Context) error {
	if ctx == nil {
		return errors.New("Windows clipboard context is required")
	}
	for attempt := 0; attempt < 20; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		select {
		case <-p.done:
			return errors.New("Windows clipboard adapter is closed")
		default:
		}
		if result, _, _ := procOpenClipboard.Call(0); result != 0 {
			return nil
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-p.done:
			timer.Stop()
			return errors.New("Windows clipboard adapter is closed")
		}
	}
	return errors.New("Windows clipboard is busy")
}

func clipboardHash(text string) [sha256.Size]byte { return sha256.Sum256([]byte(text)) }

func windowsCallError(operation string, err error) error {
	if err == nil || errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("%s failed", operation)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

var _ clipboard.Adapter = (*Platform)(nil)
