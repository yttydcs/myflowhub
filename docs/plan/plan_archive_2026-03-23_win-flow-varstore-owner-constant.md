# 2026-03-23 Win Flow VarStore Owner Constant Fix

## 项目目标与当前状态

### 目标

- 修复 Win Flow 编辑器 visual form 中数字字段 literal 输入在运行时触发 `raw.trim is not a function` 的缺陷。
- 保持 `args_template + inputs` 的既有数据映射契约不变，不扩大到协议或 store 层改造。

### 当前状态

- `FlowEditorWindow.vue` 的字段草稿值在运行时可能来自 `type="number"` / `v-model.number`，实际类型并不稳定为字符串。
- `VarStore Set` 等 visual form 数字字段在提交时直接对非字符串草稿值调用 `trim()`，会导致 `Owner` 等 literal 常量输入失败。
- 需求与技术契约未变更，问题定位在 UI 草稿值归一化层。

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`fix/flow-varstore-owner-constant`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- 当前阶段：`4 归档变更（已完成，等待 workflow 结束确认）`

## Related Requirements

- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`

## Related Specs

- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`

## Requirements Impact

- `none`

## Specs Impact

- `none`

## 可执行任务清单（Checklist）

- [x] `FLOWFIX-1` 修正视觉表单草稿值解析与类型声明
- [x] `FLOWFIX-2` 补充回归验证
- [x] `FLOWFIX-3` 完成 Code Review
- [x] `FLOWFIX-4` 完成 `docs/change` 归档

## Task 详情

### `FLOWFIX-1` 修正视觉表单草稿值解析与类型声明

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-flow-varstore-owner-constant.md`
- 任务目标：
  - 让 visual form 字段草稿值兼容运行时宽类型输入。
  - 提交前统一做文本归一化，避免对非字符串直接调用 `trim()`。
- 涉及模块 / 文件：
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write set：
  - `frontend/src/windows/FlowEditorWindow.vue`
- 验收条件：
  - `Owner` 等数字字段 literal 常量输入不再触发 `raw.trim is not a function`
  - `call` 节点既有 `args_template + inputs` 行为保持不变
- 测试点：
  - `VarStore Set` 的 `Owner`
  - 其它 visual form 的 `number/select/json/text/textarea` 提交路径
- 回滚点：
  - 回退 `frontend/src/windows/FlowEditorWindow.vue`
- 风险与注意事项：
  - 只修复 UI 草稿归一化，不引入协议或 store 层行为变化

### `FLOWFIX-2` 补充回归验证

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-flow-varstore-owner-constant.md`
- 任务目标：
  - 补齐新 worktree 的前端依赖与生成物，完成构建级验证。
- 涉及模块 / 文件：
  - `frontend/**`
- Write set：
  - 仅限现有授权写集
- 验收条件：
  - `frontend/ npm ci` 通过
  - `GOWORK=off wails generate module` 通过
  - `frontend/ npm run build` 通过
- 测试点：
  - 依赖安装
  - Wails 前端生成物
  - 前端生产构建
- 回滚点：
  - 若验证失败，返回 `FLOWFIX-1` 调整实现
- 风险与注意事项：
  - worktree 初始环境可能缺少 `node_modules` 与 `frontend/wailsjs` 生成物

### `FLOWFIX-3` 完成 Code Review

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-flow-varstore-owner-constant.md`
- 任务目标：
  - 按 workflow 清单审查最小修复是否覆盖问题且未扩大行为面。
- 涉及模块 / 文件：
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write set：
  - 如 review 发现问题，仅限已有写集
- 验收条件：
  - 需求覆盖、稳定性与测试结论完整
- 测试点：
  - review 结论与构建验证一致
- 回滚点：
  - 若 review 不通过，返回 `FLOWFIX-1`
- 风险与注意事项：
  - 重点检查是否引入了额外 schema / runtime 语义变化

### `FLOWFIX-4` 完成 `docs/change` 归档

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-flow-varstore-owner-constant.md`
- 任务目标：
  - 使用 `$docs-governor` 记录第一轮修复结果和 impact 结论。
- 涉及模块 / 文件：
  - `docs/change/2026-03-23_win-flow-varstore-owner-constant.md`
  - `docs/change/README.md`
- Write set：
  - `docs/change/2026-03-23_win-flow-varstore-owner-constant.md`
  - `docs/change/README.md`
- 验收条件：
  - change 文档记录目标、设计决策、验证结果、回滚与 impact
  - 索引同步更新
- 测试点：
  - 归档路径正确
  - impact 与事实一致
- 回滚点：
  - 删除本轮归档并回退索引
- 风险与注意事项：
  - `change` 只记录结果，不替代 requirements/specs

## 依赖关系

- `FLOWFIX-1` -> `FLOWFIX-2` -> `FLOWFIX-3` -> `FLOWFIX-4`

## 并行性与子Agent评估

- 当前任务不计划派发子Agent。
- 原因：
  - 缺陷定位、修复、验证和归档处于同一条关键路径。
  - 修改文件集中在 `FlowEditorWindow.vue`，并行拆分不能显著缩短总时长，反而会增加上下文切换成本。

## 风险与注意事项

- 若把问题误判为协议或 schema 缺陷，容易导致不必要的跨层改动。
- 若只放宽类型声明而不统一归一化入口，类似问题仍可能在其它字段分支重复出现。
- 本轮必须坚持最小修复，只处理 visual form 草稿值类型假设错误。

## 执行结果摘要

- `FLOWFIX-1`：已完成。`FlowEditorWindow.vue` 已支持对宽类型字段草稿值做文本归一化，并修复 `number/select/json/text/textarea` 分支的直接 `trim()` 调用。
- `FLOWFIX-2`：已完成。`frontend/ npm ci`、`GOWORK=off wails generate module`、`frontend/ npm run build` 均通过。
- `FLOWFIX-3`：已完成。Code Review 结论通过，无需回退。
- `FLOWFIX-4`：已完成。已新增 `docs/change/2026-03-23_win-flow-varstore-owner-constant.md` 并更新索引。
