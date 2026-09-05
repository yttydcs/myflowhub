# ClipboardNode vNext

## Purpose

ClipboardNode 在明确配置的节点之间同步 UTF-8 文本剪贴板。它是普通 Node：节点在权威树中的父子关系负责信任、控制和路由；剪贴板正文是节点下的资源，不再使用 Topic 或子协议建立第二套寻址和权限模型。

## Resource model

| Resource | Kind | Meaning | Permission |
| --- | --- | --- | --- |
| `clipboard/events` | Stream | 本节点发布的文本事件 | `clipboard.events.read` |
| `clipboard/status` | Variable | 连接无关的同步状态和安全摘要 | `clipboard.status.read` |
| `clipboard/config` | Variable | 持久配置和显式接收 peer | `clipboard.config.read` |
| `clipboard/config/update` | Command | 乐观并发配置更新 | `clipboard.config.write` |
| `clipboard/send` | Command | 显式发布一段本地文本 | `clipboard.send` |
| `clipboard/apply` | Command | 应用一个等待确认的远端事件 | `clipboard.apply` |
| `clipboard/history/clear` | Command | 清理本地历史 | `clipboard.history.clear` |

每个接收端只订阅配置中 `receive=true` 的 peer 的 `clipboard/events`。订阅由 SDK 在父链路断开后恢复；更换 TCP、QUIC、RFCOMM 或其他 `link.Driver` 不改变资源 ID、配置和同步算法。

```text
Node A / clipboard/events ─┐
                           ├─ parent tree routing ─→ Node B durable subscription
Node C / clipboard/events ─┘                         └─ local policy / apply
```

## Privacy and limits

- 正文只存在于受权限保护的 Stream/Command、平台写入队列，以及用户选择保留的本地有界历史中。
- status、config、进程日志和公开错误只包含 event ID、字节数、hash 前缀和稳定错误描述，不包含正文。
- 单条协议上限为 256 KiB；默认本地 inline 上限为 64 KiB，超限显式失败。
- pending 队列最多 64 条；历史同时受条数、正文总字节和 TTL 限制，可选择 `body`、`metadata` 或 `none`。
- event ID 与 SHA-256 的近期集合负责去重；成功应用远端正文后，短期 suppression 集合阻止平台 watcher 把同一正文重新发布。
- 非 UTF-8、空文本、来源与订阅 peer 不一致、回放、未授权 peer 和不可用平台均显式拒绝或标记为 ignored。

## Product surfaces

- Go core、同步引擎和持久状态：`apps/nodes/clipboard`。
- Windows 系统剪贴板适配器：`apps/nodes/clipboard/platform/windows`，直接调用 Win32 Unicode clipboard API。
- 有界 NDJSON 本地桥：`apps/nodes/clipboard/bridge`；每条请求/响应最多 512 KiB，Flutter 不接触 Go 对象或网络帧。
- Windows/Web Flutter 壳：`apps/nodes/clipboard/app`。Windows 启动同目录 `mfh-clipboard.exe -bridge`；Web 是明确标注的配置/UI 预览，不伪装为可运行节点。
- 无 UI Windows 入口：`cmd/mfh-clipboard`。

Windows 产品由 NodeHost 持有身份、Node 和父连接；同步引擎使用 attached SDK 与只读连接状态。资源在 Host.Start 前注册，启动失败与退出释放 Host，现有身份、clipboard.json 和 clipboard_history.json 继续复用。Android 实现已移除，见[重新设计待办](../requirements/mobile-embedded-redesign.md)。

## Build and verification

Go 命令从 monorepo 根运行并设置 `GOWORK=off`：

```text
go test ./apps/nodes/clipboard/... ./cmd/mfh-clipboard
go test -race ./apps/nodes/clipboard/...
```

在 `apps/nodes/clipboard/app` 运行：

```text
flutter analyze
flutter test
flutter build web --release
flutter build windows --release
```

门禁包括：core/bridge 单元测试、真实父树双节点同步、Windows 平台 smoke、真实 TCP Hub 进程 E2E、Flutter analyze/test、Windows/Web release 与 Host 状态保留/失败回滚测试。

## Non-goals

- 不恢复 TopicBus、VarStore、SubProto 或旧 SDK 兼容层。
- 不把物理链路名称写入剪贴板资源和权限。
- 不承诺平台禁止后台剪贴板访问时仍可自动同步。
- 不在 FM10 发布、签名或上传安装包。
