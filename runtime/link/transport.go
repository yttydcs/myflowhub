package link

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrClosed    = errors.New("link is closed")
	ErrQueueFull = errors.New("link queue is full")
	ErrIdle      = errors.New("link idle timeout")
)

type Endpoint string

func (e Endpoint) Validate() error {
	if strings.TrimSpace(string(e)) == "" {
		return errors.New("endpoint is required")
	}
	return nil
}

type Pipe interface {
	io.Reader
	io.Writer
	io.Closer
}

type Listener interface {
	Accept(context.Context) (Pipe, error)
	Addr() Endpoint
	Close() error
}

type Driver interface {
	Dial(context.Context, Endpoint) (Pipe, error)
	Listen(context.Context, Endpoint) (Listener, error)
}

func ValidateDriver(driver Driver) error {
	if driver == nil {
		return fmt.Errorf("transport driver is required")
	}
	return nil
}
