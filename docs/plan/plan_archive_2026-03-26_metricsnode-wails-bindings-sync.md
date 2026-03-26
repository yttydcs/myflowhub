# Plan - metricsnode-fix-wails-bindings-sync

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-MetricsNode`
- Branch: `fix/metricsnode-wails-bindings-sync`
- Base: `main` @ `e88998d`
- Worktree: `D:\project\MyFlowHub3\worktrees\metricsnode-fix-wails-bindings-sync`
- Current Stage: `4 - Change Archive (completed, awaiting workflow end confirmation)`

## Stage Records

### Initialization
- `guide.md`: read from control-plane repo `D:\project\MyFlowHub3\guide.md`
- base/worktree confirmation:
  - implementation work is isolated in the dedicated worktree above
  - control-plane repo path remains read-only except for worktree management and final convergence
  - participating module is `windows` in `MyFlowHub-MetricsNode`

### Stage 1 - Requirements Analysis
#### Goal
- Restore a reproducible Windows frontend build path for `MyFlowHub-MetricsNode` when `frontend/src/App.vue` depends on the current Wails `App` bindings.

#### Scope
- Must:
  - verify the clean worktree baseline for `windows/frontend/wailsjs`
  - identify whether the failure is caused by stale or cross-project generated bindings
  - apply the minimum safe remediation needed for this repo and validate the build chain
  - archive the diagnosis and recovery path
- Optional:
  - strengthen repo-local docs or scripts if they are missing a guardrail that would reasonably prevent recurrence
- Not in scope:
  - redesign the MetricsNode Windows UI
  - change runtime behavior unrelated to bindings generation
  - overwrite dirty generated files in the control-plane working copy without an explicit workflow convergence step

#### Use Cases
- A developer runs `npm run build` under `windows/frontend` and expects the generated `wailsjs` exports to match `windows/app.go`.
- A developer runs the documented Wails build flow and expects stale bindings to be replaced with the correct MetricsNode bindings.
- Future debugging should quickly distinguish “frontend code bug” from “wrong Wails generated artifacts”.

#### Functional Requirements
- `windows/frontend/wailsjs/go/main/App.*` must export the methods used by `windows/frontend/src/App.vue`.
- The active remediation path must produce bindings from the MetricsNode Windows backend, not from another project.
- Validation must confirm whether `npm run build` and any required Wails generation step succeed in the clean worktree.
- Workflow output must record the recovery steps and root-cause indicators.

#### Non-functional Requirements
- Keep the change surface minimal and isolated to the affected repo/worktree.
- Do not silently swallow tooling errors; capture exact failing steps if validation fails.
- Preserve existing module boundaries; do not replace generated bindings with hand-maintained shims unless generation is proven impossible.

#### Inputs / Outputs
- Inputs:
  - `windows/app.go`
  - `windows/main.go`
  - `windows/frontend/src/App.vue`
  - `windows/frontend/wailsjs/**`
  - `windows/README.md`
  - `scripts/build-windows.ps1`
- Outputs:
  - validated binding state and build result
  - code or docs updates only if needed
  - `docs/change/2026-03-26_metricsnode-wails-bindings-sync.md`
  - a reusable lesson if the issue proves non-obvious and likely to recur

#### Edge Cases
- control-plane repo contains dirty generated files that do not match the clean worktree baseline
- `node_modules` is absent in the fresh worktree
- `wails` CLI is unavailable or `GOWORK` interferes with generation
- `frontend/dist` preconditions affect `wails generate module`

#### Acceptance Criteria
- A clean worktree can demonstrate the correct `wailsjs` export set for the MetricsNode Windows app.
- The Windows frontend build path is validated in the active worktree, or any remaining blocker is explicit and tool-verified.
- Any repo changes are limited to the minimum files needed to keep the build path reproducible and understandable.
- The resulting docs clearly state whether requirements/specs changed and how to recognize this failure mode next time.

#### Risks
- The user’s control-plane working copy already contains uncommitted generated-file drift that this workflow must not overwrite silently.
- Wails-generated files may be tracked in git, so regeneration can create noisy diffs.
- Tooling drift may reveal an additional build-chain problem after bindings are corrected.

#### Issue List
- None.

### Stage 2 - Architecture Design
#### Overall Solution
- Use the clean worktree as the source of truth, verify the current tracked MetricsNode bindings, and validate the documented generation/build flow before deciding whether any code or doc changes are required.

#### Alternatives Considered
- Manually edit `windows/frontend/wailsjs/**`:
  - rejected because these files are generated and would drift again on the next Wails generation.
- Patch `frontend/src/App.vue` to avoid the missing exports:
  - rejected because the Go backend already exposes the required methods; the mismatch indicates binding drift, not a frontend contract change.
- Blindly overwrite the control-plane repo:
  - rejected during execution because `m-autoflow` requires worktree isolation and the control-plane copy is dirty.

#### Module Responsibilities
- `windows/app.go`:
  - authoritative Go binding surface for the MetricsNode Windows app
- `windows/main.go`:
  - Wails app bootstrap and `Bind` registration
- `windows/frontend/wailsjs/**`:
  - generated TypeScript and runtime bindings
- `windows/frontend/src/App.vue`:
  - consumer of the generated `main/App` exports
- `scripts/build-windows.ps1` and `windows/README.md`:
  - operational guardrails for deterministic regeneration

#### Data / Call Flow
- Go `App` methods are bound via `windows/main.go`
- Wails generates `windows/frontend/wailsjs/**` from the bound Go surface
- `frontend/src/App.vue` imports from `../wailsjs/go/main/App`
- `vue-tsc --noEmit` fails when generated exports do not match the frontend imports

#### Interface Drafts
- Expected frontend-visible methods include:
  - `BootstrapGet`
  - `BootstrapSet`
  - `ClearAuth`
  - `Connect`
  - `Disconnect`
  - `EnsureKeys`
  - `Login`
  - `MetricsSettingsGet`
  - `MetricsSettingsSet`
  - `Register`
  - `StartReporting`
  - `Status`
  - `StopReporting`

#### Error Handling and Safety
- Prefer documented regeneration over manual edits to generated bindings.
- Treat control-plane dirty files as a separate convergence problem unless the user explicitly authorizes overwrite during workflow end.
- If validation reveals a new blocker, stop and record it instead of masking it with partial fixes.

#### Performance and Testing Strategy
- Use targeted file comparisons and the existing build commands.
- Run the smallest set of commands that verifies:
  - correct binding exports
  - frontend TypeScript/build health
  - Wails generation path if the current artifacts require refresh

#### Extensibility Design Points
- Keep recovery logic in repo docs/scripts so fresh worktrees and future binding-surface changes follow the same path.
- Prefer operational guardrails over ad hoc one-time fixes.

#### Issue List
- None.

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal: close the MetricsNode Windows `TS2305` build failure with a worktree-isolated, auditable fix path.
- Current state:
  - clean worktree is on `fix/metricsnode-wails-bindings-sync`
  - the clean worktree currently contains a correct-looking `windows/frontend/wailsjs/go/main/App.d.ts`
  - the control-plane repo contains dirty generated files matching `MyFlowHub-Win`, which likely explains the user-observed failure
  - validation completed:
    - `windows/frontend`: `npm ci` and `npm run build` passed
    - repo root: `powershell -ExecutionPolicy Bypass -File scripts/build-windows.ps1` passed
    - minimum remediation chosen: add a binding-surface sanity check to the build script and document the symptom/recovery path

#### Docs Governance Routing Decision
- Using `$m-docs`:
  - Requirements impact: `none`
  - Specs impact: `none`
  - stable truth remains in existing code and prior change docs; this workflow does not change product behavior
  - workflow execution record lives in worktree-root `plan.md`
  - completed diagnosis/fix record goes to `docs/change`
  - reusable troubleshooting knowledge should go to `docs/lessons` if the issue remains non-obvious after validation
- Related requirements: `none`
- Related specs: `none`
- Related lessons: `none yet`

#### Related Requirements / Specs / Lessons
- Related change context:
  - `docs/change/2026-03-03_metricsnode-settings-ui.md`
- No dedicated `requirements` or `specs` docs currently exist for this capability in this repo.

#### Executable Task List
- [x] `MNWB-1` verify clean worktree binding surface and reproduce/validate the frontend build state
- [x] `MNWB-2` apply the minimum remediation only if validation reveals drift or a missing guardrail
- [x] `MNWB-3` run review and archive change/lesson outputs

#### Task Details
##### MNWB-1 - Verify Binding Baseline
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\metricsnode-fix-wails-bindings-sync`
- Plan Path: `D:\project\MyFlowHub3\worktrees\metricsnode-fix-wails-bindings-sync\plan.md`
- Goal: prove the current MetricsNode Windows binding surface and frontend build state inside the isolated worktree
- Files / Modules:
  - `windows/app.go`
  - `windows/main.go`
  - `windows/frontend/src/App.vue`
  - `windows/frontend/wailsjs/**`
- Write Set:
  - none expected
- Acceptance:
  - exported bindings are compared against frontend imports
  - build result is captured with exact command output summary
- Test Points:
  - `npm run build`
  - targeted binding diff / export inspection
- Rollback:
  - no-op

##### MNWB-2 - Apply Minimum Remediation
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\metricsnode-fix-wails-bindings-sync`
- Plan Path: `D:\project\MyFlowHub3\worktrees\metricsnode-fix-wails-bindings-sync\plan.md`
- Goal: correct any validated drift with the smallest safe change, preferring generation or operational guardrails over hand-editing generated files
- Files / Modules:
  - `windows/frontend/wailsjs/**`
  - `windows/README.md`
  - `scripts/build-windows.ps1`
- Write Set:
  - `windows/frontend/wailsjs/**` only if regeneration is required in the worktree
  - docs/script files only if current guidance is insufficient
- Acceptance:
  - remediation is reproducible from repo-local commands
  - no unrelated runtime behavior changes are introduced
- Test Points:
  - `wails generate module` or `scripts/build-windows.ps1` as needed
  - follow-up `npm run build`
- Rollback:
  - revert the regenerated/generated-file or docs/script changes from this task only

##### MNWB-3 - Review and Archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\metricsnode-fix-wails-bindings-sync`
- Plan Path: `D:\project\MyFlowHub3\worktrees\metricsnode-fix-wails-bindings-sync\plan.md`
- Goal: verify requirement coverage and archive a reusable recovery record
- Files / Modules:
  - `plan.md`
  - `docs/change/**`
  - `docs/lessons/**`
  - docs index files if new lesson/index paths are created
- Write Set:
  - archive and lesson docs only
- Acceptance:
  - stage `3.3` checklist is explicitly reviewed
  - change archive records requirements/specs/lessons impact
  - lesson/index updates are included if the issue is reusable
- Test Points:
  - review checklist completion
  - docs path and cross-link verification
- Rollback:
  - remove only the new archive/lesson/index files from this workflow

#### Dependencies
- `npm` and project frontend dependencies
- `wails` CLI availability if regeneration is needed
- existing tracked `windows/frontend/wailsjs/**` content in the clean worktree

#### Risks and Notes
- The clean worktree may already prove that no source-code fix is required; in that case the workflow outcome becomes validation + documentation, not an artificial patch.
- If convergence back to the control-plane repo requires overwriting dirty generated files, handle that only at workflow end with explicit user confirmation.

#### Parallelism Assessment
- No sub-agents planned.
- The task is narrow, single-repo, and the critical path depends on direct validation of one build chain.

#### Issue List
- None.

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖“识别并恢复正确 Wails bindings”“验证前端 build”“沉淀可复用排查路径”。
- 架构合理性：通过
  - 没有改业务接口，修复点落在构建脚本校验与文档护栏，符合最小安全改动。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅新增一次构建前的 `App.d.ts` 文本校验，开销可忽略。
- 可读性与一致性：通过
  - 脚本检查点与 README/lesson 用词一致，均围绕相同症状关键词。
- 可扩展性与配置化：通过
  - 绑定面检查集中在构建脚本，后续如导出面扩展，只需更新脚本中的必需导出集合。
- 稳定性与安全：通过
  - 对异常绑定面 now fail fast，避免把错误延迟到 `vue-tsc` 或更晚阶段。
- 测试覆盖情况：通过
  - 已执行 `npm ci`、`npm run build`、`scripts/build-windows.ps1`。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未使用子 Agent。

### Stage 4 - Change Archive
- `$m-docs` routing decision executed.
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `updated`
- Related requirements: `none`
- Related specs: `none`
- Related lessons:
  - `docs/lessons/wails-bindings-cross-project.md`
- Archive outputs:
  - `docs/change/2026-03-26_metricsnode-wails-bindings-sync.md`
  - `docs/lessons/wails-bindings-cross-project.md`
  - `docs/README.md`
  - `docs/change/README.md`
  - `docs/lessons/README.md`
- Searchable lesson cues recorded:
  - symptoms: `TS2305`, missing `BootstrapGet`, missing `MetricsSettingsGet`
  - trigger conditions: foreign `wailsjs` contamination, skipped clean generate
  - keywords: `AboutState`, `FlowProjectsState`, `SaveHomeState`, `App.d.ts`
  - quick checks: inspect `App.d.ts`; rerun `scripts/build-windows.ps1`
