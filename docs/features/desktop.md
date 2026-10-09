# Desktop

## 按需拓扑发现

默认从直接父节点加载一层，展开已知节点再请求孩子；搜索只覆盖已加载节点。目录按当前选择与 View 引用独立加载，明确区分未加载、空与失败。显式“加载完整子树”受独立权限约束；错误不会自动扩大查询范围。实现与验收状态见[拓扑发现合同](../specs/topology-discovery.md)。

浏览起点、返回直接父节点和“加载完整子树”统一放在 Settings → 连接 → 节点浏览。左侧节点区保留搜索和树导航，不常驻范围配置表单。

## 定位

Desktop 是 Node/Resource 平台的通用工作台，不是另一套 Hub 或业务 runtime。它拥有独立的 NodeHost，
默认以 Parent-only leaf 连接一个父节点，从 authoritative Node tree 和每个 Node 的 `system/catalog` 发现资源；前端不解析 wire、
不推断路由，也不把按钮可见性当作权限边界。

Desktop 与 MetricsNode 完全独立：二者使用不同 Node identity、state、进程、Resource owner 和安装包。共享 NodeHost/SDK package 不会让 Desktop 拥有 Metrics 资源，也不会赋予 Desktop 默认 Listener、中继或网络特权。

Settings 中的 Policy Console 是 Authority Resources 的普通调用方：它读取 Definitions、Subject Bindings、exact Grants 和 effective preview，并可在 Authority 允许时创建/更新/撤销记录。UI 会警告 all/superadmin 的未来能力覆盖，但不会因为当前产品是 Desktop 而推断允许；服务端 Forbidden 始终是最终结果。

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

登录页把“使用现有 Profile”和“首次连接”分成两个顶层入口。有保存记录时先显示 Profile 选择，没有记录时直接显示首次连接；不再用编号步骤解释填写顺序。现有 Authority Profile 的 `missing/device/pending/enrolled/error` 状态由受保护 Enrollment credential 的只读投影提供，而不是根据 Profile 的 `node_id` 猜测；Legacy Profile 保持独立兼容状态。Pending 重试复用原 request 与已固定的首次观察，Enrolled 直接连接，凭据损坏或 Profile/Grant Node ID 冲突会阻止连接并显示恢复提示。

新建 Profile 使用 Enrollment：客户端自动生成稳定、合法的本地 Profile ID，先准备仅含密钥的 Device identity，连接父 endpoint 后通过 Permit 直接注册或等待审批；Node ID 与签名 Grant 由 Admission Authority 下发并原子保存。普通表单不要求用户输入 Profile ID、Node ID 或公钥 pin；父节点公钥不再是常规必填项，无预置锚时采用显式 TOFU 并在首次成功后固定。注册前没有 Node ID，因此 authority Profile 只在 `missing/device/pending` 阶段打开窄 Enrollment bootstrap；Grant 持久化后必须先关闭 bootstrap，再由受保护凭据作为 NodeHost 的身份与父信任来源。已有 `enrolled` Profile 启动时跳过 Enrollment。所有活动的 authority 与 version 2 Legacy Profile 都由 Parent-only NodeHost 持有运行时和 attached Client。

设置格式为 version 2。每个 Profile 隔离 endpoint、本机 Node identity、受信任父节点和 Views；单个应用
实例只激活一个 Profile。登录成功后保存 Profile 与身份，下次启动可自动连接。切换 Profile 会先关闭旧
Host，其 subscription、session 和 connection 随之清理，再打开新身份。连接失败后的重试创建新的候选 Host，只有成功后才替换活动实例。

- 登录页支持新建、选择、编辑和显式确认删除 Profile；
- 全新 Profile 可先准备受保护的本机 identity 并复制 raw-base64 Ed25519 公钥；准备动作只保存非 active Profile，不连接、不登录，也不返回私钥；
- admission permit 只传给本次连接，不写入 settings 或日志；
- Profile 列表只读取并验证已有受保护凭据，不会因为展示选择器而创建目录、密钥或 request ID，也不会把私钥、公钥正文、Permit 或 Grant 签名投影到前端；
- Settings 的“返回 Profile 选择”会清空 active Profile 并关闭 client/subscription/session，但保留 Profile、受保护凭据、runtime state、Views 与 preference；它与“断开连接”和“删除 Profile”是三个不同动作；
- Windows 使用当前用户作用域 DPAPI 保存 Ed25519 identity；Legacy 密文位于 Profile 目录的
  `identity.dpapi`，Enrollment 设备密钥、观察到的信任锚和 Grant 位于独立的 `enrollment.dpapi`。authority Profile 不再复制生成 `identity.dpapi`；
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

