# Hub

## 按深度查询拓扑

已有 topology Resource 增加 children/subtree，复用维护中的本地子树和周期快照；children 仅允许直接孩子。旧 read/subscribe 全量语义保持。新能力独立授权，不自动改写已有 policy；见[拓扑发现合同](../specs/topology-discovery.md)。

## Purpose

Hub 是权威节点树的根或中继宿主，负责持久身份、子节点准入、路由、策略裁决和内建管理资源。它不是全局业务模块容器。

## Observable behavior

- 首次启动原子创建身份与版本化配置，后续启动保留 NodeID 与信任状态。
- 可同时监听多个已配置 Transport；链路差异不泄漏到资源和权限层。
- 通过 catalog、topology、health、config Variables 暴露状态；准入与配置仍提供 Commands，权限通过 Definitions、Bindings、exact Grants 三个 Collections 管理并保留 legacy grant/revoke aliases。
- 断连、重连、reparent 和 revoke 会使旧树边、旧策略世代与相关订阅明确失效。

## Permissions

- 只有当前直接父节点可以下发父控子 control；普通下级请求必须在权威路径上裁决。
- 准入许可必须绑定父身份、子身份或一次性挑战、角色、有效期和使用状态。
- 管理 Commands 使用独立权限点并写入不含敏感材料的审计事件。
- Policy Definition 表达可复用规则，Binding 绑定精确 Subject 与 owner 作用域；legacy `system/policy/grant/revoke` 继续修改精确元组。
- 所有在线 policy mutation 都二次要求真实调用 Subject 拥有本 Authority-domain 的 superadmin Binding；仅获得管理 Resource capability 不足以提权。

## 首次引导

新版本首次准入以 [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md) 为准：设备注册前只有密钥指纹，没有 Node ID；可使用 Authority 签发的 Permit 直接注册，或在父节点验证 proof 后进入 Pending 审批。下面的显式 Node ID 与旧 Permit 命令在迁移期继续作为兼容流程。

Hub 预置的是未绑定、不可变的 `superadmin` Definition，不预置超级用户，也不会为了方便而关闭默认拒绝。新的 Authority 注册按以下顺序完成：

1. 保持 Hub 停止，使用 `mfh-hub -identity` 读取持久 NodeID 与 raw-base64 Ed25519 公钥；
2. 子设备运行 `mfh-admin -op identity` 准备设备公钥但不生成 Node ID；Authority Hub 可用 `-issue-public-key`、`-issue-target-id`、`-issue-role` 签发绑定设备指纹和目标父节点的短期 Permit，也可让设备无 Permit 连接后进入 Pending；
3. 设备运行 `mfh-admin -op enroll`，Permit 路径直接获得 Grant，Pending 路径等待集中审批后用同一命令重试；
4. Grant 持久化后再使用 `-op snapshot` / `-op invoke`，系统根据已分配 Node ID 走普通 Join；对 `system/admission/*` 的命令默认路由到已绑定的 Authority，也可用 `-owner-id` 显式指定资源 owner；
5. Hub 停止时可用 `-policy bind -subject <id> -definition superadmin -policy-scope authority-domain` 显式建立第一个管理员；常规主体继续通过窄 Definition/Binding 或 exact Grant 获得权限。旧 `-issue-node-id` 只保留给 Legacy Join。

中继 Hub 使用 `-admission-authority-id` 和 `-admission-authority-key` 固定同一个远端 Authority；父节点必须在 Authority 上获得 `system/admission/submit-enrollment` 的 `invoke` 权限，Authority 也必须在该父节点上获得 `system/admission/apply-revocation` 的 `invoke` 权限。Authority 不可达时，新注册明确失败，已有 Grant 的节点仍使用父节点缓存的直接信任重连。

离线命令直接原子更新同一持久状态，支持 policy show/bind/revoke-binding 与兼容 exact grant/revoke，必须在 Hub 停止时运行；它不是绕过在线权限的并发旁路。Binding 操作输出 ID 和 generation 供审计/回滚。详见 [Scoped Policy Authorization](../specs/scoped-policy-authorization.md)。

## Non-goals

- 不提供旧 SubProto handler、兼容 wire bridge 或隐式 defaultset。
- 不根据物理 Transport 类型决定权限。

## Acceptance

重启身份稳定；多 listener 可用；allow/deny/revoke 可重复验证；损坏配置显式失败；优雅关闭不遗留会话或 pending 请求。

## Related Docs

- [Scoped Policy Authorization](../requirements/scoped-policy-authorization.md)
- [Scoped Policy Authorization Spec](../specs/scoped-policy-authorization.md)
- [作用域 Binding ADR](../decisions/2026-09-01_scoped-policy-bindings-over-product-privilege.md)
- [作用域策略定义与 Authority 权限绑定](../change/2026-09-01_scoped-policy-authority.md)
- [策略状态迁移与真实授权证明](../lessons/scoped-policy-state-migration-and-live-proof.md)
