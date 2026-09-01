# 2026-09-01 NodeHost、Enrollment 与 Profile 生命周期收敛

## Source

- 来源：用户在当前 Codex 会话中要求检查最近更新是否互相覆盖，并进一步确认 NodeHost 与登录/注册更新是否存在问题。
- 日期：2026-09-01。
- Canonical repository：`repo/MyFlowHub`。
- 诊断基线：`master@a9eb270`。

## Request / Source-preserving Summary

用户关注 NodeHost 重构与 Desktop 登录、首次注册更新之间是否发生覆盖或语义冲突。检查确认最近合并没有删除 NodeHost、Profile 或 Enrollment 的关键实现与测试，但当前 authority Enrollment Profile 仍保留一条旧的 runtime-owning binding 路径：注册前 bootstrap 和注册后的常规重连都绕过 NodeHost。普通/Legacy Profile 已使用 Parent-only NodeHost，导致两类 Profile 的生命周期、连接幂等性和 runtime ownership 不一致。

用户已确认长期方向仍是一种通用 NodeHost：Enrollment 只负责尚无 Node ID 时的注册，Authority Grant 持久化后，节点应和其他普通节点一样由 NodeHost 持有；Profile 是本机身份/连接选择，不是人类账号或树内特权。

## Confirmed Direction

- 未获得 Grant 前只有 Device identity 和 Enrollment bootstrap，不创建普通 Node 或树边。
- Grant 原子持久化后立即关闭 bootstrap，由受保护 Enrollment credential 派生正式 Node identity、Node ID 和父节点信任锚。
- authority 与 Legacy Profile 的常规运行统一使用同一个 Parent-only NodeHost 和 attached SDK Client。
- 不新增 `EnrollmentHost`、`nodeclient` 或 Desktop 专用 Host；差异通过通用 Host 的凭据来源配置表达。
- Grant/受保护 Enrollment credential 是 authority 身份与父信任的权威事实源；Profile 中的重复字段不得参与运行时裁决。
- `Connect` 对已经 running/connecting 的 Profile 使用同一 Host supervisor 幂等等待，不重复创建 runtime。
- 保留 MFHE/MFH4、集中式 Authority、Permit/Pending、DPAPI 和 Legacy Profile 兼容；不要求已有 authority Profile 重新注册。

## Problems To Resolve

1. `authority` Profile 在注册完成和重启后仍走 owning binding，没有获得 NodeHost 的统一 lifecycle、state-directory ownership 和 attached Client 边界。
2. NodeHost Profile 的重复 `Connect` 会等待已有 supervisor；owning binding Profile 的重复 Start 返回 `already started`。
3. 受保护 Enrollment credential 与 Profile settings 同时保存 Node/parent/Authority 摘要；当前只检查 Node ID 冲突，其他字段可能显示旧事实。
4. `LoginJSON` 同时承担身份准备、Enrollment、Grant 保存、Profile 激活、runtime 创建和父连接，缺少清晰的 bootstrap → Host 交接点。

## Non-goals

- 不修改 Enrollment wire、Authority 分配、Permit/Pending、撤销或普通 Join 协议。
- 不引入人类账号、密码、云会话或产品特权。
- 不在本轮迁移 Android/Embedded 的 MFHE 首次注册 UI。
- 不立即删除全部已发布 runtime-owning SDK 兼容 API；Desktop 先停止使用，全面删除需独立 downstream 审计。
- 不升级 Desktop settings major version；移除 authority Profile 冗余字段的 clean break 延期处理。

## Related Stable Docs

- [统一节点运行时](../requirements/unified-node-runtime.md)
- [受控准入](../requirements/auth-controlled-admission.md)
- [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)
- [NodeHost Runtime](../specs/node-host-runtime.md)
- [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md)
- [Desktop Profile 入口](../specs/desktop-profile-entry.md)
- [通用 NodeHost 与非 owning SDK Client](../decisions/2026-08-30_generic-node-host-and-non-owning-sdk-client.md)
- [Desktop Binding 重连与准入诊断](../lessons/desktop-binding-reconnect-and-admission-diagnostics.md)

## Open Questions

- Blocking：无。
- Deferred：旧 runtime-owning binding 的全量删除、Android/Embedded Enrollment 迁移、Desktop settings v3 与多 Authority/reparent。

