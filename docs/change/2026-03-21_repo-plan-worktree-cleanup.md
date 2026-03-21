# 2026-03-21 repo 旧计划与 worktree 残留清理

## 变更背景 / 目标

在根级 `plan.md`、`repos.md` 和 `docs/` 入口收敛之后，`repo/*` 主线目录里仍残留一批已完成 workflow 的 `plan.md` / `todo.md`，同时 `worktrees/` 与部分仓库内部还保留旧的 worktree/junction 残留。

这些内容已经不再是当前入口，继续保留会带来两个问题：

- 接手者容易把历史 workflow 计划误判为当前主线计划；
- worktree 残留会继续制造“目录像还在用、git 实际却不再注册”的假象。

本次清理目标：

- 删除 `repo/*` 主线目录下已过时的 workflow `plan.md` / `todo.md`；
- 删除未再注册的旧 worktree/junction 残留；
- 同步修正根级 `plan.md`，明确“临时计划放 worktree，完成后归档，不留在 repo 主线”。

## 具体变更内容

### 修改

- 更新根级 `plan.md`
  - 移除将 `repo/*/plan.md` 作为常规入口的描述；
  - 明确仓库主线目录不应长期保留历史 workflow 的 `plan.md` / `todo.md`。

### 删除

- `repo/*` 下过时计划文档：
  - `repo/MyFlowHub-Core/plan.md`
  - `repo/MyFlowHub-Proto/plan.md`
  - `repo/MyFlowHub-Server/plan.md`
  - `repo/MyFlowHub-Server/todo.md`
  - `repo/MyFlowHub-SubProto/plan.md`
  - `repo/MyFlowHub-SubProto/todo.md`
  - `repo/MyFlowHub-SDK/plan.md`
  - `repo/MyFlowHub-SDK/todo.md`
  - `repo/MyFlowHub-Win/plan.md`
  - `repo/MyFlowHub-Win/todo.md`
  - `repo/MyFlowHub-Android/plan.md`
  - `repo/MyFlowHub-Android/todo.md`
  - `repo/MyFlowHub-MetricsNode/plan.md`
  - `repo/MyFlowHub-MetricsNode/todo.md`
- worktree 残留：
  - 根级 `worktrees/` 下指向 `repo/*` 的旧 junction；
  - `repo/MyFlowHub-Core/`、`repo/MyFlowHub-Proto/`、`repo/MyFlowHub-Server/` 内部遗留的 `repo/` / `worktrees/` 残留目录。

## 任务映射

- 文档入口收敛后的清理阶段：
  - 删除主线仓内已归档计划；
  - 删除无注册记录的旧 worktree 残留；
  - 修正根级入口文档口径。

## 关键设计决策与权衡

1. 主线仓库不再承担 workflow 历史存放职责。
   - 已完成计划的审计价值由 `docs/change` 与 `docs/plan_archive` 承接。
2. 删除的是“历史计划正文”，不是“长期仓库说明”。
   - 如果某个仓库未来需要持续维护接手文档，应放到 README 或正式 docs，而不是继续堆 `plan.md` / `todo.md`。
3. 只清理未再注册的 worktree 残留。
   - 实际仍在 `git worktree list` 中的工作区不在本次清理范围内。

## 测试与验证方式 / 结果

- 校验根级 `plan.md` 已不再把 `repo/*/plan.md` 当作常规入口。
- 校验 `git worktree list` 与各子仓 `git worktree list` 不再依赖被清理的残留目录。
- 校验要删除的 `repo/*/plan.md` / `todo.md` 均为已完成 workflow 的历史文件，相关执行记录已在 `docs/change` / `docs/plan_archive` 中存在。

## 潜在影响与回滚方案

### 潜在影响

- 若有人仍依赖 `repo/*/plan.md` 直接回看历史，需要改为通过 `docs/change/README.md` 与 `docs/plan_archive/README.md` 进入。
- 这是有意调整，目的是把当前入口和历史归档明确分层。

### 回滚

- 文档口径可通过回滚本次对根级 `plan.md` 的修改恢复；
- 某个具体 `plan.md` / `todo.md` 若确需恢复，可从对应仓库 git 历史恢复；
- worktree/junction 残留若误删，需要按当时 workflow 重新创建，而不是恢复旧残留目录。
