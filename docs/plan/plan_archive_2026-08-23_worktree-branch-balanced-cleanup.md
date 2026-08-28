# Plan Archive - MyFlowHub 本地 worktree 与分支平衡清理

> Archived from `D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup\plan.md` on 2026-08-23.
> Archive status: `closeout complete`.
> Publication status: local-only; no stage, commit, merge, push, or remote publication.


## Workflow Information
- Repo: `D:\project\MyFlowHub3`（控制面仓库）
- Branch: `chore/worktree-branch-cleanup`
- Base: `master` @ `c238aa8aec2c97cd8623439e2ca923a0fc1b2a62`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\docs`
- Code Repos:
  - `D:\project\MyFlowHub3`（控制面）
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Android`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-ClipboardNode`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Core`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-EmbeddedSDK`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-MetricsNode`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Proto`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SDK`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Active Worktree: `D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup`
- Current Stage: `4 Archive - complete`
- Mutation ownership: 仅控制面 Git/worktree 管理；各产品仓是被管理的 Git 注册表，不产生业务代码、测试或文档提交，因此不为每个目标仓再创建清理分支/worktree，避免为清理而新增十余个 worktree。

## Stage Records

### Initialization
- `guide.md`: 已读取；所有 worktree 必须位于 `D:\project\MyFlowHub3\worktrees`。
- project/docs/code repo confirmation: 已确认一个控制仓、十个产品仓和项目级私有文档根。
- base/worktree confirmation: 已从控制仓 `master` 创建专用语义分支和 worktree。
- baseline caveats:
  - 控制仓主路径 `master` 已有大量用户未提交内容；除本计划明确列出的 canonical Docs Root 窄幅归档写集外，不参与写入、合并或重置。
  - 当前只读复核显示控制仓主路径共有 136 个未提交项，且 `docs/README.md`、`docs/plan/README.md`、`docs/change/README.md`、`docs/lessons/README.md` 均已修改；归档必须在当前内容上做窄幅追加并保持 local-only，不暂存、不提交、不重置这些用户改动。
  - `MyFlowHub-MetricsNode/main` 已有用户未提交内容并领先 upstream，不参与写入、合并或重置。
  - 其余产品仓主路径在讨论审计时为 clean；执行前仍须重新复核。

### Discuss - Discovery And Requirements Shaping
#### Goal
- 降低 MyFlowHub 本地 worktree 与历史本地分支噪音，同时不丢失未提交实现、独有文档或不可恢复提交。

#### Scope
- 删除 12 个已经确认可恢复、仅残留旧 `todo.md` / 冗余 `.gitignore` 的 MyFlowHub worktree。
- 删除上述 12 个 worktree 对应的本地分支。
- 删除 281 个已经合入各自主线、未绑定任何 worktree 的本地分支。
- 保留全部远程分支、tag、主工作区、外部项目 worktree，以及剩余 15 个需复核的 MyFlowHub worktree。

#### Assumptions
- 用户选择“平衡清理”，批准范围是 MyFlowHub 本地 worktree 与本地分支。
- 外部 `ai-workflow-orchestrator` / `monkeys` 仓库和物理残留目录不在本轮范围。
- 只有完全匹配本计划安全快照的脏文件允许被丢弃；状态漂移即阻塞。

#### Open Questions
- 无需求层阻塞问题；等待用户批准本计划的精确 Task IDs 后方可执行。

#### Options Considered
- 保守：只删 281 个已合并未挂载分支；风险最低但 worktree 噪音仍在。
- 平衡：保守方案加 12 个可恢复 worktree；用户已选定。
- 彻底：先救援剩余成果，再清理所有历史项；工作量与误删风险较高，延期。

#### Rejected Options
- 直接递归删除 `worktrees` 子目录：会绕过 Git worktree 注册表并可能破坏外部项目。
- 对所有分支批量使用 `git branch -D`：会绕过未合并保护。
- 运行不受控的全仓 `git worktree prune`：可能顺带回收未纳入审批的 stale metadata。
- 清理远程分支/tag：超出用户批准的本地范围。

#### Recommended Direction
- 采用带状态哈希门禁的两阶段清理：先精确移除 12 个 worktree，再删除对应分支和 281 个已合并未挂载分支，最后做跨仓只读复验。

#### Research Summary
- 未使用外部研究；依据当前本地 Git 状态、已有清理归档和 Git 自身可恢复性判断。

#### Worktree / Branch / Docs Root Status
- 讨论快照：339 个本地分支，其中 299 个非主线分支已合并、29 个未合并、11 个主线分支。
- 计划初始化后：新增本计划分支，当前本地分支总数为 340；281 个已合并未挂载候选数量未改变。
- 原 MyFlowHub 次级 worktree 27 个；本计划 worktree 不计入原审计集合。
- Docs root 已确认：`D:\project\MyFlowHub3\docs`。

#### Issue List
- 无。

