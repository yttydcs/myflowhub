# NodeHost Runtime

## Status

Accepted design contract；Desktop、Metrics、Clipboard 已使用 NodeHost，Hub 迁移延期。本文定义完整 Go 节点的稳定组合边界，不取代 wire、Node、Resource 或产品 feature 规范。

## Purpose

`host/nodehost` 是创建和运行普通 Node 的平台中立入口。它统一持久身份、一个 Node、可选父连接、可选 Listeners、同一 SDK Client 和关闭顺序，使产品不再重复装配运行时。

NodeHost 不是 Hub 基类、产品容器或权限捷径。产品名不赋予网络角色；Node identity、Parent/Listeners 配置和 policy 决定拓扑与权限。

## Ownership Model

一个 Host 恰好拥有：

- 一份持久 auth state，以及 Legacy `IdentityStore` 或只读 `NodeCredentialSource` 提供的身份；
- 一个 `runtime/node.Node`；
- 一个绑定该 Node 的非 owning `*sdk.Client`；
- 最多一个 ParentSupervisor；
- 零个或多个由 Host 启动的 Listeners；
- 该 Node 的 Resource Registry；
- Host 创建的后台任务、取消函数和状态观察器。

`Host.Node()`、`Host.Client()` 与 `Host.Resources()` 始终指向同一个 Node。重复调用 `Client()` 不创建 Node、身份、连接或发送队列。

SDK Client 不拥有 Host。关闭或清理 Client-local subscription 不得关闭 Node、ParentSupervisor 或 Listener；Host 生命周期只能由 Host owner 结束。

## Topology Roles

角色由配置组合产生：

| Parent | Listeners | Effective role | Typical product |
| --- | --- | --- | --- |
| configured | none | leaf | Desktop、MetricsNode、ClipboardNode |
| configured | one or more | relay | 可同时接入上游和子节点的普通 Node |
| none | one or more | root | authority root |
| none | none | offline/local | 本地资源和测试 |

不存在独立的 `nodeclient`、`LeafHost` 或 `RelayHost` 运行时。是否监听不改变 Node 的协议、资源或权限模型。

## Construction and Lifecycle

### New

`New` 负责验证不变量、打开持久状态并创建 Node 与 attached SDK Client。它不建立父树边、不监听端口，也不向网络宣告 healthy。调用方可以在 `Start` 前注册固定资源，使第一次可见 Catalog 完整。

Legacy 输入至少包括 state directory、NodeID、可选 IdentityStore、Node limits、可选 Parent 和 Listener 配置。credential-backed 输入提供 state directory、`NodeCredentialSource`、Parent endpoint/Driver 与可选的兼容缓存约束。两种身份模式不能同时隐式生效；显式 NodeID、Parent NodeID 或公钥若存在，必须与 credential 完全一致。Parent identity、public key、permit、endpoint、Driver 与 Listener 配置必须在启动外部工作前验证。

`NodeCredentialSource` 只读取一个已经注册的完整凭据：Node identity、Grant 绑定的直接父节点、Authority provenance 和 Enrollment ID。NodeHost 只消费身份与直接父信任；Authority provenance 不自动授予 Resource 权限。`missing/device/pending`、签名失败、key/NodeID 不匹配或 Profile 约束冲突都在 Listener/Dial 前 fail closed。credential-backed state 会初始化普通 trust/policy/admission 文件，但不得生成或复制 identity 文件。

### Start

`Start` 是单次状态转换：

1. 冻结影响 identity、Parent 和 Listener ownership 的配置；
2. 按配置启动 Listeners；
3. 启动 Parent supervision；
4. 仅在启动约束满足后进入 running。

多个 Listener 中途失败时，已启动 Listener 按逆序关闭。Parent supervision 启动失败时，同次 Start 激活的外部入口也必须回滚。失败返回包含 Listener index、endpoint 或 Parent 摘要的可行动错误，但不得泄漏 private key、permit 或 payload。

一个 stopped 或启动失败后完成回滚的 Host 不被隐式重新启动；需要重试的产品创建新实例。Desktop 继续遵守失败连接后创建候选 runtime、成功后原子替换的规则。

### Close

`Close` 幂等并遵循 ownership 的逆序：停止接纳新工作、取消 Parent supervision、关闭 Listeners、取消 Host-owned tasks、关闭 Node，并有界等待。未启动、部分启动失败和并发关闭都必须安全。

## SDK Operation Path

SDK 表达 Catalog、Snapshot、Operate/Invoke、Publish、Subscribe 和 Session 等操作。attached Client 直接调用 Host 的同一 Node：

```text
host.Client()
      ↓
sdk.Client operation
      ↓
runtime/node.Node
      ├── local target → local Registry/handler
      └── remote target → topology route → Session.Send → Driver/Pipe
```

NodeHost 和 SDK 不增加第二套数据面队列、额外序列化或每次请求新连接。批处理、背压、优先级、路由和 Transport 优化属于 Node/Session/Driver 层。

## Resource Registration

`Host.Resources()` 暴露绑定本 Node owner 的 Registry facade。固定资源通常在 `Start` 前注册；产品的动态能力可以在 running 时注册/注销，Catalog 更新仍由 Registry 原子维护。

Variable 便捷声明必须显式提供 local name、content type、schema、read permission、payload limit 和 initial payload；owner 自动取 Host NodeID。默认 Variable 对远端只读；只有显式 write permission 才能开放远端写入。

便捷声明不得：

