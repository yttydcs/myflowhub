# Win MCP Shared HTTP Server

## 变更背景 / 目标

- 用户会同时开启多个 Codex 会话；原 `stdio` MCP 模式会让每个 Codex 会话各自启动一个 `myflowhub-mcp` 进程，并各自直连 Hub。
- 多进程共用同一 `config-dir/device-id` 会带来本地 settings/node keys 写入竞争，以及远端同身份在线状态互相覆盖风险。
- 本轮目标是在保留 `stdio` 兼容入口的前提下，新增本机 HTTP MCP Server 模式，让多个 Codex 会话连接同一个 MCP endpoint，共享一个进程内 runtime、Hub 连接、登录态和配置目录。

## 具体变更内容

- 更新 `docs/requirements/mcp-client.md`
  - 增加多 Codex 并行场景下的 HTTP MCP Server 需求。
  - 明确 HTTP 模式共享一个 `mcpapp.Runtime`，默认只监听 localhost。
- 更新 `docs/specs/mcp-client.md`
  - 增加 `stdio` / `http` 双 transport 契约。
  - 约定 HTTP 默认 `127.0.0.1:17688`、默认 path `/mcp`、非 loopback listen 拒绝、非 loopback Origin 拒绝。
  - 约定 Codex HTTP 配置为 `type = "http"` / `url = "http://127.0.0.1:<port>/mcp"`。
- 修改 `internal/mcp/server.go`
  - 将单条 JSON-RPC 请求处理抽为 `HandleRequest`，供 stdio 与 HTTP transport 共用。
  - 保留原 stdio line-loop 行为。
- 修改 `internal/mcp/tools.go`
  - `myflowhub_session_status` 新增 `mcp_server` 运行摘要，直接返回当前 MCP transport、listen address、path 和 URL。
- 新增 `internal/mcp/http_server.go`
  - 实现本机 HTTP MCP endpoint。
  - 支持 POST JSON-RPC request/notification。
  - 对 GET 返回 405，对非本地 Origin 返回 403。
  - 默认拒绝 `0.0.0.0:port` 和 `:port` 这类非 loopback listen。
- 更新 `cmd/myflowhub-mcp/main.go`
  - 新增 `--transport stdio|http`、`--listen`、`--mcp-path`。
  - HTTP 模式下同一进程只创建一个 runtime 和一个 MCP server 实例。
- 更新脚本
  - `scripts/start-myflowhub-mcp.ps1` 吸收 Windows PowerShell 5.1 兼容修复。
  - `scripts/install-codex-myflowhub-mcp.ps1` 新增 `-Transport http`、`-Listen`、`-McpPath`、`-Url`，可生成 HTTP MCP 配置。
- 更新 `README.md`
  - 增加 HTTP shared server 启动和 Codex 安装示例。
- 新增测试
  - `internal/mcp/http_server_test.go` 覆盖 POST、notification、GET 405、Origin 403、localhost Origin、非 loopback listen 拒绝、同 handler 复用。
  - `internal/mcp/server_test.go` 增加 transport-independent `HandleRequest` 测试。

## Requirements impact: updated

Related requirements:

- `docs/requirements/mcp-client.md`

## Specs impact: updated

Related specs:

- `docs/specs/mcp-client.md`
- MCP official transport spec: `https://modelcontextprotocol.io/specification/2025-06-18/basic/transports`

## Lessons impact: updated

Related lessons:

- `docs/lessons/powershell-utf8-nobom-parse.md`

## 对应 `plan.md` 任务映射

- `DOCS-1`
  - `docs/requirements/mcp-client.md`
- `DOCS-2`
  - `docs/specs/mcp-client.md`
- `MCP-1`
  - `internal/mcp/server.go`
  - `internal/mcp/server_test.go`
- `MCP-2`
  - `internal/mcp/tools.go`
  - `internal/mcp/tools_test.go`
  - `internal/mcp/http_server.go`
  - `internal/mcp/http_server_test.go`
  - `cmd/myflowhub-mcp/main.go`
- `MCP-3`
  - `scripts/start-myflowhub-mcp.ps1`
  - `scripts/install-codex-myflowhub-mcp.ps1`
  - `README.md`
- `MCP-4`
  - validation and code review