### Plan - Requirements And Architecture
#### Discussion Summary
- 用户确认采用“平衡清理”：清理低风险本地噪音，保留存在独有成果或判断不充分的项目。

#### Accepted / Rejected Requirements
- Accepted: 仅本地、仅 MyFlowHub、可恢复性优先、状态漂移时停止、保留 15 个需复核 worktree。
- Rejected: 外部项目清理、远程引用删除、强制处理新增脏状态、业务代码改动。

#### Requirements Analysis
##### Goal
- 在不损失有价值成果的前提下，将 MyFlowHub 次级 worktree 从原 27 个降至 15 个，并删除 293 个确定无须继续保留的本地分支引用。

##### Scope
- Will execute after approval in Stage 4 `$m-archive`: `ARC-1`, `WT-1`, `BR-1`, `BR-2`, `VAL-1`, `DOC-1`。
- Will not execute now: `PRE-1`（已完成，不重复运行）、`REV-1`、`EXT-1`、`REMOTE-1`。

##### Use Cases
- 后续查看 `git worktree list` 时不再把已发布/已合并 workflow 误认为活跃任务。
- 后续创建语义分支时减少历史本地分支干扰。
- 需要回溯时可由主线、tag、远程分支或记录的 tip SHA 恢复。

##### Functional Requirements
- 删除前必须逐项验证 owner repo、绝对路径、分支、tip SHA、恢复引用和唯一允许残留文件哈希。
- 12 个 worktree 必须通过 owning repo 的 `git worktree remove` 移除，不使用通用目录递归删除。
- 281 个历史分支只允许通过 `git branch -d` 删除；任何未合并错误必须停止，不升级为 `-D`。
- 12 个关联分支中，只有已经由 tag/远程引用保护的未合并分支允许在复验后使用 `-D`。
- 不删除或修改远程分支、tag、主工作区与保留集合。

##### Non-functional Requirements
- 安全：精确路径、精确 owner、哈希门禁、状态漂移 fail closed。
- 可审计：记录删除前后数量、分支 tip、恢复引用、失败项和保留项。
- 最小化：不修改业务文件，不运行格式化，不触碰外部项目。
- 可恢复：所有删除分支的 commit 必须仍由主线、tag 或远程 ref 可达。

##### Inputs / Outputs
- Inputs: 11 个 MyFlowHub Git 注册表、12 个目标 worktree、安全快照、281 个谓词候选。
- Outputs: 12 个目标路径和分支消失；281 个已合并未挂载分支消失；保留集合不变；最终审计结果。

##### Edge Cases
- 目标 worktree 新增/修改文件、HEAD 改变、分支切换：停止整个破坏性阶段。
- tag/远程引用不再包含未合并 tip：保留该 worktree/分支并报告阻塞。
- 文件被进程占用导致 `git worktree remove` 失败：停止该项，不使用 `Remove-Item` 绕过。
- `git branch -d` 拒绝删除：保留分支，不改用 `-D`。
- `git worktree prune --dry-run` 出现计划外条目：不执行实际 prune。
- 用户在执行中修改主工作区或保留 worktree：不清理该状态，只验证本轮没有覆盖它。

##### Acceptance Criteria
- 精确 12 个目标 worktree 均不再注册且路径不存在。
- 精确 12 个关联本地分支均删除，tip 仍由主线、tag 或远程 ref 可达。
- 281 个讨论时已确认的合并未挂载分支通过安全删除；无 `-D`。
- 原 15 个保留 MyFlowHub worktree、本计划 worktree、11 个主工作区和 20 个外部注册 worktree 均未被删除。
- 跨 11 仓本地分支总数预计从计划初始化后的 340 降到 47；若执行前合法状态变化导致数量偏移，重新生成计划而非套用旧数字。
- 不产生业务文件 diff；主路径既有状态保持不变。

##### Risks
- `--force` 会丢弃目标 worktree 的本地残留，因此只允许在状态路径和 SHA-256 同时匹配时使用。
- Windows 文件占用可能使目录移除失败。
- 远程跟踪引用是本地快照；未执行 fetch，执行只依赖当前 refs 和本地 tag，不声明远端实时状态。

#### Architecture Design
##### Overall Solution
1. Stage 3.2 execution record：保留已经通过的 `PRE-1` 只读预检结果，不重复审计、不执行清理。
2. Archive readiness：由 `$m-archive` 显式调用 `$m-docs`，在 canonical Docs Root 当前脏状态上窄幅完成计划/变更归档、stable-doc impact、索引及无关脏状态保护检查；归档保持 local-only，不暂存、不提交。
3. Worktree cleanup：逐 owner repo 精确移除 12 个目标 worktree。
4. Branch cleanup A：删除 12 个已解绑的关联本地分支。
5. Branch cleanup B：按每仓预先核对的集合用 `git branch -d` 删除 281 个已合并未挂载分支。
6. Validation and closeout：复核路径、worktree registry、branch refs、保留集合和主路径状态，直接向 canonical Docs Root 写回实际结果；确认索引窄幅追加未覆盖既有内容后，移除本计划 worktree/分支。若任一门禁失败，保留恢复现场并停止。

