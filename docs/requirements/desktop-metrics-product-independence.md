# Desktop 与 Metrics 产品独立性

## Status

产品边界已确认。当前 canonical source 和构建入口已经分离，独立安装、升级与卸载契约仍待后续发布工作实现和验证。

## Background

Desktop 提供通用 Node/Resource 浏览、预览和组合工作区；MetricsNode 负责设备指标采集及获授权的本地执行。把 Metrics 合并到 Desktop 会使交互工作台同时承担长期后台采集和设备执行职责，也会迫使只需要 Metrics 的设备安装 Desktop。

MyFlowHub 的产品名称不产生网络特权。Desktop 与 Metrics 即使运行在同一台设备，也应作为两个普通 Node 独立加入 authoritative Node tree，由各自身份、拓扑关系和策略获得权限。

## Goal

- 保持 Desktop Node 与 Metrics Node 在身份、进程、状态、权限、生命周期和交付物上的完整独立。
- 允许用户只安装 Desktop、只安装 Metrics，或同时安装二者，而不改变任何一方的核心行为。
- 让 Desktop 对 Metrics 的观察和控制继续使用通用 Resource/capability 与树上权限路径，不建立产品私有旁路。
- 在不破坏产品边界的前提下，允许 monorepo 内共享 Core、协议、SDK、基础组件和构建工具。

## Product Boundary

| 边界 | Desktop | Metrics |
| --- | --- | --- |
| Node identity | 独立 NodeID、密钥、准入和父链 | 独立 NodeID、密钥、准入和父链 |
| Process | 独立 Desktop 进程 | 独立 Metrics 后台/界面进程 |
| State | 独立 Profile、View、凭证和 UI 状态 | 独立采集配置、凭证、平台状态和日志 |
| Resource ownership | 拥有自身声明的 Desktop Resource | 拥有指标、配置和设备执行 Resource |
| Package | 独立安装、升级和卸载 | 独立安装、升级和卸载 |
| Network authority | 只来自身份、拓扑与策略 | 只来自身份、拓扑与策略 |

这两个产品可以共享源码和发布基础设施，但不得共享 Node identity、可变运行状态、隐式权限或必需的私有 IPC。

## Scope

### Included

- Desktop 与 Metrics 的 Node identity、admission、policy 和 topology 隔离。
- 独立进程、启动/停止、崩溃恢复和后台生命周期。
- 独立配置根、凭证存储、日志和临时状态。
- 独立构建产物、安装包、版本、升级和卸载路径。
- 同机共存时的端口、文件、服务名、自动启动和资源占用冲突防护。
- Desktop 通过 canonical SDK 和 Resource capability 访问 Metrics Node。
- 共享 Core/SDK/协议、无状态库、设计资产和构建流水线的允许边界。

### Out of Scope

- 把 Metrics collector、actuator 或后台服务内嵌到 Desktop 进程。
- 让 Desktop 安装包强制携带或启动 Metrics。
- 让 Metrics 安装包强制携带或启动 Desktop。
- 因为同机部署而共享 NodeID、私钥、permit、Profile 或权限。
- 使用私有本地 IPC 绕过 authoritative Node tree、Resource owner 或策略裁决。
- 将 Desktop 的“工作台/控制能力”解释为网络级控制面特权。

## Scenarios

- 用户只安装 Metrics，在没有 Desktop 的设备上完成准入、指标采集、后台运行和资源发布。
- 用户只安装 Desktop，通过网络发现并查看其他设备上的 Metrics Node。
- 同一设备同时安装二者，各自完成准入；它们可以连接同一父节点，也可以按显式配置连接不同父节点。
- Desktop 获授权后订阅 Metrics Variable 或调用设备控制 capability；撤权后操作明确失败，与是否同机无关。
- 升级或卸载 Desktop 后，Metrics 的 Node identity、配置、后台服务和网络资源保持不变。
- Metrics 崩溃或被禁用后，Desktop 继续运行，并把对应远端 Resource 显示为不可用而不是自身故障。

## Functional Requirements

