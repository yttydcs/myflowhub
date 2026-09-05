# ClipboardNode

这里是 ClipboardNode 的 canonical product source。核心把剪贴板建模为 Variable、Stream 和 Command；peer 同步使用 SDK durable subscription，物理传输由运行时 `link.Driver` 提供。

## Layout

- 根 package：schema、controller、history、dedupe、runtime 和 peer subscriptions。
- `bridge`：Windows Flutter 后端使用的 512 KiB 有界 NDJSON bridge。
- `platform/windows`：Win32 Unicode clipboard adapter。
- `app`：Windows Flutter shell 与 Web UI 预览。
- NodeHost：统一管理身份、Node、父连接；同步引擎只使用 attached SDK。
- `../../../../cmd/mfh-clipboard`：独立 Windows 进程和 bridge 入口。

稳定行为、权限、隐私边界和构建命令见 `docs/features/clipboard-node.md`。
