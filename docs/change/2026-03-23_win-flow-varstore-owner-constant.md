# 2026-03-23_win-flow-varstore-owner-constant

## 变更背景 / 目标

- 背景：
  - Win Flow 编辑器 `call` 节点普通模式中的 `VarStore Set` 等数字字段支持 literal 输入，但 `Owner` 输入节点 ID 常量时会提示 `Failed to update field value.`。
  - 根因是 `frontend/src/windows/FlowEditorWindow.vue` 把字段草稿值固定当成字符串处理，`type="number"` 的输入在运行时写入 number 后触发 `raw.trim is not a function`。
- 目标：
  - 修复数字字段 literal 输入链路。
  - 保持 `args_template + inputs` 的既有映射契约不变。

## 具体变更内容（新增 / 修改 / 删除）

### 修改

- `frontend/src/windows/FlowEditorWindow.vue`
  - 将 `fieldDrafts` 的静态类型从只接收字符串调整为接收运行时宽类型。
  - 新增 `normalizeFieldDraftText()`，在字段提交前统一把草稿值归一化为文本。
  - 更新 `parseFieldDraftValue()` 的 `number/select/json/text/textarea` 分支，避免直接对非字符串值调用 `trim()`。

### 删除

- 无。

## 对应 `plan.md` 任务映射

- `FLOWFIX-1` 修正视觉表单草稿值解析与类型声明
- `FLOWFIX-2` 补充回归验证
- `FLOWFIX-3` 完成 Code Review
- `FLOWFIX-4` 完成 `docs/change` 归档

## Related Plan

- `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-flow-varstore-owner-constant.md`

## Related Requirements

- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`

## Related Specs

- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`

## Requirements Impact

- `none`

## Specs Impact

- `none`

## 关键设计决策与权衡（尤其性能 / 扩展性）

- 决策：修复点放在 UI 草稿解析层，而不是修改 `varstore::set` schema 或 store 层。
  - 原因：问题是字段草稿值类型假设错误，不是业务契约错误。
  - 权衡：改动面最小，同时保留 `flow.ts` 和运行时映射稳定。

- 决策：将草稿值归一化策略扩展到 `select/json/text/textarea` 分支。
  - 原因：避免后续其它控件在接收到非字符串值时重复触发同类异常。
  - 权衡：每次字段提交增加一次常数级 `String()` 转换，开销可以忽略。

## 测试与验证方式 / 结果

- `frontend/ npm ci`
  - 结果：通过
  - 说明：新建 worktree 初始缺少 `node_modules`，先补齐依赖后再验证。

- `GOWORK=off wails generate module`
  - 结果：通过
  - 说明：补齐 `frontend/wailsjs` 生成物，消除构建基线阻塞。

- `frontend/ npm run build`
  - 结果：通过
  - 说明：本次修改后的前端生产构建成功；仅保留既有大 chunk 警告，不影响本缺陷修复。

## 潜在影响与回滚方案

- 潜在影响：
  - 字段草稿值在 UI 层由“字符串假设”改为“宽类型归一化”，若后续有依赖其严格字符串类型的隐藏逻辑，需要继续统一梳理。
- 回滚方案：
  - 直接回退 `frontend/src/windows/FlowEditorWindow.vue` 本次改动即可恢复原行为。
  - 若只需临时规避，可把数字字段改回手工字符串录入，但不建议作为长期方案。

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）

- 本次 workflow 未使用子Agent。
- `FLOWFIX-1` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → `frontend/src/windows/FlowEditorWindow.vue` → 通过
- `FLOWFIX-2` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → 环境补齐与 `frontend` 构建验证 → 通过
- `FLOWFIX-3` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → Review 记录 → 通过
- `FLOWFIX-4` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → 当前文档与索引 → 通过
