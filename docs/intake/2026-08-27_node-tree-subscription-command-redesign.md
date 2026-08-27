# 2026-08-27 节点树、订阅与指令重构诉求

## Source

- 来源：用户在当前架构讨论中的连续确认。
- 日期：2026-08-27。

## Request Text / Source-preserving Summary

- 计划对当前过度拆分的架构进行大范围重构，并重新定义“树”。
- 不再让子协议拆分和任意实现替换优先于日常开发便利性；子协议不再被视为必要的一级架构边界。
- 将订阅提升为一级概念。变量和流是主要的可订阅资源，用以表达绝大多数状态与事件场景。
- 将指令提升为一级概念，作为主动控制和无法由订阅表达的补充手段。
- 沿用既有父子权限模型：父节点有权控制子树，子节点信任父节点下发的已裁决控制。
- 权限树与链路树保持为同一棵节点树；资源归属于节点，不另建独立资源树。
- 链路承载必须可扩展，可由 TCP、Bluetooth、串口、QUIC 或其他协议实现，同时保持统一的节点、权限、路由、订阅与指令语义。
- 架构相关文档必须作为长期资料保留，不能只存在于聊天、临时计划或历史变更记录中。
- 当前没有项目外部用户依赖既有 Go module path、旧 SubProto API 或旧 wire 协议，可以按新架构进行破坏性调整。

## Context

当前工程已经具备 `IPipe`、`IConnection`、多 Listener、RFCOMM、QUIC 等多承载基础，但长期架构真相分散在 Server spec、论文、历史 plan/change 和各仓实现文档中。本次讨论希望在推倒重来之前，先固定新的核心概念和不可破坏的架构约束。

## Confirmed Requirements

- 运行期权威关系由唯一父子节点树表达。
- 经过认证的父子链路既是路由边，也是运行期权限边。
- 节点拥有资源；资源不参与父子拓扑，也不拥有独立 authority parent。
- Variable 和 Stream 可订阅；Command 可调用；Subscription 是一级运行时关系。
- 具体传输协议通过统一链路契约适配，不向资源、权限和业务语义泄漏。
- 原有权限模型及其历史依据继续保留，并由新的架构决策和稳定规范交叉引用。
- 所有第一方代码收敛到一个 canonical monorepo；构建产物可以独立，但内部架构不再依赖独立 Git 仓库和 module 表达。

## Open Questions

- Variable、Stream、Command 的最终 wire envelope、标识符和版本协商尚未定义。
- 订阅聚合、租约、恢复、背压和跨 reparent 行为需要在后续技术计划中细化。
- 现有 SubProto API 的兼容期、迁移顺序和最终删除范围尚未决定。

## Routed Docs

- [统一节点运行时需求](../requirements/unified-node-runtime.md)
- [统一权威节点树与可插拔链路架构决策](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [单一 Canonical Monorepo 架构决策](../decisions/2026-08-27_single-canonical-monorepo.md)
- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
- [Core Pipe 抽象与多承载基础](../change/2026-03-12_transport-pipe-core.md)
- [Bluetooth RFCOMM Transport](../change/2026-03-12_bluetooth-rfcomm-transport.md)
- [Link Router Kernel 重构](../change/2026-03-13_link-router-kernel-major-refactor.md)
