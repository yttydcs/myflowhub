# Win Flow 编辑器顶部标题移除

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：refactor/win-editor-header-title-trim
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\refactor-win-editor-header-title-trim\MyFlowHub-Win
- 当前阶段：4 归档变更

## 1. 需求分析

### 目标
- 移除 Win Flow 独立编辑窗口顶部的 `Flow Project Editor` 文案，让编辑窗口 header 更简洁。

### 范围
- 必须：
  - 删除 `FlowEditorWindow.vue` header 中的 `Flow Project Editor` 静态标签。
- 可选：
  - 保留项目标题 `loadedProjectName / Untitled Project`。
  - 保留现有工具栏按钮、布局、抽屉、画布与方法选择对话框行为。
- 不做：
  - 不修改 Flow 编辑器其他字段、能力查询、部署逻辑和节点抽屉。
  - 不修改 Flow 项目中心列表页。
  - 不调整任何 store、Wails 接口、路由和状态结构。

### 使用场景
- 用户从项目列表进入独立 Flow 编辑窗口时，希望顶部只显示项目名和操作工具，不再出现额外标题说明。

### 功能需求
- 页面加载后不再显示 `Flow Project Editor` 文案。
- 编辑器现有保存、连线、节点详情抽屉、方法对话框能力保持不变。

### 非功能需求
- 改动最小化，只触达必要模板节点。
- 不引入新的滚动、间距异常或 header 布局抖动。
- 保持代码可读性，不保留无意义结构。

### 输入输出
- 输入：打开任一 Flow 项目的独立编辑窗口。
- 输出：header 仅展示项目标题和工具栏，不再展示顶部说明标签。

### 边界异常
- 项目名可能为空，仍需保留 `Untitled Project` 兜底文案。
- 工具栏必须保持可用，不能因删除标签破坏 header 布局。

### 验收标准
- `frontend/src/windows/FlowEditorWindow.vue` 中不再渲染 `Flow Project Editor`。
- `cd frontend && npm run build` 通过。

### 风险
- 若直接移除节点导致 header 间距变化，需同步收敛标题 `margin-top`，保证视觉对齐。

### 问题清单
- 阻塞：否

## 2. 架构设计（分析）

### 总体方案
- 仅修改 `frontend/src/windows/FlowEditorWindow.vue` 的模板层 header 区域。
- 删除静态 `<p>` 标签后，按需调整 `<h1>` 的顶部间距类名，避免因原副标题消失导致标题下沉。

### 选型理由 / 备选对比
- 方案 A：直接删除 `<p>`，并移除 `<h1>` 的 `mt-1`。
  - 优点：改动最小，不引入额外条件渲染和状态。
  - 缺点：仅适用于该标签永久移除的场景。
- 方案 B：保留 DOM，改为空字符串或 `v-if`。
  - 缺点：保留无意义结构，增加模板噪音，没有工程价值。
- 结论：采用方案 A。

### 模块职责
- `frontend/src/windows/FlowEditorWindow.vue`
  - 负责 Flow 独立编辑窗口的 header、画布、右侧抽屉和弹窗布局。
  - 本次只调整 header 左侧标题区的静态文本结构。

### 数据 / 调用流
- 本次不改任何响应式状态、store、事件、Wails 调用。
- 仅影响静态 DOM 输出。

### 接口草案
- 无新增接口。
- 无状态字段变更。

### 错误与安全
- 不涉及输入校验、鉴权、部署或能力查询逻辑变更。
- 不修改用户可写数据，安全风险可忽略。

### 性能与测试策略
- 性能：减少一个静态文本节点，运行时性能影响可忽略。
- 测试：
  - 静态检查目标文本已移除。
  - 执行 `cd frontend && npm run build` 验证 SFC 与构建链路。

### 可扩展性设计点
- 后续若继续极简化编辑器 header，应继续在该组件模板层处理，避免扩散到 store 或路由层。

## 3.1 计划拆分（Checklist）

