# MetricsNode vNext

本目录是 MetricsNode 的 canonical product source。资源语义与物理链路分层：指标、配置和通知使用共享 Variable/Stream/Command 模型；产品 runtime 通过 `link.Driver` 接入父节点。更换 TCP、RFCOMM 或其他链路不需要重写指标资源。

## Layout

- `controller.go`：标准类型 Variables、同资源 write、持久配置。
- `notifications.go`：父节点通知 Stream 的 durable subscription 和有界 inbox。
- `platform/windows`：Windows collector/actuator；平台写操作只发生在标准 write 通过值校验、CAS 和授权后，并回读确认实际值。
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

变量 read/snapshot 使用 `mfh.variable-value.v1`，后续 data 省略 definition。十个指标只保存公共 Variable 状态；Wails 的 `metrics[].state` 直接来自 `Variable.Snapshot().State`。配置是标准对象 Variable，持久化更新继续使用 `metrics/config/update`。

音量、静音、亮度使用原指标资源的 `write`，输入为 `mfh.variable-write.v3`，权限描述为 `metrics.write`；授权仍按资源和 capability 精确判定。旧控制资源和格式已移除，不自动转换策略。采集或外部写失败不会伪造 0/false；未知外部执行结果报错并等待下一次采样，不自动重试。

每指标 I/O 锁协调采集、外部写和配置切换。采集释放 I/O 锁后按原 revision 发布；被较新状态替代的采样直接丢弃。配置切换先等待进行中的 I/O，暂停新的 I/O，在释放状态锁和 I/O 锁后持久化及发布，再恢复采集。handler 只返回确认状态，不重入 Variable 的本地更新入口。

Metrics 前端的平台变量类型由 canonical Go 类型生成：`GOWORK=off go generate ./sdk/bindings`。后端测试检查生成物 freshness；Wails 方法绑定另外运行 `wails generate module`。

真实 Windows 音量/静音写入 smoke（会临时调整 1% 音量和切换静音，标准 write 确认后恢复；失败清理同样恢复原始标量和静音状态）：

```powershell
$env:GOWORK = 'off'
$env:MFH_METRICS_WINDOWS_WRITE_SMOKE = '1'
go test ./apps/nodes/metrics/platform/windows -run '^TestWindowsStandardAudioWriteSmoke$' -count=1 -v
Remove-Item Env:MFH_METRICS_WINDOWS_WRITE_SMOKE
```

该测试创建隔离 Node 和临时配置目录，不连接或重启既有手动环境。默认测试运行会跳过物理写入；亮度真实写入仍需另行联调。

Windows 音频边界使用四字节 Win32 BOOL 读取静音，并将 `SetMute` 的 `S_FALSE` 按成功处理。回归测试覆盖非零/零 BOOL、相邻音量值不受破坏、S_FALSE 与失败 HRESULT；真实 smoke 恢复后要求音量标量误差小于 `0.00001`、静音完全相同。
