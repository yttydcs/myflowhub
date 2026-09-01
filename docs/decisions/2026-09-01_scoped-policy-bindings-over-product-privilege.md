# 使用作用域 Policy Binding，而不是产品特权

日期：2026-09-01

## Context

当前 exact policy grants 能表达最小权限，但新 Node/Resource 加入时需要逐项扩展。重构前的 role strings 又缺少当前 Resource owner、Capability、拓扑 epoch 与 generation 语义。与此同时，Desktop 需要持久访问 Metrics，但 Desktop、Metrics 与 Agent Gateway 都应当只是网络中的普通 Node。

## Decision

引入可复用 Policy Definitions 和 Authority-owned persistent Bindings。Definition 只描述有界 Resource/Capability selectors；Binding 把它分配给一个精确 Subject 和 owner/subtree/authority-domain scope。保留 exact Grants 作为兼容和最小权限层。

提供不可变、默认未绑定的 `superadmin` Definition。任何在线 policy mutation 必须由有效的 Authority-domain superadmin Subject 发起；首次 Binding 通过停止 Hub 后的显式离线命令创建。产品名、安装包、首个 Enrollment 与固定 Node ID 都不触发隐式授权。

## Alternatives

- 每次 catalog 变化展开 exact grants：写放大、竞态和未来资源遗漏不可接受。
- 恢复旧 role permission strings：无法保持 owner/topology/generation 语义。
- 让 Desktop 或 Agent Gateway 使用 `AllowAll`：把产品变成网络特权主体，违背普通 Node 模型。
- 只添加 selector、不添加 Definition：重复规则难以审计和一致更新。

## Consequences

- 新资源在已绑定 scope 中自动受规则覆盖；superadmin 的 all 也会覆盖未来 Capability，因此 UI/CLI 必须显式警告。
- 权限事实集中于 Authority policy state，动态成员关系复用唯一权威 Node tree。
- Policy mutation 有严格 bootstrap：全新 Authority 必须停机显式建立第一个管理员 Binding。
- Desktop、CLI、Android 与 Agent Gateway 可以共享 protocol/generic SDK，不共享默认权限。
- phase one 暂不提供 deny、继承、groups、delegation ceiling 或 federation；后续增加时必须定义 precedence 和 Authority 边界。

## Related Docs

- [作用域策略需求](../requirements/scoped-policy-authorization.md)
- [作用域策略技术合同](../specs/scoped-policy-authorization.md)
- [作用域策略变更归档](../change/2026-09-01_scoped-policy-authority.md)
- [策略状态迁移与真实授权证明](../lessons/scoped-policy-state-migration-and-live-proof.md)
