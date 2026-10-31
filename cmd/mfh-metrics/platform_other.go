//go:build !windows && !linux

package main

import (
	"errors"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
)

func nativePlatform() (metrics.Collector, metrics.Actuator, string, error) {
	return nil, nil, "", errors.New("mfh-metrics supports Windows and Linux")
}
