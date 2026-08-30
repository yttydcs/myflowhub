# Desktop Explorer 可折叠分区与 Resource path tree

## Source

- 日期：2026-08-30
- 来源：用户对真实 Desktop 工作区的手动验收截图与补充需求。
- 原始诉求保真摘要：左侧“节点”和“资源”分区的标题应可点击收起；下方 Resource 不再使用平铺分组列表，
  而应使用树状展示。

## Problem / Opportunity

当前 Explorer 已把 authoritative Node tree 与当前 Node 的直接 Resources 分为上下两区，并允许调整高度，
但两个分区始终同时占用空间；当用户只关注 Node 或只关注 Resource 时，无法把另一部分临时收起。Resource
区域只按名称首段分组，`system/config/update` 等深层相对路径仍被压成同一层列表，不能表达路径之间的浏览关系。

协议已经把 Resource name 定义为由非空 `/` segment 组成的相对 UTF-8 path。Resource path 只是当前
owner 下的展示与寻址结构，不是第二棵 authority tree。实现还必须处理一个 segment 既代表真实 Resource、
又是其他 Resource 前缀的情况，例如 `system/config` 与 `system/config/update` 同时存在。

## Confirmed Goals

- “节点”和“资源”标题行都成为可发现的 disclosure control，可通过指针、Enter 与 Space 展开/收起。
- 收起一个分区后，另一个分区占满 Explorer 可用高度；重新展开后恢复上次上下比例。
- 下方只展示当前 Node 直接拥有的 Resources，但按完整 Resource name 的任意深度 path 构建树。
- namespace、中间 segment、真实 Resource leaf，以及“既是 Resource 又有 children”的混合项都能正确显示。
- Resource tree 保留搜索、选择、Inspector preview、类型图标、拖拽和“添加到工作区”的等价入口。
- Node tree、Resource tree、折叠状态、独立滚动与左下 Profile/Connection footer 保持互不干扰。
- 树与 disclosure 遵循 WAI-ARIA 键盘和状态语义，状态按 Profile 隔离保存。

## Non-goals

- 不把 Resource 插回 authoritative Node tree，也不建立第二套权限、路由或 ownership tree。
- 不改变 Hub catalog、Resource ID、权限裁决、Wails API 或 View/Workspace docking 数据模型。
- 不以 Resource type 作为第一层树根；type 继续决定图标与 renderer，不覆盖 path 结构。
- 不新增布局按钮、依赖库、服务端分页或伪造按需加载。
- 不处理品牌图标、发布、推送或其他主检出中的独立改动。

## Assumptions

- 两个分区默认同时展开；最多只允许一个分区处于收起状态，避免 Explorer 只剩空白区域。
- 点击已收起的标题恢复双区；当另一区已收起时，尝试收起当前唯一展开区不会产生“双收起”状态。
- disclosure 只隐藏受控内容，不清空搜索词、当前选择或树展开状态。
- Resource tree 默认展开第一层 path，深层分支保持收起；搜索期间临时展开匹配项祖先，清空搜索后恢复用户状态。
- Resource path 展开状态和 Explorer 收起状态使用现有 per-Profile、versioned、非秘密 UI preference 保存；
  新字段保持可选，因此不需要迁移旧偏好文档。

## Constraints And Risks

- `system/config` 与 `system/config/update` 要求同一 treeitem 同时支持选择/操作与展开/收起；不能用
  “folder 一定不可选、leaf 一定无 children”的文件系统假设。
- disclosure header、treeitem 主动作、拖动手柄和添加按钮必须保持清晰的焦点顺序；不能让整行嵌套按钮。
- 搜索只改变派生可见行，不能把临时展开结果污染持久状态。
- 树构建与扁平化必须保持线性有界；大型 catalog 继续使用单一派生模型、扁平可见行和离屏优化。
- 当前 topology/catalog API 不支持服务端分页；真正 lazy loading 或 DOM windowing 仍是后续明确分期边界。

## Options Considered

| 方向 | 结构 | 优点 | 主要问题 | 结论 |
| --- | --- | --- | --- | --- |
| A. Resource path tree | 当前 Node → 任意深度 path segment → Resource | 与 Resource ID 一致；支持深层浏览；不改变 authority | 需要处理混合项、展开状态与完整 tree 键盘语义 | 推荐 |
| B. Type-first tree | 当前 Node → type → path → Resource | 同类型资源易扫描 | 人为增加一层；同一路径按 type 分裂；type 已由图标表达 | 拒绝 |
| C. Node/Resource 混合树 | Node children 与 Resource children 共用一棵树 | 表面上只有一个浏览控件 | 重回已验证过的拥挤结构；混淆 authority 与 presentation | 拒绝 |
| D. 可折叠分组列表 | 只让现有首段分组变成 accordion | 改动最小 | 仍不能表达任意深度，也不满足真正树导航 | 拒绝 |

