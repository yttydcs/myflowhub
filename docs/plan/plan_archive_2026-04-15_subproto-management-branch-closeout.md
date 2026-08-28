# Plan - subproto-management-branch-closeout

## Workflow Information
- Repo: `MyFlowHub-SubProto`
- Branch: `chore/subproto-management-branch-closeout`
- Base: `origin/main`
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md: `D:\project\MyFlowHub3\guide.md` 已读取；当前工作区要求 worktree 位于 `D:\project\MyFlowHub3\worktrees\`，提交信息使用中文。
- base/worktree confirmation:
  - 主执行仓库：`D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
  - 活跃执行 worktree：`D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout`
  - 当前主路径存在未提交改动，因此本次 workflow 仅在独占 worktree 内准备控制文档与验证记录，在控制面主路径执行最终 merge / remote 修正。
  - 显式 base 使用 `origin/main`，不信任 `origin/HEAD`，因为当前远端默认头仍指向 `origin/refactor/subproto-management-module`。

### Stage 1 - Requirements Analysis
#### Goal
确认并收口 `MyFlowHub-SubProto` 中仍停留在 `origin/refactor/subproto-management-module` 的未合入提交，使 `main` 成为包含 `flow/v0.1.5` 的真实主线，同时消除远端默认分支仍指向旧 refactor 分支的协作风险。

#### Scope
- 必须：
  - 复核 `origin/main` 与 `origin/refactor/subproto-management-module` 的真实差异。
  - 将分支独有提交合入 `main`，前提是合并无冲突且关键验证通过。
  - 验证 `flow` module 在合并后 `GOWORK=off` 可通过定向测试。
  - 校正远端默认分支从旧 `refactor/subproto-management-module` 切回 `main`。
  - 归档本次 merge / remote fix 的过程、验证结果、风险与回滚方案。
- 可选：
  - 在合并完成且默认分支切换后，删除远端陈旧分支。
  - 若需要，补充一个 lessons 文档记录“默认分支仍指向旧 release/refactor 分支”的排查线索。
- 不做：
  - 不修改 `flow` 业务逻辑。
  - 不改动 `requirements` 或长期 `specs` 真相源。
  - 不处理 `MyFlowHub-Server` 同名历史分支之外的其它仓库分支清理。

#### Use Cases
- 新 clone / CI / 人工核查默认进入 `main`，而不是进入历史 refactor 分支。
- 下游仓库、release 或人工审计在读取 `myflowhub-subproto/flow` 主线时，能直接看到 `flow/v0.1.5` 所对应的 `proto v0.1.7` 依赖和去掉的本地 `replace`。
- 后续协作者不会因为 `origin/HEAD` 指向旧分支而误判未合并状态。

#### Functional Requirements
- 明确识别 branch-only commit、main-only commit 和 merge base。
- 在控制面完成 `origin/refactor/subproto-management-module -> main` 的合并。
- 记录并验证 merge 后 `flow/go.mod` 与 `flow/go.sum` 的期望状态。
- 修正远端默认分支指向。
- 更新对应归档索引。

#### Non-functional Requirements
- 使用最小安全改动完成分支收口。
- 不污染已有脏工作区。
- 所有结论必须有实际 git / go 验证证据。
- 不重写既有 tag。

#### Inputs / Outputs
- 输入：
  - 目标提交：`84642c20919de6ad71505f1e269a2cb4f06fb1fd`
  - 目标分支：`origin/refactor/subproto-management-module`
  - 基线分支：`origin/main`
- 输出：
  - `main` 包含目标提交的合并结果
  - 远端默认分支修正结果
  - 根级归档：`docs/change/YYYY-MM-DD_*.md` 与 `docs/plan/plan_archive_YYYY-MM-DD_*.md`

#### Edge Cases
- `origin/main` 在 workflow 过程中继续前进，需要重新 fetch 并基于最新主线合并。
- 远端默认分支需要 GitHub API/CLI 权限；若当前环境缺权限，则至少完成主线合并并明确留下待办。
- 分支删除若被保护规则阻挡，不阻塞主线合并和默认分支修正。

