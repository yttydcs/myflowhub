# MyFlowHub MCP Shared Server Plan

## Project Goal

将当前 `myflowhub-mcp` 从单会话 `stdio` MCP client 扩展为可被多个 Codex 会话共享状态的本机 MCP Server。

目标形态：

```text
Codex A/B/C
  -> http://127.0.0.1:<port>/mcp
  -> one myflowhub-mcp process
  -> one Hub connection / config-dir / device identity / auth state
```

保留现有 `stdio` 模式作为兼容入口，但多 Codex 并行使用应走本地 HTTP MCP endpoint。

## Current State

- 当前 MCP 源码位于 `MyFlowHub-Win`：
  - `cmd/myflowhub-mcp/main.go`
  - `internal/mcp/server.go`
  - `internal/mcp/tools.go`
  - `internal/mcpapp/runtime.go`
  - `scripts/start-myflowhub-mcp.ps1`
  - `scripts/install-codex-myflowhub-mcp.ps1`
- 当前实现只支持 `stdio`：
  - Codex 每个会话会启动自己的 `myflowhub-mcp` 子进程。
  - 多个 Codex 会话会共享同一个 `config-dir/device-id` 但各自持有独立 Hub 连接。
  - `Runtime` / `Store` 只有进程内锁，没有跨进程锁。
- 主线 `repo/MyFlowHub-Win` 当前状态：
  - `main` 位于 `b80ba77 feat: add MCP topicbus publish tools`，ahead `origin/main` 1。
  - 主线有一次未提交脚本兼容性修复：`scripts/start-myflowhub-mcp.ps1` 的 Windows PowerShell 5.1 解析修复。本 workflow 必须在 worktree 内重新吸收该修复，最终不要依赖主线脏改。

## Workflow Metadata

- Active stage: `4 Change Archive`
- Control workspace: `D:\project\MyFlowHub3`
- Active repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/mcp-shared-server`
- Base commit: `b80ba77`
- Active worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-shared-server`
- Active plan: `D:\project\MyFlowHub3\worktrees\feat-mcp-shared-server\plan.md`

## Requirements And Specs Impact

使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。

- Requirements impact: `clarify`
  - Existing `docs/requirements/mcp-client.md` only specifies `stdio` MCP client behavior.
  - This workflow must clarify that MCP also supports a shared local server mode for multi-Codex sessions.
- Specs impact: `clarify`
  - Existing `docs/specs/mcp-client.md` only constrains `cmd/myflowhub-mcp` as `stdio`.
  - This workflow must add a technical contract for HTTP MCP transport, local bind/origin rules, session handling, and Codex config.
- Lessons impact: `none` at planning time
  - If implementation exposes reusable troubleshooting knowledge, promote it during Stage 4.

Related requirements:

- `docs/requirements/mcp-client.md`

Related specs:

- `docs/specs/mcp-client.md`
- MCP official transport spec: `https://modelcontextprotocol.io/specification/2025-06-18/basic/transports`

Related lessons:

- none currently known

## Stage 1 Requirements Summary

### Goal

多个 Codex 会话可以通过同一个 MyFlowHub MCP endpoint 使用共享 Hub 连接、登录态、配置目录和节点身份。

### Scope

Must:

- 保留现有 `stdio` MCP 工具集和调用语义。
- 新增本地 HTTP MCP endpoint，用于多 Codex 会话共享状态。
- HTTP endpoint 默认只监听 `127.0.0.1`。
- HTTP 请求必须校验 Origin/Host 或提供本地安全默认值，避免本地服务被网页跨源滥用。
- Codex 安装脚本支持生成 HTTP MCP 配置。
- 启动脚本支持以常驻 HTTP MCP server 方式启动。
- `session_status` 能显示当前 transport / listen address 或足够排查共享入口状态的信息。
- 文档更新 requirements/specs，不能只写在 change 归档。
- 吸收主线未提交的 PowerShell 5.1 启动脚本兼容性修复。

Optional:

- 提供最小 HTTP smoke 脚本或扩展现有 smoke 脚本验证 initialize/tools/list/status。
- 支持可配置 endpoint path，默认 `/mcp`。

Out of scope for this workflow:

- 不直接创建 `repo/MyFlowHub-MCP` 独立仓库。
- 不从 `MyFlowHub-Win/internal` 完整迁移依赖到 SDK/Core。
- 不实现完整 MCP SSE server-to-client 通知流，除非 POST request/response 语义无法满足 Codex。
- 不改变 Hub 协议 wire。
- 不改变 NotifyNode 的 topic 订阅语义。

