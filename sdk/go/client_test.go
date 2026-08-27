package sdk

import "testing"

func TestClientRequiresRuntime(t *testing.T) {
	if _, err := NewClient(nil); err == nil {
		t.Fatal("nil runtime was accepted")
	}
}
