# 2026-03-26 Auth 半中心 authority 策略同步与断链退化

## 变更背景 / 目标
- 现有 auth 受控准入已经支持 pending 审批、permit 和 authority fail-closed，但 authority 语义仍容易退化为“直接父节点就是 authority”。
- 在多 hop 拓扑下，这会带来两个问题：
  - `authority.node_id` 一旦被理解为持久化下发配置，断链和重启后的陈旧 authority 状态会变得难以收敛。
  - `assist_register / assist_login / assist_query_credential` 若每跳都尝试落本地 pending / binding，会把中间 hub 错误当成 authority，污染本地准入状态。
- 本次目标是把 auth authority 收敛为用户确认的“半中心”方案：
  - root 仍是单一有效 authority
  - root 通过运行时 `authority_policy_sync` 下发 `effective_authority_id`
  - 父链断开时冻结新准入，但允许本地已知身份继续登录

## 具体变更内容
### MyFlowHub-Proto
- 更新 [protocol/auth/types.go](../../repo/MyFlowHub-Proto/protocol/auth/types.go)
  - 新增 `ActionAuthorityPolicySync = "authority_policy_sync"`
  - 新增 `AuthorityPolicySyncData`
    - `mode`
    - `effective_authority_id`
    - `epoch`
    - `ttl_sec`
- 更新 [docs/protocol_map.md](../../repo/MyFlowHub-Proto/docs/protocol_map.md)
  - 重新生成协议映射，纳入 `authority_policy_sync`
- 归档：[2026-03-26_auth-semi-central-authority.md](../../repo/MyFlowHub-Proto/docs/change/2026-03-26_auth-semi-central-authority.md)

### MyFlowHub-SubProto/auth
- 更新 [auth/auth.go](../../repo/MyFlowHub-SubProto/auth/auth.go)
  - 增加半中心 authority runtime state
  - root 在无父链场景下周期性广播 authority policy
  - `OnReceive` 增加 auth-local `TargetID` 转发判定
- 新增 [auth/authority_policy.go](../../repo/MyFlowHub-SubProto/auth/authority_policy.go)
  - 解析 `auth.authority_mode` / `auth.authority_policy_ttl_sec`
  - 维护 `mode/effective_authority_id/epoch/ttl` 的 in-memory lease
- 新增 [auth/actions_authority_policy.go](../../repo/MyFlowHub-SubProto/auth/actions_authority_policy.go)
  - 仅接受来自父连接的 policy sync
  - 忽略 stale epoch
  - 将最新 policy 继续广播给下游
- 新增 [auth/auth_forward.go](../../repo/MyFlowHub-SubProto/auth/auth_forward.go)
  - 为 `assist_register / assist_login / assist_query_credential` 增加 `SourceID + TargetID` 多跳转发
  - 无上游 authority 路径时显式返回 `authority unavailable`
- 更新 [auth/routing.go](../../repo/MyFlowHub-SubProto/auth/routing.go)
  - `resolveAuthority()` 在半中心模式下优先使用 runtime lease
  - policy 缺失或过期但父链仍在线时，仍允许沿父链逐级上送，避免 bootstrap 死锁
- 更新 [auth/actions_register.go](../../repo/MyFlowHub-SubProto/auth/actions_register.go)
  - intermediate assist path 只转发，不创建本地 pending / binding
- 更新 [auth/actions_login.go](../../repo/MyFlowHub-SubProto/auth/actions_login.go)
  - 断链退化期保留本地已知身份登录
  - 仍依赖上游 authority 的分支在断链时显式失败
- 更新 [auth/actions_query.go](../../repo/MyFlowHub-SubProto/auth/actions_query.go)
  - `assist_query_credential` 走相同的多跳 authority forwarding
- 新增 [auth/authority_policy_test.go](../../repo/MyFlowHub-SubProto/auth/authority_policy_test.go)
  - 覆盖 root 下发、policy apply、stale epoch、header-target forward、degraded login
- 归档：[2026-03-26_auth-semi-central-authority.md](../../repo/MyFlowHub-SubProto/docs/change/2026-03-26_auth-semi-central-authority.md)

