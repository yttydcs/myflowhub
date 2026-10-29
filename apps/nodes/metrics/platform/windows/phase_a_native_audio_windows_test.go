//go:build windows

package windows

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type nativeAudioObservation struct {
	UnixNano      int64   `json:"unix_nano"`
	Level         float32 `json:"level"`
	VolumePercent float64 `json:"volume_percent"`
	Muted         bool    `json:"muted"`
}

// The observer is a separate process and reads CoreAudio directly. It never
// writes except in explicitly selected restoration/external-change fixture modes.
func TestPhaseANativeAudioObserver(t *testing.T) {
	root := os.Getenv("MFH_PHASE_A_NATIVE_ROOT")
	if root == "" {
		t.Skip("set MFH_PHASE_A_NATIVE_ROOT and MFH_PHASE_A_AUDIO_MODE for native audio evidence")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	temporary, err := filepath.Abs(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(temporary, root)
	if err != nil || strings.HasPrefix(relative, "..") || !strings.HasPrefix(filepath.Base(root), "mfh-phase-a-native-") {
		t.Fatal("audio observer requires a dedicated native fixture directory under TEMP")
	}
	originalPath := filepath.Join(root, "audio-original.json")
	if os.Getenv("MFH_PHASE_A_AUDIO_MODE") == "restore" {
		raw, err := os.ReadFile(originalPath)
		if err != nil {
			t.Fatal(err)
		}
		var original nativeAudioObservation
		if err := json.Unmarshal(raw, &original); err != nil {
			t.Fatal(err)
		}
		if original.Level < 0 || original.Level > 1 || original.UnixNano <= 0 {
			t.Fatal("original audio observation is invalid")
		}
		if err := restoreSmokeAudio(smokeAudioState{level: original.Level, muted: original.Muted}); err != nil {
			t.Fatal(err)
		}
		actual, err := readSmokeAudio()
		if err != nil || actual.level != original.Level || actual.muted != original.Muted {
			t.Fatalf("original audio did not restore exactly: got=%+v error=%v", actual, err)
		}
		return
	}
	mode := os.Getenv("MFH_PHASE_A_AUDIO_MODE")
	if mode == "external20" {
		raw, err := os.ReadFile(originalPath)
		if err != nil {
			t.Fatal("external change requires the saved restoration baseline: ", err)
		}
		var original nativeAudioObservation
		if err := json.Unmarshal(raw, &original); err != nil || original.UnixNano <= 0 || original.Level < 0 || original.Level > 1 {
			t.Fatal("invalid restoration baseline")
		}
		before, err := readSmokeAudio()
		if err != nil {
			t.Fatal(err)
		}
		started := time.Now()
		if err := restoreSmokeAudio(smokeAudioState{level: 0.2, muted: before.muted}); err != nil {
			t.Fatal(err)
		}
		actual, err := readSmokeAudio()
		if err != nil {
			t.Fatal(err)
		}
		if actual.level < 0.19 || actual.level > 0.21 || actual.muted != before.muted {
			t.Fatalf("external change not confirmed: %+v", actual)
		}
		file, err := os.CreateTemp(root, "audio-external20-*.json")
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := json.NewEncoder(file).Encode(map[string]any{"started_unix_nano": started.UnixNano(), "confirmed_unix_nano": time.Now().UnixNano(), "level": actual.level, "muted": actual.muted})
		if err := errors.Join(encodeErr, file.Close()); err != nil {
			t.Fatal(err)
		}
		return
	}
	if mode != "observe" && mode != "resume" {
		t.Fatal("audio observer mode must be observe, resume, restore or external20")
	}
	observe := func() nativeAudioObservation {
		t.Helper()
		value, err := readSmokeAudio()
		if err != nil {
			t.Fatal(err)
		}
		return nativeAudioObservation{UnixNano: time.Now().UnixNano(), Level: value.level,
			VolumePercent: float64(value.level) * 100, Muted: value.muted}
	}
	if mode == "observe" {
		original := observe()
		raw, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(originalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(raw); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	} else if _, err := os.Stat(originalPath); err != nil {
		t.Fatal("resume requires the preserved original audio observation: ", err)
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if mode == "resume" {
		flags = os.O_WRONLY | os.O_APPEND
	}
	output, err := os.OpenFile(filepath.Join(root, "audio-observations.jsonl"), flags, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	encoder := json.NewEncoder(output)
	timer := time.NewTimer(10 * time.Minute)
	defer timer.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := encoder.Encode(observe()); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(root, "audio-observer.stop")); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			return
		}
	}
}
