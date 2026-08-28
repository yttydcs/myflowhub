# 2026-08-28 Canonical checkout 迁入 repo/MyFlowHub

## Background

目录清理后，canonical monorepo 一度直接位于 `MyFlowHub3` 工作区根。用户确认该目录应作为控制工作区，所有标准代码 checkout 统一放在 `repo/` 下，附加 worktree 放在 sibling `worktrees/` 下。

## Changes

- 将 `.git`、源码、测试、`docs/`、`migration/`、构建入口和全部现有未提交仓库内容整体迁入 `repo/MyFlowHub`。
- 创建工作区级 `worktrees/`，保持其与 `repo/` 平级。
- `_android-sdk`、`附件/` 和 `附件.zip` 保留在工作区根，不进入代码仓库。
- `论文/` 含 44 个 Git 跟踪文件及未提交修改，为保持工作树状态完整而随 checkout 迁移。
- 更新当前仓库入口、checkout 布局规范和文档治理入口；历史归档中的旧绝对路径保留为当时证据。
- Windows 移动隐藏 `.git` 时留下的空源目录，仅在确认目标 HEAD、index、objects、refs 与 stash 完整后删除。

## Related Plan

- 无独立计划；本次执行用户确认的工作区布局修正。

## Related Intake

- [工作区目录清理与文档治理](../intake/2026-08-27_workspace-directory-docs-governance.md)

## Related Features

- none

## Related Requirements

- [统一节点运行时](../requirements/unified-node-runtime.md)

## Related Specs

- [仓库与模块边界](../specs/repository-and-module-boundaries.md)
- [构建与 CI](../specs/build-and-ci.md)

## Related Decisions

- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)

## Lessons Impact

- none：迁移过程未形成需要独立维护的新运行时或产品排障规则。

## Searchable Lessons Summary

- 移动 Windows checkout 时，隐藏 `.git` 可能报告非终止权限错误；必须分别验证源、目标的 HEAD、index、objects、refs 和 stash，不能仅依据移动命令退出状态判断。
- 关键词：`repo/MyFlowHub`、workspace root、checkout relocation、hidden `.git`、sibling worktrees。

## Intake Impact

- updated：记录用户对标准 checkout 位置的补充确认。

## Feature Impact

- none：不改变产品行为。

## Requirements Impact

- none：不改变统一节点运行时需求。

## Specs Impact

- updated：明确 workspace、canonical checkout、sibling worktree 和本机资料的边界。

## Decision Impact

- none：仍执行单一 canonical monorepo 决策，仅修正其物理 checkout 位置。

## Validation

- 新仓库根解析为 `master` 和迁移前同一 HEAD `133d6874e925d909faebbed43580bea4cc07d98b`；index、objects、refs 与两条 stash 完整。
- 旧工作区根不再包含 `.git`，`repo/MyFlowHub` 是唯一主 worktree。
- 八类文档的 601 个直接叶子文档全部进入分类索引；索引断链和稳定文档断链均为 0。
- `go test ./...` 与 `go vet ./...` 从新仓库根通过。
- `./scripts/mfh.ps1 -Action list` 通过；`./scripts/run-dev.ps1 -DryRun` 正确解析 canonical root 且未创建目录或进程。

## Rollback

- 本次为同盘目录移动，没有删除仓库内容；可在停止相关进程后将 checkout 全部内容连同 `.git` 逆向移动回工作区根。
- `_android-sdk` 与附件始终保留在工作区根，无需恢复。
