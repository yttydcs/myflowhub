# Plan - defaultset-deps-release

## Workflow Information
- Repo: `MyFlowHub-Server`
- Branch: `fix/defaultset-deps-release`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md:
  - 已读取 `D:\project\MyFlowHub3\guide.md`。
  - 当前工作区要求所有实现改动仅在 `D:\project\MyFlowHub3\worktrees\` 下执行。
- base/worktree confirmation:
  - 主控 workspace：`D:\project\MyFlowHub3`
  - Participating repos:
    - `MyFlowHub-Server` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release` -> `fix/defaultset-deps-release`
    - `MyFlowHub-SubProto` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release` -> `fix/defaultset-deps-release`
    - `MyFlowHub-Proto` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release` -> `fix/defaultset-deps-release`
  - `repo/` 主路径仅作控制面；本轮实现、测试、归档均在以上 worktree 中完成。

### Stage 1 - Requirements Analysis
#### Goal
- 正式修复 `MyFlowHub-Server` 单仓构建时 `modules/defaultset` 因 `WithDeps` / `runtimedeps` / `flow delete` 发布链未收口导致的编译错误。
- 让修复不依赖 workspace-local `go.work`，而是通过已发布的 semver 版本和下游依赖升级闭环。

#### Scope
- 必须:
  - 确认 `MyFlowHub-Proto`、`MyFlowHub-SubProto/broker` 与 `exec/file/flow/topicbus/varstore/management` 的当前源码 API / package 边界已稳定可发布。
  - 发布包含 `WithDeps` / `runtimedeps` / `flow delete` 所需能力的新 patch tag。
  - 升级 `MyFlowHub-Server` 对应依赖并验证 `GOWORK=off` 构建/测试通过。
  - 记录跨仓 release chain 的变更和验证结果。
- 可选:
  - 补充最小版本说明或 release 相关文档，便于后续下游同步。
- 不做:
  - 不回退 `Server` 到旧构造函数或旧 `cfg` 共享路径。
  - 不依赖临时 `replace` 或根 `go.work` 作为正式修复。
  - 不修改与本次 release chain 无关的业务语义、协议 schema 或默认装配策略。

#### Use Cases
- 在 `MyFlowHub-Server` 仓库单独执行 `go build ./...` 或 `GOWORK=off go test ./...` 时应正常解析到发布版 `SubProto` 依赖链。
- 未来 CI 或其他下游仓库升级 `Server` 后，不再因为本地 worktree / `go.work` 掩盖发布链缺口而失败。

#### Functional Requirements
- `MyFlowHub-Server/modules/defaultset` 当前依赖的以下已发布符号 / package 必须存在于发布版中：
  - `github.com/yttydcs/myflowhub-subproto/exec/capability`
  - `github.com/yttydcs/myflowhub-subproto/exec/runtimedeps`
  - `github.com/yttydcs/myflowhub-subproto/broker.SharedExecCapQueryBroker`
  - `github.com/yttydcs/myflowhub-proto/protocol/flow` 中 `ActionDelete` / `DeleteReq` / `DeleteResp` / `PermFlowDelete`
  - `exec/file/flow/topicbus/varstore/management` 的 `NewHandlerWithDeps`
- `MyFlowHub-Server/go.mod` 必须升级到包含以上 API 的 `proto` / `exec` / `file` / `flow` / `topicbus` / `varstore` / `management` 新版本。
- 修复完成后，`go list` 必须解析到新 semver，而不是本地 sibling worktree。
- 归档文档必须明确记录：
  - 触发症状
  - 根因是“上游 tag 未发布 / 下游版本未升级”
  - 正式修复路径

#### Non-functional Requirements
- 改动面最小，只处理 release chain 与必要的依赖文件 / 归档文档。
- 保持模块边界：`SubProto` 负责发布 API，`Server` 只升级依赖并验证装配路径。
- 验证必须覆盖 `GOWORK=off` 或等价单仓解析路径，避免再次被 `go.work` 掩盖。

#### Inputs / Outputs
- Inputs:
  - `MyFlowHub-Proto` 当前 `main` 上已存在但未发布的 `flow delete` 协议符号
  - `MyFlowHub-SubProto` 当前 `main` 上已存在但未发布的 `broker` / `WithDeps` API 与 `exec` shared packages
  - `MyFlowHub-Server` 当前 `go.mod` 仍锁定旧版 `proto` / `exec` / `file` / `flow` / `topicbus` / `varstore` / `management`
  - lesson: `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`
