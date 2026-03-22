# Win Flow Canvas 拖拽连线修复

## 变更背景 / 目标
- 背景：用户反馈 Win Flow 编辑器中，节点头尾颜色已区分，但从 `Out` 拖到另一节点时仍无法进入有效连线状态，导致无法落边。
- 目标：
  - 恢复自定义节点的真实拖拽连线能力。
  - 保持节点详情抽屉、节点头尾视觉区分等上一轮交互不回退。

## 对应 Plan 任务映射
- `CANVAS-DRAG-1`
  - `frontend/src/components/flow/FlowNode.vue`
- `CANVAS-DRAG-2`
  - `frontend/src/components/flow/FlowCanvas.vue`
- `CANVAS-DRAG-3`
  - `plan.md`
  - `docs/change/2026-03-21_win-canvas-drag-state.md`

## 具体变更内容
- 修改 `frontend/src/components/flow/FlowNode.vue`
  - 自定义节点改为使用 `NodeProps<FlowNodeData>`，与 Vue Flow 官方 custom node 约定对齐。
  - 两侧 `Handle` 显式透传 `connectable`。
  - 两侧 `Handle` 显式使用节点传入的 `targetPosition / sourcePosition`，确保左右手柄布局稳定。
- 修改 `frontend/src/components/flow/FlowCanvas.vue`
  - `canvasNodes` 为每个节点补齐 `connectable: true`、`targetPosition: Position.Left`、`sourcePosition: Position.Right`。
  - 移除 `VueFlow` 级别的 `is-valid-connection` 预校验。
  - 连线最终约束继续由 `FlowEditorWindow -> flowStore.addEdge(...)` 负责，仍会阻断自环、重复边、未知节点和成环边。

## 关键设计决策与权衡
- 根因拆分
  - 第一层问题是 custom node 未完整接入 `NodeProps`，导致 handle 仅有视觉样式，没有完整交互能力。
  - 第二层问题是 Vue Flow 预校验在当前受控 custom node 场景下把有效拖拽误判为 `invalid`，表现为拖拽链路已启动、目标 handle 可识别，但始终不给有效落边。
- 最终方案
  - 保留 Vue Flow 的基础拖拽连线能力，只做 `connectable` 和节点位置信息透传。
  - 移除错误拦截的预校验，把最终数据合法性收敛到 `flowStore.addEdge(...)` 单点校验。
- 权衡
  - 这样做牺牲了一部分“拖拽过程中的即时非法提示”，但换回了稳定可用的连线能力。
  - 数据完整性没有放松，因为最终写边仍经过 store 校验。

## 性能 / 可扩展性说明
- 性能
  - 本次修改只涉及前端视图层的 props 透传与校验链路收敛，没有新增网络 I/O、额外轮询或高频重计算。
- 可扩展性
  - 自定义节点已对齐 `NodeProps`，后续若增加多 handle、动态 handle 或不同节点类型，可继续沿用 Vue Flow 官方扩展方式。
  - 边合法性集中在 store 层，后续若补充“禁止跨分支回连”“入口节点限制”等规则，只需扩展 `addEdge(...)`。

## 测试与验证方式 / 结果
- 构建验证
  - 在 `frontend/` 执行 `npm run build`，结果通过。
- 浏览器冒烟验证
  - 使用 Chrome DevTools 打开 `#/flow-editor-window?projectId=test-proj`，注入 Wails mock。
  - 验证结果：
    - handle DOM 含 `connectable / connectionindicator` class；
    - 触发 `n1 -> n2` 拖拽后，连接线 class 为 `valid`；
    - 松手后生成 1 条 edge，DOM 中出现 `e:n1->n2`；
    - 再次重复拖拽不会生成第二条重复边。

## 3.3 Code Review 结论
- 需求覆盖：通过。已恢复用户核心诉求中的拖拽连线能力。
- 架构合理性：通过。改动限定在 Flow 画布组件与自定义节点组件，未扩散到后端协议或项目管理逻辑。
- 性能风险：通过。无新增 N+1、重复 I/O、频繁深拷贝或高频 watcher。
- 可读性与一致性：通过。自定义节点实现回到 Vue Flow 官方 `NodeProps` 模式，降低后续维护成本。
- 可扩展性与配置化：通过。节点交互能力与边校验职责分离，后续扩展多类型节点和更复杂边规则更稳定。
- 稳定性与安全：通过。最终边合法性仍由 store 把关，没有放松数据约束。
- 测试覆盖情况：通过。已完成构建验证和浏览器级拖拽冒烟；未补自动化测试，残余风险主要在未来若重新引入 Vue Flow 预校验时需要重新核对 custom node 行为。
- 子Agent治理与审计：通过。本轮未派发子Agent，所有实现、验证、归档均由主Agent在当前 worktree 内完成。

## 潜在影响与回滚方案
- 潜在影响
  - 拖拽过程中的“预校验高亮”不再承担最终约束职责，非法边会在松手后由 store 拒绝。
  - 若未来要恢复拖拽阶段即时非法提示，需要重新评估 Vue Flow 预校验与当前 custom node 受控状态的兼容性。
- 回滚方案
  - 回滚 `frontend/src/components/flow/FlowNode.vue`
  - 回滚 `frontend/src/components/flow/FlowCanvas.vue`
  - 删除本变更文档

## 子Agent执行轨迹
- 本轮未使用子Agent。
