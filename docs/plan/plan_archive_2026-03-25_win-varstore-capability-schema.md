# Plan - win-varstore-capability-schema

## Workflow Information
- Repo: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
- Branch: `feat/win-varstore-capability-schema`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`: 已读取；commit 信息使用中文，允许使用 `chrome-devtools` 做界面验证，子协议长期文档以 `repo\MyFlowHub-Server\docs` 为准，所有 worktree 必须放在 `D:\project\MyFlowHub3\worktrees\`。
- base/worktree confirmation:
  - Main execution worktree: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
  - Participating repo: `repo\MyFlowHub-Win`
  - Active branch: `feat/win-varstore-capability-schema`
  - Cross-repo dependency:
    - `repo\MyFlowHub-SubProto`
    - Worktree: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema`
    - Branch: `feat/subproto-varstore-capability-schema`
    - Plan: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema\plan.md`

### Stage 1 - Requirements Analysis
#### Goal
- 将 `varstore::get/set/revoke` 的 ordinary mode schema 真相源迁到后端 capability registry，并保留 `varstore::set.value` 的多行输入体验。

#### Scope
- Must:
  - Win 不再依赖 `varstore::*` 的完整本地 override。
  - Win 必须消费后端返回的 `varstore::* input_schema` 生成 ordinary mode 表单。
  - Win 必须支持受限 `x-ui-control` 扩展，至少覆盖 `textarea`。
  - Win 必须继续保留 `output_schema` 到 route 状态，供后续结果提示使用。
  - 补自动化测试锁定 `varstore::*` 已由 capability schema 驱动。
- Optional:
  - 若 `output_schema` 消费入口能低成本补最小提示测试，可一并补充。
- Out of scope:
  - 本轮不做 `output_schema` 驱动的 `node_result` 强校验。
  - 本轮不扩展到数组、`oneOf/$ref` 等复杂 schema 特性。
  - 本轮不修改 Flow DAG 运行时绑定语义。

#### Use Cases
- 用户新建 `varstore::set` 节点时，ordinary mode 直接由 capability schema 生成字段表单。
- 用户编辑 `varstore::set.value` 时，字段仍以 `textarea` 展示。
- 后续若要做 `node_result` 字段提示，Win 已持有 `varstore::* output_schema` 元数据。

#### Functional Requirements
1. Win 必须按 capability `input_schema` 解析 `varstore::get/set/revoke` 的表单字段。
2. Win 必须支持 `x-ui-control: "textarea"` 并映射到 `MethodFieldControl="textarea"`。
3. Win 必须移除对 `varstore::*` 完整本地 schema 的依赖。
4. Win 必须继续保留 `output_schema` 于 capability route 模型。
5. 相关 resolver 测试必须覆盖来源为 `capability`，而非 `local_override`。
6. 已有 `call` 节点在重新打开项目或重新选中时，ordinary mode 不得因为 capability 尚未手动查询而退回 `Advanced JSON`。

#### Non-functional Requirements
- 最小安全改动，不引入新的 Flow 持久化格式。
- capability schema 解析失败时仍安全回退到 `Advanced JSON`。
- 不增加 capability 请求次数或全图重复计算。
- 自动补 capability 只能按选中节点精确查询，避免把方法选择器的完整列表语义意外缩窄。

#### Inputs / Outputs
- Inputs:
  - capability `input_schema`
  - capability `output_schema`
  - `x-ui-control`
- Outputs:
  - capability 驱动的 `varstore::*` visual schema
  - Win resolver / 测试更新
  - 必要的 Win spec 澄清

#### Edge Cases
- 未知 `x-ui-control` 必须忽略并回退默认控件。
- capability schema 缺失或非法时，ordinary mode 必须隐藏。
- `value` 仍按当前协议语义视为 `string`，不是任意 JSON。

#### Acceptance Criteria
1. `varstore::get/set/revoke` 的 visual schema 来源为 `capability`。
2. `varstore::set.value` 控件仍为 `textarea`。
3. Win 自动化测试覆盖 capability schema、`x-ui-control` 和 fallback 边界。
4. ordinary mode 不因本轮迁移回退到 `Advanced JSON`。
5. 已保存的 `varstore::*` 节点在重开编辑器后，选中即可恢复 ordinary mode。

#### Risks
- 若本地 override 未完全移除，迁移会停留在“看似后端化，实际前端优先”的半成品状态。
- 若 `x-ui-control` 解析过宽，未来可能把不受控 UI 语义引入 resolver。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 让 `varstore::*` 的业务字段结构完全来自 capability schema；Win 仅负责解析受限 JSON Schema 子集并消费极薄的 `x-ui-control` 展示提示。`output_schema` 本轮仅保留到 route 状态，不做 binding hard validation。

#### Alternatives Considered
- 保留完整本地 override：
  - 不采用；会继续与后端 schema 漂移。
- 纯后端 schema，不保留任何 UI 提示：
  - 不采用；`varstore::set.value` 会退化为单行输入。
- 后端 schema + 受限 `x-ui-control`：
  - 采用；既统一真相源，又能保留必要 UI 体验。

#### Module Responsibilities
- `frontend/src/stores/flow_schema_resolver.ts`
  - 支持解析 `x-ui-control`。
- `frontend/src/stores/flow_method_schemas.ts`
  - 删除 `varstore::*` 完整本地 override。
- `frontend/src/stores/flow_schema_resolver.test.ts`
  - 锁定 `varstore::*` capability schema 解析结果。
- `docs/specs/flow-editor-visual-form.md`
  - 澄清 capability schema 支持的受限 UI 扩展。

#### Data / Call Flow
- `cap_query(include_schema=true)`
  - route 携带 `input_schema/output_schema`
  - `resolveMethodVisualSchema(method, route)`
  - capability schema 解析
  - `x-ui-control` 覆盖默认控件
  - ordinary mode 生成字段表单

#### Interface Drafts
- 支持的 UI 扩展：
  - `x-ui-control: "textarea"`
- 未知 UI 扩展：
  - 忽略并回退到基础类型默认控件

#### Error Handling and Safety
- schema 不满足当前受限子集时，继续回退 `Advanced JSON`。
- `x-ui-control` 不改变字段语义，只改变展示控件。

#### Performance and Testing Strategy
- 性能：
  - 不新增请求，只在已有 schema 解析路径多读取一个扩展字段。
- 测试：
  - `frontend/src/stores/flow_schema_resolver.test.ts`
  - `cd frontend && npm test`

#### Extensibility Design Points
- `x-ui-control` 先做受限白名单，后续如需更多控件可渐进扩展。
- `output_schema` 留作后续 `node_result` 字段提示的基础元数据。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 Win 虽已支持 capability schema 消费，但 `varstore::*` 仍被本地 override 抢占，无法真正以后端 schema 为真相源。
- 本轮目标是在不回退 ordinary mode 体验的前提下，将 `varstore::*` 迁到 capability schema，并为后续结果提示沉淀 `output_schema` 元数据。
- 实施中发现补充风险：`getNodeVisualForm()` 仅依赖内存中的 `execCapabilities`，已有节点在项目重载后可能缺少 route cache，从而退回 `Advanced JSON`；需一并修复。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Docs tree bootstrapping / repair: `none`
- Canonical destinations:
  - Stable truth: `docs/specs/flow-editor-visual-form.md`
  - Workflow control: worktree 根 `plan.md`
  - Completed result: `docs/change/YYYY-MM-DD_topic.md`
  - Reusable troubleshooting: 当前预计 `none`
- Requirements impact: `none`
- Specs impact: `clarify`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- Related lessons:
  - `none`

#### Related Requirements / Specs / Lessons
- Related changes for context:
  - `D:\project\MyFlowHub3\docs\change\2026-03-25_win-flow-capability-backend-schema-consumption.md`
  - `D:\project\MyFlowHub3\docs\change\2026-03-25_win-flow-varstore-visual-form-stale.md`

#### Executable Task List
- [ ] `WINVAR-1` 切换 Win `varstore::*` 到 capability schema，并支持 `x-ui-control`
- [ ] `WINVAR-2` 为已存在 call 节点自动补齐 capability schema 缓存
- [ ] `WINVAR-3` 更新 Win resolver / editor 回归测试
- [ ] `WINVAR-4` 澄清 Win stable spec 中的 capability UI 扩展约束
- [ ] `WINVAR-5` 完成 Win 仓归档

#### Task Details
##### `WINVAR-1` - capability schema 驱动 Win varstore ordinary mode
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\plan.md`
- Goal:
  - 删除 `varstore::*` 完整 local override，并让 resolver 能消费 `x-ui-control`。