1. Desktop 和 Metrics 必须生成并持有不同的 Node identity；任何迁移或安装流程不得复制另一产品的私钥作为默认行为。
2. 两个产品必须分别执行 Node admission、父节点配置、登录、reconnect 和 reparent 生命周期。
3. 同机安装不得自动授予父子、兄弟或本地可信权限；任何拓扑关系和权限都必须由 canonical 配置与策略显式产生。
4. Desktop 和 Metrics 必须运行在不同进程中，并可以独立启动、停止、重启和崩溃恢复。
5. Desktop 不得加载 Metrics collector/actuator 作为其进程内必选模块；Metrics 不得依赖 Desktop renderer、Profile 或窗口生命周期完成采集和执行。
6. 两个产品必须使用不同且可配置的状态根、credential namespace、日志位置、临时目录、锁文件和服务标识。
7. 一个产品的 reset、repair 或 uninstall 不得删除、重置或迁移另一产品的身份与状态。
8. 每个受支持平台必须为 Desktop 与 Metrics 产出独立可安装 artifact；用户不得因安装其中一个而被迫安装另一个。
9. 两个 artifact 必须可以独立版本化、升级、回滚和卸载；版本差异通过稳定网络/SDK 契约处理，而不是强制同步升级。
10. 若发布系统未来提供套件级下载或安装编排，它只能组合两个独立 artifact，不能成为任一产品的唯一安装路径，也不能合并其身份、进程或卸载边界。
11. Desktop 发现、读取、订阅、配置或调用 Metrics Resource 时，必须使用与其他 Node 相同的 descriptor/capability、schema、limits 和权限裁决。
12. Metrics Resource 必须由 Metrics Node 拥有；Desktop 的 View、renderer 或缓存不得改变 owner，也不得把 Metrics 数据重新声明为 Desktop 本地资源以获得权限捷径。
13. 同机优化不得成为正确性、可用性或授权所必需的路径；任何可选优化也必须保留与网络路径等价的身份和策略检查。
14. 共享代码必须位于明确的 Core/SDK/基础库边界，不能通过共享全局可变状态或产品包反向依赖耦合两个 runtime。
15. 根级开发脚本可以同时启动二者，但必须保留独立参数、端口、状态目录、日志和停止行为。

## Non-functional Requirements

- 隔离：任一产品的崩溃、死锁、资源耗尽和更新失败不得直接终止另一产品进程。
- 安全：密钥、permit、credential store 和策略缓存不得跨产品复用或静默导入。
- 可维护：共享代码通过稳定依赖方向复用，不复制产品实现，也不形成 Desktop ↔ Metrics 循环依赖。
- 可部署：单产品安装的磁盘、后台服务和依赖集合必须自洽且可验证。
- 可升级：独立版本在受支持兼容窗口内可以互操作；不兼容时明确报告契约版本错误。
- 可审计：网络行为分别归属于 Desktop NodeID 或 Metrics NodeID，不能因同机部署混为一个主体。

## Edge Cases

- 同机安装时两个产品选择了相同端口、状态目录、服务名或锁文件。
- 用户先卸载 Desktop，再升级 Metrics；或反向操作。
- Desktop 和 Metrics 使用不同版本的 Resource descriptor/SDK contract。
- 用户误把同一个 identity 文件分别导入两个产品。
- 一个产品 reset settings，另一个仍在运行并持有自己的状态。
- 两个 Node 连接同一父节点，但被分配不同权限和 Resource scope。
- Desktop 当前 View 引用了本机 Metrics Resource，而 Metrics 服务暂时停止或被卸载。
- 套件安装器只完成一半，必须能够识别两个独立产品的真实安装状态并安全恢复。

## Acceptance Criteria

- 在干净环境中分别完成 Desktop-only、Metrics-only 和二者共存三种安装/运行门禁。
- 两个产品运行时具有不同进程、NodeID、credential namespace、状态根、日志和可独立停止的生命周期。
- Desktop-only 环境不包含或启动 Metrics collector；Metrics-only 环境不包含或启动 Desktop 工作区。
- 同机共存时，一方停止、崩溃、升级、回滚或卸载不会删除另一方状态，也不会终止另一方进程。
- 权限测试证明 Desktop 只能按获授权的 Resource capability 访问 Metrics；同机部署、相同操作系统用户或套件安装均不产生额外权限。
- 发布清单分别列出 Desktop 与 Metrics artifact、版本、校验和、安装/卸载命令及回滚结果。
- 共享依赖审计证明 Core/SDK 可以复用，但不存在 Desktop 产品包依赖 Metrics 产品包、Metrics 产品包依赖 Desktop 产品包或两者循环依赖。

## Related Features

- [Desktop](../features/desktop.md)
- [MetricsNode](../features/metrics-node.md)

## Related Requirements

- [统一节点运行时](unified-node-runtime.md)
- [Desktop 资源工作区](desktop-resource-workspace.md)

## Related Specs

- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- [Build and CI](../specs/build-and-ci.md)

## Related Decisions

- [Desktop 与 Metrics 保持独立 Node、进程和安装包](../decisions/2026-08-30_desktop-metrics-independent-products.md)
- [Agent Gateway 采用普通节点、多客户端令牌的独立 Web 服务](../decisions/2026-08-29_agent-gateway-ordinary-node-web-service.md)

## Related Intake

- [Desktop Node 与 Metrics Node 产品独立性](../intake/2026-08-30_desktop-metrics-product-independence.md)

