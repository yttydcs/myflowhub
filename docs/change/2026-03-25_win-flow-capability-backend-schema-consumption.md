# Win Flow Capability Backend Schema Consumption

## 变更背景 / 目标

- Win Flow 编辑器的 ordinary mode 已经按“本地 override 优先、后端 `input_schema` 兜底”的顺序消费 capability schema。
- 本轮 SubProto 为 `topicbus::publish`、`file::mkdir`、`file::list`、`file::read_text` 补齐了后端 schema；Win 侧需要确认这些 schema 能直接生成表单，不再依赖额外本地硬编码。

## 具体变更内容

- 为 `frontend/src/stores/flow_schema_resolver.test.ts` 新增第一批后端 capability schema 的消费用例。
- 覆盖 `topicbus::publish`、`file::mkdir`、`file::list`、`file::read_text` 的字段顺序、required 约束和控件类型。
- 本轮未新增 Win 业务代码路径；现有 resolver 已能兼容这批 schema，产品逻辑保持不变。

## Requirements impact: none

## Specs impact: none

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\requirements\flow-editor-visual-form.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `WINSCHEMA-1`
  - `frontend/src/stores/flow_schema_resolver.test.ts`
- `WINSCHEMA-2`
  - 未触发；现有 resolver 无需兼容性修正
- `WINSCHEMA-3`
  - `docs/change/README.md`
  - `docs/change/2026-03-25_win-flow-capability-backend-schema-consumption.md`

## 经验 / 教训摘要

- 对接后端 capability schema 时，Win 侧优先补 resolver 测试，而不是先动业务代码，可以更快确认“到底是 schema 不兼容，还是只是缺保护用例”。
- 这批 schema 只要遵守当前受限 JSON Schema 子集，Win ordinary mode 无需新增逻辑即可消费。

## 可复用排查线索

- 症状：
  - 方法已经在 capability picker 中显示 `schema`
  - 但 inspector 仍只显示 `Advanced JSON`
- 触发条件：
  - route 的 `inputSchema` 为空
  - schema 使用了 Win resolver 不支持的特性
  - method 名称与 route method 不一致
- 关键词：
  - `resolveMethodVisualSchema`
  - `missing_schema`
  - `inputSchema`
  - `flow_schema_resolver`
- 快速检查：
  - 在 `flow_schema_resolver.test.ts` 复刻后端 schema
  - 确认 `route.inputSchema` 已随 `cap_query(include_schema=true)` 返回
  - 检查字段类型是否只使用 `string/integer/number/boolean/object/enum`

## 关键设计决策与权衡

- 维持“后端 schema 为主，本地 override 为例外”的架构方向，不为这批方法继续扩充 Win 本地硬编码。
- 本轮只补消费验证，不主动改 resolver，实现最小安全变更；只有测试证明存在兼容性缺口时，才会进入 Win 业务代码修正。
- `varstore::*` 继续保留前端 override，本轮不迁移到后端。

## 测试与验证方式 / 结果

- `npx vitest run src/stores/flow_schema_resolver.test.ts`
  - 结果：通过（1 个测试文件，4 个用例）
- `npm test`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\frontend`
  - 结果：通过（6 个测试文件，16 个用例）

## 潜在影响与回滚方案

- 潜在影响：
  - 这批 capability 在 ordinary mode 下将直接显示字段表单；若后端 schema 未来扩展到不受支持特性，Win 会退回只显示 `Advanced JSON`。
- 回滚方案：
  - 回退 `frontend/src/stores/flow_schema_resolver.test.ts` 新增用例
  - 如需彻底回退体验，再同步回退对应的 SubProto schema 变更

## 子Agent执行轨迹

- none