- 接受 foreign owner；
- 绕过 descriptor/payload 校验；
- 绕过 Registry 重名检查和 `system/catalog` 更新；
- 改变 `Set` 的 revision、watcher、snapshot-first subscription 或远端事件语义。

## Product and Platform Composition

- Desktop：拥有自己的 Profile identity 和 state；Legacy 与已经 granted 的 authority Profile 都使用 Parent-only NodeHost，不因 UI 或产品名获得 Listener/relay 权限。尚无 Node ID 的 authority Profile 只打开窄 Enrollment bootstrap；Grant 持久后先关闭 bootstrap，再以同一受保护凭据创建 NodeHost。
- MetricsNode：拥有独立 NodeHost，在 `Start` 前注册指标、配置、控制和通知资源；不依赖 Desktop。
- ClipboardNode：拥有 Parent-only NodeHost，在 Start 前注册剪贴板资源；使用 attached SDK 和只读 ConnectionStatus 恢复 peer 订阅，保留现有持久身份、配置与历史。
- Hub：后续在 NodeHost 上组合 Management/File/Notification 等 feature；当前迁移延期。
- Agent Gateway：后续作为普通 NodeHost 产品，在外层实现 Web/API/MCP、token 权限交集和审计；不成为树内特权。

NodeHost 不导入 UI 框架、平台 API、产品 controller 或具体 Transport。产品通过凭据、Driver 和 adapter 提供平台能力。`sdk/go` 与 bindings 不依赖 `host/`；原 Android 单文件 import 例外随实现删除。

Android、C/ESP32/MicroPython 的旧实现已移除，平台重启需按[重新设计待办](../requirements/mobile-embedded-redesign.md)另行确定方案。浏览器的完整节点运行方式仍需单独决策；现有 Clipboard Web 仅是 UI 预览。

## Error and Security Invariants

- 未认证连接不能成为树边；Host composition 不改变 authority 规则。
- 同一 state directory 不允许两个运行 Host 并发写。
- 运行后不得静默更换 Parent identity、trust 或 Node identity。
- 平台权限、产品名、UI 按钮和 Transport 类型都不能推导 Resource permission。
- Host status 只暴露 endpoints、连接状态、重试摘要和最后错误，不含 secret 或业务 payload。
- 产品 controller 注册失败必须按明确 ownership 关闭已注册 controller 和 Host。

## Compatibility

保留 wire envelope、Resource descriptor/schema、NodeID、持久 identity、Profile/config schema 和 Parent supervision 行为。Clipboard 复用原持久目录。

SDK 源码 API 有意收敛：删除 Go SDK NewClient、Close、Connect、ConnectManaged 和 owning Connection，以及 bindings 的 NewClient/NewEnrollmentClient、TrustParent、StartTCP/StartRFCOMM/StartEnrolledTCP。调用方使用 NodeHost.New → 注册资源 → Start，获取 Host.Client 与 ParentStatus；Host owner 调用 Close。

SDK Client 只有资源操作，不提供节点关闭方法。bindings.NewAttachedClient 包装现有 SDK；其 Close 只取消 facade 订阅。只读 ConnectionStatus 使用 Snapshot/WaitChange，不能启动或停止父连接。

Desktop authority Enrollment 继续由独立 EnrollmentBootstrap 执行 MFHE 和 Grant 持久化，随后关闭 bootstrap 并创建 NodeHost。Legacy Join 与现有 host/hub 保留；后者迁移是独立工作。

## Verification

- memory Driver 覆盖 leaf、relay、root、offline；
- 覆盖 multiple Listener rollback、Parent reconnect/reparent、Start/Close race 和 goroutine cleanup；
- 证明 `Host.Client()` pointer identity，以及 bindings-local Close 不影响 Host，SDK 不再暴露节点生命周期方法；
- 证明本地调用短路、远端调用仍走 Node route 与 Session queue；
- 证明 Start 前/后资源注册、Variable Set、Catalog 和远端 subscription；
- 证明 credential-backed Host 不创建 identity 文件、拒绝未 granted/冲突凭据，并让重复 Connect 复用同一 ParentSupervisor；
- 产品门禁覆盖 Desktop Wails、Metrics TCP/process、Clipboard 双节点同步、TCP/process、持久状态保留、失败回滚及 Flutter analyze/test/build。

## Related

- [统一节点运行时 requirement](../requirements/unified-node-runtime.md)
- [Operational lifecycle](operational-lifecycle.md)
- [节点树、链路与资源架构](node-tree-link-resource-architecture.md)
- [Resource Platform v2](resource-platform-v2.md)
- [仓库与模块边界](repository-and-module-boundaries.md)
- [通用 NodeHost、非 owning SDK Client 与薄平台适配 ADR](../decisions/2026-08-30_generic-node-host-and-non-owning-sdk-client.md)
- [本轮讨论 intake](../intake/2026-08-30_node-host-runtime-and-product-boundaries.md)
- [NodeHost runtime 与产品边界收敛 change](../change/2026-08-31_nodehost-runtime-and-product-boundaries.md)
- [NodeHost、Enrollment 与 Profile 生命周期收敛](../change/2026-09-01_nodehost-enrollment-profile-convergence.md)
- [Android Gradle daemon loopback 不可用 lesson](../lessons/android-gradle-loopback-daemon-unavailable.md)

- [平台退役与 SDK 收敛决策](../decisions/2026-09-06_retire-mobile-embedded-and-owning-bindings.md)
