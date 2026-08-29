# Desktop 资源工作区

## Background

当前 Desktop 已能连接父 Hub、浏览 catalog、订阅和调用资源，但前端仍是 Vanilla TypeScript 与固定产品页面。新的产品方向要求 Desktop 成为 Node/Resource 平台的主要可视化入口，支持多个连接身份、资源树、类型化预览、拖放组合和可持久化 View。

## Goal

提供极简、舒适、可访问的 Desktop 工作区，让用户可以安全登录不同 Profile，发现整棵 Node tree 下的 Resource，预览资源，并把资源组合成可恢复的 View。

## Scope

- React + TypeScript + Vite、Radix/shadcn 风格组件 primitives 与统一 design tokens 前端。
- 登录/首次准入、持久身份、自动重连和多 Profile 管理。
- 左侧 Resource Explorer 与 View Manager Tabs。
- 右侧 Node/Resource preview 与响应式可调整 Workspace。
- Resource Renderer Registry、通用 fallback renderer 和基础类型 widgets。
- View 按 Profile local-first 持久化。
- Wails Go host、generated bindings、前端状态与错误边界重构。

## Scenarios

- 首次启动创建 Profile、配置父 Hub、完成准入并进入工作区。
- 再次启动恢复上次 Profile，并在身份仍有效时自动连接。
- 在两个 Profile 之间切换，旧连接和订阅被明确释放，新 Profile 状态独立加载。
- 在资源树中搜索/展开 Node，单击 Node 查看概览，单击 Resource 查看预览。
- 通过拖放、按钮或键盘把 Resource 添加到当前 View。
- 调整 Widget 顺序和尺寸，保存、重命名、复制、删除并重新打开 View。
- Resource 离线、被撤权或类型升级后，View 保留位置并显示可恢复错误状态。

## Functional Requirements

1. Login shell 必须支持 Profile chooser、create/edit/delete、首次 admission 和明确错误恢复；全新 Profile 必须能在连接前生成或复用 CredentialStore identity，并只展示用于父节点签发 Permit 的公开身份。
2. 每个 Profile 隔离 Hub/Transport 设置、本地 Node identity、受信任父节点、View、最近项和 UI preferences。
3. 首版每个应用实例只能激活一个 Profile；切换必须先关闭旧连接、subscription 和 session。
4. 身份准备不得激活 Profile、启动连接或导出私钥；一次性 permit 在成功 admission 后不得继续作为长期明文配置保存，私钥或 refresh secret 必须通过 CredentialStore abstraction 保护。
5. Resource Explorer 必须按 authoritative Node tree 展示 Node，并在 owner 下展示 Resource；Resource path 分组只影响显示，不建立第二棵 authority tree。
6. Explorer 必须支持搜索、展开/折叠、选择、加载、空状态、断线状态和键盘导航。
7. Explorer 必须支持任意深度的 Node 关系、独立资源类型图标、深层 Node 聚焦、路径面包屑和返回完整树；
   实现不得依赖固定层数或演示数据。
8. Node preview 至少显示身份、连接/健康状态、父子关系和资源摘要；Node/Resource 临时预览位于右侧
   Inspector，不得被误写入 View。
9. Renderer Registry 必须按 type/schema/capability 选择 preview、widget 和 editor/action，并为未知类型提供通用 inspector。
10. Workspace 必须支持添加、移除、重排和调整 Widget；拖放不是唯一入口。
11. View 只保存 Resource reference、renderer ID/version、layout 和局部展示设置，不复制 Resource 正文或权限。
12. View store 必须版本化、原子提交、按 Profile 隔离，并拒绝损坏或不兼容数据的静默覆盖。
13. Variable、Stream、Topic、Command、File 至少有可用的第一方 renderer；Media 在生产数据面完成前使用明确 capability/session placeholder。
14. 所有权限错误、资源消失、连接恢复、subscription gap 和 session 失败必须在对应 Widget 内明确显示。
15. 左下角必须显示 active Profile 与 Connection state，并打开铺满主区的 Settings Tab；Settings 至少包含
   Connection、Profile、Appearance，顶栏不得重复 Profile 或 Connection。

## Non-functional Requirements

- 视觉遵循矿物蓝 + 石墨灰、平面分栏、低噪声、克制圆角、清晰层级和系统字体/CJK fallback；不使用
  副标题、装饰性卡片墙、渐变或发光。
- 浅色为默认主题，深色必须完整重配色；支持减少动画偏好。
- 树浏览、拖放、View 管理和核心操作必须具有键盘等价路径与可读辅助技术标签；Tree 遵循 WAI-ARIA
  tree 的 roving focus 与方向键/Home/End/Enter/Space 交互。
- Explorer 必须使用单一派生树模型和扁平可见行，避免递归组件状态与热点 O(n²)；超过 50 行使用
  `content-visibility` 或等价离屏策略。服务端按需加载或真正 DOM windowing 在现有 topology/catalog API
  不支持时必须作为明确分期边界记录，不能用第二套资源模型伪装。
- 前端不得自行裁决权限、解析 wire 或复制产品 runtime。
- Wails binding input 在 Go boundary 再次执行 schema、大小和路径校验。

## Edge Cases

- 上次 Profile 被删除、凭据不可读、Hub 不可达或父身份改变。
- View 引用不存在的 Node、Resource、renderer 或旧 major schema。
- 同名 Resource 位于不同 Node。
- 拖放中连接断开或 Resource 被撤权。
- Explorer 节点数很大、节点循环加载失败或局部重连。
- View store 损坏、磁盘只读或原子 rename 失败。

## Acceptance Criteria

- 两个 Profile 的身份、连接设置和 Views 相互隔离；切换后旧订阅数量归零。
- 成功登录后重启可以自动进入/连接；无效身份必须回到明确登录恢复流程。
- Resource Explorer 展示跨子树 Node 和 Resource，并通过鼠标与键盘完成预览/添加。
- 至少 6 层 Node fixture 可展开、搜索、聚焦并用 breadcrumb 返回；Arrow/Home/End/Enter/Space 键盘门禁通过。
- Resource 可拖入工作区、调整布局、保存 View，重启后恢复完全一致。
- 左下 Profile/Connection 入口能打开 Settings Tab；连接、Profile CRUD/切换和浅/深色在真实 API/mock boundary 下可操作，顶栏无重复状态。
- 未知类型、离线、Forbidden、Expired、Gap 和 Schema mismatch 均有独立可测试状态。
- Vitest/Testing Library、TypeScript、Vite production build、Wails production build、浏览器交互与真实 Wails GUI smoke 通过。

## Related Features

- [当前 Desktop 行为](../features/desktop.md)

## Related Requirements

- [可扩展资源平台](extensible-resource-platform.md)
- [受控准入](auth-controlled-admission.md)

## Related Decisions

- [可扩展 Resource type system 与 Desktop workspace](../decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)

## Related Intake

- [可扩展资源平台与 Desktop 工作区重构](../intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)
