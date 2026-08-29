# 2026-08-29 Desktop Codex 风格界面生产化

## Source

- 来源：上游 Codex 任务委派中的用户直接请求。
- 日期：2026-08-29。
- Project root：`D:/project/MyFlowHub3`。
- Canonical repository：`D:/project/MyFlowHub3/repo/MyFlowHub`。
- High-fidelity prototype：`D:/project/MyFlowHub3/repo/MyFlowHub/design-demos/myflowhub-desktop-codex-inspired.html`。
- Prototype evidence：同目录 `verification/` 下的 login、workspace、deep-tree、focused-tree 与 settings 浅/深色截图。
- Workflow branch：`refactor/desktop-codex-interface`。
- Workflow worktree：`D:/project/MyFlowHub3/worktrees/desktop-codex-interface`。

## Request Text / Source-preserving Summary

用户要求把已确认的 Codex-inspired 高保真原型实现为 MyFlowHub Desktop 的生产界面，同时继续使用现有 Wails、React、Profile、Resource、Renderer 与 View 模型，不能把原型中的硬编码演示数据复制成生产状态。

必须覆盖：

- 登录与持久化、多 Profile；
- 浅色默认与完整深色主题；
- 左侧 Resource / View 切换；
- 任意深度的 Node/Resource tree、独立类型图标、紧凑缩进、展开状态、搜索、深层节点聚焦、路径面包屑与返回；
- Node/Resource preview；
- Resource 拖放或按钮添加到 Workspace，并保存为 View；
- 左下角显示当前 Profile 与连接状态，点击后打开 Settings；
- Settings 作为完整 Tab 内容铺满主内容区，包含 Connection、Profile 与 Appearance；
- 顶栏不重复显示 Connection 或 Profile；
- 矿物蓝 + 石墨灰、平面分栏、无副标题、无卡片墙、克制圆角；
- 树模型不依赖固定层数，大树需要明确的渲染策略并遵循 WAI-ARIA tree 键盘交互。

## Problem / Opportunity

现有生产前端已经具备安全 Profile、自动连接、资源发现、Renderer、拖放和 View 持久化，但界面仍与确认原型存在结构性差异：顶栏重复 Connection/Profile；预览嵌入 Workspace；Settings 仍借用 Login shell；Explorer 递归组件把展开状态分散在每个节点，缺少完整 tree 键盘模型、深层聚焦和路径返回；视觉仍包含宣传式登录、英文 eyebrow、胶囊状态和独立卡片。

本轮机会是在不改变 Resource 协议和持久化边界的前提下，把现有能力重新组织成连续、低噪声、适合长期使用的桌面工作台，并使深树交互真正数据驱动。

## Confirmed Goals

- 以原型的布局、视觉和交互层级作为主要规格，以现有代码与 stable docs 作为数据、安全和架构事实源。
- 保留 Wails Go host、`DesktopAPI`、Profile/CredentialStore、Renderer Registry 和 View store 的既有边界。
- Explorer 使用迭代构建与扁平可见行，不以递归组件深度或固定 demo 层数作为交互状态。
- 实现 WAI-ARIA tree 的 roving focus 和 Arrow/Home/End/Enter/Space 键盘路径。
- 搜索保留命中节点的祖先路径；深层 Node 可聚焦为子树，并提供 breadcrumb 和返回完整树。
- 大树首版采用扁平可见行、行级稳定键、`content-visibility` 和单次 O(n) 索引；不新增与当前全量 topology/catalog API 不匹配的伪 lazy-load 模型。
- Theme 与 Explorer 展开/聚焦偏好按 Profile 使用版本化本地 UI preference 保存；secret、permit 和资源正文不进入该存储。
- Settings 成为主区 Tab，并在打开时隐藏 Inspector；Connection/Profile 操作继续调用真实 `DesktopAPI`。
- 项目图标由另一任务独立处理；本 workflow 保持现有品牌槽和图标资产不变，不等待图标结论。

## Non-goals

- 不改变 Resource descriptor、wire、authority、permission、subscription 或 session 协议。
- 不实现新的服务端 tree pagination/lazy-load API；该能力需以后单独设计协议与缓存一致性。
- 不新增生产 Media、View cloud sync、多 Profile 同时在线或移动端工作区。
- 不复制原型中的 Node、Resource、Profile 或 View 演示数据。
- 不设计、替换、定稿、生成或归档任何项目图标资产；相关方向和平台派生属于另一任务。
- 不推送、发布、签名或部署。

