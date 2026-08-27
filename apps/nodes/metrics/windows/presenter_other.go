//go:build !windows

package main

import (
	"context"
	"errors"

	"github.com/yttydcs/myflowhub/protocol"
)

func showSystemNotification(context.Context, protocol.NotificationEventV1) error {
	return errors.New("system notification presenter is only available on Windows")
}
