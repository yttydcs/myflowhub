# Plan - Fix BSD reference image preview to use history zoom viewer

## Workflow Information
- Repo: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview`
- Branch: `fix/bsd-reference-image-preview`
- Base: `main`
- Worktree: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview`
- Current Stage: `4 - Change Archive`

## Stage Records

### Initialization
- guide.md: read from `D:/project/monkeys/guide.md`
- base/worktree confirmation:
  - control-plane repo remains `D:/project/monkeys/repo/monkey`
  - active implementation worktree is `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview`
  - `ui/.env` copied into the worktree
  - `node_modules` and `ui/node_modules` linked into the worktree
  - note: current worktree baseline already contains an unrelated `.gitignore` modification adding `.ace-tool/`; do not include it in this task

### Stage 1 - Requirements Analysis
#### Goal
- Make the BSD custom page upload card preview use the same zoomable viewer used by BSD history generation.
- Correct the previous miss: the live page path is `ReferenceImageCard`, not the generic `VinesUploader` preview path the previous change targeted.

#### Scope
- Must:
  - replace the current `ReferenceImageCard` image preview dialog titled `图片预览`
  - reuse the history zoom viewer already used by BSD history generation
  - preserve current upload, delete, edit, hover-preview, mask-open, and personal-asset selection behaviors
- Optional:
  - add a focused UI test if the local harness cost stays reasonable
- Not doing:
  - broad refactor of all uploader preview flows
  - changes to non-BSD generic uploaders beyond the already-existing code
  - changes to personal asset browser preview

#### Use Cases
- User clicks an existing image in BSD custom scenes such as local edit, free fission, runway, style fusion, smart outfit, light effect, line-to-garment, garment-on-model, atmosphere.
- User clicks the base slot when `onOpenMask` is provided; mask flow must still win over generic preview.
- User hovers over an existing image; the small hover preview should remain available.

#### Functional Requirements
- Existing image click in `ReferenceImageCard` should open the zoomable history-style viewer instead of the old scroll-only dialog.
- The viewer should support zoom in, zoom out, drag, and reset.
- The viewer should hide footer actions for this uploader-card reuse case, while keeping the history viewer's existing details-toggle behavior.
- Base-slot mask behavior must remain unchanged.
- Empty-card upload entry, local upload, personal asset selection, remove image, and edit image behaviors must remain unchanged.

#### Non-functional Requirements
- Keep the change surface minimal and localized.
- Avoid changing unrelated shared preview behavior.
- Do not add full-build validation such as `yarn build`.

#### Inputs / Outputs
- Input:
  - `ReferenceImageCard` receives `slot.previewUrl` / `slot.uploadedUrl`
  - optional `maskPreviewUrl`, `onOpenMask`, `onEdit`
- Output:
  - click existing image opens `HistoryImageDetailDialog` with the current image URL and slot label

#### Edge Cases
- No `previewUrl`: clicking should not open preview.
- `slot.id === 'base'` and `onOpenMask` exists: continue opening mask flow instead of history viewer.
- Remote URL may come from uploaded or externally selected assets; preview should work for both.
- Existing hover preview and overlay buttons must not accidentally trigger the wrong action.

#### Acceptance Criteria
- On BSD custom pages using `ReferenceImageCard`, clicking an existing image no longer opens a dialog titled `图片预览`.
- The opened preview supports zoom controls and drag behavior like BSD history generation.
- Base-slot mask click behavior still works.
- Hover preview, remove, edit, upload, and asset-picker flows still work.

#### Risks
- `ReferenceImageCard` is reused across multiple BSD custom scene panels, so any regression propagates widely.
- Importing the history viewer into the card must avoid breaking current scene dependencies.
- There is no existing direct component test for this card/viewer combination.

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- Patch `ReferenceImageCard` directly to replace its local `Dialog + 图片预览` implementation with `HistoryImageDetailDialog`.
- Reuse `HistoryImageDetailDialog` itself, but only hide footer actions and keep the viewer's existing details panel behavior.