#### Acceptance Criteria
- `origin/main` 或本地 `main` 合并后包含 `84642c20919de6ad71505f1e269a2cb4f06fb1fd`。
- 合并模拟或真实 merge 无冲突。
- `flow` module 在合并态 `GOWORK=off go test ./... -count=1 -p 1` 通过。
- 远端默认分支不再是 `refactor/subproto-management-module`。
- 归档和索引更新完成。

#### Risks
- 本地 `repo/MyFlowHub-SubProto` 主路径存在脏改动，不能直接在该路径做实现性编辑。
- 若 GitHub 默认分支修改权限不足，需分离“已完成的 merge”和“待执行的 repo setting 变更”。
- 若远端分支保护要求 PR 而不允许直接 push，需改为推送收口分支后走 PR。

#### Issue List
- 当前无阻塞；远端默认分支修改权限待实操验证。

### Stage 2 - Architecture Design
#### Overall Solution
采用“worktree 内计划、merge、验证，再把结果推送到远端 `main` 并修正 remote 设置”的方案。理由：
- 本次目标本质上是控制面收口，而不是业务代码开发。
- 主路径已有脏改动，不能在其上安全 checkout / merge。
- 在独占 worktree 分支完成 merge 后，可用 `HEAD:main` 推送远端主线，避免干扰现有主路径基线。

#### Alternatives Considered
- 方案 A：直接在 `repo/MyFlowHub-SubProto` 主路径 merge。
  - 否决原因：主路径已有大量未提交改动，风险高，审计边界不清晰。
- 方案 B：只保留 read-only 结论，不执行 merge。
  - 否决原因：用户已明确要求继续推进，且当前问题具备可直接收口条件。
- 方案 C：新建工作分支 / worktree 做计划与预验证，再在控制面主路径执行 merge / push / remote fix。
  - 采用：最符合当前 workflow 规则与主路径脏基线约束。

#### Module Responsibilities
- worktree `plan.md`
  - 承载 Stage 1/2/3.1 记录和任务分解。
- workflow worktree
  - 执行 fetch、merge、验证，并以 `HEAD:main` 推送远端主线。
- GitHub 远端设置
  - 修正默认分支到 `main`。
- root `docs/change` / `docs/plan`
  - 记录本次变更和归档索引。

#### Data / Call Flow
1. worktree 中记录计划并再次确认 live refs。
2. 在 workflow worktree fetch 最新远端并校验 `HEAD` 与 `origin/main` 的关系。
3. 在 workflow 分支上 merge `origin/refactor/subproto-management-module`。
4. 在合并态对 `flow` module 执行 `GOWORK=off` 定向测试。
5. 以 `HEAD:main` push 远端主线。
6. 修改 GitHub repo 默认分支为 `main`。
7. 视结果删除陈旧远端分支或记录残留。
8. 归档到根 `docs/`。

#### Interface Drafts
- Git:
  - `git fetch --prune origin`
  - `git merge --no-ff origin/refactor/subproto-management-module`
  - `git push origin HEAD:main`
  - `git push origin --delete refactor/subproto-management-module`（可选）
- Validation:
  - `GOWORK=off go test ./... -count=1 -p 1` in `flow/`
- Remote settings:
  - 优先 `gh api repos/yttydcs/myflowhub-subproto -X PATCH -f default_branch=main`
  - 若 `gh` 不可用，再退回 GitHub REST API 或仅报告阻塞

#### Error Handling and Safety
- merge 前再次确认 workflow worktree 干净，且基于最新 `origin/main`。
- 若 merge 冲突，立即中止并回到 `3.1` 更新计划。
- 若 push 或默认分支设置失败，保留本地 merge 结果并明确记录未完成项，不静默吞错。
- 不在主路径新增临时文件。

#### Performance and Testing Strategy
- 本次仅做目标分支差异与 `flow` module 定向测试，不做全仓 build。
- 依赖 `GOWORK=off`，避免本地 workspace 掩盖 semver / replace 问题。

