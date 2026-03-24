# 2026-03-23_win-frontend-number-input-normalization

## 变更背景 / 目标

- 背景：
  - 上一轮已修复 `FlowEditorWindow` 中 `number` 字段 literal 输入触发 `raw.trim is not a function` 的缺陷。
  - 继续排查后发现，前端多个页面仍各自维护 `trim + parseInt/parseFloat` 的数字输入解析样板，存在重复实现和未来再次分叉的风险。
- 目标：
  - 将前端数字输入归一化与整数校验收敛为共享 helper。
  - 在不改变现有业务语义和 Wails 接口契约的前提下，统一关键页面的数字输入处理路径。

## 具体变更内容（新增 / 修改 / 删除）

### 新增

- `frontend/src/lib/numberInput.ts`
  - 提供 `normalizeFormInputText()`、`isBlankFormInput()`、`parseIntegerInput()`、`parseFloatInput()`、`parseNumberInput()`。
  - 统一兼容 `string | number | boolean | null | undefined` 输入值。

### 修改

- `frontend/src/pages/Presets.vue`
  - 使用共享 helper 替换原有 `parseTarget`、`parseOwner` 以及重复的正/非负整数解析逻辑。
  - 保持 target / owner / total / payloadSize / maxBytes 等字段的现有错误文案与 fallback 语义。

- `frontend/src/pages/Showcase.vue`
  - 使用共享 helper 替换本地的 `parsePositiveInt`、`parseNonNegativeInt`、`parseIntInRange`、`parseFloatStrict` 样板。
  - 保持列宽、变量 owner、slider 配置等业务语义不变。

- `frontend/src/windows/FlowEditorWindow.vue`
  - 复用共享 helper 的文本归一化与 strict number 解析逻辑。
  - 继续保持 visual form 数字字段对非法 number 的严格校验。

### 删除

- 删除上述页面内重复的局部数字解析实现。

## 对应 `plan.md` 任务映射

- `NUMFORM-1` 建立共享数字输入归一化 helper
- `NUMFORM-2` 收敛 `Presets` / `Showcase` / `FlowEditorWindow` 的重复解析逻辑
- `NUMFORM-3` 执行验证并修正问题
- `NUMFORM-4` 完成 Code Review
- `NUMFORM-5` 完成 `docs/change` 归档

## Related Plan

- `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-23_win-frontend-number-input-normalization.md`

## Related Requirements

- `none`

## Related Specs

- `none`

## Requirements Impact

- `none`

## Specs Impact

- `none`

## 关键设计决策与权衡（尤其性能 / 扩展性）

- 决策：共享 helper 只负责输入归一化与数字解析，不承接业务文案和页面语义。
  - 原因：避免把简单解析工具膨胀成业务规则中心。
  - 权衡：页面层仍保留少量薄封装，但重复样板明显下降。

- 决策：保留各页面原有的解析语义差异。
  - 例如：
    - `Presets` / `Showcase` 的整数解析继续沿用 `parseInt` 语义
    - `FlowEditorWindow` 的数字字段继续沿用 strict `Number(...)` 语义
  - 原因：本轮目标是统一输入处理，不是改写业务校验标准。
  - 权衡：helper 提供多种解析入口，避免“统一”变成行为回归。

- 决策：本轮先收敛风险最高且重复最明显的三处页面。
  - 原因：控制改动面，优先覆盖已确认的 number 表单主路径。
  - 权衡：其它 store 内的 target/owner 解析重复可在后续独立迭代继续收敛。

## 测试与验证方式 / 结果

- `frontend/ npm run build`
  - 结果：通过
  - 说明：共享 helper、新增导入链路和页面替换后的前端生产构建成功。

## 潜在影响与回滚方案

- 潜在影响：
  - 若调用方错误选择 `parseIntegerInput` / `parseFloatInput` / `parseNumberInput` 的语义，可能造成局部校验行为偏差。
  - 本轮未处理的 store 级 target/owner 解析重复仍然存在，但不属于当前回归风险主路径。

- 回滚方案：
  - 直接回退 `frontend/src/lib/numberInput.ts` 与本轮修改的三个页面文件即可恢复原状。
  - 若仅需临时恢复个别页面，可按文件粒度回退，不影响其它页面。

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）

- 本次 workflow 未使用子Agent。
- `NUMFORM-1` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → `frontend/src/lib/numberInput.ts` → 通过
- `NUMFORM-2` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → `frontend/src/pages/Presets.vue`, `frontend/src/pages/Showcase.vue`, `frontend/src/windows/FlowEditorWindow.vue` → 通过
- `NUMFORM-3` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → `frontend` 构建验证 → 通过
- `NUMFORM-4` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → Review 记录 → 通过
- `NUMFORM-5` → 主Agent → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-flow-varstore-owner-constant` → 当前文档与索引 → 通过
