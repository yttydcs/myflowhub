# Win Flow Method Capability Query Node Plan

## 项目目标与当前状态
- 目标：修复 Win Flow 编辑器方法能力选择器在 popup/editor 窗口中的两类问题：
  - 能力查询只能打当前 executor，不能临时指定查询 `nodeId`
  - popup 窗口未继承实时 session 身份时，能力查询会直接报 `Login required to send Flow requests.`
- 当前状态：
  - `frontend/src/windows/FlowEditorWindow.vue` 已有独立方法选择对话框
  - `frontend/src/stores/flow.ts` 已有 `queryExecCapabilities(...)` 与 `applyCallCapability(...)`
  - 当前查询链路未区分“临时查询目标”和“节点运行时 target”
  - 当前 worktree 尚未进行本轮代码修改

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`refactor/win-method-cap-query-node`
- Base：`main`
- Worktree：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win/plan.md`
- 当前阶段：`4`
- 当前状态：已完成代码、Review 与归档，待用户确认是否结束本次 workflow
- 子Agent策略：本轮不派发
  - 原因：改动集中在 `FlowEditorWindow.vue` 与 `flow.ts`，状态语义和写集高度耦合，拆分后容易把“查询目标”和“运行时 target”再次混淆

## 需求与架构摘要
- 方法选择对话框新增临时 `query node id` 输入，只用于能力查询。
- 能力查询时允许指定 override node id；该值不得写回图数据。
- popup/editor 需要从 `HomeState` 加载持久化身份兜底，避免仅因窗口缺少 session 事件而失败。
- `applyCallCapability(...)` 继续只回填 `method` 与规范化后的运行时 `target`；`args` 为空时补 `{}`，否则保持原值。

## 可执行任务清单
- [x] `FLOW-CAP-QUERY-1` 在方法对话框增加临时查询 nodeId 和身份兜底
- [x] `FLOW-CAP-QUERY-2` 扩展 flow store 能力查询接口，支持 query node override 且不落库
- [x] `FLOW-CAP-QUERY-3` 执行验证、Code Review 与归档

## Task 详情

### `FLOW-CAP-QUERY-1`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win/plan.md`
- 目标：
  - 在 `FlowEditorWindow.vue` 中引入 popup 身份兜底
  - 在方法对话框中增加临时查询 `nodeId` 输入与默认值
  - 刷新能力时把临时查询 `nodeId` 传给 store
- 涉及模块 / 文件：
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write set：
  - `frontend/src/windows/FlowEditorWindow.vue`
- 验收条件：
  - 对话框中存在可编辑的查询 `nodeId`
  - 刷新能力时会使用该值
  - 该值不会被直接写回当前节点配置
  - popup 窗口能从 `HomeState` 补到 identity
- 测试点：
  - 默认值回填为当前 executor / hub
  - `HomeState` 存在身份时不再直接触发登录缺失错误
  - 非法 query nodeId 能给出错误提示
- 回滚点：
  - 回退 `frontend/src/windows/FlowEditorWindow.vue`
- 风险与注意事项：
  - 仅做本窗口局部 fallback，不改全局 session store 初始化策略
- 关键上下文引用：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`

### `FLOW-CAP-QUERY-2`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win/plan.md`
- 目标：
  - 为 `queryExecCapabilities(...)` 增加查询目标 override
  - 保证 override 仅参与 `ExecCapQuerySimple` 调用
  - 保持 `applyCallCapability(...)` 的写回语义不变
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`
- Write set：
  - `frontend/src/stores/flow.ts`
- 验收条件：
  - store 可接受 query node override
  - override 非正整数直接失败
  - 能力应用后只写 `method` 与规范化后的 `target`
- 测试点：
  - override 有值时优先使用 override
  - override 为空时退回现有 executor 解析逻辑
  - 选择能力后图节点未保存 query node 临时值
- 回滚点：
  - 回退 `frontend/src/stores/flow.ts`
- 风险与注意事项：
  - 不能污染 `state.targetId`
- 关键上下文引用：
  - `frontend/src/stores/flow.ts`

### `FLOW-CAP-QUERY-3`
- Owner：主Agent
- Worktree：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win`
- Plan：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win/plan.md`
- 目标：
  - 完成本轮验证、3.3 Code Review、4 归档文档
- 涉及模块 / 文件：
  - `plan.md`
  - `docs/change/`
- Write set：
  - `plan.md`
  - `docs/change/2026-03-21_win-flow-method-capability-query-node.md`
- 验收条件：
  - 记录验证命令及结果
  - 输出 Code Review 结论
  - 生成归档文档
- 测试点：
  - `git diff --check`
  - 能运行的前端校验命令
  - 如环境允许则进行浏览器级检查
- 回滚点：
  - 删除本次新增文档并回退前述代码文件
- 风险与注意事项：
  - 基线若缺失 `wailsjs` 生成物，需如实记录无法完成的验证项

## 依赖关系
- `FLOW-CAP-QUERY-1` 依赖现有方法对话框结构，可先执行
- `FLOW-CAP-QUERY-2` 与 `FLOW-CAP-QUERY-1` 强相关，需在集成时一起验证
- `FLOW-CAP-QUERY-3` 依赖前两项完成

## 禁止修改范围
- 不修改后端协议实现
- 不修改 `repo/` 主工作区文件
- 不修改与本问题无关的页面或 store
