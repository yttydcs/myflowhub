# Plan Archive - Auth 半中心 authority 策略同步与断链退化

## Goal
- 将 auth authority 收敛为“root 是唯一有效 authority”的半中心方案。
- 通过运行时 `authority_policy_sync` 下发 `effective_authority_id`，而不是持久化 `authority.node_id`。
- 在断链场景下冻结新准入，但允许本地已知身份继续登录。

## Current State
- auth 受控准入已经支持 pending / permit / approval / authority fail-closed。
- authority 选择仍主要依赖静态 `authority.node_id` 或直接父链语义，无法稳定表达“root 才是最终 authority”。
- 多 hop assist 链路若继续按“每跳都可能落本地 state”处理，会把中间 hub 错误当成 authority。

## Workflow Info
- Control repo: `MyFlowHub3`
- Branch: `feat/auth-semi-central-authority`
- Base: `main`
- Stage: `4`
- Participating repos:
  - `MyFlowHub-Proto`
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Server`

## Docs Routing
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- 结论：
  - Requirements impact: `none`
  - Specs impact: `updated`
  - Related requirements: `docs/requirements/auth-controlled-admission.md`
  - Related specs:
    - `repo/MyFlowHub-Server/docs/specs/auth.md`
    - `repo/MyFlowHub-Server/docs/specs/protocol_map.md`
  - Related lessons: `none`
- Canonical destination:
  - 稳定 authority runtime contract 写入 `repo/MyFlowHub-Server/docs/specs/auth.md`
  - workflow 结果归档到 `docs/change/`
  - root 级 workflow 正文归档到 `docs/plan/`

## Stage 1 - Requirements Analysis
- 目标：
  - root 维持单一 authority 语义，非 root 只消费并传播 runtime policy。
- 范围：
  - 必须：
    - 定义 `authority_policy_sync` wire contract
    - 增加 in-memory authority policy state
    - 半中心模式下优先用 runtime lease 决定 authority
    - 父链在线但 lease 未到或过期时，仍允许沿父链逐级上送，避免 bootstrap 死锁
    - 父链断开时冻结新准入，但允许本地已知身份登录
    - 更新稳定 spec 和协议映射副本
    - 增加回归测试
  - 不做：
    - 不做 pending 逐级复制或 root 聚合 pending 缓存
    - 不做 `authority.node_id` 持久配置下发
    - 不做审批远程化或 Win UI 改造
- 使用场景：
  - root 在线时，所有后代节点的新准入请求都沿父链上送到 root
  - 子树断链时，新 register / approve / permit 不可继续
  - 子树断链时，本地已知设备仍可基于已有 binding / pubkey 登录
- 功能需求：
  - 非 root 节点接收、应用并继续广播 authority policy
  - 仅接受来自父连接的 policy sync，忽略 stale epoch
  - assist register/login/query 需要支持多 hop `SourceID + TargetID` 转发
- 非功能需求：
  - 不引入新的持久化 state
  - 保持变更面最小、回滚清晰
  - 保持 Proto 只定义 wire、SubProto 定义 runtime、Server 定义稳定 spec
- 验收标准：
  - root 可以广播 authority policy
  - 非 root 只接受最新 policy，并在父链断开时 fail-closed
  - 中间 hub 不创建本地 pending / binding 污染
  - 已知设备登录在退化期仍可成功
- 风险：
  - 把 lease 设计成硬门禁会破坏 parent bootstrap register
  - 若响应终点处理错误，多 hop assist 会污染中间节点状态

## Stage 2 - Architecture Design
- 总体方案：
  - 新增 `authority_policy_sync` action，由 root 周期性广播 `effective_authority_id=root` 的 runtime lease。
  - `resolveAuthority()` 在半中心模式下优先读取 lease；若 lease 缺失或过期但父链仍在线，仍通过直接父链逐级上送到 root。
  - assist 响应改为面向 edge hub 的 targeted response，中间节点只转发不落本地 admission state。
- 模块职责：
  - `MyFlowHub-Proto/protocol/auth`
    - 定义 action 常量与 payload struct
  - `MyFlowHub-SubProto/auth`
    - 维护 runtime policy state
    - 处理 policy apply / forward
    - 实现半中心退化路由和多跳 assist forwarding
  - `MyFlowHub-Server/docs/specs/auth.md`
    - 记录稳定 authority contract 与退化语义
- 数据 / 调用流：
  - root 广播 `authority_policy_sync`
  - 非 root 从父连接接收 policy，更新 lease 并向子连接广播
  - `register / assist_register / assist_query_credential / assist_login` 在半中心模式下沿父链逐级上送
  - 父链断开时，新 admission 请求返回 `authority unavailable`
  - 本地已知设备 login 直接用本地 binding / pubkey 校验
- 接口草案：
  - action: `authority_policy_sync`
  - payload:
    - `mode`
    - `effective_authority_id`
    - `epoch`
    - `ttl_sec`
- 错误与安全：
  - policy sync 仅接受父连接来源
  - stale epoch 不生效
  - 不持久化 runtime authority policy
  - approve / reject / permit 继续是 authority 本地操作，不假装已支持全网任意节点远程审批
- 性能与测试策略：
  - runtime state 为内存常量时间读写
  - 不新增热路径磁盘 I/O
  - 重点测试：
    - root broadcast
    - newest-only apply
    - header-target forward
    - degraded known-device login
- 可扩展性设计点：
  - 后续若要做审批远程化，可复用当前 authority resolution 与 targeted response 语义
  - 当前不把 runtime lease 混进持久配置，避免后续配置层耦合

## Participating Repos
- MyFlowHub-Proto
  - Worktree: `D:/project/MyFlowHub3/worktrees/proto-auth-semi-central-authority`
  - Plan: `D:/project/MyFlowHub3/worktrees/proto-auth-semi-central-authority/plan.md`
  - Write set: `protocol/auth/types.go`, `docs/protocol_map.md`, `docs/change/*`
- MyFlowHub-SubProto
  - Worktree: `D:/project/MyFlowHub3/worktrees/subproto-auth-semi-central-authority`
  - Plan: `D:/project/MyFlowHub3/worktrees/subproto-auth-semi-central-authority/plan.md`
  - Write set: `auth/*`, `docs/change/*`
- MyFlowHub-Server
  - Worktree: `D:/project/MyFlowHub3/worktrees/server-auth-semi-central-authority`
  - Plan: `D:/project/MyFlowHub3/worktrees/server-auth-semi-central-authority/plan.md`
  - Write set: `docs/specs/auth.md`, `docs/specs/protocol_map.md`, `docs/change/*`

## Task Checklist
- [x] `AUTHPOL-PROTO-1` 定义 `authority_policy_sync` 协议常量与 payload struct
- [x] `AUTHPOL-SUB-1` 增加 runtime authority policy state 和 sync handler
- [x] `AUTHPOL-SUB-2` 实现半中心 authority resolution、多跳 assist forwarding 与断链退化
- [x] `AUTHPOL-SUB-3` 补齐回归测试
- [x] `AUTHPOL-SRV-1` 更新稳定 auth spec 与协议映射副本
- [x] `REV-1` 完成 3.3 review
- [x] `ARC-1` 完成 repo/workspace 归档
- [x] `MERGE-1` 合并回 `repo/` 控制仓主线

## Repo Tasks
- `AUTHPOL-PROTO-1`
  - Goal: 为下游提供 canonical auth wire symbol
  - Files: `protocol/auth/types.go`, `docs/protocol_map.md`
  - Acceptance: 下游能直接引用 `ActionAuthorityPolicySync` / `AuthorityPolicySyncData`
  - Tests: `GOWORK=off go test ./... -count=1 -p 1`
  - Rollback: 回退新增 action / struct 与协议映射
- `AUTHPOL-SUB-1` / `AUTHPOL-SUB-2` / `AUTHPOL-SUB-3`
  - Goal: 在 auth runtime 中落地半中心 authority、转发与退化逻辑
  - Files: `auth/auth.go`, `auth/routing.go`, `auth/transport.go`, `auth/actions_*`, `auth/authority_policy*.go`
  - Acceptance:
    - root 可广播 policy
    - assist 多跳只转发不落本地 pending / binding
    - 断链后只允许本地已知身份登录
  - Tests: `go test ./... -count=1 -p 1` with `GOWORK=D:/project/MyFlowHub3/.tmp/auth-semi-central-work/go.work`
  - Rollback: 回退 runtime policy / forward / degraded login 改动
- `AUTHPOL-SRV-1`
  - Goal: 让稳定 spec 与实际 runtime 语义一致
  - Files: `docs/specs/auth.md`, `docs/specs/protocol_map.md`
  - Acceptance: 文档完整覆盖半中心 authority 语义与边界
  - Tests: docs consistency review
  - Rollback: 回退 spec / protocol map 副本

## Parallelism Assessment
- 本轮未派发子 agent。
- 原因：
  - Proto / SubProto / Server 三处写集虽然分离，但语义高度耦合
  - 关键路径在 authority contract 对齐与行为收口，拆分并行收益不高

## Notes
- 3.3 Review：
  - 需求覆盖：通过
  - 架构合理性：通过
  - 性能风险：通过
  - 可读性与一致性：通过
  - 可扩展性与配置化：通过
  - 稳定性与安全：通过
  - 测试覆盖情况：通过
  - 子Agent治理与审计：通过
- 验证结果：
  - `MyFlowHub-Proto`: `GOWORK=off go test ./... -count=1 -p 1`
  - `MyFlowHub-SubProto/auth`: `go test ./... -count=1 -p 1` with `GOWORK=D:/project/MyFlowHub3/.tmp/auth-semi-central-work/go.work`
  - `MyFlowHub-Server`: 文档与实现对齐审阅完成
- Merge 结果：
  - `repo/MyFlowHub-Proto` fast-forward 到 `4b7bd48`
  - `repo/MyFlowHub-SubProto` fast-forward 到 `ceaecfd`
  - `repo/MyFlowHub-Server` fast-forward 到 `16b3059`
- 阻塞：否
- Stage：`4`
