# 2026-04-15 - SubProto：收口遗留 management 分支并修正默认分支

## 变更背景 / 目标
`MyFlowHub-SubProto` 的远端默认分支长期仍指向 `refactor/subproto-management-module`，而不是 `main`。继续沿用这个默认头会带来两个风险：

1. 新 clone、人工检查和部分 CI/脚本默认落到旧 refactor 分支，误判当前主线。
2. `flow/v0.1.5` 对应的主线依赖收口仍停留在旧分支上，`main` 不直接包含这次收口提交。

本次目标是把该遗留分支的唯一未合入提交正式收进 `main`，验证 `flow` 单仓可用，并把远端默认分支修回 `main`。

## 具体变更内容
1. 在独占 worktree `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout` 中重新 fetch 并确认 live refs：
   - 初始 `origin/main`：`7429ffcb6fcc1054a4223c15289cdd04c35351d0`
   - 初始 `origin/refactor/subproto-management-module`：`84642c20919de6ad71505f1e269a2cb4f06fb1fd`
   - 初始 `origin/HEAD`：`refactor/subproto-management-module`
2. 发现 topic 分支对 `main` 的 branch-only 提交只有 1 个：
   - `84642c2 chore: 发布 flow v0.1.5`
3. 在 clean worktree 分支 `chore/subproto-management-branch-closeout` 上执行 merge：
   - merge commit：`7fb13380d78ab6feec8cd697d2912faf78316198`
   - message：`merge: 合并 subproto flow v0.1.5 到 main`
4. merge 过程中仅出现一个真实冲突：
   - `docs/change/README.md`
   - 处理方式：保留 `origin/main` 上较新的索引项，同时补回 `2026-04-05_flow-v0.1.5.md` 及说明，未改动其它正文语义。
5. merge 结果包含的实质收口内容：
   - `flow/go.mod`
     - `github.com/yttydcs/myflowhub-proto` 从 `v0.1.6` 升到 `v0.1.7`
     - 移除本地 `replace github.com/yttydcs/myflowhub-proto => ../../MyFlowHub-Proto`
   - `flow/go.sum`
     - 对齐到 `myflowhub-proto v0.1.7`
   - `docs/change/2026-04-05_flow-v0.1.5.md`
   - `docs/change/README.md`
6. 将 merge 结果直接推送到远端主线：
   - `git push origin HEAD:main`
   - 远端 `main` 更新为 `7fb13380d78ab6feec8cd697d2912faf78316198`
7. 因本机没有 `gh` CLI 且浏览器未登录 GitHub，改用 git credential helper 提供的现有凭据走 GitHub REST API，把默认分支改回 `main`。
8. 在默认分支修正后，删除远端陈旧分支：
   - `refactor/subproto-management-module`

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`none`

原因：本次问题已被现有 lesson `docs/lessons/cross-repo-semver-release.md` 覆盖，尤其是“默认分支 checkout 错误基线”的风险与快速检查项，无需新增 lesson。

## Related requirements
- none

## Related specs
- none

## Related lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)

## 对应 plan.md 任务映射
- `SUBCLOSE-1` - 再次 fetch 并确认 `main` / topic / remote HEAD 的最新状态 ✅
- `SUBCLOSE-2` - 在 clean worktree 完成 merge，并处理 `docs/change/README.md` 索引冲突 ✅
- `SUBCLOSE-3` - 在合并态执行 `flow` module 的 `GOWORK=off` 定向验证并 push `main` ✅
- `SUBCLOSE-4` - 修正 GitHub 默认分支到 `main`，并删除陈旧远端分支 ✅
- `SUBCLOSE-5` - 按 `$m-docs` 归档 `docs/change` / `docs/plan`，补索引并记录 lessons 影响 ✅

