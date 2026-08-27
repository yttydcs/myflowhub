# Specs

存放 MyFlowHub3 meta workspace 的长期技术约束、协议入口与跨仓稳定导航。

## How To Use
- 先看根级 [../README.md](../README.md)。
- 需要 vNext 协议映射或稳定技术入口时，从这里进入。

## What Belongs Here
- workspace 级技术约束
- 跨仓稳定 spec 入口
- 受保护 generated 文档的维护说明

## Current Specs
- [command-vnext.md](command-vnext.md)
  - Command authorization origin、deadline、去重、晚到结果与 panic 隔离
- [subscription-vnext.md](subscription-vnext.md)
  - Subscription 租约、聚合、Variable 合并与 Stream gap/backpressure 语义
- [resource-model-vnext.md](resource-model-vnext.md)
  - 节点资源寻址、Variable revision、Stream sequence 与 Command 边界
- [operational-lifecycle.md](operational-lifecycle.md)
  - Host、连接和后台任务的启动、监督、重连、reparent 与关闭语义
- [resource-catalog.md](resource-catalog.md)
  - `system/catalog` Variable schema、revision、排序、权限和更新规则
- [notification-vnext.md](notification-vnext.md)
  - Notification Stream/Command 组合及 payload/权限限制
- [file-transfer-vnext.md](file-transfer-vnext.md)
  - File transfer 会话、chunk、校验、幂等、路径和存储约束
- [flow-vnext.md](flow-vnext.md)
  - Flow definition/status/run/cancel/archive 的资源组合与权限传播
- [wire-protocol-vnext.md](wire-protocol-vnext.md)
  - vNext 固定帧、路由 phase、operation、错误码与尺寸上限
- [node-tree-link-resource-architecture.md](node-tree-link-resource-architecture.md)
  - 统一权威节点树、资源归属、订阅/指令和可插拔链路的长期约束
- [repository-and-module-boundaries.md](repository-and-module-boundaries.md)
  - canonical monorepo 布局、依赖方向、兼容边界与迁移规则
- [build-and-ci.md](build-and-ci.md)
  - 根级构建、生成、产物清单、工具链与无路径遗漏 CI 门禁
- [protocol_map.md](protocol_map.md)
  - vNext envelope operation、三资源模型和内置资源族速查
- [management-config-layering.md](management-config-layering.md)
  - `system/config`、显示名、revision、持久化与在线配置边界
- [../../migration/sources.yaml](../../migration/sources.yaml)
  - 已退役旧 Server、Core、SubProto 等仓库的精确来源基线

## Rules
- 使用稳定文件名，不使用日期前缀。
- 变更技术真相时，先更新 spec，再更新 `plan/change`。
