//go:build linux

package main

import "testing"

func TestLinuxCLISelectsReadOnlyCollector(t *testing.T) {
	collector, actuator, platform, err := nativePlatform()
	if err != nil || collector == nil || actuator != nil || platform != "linux" {
		t.Fatalf("unexpected native platform: %v %v %s %v", collector, actuator, platform, err)
	}
}