##### Alternatives Considered
- 单纯 `git worktree prune`：不能移除仍有效注册的 worktree，且范围不够精确。
- 先删分支再删 worktree：Git 会拒绝已 checkout 分支，顺序错误。
- 先打临时 backup tag：会把本地分支噪音转成 tag 噪音；现有主线/tag/远程引用已满足恢复要求。

##### Module Responsibilities
- 控制面 plan worktree：保存计划、执行清单和最终审计证据。
- 各 owning repo 主路径：仅作为 `git -C` 的注册表入口，不修改工作树文件。
- Git refs：提供主线/tag/远程引用可恢复性。
- Filesystem：仅由 `git worktree remove` 删除精确目标目录。

##### Data / Call Flow
- `candidate snapshot -> safety predicates -> worktree remove -> branch delete -> registry/ref/status validation`

##### Interface Drafts
- Read-only: `git worktree list --porcelain`, `git status --porcelain`, `git merge-base --is-ancestor`, `git branch --contains`, `git tag --contains`, `Get-FileHash`。
- Mutating: `git worktree remove --force <exact-path>`, `git branch -d <merged-branch>`, and guarded `git branch -D <protected-unmerged-branch>`。
- Validation: `git worktree prune --dry-run --verbose`; actual prune only when output contains no plan-external entry and is necessary。

##### Error Handling and Safety
- 每个破坏性命令前重新验证解析后的绝对路径位于 `D:\project\MyFlowHub3\worktrees`，且不等于 project root、任何 `repo/*` 主路径或本计划 worktree。
- 不使用 glob、未解析环境变量或字符串拼接目录作为删除目标。
- 任一 guard 失败即停止剩余破坏性操作并报告已完成/未完成清单。
- 不吞掉 Git 非零退出码，不使用静默 fallback。

##### Performance and Testing Strategy
- 无业务测试；验证重点是 Git 元数据与文件系统状态。
- 跨仓只读检查可批量运行；删除阶段串行，便于在首个异常处停止。
- 使用前后快照比较，确保主路径与保留 worktree 状态未被覆盖。

##### Extensibility Design Points
- 后续可复用同一 predicate：`not checked out + merged into base + local only` 清理历史分支。
- 对含独有成果的 worktree 另走救援/归档计划，不把判断逻辑塞入本轮。

#### Issue List
- 无。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标：执行一次可恢复、可审计、严格本地化的平衡清理。
- 当前：`PRE-1` 只读预检已通过，尚未执行任何目标删除；阶段映射已修正，剩余清理与归档统一由拥有 worktree cleanup 权限的 `$m-archive` 承接，等待用户批准。

#### Docs Governance Routing Decision
- 分类：`plan`。
- 当前根级 `plan.md` 是活动控制面例外；批准后由 `$m-archive` 归档为 `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-08-23_worktree-branch-balanced-cleanup.md` 并窄幅更新 canonical `docs/plan/README.md`。
- 完成结果归入 `D:\project\MyFlowHub3\docs\change\2026-08-23_worktree-branch-balanced-cleanup.md` 并窄幅更新 canonical `docs/change/README.md`。
- `PRE-1` 暴露了可复用的 PowerShell 嵌套数组自动展开陷阱；新增 `D:\project\MyFlowHub3\docs\lessons\powershell-nested-array-flattening.md` 并窄幅更新 canonical `docs/lessons/README.md`。
- Docs Root 位于当前控制仓主路径且已有用户未提交修改；本 workflow 的归档文件和索引追加保持 local-only，不执行 stage/commit/push，不把用户脏状态带入本计划分支。
- 本次不改变用户可见行为、长期需求、技术契约或架构决策；`docs/README.md` 拓扑不变，无需更新。
- `lessons` 路由已确认：记录 PowerShell 嵌套数组导致跨仓预检误报的症状、触发条件、关键词、快速检查与对象/映射修复方式；不把一次性清理结果重复写入 lesson。

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons
- Related intake: none
- Related features: none
- Related requirements: none
- Related specs: none
- Related decisions: none
- Related lessons: none existing; planned `docs/lessons/powershell-nested-array-flattening.md`
- Related plan: `docs/plan/plan_archive_2026-03-14_cleanup-residual-worktrees.md`
- Related changes:
  - `docs/change/2026-03-14_cleanup-residual-worktrees.md`
  - `docs/change/2026-03-21_repo-plan-worktree-cleanup.md`

#### Stable Docs Impact
- Intake impact: none
- Feature impact: none
- Requirements impact: none
- Specs impact: none
- Decision impact: none
- Lessons impact: add `docs/lessons/powershell-nested-array-flattening.md`；原因是首次 `PRE-1` 误报来自非显然且可能复发的 PowerShell 集合语义。

