# Plan - defaultset-deps-release

## Workflow Information
- Repo: `MyFlowHub-Proto`
- Branch: `fix/defaultset-deps-release`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md:
  - 已读取 workspace `D:\project\MyFlowHub3\guide.md`。
- base/worktree confirmation:
  - 当前 repo-local 实现工作树：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release`
  - 对应上游 / 下游 worktree：
    - `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-fix-defaultset-deps-release`
    - `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-fix-defaultset-deps-release`

### Stage 1 - Requirements Analysis
#### Goal
- 发布包含 `flow delete` 契约的 `MyFlowHub-Proto` patch 版本，解除 `flow` module 的真实依赖解析缺口。

#### Scope
- 必须:
  - 确认 `protocol/flow/types.go` 中 `delete` 相关符号已在当前主线稳定存在。
  - 发布新 patch tag。
  - 创建 repo-level change archive 并维护索引。
- 可选:
  - 补充最小协议说明。
- 不做:
  - 不新增新的协议动作；仅发布已存在契约。

#### Use Cases
- `MyFlowHub-SubProto/flow` 在 `GOWORK=off` 下解析到新 proto 版本后可成功编译。

#### Functional Requirements
- 新版本必须包含：
  - `ActionDelete`
  - `ActionDeleteResp`
  - `PermFlowDelete`
  - `DeleteReq`
  - `DeleteResp`

#### Non-functional Requirements
- 只做最小必要 patch 发布，不扩散到无关协议面。

#### Inputs / Outputs
- Inputs:
  - 当前主线已存在的 `flow delete` 契约
  - 下游 `flow@v0.1.2` 在真实依赖解析下的未定义符号报错
- Outputs:
  - `myflowhub-proto v0.1.3`
  - repo-level archive

#### Edge Cases
- 若当前 HEAD 还有未准备发布的协议破坏性变更，需要停止并重新裁切提交。

#### Acceptance Criteria
- `go list -m github.com/yttydcs/myflowhub-proto@v0.1.3` 可解析。

#### Risks
- 若 tag 仅本地存在，下游依然无法拉取到新契约。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 不改协议设计，只发布当前主线已存在的 `flow delete` 契约。

#### Alternatives Considered
- 回退 `flow` module 到不依赖 `delete`：放弃。
- 正式发布新的 proto patch：采用。

#### Module Responsibilities
- `MyFlowHub-Proto`
  - 对外提供 `flow delete` 契约。
- repo docs
  - 记录发布背景、验证与回滚。

#### Data / Call Flow
- `proto tag` -> `SubProto/flow` 升级依赖 -> `Server` 升级整条版本链 -> `GOWORK=off` 验证

#### Interface Drafts
- `protocol/flow.ActionDelete`
- `protocol/flow.ActionDeleteResp`
- `protocol/flow.PermFlowDelete`
- `protocol/flow.DeleteReq`
- `protocol/flow.DeleteResp`

#### Error Handling and Safety
- 发布前确认 tag 所指提交包含目标符号。

#### Performance and Testing Strategy
- 以符号检查和 `go list -m` 为主。

#### Extensibility Design Points
- 本次仍遵守跨仓 semver 先 Proto 后下游的发布顺序。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 发布 `myflowhub-proto v0.1.3`，为 `flow` module 补齐真实依赖解析所需的 `delete` 契约。
- Current state:
  - 当前主线已包含 `flow delete` 符号。
  - 已发布 `v0.1.2` 仍缺失这些符号，导致下游 `flow` 在 `GOWORK=off` 下编译失败。

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
- 本仓不新增长期需求或技术契约，只做 patch release archive。

#### Executable Task List
- [x] `PROTOREL1` 校验 `flow delete` 契约发布边界
- [x] `PROTOREL2` 创建 `v0.1.3` tag 并补 repo-level archive

#### Task Details
##### PROTOREL1 - Confirm Flow Delete Release Surface
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release\plan.md`
- Goal: 确认当前 `flow delete` 契约适合 patch 发布。
- Files / Modules:
  - `protocol/flow/*`
- Write Set:
  - repo files and tags
- Acceptance:
  - 目标版本与符号范围明确。
- Test Points:
  - symbol audit
- Rollback:
  - 放弃当前发布并回到 planning

##### PROTOREL2 - Publish Proto Patch
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release`
- Plan Path: `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-fix-defaultset-deps-release\plan.md`
- Goal: 发布 `myflowhub-proto v0.1.3` 并补 archive。
- Files / Modules:
  - repo docs/change
  - git tags / commit
- Write Set:
  - repo archive docs and git metadata
- Acceptance:
  - `go list -m github.com/yttydcs/myflowhub-proto@v0.1.3` 可解析。
- Test Points:
  - `go list -m ...@v0.1.3`
- Rollback:
  - 删除未推送本地 tag

#### Dependencies
- `PROTOREL1` -> `PROTOREL2`

#### Risks and Notes
- `v0.1.3` 与 `fix/defaultset-deps-release` 已推送到 `origin`，下游可通过真实 semver 拉取。

#### Parallelism Assessment
- Current decision: `不并行实现`
- Reason:
  - 发布动作与验证强顺序依赖，且本轮主要阻塞在真实 release chain。

### Stage 3.2 - Implementation Record
- `PROTOREL1`
  - 已确认 `protocol/flow/types.go` 当前主线包含 `ActionDelete`、`ActionDeleteResp`、`PermFlowDelete`、`DeleteReq`、`DeleteResp`。
- `PROTOREL2`
  - 已创建 `v0.1.3` tag。
  - 已补齐 repo-level archive：`docs/change/2026-03-25_proto-flow-delete-release.md`
  - 已推送分支 `fix/defaultset-deps-release` 与 tag `v0.1.3` 到 `origin`。

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已发布 `flow delete` 契约并补档。
- 架构合理性：通过
  - 采用 patch release，不引入新协议设计。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 本轮无运行时实现变更。
- 可读性与一致性：通过
  - 仅补 release archive 和 tag，命名与现有版本线一致。
- 可扩展性与配置化：通过
  - 不新增耦合或硬编码。
- 稳定性与安全：通过
  - 仅发布已存在契约，降低变更面。
- 测试覆盖情况：通过
  - 通过符号审计和下游 `go list -m` / `GOWORK=off` 消费验证完成收口。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子 Agent。

### Stage 4 - Archive Record
- repo archive
  - `docs/change/2026-03-25_proto-flow-delete-release.md`
- workspace archive targets
  - `D:\project\MyFlowHub3\docs\change\2026-03-25_proto-flow-delete-release.md`
  - `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-25_proto-defaultset-deps-release.md`
- lessons
  - 复用 `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`

#### Issue List
- none

阻塞：否
Stage 4 完成，等待 workflow 结束归档
