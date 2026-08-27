# Desktop

## 定位

Desktop 是节点树的通用控制台，而不是另一个 Hub。它使用 canonical SDK 连接一个父节点，通过资源目录发现 Variable、Stream 与 Command。它不解析 wire phase，不推断路由，不持有可绕过父节点的管理旁路。

源码位于：

- `apps/desktop`：Wails host、版本化本地设置与产品 UI；
- `sdk/bindings/desktop`：适合 Wails 的 JSON/polling binding；
- `apps/desktop/mcp`：面向本地自动化的同模型 MCP surface；
- `cmd/mfh-desktop`、`cmd/mfh-desktop-mcp`：无 UI identity 工具与 stdio MCP 入口。

## 架构

```text
Wails UI / MCP host
        │
        ▼
desktop binding: identity · connection · catalog · snapshot · subscribe · invoke
        │
        ▼
canonical Go SDK + ParentSupervisor
        │
        ▼
Hub authority tree ── node-owned resources
```

页面不会为 Metrics、Clipboard、File 或 Flow 再创建一套业务 runtime。专用页面只是稳定 resource identity 的产品化视图；资源浏览器可以呈现未来新增能力，因此扩展 Transport 或节点实现不要求重写 Desktop service 层。

## 页面与资源映射

| 页面 | 数据面 | 写入面 |
| --- | --- | --- |
| 运行概览 | connection snapshot、`system/health` | connect/disconnect |
| 节点树 | `system/topology` Variable | 无 |
| 资源浏览器 | `system/catalog`、任意 Variable/Stream | 任意已发现 Command |
| Metrics | `metrics/config` 和 `metrics/*` Variables | `metrics/*/set`、`metrics/config/update` |
| Clipboard | `clipboard/status`、`clipboard/config` | `clipboard/send` 等 Commands |
| File | `file/transfers` | offer/chunk/complete/cancel；Desktop 上传限制 64 MiB |
| Flow | `flow/definitions`、`flow/runs` | create/update/run/cancel/archive |
| 权限与准入 | management config/topology | issue/revoke permit、revoke node、grant/revoke policy |

资源 descriptor 和服务端裁决是事实来源。UI 可以根据 descriptor 改善可用性，但按钮隐藏绝不是权限边界。deny、revoke、过期、not-found 和 reconnecting 都必须以 SDK 错误或连接状态显示。

## 本地状态

设置文件位于用户配置目录的 `MyFlowHub/desktop-vnext/settings.json`，也可用 `MFH_DESKTOP_CONFIG_DIR` 覆盖。格式固定为 version 1，并使用同目录临时文件、sync 和 atomic rename 提交。

- profile 只能使用受限名称，不能逃逸状态根目录；
- identity、trust、policy 与 admission 存在 `profiles/<profile>/state`；
- 保存连接设置会关闭并重开会话，但不会重建 identity；
- 未知、损坏或不兼容的设置不会被猜测迁移；错误明确要求 reset；
- 显式 reset 口令为 `RESET DESKTOP V1`，只重置设置，不删除身份状态。无法启动时可设置 `MFH_DESKTOP_RESET_CONFIRM=RESET DESKTOP V1` 启动一次；错误口令会被拒绝。

这是对旧 Win store 的 clean break。没有外部用户，因此不引入长期双格式兼容层。

## 隐私与边界

- Desktop 操作日志最多 500 条，只记录生命周期、资源名和错误；不记录 Command request、文件块或剪贴板正文。
- MCP 单条 NDJSON 请求限制 1 MiB；Command 默认关闭，只有显式 `--allow-write` 才能调用。
- 文件上传读取普通文件、限制 64 MiB、每块不超过 64 KiB并逐块 SHA-256；失败会尝试显式 cancel。
- 前端 JSON 只作为 boundary value，所有准入和管理 request 在 Go 侧按 protocol schema 再验证。
- Wails host 通过 `GrantPolicyJSON` / `RevokePolicyJSON` 暴露同一 canonical management Command，不直接编辑 Hub 状态文件。

## 构建与验证

```powershell
$env:GOWORK='off'
go test ./apps/desktop/... ./sdk/bindings/desktop
go vet ./apps/desktop/... ./cmd/mfh-desktop ./cmd/mfh-desktop-mcp

cd apps/desktop
wails generate module
cd frontend
npm ci
npm test
npm run build
cd ..
wails build -platform windows/amd64
```

生产产物是 `apps/desktop/build/bin/myflowhub-desktop.exe`。Wails binding、前端单测/构建、Windows production build、CLI/MCP JSON smoke 和真实 GUI 进程存活 smoke 是 FM11 门禁。

## 明确移除

- 旧 TopicBus、VarPool、Stream/Management/Auth service wrapper；
- sibling module `replace` 和旧 core/proto/sdk import；
- 前端对 frame phase、Transport 类型和物理路由的判断；
- 旧配置格式的静默猜测或自动覆盖。
