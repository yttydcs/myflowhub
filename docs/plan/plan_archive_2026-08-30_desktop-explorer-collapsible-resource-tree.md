# Plan Archive - Desktop Explorer 可折叠分区与 Resource path tree

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `feat/desktop-explorer-resource-tree`
- Base: `master` @ `550dfeebd55da5af4abce8a4c718ff30b77f5981`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\desktop-explorer-resource-tree\docs`
- Code Repos: canonical monorepo `MyFlowHub` only
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-resource-tree`
- Participating Modules: `apps/desktop/frontend`、canonical `docs/`
- Current Stage: `$m-continue` / `$m-test` Passed；`$m-archive` closeout in progress
- Publication: local-only；不新增 remote，不 push/release/publish

## Stage Records

### Initialization

- `guide.md`: 已读取；遵守中文提交、canonical docs、`GOWORK=off` 与 sibling worktree 规则。
- Project/docs/code repo confirmation: project root、canonical repo 与 repo-local governed docs root 已确认。
- Base/worktree confirmation: 独立 branch/worktree 已由 `$m-discuss` 创建；原主检出仍有品牌、文档、源码和
  generated dist 的未提交改动，本 worktree 不暂存、覆盖、还原或归档这些改动。
- Root control files: 上一轮完成状态已保存在
  `docs/plan/plan_archive_2026-08-30_desktop-nested-docking-layout.md` 与对应 `docs/change`；根级
  `plan.md`/`todo.md` 现切换为本轮活动控制面。

### Discuss - Discovery And Requirements Shaping

#### Goal

让 Node/Resource 上下分区可通过标题收起，并把当前 Node 的 Resource 首段分组列表升级为完整相对路径树，
同时保留现有预览、拖放、添加、独立滚动、Profile footer 与可访问性。

#### Scope

- 两个 section disclosure；最多收起一个；另一块满高；恢复旧比例。
- 当前 owner 下任意深度 Resource path tree、混合 Resource/parent、搜索祖先和完整键盘交互。
- 折叠与 Resource path 展开状态按 Profile 保存。
- React/Vitest/CSS/production dist、稳定文档与真实 Wails GUI 验证。

#### Assumptions

- 默认双区展开；Resource 第一层 path 默认展开。
- 折叠不清空搜索、选择或展开状态；搜索临时展开不写回 preference。
- Resource descriptors 已由 canonical protocol/Wails boundary 校验；前端不重写或规范化 Resource ID。

#### Open Questions

- 无阻塞问题。缩进、行高和默认第一层展开密度可在 GUI 验收中微调，不改变信息模型。

#### Options Considered

- Resource path tree：采用。
- type-first tree：拒绝；重复 type 层且割裂相同 path。
- Node/Resource 混合 tree：拒绝；混淆 authority 与 presentation，并重现拥挤问题。
- 可折叠首段分组 list：拒绝；不满足任意深度 tree。

#### Recommended Direction

使用纯 trie/index 派生层和扁平可见行；UI 分别实现 disclosure header、Resource treeitem 与独立 action；
可选 preference 字段保持 version 1 向后兼容，不引入依赖或后端 schema。

#### Research Summary

- W3C APG Disclosure Pattern：button + `aria-expanded`，Enter/Space 切换，可选 `aria-controls`。
- W3C APG Tree View Pattern：parent 的 `aria-expanded`、roving focus、方向键/Home/End 与焦点/选择分离。
- 研究只用于交互契约；具体实现沿用现有 Node tree 与项目组件体系。

#### Worktree / Branch / Docs Root Status

- Ready。分支、worktree、docs root 与原始 intake 均已就绪。

#### Issue List

- 无。

### Plan - Requirements And Architecture

#### Discussion Summary

用户确认需要两个可点击收起的 Explorer section，并要求 Resource 为树状。讨论已明确 Resource tree 按
协议的相对 path 派生，保持与 Node authority tree 分离，并处理真实 Resource 同时作为 path parent 的情况。

#### Accepted / Rejected Requirements

- Accepted: disclosure、path trie、任意深度、hybrid item、搜索祖先、ARIA tree、per-Profile state。
- Rejected: Node/Resource 混排、type-first、固定深度、服务端层级伪造、专用依赖与双收起空侧栏。

#### Requirements Analysis

