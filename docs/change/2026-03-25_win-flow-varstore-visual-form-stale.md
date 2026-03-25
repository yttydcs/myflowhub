# Win Flow VarStore Visual Form Stale State

## 变更背景 / 目标

- Flow 编辑器的 ordinary mode 会在 `args_template` 或 `inputs` 含有 visual schema 未覆盖的字段时保护性降级到 `Advanced JSON`。
- 用户在把已有 `call` 节点从 `varstore::set` 改成 `varstore::get` 后，节点里原有的 `/visibility` 仍残留在 `args_template`，导致 ordinary mode 提示 `extra_literal_field(/visibility)`。
- 本轮目标是在不放宽 compatibility 规则的前提下，修复方法切换时的 form 状态收敛，让交集字段保留、陈旧字段清理。

## 具体变更内容

- 在 `frontend/src/stores/flow.ts` 新增 form 模式专用的 schema 收敛 helper。
- `applyCallCapability()` 在方法实际发生变化且目标方法存在 visual schema 时：
  - 只保留新 schema pointer 对应的 literal 值
  - 过滤掉新 schema 不认识的 binding
  - 再补目标 schema 默认值
- 收敛逻辑只作用于 form 模式，不对 `Advanced JSON` 模式静默删字段。
- 在 `frontend/src/stores/flow.test.ts` 新增 `varstore::set -> varstore::get` 的回归测试，锁定：
  - `/visibility` 被清理
  - `/name` binding 保留
  - `owner` / `name` literal 保留
  - `getNodeVisualForm(...).compatibility.supported === true`

## Requirements impact: none

## Specs impact: none

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\requirements\flow-editor-visual-form.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\specs\flow-editor-visual-form.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `VFSTALE-1`
  - `frontend/src/stores/flow.ts`
- `VFSTALE-2`
  - `frontend/src/stores/flow.test.ts`
- `VFSTALE-3`
  - `docs/change/README.md`
  - `docs/change/2026-03-25_win-flow-varstore-visual-form-stale.md`

## 经验 / 教训摘要

- ordinary mode 的 compatibility 规则没错，问题在于“方法切换路径没有同步清理旧 schema 状态”。
- 对方法切换场景，最稳的策略不是放宽 compatibility，而是按新 schema pointer 进行状态收敛，保留交集字段、删掉陈旧字段。
- 这种清理必须限制在 form 模式，避免影响 advanced JSON 用户的显式高级配置。

## 可复用排查线索

- 症状：
  - `Visual form unavailable`
  - `Args template contains a field that is not covered by the visual form schema (/visibility).`
  - 方法显示为 `varstore::get`，但节点像还带着 `varstore::set` 的字段
- 触发条件：
  - 已存在 `call` 节点从一个方法切到另一个方法
  - 新旧方法 schema 不完全相同
- 关键词：
  - `extra_literal_field`
  - `/visibility`
  - `applyCallCapability`
  - `varstore::get`
- 快速检查：
  - 查看节点 `args_template` 是否还带旧方法字段
  - 查看节点 `inputs[].to` 是否仍指向新 schema 不支持的 pointer
  - 检查 `applyCallCapability()` 是否在 form 模式下按新 schema 收敛状态

## 关键设计决策与权衡

- 决策：保留新旧 schema 的交集字段，而不是全量重置 `args_template` / `inputs`
  - 原因：`owner`、`name` 这类公共字段仍然有效，没必要让用户重填。
- 决策：不修改 `analyzeVisualCompatibility()` 规则
  - 原因：compatibility 拒绝 schema 外字段本来就是正确保护，不应该靠放宽规则掩盖切换路径的数据残留。
- 决策：只在 form 模式收敛，不在 json 模式静默删字段
  - 原因：高级模式应保留用户完整控制权。

## 测试与验证方式 / 结果

- `npx vitest run src/stores/flow.test.ts`
  - 结果：通过（1 个文件，5 个用例）
- `npm test`
  - 结果：通过（6 个文件，17 个用例）

## 潜在影响与回滚方案

- 潜在影响：
  - form 模式下切方法时，旧 schema 独有字段会被自动清理；这是有意行为，用来保持 ordinary mode 可用。
- 回滚方案：
  - 回退 `frontend/src/stores/flow.ts` 中的 schema 收敛 helper 和 `applyCallCapability()` 变更
  - 回退 `frontend/src/stores/flow.test.ts` 新增测试

## 子Agent执行轨迹

- none