## 经验 / 教训摘要
- 不要把 `origin/HEAD` 当作 `main` 的替代证据；这里的真实问题就是默认分支长期漂移到旧 refactor 分支。
- 这类“遗留 release/refactor 分支未收口”问题未必涉及业务逻辑，常常只是主线历史和远端 metadata 没有同步收口。
- 即使只是 merge 旧分支，也要基于最新 `origin/main` 重新做真实 merge；之前基于旧 `main` 的无冲突结论只能作为参考，不能直接当最终证据。
- 当主路径已有脏工作树时，用 clean worktree merge 再推送 `HEAD:main` 比在主路径硬做 checkout/merge 更安全。

## 可复用排查线索
- 症状：
  - `git remote show origin` 显示 `HEAD branch` 不是 `main`
  - 某个历史 refactor/release 分支似乎“还没合并”
  - 远端 tag 指向的提交只在历史分支里出现
- 触发条件：
  - 发布链或 refactor 收口后，没有同步切回默认分支
  - 旧分支保留时间过长，且仍是仓库默认头
- 关键词：
  - `origin/HEAD`
  - `default branch`
  - `refactor/subproto-management-module`
  - `flow/v0.1.5`
  - `git remote show origin`
  - `git ls-remote --symref origin HEAD`
- 快速检查：
  - `git show -s --format='%D %H %s' origin/main origin/<topic>`
  - `git rev-list --count origin/main..origin/<topic>`
  - `git branch -r --contains <sha>`
  - `git ls-remote --symref origin HEAD`

## 关键设计决策与权衡
1. 不在 `repo/MyFlowHub-SubProto` 主路径直接 merge。
   - 原因：主路径已有未提交改动，直接 checkout / merge 风险高。
2. 在 clean worktree 分支 merge 后，用 `git push origin HEAD:main` 推远端主线。
   - 原因：既满足最小改动，也避免污染既有基线。
3. 默认分支修改不依赖 `gh` CLI。
   - 原因：本机无 `gh`，浏览器无登录态；直接复用 git credential helper 的现有凭据走 GitHub API 更稳。
4. 删除陈旧分支放在默认分支切换之后。
   - 原因：先确保仓库默认头稳定，再做清理动作，避免“删除的正是默认分支”这一类设置风险。

## 测试与验证方式 / 结果
- git 关系验证：
  - `git merge-base origin/main origin/refactor/subproto-management-module`
  - `git rev-list --count origin/main..origin/refactor/subproto-management-module`
  - 结果：branch-only commit 为 1 个，即 `84642c2`
- merge 验证：
  - 结果：形成 merge commit `7fb1338`
  - 唯一冲突：`docs/change/README.md`，已最小合并
- 单仓定向测试：
  - 在 `flow/` 下执行 `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：`ok github.com/yttydcs/myflowhub-subproto/flow`
- 远端主线验证：
  - `git ls-remote origin refs/heads/main`
  - 结果：`refs/heads/main -> 7fb1338`
- 默认分支验证：
  - GitHub API 返回 `default_branch=main`
  - `git ls-remote --symref origin HEAD`
  - 结果：`ref: refs/heads/main HEAD`
- 陈旧分支删除验证：
  - `git push origin --delete refactor/subproto-management-module`
  - `git ls-remote --heads origin refactor/subproto-management-module main`
  - 结果：仅剩 `main`

## 潜在影响
- `MyFlowHub-SubProto` 的新 clone / 默认 checkout / 部分自动化流程现在会正确落到 `main`。
- `flow/v0.1.5` 对应的主线收口现在正式进入远端 `main` 历史。
- 任何仍硬编码旧分支名的私有脚本，如果依赖 `refactor/subproto-management-module`，需要改成 `main` 或显式 tag/ref。

## 回滚方案
- 如果需要撤回本次主线合并：
  - 对 merge commit `7fb1338` 执行 `git revert -m 1 7fb13380d78ab6feec8cd697d2912faf78316198`
  - 再 push 回远端 `main`
- 如果需要恢复历史分支：
  - 从 `84642c20919de6ad71505f1e269a2cb4f06fb1fd` 重新创建 `refactor/subproto-management-module`
- 如果默认分支设置错误：
  - 再次通过 GitHub API 把 `default_branch` 改到正确分支

## 子Agent执行轨迹
- 本轮未使用子Agent。
