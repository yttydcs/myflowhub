# Desktop

## 定位

Desktop 是 Node/Resource 平台的通用工作台，不是另一套 Hub 或业务 runtime。它拥有独立的 NodeHost，
默认以 Parent-only leaf 连接一个父节点，从 authoritative Node tree 和每个 Node 的 `system/catalog` 发现资源；前端不解析 wire、
不推断路由，也不把按钮可见性当作权限边界。

Desktop 与 MetricsNode 完全独立：二者使用不同 Node identity、state、进程、Resource owner 和安装包。共享 NodeHost/SDK package 不会让 Desktop 拥有 Metrics 资源，也不会赋予 Desktop 默认 Listener、中继或网络特权。

源码边界：

- `apps/desktop`：Wails host、Profile/CredentialStore、View store 和 React 工作区；
- `sdk/bindings/desktop`：字符串 Node ID、JSON operation、subscription polling 和 File session boundary；
- `cmd/mfh-desktop`：无 UI identity 工具。

```text
React workspace
          │
          ▼
Desktop binding: catalog · operate · subscribe · session
          │
          ▼
NodeHost + attached canonical Go SDK
          │
          ▼
authoritative Node tree ── node-owned Resources
```

## Profile 与登录

新建 Profile 使用 Enrollment：客户端先准备仅含密钥的 Device identity，连接父 endpoint 后通过 Permit 直接注册或等待审批；Node ID 与签名 Grant 由 Admission Authority 下发并原子保存。父节点公钥不再是常规必填项，无预置锚时采用显式 TOFU 并在首次成功后固定。已有 version 2 Profile 仍按原 Node ID、父节点 ID、公钥和 Join Permit 路径连接。

设置格式为 version 2。每个 Profile 隔离 endpoint、本机 Node identity、受信任父节点和 Views；单个应用
实例只激活一个 Profile。登录成功后保存 Profile 与身份，下次启动可自动连接。切换 Profile 会先关闭旧
Host，其 subscription、session 和 connection 随之清理，再打开新身份。连接失败后的重试创建新的候选 Host，只有成功后才替换活动实例。

- 登录页支持新建、选择、编辑和显式确认删除 Profile；
- 全新 Profile 可先准备受保护的本机 identity 并复制 raw-base64 Ed25519 公钥；准备动作只保存非 active Profile，不连接、不登录，也不返回私钥；
- admission permit 只传给本次连接，不写入 settings 或日志；
- Windows 使用当前用户作用域 DPAPI 保存 Ed25519 identity；Legacy 密文位于 Profile 目录的
  `identity.dpapi`，Enrollment 设备密钥、观察到的信任锚和 Grant 位于独立的 `enrollment.dpapi`；
- 没有系统保护 backend 的平台明确使用 session-only identity，不写明文 secret；
- 删除 Profile 会删除其本地身份、runtime state 和 Views；settings reset 不删除这些 Profile 状态。

默认根目录为用户配置目录下 `MyFlowHub/desktop-resource-workspace`，可用
`MFH_DESKTOP_CONFIG_DIR` 覆盖。settings、Views 与保护后的 identity 都使用同目录临时文件、sync、
backup 和 rename 提交。损坏或不兼容的 settings/View 不会被静默覆盖；settings 显式重置口令为
`RESET DESKTOP V2`。

## 工作区

顶部只保留产品标识、资源刷新和主题动作，不重复展示连接状态或 Profile。左侧通过 Tabs 切换：

- Resource Explorer：上半区只展示 authoritative Node tree，下半区展示当前 Node 的直接 Resources；
  两区标题都是可访问的 disclosure control，最多收起一个分区。收起后另一分区占满可用高度，重新展开
  恢复上次上下比例；两区同时展开时使用可通过指针或键盘调整的水平分隔条。折叠状态、分隔比例与两棵树
  的展开状态按 Profile 保存。Resource 按完整相对 name 的 `/` segment 构建任意深度 path tree，树只影响
  展示，不改变完整 owner/name 身份或 authority；同一 path item 可以既代表真实 Resource 又拥有 children。
  Node 与 Resource 使用独立搜索和滚动；Node 任意深度由 `parent_id` 数据驱动，深层 Node 可聚焦为子树并
  通过路径面包屑返回完整树；
- View Manager：创建、打开和删除当前 Profile 的本地 Views。

左下角固定显示当前 Profile 和连接状态，点击进入主区 Settings Tab。Settings 铺满可用主区并隐藏右侧
Inspector，包含 Connection、Authority 准入管理、Profile 和 Appearance 设置；准入页把 Permit、Pending 请求和 Enrollment 统一路由到 Profile 绑定的 Authority，Profile 切换、编辑、删除和连接动作继续经过
Wails boundary。主区顶部使用可关闭的内容 Tab；Settings 是其中一种内容，不弹出第二个设置窗口。

右侧 Inspector 承载临时 Node/Resource preview，不与 View widget 混在同一内容流。点击 Node 显示父节点
控制关系、generation 和资源类型摘要；点击 Resource 选择 renderer。资源可通过拖放或按钮加入 View：
首个 widget 默认占满可用区，直接添加后续 widget 默认在整个工作区右侧分栏；拖动 Resource 或 widget
到目标 pane 的左/右/上/下边缘可局部分割该 pane，拖到相邻 divider 可精确插入同级位置，拖到工作区
外缘可分割整个 root。中心投放不创建隐藏标签栈，界面不提供“左右 / 上下 / 交换”布局按钮。