### Scenarios

- 用户同时开启多个 Codex 会话，每个会话配置同一个 HTTP MCP URL。
- 只有一个 `myflowhub-mcp` 常驻进程连接 Hub。
- 任一 Codex 发送 `myflowhub_topicbus_publish` 后，NotifyNode 收到 `dev.codex.msg` 并弹系统通知。
- 用户仍可用旧 `stdio` 模式做单会话调试或兼容旧 MCP host。

### Acceptance Criteria

- `go test ./internal/mcp... ./internal/mcpapp...` 或等效定向测试通过。
- `go build ./cmd/myflowhub-mcp` 成功。
- `myflowhub-mcp --transport http --listen 127.0.0.1:<port>` 可启动。
- HTTP POST `initialize` 返回 MCP initialize result。
- HTTP POST `tools/list` 返回包含现有工具，尤其是 `myflowhub_topicbus_publish`。
- 多个独立 HTTP client 连同一个 endpoint 时，后端只存在一个 shared `mcpapp.Runtime` 实例。
- `scripts/install-codex-myflowhub-mcp.ps1` 可以生成 HTTP MCP 配置。
- `scripts/start-myflowhub-mcp.ps1 --version` 在 Windows PowerShell 5.1 下通过。

## Stage 2 Architecture Summary

### Overall Approach

采用 MCP 2025-06-18 Streamable HTTP transport 的本地实现作为共享入口。

理由：

- `stdio` 的标准模型是 client 启动 server 子进程，天然不适合多个 Codex 共享状态。
- Streamable HTTP server 是独立进程，可以处理多个 client connection，适合本机共享状态。
- MCP 官方规范要求本地 HTTP server 绑定 localhost 并校验 Origin，这与本需求的安全边界一致。

备选对比：

- 每个 Codex 一个独立 device：实现简单，但节点/权限/审批会膨胀，不适合通知入口。
- stdio proxy + 私有 daemon：可行，但多一层非 MCP 私有协议，维护成本更高。
- 直接新建独立 `MyFlowHub-MCP` 仓：长期合理，但当前 MCP 仍依赖 Win internal services；本轮先抽运行边界，降低一次性迁移风险。

### Module Responsibilities

- `cmd/myflowhub-mcp`
  - 增加 transport/listen/path 参数。
  - 根据 `--transport stdio|http` 启动对应 server。
- `internal/mcp`
  - 保留工具 schema、tool handler 和 JSON-RPC dispatch。
  - 将 stdio line-loop 与 dispatch 解耦，便于 HTTP 复用。
  - 新增 HTTP handler/server，处理 POST request/response。
- `internal/mcpapp`
  - 继续作为唯一 Hub connection 和共享状态持有者。
  - 不引入跨进程共享，HTTP 模式通过单进程常驻避免 config-dir 竞争。
- `scripts/start-myflowhub-mcp.ps1`
  - 继续定位 binary / go run。
  - 支持透传 HTTP 参数。
  - 吸收 Windows PowerShell 5.1 解析修复。
- `scripts/install-codex-myflowhub-mcp.ps1`
  - 新增 HTTP 安装模式，生成 `type = "http"` / `url = ".../mcp"` 配置。
  - 保留 stdio 安装模式。

### Data / Call Flow

```text
Codex HTTP MCP client
  -> POST /mcp initialize/tools/list/tools/call
  -> internal/mcp HTTP handler
  -> shared JSON-RPC dispatcher
  -> tool handler
  -> single mcpapp.Runtime
  -> Hub
```

### Interface Draft

CLI:

```powershell
myflowhub-mcp --transport stdio ...
myflowhub-mcp --transport http --listen 127.0.0.1:17688 --mcp-path /mcp ...
```

Codex config:

```toml
[mcp_servers.myflowhub]
type = "http"
url = "http://127.0.0.1:17688/mcp"
```

### Errors And Safety

- Reject non-local listen addresses by default unless an explicit unsafe flag is introduced.
- Validate Origin for HTTP requests when Origin is present.
- Return HTTP 405 for unsupported methods if no SSE support is implemented.
- Preserve JSON-RPC error semantics for malformed requests.
- Keep Hub write tools behind existing `allow_write` gate.

### Performance And Tests

