# Plan - MyFlowHub-Win DAG 画布编辑器增强（PR2）

> 工作流目录：`d:\project\MyFlowHub3\worktrees\win-dag-editor-ux\`
> 目标仓库工作区：`d:\project\MyFlowHub3\worktrees\win-dag-editor-ux\MyFlowHub-Win\`（分支：`feat/win-dag-editor-ux`）
>
> 约束（已确认）：
> - ✅ 不修改 HeaderTcp 头部（字段/长度/路由语义不变）
> - ✅ 允许修改 payload（JSON data），继续使用 `node.spec._ui` 承载 UI 布局
> - ✅ `node.id` 创建后不可修改
> - ✅ Status overlay 先采用“点击 Status 刷新”的模式（不做定时轮询）
> - ✅ Auto layout 默认方向：TB（从上到下）
> - ✅ Undo/Redo 覆盖：结构编辑 + 节点移动 + auto layout + Node Detail 参数编辑（合并策略见 W4）

---

## 1. 项目目标与当前状态

### 1.1 目标（PR2 交付）
在 PR1 已具备“拖拽/连线/布局持久化”的基础上，补齐编辑器的核心体验：

1) **自动布局（Auto Layout）**：一键将当前 DAG 按 TB 方向排布，并写回 `spec._ui.{x,y}`。
2) **撤销/重做（Undo/Redo）**：覆盖
   - 增删节点/边
   - 节点移动（drag stop）
   - auto layout
   - Node Detail 参数编辑（method/args/timeout/retry/allow_fail/kind/target）
3) **快捷键**：
   - Delete：删除选中节点/边（优先边）
   - Ctrl+Z / Ctrl+Y（或 Ctrl+Shift+Z）：撤销/重做
   - Ctrl+S：保存（触发 `flow.set`）
4) **画布辅助组件**：
   - MiniMap
   - Zoom controls（含 Fit view）
   - Background（网格/点阵）
5) **运行状态 Overlay**：将 `flow.status` 的节点状态渲染到画布节点上（颜色/徽标/文本）。

### 1.2 当前事实（已存在）
- PR1 已落地：
  - `frontend/src/components/flow/FlowCanvas.vue`
  - `frontend/src/pages/Flow.vue`（画布主导，`node.id` 只读）
  - `frontend/src/stores/flow.ts`（`spec._ui` 持久化、成环校验）

---

## 2. 任务清单（Checklist）

### W1 - 基线确认
- 目标：确保在独占 worktree 内实现与验证。
- 验收：
  - 分支为 `feat/win-dag-editor-ux`，且工作区干净。
  - `GOWORK=off go test ./...` 可通过（不要求新增测试）。
- 回滚：删除 worktree + 分支。

### W2 - 引入 Vue Flow 扩展组件依赖
- 目标：为 MiniMap/Controls/Background 提供官方组件。
- 依赖（预期）：
  - `@vue-flow/minimap`
  - `@vue-flow/controls`
  - `@vue-flow/background`
- 涉及文件：
  - `frontend/package.json`
  - `frontend/package-lock.json`
- 验收：
  - `wails build -nopackage` 可通过（Wails 会生成 `frontend/wailsjs` 并驱动 `vite build`）。
- 回滚：回退依赖变更（package.json + lockfile）。

### W3 - 画布组件增强（MiniMap/Controls/Background + Status Node）
- 目标：
  1) 在画布中加入 MiniMap/Controls/Background（默认开启）。
  2) 引入自定义节点渲染（nodeTypes）以支持状态 Overlay（成功/失败/未知）。
- 设计要点：
  - Overlay 不改变 wire，仅消费 `flowStore.state.lastStatus`。
  - 节点 UI 可读性优先：默认显示 nodeId + 状态 badge（可选显示 code/msg 摘要）。
- 涉及文件：
  - `frontend/src/components/flow/FlowCanvas.vue`
  - 新增：`frontend/src/components/flow/FlowNode.vue`（自定义节点渲染）
- 验收：
  - Status 返回后，节点样式随状态变化。
  - MiniMap/Controls/Background 正常显示与交互。
- 回滚：恢复默认节点渲染与移除扩展组件。

### W4 - Store：Undo/Redo + Auto Layout + Node Detail 变更合并
- 目标：
  1) 新增历史栈：可 Undo/Redo，且与画布/右侧面板一致。
  2) Auto layout：TB 排布，更新所有节点的 `x/y`。
  3) Node Detail 编辑纳入历史：对输入频繁的字段采用“合并提交”策略，避免每个字符一个快照。
- 合并策略（建议实现）：
  - 结构性操作（增删节点/边、auto layout、drag stop）：操作完成后立即写入历史。
  - Node Detail：
    - checkbox/select：变更即提交
    - method/args/number：在 `blur` 时提交（首版），并在后续可演进为 debounce/idle。
- 涉及文件：
  - `frontend/src/stores/flow.ts`
- 验收：
  - Undo/Redo 可正确回退/重做：
    - 新增边/删除边
    - 移动节点
    - 修改 method/args 后 blur 再 undo 可回退
    - auto layout 后 undo 回到原布局
  - Auto layout 不生成环、且对已有环（若出现）给出错误提示并不改动布局。
- 回滚：
  - 回退 store 历史与布局算法变更。

### W5 - 页面：按钮与快捷键接入
- 目标：
  - 提供 UI 按钮：Auto Layout / Undo / Redo
  - 接入快捷键：Delete / Ctrl+Z/Y / Ctrl+S
  - 在 Node Detail 输入控件上接入 blur/change 提交历史
- 涉及文件：
  - `frontend/src/pages/Flow.vue`
- 验收：
  - 快捷键生效且不误伤输入框常规行为（Delete/Undo 在输入框中不触发全局删除/撤销）。
  - Ctrl+S 能触发保存并阻止默认浏览器行为。
- 回滚：恢复 PR1 的页面交互。

### W6 - 验证与冒烟
- 命令：
  - `cd MyFlowHub-Win && GOWORK=off go test ./...`
  - `cd MyFlowHub-Win && GOWORK=off wails build -nopackage`
- 手工冒烟：
  1) New → Add Node x2 → 连线 → Auto layout → Undo/Redo
  2) 编辑 Node Detail（method/args）→ blur → Undo/Redo
  3) 点击 Status → 画布节点显示 succeeded/failed
  4) Delete 删除选中边/节点；Ctrl+S 保存
- 回滚点：回退该分支提交即可。

---

## 3. 风险与注意事项
- Undo/Redo 的快照对象必须稳定、可序列化；避免混入 Vue 响应式 proxy。
- Node Detail 的 args 为 JSON 文本，编辑体验与撤销边界需明确（本 PR 采用 blur 提交，避免字符级快照）。
- Auto layout 首版采用“稳定可用”优先（不追求最优交叉最少），避免引入重依赖。

---

## 4. 执行记录（已完成）

- 完成日期：2026-02-21
- 代码仓库：`MyFlowHub-Win`（分支：`feat/win-dag-editor-ux`）
- 关键提交：`14df58d`
- 验证：
  - `GOWORK=off go test ./...` ✅
  - `GOWORK=off wails build -nopackage` ✅（生成 `build/bin/myflowhub-win.exe`）
