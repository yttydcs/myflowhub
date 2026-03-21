# 2026-03-01 · Workspace · 一键启动脚本新增 MetricsNode（Server + Win + MetricsNode）

## 变更背景 / 目标

本地联调时，除了启动 `MyFlowHub-Server` 与 `MyFlowHub-Win`，还需要同时启动 `MyFlowHub-MetricsNode`（Windows/Wails）来上报电量/音量到 VarStore，以便在 Win 侧订阅并观察实时更新。

目标：扩展 workspace 级别脚本 `scripts/run-dev.ps1`，默认一键启动三端（Server + Win + MetricsNode），同时保留按需跳过的能力。

## 具体变更内容

### 修改

- `scripts/run-dev.ps1`
  - 默认新增启动第三个窗口：`repo/MyFlowHub-MetricsNode/windows` → `wails dev`
  - 新增参数：`-SkipMetricsNode`，用于只启动 Server + Win（保持旧行为）
  - 改进启动前检查：仅在需要启动对应端时检查目录/依赖命令（`go`/`wails`）
  - `-WaitServer` 行为：当需要启动任一客户端（Win 或 MetricsNode）时等待端口就绪（超时仅告警，不中断）

## plan.md 任务映射

- `scripts/plan.md`
  - S4：扩展 `run-dev.ps1`：启动 MetricsNode（本变更）
  - S5：更新归档（本文件）

## 关键设计决策与权衡

1) **继续使用独立窗口输出日志**  
   沿用 `Start-Process pwsh -NoExit`，让 Server / Win / MetricsNode 各自独立输出，便于定位问题与并行观察。

2) **MetricsNode 启动方式选择 `wails dev`**  
   目标是本地开发联调与快速冒烟验证，`wails dev` 能提供最短反馈闭环；打包构建不在本脚本范围内。

3) **通过开关参数保持兼容性**  
   新增 `-SkipMetricsNode` 作为“回退开关”，确保需要时可恢复到旧的两端启动方式。

## 测试与验证方式 / 结果

快速验证（不拉起窗口，仅验证脚本可运行与参数解析）：

```powershell
pwsh -File scripts/run-dev.ps1 -SkipServer -SkipWin -SkipMetricsNode
Get-Help .\scripts\run-dev.ps1 -Detailed
```

手工冒烟（端到端）：

1) 运行：
   ```powershell
   cd d:\project\MyFlowHub3
   .\scripts\run-dev.ps1
   ```
2) 预期弹出 3 个窗口（Server / Win / MetricsNode），并分别进入持续运行状态。
3) MetricsNode：Connect → Register/Login → Start Reporting。
4) Win：VarPool 订阅 MetricsNode 的 `sys_battery_percent/sys_volume_percent/sys_volume_muted`，调节系统音量/静音，观察订阅值自动更新。

## 潜在影响与回滚方案

潜在影响：
- 无（仅修改 workspace 脚本，不影响任何仓库业务逻辑与运行时行为）。

回滚：
- 使用 `-SkipMetricsNode` 跳过 MetricsNode 启动；或回滚 `scripts/run-dev.ps1` 到上一个归档版本即可。

