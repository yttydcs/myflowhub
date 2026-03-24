# Plan - MyFlowHub-Win Showcase Watched Variable Picker

## Workflow Information

- Repo: `MyFlowHub-Win`
- Branch: `fix/showcase-var-picker-watch-all`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all-win`
- Control-plane worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all`
- Current Stage: `4 归档变更（已完成，等待 workflow 结束确认）`

## Stage Records

### Initialization

- `guide.md`:
  - 已阅读，遵守以下约束：
    - commit 信息使用中文，前缀可英文
    - worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`
    - 可尝试使用 `chrome-devtools` 做界面验证
- skill availability:
  - `$rigorous-execution` 与 `$docs-governor` skill 文件在当前 session 中不可读
  - 本轮按同等 staged 规则手动执行初始化、阶段分析、计划、review 与归档
- base/worktree confirmation:
  - 控制面工作区：`D:\project\MyFlowHub3`
    - 仅用于 workflow 编排、归档和 worktree 管理
  - 实现仓库：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 活跃执行 worktree：`D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all-win`
  - 参与模块：
    - `frontend/src/pages/Showcase.vue`
    - 参考：`frontend/src/stores/varpool.ts`
  - 参考规范：
    - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`

### Stage 1 - Requirements Analysis

#### Goal

- 将 Showcase Editor 的变量快捷选择从“只展示已订阅 / 当前上下文变量”调整为“展示所有被 watch 的变量”，同时继续保留 mine 变量快捷选择。

#### Scope

##### Must

- 变量选择器必须展示所有 watch list 中、且具有有效 `owner` 的变量。
- 变量选择器不得再要求变量必须处于 subscribed 状态才显示。
- mine 变量快捷加载与回填行为必须保持可用。
- 点击候选后继续回填 `Owner NodeID` 与 `Variable Name`。

##### Optional

- 将弹窗分组标题、提示文案和空态从 `Subscribed` 收敛为 `Watched`，以匹配实际语义。

##### Out of Scope

- 不改 VarPool / VarStore 协议。
- 不重做 Showcase 编辑器布局结构。
- 不新增新的后端接口、store 持久化字段或全局缓存机制。

#### Use Cases

- 用户已经在 `VarPool` 页面 watch 了一批变量，即使其中部分尚未订阅，也希望在 Showcase Editor 里直接选中它们。
- 用户希望通过 `Watched` 分组快速复用现有 watch 列表，而不是只能看到当前 screen 或已订阅的少量变量。
- 用户仍希望在需要时额外加载 mine 变量，并和 watch 列表一起去重展示。

#### Functional Requirements

1. 快捷选择器必须读取 `VarPool` 的 watch list 作为主候选来源。
2. watch list 中合法的 `(owner, name)` 候选必须全部可见，不得因 `subscribed=false` 被过滤。
3. mine 变量继续通过显式刷新加载，并与 watched 候选按 `(owner:name)` 去重。
4. 若同一变量同时属于 watched 和 mine，展示时必须保留双重标记。
5. 搜索过滤必须继续支持按 `name` / `owner` 匹配。
6. 点击候选回填行为不得回退。

#### Non-functional Requirements

- 最小改动：
  - 优先只修改 `frontend/src/pages/Showcase.vue`
- 一致性：
  - UI 文案应反映真实语义，不再把 watched 误称为 subscribed
- 稳定性：
  - watch list 为空、mine 拉取失败、未登录等情况必须有明确降级路径
- 性能：
  - 继续在前端内存中合并、过滤与去重
  - 不新增轮询或额外订阅

#### Inputs / Outputs

- Inputs:
  - `varpool.state.keys` 中的 watch list
  - Session 登录态中的 `selfNodeId` / `hubId`
  - 用户搜索关键词
- Outputs:
  - `Watched` + `Mine` 两类候选
  - 点击候选后更新的 `ownerId` / `varName`

#### Edge Cases

- watch list 含旧数据，`owner=0` 无法形成有效候选。
- watch list 为空。
- 未登录，mine 无法拉取。
- 同一变量同时存在于 watched 与 mine。

#### Acceptance Criteria

1. 快捷选择器中的主列表改为展示所有有效 watched 变量。
2. 未订阅但已 watch 的变量也能被选中。
3. `Mine` 列表和回填逻辑继续正常。
4. 分组文案、空态和 tooltip 与 watched 语义一致。
5. 其余 Showcase 编辑行为不回退。

