# 2026-08-30 Desktop 与 Metrics 保持独立 Node、进程和安装包

## Status

Accepted。当前源码目录、产品入口和二进制构建已经分离；独立安装、升级、卸载与故障隔离仍需后续发布 workflow 按本决策完成验证。

## Context

MetricsNode 提供长期后台指标采集和受控设备执行，Desktop 提供通用 Node/Resource 工作台。把 Metrics 合并进 Desktop 可以减少表面上的应用数量，但会使 Desktop 同时承担交互 UI、后台采集和执行职责，并让只需要采集的设备被迫安装完整 Desktop。

MyFlowHub 采用 authoritative Node tree。Desktop、Metrics、Agent Gateway 等产品没有由名称赋予的特权；Node identity、父子拓扑和策略才决定权限。因此同机部署不是合并身份或建立私有控制通道的理由。

当前 canonical monorepo 已有独立 Desktop/Metrics 产品目录、无 UI 入口和 `mfh-desktop.exe` / `mfh-metrics.exe` 构建目标，但尚未把独立安装包和升级/卸载边界定义为稳定产品要求。

## Options Considered

### 1. 将 Metrics collector 和 actuator 合并进 Desktop 进程及安装包

- 优点：用户只看到一个应用，可能复用部分配置和 UI。
- 缺点：Node identity、后台生命周期、权限、故障域和发布周期被迫耦合；Metrics-only 场景必须安装 Desktop。
- 结论：拒绝。

### 2. Desktop 主进程附带并管理一个 Metrics sidecar，但共同安装和升级

- 优点：进程层面部分隔离，Desktop 可提供统一启停体验。
- 缺点：Metrics 仍无法独立安装和升级；Desktop 继续成为其生命周期 owner，卸载和故障边界含糊。
- 结论：拒绝作为 canonical 产品模型。

### 3. 两个独立产品，共享 monorepo 基础代码

- 优点：身份、权限、进程、状态、安装和故障域清晰；用户可以按需安装；Core/SDK 仍可复用，避免代码复制。
- 缺点：需要两个安装/升级路径，同机共存必须治理端口、状态和服务命名冲突。
- 结论：采用。

### 4. 完全拆成不同源码仓库以证明独立

- 优点：源码边界物理明显。
- 缺点：恢复跨仓发布和契约同步成本；产品独立不要求仓库独立。
- 结论：拒绝。继续使用 canonical monorepo 和明确依赖边界。

## Decision

1. Desktop Node 与 Metrics Node 是两个相互独立的产品，分别拥有 NodeID、密钥、准入、父链、策略上下文和 Resource ownership。
2. 两者必须运行在不同进程中，分别管理启动、停止、重连、崩溃恢复和安全关闭；任何一方都不是另一方的进程内模块或必需 sidecar。
3. 两者必须使用独立的配置、凭证、状态、日志、临时目录、锁和系统服务命名空间。
4. 两者必须产出独立安装包，并可以独立安装、版本化、升级、回滚和卸载；安装任一产品不得强制安装另一产品。
5. 同机共存不产生共享身份、默认父子关系或额外权限。显式配置的正常网络拓扑仍被允许，但必须遵守与其他 Node 相同的准入和策略。
6. Desktop 访问 Metrics 时只使用 authoritative Node tree 上的 Resource descriptor/capability、subscription、operation 和 session 边界；不得因为同机部署使用权限旁路。
7. Metrics Resource 始终由 Metrics Node 拥有。Desktop 可以展示、组合或缓存引用，但不成为这些 Resource 的 owner。
8. 产品边界不等于源码隔离。两者可以共享 Core、protocol、SDK、无状态基础组件、设计资产和构建工具，但不能共享 Node identity 或产品可变状态，也不能形成产品包循环依赖。
9. 根级开发、CI 或未来套件工具可以编排两个独立产品，但编排层不能成为唯一安装路径，也不能合并其进程、身份和卸载边界。
10. “Desktop 是工作台、Metrics 是采集/执行节点”只是能力描述，不构成控制面相对执行面的固有权限；实际权限继续来自身份、拓扑和策略。

## Consequences

### Positive

- Metrics 可以在服务器、弱交互设备或无人值守环境中独立运行，不承担 Desktop UI 体积和生命周期。
- Desktop 保持通用工作台职责，不被后台采集、平台 actuator 和持续运行要求污染。
- 两个 Node 可以获得不同的最小权限，并分别审计、撤权和 reparent。
- 崩溃、升级和卸载的爆炸半径被限制在单个产品。
- monorepo 仍能进行原子协议演进和共享构建门禁，不需要复制 Core/SDK。

### Costs and Risks

- 需要维护两个安装包、服务/快捷方式、升级通道和平台发布验证。
- 同机共存需要稳定且不冲突的状态目录、端口、服务名和自动启动策略。
- 独立版本可能出现契约兼容窗口，必须通过 descriptor/version negotiation 和明确错误处理。
- 未来若提供统一套件安装体验，必须持续证明它只是编排层，没有悄然重新耦合产品生命周期。

## Confidence

High。现有源码与构建目标已经证明独立进程和二进制可行；剩余工作集中在安装、升级、卸载和隔离门禁，不需要改变 authoritative Node tree 或恢复多仓架构。

## Supersedes / Superseded By

- 明确拒绝“把 MetricsNode 合并到 Desktop”作为 canonical 产品方向。
- 不禁止历史或未来开发脚本同时启动二者，但这些脚本不构成产品依赖。
- 若未来确需把两者合并为一个 Node 或一个必选安装单元，必须由新 ADR 明确取代本决策，并处理身份、权限、状态迁移和卸载风险。

## Related Features

- [Desktop](../features/desktop.md)
- [MetricsNode](../features/metrics-node.md)

## Related Requirements

- [Desktop 与 Metrics 产品独立性](../requirements/desktop-metrics-product-independence.md)
- [统一节点运行时](../requirements/unified-node-runtime.md)

## Related Specs

- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- [Build and CI](../specs/build-and-ci.md)

## Related Decisions

- [Agent Gateway 采用普通节点、多客户端令牌的独立 Web 服务](2026-08-29_agent-gateway-ordinary-node-web-service.md)
- [统一权威节点树与可插拔链路](2026-08-27_authoritative-node-tree-and-pluggable-links.md)

## Related Intake

- [Desktop Node 与 Metrics Node 产品独立性](../intake/2026-08-30_desktop-metrics-product-independence.md)

