# 2026-03-21 Win Canvas Connectors

## 变更背景 / 目标
- 背景：
  - Flow 画布节点的输入端/输出端都是小尺寸同色 handle，头尾方向不够明确。
  - 节点详情使用全屏 `Overlay` 从右侧弹出，backdrop 会覆盖画布，导致选中节点后无法继续拖线连接。
- 目标：
  - 强化节点头尾辨识度和拖线命中区域。
  - 将节点详情改成非阻断式右侧抽屉，恢复画布连线交互。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `frontend/src/windows/FlowEditorWindow.vue`
  - 节点详情从全屏 `Overlay` 改为画布容器内的绝对定位右侧抽屉。
  - 画布内容区域在抽屉打开时增加右侧内边距，避免节点被抽屉完全压住。
  - 抽屉外层使用 `pointer-events-none`，只让抽屉本身接收事件，不再吞掉画布操作。

- `frontend/src/components/flow/FlowNode.vue`
  - source/target handle 从小尺寸同色圆点改为明显不同颜色的大尺寸 handle。
  - 输入端使用蓝色，输出端使用绿色，并在节点头部增加 `In / Out` 方向标签。
  - 选中节点时阴影和 ring 略增强，提升当前编辑焦点。

- `frontend/src/components/flow/FlowCanvas.vue`
  - 显式设置 `:nodes-connectable="true"`，避免依赖三方库默认值。

### 新增
- `docs/change/2026-03-21_win-canvas-connectors.md`

### 删除
- 无

## 对应 `plan.md` 任务映射
- `CANVAS-CONNECT-1`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `CANVAS-CONNECT-2`
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/components/flow/FlowCanvas.vue`
- `CANVAS-CONNECT-3`
  - `plan.md`
  - `docs/change/2026-03-21_win-canvas-connectors.md`
  - 验证记录
  - Code Review 结论

## 关键设计决策与权衡（尤其性能 / 扩展性）
- 决策：节点详情不再复用全屏 `Overlay`。
  - 原因：详情抽屉不是模态窗口，模态遮罩会直接阻断画布拖线、选边和其它交互。
  - 权衡：少了全屏 backdrop 的“聚焦感”，但换回了节点编辑器最关键的连续交互能力。

- 决策：抽屉打开时给画布内容区增加右侧 padding。
  - 原因：保持“从右侧滑出”的抽屉体验，同时尽量避免右侧节点被抽屉遮住。
  - 性能：仅是常量级 class 切换，没有额外 I/O 或复杂计算。

- 决策：handle 颜色和尺寸同时调整，而不是只换颜色。
  - 原因：当前问题既有方向辨识不足，也有拖线命中面积太小的可用性问题。
  - 扩展性：后续若要支持多输入/多输出 handle，可继续沿用这套颜色语义。

## 测试与验证方式 / 结果
- `git diff --check`
  - 结果：通过
  - 说明：仅有 Windows 行尾提示，不影响本次变更内容

- 前端构建 / 浏览器冒烟
  - 结果：未执行
  - 原因：
    - 当前 worktree 不存在 `frontend/node_modules`
    - 当前 worktree 不存在 `frontend/wailsjs`
  - 结论：本轮只能完成 diff 级校验与代码级 review，无法在当前环境中拉起前端做交互冒烟

## Code Review（3.3）
- 需求覆盖：通过
  - 节点头尾已经用不同颜色和标签区分。
  - 节点详情不再通过全屏遮罩阻断画布，连线通路恢复到可操作状态。

- 架构合理性：通过
  - 详情抽屉职责留在 `FlowEditorWindow.vue`。
  - 节点视觉增强收敛在 `FlowNode.vue`。
  - 连线链路仍由 `FlowCanvas.vue -> FlowEditorWindow.vue -> flowStore.addEdge(...)` 负责，边界清晰。

- 性能风险：通过
  - 仅新增 class 切换和更显眼的 handle 渲染，无额外网络、循环或序列化成本。

- 可读性与一致性：通过
  - 改动集中在三个前端文件，命名与现有风格一致。
  - `In / Out` 标签使头尾语义更直接。

- 可扩展性与配置化：通过
  - 后续扩展多 handle 或自定义方向样式时，主要继续演进 `FlowNode.vue` 即可。

- 稳定性与安全：通过
  - 未更改协议、身份、存储或部署路径。
  - DAG 校验和 `addEdge(...)` 逻辑保持不变。

- 测试覆盖情况：部分通过
  - 已完成 diff 级检查和代码审查。
  - 受缺失前端依赖与 `wailsjs` 生成物影响，未完成构建和浏览器级冒烟。

- 子Agent治理与审计：通过
  - 本次未使用子Agent。
  - 原因：当前环境未获得显式子Agent授权，且这轮改动集中在同一条前端交互链路，由主Agent本地连续实现更安全。

## 潜在影响与回滚方案

### 潜在影响
- 抽屉打开时画布内容区会右缩，为抽屉预留空间。
- 节点视觉更强烈，和之前的极简风格相比会更“工具化”。

### 回滚方案
- 回退以下文件即可恢复本轮变更：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/components/flow/FlowCanvas.vue`
  - `plan.md`
  - `docs/change/2026-03-21_win-canvas-connectors.md`

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）
- 本次未使用子Agent
  - Task ID：`CANVAS-CONNECT-1` / `CANVAS-CONNECT-2` / `CANVAS-CONNECT-3`
  - Agent：主Agent
  - Worktree：`D:/project/MyFlowHub3/worktrees/fix-win-canvas-connectors/MyFlowHub-Win`
  - 文件：
    - `frontend/src/windows/FlowEditorWindow.vue`
    - `frontend/src/components/flow/FlowNode.vue`
    - `frontend/src/components/flow/FlowCanvas.vue`
    - `plan.md`
    - `docs/change/2026-03-21_win-canvas-connectors.md`
  - 验收结果：已完成实现、Review 与归档；待用户确认是否结束 workflow
