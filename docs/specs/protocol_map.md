# vNext Protocol And Resource Mapping

本文是 vNext 协议与资源模型的人工速查入口。协议真相位于 `protocol/`，跨语言公开面由 `sdk/bindings/generated/contracts.json` 生成并接受架构测试校验；本文不再同步任何旧 SubProto、VarStore 或 TopicBus 动作表。

## Envelope Operations

| 语义 | Operation | Phase | 资源要求 |
|---|---|---|---|
| 建立父子关系 | `Join` / `JoinAck` | request / response | 无 |
| 发布或撤销子树路由 | `RouteAnnounce` / `RouteWithdraw` | event | 无 |
| 建立或取消订阅 | `Subscribe` / `Unsubscribe` / `SubscribeAck` | request/control/response | Variable 或 Stream |
| Variable 初值与变更 | `VariableSnapshot` / `VariableUpdate` | response / event | Variable |
| Stream 数据与缺口 | `StreamEvent` / `StreamGap` | event | Stream |
| 调用 Command | `CommandCall` / `CommandResult` | request/control/response | Command |
| 显式错误 | `Error` | response | 无 |
| 链路保活 | `Heartbeat` | event | 无 |

`Source` 表示当前发送节点，`Principal` 在转发时保留原始行为主体，`Target` 表示资源所有者；权限判断使用 `Envelope.Subject()`，不能把中继节点误当成授权主体。路由、订阅和指令的详细时序分别见 [wire-protocol-vnext.md](wire-protocol-vnext.md)、[subscription-vnext.md](subscription-vnext.md) 与 [command-vnext.md](command-vnext.md)。

## Resource Mapping

只有三种资源：

- Variable：订阅后先得到 snapshot，再按 revision 接收 update；
- Stream：订阅后按 sequence 接收 event，丢失或背压必须以 gap 显式可见；
- Command：有界请求/响应，用于无法自然表达为状态或事件的操作。

内置资源族如下；具体 payload schema 由 `protocol/schema_*.go` 定义，完整机器可读清单见 `sdk/bindings/generated/contracts.json`。

| 资源族 | Variable | Stream | Command |
|---|---|---|---|
| system | `system/catalog`、`system/config`、`system/health`、`system/topology` | `system/audit` | admission、config update、node revoke、policy grant/revoke |
| notifications | — | `notifications/events` | `notifications/publish` |
| file | `file/transfers` | `file/progress` | offer、chunk、complete、cancel |
| flow | `flow/definitions`、`flow/runs` | `flow/events` | create、update、run、cancel、archive |

第一方产品资源继续遵守同一三资源模型，详见 [resource-model-vnext.md](resource-model-vnext.md) 和 [../features/README.md](../features/README.md)，不得新增旁路协议或第二套 dispatcher。

## Maintenance

1. 先修改 `protocol/` 中的版本化类型与 schema。
2. 运行 `./scripts/mfh.ps1 -Action generate -Target generated` 刷新公开 contract。
3. 更新受影响的 spec/feature dossier，而不是从旧仓库复制协议文档。
4. 运行架构、协议和绑定一致性测试；生成文件漂移必须显式失败。