##### Goal

提高窄侧栏中 Node/Resource 浏览效率，同时保持资源身份、权限边界和操作路径不变。

##### Scope

- Frontend domain、Explorer component、split layout、UI preference、styles、tests、generated production dist。
- 不改 Go protocol、catalog、Wails bindings、View v3 或 Workspace docking。

##### Use Cases

1. 用户收起 Node 区，让 Resource tree 满高浏览；点击标题恢复双区和旧比例。
2. 用户收起 Resource 区，让深层 Node tree 满高浏览；不会把两个区域同时收起。
3. 用户展开 `system/config`，既可预览/添加该 Resource，也可继续进入 `update` child。
4. 用户搜索深层 Resource；树显示匹配项和祖先，清空后恢复先前展开状态。
5. 用户重启或切换 Profile；每个 Profile 恢复自己的 section/path 展开偏好。

##### Functional Requirements

- Header disclosure 的指针与键盘等价路径、正确状态语义和焦点隐藏。
- Resource tree 的 owner/path 唯一 key、任意深度、hybrid node、稳定排序和扁平可见行。
- WAI-ARIA tree navigation、Resource selection、Inspector、DnD 与 add action 无回归。
- Preference optional fields 的验证、上限、旧文档兼容和 Profile 隔离。

##### Non-functional Requirements

- 无新增依赖；构建/运行继续使用现有 React、dnd-kit、Radix/shadcn primitives 与 CSS tokens。
- build O(total path segments)，flatten O(visible rows)，搜索不引入逐行全树扫描。
- 浅/深色、reduced-motion、窄窗口、固定 footer 与独立滚动可用。

##### Inputs / Outputs

- Input: 当前 Node 的 validated `ResourceDescriptor[]`、query、expanded key set、collapsed pane、split ratio。
- Output: Resource path index、visible rows、selection/DnD/add callbacks、validated per-Profile preference。
- Persisted output: optional `collapsed_explorer_pane`、`expanded_resource_paths` 和现有 split ratio。

##### Edge Cases

- 单 segment、六层以上 path、Unicode segment、numeric segment 排序。
- 同一 path 同时有 Resource 与 children；不同 owner 存在同名 path。
- 空 Resource tree、搜索无结果、catalog 更新后存在陈旧 expanded key。
- 在内容或 header 获得焦点时收起；折叠期间窗口 resize；清空搜索恢复状态。
- 10,000 Resources 共享前缀或高度分散；Profile preference 损坏/超限。

##### Acceptance Criteria

- 两个 header 可指针/Enter/Space 切换，状态和 `aria-controls` 正确，最多一个收起。
- 收起后 sibling 满高、separator 不可达；展开后比例不变，footer 始终固定。
- 任意深度、namespace、leaf、hybrid item 均有正确可见行和 ARIA 元数据。
- Resource keyboard、preview、drag/add、搜索祖先与 per-Profile 恢复通过自动化和 packaged GUI。

##### Risks

- Explorer.tsx 已承载 Node tree、Resource list 与 DnD，若直接继续堆逻辑会降低可维护性。
- treeitem 与 drag/add/disclosure 是多个交互入口，错误 DOM 结构会造成重复 Tab stop 或嵌套按钮。
- 原主检出的 `App.tsx`、`style.css` 与 generated dist 有独立未提交改动，归档合并必须再次安全收敛。

#### Architecture Design

##### Overall Solution

新增独立纯函数模块 `src/lib/resource-tree.ts`，把当前 Node Resources 构建为 trie/index，并根据 expanded/query
输出扁平行；`Explorer.tsx` 只持有查询、roving focus 与事件编排。`ExplorerSplitPane` 接收 union collapse
state 并切换 grid rows/separator。`UIPreferences.version=1` 增加可选字段，由 App 传递和持久化。

##### Alternatives Considered

- 把 trie 继续塞入 `store.ts`：可行但会扩大已经混合 Node/View helpers 的模块；拒绝。
- 递归 `ResourceTreeNode` 组件持有局部 state：易写但状态分散、搜索/性能/键盘困难；拒绝。
- 引入 tree/virtualization library：当前上限与组件体系不需要；拒绝。

##### Module Responsibilities

