# Data DAG Specs Plan

## 项目目标与当前状态

### 目标

- 为 Flow 数据流 DAG 增强补齐长期 requirement 与 spec
- 保证后续 `SubProto/flow` 与 `MyFlowHub-Win` 的实现以同一套长期文档为准

### 当前状态

- `docs/requirements/` 尚无 Flow 数据流能力叶子文档
- `docs/specs/flow.md` 尚未覆盖节点结果、输入绑定和 `compose` 节点

## Workflow 信息

- 当前仓库：`MyFlowHub-Server`
- 当前分支：`chore/data-dag-specs`
- Base 分支：`main`
- 当前 worktree：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs`
- 当前阶段：`4`
- 关联主计划：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings\plan.md`

## Docs Governor 结论

- 使用 `$docs-governor` 进行文档路由与索引检查
- Requirements impact：`add`
- Specs impact：`update`
- Related requirements：
  - `docs/requirements/flow_data_dag.md`
- Related specs：
  - `docs/specs/flow.md`
- 需要更新的索引：
  - `docs/requirements/README.md`
  - `docs/change/README.md`

## 可执行任务清单（Checklist）

- [x] `DAG-DOC-1` 新增 requirement 文档并更新索引
- [x] `DAG-DOC-2` 更新 Flow spec
- [x] `DAG-DOC-3` 在实现完成后执行归档与索引复核

## Task 详情

### `DAG-DOC-1`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs`
- Plan：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs\plan.md`
- 目标：
  - 新增 `docs/requirements/flow_data_dag.md`
  - 更新 `docs/requirements/README.md`
- 涉及模块 / 文件：
  - `docs/requirements/flow_data_dag.md`
  - `docs/requirements/README.md`
- Write set：
  - `docs/requirements/flow_data_dag.md`
  - `docs/requirements/README.md`
- 验收条件：
  - requirement 文档覆盖范围、场景、验收和边界
  - requirements 索引可导航
- 测试点：
  - 人工一致性审阅
- 回滚点：
  - 回退上述两个文件
- 依赖关系：
  - 无
- 关键上下文引用：
  - `docs/specs/flow.md`

### `DAG-DOC-2`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs`
- Plan：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs\plan.md`
- 目标：
  - 更新 `docs/specs/flow.md`
  - 明确数据流 DAG 的写入契约与执行语义
- 涉及模块 / 文件：
  - `docs/specs/flow.md`
- Write set：
  - `docs/specs/flow.md`
- 验收条件：
  - `call/compose`、输入绑定、运行上下文、结果可见性被明确定义
- 测试点：
  - 人工对照 requirement 与现有动作集审阅
- 回滚点：
  - 回退 `docs/specs/flow.md`
- 依赖关系：
  - `DAG-DOC-1`
- 关键上下文引用：
  - `docs/requirements/flow_data_dag.md`

### `DAG-DOC-3`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs`
- Plan：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs\plan.md`
- 目标：
  - 在阶段 4 前复核 requirement/spec impact 与索引状态
  - 归档本次 Server 文档变更
- 涉及模块 / 文件：
  - `docs/change/2026-03-22_server-flow-data-dag-docs.md`
  - 视需要补充 `docs/change/README.md`
- Write set：
  - `docs/change/*`
  - `docs/change/README.md`
- 验收条件：
  - impact 记录完整
  - 索引保持一致
- 测试点：
  - docs-governor 复核
- 回滚点：
  - 回退本次 change 文档
- 依赖关系：
  - 实现完成后的最终事实
- 关键上下文引用：
  - 本计划
