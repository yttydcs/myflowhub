# Resource Collections and Actions

## Status

Current implementation contract。Collection protocol、multi-capability runtime Resource、Flow definitions/runs、
opt-in read-only filesystem provider、typed Go SDK、generic bindings/generated contracts、Desktop shared action seam
与 Collection/content renderers 已实现，并已完成 full product、Wails package 与真实 browser/packaged GUI `QA01`。

| Area | Current status |
| --- | --- |
| Collection schemas/runtime handler | current |
| Flow definitions/runs Collections | current |
| Filesystem `get/list/read` provider | current、opt-in |
| Go SDK/bindings/generated contracts | current |
| Desktop Resource action derivation/entry seam | current |
| Desktop Collection/content renderers | current |
| Full product/package evidence | current；`QA01` passed |

## Core concepts

- Resource 是网络中的独立寻址、发现、授权和审计边界，以 `(owner NodeID, local name)` 标识。
- Capability 是调用方可以对 Resource 执行的操作。对已有 Resource 的普通操作默认不是新的 Resource。
- Collection 是一种 Resource contract：Resource 自身保持一个全局身份，同时管理一组可枚举、可定位的
  内部 member。
- Collection member 默认只有 collection-scoped identity，不自动进入 Node catalog，也不自动成为全局
  Resource。
- Resource name 中的 `/` 只允许 Desktop 派生 presentation namespace；纯 namespace 不是 Resource、
  Collection、permission scope 或 routing hop。

Collection 不等同于磁盘目录。它可以由本地目录、对象存储、数据库记录、Flow definition、运行实例、
设备集合或动态生成内容提供。provider 的物理路径、表名、bucket 和其他部署细节不得成为网络 ResourceID。

## Action ownership

对现有对象的行为声明为该 Resource 的 Capability。例如取消运行是 `flow/runs` 的 `cancel` capability，
不再默认建模为 `flow/cancel` Command Resource。

只有当一个操作自身形成稳定、独立的寻址、schema、权限、生命周期和审计边界，并且没有自然目标 Resource
时，才建模为独立 Command Resource。多个紧密相关的系统操作可以由一个 Service Resource 提供多个
capabilities，例如 `system/admission` 的 `issue` 与 `revoke`。

任意操作系统命令执行不得因为属于 Command Resource 就获得隐式安全性。不得默认暴露接受任意命令字符串的
远程 shell；预定义任务或受控执行服务必须限制 executable、arguments、working directory、environment、
identity、deadline、output size 和 cancellation，并继续经过 Resource policy 与 owner validation。

## Collection contract

Collection descriptor 必须显式声明其支持的 capabilities 和对应 schema，不因类型名自动获得操作。
`list` 与 `get` 是第一阶段可复用的通用 CapabilityIDs；`read/create/update/archive/run/subscribe/cancel` 等
属于 provider/domain contract。实际 Collection 只声明其真实支持的子集。

第一阶段使用扩展 type ID `mfh.collection`，并定义有界的 `list request`、`member descriptor` 与分页
`collection page` schema。`list` 是通用发现 seam；`get/read/create/update/delete/run/cancel/subscribe` 等
仍按 Collection descriptor 分别声明输入、输出或事件 schema，不能从 type 名隐式推断。

通用 wire payload 至少固定以下语义：

- list request：version、可为空的 provider-scoped parent、opaque cursor 与 positive bounded limit；
- member request：version 与 non-empty bounded opaque member key；
- member descriptor：bounded key/kind/label、可选 content type/schema、sorted unique capability subset 与
  bounded attributes；它不包含 owner NodeID 或 ResourceID；
- collection page：version、`1..9007199254740991`（`2^53-1`）范围内且可被 JavaScript/IEEE-754 客户端精确
  往返的 revision、请求 parent、deterministically ordered members 与 opaque next cursor；provider 在 revision
  耗尽时必须明确失败，不能回绕或复用旧 revision。

所有字符串、member 数、attributes、cursor 和 response payload 必须使用 protocol 导出的 hard limits；调用方
不得解析 cursor，provider 不得返回未在 Collection descriptor 声明的 member capability。