Resource path tree 只在真实 Resource 行上提供上下文菜单，纯 namespace 没有 Resource 操作。菜单可由指针右键、
`Menu` 或 `Shift+F10` 打开，`Escape` 关闭后焦点回到原 tree item；查看/选择、添加到当前 View 和 capability
actions 都有键盘路径。action 列表只从当前 descriptor 的 capabilities 派生：`open` 是 session，只有声明
event schema 且没有 input/output schema 的纯 event-only capability 是 observe，其余（包括 event + operation
混合 capability）都进入显式 operation。Resource name、`permission` compatibility string 和 presentation metadata
都不参与动作分派；input/output schema 只进入共享操作面板的表单/result 路径，打开菜单或选择动作不会发起 operation。

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
资源引用、renderer、布局和 allowlisted 局部设置；当前 View document 保持 version 3，Collection member 选择、
已读取内容和 operation draft 都只存在于组件内存中，不进入 View。资源暂不可用时保留 leaf 并显示 detached 状态。

分栏达到最小尺寸后，其余面板仍按相对权重铺满剩余空间；渲染与停靠预览共用同一比例换算，不改变保存的权重。布局结构变化时废弃旧结构的临时拖动比例，避免新旧子节点数量不一致。

## Renderer

Variable 的只读和可写界面共用标题行：优先显示 definition.title，未提供时显示资源名最后一段；订阅状态点紧邻标题右侧，数组数量同排显示。根编辑器不再重复视觉标题，字段辅助技术标签与嵌套结构标题保留。

只读对象与数组使用独立的通用结构展示：复杂字段占满内容宽度，数组中的对象默认折叠，逐项展开后再显示字段，避免侧栏中递归的标签/值双列不断压缩内容。空对象和空数组明确标识；数组、字段数量和展示深度均有上限。读取成功但订阅被拒绝时保留快照，显示“没有订阅权限，当前显示已读取的快照；可手动刷新”，原始订阅诊断保留在详情中；展示层不改变权限。

工作视图中的 Variable 编辑区填满卡片内容宽度，开关对齐右侧。常态不显示底部分隔线、详情与刷新/重置/应用操作；右侧 Inspector 保留手动应用。同一资源的 View 与 Inspector 共享草稿，但仅 View 内的新操作触发自动应用：开关和枚举选择点击提交，数字和单行文本按 Enter 或失焦提交，滑块结束拖动或键盘调整后提交；多行文本与高级 JSON 保留 Enter 换行、失焦提交。中文输入法确认键不触发提交，Enter 紧接失焦只写一次，未改变的值不重复写入。加载态位于控件内部，等待期间禁用重复操作；开关和枚举显示上次确认值，写入完成后采用服务端确认值。正常输入和提交不在标题行闪现“未应用/正在应用”，无效输入和失败仍明确显示。读取、加载已有草稿、Inspector 编辑和重置不会触发自动应用；关闭 View、切换会话或资源实例会取消待提交操作。无效输入、不可用、撤权和冲突阻止提交，写入仍使用精确 CAS 并以服务端确认值为准。失败不自动重试，View 显示原因并按状态提供重试应用或放弃修改入口。

renderer 由 descriptor 的 type、capability、schema、operation result schema 和 content type 驱动：

所有可写 Variable 只展示一组编辑控件，不提供重复的“查看当前值”区域；修改先形成草稿，Inspector 手动应用、View 自动应用；尚未通过校验或未确认写入的输入仍标记为“未应用”。最外层对象/数组仅展示标题和数量并保持展开，嵌套结构仍可折叠。数组每项使用紧凑输入行，序号只展示一次，完整字段标签仍供辅助技术使用。枚举切换为文本编辑后继续按原 enum 校验；代码文本采用等宽字体。只读数据保持直接展示，长文本使用正文排版。过期、不可用和冲突提示不会收进折叠区域；尚未取得可编辑值时只保留刷新入口，已有草稿因不可用而被禁用时仍保留以便恢复。

- Resource provider 通过 schema 定义值类型、范围、步长、枚举、格式、字段和校验约束；Desktop 不从
  Resource name 猜测这些语义；
- Desktop renderer registry 根据 schema、capability 和 pane 尺寸过滤兼容组件并选择安全默认项；用户可在
  Widget 标题区切换兼容 renderer，切换只更新 View 展示设置，不调用 Resource operation；
- Variable：从标准快照直接取得 definition，布尔、枚举、数字、文本、对象和数组使用通用展示/控件，无需节点专用 schema。正常订阅状态用小圆点表示，悬停或聚焦可查看状态与更新时间；owner、revision、instance、capability 等技术信息收进默认折叠的“详情”，订阅失败、数据不可用与写入冲突仍直接显示。Inspector/View 共用持续订阅与会话控制器；可写值先形成 draft，Reset/Apply
  后通过 `expected_revision` 条件写入，失败或冲突保留 draft；
- Stream/Topic：使用有界 log/table/timeline，支持 pause、clear、filter、autoscroll，并明确显示
  publisher/sequence、gap、expired 和 subscription state；Topic publish 复用 schema form；
- Command/通用 operation：支持 schema form、显式 Execute、typed result 和 Advanced JSON fallback；
- Resource action：Explorer、Inspector 与 View Widget 复用同一个 `ResourceOperationPanel`。选择 operation 只打开
  表单/Advanced JSON，显式 Execute 后才调用；只有纯 event-only observe 与 `open` session 不走普通 operate，
  混合 event + input/output capability 仍要求显式 Execute。descriptor 更新后若 capability 消失，旧选择会明确
  失效而不是继续执行；
