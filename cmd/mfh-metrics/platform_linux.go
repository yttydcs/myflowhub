//go:build linux

package main

import (
	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	metricslinux "github.com/yttydcs/myflowhub/apps/nodes/metrics/platform/linux"
)

func nativePlatform() (metrics.Collector, metrics.Actuator, string, error) {
	return metricslinux.New(), nil, "linux", nil
}