- Runtime is constructed once per process in HTTP mode.
- Tool calls reuse existing long Hub connection.
- Unit tests cover:
  - shared dispatcher behavior
  - stdio still works
  - HTTP initialize/tools/list/tools/call
  - invalid method / invalid content
  - Origin/local safety checks
- Script smoke covers:
  - `--version`
  - HTTP server starts
  - initialize/tools/list/status over HTTP

### Extensibility

- This work creates a clean boundary for a future `MyFlowHub-MCP` repo:
  - transport-independent MCP protocol code
  - shared runtime abstraction
  - install/start/smoke scripts
- Future repo extraction should replace Win internal service imports with SDK/Core-facing packages before physically moving the code.

## Task Checklist

### DOCS-1: Clarify MCP shared-server requirements

- Files:
  - `docs/requirements/mcp-client.md`
  - `docs/requirements/README.md` only if title/index wording needs adjustment
- Goal:
  - Add multi-Codex shared state and local HTTP MCP server requirements.
- Acceptance:
  - Requirements mention both stdio compatibility and HTTP shared server mode.
- Tests:
  - Documentation review.
- Rollback:
  - Revert requirement edits.

### DOCS-2: Clarify MCP HTTP transport spec

- Files:
  - `docs/specs/mcp-client.md`
  - `docs/specs/README.md` only if title/index wording needs adjustment
- Goal:
  - Add transport contract, local security defaults, HTTP endpoint behavior, and Codex config shape.
- Acceptance:
  - Spec describes `stdio` and HTTP modes without conflicting with existing tool contracts.
- Tests:
  - Documentation review against MCP transport spec.
- Rollback:
  - Revert spec edits.

### MCP-1: Refactor MCP JSON-RPC dispatch for transport reuse

- Files:
  - `internal/mcp/server.go`
  - `internal/mcp/server_test.go`
- Goal:
  - Keep existing stdio behavior while making request handling reusable by HTTP transport.
- Acceptance:
  - Existing stdio tests pass.
  - Dispatch can process a single JSON-RPC request and return a response without direct stdin/stdout coupling.
- Tests:
  - `go test ./internal/mcp -run Test`
- Rollback:
  - Revert `internal/mcp` changes.

### MCP-2: Add HTTP MCP server mode

- Files:
  - `internal/mcp/http_server.go`
  - `internal/mcp/http_server_test.go`
  - `cmd/myflowhub-mcp/main.go`
- Goal:
  - Start a local HTTP MCP endpoint sharing one runtime.
- Acceptance:
  - POST initialize/tools/list/tools/call works.
  - GET returns 405 unless SSE is explicitly supported.
  - Origin/local safety checks are enforced.
- Tests:
  - `go test ./internal/mcp -run Http`
  - `go build ./cmd/myflowhub-mcp`
- Rollback:
  - Remove HTTP mode files and CLI flags.

### MCP-3: Preserve and validate script entrypoints

- Files:
  - `scripts/start-myflowhub-mcp.ps1`
  - `scripts/install-codex-myflowhub-mcp.ps1`
  - optional: `scripts/test-myflowhub-mcp-smoke.ps1`
- Goal:
  - Install/start scripts support HTTP MCP mode while keeping stdio mode.
  - Port the Windows PowerShell 5.1 parsing fix into this worktree.
- Acceptance:
  - `start-myflowhub-mcp.ps1 --version` works under Windows PowerShell 5.1.
  - install script can preview HTTP config.
- Tests:
  - `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 --version`
  - `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -WhatIf -Transport http ...`
- Rollback:
  - Revert script edits.

### MCP-4: Validation and review

- Files:
  - no planned production file changes beyond prior tasks
- Goal:
  - Run targeted validation and code review checklist.
- Acceptance:
  - Tests/builds pass or blockers are explicitly recorded.
  - Stage 3.3 checklist all pass.
- Tests:
  - `go test ./internal/mcp ./internal/mcpapp -count=1`
  - `go build ./cmd/myflowhub-mcp`
  - script checks above
- Rollback:
  - Document failed task and revert affected task files.

## Dependencies

- MCP transport spec: `https://modelcontextprotocol.io/specification/2025-06-18/basic/transports`
- Existing Win internal services:
  - `internal/services/session`
  - `internal/services/auth`
  - `internal/services/management`
  - `internal/services/flow`
  - `internal/services/topicbus`
  - `internal/services/varpool`

## Risks

