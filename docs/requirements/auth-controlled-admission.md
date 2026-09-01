# Auth Controlled Admission

## Goal

为 MyFlowHub 权威节点树提供受控首次准入：未知设备不能通过普通 Join 自行声明正式身份；成功注册前没有 Node ID，成功后由单一逻辑 Admission Authority 下发签名身份。

## Scope

### Must

- 支持预先签发 Permit 后直接注册。
- 支持没有 Permit 时进入 Pending，并由受权管理者批准或拒绝。
- Permit 与 Pending 必须进入同一 Grant、Node ID 分配和持久化流程。
- 全树只有一个逻辑 Admission Authority 负责 Permit、Pending、Enrollment、撤销、Node ID 与 tombstone。
- 直接父节点负责链路挑战、连接控制和 Grant 缓存；管理操作可从任意受权节点沿权威树路由到 Authority。
- Authority、父节点或所需上游不可达时，新准入必须 fail closed，不得回退为本地随机身份。
- 已注册节点继续使用既有 MFH4 Join，普通重连不应实时依赖 Authority。
- Enrollment client 只拥有获得并原子保存 Grant 的 bootstrap 生命周期；Grant 持久化后必须关闭 bootstrap，并把同一受保护凭据交给普通 NodeHost，不得保留第二套 post-Grant runtime。
- 迁移期保留旧版显式 Node ID、父节点公钥与 `ProvisioningPermitV1` 路径。

### Out Of Scope

- 多个独立 Authority 之间的 Permit 合并、Node ID 去重、CRDT 或共识复制。
- 可转移的离线准入 ticket、共享长期 join key、reparent 和 rekey。
- 新的 approver 节点类型或隐式角色权限。
- 本轮迁移 Android、C、MicroPython 与 ESP32 首次注册协议。

## User Scenarios

- 运维人员为一个已知设备公钥指纹签发短期、一次性 Permit，使其连接指定父节点或授权子树时直接注册。
- 无 GUI 设备直接连接父节点，完成 proof-of-possession 后进入 Pending；管理员随后从统一界面批准。
- 管理节点 A 为目标父节点 B 发起 Permit 签发；命令由 Authority 统一执行和签名，A、B 不维护需要合并的独立集合。
- 管理员列出全部 Permit、Pending 和 Enrollment，并执行撤销；同一事实源保证列表和操作一致。
- 已注册子节点在 Authority 暂时不可达时，仍可凭父节点缓存的 Grant 完成常规 Join。

## Functional Requirements

- Device 首次启动只生成并持久化密钥；在获得 Grant 前不得创建正式 Node ID 或进入正常路由树。
- Enrollment 使用独立 pre-auth frame family，不得用 Source/Target 为零的普通 MFH4 Envelope 绕过 Node ID 校验。
- 父节点必须先提供有时限的一次性 challenge，并验证设备对会话上下文的签名，再向 Authority 提交。
- Permit 至少绑定设备公钥指纹、目标父节点或子树、admission profile、有效期、一次性状态与 Authority 签名。
- 无 Permit 的有效申请必须获得稳定 request ID，并支持列出、批准、拒绝、过期与幂等查询。
- Approve 与 Permit 消费必须调用同一原子 Grant 流程。
- Node ID 只能由 Authority 使用密码学安全随机源生成，为非零正 63 位整数，并与 active、revoked、tombstone 唯一索引检查。
- Authority 必须原子提交 Node ID、Enrollment、Pending 终态以及 Permit 消费状态；重试同一请求必须返回同一 Grant。
- Grant 必须绑定 Node ID、设备公钥、直接父节点、admission profile、Authority epoch，并由 Authority 签名。
- admission profile 不得自动赋予业务 Resource 权限；访问仍受显式 policy 控制。
- Permit、Pending、Enrollment 和撤销管理动作必须分别受显式权限点保护；状态变更审计摘要至少记录操作者 Node ID、目标 ID 和结果状态。
- 撤销 Enrollment 必须保留 Node ID tombstone，并使对应父节点信任与活动会话明确失效。

## Trust Bootstrap

- Permit 路径通过 Authority 签名和目标父节点绑定锚定预期父节点。
- 没有预置锚的审批路径采用显式 TOFU：客户端在首次成功时固定父节点/Authority 指纹；无 GUI 部署可以预置指纹关闭 TOFU。
- 新建 Profile 不以手工输入父节点公钥为常规前提；旧 Profile 的固定公钥行为保持兼容。

## Abuse And Capacity Controls

- Pending 只在 challenge 与 proof 验证成功后创建。
- Authority 和父节点必须设置总量、每来源、每父节点、每设备的请求与速率上限。
- 恶意设备可以申请 Pending，但不能自行获得 Node ID；只有消费有效 Permit 或管理员批准时才分配。活动 Pending、每父节点 Pending 与保留 request 记录总数都必须有硬上限。
- 过期和拒绝记录按明确保留策略清理，已分配 Node ID tombstone 不得因常规清理被重新使用。

## Non-functional Requirements

- 所有拒绝、pending、Permit 失败和 Authority 不可达路径必须返回明确且可操作的状态。
- Authority 状态、签名密钥与客户端 Grant 必须版本化、原子持久化；损坏或密钥不匹配不得静默重建。
- 客户端 Grant 是已注册身份、父节点锚点与 Authority provenance 的唯一事实源；兼容 Profile 缓存只能校验或展示，不能覆盖 Grant，也不能触发新身份生成。
- 敏感材料不得写入日志、settings 或审计事件；Permit 只在必要的单次 Enrollment 中传输。
- 新协议设有严格帧大小、字段长度、过期时间和签名校验。
- 默认升级不得破坏已有持久身份或旧客户端 Join。

## Acceptance

- 全新客户端无需填写 Node ID；未获批前本地和服务端都没有为它创建正式 Node ID。
- 有效 Permit 可让绑定设备直接得到一个且仅一个 Grant；重试不会再次分配 ID。
- 无 Permit 连接产生一个可在统一管理界面列出、批准或拒绝的 Pending；批准后客户端能持久化身份并完成普通 Join。
- 管理节点 A 可以为父节点 B 签发和撤销 Permit，但 Authority 列表中只有一份权威记录。
- 并发注册、随机 ID 冲突和重复提交不会产生重复 Node ID 或重复 Enrollment。
- 撤销 Enrollment 后，Node ID 不再被分配，对应 Join 与活动会话失效。
- Authority 不可达时新准入失败；已有缓存 Grant 的节点仍可按既有 Join 规则重连。
- Desktop authority Profile 在 Grant 后与 Legacy Profile 一样由 Parent-only NodeHost 运行；重复 Connect 复用 Host supervisor，不进入 owning Enrollment runtime。
- 旧 Desktop/Profile、Android 与 Embedded 显式身份路径通过回归测试。

## Canonical Design

详细协议、状态和资源契约见 [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md)。本需求取代旧多仓时期的 `device_id`、local-authority bootstrap epoch 与开放注册兼容语义。

## Related Changes

- [Node Enrollment 与集中式 Admission Authority](../change/2026-08-30_node-enrollment-admission-authority.md)
