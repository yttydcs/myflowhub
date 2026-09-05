# 统一节点运行时

## Background

MyFlowHub 曾通过多个 Git 仓库、多个 Go module 和多个 SubProto 分别表达协议、连接、权限、变量、主题、流、指令和运行时装配。canonical monorepo 已完成源码收敛，Desktop、MetricsNode 与 Clipboard 现已通过 NodeHost 统一运行时；Android 和嵌入式旧实现已退役并延期重新设计。

系统需要以一棵权威节点树为运行期基础，将 Variable、Stream、Command 和 Subscription 提升为一级概念，同时保持 TCP、Bluetooth、QUIC、串口等链路承载可扩展。

## Goal

- 用统一节点树表达连接、路由和运行期 authority。
- 节点拥有资源；Variable 和 Stream 可订阅，Command 可调用。
- 所有 Transport 通过统一 LinkSession 契约接入，不改变上层权限和资源语义。
- 在单一 canonical monorepo 中完成第一方协议、运行时、SDK 与当前支持应用的原子演进。
- 为完整 Go 节点提供平台中立的 NodeHost，使产品共享运行时组合但保持独立身份、进程、资源归属和安装边界。
- 降低日常开发、集成验证和发布同步成本。

## Scope

包含：

- 新 protocol envelope 和内部契约；
- Pipe/Connection、LinkSession 和可插拔 Transport；
- 单父节点树、路由、LCA 裁决和 reparent；
- 身份认证、请求阶段与控制阶段；
- Resource Registry、Variable、Stream、Command；
- Subscription 的租约、生命周期、聚合和背压；
- Hub 与 Go client 的最小纵向闭环；
- 通用 NodeHost、非 owning SDK operation Client、资源声明和薄平台适配；
- monorepo 目录、package、测试和文档治理；
- 现有第一方应用的运行时收敛；Android 与嵌入式另列[重新设计待办](mobile-embedded-redesign.md)。

不包含长期兼容旧 Go module path、SubProto API 或 wire 协议。

## Scenarios

1. 一个节点通过任意受支持 Transport 与唯一父节点建立经过认证的 LinkSession，并加入节点树。
2. 一个节点订阅同子树或跨子树节点拥有的 Variable，先收到当前快照，再收到后续变化。
3. 一个节点订阅 Stream，只接收约定起点后的事件，并在慢消费者时获得明确的背压或缺口语义。
4. 一个节点调用同子树或跨子树 Command，请求在 LCA 或可裁决节点完成权限判断后沿控制链下发。
5. 父节点直接控制子树资源时，子节点接受来自当前已认证父节点的显式控制消息，但仍执行输入、限流和业务校验。
6. 节点更换父身份时触发 reparent，旧父控制、旧路由和跨旧父订阅失效。
7. 同一套运行时语义可以通过内存、TCP 和其他 Transport 适配器验证，而业务层不出现 Transport 类型分支。
8. Desktop、MetricsNode 或其他产品可以创建同一 Host 类型，并仅通过 Parent/Listeners 配置成为 leaf、relay、root 或 offline Node；产品名不产生网络特权。

## Functional Requirements

- 每个非根节点在一个 authority 域内只能有一个当前有效父节点。
- 连接只有在对端认证和角色激活后才能成为树边。
- 父节点对其子树具有控制权；子节点只信任当前父节点下发的显式控制阶段消息。
- 来自子连接的主体必须能够由该子节点或其后代关系证明。
- 目标不在本地下游的请求必须沿父链上送，并在 LCA 或可裁决节点完成授权。
- Variable 必须具有当前值和单调修订，并支持快照加变化订阅。
- Stream 必须具有事件序列或等价缺口识别能力，不强制持有当前值。
- Command 必须支持请求标识、输入校验、结果、错误和超时。
- Subscription 必须记录订阅者、资源、租约、授权范围、状态和拓扑/策略世代。
- 订阅、控制和数据队列必须有边界，不能因慢消费者无限增长。
- Transport Driver 只负责承载差异，上层资源与权限代码不得依赖具体 Transport 类型。
- 一个 NodeHost 恰好拥有一个 Node、一个绑定该 Node 的非 owning SDK Client、最多一个 ParentSupervisor 和零个或多个 Listeners。
- NodeHost 必须能从只读、已验证的 credential source 打开既有身份和父信任；该模式不得在凭据缺失、pending、损坏或冲突时生成替代身份，也不得创建第二份 identity 持久化。
- SDK Client 只表达资源操作，不打开持久状态、不创建树边、不监听端口，也不关闭其所属 Host。
- NodeHost 必须允许在网络启动前注册固定资源，并允许运行期按 Registry 契约动态注册或注销资源。
- 平台能力必须通过身份存储、Driver 与产品 adapter 注入；Android/iOS 的具体集成方式需在恢复支持前另行确定；NodeHost 不依赖原生 UI 或 OS 生命周期 API。
- Desktop、MetricsNode、Hub、Agent Gateway 等产品的 Node identity、state、process、Resource owner 和权限相互独立。
- 第一方源码必须收敛到 canonical monorepo；内部能力默认使用 package 而不是独立 Go module。

