# MetricsNode vNext

本目录是 MetricsNode 的 canonical product source。资源语义与物理链路分层：指标、配置和通知使用共享 Variable/Stream/Command 模型；产品 runtime 通过 `link.Driver` 接入父节点。更换 TCP、RFCOMM 或其他链路不需要重写指标资源。

## Layout

- `controller.go`：指标 Variables、控制 Commands、持久配置。
- `notifications.go`：父节点通知 Stream 的 durable subscription 和有界 inbox。
- `platform/windows`：Windows collector/actuator；平台写操作只发生在显式 Command 后。
- `windows`：Wails UI，边界只暴露版本化 JSON。
- `cmd/mfh-metrics`：Windows 无 UI 运行入口。

## Build and test

所有 Go 命令从 monorepo 根运行并设置 `GOWORK=off`：

```text
go test ./apps/nodes/metrics/...
go test -race ./apps/nodes/metrics/...
```

Windows 产品在 `apps/nodes/metrics/windows` 运行 `wails generate module` 和 `wails build -clean`。

Android 实现和移动 binding 已移除，重新设计见 [待办](../../../docs/requirements/mobile-embedded-redesign.md)。Wails binary 是可重建产物，不进入源代码。