#### Safety Snapshot - Approved Worktrees

| Owner | Branch | Tip | Recovery | Allowed residue |
| --- | --- | --- | --- | --- |
| Android | `chore/release-chain-20260412-android` | `2f0f11b5707190821893bed59489a90c0f6537c1` | `origin/chore/release-chain-20260412-android`, `v0.1.31` | `?? todo.md` SHA-256 `449f8da8e75b11ee94bc1fcd5286f1a864efa42cad39d136e9adb6671e7cdffb` |
| Core | `chore/release-chain-20260412-core` | `536b3d56e83984bd9fa64e8964c6b7937be69222` | merged into `master` | `?? todo.md` SHA-256 `4989d1a98b8177aef0cc0d082151b9bd2961a33bbfb0a296e8f19033bd863e92` |
| MetricsNode | `chore/release-chain-20260412-metrics` | `a5e14fbe4e4fa7573f586a5277293ede24406766` | `origin/chore/release-chain-20260412-metrics`, `v0.1.3` | `?? todo.md` SHA-256 `85a1aeeb181d85e1c46de1b36de0c66984c6a97524af234a39b708945800854d` |
| Proto | `chore/server-release-align` | `fe2f79004e10dc845b0dbea84667766fefe12c06` | `v0.1.6` | `?? todo.md` SHA-256 `57d1d184fa21d11c4ce63b6781ae627c5821ee840d91b635e58d6207a34365db` |
| Proto | `feat/proto-stream-subproto` | `73ec78c0e09b54602aeb1cc3c8073505202cd035` | merged into `main` | `?? .gitignore` SHA-256 `6d82433e455297165d3f05dea434905bd5e8e90772cbef80062dedd980c04a64` |
| Proto | `chore/release-chain-20260412-proto` | `c8f0352e8a512612caf827fe7f46324eef52f985` | merged into `main` | `?? todo.md` SHA-256 `7869ff86d9d76fc77bcff264c2b0e3fdceefc7695c501cbb74ea904dd45ed025` |
| SDK | `chore/release-chain-20260412-sdk` | `27e7476d5ab36517f40cb1c1aad535eb52181533` | `origin/chore/release-chain-20260412-sdk`, `v0.1.14` | `?? todo.md` SHA-256 `099d5b28ed651431731e4c3b52ad5556919fe8f2f48df96340229046709cea00` |
| Server | `chore/release-chain-20260412-server` | `2b0d0a311a66d107dad424c67dfdb78e1f3985d4` | merged into `main` | `?? todo.md` SHA-256 `6560b5d85660dfd8dacb4077956f7202d831b4cd66bef4604058540e36f9ea43` |
| SubProto | `chore/release-chain-20260412-subproto` | `1917fdd10a542744c48ac2815620ab86eb4ca1cb` | `origin/chore/release-chain-20260412-subproto`, `auth/v0.1.6` | `?? todo.md` SHA-256 `fecbf920e0f07518b6ecd0d092f50391309fe4e0df482742eaa8f0a30928db3d` |
| SubProto | `chore/server-release-align` | `3f61a6e146a31daf78cf516d5603abf5fdeb493f` | `flow/v0.1.4` | `?? todo.md` SHA-256 `6dd082506f438b4c76f182b1200ed0cdad4927825d61d06af2cf91d477a59811` |
| SubProto | `feat/subproto-stream-subproto` | `77ff6c8434120b7d471d122696dcca63fe056b88` | `origin/feat/subproto-stream-subproto`, `stream/v0.1.0` | `?? .gitignore`, `?? stream/.gitignore`; both SHA-256 `6d82433e455297165d3f05dea434905bd5e8e90772cbef80062dedd980c04a64` |
| Win | `chore/release-chain-20260412-win` | `77a21fd75dcd919c7d3a20da686bcf8930cc0a5e` | `origin/chore/release-chain-20260412-win`, `v0.0.16` | ` M todo.md` SHA-256 `b728ba217224a9fe1a6bdcf0295c05bff176c767ecd242e769dfd96b13e7e7d4` |

