# Credential 状态检查必须是只读路径

## Summary

登录选择器、诊断页和状态列表只是读取现有 credential 的投影。它们不得复用 load-or-create、prepare 或会创建目录的构造器；否则“打开页面”本身就会改变设备身份和准入状态，并可能把损坏凭据伪装成全新设备。

## Lookup Hints

- Profile chooser creates directory
- credential status inspection
- load-or-create side effect
- DPAPI read-only
- missing / device / pending / enrolled
- corrupt credential silently replaced
- secret projection

## Symptoms

- 仅打开 Profile 选择器就出现新的 Profile state 目录或 credential 文件。
- 原本应为 `missing` 的 Profile 变成 `device`，并生成新的 request/identity。
- 损坏或不可解密的 credential 被新的身份覆盖，错误提示消失。
- 状态 API 无意间返回私钥、Permit、Grant 签名或完整公钥正文。

## Impact

读取操作变成不可见的本地状态变更，会破坏 TOFU、Pending request 稳定性、审计可解释性和恢复路径。最坏情况下，用户只是查看状态就丢失原设备身份，后续审批和已签发 Grant 无法继续使用。

## Trigger Conditions

- 状态代码调用 `LoadOrCreate*`、`Prepare*` 或默认会 `MkdirAll` 的 backend 构造器。
- “不存在”和“损坏”被折叠成同一个 fallback 分支。
- 领域 credential 直接序列化到前端，而不是建立明确的 allowlist DTO。
- 平台 credential backend 只有读写一体接口，调用者无法表达“只打开已存在对象”。

## Root Cause

身份建立与身份观察共享了同一个便利入口。该入口适合首次连接，但不适合列表和诊断：它把不存在视为需要创建，把读取失败视为可重新初始化，并让文件系统副作用藏在构造阶段。

## Investigation Trail

1. 从 UI 的 Profile 状态刷新链路反查到 Wails `ProfileStatesJSON`。
2. 沿 credential store 检查 backend 构造、目录创建和 load/save 调用。
3. 用不存在、正常、Pending、Enrolled、损坏和身份冲突 fixture 比较调用前后的目录与文件。
4. 检查前端 JSON，只允许状态、request ID 和必要 Node ID，不允许 secret 或签名材料。

## Resolution

- 增加独立的 `InspectEnrollmentClientState`：只读取并验证已存在 credential，`found=false` 时不创建或保存任何状态。
- 平台 backend 构造与持久化分离；Windows DPAPI backend 可被打开用于读取，但不会因为构造而创建目录。
- 对损坏、不可解密和 Profile/Grant Node ID 冲突显式返回 `error`，不自动换新身份。
- Desktop 使用 allowlist 状态 DTO 投影 `missing/device/pending/enrolled/error/legacy`，不暴露 credential 本体。

## Prevention / Guardrails

- 为 missing 状态增加 zero-write 测试，比较调用前后的目录、文件和存储调用次数。
- 为损坏 credential 增加 fail-closed 测试，禁止 fallback 到创建路径。
- 状态 DTO 使用显式字段，不直接 marshal credential/domain object。
- API 命名区分 `Inspect`、`Load`、`Prepare` 和 `LoadOrCreate`，代码评审时把构造器副作用视为边界行为。
- UI 刷新和诊断路径只依赖 `Inspect`；创建身份只能由用户明确触发的首次连接动作进入。

## Related Docs

- [Desktop Profile 入口](../specs/desktop-profile-entry.md)
- [Desktop 当前行为](../features/desktop.md)
- [Desktop 资源工作区需求](../requirements/desktop-resource-workspace.md)
- [Desktop Profile 入口生产化归档](../change/2026-08-31_desktop-profile-entry-production.md)
