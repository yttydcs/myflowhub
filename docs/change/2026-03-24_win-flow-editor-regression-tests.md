# 2026-03-24 Win Flow Editor Regression Tests

## Background

- `FLOW-ENH-5` 的目标不是继续扩 Win Flow 编辑器功能，而是为前几轮已经落下的 visual form、draft reliability 和 editor state 行为补齐自动化回归基线。
- 当前前端已经有最小 `vitest` 栈，以及 `flow_visual_form.test.ts`、`Overlay.test.ts` 两个基线测试，但 `flow_schema_resolver.ts` 和 `flow.ts` 仍缺少针对核心语义的自动化保护。

## Goal

- 为 capability schema resolver 建立稳定的子集解析回归测试。
- 为 flow store 建立 ancestor、binding、spec mode 和 graph editor state 的回归测试。
- 保持这轮只补验证，不改变既有 requirements/specs 和运行时行为。

## Changes

- 新增 `frontend/src/stores/flow_schema_resolver.test.ts`
  - 覆盖 local override 优先级。
  - 覆盖 capability `input_schema` 的受限 JSON Schema 子集解析。
  - 覆盖 nested object flatten、enum/select 映射和 unsupported feature rejection。
- 新增 `frontend/src/stores/flow.test.ts`
  - 通过 `useFlowStore()` 的公开 API 覆盖 ancestor 列表和非祖先 binding 拒绝。
  - 覆盖字段 literal / binding 写入、binding 清理和 `exportGraphDraft()` 输出。
  - 覆盖 JSON/form spec mode 切换的回写路径。
  - 覆盖 `exportGraphEditorState()` / `loadGraphEditorState()` 与 `graphEditorSignature()` 的 dirty-state 边界。
- 未新增测试框架或运行时代码变更：
  - 继续沿用现有 `vitest`。
  - `frontend/vitest.config.ts`、requirements 和 specs 均无需修改。

## Related Plan

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Task mapping:
  - `FLOW-ENH-5-A` Schema Resolver 回归测试
  - `FLOW-ENH-5-B` Flow Store 回归测试
  - `FLOW-ENH-5-C` 验证与归档准备

## Related Requirements

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-draft-reliability.md`
- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`

## Related Specs

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-draft-reliability.md`
- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`

## Lessons Impact

- `none`

## Related Lessons

- `none`

## Searchable Lessons Summary

- Symptoms:
  - visual form / resolver 逻辑改动后缺少快速回归手段
  - `flow.ts` 的 ancestor、binding 或 dirty-state 边界容易被后续改动带坏
  - 只能依赖手工打开 editor 做冒烟
- Trigger Conditions:
  - 调整 capability `input_schema` 解析
  - 修改 `flow.ts` 的 graph editor state、binding 写回或 JSON/form 切换逻辑
- Keywords:
  - `flow_schema_resolver`
  - `graphEditorSignature`
  - `setFieldBinding`
  - `setNodeSpecEditorMode`
  - `vitest`
- Quick Checks:
  - 先跑 `cd frontend && npm test`
  - 看 resolver 用例是否覆盖 local override / enum / unsupported schema
  - 看 store 用例是否覆盖 ancestor、binding、spec mode 和 signature

## Requirements Impact

- `none`

## Specs Impact

- `none`

## Experience Summary

- 对 Flow 编辑器这类仍在收敛中的前端模块，先锁住 store / resolver 公开边界，比过早搭完整 UI 级回归更划算。
- 共享 singleton store 可以测试，但必须把 reset 作为夹具的一部分，否则用例会互相污染。

## Design Decisions and Tradeoffs

- 本轮只补 store / resolver 回归，不补更重的 editor UI 测试：
  - 好处是成本低、执行快、覆盖的正是当前最易回归的协议与草稿边界。
  - 代价是更完整的 editor 交互仍主要依赖现有 overlay 测试和手工冒烟。
- store 测试只通过公开 API 断言：
  - 好处是后续可以继续拆 helper 而不必同步重写整批测试。
  - 代价是个别内部细节不会被精确钉死，但这正是本轮想保留的重构空间。

## Validation

- `cd frontend && npm test`：通过
  - 共 4 个测试文件、13 个用例全部通过。
- `cd frontend && npm run build`：通过
  - 保留单 chunk 约 `969.91 kB` 的既有告警，本轮未处理 chunk 拆分。
- `$env:GOWORK='off'; wails generate module`：通过
  - 命令仍打印 `Not found: time.Time`，但最终成功完成。

## Potential Impact and Rollback

- 潜在影响：
  - 本轮只新增测试文件，不影响 Flow 编辑器运行时行为。
  - 后续若这些测试失败，通常意味着 resolver 或 store 的稳定边界被改动，需要先回看相应 requirements/specs。
- Rollback:
  - 回退 `frontend/src/stores/flow_schema_resolver.test.ts`
  - 回退 `frontend/src/stores/flow.test.ts`
  - 回退 `docs/change/2026-03-24_win-flow-editor-regression-tests.md`
  - 回退 `docs/change/README.md`
  - 回退 `plan.md` 中的 `FLOW-ENH-5` review / archive 记录

## SubAgent Trace

- `none`