每次 member operation 必须携带由 provider schema 约束的 member locator。locator 可以是 path、stable ID、
object key 或组合键，但必须有界、可验证且只在当前 Collection 内解释。对文件系统 provider，owner 必须在
解析 traversal、symlink/junction 和大小写语义后验证目标仍位于配置 root 内。

授权分为两层：

1. 网络 authority 继续按 exact `subject + ResourceID + CapabilityID` 裁决；descriptor 中的 `Permission`
   兼容字段不是第二个授权 key；
2. Collection owner 根据同一 authority context 和 provider policy 校验 member locator、root/domain scope 和
   业务所有权。

UI namespace、presentation hint 和物理存储位置都不得扩大权限。Collection member 只有在需要独立全局授权、
长期跨系统引用、独立订阅、独立生命周期或独立 owner 时，才提升为普通 Resource 并进入 catalog。

## Filesystem example

三个需要分别授权的本机目录应注册为三个 Collection Resources，而不是为每个文件注册 Resource：

```yaml
resources:
  - name: storage/a
    provider: filesystem
    root: D:\data\a
  - name: storage/b
    provider: filesystem
    root: E:\shared\b
  - name: storage/c
    provider: filesystem
    root: C:\workspace\c
```

网络只发现 `storage/a`、`storage/b`、`storage/c`；物理 root 保留在 owner 本地配置。第一阶段严格保持
“一个配置 root 对应一个 Collection Resource”，只声明 `list/get/read` 且默认、实际均为只读。虚拟
multi-root named mounts、write/delete 和大文件 download/session 延期到 `FS02`。未来只有多个 root 始终共享
权限、owner 与生命周期时，才可在一个 Collection 内使用 named mounts；需要分别授权时仍保持多个 Resources。

当前 provider 由产品显式调用等价于 `filesystem.Register(node, []Mount)` 的 API；Core、NodeHost、Desktop 和
Metrics 不默认 import 或自动挂载它。注册前会原子校验全部 mounts：Resource name 与 canonical root 唯一、root
absolute/existing/real directory，root 本身的 symlink/junction 拒绝；中途注册失败按反向顺序回滚。

locator 只接受 provider-scoped `/` 分隔相对 key，拒绝 absolute、backslash、空/dot/parent segment 与 NUL。
每次 list/get/read 都重新验证 root identity、`EvalSymlinks` 后的 canonical containment；list 使用 hard bounded
scan，并用完整 SHA-256 fingerprint 检测每个 parent 的快照变化。provider 以容量 1024 的 LRU 有界跟踪活跃
parents：仍在跟踪时，相同 snapshot 复用 revision，变化后 revision 递增且不回绕；parent 被淘汰后再次访问可从
1 重建，因此 numeric revision 不承诺跨淘汰永久单调。cursor 同时绑定 parent、revision 与完整 fingerprint；
重建后 snapshot 相同仍可接受，内容不同则即使 numeric revision 重用也明确 stale。get 只读取 metadata，不遍历
完整文件。read 只接受 regular file，
请求与 mount 双重限制且 decoded content 最大 128 KiB，超限整体失败而不截断；open 后执行 file identity 和
post-read revision/size 复核。有效 UTF-8 文本返回 `utf-8`，其他内容返回 base64，MIME 如实保留；物理 root
不会进入 descriptor、wire payload 或正常错误。

HTML/SVG 在 wire 上只是非可执行数据。当前 Desktop renderer 会转义显示文本、结构化 JSON，解码 allowlisted
PNG/JPEG/GIF/WebP raster，并为未知或不支持的 binary 显示明确 fallback；HTML 只作为转义文本，SVG 不嵌入，
二者都不执行。

## Flow example

Flow 默认提供两个 Collection Resources：

- `flow/definitions` 管理可执行定义，第一阶段提供 `list/get/create/update/archive/run`；
- `flow/runs` 管理执行实例，第一阶段提供 `list/get/subscribe/cancel`。

调用 `flow/definitions.run(definitionID, input)` 产生 `flow/runs` 中的 member。Definition 与 Run 默认不逐项
进入全局 catalog；只有满足独立 Resource 提升条件时才注册为普通 Resource。
Flow state load/commit 同样把 Collection revision 限定在 `1..9007199254740991`；越界持久状态拒绝加载，达到
上限后的 mutation 明确失败且不回绕。

## Desktop interaction

