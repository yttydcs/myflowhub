# Plan - MCP Ensure Running Startup

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub-Win`
- Branch: `feat/mcp-ensure-running`
- Base: local `main` at `1ecf3a2 feat: add shared HTTP MCP server`
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-ensure-running`
- Current Stage: 3.2 implementation

## Stage Records

### Initialization

- guide.md: read from `D:/project/MyFlowHub3/guide.md`; worktrees must be under `D:/project/MyFlowHub3/worktrees`; final work should notify `dev.codex.msg` when MCP is available.
- base/worktree confirmation: dedicated worktree created at `D:/project/MyFlowHub3/worktrees/feat-mcp-ensure-running` on branch `feat/mcp-ensure-running`.
- participating modules: `scripts/start-myflowhub-mcp.ps1`, MCP client docs, README, change archive.

### Stage 1 - Requirements Analysis

#### Goal

Add a lightweight `ensure-running` startup mode for the local HTTP MyFlowHub MCP server so repeated invocations reuse an existing local server and only start a background process when no MCP endpoint is available.

#### Scope

Must:

- Preserve existing `stdio` and direct `http` startup behavior.
- Add a script-level `EnsureRunning` mode for the shared HTTP endpoint.
- Probe the configured HTTP MCP URL before starting a new process.
- Reuse an already responsive MCP endpoint.
- Fail explicitly when the target URL responds but is not a valid MCP endpoint.
- Start a hidden background HTTP MCP server when the endpoint is not reachable.
- Poll until the endpoint is ready or timeout.
- Document the mode and validation path.

Optional:

- Support CLI-style aliases such as `--ensure-running`.
- Keep configurable listen address and MCP path through existing forwarded CLI args.

Out of scope:

- No system service, scheduled task, or OS daemon registration.
- No stdio-to-HTTP proxy.
- No cross-process lock file or PID manager.
- No remote or non-loopback default exposure.

#### Use Cases

- User runs the startup command before opening multiple Codex sessions; the first run starts the shared HTTP server, later runs reuse it.
- User includes the startup command in a local shell/bootstrap script without worrying about duplicate server processes.
- User gets an explicit failure when port/path is occupied by a different HTTP service.

#### Functional Requirements

- `start-myflowhub-mcp.ps1 -EnsureRunning` defaults to HTTP server mode with `--transport http --listen 127.0.0.1:17688 --mcp-path /mcp`.
- Existing forwarded arguments may override `--listen` and `--mcp-path`.
- Existing forwarded arguments may include `--endpoint`, `--config-dir`, `--device-id`, `--display-name`, `--allow-write`, and other MCP CLI flags.
- If `--transport` is provided in ensure mode, it must be `http`; `stdio` is rejected.
- A responsive endpoint must be verified by JSON-RPC `initialize`, not only by port-open checks.
- Background startup must not leave a visible interactive PowerShell window.

#### Non-functional Requirements

- Keep changes script-local and small.
- Avoid brittle process-name guessing for correctness; endpoint readiness is the source of truth.
- Keep output human-readable for manual use.
- Preserve Windows PowerShell 5.1 compatibility.

#### Inputs / Outputs

- Input: PowerShell startup script parameters plus forwarded MCP CLI args.
- Output: reuse/start/failure message and exit code.

#### Edge Cases

- Missing value after `--listen`, `--mcp-path`, or `--transport`.
- Endpoint connection refused.
- Endpoint returns non-JSON, non-200, or JSON without an MCP initialize result.
- Server process starts but never becomes ready.
- User requests ensure mode with `--transport stdio`.

#### Acceptance Criteria

- Existing `--version` passthrough still works.
- `-EnsureRunning` starts a local HTTP MCP server on a free port.
- Re-running `-EnsureRunning` against the same URL reuses the existing server.
- `-EnsureRunning` fails explicitly against an occupied non-MCP URL.
- README and stable docs describe the command and its limits.

#### Risks

- PowerShell argument quoting for background process startup.
- Ensure mode can only manage HTTP endpoint readiness; it is not a system-wide singleton.

#### Issue List

- none

### Stage 2 - Architecture Design

#### Overall Solution

Add `EnsureRunning` as a launcher-only mode in `scripts/start-myflowhub-mcp.ps1`. The launcher builds an HTTP MCP argument set, probes the endpoint using a minimal JSON-RPC `initialize` request, and either exits successfully on reuse or starts a hidden background PowerShell process that runs the same script in normal HTTP mode.

#### Alternatives Considered

- PID/lock file singleton: stronger process ownership, but more state and stale-lock handling than needed for this request.
- Windows service/daemon: useful later, but too heavy and requires install/uninstall lifecycle.
- stdio-to-HTTP proxy: would let Codex auto-launch through `command`, but it changes MCP transport behavior and is larger than this step.