- `src/lib/resource-tree.ts`: path validation assumptions、trie/index、default expansion、query ancestors、flatten rows。
- `src/components/Explorer.tsx`: section header、Resource row/tree ARIA、roving focus、selection、drag/add wiring。
- `src/components/ExplorerSplitPane.tsx`: 双展开/单折叠 layout、separator availability、ratio preview/commit。
- `src/preferences.ts` + `App.tsx`: optional state validation、Profile persistence、props/callback plumbing。
- `src/style.css`: disclosure header、tree indentation、collapsed layout、light/dark/focus/overflow。
- tests: domain invariants、component interaction、preference compatibility、integration persistence。

##### Data / Call Flow

`catalog → current owner Resources → buildResourceTree → flattenResourceTreeRows(expanded, query) → Explorer rows →
selection / preview / DnD / add`。

`localStorage(Profile) → UIPreferences → App → Explorer/ExplorerSplitPane → callbacks → validated saveUIPreferences`。

##### Interface Drafts

```ts
type ExplorerCollapsedPane = 'node' | 'resource'

interface ResourcePathNode {
  key: string
  ownerNodeID: string
  path: string
  segment: string
  resource?: ResourceDescriptor
  children: string[]
}

interface ResourceTreeRow {
  key: string
  parentKey?: string
  depth: number
  posInSet: number
  setSize: number
  node: ResourcePathNode
}

interface UIPreferences {
  version: 1
  collapsed_explorer_pane?: ExplorerCollapsedPane
  expanded_resource_paths?: string[]
}
```

最终命名可按周边代码微调，但 owner/full path 唯一性、optional preference 与扁平行字段不可丢失。

##### Error Handling and Safety

- 不重写 validated Resource name；duplicate identity 继续由 catalog boundary 拒绝。
- preference union、数组长度和 key 长度不合法时沿用显式默认值与 warning，不静默部分接受损坏文档。
- 陈旧展开 key 只在派生时无匹配，不影响 catalog、selection 或保存的 View。
- 收起时移除受控内容的可见性与焦点可达性，不以负尺寸或 CSS 覆盖隐藏可交互元素。

##### Performance and Testing Strategy

- 纯函数测试覆盖 10,000 Resources、深层/共享前缀、hybrid、query ancestors 与 owner isolation。
- Testing Library 覆盖 role/state/keyboard/focus/DnD/add；现有 split pointer capture 和 ratio tests 保留。
- 分层执行 frontend test/build、全仓 `GOWORK=off go test ./...`、Wails production build 和真实 GUI smoke。

##### Extensibility Design Points

- path node 保留 optional Resource，使未来 namespace metadata 不需要重写可见行算法。
- 扁平行可在 API 支持分页后接入 DOM windowing，但本轮不提前引入分页状态。
- Collapse union 可以未来扩展为 sidebar-wide hide，但不与 Workspace docking layout 混合。

#### Issue List

- 无。

### Stage 3.1 - Planning

#### Project Goal and Current State

上一轮 Node/Resource 上下分区和 n 元停靠已在 `master` 完成。本轮只升级 Explorer 分区 disclosure 与 Resource
展示树，不改变已归档的 View v3 和 docking contract。当前只有 planning/docs 改动，业务逻辑和测试未修改。

#### Docs Governance Routing Decision

- Docs root 是 canonical repo 的 `docs/`；本 worktree 内编辑，之后随代码本地归档。
- 原始请求：`docs/intake/2026-08-30_desktop-explorer-collapsible-resource-tree.md`。
- 用户行为、长期验收和技术契约分别路由到 feature、requirement、spec；无需新 ADR 或 lesson。
- 根级 `plan.md`/`todo.md` 是活动控制面例外；完成后归档到 `docs/plan` 和 `docs/change`。

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake: `docs/intake/2026-08-30_desktop-explorer-collapsible-resource-tree.md`
- Feature: `docs/features/desktop.md`
- Requirement: `docs/requirements/desktop-resource-workspace.md`
- Spec: `docs/specs/desktop-resource-workspace-v3.md`
- Decisions: `docs/decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md`、
  `docs/decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md`
- Lessons: `docs/lessons/frontend-and-powershell-preflight.md`、
  `docs/lessons/windows-clean-checkout-eol-and-generated-drift.md`、
  `docs/lessons/frontend-worktree-wailsjs-missing.md`

