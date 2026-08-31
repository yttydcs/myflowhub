# 2026-08-31 Resource Collection、Capability 与 Desktop 操作入口

## Source

- 来源：用户在 MyFlowHub 架构讨论中的连续澄清。
- 日期：2026-08-31。

## Source-preserving summary

用户希望重新整理 Desktop 左侧资源目录，不只处理文件，还要统一表达其他资源、权限和命令。讨论确认：

- Resource 的普通操作不应继续被拆成独立 Resource；只有操作自身构成独立安全与审计边界时，才使用
  Command Resource。
- Collection 是对 Resource 模型的补充：一个 Resource 管理多个内部 member，member 默认不逐项进入全局
  catalog。
- 本地多个目录 `a/b/c` 应能按目录 root 方便注册为可分别授权的 Resources；物理路径不暴露为网络身份。
- Flow 的 definitions 与 runs 适合作为 Collections，create/update/run/cancel 等成为相应 Resource 的
  capabilities。
- Desktop 应在真实 Resource 上提供右键快捷操作；加入 View 后可显示常用操作按钮，并按
  type/schema/capability/content type 选择文本、代码、图片等专用展示。
- Desktop、Metrics、Agent Gateway 等仍是普通、独立 Node 产品；该资源模型不赋予任何产品特权。

## Confirmed requirements

- 唯一 authoritative Node tree、Resource owner 和 `(NodeID, local name)` 身份保持不变。
- UI path namespace 只用于展示，不成为 Resource、Collection、permission 或 routing tree。
- Collection member locator 必须有界并由 provider 验证；网络 authority 与 owner member validation 均不能绕过。
- 同一 capability 从 Explorer、Inspector 或 View Widget 触发时必须走同一 SDK、policy、schema、错误和审计路径。
- 任意操作系统命令不得包装成无约束远程 shell；只允许预定义任务或受控执行服务。

## Planning boundary

首个实施阶段应建立通用 Collection/多 Capability Resource 基础，迁移 Flow 作为第一方样板，增加可独立组合的
只读 filesystem Collection provider，并完成 Desktop 上下文菜单、View action 与安全内容 renderer。其他
Management/Admission/Notification 命令迁移、通用 member selector policy、有效权限发现、可写文件和下载
data lane 作为显式后续任务，不在本阶段伪装完成。

## Routed docs

- [可扩展资源平台 requirement](../requirements/extensible-resource-platform.md)
- [Desktop 资源工作区 requirement](../requirements/desktop-resource-workspace.md)
- [Resource Collections and Actions spec](../specs/resource-collections-and-actions.md)
- [Flow vNext spec](../specs/flow-vnext.md)
- [Collection Resource 与 Capability Action 决策](../decisions/2026-08-31_collection-resource-and-capability-actions.md)
- [实现与验证归档](../change/2026-08-31_resource-collections-and-capability-actions.md)
