# 2026-08-30 NodeHost、SDK 与产品运行时边界讨论

## Source

- 来源：用户在当前 Codex 会话中的直接讨论与确认。
- 日期：2026-08-30。
- Canonical repository：`repo/MyFlowHub`。
- Execution branch：`refactor/nodehost-runtime`。
- Execution worktree：`worktrees/nodehost-runtime`。

## Request Text / Source-preserving Summary

讨论从 Desktop 与 MetricsNode 是否合并开始。用户确认二者应保持完全独立：它们是两个 Node、两个进程和两个安装包，产品名不赋予网络特权。MetricsNode 不应因为只想采集或执行而被迫安装 Desktop；Desktop 也不应因为共享底层能力而隐式成为 MetricsNode 的资源 owner。

用户进一步确认 Agent Gateway 也只是普通 Node。它未来作为独立 Web 服务，把网络操作以 API/MCP 方式暴露给一个或多个 Codex/Agent，并通过不同准入 token 赋予不同权限；它不是树内特权产品，本轮不实现。

围绕节点运行时，用户希望：

- 像 `new` 一个 server 对象一样方便地创建节点；
- Host 统一拥有身份、Node、可选父连接、可选 Listeners 和生命周期；
- `client := host.Client()` 直接取得可发送 Catalog、Snapshot、Operate、Publish、Subscribe 等请求的 SDK Client；
- Host Client 与手工 `sdk.NewClient(host.Node())` 使用同一 Node 和发送路径，不建立第二条连接或第二套队列；
- 不再为“没有 Listener 的 Host”单独维护 `nodeclient` 包装；
- 调用方能方便声明本节点 Variable，并在后续更新值；
- Android 等平台复用同一个节点模型，平台外壳只负责系统生命周期、权限、密钥和硬件适配。

## Confirmed Architecture Direction

### One neutral Host

新增纯 Go `host/nodehost` 作为完整节点的推荐组合入口。一个 Host 恰好拥有一个 `runtime/node.Node`、一个绑定该 Node 的非 owning `sdk.Client`、最多一个 ParentSupervisor、零个或多个 Listeners，以及该 Node 的 Resource Registry。

配置组合表达角色，而不是使用不同产品类型：

| Parent | Listeners | Role |
| --- | --- | --- |
| yes | none | leaf |
| yes | one or more | relay |
| none | one or more | root |
| none | none | offline/local |

### SDK is an operation facade

SDK 回答“应用如何使用节点能力”，NodeHost 回答“节点如何运行”。SDK 负责稳定的 typed operation、payload/schema、错误和跨语言 facade；不负责打开持久状态、创建身份、连接 Parent、监听端口或关闭 Host。

`host.Client()` 返回 Host 创建的同一个 `*sdk.Client`。Catalog、Operate、Subscribe 等仍进入同一 `Node`，本地目标短路，远端目标继续走 topology routing、`Session.Send`、有界队列和 Transport。

### Products stay independent

- Desktop 是 Parent-only 的资源工作台，不默认监听或中继；
- MetricsNode 是独立 Parent-only 采集/执行节点，在网络启动前注册自己的资源；
- Hub 后续可在 NodeHost 上组合 Management/File/Flow 等能力，但本轮不迁移；
- Agent Gateway 后续是独立普通 Node + Web/API/MCP 外壳，本轮延期；
- 产品名不决定 identity、authority、Resource ownership 或权限，配置和 policy 才决定。

### Thin platform adapters

Android/iOS 通过 gomobile 复用纯 Go NodeHost。Kotlin/Swift 外壳拥有 Service/Activity、前后台、通知、系统权限和网络回调；Keystore、RFCOMM、collector/actuator 等通过接口注入。Android MetricsNode 使用相同 Host；现有 Android in-process Hub 的迁移延期，不在本轮伪装成已完成。

浏览器通常通过 Agent Gateway API 访问；C、ESP32、MicroPython 等不能直接嵌入 Go 的环境继续保留轻量实现，并以 wire/contract tests 对齐。

## Resource Declaration Direction

Variable、Registry、revision、snapshot-first subscription 和更新通知已存在。新增的便捷入口只减少 descriptor boilerplate：owner 自动绑定 Host NodeID，name、schema、content type、read permission 和 payload limit 仍显式；远端写入必须显式声明 write permission。便捷入口不得绕过 Registry、Catalog、payload validation 或 authority。

## Options Considered

1. 合并 Desktop 与 MetricsNode：拒绝，会混合产品职责、Node identity、安装与资源归属。
2. 每个产品继续自行装配 state、Node、SDK 和 ParentSupervisor：拒绝，会重复生命周期、回滚和错误语义。
3. 让 SDK Client 创建并拥有 Node：拒绝，会混合运行时所有权与操作面。
4. 为 leaf、relay、root 分别建立 Host 类型：拒绝，角色只是 Parent/Listeners 配置组合。
5. 把 Kotlin/Wails/具体 Transport 放进 NodeHost：拒绝，会污染平台中立边界。
6. 纯 Go NodeHost + 非 owning SDK Client + 薄平台/产品适配器：采用。

## Scope Confirmed for the First Migration

- 新增 NodeHost、owner-bound Variable helper 和 ownership tests；
- 收敛 SDK attached Client 的关闭语义；
- 首批迁移 Desktop、Metrics Windows/CLI、Metrics Android 和通用 Android leaf Client；
- 保持 wire、资源 schema、Profile/identity 和产品行为兼容。

Hub、ClipboardNode、Agent Gateway、浏览器完整 Node、Embedded runtime 和安装发布在后续独立阶段处理。

## Stable Docs Impact

- Requirement：澄清统一运行时还包括节点组合和产品无特权原则。
- Feature：澄清 Desktop、MetricsNode、Android 共享 Host contract，但产品和平台职责独立。
- Spec：新增 [NodeHost runtime](../specs/node-host-runtime.md)，并校准 lifecycle 与 Resource helper。
- Decision：新增 [通用 NodeHost、非 owning SDK Client 与薄平台适配](../decisions/2026-08-30_generic-node-host-and-non-owning-sdk-client.md)。
- Change/lesson：实现、重门禁与归档已完成，见 [NodeHost runtime 与产品边界收敛](../change/2026-08-31_nodehost-runtime-and-product-boundaries.md) 和 [Android Gradle daemon loopback 不可用](../lessons/android-gradle-loopback-daemon-unavailable.md)。