- Collection：只有 `mfh.collection` 且 list schema 为 `mfh.collection.list-request.v1` →
  `mfh.collection.page.v1` 时进入 browser。browser 支持有界分页、opaque cursor、revision 一致性、目录面包屑/
  返回、member selection、通用 member get 和 domain-specific detail。member-scoped `list/get/read` 分别要求
  member capability 与 Resource descriptor 双侧声明：无 `get` 的 member 只展示 list metadata 且不发起 API；
  目录进入还要求兼容的 Collection list schemas，文件读取还要求兼容的 filesystem read/content schemas；
- Filesystem content：单次 decoded payload 上限 128 KiB。JSON 使用安全结构化视图，其他文本进入转义后的
  `<pre>`，allowlisted PNG/JPEG/GIF/WebP 通过 Blob URL 展示，未知/不支持 binary 明确 fallback；HTML 只作为
  转义文本，SVG 不嵌入，二者都不会执行。异步 list/get/read 使用 generation token 丢弃过期结果，raster
  内容替换和组件卸载都会 revoke Blob URL；
- File：通过原生文件选择、目标路径、in-flight/error 状态完成上传；owner 提供的 `file/progress` 与
  `file/transfers` 可作为独立 Widget 展示实际进度。当前阻塞式上传 binding 尚不提供单次 transfer cancel；
- first-party catalog/topology/health/config/audit/notification/file payload 使用 schema ID 选择结构化
  table/status/timeline/progress adapter；
- unknown/unsupported：说明缺少或不支持的 schema，完整保留 descriptor、JSON tree/raw 数据和安全 fallback。

provider presentation metadata 只能在通过 type/schema/capability 兼容性过滤后作为默认 renderer hint，不能参与
operation 分派、指定可执行组件或放宽数据约束。View 只保存版本化
renderer ID 与 allowlisted 非秘密显示设置；不保存 value、draft、operation payload、event body、文件内容、
permission 或 credential。旧 `mfh.variable/stream/topic/command/file` renderer ID 作为 automatic alias 继续可读。

View Widget 按 pane density 暴露有界数量的 compact action buttons，其余动作进入可访问 overflow；按钮与菜单
一样只聚焦共享操作面板，不在可见、展开或切换时产生网络请求。在 `AUTHZ02` 的 authoritative effective-capability
discovery 落地前，Desktop 只能展示 descriptor 声明的“受支持操作”；最终授权仍以 server/Resource owner 返回的
`Forbidden` 为准，UI 可见性和 `permission` 字符串都不是授权结论。

加载、空、离线、Forbidden、订阅失败、缺失资源和未知 renderer 都会明确呈现。拖放具有独立键盘
激活手柄和添加按钮等价路径；Node tree 与 Resource path tree 都使用 roving focus，并实现 Arrow
Up/Down/Left/Right、Home、End、Enter 与 Space 的 WAI-ARIA 键盘路径。分区标题使用 button、
`aria-expanded` 和 `aria-controls`；上下区分隔条使用 horizontal separator 语义，支持方向键、Home/End
与双击复位；Tabs、表单和动作有辅助技术标签。通用表单采用项目内维护的 shadcn/ui 组件源码（Radix 交互基础），统一下拉、开关、复选框、滑块、进度与输入框；Tailwind 样式映射到现有 CSS tokens，不引入全局 preflight。视觉使用
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
- [Desktop Profile 入口](../specs/desktop-profile-entry.md)
- [NodeHost Runtime](../specs/node-host-runtime.md)
- [Credential 状态检查必须是只读路径](../lessons/credential-status-inspection-must-be-read-only.md)
- [Desktop Profile 入口生产化归档](../change/2026-08-31_desktop-profile-entry-production.md)
- [NodeHost、Enrollment 与 Profile 生命周期收敛](../change/2026-09-01_nodehost-enrollment-profile-convergence.md)
- [Scoped Policy Authorization](../requirements/scoped-policy-authorization.md)
- [作用域策略定义与 Authority 权限绑定](../change/2026-09-01_scoped-policy-authority.md)
- [策略状态迁移与真实授权证明](../lessons/scoped-policy-state-migration-and-live-proof.md)
- [Explorer Resource tree 变更归档](../change/2026-08-30_desktop-explorer-collapsible-resource-tree.md)

## 明确移除

- Vanilla TypeScript imperative UI 与固定产品页面；
- fixed Variable/Stream/Command kind switch 与独立子协议；
- settings version 1 的猜测迁移；
- 明文持久 permit/private identity 和前端权限裁决；
- Desktop 内置 MCP/stdio 自动化入口；未来对外 Agent 接口作为独立产品重新设计。

统一 Variable 当前合同见 [规格](../specs/unified-variable-resource.md)：仅 type_version=2，read 携带实例和精确字符串 revision；元数据更新保留草稿并推进基准，值变化提示冲突，撤权/重建禁止提交旧草稿。
