# 2026-04-05_graph-contract-release-chain-publish

## 变更背景 / 目标
- `flow graph/node contract` 相关代码、文档和运行时闭环已经在各仓 worktree 中完成，但正式 semver 依赖链尚未全部公开。
- 本次目标是在不混入额外功能变更的前提下，按依赖方向完成 `Proto -> SubProto/flow -> Server -> Win` 的发布收口，并用真实 `GOWORK=off` 验证证明下游可解析、可测试、可消费。

## 具体变更内容
### MyFlowHub-Proto
- 提交：`8e7341c`
- 发布：`v0.1.7`
- 说明：
  - 本轮不改 canonical contract 内容，只把当前 graph contract closure 的 canonical 基线正式公开。
  - 为通过 repo 内 contract freshness 校验，先执行：
    - `go run ./cmd/flowcontractgen -write -md-out docs/flow_contract.md -ts-out generated/flow_contract.ts`
  - 随后在单仓 `GOWORK=off` 环境下复验通过。

### MyFlowHub-SubProto/flow
- 提交：`84642c2`
- 发布：`flow/v0.1.5`
- 修改：
  - `flow/go.mod`
    - `github.com/yttydcs/myflowhub-proto`：`v0.1.6 -> v0.1.7`
    - 删除 `replace github.com/yttydcs/myflowhub-proto => ../../MyFlowHub-Proto`
  - `flow/go.sum`
    - 同步新的 Proto 校验和
- 说明：
  - 目标是让 `flow` 在无本地 sibling `replace` 的情况下可作为真实 module 发布点被消费。

### MyFlowHub-Server
- 提交：`c1ec782`
- 发布：`v0.0.15`
- 修改：
  - `go.mod`
    - `github.com/yttydcs/myflowhub-proto`：`v0.1.6 -> v0.1.7`
    - `github.com/yttydcs/myflowhub-subproto/flow`：`v0.1.4 -> v0.1.5`
  - `go.sum`
    - 同步新的 Proto / flow 校验和
- 说明：
  - 本轮不改 Server 业务逻辑，只做依赖消费升级与 release-state 收口。

### MyFlowHub-Win
- 提交：`aa1b482`
- 发布：`v0.0.15`
- 修改：
  - `go.mod`
    - `github.com/yttydcs/myflowhub-proto`：`v0.1.5 -> v0.1.7`
    - 删除 stale `replace github.com/yttydcs/myflowhub-proto => ../../worktrees/proto-server-release-align`
  - `go.sum`
    - 同步新的 Proto 校验和
- 说明：
  - 本轮不扩张 Win 功能面，只消除 stale 发布态依赖并确认 Wails / Go 目标验证可过。

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`none`

## Related requirements
- `D:\project\MyFlowHub3\worktrees\server-graph-contract-release-chain\docs\requirements\flow_data_dag.md`

## Related specs
- `D:\project\MyFlowHub3\worktrees\server-graph-contract-release-chain\docs\specs\flow.md`
- `D:\project\MyFlowHub3\worktrees\proto-graph-contract-release-chain\docs\flow_contract.md`

## Related lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)
- [frontend-worktree-wailsjs-missing.md](../lessons/frontend-worktree-wailsjs-missing.md)
- [frontend-build-empty-node-modules.md](../lessons/frontend-build-empty-node-modules.md)

## 对应 plan.md 任务映射
- `REL-1` - Proto release doc + commit + tag `v0.1.7`
- `REL-2` - SubProto/flow remove `replace`, bump Proto, validate, tag `flow/v0.1.5`
- `REL-3` - Server bump `proto/flow`, validate, tag `v0.0.15`
- `REL-4` - Win remove stale proto `replace`, bump Proto, validate, tag `v0.0.15`
- `REL-5` - Push branches/tags in dependency order and verify resolution
- `DOC-1` - Workspace Stage 4 archive

## 经验 / 教训摘要
- 发布顺序必须严格按依赖方向执行，不能跳过上游 tag 直接验证下游。
- `go list -m` / `go test` 的真实发布校验必须显式使用 `GOWORK=off`；否则顶层 `go.work` 可能把 sibling `replace` 混入，掩盖真实依赖链状态。
- Win worktree 的最小可靠验证链路仍然是：
  - `frontend/npm install`
  - `GOMAXPROCS=1 GOFLAGS=-p=1 GOWORK=off wails generate module`
  - targeted Go tests

## 可复用排查线索
- 症状：
  - `go.work` 联调通过，但单仓 `go list -m` / `go test` 仍失败
  - `go list -m` 报 `conflicting replacements`
  - Win 侧 `wails generate module` 在 fresh worktree 中因资源压力或前端依赖状态不稳定
- 触发条件：
  - workspace 根仍存在全局 `go.work`
  - 下游仓仍保留 sibling `replace`
  - 新 worktree 尚未完成 `frontend/node_modules` 与 `frontend/wailsjs` 准备
- 关键词：
  - `GOWORK=off`
  - `go list -m`
  - `conflicting replacements`
  - `flowcontractgen`
  - `wails generate module`
  - `GOMAXPROCS=1`
  - `GOFLAGS=-p=1`
- 快速检查：
  - 先用 `git ls-remote --tags origin <tag>` 确认 tag 确实已在远端存在
  - 再在真实下游仓执行 `GOWORK=off go list -m ...`
  - Win 若首轮失败，拆开执行 `npm install`、`wails generate module`、targeted Go tests