#### Stable Docs Impact

- Intake impact: add
- Feature impact: clarify
- Requirements impact: clarify
- Specs impact: clarify
- Decision impact: none
- Lessons known at planning time: existing three lessons apply；不新增 lesson，执行/测试后复核

#### Executable Task List

| Task ID | Title | Scope | Depends On |
| --- | --- | --- | --- |
| DOC01 | 稳定文档与控制面收敛 | Will execute | none |
| TREE01 | Resource path-tree 领域层 | Will execute | DOC01 |
| UI01 | Resource tree 与 section disclosure UI | Will execute | TREE01 |
| PREF01 | 折叠/展开 per-Profile preference 与 split layout | Will execute | TREE01 |
| QA01 | 自动化、构建与真实 GUI 验证 | Will execute | UI01, PREF01 |
| LAZY01 | 服务端分页、lazy loading 与真正 DOM windowing | Will not execute now | Deferred: API 不支持，独立性能阶段 |
| BRAND01 | 品牌图标与平台资产 | Will not execute now | Out of scope: 独立任务拥有 |
| ARC01 | change/plan 归档、合并与 worktree 清理 | Will not execute now | 仅在 QA 通过且用户调用 `$m-archive` 后执行 |
| PUB01 | push/release/publication | Will not execute now | 未授权且仓库无 remote |

#### Execution Scope After Approval

##### Will Execute

- `DOC01, TREE01, UI01, PREF01, QA01`

##### Will Not Execute Now

- `LAZY01`：现有 topology/catalog API 无分页；另立性能工作流。
- `BRAND01`：独立品牌任务拥有，避免覆盖主检出未提交资产。
- `ARC01`：属于 `$m-archive`，必须在实施和测试通过后单独进入。
- `PUB01`：没有用户授权且无 remote。

#### Task Details

##### DOC01 - 稳定文档与控制面收敛