- Resource Explorer 只在真实 Resource 上提供上下文菜单；纯 presentation namespace 只负责展开和折叠。
- 上下文菜单根据 descriptor 提供受支持的快捷操作，并提供“添加到当前 View”等可访问的键盘等价路径；若
  authority 提供有效权限结果则可进一步隐藏或禁用，否则必须保留明确的 Forbidden 反馈，不能把可见性当授权。
- Resource 加入 View 后，由兼容 renderer 决定主要内容和常用操作按钮。按钮只是 capability 的入口，不能
  授权、隐藏 owner validation 或把 presentation hint 变成行为事实。
- Collection member 的 `list/get/read` 入口执行双侧 gate：member 必须声明对应 capability，Resource descriptor
  也必须声明兼容 capability/schema。无 `get` 的 member 只展示 list metadata 并产生零 API 调用；不能从
  Resource-level capability 推断所有 members 都允许相同操作。
- Renderer 按 type、schema、capability、content type 和 pane 条件匹配，不按 Resource name 猜测。例如
  filesystem Collection 的文本 member 可以使用文本/代码 renderer，图片可以使用图片 renderer，未知格式
  回退到 metadata/raw placeholder；只有未来 descriptor 明确声明 download/session capability 时才出现下载入口。
- 同一 capability 无论从上下文菜单、Inspector 还是 View Widget 触发，都必须经过同一 SDK 调用、policy、
  schema validation、deadline、错误与审计路径。

当前 shared descriptor-driven action derivation（纯 event-only 才 observe，混合 event + operation 仍显式 Execute）、
generic operation controller、可访问入口、Collection 分页/
目录/member detail、filesystem 安全 preview 与 View renderer 均已实现。异步 generation token、Blob URL cleanup、
compact action overflow 和 View v3 不持久化 member/content/draft 由 focused frontend tests 覆盖；这不等于
packaged Wails/GUI `QA01` 已完成。

## SDK and platform callers

- Go SDK `CollectionClient` 提供通用 list/get；domain-specific get 继续通过 typed `OperatePayload` 解码真实输出，
  `FlowClient` 提供当前 definitions/runs 全部 typed operations；
- bindings 保留 generic `OperateJSON` 与 `SubscribeCapability`，继续走同一个 Node operation/subscription path；
- Android sample 已迁移到两个 Flow Collections 与 generic capability 操作/订阅，不再调用旧 Flow Commands；
- Android 已完成双 ABI AAR 与离线 Gradle unit/lint/assemble gate；因没有连接物理设备，device smoke 明确记为
  unavailable，而不是 passed。

## Deferred protocol details

第一阶段只交付上述通用 list/get/member/page schema、Flow clean-break 和只读 filesystem provider。以下边界
明确延期，本文不把它们宣称为 current API：

- `AUTHZ02`：generic member selector policy、filtered catalog 与 authoritative effective-capability discovery；
- `CMD02`：Management、Admission、Notification 及其他 endpoint-style Command Resources 的全量迁移；
- `FS02`：filesystem write/delete、virtual multi-root mounts 与 large-file download/session data lane；
- provider/domain 后续 capability：`watch/delete` 等只有在 schema、policy、lifecycle 与迁移策略单独固定后才可声明。

`PUB01` 仍未授权 push/release/publication；这不改变上述 runtime contract。`RENDER01` 与 `QA01` 均已完成。

在 `AUTHZ02` 前，Desktop 只从 descriptor 推导“受支持操作”，可选使用已经权威获得的授权状态，并始终保留
Forbidden 的最终反馈；UI 可见性不是授权。

## Related docs

- [Resource Platform v2](resource-platform-v2.md)
- [Flow vNext](flow-vnext.md)
- [Desktop Resource Workspace v3](desktop-resource-workspace-v3.md)
- [Desktop Schema Rendering](desktop-schema-rendering.md)
- [可扩展资源平台 requirement](../requirements/extensible-resource-platform.md)
- [Desktop 资源工作区 requirement](../requirements/desktop-resource-workspace.md)
- [实现与验证归档](../change/2026-08-31_resource-collections-and-capability-actions.md)
- [Collection browser 的跨运行时契约边界](../lessons/collection-browser-cross-runtime-contract-boundaries.md)
