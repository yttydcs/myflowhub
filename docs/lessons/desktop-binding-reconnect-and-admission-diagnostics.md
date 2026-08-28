# Desktop Binding 重连与准入诊断

## Summary

Desktop 的 binding client 是单次启动生命周期对象。一次连接或首次准入失败后，如果下一次重试继续复用已经启动过的 client，会把真正的身份、权限或链路错误遮蔽成生命周期错误；如果监督循环只返回 `context deadline exceeded`，用户也无法判断是 permit、父节点公钥、网络还是权限配置有问题。

## Symptoms And Triggers

- 症状：第一次连接失败后，再次点击连接立即出现 `parent trust cannot change after the binding client starts`。
- 症状：界面只显示“连接超时”，但运行时此前已经得到更具体的准入拒绝或父节点身份错误。
- 症状：用户修正地址或 permit 后仍无法重试，必须重启 Desktop 才恢复。
- 触发：复用已经 `Start` 过的 binding client，或在失败登录后立即清空一次性 permit。
- 关键词：`parent trust cannot change`、`context deadline exceeded`、`admission rejected`、`last_error`、`permit`。

## Root Cause

连接配置、父节点信任和启动生命周期被绑定在同一个 client 实例上。失败实例仍保留“已经启动”的状态，不能重新设置 parent trust；同时，若上层只等待 deadline 并丢弃监督循环最后一次错误，最具诊断价值的信息会消失。permit 又属于一次性准入输入，失败时清空会让下一次尝试缺少原始条件。

## Resolution Pattern

1. 每次 `Connect` 重试创建新的候选 binding client，只在连接成功后替换当前活动实例。
2. 用 lifecycle mutex 串行化连接、切换和关闭，避免旧连接清理与新连接建立交叉。
3. 超时时优先返回监督循环记录的最后一个具体错误，并用 deadline 作为外层上下文，而不是覆盖根因。
4. 只在成功登录后清空 UI 中的一次性 permit；失败时保留，允许用户修正其他字段后重试。
5. Profile、node identity 与 view store 必须按 profile ID 隔离；切换前先清理旧 subscription/session。

## Quick Checks

- 查看 `StatusJSON` 的 `last_error`，确认超时前是否已有更具体的链路或准入错误。
- 检查重试路径是否重新调用 client factory，而不是对已有实例再次 `Start`。
- 核对输入的 parent public key 与测试 Hub 实际 identity 完全一致。
- 检查失败路径是否提前清空 permit，成功路径是否确实清空。
- 在隔离配置目录连续执行“失败连接 → 修正输入 → 成功连接 → 重启自动连接”，不要只验证冷启动成功。

## Prevention Rules

- 将 binding client 视为 disposable session object，不设计成可无限重启的全局单例。
- UI 错误必须保留 runtime 的最后一个可操作诊断；不得用通用 timeout 静默降级。
- 涉及一次性凭据的测试必须覆盖失败后重试和成功后清理两个方向。
- 重连、切换 Profile 与 shutdown 必须进入 race gate，并使用真实 production Wails 路径做至少一次验收。

## Related Docs

- [Desktop feature](../features/desktop.md)
- [Desktop resource workspace v2](../specs/desktop-resource-workspace-v2.md)
- [Resource sessions v2](../specs/resource-sessions-v2.md)
- [Auth-controlled admission](../requirements/auth-controlled-admission.md)
- [2026-08-29 resource platform and Desktop closeout](../change/2026-08-29_extensible-resource-platform-desktop-workspace.md)