- Owner: primary agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-resource-tree`
- Plan Path: root `plan.md` / `todo.md`
- Goal: 保持 intake → feature → requirement → spec → plan 的可追溯一致性。
- Files / Modules: `docs/intake/**`、`docs/features/desktop.md`、
  `docs/requirements/desktop-resource-workspace.md`、`docs/specs/desktop-resource-workspace-v3.md` 与 indexes。
- Write Set: 仅上述 docs 与根控制面；不写 change/plan archive。
- Acceptance: 旧“首段分组 list”不再作为当前目标；索引与链接完整；decision/lessons impact 明确。
- Test Points: link/path 检查、`git diff --check`、稳定文档术语搜索。
- Rollback: 恢复本轮 docs/control-plane patch，不影响 runtime。

##### TREE01 - Resource path-tree 领域层

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 以纯函数构建 arbitrary-depth trie/index 和扁平可见行。
- Files / Modules: `apps/desktop/frontend/src/lib/resource-tree.ts`、对应 `.test.ts`，必要时清理 `store.ts` 的
  `groupResources`/`ResourceGroup`。
- Write Set: 纯 TypeScript domain/test；不触碰 Go protocol。
- Acceptance: namespace/leaf/hybrid/owner isolation/排序/search ancestors/default expansion/10k fixture 通过。
- Test Points: focused Vitest + existing `store.test.ts` 回归。
- Rollback: 删除新模块并恢复旧 `groupResources` 消费。

##### UI01 - Resource tree 与 section disclosure UI

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 用扁平行渲染完整 Resource tree，并让两个 heading 成为可访问 disclosure。
- Files / Modules: `components/Explorer.tsx`、`components/Explorer.test.tsx`、必要时新增局部 Resource row component、
  `style.css`。
- Write Set: Explorer component/styles/tests；保留 Node tree、Inspector、DnD 和 add callback。
- Acceptance: ARIA roles/states、roving focus、hybrid item、搜索、preview/drag/add、light/dark 均正确。
- Test Points: Testing Library role/keyboard/focus；无嵌套按钮或重复主 Tab stop。
- Rollback: 恢复旧 grouped list JSX/CSS；domain 模块可独立移除。

##### PREF01 - 折叠/展开 per-Profile preference 与 split layout

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 持久化 collapse/resource expansion，并让 split 在双展开和单折叠间稳定切换。
- Files / Modules: `preferences.ts/.test.ts`、`App.tsx/.test.tsx`、
  `components/ExplorerSplitPane.tsx/.test.tsx`、相关 props/types/style。
- Write Set: optional v1 fields、App plumbing、collapsed grid rows/separator。
- Acceptance: 旧 preference 可读；非法新字段显式回退；最多一个收起；ratio 不被折叠覆盖；Profile 隔离。
- Test Points: localStorage integration、pointer/keyboard separator regression、hidden focus exclusion。
- Rollback: 删除可选 fields/props，恢复 always-expanded grid；旧持久文档仍兼容。

##### QA01 - 自动化、构建与真实 GUI 验证

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 以风险相称验证功能、性能、generated dist 与 packaged Desktop。
- Files / Modules: frontend tests、`frontend/dist/**`、必要的 `artifacts/m-test/**`（仅在重测试阶段）。
- Write Set: 测试修正和 production dist；不写品牌资产或主检出用户文件。
- Acceptance: frontend/Go/build/Wails 通过；真实 Hub 下 disclosure、hybrid path、search、DnD/add、重启持久化通过。
- Test Points:
  - 新 worktree 先检查锁文件依赖与 `wailsjs`；缺失时只运行 canonical generate 入口，不跨 worktree 复制
  - `npm ci`（仅在依赖未安装或不完整时）
  - `npm test`
  - `npm run build`
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `wails build -clean -platform windows/amd64`
  - packaged Wails light/dark、窄窗口、固定 footer、Profile restart smoke
- Rollback: 恢复实现提交并重新生成旧 production dist；不修改用户 Profile/View 数据。

#### Dependencies

- `TREE01 → UI01/PREF01 → QA01`；DOC01 已在计划阶段建立契约，执行时随实现校准。
- UI01 与 PREF01 都修改 Explorer props/tests，顺序执行以避免冲突。
- Archive/merge 必须在 QA 通过后单独处理原主检出 dirty state。

#### Risks and Notes

- `system/config` hybrid fixture 是结构门禁，不得只测试普通 folder/leaf。
- preference 保持 version 1 optional extension；不要为了两个字段强制迁移。
- 不提交 `apps/desktop/build/bin`；tracked `frontend/dist` 必须由最终源码生成。
- Windows 上在 generate/build 前后检查 tracked status 与 EOL drift；不得把环境噪声误当业务变更。
- 原主检出当前 dirty 内容是用户/独立任务资产；archive 时需要逐路径保存、合并和复验。

#### Parallelism Assessment

- 不派发实现 sub-agent。TREE/UI/PREF 共享 Explorer 类型、焦点和 preference 契约，顺序执行更安全；当前用户也
  未授权额外 delegation。若后续明确授权，仅可把纯 domain test 或只读审查作为有界子任务。

#### Issue List

- 无。

### Execute - Implementation

- Completed Task IDs: `DOC01, TREE01, UI01, PREF01, QA01`。
- `TREE01`: 新增独立 `resource-tree.ts`，以 owner/full-path key 构建 arbitrary-depth trie/index，支持纯
  namespace、leaf、Resource/parent hybrid、numeric ordering、search ancestors、默认首层展开与扁平可见行；
  删除旧首段 `groupResources` presentation model。
- `UI01`: Explorer 的 Resource 区改为单选 WAI-ARIA tree，保留类型图标、Inspector selection、drag handle
  与 add action；Node/Resource heading 改为原生 disclosure button，单折叠时 sibling 满高且 separator 移除。
- `PREF01`: `UIPreferences.version=1` 增加 optional `expanded_resource_paths` 与
  `collapsed_explorer_pane`，保持旧文档兼容、Profile 隔离、长度/union 上限与显式损坏回退；折叠不覆盖 split ratio。
- `QA01`: production dist 已从最终源码生成，Wails Windows/amd64 production executable 已构建；未修改品牌资产、
  Go protocol/View schema、remote 或原主检出用户改动。

#### Verification Evidence

- Frontend: `npm test`，9 files / 53 tests passed；包含 10,000 Resource、hybrid path、search restore、ARIA
  keyboard、collapse/split、preference compatibility 与 App localStorage integration。
- Frontend production: `npm run build` passed (`tsc --noEmit` + Vite production build)。
- Repository: `$env:GOWORK='off'; go test ./... -count=1` passed。
- Desktop production: Wails CLI v2.11.0 `wails build -clean -platform windows/amd64` passed；输出为
  `apps/desktop/build/bin/myflowhub-desktop.exe`（ignored，不纳入提交）。
- Connected GUI smoke: Windows packaged app 连接 `127.0.0.1:7331` 成功；验证两个 pane 互斥折叠/满高、固定
  Profile footer、独立 Resource 滚动、`system/config → update` hybrid 展开、Resource drag 到 Workspace、浅/深色
  与重启持久化。临时拖入的未保存 widget 已通过重启丢弃，用户原有 View 保持 4 widgets；最终 production build
  已重新启动供本地查看。
- Hygiene: `git diff --check` passed；Wails 产生的 `go.mod` stat-only noise 已核对 hash 与 HEAD 相同且未保留；
  main checkout 未被暂存、覆盖、还原或归档。

#### Execution Issues

- 无实现阻塞。Computer Use 的 WebView accessibility element click 对滚动后元素索引不稳定，GUI smoke 改用每步
  刷新的 screenshot-relative coordinate；该工具限制未影响产品结果或自动化门禁。

### Continue - Heavy Validation

- Terminal status: `Passed`；1 次 validation-only cycle 收敛，无需返回 `$m-execute`。
- Task IDs: `DOC01, TREE01, UI01, PREF01, QA01` 全部保持完成；未产生计划外实现修复。
- Automated: `npm test` 9 files / 53 tests passed；10,000 Resource fixture 578 ms（门槛 750 ms）；
  `$env:GOWORK='off'; go test ./... -count=1` passed；`npm run build` passed；`git diff --check` passed。
- Packaged UI: 对既有最终 Wails production executable 实际操作；验证 Node 收起后 Resource 满高、Resource 收起后
  Node 满高、separator 移除、`system/config` hybrid 展开到 `system/config/update`、Tree `ArrowLeft` 逐级收起、固定
  Profile/连接 footer 与原 4-widget View 保持不变。
- Evidence: `artifacts/m-test/desktop-explorer-resource-tree/README.md` 与同目录三张 GUI 截图。
- Review: 需求覆盖、架构、性能、可用性、可读性、扩展性、稳定性/安全、权限/数据暴露、测试覆盖、联调和
  子 Agent 治理均通过；本轮未使用 sub-agent，未发生写集冲突或审计缺口。
- Residual risk: 真正服务端分页/lazy loading/DOM windowing 仍是已明确延期的 `LAZY01`；品牌与发布仍不在本任务范围。
- Rollback: 回退本分支实现和 generated dist，并保留原 Profile/View 数据；原主检出无关 dirty change 未触碰。
- Decision: ready for `$m-archive`；本阶段未创建 `docs/change`、未提交、未合并、未清理、未推送。

### Archive - Documentation Candidate

- Docs root: `docs/` in the canonical monorepo；local-only，无 remote/push/publication。
- Change entry: `docs/change/2026-08-30_desktop-explorer-collapsible-resource-tree.md`。
- Plan archive: `docs/plan/plan_archive_2026-08-30_desktop-explorer-collapsible-resource-tree.md`。
- Evidence index: `artifacts/m-test/desktop-explorer-resource-tree/README.md`。
- Intake impact: updated；Feature impact: updated；Requirements impact: updated；Specs impact: updated。
- Decision impact: none；Lessons impact: none；既有 Windows/Wails/frontend lessons 已覆盖可复用线索。
- Index impact: intake/spec indexes 已在执行阶段更新；change/plan indexes 已在归档阶段更新。
- Sub-agent trace: none；host policy/user authorization 不允许主动委派，且共享焦点/preference 写集不宜拆分。
- Control-plane closeout: pending local merge、unrelated-dirt restoration verification、worktree/branch cleanup；任何恢复冲突
  必须保留 worktree 与 feature commits 并停止。

## Approval Gate

- Planned execution Task IDs: `DOC01, TREE01, UI01, PREF01, QA01`
- Technical blockers: none
- Approved execution Task IDs: `DOC01, TREE01, UI01, PREF01, QA01`
- Blocked: no
- Implementation started: yes
- Enter execution under the approved write set only。
- Do not dispatch implementation sub-agents。
