# Requirements

存放 MyFlowHub3 meta workspace 的长期需求、范围和验收口径。

## How To Use
- 先看根级 [../README.md](../README.md) 了解阅读顺序。
- 只有当某个需求是长期稳定真相时，才在这里新增叶子文档。

## What Belongs Here
- workspace 级需求边界
- 长期接手约束
- 可被反复引用的验收标准

## Current Status

- [Android 与嵌入式重新设计](mobile-embedded-redesign.md)：已退役实现，重新设计延期。
- [flow-redesign.md](flow-redesign.md)
  - 已移除旧 Flow；通用自动化模型、语言与执行语义的延期待办，尚未选型。
- [brand-identity.md](brand-identity.md)
  - Coupled Seam 品牌主标、canonical 入口、多尺寸资产与平台派生的长期要求。
- [extensible-resource-platform.md](extensible-resource-platform.md)
  - Node-owned 可扩展 Resource type、capability、Topic、session 和全产品迁移的长期需求。
- [desktop-resource-workspace.md](desktop-resource-workspace.md)
  - Desktop Profile、登录持久化、Node/Resource Explorer、Renderer Registry、Workspace 和 View 的长期需求。
- [unified-node-runtime.md](unified-node-runtime.md)
  - 统一权威节点树、通用 NodeHost、非 owning SDK、可插拔链路和 monorepo 收敛的长期需求与验收口径。
- [management-node-display-name.md](management-node-display-name.md)
  - 设备管理中的节点显示名需求、范围与验收口径。
- [auth-controlled-admission.md](auth-controlled-admission.md)
  - 无 Node ID 首次注册、Permit/Pending 双路径、集中式 Authority、撤销与兼容 Join 的长期需求。
- [scoped-policy-authorization.md](scoped-policy-authorization.md)
  - 产品无特权、可复用 Policy Definition、精确 Subject Binding、拓扑作用域与安全 bootstrap 的长期需求。

## Rules
- 使用稳定文件名，不使用日期前缀。
- 需求变更先落这里，再写对应的 `plan/change`。