- Files / Modules:
  - `frontend/src/stores/flow_method_schemas.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow.ts`
- Write Set:
  - Win resolver / capability cache 文件
- Acceptance:
  - `varstore::*` 来源为 capability
  - `varstore::set.value` 仍为 `textarea`
  - 不再依赖 `varstore::*` 本地 override
- Test Points:
  - resolver 单测
- Rollback:
  - 回退 local override 删除与 `x-ui-control` 解析

##### `WINVAR-2` - 选中节点时自动补齐 capability schema
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\plan.md`
- Goal:
  - 避免已有 call 节点因为缺少 route cache 而退回 `Advanced JSON`。
- Files / Modules:
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write Set:
  - 仅限 Flow editor capability cache 相关文件
- Acceptance:
  - 仅对选中 call 节点按 `method + provider` 精确补 capability schema
  - 不污染方法选择器的完整 capability 列表语义
- Test Points:
  - store / window 测试
- Rollback:
  - 回退自动补 capability 逻辑

##### `WINVAR-3` - resolver / editor 回归测试
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\plan.md`
- Goal:
  - 锁定 `varstore::*` capability schema 解析、`x-ui-control`、已有节点 capability hydration 和 fallback 行为。
- Files / Modules:
  - `frontend/src/stores/flow_schema_resolver.test.ts`
  - `frontend/src/stores/flow.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
- Write Set:
  - 仅限相关测试
- Acceptance:
  - 测试明确断言 `source === "capability"`
  - `value` 控件为 `textarea`
  - 已有节点可自动补齐 capability schema
- Test Points:
  - `npm test`
- Rollback:
  - 回退新增测试

##### `WINVAR-4` - Win stable spec 澄清
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\plan.md`
- Goal:
  - 在 Win visual form spec 中记录 `x-ui-control` 扩展支持边界。
