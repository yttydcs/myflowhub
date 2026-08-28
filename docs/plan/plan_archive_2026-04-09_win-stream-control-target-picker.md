# Plan - MyFlowHub-Win Stream Control Target Picker

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `feat/win-stream-control-target-picker`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - workspace root `D:\project\MyFlowHub3\guide.md` 已阅读
  - 遵守 `AGENTS.md` 与 `$m-autoflow` 约束：实现只在 `worktrees/` 中进行，计划确认前不进入编码，commit 信息若后续需要提交则使用中文
- base/worktree confirmation:
  - 目标仓库：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 主仓默认分支：`main`
  - `repo/MyFlowHub-Win` 主仓当前存在用户自己的未提交改动：`go.mod`、`myflowhub-mcp.exe`
  - 主仓当前仅作控制面读取与 worktree 管理，不做实现修改
  - 本轮唯一执行 worktree：`D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker`
  - 本轮仅涉及 `MyFlowHub-Win`，不涉及跨 repo 实现

### Stage 1 - Requirements Analysis
#### Goal
- 调整 `Stream` 的 `Control` 页面，使主页面不再常驻 `Source Catalog` 与 `Consumer Catalog`。
- 保留 `Control Target` 与 runtime delivery 管理能力，并把 source / consumer 的查询和连接动作收敛为弹出式选择框。
- 在不改协议、不改后端 binding 的前提下，让 `Control` 页继续满足“主画面简洁、动作走弹窗”的 Win 产品风格。

#### Scope
- Must:
  - `Control` 页移除常驻 `Source Catalog` / `Consumer Catalog`
  - `Control Target` 区域保留目标节点输入和 refresh 能力
  - `connect` 通过弹窗完成 source / consumer 的查询、选择和提交
  - `subscribe` 能力继续保留，且不依赖已移除的常驻 catalog
  - `Runtime Deliveries` 的查看、`disconnect`、`unsubscribe`、`signal` 能力保持不变
- Optional:
  - 在弹窗中显示当前选中 pair 摘要
  - 在弹窗中沿用现有 kind/tag/node 查询项，但布局更紧凑
- Not in scope:
  - 不改 `stream` 协议 wire
  - 不改 Go `StreamService`、Wails binding 或 store 的数据模型
  - 不新增独立 control window、独立路由或新的后端查询接口

#### Use Cases
- 用户进入 `Control` 页时，希望先看到 target 与当前 delivery 状态，而不是两个大 catalog 面板。
- 用户点击连接动作后，在一个弹窗中完成 source / consumer 的查询和配对，再执行 `connect`。
- 用户仍可在 `Control` 相关交互里执行 `subscribe`，但不需要把主页面重新展开成目录页。

#### Functional Requirements
1. `Control` 标签页不得再在主画面渲染 `Source Catalog` 和 `Consumer Catalog` 两块常驻区域。
2. 页面必须保留 `Control Target` 输入，并继续作为 `list_sources`、`list_consumers`、`connect`、`subscribe`、`disconnect`、`signal` 的调用目标路由。
3. 页面必须通过弹窗完成 source / consumer 的查询与选择，并在弹窗内执行至少 `connect`。
4. 页面必须继续支持 `subscribe`，且新的交互不得依赖已经移除的常驻 catalog。
5. 若 source / consumer `kind` 不匹配，必须继续显式阻止连接。
6. 弹窗关闭时不得破坏 runtime delivery 的现有选择与操作。

#### Non-functional Requirements
- 变更面最小化：
  - 优先收敛到 `Stream.vue`、相关页面测试和少量 i18n
- 一致性：
  - 继续复用现有 `Overlay` 弹窗模式和已有 toast / store 调用链
- 可维护性：
  - 不引入新的 store contract 或重复的协议调用封装
- 可用性：
  - 主页面更聚焦，弹窗内仍提供足够的过滤条件完成查找

#### Inputs / Outputs
- Inputs:
  - `targetId`
  - source 查询条件：`producer` / `kind` / `tag`
  - consumer 查询条件：`consumer` / `kind` / `tag`
  - 被选中的 source / consumer
- Outputs:
  - 调用现有 `stream.listSources(...)` / `stream.listConsumers(...)`
  - 调用现有 `stream.connect(...)` / `stream.subscribe(...)`
  - 刷新后的 `deliveries` / toast 反馈 / 选中态

#### Edge Cases
- 未选全 pair 就尝试提交
- source / consumer `kind` 不匹配
- 远端 catalog 为空或查询失败
- 弹窗打开时查询条件未初始化或 target 为空
- 关闭弹窗后遗留不可见 selection，导致后续交互混淆

