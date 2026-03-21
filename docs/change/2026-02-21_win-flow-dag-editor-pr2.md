# 2026-02-21 · MyFlowHub-Win · Flow DAG 编辑器增强（PR2）

## 背景 / 目标

在 PR1 已具备 DAG 画布基础能力（拖拽节点、连线建边、删除节点/边、布局持久化到 `node.spec._ui`、新增边即时阻止成环）的前提下，PR2 目标是补齐日常编辑体验与可观测性：

1) **Auto Layout（TB 从上到下）**：一键将 DAG 自动排布，并写回每个节点的 `x/y`（最终仍落在 payload 的 `spec._ui`）。
2) **Undo/Redo**：覆盖结构编辑、节点拖拽移动、Auto Layout，以及 Node Detail 参数编辑。
3) **快捷键**：Delete（优先删边）、Ctrl+Z / Ctrl+Y（或 Ctrl+Shift+Z）、Ctrl+S 保存。
4) **画布辅助**：MiniMap + Controls + Background。
5) **节点状态 Overlay**：点击 `Status` 刷新后，将节点运行状态渲染到画布节点上。

约束（已确认）：
- 不修改 HeaderTcp 头部（字段/长度/路由语义不变）
- 允许修改 payload（JSON data），继续使用 `node.spec._ui` 承载布局
- `node.id` 禁止修改
- Status 采用“点击刷新”，不做定时轮询

## 具体变更内容

### 新增
- 自定义节点渲染组件：`frontend/src/components/flow/FlowNode.vue`
  - 显示 nodeId + 状态 badge（succeeded/failed/running/queued/unknown）
  - 显示 code/msg 摘要（有状态时）

### 修改
- `frontend/src/components/flow/FlowCanvas.vue`
  - 引入 `MiniMap / Controls / Background`
  - 使用 `nodeTypes` 将节点渲染切换为 `FlowNode`
  - 将 `lastStatus.nodes` 注入到节点 `data`，用于 Overlay 展示

- `frontend/src/stores/flow.ts`
  - 引入 Draft 历史栈：`commitHistory / undo / redo`（最大 120 条）
  - Auto Layout（TB）实现：拓扑分层 + 层内稳定排序 + 居中对齐，写回 `node.x/y`
  - `newDraft` 与 `getFlow` 后重置历史基线（避免跨 Flow 串历史）

- `frontend/src/pages/Flow.vue`
  - 工具栏按钮：Undo / Redo / Auto Layout
  - 快捷键：
    - Delete：删除选中边/节点（优先边）
    - Ctrl+Z / Ctrl+Y（或 Ctrl+Shift+Z）：Undo/Redo
    - Ctrl+S：Save（阻止浏览器默认行为）
  - Node Detail 输入合并策略：select/checkbox 用 change 提交历史；method/args/number 用 blur 提交历史
  - 节点拖拽停止后提交历史（drag stop）

### 依赖
- 前端新增依赖（Vue Flow 官方插件）：
  - `@vue-flow/minimap`
  - `@vue-flow/controls`
  - `@vue-flow/background`

## 对应 plan.md 任务映射
- W2：引入 Vue Flow 扩展组件依赖（已完成）
- W3：画布组件增强 + Status Node（已完成）
- W4：Store：Undo/Redo + Auto Layout + Node Detail 合并提交（已完成）
- W5：页面：按钮与快捷键接入（已完成）
- W6：构建验证与冒烟（已完成，见下）

## 关键设计决策与权衡

1) **Undo/Redo 采用“快照 + 指针”**  
   - 优点：实现简单、与 Vue 响应式解耦（快照全是可序列化对象）、回滚稳定可预期  
   - 代价：大图时快照可能占用更多内存；通过 **只在关键节点提交**（drag stop / blur / 操作完成）+ **最大 120 条** 控制成本

2) **Node Detail 合并策略：blur/change 提交**  
   - 文本/JSON（method/args）不做字符级历史，避免每次输入都产生快照（性能 + 体验）  
   - 选择器/checkbox 变更频率低，直接 change 提交

3) **Auto Layout（TB）首版“稳定可用优先”**  
   - 采用拓扑分层（level）进行排布：`y = level * gap`  
   - 层内稳定排序基于原 node 顺序，避免每次布局抖动  
   - 不追求最小交叉/最优美观，以避免引入重依赖与复杂度

4) **状态 Overlay 仅消费 Status 响应，不改变 wire**  
   - Overlay 只读取 `lastStatus.nodes`，通过点击 `Status` 触发刷新  
   - 保持协议层/头部完全不变

## 测试与验证方式 / 结果

命令验证：
- `GOWORK=off go test ./...` ✅
- `GOWORK=off wails build -nopackage` ✅（生成 `build/bin/myflowhub-win.exe`）

手工冒烟建议步骤：
1) New → Add Node x2 → 连线 → Auto Layout → Undo/Redo
2) 拖拽节点 → 松手 → Undo/Redo
3) 编辑 Node Detail（method/args/timeout/retry/allow_fail/kind/target）→ blur/change → Undo/Redo
4) 点击 Status → 画布节点显示 succeeded/failed/running/queued
5) Delete 删除选中边/节点；Ctrl+S 保存

## 潜在影响与回滚方案

潜在影响：
- Undo/Redo 基于快照，若未来节点数据结构扩展，需确保 `takeSnapshot/applySnapshot` 同步更新字段。
- Auto Layout 目前按拓扑层级排列，复杂 DAG 的边交叉可能仍较多（可后续演进）。

回滚：
- 回退 MyFlowHub-Win 仓库本次提交即可（不涉及协议/头部变更）。

