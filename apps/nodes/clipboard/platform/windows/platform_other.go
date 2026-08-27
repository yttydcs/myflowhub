//go:build !windows

package windows

import (
	"context"
	"errors"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
)

type Platform struct{}

func New() *Platform { return &Platform{} }

func (*Platform) ReadText(context.Context) (string, error) {
	return "", errors.New("Windows clipboard is unavailable on this platform")
}

func (*Platform) WriteText(context.Context, string) error {
	return errors.New("Windows clipboard is unavailable on this platform")
}

func (*Platform) WatchText(context.Context) (<-chan clipboard.TextObservation, <-chan error, error) {
	return nil, nil, errors.New("Windows clipboard is unavailable on this platform")
}

func (*Platform) Close() error { return nil }

var _ clipboard.Adapter = (*Platform)(nil)
