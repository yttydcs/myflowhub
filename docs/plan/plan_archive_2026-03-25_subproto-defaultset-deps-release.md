# Plan - defaultset-deps-release

## Workflow Information
- Repo: `MyFlowHub-SubProto`
- Branch: `fix/defaultset-deps-release`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md:
  - 已读取 workspace `D:\project\MyFlowHub3\guide.md`。
- base/worktree confirmation:
  - 当前 repo-local 实现工作树：`D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
  - 对应下游 worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release`
  - 对应上游 worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release`

### Stage 1 - Requirements Analysis
#### Goal
- 为 `broker` / `exec` shared packages 与 `exec/file/flow/topicbus/varstore/management` 正式发布包含 `WithDeps` / `runtimedeps` 的 patch 版本，供 `MyFlowHub-Server` 使用。

#### Scope
- 必须:
  - 确认 `broker` / `exec` shared packages 与 `exec/file/flow/topicbus/varstore/management` 当前 API 适合 patch 发布。
  - 进行 repo-local 联调验证。
  - 创建 repo-level change archive 并维护索引。
- 可选:
  - 补充最小兼容说明。
- 不做:
  - 不新增业务能力。
  - 不变更既有协议 schema。

#### Use Cases
- 下游 `Server` 在单仓 `GOWORK=off` 模式下拉取新版本即可通过构建。

#### Functional Requirements
- `broker` 必须导出 `SharedExecCapQueryBroker`。
- `exec` 必须导出 `capability`、`runtimedeps` 子包。
- `file`、`flow`、`topicbus`、`varstore`、`management` 必须导出 `NewHandlerWithDeps(...)`。
- 新 tag 必须可被 `go list -m` 解析。

#### Non-functional Requirements
- 仅发布最小必要 patch。
- 不引入与本次 release chain 无关的实现改动。

#### Inputs / Outputs
- Inputs:
  - 当前 `main` 上已存在的 `WithDeps` 实现与 `exec` shared packages
  - 下游 `Server` 失败症状
- Outputs:
  - `broker v0.1.1`
  - `exec v0.1.2`
  - `file v0.1.4`
  - `flow v0.1.2`
  - `topicbus v0.1.2`
  - `varstore v0.1.4`
  - `management v0.1.4`
  - repo-level archive

#### Edge Cases
- 若当前 HEAD 同时包含不适合 patch 的破坏性变化，需要停止发布并重新裁切提交。

#### Acceptance Criteria
- `go list -m github.com/yttydcs/myflowhub-subproto/exec@v0.1.2`
- `go list -m github.com/yttydcs/myflowhub-subproto/file@v0.1.4`
- `go list -m github.com/yttydcs/myflowhub-subproto/flow@v0.1.2`
- `go list -m github.com/yttydcs/myflowhub-subproto/topicbus@v0.1.2`
- `go list -m github.com/yttydcs/myflowhub-subproto/varstore@v0.1.4`
- `go list -m github.com/yttydcs/myflowhub-subproto/management@v0.1.4`
- 上游联调测试通过。

#### Risks
- tag 仅本地存在会导致下游仍不可解析。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 不回退代码；直接基于当前含 `WithDeps` / `runtimedeps` 的提交发 patch tag。

#### Alternatives Considered
- 回退 `Server` 到旧 API：放弃。
- 使用 `go.work` / `replace` 作为正式修复：放弃。
- 发布新 patch：采用。

#### Module Responsibilities
- `exec`：
  - 对外提供 `capability` / `runtimedeps` 包。
- `file` / `flow` / `topicbus` / `varstore` / `management`：
  - 对外提供已存在的 `WithDeps` 构造函数。
- repo docs：
  - 记录发布背景、验证与回滚。

#### Data / Call Flow
- `tag` -> `go proxy` / module download -> `Server` bump -> `GOWORK=off` 验证

#### Interface Drafts
- `exec/capability`
- `exec/runtimedeps`
- `file.NewHandlerWithDeps`
- `flow.NewHandlerWithDeps`
- `topicbus.NewHandlerWithDeps`
- `varstore.NewHandlerWithDeps`
- `management.NewHandlerWithDeps`

#### Error Handling and Safety
- 先做 repo-local 联调测试，再发 tag。

#### Performance and Testing Strategy
- 使用 repo-local `go.work` 绑定 `Core/Proto/exec/file/flow/topicbus/varstore/management` 做上游联调测试。
- `go list -m ...@<new-version>`

#### Extensibility Design Points
- 本次继续遵守跨仓 semver 发布顺序，为后续同类修复复用。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 发布 `defaultset` 依赖链所需的 `exec` / `file` / `flow` / `topicbus` / `varstore` / `management` patch 版本，解除 `Server` 的缺包与未定义符号阻塞。
- Current state:
  - 本地源码已有目标 API / packages。
  - 已发布 `broker v0.1.0` 缺失 `SharedExecCapQueryBroker`。
  - 已发布 `exec v0.1.1` 缺失 `capability` / `runtimedeps` 子包。
  - 已发布 `file/flow/topicbus/varstore/management` 版本链未覆盖当前 `WithDeps` 依赖。

#### Docs Governance Routing Decision
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `none`
- Related specs:
  - `none`
- Related lessons:
  - `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`

#### Related Requirements / Specs / Lessons
- 本仓长期真相不在 `requirements/specs`，只做 release archive。

#### Executable Task List
- [x] `SUBREL1` 核对 patch 发布边界
- [x] `SUBREL2` 使用 repo-local `go.work` 完成模块联调验证
- [x] `SUBREL3` 发布 `broker` / `exec` / `file` / `flow` / `topicbus` / `varstore` / `management` tag
- [x] `DOC1` 更新 repo-level change archive 与索引

#### Task Details
##### SUBREL1 - Confirm Patch Release Surface
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release\plan.md`
- Goal: 确认 `broker` / `exec` shared packages 与 `exec/file/flow/topicbus/varstore/management` 当前代码适合 patch 发布。
- Files / Modules:
  - `broker/*`
  - `exec/*`
  - `file/*`
  - `flow/*`
  - `topicbus/*`
  - `varstore/*`
  - `management/*`
