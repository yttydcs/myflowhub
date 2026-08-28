# 2026-05-28_win-mcp-ensure-running

## 变更背景 / 目标

`myflowhub-mcp` 已支持本机 HTTP shared server，但多 Codex 会话共享状态前仍需要用户手动启动一个常驻 server。本次补一个轻量级 `ensure-running` 启动入口，让启动脚本可以幂等复用已有 HTTP MCP endpoint，未运行时再自动拉起隐藏后台进程。

## 具体变更内容

- 更新 `scripts/start-myflowhub-mcp.ps1`
  - 新增 `-EnsureRunning` / `--ensure-running`。
  - 默认补齐 `--transport http --listen 127.0.0.1:17688 --mcp-path /mcp`。
  - 支持用户通过 `--listen`、`--mcp-path` 覆盖 endpoint。
  - 通过 HTTP POST `initialize` 验证 endpoint 是否为有效 MCP endpoint。
  - endpoint 已 ready 时直接复用并退出 0。
  - endpoint 不可达时用隐藏 PowerShell 后台进程启动 HTTP MCP server，并轮询至 ready。
  - endpoint 可达但不是 MCP 时显式失败，避免端口占用误判。
  - 修复开发态 fallback 中 `go run` stdout 被函数返回值捕获的问题，确保 `--version` 和 stdio 输出正常透传。
- 更新 `scripts/install-codex-myflowhub-mcp.ps1`
  - HTTP 安装预演时输出可直接执行的 `EnsureRunning` 命令。
  - 该命令包含 `config-dir`、`device-id`、`display-name`、可选 `endpoint` 和 `allow-write`。
- 更新 `README.md`
  - 增加 `-EnsureRunning` 使用示例。
  - 明确该模式是轻量级 launcher，不是 Windows service。
- 更新 `docs/requirements/mcp-client.md`
  - 增加 HTTP MCP shared server 的脚本级启动/复用需求。
- 更新 `docs/specs/mcp-client.md`
  - 增加 `-EnsureRunning` 入口契约、探测方式、失败条件和非 daemon 边界。
- 新增 `docs/lessons/powershell-args-automatic-variable.md`
  - 记录 PowerShell `$Args` 自动变量与函数参数名冲突导致 forwarded args 解析失败的问题。
- 更新 `docs/lessons/README.md`
  - 增加该 lesson 的检索入口。

## Requirements impact

updated

- `docs/requirements/mcp-client.md`

## Specs impact

updated

- `docs/specs/mcp-client.md`

## Lessons impact

updated

- `docs/lessons/powershell-args-automatic-variable.md`
- `docs/lessons/README.md`

## Related requirements

- `docs/requirements/mcp-client.md`

## Related specs

- `docs/specs/mcp-client.md`

## Related lessons

- `docs/lessons/powershell-utf8-nobom-parse.md`
- `docs/lessons/powershell-args-automatic-variable.md`

## 对应 plan.md 任务映射

- `MCP-ENSURE-1`
  - `scripts/start-myflowhub-mcp.ps1`
- `MCP-ENSURE-2`
  - `scripts/install-codex-myflowhub-mcp.ps1`
  - `README.md`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `docs/change/2026-05-28_win-mcp-ensure-running.md`
  - `docs/lessons/powershell-args-automatic-variable.md`
  - `docs/lessons/README.md`
- `MCP-ENSURE-3`
  - validation and review

## 经验 / 教训摘要

- 多会话共享 MCP 状态的核心仍是 HTTP endpoint；脚本单例判断应以 endpoint readiness 为准，而不是猜进程名。
- 端口打开不等于 MCP 可用，必须用 JSON-RPC `initialize` 验证。
- PowerShell helper 不要把参数命名为 `$Args`；这是自动变量，容易让 forwarded args 变空。
- `go run` wrapper 不应把子进程 stdout 当作函数返回值捕获，否则会破坏 `stdio` MCP 或 `--version` 输出。

