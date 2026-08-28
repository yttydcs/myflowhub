# MFH4 v2 Protocol And Resource Mapping

本文是当前协议与资源模型的人工速查入口。机器真相位于 `protocol/`，跨语言公开面由
`sdk/bindings/generated/contracts.json` 生成并接受架构测试校验。

## Envelope operations

| 语义 | Operation | Phase | Resource/capability |
|---|---|---|---|
| 建立父子关系 | `Join` / `JoinAck` | request / response | 无 |
| 发布或撤销子树路由 | `RouteAnnounce` / `RouteWithdraw` | event | 无 |
| 建立或取消观察关系 | `Subscribe` / `Unsubscribe` / `SubscribeAck` | request/control/response | 任意 observable capability |
| 资源事件与缺口 | `ResourceEvent` / `ResourceGap` | response/event | 原 subscription capability |
| 通用资源操作 | `Operate` / `OperateResult` | request/control/response | `read`、`write`、`publish`、`invoke` 或扩展 capability |
| 打开会话 | `SessionOpen` / `SessionOpenResult` | request/control/response | session-oriented `open` capability |
| 会话数据与关闭 | `SessionData` / `SessionClose` | request/control/response | 与 grant 相同的 Resource/capability |
| 显式错误 | `Error` | response | 关联原请求 |
| 链路保活 | `Heartbeat` | event | 无 |

frame magic 为 `MFH4`、major version 为 2。`Source` 表示当前发送 Node，`Principal` 在合法委托时保留
原始行为主体，`Target` 表示 Resource owner；权限使用 `Envelope.Subject()` 与 exact capability，不能把
中继 Node 当成授权主体。Join/ack/permit 的签名 domain 同步使用 MFH4。

## Resource types

| Type | 基础 capability | 关键语义 |
|---|---|---|
| `mfh.variable` | `read`、可选 `write`、`subscribe` | snapshot-first、revision、expected-revision 条件写 |
| `mfh.stream` | `subscribe` | owner sequence、有界 delivery、explicit gap |
| `mfh.topic` | `publish`、`subscribe` | Node-owned broker、多 publisher、per-publisher sequence、默认无 replay |
| `mfh.command` | `invoke` | 独立 input/output schema、deadline、dedupe、panic isolation |
| `mfh.file` | `open` | subject/link/policy-bound session、有界 data lane、checksum/atomic commit |

类型 ID 与 capability 是受限的可扩展字符串。未知 type 仍能进入 catalog 与 Desktop inspector；未声明的
capability、schema mismatch 或未知 major 明确失败。Core 通过 `Resource`、`Observable` 与
`SessionResource` 接口扩展，不维护产品 type switch。

## Built-in resources

| 资源族 | Observable state/events | Operations/sessions |
|---|---|---|
| system | `system/catalog`、`system/config`、`system/health`、`system/topology`、`system/audit` | admission、config update、node revoke、policy grant/revoke |
| notifications | `notifications/events` | `notifications/publish` invoke |
| file | `file/transfers`、`file/progress` | `file/upload` open session |
| flow | `flow/definitions`、`flow/runs`、`flow/events` | create、update、run、cancel、archive invoke |

第一方 Metrics 与 Clipboard 也只注册 Node-owned descriptor/capability，不建立旁路 dispatcher。

## Maintenance

1. 先修改 `protocol/` 中的版本化类型与 schema。
2. 运行 `./scripts/mfh.ps1 -Action generate -Target generated` 刷新公开 contract。
3. 更新受影响的 current spec/feature，而不是从旧仓复制协议文档。
4. 运行协议、架构、bindings、Embedded fixture 与 generated drift 门禁。