## Non-functional Requirements

- 安全：未认证连接不得进入路由或权限索引；控制阶段不能由未经授权的下级自行声明。
- 正确性：断连、重连和 reparent 必须显式清理或重建相关路由、pending 请求和订阅状态。
- 可维护性：协议、运行时、Transport、Feature、Host、SDK 和 App 依赖方向清晰且无循环。
- 可组合性：网络角色由 Parent/Listeners 配置表达，不为 leaf、relay、root 或平台复制运行时实现。
- 可测试性：核心语义可使用内存 Transport 完整验证，并由真实 TCP 运行集成闭环。
- 性能：队列、重组缓冲、订阅状态和历史事件均有明确上限；控制消息不被大体积数据长期阻塞。
- 可追溯性：所有旧仓迁移必须记录来源 commit/tag、新目录和验证证据；原架构文档和历史仓库不得被破坏性删除。

## Edge Cases

- 同一 NodeID 同时从多个连接声明父子关系。
- 子节点伪造 sibling 或其他子树 SourceID。
- 未裁决请求伪造 control phase。
- 来自父连接的未知目标被错误回送父链形成环路。
- reparent 后旧父延迟消息到达。
- Variable 更新合并期间订阅者重连。
- Stream 队列溢出、事件缺口和消费者恢复。
- Command 重复请求、处理超时和响应晚到。
- 低 MTU Transport 的分片超限或中途断开。

## Acceptance Criteria

### 第一阶段纵向闭环

- canonical monorepo 具备一个根 Go module、稳定目录和依赖边界检查。
- 至少有内存与 TCP 两种 Transport 通过相同 LinkSession contract tests。
- 三层以上测试节点树能够加入、断开、重连和 reparent。
- 跨子树 Variable 订阅能够完成快照、变化通知、权限允许与拒绝。
- 跨子树 Stream 订阅能够传递事件并暴露有界背压/缺口结果。
- 跨子树 Command 能够完成 LCA 裁决、下行执行、成功、拒绝和超时返回。
- 子节点拒绝非当前父节点或未显式裁决的下行控制。
- 集成测试不依赖旧 SubProto handler 或旧 module 发布链。

### 完整迁移

- Desktop、Hub、Windows 节点应用均使用新契约；已退役平台不再是当前验收对象。
- 旧 SubProto、旧多 module 装配和内部 `replace` 链退出生产构建。
- 旧仓来源、迁移状态、验证证据和最终归档状态完整可查。
- 只有在完整迁移验证后，旧仓才能停止发布或设置只读。

### 通用 Host 收敛

- memory contract 覆盖 Parent-only、Parent + Listeners、Listeners-only 和无外部链路四种组合。
- `host.Client()` 每次返回绑定同一 Node 的非 owning Client；本地与远端操作继续使用现有 Node 路由和 Session queue。
- 受保护 Enrollment Grant 原子保存后，bootstrap 必须关闭；已注册节点由同一 NodeHost/attached Client 路径运行，重启不得重新 Enrollment 或要求 Permit。
- Desktop 与 MetricsNode 首批迁移后仍是两个独立 Parent-only Node，且不互相依赖安装或资源 ownership。
- Clipboard 使用 NodeHost、attached SDK 和只读连接状态，迁移保留原身份、配置、历史、同步和退出行为；Android 与嵌入式重启时重新定义平台验收。

## Related Features

- [Hub](../features/hub.md)
- [Desktop](../features/desktop.md)
- [Android](../features/android.md)
- [MetricsNode](../features/metrics-node.md)
- [ClipboardNode](../features/clipboard-node.md)
- [File transfer](../features/file-transfer.md)
- [Flow](../features/flow.md)

## Related Specs

- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [仓库与模块边界规范](../specs/repository-and-module-boundaries.md)
- [Wire Protocol vNext](../specs/wire-protocol-vnext.md)
- [Resource Model vNext](../specs/resource-model-vnext.md)
- [Subscription vNext](../specs/subscription-vnext.md)
- [Command vNext](../specs/command-vnext.md)
- [Operational lifecycle](../specs/operational-lifecycle.md)
- [NodeHost Runtime](../specs/node-host-runtime.md)
- [Resource catalog](../specs/resource-catalog.md)
- [Notification vNext](../specs/notification-vnext.md)
- [File transfer vNext](../specs/file-transfer-vnext.md)
- [Flow vNext](../specs/flow-vnext.md)

## Related Decisions

- [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)
- [通用 NodeHost、非 owning SDK Client 与薄平台适配](../decisions/2026-08-30_generic-node-host-and-non-owning-sdk-client.md)

## Related Changes

- [NodeHost、Enrollment 与 Profile 生命周期收敛](../change/2026-09-01_nodehost-enrollment-profile-convergence.md)
- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
