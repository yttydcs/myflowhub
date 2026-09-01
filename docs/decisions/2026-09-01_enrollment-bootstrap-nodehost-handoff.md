# 2026-09-01 Enrollment bootstrap → NodeHost handoff

## Status

Accepted.

## Context

authority Profile 在获得 Admission Grant 前没有 Node ID，必须使用 MFHE pre-auth Enrollment；但 Grant 持久化后已经具备完整 Node identity、直接父锚点和 Authority provenance。继续让 Enrollment binding 拥有普通 Node runtime，会让 Desktop 同时存在 Host-owned 与 binding-owned 两套连接、关闭和重试语义，并导致重复 Connect 返回生命周期错误。

## Decision

- Enrollment 是 Grant 前的窄 bootstrap，只暴露状态、Enroll 和 Close，不提供普通 Resource operation 或 Parent runtime。
- Grant 与 Device key 在受保护 credential 中原子保存；该 credential 是 authority Profile 身份、直接父锚点和 Authority provenance 的唯一事实源。
- Grant 保存后必须先关闭 bootstrap，再以只读 `NodeCredentialSource` 创建中立 NodeHost。
- NodeHost 从 credential 派生 Node identity 和直接父信任；Profile 中的兼容缓存只能为空或完全匹配，Authority provenance 不自动产生 Resource 权限。
- Legacy NodeID/IdentityStore 模式保持兼容；旧 runtime-owning binding API 暂时保留给未迁移调用方，但 Desktop 不再使用其 post-Grant Start 路径。
- missing、device、pending、损坏、签名不匹配或缓存冲突均 fail closed，不生成替代身份，不创建第二份 identity 文件，也不进行网络 Start。

## Consequences

- Desktop 的 Legacy 与 authority Profile 在活动状态下具有相同 ownership：一个 Parent-only NodeHost 和一个 attached Client。
- 已 enrolled Profile 重启不再需要 Permit、TOFU 或 Authority 在线；重复 Connect 复用 Host supervisor。
- Grant durable 而 Host 创建失败时，下次直接重试 Host，不重新 Enrollment。
- credential source 与 Host 配置需要专门的 mismatch、只读、并发和无重复身份回归门禁。

## Compatibility

MFHE、MFH4、集中式 Authority、Permit/Pending、DPAPI、Desktop settings v2 与 Legacy Profile 均不改变。Android/Embedded Enrollment 迁移和旧 owning API 的删除需要后续独立决策。

## Related

- [通用 NodeHost、非 owning SDK Client 与薄平台适配](2026-08-30_generic-node-host-and-non-owning-sdk-client.md)
- [集中式 Admission Authority](2026-08-30_centralized-admission-authority.md)
- [NodeHost Runtime](../specs/node-host-runtime.md)
- [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md)
- [Desktop Profile 入口](../specs/desktop-profile-entry.md)
