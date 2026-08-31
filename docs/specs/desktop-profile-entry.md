# Desktop Profile 入口

## 定位

Desktop 的“登录”是选择或建立本机 Profile 身份，不是人类账号/密码会话。Profile ID 只是本地隔离目录与设置记录的稳定标识；Authority 模式的 Node ID 由 Admission Authority 在 Enrollment Grant 中下发，客户端不得自行指定。

## 入口模型

未激活 Profile 时，界面只有两个顶层入口：

- **使用现有 Profile**：列出已经保存的 Profile，并按受保护凭据的真实状态提供连接、检查审批或继续首次连接动作。
- **首次连接**：收集 Profile 名称、父节点地址和准入方式；普通流程不要求用户输入 Profile ID、Node ID 或公钥 pin。

有已保存 Profile 时默认进入“使用现有 Profile”，没有 Profile 时默认进入“首次连接”。本轮保持既有兼容行为：若 `active_profile_id` 仍存在，应用启动后直接进入该 Profile；只有用户显式“返回 Profile 选择”后，下次启动才停留在选择器。

## 状态真相

Authority Profile 的生命周期唯一来自该 Profile 的受保护 `EnrollmentCredential`：

| 状态 | 含义 | 默认动作 |
| --- | --- | --- |
| `missing` | 尚无本机 Enrollment 凭据 | 继续首次连接 |
| `device` | 已生成设备身份，尚未观察父节点/Authority | 继续首次连接 |
| `pending` | 已提交申请并固定首次观察 | 检查审批并连接 |
| `enrolled` | 已保存 Authority 签名 Grant | 连接父节点 |
| `error` | 凭据损坏、不可读或与 Profile 身份冲突 | 阻止连接并给出恢复提示 |
| `legacy` | version 2 Legacy Profile | 兼容连接 |

读取 Profile 列表必须使用只读检查路径。它可以读取并验证已有凭据，但不得创建目录、密钥、request ID、Grant 或写入文件；损坏状态不得静默重置成新的 Device identity。

Desktop 只向前端投影 allowlist 字段：Profile ID、状态、可选 request ID、Node ID、父 Node ID、Authority Node ID 和可操作错误。不得输出私钥、Permit、Grant 签名、完整凭据或公钥正文。

`settings.json` 不复制 `device/pending/enrolled` 状态。已 Enrollment 但尚未来得及把 Node ID 回写 Profile 的凭据仍显示为 `enrolled`；若 Profile 已有非空 Node ID 且与 Grant 冲突，则显示 `error`。

## 首次连接

- 客户端进入首次连接时生成一次符合 `[a-z0-9][a-z0-9._-]{0,63}` 的随机 Profile ID，并对现有 ID 做有界碰撞重试。名称或地址变化不得改变该 ID。
- 审批模式在没有预置或已观察信任锚时要求一次显式 TOFU；进入 `pending` 后重试复用同一 request ID 和信任观察，不再次要求 TOFU。
- Permit 模式可先准备/复用受保护设备身份并展示可复制的设备公钥，然后只在本次 `LoginJSON` 调用中传入 Permit。Permit 不持久化。
- 高级设置可承载预置父节点/Authority pin 与 Legacy 兼容字段，但普通 Authority 流程不要求它们。

## 返回 Profile 选择

“返回 Profile 选择”与“断开连接”和“删除 Profile”不同：

- 清空并持久化 `active_profile_id`；
- 关闭当前 client、subscription 和 session；
- 保留 Profile、受保护身份、Enrollment 状态、runtime state、Views 和 UI preference；
- 前端在失败或部分清理后重新读取权威 Settings，不能猜测最终状态。

工作区存在未保存改动时，前端必须先走现有的放弃更改确认。

## 兼容与安全边界

- 不改变 MFHE/MFH4、Authority、Permit、Node ID 分配、撤销或 headless Enrollment 语义。
- 不升级 Desktop settings version，不新增明文 secret 存储。
- Windows 继续使用当前用户作用域 DPAPI；无系统保护 backend 的平台继续明确使用 session-only identity。
- 状态读取按现有最多 64 个 Profile 有界执行，只在启动或 Profile/登录变更后刷新，不做高频轮询。
- Profile 选择、Tab、表单、错误和动作必须可通过键盘与辅助技术操作，并同时支持浅色/深色主题。

## 相关文档

- [Node Enrollment 与 Admission Authority](node-enrollment-and-admission-authority.md)
- [Desktop Resource Workspace v3](desktop-resource-workspace-v3.md)
- [Desktop 资源工作区需求](../requirements/desktop-resource-workspace.md)
- [Desktop 当前行为](../features/desktop.md)
- [Desktop Profile 入口生产化 intake](../intake/2026-08-31_desktop-profile-entry-production.md)
- [Credential 状态检查必须是只读路径](../lessons/credential-status-inspection-must-be-read-only.md)
- [Desktop Profile 入口生产化归档](../change/2026-08-31_desktop-profile-entry-production.md)
