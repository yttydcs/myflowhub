//go:build !windows

package windows

import (
	"context"

	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
)

type Platform struct{}

func New() *Platform { return &Platform{} }

func (*Platform) Collect(context.Context, metrics.Name) (string, error) {
	return "", metrics.ErrUnsupported
}

func (*Platform) Set(context.Context, metrics.Name, string) (metrics.ApplyResult, error) {
	return metrics.ApplyResult{}, metrics.ErrUnsupported
}
