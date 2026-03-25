# Plan - subproto-varstore-capability-schema

## Workflow Information
- Repo: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema`
- Branch: `feat/subproto-varstore-capability-schema`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`: 已读取；commit 信息使用中文，子协议长期文档以 `repo\MyFlowHub-Server\docs` 为准，worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`。
- base/worktree confirmation:
  - Participating repo: `repo\MyFlowHub-SubProto`
  - Active branch: `feat/subproto-varstore-capability-schema`
  - Active worktree: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema`
  - Dependent consumer worktree:
    - `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema`
    - branch: `feat/win-varstore-capability-schema`
    - plan: `D:\project\MyFlowHub3\worktrees\win-varstore-capability-schema\plan.md`

### Stage 1 - Requirements Analysis
#### Goal
- 为 `varstore::get/set/revoke` capability descriptor 补齐稳定的 `input_schema/output_schema`，让下游 Win ordinary mode 可完全依赖后端 schema。

#### Scope
- Must:
  - `varstore::get/set/revoke` descriptor 注册 `input_schema`。
  - `varstore::get/set/revoke` descriptor 注册 `output_schema`。
  - `varstore::set` 的 schema 中提供 `x-ui-control: "textarea"` 展示提示。
  - 测试锁定 capability schema 已注册，且与真实 invoke 参数/返回结构一致。
- Optional:
  - 如测试需要，可补 schema 内容断言 helper。
- Out of scope:
  - 本轮不改变 `varstore` 实际业务入参与返回语义。
  - 本轮不修改 `exec` registry wire 协议。
  - 本轮不做 `output_schema` 的 consumer 端校验逻辑。

#### Use Cases
- Win 通过 `cap_query(include_schema=true)` 获取 `varstore::*` schema 并生成 ordinary mode 表单。
- 后续其他 consumer 也能从 capability registry 读取 `varstore` 的输入输出元数据。

#### Functional Requirements
1. `varstore::get` 的 `input_schema` 必须覆盖 `owner/name`。
2. `varstore::set` 的 `input_schema` 必须覆盖 `owner/name/value/type/visibility`，并为 `visibility` 提供 `enum/default`。
3. `varstore::revoke` 的 `input_schema` 必须覆盖 `owner/name`。
4. `get/set` 的 `output_schema` 必须覆盖 `owner/name/value/type/visibility/is_public`。
5. `revoke` 的 `output_schema` 必须覆盖 `owner/name/deleted`。

#### Non-functional Requirements
- schema 必须与当前 handler 真实行为严格一致。
- 不引入新的运行时分支或多余 I/O。
- vendor extension 仅限最小必要的 `x-ui-control`。

#### Inputs / Outputs
- Inputs:
  - `varstore` 现有 capability args/result 语义
  - Win consumer 对 `x-ui-control` 的约定
- Outputs:
  - descriptor 上的 `input_schema/output_schema`
  - schema 注册回归测试

#### Edge Cases
- `value` 当前必须继续是 `string`。
- `visibility` 缺省时仍由运行时回退到 `private`。
- `output_schema` 需要同时覆盖 `set` 和 `get` 的公共返回结构。

#### Acceptance Criteria
1. capability registry 中 `varstore::*` descriptor 带上 schema。
2. schema 与当前 invoke 行为匹配，不引入额外字段漂移。
3. `varstore` 相关测试通过。

#### Risks
- 若 schema 与真实 handler 语义不一致，会让下游 consumer 得到错误表单或错误结果提示。
- 若 vendor extension 超出当前 Win 支持边界，consumer 会忽略或退回 fallback。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 直接在 `varstore` handler 内以常量 `json.RawMessage` 形式声明 `input_schema/output_schema`，并在 capability descriptor 注册时挂载；测试从 registry 读取 descriptor 并断言 schema 内容。

#### Alternatives Considered
- 继续不提供 schema，仅由 Win 本地 override 兜底：
  - 不采用；违背当前后端 schema 真相源方向。
- 将 schema 动态由 Go struct 反射生成：
  - 不采用；当前字段少且稳定，常量 schema 更直接、可审计。

#### Module Responsibilities
- `varstore/varstore.go`
  - 定义并注册 capability input/output schema。
- `varstore/capability_provider_test.go`
  - 断言 descriptor schema 与 invoke 行为。

#### Data / Call Flow
- handler 初始化
  - `registerCapabilities()`
  - registry 注册 descriptor with schema
  - `cap_query(include_schema=true)` 可返回 schema
  - consumer 消费

#### Interface Drafts
- `input_schema`
  - `get`: `owner`, `name`
  - `set`: `owner`, `name`, `value`, `type`, `visibility`
  - `revoke`: `owner`, `name`
- `output_schema`
  - `get/set`: record object
  - `revoke`: revoke result object

#### Error Handling and Safety
- schema 常量只描述现有语义，不新增行为。
- 测试必须覆盖 enum/default 与返回字段结构，防止漂移。

#### Performance and Testing Strategy
- 性能：
  - 常量 `json.RawMessage`，不引入运行时生成成本。
- 测试：
  - `varstore/capability_provider_test.go`

