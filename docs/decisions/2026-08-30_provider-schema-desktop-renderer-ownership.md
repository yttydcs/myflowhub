# 2026-08-30 Provider 拥有数据 Schema，Desktop 拥有 Renderer 选择

## Status

Accepted for implementation。

## Context

Desktop 当前只能从 schema ID 和 content type 得知 payload 格式，因而使用 JSON textarea/pre。用户要求
provider 决定变量类型、范围、step 等有效域，显示方决定使用 slider、number input、status、single-line、
multiline 等组件，并允许用户在兼容组件间切换。

## Options Considered

1. provider 直接指定 Desktop component：耦合具体客户端，无法支持不同平台和用户选择。
2. Desktop 从 Resource path/value 猜类型：缺少权威约束，可能提供错误或不安全控件。
3. 远程下载 executable renderer：扩大供应链、沙箱和信任风险。
4. provider-owned declarative schema + Desktop-owned ranked registry + user View preference：职责稳定且可扩展。

## Decision

- provider schema 是数据类型、约束和语义的权威声明；runtime owner 继续最终校验。
- Desktop renderer registry 只提供 allowlisted 本地组件，负责 compatibility、default rank 和 responsive mode。
- 用户选择保存在 View，改变 presentation 不改变 value 或权限。
- provider presentation metadata 只作为 hint，不能强制 executable component 或放宽 schema。
- phase 1 从 canonical protocol definitions 生成 first-party Desktop schema artifact，不改变 wire。
- owner-served declarative schema discovery 延期到独立协议 workflow；远程 executable renderer 不属于该能力。
- unknown、unsupported、invalid 或 incompatible 情况必须显式 fallback 到 structured/raw inspector。

## Consequences

- `Renderer.tsx` 需要拆分为 schema resolver、registry、resource controller 和 presentation primitive。
- View v3 无需升级；现有 renderer/settings 字段承载选择和 display-only settings，legacy ID 作为 alias。
- Go validation 与 declarative definition 可能漂移，必须通过 deterministic generation、coverage、freshness 和
  representative parity tests 防止。
- bounded schema vocabulary 不能忠实表达的复杂结构保留 Advanced JSON，而不是实现不完整的通用引擎。

## Supersedes / Superseded By

- 细化 [可扩展 Resource type system 与 Desktop workspace](2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)
  的 Renderer Registry 选择；不改变 Node/Resource ownership、capability 或 View layout 决策。

## Related Docs

- [Desktop schema rendering](../specs/desktop-schema-rendering.md)
- [Desktop requirements](../requirements/desktop-resource-workspace.md)
- [Discussion intake](../intake/2026-08-30_desktop-schema-driven-resource-widgets.md)
