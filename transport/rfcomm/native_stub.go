//go:build !windows && (!linux || android)

package rfcomm

// 本文件承载 Core 框架中与 `native_stub` 相关的通用逻辑。

import (
	"context"
	"net"

	"github.com/yttydcs/myflowhub/runtime/link"
)

func dialNative(ctx context.Context, opts DialOptions) (link.Pipe, net.Addr, net.Addr, error) {
	return nil, nil, nil, ErrUnsupported
}

func listenNative(opts Options) (nativeListener, error) {
	return nil, ErrUnsupported
}