#### Acceptance Criteria
1. `Control` 页不再显示 `Source Catalog` / `Consumer Catalog`。
2. 用户可通过弹窗完成 source / consumer 配对并成功触发 `connect`。
3. 用户仍可触发 `subscribe`，且交互不依赖被移除的常驻 catalog。
4. `Runtime Deliveries` 区域及其操作不回退。
5. 定向页面测试覆盖新的 control 弹窗交互并通过。

#### Risks
- 现有 `selectedSourceId` / `selectedConsumerId` 是 store 级状态；若直接复用，需要处理弹窗关闭后的可见性与重置策略。
- `subscribe` 原先复用相同 catalog 数据源，重构时要避免只顾 `connect` 而破坏 `subscribe` 路径。
- fresh worktree 的前端构建可能仍受 `node_modules` 或 Wails 生成物影响，需把环境问题与本次页面回归区分开。

#### Issue List
- 无新增阻塞。

### Stage 2 - Architecture Design
#### Overall Solution
- 保留 `Control Target` 与 `Runtime Deliveries` 两块主页面区域。
- 删除 `Control` 主页面中的常驻 `Source Catalog` 与 `Consumer Catalog`。
- 新增一个共享 `Overlay` 配对弹窗，弹窗内承载：
  - source 查询条件与结果列表
  - consumer 查询条件与结果列表
  - 当前 pair 选择状态
  - 在同一弹窗内执行 `connect` 或 `subscribe`
- 主页面上的控制入口改为“打开配对弹窗”，而不是直接依赖主页面常驻目录完成配对。
- 继续复用现有 `stream.listSources`、`stream.listConsumers`、`stream.connect`、`stream.subscribe`、`stream.selectSource`、`stream.selectConsumer`，不改 store 数据模型。

#### Alternatives Considered
- 保留主页面 catalog，但折叠成 accordion：
  - 放弃。视觉上仍旧占据主页面，不能真正满足“移除掉”的要求。
- 新开独立 control window：
  - 放弃。超出当前需求，且引入额外路由/窗口复杂度。
- 使用弹窗承载查询与提交：
  - 采用。与现有 `Subscribe Consumer` 弹窗模式一致，且改动面最小。

#### Module Responsibilities
- `frontend/src/pages/Stream.vue`
  - 新增 control 配对弹窗状态、打开/关闭逻辑、查询刷新时机
  - 移除主页面常驻 catalog 区域
  - 把 control 动作入口改为弹窗触发
- `frontend/src/i18n/messages/stores.ts`
  - 更新 control 页相关文案，补充配对弹窗标题/描述/空态/按钮文本
- `frontend/src/pages/Stream.test.ts`
  - 覆盖 catalog 移除与 control 配对弹窗连接路径
- `frontend/src/stores/stream.ts`
  - 预计不改；仅复用现有 target / query / connect / subscribe 调用边界

#### Data / Call Flow
1. `Stream` 页面初始化时继续设置 identity、读取 prefs、加载 deliveries，并使用当前 `targetId` 作为 stream 调用目标。
2. 用户进入 `Control` 页后，只看到 `Control Target` 和 `Runtime Deliveries`。
3. 用户点击 control 动作按钮，页面打开配对弹窗。
4. 弹窗内根据 source / consumer 查询条件调用 `stream.listSources(...)` 和 `stream.listConsumers(...)`。
5. 用户在弹窗内选择 source / consumer。
6. 用户在弹窗内提交：
  - `connect` 调用 `stream.connect(...)`
  - `subscribe` 调用 `stream.subscribe(...)`
7. 成功后复用现有 delivery 更新逻辑，并关闭弹窗；失败时沿用现有 toast 错误反馈。

#### Interface Drafts
- 新增本地页面状态草案：
  - `controlDialogOpen: boolean`
- 复用现有查询状态：
  - `controlSourceQuery`
  - `controlConsumerQuery`
- 复用现有选择态：
  - `stream.state.selectedSourceId`
  - `stream.state.selectedConsumerId`
- 主页面 control 操作入口草案：
  - `openControlDialog()`

#### Error Handling and Safety
- 弹窗提交前必须校验 pair 是否完整。
- `connect` 继续使用现有 `canConnect` 逻辑阻止 kind 不匹配。
- `subscribe` 继续使用现有 `canSubscribe` 逻辑限制 consumer 必须是本机 consumer。
- 弹窗关闭时保留当前已选 pair 摘要，但不得影响 runtime delivery 的现有选择与操作。
- 所有刷新与提交动作继续通过 `withToast(...)` 包装，避免静默失败。

