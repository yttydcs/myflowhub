# 2026-03-21 Codex MCP Runtime Cleanup

## Change Background / Goal
- Background:
  - Long-running Codex sessions on Windows were accumulating large numbers of MCP-related `node.exe` runtimes.
  - The main growth came from `chrome-devtools-mcp`, with additional accumulation from `ace-tool` and `context7-mcp`.
  - The user requested a temporary mitigation that can be launched manually, without force-killing current sessions from this workflow.
- Goal:
  - Provide a local script that can preview and manually clean old MCP runtimes.
  - Provide double-click launchers so the cleanup can be triggered without typing PowerShell commands.

## Concrete Changes
- Added `scripts/cleanup-codex-mcp.ps1`
  - Enumerates `node.exe` / `cmd.exe` processes.
  - Matches known MCP command lines for `chrome`, `ace`, and `context7`.
  - Prints summary statistics and eligible targets by age.
  - Supports `Preview` and `Kill` modes.
- Added `scripts/cleanup-codex-mcp-preview.cmd`
  - Double-click preview entry point.
- Added `scripts/cleanup-codex-mcp-chrome-safe.cmd`
  - Double-click safe cleanup for old `chrome-devtools-mcp` processes.
- Added `scripts/cleanup-codex-mcp-all-safe.cmd`
  - Double-click safe cleanup for old `chrome + ace + context7` processes.
- Added `todo.md`
  - Records the workflow goal, task breakdown, acceptance criteria, tests, and rollback points.

## Plan Mapping
- T1 统计并固化“主要涨得快”的 MCP 类型与默认清理策略
  - Result:
    - Confirmed `chrome-devtools-mcp` as the fastest-growing family and the default cleanup target.
- T2 实现主 PowerShell 清理脚本
  - Result:
    - Implemented `scripts/cleanup-codex-mcp.ps1`.
- T3 实现双击启动入口
  - Result:
    - Implemented three `.cmd` launchers.
- T4 验证脚本预览与参数路径
  - Result:
    - Verified preview and safe no-target kill path.
- T5 执行 Code Review 并归档变更
  - Result:
    - Completed review and this change archive.

## Key Design Decisions / Tradeoffs
- Prioritized `chrome-devtools-mcp` as the default cleanup target
  - Reason:
    - It was the fastest-growing runtime family in the observed system snapshot.
    - It also includes extra watchdog child processes, so its footprint grows faster.
- Used a single PowerShell script with thin `.cmd` wrappers
  - Reason:
    - PowerShell is the right place for process enumeration, age filtering, and safe batch termination.
    - `.cmd` wrappers make the tool double-clickable and bypass PowerShell execution-policy friction.
- Kept the default cleanup conservative
  - Reason:
    - The goal is temporary mitigation, not aggressive cleanup.
    - Age-based filtering reduces the chance of killing recently started MCP runtimes.
- Kept the script ASCII-only
  - Reason:
    - Windows PowerShell compatibility was broken by non-ASCII script text during verification.
    - ASCII text removes that encoding risk.

## Performance / Resource Notes
- The script performs one process enumeration and does all filtering and aggregation in memory.
- No background polling or persistent process is introduced.
- The script does not add new runtime overhead outside the manual execution window.

## Test / Verification
- Verified with `powershell.exe`:
  - `powershell -NoLogo -ExecutionPolicy Bypass -File .\scripts\cleanup-codex-mcp.ps1 -Mode Preview -Kinds chrome,ace,context7 -MinAgeMinutes 30`
  - Result:
    - Summary output worked and showed `chrome` as the largest / fastest-growing family in the current snapshot.
- Verified with `powershell.exe` no-target kill path:
  - `powershell -NoLogo -ExecutionPolicy Bypass -File .\scripts\cleanup-codex-mcp.ps1 -Mode Kill -Kinds chrome -MinAgeMinutes 10080`
  - Result:
    - Script reported no eligible targets and exited safely without killing anything.
- Verified double-click entry path:
  - `cmd /c "echo.| scripts\cleanup-codex-mcp-preview.cmd"`
  - Result:
    - Wrapper successfully resolved and invoked the PowerShell script.

## Potential Impact
- Positive:
  - Gives the user a fast manual mitigation path when MCP runtimes accumulate.
  - Reduces the need to inspect process trees manually.
- Negative / Risk:
  - If a still-active Codex session is using a matched MCP runtime older than the age threshold, manual cleanup can still disrupt that session.
  - This does not fix the root cause of MCP lifecycle accumulation.

## Rollback Plan
- Remove the following files:
  - `scripts/cleanup-codex-mcp.ps1`
  - `scripts/cleanup-codex-mcp-preview.cmd`
  - `scripts/cleanup-codex-mcp-chrome-safe.cmd`
  - `scripts/cleanup-codex-mcp-all-safe.cmd`
  - `todo.md`
- Remove this archive file:
  - `docs/change/2026-03-21_codex-mcp-runtime-cleanup.md`

## Sub-Agent Trace
- No sub-agents were used in this workflow.
- Reason:
  - The write set was small and concentrated in `scripts/` plus a single archive document.
  - Splitting the work would have increased coordination cost without reducing risk.

