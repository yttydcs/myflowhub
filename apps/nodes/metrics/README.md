# MetricsNode vNext

本目录是 MetricsNode 的 canonical product source。资源语义与物理链路分层：指标、配置和通知使用共享 Variable/Stream/Command 模型；产品 runtime 通过 `link.Driver` 接入父节点。更换 TCP、RFCOMM 或其他链路不需要重写指标资源。

## Layout

- `controller.go`：指标 Variables、控制 Commands、持久配置。
- `notifications.go`：父节点通知 Stream 的 durable subscription 和有界 inbox。
- `platform/windows`：Windows collector/actuator；平台写操作只发生在显式 Command 后。
- `windows`：Wails UI，边界只暴露版本化 JSON。
- `android/mobile`：gomobile JSON/primitive binding 和有界 platform action queue。
- `android/app`：Compose UI、前台服务、Android collector/actuator 和系统通知呈现。
- `cmd/mfh-metrics`：Windows 无 UI 运行入口。

## Build and test

所有 Go 命令从 monorepo 根运行并设置 `GOWORK=off`：

```text
go test ./apps/nodes/metrics/...
go test -race ./apps/nodes/metrics/...
```

Windows 产品在 `apps/nodes/metrics/windows` 运行 `wails generate module` 和 `wails build -clean`。

Android AAR 从 monorepo 根生成：

```text
gomobile bind -target=android -androidapi=26 -o apps/nodes/metrics/android/app/libs/metricsmobile.aar github.com/yttydcs/myflowhub/apps/nodes/metrics/android/mobile
```

然后在 `apps/nodes/metrics/android` 使用 Gradle wrapper 运行：

```text
gradlew.bat testDebugUnitTest assembleDebug lintDebug
```

`app/libs/*.aar`、Gradle build 和 Wails binary 是可重建产物，不进入源代码。移动工具链版本由根 `go.mod` 的 `golang.org/x/mobile` 固定，避免 `gomobile`/`gobind` 生成器漂移。
