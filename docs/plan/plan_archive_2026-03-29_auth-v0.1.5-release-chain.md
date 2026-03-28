# Plan Archive - Auth v0.1.5 发布链收口

## Goal
- 为 remote authority admin 相关 auth 变更补齐正式 patch 发布。
- 让 `MyFlowHub-Server` 从 `github.com/yttydcs/myflowhub-subproto/auth v0.1.4` 升级到 `v0.1.5`。
- 以 `GOWORK=off` 证明 `SubProto auth -> Server` 的真实 semver 依赖链可解析、可测试。

## Related Requirements
- [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)

## Related Specs
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## Related Lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)

## Requirements Impact
- `none`

## Specs Impact
- `none`

## Lessons Impact
- `updated`

## Workflow Information
- Control plane repo: `MyFlowHub3`
- Control plane branch: `master`
- Root archive path: `D:\project\MyFlowHub3`
- Stage: `4`
- Participating repos:
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Server`
- Feature worktrees:
  - `D:\project\MyFlowHub3\worktrees\chore-subproto-remote-authority-auth-release`
  - `D:\project\MyFlowHub3\worktrees\chore-server-remote-authority-auth-release`

## Docs Routing
- 使用 `$m-docs` 校验本轮 root 归档与索引落点。
- 结论：
  - Requirements impact: `none`
  - Specs impact: `none`
  - Lessons impact: `updated`
  - Related requirements:
    - `docs/requirements/auth-controlled-admission.md`
  - Related specs:
    - `repo/MyFlowHub-Server/docs/specs/auth.md`
  - Related lessons:
    - `docs/lessons/cross-repo-semver-release.md`
- Canonical destination:
  - 发布链结果归档：`docs/change/2026-03-28_auth-v0.1.5-release-chain.md`
  - 计划归档：`docs/plan/plan_archive_2026-03-29_auth-v0.1.5-release-chain.md`
  - 可复用排查经验：继续沉淀到 `docs/lessons/cross-repo-semver-release.md`

## Confirmed Scope
- 必须：
  - 为 `MyFlowHub-SubProto/auth` 补齐真实最低依赖版本并发布 `auth/v0.1.5`
  - 将 `MyFlowHub-Server` 的 auth 依赖升级到 `v0.1.5`
  - 对 `auth` module 和 `Server` 分别执行 `GOWORK=off` 验证
  - 完成本地主线合并与 worktree cleanup
- 不做：
  - 不修改 `MyFlowHub-Win`
  - 不新增 auth 功能或协议字段
  - 不扩展为其他下游消费仓的版本扫平

## Architecture Summary
- 先收口上游 `auth` module，再收口下游 `Server`：
  1. 修正 `auth/go.mod/go.sum` 的最低 `Core/Proto` 版本
  2. 发布 `auth/v0.1.5`
  3. `Server` 执行最小 `go.mod/go.sum` 升级
  4. 在 `GOWORK=off` 下验证真实依赖链
- workflow 结束时：
  - `repo/*` 主线不保留 worktree 临时 `plan.md`
  - 详细计划统一归档到 root `docs/plan/`

## Participating Repos
- `MyFlowHub-SubProto`
  - Feature branch: `chore/remote-authority-auth-release`
  - Release commit: `47b6d8e`
  - Main cleanup commit: `89b7970`
- `MyFlowHub-Server`
  - Feature branch: `chore/remote-authority-auth-release`
  - Release commit: `0560068`
  - Main cleanup commit: `bbdb4fc`

## Task Checklist
- [x] `AUTHREL-1` 校验 `auth/v0.1.4..HEAD` 的发布面
- [x] `AUTHREL-2` 修正 `auth/go.mod/go.sum` 的最低依赖版本
- [x] `AUTHREL-3` `GOWORK=off` 验证 auth module 可发布
- [x] `AUTHREL-4` 推送分支并发布 `auth/v0.1.5`
- [x] `SRVAUTH-1` 确认 `auth/v0.1.5` 可解析
- [x] `SRVAUTH-2` 升级 `MyFlowHub-Server` 到 `auth v0.1.5`
- [x] `SRVAUTH-3` `GOWORK=off` 验证 Server 真实依赖链
- [x] `SRVAUTH-4` 提交、推送与归档
- [x] `MERGE-1` 本地主线快进合并并清理 repo 主线里的临时 `plan.md`
- [x] `CLEANUP-1` 清理本地 worktree 与本地 feature branch

## Validation Results
- `MyFlowHub-SubProto/auth`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-SubProto/auth`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/auth@v0.1.5`
  - 结果：解析到 `v0.1.5`
- `MyFlowHub-Server`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/auth`
  - 结果：`v0.1.5`
- `MyFlowHub-Server`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过

## Merge Results
- `repo/MyFlowHub-SubProto/main`
  - 已本地 fast-forward 到 `47b6d8e`
  - 后续清理提交：`89b7970`（删除误带入主线的 workflow `plan.md`）
  - `auth/v0.1.5` tag 已 push
- `repo/MyFlowHub-Server/main`
  - 已本地 fast-forward 到 `0560068`
  - 后续清理提交：`bbdb4fc`（删除误带入主线的 workflow `plan.md`）
  - `main` 已 push 到远端

## Worktree Cleanup
- 已移除并 prune：
  - `D:\project\MyFlowHub3\worktrees\chore-subproto-remote-authority-auth-release`
  - `D:\project\MyFlowHub3\worktrees\chore-server-remote-authority-auth-release`
- 已删除本地 feature branch：
  - `MyFlowHub-SubProto/chore/remote-authority-auth-release`
  - `MyFlowHub-Server/chore/remote-authority-auth-release`

## Remote Sync Notes
- 已完成：
  - `auth/v0.1.5` tag 已 push
  - `origin/chore/remote-authority-auth-release` 两个 feature branch 都已 push
  - `MyFlowHub-Server/main` 已 push
- 归档时仍待网络恢复后重试：
  - `MyFlowHub-SubProto/main` 推送
  - 两个仓库远端 feature branch 删除

## Rollback
- 发布层：
  - `auth/v0.1.5` 已 push，不改写；若发现问题，发更高 patch 修复
- 代码层：
  - `MyFlowHub-Server` 可回退 `go.mod/go.sum` 到 `auth v0.1.4` 并重新验证
  - `MyFlowHub-SubProto` 如需修正，继续在 `auth/v0.1.5` 之上发新 patch
- 清理层：
  - 已删除 worktree 只能通过重新创建 worktree 恢复，不做 silent rollback

## Notes
- 本轮未使用子 agent。
- 发布链功能面已经闭环；剩余仅是 `MyFlowHub-SubProto/main` 与远端 feature branch 的网络重试型同步动作。