## 可复用排查线索

- 症状：
  - 多次启动 `myflowhub-mcp` 后状态不共享。
  - `-EnsureRunning --listen 127.0.0.1:<port>` 仍启动到 `127.0.0.1:17688`。
  - 端口被占用但脚本没有明确说明是不是 MCP endpoint。
  - `start-myflowhub-mcp.ps1 -PreferSource --version` 退出 0 但没有输出。
- 触发条件：
  - 多 Codex 会话同时需要共享 MCP runtime。
  - PowerShell wrapper 需要解析 `ValueFromRemainingArguments`。
  - 使用 `go run` fallback 包装 stdio MCP。
- 关键词：
  - `EnsureRunning`
  - `--ensure-running`
  - `myflowhub-mcp`
  - `http://127.0.0.1:17688/mcp`
  - `param([string[]]$Args)`
  - `ForwardArgs`
  - `initialize`
- 快速检查：
  - `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 -PreferSource --version`
  - `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 -EnsureRunning --listen 127.0.0.1:17688 --mcp-path /mcp`
  - POST `initialize` 到 `http://127.0.0.1:17688/mcp`
  - `Get-NetTCPConnection -LocalAddress 127.0.0.1 -LocalPort 17688 -State Listen`

## 关键设计决策与权衡

- 决策：先做 launcher 级 `ensure-running`，不做 Windows service。
  - 原因：满足多 Codex 共享 server 的当前痛点，变更面小，用户仍可后续选择服务化。
- 决策：探测 MCP `initialize` 而不是只看端口。
  - 原因：能区分真正可复用的 MCP endpoint 和端口被其他 HTTP 服务占用。
- 决策：不引入 PID/lock 文件。
  - 原因：当前用 endpoint readiness 已足够；锁文件会引入 stale lock 清理和跨用户路径策略。
- 决策：安装脚本仍只写 Codex HTTP 配置，但打印 `EnsureRunning` 命令。
  - 原因：Codex 的 HTTP config 负责连接，server 生命周期仍由用户显式启动或后续服务化管理。

## 测试与验证方式 / 结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 -PreferSource --version`
  - 结果：通过，输出 `dev`。
- `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp -count=1`
  - 结果：通过。
- `$env:GOWORK='off'; go build -o $env:TEMP\myflowhub-mcp-ensure-running-test.exe ./cmd/myflowhub-mcp`
  - 结果：通过。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Transport http -Listen 127.0.0.1:17688 -McpPath /mcp -WhatIf`
  - 结果：通过，输出 HTTP MCP config 和 `EnsureRunning` 命令。
- 临时端口启动 / 复用 smoke：
  - 命令：`start-myflowhub-mcp.ps1 -EnsureRunning --listen 127.0.0.1:17891 --mcp-path /mcp ...`
  - 结果：第一次启动并 ready，第二次复用同一 endpoint。
- `-EnsureRunning --transport stdio`
  - 结果：失败符合预期，提示 ensure 模式要求 HTTP transport。
- 非 MCP HTTP endpoint 占用 `127.0.0.1:17892`
  - 结果：失败符合预期，提示 endpoint 可达但不是有效 MyFlowHub MCP server。

## 潜在影响

- `-EnsureRunning` 会启动隐藏后台 PowerShell 进程承载 HTTP MCP server；用户需要主动停止进程或关闭会话环境。
- 当前仍不是系统服务，重启电脑后不会自动恢复。
- 如果默认端口 `17688` 被其他服务占用，脚本会失败而不是换端口。

## 回滚方案

- 回退 `scripts/start-myflowhub-mcp.ps1` 中 `-EnsureRunning` 相关 helper 和入口。
- 回退 `scripts/install-codex-myflowhub-mcp.ps1` 中的 `EnsureRunning` 输出。
- 回退 README、requirements、specs、change 和 lessons 文档更新。

## 子Agent执行轨迹

- none
