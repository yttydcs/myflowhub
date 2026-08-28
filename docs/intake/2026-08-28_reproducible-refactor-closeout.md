# 2026-08-28 重构可复现性与文档治理收口

## Source

- 来源：用户在 vNext 全量迁移、旧仓退役、目录治理和两轮归档后，要求继续检查整体重构状态，并明确要求“自己设置 goal，把整个重构流程跑完”。
- 日期：2026-08-28。

## Request Text / Source-preserving Summary

- 继续完成当前 MyFlowHub3 重构，不停留在“当前脏工作树能跑”的状态。
- 自行建立持续 Goal，完整经过讨论、计划、执行、测试和归档阶段。
- 保留架构与迁移相关文档，并按受治理文档目录维护索引。
- canonical 代码仓库继续位于 `repo/MyFlowHub`，workflow 实现在 `worktrees/` 下的独立工作树进行。

## Problem / Opportunity

`master@078f4b7` 已包含统一节点运行时、第一方产品迁移和 Wails 并发启动修复，但 canonical 主 checkout 仍有尚未进入 HEAD 的关键构建输入、测试修正和从旧仓提取的受治理文档。当前工作树验证通过不能证明全新 checkout 可复现；尤其 `build/toolchain.json` 未被 HEAD 跟踪，而已提交的架构测试依赖该文件。

## Confirmed Goals

- 使 canonical HEAD 在全新 checkout 中自包含并可执行完整本地门禁。
- 受控导入与本轮重构直接相关的构建输入、并发测试修正、文档迁移结果和索引变化。
- 保留现有论文、附件及其他无关工作区资产，不移动、不删除、不提交到本 workflow。
- 完成本地 commit、测试、归档、合并和 workflow worktree 清理。

## Non-goals

- 不新增 serial、USB 或 WebSocket Transport。
- 不恢复 legacy SubProto 或旧 module compatibility bridge。
- 不 push、发布、签名、创建远端或改变远端仓库状态。
- 不把真实硬件、商店或签名平台的外部认证伪装为本地通过。

## Assumptions

- `D:/project/MyFlowHub3/repo/MyFlowHub/docs` 是用户已选择的 canonical `docs_root`；当前仓库没有 remote，文档保持本地。
- 主 checkout 的论文目录变化属于独立用户资产，不是重构收口输入。
- 旧仓提取出的 `docs/plan`、`docs/change`、`docs/lessons` 只有在索引可达、相对链接有效且内容属于 MyFlowHub 历史时才纳入。

## Options Considered

1. 最小修复：只提交 `build/toolchain.json` 和两处测试修正。优点是改动小；缺点是继续遗留大量已提取但未治理的历史文档，不满足此前目录治理目标。
2. 受控导入：按写集导入重构关键文件和受治理文档，排除论文与外部动作，并在干净 worktree 完整验证。优点是 HEAD 自包含、历史可查、风险边界清晰；成本是需要逐类审计文档和索引。
3. 整体快照：直接提交主 checkout 的全部未提交内容。优点是快速；缺点是会混入论文、二进制和无关个人资产，无法审查和回滚。

## Rejected Options

- 拒绝最小修复作为最终方案：它只能修复构建门禁，不能完成用户已要求的文档迁移与目录治理。
- 拒绝整体快照：写集不可控，会侵犯无关资产边界并降低仓库可维护性。

## Recommended Direction

采用受控导入。所有实现和文档编辑只发生在 `refactor/reproducible-closeout` 工作树；先建立可执行 Task ID，再按文件来源和文档分类迁移，随后用全新 checkout 运行完整门禁。通过后创建 change/lesson 归档，本地合并到 `master`，删除 workflow worktree；主 checkout 原有无关变化保持不动。

## Open Questions

- 阻塞问题：无。
- 外部硬件、签名和发布证据继续明确记录为未执行，不阻塞本地重构收口。

## Worktree / Branch / Docs Root Status

- Project root: `D:/project/MyFlowHub3`
- Code repo: `D:/project/MyFlowHub3/repo/MyFlowHub`
- Docs root: `D:/project/MyFlowHub3/repo/MyFlowHub/docs`
- Base: `master@078f4b75ee8316b0baaed67c4be4d6f7fbf301d9`
- Branch: `refactor/reproducible-closeout`
- Active worktree: `D:/project/MyFlowHub3/worktrees/reproducible-closeout`
- Handoff to `$m-plan`: ready。

## Routed Docs

- [统一节点运行时需求](../requirements/unified-node-runtime.md)
- [构建与 CI 契约](../specs/build-and-ci.md)
- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)
- [vNext 全量迁移与最终切换](2026-08-27_vnext-full-migration.md)

## Related Changes

- [Wails 开发启动并发修复与重构收口](../change/2026-08-28_wails-dev-concurrency-and-refactor-closeout.md)
