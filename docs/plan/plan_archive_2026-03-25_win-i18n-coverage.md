# Plan - win-i18n-coverage

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-i18n-coverage`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage`
- Current Stage: `4 Archive (completed, awaiting workflow-end confirmation)`

## Stage Records

### Initialization
- `guide.md`: 不存在于仓库根，已读取外层控制仓 `guide.md`，worktree 规则与提交约束已确认。
- base/worktree confirmation:
  - 当前实现仓：`MyFlowHub-Win`
  - 基线分支：`main`
  - 独立 worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage`
  - 主仓仅作为控制面使用，本轮实现仅在 worktree 内执行。

### Stage 1 - Requirements Analysis
#### Goal
- 补齐 Win 前端剩余未接入或缺失词条的 i18n，重点覆盖最近新增的 Flow Editor、Showcase 和 Permissions 相关文案。

#### Scope
- Must
  - 修复当前静态扫描 `TOTAL_MISSING=105` 的真实 i18n 缺口。
  - 补齐会直接透出到 UI 的校验错误与 toast 错误文案。
  - 保持现有页面行为、数据结构和交互契约不变。
- Optional
  - 对少量应进入 i18n 的技术示例 placeholder 做无语义变更接线。
- Out of scope
  - 不调整 requirements/specs 行为定义。
  - 不重构 i18n 框架或消息表组织方式。
  - 不翻译 `flow_id`、`run_id`、JSON Pointer 示例、协议字段值这类技术常量本身。

#### Use Cases
- 中文界面下打开 Flow Editor 普通模式，不出现英文按钮、说明、校验错误或缺失 key。
- 中文界面下编辑 Showcase 新展示模式，不出现英文说明和空态提示。
- 中文界面下使用 Permissions 页面，store 抛出的错误信息能够直接本地化显示。

#### Functional Requirements
- 组件和页面中已经通过 `t(...)` 引用但未落消息表的 key 必须补齐。
- 直接通过 `Error.message` 进入 UI 的前端校验错误必须本地化。
- 缺失 key 扫描结果必须归零。

#### Non-functional Requirements
- 采用最小安全改动，避免扩大写集。
- 不新增运行时依赖，不增加额外 I/O。
- 继续维持消息表按域拆分。

#### Inputs / Outputs
- Inputs
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `docs/requirements/showcase-display-widgets.md`
  - `docs/specs/showcase-display-widgets.md`
  - `frontend/src/**` 当前 UI 文案引用与 store 错误路径
- Outputs
  - 补齐后的 `frontend/src/i18n/messages/**`
  - 必要的组件/store 接线改动
  - 验证记录与 change 归档

#### Edge Cases
- 技术常量和示例值可以继续保留原文，但其解释性文案和错误文案必须本地化。
- store 中未走 `t(...)` 的错误若继续保留英文，会绕过页面级 fallback，直接显示给用户。
- 新增 key 若与旧 key 重名但语义不同，可能再次产生覆盖风险。

#### Acceptance Criteria
- 缺失 key 扫描 `TOTAL_MISSING=0`。
- 本轮范围内的页面/弹窗/错误提示不再暴露英文 fallback。
- 不修改 Flow / Showcase / Permissions 的行为语义。

#### Risks
- `automation.ts` 和 `showcase.ts` 的写集集中，容易出现重复 key 或错误归类。
- `flow.ts` 与 `flow_json_pointer.ts` 的错误文本若补 key 不同步，扫描归零但运行时仍可能漏翻。

#### Issue List
- 无阻塞项。

### Stage 2 - Architecture Design
#### Overall Solution
- 继续使用现有轻量 i18n 结构，针对缺口补齐消息表并修复少量未本地化的错误/placeholder 接线。

#### Alternatives Considered
- 方案 A：引入新的 i18n 抽象或自动扫描框架。
  - 放弃原因：本轮问题是覆盖缺口，不是能力缺失；改造面过大。
- 方案 B：只补消息表，不改 store 错误。
  - 放弃原因：`Error.message` 直接进 toast，仍会向用户暴露英文。
- 结论：采用“按域补消息表 + 必要调用点接线”的最小方案。

#### Module Responsibilities
- `frontend/src/i18n/messages/automation.ts`
  - 补齐 Flow Editor、Flow Node、flow store、JSON Pointer 校验相关词条。
- `frontend/src/i18n/messages/showcase.ts`
  - 补齐 Showcase 页面、新 widget 模式与展示组件词条。
- `frontend/src/i18n/messages/operations.ts`
  - 补齐 Permissions 页面/策略校验词条。
- `frontend/src/stores/permissions.ts`
  - 将直接抛给 UI 的错误接到 `t(...)`。
- `frontend/src/stores/flow_json_pointer.ts`
  - 将 JSON Pointer 解析错误接到 `t(...)`。

#### Data / Call Flow
- 页面 / 组件 / store -> `t(key, params?)` -> 消息表 -> 本地化 UI。
- store 校验失败 -> `throw new Error(t(...))` -> 页面 `toast.errorOf(...)` -> 用户看到已翻译错误。

#### Interface Drafts
- 不新增对外接口。
- 不改 Wails binding、store public API 或后端数据结构。

#### Error Handling and Safety
- 仅替换错误文案来源，不调整校验条件和抛错时机。
- 对技术常量保持原值，避免误导输入或破坏协议兼容。

#### Performance and Testing Strategy
- 验证顺序：
  1. 缺失 key 静态扫描
  2. `frontend` 构建验证
  3. `git diff --check`
- 词条补齐为纯内存静态对象修改，不新增运行时开销。

#### Extensibility Design Points
- 继续按 `automation / showcase / operations` 域拆分消息表，便于后续模块追加词条。
- 把错误消息保持在实际抛错点 `t(...)`，避免页面层重复包装。

#### Issue List
- 无阻塞项。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标：完成 Win 前端剩余高频路径 i18n 收口，使 Flow Editor、Showcase、Permissions 在中文环境下不再暴露英文 fallback。
- 当前状态：
  - 现有扫描显示 `TOTAL_MISSING=105`。
  - 缺口集中在 8 个文件：
    - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
    - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
    - `frontend/src/stores/flow.ts`
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/components/showcase/ShowcaseWidgetCardContent.vue`
    - `frontend/src/pages/Permissions.vue`
    - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
    - `frontend/src/components/flow/FlowNode.vue`
  - 另有 `frontend/src/stores/permissions.ts` 与 `frontend/src/stores/flow_json_pointer.ts` 存在未本地化错误字符串。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/requirements/showcase-display-widgets.md`
- Related specs:
  - `docs/specs/flow-editor-visual-form.md`
  - `docs/specs/showcase-display-widgets.md`
- Related lessons: `none`
- 路由结论：
  - 本轮属于既有能力的 i18n 收口，不改长期行为真相；稳定真相仍留在 requirements/specs。
  - 本 workflow 控制文档使用 worktree 根 `plan.md`。
  - 完成后结果归档到 `docs/change/`；若未产生可复用故障经验，则不新建 lessons。

#### Executable Task List
- [x] `T1` 补齐 Flow Editor / Flow store 缺失词条与错误文案
- [x] `T2` 补齐 Showcase 页面与 widget 展示词条
- [x] `T3` 补齐 Permissions 页面与 store 错误文案
- [x] `T4` 运行缺失 key 扫描、构建校验与 code review
- [x] `T5` 归档 `docs/change` 并更新索引

#### Task Details
##### T1 - Flow i18n 收口
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage\plan.md`
- Goal:
  - 清零 Flow Editor 与 flow store 当前缺失的 i18n key。
  - 保证 JSON Pointer 和普通模式校验错误直接本地化。
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_json_pointer.ts`
  - `frontend/src/i18n/messages/automation.ts`
- Write Set:
  - 上述文件
- Acceptance:
  - Flow 相关缺失 key 为 0。
  - Flow 相关校验错误不再直接显示英文。
- Test Points:
  - 缺失 key 扫描
  - 前端构建
- Rollback:
  - 回退上述 Flow 相关文件改动。

##### T2 - Showcase i18n 收口
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage\plan.md`
- Goal:
  - 补齐 Showcase 新展示模式和 widget 卡片的词条与空态提示。
- Files / Modules:
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/components/showcase/ShowcaseWidgetCardContent.vue`
  - `frontend/src/i18n/messages/showcase.ts`
- Write Set:
  - 上述文件
- Acceptance:
  - Showcase 相关缺失 key 为 0。
  - `metric / badge / progress` 相关中文文案完整。
- Test Points:
  - 缺失 key 扫描
  - 前端构建
- Rollback:
  - 回退上述 Showcase 相关文件改动。

##### T3 - Permissions i18n 收口
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage\plan.md`
- Goal:
  - 补齐 Permissions 页面与 store 仍未进入 i18n 的校验文案。
- Files / Modules:
  - `frontend/src/pages/Permissions.vue`
  - `frontend/src/stores/permissions.ts`
  - `frontend/src/i18n/messages/operations.ts`
- Write Set:
  - 上述文件
- Acceptance:
  - Permissions 相关缺失 key 为 0。
  - store 抛出的错误信息本地化。
- Test Points:
  - 缺失 key 扫描
  - 前端构建
- Rollback:
  - 回退上述 Permissions 相关文件改动。

##### T4 - 验证与审查
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage\plan.md`
- Goal:
  - 验证本轮 i18n 收口结果并执行 3.3 Code Review。
- Files / Modules:
  - 全部改动文件
- Write Set:
  - `plan.md`
- Acceptance:
  - 缺失 key 扫描归零。
  - `frontend/npm run build` 通过，或明确记录非本轮阻塞。
  - 3.3 检查项全部通过。
- Test Points:
  - 静态扫描
  - `frontend/npm run build`
  - `git diff --check`
- Rollback:
  - 退回相关任务修复问题后重跑验证。

##### T5 - Change Archive
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-i18n-coverage\plan.md`
- Goal:
  - 归档本轮 workflow 结果并更新必要索引。
- Files / Modules:
  - `docs/change/2026-03-25_win-i18n-coverage.md`
  - `docs/change/README.md`
  - `plan.md`
- Write Set:
  - 上述文档
- Acceptance:
  - archive 记录 requirements/specs/lessons impact、测试结论和任务映射。
- Test Points:
  - 文档自检与索引更新
- Rollback:
  - 删除本轮 archive 并回退索引修改。

#### Dependencies
- `T1` / `T2` / `T3` 完成后才能进行 `T4`。
- `T4` 通过后才能进行 `T5`。

#### Risks and Notes
- Flow 相关词条需要统一落在 `automation.ts`，避免散落到其他 message 文件。
- 技术示例 placeholder 若保持原文，需要在 change 中明确说明其为有意保留。

#### Parallelism Assessment
- 不使用子Agent。
- 原因：
  - `automation.ts` 与 `showcase.ts` 为集中写集。
  - 当前任务以同一套 i18n 词条和相邻 UI 文件为主，冲突概率高。
  - 未获得用户显式委派并行子Agent授权。

#### Issue List
- 无阻塞项。

### Stage 3.2 - Implementation
#### Result
- `T1`
  - 已补齐 `automation.ts` 中 Flow Editor、compose、JSON Pointer、advanced spec 与绑定校验词条。
  - `flow_json_pointer.ts` 解析错误已改为 `t(...)`。
- `T2`
  - 已补齐 `showcase.ts` 中 `metric / badge / progress` 模式、空态和范围校验文案。
- `T3`
  - 已补齐 `operations.ts` 中 Permissions 缺失词条。
  - `permissions.ts` 的 raw 英文错误已改为 `t(...)`。
  - `appSettings.ts`、`devices.ts`、`flowProjects.ts`、`language.ts` 的 binding unavailable 错误已改为 `t(...)`。

#### Notes
- 本轮仅补文案与错误消息入口，不改页面行为、数据结构和后端契约。

### Stage 3.3 - Code Review
#### Validation Results
- quote-aware 缺失 key 扫描：`TOTAL_MISSING=0`
- `git diff --check`：通过（仅 LF/CRLF warning）
- 变更文件 TypeScript 局部转译：`TS_TRANSPILE=PASS`
- `frontend/npm run build`：
  - 未通过
  - 原因：既有 `wailsjs/go/main/App` 绑定缺失，`src/windows/TopicBusWindow.vue` 解析失败，与本轮改动文件无直接关联

#### Review Checklist
- 需求覆盖：通过
  - Flow Editor、Showcase、Permissions 的缺失 key 与高频错误路径已覆盖。
- 架构合理性：通过
  - 继续复用现有 message 分域结构，无新增耦合层。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅静态消息表和错误文本改造，无额外运行时 I/O。
- 可读性与一致性：通过
  - 错误消息统一在抛错点本地化，减少页面层特殊处理。
- 可扩展性与配置化：通过
  - 新增词条仍按域组织，后续功能可按模块增补。
- 稳定性与安全：通过
  - 未改业务校验条件和执行路径，仅调整文案来源。
- 测试覆盖情况：通过（带环境阻塞说明）
  - 静态扫描与局部转译通过；完整 build 受既有 `wailsjs` 缺失阻塞。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子Agent。

#### Conclusion
- Review 通过，可进入 Stage 4 归档。

阻塞：否
Stage 4 已完成，等待用户确认是否结束 workflow
