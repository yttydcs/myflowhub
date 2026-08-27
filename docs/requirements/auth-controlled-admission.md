# Auth Controlled Admission

## Goal

为 MyFlowHub 网络引入受控准入机制，避免节点仅凭 `register` 即自动加入网络并获得正式身份。

## Scope

### Must

- 支持“普通注册进入待审批”流程。
- 支持“一次性、角色绑定、至少绑定 `device_id` 的 permit”直接准入流程。
- 支持 local authority 冷启动场景下的“一次性首个注册 bootstrap”例外：仅对显式配置的 `device_id` 生效，可选绑定 `pubkey`，消费后恢复正常审批流。
- 普通注册在获批前不得获得正式 `node_id`，也不得被视为已入网节点。
- 审批、拒绝、permit 签发与撤销通过 `auth` 权限动作执行，而不是通过新的节点类型建模。
- 父链自动上联场景在启用审批模式时必须可通过显式 permit 完成注册。
- 审批通过后，节点应按批准结果成为指定角色，并进入现有权限解析链路。
- authority 选择必须 fail-closed：显式 authority 或已配置父节点不可达时，不得回退为本地 authority。

### Optional

- 后续扩展 permit 绑定 `pubkey`。
- 后续补充审批通知、审计查询和 UI 管理入口。

### Out Of Scope

- 本轮不引入共享长期 join key。
- 本轮不引入新的“approver 节点类型”。
- 本轮不重做完整 token/RBAC 体系。

## User Scenarios

- 运维人员希望默认拒绝未知节点直接入网，先看到申请再决定是否批准。
- 运维人员希望预先给某个明确 `device_id` 下发一个一次性 permit，使其首次注册时直接成为指定角色。
- 运维人员希望在全新 root/authority 冷启动时，为一个预先知道的设备开一个一次性 bootstrap 槽位，使其直接成为首个审批者，之后恢复正常审批。
- 某些已授权节点需要拥有审批或签发 permit 的能力，但这种能力应来自 `auth.*` 权限，而不是来自新的节点类型。
- 子 Hub 作为网络中的一个节点自动上联父 Hub 时，在审批模式下仍应能通过配置的 permit 成功加入网络。

## Functional Requirements

- 当审批模式启用时，普通 `register` 请求必须返回 pending 状态，并记录待处理申请。
- pending 申请必须支持列出、批准、拒绝与过期清理。
- permit 必须至少绑定 `device_id` 与目标角色，并且只能成功消费一次。
- permit 消费成功后，注册节点应立即成为 permit 指定角色。
- first-register bootstrap 必须至少绑定 `device_id`；若配置了 `pubkey`，则 `device_id + pubkey` 必须同时匹配才可生效。
- first-register bootstrap 仅允许在 local authority 上使用；存在显式 authority 或 parent 配置时不得启用。
- first-register bootstrap 必须按 `epoch` 一次性消费，并能通过手工提升 `epoch` 显式重新开启。
- 审批通过后，系统才允许分配正式 `node_id`、创建 binding/trusted 信息并建立路由索引。
- 未获批的 pending 申请不得进入正常 `login`、`up_login`、management tree 等正式网络路径。
- 审批与 permit 相关动作必须受显式 `auth.*` 权限控制。
- 当审批模式关闭时，系统应保持现有开放注册兼容行为。
- 显式配置 `authority.node_id` 时，只能使用该节点作为 authority；不可达时必须显式失败。
- 未配置 `authority.node_id` 但已配置父节点时，直接父节点是唯一 authority；父节点不可达时必须显式失败。
- 仅在既未配置 `authority.node_id` 也未配置父节点时，才允许本地作为 authority。

## Non-functional Requirements

- 所有拒绝、待审批、permit 失败路径必须返回明确错误或状态，不得 silent fail。
- 新增状态应能在重启后恢复，避免因重启丢失审批上下文。
- permit 的校验与消费必须具备并发安全性，避免重复使用。
- first-register bootstrap 的消费状态必须持久化；`auth.disable_persist=true` 时不得启用该能力。
- 默认配置应保持现有部署兼容，不得在未开启审批模式时破坏现有网络。

## Acceptance

- 启用审批模式后，普通注册不会立即产生正式 `node_id` 或正式网络身份。
- 审批通过后，节点能够完成正常登录、被管理接口看到，并拥有对应角色权限。
- 一次性 permit 可让指定 `device_id` 直接加入网络并成为指定角色，且无法重复使用。
- 启用 first-register bootstrap 后，只有命中配置 `device_id`（以及可选 `pubkey`）的首个注册者会直接成为指定角色；同一 `epoch` 消费后其余注册者恢复 `pending` 流程。
- 具备审批/permit 权限的节点可以完成对应 auth 管理动作；无权限节点会被明确拒绝。
- 父链自动上联在审批模式下可通过配置 permit 成功注册；缺失 permit 时会显式失败。
- authority 或 parent 不可达时，register 和依赖上游 authority 的 login 路径会显式失败，不会静默回退为本地 authority。

## Notes

- 该需求源自旧多仓时期；vNext 的 canonical 落点是 `runtime/auth`、`feature/management`、`host/hub` 与 `system/admission/*` / `system/policy/*` 资源。旧 action 名只用于追溯原始需求，不是新增实现依据。
