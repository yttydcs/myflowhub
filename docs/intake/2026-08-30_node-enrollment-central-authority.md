# 2026-08-30 Node Enrollment 与集中式准入 Authority

## Source

用户在检查当前登录与准入流程时指出：首次连接不应要求客户端手工填写 Node ID 或父节点公钥；Node ID 应在成功注册时由服务端下发。准入需要同时覆盖两种路径：预先签发 Permit 后直接进入，以及没有 Permit 时进入待审批队列。

管理侧还需要统一列出、签发、撤销 Permit 与处理待审批请求；在分布式节点树中，管理节点 A 可以为目标父节点 B 发起签发，但不应在各节点维护并合并彼此独立的 Permit 集合，也不能依赖父节点各自随机生成 Node ID 后再去重。

## Confirmed Requirements

- 未成功注册的设备只有本地密钥身份，没有正式 Node ID。
- Permit 直入和 Pending 审批是同一 Enroll 状态机的两条分支。
- 直接父节点控制链路接入与对子节点的本地信任，但全树只有一个逻辑 Admission Authority 管理准入状态。
- 任意有权限的管理节点都可以发起管理操作；操作沿权威树路由到 Admission Authority 执行和签名。
- Permit 至少绑定子设备公钥指纹、目标父节点或子树、准入档案、有效期与一次性状态。
- 已注册节点后续使用现有 Join；常规重连不要求每次访问 Authority。
- 迁移期保留旧版显式 Node ID、父节点公钥与 Join Permit 流程。

## Non-goals

- 本阶段不实现多个独立 Authority、CRDT 合并或跨 Authority 高可用共识。
- 本阶段不实现离线可转移 ticket、reparent 或 rekey。
- 本阶段不迁移 Android、C、MicroPython 与 ESP32 的首次注册协议；它们继续走兼容 Join。

## Routed Documents

- [受控准入需求](../requirements/auth-controlled-admission.md)
- [Node Enrollment 与 Admission Authority 规范](../specs/node-enrollment-and-admission-authority.md)
- [集中式 Admission Authority 决策](../decisions/2026-08-30_centralized-admission-authority.md)
- [Hub](../features/hub.md)
- [Desktop](../features/desktop.md)
- [实施与验证归档](../change/2026-08-30_node-enrollment-admission-authority.md)