对应绝对路径固定为：
- `D:\project\MyFlowHub3\worktrees\release-all-20260412-android`
- `D:\project\MyFlowHub3\worktrees\release-all-20260412-core`
- `D:\project\MyFlowHub3\worktrees\release-all-20260412-metrics`
- `D:\project\MyFlowHub3\worktrees\proto-server-release-align`
- `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
- `D:\project\MyFlowHub3\worktrees\release-all-20260412-proto`
- `D:\project\MyFlowHub3\worktrees\release-all-20260412-sdk`
- `D:\project\MyFlowHub3\worktrees\release-all-20260412-server`
- `D:\project\MyFlowHub3\worktrees\release-all-20260412-subproto`
- `D:\project\MyFlowHub3\worktrees\subproto-server-release-align`
- `D:\project\MyFlowHub3\worktrees\subproto-stream-subproto`
- `D:\project\MyFlowHub3\worktrees\release-all-20260412-win`

#### Safety Snapshot - Merged Unattached Branches

| Repo | Base | Approved count |
| --- | --- | ---: |
| Control | `master` | 5 |
| Android | `main` | 25 |
| ClipboardNode | `master` | 2 |
| Core | `master` | 14 |
| EmbeddedSDK | `main` | 5 |
| MetricsNode | `main` | 25 |
| Proto | `main` | 12 |
| SDK | `main` | 13 |
| Server | `main` | 55 |
| SubProto | `main` | 31 |
| Win | `main` | 94 |
| Total | — | 281 |

#### Executable Task List

##### Will Execute
- `ARC-1` - 在任何删除前完成计划/变更归档骨架、索引、影响记录与无关脏状态保护检查。
- `WT-1` - 精确移除 12 个已批准 worktree。
- `BR-1` - 删除 12 个已解绑且可恢复的关联本地分支。
- `BR-2` - 删除 281 个已合并、未挂载的本地分支。
- `VAL-1` - 跨仓回归验证、数量收敛与保留项核对。
- `DOC-1` - 写回实际结果、完成归档、按安全门禁合并归档提交并收尾本计划 worktree/分支。

##### Will Not Execute Now
- `PRE-1` - 重新复核安全快照与保留边界；原因：已在 Stage 3.2 完成且结果为 Passed，本阶段不重复运行。
- `REV-1` - 救援/复核剩余 15 个 MyFlowHub worktree；原因：含独有实现、文档或未决计划，需要独立审批。
- `EXT-1` - 清理 20 个外部注册 worktree 和 20 个物理残留目录；原因：属于 AIWO/Monkeys 或非 Git 证据，超出 MyFlowHub 范围。
- `REMOTE-1` - 删除远程分支/tag；原因：用户只批准本地清理。

#### Task Details

##### PRE-1 - 重新复核安全快照与保留边界
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup`
- Plan Path: `D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup\plan.md`
- Goal: 在任何删除前证明当前状态仍与批准快照完全一致。
- Files / Modules: 11 个 Git repo 注册表、12 个目标 worktree、15 个保留 worktree、20 个外部注册 worktree。
- Write Set: none。
- Acceptance:
  - 12 个目标的 owner/path/branch/tip/status/hash/recovery 全匹配。
  - 已合并未挂载候选仍按每仓计数匹配 281；否则停止并回到规划。
  - 保存主路径与保留 worktree 的状态签名供 `VAL-1` 比较。
- Test Points: `git worktree list --porcelain`, `git status --porcelain`, `git merge-base --is-ancestor`, `git branch/tag --contains`, `Get-FileHash`。
- Rollback: none；只读任务。
- Result:
  - Passed。
  - 12 个目标 branch/tip/recovery 全匹配。
  - 13 个允许残留文件的路径、状态和 SHA-256 全匹配。
  - 已合并未挂载分支为 281，保留 MyFlowHub worktree 为 15，主工作区为 11，外部注册 worktree 为 20。
  - 首次预检脚本因 PowerShell 嵌套数组被展开而产生误报；已改为对象/映射结构重跑，未发生任何删除或写入目标仓库。

