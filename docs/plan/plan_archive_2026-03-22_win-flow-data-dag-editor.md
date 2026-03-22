# Data DAG Editor Plan

## 项目目标与当前状态

### 目标

- 让 Win Flow 编辑器支持数据流 DAG 的可视化配置：
  - `call` 节点支持模板化输入与结构化绑定
  - 新增 `compose` 节点编辑能力
  - 默认表单模式，保留高级 JSON 模式

### 当前状态

- 现有 store 草稿模型仅包含 `method / target / args`
- 编辑器详情抽屉仍以手工编辑参数字段为主
- 还没有 `compose` 节点的前端模型和 UI

## Workflow 信息

- 当前仓库：`MyFlowHub-Win`
- 当前分支：`feat/data-dag-editor`
- Base 分支：`main`
- 当前 worktree：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor`
- 当前阶段：`4`
- 关联主计划：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings\plan.md`

## Docs 与依赖

- Requirements impact：`add`
- Specs impact：`update`
- Related requirements：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\requirements\flow_data_dag.md`
- Related specs：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\specs\flow.md`

## 可执行任务清单（Checklist）

- [x] `DAG-WIN-1` 扩展 store 节点模型与 spec 导入导出
- [x] `DAG-WIN-2` 实现输入绑定 / compose 的编辑 UI
- [x] `DAG-WIN-3` 完成验证、模式联动与回归

## Task 详情

### `DAG-WIN-1`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor`
- Plan：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor\plan.md`
- 目标：
  - 扩展 `FlowNodeDraft`
  - 支持 `call` / `compose` 新 spec 的 parse/build
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`
- Write set：
  - `frontend/src/stores/flow.ts`
- 验收条件：
  - 新旧 payload 均可加载
  - 新 spec 可稳定导出
- 测试点：
  - round-trip
  - 旧 `args` 节点兼容加载
- 回滚点：
  - 回退 `frontend/src/stores/flow.ts`
- 依赖关系：
  - `DAG-DOC-2`
- 风险与注意事项：
  - 不能破坏 `_ui` 布局字段
- 关键上下文引用：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\specs\flow.md`

### `DAG-WIN-2`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor`
- Plan：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor\plan.md`
- 目标：
  - 提供结构化输入绑定编辑 UI
  - 提供 `compose` 节点模板编辑 UI
- 涉及模块 / 文件：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - 视实现需要新增 `frontend/src/components/flow/*`
- Write set：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/*`
- 验收条件：
  - 用户无需手写复杂模板字符串即可配置依赖
  - `compose` 节点可被创建和编辑
- 测试点：
  - call 绑定编辑
  - compose 模板编辑
- 回滚点：
  - 回退编辑器 UI 改动
- 依赖关系：
  - `DAG-WIN-1`
- 风险与注意事项：
  - 默认表单模式必须保留高级 JSON 退路
- 关键上下文引用：
  - `frontend/src/stores/flow.ts`

### `DAG-WIN-3`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor`
- Plan：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor\plan.md`
- 目标：
  - 完成前端验证、模式联动与冒烟验证
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write set：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- 验收条件：
  - 非祖先引用、非法路径等错误有明确提示
  - 表单模式与高级 JSON 模式切换不损坏草稿
- 测试点：
  - 错误提示
  - 模式切换
  - build 冒烟
- 回滚点：
  - 回退前端验证与模式联动改动
- 依赖关系：
  - `DAG-WIN-1`
  - `DAG-WIN-2`
- 风险与注意事项：
  - 不要把协议解析逻辑扩散到多个 Vue 组件
- 关键上下文引用：
  - 本计划
