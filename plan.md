# Plan - MyFlowHub3 全局索引

> 本文件只保留：当前状态、进行中事项、文档入口。
> 已完成 workflow 的正文、Checklist、Review、Merge 记录不再堆叠在这里，统一沉淀到 `docs/plan_archive/` 与 `docs/change/`。
> 压缩前的完整根级快照：`docs/plan_archive/plan_archive_2026-03-21_global-plan-pre-slim.md`。

## Current Status
- 协议字典以 `MyFlowHub-Proto`（`github.com/yttydcs/myflowhub-proto/protocol/*`）为准；`MyFlowHub-Server/protocol/*` 仅保留兼容壳语义。
- 客户端基础能力以 `MyFlowHub-SDK` 为统一入口；`MyFlowHub-Win`、`MyFlowHub-Android`、`MyFlowHub-MetricsNode` 作为上层应用尽量复用 SDK / Core / Proto。
- `repo/` 是控制面与集成面；`worktrees/` 是实现面；`docs/` 负责全局归档与接手材料。
- 根级 `plan.md` 不再承载已完成 workflow 的详细正文，避免历史内容反复堆积。

## Active Items
- 当前无根级进行中的 Checklist。
- 如后续开启新的全局 workflow：
  - 根级 `plan.md` 只保留一行摘要、状态和入口链接。
  - 详细执行计划放到对应 worktree 根目录的 `plan.md` / `todo.md`。
  - workflow 完成后，将详细内容归档到 `docs/plan_archive/` 与 `docs/change/`，并从这里移除正文。

## Documentation Map
- `docs/README.md`
  - `docs/` 目录总入口；串联 change、plan_archive、protocol_map 等文档。
- `target.md`
  - 当前根目录中不存在；仅作为历史归档中的旧引用保留理解。
- `repos.md`
  - 记录仓库职责、依赖边界、推进顺序、接手说明。
- `plan.md`
  - 记录全局当前状态、进行中事项入口、文档分工。
- `repo/*`
  - 各个实际源码仓库。
  - 仓库主线不再长期保留已完成 workflow 的 `plan.md` / `todo.md`；这类文档应优先存在于 worktree 并在完成后归档到 `docs/`。
- `docs/change/`
  - 记录已完成变更的背景、结果、验证、影响与回滚。
  - 入口索引：`docs/change/README.md`
- `docs/plan_archive/`
  - 记录历史 workflow 的完整计划正文、Checklist、Review 证据，以及根级旧版 plan 快照。
  - 入口索引：`docs/plan_archive/README.md`
- `docs/protocol_map.md`
  - 记录协议映射速查表。

## Historical Entry Points
- 文档总入口：`docs/README.md`
- 根级旧版全量历史：`docs/plan_archive/plan_archive_2026-03-21_global-plan-pre-slim.md`
- `Codex MCP runtime cleanup` workflow 计划归档：`docs/plan_archive/plan_archive_2026-03-21_codex-mcp-runtime-cleanup.md`
- `Codex MCP runtime cleanup` 变更归档：`docs/change/2026-03-21_codex-mcp-runtime-cleanup.md`
- 变更归档总索引：`docs/change/README.md`
- 计划归档总索引：`docs/plan_archive/README.md`
- 历史 workflow 计划正文：`docs/plan_archive/`
- 历史变更结果与验证记录：`docs/change/`

## Maintenance Rules
- 已完成内容不再直接写回根级 `plan.md` 正文。
- 根级 `plan.md` 优先做索引，不重复拷贝 `docs/change/` 或 `docs/plan_archive/` 的大段内容。
- `repo/*` 主线目录不应堆积历史 workflow 的 `plan.md` / `todo.md`；若仍需仓内长期说明，应改写到 README 或正式 docs。
- 若后续发现某段历史内容仍只存在于临时计划中，应先归档到 `docs/`，再在根级 `plan.md` 保留简要引用。
