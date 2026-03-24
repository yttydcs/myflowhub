# 2026-03-23 Win Frontend Number Input Normalization

## 项目目标与当前状态

### 目标

- 将前端分散的数字输入归一化与整数校验逻辑收敛为共享 helper。
- 降低 `type="number"` / `v-model.number` 带来的运行时值类型漂移风险，避免后续再次出现 `raw.trim is not a function` 一类缺陷。
- 在不改变现有业务语义和 Wails 调用契约的前提下，统一 `Presets`、`Showcase`、`FlowEditorWindow` 的数字输入处理路径。

### 当前状态

- 同一 worktree 内上一轮缺陷修复已完成并归档：
  - `frontend/src/windows/FlowEditorWindow.vue` 已修复字段草稿值的宽类型归一化问题。
  - `docs/change/2026-03-23_win-flow-varstore-owner-constant.md` 已记录上一轮结果。
- 当前前端数字输入处理存在重复实现：
  - `frontend/src/pages/Presets.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- 已确认的共同风险点：
  - number 输入在运行时可能写入 `number` 而不是 `string`
  - 页面层仍重复维护 `trim + parseInt/parseFloat` 逻辑，未来容易再次分叉

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`fix/flow-varstore-owner-constant`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- 当前阶段：`4 归档变更（已完成，等待 workflow 结束确认）`

## Related Requirements

- `none`

## Related Specs

- `none`

## Requirements Impact

- `none`

## Specs Impact

- `none`

## 可执行任务清单（Checklist）

- [x] `NUMFORM-1` 建立共享数字输入归一化 helper
- [x] `NUMFORM-2` 收敛 `Presets` / `Showcase` / `FlowEditorWindow` 的重复解析逻辑
- [x] `NUMFORM-3` 执行验证并修正问题
- [x] `NUMFORM-4` 完成 Code Review
- [x] `NUMFORM-5` 完成 `docs/change` 归档

## Task 详情

### `NUMFORM-1` 建立共享数字输入归一化 helper

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-frontend-number-input-normalization.md`
- 任务目标：
  - 在前端共享层新增统一 helper，封装 number 输入的文本归一化、正整数/非负整数解析，以及可空文本判断。
- 涉及模块 / 文件：
  - `frontend/src/lib/` 下新 helper 文件
- Write set：
  - `frontend/src/lib/*`
- 验收条件：
  - helper 能兼容 `string | number | null | undefined`
  - helper 能表达正整数、非负整数、可空文本三类常见语义
- 测试点：
  - string 输入解析
  - number 输入解析
  - 空值、非法值、负值校验
- 回滚点：
  - 回退新增 helper 文件
- 风险与注意事项：
  - helper 只负责输入归一化与数字解析，不耦合具体页面业务
- 关键上下文引用：
  - `frontend/src/lib/utils.ts`
  - `frontend/src/pages/Presets.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`

### `NUMFORM-2` 收敛 `Presets` / `Showcase` / `FlowEditorWindow` 的重复解析逻辑

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-frontend-number-input-normalization.md`
- 任务目标：
  - 用共享 helper 替换页面内重复的 number 解析样板。
  - 保持原有错误文案、fallback 语义和业务行为不变。
- 涉及模块 / 文件：
  - `frontend/src/pages/Presets.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write set：
  - `frontend/src/pages/Presets.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/lib/*`
- 验收条件：
  - 页面不再各自维护重复的 `trim + parseInt/parseFloat` 样板
  - 运行时 number 输入仍能正确提交
  - 现有错误提示和 fallback 行为不回退
- 测试点：
  - `Presets` 的 target / owner / total / payload size / max bytes
  - `Showcase` 的 ownerId / slider / 数值配置
  - `FlowEditorWindow` 的 visual form number 字段
- 回滚点：
  - 回退上述页面文件与 helper 文件
- 风险与注意事项：
  - 不要把“正整数”与“非负整数”语义混用
  - 不要将 generic helper 扩张成业务文案仓库
- 关键上下文引用：
  - `frontend/src/pages/Presets.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`

