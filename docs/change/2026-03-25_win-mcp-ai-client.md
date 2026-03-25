# Win MCP AI Client

## 变更背景 / 目标

- `MyFlowHub-Win` 只有 Wails GUI 入口，AI host 无法直接以 MCP `stdio` 方式复用现有 `session/auth/management/varstore` 能力。
- 用户已确认本轮目标不是驱动现有 Win 界面，而是新增一个独立、无界面的 MyFlowHub 客户端，并作为独立节点接入 Hub。
- 本轮目标是在不破坏 GUI 行为的前提下，为 Win 仓交付首版 MCP CLI，并保证本地配置和 node keys 与 GUI 隔离。

## 具体变更内容

- 新增 `cmd/myflowhub-mcp/main.go`
  - 提供独立 CLI 入口
  - 支持 `--endpoint`、`--config-dir`、`--device-id`、`--display-name`、`--default-target`、`--timeout`、`--allow-write`
- 新增 `internal/mcpapp/runtime.go`
  - 组装无界面 runtime
  - 创建独立 store / logs / session / auth / management / varpool
  - 持久化 `mcp.*` 配置
  - 将日志桥接到 `stderr`
- 新增 `internal/mcp/server.go`
  - 实现最小 MCP `stdio` server
  - 支持 `initialize`、`tools/list`、`tools/call`、`ping`、`shutdown`、`exit`
- 新增 `internal/mcp/tools.go`
  - 暴露首版工具：
    - `myflowhub_session_status`
    - `myflowhub_session_connect`
    - `myflowhub_session_disconnect`
    - `myflowhub_auth_register`
    - `myflowhub_auth_login`
    - `myflowhub_management_list_nodes`
    - `myflowhub_management_node_info`
    - `myflowhub_varstore_list`
    - `myflowhub_varstore_get`
    - `myflowhub_varstore_set`
    - `myflowhub_varstore_revoke`
  - 集中实现参数校验、默认 `source_id/target_id` 回退和 write gate
- 修改 `internal/storage/store.go`
  - 新增 `NewStoreWithBaseDir()`
  - 保留 GUI `NewStore()` 默认路径行为不变
- 修改 `internal/services/auth/service.go` 与 `internal/services/management/service.go`
  - 本地显示名优先读取 `mcp.display_name`
- 更新 `README.md`
  - 增加 MCP CLI build / run / host config 说明

## Requirements impact: updated

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\mcp-client.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\mcp-client.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `DOCS-1`
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
- `WIN-MCP-1`
  - `internal/storage/store.go`
  - `internal/storage/store_test.go`
  - `internal/mcpapp/runtime.go`
- `WIN-MCP-2`
  - `internal/mcp/server.go`
  - `cmd/myflowhub-mcp/main.go`
- `WIN-MCP-3`
  - `internal/mcp/tools.go`
  - `internal/services/auth/service.go`
  - `internal/services/management/service.go`
- `WIN-MCP-4`
  - `internal/mcp/server_test.go`
  - `internal/mcp/tools_test.go`
  - `internal/services/auth/service_test.go`
  - `internal/services/management/service_test.go`
  - `README.md`
- `ARCHIVE-1`
  - `docs/change/README.md`
  - `docs/change/2026-03-25_win-mcp-ai-client.md`

## 经验 / 教训摘要

- 对现有 GUI 客户端补充 headless 能力时，优先复用业务 service，但必须把 bootstrap、配置目录和状态持久化独立出来，否则后续会被 UI 语义反向污染。
- MCP tool 层应只负责参数和默认值编排，不要把协议和状态决策散落到 CLI 或具体 service 中。
- `stdout` / `stderr` 的职责必须从一开始就分清，否则 MCP 宿主和调试日志很容易互相污染。

## 可复用排查线索

- 症状：
  - MCP host 连接后没有返回 `initialize` / `tools/list`
  - GUI 和 MCP 共享了同一份 `settings.json` 或 node keys
  - `varstore_set` / `varstore_revoke` 在默认配置下仍然可写
- 触发条件：
  - `stdout` 被普通日志污染
  - headless runtime 没有使用独立 `config-dir`
  - tool 层绕过了 `allow_write` gate
- 关键词：
  - `myflowhub-mcp`
  - `mcp.display_name`
  - `NewStoreWithBaseDir`
  - `allow_write`
  - `tools/list`
- 快速检查：
  - 直接执行 `myflowhub-mcp`，喂 `initialize` / `tools/list` 看是否返回 JSON-RPC
  - 查看 `session_status` 返回的 `config.base_dir`
  - 检查 `settings.json` 是否落在独立 `config-dir`
  - 在未开启 `--allow-write` 时调用 `myflowhub_varstore_set`

## 关键设计决策与权衡

- 决策：在 `MyFlowHub-Win` 仓内新增 `cmd/myflowhub-mcp`
  - 原因：可以直接复用现有 `session/auth/management/varpool`，首版改动最小。
- 决策：MCP runtime 使用独立 `mcp.*` namespace 和独立默认配置目录
  - 原因：避免与 GUI 的 `home.*` / `app.*` / node keys 互相污染。
- 决策：写工具默认关闭
  - 原因：AI 默认应只读，写行为必须由显式启动参数开启。
- 决策：MCP `stdio` 采用最小 server 自实现
  - 原因：首版范围聚焦于工具暴露和独立节点接入，不扩展到额外 SDK 或第三方依赖。

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; go test ./... -count=1`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
  - 结果：通过
- `$env:GOWORK='off'; go build -o (Join-Path $env:TEMP 'myflowhub-mcp.exe') ./cmd/myflowhub-mcp`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
  - 结果：通过
- 进程级 smoke
  - 直接拉起 `myflowhub-mcp`，发送 `initialize`、`tools/list`
  - 结果：返回正常 JSON-RPC 响应和完整 tool 列表

## 潜在影响与回滚方案

- 潜在影响：
  - 仓内新增了一个长期维护的 headless 入口，后续若 service 接口调整，需要同步关注 MCP runtime 的兼容性。
  - `mcp.*` 本地状态会在独立 `config-dir` 下长期保留；如果用户换目录，默认快照和 node keys 也会一起切换。
- 回滚方案：
  - 回退 `cmd/myflowhub-mcp`、`internal/mcpapp`、`internal/mcp` 以及 `internal/storage/store.go`
  - 如需彻底清理本地状态，删除 MCP 使用的独立 `config-dir`

## 子Agent执行轨迹

- none