#### Extensibility Design Points
- 把“默认分支漂移”作为可检索 lesson 候选，方便后续其它 repo 复用排查。
- 归档中记录远端 HEAD、branch contains、merge base、测试命令，便于后续重复执行。

#### Issue List
- 远端默认分支修改是否具备权限，待进入 `3.2` 实操确认。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 live 状态：
  - 初始 `origin/main` -> `7429ffcb6fcc1054a4223c15289cdd04c35351d0`
  - 初始 `origin/refactor/subproto-management-module` -> `84642c20919de6ad71505f1e269a2cb4f06fb1fd`
  - 初始 `origin/HEAD` -> `origin/refactor/subproto-management-module`
  - 完成后远端 `main` -> `7fb13380d78ab6feec8cd697d2912faf78316198`
  - 完成后远端 `HEAD` -> `refs/heads/main`
- 已确认 branch-only 目标提交是 `84642c2 chore: 发布 flow v0.1.5`。
- 该提交变更集中在：
  - `flow/go.mod`
  - `flow/go.sum`
  - `docs/change/2026-04-05_flow-v0.1.5.md`
  - `docs/change/README.md`
  - 由于 `repo/MyFlowHub-SubProto` 主路径脏工作树未清理，本轮 live 执行改为在当前 clean worktree 分支完成 merge，再 push 到远端 `main`。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- 稳定真相仍以根级 `guide.md` / `repos.md` 和已有 lessons 为准。
- 本次新增内容的 canonical destination：
  - workflow 计划 -> 本 worktree `plan.md`，完成后归档到 root `docs/plan/`
  - workflow 结果 -> root `docs/change/`
  - 可复用默认分支漂移经验 -> 沿用既有 lesson `cross-repo-semver-release.md`

#### Related Requirements / Specs / Lessons
- Related requirements: `none`
- Related specs: `none`
- Related lessons:
  - `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`
- Related change / plan references:
  - `D:\project\MyFlowHub3\docs\change\2026-04-05_graph-contract-release-chain-publish.md`
  - `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-04-05_graph-contract-release-chain-publish.md`
  - `D:\project\MyFlowHub3\docs\change\2026-02-19_subproto-management-v0.1.0.md`

#### Executable Task List
- [x] `SUBCLOSE-1` 再次 fetch 并确认 `main` / topic / remote HEAD 的最新状态
- [x] `SUBCLOSE-2` 在 clean worktree 完成 merge，并处理 `docs/change/README.md` 索引冲突
- [x] `SUBCLOSE-3` 在合并态执行 `flow` module 的 `GOWORK=off` 定向验证并 push `main`
- [x] `SUBCLOSE-4` 修正 GitHub 默认分支到 `main`，并删除陈旧远端分支
- [x] `SUBCLOSE-5` 按 `$m-docs` 归档 `docs/change` / `docs/plan`，补索引并记录 lessons 影响

#### Task Details
##### SUBCLOSE-1 - Live refs confirmation
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout\plan.md`
- Goal: 确认 `origin/main`、topic 分支、remote HEAD 与 branch protection 前置事实
- Files / Modules:
  - git refs only
- Write Set:
  - none
- Acceptance:
  - 有最新 SHA、contains 关系和 remote HEAD 证据
- Test Points:
  - `git show -s`, `git branch -r --contains`, `git remote show origin`
- Rollback:
  - none

##### SUBCLOSE-2 - Merge branch into main
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout\plan.md`
- Goal: 在当前 clean worktree 分支合入 `origin/refactor/subproto-management-module`，并准备把 merge 结果推送到远端 `main`
- Files / Modules:
  - `flow/go.mod`
  - `flow/go.sum`
  - `docs/change/2026-04-05_flow-v0.1.5.md`
  - `docs/change/README.md`
- Write Set:
  - 当前 worktree git 历史
- Acceptance:
  - merge 成功且当前 `HEAD` 包含 `84642c2`
