# Plan - MyFlowHub-Win DAG 画布编辑器（PR1）

> 工作流目录：`d:\project\MyFlowHub3\worktrees\win-dag-editor\`
> 目标仓库工作区：`d:\project\MyFlowHub3\worktrees\win-dag-editor\MyFlowHub-Win\`（分支：`feat/win-dag-editor`）
>
> 说明：
> - 本 plan 仅覆盖 **PR1（最小可用）**：引入 DAG 画布（拖拽节点 + 拖拽连线）并可保存/加载布局。
> - 不在本 PR 做：auto layout、undo/redo、快捷键、mini-map、运行状态 overlay（这些进入 PR2 单独 workflow）。
> - 约束已确认：**不修改 HeaderTcp 头部**；允许对 **payload(JSON data)** 做兼容扩展；`node.id` 创建后不可修改；连线时**即时阻止成环**。

---

## 1. 项目目标与当前状态

### 1.1 目标（PR1 交付）
1) Win 的 Flow 编辑页从“列表/表单”升级为“图形化 DAG 画布”：
   - 节点可拖拽移动位置。
   - 通过拖拽连线创建边（from→to），支持删除边/删除节点（节点删除需清理相关边）。
2) 布局可跨端共享：把节点坐标写入 `flow.Node.Spec` 扩展字段 `spec._ui`（仅 payload 变化，不影响执行）。
3) 结构约束：
   - 禁止自环、禁止引用不存在节点、禁止重复边。
   - **即时阻止成环**（连线时校验，非法连接直接拒绝）。
4) 保持现有协议与调用路径：
   - 仍通过 Wails `go.flow.FlowService` 调用 `ListSimple/GetSimple/SetSimple/RunSimple/StatusSimple`。
   - 不改 HeaderTcp、不改 SubProto/action 名称、不改 Server 行为。

### 1.2 当前事实（已存在）
- 页面：`MyFlowHub-Win/frontend/src/pages/Flow.vue`（当前为 Nodes/Edges 列表 + 右侧表单编辑）。
- 状态：`MyFlowHub-Win/frontend/src/stores/flow.ts`（负责 flow CRUD + buildGraph/buildSpec）。
- 后端：`MyFlowHub-Win/internal/services/flow/service.go`（payload JSON envelope + SendCommandAndAwait）。
- 协议：`MyFlowHub-Proto/protocol/flow/types.go`（`Node.Spec` 为 `json.RawMessage`，可兼容扩展字段）。

---

## 2. 任务清单（Checklist）

> 规则：每个任务必须可独立验收；不引入计划外改动。若发现必须新增任务，需先更新本 plan 并再确认。

### W1 - 工作区准备与基线确认
- 目标：确保实现改动只发生在独占 worktree 中，并确认基线可构建。
- 涉及：
  - Worktree：`worktrees/win-dag-editor/MyFlowHub-Win`（已创建）
- 验收：
  - `git status` 干净；基线分支为 `feat/win-dag-editor`，跟踪 `origin/main`。
  - `frontend` 现有 `npm run build` 可运行（若环境允许）。
- 回滚：
  - 删除 worktree + 分支。

### W2 - 引入画布组件库（Vue Flow）
- 目标：为 Vue3 提供成熟的“节点/连线”画布能力。
- 方案：引入 `@vue-flow/core`（后续 PR2 可再按需引入 controls/minimap/background）。
- 涉及文件：
  - `MyFlowHub-Win/frontend/package.json`
  - `MyFlowHub-Win/frontend/package-lock.json`
- 验收：
  - `npm install` 后 lockfile 更新。
  - `npm run build` 通过。
- 回滚：
  - 回退依赖变更（package.json + lockfile）。

### W3 - 新增 `FlowCanvas` 组件（画布渲染与交互）
- 目标：把 store 的 nodes/edges 映射为画布节点/连线，并处理交互事件。
- 设计要点：
  - NodeId 作为画布 node key（与协议 `node.id` 对齐），**不可修改**。
  - 位置来源：`node._ui.x/y`（在 store 中表现为 `node.x/y`）。
  - 拖拽结束（drag stop）才写回位置，避免高频写入。
  - 连线创建时调用“即时成环校验”，非法连接拒绝。
- 涉及文件（新增/修改）：
  - 新增：`MyFlowHub-Win/frontend/src/components/flow/FlowCanvas.vue`
  - 可能新增：`MyFlowHub-Win/frontend/src/components/flow/types.ts`（仅当需要拆分类型）
- 验收：
  - 能渲染已有 flow 的 DAG（节点位置可见，连线正确）。
  - 节点可拖动，拖动后 store 中坐标更新。
  - 拖拽连线创建边成功；成环连接被即时拒绝（UI 给出提示）。
  - 删除节点/边后 store 同步更新。
- 回滚：
  - 删除新增组件并恢复旧页面。

### W4 - 扩展 `flow` store：布局读写、成环检测、边去重
- 目标：让 store 成为唯一真相（SSOT），画布只做映射；并把布局写入 `spec._ui`。
- 关键改动：
  1) `FlowNodeDraft` 增加 `x/y`（数值，单位为画布坐标系）。
  2) `parseSpec` 支持解析 `spec._ui.x/y`（缺省则给稳定的网格默认值）。
  3) `buildSpec` 写回 `_ui: { x, y }`（保留既有 `method/args/target`）。
  4) `addEdge` 额外校验：
     - 禁止重复边（from,to 相同视为重复）。
     - 禁止成环（新增边前做可达性判断：`to` 是否能到达 `from`）。
  5) 新增 API：`selectNodeById`、`removeNodeById`、`removeEdgeByEndpoints`、`setNodePosition` 等（供画布调用）。
- 涉及文件：
  - `MyFlowHub-Win/frontend/src/stores/flow.ts`
- 验收：
  - 加载旧 flow（无 `_ui`）仍可渲染并可保存（保存后 `spec._ui` 出现在 payload 中）。
  - 保存后的 flow 重新加载，布局不丢。
  - 所有边/图校验错误能在 UI 显示明确报错（至少包含原因文本）。
- 回滚：
  - 回退 store 变更（恢复旧 spec 读写逻辑）。

### W5 - 更新 Flow 页面布局：以画布为主
- 目标：弱化/移除 Nodes/Edges 列表，改为“左列表 + 中画布 + 右详情/状态”。
- 关键改动：
  - 引入 `FlowCanvas`，并将节点/边选择改为由画布驱动。
  - Node Detail 中 `node.id` 改为只读展示（禁止编辑）。
  - 保留 “Add Node” 对话框；“Add Edge” 对话框可移除（连线改为画布拖拽）。
- 涉及文件：
  - `MyFlowHub-Win/frontend/src/pages/Flow.vue`
- 验收：
  - 页面功能完整：Refresh/New/Save/Run/Status 仍可用。
  - Add Node 后立即出现在画布，默认位置不与已有节点完全重叠（网格/偏移策略）。
  - 右侧编辑能修改 method/args/timeout/retry 等并可保存生效。
- 回滚：
  - 恢复旧 Flow.vue。

### W6 - 验证与冒烟（本地）
- 目标：给出可执行的验证步骤（覆盖关键路径与边界）。
- 验证命令（建议）：
  - 前端：`cd MyFlowHub-Win/frontend && npm run build`
  - Go：`cd MyFlowHub-Win && go test ./...`
- 手工冒烟：
  1) 连接到 server（或已有连接配置）。
  2) Flow 页面：New → Add Node x2 → 拖拽移动 → 拖拽连线 → Save → Refresh → 重新加载，确认布局仍在。
  3) 尝试创建环（A→B 后再 B→A）：应被即时拒绝并提示原因。
- 回滚点：
  - 仅为 UI/前端变更；回退该分支提交即可。

---

## 3. 风险与注意事项
- 布局字段写入 payload：会被 server 落盘到 `./flows/<flow_id>.json` 并随网络传播；必须控制字段体积与命名（统一用 `spec._ui`）。
- 画布交互复杂度：PR1 只做最小可用，避免把 undo/redo/auto-layout 一次性塞进同一 PR。

