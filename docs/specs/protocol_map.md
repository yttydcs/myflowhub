# MFH4 v2 Protocol And Resource Mapping

本文是当前协议与资源模型的人工速查入口。机器真相位于 `protocol/`；第一方 Resource manifest 位于
`sdk/bindings/generated/contracts.json`，数据 schema 位于 Desktop generated schema artifact，并由生成门禁校验。

## Envelope operations

| 语义 | Operation | Phase | Resource/capability |
| --- | --- | --- | --- |
| 建立父子关系 | `Join` / `JoinAck` | request / response | 无 |
| 发布或撤销子树路由 | `RouteAnnounce` / `RouteWithdraw` | event | 无 |
| 建立或取消观察关系 | `Subscribe` / `Unsubscribe` / `SubscribeAck` | request/control/response | descriptor 声明的 observable capability |
| 资源事件与缺口 | `ResourceEvent` / `ResourceGap` | response/event | 原 subscription capability |
| 通用资源操作 | `Operate` / `OperateResult` | request/control/response | descriptor 声明的任意 capability |
| 打开会话 | `SessionOpen` / `SessionOpenResult` | request/control/response | session-oriented `open` capability |
| 会话数据与关闭 | `SessionData` / `SessionClose` | request/control/response | 与 grant 相同的 Resource/capability |
| 显式错误 | `Error` | response | 关联原请求 |
| 链路保活 | `Heartbeat` | event | 无 |

frame magic 为 `MFH4`、major version 为 2。`Source` 表示当前发送 Node，`Principal` 只在合法委托时保留原始
行为主体，`Target` 表示 Resource owner。权限使用 `Envelope.Subject()`、完整 ResourceID 与 exact
CapabilityID；中继 Node 不是授权主体。Join/ack/permit 的签名 domain 同步使用 MFH4。

## Descriptor and authorization boundary

`ResourceDescriptorV2` 声明 ResourceID、可扩展 type/type version、排序 capability 列表、引用的 schemas、limits
与 presentation。每个 capability 声明 `name`、required `permission` compatibility string、input/output/event
schema 与最大 payload。

`permission` 仍保留在 v2 catalog 以兼容现有 descriptor/provider，并可描述领域意图；它不会进入 Envelope 成为
第二个授权 selector。runtime policy 把 `Action` 规范化为当前 CapabilityID，并按 subject + ResourceID +
CapabilityID 裁决。因此两个 capabilities 即使都写 `filesystem.read`，grant 也不互相继承。presentation、UI
namespace、物理 path 与产品名同样不构成权限。

## Resource types

| Type | Capability shape | Current semantics |
| --- | --- | --- |
| `mfh.variable` | `read`、可选 `write`、`subscribe` | snapshot-first、revision、expected-revision 条件写 |
| `mfh.stream` | `subscribe` | owner sequence、有界 delivery、explicit gap |
| `mfh.topic` | `publish`、`subscribe` | Node-owned broker、多 publisher、per-publisher sequence、默认无 replay |
| `mfh.command` | `invoke` | 独立 input/output schema、deadline、dedupe、panic isolation |
| `mfh.file` | `open` | subject/link/policy-bound session、有界 data lane、checksum/atomic commit |
| `mfh.collection` | 通用 `list/get` 加 provider 声明的 capability 子集 | 一个 Resource 管理有界 provider-scoped members；member 默认不进入 catalog |

类型 ID 与 CapabilityID 都是受限的可扩展字符串。未知 type 仍能进入 catalog 与 generic inspector；未声明的
capability、schema mismatch 或未知 major 明确失败。Core 通过 `Resource`、`Observable`、`SessionResource`
与 exact-handler Resource 扩展，不维护产品 type switch。

## Collection and filesystem schemas

| Schema | Direction | Stable fields and bounds |
| --- | --- | --- |
| `mfh.collection.list-request.v1` | request | version、optional parent ≤1024 bytes、opaque cursor ≤1024 bytes、limit `1..256` |
| `mfh.collection.member-request.v1` | request | version、non-empty provider-scoped key ≤1024 bytes |
| `mfh.collection.member.v1` | response/member | key、kind、label、optional content/schema、sorted capability subset、bounded attributes；无 owner/ResourceID |
| `mfh.collection.page.v1` | response | version、JSON-safe revision `1..9007199254740991`、parent、最多 256 个有序 members、next cursor ≤1024 bytes |
| `mfh.filesystem.read-request.v1` | request | key ≤1024 bytes、max bytes `1..131072`、optional expected revision ≤128 bytes |
| `mfh.filesystem.content.v1` | response | key、content type ≤127 bytes、`utf-8|base64` data、size、modified time、revision ≤128 bytes；最大 decoded size 128 KiB |

