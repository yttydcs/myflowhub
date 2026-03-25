# Plan - subproto-capability-input-schema

## Workflow Information
- Repo: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema`
- Branch: `feat/subproto-capability-input-schema`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`: 不适用（本 repo 无独立 `guide.md`）。
- base/worktree confirmation:
  - Participating repo: `repo\MyFlowHub-SubProto`
  - Active branch: `feat/subproto-capability-input-schema`
  - Active worktree: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema`
  - Cross-repo dependency:
    - Win worktree: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux`
    - Win branch: `fix/win-capability-picker-ux`
    - Win plan: `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\plan.md`

### Stage 1 - Requirements Analysis
#### Goal
- 为第一批常用 capability 补齐 `InputSchema`，让 Win Flow 编辑器 ordinary mode 可以直接基于 capability schema 生成表单。

#### Scope
- Must:
  - 为 `topicbus::publish`、`file::mkdir`、`file::list`、`file::read_text` 注册 capability 时补 `InputSchema`。
  - schema 必须与当前 invoke 函数实际接受的 JSON 参数一致。
  - schema 必须落在 Win 端当前支持的 JSON Schema 子集内。
- Optional:
  - 为部分 capability 补最小 `OutputSchema`，提升 UI metadata 完整性。
- Out of scope:
  - 本轮不迁移 `varstore::*` 到后端 schema，继续保留 Win 前端 override。
  - 本轮不扩展复杂 schema 特性（`oneOf/anyOf/allOf/$ref/array`）。

#### Use Cases
- 用户为 `topicbus::publish` 节点填写 topic、name、payload，而不手写 JSON。
- 用户为 `file::mkdir` / `file::list` / `file::read_text` 节点在 Flow 编辑器中直接配置参数。

#### Functional Requirements
1. 目标 capability 必须经 `cap_query(include_schema=true)` 返回可解析的 `input_schema`。
2. schema 字段名、类型、required 约束必须与 invoke 函数真实入参一致。
3. 不得改变 capability 方法名、权限和既有运行时执行语义。

#### Non-functional Requirements
- 仅作 capability metadata 扩展，不引入新的网络 action 或路由逻辑。
- schema 应保持简洁、稳定、可跨端复用。

#### Inputs / Outputs
- Inputs:
  - `topicbus.PublishReq`
  - `file.ReadReq`
  - `file.WriteReq`
  - capability registry descriptor
- Outputs:
  - 带 `InputSchema` 的 capability descriptor
  - `cap_query_resp.routes[].input_schema`

#### Edge Cases
- 可选字段必须允许为空或省略。
- `file::list` 与 `file::read_text` 共用 `ReadReq`，但 schema 只暴露当前 capability 真正需要的字段。
- `topicbus::publish.payload` 需要兼容任意 JSON 对象输入。

#### Acceptance Criteria
1. 目标 capability 在 registry 中带有合法 `InputSchema`。
2. `cap_query(include_schema=true)` 返回的 route 带有对应 schema。
3. SubProto 侧自动化测试覆盖 descriptor/schema 暴露结果。

#### Risks
- 若 schema 与 invoke 参数不一致，Win ordinary mode 会生成错误字段，导致运行时调用失败。
- 若 schema 误用当前前端不支持的特性，会继续被 Win resolver 拒绝。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 直接在 capability 注册点为目标方法补 `InputSchema` 常量，复用现有 `execcap.Descriptor` 和 `cap_query(include_schema=true)` 传输链路。

#### Alternatives Considered
- 备选：全部继续在 Win 前端补 local override。
- 不采用原因：schema 真相会继续散落在 UI 侧，无法被其他端复用，也不符合 capability registry 作为统一元数据源的方向。

#### Module Responsibilities
- `topicbus/topicbus.go`
  - 为 `topicbus::publish` 提供 capability schema。
- `file/handler.go`
  - 为 `file::mkdir`、`file::list`、`file::read_text` 提供 capability schema。
- `exec/handler.go` / capability registry
  - 继续负责把 descriptor 中的 schema 透传到 `cap_query_resp`。
- 测试文件
  - 校验 registry/list/query route 中 schema 存在且结构正确。

#### Data / Call Flow
- `execcap.Registry.Register(Descriptor{ InputSchema: ... })`
  - `exec` handler 聚合 local capability
  - `cap_query(include_schema=true)`
  - Win `flow.ts` 接收 `route.inputSchema`
  - Win resolver 解析为 visual schema

#### Interface Drafts
- 无新增 action / protocol。
- 仅扩充已有 `Descriptor.InputSchema` 内容。

#### Error Handling and Safety
- schema 只作为 metadata，不改变 invoke 路径和权限校验。
- 避免使用 Win 当前不支持的复杂 schema 特性。

#### Performance and Testing Strategy
- 性能：
  - 仅增加少量 capability metadata，查询路径无额外 I/O。
