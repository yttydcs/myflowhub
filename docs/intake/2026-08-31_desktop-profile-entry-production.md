# 2026-08-31 Desktop Profile 入口生产化

## Source

- 来源：用户在确认 `design-demos/login-profile-flow.html` 双入口原型后，要求评估并规划应用到生产 Desktop。
- 日期：2026-08-31。
- 原型：`design-demos/login-profile-flow.html`。
- 生产入口：`apps/desktop/frontend/src/components/LoginScreen.tsx`。

## Request Text / Source-preserving Summary

用户要求把登录界面明确分成“使用现有 Profile”和“首次连接”两类，不再用 01/02/03 编号解释从上到下填写的顺序，并以简洁、低副标题密度为优先。现有 Profile 应先选择后连接；只有首次连接才展示 Profile、父节点地址、审批或 Permit 等准入输入。

生产化讨论进一步确认：界面不能根据 `node_id` 是否为空把所有未注册 Profile 都猜成“等待审批”，而应读取本机受保护 Enrollment credential 中已经存在的 `device`、`pending`、`enrolled` 和稳定 request ID。Profile ID 是本地存储标识，不是 Node ID，应由客户端生成合法、稳定、无碰撞的 ASCII 值，常规用户不必手工填写。

## Confirmed Requirements

- 有已保存 Profile 时默认显示“使用现有 Profile”；没有 Profile 时默认进入“首次连接”。
- Existing Profile 必须区分 Legacy、需要准备、Device、Pending、Enrolled 和凭据异常，不得只靠 `profile.node_id` 推断。
- Pending 使用同一 Profile 和稳定 request ID 检查审批；本机已保存的父节点与 Authority 观察值继续作为重试信任锚，不重复要求 TOFU。
- Enrolled Profile 的常规动作只是连接父节点；Device Profile 继续首次连接；Legacy 保持兼容入口。
- 首次连接默认只展示 Profile 名称、父节点地址、审批/Permit、自动连接和必要动作；Node ID、公钥 pin、Legacy 字段留在结果区或高级设置。
- Permit 路径可以生成/复用设备身份并复制本机公钥；私钥和 Permit 不进入 settings、日志或 UI preference。
- Profile ID 在进入首次连接时生成一次，符合后端 `[a-z0-9][a-z0-9._-]{0,63}` 约束，和名称编辑解耦，并对现有 ID 做碰撞检查。
- 增加“返回 Profile 选择”能力：清除 active Profile 并关闭活动 client，但保留 Profile、受保护凭据和 Views；它不等同于删除 Profile。
- 已激活 Profile 的下次启动与 `auto_connect` 既有行为保持兼容；本轮不强制每次启动都经过 Profile 选择页。
- UI 使用真实 Wails/React 数据和现有 BrandMark/design tokens，不复制原型中的硬编码 Profile、Toast 或命令式 DOM。

## Non-goals

- 不改变 MFHE/MFH4 wire、Authority、Permit、Node ID 分配、撤销或父节点控制语义。
- 不升级 persisted settings version，也不在 settings 中复制 Enrollment 状态。
- 不实现人类账号、密码或云端会话系统。
- 不迁移 Android、C、MicroPython 或 ESP32 的首次注册 UI。
- 不改变 headless Enrollment/管理流程。
- 不推送、发布或部署。

## Routed Docs

- [Desktop](../features/desktop.md)
- [Desktop 资源工作区需求](../requirements/desktop-resource-workspace.md)
- [受控准入需求](../requirements/auth-controlled-admission.md)
- [Desktop Profile 入口规格](../specs/desktop-profile-entry.md)
- [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md)
- [集中式 Admission Authority](../decisions/2026-08-30_centralized-admission-authority.md)
- [Desktop Binding 重连与准入诊断](../lessons/desktop-binding-reconnect-and-admission-diagnostics.md)
- [Desktop Profile 入口生产化归档](../change/2026-08-31_desktop-profile-entry-production.md)

## Open Questions

- Blocking: none。
- Deferred: 是否让未退出的 active Profile 在每次启动都重新显示选择器；本轮保留现有持久激活和自动连接语义。
- Deferred: Android/Embedded 原生 Enrollment UI 与多 Authority 高可用，均需独立批准。
