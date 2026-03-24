# Plan - MyFlowHub-Win Showcase Variable Picker Fix

## Workflow Information

- Repo: `MyFlowHub-Win`
- Branch: `fix/showcase-editor-var-picker`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker-win`
- Control-plane worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker`
- Current Stage: `4 归档变更（已完成，等待 workflow 结束确认）`

## Stage Records

### Initialization

- `guide.md`:
  - 已阅读，遵守以下约束：
    - commit 信息使用中文，前缀可英文
    - 优先可尝试 `chrome-devtools` 做界面验证
    - worktree 必须在 `D:\project\MyFlowHub3\worktrees\`
- base/worktree confirmation:
  - 控制面工作区：`D:\project\MyFlowHub3`
    - 用途：workflow 编排、文档治理、worktree 管理
    - 不在此处做业务实现改动
  - 实现仓库：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 活跃执行 worktree：`D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker-win`
  - 参与模块：
    - `frontend/src/pages/Showcase.vue`
    - 参考：`frontend/src/stores/varpool.ts`
    - 参考：`frontend/src/stores/showcase.ts`
  - 参考规范：
    - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`

### Stage 1 - Requirements Analysis

#### Goal

- 修复 Showcase Editor 中新增 / 编辑 `var` widget 的两个可见问题：
  - `Variable Name` 的选择按钮与输入框需要处于同一行
  - 变量选择弹窗应能稳定显示候选项，而不是常态空白

#### Scope

##### Must

- 修正 `Variable Name` 输入区布局，使输入框与选择按钮并排显示。
- 修正变量快捷选择弹窗的数据来源，使其在 Showcase Editor 场景下能显示：
  - 当前编辑上下文里已订阅 / 已存在值快照的变量
  - 当前登录节点的 mine 变量
- 保持点击候选后自动回填 `Owner NodeID` 与 `Variable Name`。
- 保持手工输入路径不受影响。

##### Optional

- 收敛空状态提示，让“无候选”与“未登录 / 未加载”的表现更清晰。

##### Out of Scope

- 不改 VarStore / VarPool 协议。
- 不重做 VarPool 页面、Node Vars 弹窗或 Showcase 整体编辑器结构。
- 不新增新的后端接口或持久化字段。

#### Use Cases

- 用户在 Showcase Editor 中新增 `var` widget，希望直接在输入框右侧点选变量，而不是手输名称。
- 用户当前并未在 `#/varpool` 页面维护 watch 列表，但仍希望在 Showcase Editor 中看到当前 screen 正在使用的变量或自己的变量。
- 用户选择某个候选后，希望 `Owner NodeID` 与 `Variable Name` 一次性正确回填。

#### Functional Requirements

1. `Variable Name` 输入框和选择按钮必须位于同一视觉行。
2. 打开变量选择弹窗后，必须展示当前 Showcase 编辑上下文中的可选变量。
3. 当前 screen 已有订阅 / 值快照的变量必须能进入 `Subscribed` 候选。
4. 当前登录节点的 mine 变量必须能进入 `Mine` 候选，并允许手动刷新。
5. 同一 `(owner, name)` 候选若同时属于 `Subscribed` 和 `Mine`，展示时应去重并保留双重标记。
6. 点击候选后必须继续回填 `Owner NodeID` 与 `Variable Name`。
7. 变量名过滤能力必须继续支持按 `name` / `owner` 搜索。

#### Non-functional Requirements

- 最小改动：
  - 优先局部修改 `Showcase.vue`，避免扩散到无关模块。
- 一致性：
  - 候选来源应与 Showcase Editor 当前上下文一致，而不是依赖用户是否提前打开过其他页面。
- 稳定性：
  - 登录态缺失、mine 拉取失败、无候选时必须显式降级，不得抛未处理异常。
- 性能：
  - 候选合并在前端内存完成。
  - mine 列表请求只在打开弹窗或显式刷新时触发，不引入轮询。

#### Inputs / Outputs

- Inputs:
  - Showcase 当前草稿 / 当前 screen 的变量值快照
  - Session 登录态中的 `selfNodeId` / `hubId`
  - 用户在弹窗中的搜索关键词
