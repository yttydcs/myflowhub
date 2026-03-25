# Plan - win-capability-picker-ux

## Workflow Information
- Repo: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux`
- Branch: `fix/win-capability-picker-ux`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`: 已读取；commit 信息使用中文，worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`，可使用 chrome-devtools 做界面验证，子协议文档以 `repo\MyFlowHub-Server\docs` 为准。
- base/worktree confirmation:
  - Participating repo: `repo\MyFlowHub-Win`
  - Active branch: `fix/win-capability-picker-ux`
  - Active worktree: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux`
  - Cross-repo dependency:
    - SubProto worktree: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema`
    - SubProto branch: `feat/subproto-capability-input-schema`
    - SubProto plan: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema\plan.md`

### Stage 1 - Requirements Analysis
#### Goal
- 验证并锁定 Win Flow 编辑器对后端 capability `input_schema` 的消费结果，使第一批后端补 schema 的方法可以在 ordinary mode 直接显示表单输入。

#### Scope
- Must:
  - 验证 `topicbus::publish`、`file::mkdir`、`file::list`、`file::read_text` 的 capability schema 能被 Win resolver 正确解析。
  - 补最小回归测试，覆盖 schema 解析后的字段定义。
  - 不回退前一轮已完成的 capability picker UX 修复。
- Optional:
  - 如发现 Win 侧对目标 schema 的兼容性缺口，可做最小兼容性修正。
- Out of scope:
  - 本轮不迁移 `varstore::*` override 到后端。
  - 本轮不扩展 Win resolver 对复杂 JSON Schema 特性的支持。

#### Use Cases
- 用户为 `topicbus::publish` 节点在 ordinary mode 直接填写 `topic`、`name`、`ts`、`payload`。
- 用户为 `file::mkdir`、`file::list`、`file::read_text` 节点在 ordinary mode 直接配置目录和文件参数。

#### Functional Requirements
1. Win 端继续按“本地 override 优先，后端 `input_schema` 兜底”的顺序解析 schema。
2. 目标 capability 若返回 Win 支持子集内的 `input_schema`，ordinary mode 必须可生成字段表单。
3. 若 Win 侧不需要业务代码修改，也必须有自动化测试锁定消费结果。

#### Non-functional Requirements
- 维持最小安全改动，不改变 Flow 保存格式与运行时契约。
- 不新增额外 capability 请求或重复计算。

#### Inputs / Outputs
- Inputs:
  - capability route `input_schema`
  - Win `resolveMethodVisualSchema(...)`
- Outputs:
  - 可解析的 visual schema
  - 对应回归测试

#### Edge Cases
- `payload` 需要作为 JSON 对象字段处理。
- `file::list` 没有 required 字段，仍应生成可用表单。
- 后端 schema 为空或非法时，仍需按现有 fallback 只显示 `Advanced JSON`。

#### Acceptance Criteria
1. 目标 4 个 capability 在 Win resolver 下都能解析成 visual schema。
2. 自动化测试覆盖关键字段、required 约束和控件类型。
3. 前一轮 capability picker UX 修复不回归。

#### Risks
- 若后端 schema 与 Win 支持子集不匹配，ordinary mode 仍会不可用。
- 若 Win 测试只验证 schema 存在而不验证字段结构，后续回归不易发现。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 不新增新的 Win 业务逻辑路径，直接复用现有 `flow_schema_resolver.ts` 的 capability schema 解析能力，通过测试锁定目标 capability 的消费结果；只有在发现兼容性缺口时才做最小代码修正。

#### Alternatives Considered
- 备选：继续在 Win 本地补硬编码 schema。
- 不采用原因：会再次分裂 schema 真相来源，和本轮“以后端 capability registry 为主”的方向冲突。

#### Module Responsibilities
- `frontend/src/stores/flow_schema_resolver.ts`
  - 继续负责把 capability `input_schema` 转换为 visual schema。
- `frontend/src/stores/flow_schema_resolver.test.ts`
  - 锁定第一批后端 schema 的 Win 消费结果。
- 现有 `FlowEditorWindow` / `FlowNodeInspector`
  - 保持上一轮 UX 修复结果，不做额外行为扩张。

#### Data / Call Flow
- `cap_query(include_schema=true)`
  - `route.inputSchema`
  - `resolveMethodVisualSchema(method, capability)`
  - ordinary mode visual schema

#### Interface Drafts
- 无新增接口。
- 若无兼容性缺口，Win 侧只增加测试，不改对外行为。

#### Error Handling and Safety
- 继续沿用现有 schema 解析失败回退逻辑。
- 不放宽对复杂 schema 特性的拒绝条件。

#### Performance and Testing Strategy
- 性能：
  - 仅增加解析测试，不引入运行时开销。
