# Plan - MyFlowHub Desktop Codex 风格生产界面

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub`
- Branch: `refactor/desktop-codex-interface`
- Base: `master@b92f0c7239ef331ed06fc3ac258851ff7b20cc1b`
- Project Root: `D:/project/MyFlowHub3`
- Docs Root: `D:/project/MyFlowHub3/worktrees/desktop-codex-interface/docs`
- Code Repos: canonical monorepo `MyFlowHub` only
- Worktree: `D:/project/MyFlowHub3/worktrees/desktop-codex-interface`
- Current Stage: `5 archive in progress`
- Prototype source: `D:/project/MyFlowHub3/repo/MyFlowHub/design-demos/myflowhub-desktop-codex-inspired.html`
- Compatibility: 保留现有 Desktop API、Profile/CredentialStore、Resource/Renderer/View contract；允许 UI 组件结构 clean break

## Stage Records

### Initialization

- `guide.md`: 已读；commit 使用中文、UI 验证不能替代单测与 production build、docs 由 `docs/README.md` 进入、worktree 必须位于 workspace sibling、canonical Go 使用 `GOWORK=off`。
- project/docs/code repo confirmation: `project_root=D:/project/MyFlowHub3`；code 与 governed docs 位于单一 canonical monorepo。
- base/worktree confirmation: dedicated branch/worktree 已创建；主检出 `guide.md`、`论文/**` 和未跟踪 `design-demos/` 不进入本 worktree dirt。
- remote: none；push、publish、release 未授权。

### Discuss - Discovery And Requirements Shaping

#### Goal

把已确认的 Codex-inspired 高保真原型转换为 MyFlowHub Desktop 的生产 shell，同时保留真实 Profile、Credential、Topology、Catalog、Renderer 和 View 模型。

#### Scope

- 登录与持久化、多 Profile；
- 浅色默认、完整深色主题与 per-Profile 非秘密 UI preference；
- 项目图标保持现状；设计、替换、生成、定稿与归档由另一任务负责；
- 平面三栏 shell、Resource/View sidebar、内容 Tabs、右侧 Inspector；
- 任意深度、搜索、展开、聚焦、breadcrumb、返回与 WAI-ARIA keyboard tree；
- Node/Resource preview、Resource drag/add、Workspace/View save；
- 左下 Profile/Connection Settings 入口与 Connection/Profile/Appearance 完整设置。

#### Assumptions

- 现有 topology/catalog API 是本轮唯一生产数据源；不复制 prototype demo data。
- 服务端 lazy-load/pagination 不在当前 API 中；首版用扁平 visible rows、O(n) 索引、`content-visibility`，协议级按需加载另行规划。
- 主题、树展开和聚焦是非秘密 UI preference，可在 WebView local storage 中按 Profile 版本化保存。

#### Open Questions

- Blocking: none。
- Deferred: 服务端 topology/catalog pagination、DOM windowing 阈值自动切换、打开 Tab 会话跨重启恢复。

#### Options Considered

- CSS-only：拒绝，不能解决信息架构和 tree interaction。
- 直接移植 prototype HTML/JS：拒绝，固定 demo data 与命令式 DOM 不可生产化。
- 保留 domain/API、重构 React shell 与 Explorer：接受。
- 新增 tree/UI framework：拒绝，现有 primitives 足够且会增加依赖面。

#### Recommended Direction

React shell 使用平面连续分栏；Explorer 以纯函数构建/过滤/扁平化并集中管理 expansion/focus/active row；Settings 进入主区 Tab；Inspector 与 Workspace 分离；Login/Settings 复用 Profile 表单语义；CSS token 落实矿物蓝、石墨灰、浅色默认和深色重配色。

#### Discussion Brief

- [Desktop Codex 风格界面生产化](docs/intake/2026-08-29_desktop-codex-interface-production.md)

#### Worktree / Branch / Docs Root Status

- branch/worktree/docs root ready；图标工作已按用户要求移出本 workflow；Blocking: no。

### Plan - Requirements And Architecture

#### Discussion Summary

当前生产功能比 prototype 数据模型更完整，但 shell、Settings、Inspector、tree state 与确认视觉不一致。本计划只改变 Desktop UI/interaction 与相关 stable docs，不改变 Resource platform contract，也不触碰图标资产。

#### Accepted / Rejected Requirements

- Accepted: 用户列出的全部 Desktop UI 范围、真实 UI 验证、独立 worktree、full m-autoflow closeout。
- Excluded: ICON01 project brand icon；由另一任务独立处理，本 workflow 不等待也不归档其结果。
- Accepted with boundary: 大树用扁平 visible rows + `content-visibility`；服务端 lazy-load/pagination deferred。
- Rejected: prototype hard-coded data、固定递归层数、第二套 Resource ownership、顶栏重复 Connection/Profile、未授权 publish/push。

#### Requirements Analysis

##### Goal

交付可长期使用、可访问、可测试、与现有生产模型一致的 MyFlowHub Desktop 新界面。

##### Scope

- `apps/desktop/frontend/src/**` UI/domain derivation/tests/style。
- `docs/intake`、`docs/features/desktop.md`、`docs/requirements/desktop-resource-workspace.md`、`docs/specs/desktop-resource-workspace-v2.md`、plan/change/index。
- 必要时只同步 generated Wails bindings；本计划不预期 Go API 变更。

##### Use Cases

1. 首次创建 Profile 并登录；再次启动恢复 active Profile/auto-connect。
2. 从已保存 Profile 切换、编辑、删除或新建 Profile。
3. 在 Resource sidebar 展开任意深度 Node，搜索命中深层后代，聚焦子树并沿 breadcrumb 返回。
4. 仅用键盘在 Tree 中移动、展开、折叠、选择并添加 Resource。
5. 选择 Node/Resource 在 Inspector 预览；关闭 Inspector 不改变 View。
6. 拖放或按钮添加 Resource，编辑布局/名称并保存 View。
7. 通过左下 Profile/Connection 入口打开 Settings Tab，管理连接、Profile 和 Appearance。
8. 在浅/深色间切换并按 Profile 恢复；损坏 preference 回退 light。

##### Functional Requirements

- 以更新后的 [Desktop feature](docs/features/desktop.md) 与 [Desktop requirement](docs/requirements/desktop-resource-workspace.md) 为准。

##### Non-functional Requirements

- WAI-ARIA tree roving focus；可见 focus；drag 的按钮/键盘等价入口。
- 深度无递归组件状态；构建/filter/flatten 保持 O(nodes + resources)。
- 2,000 Nodes + 10,000 Resources build/filter 保持 `<750ms`；超过 50 visible rows 启用离屏绘制优化。
- 无新增依赖、无 secret/payload localStorage、无 plan-external Go/协议变化。
- 浅色/深色在 1440×900 production UI 通过视觉走查。

##### Inputs / Outputs

- Inputs: Settings/Profile、ConnectionStatus、Topology、ResourceDescriptor[]、ViewDocument、pointer/keyboard actions、versioned UI preferences。
- Outputs: visible Explorer rows、Workspace/Settings active content、Inspector selection、saved View、saved non-secret theme/tree preference、explicit errors/status。

##### Edge Cases

- topology 空、孤儿 Node、父引用不存在、极深树、搜索后 active row 不可见。
- focus root/leaf、focused Node 消失、expanded ID 不再存在、local storage 损坏/不可写。
- saveProfile 后连接被关闭；connect/disconnect/switch 失败；active Profile 被删除。
- unsaved View 时打开/关闭 Tab、切换 Profile 或删除 View。
- Resource offline/unknown/detached；Inspector close；drag drop target outside Workspace。

##### Acceptance Criteria

- 全部 UI 功能来自 `DesktopAPI` mock 或真实 Wails state，无 prototype demo object。
- 至少 6 层 fixture 通过 search/focus/breadcrumb/back 与 ARIA keyboard tests。
- Sidebar footer 打开全宽 Settings Tab；Connection/Profile/Appearance 可操作；顶栏无 Connection/Profile 重复项。
- Node/Resource Inspector、drag/add/save View 保留并回归通过。
- Vitest、TypeScript/Vite production build、Go Desktop tests、Wails production build、`git diff --check` 通过。
- 实际 application/page 被打开并操作，保留 1440×900 light/dark/login/settings/deep-tree/workspace screenshot evidence。

##### Risks

- App shell 与 View dirty guard 耦合；用明确 content mode/open tab state 保留 guard。
- Tree keyboard/focus/search 组合复杂；将索引与 transition 设计为纯函数并增加组件 tests。
- Settings profile edits会重开 client；UI 必须在保存后刷新 settings/status 并诚实显示 disconnected。
- Renderer effects可能因 Inspector切换重订阅；依靠既有 cleanup并在 tests/UI smoke观察。

#### Architecture Design

##### Overall Solution

```text
DesktopAPI state
  ├─ Settings / Connection ──> Login + Settings
  ├─ Topology + Catalogs ────> Explorer index -> filtered tree -> flat visible rows
  └─ Views + Selection ──────> Content tabs -> Workspace + Inspector

Profile-scoped UI preferences
  └─ theme + expanded node IDs + focused node ID (versioned, non-secret)
```

##### Alternatives Considered

- 保留递归 `NodeBranch useState`：拒绝，深树焦点与 keyboard 无法集中推导。
- 引入 virtualizer：当前先拒绝；依赖成本高且 existing performance contract 已允许 `content-visibility`。若真实 DOM/paint 超预算，后续 Task 以测量触发。
- 把 UI preference 加进 Go settings v3：本轮拒绝；会扩大迁移/API/credential边界，纯 UI preference 不需要。

##### Module Responsibilities

- `store.ts`: authority tree indexes、filter、flatten、parent/child navigation、View helpers。
- `preferences.ts`: versioned per-Profile non-secret preference parsing/persistence、light default。
- `Explorer.tsx`: ARIA tree、roving focus、search、expansion、focus subtree、drag/add。
- `App.tsx`: production state orchestration、shell/content tabs、profile/connection actions、dirty guards。
- `LoginScreen.tsx` / `ProfileEditor.tsx`: minimal login and complete production Profile inputs。
- `Settings.tsx`: Connection/Profile/Appearance panes。
- `Inspector.tsx`: Node/Resource preview and focus/add actions。
- `Workspace.tsx`: View toolbar/grid/widgets only；不再嵌入 transient preview。
- `style.css`: prototype-derived design tokens/layout/theme/responsive/a11y states。
- tests: pure tree/prefs performance and component integration/keyboard paths。

##### Data / Call Flow

1. App loads settings；active Profile key loads UI preference and applies theme before/after shell render。
2. Connected Profile loads topology and catalogs；store builds iterative tree/index once。
3. Explorer applies query/focus/expanded Set to derive visible flat rows；row key drives roving focus。
4. Row selection updates App selection -> Inspector；Resource add updates active View -> Workspace dirty。
5. Sidebar footer opens Settings content tab；settings actions call DesktopAPI and refresh settings/status/platform。
6. Theme/expansion/focus writes only versioned UI preference for current Profile。

##### Interface Drafts

```ts
type ExplorerIndex = {
  roots: ExplorerNode[]
  nodesByID: Map<string, ExplorerNode>
  parentByID: Map<string, string | undefined>
}

type ExplorerRow = {
  key: string
  kind: 'node' | 'resource'
  depth: number
  parentKey?: string
  posInSet: number
  setSize: number
  node?: ExplorerNode
  resource?: ResourceDescriptor
}

type UIPreferencesV1 = {
  version: 1
  theme: 'light' | 'dark'
  expandedNodeIDs: string[]
  focusedNodeID?: string
}
```

##### Error Handling and Safety

- preference parse/storage errors只回退可见默认值，不改变 Go settings；storage payload 不含 permit/key/resource body。
- DesktopAPI errors进入现有 role=alert；连接/保存/Profile destructive action显式确认。
- 无效 tree parent 作为 root显示，不产生循环递归；检测 parent cycle 后显式截断派生边并保持 Node 可发现。
- saveProfile 后刷新 status；不静默宣称 connected。

##### Performance and Testing Strategy

- Map/Set 索引、迭代 traversal、`useDeferredValue(query)`、memoized visible rows、stable callbacks。
- Row CSS 使用 `content-visibility:auto` 与 intrinsic size；Tree 规模基准继续 `<750ms`。
- Vitest/Testing Library 覆盖 Login、Settings、deep tree、keyboard、focus、Inspector、View dirty/save。
- Heavy test 启动真实 Wails/Vite surface，操作关键路径并截图；production Wails build 作为最终包装门禁。

##### Extensibility Design Points

- Resource icon map以 type ID 注册并保留 unknown icon。
- Tree derivation不假设最大 depth；后续 virtualizer/lazy-load可消费同一 flat rows/index。
- Settings pane枚举与 content tab state独立于 backend schema；未来可增加非秘密 preference。

### Stage 3.1 - Planning

#### Project Goal and Current State

- Current: production model完整，但 shell、Settings、Inspector、深树 interaction和视觉未符合已确认 prototype。
- Target: production API不变的 Codex-inspired Desktop UI，包含可访问深树和完整 Settings。

#### Docs Governance Routing Decision

- Docs root: `D:/project/MyFlowHub3/worktrees/desktop-codex-interface/docs`。
- Original request -> new intake；current visible truth -> Desktop feature；durable acceptance -> Desktop requirement；technical UI contract -> Desktop spec；workflow result -> plan/change；ADR unchanged。

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake: [Desktop Codex 风格界面生产化](docs/intake/2026-08-29_desktop-codex-interface-production.md)
- Features: [Desktop](docs/features/desktop.md)
- Requirements: [Desktop resource workspace](docs/requirements/desktop-resource-workspace.md), [Extensible resource platform](docs/requirements/extensible-resource-platform.md)
- Specs: [Desktop workspace v2](docs/specs/desktop-resource-workspace-v2.md), [Resource platform v2](docs/specs/resource-platform-v2.md)
- Decisions: [Extensible Resource type system and Desktop workspace](docs/decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)
- Lessons: [frontend worktree WailsJS](docs/lessons/frontend-worktree-wailsjs-missing.md), [frontend preflight](docs/lessons/frontend-and-powershell-preflight.md), [empty node_modules](docs/lessons/frontend-build-empty-node-modules.md)

#### Stable Docs Impact

- Intake impact: add（completed）。
- Feature impact: clarify（completed in planning）。
- Requirements impact: clarify（completed in planning）。
- Specs impact: clarify（completed in planning）。
- Decision impact: none。
- Lessons impact: none known；test/archive reassess。

#### Executable Task List

- [x] DOC01 — stable docs、active plan 与索引治理。
- [x] UI01 — theme/preferences、minimal login、flat shell 与完整 Settings。
- [x] UI02 — arbitrary-depth Explorer、flat visible rows、focus/breadcrumb 与 WAI-ARIA keyboard。
- [x] UI03 — content tabs、Inspector 分离、Workspace/View drag/add/save 整合。
- [x] VAL01 — focused tests、performance、frontend/Go/Wails build、真实 UI operation 和 screenshot evidence。
- [ ] TREE02 — server-side topology/catalog lazy loading or pagination（deferred）。
- [ ] SYNC01 — open content tab session persistence across restarts（deferred）。
- [ ] PUB01 — remote push/release/publish（separate authorization）。
- [ ] ICON01 — project brand icon design/generation/integration（out of scope；owned by separate task）。

#### Execution Scope After Approval

##### Will Execute

- DOC01, UI01, UI02, UI03, VAL01（仅在整个 plan 解除阻塞并获批后）。

##### Will Not Execute Now

- TREE02：需要新的服务端/API pagination、缓存与一致性设计；本轮以 flat rows + content visibility 满足当前规模 contract。
- SYNC01：用户要求保存 View，不要求跨重启恢复“哪些 Tab 当前打开”；避免扩大持久化 schema。
- PUB01：仓库无 remote，且用户明确禁止未授权 push/publish。
- ICON01：用户明确交给另一任务；本 workflow 不设计、替换、生成、定稿或归档项目图标。

#### Task Details

##### DOC01 - Stable docs and workflow control

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/desktop-codex-interface`
- Plan Path: `D:/project/MyFlowHub3/worktrees/desktop-codex-interface/plan.md`
- Goal: 固化 intake、当前 Desktop 行为、长期验收、技术 contract 与 Task IDs。
- Files / Modules: `plan.md`, `docs/intake/**`, `docs/features/desktop.md`, `docs/requirements/desktop-resource-workspace.md`, `docs/specs/desktop-resource-workspace-v2.md`, indexes/archive。
- Write Set: 上述 Markdown。
- Acceptance: docs 分类和索引正确，无 competing truth，plan可脱离聊天执行。
- Test Points: link/path review；`git diff --check`。
- Rollback: revert DOC01 docs commit，不影响 runtime。

##### UI01 - Shell, theme, login and settings

- Owner: main agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 实现浅色默认/深色、minimal login、flat shell、sidebar footer与完整 Settings。
- Files / Modules: `frontend/src/App.tsx`, `components/LoginScreen.tsx`, new `components/ProfileEditor.tsx`, new `components/Settings.tsx`, new `preferences.ts`, `types.ts`, `style.css`, focused tests。
- Write Set: 仅上述 frontend files/tests。
- Acceptance: 顶栏无 Profile/Connection；footer打开 Settings；Connection/Profile/Appearance真实可用；theme按 Profile保存；Login保留所有生产必需字段/permit语义。
- Test Points: App/Settings/Login Vitest；TypeScript build。
- Rollback: revert UI01；Go settings/credentials不迁移。

##### UI02 - Data-driven accessible explorer

- Owner: main agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 用集中式 flat rows替换递归 Node state，实现 arbitrary depth、search、expansion、focus/breadcrumb/back与ARIA keyboard。
- Files / Modules: `frontend/src/store.ts`, `components/Explorer.tsx`, `preferences.ts`, `store.test.ts`, `App.test.tsx`, `style.css`。
- Write Set: 上述 Explorer/domain/tests/styles。
- Acceptance: 6+层 fixture和10k resources门禁；ARIA row metadata与方向键/Home/End/Enter/Space；deep focus/back；unknown type icon。
- Test Points: pure store tests、Testing Library keyboard tests、performance threshold。
- Rollback: revert UI02；Topology/Catalog contract不变。

##### UI03 - Content tabs, inspector and workspace integration

- Owner: main agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: Settings/View content tabs、右侧 Inspector、Workspace-only preview separation与原有drag/add/save整合。
- Files / Modules: `frontend/src/App.tsx`, new `components/Inspector.tsx`, `components/Workspace.tsx`, `components/Renderer.tsx` if presentation hooks required, `App.test.tsx`, `style.css`。
- Write Set: 上述 frontend files/tests。
- Acceptance: Node/Resource preview位于 Inspector；Resource按钮/drag添加并保存；Settings铺满主区；dirty guard与View delete确认不回归。
- Test Points: App integration tests、renderer cleanup review、production build。
- Rollback: revert UI03；View documents保持兼容。

##### VAL01 - Validation and UI evidence

- Owner: main agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 证明功能、性能、可访问性、深色与Wails包装路径满足 acceptance。
- Files / Modules: tests、`.tmp/m-test-desktop-codex-interface/evidence/**`（临时/不提交）、plan status、archive docs。
- Write Set: focused tests/evidence and workflow docs only；不改 plan-external product behavior。
- Acceptance: Vitest/TS/Vite、Go Desktop、Wails build、UI operation/screenshot、review checklist全通过；失败映射回 UI01-UI03并经 `$m-continue` 修复。
- Test Points: `npm test`, `npm run build`, `go test ./apps/desktop/...`, `wails build -clean -platform windows/amd64`, actual UI smoke, `git diff --check`。
- Rollback: evidence可删除；产品修复按所属 UI Task回退。

#### Dependencies

- Plan confirmation -> DOC01 -> UI01 -> UI02/UI03 -> VAL01。
- UI02 与 UI03 同时修改 App/style/tests，写集高度耦合；串行集成。

#### Risks and Notes

- 不使用 sub-agent：用户未显式请求 sub-agent；host policy禁止主动委派，且 UI01-UI03共享 App/style/tests写集。
- 不新增 npm dependency；复用 React、Radix、dnd-kit、lucide。
- prototype 是视觉/interaction输入，不进入 production data model。

#### Parallelism Assessment

- 两个以上 Task存在，但 App/style/tests 写集重叠且 shell/tree/inspector状态强耦合；即使允许委派也不宜并行编辑。
- Validation commands可在 implementation后并行运行，但阶段内由主 agent统一复核。

#### Approval Gate

- Plan status: Approved by the user's explicit instruction to continue and complete the non-icon Desktop workflow after the direct gate was displayed。
- Will execute after approval: DOC01, UI01, UI02, UI03, VAL01。
- Will not execute now: TREE02, SYNC01, PUB01, ICON01。
- Blocked: no。
- Execution/test complete；enter archive closeout。

### Execute - Implementation Result

- UI01: 已实现浅色默认与完整深色 token、per-Profile versioned UI preferences、minimal login、底部 Profile/connection 入口和 Connection/Profile/Appearance 完整 Settings Tab。
- UI02: 已用 iterative index + flat visible rows 替换递归组件状态；支持任意深度、精确搜索、展开持久化、focused subtree、breadcrumb/back 和 WAI-ARIA roving focus keyboard。
- UI03: 已将 Inspector 从 Workspace 分离；Node/Resource preview、按钮/drag add、View dirty/save/delete 与 content tabs 保持真实 DesktopAPI 数据流。
- 没有修改 Go API、Profile schema、CredentialStore、Resource/View contract 或任何项目图标资产。

### Test / Continue - Verification Result

- `npm test`: 4 files / 19 tests passed；覆盖 3,000 深度、2,000 Nodes + 10,000 Resources 性能、parent cycle、精确搜索、ARIA keyboard、Profile UI preferences、Login/Settings/Inspector/View integration。
- `npm run build`: TypeScript `--noEmit` 与 Vite production build passed。
- `GOWORK=off go test ./apps/desktop/...`: Desktop 与 Desktop MCP packages passed。
- `GOWORK=off wails build -clean -platform windows/amd64`: Wails 2.11 Windows production package passed。
- `git diff --check`: passed。
- 1440×900 实际页面操作覆盖 light workspace、deep tree、resource/node inspector、focused subtree breadcrumb、full settings、dark theme 和 login；临时 visual bridge 已删除。
- `$m-continue` 修复：空 workspace 居中、深 breadcrumb 可见、同节点资源搜索精确性、刷新后的 stale Inspector selection。

### Archive - Closeout

- Stable feature/requirement/spec 已同步；不新增图标 ADR，不归档图标决策。
- Plan archive: `docs/plan/plan_archive_2026-08-29_desktop-codex-interface-production.md`。
- Change archive: `docs/change/2026-08-29_desktop-codex-interface-production.md`。
- Push/release/publish: not performed（no remote；not authorized）。