- Outputs:
  - `MyFlowHub-Proto` 新 tag：`v0.1.3`
  - `MyFlowHub-SubProto` 新 tag：`broker v0.1.1`、`exec v0.1.2`、`file v0.1.4`、`flow v0.1.2`、`topicbus v0.1.2`、`varstore v0.1.4`、`management v0.1.4`
  - `MyFlowHub-Server` 升级后的 `go.mod` / `go.sum`
  - 对应 repo-level 与 workspace-level change archive

#### Edge Cases
- 若 `SubProto` 当前源码除 `WithDeps` 外还包含未准备对外发布的变更，需要确认 patch 发布是否安全。
- 若 `Server` 升级后仍暴露出更多未发布 API 依赖，必须继续补齐同一 release chain，而不是只修最先报错的两个符号。
- 若 `GOWORK=off` 下测试受外部既有问题阻塞，需明确区分“本次已修复的问题”和“仓内既有阻塞”。

#### Acceptance Criteria
- `MyFlowHub-Proto v0.1.3` 与 `MyFlowHub-SubProto/broker v0.1.1`、`exec v0.1.2`、`file v0.1.4`、`flow v0.1.2`、`topicbus v0.1.2`、`varstore v0.1.4`、`management v0.1.4` 已指向包含所需符号的提交。
- `MyFlowHub-Server` 升级依赖后，`GOWORK=off go build ./...` 至少通过；如测试存在既有阻塞，需明确记录。
- release / bump 过程与验证证据完成归档，可供后续下游复用。

#### Risks
- patch tag 可能同时带出其他未发布但兼容的代码变化，需要核对是否仍属于 patch 级别。
- 若只本地打 tag 不推送远端，则正式修复仍不成立。
- `Server` 升级新版本后，可能继续暴露其他模块版本未跟进的问题。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 采用“上游先发布、下游再升级”的正式 release chain 修复。
- `MyFlowHub-Proto` 负责补齐 `flow delete` 协议符号发布。
- `MyFlowHub-SubProto` 负责补齐 `broker` / `exec` shared packages 与 `exec/file/flow/topicbus/varstore/management` `WithDeps` 构造路径发布。
- `MyFlowHub-Server` 仅升级 `go.mod/go.sum` 到新 patch 版本，并在真实依赖解析模式下验证。

#### Alternatives Considered
- 方案 A: 在 `Server` 回退为旧 `NewHandler` / `NewHandlerWithConfig`
  - 放弃原因：会撤销显式 runtime deps 设计，和已归档的 pluggable-state-backend 路径冲突。
- 方案 B: 只在根 `go.work` 中加入缺失 modules
  - 放弃原因：只能本地联调，不能修复单仓构建、CI 和真实 semver 消费。
- 方案 C: 正式发布 `SubProto` patch 版本并升级 `Server`
  - 采用原因：符合 lesson 中的 guardrail，能修复真实下游解析路径。

#### Module Responsibilities
- `MyFlowHub-Proto`
  - 发布 `flow delete` 协议符号对应 patch 版本。
- `MyFlowHub-SubProto`
  - 核对 `broker` / `exec` shared packages 与 `exec/file/flow/topicbus/varstore/management` 当前源码是否已具备可发布 API。
  - 产出 patch tag 和 repo-level 归档。
- `MyFlowHub-Server`
  - 升级 `exec` / `file` / `flow` / `topicbus` / `varstore` / `management` 版本。
  - 验证 `modules/defaultset` 在真实 semver 解析下可编译可测试。
- workspace docs
  - 归档本次跨仓 release chain 结果，并复用已有 `cross-repo-semver-release` lesson。

#### Data / Call Flow
- `Server` `go.mod` -> 解析到发布版 `Proto v0.1.3` 与 `SubProto/broker/exec/file/flow/topicbus/varstore/management`
- `modules/defaultset` -> 导入已发布的 `exec/capability`、`exec/runtimedeps` 并调用各模块已发布的 `NewHandlerWithDeps`
- `go build` / `go test` -> 在 `GOWORK=off` 模式下验证真实依赖链

