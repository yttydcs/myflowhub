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
- [command-vnext.md](command-vnext.md)
  - Command authorization origin、deadline、去重、晚到结果与 panic 隔离
- [subscription-vnext.md](subscription-vnext.md)
  - Subscription 租约、聚合、Variable 合并与 Stream gap/backpressure 语义
- [resource-model-vnext.md](resource-model-vnext.md)
  - 节点资源寻址、Variable revision、Stream sequence 与 Command 边界
- [wire-protocol-vnext.md](wire-protocol-vnext.md)
  - vNext 固定帧、路由 phase、operation、错误码与尺寸上限
- [node-tree-link-resource-architecture.md](node-tree-link-resource-architecture.md)
  - 统一权威节点树、资源归属、订阅/指令和可插拔链路的长期约束
- [repository-and-module-boundaries.md](repository-and-module-boundaries.md)
  - canonical monorepo 布局、依赖方向、兼容边界与迁移规则
- [protocol_map.md](protocol_map.md)
  - 来自 `MyFlowHub-Proto` 的协议映射同步副本
- [management-config-layering.md](management-config-layering.md)
  - 管理子协议中的节点显示名与 `hubruntime` 配置分层约束
- [../../migration/sources.yaml](../../migration/sources.yaml)
  - 旧 Server、Core、SubProto 等规范和实现的精确来源基线

## Rules
- 使用稳定文件名，不使用日期前缀。
- 变更技术真相时，先更新 spec，再更新 `plan/change`。
