# Win Canvas Drag State Plan

## 项目目标与当前状态
- 目标：
  - 修复 Flow 节点从 `Out` 拖到另一节点时没有进入可连线状态的问题。
  - 保持上一轮已完成的头尾视觉增强与非阻断式详情抽屉。
- 当前状态：
  - `FlowNode.vue` 当前只声明了 `id/data/selected`，未按 Vue Flow 官方 custom node 方式接收 `NodeProps`。
  - `Handle` 当前未显式绑定 `connectable`。
  - `FlowCanvas.vue` 只有全局 `nodes-connectable`，节点对象本身未显式带 `connectable`。

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`fix/win-canvas-drag-state`
- Base：`main`
- Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-drag-state/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-drag-state/MyFlowHub-Win/plan.md`
- 当前阶段：`4`
- 子Agent策略：本轮不派发
  - 原因：当前环境未获得显式子Agent授权；且改动集中在同一条 Vue Flow custom node 交互链路。

## 需求与架构摘要
- 修复重点不是布局，而是 custom node 对 Vue Flow 交互 props 的接收与透传。
- `FlowNode.vue` 按官方 `NodeProps` 方式实现，`Handle` 显式绑定 `connectable`。
- `FlowCanvas.vue` 对节点对象补 `connectable: true`，与全局配置形成双保险。

## 可执行任务清单
- [x] `CANVAS-DRAG-1` 修复 FlowNode custom node props 与 Handle connectable 透传
- [x] `CANVAS-DRAG-2` 在 FlowCanvas 节点模型上补节点级 connectable 并复核连线链路
- [x] `CANVAS-DRAG-3` 验证、Code Review 与归档

## Task 详情

### `CANVAS-DRAG-1`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-drag-state/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-drag-state/MyFlowHub-Win/plan.md`
- 目标：
  - 将 `FlowNode.vue` 改为官方推荐的 `NodeProps` custom node
  - 显式把 `connectable` 传给 `Handle`
- 涉及模块 / 文件：
  - `frontend/src/components/flow/FlowNode.vue`
- Write set：
  - `frontend/src/components/flow/FlowNode.vue`
- 验收条件：
  - 自定义节点具备 Vue Flow 的完整交互 props
  - `Handle` 不再处于“只有视觉没有交互状态”的不完整实现
- 测试点：
  - 代码层确认 `NodeProps` / `connectable` 已接入
- 回滚点：
  - 回退 `frontend/src/components/flow/FlowNode.vue`

### `CANVAS-DRAG-2`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-drag-state/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-drag-state/MyFlowHub-Win/plan.md`
- 目标：
  - 在 `canvasNodes` 上显式设置节点级 `connectable`
  - 复核 `onConnect -> addEdge(...)` 链路不变
- 涉及模块 / 文件：
  - `frontend/src/components/flow/FlowCanvas.vue`
- Write set：
  - `frontend/src/components/flow/FlowCanvas.vue`
- 验收条件：
  - `canvasNodes` 节点对象具备明确的 connectable 状态
  - 连线事件链路保持现有逻辑
- 测试点：
  - 代码层确认 `connectable` 已补齐
  - `isValidConnection` 仍处理重复边/自环/未知节点
- 回滚点：
  - 回退 `frontend/src/components/flow/FlowCanvas.vue`

### `CANVAS-DRAG-3`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-drag-state/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-drag-state/MyFlowHub-Win/plan.md`
- 目标：
  - 完成验证、3.3 Code Review、4 归档
- 涉及模块 / 文件：
  - `plan.md`
  - `docs/change/`
- Write set：
  - `plan.md`
  - `docs/change/2026-03-21_win-canvas-drag-state.md`
- 验收条件：
  - 记录验证结果和未完成验证的环境原因
  - 生成 Code Review 与归档文档
- 测试点：
  - `git diff --check`
  - 如环境允许则做前端依赖安装与浏览器冒烟
- 回滚点：
  - 删除本次新增文档并回退前述代码文件

## 依赖关系
- `CANVAS-DRAG-1` 与 `CANVAS-DRAG-2` 都属于同一条连线交互链路，需要一起验证。
- `CANVAS-DRAG-3` 依赖前两项完成。

## 执行结果
- `CANVAS-DRAG-1`
  - 已完成。`FlowNode.vue` 已改为 `NodeProps<FlowNodeData>`，并显式透传 `connectable` 与左右 handle 位置。
- `CANVAS-DRAG-2`
  - 已完成。`FlowCanvas.vue` 已补节点级 `connectable`、`targetPosition`、`sourcePosition`，并移除会把自定义节点拖拽误判为 `invalid` 的 Vue Flow 预校验。
- `CANVAS-DRAG-3`
  - 已完成。`npm run build` 通过；浏览器冒烟验证确认：
    - handle 具备 `connectable / connectionindicator` class；
    - 拖拽 `n1 -> n2` 可生成 edge；
    - 重复拖拽不会新增第二条重复边。

## 禁止修改范围
- 不修改后端协议
- 不修改 `repo/` 控制面代码
- 不回退上一轮已完成的节点头尾视觉与抽屉交互改动