## Recommended Direction

### Explorer section disclosures

- Node 与 Resource heading 各使用一个原生 `button`，包含方向 chevron、名称、数量和
  `aria-expanded`；`aria-controls` 指向各自内容容器。
- 标题按钮占满标题行的可点击区域，但不包裹其他可交互元素；Enter/Space 使用浏览器原生按钮行为。
- 两区展开时显示现有 separator；任一区收起时隐藏 separator，由另一分区占满剩余高度。
- 重新展开恢复持久的 `explorer_split_ratio`，收起动作本身不重写比例。
- 状态建议表示为可选 `collapsed_explorer_pane: "node" | "resource"`，从数据模型上排除双收起。

### Resource path tree

- 从当前 Node 的 `ResourceDescriptor[]` 构建一次 `ResourcePathNode` trie；节点 key 必须包含 owner 与完整
  path，不能只用最后一个 segment。
- 每个 path node 可同时拥有 `resource?: ResourceDescriptor` 和 `children`。只有拥有 `resource` 的项可选择、
  preview、拖拽或添加；纯 namespace 项只负责展开与导航。
- 可见行由展开集合迭代扁平化，并声明 `aria-level`、`aria-posinset`、`aria-setsize`、`aria-expanded` 与
  单一 roving tab stop。Right/Left、Up/Down、Home/End、Enter/Space 与可选 `*` 沿用 Node tree 约定。
- Resource item 继续显示类型图标；纯 namespace 使用中性的路径/文件夹图标。混合项显示 Resource 类型图标
  和 disclosure affordance，不牺牲任何一个动作。
- 搜索匹配完整 name、presentation label、type 与 segment；匹配结果保留祖先并临时展开路径。清空搜索后
  回到 `expanded_resource_paths`，不修改用户保存的展开集合。

## Research Summary

- W3C APG Disclosure Pattern 要求控制元素使用 button，通过 `aria-expanded` 表达状态，可选
  `aria-controls` 指向内容；Enter 和 Space 切换可见性。
- W3C APG Tree View Pattern 要求所有可导航项使用 treeitem，父项只有在确有 children 时声明
  `aria-expanded`，并实现方向键、Home/End 与单一复合控件焦点模型。焦点和选择是不同状态。
- 这支持“分区标题用 disclosure、资源内容用完整 tree”的职责拆分；不应把 disclosure accordion 冒充
  Resource tree。

## Stable Docs Impact

- Intake impact: added（本文）。
- Feature impact: clarified `docs/features/desktop.md` 的 Explorer 目标行为。
- Requirements impact: clarified `docs/requirements/desktop-resource-workspace.md`，用 path tree 取代“首段分组
  list”，并加入分区 disclosure 验收。
- Specs impact: clarified `docs/specs/desktop-resource-workspace-v3.md`，加入 disclosure、Resource trie、
  扁平可见行、搜索与 preference 契约。
- Decision impact: none；这是现有 Node ownership、Resource relative path 与 Desktop Explorer 边界内的可逆 UI 结构。
- Lessons impact: none；尚未形成可复用事故模式。

## Worktree / Repository Status

- Project root: `D:\project\MyFlowHub3`
- Docs root: `D:\project\MyFlowHub3\worktrees\desktop-explorer-resource-tree\docs`
- Code repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Base branch / commit: `master` / `550dfeebd55da5af4abce8a4c718ff30b77f5981`
- Active branch: `feat/desktop-explorer-resource-tree`
- Active worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-resource-tree`
- 原主检出的未提交品牌、文档和生成物改动未被暂存、覆盖、还原或带入本 worktree。

## Open Questions

- 无阻塞问题。第一层默认展开数量、缩进和行高允许在真实 GUI 测试中微调，但不得改变上述信息模型、
  键盘行为或持久化边界。

## Handoff Criteria For `$m-plan`

- 以方向 A 为唯一目标方案。
- 计划必须覆盖数据派生、Resource tree 交互、section disclosure、per-Profile preference、样式、测试与受治理文档。
- 验收至少覆盖任意深度 path、混合 Resource/branch、搜索祖先、折叠后焦点不可达、比例恢复、浅/深色和真实 Wails GUI。
- 当前讨论已达到可规划状态；尚未批准实施，也未创建 `plan.md` / `todo.md`。

## Related Change

- [Desktop Explorer 可折叠分区与 Resource path tree](../change/2026-08-30_desktop-explorer-collapsible-resource-tree.md)