- `ARCHIVE-1`
  - `docs/change/2026-05-28_win-mcp-shared-http-server.md`
  - `docs/lessons/powershell-utf8-nobom-parse.md`
  - `docs/change/README.md`
  - `docs/lessons/README.md`

## 经验 / 教训摘要

- 多 Codex 并行时，`stdio` MCP 不适合作为共享状态入口，因为每个 MCP client 都会拉起自己的子进程。
- 本机共享状态应优先落在 HTTP MCP Server 的进程边界里，而不是让多个 stdio 进程共享同一个 config-dir。
- Windows PowerShell 5.1 读取 UTF-8 无 BOM 脚本时可能误解中文注释并报错；外部 host 拉起的 PowerShell 脚本应优先保持 ASCII 注释，或显式使用带 BOM/兼容编码。

## 可复用排查线索

- 症状：
  - 多个 Codex 会话同时启动多个 `myflowhub-mcp.exe`。
  - `config.toml` 指向 `stdio` command/args，多个会话各自拉起子进程。
  - `powershell.exe -File scripts/*.ps1` 报 `Unexpected token '}'`，但 PowerShell 7 解析正常。
- 触发条件：
  - 多 Codex 并行使用同一个 stdio MCP server 配置。
  - Windows PowerShell 5.1 执行 UTF-8 无 BOM 且含中文注释的 `.ps1`。
- 关键词：
  - `myflowhub-mcp`
  - `--transport http`
  - `type = "http"`
  - `Origin`
  - `Unexpected token '}'`
  - `UTF-8 no BOM`
- 快速检查：
  - `Get-Process myflowhub-mcp | Select Id,Path,StartTime`
  - `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 --version`
  - `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Transport http -WhatIf`
  - POST `initialize` 到 `http://127.0.0.1:17688/mcp`

## 关键设计决策与权衡

- 决策：保留 `stdio`，新增 HTTP transport。
  - 原因：不破坏既有 MCP host 兼容；多 Codex 使用切到 HTTP 共享入口。
- 决策：HTTP 默认只允许 loopback listen，并校验 request Origin。
  - 原因：本机 MCP 带有写工具和本地登录态，不能默认暴露给网页或局域网。
- 决策：本轮不创建独立 `MyFlowHub-MCP` 仓。
  - 原因：当前 MCP runtime 仍依赖 Win internal services；先收敛运行边界，后续再迁仓风险更低。
- 决策：暂不实现 SSE/GET。
  - 原因：当前 Codex 主要需要 request/response tool call；GET 未实现时明确返回 405，后续如 host 需要 SSE 再补。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp -count=1`
  - 结果：通过。
- `$env:GOWORK='off'; go build -o $env:TEMP\myflowhub-mcp-shared-server-test.exe ./cmd/myflowhub-mcp`
  - 结果：通过。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 --version`
  - 结果：通过。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Transport http -Listen 127.0.0.1:17688 -McpPath /mcp -WhatIf`
  - 结果：通过，输出 HTTP MCP config。
- HTTP 进程级 smoke：
  - 启动 `start-myflowhub-mcp.ps1 --transport http --listen 127.0.0.1:<temp> --mcp-path /mcp`
  - POST `initialize`
  - POST `tools/list`
  - POST `tools/call myflowhub_session_status`
  - 结果：通过，`tools/list` 返回 34 个工具并包含 `myflowhub_topicbus_publish`。
- `git diff --check`
  - 结果：通过。

## 潜在影响与回滚方案

- 潜在影响：
  - HTTP MCP Server 模式需要用户额外启动一个常驻进程；安装脚本只写 Codex URL，不负责系统服务托管。
  - 若 MCP host 严格要求 Streamable HTTP 的 SSE GET 行为，当前 HTTP 模式需要后续补 SSE。
  - `--transport` 非 `http` 值会回退到 `stdio`，避免旧命令失败，但拼写错误不会硬失败。
- 回滚方案：
  - 回退 `cmd/myflowhub-mcp/main.go`、`internal/mcp/server.go`、`internal/mcp/http_server.go`、`internal/mcp/*_test.go`。
  - 回退脚本和 README 修改。
  - 保留或回退 requirements/specs 取决于是否继续保留 HTTP shared server 作为路线。

## 子Agent执行轨迹

- 未派发子Agent。