- Codex HTTP MCP support may differ by version. If current Codex cannot consume local HTTP MCP reliably, keep stdio config as fallback and record blocker.
- Full Streamable HTTP SSE behavior may be more than needed. This plan starts with request/response POST and 405 for GET unless client requires SSE.
- Direct independent repo extraction is deferred; if required immediately, this workflow must return to Stage 1/2 and become multi-repo or new-repo scaffolding work.
- Main repo has an unrelated/uncommitted script fix from before workflow initialization; do not rely on it. Reapply inside worktree.

## Parallelism Assessment

Sub-agent delegation: not planned.

Reason:

- Write set is small but cross-cutting (`cmd`, `internal/mcp`, scripts, docs).
- Transport refactor and HTTP behavior depend on one coherent design.
- Main agent will keep ownership to avoid conflicting edits.

## Stage 3.3 Code Review

- 需求覆盖：通过
  - `stdio` 兼容入口保留，新增本机 HTTP MCP Server 模式。
  - Codex HTTP 配置预演可生成 `type = "http"` / `url = "http://127.0.0.1:17688/mcp"`。
  - `myflowhub_session_status` 直接返回 `mcp_server.transport/listen_addr/path/url`，满足共享入口排查要求。
- 架构合理性：通过
  - JSON-RPC dispatch 从 stdio loop 中抽出，HTTP 与 stdio 共享同一套 tool dispatch。
  - HTTP 模式在进程启动时构建一次 `mcpapp.Runtime`，所有 request 共享该 runtime。
- 性能风险：通过
  - 无每请求重建 Hub session/store/service；HTTP request 只做一次 body 读取和 JSON-RPC 分发。
  - body 读取使用 4 MiB 上限，避免无限读入。
- 可读性与一致性：通过
  - CLI flag 与脚本参数沿用现有显式命名。
  - HTTP transport 独立在 `internal/mcp/http_server.go`，未混入 tool 业务逻辑。
- 可扩展性与配置化：通过
  - `--transport stdio|http`、`--listen`、`--mcp-path` 均可配置。
  - 后续迁入独立 `MyFlowHub-MCP` 仓时，transport 边界已先收敛。
- 稳定性与安全：通过
  - HTTP 默认只允许 loopback listen，拒绝 `0.0.0.0` 与 `:port`。
  - HTTP request 带非 loopback `Origin` 时返回 403。
  - 写工具仍沿用现有 `allow_write` gate。
- 测试覆盖情况：通过
  - `GOWORK=off go test ./internal/mcp ./internal/mcpapp -count=1` 通过。
  - `GOWORK=off go build -o $env:TEMP\myflowhub-mcp-shared-server-test.exe ./cmd/myflowhub-mcp` 通过。
  - `scripts/start-myflowhub-mcp.ps1 --version` 通过。
  - `scripts/install-codex-myflowhub-mcp.ps1 -Transport http ... -WhatIf` 通过。
  - HTTP 进程级 smoke 通过：`initialize`、`tools/list`、`myflowhub_session_status`。
- 子Agent治理与审计：通过
  - 未派发子Agent，无外部结果需要整合。

## Stage 4 Change Archive

使用 `$m-docs` 校验变更归档、requirements/specs 影响、lessons 抽取和索引维护。

- Requirements impact: `updated`
  - Updated `docs/requirements/mcp-client.md`.
- Specs impact: `updated`
  - Updated `docs/specs/mcp-client.md`.
- Lessons impact: `updated`
  - Added `docs/lessons/powershell-utf8-nobom-parse.md`.
- Change archive:
  - Added `docs/change/2026-05-28_win-mcp-shared-http-server.md`.
- Index updates:
  - Updated `docs/change/README.md`.
  - Updated `docs/lessons/README.md`.
- Final verification:
  - `GOWORK=off go test ./internal/mcp ./internal/mcpapp -count=1`: passed.
  - `GOWORK=off go build -o $env:TEMP\myflowhub-mcp-shared-server-test.exe ./cmd/myflowhub-mcp`: passed.
  - `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 --version`: passed.
  - `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Transport http -Listen 127.0.0.1:17688 -McpPath /mcp -WhatIf`: passed.
  - `git diff --check`: passed; only CRLF normalization warnings were emitted.
- Workflow end:
  - Awaiting user confirmation before merge and worktree cleanup.

## Stage Gate

阻塞：否

Stage 4 complete. Awaiting workflow end confirmation.
