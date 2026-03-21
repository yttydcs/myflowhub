# 2026-03-01 · Workspace · go.work 加入 MetricsNode modules（修复 wails dev 报错）

## 变更背景 / 目标

在 workspace 根目录存在 `go.work` 的情况下，Go 默认启用 workspace mode。

`MyFlowHub-MetricsNode/windows` 是一个独立 Go module（用于 Wails App）。运行 `wails dev` 时会执行 `go mod tidy`，但当该 module 未被列入 workspace `go.work` 的 `use (...)` 时，Go 会直接报错并退出：

- `current directory is contained in a module that is not one of the workspace modules listed in go.work`

目标：更新 workspace `go.work`，将 MetricsNode 相关 modules 加入 `use (...)`，使 MetricsNode Windows 端可在 workspace mode 下正常 `wails dev`。

## 具体变更内容

### 修改

- `go.work`
  - 在 `use (...)` 中加入：
    - `./repo/MyFlowHub-MetricsNode`
    - `./repo/MyFlowHub-MetricsNode/windows`

## plan.md 任务映射

- `scripts/plan.md`
  - S6：更新 workspace `go.work`：加入 MetricsNode modules
  - S7：更新归档（本文件）

## 测试与验证方式 / 结果

在 workspace mode 下验证（不修改任何 go.mod）：

```powershell
cd d:\project\MyFlowHub3\repo\MyFlowHub-MetricsNode\windows
go list -m
```

预期：不再出现 go.work module 报错。

手工验证 MetricsNode 启动：

```powershell
cd d:\project\MyFlowHub3\repo\MyFlowHub-MetricsNode\windows
wails dev
```

## 潜在影响与回滚方案

潜在影响：
- `go.work` 的 `use (...)` 列表增加两个本地 module，workspace 下 `go list -m` 的可见 module 集合会扩大（属于预期行为）。

回滚：
- 从 `go.work` 的 `use (...)` 中移除上述两行；如需清理 workspace 影响，可在 MetricsNode 启动时使用环境变量绕过：
  - PowerShell：`$env:GOWORK='off'; wails dev`