## 关键设计决策与权衡
- 已公开 tag 一律不改写；本轮通过追加更高 patch 收口，不做历史 tag 复写。
- Server 和 Win 都采用“最小依赖升级 + release docs”策略，不顺手混入额外 runtime 改动，减少回归面。
- 并行策略只用于不重叠 write set 的准备阶段；Win 子Agent尝试因主机资源压力未接管成功后，立即收回由主Agent完成，避免并行放大环境波动。

## 测试与验证方式 / 结果
### MyFlowHub-Proto
- `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过

### MyFlowHub-SubProto/flow
- `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过

### MyFlowHub-Server
- `GOWORK=off go test ./tests -run TestIntegrationFlowRoundTrip -count=1 -p 1`
  - 结果：通过
- `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto github.com/yttydcs/myflowhub-subproto/flow`
  - 结果：
    - `github.com/yttydcs/myflowhub-proto v0.1.7`
    - `github.com/yttydcs/myflowhub-subproto/flow v0.1.5`

### MyFlowHub-Win
- `frontend/npm install`
  - 结果：通过
- `GOMAXPROCS=1 GOFLAGS=-p=1 GOWORK=off wails generate module`
  - 结果：通过
- `GOWORK=off go test ./internal/services/flow ./internal/mcp ./internal/mcpapp -count=1 -p 1`
  - 结果：通过
- `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
  - 结果：`github.com/yttydcs/myflowhub-proto v0.1.7`

### 远端发布结果
- 发布阶段完成：
  - Proto
    - `origin/chore/graph-contract-release-chain`
    - `v0.1.7`
  - SubProto
    - `origin/chore/graph-contract-release-chain`
    - `flow/v0.1.5`
  - Server
    - `origin/chore/graph-contract-release-chain`
    - `v0.0.15`
  - Win
    - `origin/chore/graph-contract-release-chain`
    - `v0.0.15`
- 远端 tag 复验：
  - `git ls-remote --tags origin refs/tags/v0.1.7`
  - `git ls-remote --tags origin refs/tags/flow/v0.1.5`
  - `git ls-remote --tags origin refs/tags/v0.0.15`
  - 结果：全部可见

### Workflow End 收尾结果
- Proto
  - 本地：`refactor/proto-extract` 已 fast-forward 到 `8e7341c`
  - 远端：`origin/refactor/proto-extract` 已 push
  - 清理：远端 `chore/graph-contract-release-chain` 已删除；本地 feature branch 与 feature worktree 已删除
- SubProto
  - 本地：`refactor/subproto-management-module` 已 fast-forward 到 `84642c2`
  - 远端：`origin/refactor/subproto-management-module` 已 push
  - 清理：远端 `chore/graph-contract-release-chain` 已删除；本地 feature branch 与 feature worktree 已删除
- Server
  - 本地：`main` 已 fast-forward 到 `c1ec782`
  - 远端：未 push `main`
  - 原因：本地 `main` 在本轮结束后相对 `origin/main` 领先 `5` 个提交，其中包含本轮 release commit 之外的既有本地提交
  - 清理：本地 feature branch 与 feature worktree 已删除；远端 `chore/graph-contract-release-chain` 保留
- Win
  - 本地：`main` 已 fast-forward 到 `aa1b482`
  - 远端：未 push `main`
  - 原因：本地 `main` 在本轮结束后相对 `origin/main` 领先 `30` 个提交，其中包含本轮 release commit 之外的既有本地提交
  - 清理：本地 feature branch 与 feature worktree 已删除；远端 `chore/graph-contract-release-chain` 保留

## Code Review（3.3）结论
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计：通过
  - `Galileo` 参与 Server 本地准备，主Agent完成集成与远端发布。
  - Win 并行尝试因主机内存压力未接管成功，最终由主Agent本地完成。

## 潜在影响
- 正向影响：
  - 当前 `flow graph/node contract` 已有完整正式依赖链可供外部消费。
  - `Server` 与 `Win` 都已切到本轮正式 Proto / flow 版本，不再依赖本地 worktree `replace` 才能验证。
- 注意事项：
  - `Server` / `Win` 的远端 `main` 仍未包含本轮 release commit；当前远端可见入口仍是 release tag 和保留的 feature branch。
  - workspace 根控制面文档已更新，但根仓还有其他无关脏改，本轮未尝试整理或单独提交这些根级归档。

## 回滚方案
- 已 push 的 `v0.1.7` / `flow/v0.1.5` / `v0.0.15` / `v0.0.15` 不改写；若发现问题，只追加更高 patch 修复。
- `Proto` 或 `SubProto` 若需继续修正，在各自已推进的 `refactor/...` 基线上追加更高 patch；不回收已公开 tag。
- `Server` 或 `Win` 若需临时回退，可在各自主仓 `main` 上恢复旧依赖版本并重新验证，但不回收已公开 tag。
- 已删除的 feature worktree 只能通过重新创建 worktree 恢复，不做 silent rollback。

## 子Agent执行轨迹
- `Galileo`
  - 用途：`REL-3` 的 Server 本地准备与状态回传
  - 最终状态：返回本地发布链已就位、远端发布待 GitHub 连通性恢复
- Win 并行委派
  - 状态：早期尝试因主机资源不足未接管，未产生最终写入结果
  - 处理：由主Agent收回并本地完成验证与发布

## Notes
- GitHub `443` 在本 workflow 中一度出现 `Connection was reset` / 无法连接；后续重试恢复，最终所有远端 push 都已完成。
- workflow end 已执行到“本地 merge + worktree cleanup”。
- root `MyFlowHub3` 控制仓中，本轮新增的 root archive 文件仍处于未提交状态，因为 `docs/change/README.md` 与 `docs/plan/README.md` 同时承载了其他未归属到本轮的修改，不适合在本次收尾中一并提交。
