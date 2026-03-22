# 2026-03-22 Workspace Docs Governance

## Background
- 根仓库 `docs/` 原先同时承载导航层、变更归档、计划归档和顶层协议映射副本，但缺少受治理的 `requirements/specs/plan/change/lessons` taxonomy。
- 根级 `docs/README.md`、`plan.md`、`repos.md` 仍将读者引向 `docs/plan_archive/` 与顶层 `docs/protocol_map.md`，与新的文档治理目标不一致。

## Changes
- 创建根仓库标准分类目录与索引：
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
  - `docs/plan/README.md`
  - `docs/change/README.md`
  - `docs/lessons/README.md`
- 将 `docs/plan_archive/` 纯迁移到 `docs/plan/`，保留历史文件名以维持审计连续性。
- 将根仓库协议映射副本迁移到 `docs/specs/protocol_map.md`，并补充它是 `MyFlowHub-Proto` canonical 文档同步副本的说明。
- 重写根级入口：
  - `docs/README.md`
  - `plan.md`
  - `repos.md`
- 批量更新根仓库内对旧 canonical 路径的显式引用，统一改指向 `docs/plan/`、`docs/specs/` 和 Server 仓库新的 spec 入口。

## Related Plan
- `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-22_workspace-docs-governance.md`

## Related Requirements
- 无现存 workspace 级叶子需求文档需要改写

## Related Specs
- `D:\project\MyFlowHub3\docs\specs\README.md`
- `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\README.md`

## Requirements Impact
- none

## Specs Impact
- updated

## Plan Task Mapping
- `ROOT-DOC-001`：完成。根仓库 taxonomy 与 category README 已补齐。
- `ROOT-DOC-002`：完成。`docs/plan_archive/` 已迁到 `docs/plan/`。
- `ROOT-DOC-003`：完成。根仓库 `protocol_map.md` 已迁到 `docs/specs/`。
- `ROOT-DOC-004`：完成。根级导航文档已改到新 taxonomy。
- `ROOT-DOC-005`：完成。根仓库内高价值 canonical 引用已改到新路径。
- `ROOT-DOC-006`：完成。目录存在性、入口残留和 generated 区块已验证。

## Design Decisions And Tradeoffs
- 选择“纯净优先”策略：不保留 legacy redirect/stub，而是直接迁移 canonical 路径并更新导航。
- `docs/plan/` 保留 `plan_archive_*.md` 历史文件名，而不是继续大规模重命名叶子文件。
  - 这样目录 taxonomy 干净，同时保留历史检索键和归档审计连续性。
- 根仓库 `specs/` 只放 workspace 级长期 spec 入口和协议映射副本，不重复复制 Server 仓库的完整长期规范正文。

## Validation
- 目录与索引存在性检查：
  - `docs/README.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
  - `docs/plan/README.md`
  - `docs/change/README.md`
  - `docs/lessons/README.md`
  - `docs/specs/protocol_map.md`
- 入口层残留检查：
  - `rg -n "docs/plan_archive|docs/protocol_map\\.md|repo/MyFlowHub-Server/docs$|repo/MyFlowHub-Server/docs[^/]|plan_archive/README\\.md" plan.md repos.md docs/README.md docs/change/README.md docs/plan/README.md docs/specs/README.md -S`
  - 结果：无命中
- generated 区块完整性检查：
  - `rg -n "BEGIN GENERATED|END GENERATED" docs/specs/protocol_map.md -S`
  - 结果：区块边界保留

## Docs Governor Check
- Requirements impact: none
- Specs impact: updated
- Lessons impact: none
- Index update required: yes
- 结论：
  - 本次无需新增 `requirements` 叶子文档
  - `specs` 与 docs taxonomy 已更新
  - 无独立 `lessons` 产出
  - 根级与分类索引已同步更新

## Rollback
- 回退本次根仓库 docs governance 相关提交即可恢复：
  - `docs/README.md`
  - `docs/requirements/`
  - `docs/specs/`
  - `docs/plan/`
  - `docs/change/README.md`
  - `docs/lessons/`
  - `plan.md`
  - `repos.md`

## Agent Trace
- 本轮未使用子 Agent。
- 执行轨迹：
  - `ROOT-DOC-001` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\root-docs-governance` -> `docs/README.md`, `docs/requirements/*`, `docs/specs/*`, `docs/plan/*`, `docs/lessons/*` -> 通过
  - `ROOT-DOC-002` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\root-docs-governance` -> `docs/plan/*` -> 通过
  - `ROOT-DOC-003` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\root-docs-governance` -> `docs/specs/protocol_map.md` -> 通过
  - `ROOT-DOC-004` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\root-docs-governance` -> `docs/README.md`, `plan.md`, `repos.md` -> 通过
  - `ROOT-DOC-005` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\root-docs-governance` -> root markdown refs -> 通过
  - `ROOT-DOC-006` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\root-docs-governance` -> validation evidence -> 通过
