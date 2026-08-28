# Plan Archive - Graph Contract 发布链收口

## Goal
- 以最小变更把当前 `flow graph/node contract` 的本地闭环收口为正式 semver 依赖链。
- 依赖顺序发布：
  - `MyFlowHub-Proto v0.1.7`
  - `MyFlowHub-SubProto/flow flow/v0.1.5`
  - `MyFlowHub-Server v0.0.15`
  - `MyFlowHub-Win v0.0.15`
- 用真实 `GOWORK=off` 验证证明下游消费链可解析、可测试。

## Related Requirements
- `D:\project\MyFlowHub3\worktrees\server-graph-contract-release-chain\docs\requirements\flow_data_dag.md`

## Related Specs
- `D:\project\MyFlowHub3\worktrees\server-graph-contract-release-chain\docs\specs\flow.md`
- `D:\project\MyFlowHub3\worktrees\proto-graph-contract-release-chain\docs\flow_contract.md`

## Related Lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)
- [frontend-worktree-wailsjs-missing.md](../lessons/frontend-worktree-wailsjs-missing.md)
- [frontend-build-empty-node-modules.md](../lessons/frontend-build-empty-node-modules.md)

## Requirements Impact
- `none`

## Specs Impact
- `none`

## Lessons Impact
- `none`

## Workflow Information
- Control plane repo: `MyFlowHub3`
- Control plane branch: `master`
- Root archive path: `D:\project\MyFlowHub3`
- Stage: `4`
- Participating repos:
  - `MyFlowHub-Proto`
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Server`
  - `MyFlowHub-Win`
- Feature worktrees:
  - `D:\project\MyFlowHub3\worktrees\proto-graph-contract-release-chain`
  - `D:\project\MyFlowHub3\worktrees\subproto-graph-contract-release-chain`
  - `D:\project\MyFlowHub3\worktrees\server-graph-contract-release-chain`
  - `D:\project\MyFlowHub3\worktrees\win-graph-contract-release-chain`

## Docs Routing
- 使用 `$m-docs` 校验本轮 root 归档与索引落点。
- 结论：
  - Requirements impact: `none`
  - Specs impact: `none`
  - Lessons impact: `none`
- Canonical destination:
  - 发布链结果归档：`docs/change/2026-04-05_graph-contract-release-chain-publish.md`
  - 计划归档：`docs/plan/plan_archive_2026-04-05_graph-contract-release-chain-publish.md`
  - lessons 仅复用既有排障文档，不新增新的 lesson 叶子文档

## Confirmed Scope
- 必须：
  - 发布 Proto `v0.1.7`
  - 发布 SubProto/flow `flow/v0.1.5`
  - 发布 Server `v0.0.15`
  - 发布 Win `v0.0.15`
  - 每层都在 `GOWORK=off` 下完成最小充分验证
  - 补齐 repo-level release docs 与 workspace-level archive
- 不做：
  - 不扩张到 `auth`、`exec`、`file` 等无关模块
  - 不改写已公开 tag
  - 不在本阶段自动执行 merge / worktree cleanup

## Architecture Summary
- 发布链按依赖方向推进：
  1. 先公开 Proto canonical contract 基线
  2. 再让 `flow` 删除本地 proto `replace`，发布为真实 module
  3. 然后让 Server 消费 `proto v0.1.7 + flow v0.1.5`
  4. 最后让 Win 去掉 stale proto `replace`，消费 `proto v0.1.7`
- 验证链按“最小但真实”原则执行：
  - Proto / flow：单仓 `GOWORK=off go test`
  - Server：行为级 round-trip 测试
  - Win：`npm install` + `wails generate module` + targeted Go tests

## Participating Repos
- `MyFlowHub-Proto`
  - Feature branch: `chore/graph-contract-release-chain`
  - Release commit: `8e7341c`
  - Release tag: `v0.1.7`
- `MyFlowHub-SubProto`
  - Feature branch: `chore/graph-contract-release-chain`
  - Release commit: `84642c2`
  - Release tag: `flow/v0.1.5`
- `MyFlowHub-Server`
  - Feature branch: `chore/graph-contract-release-chain`
  - Release commit: `c1ec782`
  - Release tag: `v0.0.15`
- `MyFlowHub-Win`
  - Feature branch: `chore/graph-contract-release-chain`
  - Release commit: `aa1b482`
  - Release tag: `v0.0.15`

## Task Checklist
- [x] `REL-1` Proto release doc + commit + tag `v0.1.7`
- [x] `REL-2` SubProto/flow remove `replace`, bump Proto, validate, tag `flow/v0.1.5`
- [x] `REL-3` Server bump `proto/flow`, validate, tag `v0.0.15`
- [x] `REL-4` Win remove stale proto `replace`, bump Proto, validate, tag `v0.0.15`
- [x] `REL-5` Push branches/tags in dependency order and verify resolution
- [x] `DOC-1` Workspace Stage 4 archive

## Validation Results
- `MyFlowHub-Proto`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-SubProto/flow`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Server`
  - `GOWORK=off go test ./tests -run TestIntegrationFlowRoundTrip -count=1 -p 1`
  - 结果：通过
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto github.com/yttydcs/myflowhub-subproto/flow`
  - 结果：
    - `github.com/yttydcs/myflowhub-proto v0.1.7`
    - `github.com/yttydcs/myflowhub-subproto/flow v0.1.5`