View 使用任意深度的 n 元 horizontal/vertical split tree。所有相邻 pane 之间的 separator 都可通过指针
连续调整或通过键盘细调；拖动期间只预览，释放后提交。拓扑、顺序与权重随 View 保存，删除或移动后自动
折叠空/单子节点并合并同轴 split。可用空间不足时 Workspace 滚动，不静默重排已保存布局。View 只保存
资源引用、renderer、布局和局部设置；资源暂不可用时保留 leaf 并显示 detached 状态。

## Renderer

renderer 由 descriptor 的 type、capability、schema 和 presentation hint 驱动：

- Resource provider 通过 schema 定义值类型、范围、步长、枚举、格式、字段和校验约束；Desktop 不从
  Resource name 猜测这些语义；
- Desktop renderer registry 根据 schema、capability 和 pane 尺寸过滤兼容组件并选择安全默认项；用户可在
  Widget 标题区切换兼容 renderer，切换只更新 View 展示设置，不调用 Resource operation；
- Variable：布尔、枚举、数字、文本、时间、对象和数组使用对应展示/控件；可写值先形成 draft，Reset/Apply
  后通过 `expected_revision` 条件写入，失败或冲突保留 draft；
- Stream/Topic：使用有界 log/table/timeline，支持 pause、clear、filter、autoscroll，并明确显示
  publisher/sequence、gap、expired 和 subscription state；Topic publish 复用 schema form；
- Command/通用 operation：支持 schema form、显式 Execute、typed result 和 Advanced JSON fallback；
- File：通过原生文件选择、目标路径、in-flight/error 状态完成上传；owner 提供的 `file/progress` 与
  `file/transfers` 可作为独立 Widget 展示实际进度。当前阻塞式上传 binding 尚不提供单次 transfer cancel；
- first-party catalog/topology/health/config/flow/audit/notification/file payload 使用 schema ID 选择结构化
  table/status/timeline/progress adapter；
- unknown/unsupported：说明缺少或不支持的 schema，完整保留 descriptor、JSON tree/raw 数据和安全 fallback。

provider presentation metadata 只能作为默认 hint，不能指定可执行组件或放宽数据约束。View 只保存版本化
renderer ID 与 allowlisted 非秘密显示设置；不保存 value、draft、operation payload、event body、文件内容、
permission 或 credential。旧 `mfh.variable/stream/topic/command/file` renderer ID 作为 automatic alias 继续可读。

加载、空、离线、Forbidden、订阅失败、缺失资源和未知 renderer 都会明确呈现。拖放具有独立键盘
激活手柄和添加按钮等价路径；Node tree 与 Resource path tree 都使用 roving focus，并实现 Arrow
Up/Down/Left/Right、Home、End、Enter 与 Space 的 WAI-ARIA 键盘路径。分区标题使用 button、
`aria-expanded` 和 `aria-controls`；上下区分隔条使用 horizontal separator 语义，支持方向键、Home/End
与双击复位；Tabs、表单和动作有辅助技术标签。视觉使用 Radix/shadcn 风格 primitives、CSS tokens、
浅色默认、完整深色重配色与 reduced-motion，不维护第二套旧 UI。主题、Node 展开/聚焦、Resource path
展开、Explorer 收起状态与分区比例作为版本化非秘密 UI preference 按 Profile 保存，损坏 preference
只回退到显式默认值。

视觉采用矿物蓝与石墨灰，使用连续平面分栏、1px 边界、紧凑缩进和克制圆角。界面不使用宣传式副标题、
英文 eyebrow、渐变、发光、装饰性卡片墙或重复状态胶囊。


## 安全边界

- Wails boundary 再次校验 Node ID、Profile、JSON 大小、capability/schema 和 View layout；
- Desktop 日志最多 500 条，不记录 operation payload、文件块、剪贴板正文、private key 或 permit；
- 权限、schema、session 与路径最终由 authority 和 Resource owner 校验；
- schema form 的客户端校验只提供即时反馈，不能替代 owner/runtime 的权威校验；
- File 只接受普通本地文件，服务端路径固定在配置根内，chunk 有大小与 SHA-256 校验。

## 构建与验证

```powershell
$env:GOWORK='off'
go test ./apps/desktop/... ./sdk/bindings/desktop
go vet ./apps/desktop/... ./cmd/mfh-desktop

cd apps/desktop/frontend
npm ci
npm test
npm run build
cd ..
wails build -clean
```

生产产物为 `apps/desktop/build/bin/mfh-desktop.exe`。目标契约见
[Desktop Resource Workspace v3](../specs/desktop-resource-workspace-v3.md)。

## 相关文档

- [本轮 Explorer 请求](../intake/2026-08-30_desktop-explorer-collapsible-resource-tree.md)
- [Desktop 资源工作区 requirements](../requirements/desktop-resource-workspace.md)
- [Desktop Resource Workspace v3](../specs/desktop-resource-workspace-v3.md)
- [NodeHost Runtime](../specs/node-host-runtime.md)
- [Explorer Resource tree 变更归档](../change/2026-08-30_desktop-explorer-collapsible-resource-tree.md)

## 明确移除

- Vanilla TypeScript imperative UI 与固定产品页面；
- fixed Variable/Stream/Command kind switch 与独立子协议；
- settings version 1 的猜测迁移；
- 明文持久 permit/private identity 和前端权限裁决；
- Desktop 内置 MCP/stdio 自动化入口；未来对外 Agent 接口作为独立产品重新设计。