#### Alternatives Considered
- Patch generic `VinesUploader` only:
  - rejected because the user-facing BSD custom-page upload card does not use that preview path for the click behavior being reported
- Extract a brand new shared zoom viewer:
  - rejected for now because `HistoryImageDetailDialog` already provides the needed interaction and a reuse mode exists
- Keep the previous uploader-specific lightweight compatibility layer:
  - rejected because the user explicitly required reverting commit `f02d81018...`

#### Module Responsibilities
- `ReferenceImageCard.tsx`
  - owns the click entry for existing image cards on BSD custom pages
  - should hold the open state and preview image payload for the history viewer
- `HistoryImageDetailDialog.tsx`
  - remains the shared zoomable viewer implementation
  - reused with hidden footer only

#### Data / Call Flow
- User clicks existing card image
- `ReferenceImageCard` checks mask precedence
- if not mask flow, build a minimal history-image payload from `previewUrl` and `slot.label`
- open `HistoryImageDetailDialog`
- close dialog resets preview state

#### Interface Drafts
- No public API changes planned for parent scene panels.
- `ReferenceImageCard` local state:
  - `detailOpen: boolean`
  - `detailImage: { id: string; url?: string; title?: string; status?: string } | null`
- Expected viewer props:
  - `open`
  - `image`
  - `hideFooterActions`
  - `onOpenChange`

#### Error Handling and Safety
- Guard against empty preview URLs before opening the dialog.
- Keep current mask branch as the first exit path.
- Do not touch upload or asset-selection logic.

#### Performance and Testing Strategy
- No new expensive processing; only swap the preview container.
- Validation plan:
  - inspect diff scope
  - run `git diff --check`
  - run a targeted test only if a low-cost component test can be added without large environment scaffolding
  - otherwise record the automated test gap and rely on targeted manual verification points

#### Extensibility Design Points
- Keep preview wiring local to `ReferenceImageCard` so future BSD custom-scene preview changes remain isolated.
- Reuse existing history-viewer props rather than introducing another parallel preview implementation.

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - fix the actual BSD custom-scene upload preview path so existing image click opens the history zoom viewer
  - revert the earlier mis-scoped change introduced by commit `f02d81018e135cb30cca939cb9f7bd9c5baffbe1`
- Current state:
  - previous merged change updated generic `ui/src/components/ui/vines-uploader/files.tsx`
  - the reported live behavior still comes from `ui/src/components/layout/workbench/view/customers/scenes/components/ReferenceImageCard.tsx`
  - that component currently opens `DialogTitle>图片预览</DialogTitle>` with a scroll-only `<img>`
  - user has requested that the previous generic-uploader change be rolled back as part of this workflow

#### Docs Governance Routing Decision
- Using `$docs-governor` to check plan/change routing and requirement/spec impact.
- Docs tree state:
  - no `docs/README.md`
  - no `docs/requirements`
  - no `docs/specs`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements: `none found`
- Related specs: `none found`
- Canonical destinations:
  - execution control doc: this worktree-root `plan.md`
  - completion archive: project-root `docs/change/`

#### Executable Task List
- [x] T1 Replace `ReferenceImageCard` local preview dialog with `HistoryImageDetailDialog`
- [x] T2 Revert commit `f02d81018` changes from generic uploader / shared viewer
- [x] T3 Verify affected interaction paths and keep unrelated behaviors unchanged
- [x] T4 Run lightweight validation and prepare review notes

#### Task Details
##### T1 - Replace Reference Card Preview
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview`
- Plan Path: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview/plan.md`
- Goal:
  - remove the current `图片预览` dialog path from `ReferenceImageCard`
  - wire existing-image click to the history zoom viewer
- Files / Modules:
  - `ui/src/components/layout/workbench/view/customers/scenes/components/ReferenceImageCard.tsx`
- Write Set:
  - `ui/src/components/layout/workbench/view/customers/scenes/components/ReferenceImageCard.tsx`
