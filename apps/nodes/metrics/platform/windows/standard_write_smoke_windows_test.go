//go:build windows

package windows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
)

// TestWindowsStandardAudioWriteSmoke briefly changes the default audio endpoint.
// It is opt-in and always restores the original scalar and mute state, including
// when a standard write fails after the external operation may have completed.
func TestWindowsStandardAudioWriteSmoke(t *testing.T) {
	if os.Getenv("MFH_METRICS_WINDOWS_WRITE_SMOKE") != "1" {
		t.Skip("set MFH_METRICS_WINDOWS_WRITE_SMOKE=1 to test real audio writes and restore")
	}
	original, err := readSmokeAudio()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("original audio scalar=%.9g muted=%t", original.level, original.muted)
	t.Cleanup(func() {
		if err := restoreSmokeAudio(original); err != nil {
			t.Errorf("restore original Windows audio settings: %v", err)
		}
		actual, err := readSmokeAudio()
		if err != nil {
			t.Errorf("verify restored Windows audio settings: %v", err)
			return
		}
		if math.Abs(float64(actual.level-original.level)) > 0.00001 || actual.muted != original.muted {
			t.Errorf("audio restoration differs: got %+v, want %+v", actual, original)
		} else {
			t.Logf("restored original audio within 0.00001 scalar tolerance: scalar=%g muted=%t", actual.level, actual.muted)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	identity, err := auth.GenerateIdentity(10)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	n, err := node.New(ctx, node.Config{Identity: identity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	platform := New()
	controller, err := metrics.Register(metrics.ControllerConfig{Node: n, Store: store, Platform: "windows", Actuator: platform})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	for _, name := range []metrics.Name{metrics.VolumePercent, metrics.VolumeMuted} {
		value, err := platform.Collect(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		if err := controller.Update(name, value, nil); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name metrics.Name, value any) error {
		state, ok := controller.State(name)
		if !ok {
			return errors.New("missing smoke metric")
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		request := protocol.VariableWriteV3{Version: 3, ExpectedInstanceID: state.InstanceID, ExpectedRevision: state.Revision, Value: raw}
		input, err := protocol.EncodeJSONPayload(&request, protocol.DefaultMaxPayload)
		if err != nil {
			return err
		}
		result, err := n.Operate(ctx, protocol.ResourceID{Owner: n.ID(), Name: metrics.ResourceName(name)}, protocol.CapabilityWrite, protocol.SchemaVariableWriteV3, input)
		if err != nil {
			return err
		}
		var confirmed protocol.VariableValueV1
		if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &confirmed); err != nil {
			return err
		}
		actual, err := platform.Collect(ctx, name)
		if err != nil {
			return err
		}
		actualJSON, err := json.Marshal(actual)
		if err != nil {
			return err
		}
		if confirmed.Status != "ready" || confirmed.Definition == nil || string(confirmed.Value) != string(actualJSON) || string(confirmed.Value) != string(raw) || confirmed.Revision == state.Revision {
			return fmt.Errorf("%s did not confirm requested/physical state: requested=%s actual=%s envelope=%s", name, raw, actualJSON, result.Payload)
		}
		t.Logf("standard %s write confirmed value=%s revision=%s", name, confirmed.Value, confirmed.Revision)
		return nil
	}
	originalPercent := int(math.Round(float64(original.level * 100)))
	targetPercent := originalPercent + 1
	if targetPercent > 100 {
		targetPercent = originalPercent - 1
	}
	if err := write(metrics.VolumePercent, targetPercent); err != nil {
		t.Fatal(err)
	}
	if err := write(metrics.VolumeMuted, !original.muted); err != nil {
		t.Fatal(err)
	}
	// Restore through the standard contract first. Cleanup also restores the exact
	// original hardware scalar, which can contain finer precision than integer %.
	if err := write(metrics.VolumeMuted, original.muted); err != nil {
		t.Fatal(err)
	}
	if err := write(metrics.VolumePercent, originalPercent); err != nil {
		t.Fatal(err)
	}
}

type smokeAudioState struct {
	level float32
	muted bool
}

func readSmokeAudio() (smokeAudioState, error) {
	var state smokeAudioState
	err := withCOM(func() error {
		endpoint, release, err := openDefaultEndpointVolume()
		if err != nil {
			return err
		}
		defer release()
		if err := endpoint.GetMasterVolumeLevelScalar(&state.level); err != nil {
			return err
		}
		state.muted, err = readEndpointMute(endpoint)
		return err
	})
	return state, err
}

func restoreSmokeAudio(state smokeAudioState) error {
	return withCOM(func() error {
		endpoint, release, err := openDefaultEndpointVolume()
		if err != nil {
			return err
		}
		defer release()
		volumeErr := endpoint.SetMasterVolumeLevelScalar(state.level, nil)
		muteErr := setEndpointMute(endpoint, state.muted)
		return errors.Join(volumeErr, muteErr)
	})
}
