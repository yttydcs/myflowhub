# Desktop

## 定位

Desktop 是 Node/Resource 平台的通用工作台，不是另一套 Hub 或业务 runtime。它通过 canonical SDK
连接一个父节点，从 authoritative Node tree 和每个 Node 的 `system/catalog` 发现资源；前端不解析 wire、
不推断路由，也不把按钮可见性当作权限边界。

源码边界：

- `apps/desktop`：Wails host、Profile/CredentialStore、View store 和 React 工作区；
- `sdk/bindings/desktop`：字符串 Node ID、JSON operation、subscription polling 和 File session boundary；
- `apps/desktop/mcp`：与 Desktop 使用同一 binding 的本地自动化入口；
- `cmd/mfh-desktop`、`cmd/mfh-desktop-mcp`：无 UI identity 工具与 stdio MCP 入口。

```text
React workspace / MCP host
          │
          ▼
Desktop binding: catalog · operate · subscribe · session
          │
          ▼
canonical Go SDK + ParentSupervisor
          │
          ▼
authoritative Node tree ── node-owned Resources
```

## Profile 与登录

设置格式为 version 2。每个 Profile 隔离 endpoint、本机 Node identity、受信任父节点和 Views；单个应用
实例只激活一个 Profile。登录成功后保存 Profile 与身份，下次启动可自动连接。切换 Profile 会先关闭旧
client，其 subscription、session 和 connection 随之清理，再打开新身份。

- 登录页支持新建、选择、编辑和显式确认删除 Profile；
- 全新 Profile 可先准备受保护的本机 identity 并复制 raw-base64 Ed25519 公钥；准备动作只保存非 active Profile，不连接、不登录，也不返回私钥；
- admission permit 只传给本次连接，不写入 settings 或日志；
- Windows 使用当前用户作用域 DPAPI 保存 Ed25519 identity；密文位于 Profile 目录的
  `identity.dpapi`；
- 没有系统保护 backend 的平台明确使用 session-only identity，不写明文 secret；
- 删除 Profile 会删除其本地身份、runtime state 和 Views；settings reset 不删除这些 Profile 状态。

默认根目录为用户配置目录下 `MyFlowHub/desktop-resource-workspace`，可用
`MFH_DESKTOP_CONFIG_DIR` 覆盖。settings、Views 与保护后的 identity 都使用同目录临时文件、sync、
backup 和 rename 提交。损坏或不兼容的 settings/View 不会被静默覆盖；settings 显式重置口令为
`RESET DESKTOP V2`。

## 工作区

顶部只保留产品标识、资源刷新和主题动作，不重复展示连接状态或 Profile。左侧通过 Tabs 切换：

- Resource Explorer：上半区只展示 authoritative Node tree，下半区展示当前 Node 的直接 Resources；
  两区使用独立搜索与滚动，水平分隔条可通过指针或键盘调整高度并按 Profile 保存。Resource 按名称首段
  分组，分组仅影响显示，完整 owner/name 身份仍用于预览、拖放和“添加到工作区”。Node 任意深度由
  `parent_id` 数据驱动，深层 Node 可聚焦为子树并通过路径面包屑返回完整树；
- View Manager：创建、打开和删除当前 Profile 的本地 Views。

左下角固定显示当前 Profile 和连接状态，点击进入主区 Settings Tab。Settings 铺满可用主区并隐藏右侧
Inspector，包含 Connection、Profile 和 Appearance 三类设置；Profile 切换、编辑、删除和连接动作继续经过
Wails boundary。主区顶部使用可关闭的内容 Tab；Settings 是其中一种内容，不弹出第二个设置窗口。

右侧 Inspector 承载临时 Node/Resource preview，不与 View widget 混在同一内容流。点击 Node 显示父节点
控制关系、generation 和资源类型摘要；点击 Resource 选择 renderer。资源可通过拖放或按钮加入 12 列响应式
View，widget 可左右移动、调整宽度、移除，View 可重命名并保存。View 只保存资源引用、renderer、布局和
局部设置；资源暂不可用时保留布局并显示 detached 状态。

## Renderer

renderer 由 descriptor 的 type、capability、schema 和 presentation hint 驱动：

- Variable：读取 JSON 快照与刷新；
- Stream：有界实时事件列表和订阅状态；
- Topic：订阅事件，展示 publisher/sequence 数据，并通过 `publish` capability 发布；
- Command：按 input schema 提供 JSON operation 表单并显示 typed result/error；
- File：选择本地路径和目标路径，通过绑定到 Resource 的 session 上传；
- unknown：完整显示 descriptor，不隐藏未知 type。

加载、空、离线、Forbidden、订阅失败、缺失资源和未知 renderer 都会明确呈现。拖放具有独立键盘
激活手柄和添加按钮等价路径；Tree 使用 roving focus，并实现 Arrow Up/Down/Left/Right、Home、End、
Enter 与 Space 的 WAI-ARIA 键盘路径；上下区分隔条使用 horizontal separator 语义，支持方向键、
Home/End 与双击复位；Tabs、表单和动作有辅助技术标签。视觉使用 Radix/shadcn 风格 primitives、CSS
tokens、浅色默认、完整深色重配色与 reduced-motion，不维护第二套旧 UI。主题、树展开/聚焦与 Explorer
分区比例作为版本化非秘密 UI preference 按 Profile 保存，损坏 preference 只回退到显式默认值。

视觉采用矿物蓝与石墨灰，使用连续平面分栏、1px 边界、紧凑缩进和克制圆角。界面不使用宣传式副标题、
英文 eyebrow、渐变、发光、装饰性卡片墙或重复状态胶囊。


## 安全边界

- Wails boundary 再次校验 Node ID、Profile、JSON 大小、capability/schema 和 View layout；
- Desktop 日志最多 500 条，不记录 operation payload、文件块、剪贴板正文、private key 或 permit；
- MCP 单条 NDJSON 请求限制 1 MiB，写操作必须显式启用；
- 权限、schema、session 与路径最终由 authority 和 Resource owner 校验；
- File 只接受普通本地文件，服务端路径固定在配置根内，chunk 有大小与 SHA-256 校验。

## 构建与验证

```powershell
$env:GOWORK='off'
go test ./apps/desktop/... ./sdk/bindings/desktop
go vet ./apps/desktop/... ./cmd/mfh-desktop ./cmd/mfh-desktop-mcp

cd apps/desktop/frontend
npm ci
npm test
npm run build
cd ..
wails build -clean
```

生产产物为 `apps/desktop/build/bin/mfh-desktop.exe`。目标契约见
[Desktop Resource Workspace v2](../specs/desktop-resource-workspace-v2.md)。

## 明确移除

- Vanilla TypeScript imperative UI 与固定产品页面；
- fixed Variable/Stream/Command kind switch 与独立子协议；
- settings version 1 的猜测迁移；
- 明文持久 permit/private identity 和前端权限裁决。