#### Module Responsibilities

- `scripts/start-myflowhub-mcp.ps1`: owns ensure-mode parsing, probing, background start, and readiness polling.
- `README.md`: gives user-facing command examples.
- `docs/requirements/mcp-client.md`: records the launcher requirement.
- `docs/specs/mcp-client.md`: records the script contract and non-daemon limit.
- `docs/change/*`: archives the completed workflow.

#### Data / Call Flow

```text
start-myflowhub-mcp.ps1 -EnsureRunning
  -> normalize transport/listen/path
  -> POST initialize to http://listen/path
  -> if MCP result: reuse and exit 0
  -> if connection refused: Start-Process hidden normal HTTP server
  -> poll initialize until ready or timeout
```

#### Interface Drafts

```powershell
.\scripts\start-myflowhub-mcp.ps1 -EnsureRunning
.\scripts\start-myflowhub-mcp.ps1 --ensure-running --listen 127.0.0.1:17688 --mcp-path /mcp
.\scripts\start-myflowhub-mcp.ps1 -EnsureRunning --endpoint 10.3.3.5:9000 --allow-write
```

#### Error Handling and Safety

- Reject ensure mode when normalized transport is not `http`.
- Treat a live but invalid HTTP response as occupied/non-MCP and fail instead of launching another process.
- Keep default listen loopback through the MCP CLI default and generated args.
- Use hidden background process startup.

#### Performance and Testing Strategy

- Probing uses one small JSON-RPC POST.
- Polling has a bounded timeout.
- Tests are script smoke tests plus existing Go MCP tests/build.

#### Extensibility Design Points

- The probe and startup helpers remain script-local and can later back a scheduled-task or service installer.
- Endpoint readiness is decoupled from process identity, so future standalone MCP repo extraction keeps the same launcher contract.

#### Issue List

- none

### Stage 3.1 - Planning

#### Project Goal and Current State

Current `main` already supports a shared HTTP MCP server but requires a user to keep one server process running manually. This workflow adds an idempotent launcher mode that starts or reuses that server.

#### Docs Governance Routing Decision

Using `$m-docs` for routing and impact checks.

- Stable behavior change belongs in `docs/requirements/mcp-client.md` and `docs/specs/mcp-client.md`.
- Workflow result belongs in `docs/change/2026-05-28_win-mcp-ensure-running.md`.
- No new reusable lesson is expected unless validation exposes a recurring PowerShell/runtime pitfall.

#### Related Requirements / Specs / Lessons

- Requirements impact: clarify
- Specs impact: clarify
- Related requirements:
  - `docs/requirements/mcp-client.md`
- Related specs:
  - `docs/specs/mcp-client.md`
- Related lessons:
  - `docs/lessons/powershell-utf8-nobom-parse.md`

#### Executable Task List

- `MCP-ENSURE-1`: add ensure-running launcher mode.
- `MCP-ENSURE-2`: update docs and archive.
- `MCP-ENSURE-3`: validate and review.

#### Task Details

##### MCP-ENSURE-1 - Add Ensure-Running Launcher Mode

- Owner: Codex
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-ensure-running`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-mcp-ensure-running/plan.md`
- Goal: make the start script idempotently reuse or start the local HTTP MCP server.
- Files / Modules: `scripts/start-myflowhub-mcp.ps1`
- Write Set: script only
- Acceptance: free port starts server; second run reuses; invalid live endpoint fails.
- Test Points: `--version`, ensure start/reuse on a temp port, invalid endpoint smoke.
- Rollback: revert `scripts/start-myflowhub-mcp.ps1`.

##### MCP-ENSURE-2 - Update Documentation

- Owner: Codex
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-ensure-running`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-mcp-ensure-running/plan.md`
- Goal: document the launcher contract and limits.
- Files / Modules: `README.md`, `docs/requirements/mcp-client.md`, `docs/specs/mcp-client.md`, `docs/change/README.md`, `docs/change/2026-05-28_win-mcp-ensure-running.md`
- Write Set: docs only
- Acceptance: docs distinguish ensure mode from daemon/service and from HTTP Codex config.
- Test Points: markdown review and `git diff --check`.
- Rollback: revert docs changes.

##### MCP-ENSURE-3 - Validate and Review

- Owner: Codex
- Worktree: `D:/project/MyFlowHub3/worktrees/feat-mcp-ensure-running`
- Plan Path: `D:/project/MyFlowHub3/worktrees/feat-mcp-ensure-running/plan.md`
- Goal: confirm behavior and code quality before closeout.
- Files / Modules: no direct write set unless review finds gaps.
- Write Set: none by default
- Acceptance: validation commands pass; review checklist is recorded.
- Test Points: PowerShell smoke, Go targeted tests/build, `git diff --check`.
- Rollback: use per-task rollback above.