- Files / Modules:
  - `docs/specs/flow-editor-visual-form.md`
- Write Set:
  - 仅限 Win stable spec
- Acceptance:
  - spec 清楚描述支持的扩展和 fallback 规则
- Test Points:
  - 文档自洽检查
- Rollback:
  - 回退 spec 澄清

##### `WINVAR-5` - 仓内归档
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\plan.md`
- Goal:
  - 记录 Win 侧从 local override 到 capability schema 的迁移结果。
- Files / Modules:
  - `docs/change/*.md`
  - `docs/change/README.md`
  - `plan.md`
- Write Set:
  - 仅限 docs 与 plan
- Acceptance:
  - 归档可检索到 `varstore`、`x-ui-control`、`output_schema`
- Test Points:
  - change 索引包含新文档
- Rollback:
  - 回退新增归档

#### Dependencies
- `WINVAR-1` 依赖 SubProto `SUBVAR-1` 提供稳定 `varstore::* input_schema/output_schema`
- `WINVAR-2` 依赖 `WINVAR-1`
- `WINVAR-3` 依赖 `WINVAR-1` 和 `WINVAR-2`
- `WINVAR-4` 依赖设计定稿
- `WINVAR-5` 依赖实现与验证结果明确

#### Risks and Notes
- 若 Win 仍优先命中 `varstore::*` 本地 schema，本轮迁移即未完成。
- `output_schema` 本轮只做元数据沉淀，不进入 `node_result` 强校验，避免扩大变更面。
- 若自动补 capability 直接覆盖当前缓存列表，方法选择器可能误显示成“只有当前方法”；必须采用精确查询 + 合并缓存。

#### Parallelism Assessment
- 评估结论：不派发子Agent。
- 原因：
  - 当前 host 规则下未获得用户对子Agent的显式授权。
  - Win 与 SubProto 写集虽分仓，但设计决策和验证链路高度耦合，主Agent顺序推进更稳。

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Execution Summary
- `WINVAR-1`
  - 已删除 `varstore::*` 本地 override，并让 resolver 消费 capability schema + `x-ui-control: "textarea"`。
- `WINVAR-2`
  - 已在 store 增加 `ensureNodeCapabilityLoaded()`，对选中 call 节点按 `method + providerNode` 精确补齐 capability schema，并只做缓存合并。
  - 已在 `FlowEditorWindow.vue` 增加节点级静默 hydration，避免已有节点在项目重开后退回 `Advanced JSON`。
- `WINVAR-3`
  - 已补 resolver / store / window 回归测试，覆盖 capability 来源、textarea、旧节点 hydration。
- `WINVAR-4`
  - 已更新 `docs/specs/flow-editor-visual-form.md`，记录 `x-ui-control` 白名单和 fallback 规则。

#### Validation
- `npm test`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\frontend`
  - result: passed (`6` files / `19` tests)

#### Rollback Notes
- 若需要回退，可按任务边界回退：
  - `WINVAR-1/2`：回退 `flow_method_schemas.ts`、`flow_schema_resolver.ts`、`flow.ts`、`FlowEditorWindow.vue`
  - `WINVAR-3`：回退新增测试
  - `WINVAR-4`：回退 spec 澄清

### Stage 3.3 - Code Review
#### Review Iteration Note
- 首轮 review 发现 `execCapabilitiesLoading` 在“图重置 + 旧请求延迟返回”场景下存在竞争风险，已回退到 `3.2` 补充 `load epoch` 防护后重新验证。

#### Checklist
- 需求覆盖：通过
  - `varstore::*` 已改为 capability schema 驱动，`textarea` 与旧节点 ordinary mode 均保留。
- 架构合理性：通过
  - 真实 schema 来源收敛到后端；Win 仅保留受限 UI hint 消费与节点级 hydration。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅在选中 call 节点且缺少 route 时精确查询一次，不做全图预取；并发 loading 竞争已加 epoch 防护。
- 可读性与一致性：通过
  - 逻辑集中在 `flow.ts` / `FlowEditorWindow.vue`，命名与现有 capability 查询路径一致。
- 可扩展性与配置化：通过
  - `x-ui-control` 继续走白名单扩展，后续新增控件可渐进补充。
- 稳定性与安全：通过
  - capability 缺失时仍安全回退 `Advanced JSON`；项目切换或图重置不会被旧 hydration 请求污染状态。
- 测试覆盖情况：通过
  - resolver、store、window 回归均已覆盖，`npm test` 全量通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未派发子Agent；任务、写集、验证和回滚点均由主Agent记录。

### Stage 4 - Change Archive
#### Docs Routing Confirmation
- 使用 `$m-docs` 校验归档落点、requirements/specs impact 和索引维护。
- Requirements impact: `none`
- Specs impact: `updated`
- Lessons impact: `none`

#### Archive Outputs
- `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\docs\change\2026-03-25_win-varstore-capability-schema.md`
- `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\docs\change\README.md`

#### Notes
- 当前无需新增 lessons；排查线索已写入本次 change，且问题主要是本轮迁移实现细节，不构成独立长期 troubleshooting 入口。