#### Risks

- 若仅修改过滤条件而不改 UI 文案，会继续造成“Subscribed 实际是 Watched”的语义错配。
- watch list 中的旧格式条目若没有 `owner`，仍无法用于快捷选择；需要保持显式过滤，不能猜 owner。

#### Issue List

- 无

### Stage 2 - Architecture Design

#### Overall Solution

- 保持前端局部修复，只改 `frontend/src/pages/Showcase.vue`。
- 快捷选择候选改为两类来源：
  - `Watched`：
    - 直接来自 `varpool.state.keys`
    - 只要求 `name` 非空且 `owner > 0`
    - 不再检查 `subKnown/subscribed`
  - `Mine`：
    - 保持调用 `varpool.listOwnerNames(selfNodeId)` 显式拉取
- 继续用 `(owner:name)` 去重，并把布尔标记从 `subscribed` 改为 `watched`。

#### Alternatives Considered

- 保留当前“current screen + mine”方案：
  - 不采用。与用户新确认的“所有 watched”要求冲突。
- 继续使用 watched 数据，但保留 `Subscribed` UI 命名：
  - 不采用。语义不准确，会制造新的理解偏差。
- 改 `varpool` store，把 mine 写回 watch list：
  - 不采用。会污染 watch list 语义，且不是这次需求目标。

#### Module Responsibilities

- `frontend/src/pages/Showcase.vue`
  - 用 watched 替换当前主候选来源
  - 将布尔标记和分组标题从 `subscribed` 调整为 `watched`
  - 保留 mine 加载、搜索、去重与回填
- `frontend/src/stores/varpool.ts`
  - 本轮只作为现有 watch list 与 `listOwnerNames` 的提供者，不改契约

#### Data / Call Flow

1. 用户打开变量快捷选择弹窗。
2. 页面立即从 `varpool.state.keys` 构建 watched 候选。
3. 页面并发触发 mine 拉取。
4. watched 与 mine 候选按 `(owner:name)` 合并。
5. 用户搜索过滤。
6. 用户点击某项后回填并关闭弹窗。

#### Interface Drafts

- `type VarQuickPickItem = { name: string; owner: number; mine: boolean; watched: boolean }`
- 分组：
  - `watchedVarQuickPickItems`
  - `mineVarQuickPickItems`

#### Error Handling and Safety

- `owner <= 0` 的 watched 变量：
  - 显式忽略，不尝试猜测 owner
- 未登录：
  - watched 列表仍可显示
  - mine 列表清空并保持错误提示路径
- mine 拉取失败：
  - watched 列表不受影响

#### Performance and Testing Strategy

- watched 候选直接来自现有内存态 `varpool.state.keys`
- mine 拉取仍只在打开弹窗或显式刷新时触发
- 验证策略：
  - `git diff --check`
  - `npm run build`
  - `@vue/compiler-sfc` 定向解析 `Showcase.vue`

#### Extensibility Design Points

- 后续若需继续加入“当前 screen”“最近使用”等来源，可沿用同一去重链路追加标记。
- 将 `watched` / `mine` 语义显式拆开后，未来调整 UI 分组不会影响回填逻辑。

#### Issue List

- 无

### Stage 3.1 - Planning

#### Project Goal and Current State

- 当前主线状态：
  - 上一轮已把主候选改成“当前 screen + mine”
  - 与用户最新确认的“所有 watched”要求不一致
  - 弹窗分组标题仍显示 `Subscribed`
- 本轮目标：
  - 把主候选语义收敛为 watched，并同步修正文案

#### Docs Governance Routing Decision

- docs tree 已存在且结构完整，无需 bootstrap。
- Canonical destination:
  - 稳定需求 / 技术真相：本轮无需修改 requirements/specs
  - 执行控制：worktree 根 `plan.md`
  - 完成结果：`docs/change/2026-03-24_win-showcase-var-picker-watch-all.md`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `docs/requirements/showcase-display-widgets.md`
- Related specs:
  - `docs/specs/showcase-display-widgets.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`

#### Executable Task List

- [x] `SHVW-1` 将快捷选择主候选改为 watched 语义
- [x] `SHVW-2` 同步更新文案、分组和空态
- [x] `SHVW-3` 验证、Code Review 与归档

#### Task Details

