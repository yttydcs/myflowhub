package nodehost

import (
	"context"
	"errors"
	"sync"

	"github.com/yttydcs/myflowhub/runtime/link"
)

type captureDriver struct {
	link.Driver
	listener *managedListener
}

func (d *captureDriver) Listen(ctx context.Context, endpoint link.Endpoint) (link.Listener, error) {
	listener, err := d.Driver.Listen(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	managed := &managedListener{Listener: listener}
	d.listener = managed
	return managed, nil
}

type managedListener struct {
	link.Listener
	once sync.Once
	err  error
}

func (l *managedListener) Close() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() { l.err = l.Listener.Close() })
	return l.err
}

func (h *Host) startListener(config ListenerConfig) (*managedListener, link.Endpoint, error) {
	driver := &captureDriver{Driver: config.Driver}
	endpoint, err := h.node.Listen(driver, config.Endpoint)
	if err != nil {
		if driver.listener != nil {
			_ = driver.listener.Close()
		}
		return nil, "", err
	}
	if driver.listener == nil {
		return nil, "", errors.New("transport driver returned no listener")
	}
	return driver.listener, endpoint, nil
}

func closeListenersReverse(listeners []*managedListener) []error {
	var failures []error
	for index := len(listeners) - 1; index >= 0; index-- {
		if err := listeners[index].Close(); err != nil {
			failures = append(failures, err)
		}
	}
	return failures
}