- Acceptance:
  - click existing image opens history zoom viewer
  - old `图片预览` dialog is no longer used here
  - mask click precedence is preserved
- Test Points:
  - existing image click
  - base-slot mask click
  - remove and edit buttons stop propagation correctly
- Rollback:
  - restore original local `Dialog` preview block in `ReferenceImageCard.tsx`

##### T2 - Revert Mis-scoped Generic Uploader Change
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview`
- Plan Path: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview/plan.md`
- Goal:
  - remove the previous BSD uploader special preview path that was added in the wrong component chain
- Files / Modules:
  - `ui/src/components/ui/vines-uploader/files.tsx`
  - `ui/src/components/layout/workbench/view/customers/scenes/components/HistoryImageDetailDialog.tsx`
- Write Set:
  - `ui/src/components/ui/vines-uploader/files.tsx`
  - `ui/src/components/layout/workbench/view/customers/scenes/components/HistoryImageDetailDialog.tsx`
- Acceptance:
  - commit `f02d81018` behavior no longer exists in generic uploader path
  - `HistoryImageDetailDialog` returns to serving history/detail usage only
- Test Points:
  - generic uploader image preview path restored
  - BSD history viewer existing behavior preserved
- Rollback:
  - reapply the reverted generic-uploader BSD preview logic

##### T3 - Verify Shared Viewer Reuse Compatibility
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview`
- Plan Path: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview/plan.md`
- Goal:
  - ensure `ReferenceImageCard` can safely pass minimal image payload to `HistoryImageDetailDialog`
- Files / Modules:
  - `ui/src/components/layout/workbench/view/customers/scenes/components/ReferenceImageCard.tsx`
  - `ui/src/components/layout/workbench/view/customers/scenes/components/HistoryImageDetailDialog.tsx` only if compatibility gap remains after T2
- Write Set:
  - planned write target is `ReferenceImageCard.tsx`
  - `HistoryImageDetailDialog.tsx` only if required to support the card reuse after revert
- Acceptance:
  - no parent scene API changes
  - no regression to existing BSD history usage
- Test Points:
  - payload shape compatible with `HistoryImage`
  - dialog open/close lifecycle is clean
- Rollback:
  - revert the reuse wiring and return to the local dialog

##### T4 - Validation and Review Prep
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview`
- Plan Path: `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview/plan.md`
- Goal:
  - run lightweight validation proportional to the change
- Files / Modules:
  - modified files only
- Write Set:
  - none unless a failing validation requires a small follow-up fix within plan
- Acceptance:
  - `git diff --check` passes
  - any targeted test run is recorded
  - review checklist can be completed
- Test Points:
  - diff hygiene
  - targeted behavior verification list
- Rollback:
  - revert only task-scoped edits in the worktree

#### Dependencies
- Existing shared viewer: `ui/src/components/layout/workbench/view/customers/scenes/components/HistoryImageDetailDialog.tsx`
- Actual BSD custom-scene entry component: `ui/src/components/layout/workbench/view/customers/scenes/components/ReferenceImageCard.tsx`

#### Risks and Notes
- Keep unrelated `.gitignore` dirty state out of this workflow.
- Do not format unrelated files.
- Do not run full builds.
- Commit messages must remain English.

#### Parallelism Assessment
- No sub-agent dispatch planned.
- Reason:
  - change scope is small and tightly coupled to one component plus one shared viewer contract
  - user requires plan confirmation before coding

#### Issue List
- none

阻塞：否
进入 3.2
禁止派发子Agent

### Stage 3.3 - Code Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
  - 已完成 `git diff --check`
  - 已完成修改文件的 `prettier` 与 `eslint`
  - 未执行浏览器人工回归
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过

### Stage 4 - Change Archive
- change file:
  - `D:/project/MyFlowHub3/worktrees/monkey-fix-bsd-reference-image-preview/docs/change/2026-03-23_bsd-reference-image-preview.md`
- code commit:
  - `17bf559af fix: use history viewer for bsd reference image preview`