##### `SHVW-1` - 将快捷选择主候选改为 watched 语义

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all-win\plan.md`
- Goal:
  - 用 `varpool.state.keys` 的所有有效 watch 条目替换当前主候选来源，并取消订阅态过滤。
- Files / Modules:
  - `frontend/src/pages/Showcase.vue`
- Write Set:
  - `frontend/src/pages/Showcase.vue`
- Acceptance:
  - 未订阅但已 watch 的变量可出现在快捷选择器中
  - mine 合并与点击回填继续正常
- Test Points:
  - 手工打开弹窗，检查 watched 分组是否包含非订阅 watch 项
- Rollback:
  - 回滚 `Showcase.vue` 中快捷选择候选构建逻辑

##### `SHVW-2` - 同步更新文案、分组和空态

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all-win\plan.md`
- Goal:
  - 把弹窗和 tooltip 文案从 subscribed 收敛为 watched，避免语义错配。
- Files / Modules:
  - `frontend/src/pages/Showcase.vue`
- Write Set:
  - `frontend/src/pages/Showcase.vue`
- Acceptance:
  - UI 文案与实际候选来源一致
- Test Points:
  - 静态审查弹窗标题、描述、空态和 badge 文案
- Rollback:
  - 回滚相关模板文案

##### `SHVW-3` - 验证、Code Review 与归档

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all-win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-showcase-var-picker-watch-all-win\plan.md`
- Goal:
  - 完成验证、review 和记入新归档。
- Files / Modules:
  - `frontend/src/pages/Showcase.vue`
  - `docs/change/2026-03-24_win-showcase-var-picker-watch-all.md`
  - `docs/change/README.md`
- Write Set:
  - `frontend/src/pages/Showcase.vue`
  - `docs/change/2026-03-24_win-showcase-var-picker-watch-all.md`
  - `docs/change/README.md`
- Acceptance:
  - review 结论完整
  - change 归档包含 impact、验证和回滚
- Test Points:
  - `git diff --check`
  - `npm run build`
  - `@vue/compiler-sfc` 定向解析 `Showcase.vue`
- Rollback:
  - 回滚本轮代码与文档改动

#### Dependencies

- `AppShell` 已在应用层加载 `varpool.loadWatchList()`，保证 `varpool.state.keys` 可作为 watched 来源。
- `varpool.listOwnerNames(selfNodeId)` 继续作为 mine 拉取能力。

#### Risks and Notes

- 旧 watch 数据若缺少 `owner`，不会出现在快捷选择器中；这是受当前回填契约限制的显式降级。
- 当前仓库若仍缺少 `frontend/wailsjs` 生成物，前端全量 build 可能继续被基线问题阻断。

#### Parallelism Assessment

- 本轮不派发子Agent。
- 原因：
  - 写集集中在 `frontend/src/pages/Showcase.vue`
  - 改动范围小，拆分并行收益低
  - 关键路径是单文件脚本与模板同步收敛

#### Issue List

- 无

阻塞：否
进入 3.2

### Stage 3.3 - Code Review

- 需求覆盖：通过
  - 快捷选择主候选已切回 watched 语义，未订阅但已 watch 的变量不再被过滤。
- 架构合理性：通过
  - 保持局部前端改动，直接复用 `varpool.state.keys` 与现有 mine 拉取能力，无新增接口。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - watched 候选直接来自现有内存态；mine 仍按需加载，无额外订阅或轮询。
- 可读性与一致性：通过
  - `subscribed` 语义已统一收敛为 `watched`，文案、分组与布尔标记保持一致。
- 可扩展性与配置化：通过
  - 候选来源仍采用标记合并模式，后续可继续追加来源而不改回填逻辑。
- 稳定性与安全：通过
  - `owner<=0` 的 watch 条目会被显式过滤；未登录时 mine 列表继续降级，不影响 watched 列表。
- 测试覆盖情况：部分通过
  - `git diff --check` 通过
  - `@vue/compiler-sfc` 定向解析 `Showcase.vue` 通过
  - `npm ci` 通过
  - `npm run build` 失败，原因为仓库基线缺失 `frontend/wailsjs` 生成物，失败点在 `Home.vue` 的 `../../wailsjs/go/session/SessionService`，非本次改动引入
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent。

### Stage 4 - Change Archive

- 手动按 `$docs-governor` 规则完成归档路由与 impact 检查。
- Archive Path:
  - `docs/change/2026-03-24_win-showcase-var-picker-watch-all.md`
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