- Test Points:
  - `git merge --no-ff`
  - `git log --decorate --oneline --graph -n 10`
- Rollback:
  - merge 未 push 前 `git merge --abort` 或 `git reset --hard ORIG_HEAD`
  - 已 push 后用 revert merge commit，不重写历史

##### SUBCLOSE-3 - Validate merged flow module and push
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout\plan.md`
- Goal: 确保 `flow v0.1.5` 所需依赖状态已真实进入主线并可通过单仓验证
- Files / Modules:
  - `flow/**`
- Write Set:
  - none beyond merge result
- Acceptance:
  - `GOWORK=off go test ./... -count=1 -p 1` in `flow/` 通过
  - `git push origin HEAD:main` 成功
- Test Points:
  - `go test`
  - `git ls-remote origin refs/heads/main`
- Rollback:
  - 若测试失败，停止 push 并回退 merge

##### SUBCLOSE-4 - Fix remote default branch and stale branch
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout\plan.md`
- Goal: 把 GitHub 默认分支改回 `main`，并清理陈旧 refactor 分支
- Files / Modules:
  - GitHub repo settings
- Write Set:
  - remote metadata
- Acceptance:
  - `git remote show origin` 不再显示 `HEAD branch: refactor/subproto-management-module`
- Test Points:
  - `gh api repos/yttydcs/myflowhub-subproto`
  - `git remote show origin`
- Rollback:
  - 如误切，立即改回正确默认分支；分支删除失败则保留并在归档中记录

##### SUBCLOSE-5 - Archive workflow
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-management-branch-closeout\plan.md`
- Goal: 在 root `docs/` 留下可检索的 merge / default-branch 修正记录
- Files / Modules:
  - `D:\project\MyFlowHub3\docs\change\*.md`
  - `D:\project\MyFlowHub3\docs\plan\*.md`
  - `D:\project\MyFlowHub3\docs\change\README.md`
  - `D:\project\MyFlowHub3\docs\plan\README.md`
  - optional `D:\project\MyFlowHub3\docs\lessons\*.md`
- Write Set:
  - root docs archive only
- Acceptance:
  - archive 和索引可导航
- Test Points:
  - 索引链接与 file existence 复核
- Rollback:
  - 回退本次新增归档文件和索引变更

#### Dependencies
- 需要 git push 权限到 `yttydcs/myflowhub-subproto`
- 需要具备修改 GitHub repo 默认分支的权限
- 需要本机 `go` 可用并能执行 `GOWORK=off` 定向测试

#### Risks and Notes
- `repo/MyFlowHub-SubProto` 主路径有既有未提交改动，因此本次不能在主路径 checkout / merge，只能保持控制面只读并通过 clean worktree 推送远端主线。
- `origin/main` 在 workflow 执行中可能继续变化；所有 merge/push 前必须再 fetch。
- 远端默认分支修正若失败，不影响先把 `main` 补齐，但会保留协作风险。

#### Parallelism Assessment
- 本轮不派发子Agent。
- 原因：任务集中在同一仓库的 git refs、merge 和远端 metadata，且每一步强依赖前一步结果，不适合并行写操作。

#### Issue List
- 当前无阻塞。

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已完成 merge、定向测试、远端默认分支修正和陈旧分支删除。
- 架构合理性：通过
  - 未改业务逻辑，只做 git 历史和 remote metadata 收口。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅执行 git 与单模块定向测试，无新增运行时代码路径。
- 可读性与一致性：通过
  - merge commit 信息使用中文；索引冲突做了最小保留式合并。
- 可扩展性与配置化：通过
  - 默认分支回归 `main`，消除后续 CI / clone 的错误基线。
- 稳定性与安全：通过
  - 主路径脏基线未被覆盖；GitHub API 调用复用了现有 git credential helper，未输出凭据。
- 测试覆盖情况：通过
  - `flow` module 在合并态执行了 `GOWORK=off go test ./... -count=1 -p 1`。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子Agent。

阻塞：否
进入 4