- `MyFlowHub-Win`
  - `frontend/npm install`
  - `GOMAXPROCS=1 GOFLAGS=-p=1 GOWORK=off wails generate module`
  - `GOWORK=off go test ./internal/services/flow ./internal/mcp ./internal/mcpapp -count=1 -p 1`
  - 结果：通过
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
  - 结果：`v0.1.7`

## Remote Publish Results
- 已完成：
  - `MyFlowHub-Proto`
    - `origin/chore/graph-contract-release-chain`
    - `v0.1.7`
  - `MyFlowHub-SubProto`
    - `origin/chore/graph-contract-release-chain`
    - `flow/v0.1.5`
  - `MyFlowHub-Server`
    - `origin/chore/graph-contract-release-chain`
    - `v0.0.15`
  - `MyFlowHub-Win`
    - `origin/chore/graph-contract-release-chain`
    - `v0.0.15`
- 远端 tag 可见性已通过 `git ls-remote --tags origin ...` 复验。

## Review Outcome
- Stage 3.3 checklist 已通过：
  - 需求覆盖
  - 架构合理性
  - 性能风险
  - 可读性与一致性
  - 可扩展性与配置化
  - 稳定性与安全
  - 测试覆盖情况
  - 子Agent治理与审计

## Merge / Cleanup Status
- 已执行：
  - `MyFlowHub-Proto/refactor/proto-extract` 本地 fast-forward 到 `8e7341c`
  - `MyFlowHub-SubProto/refactor/subproto-management-module` 本地 fast-forward 到 `84642c2`
  - `MyFlowHub-Server/main` 本地 fast-forward 到 `c1ec782`
  - `MyFlowHub-Win/main` 本地 fast-forward 到 `aa1b482`
  - 4 个 feature worktree 已移除并 prune
  - 4 个本地 feature branch 已删除
- 远端同步结果：
  - `MyFlowHub-Proto/refactor/proto-extract` 已 push
  - `MyFlowHub-SubProto/refactor/subproto-management-module` 已 push
  - `MyFlowHub-Proto` / `MyFlowHub-SubProto` 的远端 `chore/graph-contract-release-chain` 已删除
  - `MyFlowHub-Server/main` 与 `MyFlowHub-Win/main` 未 push
- 未 push `Server` / `Win` main 的原因：
  - `MyFlowHub-Server/main` 相对 `origin/main` 领先 `5` 个提交
  - `MyFlowHub-Win/main` 相对 `origin/main` 领先 `30` 个提交
  - 这些领先提交不只包含本轮 release commit，因此本次收尾不擅自把它们一起推送
- 保留的远端分支：
  - `MyFlowHub-Server/chore/graph-contract-release-chain`
  - `MyFlowHub-Win/chore/graph-contract-release-chain`

## Rollback
- 发布层：
  - `v0.1.7` / `flow/v0.1.5` / `v0.0.15` / `v0.0.15` 已 push，不改写；若发现问题，只追加更高 patch 修复。
- 代码层：
  - `Server` / `Win` 可在各自 worktree 中回退依赖版本并重新验证，但不回收已公开 tag。
- 控制面：
  - root `docs/change` / `docs/plan` 归档可在 workflow 未结束前继续增补，不影响已发布 semver 事实。

## Notes
- GitHub `443` 在本 workflow 中曾短暂不可达，先前 push 被阻塞；后续重试恢复，最终全部发布成功。
- `Galileo` 参与了 Server 本地准备阶段；Win 并行委派尝试因主机资源不足未接管，最终由主Agent完成。
- root `MyFlowHub3` 的本轮 archive 文件仍未单独提交；原因是 `docs/change/README.md` 与 `docs/plan/README.md` 的 diff 同时包含其他不属于本轮的索引补录，不能安全混入一次收尾提交。
