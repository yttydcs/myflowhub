# Plan - Auth 受控准入、authority 严格判定与发布链收口

## Goal
- 为 `auth` 引入受控准入、一次性角色 permit 和 authority fail-closed 规则，并把这套能力收口成可发布、可复现的 semver 依赖链。

## Related Requirements
- [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)

## Related Specs
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## Related Lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)

## Requirements Impact
- `add`

## Specs Impact
- `add`

## Lessons Impact
- `add`

## Workflow Information
- Control plane repo: `MyFlowHub3`
- Control plane branch: `feat/auth-admission-control`
- Control plane worktree: `d:\project\MyFlowHub3\worktrees\feat-auth-admission-control\workspace`
- Base: `master`
- Participating repos:
  - `MyFlowHub-Proto`
  - `MyFlowHub-Core`
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Server`

## Confirmed Scope
- 必须：
  - 普通注册支持 pending 审批流，批准前不下发正式 `node_id`
  - 支持一次性、角色绑定、至少绑定 `device_id` 的 permit 直接准入
  - 审批与签发能力通过 `auth.*` action 权限控制，而不是新节点类型
  - `hubruntime` / `SelfRegister` 在审批模式下支持 `parent.join_permit`
  - authority 选择按用户确认改为 fail-closed
- 不做：
  - 不引入共享长期 join key
  - 不新增独立 approver 节点类型
  - 不重做完整 token/RBAC 体系

## Architecture Summary
- `register` 保持为统一准入入口：
  - 无 permit 且开启审批时进入 pending
  - 带 permit 时由 authority 本地校验并直接准入
- admission state 由 authority 本地持久化到 `trusted_nodes.json.meta`，不依赖中间节点 assist 转发。
- authority 选择收敛为三态：
  - `local`
  - `remote`
  - `unavailable`
- 父链自动上联通过 `parent.join_permit` 打通，不再依赖开放注册兜底。
- 发布链按依赖方向收口，确保 `GOWORK=off` 也能工作。

## Task Checklist
- [x] `CTRL1` - 新增 workspace requirement 并完成 docs 治理路由
- [x] `PROTO1` - 扩展 auth 协议字段、状态和 action schema
- [x] `AUTH1` - register pending flow
- [x] `AUTH2` - permit issue / revoke / consume
- [x] `AUTH3` - approve / reject / list pending actions
- [x] `AUTH4` - admission state persist / reload 与测试
- [x] `CORE1` - config / permission / bootstrap 扩展
- [x] `SRV1` - hubruntime parent permit 透传
- [x] `AUTH5` - authority strict selection and fail-closed
- [x] `DOC1` - 更新 server auth 稳定 spec
- [x] `TEST1` - workflow-local 联调验证
- [x] `REL1` - 发布 `MyFlowHub-Proto v0.1.2` 与 `MyFlowHub-Core v0.4.8`
- [x] `AUTH6` - 对齐并发布 `auth/v0.1.4`
- [x] `SRV3` - `MyFlowHub-Server` semver 依赖对齐
- [x] `TEST2` - `GOWORK=off` 回归验证
- [x] `REV1` - 3.3 Code Review
- [x] `ARC1` - Stage 4 归档

## Blockers And Resolution
- 2026-03-24 blocker：
  - 功能在 workflow-local `go.work` 下通过，但 `GOWORK=off` 下失败。
  - 根因是 `MyFlowHub-SubProto/auth` 和 `MyFlowHub-Server` 依赖了尚未发布的 `Proto/Core` 新 API。
- 解决方式：
  - 先发布 `Proto v0.1.2`
  - 再发布 `Core v0.4.8`
  - 然后对齐并发布 `auth/v0.1.4`
  - 最后升级 `Server` 到 `proto v0.1.2 + core v0.4.8 + auth v0.1.4`
  - 以 `GOWORK=off` 关键测试作为收口门槛

## Code Review
- 需求覆盖：通过
  - pending 审批、permit 直入、authority strict selection、父链 permit 透传均已落地。
- 架构合理性：通过
  - 继续复用 `register` 统一入口，避免引入平行协议；authority state 明确化后不再依赖含糊的 nil 语义。
- 性能风险：通过
  - 新增状态以本地 map + 持久化 meta 管理，未引入额外热路径扫描；`list_pending_registers` 的 O(n) 枚举符合预期。
- 可读性与一致性：通过
  - config、permission、protocol、runtime、spec 按职责分层对齐。
- 可扩展性与配置化：通过
  - permit 结构预留后续增强空间；审批能力继续走 `auth.*` 权限，不与角色名耦死。
- 稳定性与安全：通过
  - authority 改为 fail-closed；permit 一次性消费并绑定 `device_id`；不可达路径显式失败。
- 测试覆盖情况：通过
  - workflow-local 与 `GOWORK=off` 两条验证链路均已覆盖关键路径。
- 子Agent治理与审计：通过
  - 无子 agent。

## Validation
- Workflow-local 验证：
  - `go test github.com/yttydcs/myflowhub-proto/... -count=1 -p 1`
  - `go test github.com/yttydcs/myflowhub-core/bootstrap -count=1 -p 1`
  - `go test github.com/yttydcs/myflowhub-subproto/auth/... -count=1 -p 1`
  - `go test github.com/yttydcs/myflowhub-server/hubruntime/... -count=1 -p 1`
  - `go test github.com/yttydcs/myflowhub-server/tests -run TestLoginHandler -count=1 -p 1`
- 发布链验证：
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto@v0.1.2`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-core@v0.4.8`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/auth@v0.1.4`
  - `MyFlowHub-SubProto/auth`: `GOWORK=off go test ./... -count=1 -p 1`
  - `MyFlowHub-Server`: `GOWORK=off go test ./hubruntime/... -count=1 -p 1`
  - `MyFlowHub-Server`: `GOWORK=off go test ./tests -run TestLoginHandler -count=1 -p 1`
- 结果：
  - 全部通过

## Release Results
- 已 push 的远端 tag：
  - `myflowhub-proto v0.1.2`
  - `myflowhub-core v0.4.8`
  - `myflowhub-subproto/auth v0.1.4`
- 已完成本地主线合并但未 push branch：
  - `MyFlowHub-Proto/main`
  - `MyFlowHub-Core/master`
  - `MyFlowHub-SubProto/main`
  - `MyFlowHub-Server/main`
- 当前控制面状态：
  - Stage 4 归档已完成
  - 根仓文档增量合回与 worktree cleanup 需等待 workflow end confirmation

## Rollback
- 代码层：
  - 对未 push branch 的本地主线 merge，可通过常规 revert 或后续 fix commit 处理。
- 发布层：
  - 已 push 的 tag 不改写；发现问题时通过更高 patch 版本修复。
- 文档层：
  - 在 workflow end confirmation 前，仅保留于 feature worktree 的归档可独立修订。