#### Performance and Testing Strategy
- 不新增额外 store、watcher 或协议请求类型，只改变 UI 承载位置。
- 保留按需刷新 catalog 的方式，避免页面初始渲染挂载两个大列表。
- 验证策略：
  - `frontend`: `npm exec vitest run src/pages/Stream.test.ts`
  - 若需要构建验证且 `frontend/wailsjs/**` 缺失，先执行 `GOWORK=off wails generate module`
  - `frontend`: `npm run build`

#### Extensibility Design Points
- 配对弹窗如果后续还要支持更多 control 动作，可继续复用 `mode` 驱动，而不再回到主页面常驻双目录布局。
- 若未来需要更丰富筛选条件，可继续扩展弹窗内 query 区，不影响主页面结构。

#### Issue List
- 当前无需升级 requirements/specs；本轮属于既有 Stream 能力的前端交互重排。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 `Stream` 已完成产品化 tab 改造，但 `Control` 页仍保留两个常驻 catalog 面板，主页面密度偏高。
- 用户明确要求：
  - 移除 `Source Catalog`
  - 移除 `Consumer Catalog`
  - `Control Target` 使用弹出式选择框完成连接，而不是依赖常驻 catalog
- 当前代码已具备所需底层能力：
  - source / consumer 查询
  - connect / subscribe 调用
  - `Overlay` 弹窗模式
  - `Runtime Deliveries` 查看与操作
- 本轮目标是收敛 control 页交互，而不是改协议或 store contract。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Canonical destination:
  - 长期行为真相 -> `docs/requirements/stream.md`
  - 长期技术契约 -> `docs/specs/stream.md`
  - 执行控制面 -> worktree 根 `plan.md`
  - 完成结果 -> `docs/change/2026-04-09_win-stream-control-target-picker.md`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `docs/requirements/stream.md`
- Related specs:
  - `docs/specs/stream.md`
- Related lessons:
  - `docs/lessons/frontend-build-empty-node-modules.md`
  - `docs/lessons/wails-binding-proto-drift.md`

#### Executable Task List
- [x] `SCTP-1` 重构 `Stream.vue` 的 control 页面与配对弹窗状态
- [x] `SCTP-2` 更新 control 相关文案与交互提示
- [x] `SCTP-3` 更新页面测试并完成定向验证
- [x] `SCTP-4` 完成 3.3 review 与 Stage 4 归档

#### Task Details
##### `SCTP-1` - 重构 control 页面与配对弹窗
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker\plan.md`
- Goal:
  - 移除 control 主页面常驻 `Source Catalog` / `Consumer Catalog`
  - 新增弹窗式 pair picker，并保持 `Control Target` 与 `Runtime Deliveries`
- Files / Modules:
  - `frontend/src/pages/Stream.vue`
- Write Set:
  - `frontend/src/pages/Stream.vue`
- Acceptance:
  - `Control` 页不再渲染两块 catalog
  - 用户可从 control 动作入口打开配对弹窗并完成连接
  - `Runtime Deliveries` 不回退
- Test Points:
  - 配对弹窗能打开
  - 查询 source / consumer 后能选择 pair
  - `connect` 或 `subscribe` 入口按设计工作
- Rollback:
  - 回退 `frontend/src/pages/Stream.vue`

##### `SCTP-2` - 更新文案与交互提示
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker\plan.md`
- Goal:
  - 保持 control 页与弹窗文案自洽，移除过期 catalog 描述
- Files / Modules:
  - `frontend/src/i18n/messages/stores.ts`
  - `frontend/src/pages/Stream.vue`
- Write Set:
  - `frontend/src/i18n/messages/stores.ts`
  - `frontend/src/pages/Stream.vue`
- Acceptance:
  - 页面和弹窗不再出现误导性的常驻 catalog 描述
  - 新的按钮、标题、说明和空态文案可翻译且风格一致
- Test Points:
  - 中文环境下 control 页与弹窗文案可见且语义正确
- Rollback:
  - 回退 `frontend/src/i18n/messages/stores.ts`
  - 回退 `frontend/src/pages/Stream.vue`

##### `SCTP-3` - 更新页面测试与验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker\plan.md`
- Goal:
  - 让自动化测试覆盖新的 control 交互并验证未回退的 delivery 入口
- Files / Modules:
  - `frontend/src/pages/Stream.test.ts`
- Write Set:
  - `frontend/src/pages/Stream.test.ts`