- Outputs:
  - 同行布局的变量名输入区
  - 可筛选、可点击回填的候选变量列表

#### Edge Cases

- 当前未登录，无法加载 mine 变量。
- 当前 screen 没有任何变量值快照。
- mine 列表为空。
- 同一变量同时出现在当前 screen 快照和 mine 中。
- 当前字段已有 `ownerId` / `varName`，但用户选择了不同候选。

#### Acceptance Criteria

1. `Variable Name` 输入框与选择按钮在同一行显示。
2. 打开弹窗后，在存在当前 screen 变量或 mine 变量时，列表不再常态空白。
3. 搜索、分组显示和点击回填继续可用。
4. 没有候选或未登录时，界面给出明确空态 / 错误提示，不出现静默异常。
5. `Showcase` 其余 widget 编辑、保存与手工输入行为不回退。

#### Risks

- 若继续把候选完全绑定到 `varpool.state.keys`，会把 Showcase Editor 的可用性错误地依赖到 VarPool watch list。
- 若 mine 数据刷新和当前 screen 快照合并去重不当，可能出现重复条目或分组错位。
- 若布局改动过大，可能影响现有弹窗在窄宽度下的换行策略。

#### Issue List

- 无

### Stage 2 - Architecture Design

#### Overall Solution

- 保持修复范围在前端，主改 `frontend/src/pages/Showcase.vue`。
- 将快捷选择候选拆成两类来源，再在页面内合并：
  - `Subscribed`：
    - 基于 Showcase 当前上下文的变量快照构建，而不是依赖 `VarPool` watch list
  - `Mine`：
    - 打开弹窗或点击刷新时，显式调用 `varpool.listOwnerNames(selfNodeId)` 拉取
- 使用 `(owner:name)` 作为唯一键在前端去重，并保留 `mine` / `subscribed` 双标记。
- 调整变量名表单布局：
  - label 单独保留
  - 输入框与按钮改为同一 `flex` 行

#### Alternatives Considered

- 继续复用 `varpool.state.keys` 作为唯一候选源：
  - 不采用。该状态主要受 VarPool watch list 驱动，和 Showcase Editor 当前使用场景不一致，是本次空弹窗的主要原因。
- 把 `listMine()` 结果继续塞回 `varpool.state.keys` 后再复用原逻辑：
  - 不采用。仍然把 Showcase 候选逻辑绑定到全局 store 副作用，语义不清晰，且无法覆盖当前 screen 的订阅上下文。
- 新增后端接口专门给 Showcase 查询候选：
  - 不采用。当前已有 `showcase` 快照和 `varpool.listOwnerNames()` 能满足需求，新增接口属于过度设计。

#### Module Responsibilities

- `frontend/src/pages/Showcase.vue`
  - 修正候选计算来源
  - 维护 mine 列表的弹窗内状态
  - 合并去重候选
  - 调整 `Variable Name` 输入区布局
- `frontend/src/stores/showcase.ts`
  - 仅作为当前 screen 变量值快照来源，本轮不改接口
- `frontend/src/stores/varpool.ts`
  - 继续提供 `listOwnerNames(ownerId)` 能力，本轮不改契约

#### Data / Call Flow

1. 用户打开 `var` widget 编辑弹窗。
2. `Showcase.vue` 从当前 Showcase 状态提取当前 screen 的变量快照，形成 `Subscribed` 候选。
3. 用户打开变量选择弹窗：
   - 立即展示本地可得的 `Subscribed` 候选
   - 并发触发 `listOwnerNames(selfNodeId)` 拉取 mine 名单
4. mine 名单返回后，前端按 `(owner:name)` 合并到候选集合。
5. 用户输入关键词时，在合并后的集合上执行前端过滤。
6. 用户点击候选后，回填 `ownerId` 和 `varName`，关闭弹窗。

#### Interface Drafts

