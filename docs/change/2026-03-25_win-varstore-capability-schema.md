# Win VarStore Capability Schema

## 变更背景 / 目标

- `varstore::get/set/revoke` 一直依赖 Win 本地 override 生成 ordinary mode 表单，后端 capability registry 并不是这组三个方法的真实 schema 来源。
- 用户已确认本轮不需要兼容旧版，因此目标调整为：把 `varstore::*` 的表单 schema 迁到后端 capability，Win 只保留受限 UI hint 消费能力，并确保已有节点在重新打开项目后仍能直接进入 ordinary mode。

## 具体变更内容

- 删除 `frontend/src/stores/flow_method_schemas.ts` 中 `varstore::*` 的完整本地 schema override。
- 在 `frontend/src/stores/flow_schema_resolver.ts` 增加受限 `x-ui-control` 解析，当前仅支持 `textarea`，用于把 `varstore::set.value` 渲染成多行输入。
- 在 `frontend/src/stores/flow.ts` 增加 `ensureNodeCapabilityLoaded()`：
  - 仅按当前节点的 `method + providerNode` 精确查询 capability schema
  - 只合并命中的 route，不覆盖已有 capability 列表
  - 使用 cache version + load epoch 防止项目切换或图重置后被旧请求回写污染
- 在 `frontend/src/windows/FlowEditorWindow.vue` 监听选中 call 节点，静默补齐缺失的 capability schema，避免旧节点在重开后退回 `Advanced JSON`。
- 更新 `frontend/src/stores/flow_schema_resolver.test.ts`、`frontend/src/stores/flow.test.ts`、`frontend/src/windows/FlowEditorWindow.test.ts`，覆盖：
  - `varstore::*` schema 来源为 capability
  - `varstore::set.value` 使用 `textarea`
  - 旧节点选中时会自动 hydration capability schema
- 更新 `docs/specs/flow-editor-visual-form.md`，澄清 Win 支持的 `x-ui-control` 边界与 fallback 规则。

## Requirements impact: none

## Specs impact: updated

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\docs\requirements\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `WINVAR-1`
  - `frontend/src/stores/flow_method_schemas.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow.ts`
- `WINVAR-2`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `WINVAR-3`
  - `frontend/src/stores/flow_schema_resolver.test.ts`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
- `WINVAR-4`
  - `docs/specs/flow-editor-visual-form.md`
- `WINVAR-5`
  - `docs/change/README.md`
  - `docs/change/2026-03-25_win-varstore-capability-schema.md`

## 经验 / 教训摘要

- capability schema 一旦成为 ordinary mode 的真实来源，Win 就不能再悄悄依赖同方法的本地 override，否则前后端 schema 迟早漂移。
- UI hint 应收敛为白名单扩展；`x-ui-control` 只负责控件展示，不应该改变字段语义。
- 旧节点的 ordinary mode 不能依赖用户先打开方法选择器；只要节点被选中，编辑器就应具备最小 schema hydration 能力。

## 可复用排查线索

- 症状：
  - `Visual form unavailable`
  - 已有 `varstore::*` 节点重开项目后只剩 `Advanced JSON`
  - `varstore::set.value` 退回单行输入
- 触发条件：
  - `cap_query(include_schema=true)` 未返回节点方法对应的 `input_schema`
  - Win 侧仅清空后重查 capability list，没有做节点级 hydration
  - 后端返回了 Win 不支持的 schema 扩展
- 关键词：
  - `ensureNodeCapabilityLoaded`
  - `x-ui-control`
  - `missing_schema`
  - `execCapabilities`
- 快速检查：
  - 看 `flow_method_schemas.ts` 是否仍残留 `varstore::*` override
  - 看 `state.execCapabilities` 是否包含当前节点 `method + providerNode` 的 route
  - 看 capability `inputSchema.properties.value["x-ui-control"]` 是否为 `textarea`

## 关键设计决策与权衡

- 决策：采用“backend schema + x-ui-control”，不保留 `varstore::*` 完整前端 override
  - 原因：用户已明确不需要兼容旧版，应尽快收敛真实 schema 来源。
- 决策：节点级 hydration 只做精确查询并合并缓存，不直接替换 capability list
  - 原因：方法选择器需要完整列表；静默补齐不能把用户的候选方法视图缩窄成“只剩当前方法”。
- 决策：`output_schema` 本轮只随 route 保存，不在 Win 侧做结果消费强校验
  - 原因：先完成输入表单迁移，避免把 DAG 结果契约化一并拉进本轮。

## 测试与验证方式 / 结果

- `npm test`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\frontend`
  - 结果：通过（6 个测试文件，19 个用例）

## 潜在影响与回滚方案

- 潜在影响：
  - `varstore::*` 的 ordinary mode 将完全依赖 capability schema；若后端 schema 缺失或结构漂移，节点会重新退回 `Advanced JSON`。
  - 选中旧节点时会触发一次静默 capability 精确查询；如果节点很多，只有被选中的节点才会查询，不会全图预取。
- 回滚方案：
  - 回退 `frontend/src/stores/flow_method_schemas.ts`、`frontend/src/stores/flow_schema_resolver.ts`、`frontend/src/stores/flow.ts`
  - 回退 `frontend/src/windows/FlowEditorWindow.vue` 与相关测试
  - 如需恢复旧行为，再同步恢复 `varstore::*` 本地 override

## 子Agent执行轨迹

- none
