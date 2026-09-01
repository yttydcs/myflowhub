# 2026-08-30 通用 NodeHost、非 owning SDK Client 与薄平台适配

## Status

Accepted；首批实现与产品迁移进行中。

## Context

Desktop、MetricsNode、Android bindings 和其他第一方入口已经共享 protocol、`runtime/node.Node`、路由与 Session queue，但仍分别执行持久状态、Node、SDK Client、ParentSupervisor、失败回滚和关闭装配。重复所有权造成重试、关闭与跨平台 binding 语义漂移。

同时，Desktop、MetricsNode、Hub 和未来 Agent Gateway 都只是网络中的 Node。合并产品或让某个 SDK/binding 隐式拥有 runtime，会把产品组合、拓扑角色和权限等级错误绑定。

## Options Considered

1. 保持每个产品各自装配 runtime。
   - 改动最少，但生命周期、回滚、状态和关闭规则继续重复。
2. 让 `sdk.Client` 创建并拥有 Node、Parent 和 Listeners。
   - 单入口表面方便，但混淆“运行节点”和“使用节点”；Client Close 会危及共享 Host。
3. 复用 `host/hub.Hub` 作为所有产品基类。
   - 已有多 Listener 能力，但同时携带 Management/File/Flow 等 Hub feature，不是中性组合层。
4. 为 leaf、relay、root 和移动平台分别建立 Host 类型。
   - 角色直观，但会复制同一 Node 模型，并把平台差异提升为网络差异。
5. 建立纯 Go NodeHost，SDK Client 非 owning，平台和产品通过薄 adapter 组合。
   - 保持一个运行时模型、一个发送路径和清晰 ownership；需要首批迁移现有 owning bindings。

## Decision

- 新增 `host/nodehost` 作为完整 Go 节点的推荐组合入口。
- 一个 Host 拥有一个持久 auth state、一个 Node、最多一个 ParentSupervisor、零个或多个 Listeners、同一 Resource Registry 和一个 attached `*sdk.Client`。
- `Host.Client()` 返回同一非 owning Client；它不新建连接，不关闭或重配 Host。
- leaf、relay、root、offline 只由 Parent/Listeners 配置组合表达，不新增角色专用 runtime。
- SDK 只提供 Catalog、Snapshot、Operate/Invoke、Publish、Subscribe、Session 等操作 facade；运行时 ownership 从 canonical SDK 路径移出。
- Resource 便捷声明绑定 Host NodeID，但仍经过 descriptor、payload、permission、Registry 和 Catalog 契约。
- NodeHost 保持纯 Go 和平台中立。Android/iOS 通过 gomobile facade、IdentityStore、Driver/provider 与产品 adapter 注入平台能力；原生层拥有 OS lifecycle。
- Desktop、MetricsNode、Hub 与 Agent Gateway 保持独立 Node、进程、状态和安装边界。产品名不产生 authority 或 Resource ownership 特权。
- 首批迁移 Desktop、Metrics Windows/CLI、Metrics Android 和通用 Android leaf Client。Hub、Android in-process Hub、ClipboardNode 和 Agent Gateway 延期。
- Desktop 的 Legacy Profile 与 granted authority Profile 都进入 NodeHost；authority Enrollment 在注册前使用窄 bootstrap，Grant 持久化后关闭 bootstrap 并切换到 credential-backed NodeHost。

## Consequences

### Positive

- 第一方产品共享一致的创建、Start、回滚、Parent supervision 和幂等关闭语义。
- `host.Client()` 与所有 SDK operation 复用同一 Node route 和 Session queue，不增加旁路。
- 产品可以独立发布，又能共享底层运行能力。
- Android 等平台不需要第二套 Node、Resource 或权限模型。
- 后续 Hub 与 Agent Gateway 能以 feature composition 扩展，不污染通用 Host。

### Costs and Risks

- 旧 owning SDK/binding 与 attached Client 在迁移期共存，必须通过 ownership guard、弃用说明和测试避免误关 Host。
- credential-backed Host 增加一条 fail-closed 状态打开路径，必须持续防止 Profile 缓存、Grant 与 Legacy identity 形成多份事实源。
- `New` 与 `Start` 两阶段需要严格状态机；部分 Listener/Parent 失败必须完整回滚。
- gomobile 不能直接友好导出所有 Go 类型，仍需窄 facade 和生成物 ABI 验证。
- Android in-process Hub 暂未迁移，文档和调用方必须明确这是过渡例外。

## Supersedes / Superseded By

- 补充 [统一权威节点树与可插拔链路](2026-08-27_authoritative-node-tree-and-pluggable-links.md) 的 Host composition 与 SDK ownership 边界，不改变其中的树、authority、Resource ownership 和 Transport 决策。
- 若未来需要多 Parent authority、SDK-owned runtime 或平台专用 Node 模型，必须由新 ADR 明确取代本文。
- Desktop authority 的临时 post-Grant owning 例外由 [Enrollment bootstrap → NodeHost handoff](2026-09-01_enrollment-bootstrap-nodehost-handoff.md) 明确关闭。

## Related

- [NodeHost Runtime](../specs/node-host-runtime.md)
- [Operational lifecycle](../specs/operational-lifecycle.md)
- [Resource Platform v2](../specs/resource-platform-v2.md)
- [统一节点运行时 requirement](../requirements/unified-node-runtime.md)
- [本轮讨论 intake](../intake/2026-08-30_node-host-runtime-and-product-boundaries.md)
- [NodeHost runtime 与产品边界收敛 change](../change/2026-08-31_nodehost-runtime-and-product-boundaries.md)