- 测试：
  - 新增 `flow_schema_resolver` 测试覆盖 4 个 capability schema 的字段映射。
  - 回归执行 `cd frontend && npm test`。

#### Extensibility Design Points
- 让 capability schema onboarding 通过测试批量接入，后续新增方法时可沿同一模式扩展。
- 保留本地 override 仅用于 `varstore::*` 这类确有 UI 特例的方法。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Win 端 capability picker UX 修复已完成；当前缺口是第一批 capability 虽然即将由后端补 `input_schema`，但 Win worktree 还没有针对这些 schema 的消费验证。
- 本轮目标是在不扩大 Win 变更面的前提下，锁定后端 schema 接入后的 ordinary mode 能力。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 完成检查。
- Docs tree bootstrapping / repair: `none`
- Canonical destinations:
  - Stable truth: `docs/requirements/flow-editor-visual-form.md` 与 `docs/specs/flow-editor-visual-form.md`
  - Workflow control: worktree 根 `plan.md`
  - Completed result: `docs/change/YYYY-MM-DD_topic.md`
  - Reusable troubleshooting: 当前预计 `none`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`

#### Related Requirements / Specs / Lessons
- Related changes for context:
  - `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\change\2026-03-24_win-flow-editor-visual-form-ux.md`
  - `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\change\2026-03-25_win-flow-capability-picker-followup.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema\docs\flow-exec-capability-contract.md`

#### Executable Task List
- [x] `WINSCHEMA-1` 为第一批后端 capability schema 补 Win 消费验证
- [x] `WINSCHEMA-2` 如有必要做最小兼容性修正并回归测试
- [x] `WINSCHEMA-3` 完成本仓变更归档

#### Task Details
##### `WINSCHEMA-1` - 补 schema resolver 测试
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\plan.md`
- Goal:
  - 锁定 `topicbus::publish`、`file::mkdir`、`file::list`、`file::read_text` 的 capability schema 在 Win 的解析结果。
- Files / Modules:
  - `frontend/src/stores/flow_schema_resolver.test.ts`
- Write Set:
  - 仅限 resolver 测试文件
- Acceptance:
  - 覆盖字段顺序、required、控件类型、JSON 对象字段映射
- Test Points:
  - `vitest`
- Rollback:
  - 回退新增测试用例

##### `WINSCHEMA-2` - 最小兼容性修正
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\plan.md`
- Goal:
  - 仅在测试证明现有 resolver 无法消费目标 schema 时，做最小代码修正。
- Files / Modules:
  - `frontend/src/stores/flow_schema_resolver.ts`
  - 如需联动，允许修改相关单测
- Write Set:
  - 仅限 resolver 与对应测试
- Acceptance:
  - 不影响现有 override 逻辑
  - 目标 schema 全部可解析
- Test Points:
  - `vitest`
- Rollback:
  - 回退兼容性修正与对应测试

##### `WINSCHEMA-3` - 归档
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\plan.md`
- Goal:
  - 记录本轮 Win 侧“后端 schema 消费验证”结果。
- Files / Modules:
  - `docs/change/*.md`
  - `docs/change/README.md`
  - `plan.md`
- Write Set:
  - 仅限 docs 与 plan
- Acceptance:
  - 归档可导航，可回溯到 SubProto 配套变更
- Test Points:
  - `README.md` 索引包含新文档
- Rollback:
  - 回退新增归档

#### Dependencies
- `WINSCHEMA-1` 依赖 SubProto `SUBCAP-1` 输出稳定 schema。
- `WINSCHEMA-2` 仅在 `WINSCHEMA-1` 发现兼容性缺口时启用。
- `WINSCHEMA-3` 依赖验证结果明确。

#### Risks and Notes
- 预计 Win 本轮多数情况下只需要补测试；若实际出现兼容性缺口，需要回到本计划内最小修正，不得扩到 resolver 能力泛化重构。
- `varstore::*` 继续保留前端 override，本轮不迁移。

#### Parallelism Assessment
- 评估结论：本轮不派发子Agent。
- 原因：
  - Win 写集集中在 resolver 与文档，拆分收益低。
  - SubProto 结果是 Win 验证的前置依赖，主Agent 顺序推进更稳。

#### Issue List
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过

### Stage 4 - Change Archive
- 使用 `$m-docs` 完成归档。
- Change archive:
  - `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\change\2026-03-25_win-flow-capability-backend-schema-consumption.md`
  - `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\change\README.md`
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- Validation summary:
  - `npx vitest run src/stores/flow_schema_resolver.test.ts`: 通过
  - `npm test`: 通过（6 个测试文件，16 个用例）
  - `WINSCHEMA-2` 未触发业务代码修正，现有 resolver 已可消费目标 schema

阻塞：否
进入归档完成，等待用户确认是否结束 workflow