- 测试：
  - Go 测试覆盖 registry / handler route 中 schema 暴露。
  - 需要与 Win 侧 resolver 回归联合验证。

#### Extensibility Design Points
- 后续其他 capability 继续沿同一模式补 `InputSchema` / `OutputSchema`。
- `varstore::*` 未来可迁移到后端 schema，但应在字段体验与现有 Win override 对齐后单独做。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 capability registry 已支持 schema 字段，但多数 capability 实际没有填 `InputSchema`，导致 Win ordinary mode 对这些方法仍只能回退到 `Advanced JSON`。
- 本轮在 SubProto 侧补齐第一批 capability schema，并由 Win worktree 负责消费验证。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 完成检查。
- Docs tree bootstrapping / repair:
  - `docs/change/README.md` 当前缺失，Stage 4 归档时一并补齐。
- Canonical destinations:
  - Stable truth: 继续以 Win requirements/specs + Server `docs/specs/exec.md` 为准
  - Workflow control: worktree 根 `plan.md`
  - Completed result: `docs/change/YYYY-MM-DD_topic.md`
  - Reusable troubleshooting: 当前预计 `none`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema\docs\flow-exec-capability-contract.md`
- Related lessons:
  - `none`

#### Related Requirements / Specs / Lessons
- Related changes for context:
  - `D:\project\MyFlowHub3\docs\change\2026-03-18_exec-capability-registry-mvp-runtime.md`
  - `D:\project\MyFlowHub3\docs\change\2026-03-18_exec-capability-registry-upstream-sync-query-fallback.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\change\2026-03-22_win-call-visual-form.md`

#### Executable Task List
- [x] `SUBCAP-1` 为目标 capability 补 `InputSchema`
- [x] `SUBCAP-2` 为 schema 暴露链路补 Go 测试
- [x] `SUBCAP-3` 完成本仓归档与索引修复

#### Task Details
##### `SUBCAP-1` - capability 注册补 schema
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema\plan.md`
- Goal:
  - 为 `topicbus::publish`、`file::mkdir`、`file::list`、`file::read_text` 注册 capability 时补 `InputSchema`。
- Files / Modules:
  - `topicbus/topicbus.go`
  - `file/handler.go`
- Write Set:
  - 仅限上述 capability 提供方文件
- Acceptance:
  - descriptor 中带合法 `InputSchema`
  - 不改变 invoke 逻辑和权限
- Test Points:
  - capability registry / query 路径读取 schema
- Rollback:
  - 回退新增 schema 常量与 descriptor 字段

##### `SUBCAP-2` - Go 测试覆盖 schema 暴露
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema\plan.md`
- Goal:
  - 锁定目标 capability 的 schema 已注册并可被 query 返回。
- Files / Modules:
  - `topicbus/*test.go`
  - `file/*test.go`
  - 如需透过 exec handler 校验，可增量修改 `exec/*test.go`
- Write Set:
  - 仅限相关 Go 测试文件
- Acceptance:
  - `go test` 覆盖 schema 注册/暴露结果
- Test Points:
  - registry/list/query
- Rollback:
  - 回退新增测试文件和断言

##### `SUBCAP-3` - 文档归档
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema\plan.md`
- Goal:
  - 为本仓补完成果归档，并修复缺失的 `docs/change/README.md` 入口。
- Files / Modules:
  - `docs/change/*.md`
  - `docs/change/README.md`
  - `plan.md`
- Write Set:
  - 仅限 docs 与 plan
- Acceptance:
  - change 文档可导航、可追溯
- Test Points:
  - 索引包含新归档
- Rollback:
  - 回退新增 docs 叶子和 README

#### Dependencies
- `SUBCAP-2` 依赖 `SUBCAP-1`
- Win worktree 的消费验证依赖 `SUBCAP-1`

#### Risks and Notes
- 若 schema 与 proto 请求结构不一致，会直接影响 Win 端 ordinary mode 生成字段的正确性。
- `varstore::*` 继续保留前端 override，不在本轮迁移，避免引入双重变更面。

#### Parallelism Assessment
- 评估结论：本轮不派发子Agent。
- 原因：
  - 当前阶段为 `3.1`，按 `$m-autoflow` 规则禁止派发子Agent。
  - 当前写集在 SubProto 能力提供方和 Win 消费侧有强依赖，先由主Agent 顺序落地更稳。

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
- 使用 `$m-docs` 完成归档与索引修复。
- Change archive:
  - `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema\docs\change\2026-03-25_subproto-capability-input-schema.md`
  - `D:\project\MyFlowHub3\worktrees\subproto-capability-input-schema\docs\change\README.md`
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- Validation summary:
  - `go test ./...` in `file`: 通过
  - `go test ./...` in `topicbus`: 通过
  - `go test ./...` in `exec`: 通过
  - 为验证临时创建 worktree 根 `go.work` 指向本地模块，测试后已删除

阻塞：否
进入归档完成，等待用户确认是否结束 workflow
