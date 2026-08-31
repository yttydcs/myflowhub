# Flow

## Purpose

Flow 保存有界的自动化定义，调用已有 Resource，并通过同一个运行集合的订阅 capability 暴露运行事件。
当前 catalog 只注册 `flow/definitions` 与 `flow/runs` 两个 `mfh.collection` Resources；definition 和 run
是 collection-scoped members，不逐项进入全局 catalog。

## Current Resource surface

下表的 Resource/capability/schema 逐项对应当前 descriptor、canonical binding contract 与 generated contract。
`Permission` 列来自 runtime descriptor，不写入精简 binding manifest；它是兼容字段，真正的网络授权 key 是
exact ResourceID + CapabilityID。

| Resource | Capability | Permission | Input schema | Output/event schema |
| --- | --- | --- | --- | --- |
| `flow/definitions` | `archive` | `flow.write` | `mfh.flow.archive.v1` | `mfh.flow.archive.v1` |
| `flow/definitions` | `create` | `flow.write` | `mfh.flow.definition.v1` | `mfh.flow.definition.v1` |
| `flow/definitions` | `get` | `flow.read` | `mfh.collection.member-request.v1` | `mfh.flow.definition.v1` |
| `flow/definitions` | `list` | `flow.read` | `mfh.collection.list-request.v1` | `mfh.collection.page.v1` |
| `flow/definitions` | `run` | `flow.run` | `mfh.flow.run.v1` | `mfh.flow.run-summary.v1` |
| `flow/definitions` | `update` | `flow.write` | `mfh.flow.definition.v1` | `mfh.flow.definition.v1` |
| `flow/runs` | `cancel` | `flow.cancel` | `mfh.flow.cancel.v1` | `mfh.flow.run-summary.v1` |
| `flow/runs` | `get` | `flow.read` | `mfh.collection.member-request.v1` | `mfh.flow.run-summary.v1` |
| `flow/runs` | `list` | `flow.read` | `mfh.collection.list-request.v1` | `mfh.collection.page.v1` |
| `flow/runs` | `subscribe` | `flow.read` | — | event `mfh.flow.event.v1` |

两个集合都是 flat Collection：`parent` 必须为空；list 按 member key 确定性排序，返回
`1..9007199254740991` 范围内、可被 JavaScript 精确往返的 collection revision 和 opaque bounded cursor，
集合变化后旧 cursor 明确 stale；revision 耗尽时 mutation 失败而不回绕。definitions page 的 member 声明
`archive/get/run/update`，runs page 的 member 声明 `cancel/get`；collection-level `create/list/subscribe` 不伪装成
单个 member 的能力。

## Observable behavior

- definition 有 schema version、稳定 flow ID、revision、名称、节点/边数量和输入大小上限；create 要求 revision 1，
  update 要求下一 revision，活动运行存在时不能更新或归档；
- run request 有唯一 run ID、flow revision、dedupe key、deadline、输入与并发上限；run summary 只保留 run/flow ID、
  flow revision、initiator、dedupe key、state、开始/完成时间与错误。节点 outputs 只在执行过程中传递，不进入
  `FlowRunSummaryV1`；
- retry 只对声明为可重试的执行错误生效并使用受限 backoff；取消幂等，terminal state 不回退；
- `flow/runs.subscribe` 在同一个 runs Resource 上产生 `mfh.flow.event.v1`，不再注册 `flow/events` Stream；
- version 1 的持久 Flow state 仍保存 definitions、runs 与 revision；本次 clean break 只替换 catalog/operation surface，
  没有迁移或重写持久格式，重启恢复、dedupe 与 interrupted 处理保持原语义。

## Permissions

authority 按 caller subject、完整 Flow ResourceID 与具体 CapabilityID 分别授权。共享的 `flow.read`/`flow.write`
descriptor Permission 字符串不会把 `list` grant 扩大成 `get`，也不会把 `create` grant 扩大成 `update`。
Flow 代表原始 initiator 调用下游 Resource，不能用 Flow host 身份隐式提权；取消还执行 run owner/domain 校验，
未授权调用保留明确 Forbidden 或存在性保护反馈。

## Clean break

`flow/create`、`flow/update`、`flow/run`、`flow/cancel`、`flow/archive` Commands 和 `flow/events` Stream 已从
runtime catalog、canonical binding manifest 与 generated contract 移除。旧 Resource 引用保持 missing，旧
`flow/run + invoke` 等 grant 不会自动映射到新 `flow/definitions + run` capability，也没有兼容重定向。

## Current clients and validation boundary

Go SDK 提供 typed `CollectionClient` 与 `FlowClient`，list/get/create/update/archive/run/cancel/subscribe 全部复用
同一个 Node operation/subscription path。跨语言 bindings 保留 generic `OperateJSON` 与 `SubscribeCapability`；
Android sample 已改为列出两个 Collection、在 `flow/runs` 上订阅，并按 Resource + capability + schema 操作。
Android sample 已通过双 ABI AAR 与离线 Gradle unit/lint/assemble 门禁；当前没有连接物理设备，因此 device
smoke 明确记为 unavailable，不能记为 pass。

focused runtime、跨节点授权、SDK、binding contract 与 generated freshness tests 已覆盖当前 surface；Desktop
Collection renderer 已实现并通过 frontend component/full-suite/build 门禁，完整产品与 packaged GUI evidence 仍由
`QA01` 完成。

## Non-goals and deferred work

- 不恢复通用 action dispatcher、旧 Flow endpoint compatibility bridge 或任意代码执行；
- 不在 Flow 内复制 Resource、Subscription 或路由权限逻辑；
- `AUTHZ02` 的 member selector/effective-capability discovery 与 `CMD02` 的其他 endpoint-style Command 迁移不由
  Flow 实现猜测；push/release/publication 仍属于未授权的 `PUB01`。

## Related

- [Flow vNext](../specs/flow-vnext.md)
- [Resource Collections and Actions](../specs/resource-collections-and-actions.md)
- [Protocol and Resource Mapping](../specs/protocol_map.md)