- `type VarQuickPickItem = { name: string; owner: number; mine: boolean; subscribed: boolean }`
- 新增页面内状态：
  - `mineVarQuickPickItems`
  - `subscribedVarQuickPickItems` 改为从 Showcase 当前上下文计算
  - `filteredVarQuickPickItems` 改为基于合并结果过滤

#### Error Handling and Safety

- 未登录或 `selfNodeId <= 0`：
  - 不触发 mine 列表请求
  - 保留当前 screen 候选
  - `Refresh Mine` 继续禁用或报出明确提示
- mine 列表请求失败：
  - 保留已可见的本地候选
  - toast 显式提示失败
- 无候选：
  - 各分组显示明确空态文案
- 候选去重：
  - 使用 `(owner:name)` 保证无重复点击项

#### Performance and Testing Strategy

- 只在打开弹窗 / 手动刷新时发起一次 mine 查询。
- 候选去重、过滤在前端内存完成，不新增持久化或订阅。
- 验证策略：
  - 静态：`npm run build`
  - 静态：`git diff --check`
  - 手工 / 页面级：
    - 打开 Add/Edit Var，检查按钮与输入框同行
    - 检查弹窗在已有 Showcase 变量 / mine 变量场景下出现候选
    - 点击候选后检查回填

#### Extensibility Design Points

- 后续若要扩展“当前 owner 的所有变量”或“最近使用”，可继续在页面内增加新的候选来源并复用同一去重链路。
- 候选去重键与分组标记保持独立，可支持未来增加更多来源而不改点击回填逻辑。

#### Issue List

- 无

### Stage 3.1 - Planning

#### Project Goal and Current State

- 当前状态：
  - `Variable Name` 区域为“label + 按钮”一行、输入框单独下一行
  - 快捷选择候选主要取自 `varpool.state.keys`
  - `Subscribed` 语义实际上更接近 VarPool watch/subscription，而不是 Showcase 当前编辑上下文
- 直接后果：
  - 视觉上按钮没有与输入框形成一体操作区
  - 用户若未在 VarPool 页面维护 watch list，弹窗容易全空，即使当前 Showcase screen 已经在使用变量

#### Docs Governance Routing Decision

- 使用 `$docs-governor` 校验计划文档路由和 requirements/specs 影响。
- docs tree 状态：
  - `docs/README.md`、`docs/requirements/README.md`、`docs/specs/README.md` 已存在，结构完整，无需 bootstrap。
- Canonical destination:
  - 稳定需求 / 技术真相：当前无需新增或修改
  - 执行控制：worktree 根 `plan.md`
  - 完成结果：`docs/change/2026-03-23_win-showcase-var-picker-fix.md`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `docs/requirements/showcase-display-widgets.md`（引用，无改动）
- Related specs:
  - `docs/specs/showcase-display-widgets.md`（引用，无改动）
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`（引用，无改动）

#### Executable Task List

- [x] `SHVP-1` 修复 Showcase 变量快捷选择候选来源
- [x] `SHVP-2` 调整 `Variable Name` 输入区为同一行布局
- [x] `SHVP-3` 验证、Code Review 与归档

#### Task Details

##### `SHVP-1` - 修复 Showcase 变量快捷选择候选来源

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker-win\plan.md`
- Goal:
  - 将快捷选择候选改为“Showcase 当前上下文订阅 + 显式加载 mine”，消除对 VarPool watch list 的错误依赖。
- Files / Modules:
  - `frontend/src/pages/Showcase.vue`
- Write Set:
  - `frontend/src/pages/Showcase.vue`
- Acceptance:
  - 弹窗在存在当前 screen 变量或 mine 变量时有候选可见
  - 去重、分组、过滤与点击回填继续正确
- Test Points:
  - 打开 Add/Edit Var，验证 `Subscribed` / `Mine` 分组出现数据
  - 搜索过滤仍按 `name` / `owner` 生效
- Rollback:
  - 回滚 `Showcase.vue` 中快捷选择候选计算与刷新逻辑

