# 集中式 Admission Authority

## Status

Accepted for implementation by the approved 2026-08-30 workflow plan.

## Context

首次注册需要同时支持预签 Permit 与人工审批，还要允许在任意管理节点查看和操作全树准入。如果每个父节点都能本地签发 Permit 和 Node ID，就必须处理状态同步、Permit 合并、ID 冲突、撤销传播和分区一致性；这超出了当前单一权威节点树的需求。

另一方面，直接父节点确实拥有链路接纳和断开的最终控制权，而且已注册节点的常规重连不应依赖一个实时在线的全局服务。

## Options Considered

1. **每个父节点独立 Authority**：本地简单，但需要分布式 ID 协调、Permit 合并和撤销一致性。
2. **客户端随机 Node ID**：减少一次分配调用，但不能表达“未获批就没有正式 ID”，仍需解决冲突与滥用。
3. **单一逻辑 Admission Authority**：统一写入，管理操作可路由，父节点保留链路控制并缓存 Grant。
4. **多个共识 Authority**：可提升可用性，但引入成员关系、选主、日志复制和灾难恢复等本阶段不需要的复杂度。

## Decision

选择单一逻辑 Admission Authority。

- Authority 独占 Permit、Pending、Enrollment、撤销和 Node ID 分配状态。
- Node ID 由 Authority 随机生成并在持久唯一索引中检查；客户端和普通父节点不得指定。
- 管理节点 A 可以为目标父节点 B 发起签发，但 A 只转发受权命令，真正的 Permit 由 Authority 写入和签名。
- 父节点验证 pre-auth proof、提交申请、接收 Grant，并缓存 Grant 建立本地信任；它仍可拒绝或断开任何直接子连接。
- 已注册 Join 使用缓存信任，不在每次连接时同步访问 Authority。
- Authority 不可用时允许已有身份重连，但新准入和准入状态变更 fail closed。

## Consequences

### Positive

- Permit 列表、Pending 列表、撤销和 Node ID 唯一性都有明确事实源。
- 无需在父节点间合并 Permit 或事后修复 Node ID 冲突。
- GUI 与无 GUI 管理工具共享同一组可路由资源。
- 父节点离线缓存让普通重连不依赖 Authority 的实时可用性。

### Negative

- 新注册和管理操作依赖 Authority 可达。
- Authority 状态与签名密钥需要严格备份、恢复和损坏检测。
- 真正的多 Authority 高可用需要未来单独设计共识与迁移协议，不能伪装成本地合并。

## Relationship

本决策补充 [统一权威节点树与可插拔链路](2026-08-27_authoritative-node-tree-and-pluggable-links.md)：节点树仍负责路径与直接父控子，Admission Authority 专门负责首次身份授予和全局准入状态。

## Related Changes

- [Node Enrollment 与集中式 Admission Authority](../change/2026-08-30_node-enrollment-admission-authority.md)