#### Interface Drafts
- 不新增业务接口。
- 发布目标包 / 接口：
  - `github.com/yttydcs/myflowhub-proto/protocol/flow` 的 `delete` 契约
  - `github.com/yttydcs/myflowhub-subproto/broker.SharedExecCapQueryBroker`
  - `github.com/yttydcs/myflowhub-subproto/exec/capability`
  - `github.com/yttydcs/myflowhub-subproto/exec/runtimedeps`
  - `exec/file/flow/topicbus/varstore/management` 的 `NewHandlerWithDeps`
- 预计版本：
  - `proto v0.1.3`
  - `broker v0.1.1`
  - `exec v0.1.2`
  - `file v0.1.4`
  - `flow v0.1.2`
  - `topicbus v0.1.2`
  - `varstore v0.1.4`
  - `management v0.1.4`

#### Error Handling and Safety
- 发布前先确认 tag 所指提交确实包含目标 API / package。
- 依赖升级后使用 `go list -m` 与 `GOWORK=off` 验证，避免误用本地 worktree。
- 若远端 push/tag 失败，视为 release chain 未完成，不可把 workflow 标为完成。

#### Performance and Testing Strategy
- 不涉及运行时性能变更。
- 验证重点:
  - `go list -m github.com/yttydcs/myflowhub-subproto/exec`
  - `go list -m github.com/yttydcs/myflowhub-subproto/file`
  - `go list -m github.com/yttydcs/myflowhub-subproto/flow`
  - `go list -m github.com/yttydcs/myflowhub-subproto/topicbus`
  - `go list -m github.com/yttydcs/myflowhub-subproto/varstore`
  - `go list -m github.com/yttydcs/myflowhub-subproto/management`
  - `GOWORK=off go build ./...`
  - `GOWORK=off go test ./... -count=1 -p 1`（若存在既有阻塞则记录）

#### Extensibility Design Points
- 本次归档应沉淀“发布顺序必须按依赖方向执行”的实例证据，供后续 `Proto/Core/SubProto/Server` 变更复用。
- 继续复用现有 lesson，而不是新建重复 lesson，除非本次暴露出新的结构性规则。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 让 `MyFlowHub-Server` 脱离 workspace-local `go.work` 也能通过默认装配构建。
- Current state:
  - `MyFlowHub-Server` 代码已引用 `exec/capability`、`exec/runtimedeps`，并在 `exec/file/flow/topicbus/varstore/management` 上使用 `NewHandlerWithDeps`。
  - `GOWORK=off` 验证已进一步暴露：`exec` 还依赖未发布的 `broker.SharedExecCapQueryBroker`，`flow` 还依赖未发布的 `Proto flow delete` 符号。
  - 根 `go.work` 只接入了 `exec/flow/topicbus/varstore`，未接入 `file` / `management`，因此当前主线默认构建与真实依赖解析都不成立。

#### Docs Governance Routing Decision
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `none`
- Related specs:
  - `none`
- Related lessons:
  - `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`
- Canonical destinations:
  - workflow control -> 当前 worktree 根 `plan.md`
  - completed result -> repo/workspace `docs/change`
  - reusable troubleshooting knowledge -> 复用现有 lesson，按需补充索引

#### Related Requirements / Specs / Lessons
- 已确认本次不改变长期业务需求或技术契约。
- 直接复用 `cross-repo-semver-release` 作为排查与收口规则。

#### Executable Task List
- [x] `PROTOREL1` 发布 `MyFlowHub-Proto v0.1.3`
- [x] `SUBREL1` 核对 `SubProto/broker` / `exec` shared packages 与 `exec/file/flow/topicbus/varstore/management` 发布边界并准备 patch release
- [x] `SUBREL2` 在 `MyFlowHub-SubProto` 完成 repo-local 联调验证
- [x] `SUBREL3` 在 `MyFlowHub-SubProto` 创建 tag / repo-level change archive
- [x] `SRVREL1` 升级 `MyFlowHub-Server` 依赖到新的整条模块版本链
- [x] `VAL1` 在真实依赖解析模式下完成构建与测试验证
- [x] `DOC1` 完成 repo/workspace change archive 与索引更新