#### Extensibility Design Points
- 后续如更多 consumer 需要 UI hint，可沿用同一 vendor extension 模式。
- `output_schema` 为后续结果提示或 schema-aware tooling 预留基础。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 `varstore::*` 虽已注册 capability，但 descriptor 还没有 schema，导致 Win 只能靠本地 override。
- 本轮目标是在不改业务语义的前提下，补齐后端 schema 并为 Win 迁移提供稳定输入。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Docs tree bootstrapping / repair: `none`
- Canonical destinations:
  - Stable truth: 现有 `repo\MyFlowHub-Server\docs\specs\exec.md` 作为 capability 契约参考；本轮预计不新增 SubProto 稳定文档
  - Workflow control: worktree 根 `plan.md`
  - Completed result: `docs/change/YYYY-MM-DD_topic.md`
  - Reusable troubleshooting: 当前预计 `none`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto\docs\flow-exec-capability-contract.md`
- Related lessons:
  - `none`

#### Related Requirements / Specs / Lessons
- Related changes for context:
  - `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema\docs\change\2026-03-25_subproto-capability-input-schema.md`

#### Executable Task List
- [ ] `SUBVAR-1` 为 `varstore::*` 注册 `input_schema/output_schema`
- [ ] `SUBVAR-2` 更新 capability provider 测试
- [ ] `SUBVAR-3` 完成 SubProto 仓归档

#### Task Details
##### `SUBVAR-1` - 注册 varstore capability schema
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema\plan.md`
- Goal:
  - 在不改 handler 业务语义的前提下，为 `varstore::*` descriptor 补齐 schema。
- Files / Modules:
  - `varstore/varstore.go`
- Write Set:
  - 仅限 varstore handler
- Acceptance:
  - registry descriptor 含 schema
  - schema 与真实 args/result 一致
- Test Points:
  - capability provider 测试
- Rollback:
  - 回退 schema 常量与 descriptor 变更

##### `SUBVAR-2` - schema 回归测试
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema\plan.md`
- Goal:
  - 锁定 descriptor schema 内容和 invoke 结果结构。
- Files / Modules:
  - `varstore/capability_provider_test.go`
- Write Set:
  - 仅限 varstore 测试
- Acceptance:
  - 测试断言 input/output schema 关键字段
- Test Points:
  - `go test` 相关模块
- Rollback:
  - 回退新增测试

##### `SUBVAR-3` - 仓内归档
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema\plan.md`
- Goal:
  - 记录 `varstore::*` capability schema 补齐结果。
- Files / Modules:
  - `docs/change/*.md`
  - `docs/change/README.md`
  - `plan.md`
- Write Set:
  - 仅限 docs 与 plan
- Acceptance:
  - change 文档可检索到 `varstore`、`output_schema`、`x-ui-control`
- Test Points:
  - change 索引包含新文档
- Rollback:
  - 回退新增归档

#### Dependencies
- `SUBVAR-2` 依赖 `SUBVAR-1`
- `SUBVAR-3` 依赖实现与验证结果明确
- Win `WINVAR-1` 依赖 `SUBVAR-1`

#### Risks and Notes
- 若 schema 常量与真实返回漂移，会直接误导 consumer。
- `x-ui-control` 在 SubProto 侧只作为透传扩展，不参与任何业务逻辑。

#### Parallelism Assessment
- 评估结论：不派发子Agent。
- 原因：
  - 当前 host 规则下未获得用户对子Agent的显式授权。
  - SubProto 与 Win 设计联动紧密，主Agent顺序推进更稳。

#### Issue List
- none

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### Task Execution Summary
- `SUBVAR-1`
  - 已在 `varstore/varstore.go` 为 `varstore::get/set/revoke` 注册 `input_schema/output_schema`。
  - `varstore::set.value` 已透传 `x-ui-control: "textarea"`。
- `SUBVAR-2`
  - 已在 `varstore/capability_provider_test.go` 补 required/default/enum/output schema 断言。
  - `varstore/go.sum` 已补齐 `exec` module 校验项，保证本地模块验证依赖完整。

#### Validation
- `go test ./...`
  - workdir: `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema\varstore`
  - result: passed
  - note: 使用临时 `GOWORK` 指向同 worktree 内的 `varstore` 与 `exec` module；临时文件已删除。

#### Rollback Notes
- 若需要回退，可按任务边界回退：
  - `SUBVAR-1`：回退 `varstore/varstore.go`
  - `SUBVAR-2`：回退 `varstore/capability_provider_test.go` 与 `varstore/go.sum`

### Stage 3.3 - Code Review
#### Checklist
- 需求覆盖：通过
  - `varstore::get/set/revoke` 已全部注册输入输出 schema，并保留 `textarea` UI hint。
- 架构合理性：通过
  - schema 常量直接挂到 capability descriptor，契约清晰，不引入额外运行时分支。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - schema 为静态 `json.RawMessage` 常量，无额外 I/O 和运行时构造成本。
- 可读性与一致性：通过
  - schema 命名与方法名一一对应，测试 helper 复用度足够。
- 可扩展性与配置化：通过
  - `x-ui-control` 保持最小白名单；后续更多 consumer 可继续复用 registry schema。
- 稳定性与安全：通过
  - schema 未改变任何 handler 业务语义，测试锁定了关键字段和默认值。
- 测试覆盖情况：通过
  - capability provider 测试和本地 `go test ./...` 验证已通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未派发子Agent；实现、验证和归档均由主Agent完成。

### Stage 4 - Change Archive
#### Docs Routing Confirmation
- 使用 `$m-docs` 校验归档落点、requirements/specs impact 和索引维护。
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`

#### Archive Outputs
- `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema\docs\change\2026-03-25_subproto-varstore-capability-schema.md`
- `D:\project\MyFlowHub3\worktrees\subproto-varstore-capability-schema\docs\change\README.md`

#### Notes
- 当前无需新增 lessons；`go.work` 验证约束已在本次 change 中记录，暂未上升为长期文档规则。