### `NUMFORM-3` 执行验证并修正问题

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-frontend-number-input-normalization.md`
- 任务目标：
  - 对本轮收敛改动执行静态和构建验证，确认没有引入新的类型或打包问题。
- 涉及模块 / 文件：
  - `frontend/**`
- Write set：
  - 仅限本计划已有写集
- 验收条件：
  - `frontend/npm run build` 通过
  - 若新增问题出现，能够定位并修正
- 测试点：
  - `frontend/ npm run build`
  - 如需要，补 `GOWORK=off wails generate module`
- 回滚点：
  - 回退本轮修改文件
- 风险与注意事项：
  - `frontend/wailsjs` 生成物在新 worktree 中可能需要重新生成
- 关键上下文引用：
  - `guide.md`

### `NUMFORM-4` 完成 Code Review

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-frontend-number-input-normalization.md`
- 任务目标：
  - 按 workflow 强制清单审查本轮收敛改动。
- 涉及模块 / 文件：
  - 本轮新增或修改的全部前端文件
- Write set：
  - 如审查发现问题，仅限已授权写集
- 验收条件：
  - 需求覆盖、架构、性能、可维护性和测试结论完整
- 测试点：
  - Review 结论与验证记录一致
- 回滚点：
  - 若 review 不通过，返回 `NUMFORM-1` 或 `NUMFORM-2`
- 风险与注意事项：
  - 重点检查 helper 是否真正减少重复，而不是仅把重复逻辑平移
- 关键上下文引用：
  - 本计划

### `NUMFORM-5` 完成 `docs/change` 归档

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant`
- Plan：`D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-frontend-number-input-normalization.md`
- 任务目标：
  - 使用 `$docs-governor` 复核 impact，并归档本轮前端 number input 收敛结果。
- 涉及模块 / 文件：
  - `docs/change/2026-03-23_win-frontend-number-input-normalization.md`
  - `docs/change/README.md`
- Write set：
  - `docs/change/2026-03-23_win-frontend-number-input-normalization.md`
  - `docs/change/README.md`
- 验收条件：
  - change 文档包含目标、任务映射、设计决策、验证、回滚和 impact 结论
  - 索引已更新
- 测试点：
  - 文档引用路径正确
  - impact 记录与本轮事实一致
- 回滚点：
  - 删除本轮归档并回退索引修改
- 风险与注意事项：
  - `change` 只记录结果，不替代稳定 requirements/specs
- 关键上下文引用：
  - 本计划

## 依赖关系

- `NUMFORM-1` -> `NUMFORM-2` -> `NUMFORM-3` -> `NUMFORM-4` -> `NUMFORM-5`

## 并行性与子Agent评估

- 当前任务不计划派发子Agent。
- 原因：
  - 共享 helper 的设计、页面替换和验证处于同一条关键路径。
  - `Presets` / `Showcase` / `FlowEditorWindow` 都依赖同一 helper API，若并行拆分容易产生接口漂移与写集冲突。

## 风险与注意事项

- 若 helper 抽象过度，会把简单输入校验变成难理解的通用框架。
- 若 helper 命名不清晰，调用方仍会继续保留页面内自定义解析，达不到真正收敛效果。
- 本轮必须坚持“最小行为变更”，只统一输入处理，不改业务表单协议与文案语义。

## 执行结果摘要

- `NUMFORM-1`：已完成。新增 `frontend/src/lib/numberInput.ts`，统一提供输入文本归一化、整数/浮点/strict number 解析。
- `NUMFORM-2`：已完成。`Presets.vue`、`Showcase.vue`、`FlowEditorWindow.vue` 已切换到共享 helper，重复样板显著减少。
- `NUMFORM-3`：已完成。`frontend/npm run build` 构建通过。
- `NUMFORM-4`：已完成。Code Review 结论通过，无需回退到 3.2。
- `NUMFORM-5`：已完成。已新增 `docs/change/2026-03-23_win-frontend-number-input-normalization.md` 并更新 `docs/change/README.md`。