#### Task Details
##### PROTOREL1 - Publish Proto Flow Delete Contract
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release\plan.md`
- Goal: 发布包含 `flow delete` 契约的 `MyFlowHub-Proto v0.1.3`。
- Files / Modules:
  - `protocol/flow/*`
  - proto repo docs/archive
- Write Set:
  - proto repo files and tags
- Acceptance:
  - `go list -m github.com/yttydcs/myflowhub-proto@v0.1.3` 可解析。
- Test Points:
  - proto repo tests / symbol audit
- Rollback:
  - 删除未推送本地 tag

##### SUBREL1 - Confirm Patch Release Surface
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release\plan.md`
- Goal: 确认 `broker` / `exec` shared packages 与 `exec/file/flow/topicbus/varstore/management` 的 `WithDeps` / package API 已稳定且适合 patch 发布。
- Files / Modules:
  - `broker`
  - `exec`
  - `file`
  - `flow`
  - `topicbus`
  - `varstore`
  - `management`
  - module `go.mod` / `go.sum`（如需）
- Write Set:
  - 上述 modules
  - repo-level docs / release metadata
- Acceptance:
  - 明确新 tag 的目标版本与提交边界。
- Test Points:
  - module-level symbol / package audit
- Rollback:
  - 放弃发布并回到 planning，重新评估是否必须回退 `Server`

##### SUBREL2 - Validate Release Chain In Repo-Local Workspace
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release\plan.md`
- Goal: 使用 repo-local `go.work` 先验证上游整条模块链可编译可测试。
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

##### SUBREL3 - Publish SubProto Patch Versions
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release\plan.md`
- Goal: 产出 `broker` / `exec` / `file` / `flow` / `topicbus` / `varstore` / `management` 新 patch 版本并补 archive。
- Files / Modules:
  - `docs/change/*`
  - `docs/change/README.md`
  - git tags / commit
- Write Set:
  - repo-level archive docs
  - git history metadata
- Acceptance:
  - 新 tag 可被 `go mod download` / `go list -m` 解析。
- Test Points:
  - `go list -m ...@<new-version>`
- Rollback:
  - 删除未推送的本地 tag，回退未提交文档

##### SRVREL1 - Bump Server Dependencies
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release\plan.md`
- Goal: 把 `MyFlowHub-Server` 的 `exec` / `file` / `flow` / `topicbus` / `varstore` / `management` 升级到新 patch 版本。
- Files / Modules:
  - `go.mod`
  - `go.sum`
  - repo-level docs/archive as needed
- Write Set:
  - dependency files
  - repo-level archive docs
- Acceptance:
  - `go.mod` 解析到新版本，`modules/defaultset` 在 `GOWORK=off` 下不再报缺包或未定义符号。
- Test Points:
  - `go list -m`
  - `GOWORK=off go build ./...`
- Rollback:
  - 将依赖回退到上一版本并回退对应归档

##### VAL1 - Real Dependency Validation
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release\plan.md`
- Goal: 在不依赖根 `go.work` 的条件下验证正式修复成立。
- Files / Modules:
  - none or temporary env only
- Write Set:
  - none
- Acceptance:
  - `GOWORK=off` 下构建通过；测试若有既有问题，明确切分。
- Test Points:
  - `GOWORK=off go build ./...`
  - `GOWORK=off go test ./... -count=1 -p 1`
- Rollback:
  - 停止发版 / 升级，回到 `SUBREL1` / `SRVREL1` 继续补齐版本链

##### DOC1 - Archive And Index Updates
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release\plan.md`
- Goal: 归档本次 release chain 修复，并校验 lessons/index 路由。
- Files / Modules:
  - `MyFlowHub-SubProto/docs/change/*`
  - `MyFlowHub-SubProto/docs/change/README.md`
  - `MyFlowHub-Server/docs/change/*`
  - `MyFlowHub-Server/docs/change/README.md`
  - `D:\project\MyFlowHub3\docs/change/*`
  - `D:\project\MyFlowHub3\docs/change/README.md`
  - `D:\project\MyFlowHub3\docs/lessons/*`（仅在需要时）
- Write Set:
  - archive docs and indexes only
- Acceptance:
  - 归档与索引可追踪，requirements/specs/lessons impact 已明确记录。
- Test Points:
  - manual link/index check
- Rollback:
  - 回退本次新增 archive/index 文件

#### Dependencies
  - `PROTOREL1` 是 `SUBREL3` 与 `SRVREL1` 的前置条件之一。
  - `SUBREL1` 是 `SUBREL2` 与 `SUBREL3` 的前置条件。
  - `SUBREL2` 完成后才能安全发版。
  - `SUBREL3` 与 `PROTOREL1` 都完成后，`SRVREL1` 才能升级到真实可解析版本。
- `VAL1` 依赖 `SUBREL3` 与 `SRVREL1`。
- `DOC1` 依赖最终验证结果。

#### Risks and Notes
- `MyFlowHub-Proto`、`MyFlowHub-SubProto` 的 release 分支与 tags，以及 `MyFlowHub-Server` 的修复分支已推送到 `origin`。
- 本轮已确认发布链缺口不止两个 module，必须整体补齐 `defaultset` 依赖链，不能半修。
- 本地 Go `go1.25.0` 下载工具链缓存一度损坏，已通过定点重拉恢复验证环境；该问题不属于仓库代码缺陷，但需要在归档中保留证据。

#### Parallelism Assessment
- Current decision: `不并行实现`
- Reason:
  - `SUBREL3` 先于 `SRVREL1`，存在明确依赖链。
  - 当前 write set 交叉且验证依赖上游 tag 生效，不适合拆分子 Agent。

### Stage 3.2 - Implementation Record
- `PROTOREL1`
  - 已在 `MyFlowHub-Proto` worktree 发布 `v0.1.3` 对应提交，补齐 `flow delete` 协议契约。
- `SUBREL1` / `SUBREL2` / `SUBREL3`
  - 已在 `MyFlowHub-SubProto` worktree 补齐 `broker -> exec -> file/flow/topicbus/varstore/management` 版本链。
  - repo-local 联调通过，并已创建本地 patch tags：
    - `broker/v0.1.1`
    - `exec/v0.1.2`
    - `file/v0.1.4`
    - `flow/v0.1.2`
    - `topicbus/v0.1.2`
    - `varstore/v0.1.4`
    - `management/v0.1.4`
- `SRVREL1`
  - `go.mod/go.sum` 已对齐到完整版本链。
  - 未修改 `modules/defaultset` 业务装配代码；正式修复通过 semver 发布收口完成。
- `VAL1`
  - 先修复本机损坏的 `go1.25.0` 下载工具链缓存，再执行真实依赖解析验证。
  - `GOWORK=off go build ./...` -> 通过
  - `GOWORK=off go test ./... -count=1 -p 1` -> 通过

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖 `defaultset` 依赖链缺口、版本升级、`GOWORK=off` 验证和归档要求。
- 架构合理性：通过
  - 采用上游发布链补齐，不回退 `WithDeps` / `runtimedeps` 设计。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 本轮仅调整版本与归档，不引入新的运行时路径。
- 可读性与一致性：通过
  - 仅变更依赖版本和文档，命名与现有 semver/归档规范保持一致。
- 可扩展性与配置化：通过
  - 不增加硬编码或新的耦合，保留既有模块边界。
- 稳定性与安全：通过
  - 使用 `GOWORK=off` 的真实依赖解析路径完成回归验证。
- 测试覆盖情况：通过
  - `go build ./...` 与 `go test ./... -count=1 -p 1` 全通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子 Agent。

### Stage 4 - Archive Record
- repo archives
  - `MyFlowHub-Proto/docs/change/2026-03-25_proto-flow-delete-release.md`
  - `MyFlowHub-SubProto/docs/change/2026-03-25_defaultset-deps-release-chain.md`
  - `MyFlowHub-Server/docs/change/2026-03-25_server-defaultset-deps-release-chain.md`
- workspace archives
  - `D:\project\MyFlowHub3\docs\change\2026-03-25_defaultset-deps-release-chain.md`
- lesson updates
  - `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`
  - `D:\project\MyFlowHub3\docs\lessons\README.md`
- pending convergence note
  - 当前正式修复已完成；等待用户确认是否结束 workflow，再进入 merge/worktree cleanup。

#### Issue List
- none

阻塞：否
Stage 4 完成，等待用户确认是否结束 workflow
