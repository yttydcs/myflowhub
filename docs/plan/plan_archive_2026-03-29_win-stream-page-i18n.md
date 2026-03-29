# Plan - Stream 页面简化与 i18n 补齐

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/stream-page-i18n`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win`
- Current Stage: `4 archive complete, workflow ended`

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `$m-docs` 的 `SKILL.md`、`references/requirement-impact.md`、`references/templates.md`
- repo / branch / worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `feat/stream-page-i18n`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win`
  - implementation will stay inside the worktree only

### Stage 1 - Requirements Analysis
#### Goal
- 观察现有 `Stream` 页面与同仓其他成熟页面后，重做 `Stream` 的信息排布与交互密度。
- 在不改变 `Stream` 稳定能力边界的前提下，让页面更简洁、更易用，并补齐该模块的 i18n 文案覆盖。

#### Scope
- 必须
  - 将 `Stream` 主页面从“多块大表单同时常驻”改为“紧凑概览 + 列表选择 + 弹窗式新增/编辑/查询输入”。
  - 保留并暴露既有 `source / consumer / delivery` 控制能力，不删协议动作。
  - 让 `Stream` 页面与路由/导航相关文案真正接入现有 i18n 词典，而不是停留在英文 key 回退。
  - 复用仓内已有的 `Overlay`、`CardHeader`、`PageHero` 等模式，保持与 `Flow`、审批页等体验一致。
  - 为关键改动补充或更新前端测试，并执行构建验证。
- 可选
  - 收敛少量 `stream.ts` 辅助视图逻辑，使模板更容易维护。
  - 补充更聚焦的状态摘要或空态提示。
- 不做
  - 不改动 Go `StreamService`、runtime 事件协议或后端持久化契约。
  - 不新增音视频播放窗口或新的 stream 协议能力。
  - 不引入新的 i18n 框架，继续沿用现有轻量 i18n 结构。

#### Use Cases
- 用户打开 `Stream` 页后，先看到当前节点、默认目标、sources/consumers/deliveries 的简明摘要，而不是大段表单。
- 用户需要新增本地 source 或 consumer 时，通过点击按钮打开弹窗填写必要字段，完成后回到主页面继续操作。
- 用户选择 source、consumer 和 delivery 后，在主页面直接完成 connect、subscribe、disconnect、signal 和 runtime 查看。
- 用户切换到中文时，`Stream` 页面、路由标题、副标题与相关提示能正确显示中文。

#### Functional Requirements
- `Stream` 页面必须保留：
  - 查询 sources
  - 查询 consumers
  - 新增 / 撤销 source
  - 新增 / 撤销 consumer
  - `connect` / `disconnect`
  - `subscribe` / `unsubscribe`
  - `signal`
  - delivery runtime viewer
- 主页面必须减少长期常驻表单，仅保留高频动作和选择态。
- 新增 source / consumer 的录入必须通过弹窗完成，且保留原有字段能力。
- 页面必须明确区分：
  - source 选择态
  - consumer 选择态
  - delivery 选择态
  - kind 不匹配时的阻止与说明
- `Stream` 相关用户可见文案必须通过消息表提供 `zh-CN` 翻译。

#### Non-functional Requirements
- 变更面尽量收敛在 `frontend/src/pages/Stream.vue` 与 i18n 词条聚合。
- 继续复用现有 `Overlay` 组件，不引入新弹窗框架。
- 保持现有 store 的输入校验与错误处理边界，不把协议逻辑搬回页面。
- 页面结构应适配桌面主布局与较窄宽度下的纵向堆叠。

#### Inputs / Outputs
- 输入
  - 现有稳定需求：`docs/requirements/stream.md`
  - 现有稳定技术边界：`docs/specs/stream.md`
  - 现状实现：`frontend/src/pages/Stream.vue`、`frontend/src/stores/stream.ts`
  - 参考页面：`frontend/src/pages/Flow.vue`、`frontend/src/pages/RegistrationApprovals.vue`
- 输出
  - 重构后的 `frontend/src/pages/Stream.vue`
  - 更新后的 i18n message 文件
  - 必要的测试与 `docs/change` 归档

#### Edge Cases
- 未连接、未登录、无 `node_id` 时，页面仍需保留清晰空态和错误反馈。
- source 与 consumer `kind` 不匹配时，主页面必须继续阻止 `connect` / `subscribe`。
- `Stream` bindings 缺失时，错误信息仍需走现有 i18n + toast 路径。
- 没有任何 source / consumer / delivery 时，页面必须保持可理解，不出现大片空白表单区。

#### Acceptance Criteria
- `Stream` 页面不再同时常驻 source 与 consumer 两大录入表单。
- 用户可通过点击按钮打开弹窗新增 source / consumer，并完成原有创建流程。
- 页面主视图能更聚焦地完成查询、选择、连接和查看 runtime。
- `Stream` 页面及其路由/导航相关文案在中文下可正确显示。
- 前端测试与构建验证通过，未破坏现有 `stream` store 基本行为。

#### Risks
- 若只重排模板、不补齐词条，切中文后仍会显示英文 fallback。
- 若把过多业务状态迁回页面，可能和 `stream.ts` 的单一职责冲突。
- 若新增交互超出计划范围，应先回到 3.1 更新计划，避免需求扩散。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 方案 A（采用）
  - 以 `Stream.vue` 单页重构为主，保留 `stream.ts` 的控制面与 runtime 状态职责。
  - 主页面改为三层结构：
    - Hero + 摘要
    - source / consumer / delivery 紧凑列表与选择态
    - 侧栏或下方 viewer / action 区
  - 新增 source / consumer 的字段录入改为两个 `Overlay` 弹窗。
  - i18n 沿用现有 `t(key)` + `messages/*` 聚合方式，在对应 message 文件中补齐 `Stream` 词条。
- 不采用方案
  - 引入新页面拆分或多路由
    - 理由：当前问题是单页内信息密度失衡，不是模块边界错误。
  - 引入 `vue-i18n` 或其他翻译框架
    - 理由：仓库已有轻量 i18n 方案，当前问题是词条缺失，不是框架能力不足。
  - 修改 Go `StreamService` 接口
    - 理由：本轮只做 UX 和文案补齐，稳定契约无需改变。

#### Module Responsibilities
- `frontend/src/pages/Stream.vue`
  - 负责页面布局、摘要视图、弹窗开关、选择态和高频操作编排。
- `frontend/src/stores/stream.ts`
  - 继续负责 bindings 调用、输入校验、业务状态镜像和 `stream.*` 事件监听。
- `frontend/src/i18n/messages/common.ts`
  - 承载可复用通用按钮/状态文案。
- `frontend/src/i18n/messages/operations.ts`
  - 承载 `Stream` 页面新增的业务动词、空态、提示和 toast 文案。
- `frontend/src/i18n/messages/shell.ts`
  - 若导航或模块简介文案需要补齐，则在这里维护。
- `frontend/src/router/index.ts`
  - 保持 `/stream` 路由不变，仅同步标题/副标题文案。

#### Data / Call Flow
1. 页面挂载时继续调用 `stream.loadDeliveries()`，并从 `sessionStore` 同步 `selfNodeId / hubId / defaultTargetId`。
2. 用户在主页面点击查询按钮，使用现有 query state 调用 `listSources / listConsumers`。
3. 用户点击 `New Source` 或 `New Consumer` 打开弹窗，提交时复用现有 `announceSource / announceConsumer`。
4. 用户从列表中选择 source / consumer / delivery 后，主页面 action 区调用 `connect / subscribe / disconnect / unsubscribe / signal`。
5. viewer 区继续按 `selectedDelivery.kind` 读取 `textFramesFor / statsFor` 展示运行态。
6. 中文环境下，所有 `t(...)` key 在消息表中命中 `zh-CN`，英文继续走空 `en` + key fallback。

#### Interface Draft
- `Stream.vue`
  - 新增
    - `sourceDialogOpen`
    - `consumerDialogOpen`
    - 摘要卡片计算属性
    - 弹窗关闭 / 重置帮助函数
  - 保留
    - `sourceQuery`
    - `consumerQuery`
    - `sourceDraft`
    - `consumerDraft`
    - 既有 control handlers
- `stream.ts`
  - 优先不改对外接口
  - 如需简化模板，只允许新增轻量视图辅助，不改变业务契约

#### Error Handling and Safety
- 继续通过 `stream.ts` 负责必填、节点 ID、metadata JSON 的校验。
- 页面侧不吞错误；所有动作继续走 `toast.success / errorOf`。
- 弹窗关闭时只重置本地 draft，不改动 store 中已加载的 source / consumer / delivery 数据。

#### Performance and Testing Strategy
- 性能
  - 不新增额外事件监听，不改变 `stream.*` runtime 数据流。
  - 通过视图模型收敛模板内重复计算，避免在模板中反复拼装复杂字符串。
- 测试
  - `npm test -- src/stores/stream.test.ts`
  - 若新增页面测试，则追加对应 `Stream` 页面测试文件
  - `npm run build`

#### Extensibility Design Points
- 弹窗式录入可为后续 source / consumer 编辑功能复用。
- `Stream` 页面若后续新增 viewer window，可在不改变当前主页面 action 结构的前提下增加入口。
- i18n 词条按现有 message 分层维护，便于后续继续补齐更多 Stream 文案。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 `Stream` 页功能齐全，但把 source/consumer 查询、两个本地录入表单、delivery 操作和 viewer 全部平铺在主页面，导致页面过重。
- 页面已经调用 `t(...)`，但 `messages/*` 中尚未补齐大部分 `Stream` 词条，切中文仍以英文原文回退。
- `Flow.vue`、`RegistrationApprovals.vue` 已经提供成熟的“列表 + 操作按钮 + Overlay 弹窗”模式，可直接参考。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- docs tree 无需 bootstrap 或 repair。
- stable truth 继续保留在：
  - `docs/requirements/stream.md`
  - `docs/specs/stream.md`
- workflow result 进入：
  - `docs/change/2026-03-29_win-stream-page-i18n.md`
- reusable lessons：
  - 暂不预设；若本轮暴露可复用坑点，再决定是否补 `docs/lessons`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements
  - `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win\docs\requirements\stream.md`
- Related specs
  - `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win\docs\specs\stream.md`
- Related lessons
  - none

#### Executable Task List
- [x] `STREAM-UX-1` 重构 `Stream` 主页面布局与摘要区
- [x] `STREAM-UX-2` 将 source / consumer 新增录入改为弹窗式交互
- [x] `STREAM-I18N-1` 补齐 `Stream` 页面、路由、导航相关词条
- [x] `STREAM-VAL-1` 更新/补充前端测试并执行构建验证
- [x] `STREAM-REVIEW-1` 完成 3.3 checklist
- [x] `STREAM-ARCHIVE-1` 归档 `docs/change`

#### Task Details
##### `STREAM-UX-1` - Stream page layout refactor
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win\plan.md`
- Goal:
  - 让 `Stream` 主页面聚焦“摘要、列表、动作、viewer”，减少一次性暴露的输入面
- Files / Modules:
  - `frontend/src/pages/Stream.vue`
- Write Set:
  - `frontend/src/pages/Stream.vue`
- Acceptance:
  - 主页面不再常驻两大录入表单
  - 摘要、列表选择和 action/viewer 分区更清晰
- Test Points:
  - 页面关键操作仍可触发既有 handlers
  - 选择 source / consumer / delivery 的状态不回归
- Rollback:
  - 回退 `frontend/src/pages/Stream.vue`

##### `STREAM-UX-2` - Overlay dialogs for local source and consumer
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win\plan.md`
- Goal:
  - 使用仓内 `Overlay` 模式承载新增 source / consumer 的录入
- Files / Modules:
  - `frontend/src/pages/Stream.vue`
- Write Set:
  - `frontend/src/pages/Stream.vue`
- Acceptance:
  - 用户可通过按钮打开弹窗完成新增 source / consumer
  - 原有字段能力与提交结果保持一致
- Test Points:
  - 提交后关闭弹窗并重置对应 draft
  - metadata JSON 错误继续由 store 抛错并 toast 提示
- Rollback:
  - 回退 `frontend/src/pages/Stream.vue`

##### `STREAM-I18N-1` - Stream translation coverage
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win\plan.md`
- Goal:
  - 让 `Stream` 页面、路由、副标题和相关提示在中文下可读
- Files / Modules:
  - `frontend/src/i18n/messages/common.ts`
  - `frontend/src/i18n/messages/operations.ts`
  - `frontend/src/i18n/messages/shell.ts`
  - `frontend/src/router/index.ts`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/stores/stream.ts`（仅在必要时补少量缺失 key）
- Write Set:
  - `frontend/src/i18n/messages/common.ts`
  - `frontend/src/i18n/messages/operations.ts`
  - `frontend/src/i18n/messages/shell.ts`
  - `frontend/src/router/index.ts`
  - `frontend/src/pages/Stream.vue`
  - `frontend/src/stores/stream.ts`
- Acceptance:
  - 中文环境下不再大量显示 `Stream` 页面英文 key
  - 路由 / 导航文案与页面内容一致
- Test Points:
  - 关键 key 在消息表中有 `zh-CN` 映射
  - 现有 i18n 聚合不报错
- Rollback:
  - 回退上述 i18n / 路由相关文件

##### `STREAM-VAL-1` - Validation
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win\plan.md`
- Goal:
  - 确认 UI 重构没有破坏 store 行为与前端构建
- Files / Modules:
  - `frontend/src/stores/stream.test.ts`
  - optional new page test file if needed
- Write Set:
  - `frontend/src/stores/stream.test.ts`
  - optional `frontend/src/pages/Stream.test.ts`
- Acceptance:
  - 相关 vitest 通过
  - `npm run build` 通过
- Test Points:
  - store 关键行为
  - 页面弹窗/文案/关键视图状态（如新增页面测试）
- Rollback:
  - 回退新增或更新的测试文件

##### `STREAM-REVIEW-1` - Stage 3.3 review
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win\plan.md`
- Goal:
  - 对照需求、架构、测试和稳定性完成 checklist 复核
- Files / Modules:
  - `plan.md`
- Write Set:
  - `plan.md`
- Acceptance:
  - 3.3 每项给出 `通过` / `不通过`
- Test Points:
  - review checklist
- Rollback:
  - 更新 `plan.md` review 记录

##### `STREAM-ARCHIVE-1` - Change archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-stream-page-i18n\MyFlowHub-Win\plan.md`
- Goal:
  - 归档本次 Stream 页面重构与 i18n 补齐的背景、改动、验证和回滚
- Files / Modules:
  - `docs/change/2026-03-29_win-stream-page-i18n.md`
  - `docs/change/README.md`
- Write Set:
  - `docs/change/2026-03-29_win-stream-page-i18n.md`
  - `docs/change/README.md`
- Acceptance:
  - change 文档完整记录 impact、task mapping、验证与回滚
- Test Points:
  - manual doc review
- Rollback:
  - 回退新增 / 更新的 change 文档

#### Dependencies
- `frontend/src/stores/stream.ts`
  - 提供 `Stream` 页面所有控制动作与 runtime viewer 状态
- `frontend/src/components/ui/overlay/Overlay.vue`
  - 提供现有可复用弹窗能力

#### Risks and Notes
- 若页面层新增过多局部状态，可能再次让 `Stream.vue` 过重；优先用最小必要状态。
- i18n 仍以英文 key 作为 fallback，因此必须明确补齐 `zh-CN` 词条，否则中文体验不会改善。
- 本轮以单仓串行执行，不使用子 Agent。

#### Parallelism Assessment
- 不派发子Agent
- 原因
  - 当前会话未获得显式子Agent授权
  - 写集高度重叠，串行更利于保持 UI 与文案一致性

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### File-Level Change Summary
- `STREAM-UX-1` / `STREAM-UX-2`
  - `frontend/src/pages/Stream.vue`
  - 将 `Stream` 主页面重构为 `PageHero + summary cards + source/consumer 紧凑列表 + 右侧 action/runtime/viewer` 布局
  - 移除常驻 source / consumer 大表单，改为 `Overlay` 弹窗式新增
  - 增加首屏静默目录同步，在身份可用后自动查询 sources / consumers / deliveries
  - 增加测试选择器，便于验证“默认无常驻表单、点击后再打开弹窗”
- `STREAM-I18N-1`
  - `frontend/src/i18n/messages/operations.ts`
  - `frontend/src/i18n/messages/shell.ts`
  - 补齐 `Stream` 页面、toast、store 错误、导航标题、导航描述和路由副标题的 `zh-CN` 映射
- `STREAM-VAL-1`
  - `frontend/src/pages/Stream.test.ts`
  - `frontend/src/i18n/messages/shell.test.ts`
  - 新增页面测试覆盖弹窗式新增和首屏自动加载；扩展 shell 词条测试覆盖 `Stream` 导航/路由文案

#### Design Notes
- 保持 `frontend/src/stores/stream.ts` 的协议契约和输入校验不变，只在页面层重排交互密度
- 弹窗提交后继续复用 store 的错误处理和 toast 通路，不在页面层吞掉校验异常
- 路由和侧边栏继续使用原 key 文案，由 `shell.ts` 承接中文翻译，避免改动导航/路由结构

#### Validation Record
- 依赖安装
  - 执行：`npm ci`（`frontend/`）
  - 结果：通过
- 前端测试
  - 执行：`npm test`
  - 结果：通过，`17` 个测试文件 / `69` 个测试用例全部通过
- 前端构建
  - 执行：`npm run build`
  - 结果：通过
  - 备注：Vite 报现有大包体 warning（`index-*.js` 超过 `500 kB`），本轮未新增阻塞

#### Validation Notes
- worktree 首次验证时缺少被 `.gitignore` 忽略的 `frontend/wailsjs`
- 为完成本轮验证，已将 `repo/MyFlowHub-Win/frontend/wailsjs` 复制到 worktree 的 `frontend/wailsjs`
- 该目录为生成态依赖，未进入提交变更集；同时记录为可复用 lesson

### Stage 3.3 - Code Review
- `需求覆盖`：通过
  - 主页面改为摘要 + 列表 + action/viewer，source / consumer 新增改为弹窗，中文词条补齐
- `架构合理性`：通过
  - 只重构页面壳层和词典，不改动 `stream.ts` 或后端契约
- `性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）`：通过
  - 仅在身份就绪时静默拉取一次目录快照，未新增额外事件监听或重复轮询
- `可读性与一致性`：通过
  - 继续复用 `PageHero`、`CardHeader`、`Overlay`，文案和卡片结构与现有页面风格对齐
- `可扩展性与配置化`：通过
  - 弹窗式新增为后续编辑复用留出入口，未写死新的环境路径或协议分支
- `稳定性与安全`：通过
  - 仍由 store 负责 target / metadata / node ID 校验；页面层只调用现有动作
- `测试覆盖情况`：通过
  - 新增 `Stream` 页面测试；全量 vitest 和构建均已通过
- `子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）`：通过
  - 未使用子Agent

### Stage 4 - Change Archive
#### Docs Routing and Impact Check
- 使用 `$m-docs` 完成归档路由、impact 检查和 lessons 决策
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `updated`
- Related requirements
  - `docs/requirements/stream.md`
- Related specs
  - `docs/specs/stream.md`
- Related lessons
  - `docs/lessons/frontend-worktree-wailsjs-missing.md`

#### Archive Outputs
- 已创建 `docs/change/2026-03-29_win-stream-page-i18n.md`
- 已更新 `docs/change/README.md`
- 已创建 `docs/lessons/frontend-worktree-wailsjs-missing.md`
- 已更新 `docs/lessons/README.md`
