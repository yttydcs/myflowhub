# 2026-08-27 统一权威节点树与可插拔链路

## Status

Accepted

## Context

MyFlowHub 已经形成父节点控制子树、请求向上寻找裁决点、已授权控制向下执行的树形权限模型，也已通过 Core `IPipe/IConnection`、RFCOMM 和 QUIC 验证多种承载复用同一协议栈的可行性。

后续设计一度倾向把物理链路、逻辑资源、路由与权限进一步拆分，以获得任意实现替换能力。但这种拆分会削弱父子链路的直接控制语义，引入资源归属到权限关系的额外映射，并牺牲常规开发便利性。本次重构需要在适应不同承载的同时，保留原有树形自治的核心优势。

## Options Considered

### 1. 权限树、连接图和资源树完全分离

- 优点：拓扑和资源组织最灵活，可支持多路径与独立挂载。
- 缺点：端点需要独立验证 capability 或签名授权；LCA 和父控子语义不再自然成立；运行时必须持续维护多套映射。
- 结论：拒绝。它改变了现有安全模型，也会再次造成过度拆分。

### 2. 独立节点树与资源树，使用映射连接二者

- 优点：资源可以脱离节点重新组织。
- 缺点：资源 authority、路由和生命周期需要额外映射；资源移动时容易出现权限与订阅状态不一致。
- 结论：拒绝。当前没有足够收益证明这层复杂度必要。

### 3. 统一权威节点树，资源归属节点，传输可插拔

- 优点：保留父控子、LCA 裁决和局部自治；资源归属明确；新增承载不会复制权限、路由和业务实现。
- 缺点：同一 authority 域内不能同时存在多个有效父节点；父节点被攻破时影响整棵子树；reparent 是安全状态变化而不仅是路由变化。
- 结论：采用。

## Decision

1. 运行期只有一棵权威节点树。树由 `Node` 和经过认证的父子链路边组成。
2. 权限树、协议连接树和节点路由树使用同一父子关系。每个非根节点在一个 authority 域内只能有一个当前有效父节点。
3. 节点拥有资源。Variable、Stream 和 Command 是节点资源；资源不是拓扑节点，不拥有独立父节点。
4. Subscription 是面向 Variable 或 Stream 的一级运行时关系，而不是某个子协议的附属状态。
5. 父节点有权控制其子树。子节点只对“当前已认证父节点下发的明确控制阶段消息”免于重复判权；来自父连接但仍处于请求阶段的消息不自动视为已裁决。
6. 非直接父控子方向的请求沿父链上行，在能够覆盖请求者和目标的 LCA 或可裁决节点完成身份与权限检查；通过后转换为控制消息向下投递。
7. 传输协议不是节点或权限语义的一部分。TCP、Bluetooth RFCOMM、QUIC、串口等通过统一连接与链路会话契约接入；同一棵节点树的不同边可以使用不同承载。
8. 底层连接成功不等于获得父子权力。只有在身份认证、角色确认和会话激活完成后，连接才能成为树边。
9. 更换同一父身份下的承载属于链路恢复或迁移；更换父身份属于 reparent，必须更新拓扑世代并重新处理旧路由、授权和订阅状态。
10. 身份准入 authority 可以签发或验证身份，但不构成第二棵运行期权限树。父节点能够控制子节点，但不能伪装成子节点身份。

## Consequences

### Positive

- 权限、路由和连接方向使用同一事实来源，减少同步和映射错误。
- Variable、Stream、Command 和 Subscription 可以独立于具体承载发展。
- 新增传输只需满足统一链路契约，不需要新增一套权限、路由或资源实现。
- 现有 `IPipe/IConnection`、多 Listener、RFCOMM 和 QUIC 经验可以继续复用。

### Costs and Risks

- 父节点拥有其子树的高权限，父节点失陷具有明确的子树爆炸半径；父节点实现和密钥保护属于高安全优先级。
- 多父高可用不能直接表现为多个 authority parent；如未来需要多路径，应优先在同一父子会话之下进行承载切换或聚合。
- reparent 必须被视为安全事件，不能仅更新下一跳。
- 无法建立双向、可认证链路的弱设备不能作为完整 Node，应由网关节点将其表现为资源。

## Confidence

High。该决定延续已验证的权限与多承载方向，同时减少不必要的架构分裂。

## Supersedes / Superseded By

- 不取代原有树形权限设计；本决策对其进行确认、收敛并补充可插拔承载边界。
- 若未来决定支持同一 authority 域内的多父控制，必须由新的 ADR 明确取代本决策。

## Related Features

- 暂无独立 feature 文档。

## Related Specs

- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- 旧 Server 权限/路由规范与 EmbeddedSDK 控制链经验按精确 commit 保留，来源见 [迁移清单](../../migration/sources.yaml)。

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
- [Core Pipe 抽象与多承载基础](../change/2026-03-12_transport-pipe-core.md)
- [Bluetooth RFCOMM Transport](../change/2026-03-12_bluetooth-rfcomm-transport.md)
- [Link Router Kernel 重构](../change/2026-03-13_link-router-kernel-major-refactor.md)
