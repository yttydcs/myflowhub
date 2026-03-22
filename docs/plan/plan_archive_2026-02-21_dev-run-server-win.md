# Plan - 本地一键启动 Server + Win（验证脚本）

> 位置：`d:\project\MyFlowHub3\scripts\`
>
> 用户确认：
> - 脚本放在 workspace 根目录（不进任何 repo 的 git）：✅（选项 1B）
> - 默认启动方式：`go run ./cmd/hub_server` + `wails dev`：✅（选项 2A）
> - 日志/窗口：分别开两个窗口（Server 一个、Win 一个）：✅（选项 3A）

---

## 1) 目标与当前状态

### 目标
提供一个 PowerShell 脚本，能够在本机快速拉起：
- `repo/MyFlowHub-Server` 的 `hub_server`（默认监听 `:9000`）
- `repo/MyFlowHub-Win`（`wails dev`）

并输出清晰的“冒烟验证步骤”，便于你验证：
1) Win 能连接到 Server（地址 `127.0.0.1:9000`）
2) Presets → Node Echo 可发送并得到成功提示
3) Flow Builder 可进行基础操作（可选）

### 当前状态
- workspace 根目录尚无统一启动脚本（将新增）。

---

## 2) 任务清单（Checklist）

### S1 - 定义脚本接口与默认行为
**目标**
- 明确脚本参数（端口、node-id、是否等待端口就绪、是否 `GOWORK=off`、是否跳过某一端）。

**验收**
- `Get-Help .\scripts\run-dev.ps1 -Detailed` 可看到参数说明与示例。

**回滚**
- 删除脚本文件即可。

---

### S2 - 实现 `run-dev.ps1`
**目标**
- 在两个新窗口中分别启动：
  - Server：`go run ./cmd/hub_server`
  - Win：`wails dev`
- Server 通过环境变量配置（与 `cmd/hub_server/main.go` 对齐）：
  - `HUB_ADDR`、`HUB_NODE_ID`、可选 `HUB_PARENT_ADDR`
- 可选 `-WaitServer`：等待端口可连接后再启动 Win（避免 Win 先启动但 Server 还没 ready）。
- 统一设置 `GOTMPDIR`（默认 `d:\project\MyFlowHub3\.tmp\gotmp`）以规避临时目录权限/性能问题。

**涉及文件**
- `scripts/run-dev.ps1`

**验收**
- 运行 `.\scripts\run-dev.ps1` 后：
  - 弹出 2 个 PowerShell 窗口（各自持续输出日志）。
  - Server 窗口出现 `hub server started` 日志（或等价启动日志）。
  - Win 窗口进入 `wails dev` 运行状态并弹出桌面窗口。

**测试点**
- `.\scripts\run-dev.ps1 -ServerAddr ':9001' -ServerNodeId 2 -WaitServer` 可正常启动。
- 删除键/端口占用等异常能给出可理解提示。

**回滚**
- 删除脚本文件即可。

---

### S3 - 归档变更（docs/change）
**目标**
- 在 `docs/change/` 记录脚本用途、参数、冒烟步骤与回滚方式。

**涉及文件**
- `docs/change/2026-02-21_dev-run-server-win.md`

**验收**
- 文档可脱离对话直接执行并完成验证。

---

## 3) 风险与注意事项
- 若你使用 Windows Terminal，`Start-Process pwsh` 默认会打开独立控制台窗口（不是同一个 Terminal tab）。这符合“两个窗口”的要求。
- `wails dev` 依赖 Node/npm；脚本会做基本存在性检查，但不会自动安装。
- 该脚本放在 workspace 根目录，不进入任何 repo 的 git；如需团队共享，后续可迁移到 `repo/MyFlowHub-Win/scripts/` 并另起 workflow。

