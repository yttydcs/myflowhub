# Node Enrollment 与 Admission Authority

## Scope

本文定义节点首次注册、Permit、人工审批、Node ID 分配、撤销以及与既有 Join 的兼容边界。它不改变已注册节点的 MFH4 Resource、Subscription、Operation 或路由协议。

## Authority Boundary

一棵权威节点树只有一个逻辑 Admission Authority。Authority 是 Permit、Pending Request、Enrollment、撤销记录、已分配 Node ID 和 tombstone 的唯一写入者。

管理节点只负责鉴权、展示和转发管理命令。节点 A 为父节点 B 签发 Permit 时，A 将命令路由到 Authority；Permit 由 Authority 签名并写入同一状态库，所以不存在 A、B 两份 Permit 集合的合并问题。

直接父节点仍拥有链路控制权：它可以拒绝连接、限流或断开子节点，并在收到 Authority Grant 后缓存子节点身份。`admission_profile` 只记录注册类别，不隐式授予业务资源权限；资源权限仍由显式 policy 决定。

## Identity States

1. **Device**：客户端首次启动只生成 Ed25519 密钥对和稳定设备公钥指纹，不具有 Node ID。
2. **Pending**：父节点完成链路挑战后向 Authority 提交申请；Authority 返回稳定 request ID。重复提交同一设备、目标父节点和挑战上下文必须幂等。
3. **Enrolled**：Authority 批准申请或消费有效 Permit，分配随机正 63 位 Node ID，并返回签名 Grant。
4. **Joined**：客户端原子持久化 Grant、关闭 Enrollment bootstrap，再由普通 NodeHost 读取同一 credential 并使用既有 MFH4 Join。父节点用 Grant 缓存建立的信任校验 Node ID、公钥和签名。
5. **Revoked**：Authority 保留 enrollment 与 Node ID tombstone。被撤销 ID 不得重新分配；相关父节点信任和活动会话必须失效。

未成功获得并原子持久化 Grant 的客户端仍处于 Device/Pending 状态，不得自行选择 Node ID。

## Enrollment Wire

首次注册使用独立的 pre-auth frame family，magic 为 `MFHE`。它不是 Source/Target 为零的普通 MFH4 Envelope；MFH4 对非零 Node ID 的约束保持不变。

Enrollment 消息顺序为：

1. `ClientInit`：协议版本、设备公钥、目标父节点提示、可选 Permit。
2. `ServerChallenge`：父节点身份、公钥、一次性 nonce、过期时间和信任提示。
3. `ClientProof`：设备私钥对 challenge 与会话上下文的签名。
4. `EnrollResult`：`granted`、`pending` 或显式错误；Grant 包含 Authority 签名的 Node ID、公钥、父节点、档案和世代。

父节点验证 proof 后才可提交 Authority。消息设有严格长度上限，未知版本、过期 challenge、签名不匹配和字段不一致均 fail closed。

首版允许 MFHE 完成后关闭连接并以新 Node ID 重新建立 MFH4 Join；是否在原连接上切换协议是后续优化，不影响状态语义。

## Permit Path

Permit 由 Authority 签发，至少包含：

- 唯一 Permit ID；
- 子设备公钥指纹；
- 目标直接父节点或授权子树；
- admission profile；
- 签发、到期时间与 Authority epoch；
- 一次性消费状态和 Authority 签名。

有效 Permit 可以直接产生 Grant。消费与 Grant 写入必须是同一持久化事务；相同 Enrollment 重试返回同一 Grant，不得分配第二个 Node ID。Permit 撤销、过期、绑定不匹配或重复用于其他设备时明确失败。

## Approval Path

没有 Permit 的合法 proof 进入 Pending。管理界面可以列出请求的设备指纹、目标父节点、来源摘要、创建时间和状态，并执行 approve/reject。

Approve 调用 Authority 的同一 Grant 流程；Reject 保存终态和理由。客户端用 request ID 重试或查询，Authority 不因轮询创建新请求。请求按“设备公钥指纹 + 直接父节点”去重，设置审批 TTL、每父节点配额、全局容量和握手并发上限，避免恶意节点无限制造 Node ID 或未决记录。

## Node ID Allocation