- [x] `FLOW-HEADER-1`
  - Owner：主Agent
  - Worktree：D:\project\MyFlowHub3\worktrees\refactor-win-editor-header-title-trim\MyFlowHub-Win
  - Plan：D:\project\MyFlowHub3\worktrees\refactor-win-editor-header-title-trim\MyFlowHub-Win\plan.md
  - 目标：完成本轮需求分析、架构设计与任务拆分文档更新。
  - Write set：`plan.md`
  - 验收条件：文档可被接手者独立执行。
  - 测试点：核对 workflow 信息、阶段、任务 ID 与本轮目标一致。
  - 回滚点：回退 `plan.md`。
  - 关键上下文：用户要求仅移除 Flow 编辑窗口顶部 `Flow Project Editor` 文案。

- [x] `FLOW-HEADER-2`
  - Owner：主Agent
  - Worktree：D:\project\MyFlowHub3\worktrees\refactor-win-editor-header-title-trim\MyFlowHub-Win
  - Plan：D:\project\MyFlowHub3\worktrees\refactor-win-editor-header-title-trim\MyFlowHub-Win\plan.md
  - 目标：修改 `FlowEditorWindow.vue`，移除顶部标签并保持 header 对齐。
  - Write set：`frontend/src/windows/FlowEditorWindow.vue`
  - 验收条件：页面不再渲染 `Flow Project Editor`，项目标题和工具栏布局正常。
  - 测试点：代码搜索不再包含目标文案；模板结构完整。
  - 回滚点：回退 `frontend/src/windows/FlowEditorWindow.vue`。
  - 依赖：`FLOW-HEADER-1`
  - 关键上下文：只允许修改 header 左侧标题区，不得改动 store、交互或其他文案。

- [x] `FLOW-HEADER-3`
  - Owner：主Agent
  - Worktree：D:\project\MyFlowHub3\worktrees\refactor-win-editor-header-title-trim\MyFlowHub-Win
  - Plan：D:\project\MyFlowHub3\worktrees\refactor-win-editor-header-title-trim\MyFlowHub-Win\plan.md
  - 目标：执行构建验证、完成 Code Review、写入归档文档。
  - Write set：`plan.md`、`docs/change/2026-03-21_win-editor-header-title-trim.md`
  - 验收条件：`npm run build` 通过，Review 结论完整，归档文档可审计。
  - 测试点：记录构建结果和风险说明。
  - 回滚点：删除新增归档文档并回退计划文档中的 review/归档更新。
  - 依赖：`FLOW-HEADER-2`
  - 关键上下文：不引入计划外变更。

## 并行性评估
- 不使用子Agent。
- 原因：本轮仅单个 Vue 文件模板改动和文档更新，写集高度集中，不存在安全可拆分的并行任务。

## 3.3 Code Review

### 需求覆盖
- 通过：`FlowEditorWindow.vue` header 中的 `Flow Project Editor` 已移除，项目标题和工具栏仍保留，符合“只去掉顶部说明标题”的需求边界。

### 架构合理性
- 通过：改动仅发生在窗口组件模板层，没有侵入 store、Wails 接口、路由或业务流程。

### 性能风险
- 通过：仅减少一个静态 DOM 节点，没有新增计算、I/O、重复渲染或状态同步成本。
- 备注：构建期仍有既有 chunk size warning，不属于本次改动引入。

### 可读性与一致性
- 通过：删除无意义副标题后，同步清理 `<h1>` 的 `mt-1`，模板结构更直接，视觉语义一致。

### 可扩展性与配置化
- 通过：后续若继续精简 header，可继续在 `FlowEditorWindow.vue` 模板层单点调整，不会影响业务层。

### 稳定性与安全
- 通过：未修改任何用户输入、权限、部署、能力查询或持久化逻辑。

### 测试覆盖情况
- 通过：
  - `$env:GOWORK='off'; wails generate module`
  - `cd frontend && npm install`
  - `cd frontend && npm run build`
  - `rg -n "Flow Project Editor" frontend/src/windows/FlowEditorWindow.vue`

### 子Agent治理与审计
- 通过：未使用子Agent。
- 原因：单文件模板修改，无独立可并行验收任务，拆分只会增加协调成本。

## 4. 归档变更
- 已新增 `docs/change/2026-03-21_win-editor-header-title-trim.md`。
- 归档内容包含任务映射、关键设计决策、验证结果、潜在影响与回滚方案。
