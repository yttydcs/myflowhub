# Plan - fix-win-varstore-visual-form-stale

## Workflow Information
- Repo: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale`
- Branch: `fix/win-varstore-visual-form-stale`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`: 已读取；commit 信息使用中文，worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`，可使用 chrome-devtools 做界面验证。
- base/worktree confirmation:
  - Participating repo: `repo\MyFlowHub-Win`
  - Active branch: `fix/win-varstore-visual-form-stale`
  - Active worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale`

### Stage 1 - Requirements Analysis
#### Goal
- 修复 Flow 编辑器中已存在 `call` 节点切换到 `varstore::get` 后 ordinary mode 仍因旧字段残留而不可用的问题。

#### Scope
- Must:
  - 当 `call` 节点在 form 模式下切换方法时，按新方法的 visual schema 收敛 `args_template` 和 `inputs`。
  - 对 `varstore::set -> varstore::get` 场景，移除 `/visibility` 这类新 schema 不支持的旧字段残留。
  - 保留新旧 schema 的交集字段值和仍有效的 binding。
  - 补自动化测试覆盖方法切换后的残留清理行为。
- Optional:
  - 如发现同一逻辑还能顺带修正其他 `call` 方法切换残留，允许以同一 helper 覆盖。
- Out of scope:
  - 本轮不迁移 `varstore::*` schema 到后端。
  - 本轮不修改 ordinary mode 的兼容性判定规则。
  - 本轮不处理用户手动在 `Advanced JSON` 中保留的任意高级字段。

#### Use Cases
- 用户把一个原本配置为 `varstore::set` 的节点改成 `varstore::get`，希望 ordinary mode 继续可用，而不是被旧的 `/visibility` 卡住。
- 用户切换方法后，希望 `owner`、`name` 这类交集字段被保留，不必重填。

#### Functional Requirements
1. form 模式下显式切换方法时，编辑器必须按目标方法 schema 清理不再受支持的 literal 字段和 binding。
2. 目标 schema 中仍存在的字段值必须保留。
3. 目标 schema 中仍存在的 binding 必须保留。
4. 切方法后 ordinary mode 若其余条件满足，必须继续可用。

#### Non-functional Requirements
- 采用最小安全改动，不扩大到 runtime、协议或 visual compatibility 判定规则。
- 不引入额外 I/O 或全图重建。
- helper 设计应支持后续其他方法切换复用。

#### Inputs / Outputs
- Inputs:
  - `selected.method`
  - `selected.argsTemplate`
  - `selected.inputs`
  - 新方法解析得到的 visual schema
- Outputs:
  - 收敛后的节点 form 状态
  - 方法切换回归测试

#### Edge Cases
- 新方法没有 visual schema 时，不应盲目清理旧字段。
- form 模式切换时，嵌套 JSON 字段应按 schema pointer 整体保留。
- blank binding 行不应被误当成有效残留。

#### Acceptance Criteria
1. 现有节点从 `varstore::set` 切到 `varstore::get` 后，不再因 `/visibility` 残留而触发 `extra_literal_field`。
2. `owner`、`name` 等交集字段仍保留。
3. 自动化测试覆盖方法切换残留清理逻辑。

#### Risks
- 若清理逻辑写得过宽，可能误删用户仍想保留的字段。
- 若清理逻辑只删 literal 不删 binding，ordinary mode 仍会因 unknown binding 失败。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 在 `applyCallCapability()` 中，仅对 form 模式的 `call` 节点，在方法实际发生变化且新方法存在 visual schema 时，执行一次基于 schema pointer 的状态收敛：重建允许的 `argsTemplate` 文档并过滤掉不属于新 schema 的 binding，然后再补默认值。

#### Alternatives Considered
- 备选：放宽 compatibility 判定，忽略旧字段。
- 不采用原因：这会破坏“ordinary mode 不得静默忽略未覆盖字段”的边界。
- 备选：切方法时直接把 `argsTemplate/inputs` 全量重置。
- 不采用原因：会把 `owner`、`name` 这类交集字段也清空，用户体验不必要地差。

#### Module Responsibilities
- `frontend/src/stores/flow.ts`
  - 在 capability 应用路径中收敛 form 模式节点状态。
- `frontend/src/stores/flow.test.ts`
  - 锁定 `varstore::set -> varstore::get` 场景的保留/清理结果。