只有 Authority 可以分配 Node ID。ID 为密码学安全随机的正 63 位整数；`0` 保留。Authority 在持久唯一索引中检查 active、revoked 与 tombstone，冲突时重新抽取。分配、Enrollment、Permit 消费或 Pending 终结使用一次原子提交。

父节点不能本地预留或签发 Node ID；因此无需跨父节点合并或事后去重。

## Management Resources

Authority 通过受策略保护的管理资源提供：

- `system/admission/status`
- `system/admission/list-permits`
- `system/admission/issue`
- `system/admission/revoke-permit`
- `system/admission/list-requests`
- `system/admission/approve`
- `system/admission/reject`
- `system/admission/list-enrollments`
- `system/admission/revoke-enrollment`
- `system/admission/submit-enrollment`（仅供父节点或等价受权主体）

命令包含 request/idempotency ID，响应返回 Authority epoch 与稳定对象 ID。管理调用无论从哪一节点发起，最终都由同一 Authority 校验和提交。

## Persistence And Recovery

Authority 状态使用版本化文件和原子 replace 提交，损坏、未知版本或签名密钥不匹配必须阻止 Authority 启动，不能静默重建。Grant 先在 Authority 持久化，再返回父节点和客户端。客户端必须原子保存 Device private key、Grant 与 Node ID 后才进入已注册状态。该 credential 是 Node identity、直接父锚点与 Authority provenance 的唯一事实源；NodeHost 从中派生运行身份，不得另外生成、迁移或覆盖一份 identity。

Enrollment bootstrap 只暴露状态检查、Enroll 和 Close。它不能 Catalog、Operate、Subscribe 或启动普通 Parent runtime。Grant 保存后必须先关闭 bootstrap，再创建 NodeHost；若 Host 创建失败，Grant 仍保持 durable，下一次打开直接从 credential 重试 Host，不重新消费 Permit 或再次 Enrollment。

父节点缓存 Authority Grant 以支持 Authority 暂时不可用时的已注册 Join。新 Enroll、审批、签发、消费与撤销在 Authority 不可达时 fail closed；不得降级成本地随机 ID 或无签名信任。

## Parent Trust Bootstrap

- **Permit path**：Permit 中的 Authority 签名和目标父节点绑定用于锚定预期父节点；客户端仍校验 challenge 中的父身份与 Grant 一致。
- **Approval path**：无预置锚时默认是显式 TOFU。客户端展示或记录父节点指纹，首次成功 Grant 后固定；无 GUI 客户端可以通过配置预置父节点或 Authority 指纹以关闭 TOFU。
- 客户端不再把“手工输入父节点公钥”作为常规注册必填项，但兼容 Profile 保留旧行为。

## Security And Limits

- proof-of-possession 保证申请者持有设备私钥；Permit 泄漏不能替换已绑定设备密钥。
- Pending 创建前完成 challenge 校验，并实施活动 Pending 总量、每父节点和每设备去重；Authority 还限制保留 request 记录总量，任何配额拒绝都不会分配 Node ID。
- Permit 签发/撤销、request 提交/审批/拒绝和 Enrollment 撤销均写入审计事件；事件包含操作者 Node ID、管理资源、目标 ID 与结果状态，但不含私钥、设备公钥、完整 Permit 或敏感 payload。
- 父节点的链路控制不等于 Authority 管理权限，也不等于资源访问权限。

## Compatibility

迁移期间同一 listener 同时接受 `MFHE` Enrollment 与既有 `MFH4` Join。已有持久 Node ID、父公钥和旧版 `ProvisioningPermitV1` 的 Profile 继续工作。Android 与 Embedded 旧实现现已退役；保留的 Desktop/CLI Legacy 身份路径仍依赖 MFH4 Join，删除该兼容入口必须由后续单独迁移决策批准。

## Related Documents

- [受控准入需求](../requirements/auth-controlled-admission.md)
- [权威节点树与链路架构](node-tree-link-resource-architecture.md)
- [运行生命周期](operational-lifecycle.md)
- [集中式 Admission Authority 决策](../decisions/2026-08-30_centralized-admission-authority.md)
- [NodeHost、Enrollment 与 Profile 生命周期收敛](../change/2026-09-01_nodehost-enrollment-profile-convergence.md)
- [实施与验证归档](../change/2026-08-30_node-enrollment-admission-authority.md)
