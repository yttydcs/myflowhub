//go:build !android

package androidbinding

import (
	"strings"
	"testing"
)

func TestRFCOMMProviderFailsExplicitlyOffAndroid(t *testing.T) {
	if err := SetRFCOMMProvider(nil); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("non-Android RFCOMM did not fail explicitly: %v", err)
	}
}
