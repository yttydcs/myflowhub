# 2026-08-28 可扩展 Resource type system 与 Desktop workspace

## Status

Implemented。Resource/Renderer/Profile 边界保持有效；其中“响应式网格 Workspace”的布局选择已被
[Desktop n 元多面板停靠](2026-08-30_desktop-n-ary-docking-layout.md) 部分取代。

## Context

现有架构已经接受唯一 authoritative Node tree、父控子、Resource 归属 Node 和可插拔 Transport，但把 Resource kind 固定为 Variable、Stream、Command。用户要求把 Topic 等类型提升为 Resource，并让 Desktop 从固定页面升级为支持 Profile、树浏览、类型化预览、拖放组合和持久 View 的资源工作台。

## Options Considered

1. 继续扩展 Core hard-coded Resource enum。
   - 短期简单，但每种新类型都会扩大协议 switch、runtime 分支和客户端升级面。
2. 使用 extensible type ID + capability descriptor，Core 只维护通用资源生命周期。
   - 需要一次 clean-break，但能够保持未知类型可发现和类型实现解耦。
3. 把所有行为退化为 Command。
   - wire 表面最小，但丢失当前值、订阅、publisher/consumer、QoS 和类型化 UI 语义。
4. Desktop 继续固定页面。
   - 无法满足通用发现与组合需求。
5. Desktop 使用无限自由画布。
   - 灵活但会显著增加布局、可访问性和持久化复杂度。
6. Desktop 使用响应式可调整网格与 renderer registry。
   - 能满足组合 View，同时保持视觉和实现边界克制。

## Decision

- Resource 使用 versioned type ID 与显式 capability descriptor；Core 不以产品类型 switch 作为扩展机制。
- Variable、Stream、Topic、Command 是第一阶段基础类型。
- Topic 是 Node-owned、多发布者/多订阅者资源；publish 与 subscribe 分权，默认无 retention/replay，只保证单 publisher 顺序。
- File 和未来 Media 使用 resource session control；大数据面不作为普通 Stream event 传输。
- 唯一 authoritative Node tree、父控子、Resource ownership 和 Transport abstraction 保持不变。
- Desktop 使用 Wails + React + TypeScript + Vite、Radix/shadcn 风格源码内 primitives 与项目自有 CSS tokens；不为了工具名引入未使用的 Tailwind 构建层。
- Desktop 使用 Renderer Registry、Node/Resource Explorer、响应式网格 Workspace 和 local-first per-Profile View store。
- Profile 表示一个 Hub/本地 Node identity/信任域/View 集合；首版一个应用实例只激活一个 Profile。
- View 同步、多 Profile 同时在线和生产级实时媒体数据面延期。

## Consequences

- 当前 fixed-three-kind resource spec、catalog/wire operations、SDK/bindings 和 first-party products 需要不兼容迁移。
- Desktop 前端将从 Vanilla TypeScript clean-break 到 React，不保留两套 UI。
- catalog、auth action、subscription 和 generic operation 必须从 kind-specific switch 转为 descriptor-driven validation。
- 新类型可以由独立包实现，但必须通过 Core 的权限、路由、配额、生命周期和生成契约门禁。
- shadcn/ui 负责可维护的 UI primitives，不负责 Resource tree、renderer registry 或 View domain model；这些仍是产品代码。
- 生产级 Media 仍需要单独的 codec、transport/QoS、安全和平台计划，不能因存在 `Media` descriptor 就宣称视频已实现。

## Confidence

High：Node tree 和 Resource ownership 边界已经验证；主要风险集中在 clean-break 迁移面、session/data lane 与 Desktop 状态复杂度，可通过分门禁实施控制。

## Supersedes / Superseded By

- 部分取代 [统一权威节点树与可插拔链路](2026-08-27_authoritative-node-tree-and-pluggable-links.md) 中“基础资源仅为固定三类型”的封闭解释。
- 保留该决策关于唯一父节点、父控子、Resource 归属 Node、链路可插拔和权限沿树强制执行的约束。
- 实现完成时由新的 Resource platform spec 明确取代 [Resource Model vNext](../specs/resource-model-vnext.md)。
- Desktop 的响应式网格布局部分被
  [Desktop 使用 n 元分割树实现多面板停靠](2026-08-30_desktop-n-ary-docking-layout.md) 取代；
  Wails/React/Renderer Registry/local-first View store 选择保持不变。

## Related Features

- [Desktop](../features/desktop.md)
- [File transfer](../features/file-transfer.md)

## Related Requirements

- [可扩展资源平台](../requirements/extensible-resource-platform.md)
- [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)

## Related Intake

- [可扩展资源平台与 Desktop 工作区重构](../intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)
