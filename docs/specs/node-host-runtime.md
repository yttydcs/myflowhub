# NodeHost Runtime

## Status

Accepted design contract；首批实现和产品迁移进行中。本文定义完整 Go 节点的稳定组合边界，不取代 wire、Node、Resource 或产品 feature 规范。

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
| configured | none | leaf | Desktop、MetricsNode |
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

`Host.Resources()` 暴露绑定本 Node owner 的 Registry facade。固定资源通常在 `Start` 前注册；插件或动态设备可以在 running 时注册/注销，Catalog 更新仍由 Registry 原子维护。

Variable 便捷声明必须显式提供 local name、content type、schema、read permission、payload limit 和 initial payload；owner 自动取 Host NodeID。默认 Variable 对远端只读；只有显式 write permission 才能开放远端写入。

便捷声明不得：

- 接受 foreign owner；
- 绕过 descriptor/payload 校验；
- 绕过 Registry 重名检查和 `system/catalog` 更新；
- 改变 `Set` 的 revision、watcher、snapshot-first subscription 或远端事件语义。

## Product and Platform Composition

- Desktop：拥有自己的 Profile identity 和 state；Legacy 与已经 granted 的 authority Profile 都使用 Parent-only NodeHost，不因 UI 或产品名获得 Listener/relay 权限。尚无 Node ID 的 authority Profile 只打开窄 Enrollment bootstrap；Grant 持久后先关闭 bootstrap，再以同一受保护凭据创建 NodeHost。
- MetricsNode：拥有独立 NodeHost，在 `Start` 前注册指标、配置、控制和通知资源；不依赖 Desktop。
- Hub：后续在 NodeHost 上组合 Management/File/Notification 等 feature；当前迁移延期。
- Agent Gateway：后续作为普通 NodeHost 产品，在外层实现 Web/API/MCP、token 权限交集和审计；不成为树内特权。

NodeHost 不导入 Wails、Kotlin、Android API、产品 controller 或具体 Transport。Windows/Linux/macOS 可直接组合 Host；Android/iOS 通过 gomobile 和薄 facade 复用它。

依赖方向仍要求 `sdk/go` 与普通 bindings 不依赖 `host/`。Android 的唯一受控 import/composition 例外是 `sdk/bindings/android/host.go`：它作为单文件 gomobile platform composition root，可以组合通用 leaf NodeHost 与延期迁移的 in-process Hub。为保留 gomobile ABI，导出的 `android.Client` 仍是平台 lifecycle wrapper；`client.go` 不直接导入 Host，而是通过 `host.go` 的 factory 间接启动并关闭其唯一 leaf Host。该 wrapper 内部持有的通用 `bindings.Client` 才是 non-owning operation facade；例外不得扩散到其他 SDK/binding 文件。

移动平台外壳继续负责 Foreground Service/Activity、通知、权限、Doze/network callback 和进程生命周期。Keystore 通过 `IdentityStore`，RFCOMM 通过 Driver/provider，collector/actuator 通过产品 adapter 注入。每个新进程从 disconnected 开始，不能把持久 UI 状态当作 live session。

不能嵌入 Go 的 C/ESP32/MicroPython 环境保留轻量 runtime，以协议和 contract tests 对齐。浏览器默认通过 Agent Gateway API；WASM/WebSocket 完整 Node 需要单独决策。

## Error and Security Invariants

- 未认证连接不能成为树边；Host composition 不改变 authority 规则。
- 同一 state directory 不允许两个运行 Host 并发写。
- 运行后不得静默更换 Parent identity、trust 或 Node identity。
- 平台权限、产品名、UI 按钮和 Transport 类型都不能推导 Resource permission。
- Host status 只暴露 endpoints、连接状态、重试摘要和最后错误，不含 secret 或业务 payload。
- 产品 controller 注册失败必须按明确 ownership 关闭已注册 controller 和 Host。

## Compatibility

首批迁移保持 wire envelope、Resource descriptor/schema、NodeID、持久 identity、Profile/config schema 和 Parent supervision 行为。旧 runtime-owning SDK/binding API 可以暂时作为明确标记的兼容入口，但不能成为新产品的 canonical path，也不能静默改变关闭所有权。Desktop authority Enrollment 不再是 post-Grant 例外：bootstrap 只负责 MFHE 与 Grant 持久化，普通运行统一进入 NodeHost；旧 owning API 仅为未迁移调用方保持源码兼容。Android 的 Host 构造实现集中在 `sdk/bindings/android/host.go`；既有 `android.Client` 与 `android.Host` 可以作为 gomobile lifecycle wrapper 暴露 Start/Close，但其内部 portable SDK/bindings Client 始终 non-owning。

Android in-process Hub、现有 `host/hub` 和 ClipboardNode 不在首批迁移范围；其存在不改变本文的长期边界。

## Verification

- memory Driver 覆盖 leaf、relay、root、offline；
- 覆盖 multiple Listener rollback、Parent reconnect/reparent、Start/Close race 和 goroutine cleanup；
- 证明 `Host.Client()` pointer identity，以及 Client-local close 不影响 Host；
- 证明本地调用短路、远端调用仍走 Node route 与 Session queue；
- 证明 Start 前/后资源注册、Variable Set、Catalog 和远端 subscription；
- 证明 credential-backed Host 不创建 identity 文件、拒绝未 granted/冲突凭据，并让重复 Connect 复用同一 ParentSupervisor；
- 产品门禁覆盖 Desktop Wails、Metrics TCP/process、gomobile AAR 与 Android Gradle；设备不可用时明确记为 unavailable。

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
