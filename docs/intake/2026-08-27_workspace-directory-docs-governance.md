# 2026-08-27 工作区目录清理与文档治理

## Source

- 用户在 vNext 全量迁移归档完成后提出的后续整理要求。

## Request Text / Source-preserving Summary

- “整理一下目录，该清理的清理一下，然后按照 `$m-docs` 目录的要求进行处理。”

## Context

- 旧 `repo/MyFlowHub-*` 已退役，但根目录仍留有旧 `go.work`、构建目录、临时日志、本机工具配置，以及附件中的完整旧仓代码副本。
- `docs/` 已具备八类治理目录，但若干索引仍停留在旧仓时代，存在未索引叶子文档和指向已删除仓库的链接。

## Confirmed Requirements

- 清理可再生成、已过期且无 Git 跟踪的目录与文件。
- 保留 `_android-sdk`、论文、附件原件和其他无法确认可删除的用户资料；附件中的完整旧仓代码副本可清理。
- 按 `intake/features/requirements/specs/decisions/plan/change/lessons` 分类维护文档及索引。
- 稳定文档不得继续把已删除旧仓当作当前事实来源。

## Open Questions

- 无；附件与本机 SDK 不属于 governed docs，保留在工作区根。

## Follow-up Clarification

- 2026-08-28，用户确认 `MyFlowHub3` 是工作区根，标准代码 checkout 应统一位于 `repo/` 下。
- `论文/` 中存在 Git 跟踪文件和未提交修改，因此作为完整工作树的一部分随 canonical checkout 迁移；未跟踪附件仍保留在工作区根。

## Routed Docs

- [文档总入口](../README.md)
- [功能索引](../features/README.md)
- [技术规范索引](../specs/README.md)
- [经验索引](../lessons/README.md)

## Related Changes

- [工作区目录清理与文档治理](../change/2026-08-27_workspace-directory-docs-governance.md)
- [Canonical checkout 迁入 repo/MyFlowHub](../change/2026-08-28_canonical-checkout-relocation.md)
