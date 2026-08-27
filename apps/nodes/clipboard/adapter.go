package clipboard

import (
	"context"
	"time"
)

type TextObservation struct {
	Text       string
	ObservedAt time.Time
}

type Adapter interface {
	ReadText(context.Context) (string, error)
	WriteText(context.Context, string) error
	WatchText(context.Context) (<-chan TextObservation, <-chan error, error)
	Close() error
}