#### Data / Call Flow
- 用户选择新 capability
  - `applyCallCapability()`
  - 解析新方法 visual schema
  - 若 form 模式且方法变化：
    - 保留 schema 内 pointer 对应 literal
    - 过滤未知 binding
    - 应用 schema 默认值
  - 提交历史

#### Interface Drafts
- 无新增对外接口。
- 仅新增 store 内部 helper。

#### Error Handling and Safety
- 新方法无 schema 时，不执行收敛，维持现有行为。
- 仅在 form 模式做此清理，避免对 advanced JSON 用户静默删字段。

#### Performance and Testing Strategy
- 性能：
  - 只遍历目标 schema 字段和当前节点 bindings，不扫描全图。
- 测试：
  - `flow.test.ts` 新增 store 级测试，验证交集字段保留、未知字段和 binding 删除。

#### Extensibility Design Points
- 清理 helper 以 schema pointer 为中心，后续其他本地 override 或 capability schema 方法切换都可复用。
- 不把清理逻辑耦合到 `varstore::*` 特例。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 ordinary mode 对旧节点保护性降级是正确的，但方法切换路径没有同步清理旧 schema 字段，导致 `varstore::set -> varstore::get` 这类切换把 `/visibility` 残留在 `args_template` 里。
- 本轮目标是在不放宽 compatibility 规则的前提下，修正方法切换时的 form 状态收敛。

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
  - `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`

#### Related Requirements / Specs / Lessons
- Related changes for context:
  - `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\change\2026-03-24_win-flow-editor-visual-form-ux.md`
  - `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\change\2026-03-25_win-flow-capability-picker-followup.md`
  - `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\change\2026-03-25_win-flow-capability-backend-schema-consumption.md`

#### Executable Task List
- [x] `VFSTALE-1` 在方法切换路径收敛 form 模式节点状态
- [x] `VFSTALE-2` 为残留字段清理补 store 级回归测试
- [x] `VFSTALE-3` 完成本仓归档

#### Task Details
##### `VFSTALE-1` - 方法切换时收敛 form 状态
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\plan.md`
- Goal:
  - 在 form 模式切换方法时，只保留新 schema 支持的 literal/binding。
- Files / Modules:
  - `frontend/src/stores/flow.ts`
- Write Set:
  - 仅限 flow store
- Acceptance:
  - `varstore::set -> varstore::get` 不再残留 `/visibility`
  - `owner` / `name` 保留
- Test Points:
  - store 级单测
- Rollback:
  - 回退新增 helper 与 `applyCallCapability` 逻辑

##### `VFSTALE-2` - 回归测试
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\plan.md`
- Goal:
  - 锁定方法切换后 schema 外字段和 binding 被清理、交集字段保留。
- Files / Modules:
  - `frontend/src/stores/flow.test.ts`
- Write Set:
  - 仅限 store 测试文件
- Acceptance:
  - 用例稳定覆盖 `varstore::set -> varstore::get`
- Test Points:
  - `vitest`
- Rollback:
  - 回退新增测试

##### `VFSTALE-3` - 文档归档
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\plan.md`
- Goal:
  - 记录本轮 root cause、修复点和验证结果。
- Files / Modules:
  - `docs/change/*.md`
  - `docs/change/README.md`
  - `plan.md`
- Write Set:
  - 仅限 docs 与 plan
- Acceptance:
  - 归档可检索到“方法切换残留字段”问题
- Test Points:
  - README 索引含新文档
- Rollback:
  - 回退新增归档

#### Dependencies
- `VFSTALE-2` 依赖 `VFSTALE-1`
- `VFSTALE-3` 依赖修复与验证结果明确

#### Risks and Notes
- 当前证据更支持“方法切换未清理旧字段”，不是“删除节点后数据仍挂在新节点上”；若实施中发现另有草稿恢复路径问题，需要回到 `3.1` 扩计划。

#### Parallelism Assessment
- 评估结论：本轮不派发子Agent。
- 原因：
  - 写集只在 flow store 与对应测试，拆分收益低。
  - 当前阶段刚完成 `3.1`，主Agent 顺序推进更稳。

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
  - `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\change\2026-03-25_win-flow-varstore-visual-form-stale.md`
  - `D:\project\MyFlowHub3\worktrees\fix-win-varstore-visual-form-stale\docs\change\README.md`
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- Validation summary:
  - `npx vitest run src/stores/flow.test.ts`: 通过（1 个文件，5 个用例）
  - `npm test`: 通过（6 个文件，17 个用例）

阻塞：否
进入归档完成，等待用户确认是否结束 workflow