filesystem schemas 是 canonical built-in data schemas，但 filesystem Resource 不是固定 built-in manifest：产品必须
显式调用 provider 注册 mount。物理 root/config 不在 wire、catalog 或正常错误中；每个 root 是一个独立
`mfh.collection`，只声明 `get/list/read`。

| Filesystem capability | Permission compatibility field | Input | Output |
| --- | --- | --- | --- |
| `get` | `filesystem.get` | `mfh.collection.member-request.v1` | `mfh.collection.member.v1` |
| `list` | `filesystem.list` | `mfh.collection.list-request.v1` | `mfh.collection.page.v1` |
| `read` | `filesystem.read` | `mfh.filesystem.read-request.v1` | `mfh.filesystem.content.v1` |

## Retired Flow

当前内置资源与生成契约不再包含 Flow 资源或 `mfh.flow.*` schema。旧调用不做兼容重定向，旧状态文件
不再加载。历史契约见 [Flow vNext](flow-vnext.md)，后续范围见[重新设计待办](../requirements/flow-redesign.md)。

## Other built-in Resource families

| Resource family | Current surface |
| --- | --- |
| system | `system/catalog`、config/health/topology/audit、admission，以及 policy Definitions/Bindings/Grants Collections |
| notifications | `notifications/events` Stream、`notifications/publish` Command |
| file | `file/transfers` Variable、`file/progress` Stream、`file/upload` File session |

Management、Admission 与 Notification 中仍存在的 endpoint-style Commands 保持 current；全量 capability 迁移属于
延期的 `CMD02`，本文不把目标形态伪装成现状。

## Policy schemas and Resources

| Resource | Exact capabilities | Primary schemas |
| --- | --- | --- |
| `system/policy/definitions` | `create/delete/get/list/update` | `mfh.policy.definition*.v1`、Collection request/page |
| `system/policy/bindings` | `create/evaluate/get/list/revoke` | `mfh.policy.binding*.v1`、`mfh.policy.evaluate*.v1`、Collection request/page |
| `system/policy/grants` | `create/get/list/revoke` | `mfh.policy.grant.v1`、Collection request/page |

runtime 先匹配 exact Grant，再按 Subject Binding、当前原子 topology scope 和 Definition rules 匹配；默认拒绝。`superadmin`
是不可变 all/all Definition，但默认无 Binding。legacy `system/policy/grant/revoke` Commands 保留为精确 Grant compatibility aliases；
全部在线 mutation 还要求调用 Subject 有本 Authority-domain superadmin Binding。完整契约见
[Scoped Policy Authorization](scoped-policy-authorization.md)。

## Current SDK and binding surface

- Go SDK `OperatePayload` 在既有 Node optimized operation path 上完成 typed request/response 校验；
- `CollectionClient` 提供通用 list/get，`PolicyClient` 提供 Authority policy Collections 的 typed operations；
- generic bindings 公开 `OperateJSON` 与 `SubscribeCapability`，Desktop 不需要恢复 endpoint wrappers；
- Android bindings 已随平台退役移除；portable attached bindings 保留 generic capability API，不再拥有 Node 生命周期。

## Deferred and delivery boundary

- `AUTHZ02`：member selector policy、filtered catalog 与 authoritative effective-capability discovery；
- `CMD02`：其余 endpoint-style Command Resources 的领域迁移；
- `FS02`：filesystem write/delete、virtual multi-root 与 large-file download/session；
- `MAIN02`：主 checkout 已无原 unmerged index entry，但同路径仍有并发未提交修改，只阻塞 merge/archive；
- `PUB01`：push、release 与 publication 未授权。

Desktop Collection/content renderer 已实现并通过 focused/full frontend tests 与 TypeScript/Vite build；完整 Wails
package 与 GUI evidence 仍等待 `QA01`，不能从 component/build evidence 推导为 packaged UI 已完成。

## Maintenance

1. 先修改 `protocol/` 的版本化类型、Validate 与 built-in data schema annotation。
2. 修改 canonical binding manifest 后运行生成门禁，确认 contract 与 data schema artifact deterministic/fresh。
3. 更新受影响的 current spec/feature，不从旧仓或已退役 endpoint 文档复制现状。
4. 运行协议、runtime、SDK、bindings、产品构建与 generated drift 门禁；未执行的产品 gate 必须保持 pending。
