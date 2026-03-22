# Win Canvas Connectors Plan

## 项目目标与当前状态
- 目标：
  - 提升 Flow 画布节点头尾的可辨识度。
  - 修复右侧详情抽屉阻断画布交互，导致无法继续连线的问题。
- 当前状态：
  - `FlowNode.vue` 的 source/target handle 都是同色小圆点，辨识度和命中面积都偏低。
  - `FlowEditorWindow.vue` 使用全屏 `Overlay` 承载右侧详情抽屉，打开节点详情后 backdrop 会覆盖画布，阻断后续拖线。
  - `FlowCanvas.vue` 的基础 `connect` 链路存在，但当前交互被上层遮挡。

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`fix/win-canvas-connectors`
- Base：`main`
- Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win/plan.md`
- 当前阶段：`4`
- 当前状态：已完成代码、Review 与归档，待用户确认是否结束本次 workflow
- 子Agent策略：本轮不派发
  - 原因：当前环境未获得显式子Agent授权；且改动集中在同一条前端交互链路，拆分收益低于集成成本。

## 需求与架构摘要
- 节点的输入端/输出端需要明显不同的视觉语言。
- 右侧详情要保留“固定抽屉、仅抽屉内部滚动”的形态，但不能再用全屏遮罩阻断画布。
- 现有 DAG 校验、连线写回、撤销/重做语义保持不变。

## 可执行任务清单
- [x] `CANVAS-CONNECT-1` 重构右侧节点详情为非阻断式抽屉
- [x] `CANVAS-CONNECT-2` 强化节点头尾 handle 视觉与命中区域
- [x] `CANVAS-CONNECT-3` 验证、Code Review 与归档

## Task 详情

### `CANVAS-CONNECT-1`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win/plan.md`
- 目标：
  - 去掉节点详情对画布的全屏遮挡
  - 保留右侧固定抽屉和内部滚动
- 涉及模块 / 文件：
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write set：
  - `frontend/src/windows/FlowEditorWindow.vue`
- 验收条件：
  - 打开节点详情后仍可操作画布
  - 抽屉自身仍可关闭、滚动、编辑
- 测试点：
  - 选中节点后可继续从节点 handle 拖线
  - 点击空白仍可清空选择或不影响正常交互
- 回滚点：
  - 回退 `frontend/src/windows/FlowEditorWindow.vue`

### `CANVAS-CONNECT-2`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win/plan.md`
- 目标：
  - 提升 source/target handle 的方向辨识度和拖线命中面积
- 涉及模块 / 文件：
  - `frontend/src/components/flow/FlowNode.vue`
  - 如需要，`frontend/src/components/flow/FlowCanvas.vue`
- Write set：
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/components/flow/FlowCanvas.vue`
- 验收条件：
  - 输入端/输出端颜色、尺寸、位置有明确区分
  - 拖线手感优于当前细小单色 handle
- 测试点：
  - 左右 handle 视觉明显不同
  - 连线后 `onConnect -> flowStore.addEdge(...)` 链路不回归
- 回滚点：
  - 回退 `frontend/src/components/flow/FlowNode.vue`
  - 回退 `frontend/src/components/flow/FlowCanvas.vue`

### `CANVAS-CONNECT-3`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win/plan.md`
- 目标：
  - 完成验证、3.3 Code Review、4 归档
- 涉及模块 / 文件：
  - `plan.md`
  - `docs/change/`
- Write set：
  - `plan.md`
  - `docs/change/2026-03-21_win-canvas-connectors.md`
- 验收条件：
  - 记录验证结果
  - 产出 Code Review 结论
  - 生成归档文档
- 测试点：
  - `git diff --check`
  - 可运行的前端检查命令
  - 如环境允许则做浏览器冒烟
- 回滚点：
  - 删除本次新增文档并回退前述代码文件

## 依赖关系
- `CANVAS-CONNECT-1` 与 `CANVAS-CONNECT-2` 都影响连线体验，但写集相邻，需要集成后一起验证。
- `CANVAS-CONNECT-3` 依赖前两项完成。

## 禁止修改范围
- 不修改后端协议
- 不修改 `repo/` 控制面代码
- 不改部署页、项目中心页、能力查询逻辑
