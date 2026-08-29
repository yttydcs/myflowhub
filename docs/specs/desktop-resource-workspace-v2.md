# Desktop Resource Workspace v2

## Status

这是已实现的 Desktop Profile、资源浏览器、renderer 和本地 View canonical contract。

## Product boundary

Desktop 是通用 Node/Resource 工作台。它使用 SDK 和 generated Wails boundary，不解析 wire、
不自行路由，也不把按钮可见性当作权限裁决。

## Profile and login

Profile 隔离一个 Hub endpoint、节点 identity/trust、非秘密偏好和 Views。每个应用实例最多一个
active Profile。启动时若 credential store 中存在有效凭据则自动连接；否则显示登录页。

首次 admission 分为两个明确阶段：`PrepareProfileJSON` 严格校验 Profile，在该 Profile 的 CredentialStore
中生成或复用 identity，返回 Node ID 与 raw-base64 Ed25519 公钥，并把 Profile 原子保存为非 active 记录；
该阶段不设置 parent trust、不启动连接、不接受 Permit，也不返回私钥。父节点签发一次性 Permit 后，
`LoginJSON` 才尝试连接，并且仅在成功后把 Profile 设为 active。失败时 Permit 留在当前 UI 会话供修正重试，
但不写入 settings、日志或 UI preference。

secret 只能通过 `CredentialStore` 保存，首选系统安全存储；不可用时必须明确报告并允许仅会话使用，
不得退化为 settings JSON 明文。Profile switch 先关闭 subscription/session/connection，再原子激活新配置；
失败保持可恢复的未连接状态。

## Shell

顶部只放产品标识、刷新与主题动作。左侧窄工作栏使用 tabs 切换 Resource Explorer 与 View Manager，
底部显示 active Profile 与 Connection state 并打开 Settings Tab；顶部不重复这些状态。主区使用内容 Tabs
承载 View 与 Settings，右侧 Inspector 承载临时 Node/Resource preview。Settings 打开时铺满主区并隐藏
Inspector，包含 Connection、Profile 和 Appearance。

Explorer 是上下 master/detail 分区：上方只展示 authoritative Node tree，下方只展示当前 Node 直接拥有的
Resources。Node 深度只由 `parent_id` 决定，不设固定层数。选择 Node 会更新 Resource 区并打开 Node
Inspector；选择 Resource 打开对应 renderer。Resource 可通过拖放、键盘添加或按钮加入当前 View。深层
Node 可聚焦为局部子树，顶部显示从 authority root 到焦点的 breadcrumb，并能返回完整树。

Explorer 先用一次迭代索引建立 Node hierarchy、Resource ownership 与查找 Map，再派生扁平 Node visible
rows；expanded/focused/active row 由单一状态拥有，不能分散在递归 Node component 中。Node 搜索只匹配
Node ID、display name 和 role 并保留命中 Node 的祖先路径；Resource 搜索独立匹配当前 Node 的 full name、
label 和 type。Resource 按 full name 的首个 `/` 段分组，无 `/` 的项目进入明确的 fallback group；分组标签
不得替代稳定身份 `(owner_node_id, full name)`。

只有 Node row 属于 WAI-ARIA tree，并暴露 `aria-level`、`aria-posinset`、`aria-setsize`、`aria-expanded`
和 `aria-selected`；使用 roving `tabIndex`，支持 Arrow Up/Down/Left/Right、Home、End、Enter 和 Space。
Resource 区是分组 list，使用可聚焦选择动作和 Arrow Up/Down、Home、End、Enter、Space，不伪造第二棵 tree。

## Renderer registry

renderer 按 type、capability 和 schema 选择，presentation hint 只决定默认偏好。内置 renderer 覆盖：

- Variable：当前值、更新时间和有权限时的编辑器；
- Stream/Topic：有界事件列表、pause/resume、gap 与 publisher；
- Command：schema-driven form、调用状态和明确错误；
- File：会话选择、进度、校验和完成状态；
- unknown：显示完整 descriptor 与 generic capability inspector。

renderer 不认识资源时必须使用 unknown fallback，不能隐藏资源或崩溃。

## Views

View 属于 Profile，首版只在本机保存。文档包含 version、id、name、布局和 widget 配置。
布局是响应式可调整网格，不是无限画布；每个 widget 保存 resource ID、renderer ID、grid position
和非秘密设置。加载时缺失资源显示 detached 状态，损坏或未知版本不会被静默覆盖。

## Interaction and accessibility

拖放必须有等价键盘入口；树、tabs、dialog、resize handle、commands 和表单具有可见 focus、标签与
语义状态。Node 与 Resource 区各自滚动，Explorer shell 约束 overflow/min-height，使固定的 Profile/Connection
footer 不随列表滚动。水平 resize handle 使用 `role="separator"`、`aria-orientation="horizontal"`、
`aria-controls` 和当前/最小/最大值；支持指针捕获、Arrow Up/Down（含大步进）、Home/End 与双击复位。
拖动中的比例只做本地预览，释放后提交。loading、empty、offline、forbidden、expired、gap、unknown、
detached 和 corrupt 都有明确呈现。

视觉方向是浅色默认的矿物蓝 + 石墨灰工具界面；深色使用独立 token 重配色而不是简单反色。
面板依靠连续分栏与 1px 边界组织，圆角限制在小型交互控件，禁止副标题、英文 eyebrow、装饰性卡片墙、
渐变、发光和重复状态胶囊。Theme、Explorer 展开/聚焦和 `explorer_split_ratio` 属于版本化非秘密 UI
preference，按 Profile 隔离保存。分区比例默认 `0.35`，持久值必须是 `0.2`–`0.8` 的有限数；布局同时
保证 Node 区 112 px、Resource 区 144 px 的可用最小高度，容器过短时按两者比例锁定。损坏数据回退到
light/default expansion/default split，不影响 settings、identity 或 View store。

## Performance acceptance

- Explorer 的代表性基准为 2,000 个 Node、10,000 个 Resource；构建 authority/ownership index、派生 Node
  visible rows 并完成一次精确筛选/当前 Node 分组必须在 750 ms 内完成。
- 可见树必须扁平化并为重复查询建立 Map/Set 索引；超过 50 行使用浏览器跳过离屏绘制的能力或等价
  虚拟化。当前 API 仍一次返回完整 topology 并按 Node 拉取 catalog，因此服务端 lazy-load/pagination 与
  真正 DOM windowing 作为后续协议工作；首版不能因规模增大引入第二套资源模型或伪加载状态。
- 基准由 `frontend/src/store.test.ts` 执行；阈值覆盖前端数据整形与筛选，不把网络目录加载时间混入渲染预算。