#### Dependencies

- Existing HTTP MCP server support from `1ecf3a2`.
- Local Go toolchain for fallback `go run` validation.

#### Risks and Notes

- Worktree is based on local `main`, which is ahead of `origin/main` by the two prior MCP commits.
- This workflow does not remove the need for an HTTP Codex config; it only makes starting the server idempotent.

#### Parallelism Assessment

- No sub-agent delegation. Change surface is small and tightly coupled around one script plus docs.

#### Issue List

- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation Record

#### File-level Change Summary

- `MCP-ENSURE-1`
  - `scripts/start-myflowhub-mcp.ps1`
    - Added `-EnsureRunning` / `--ensure-running`.
    - Added endpoint normalization, forwarded argument parsing, MCP `initialize` probing, hidden background startup, and bounded readiness polling.
    - Rejected ensure mode when forwarded `--transport` is not `http`.
    - Fixed development fallback stdout passthrough by avoiding function-return capture of `go run` output.
- `MCP-ENSURE-2`
  - `scripts/install-codex-myflowhub-mcp.ps1`
    - Added HTTP-mode `EnsureRunning` command output with config, identity, endpoint, and write-gate arguments.
  - `README.md`
    - Added `-EnsureRunning` startup example and non-service note.
  - `docs/requirements/mcp-client.md`
    - Clarified launcher reuse/start requirement and acceptance.
  - `docs/specs/mcp-client.md`
    - Clarified `-EnsureRunning` contract and failure modes.
  - `docs/change/2026-05-28_win-mcp-ensure-running.md`
    - Archived workflow result.
  - `docs/lessons/powershell-args-automatic-variable.md`
    - Captured PowerShell `$Args` automatic-variable pitfall.
  - `docs/change/README.md`, `docs/lessons/README.md`
    - Updated indexes.

#### Design Notes

- Endpoint readiness is the source of truth; the launcher does not try to infer ownership from process names.
- `-EnsureRunning` remains explicitly HTTP-only.
- The launcher starts a hidden background process but does not install a service or daemon.
- Fallback `go run` must stream stdout directly because stdio MCP depends on stdout.

#### Validation Results

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\start-myflowhub-mcp.ps1 -PreferSource --version`
  - Result: passed; output `dev`.
- `$env:GOWORK='off'; go test ./internal/mcp ./internal/mcpapp -count=1`
  - Result: passed.
- `$env:GOWORK='off'; go build -o $env:TEMP\myflowhub-mcp-ensure-running-test.exe ./cmd/myflowhub-mcp`
  - Result: passed.
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-myflowhub-mcp.ps1 -Transport http -Listen 127.0.0.1:17688 -McpPath /mcp -WhatIf`
  - Result: passed.
- Temporary port `-EnsureRunning` smoke on `127.0.0.1:17891`
  - Result: first run started server; second run reused endpoint.
- `-EnsureRunning --transport stdio`
  - Result: failed as expected.
- Non-MCP HTTP endpoint on `127.0.0.1:17892`
  - Result: failed as expected.

### Stage 3.3 - Code Review

- 需求覆盖：通过
  - Added idempotent shared HTTP server launcher mode and documented it in requirements/specs.
- 架构合理性：通过
  - Kept lifecycle logic in the startup script; did not alter MCP protocol dispatch or runtime ownership.
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - One small initialize probe before reuse/start; readiness polling is bounded.
- 可读性与一致性：通过
  - Script helpers use explicit names and keep launcher responsibilities separate from binary discovery.
- 可扩展性与配置化：通过
  - Existing `--listen`, `--mcp-path`, config, identity, endpoint, and write-gate flags are preserved.
- 稳定性与安全：通过
  - Ensures HTTP-only mode, detects non-MCP port occupancy, and keeps loopback defaults.
- 测试覆盖情况：通过
  - Covered Go tests/build, script old entrypoint, start/reuse path, invalid transport, invalid live endpoint, and install preview.
- 子Agent治理与审计：通过
  - No sub-agents used; all changes map to confirmed task IDs.

### Stage 4 - Change Archive

使用 `$m-docs` 完成 requirement/spec/lesson 影响复核。

- Requirements impact: updated
  - `docs/requirements/mcp-client.md`
- Specs impact: updated
  - `docs/specs/mcp-client.md`
- Lessons impact: updated
  - `docs/lessons/powershell-args-automatic-variable.md`
  - `docs/lessons/README.md`
- Change archive:
  - `docs/change/2026-05-28_win-mcp-ensure-running.md`
- Index updates:
  - `docs/change/README.md`
  - `docs/lessons/README.md`
