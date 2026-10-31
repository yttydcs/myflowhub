//go:build windows

package main

import (
	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	metricswindows "github.com/yttydcs/myflowhub/apps/nodes/metrics/platform/windows"
)

func nativePlatform() (metrics.Collector, metrics.Actuator, string, error) {
	platform := metricswindows.New()
	return platform, platform, "windows", nil
}
