# 变更归档：Win Flow DAG 画布编辑器（PR1）

日期：2026-02-21  
范围：`MyFlowHub-Win`（分支：`feat/win-dag-editor`）  
工作流根：`d:\project\MyFlowHub3\worktrees\win-dag-editor\`（见 `plan.md`）

---

## 1. 背景 / 目标

现有 Win 的 Flow 编辑为“列表/表单”模式（Nodes/Edges 列表 + 右侧参数），创建边需要对话框选择 from/to，编辑效率较低。  
本 PR1 目标是在不改变 HeaderTcp 头部、不改变协议 action/wire 语义的前提下，引入 **图形化 DAG 画布**：

- 节点可拖拽移动位置
- 拖拽连线创建边（from→to）
- 删除节点/边
- 保存/加载时布局可跨端共享（布局进入 payload，但不改头部）

约束（已确认）：
- ✅ 只允许修改 payload（JSON data）；**禁止修改 HeaderTcp 头部**
- ✅ `node.id` 作为图结构主键：创建后不可修改
- ✅ 连线时即时阻止成环（新增边不得导致 cycle）

---

## 2. 具体变更

### 2.1 前端依赖
- 新增依赖：`@vue-flow/core`（Vue3 节点/连线画布组件）

### 2.2 UI：Flow 页面改为“画布主导”
- 将 `Flow.vue` 的 Nodes/Edges 列表编辑模式替换为画布：
  - 节点拖拽移动
  - 拖拽连线创建边
  - “Remove Node / Remove Edge” 按钮基于当前选择删除
- Node Detail 中 `Node ID` 改为只读展示（明确“不可修改”）

### 2.3 Store：布局读写 + 成环校验（SSOT）
- `FlowNodeDraft` 增加 `x/y` 坐标（仅 UI 状态）
- `spec._ui`（payload 扩展字段）：
  - 保存时写入：`spec._ui = { x, y }`（仅 payload；不会改 HeaderTcp）
  - 加载时读取：优先使用 `_ui.x/y`；缺省时按网格生成稳定默认位置
- `addEdge(from,to)` 增强校验：
  - 禁止重复边
  - 禁止成环（新增边前做可达性判断：若 `to` 可到达 `from`，则拒绝）

### 2.4 Wire 兼容性说明
- 头部（HeaderTcp）无改动
- Flow payload 的 `node.spec` 增加保留字段 `_ui`：
  - Server/Executor 执行时只反序列化 `method/target/args`，未知字段（如 `_ui`）会被 Go `encoding/json` 忽略，不影响执行
  - `_ui` 会随 `flow.set` 落盘到 executor 的 `./flows/<flow_id>.json`，因此布局可跨端共享

---

## 3. 对应计划（plan.md 映射）

见 `worktrees/win-dag-editor/plan.md`：
- W2：引入 Vue Flow ✅
- W3：新增 `FlowCanvas` ✅
- W4：store 布局读写/成环校验 ✅
- W5：Flow 页面画布化 ✅
- W6：验证与冒烟 ✅（见下）

---

## 4. 关键设计决策与权衡

1) **布局进入 payload，但不改头部**  
   - 选择把坐标写进 `spec._ui`，避免新增协议字段/改 Header，且对执行端天然兼容（未知字段忽略）。
   - 代价：布局会随网络传播并被 executor 落盘，需要控制字段体积与命名（统一 `_ui`）。

2) **`node.id` 不允许修改**  
   - 画布编辑里，`node.id` 同时是边引用键与节点 key；允许随意改名会导致边断裂/一致性复杂化。
   - 本 PR 明确为只读，后续若确有需求，可新增“显式 Rename（迁移 edges）”能力另做 workflow。

3) **成环校验在 store 做 SSOT**  
   - 画布只负责交互事件，所有结构约束在 store 内统一校验，避免出现“UI 允许但保存失败”的漂移。

性能关键点：
- 节点坐标只在 `nodeDragStop` 写回（避免拖拽过程高频写入与重渲染）
- 成环校验仅在新增边时执行一次 BFS（规模为几十节点/百边时开销可接受）

---

## 5. 测试与验证

### 5.1 命令验证（已执行）
在 `d:\project\MyFlowHub3\worktrees\win-dag-editor\MyFlowHub-Win`：
- `GOWORK=off go test ./...` ✅
- `GOWORK=off wails build -nopackage` ✅

### 5.2 手工冒烟（建议）
1) Win 连接到 server（已有 Session 页/Auto-connect 流程）
2) Flow 页面：
   - New → Add Node（local）→ Add Node（exec）→ 拖拽移动 → 拖拽连线 → Save
   - Refresh/重新选择同一 flow → 确认节点位置仍保持（`spec._ui` 生效）
3) 尝试创建环（A→B 后再 B→A）：应被拒绝（UI 显示错误信息或无法建立边）

---

## 6. 潜在影响与回滚方案

影响：
- 新增 `spec._ui` 字段会出现在 executor 落盘的 flow JSON 中；旧执行逻辑忽略该字段，不影响运行。
- Win Flow 编辑 UI 交互方式变化（从列表/对话框建边 → 画布拖拽连线）。

回滚：
- 回滚本分支提交即可恢复旧 UI（无需迁移数据）。
- 即使已有 flow 文件包含 `_ui`，回滚后仍可正常执行；旧 UI 会忽略该字段。

