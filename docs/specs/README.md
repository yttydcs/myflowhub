# Specs

存放 MyFlowHub3 meta workspace 的长期技术约束、协议入口与跨仓稳定导航。

## How To Use
- 先看根级 [../README.md](../README.md)。
- 需要跨仓协议映射或稳定技术入口时，从这里进入。

## What Belongs Here
- workspace 级技术约束
- 跨仓稳定 spec 入口
- 受保护 generated 文档的维护说明

## Current Specs
- [desktop-profile-entry.md](desktop-profile-entry.md)
  - Existing Profile / 首次连接入口、受保护 Enrollment 状态投影、Profile ID 与非破坏性返回选择器契约
- [node-host-runtime.md](node-host-runtime.md)
  - 纯 Go NodeHost、Parent/Listeners 角色组合、attached SDK ownership、资源注册和薄平台适配契约
- [node-enrollment-and-admission-authority.md](node-enrollment-and-admission-authority.md)
  - 首次 Enrollment、Permit/Pending、Authority、Node ID 分配、撤销和兼容 Join 的当前契约
- [scoped-policy-authorization.md](scoped-policy-authorization.md)
  - PolicyState v2、Definitions/Bindings、owner scopes、管理 Collections、SDK/Desktop 与离线 bootstrap 契约
- [brand-identity-assets.md](brand-identity-assets.md)
  - Coupled Seam canonical 几何、色彩、full/compact/tray 资产路由与平台派生边界
- [resource-platform-v2.md](resource-platform-v2.md)
  - 可扩展 Resource descriptor/capability、通用操作、Topic 与目录的当前契约
- [resource-collections-and-actions.md](resource-collections-and-actions.md)
  - Collection/runtime、policy/filesystem/SDK、Desktop actions 与 Collection/content renderer 的当前合同；完整产品 QA pending
- [resource-sessions-v2.md](resource-sessions-v2.md)
  - 有界 control/data lane、session grant 与 File 数据面的当前契约
- [desktop-resource-workspace-v3.md](desktop-resource-workspace-v3.md)
  - Profile、可折叠 Node/Resource path tree、renderer registry、n 元 split Workspace 与本地 View v3 的当前契约
- [desktop-schema-rendering.md](desktop-schema-rendering.md)
  - provider-owned data schema、Desktop renderer registry、用户选择、generated schema 与安全 fallback 契约
- [desktop-resource-workspace-v2.md](desktop-resource-workspace-v2.md)
  - 已被 v3 取代的双面板/三面板网格历史契约
- [command-vnext.md](command-vnext.md)
  - 历史固定类型细节；通用 operation 以 `resource-platform-v2.md` 为准
- [subscription-vnext.md](subscription-vnext.md)
  - 历史 Variable/Stream 细节；通用 observable 与 Topic 以 `resource-platform-v2.md` 为准
- [resource-model-vnext.md](resource-model-vnext.md)
  - 历史三类型模型；已由 `resource-platform-v2.md` 取代
- [operational-lifecycle.md](operational-lifecycle.md)
  - Host、连接和后台任务的启动、监督、重连、reparent 与关闭语义
- [resource-catalog.md](resource-catalog.md)
  - 历史 `mfh.catalog.v1`；当前目录以 `resource-platform-v2.md` 为准
- [notification-vnext.md](notification-vnext.md)
  - Notification Stream/Command 组合及 payload/权限限制
- [file-transfer-vnext.md](file-transfer-vnext.md)
  - 历史 Command chunk 模型；data lane 以 `resource-sessions-v2.md` 为准
- [flow-vnext.md](flow-vnext.md)
  - 已退役 Flow 的历史契约；当前状态与重新设计范围见 [Flow 待办](../requirements/flow-redesign.md)
- [wire-protocol-vnext.md](wire-protocol-vnext.md)
  - MFH3 fixed operation 历史；MFH4 v2 generic operation/session 以新资源规格为准
- [node-tree-link-resource-architecture.md](node-tree-link-resource-architecture.md)
  - 统一权威节点树、资源归属、订阅/指令以及可插拔 Transport/LinkSession 的长期架构约束
- [repository-and-module-boundaries.md](repository-and-module-boundaries.md)
  - 单一 canonical monorepo 的目标布局、Go module 策略、依赖方向、兼容边界和旧仓历史保留规则
- [build-and-ci.md](build-and-ci.md)
  - 根级构建、生成、产物清单、工具链与无路径遗漏 CI 门禁
- [../../migration/sources.yaml](../../migration/sources.yaml)
  - 旧 Server、Core、SubProto 等规范和实现的精确来源基线
- [protocol_map.md](protocol_map.md)
  - MFH4 v2 descriptor/capability 当前速查
- [management-config-layering.md](management-config-layering.md)
  - `system/config`、显示名、revision、持久化与在线配置边界

## Rules
- 使用稳定文件名，不使用日期前缀。
- 变更技术真相时，先更新 spec，再更新 `plan/change`。
- `protocol_map.md` 以当前 monorepo 的 `protocol/` schema 与生成门禁为准；旧 Proto 仓库仅是迁移来源，不再是 canonical truth。