##### ARC-1 - 归档就绪与删除前保护
- Owner: main agent via `$m-archive`
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup`
- Plan Path: `D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup\plan.md`
- Goal: 在任何目标删除前，完成受治理的计划/变更记录和控制面保护门禁，使清理范围、恢复依据与已通过的预检结果可独立审计。
- Files / Modules:
  - `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-08-23_worktree-branch-balanced-cleanup.md`
  - `D:\project\MyFlowHub3\docs\change\2026-08-23_worktree-branch-balanced-cleanup.md`
  - `D:\project\MyFlowHub3\docs\plan\README.md`
  - `D:\project\MyFlowHub3\docs\change\README.md`
  - `D:\project\MyFlowHub3\docs\lessons\powershell-nested-array-flattening.md`
  - `D:\project\MyFlowHub3\docs\lessons\README.md`
- Write Set: canonical Docs Root 中上述六个归档/索引文件；不写活动分支的 `docs/` 副本，不暂存或提交控制仓主路径的既有脏状态，不修改产品仓业务文件。
- Acceptance:
  - `$m-docs` impact 检查完整记录 intake/features/requirements/specs/decisions/lessons 结论。
  - 计划归档包含精确目标、安全快照、`PRE-1` 结果、Task 映射、回滚路径和待执行状态；变更记录先标记清理结果待回填。
  - 计划、变更、lesson 及三个索引可互相导航；索引只增加本 workflow 条目，已有 136 项脏状态及既有索引修改保持原样。
  - Lesson 包含 symptoms、trigger conditions、keywords/error text、quick checks、root cause、resolution 与 prevention/guardrails。
  - 记录控制仓主路径与三个目标索引的编辑前状态；若精确 patch 上下文不匹配或出现并发漂移，停止且不执行任何删除。
  - 归档明确标记为 local-only；不 stage、commit、merge 或 push canonical Docs Root 的当前修改。
- Test Points: `git status --short`, `git diff --check`, 索引链接/文件存在性检查、归档条目唯一性检查、编辑前后目标索引窄幅 diff 复核。
- Rollback: 仅删除本 workflow 三个新归档/lesson 文件及对应的三个索引条目；不重置或改写用户既有脏状态。

##### WT-1 - 精确移除 12 个已批准 worktree
- Owner: main agent via `$m-archive`
- Worktree: 同上
- Plan Path: 同上
- Goal: 通过 owning repo 的 Git worktree API 移除批准对象及唯一允许残留。
- Files / Modules: Safety Snapshot 中 12 个绝对路径。
- Write Set: 各 owning repo `.git/worktrees/*` 元数据和 12 个目标目录。
- Acceptance:
  - 每个路径在命令前重新解析并通过范围检查。
  - 每个目标 `git worktree remove --force` 成功后立即确认不再注册且路径不存在。
  - 不使用 `Remove-Item`、glob 或跨 shell 删除。
- Test Points: 每项前后 `git worktree list --porcelain`, `Test-Path`；最后 `git worktree prune --dry-run --verbose`。
- Rollback: 在 `BR-1` 前可用原分支重新 `git worktree add <path> <branch>`；残留控制文档不恢复。

##### BR-1 - 删除 12 个关联本地分支
- Owner: main agent via `$m-archive`
- Worktree: 同上
- Plan Path: 同上
- Goal: 在 worktree 解绑后删除对应本地引用，同时保证 tip 仍可恢复。
- Files / Modules: Safety Snapshot 中 12 个 branch refs。
- Write Set: 各 owning repo `refs/heads/<branch>`。
- Acceptance:
  - 已合并分支使用 `git branch -d`。
  - 未合并分支仅在 tag/远程 ref 仍包含精确 tip 时使用 `git branch -D`。
  - 不删除任何 remote ref 或 tag。
- Test Points: 删除前后 `git show-ref --verify`, `git merge-base`, `git branch -r --contains`, `git tag --contains`。
- Rollback: `git branch <original-name> <recorded-tip-sha>`；所有 tip 仍由批准恢复引用可达。

##### BR-2 - 删除 281 个已合并未挂载本地分支
- Owner: main agent via `$m-archive`
- Worktree: 同上
- Plan Path: 同上
- Goal: 清理已完整进入主线且不再被 worktree 使用的历史本地分支。
- Files / Modules: Safety Snapshot 中 11 仓计数对应的精确谓词集合。
- Write Set: 281 个 `refs/heads/*`。
- Acceptance:
  - 候选必须同时满足：非 base、`worktreepath` 为空、`merge-base --is-ancestor branch base` 成功。
  - 只使用 `git branch -d`；失败项保留并报告，禁止升级为 `-D`。
  - 每仓成功数与批准计数一致，或明确列出失败项而不扩大范围。
- Test Points: 删除前生成全名清单与计数；删除后逐 ref 验证不存在，并复核 base branch 存在。
- Rollback: 依据删除日志中的 branch/tip 重建；所有 tip 都是 base 祖先，不会成为不可达对象。

##### VAL-1 - 跨仓回归验证与状态收敛
- Owner: main agent via `$m-archive`
- Worktree: 同上
- Plan Path: 同上
- Goal: 证明只改变了获批的本地 worktree/branch 元数据。
- Files / Modules: 11 仓 worktree registry、branch refs、目标路径、保留路径和主路径状态。
- Write Set: none。
- Acceptance:
  - 12 个目标 worktree/branch 均不存在。
  - 281 个批准分支均删除或有明确未删除失败记录；无强制扩大。
  - 原 15 个保留 MyFlowHub worktree、本计划 worktree、11 个主路径和 20 个外部注册 worktree 均仍存在。
  - 主路径与保留 worktree 的既有用户状态签名不发生本轮覆盖。
  - 无计划外 prunable Git metadata；本地总分支预期 47。
- Test Points: `git worktree list --porcelain`, `git for-each-ref`, `git status --porcelain`, `git worktree prune --dry-run --verbose`, `Test-Path`。
- Rollback: 若仅验证失败，不做额外修复；报告实际终态并按记录 SHA/refs 单独恢复。

##### DOC-1 - 结果回填、归档完成与控制面收尾
- Owner: main agent via `$m-archive`
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup`
- Plan Path: `D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup\plan.md`
- Goal: 将实际清理/验证结果写入受治理文档，并在不覆盖用户既有脏状态的前提下完成活动计划 worktree/branch 收尾。
- Files / Modules: canonical Docs Root 中 `ARC-1` 的六个归档/索引文件、控制仓 Git 控制面、本计划 worktree/branch。
- Write Set:
  - 更新 canonical Docs Root 的计划/变更归档，写入实际结果、失败项、验证与回滚信息。
  - 如后续结果补充了同一 PowerShell 集合陷阱的证据，只更新已创建的 lesson；不在未重新规划时扩大到其它 lesson 文件。
  - 不暂存、提交或合并当前控制仓主路径的归档/索引修改；活动计划 worktree 的根级 `plan.md` / `todo.md` 作为临时控制状态，在成功归档后随 worktree 收尾。
- Acceptance:
  - 归档文档满足 `$m-archive` 必需字段并准确映射 `ARC-1`, `WT-1`, `BR-1`, `BR-2`, `VAL-1`, `DOC-1`。
  - 实际数量、失败项、恢复 SHA/refs、稳定文档影响、lessons 判断和本地-only 状态均如实记录。
  - 归档文件和索引条目在 canonical Docs Root 本地存在且既有用户内容未被覆盖后，才移除本计划 worktree并删除本地计划分支；merge 状态如实记录为 not applicable。
  - 若最终归档或活动计划清理门禁失败，保留当前分支/worktree和恢复证据，停止并报告，不强制覆盖、不声称 closeout 完成。
- Test Points: `git diff --check`, archive/index link check, 索引条目唯一性检查, 控制仓 `git status --short`, `git worktree list --porcelain`, `git show-ref --verify`。
- Rollback: 收尾前保留当前分支/worktree；收尾后仅移除本 workflow 归档文件/索引条目。目标 worktree/branch 按各自任务记录的 tip SHA 和恢复引用重建。

##### REV-1 - 复核并救援剩余 MyFlowHub worktree
- Owner: deferred; requires a separate approved workflow
- Worktree: not created
- Plan Path: 当前计划仅记录保留边界；后续需新建独立计划。
- Goal: 对剩余 15 个 MyFlowHub worktree 的独有提交、未提交实现、文档与未决计划逐项判断价值，并先救援再决定是否清理。
- Files / Modules: 15 个明确保留的 MyFlowHub worktree 及其 owning repo refs。
- Write Set: none in this workflow。
- Acceptance: 后续计划为每项给出 keep/rescue/remove 结论、证据和独立审批，不沿用本轮低风险删除授权。
- Test Points: `git status --porcelain`, `git log`, `git diff`, commit reachability、文档/产物清单。
- Rollback: 当前无写入；后续救援计划必须在删除前建立可恢复提交或归档。

##### EXT-1 - 外部注册 worktree 与物理残留清理
- Owner: deferred; external project owners required
- Worktree: not created
- Plan Path: 后续按 AIWO/Monkeys owning repo 分别规划。
- Goal: 在不混入 MyFlowHub 控制面的前提下，核对并清理 20 个外部注册 worktree 与 20 个物理残留目录。
- Files / Modules: AIWO/Monkeys Git 注册表和非 Git 残留目录。
- Write Set: none in this workflow。
- Acceptance: 后续计划确认每个路径 owner、活动状态、未提交内容、恢复方式与删除授权。
- Test Points: owning repo `git worktree list --porcelain`, 路径存在性、进程/文件占用和状态检查。
- Rollback: 当前无写入；后续任务按 owning repo branch/tip 或备份记录恢复。

##### REMOTE-1 - 远程分支与 tag 治理
- Owner: deferred; explicit remote-mutation approval required
- Worktree: not created
- Plan Path: 后续独立远程治理计划。
- Goal: 仅在用户另行授权时评估远程分支/tag 的保留、归档与删除策略。
- Files / Modules: 参与仓库 remote refs、tags 与托管平台保护规则。
- Write Set: none in this workflow。
- Acceptance: 后续计划基于最新远端状态、保护规则、下游引用和恢复方案逐项审批；本轮不做 fetch/push/delete。
- Test Points: 授权后再执行 remote ref listing、保护规则检查、下游引用与 release/tag 依赖核对。
- Rollback: 当前无写入；远程删除回滚依赖原始 SHA 与远端权限，必须先记录恢复引用。

#### Dependencies
- 已完成前置：`PRE-1 (Passed)`。
- 批准后的串行链：`ARC-1 -> WT-1 -> BR-1 -> BR-2 -> VAL-1 -> DOC-1`。
- `BR-1` 必须晚于 `WT-1`，否则 Git 会拒绝删除 checkout 中的分支。
- `WT-1` 前必须完成 `ARC-1` 的归档与无关脏状态保护门禁；`DOC-1` 必须晚于 `VAL-1`。

#### Risks and Notes
- 本计划授权的 `--force` 仅限 Safety Snapshot 的 13 个残留文件及其精确 SHA-256；不授权丢弃任何新增状态。
- 281 个分支不含 12 个当前仍挂载的关联分支，两个集合不重叠。
- 控制仓没有 remote；本计划分支仅本地存在，不进行 push。
- Canonical Docs Root 当前存在用户未提交修改；归档只做窄幅 local-only 追加，不通过本计划分支合并，不声明这些用户修改已提交或已备份。
- 不执行业务单元测试，因为无业务文件改动；Git 元数据/路径验证是本任务的主验收手段。

#### Parallelism Assessment
- 不并行执行破坏性步骤。
- 原因：多个命令会修改共享 `worktrees` 目录和各仓 ref，串行可在首个状态漂移或文件占用异常处停止。
- `$m-archive` 禁止派发子 Agent；归档、Git 清理和验证均由主 Agent 串行负责。

#### Issue List
- 无技术或范围阻塞；阶段所有权已修正。
- 当前仅等待用户批准精确的 `$m-archive` closeout Task IDs；批准前禁止清理。

### Stage 3.2 - Execution Record
- Executed scope: `PRE-1` only；它是 `$m-execute` 可拥有的只读预检任务。
- Result: Passed；12 个目标、安全残留哈希、恢复引用、281 个分支候选与全部保留边界均匹配批准快照。
- Changed product files: none。
- Validation: Passed；纠正 PowerShell 数组展开造成的首次脚本误报后，以对象/映射结构重跑通过。
- Code review: Not applicable；没有业务代码或产品文件改动。
- Execution status: complete for the execution-owned scope；未执行任何 worktree/branch 删除。
- Remaining scope: `ARC-1`, `WT-1`, `BR-1`, `BR-2`, `VAL-1`, `DOC-1` 已迁移到 Stage 4 `$m-archive`。

## Execution Scope After Approval

### Will Execute
- `ARC-1`
- `WT-1`
- `BR-1`
- `BR-2`
- `VAL-1`
- `DOC-1`

### Will Not Execute Now
- `PRE-1`：Stage 3.2 已完成且 Passed，不重复运行。
- `REV-1`：剩余 15 个 MyFlowHub worktree 的成果救援/价值判断，独立后续计划。
- `EXT-1`：外部项目和物理残留清理，超出当前项目边界。
- `REMOTE-1`：远程分支/tag 删除，未获授权。

## Approval Gate
- Plan status: archive and closeout complete
- Blocked: no
- Approved closeout tasks: `ARC-1`, `WT-1`, `BR-1`, `BR-2`, `VAL-1`, `DOC-1`
- Active phase: complete
- Do not dispatch implementation sub-agents

## Stage 4 - Archive And Closeout
- Entry gate: Passed。
- Execution complete: `PRE-1` Passed；没有业务代码或产品文件变更。
- Approval: 用户显式调用 `$m-archive`，批准计划列出的六个 closeout Task。
- Destructive preflight: Passed。
  - 12 个目标 worktree 的 owner/path/branch/tip/status/hash 全匹配。
  - 18 个恢复引用仍包含对应 target tip。
  - 281 个已合并未挂载分支和每仓计数全匹配；本地分支总数 340。
  - 15 个保留 MyFlowHub worktree、11 个主工作区和 20 个外部注册 worktree 已冻结签名。
- `ARC-1`: Passed；canonical Docs Root 的计划、变更、lesson 与三个索引已完成，保持 local-only。
- `WT-1`: Passed；12/12 个批准 worktree 已移除，路径与注册记录均消失。
- `BR-1`: Passed；12/12 个关联分支已删除，其中 4 个使用 `-d`，8 个在恢复引用门禁通过后使用受控 `-D`。
- `BR-2`: Changed；冻结的 281 个候选中 278 个由 `git branch -d` 删除，3 个因 upstream 未包含 tip 而被 Git 拒绝并保留，未升级为 `-D`。
- Preserved BR-2 refs:
  - `Android/chore/android-deps-sync` @ `c9c55dfb9b66e38d48105e921bf1297a49d30d6a`
  - `MetricsNode/chore/metricsnode-deps-sync` @ `0f5857e1684a769f0d51f94eaf97d7131816d3f9`
  - `Win/chore/win-deps-sync` @ `b93c5cdbcafcd37ddb01e4a931b84053a829f787`
- `VAL-1`: Passed；清理后本地分支总数 50，15 个保留 MyFlowHub worktree、11 个主工作区、20 个外部注册 worktree、48 个非目标顶层目录均保持，11 仓 prune dry-run 无计划外条目。
- `DOC-1`: Passed；活动计划 worktree 已由 Git 移除，本地计划分支使用 `-d` 删除，tip `c238aa8aec2c97cd8623439e2ca923a0fc1b2a62` 与 `master` 一致。
- Final local branch count: 49；相对初始化时的 340 共减少 291 个本地分支（12 个关联分支、278 个合并候选、1 个活动计划分支）。
- Merge status: not applicable；canonical Docs Root 归档保持 local-only，没有 workflow commit 或 remote publication。