### MyFlowHub-Server
- 更新 [docs/specs/auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
  - 新增 `auth.authority_mode=semi-central` 的 authority 选择规则
  - 明确 `authority_policy_sync` 数据契约
  - 记录“lease 缺失/过期但父链在线时仍允许 bootstrap 路由”的边界
  - 明确断链后只允许本地已知身份登录，不允许新准入
  - 明确 approve / reject / permit 仍是 authority 本地操作
- 更新 [docs/specs/protocol_map.md](../../repo/MyFlowHub-Server/docs/specs/protocol_map.md)
  - 同步 Proto canonical 协议映射副本
- 归档：[2026-03-26_auth-semi-central-authority.md](../../repo/MyFlowHub-Server/docs/change/2026-03-26_auth-semi-central-authority.md)

## Requirements impact
- none

## Specs impact
- updated

## Lessons impact
- none

## Related requirements
- [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)

## Related specs
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
- [protocol_map.md](../../repo/MyFlowHub-Server/docs/specs/protocol_map.md)

## Related lessons
- none

## 对应 plan.md 任务映射
- `AUTHPOL-PROTO-1`
- `AUTHPOL-SUB-1`
- `AUTHPOL-SUB-2`
- `AUTHPOL-SUB-3`
- `AUTHPOL-SRV-1`
- `REV-1`
- `ARC-1`
- `MERGE-1`

## 经验 / 教训摘要
- 半中心 authority 不需要把 `authority.node_id` 中心化到持久配置；runtime lease 足以表达“当前有效 authority 是 root”。
- 多 hop auth assist 链路必须以 edge hub 作为响应终点，中间节点只负责转发；否则 pending / binding 很容易被错误写到中间 hub。
- `authority_policy_sync` 只能是运行时约束和退化控制，不能被设计成 bootstrap 的前置门禁，否则 parent bootstrap register 会被时序问题反噬。

## 可复用排查线索
- 症状：
  - 非 root 节点断链后仍允许新 register
  - 中间 hub 收到 assist 回包后，本地 child 连接的 `nodeID/deviceID` 被污染
  - 文档仍按“直接父节点就是 authority”解释当前行为
- 触发条件：
  - `auth.authority_mode=semi-central`
  - 多 hop 拓扑下的 admission / login / credential query
  - authority policy 尚未收到、已过期，或者父链断开
- 关键词：
  - `authority_policy_sync`
  - `effective_authority_id`
  - `TargetID`
  - `authority unavailable`
  - `degraded login`
- 快速检查：
  - 看 [auth/routing.go](../../repo/MyFlowHub-SubProto/auth/routing.go) 是否优先走 runtime lease，并在父链在线时保留 fallback
  - 看 [auth/auth_forward.go](../../repo/MyFlowHub-SubProto/auth/auth_forward.go) 是否对 assist action 做 `TargetID` forward
  - 看 [auth/authority_policy_test.go](../../repo/MyFlowHub-SubProto/auth/authority_policy_test.go) 是否覆盖 stale epoch 和 degraded login
  - 看 [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md) 是否已记录半中心语义

## 关键设计决策与权衡
- 采用“root runtime lease + parent-chain fallback”，而不是持久化 `authority.node_id`
  - 优点：避免断链/重启后的 stale authority 配置污染
  - 代价：authority 需要依赖运行时 lease 和父链状态共同判定
- 采用“断链冻结新准入，但保留本地已知身份登录”的半中心方案，而不是强中心冻结全部 auth
  - 优点：网络抖动时已知设备仍可在本地恢复登录
  - 代价：退化期 auth 语义不再完全一致，需要稳定 spec 明确记录
- 不实现 pending 逐级复制或 root 侧聚合 pending 查询
  - 优点：避免把本轮扩成分布式审批状态重构
  - 代价：当前 approve / reject / permit 仍以 authority 本地操作为准

## 测试与验证方式 / 结果
- `MyFlowHub-Proto`: `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-SubProto/auth`: `go test ./... -count=1 -p 1`
  - 环境：`GOWORK=D:/project/MyFlowHub3/.tmp/auth-semi-central-work/go.work`
  - 结果：通过
- `MyFlowHub-Server`: 文档与实现对齐审阅
  - 结果：完成
  - 说明：本轮 Server 仓无运行时代码变更，仅更新稳定 spec / protocol_map 副本

## 潜在影响与回滚方案
- 潜在影响：
  - 仍停留在旧版 Proto / SubProto auth 的下游将无法识别 `authority_policy_sync` 或新的多跳 forwarding 语义
  - 多 hop assist 响应更依赖 `SourceID/TargetID` 语义，旧理解下容易把 root 响应误当作中间节点本地响应
- 回滚方案：
  - 回退 `MyFlowHub-Proto` 的 auth wire 扩展与 protocol map
  - 回退 `MyFlowHub-SubProto/auth` 的 authority policy / forwarding / degraded login 改动
  - 回退 `MyFlowHub-Server/docs/specs/auth.md` 与 `docs/specs/protocol_map.md`

## 子Agent执行轨迹
- 无子 agent
