# 控制仓：清理残留 worktree / workflow 目录

## 变更背景 / 目标
- 控制仓 `worktrees/` 下残留了多份已结束 workflow 的 worktree 与目录壳，容易在后续排查时误判为“仍在进行中的任务”。
- 本次目标是仅清理已确认可丢弃、且不再作为有效工作区使用的残留项，保留当前仍在使用的长期工作区。

## 具体变更内容
### 1) 删除残留 git worktree
- 已删除：
  - `worktrees/fix-subproto-varstore-action-regression`
  - `worktrees/fix-android-release-gomobile-pin/MyFlowHub-Android`
  - `worktrees/release-auth-route-index-heal/repo/MyFlowHub-Server`
  - `worktrees/release-auth-route-index-heal/repo/MyFlowHub-Android`
  - `worktrees/release-auth-route-index-heal/repo/MyFlowHub-MetricsNode`
- 相关仓库均执行了 `git worktree prune`。

### 2) 删除旧 workflow 目录壳
- 已删除：
  - `worktrees/chore-rfcomm-release-deps`
  - `worktrees/feat-showcase-var-quickpick`
  - `worktrees/refactor-transport-pipe`
  - `worktrees/release-auth-route-index-heal`
  - `worktrees/fix-android-release-gomobile-pin`

### 3) 明确保留的长期工作区
- 保留：
  - `worktrees/MyFlowHub-Core`
  - `worktrees/MyFlowHub-Proto`
  - `worktrees/MyFlowHub-Server`

## 对应计划任务映射
- `WTCL-1`：复核候选清单与安全边界
- `WTCL-2`：删除残留 git worktree
- `WTCL-3`：删除旧 workflow 目录壳
- `WTCL-4`：回归验证与状态盘点
- `WTCL-5`：Code Review + 归档变更

## 关键设计决策与权衡
- 先清理 git worktree，再清理目录壳：避免目录仍被 git 注册占用，降低删除失败或状态不一致风险。
- 对存在未提交改动的 worktree，仅在用户明确允许“全部丢弃”后才使用 `--force` 删除。
- 不删除任何远端分支 / tag，也不清理 `repo/*` 主工作区：本次仅收口本地控制面噪音，不扩大到版本治理。
- 对长期工作区采用显式保留名单，而不是“删除全部后再恢复”，风险更低、可审计性更好。

## 测试与验证方式 / 结果
- `Get-ChildItem worktrees`：
  - 结果仅剩 `MyFlowHub-Core`、`MyFlowHub-Proto`、`MyFlowHub-Server` ✅
- 各仓 `git worktree list`：
  - `MyFlowHub-SubProto`：仅保留主工作区 ✅
  - `MyFlowHub-Android`：仅保留主工作区 ✅
  - `MyFlowHub-Server`：仅保留主工作区 ✅
  - `MyFlowHub-MetricsNode`：仅保留主工作区 ✅
- 路径存在性核对：
  - 所有目标删除项均不存在 ✅
  - 三个保留项均存在 ✅

## 潜在影响
- 被删除的本地 workflow 目录和 worktree 不能直接从文件系统恢复，只能依赖 git 历史或重新创建 worktree。
- 本次不会影响任何远端仓库、tag 或主工作区内容。

## 回滚方案
- 若误删某个 worktree：
  - 从对应仓库使用原分支重新执行 `git worktree add <path> <branch>`
- 若误删某个仅含文档的 workflow 目录：
  - 从控制仓历史文档恢复，或按需重新创建目录结构