- Write Set:
  - release-related repo files only
- Acceptance:
  - 版本目标与测试策略明确。
- Test Points:
  - package / symbol audit
- Rollback:
  - 放弃当前发布并回到 planning

##### SUBREL2 - Validate Release Chain In Repo-Local Workspace
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release\plan.md`
- Goal: 通过 repo-local `go.work` 先验证上游整条模块链可编译可测试。
- Files / Modules:
  - temporary `go.work`
  - `exec/*`
  - `file/*`
  - `flow/*`
  - `topicbus/*`
  - `varstore/*`
  - `management/*`
- Write Set:
  - temporary workspace files only
- Acceptance:
  - 上游联调测试通过，确认 tag 可从当前提交发布。
- Test Points:
  - `go test ./... -count=1 -p 1`
- Rollback:
  - 删除临时 `go.work`

##### SUBREL3 - Publish Patch Tags
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release\plan.md`
- Goal: 创建可解析的新 patch tag。
- Files / Modules:
  - git tags
  - commit metadata if needed
- Write Set:
  - git metadata
- Acceptance:
  - `broker v0.1.1`、`exec v0.1.2`、`file v0.1.4`、`flow v0.1.2`、`topicbus v0.1.2`、`varstore v0.1.4`、`management v0.1.4` 可解析。
- Test Points:
  - `go list -m ...@<new-version>`
- Rollback:
  - 删除未推送本地 tag

##### DOC1 - Archive And Index
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release\plan.md`
- Goal: 记录本次 patch release 的背景与验证结果。
- Files / Modules:
  - `docs/change/*`
  - `docs/change/README.md`
- Write Set:
  - docs archive only
- Acceptance:
  - archive 与索引更新完成。
- Test Points:
  - manual index check
- Rollback:
  - 回退本次 archive/index 变更

#### Dependencies
- `SUBREL1` -> `SUBREL2` -> `SUBREL3` -> `DOC1`

#### Risks and Notes
- `fix/defaultset-deps-release` 与 `broker/exec/file/flow/topicbus/varstore/management` 目标 tags 已推送到 `origin`。
- 本轮已确认不能只发 `file` / `management`，必须一起覆盖 `defaultset` 依赖链。

#### Parallelism Assessment
- Current decision: `不并行实现`
- Reason:
  - 发布动作与验证强顺序依赖，且 write set 不适合拆分。

### Stage 3.2 - Implementation Record
- `SUBREL1`
  - 已确认真正缺口不是单独两个 `WithDeps` 符号，而是 `Proto -> broker -> exec -> file/flow/topicbus/varstore/management` 的整条依赖链。
- `SUBREL2`
  - 已使用 repo-local `go.work` 绑定本地 `Core/Proto` 完成模块联调。
  - `broker`、`exec`、`file`、`flow`、`topicbus`、`varstore`、`management` 的 `go test ./... -count=1 -p 1` 均通过。
- `SUBREL3`
  - 已推送分支 `fix/defaultset-deps-release`
  - 已推送 tags：
    - `broker/v0.1.1`
    - `exec/v0.1.2`
    - `file/v0.1.4`
    - `flow/v0.1.2`
    - `topicbus/v0.1.2`
    - `varstore/v0.1.4`
    - `management/v0.1.4`
- `DOC1`
  - 已补 repo-level archive：`docs/change/2026-03-25_defaultset-deps-release-chain.md`

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖 `SharedExecCapQueryBroker`、`capability/runtimedeps` 和各模块 `WithDeps` 发布要求。
- 架构合理性：通过
  - 通过发布 patch 版本链完成收口，不回退下游代码。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 本轮不涉及新的运行时路径。
- 可读性与一致性：通过
  - 仅调整版本链与归档，模块命名保持一致。
- 可扩展性与配置化：通过
  - 保持既有模块边界和 shared package 设计。
- 稳定性与安全：通过
  - 先做 repo-local 联调，再做下游 `GOWORK=off` 消费验证。
- 测试覆盖情况：通过
  - 上游模块联调通过，下游消费验证已在 `Server` 完成。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子 Agent。

### Stage 4 - Archive Record
- repo archive
  - `docs/change/2026-03-25_defaultset-deps-release-chain.md`
- workspace archive targets
  - `D:\project\MyFlowHub3\docs\change\2026-03-25_subproto-defaultset-deps-release-chain.md`
  - `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-25_subproto-defaultset-deps-release.md`
- lessons
  - 复用 `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`

#### Issue List
- none

阻塞：否
Stage 4 完成，等待 workflow 结束归档
