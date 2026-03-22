# Win Flow 编辑器文案精简

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：refactor/win-editor-chrome-trim
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\refactor-win-editor-chrome-trim\MyFlowHub-Win
- 当前阶段：4 归档变更

## 1. 需求分析

### 目标
- 精简 Win Flow 独立编辑窗口的界面文案，移除不必要的说明性文字，让编辑界面更干净。

### 范围
- 必须：
  - 移除编辑窗口顶部 `project_id ...` 文案。
  - 移除编辑窗口顶部 `Pure workflow editing only...` 文案。
  - 移除画布上方两段操作说明文案：
    - `Drag nodes to reposition...`
    - `Click blank canvas to close the drawer.`
- 可选：
  - 维持现有标题、按钮工具栏、画布和右侧抽屉行为不变。
- 不做：
  - 不修改 Flow 项目中心列表页。
  - 不调整节点编辑字段、部署逻辑或能力查询逻辑。
  - 不改变画布、抽屉、按钮的交互方式。

### 使用场景
- 用户打开 `FlowEditorWindow` 编辑本地 workflow 时，希望第一屏直接看到更纯粹的编辑区域，而不是额外说明文字。

### 功能需求
- 编辑窗口去掉上述静态说明文本后，页面仍能正常加载、保存、连线、打开右侧详情抽屉。

### 非功能需求
- 改动最小化。
- 不引入布局抖动或额外滚动。
- 不影响既有快捷键和组件层级。

### 输入输出
- 输入：打开某个 Flow 项目的独立编辑窗口。
- 输出：页面不再显示上述说明性文字，保留原编辑能力。

### 边界异常
- `loadedProjectName` 仍可能为空，需继续保留 `Untitled Project` 兜底标题。
- 顶部按钮区仍需保留，否则会影响编辑操作可达性。

### 验收标准
- `FlowEditorWindow.vue` 中不再渲染上述 4 段说明文案和 `project_id` 行。
- `npm run build` 通过。

### 风险
- 若直接删除对应容器结构不当，可能导致 header/画布间距变化过大，需要保留合理骨架。

### 问题清单
- 阻塞：否

## 2. 架构设计（分析）

### 总体方案
- 只改 `frontend/src/windows/FlowEditorWindow.vue` 的模板层。
- 保留 header、工具栏、画布容器、抽屉容器，只删除说明性文本节点和其无必要包裹容器。

### 模块职责
- `frontend/src/windows/FlowEditorWindow.vue`
  - 负责 Flow 独立编辑窗口的 header、画布、节点详情抽屉和对话框布局。

### 数据 / 调用流
- 本次不改 store、不改 props、不改事件流。
- 仅影响模板静态文本渲染结果。

### 接口草案
- 无新增接口。
- 无状态结构调整。

### 错误与安全
- 不改数据读写与 Wails 调用。
- 不改权限、部署、触发器相关逻辑。

### 性能与测试策略
- 性能影响可忽略，仅减少静态 DOM。
- 验证：
  - `cd frontend && npm run build`
  - 代码检查确认目标文本不再存在于模板中。

### 可扩展性设计点
- 若后续还需继续精简编辑器 chrome，可继续在该窗口组件模板层收敛，不影响 store 与业务逻辑。

## 3.1 计划拆分（Checklist）

- [x] `EDITOR-CHROME-1`
  - Owner：主Agent
  - 目标：更新当前 worktree 的 `plan.md`，记录本次需求分析、架构方案和任务拆分。
  - 涉及文件：`plan.md`
  - 验收条件：文档可脱离会话被他人接手。
  - 回滚点：回退 `plan.md`。

- [x] `EDITOR-CHROME-2`
  - Owner：主Agent
  - 目标：移除 Flow 编辑窗口中的说明性静态文案和 `project_id` 展示。
  - 涉及文件：`frontend/src/windows/FlowEditorWindow.vue`
  - 依赖：`EDITOR-CHROME-1`
  - 验收条件：目标文本不再渲染，页面结构保持稳定。
  - 测试点：代码搜索目标文案。
  - 回滚点：回退 `frontend/src/windows/FlowEditorWindow.vue`。

- [x] `EDITOR-CHROME-3`
  - Owner：主Agent
  - 目标：执行构建验证、Code Review 和归档文档。
  - 涉及文件：`plan.md`、`docs/change/*`
  - 依赖：`EDITOR-CHROME-2`
  - 验收条件：`npm run build` 通过，归档文档完整。
  - 回滚点：删除本次新增文档并回退代码改动。

## 3.3 Code Review

### 需求覆盖
- 通过：`FlowEditorWindow.vue` 中已移除 `project_id` 行、顶部说明文案和画布说明文案，符合“界面更简洁”的需求。

### 架构合理性
- 通过：改动只落在窗口组件模板层与局部无用状态清理，没有侵入 store、router、Wails binding 或业务流程。

### 性能风险
- 通过：仅减少静态 DOM，运行时不存在新增计算、I/O 或渲染开销。
- 备注：前端构建仍有既有大 chunk 告警，不属于本次变更引入。

### 可读性与一致性
- 通过：删除的都是明确冗余文案，同时移除了不再使用的 `loadedProjectId` 状态，避免死代码。

### 可扩展性与配置化
- 通过：后续若继续精简编辑器 chrome，仍可在 `FlowEditorWindow.vue` 模板层单点调整。

### 稳定性与安全
- 通过：未修改数据结构、调用参数、权限逻辑或部署逻辑。

### 测试覆盖情况
- 通过：
  - `$env:GOWORK='off'; wails generate module`
  - `cd frontend && npm install`
  - `cd frontend && npm run build`

### 子Agent治理与审计
- 通过：未使用子Agent。
- 原因：单文件模板改动，写集集中且无需并行拆分。

## 并行性评估
- 不使用子Agent。
- 原因：单文件模板收敛，写集高度集中，拆分没有收益且容易引入重复修改。