## Assumptions

- 现有 `Topology` 与 per-Node `Catalog` 仍是生产数据源；任意深度来自 `parent_id` 关系。
- Profile 编辑继续通过 `saveProfile`/`login`/`switchProfile`/`deleteProfile` 完成；本轮不升级 settings schema。
- 主题和纯 UI 树状态属于非秘密偏好，可按 Profile 存入 WebView local storage；损坏数据回退到显式默认值，不影响 Profile/CredentialStore。
- Desktop 最小窗口继续以桌面工作台为目标，不承诺移动布局。

## Open Questions

- Blocking: none。用户已明确将图标工作移出本任务，并要求界面 workflow 持续推进到归档。
- Deferred: 服务端按需拓扑/目录分页、真正的 DOM windowing 阈值和 View 多 Tab 会话恢复可在代表性规模超过现有预算后单独规划。

## Options Considered

| Direction | Benefits | Costs / Risks | Decision |
| --- | --- | --- | --- |
| 只重写 CSS，保留现有组件结构 | 改动最小 | 无法解决 Inspector、Settings、tree keyboard/focus 与顶栏信息架构问题 | Rejected |
| 把原型 HTML/JS 直接移入 React | 视觉接近快 | 硬编码数据、命令式 DOM 和固定深度会破坏生产模型 | Rejected |
| 保留后端/domain contract，重构 React shell、Explorer state 与 Settings | 同时满足生产数据、安全边界、可访问性和确认视觉 | 需要协调 App、Explorer、Workspace、Settings、Login 与测试 | Recommended |
| 引入新的 tree/grid/UI 框架 | 可获得现成能力 | 增加依赖、迁移风险，并不能替代产品状态模型 | Rejected |

## Recommended Direction

采用第三种方案：保留现有 API、Renderer 和 View domain，建立平面三栏 shell；主区维护 Workspace/Settings Tab；右侧 Inspector 只承载当前选择；Explorer 由纯函数生成可见扁平行并由单一 expansion/focus state 驱动；Login 与 Settings 共享 Profile editor 语义但保持不同入口密度；CSS tokens 直接落实矿物蓝、石墨灰、浅色默认和完整深色重配色。

## Constraints and Risks

- `SaveProfileJSON` 会关闭并重新打开 active client；Settings 保存后的连接状态必须明确刷新，不能假装仍已连接。
- Renderer 订阅和 Snapshot 不能因 Inspector/Tab 切换产生泄漏；现有 effect cleanup 必须保留。
- 搜索、展开、聚焦和 roving focus 同时存在时，必须避免把焦点留在已经不可见的行。
- 拖放必须继续有按钮/键盘等价入口；不能因为原型中的原生 drag demo 而删除 dnd-kit 的可访问路径。
- 大树优化不能引入第二套 Resource ownership；扁平行只是一种派生展示结构。

## Stable Docs Impact

- Intake impact: add（本文）。
- Feature impact: clarify/update `docs/features/desktop.md` 的 shell、Settings、Explorer、theme 与 preview 当前事实。
- Requirements impact: clarify `docs/requirements/desktop-resource-workspace.md` 的深树聚焦、breadcrumb、设置入口与浅色默认验收。
- Specs impact: clarify/update `docs/specs/desktop-resource-workspace-v2.md` 的 UI state、tree keyboard contract 和大树分期边界。
- Decision impact: none；项目图标决策不属于本 workflow。
- Lessons impact: planning 时查阅现有 frontend/Wails lessons；是否新增 lesson 在 test/archive 后决定。

## Worktree / Branch / Docs Root Status

- Branch: `refactor/desktop-codex-interface`。
- Base: `master@b92f0c7239ef331ed06fc3ac258851ff7b20cc1b`。
- Active worktree: `D:/project/MyFlowHub3/worktrees/desktop-codex-interface`。
- Docs root: `D:/project/MyFlowHub3/worktrees/desktop-codex-interface/docs`，与 canonical code 同仓；无 remote，不 push 或发布。
- 主检出中的 `guide.md`、`论文/**` 与未跟踪 `design-demos/` 仅作为输入读取，不进入 worktree dirt。

## Handoff Criteria for `$m-plan`

- Problem、目标、非目标、推荐方向和 deferred boundary 均已明确。
- stable docs 与生产代码边界已定位。
- dedicated branch/worktree 已就绪。
- Blocking: no；可返回 `$m-plan` 并展示 Task ID 审批门。