##### `SHVP-2` - 调整 `Variable Name` 输入区为同一行布局

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker-win\plan.md`
- Goal:
  - 让变量名输入框与选择按钮处于同一操作行，提升编辑效率与可发现性。
- Files / Modules:
  - `frontend/src/pages/Showcase.vue`
- Write Set:
  - `frontend/src/pages/Showcase.vue`
- Acceptance:
  - 输入框与按钮同行，窄宽度下仍可用
- Test Points:
  - 打开 Add/Edit Var 检查布局与点击行为
- Rollback:
  - 回滚 `Showcase.vue` 中变量名输入区模板

##### `SHVP-3` - 验证、Code Review 与归档

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-showcase-editor-var-picker-win\plan.md`
- Goal:
  - 完成静态验证、阶段 3.3 Review 和 `docs/change` 归档。
- Files / Modules:
  - `frontend/src/pages/Showcase.vue`
  - `docs/change/2026-03-23_win-showcase-var-picker-fix.md`
  - `docs/change/README.md`
- Write Set:
  - `frontend/src/pages/Showcase.vue`
  - `docs/change/2026-03-23_win-showcase-var-picker-fix.md`
  - `docs/change/README.md`
- Acceptance:
  - 关键验证命令执行并记录结果
  - Stage 3.3 检查项完成
  - Stage 4 归档文档完成并更新索引
- Test Points:
  - `npm run build`
  - `git diff --check`
  - 必要时页面级手工验证
- Rollback:
  - 回滚本次代码与文档变更

#### Dependencies

- `showcase` store 当前值快照必须已能反映 Editor 当前 screen 的变量上下文。
- `varpool.listOwnerNames(ownerId)` 必须继续遵守通过 Hub 查询 owner 变量名的现有契约。
- 现有 Wails binding / `wailsjs` 生成物需保持可用，便于前端构建验证。

#### Risks and Notes

- 若当前 Showcase screen 没有任何变量值快照，而 mine 也为空，弹窗仍会显示空态；这是有效空结果，不是 bug。
- 当前仓库若存在与 `wailsjs` 或其它页面无关的基线构建问题，需要在验证阶段明确区分“既有问题”与“本次引入”。

#### Parallelism Assessment

- 评估结果：本轮不派发子Agent。
- 原因：
  - 写集高度集中在 `frontend/src/pages/Showcase.vue`
  - 任务规模小，分拆后会增加同步成本，收益低
  - 当前关键路径是单文件逻辑与模板联动，需要主Agent本地一次性完成

#### Issue List

- 无

阻塞：否
进入 3.2

### Stage 3.3 - Code Review

- 需求覆盖：通过
  - 已修复候选源错误依赖，`Variable Name` 输入区也已改为输入框与按钮同行。
- 架构合理性：通过
  - 维持前端局部修复，不引入新的后端接口或全局 store 契约变更。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - mine 列表请求仅在打开弹窗或显式刷新时触发；候选合并与过滤均在前端内存完成。
- 可读性与一致性：通过
  - 通过 `mergeVarQuickPickItem` 收敛去重逻辑，并用单独的局部状态承载 mine 候选。
- 可扩展性与配置化：通过
  - 候选来源已拆分为可组合链路，后续可继续追加“当前 owner”或“最近使用”等来源。
- 稳定性与安全：通过
  - 未登录、mine 拉取失败、空候选等场景都有显式降级路径。
- 测试覆盖情况：部分通过
  - `git diff --check` 通过
  - `npm ci` 通过
  - `Showcase.vue` 定向 SFC 解析通过
  - `npm run build` 失败，原因为仓库基线缺失 `frontend/wailsjs` 生成物，失败点在 `Home.vue`，非本次改动引入
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent，由主Agent 单独完成并复核。

### Stage 4 - Change Archive

- 使用 `$docs-governor` 完成归档路由与 impact 检查。
- Archive Path:
  - `docs/change/2026-03-23_win-showcase-var-picker-fix.md`
- Requirements impact:
  - `none`
- Specs impact:
  - `none`
- Related requirements:
  - `docs/requirements/showcase-display-widgets.md`
- Related specs:
  - `docs/specs/showcase-display-widgets.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- Lessons needed:
  - `none`
- Index updates:
  - `docs/change/README.md`
