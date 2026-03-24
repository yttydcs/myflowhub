# Plan - Auth admission 下游依赖跟进

## Goal
- 将 auth admission 上游发布链继续推进到 `SDK / Win / Android` 消费层，并明确哪些仓库已可安全发布，哪些仍存在发布阻塞。

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
- Control plane branch: `chore/auth-admission-downstream`
- Control plane worktree: `d:\project\MyFlowHub3\worktrees\chore-auth-admission-downstream\workspace`
- Participating repos:
  - `MyFlowHub-SDK`
  - `MyFlowHub-Win`
  - `MyFlowHub-Android`
  - `MyFlowHub-Server`（仅作为 Android `replace` 解析目标）

## Confirmed Scope
- 必须：
  - 扫描 `SDK / Win / Android` 是否落后于 `Proto v0.1.2`、`Core v0.4.8`、`auth/v0.1.4`
  - 对需要跟进的仓库执行最小依赖升级
  - 以 `GOWORK=off` 完成验证
  - 对可以安全发布的仓库完成发布
- 不做：
  - 不新增 auth 功能
  - 不修改稳定 requirements / specs
  - 不把范围扩大成所有历史落后依赖的总清扫

## Architecture Summary
- 顺序按依赖方向执行：
  - `SDK`
  - `Win`
  - `Android`
- Android 继续保留 `replace` 到 sibling `Server/SDK` 的现有策略。
- 对 Android release 额外检查 `.github/workflows/*.yml` 中 sibling repo checkout 的真实基线。

## Task Checklist
- [x] `SCAN1` - 依赖面扫描
- [x] `SDK1` - SDK 对齐到 `core v0.4.8 + proto v0.1.2`
- [x] `WIN1` - Win 对齐到 `core v0.4.8 + proto v0.1.2 + sdk v0.1.11`
- [x] `AND1` - Android/hubmobile 对齐到新 Core/Proto/auth 口径
- [x] `TEST1` - 三仓 `GOWORK=off` 验证
- [x] `REL1` - 发布 SDK `v0.1.11` 与 Win `v0.0.11`
- [x] `REL1-B` - 识别并记录 Android release blocker
- [x] `REV1` - 3.3 Code Review
- [x] `ARC1` - Stage 4 归档

## Release Results
- 已 push tag：
  - `myflowhub-sdk v0.1.11`
  - `myflowhub-win v0.0.11`
- 未发布 tag：
  - `MyFlowHub-Android`
  - 原因：其 CI/release workflow checkout 的 `yttydcs/myflowhub-server` 默认分支尚不能保证包含本轮 auth admission server 基线

## Blockers And Resolution
- Android release blocker：
  - `hubmobile/go.mod` 通过 `replace ../../MyFlowHub-Server` 使用 sibling Server
  - Android 的 `ci.yml` / `release.yml` 通过 `actions/checkout` 拉取 `yttydcs/myflowhub-server`，但未指定包含本轮 auth admission 的公开 ref
- 结论：
  - Android 依赖升级与本地 `GOWORK=off` 验证可以完成
  - Android 远端 tag release 暂缓，直到 Server 公开基线与 CI checkout 口径一致

## Code Review
- 需求覆盖：通过
  - 三个目标消费层都已完成扫描与结论收口。
- 架构合理性：通过
  - 仍按依赖方向推进，没有把本轮扩展成全生态版本扫平。
- 性能风险：通过
  - 仅依赖版本升级，无运行时逻辑变更。
- 可读性与一致性：通过
  - Win 从 pseudo-version 回到稳定 tag，SDK/Android 版本口径明确。
- 可扩展性与配置化：通过
  - Android 继续沿用既有 `replace` 策略，没有顺带引入 CI 结构变更。
- 稳定性与安全：通过
  - 对可安全发布的仓库已经实际发 tag；对 Android 则阻止了不安全 release。
- 测试覆盖情况：通过
  - SDK / Win / Android 均完成 `GOWORK=off` 验证。
- 子Agent治理与审计：通过
  - 无子 agent。

## Validation
- `MyFlowHub-SDK`: `GOWORK=off go test ./... -count=1 -p 1`
- `MyFlowHub-Win`: `GOWORK=off go test ./... -count=1 -p 1`
- `MyFlowHub-Android/hubmobile`: `GOWORK=off go test ./... -count=1 -p 1`
- `go list -m github.com/yttydcs/myflowhub-sdk@v0.1.11`
- `go list -m github.com/yttydcs/myflowhub-win@v0.0.11`
- 结果：
  - 全部通过

## Rollback
- SDK / Win：
  - 代码层可 revert
  - 已 push tag 不改写，通过更高 patch 修复
- Android：
  - revert `hubmobile/go.mod` / `go.sum`
  - 本轮未发 tag，无远端 tag 回滚问题