- Acceptance:
  - 测试覆盖 control 页不再显示 catalog
  - 测试覆盖 control 弹窗连接路径
  - delivery output window 测试继续通过
- Test Points:
  - `npm exec vitest run src/pages/Stream.test.ts`
  - `npm run build`
- Rollback:
  - 回退 `frontend/src/pages/Stream.test.ts`

##### `SCTP-4` - Review 与归档
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-stream-control-target-picker\plan.md`
- Goal:
  - 完成 3.3 review、Stage 4 `docs/change` 归档与影响记录
- Files / Modules:
  - `docs/change/2026-04-09_win-stream-control-target-picker.md`
  - `docs/change/README.md`（如需）
- Write Set:
  - `docs/change/2026-04-09_win-stream-control-target-picker.md`
  - `docs/change/README.md`（仅在索引需要时）
- Acceptance:
  - 归档中明确记录 requirements/specs/lessons impact
  - 包含任务映射、验证结果、回滚方案和子Agent轨迹
- Test Points:
  - 归档内容与实际改动一致
- Rollback:
  - 删除本轮新增 `docs/change` 条目并恢复索引

#### Dependencies
- `frontend` 依赖已安装；若不存在需要先执行 `npm ci`
- 若构建校验需要 Wails 生成物，则依赖 `GOWORK=off wails generate module`
- 不依赖 Proto / Server / SDK 的代码改动

#### Risks and Notes
- 这是纯前端交互收敛，设计上应优先选择最小安全改动，不把 store selection 或 query 模型扩散到更多模块。
- 如实际编码中发现 `subscribe` 需要与 `connect` 完全不同的弹窗状态模型，应先回到 3.1 更新计划后再扩大改动面。
- 当前不使用子Agent；主Agent独立完成实现、验证与归档。

#### Parallelism Assessment
- 评估结果：不适合并行拆分。
- 原因：
  - 改动集中在单页 `Stream.vue`，模板、状态和交互强耦合
  - 文案与测试都依附同一页面重构，拆分会增加冲突与集成成本
- 子Agent：不使用

### Stage 3.2 - Implementation Record
- `SCTP-1`
  - 已将 `Control` 主页面重构为“Control Target + 当前配对摘要 + Runtime Deliveries”结构。
  - 已移除主页面常驻 `Source Catalog` / `Consumer Catalog`，改为 `data-stream-open-control-picker` 打开配对弹窗。
  - 已在配对弹窗内复用现有 source / consumer 查询、选择与 `connect` / `subscribe` 调用链。
- `SCTP-2`
  - 已更新 `frontend/src/i18n/messages/stores.ts`，补充 control picker、配对摘要与提示文案。
- `SCTP-3`
  - 已更新 `frontend/src/pages/Stream.test.ts`，覆盖 control 页不再显示 catalog，以及配对弹窗的 connect 路径。
  - 验证过程中先执行 `frontend/npm ci` 补齐 worktree 缺失依赖，再执行定向 Vitest。
  - `npm run build` 初次失败于既有缺失 `wailsjs/runtime/runtime`，随后按仓库约定执行 `GOWORK=off wails generate module` 后构建通过。

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - `Control` 页已移除常驻 catalog，并通过弹窗完成配对和连接。
- 架构合理性：通过
  - 保持改动收敛在页面、文案和测试；store / binding / protocol 未被扩散修改。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 未新增协议类型或额外监听；仅在打开配对弹窗缺少 catalog 数据时触发补查询。
- 可读性与一致性：通过
  - 继续复用现有 `Overlay`、toast 包装和 store 选择态，交互模式与 Win 其它页面保持一致。
- 可扩展性与配置化：通过
  - 后续若扩更多 control 动作，可继续沿用同一 picker 弹窗。
- 稳定性与安全：通过
  - `kind` 不匹配仍阻止 connect；`subscribe` 仍限制在本机 consumer；runtime delivery 操作区未回退。
- 测试覆盖情况：通过
  - `frontend`: `npm exec vitest run src/pages/Stream.test.ts`
  - `worktree root`: `$env:GOWORK='off'; wails generate module`
  - `frontend`: `npm run build`
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未派发子Agent。

### Stage 4 - Archive Record
- 使用 `$m-docs` 完成 change 路由、requirements/specs impact 复核与索引同步。
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- 相关归档：
  - `docs/change/2026-04-09_win-stream-control-target-picker.md`

#### Issue List
- 无新增阻塞；`3.2` / `3.3` / `4` 已完成，等待用户决定是否结束 workflow。

阻塞：否
3.2 / 3.3 / 4 已完成
等待是否结束 workflow
