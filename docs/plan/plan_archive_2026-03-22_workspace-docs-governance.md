# 2026-03-22 Workspace Docs Governance

## Goal
- Rebuild the root workspace docs tree to the governed taxonomy `requirements/specs/plan/change/lessons`.
- Migrate existing root docs content to the new topology without leaving legacy paths behind.
- Align the root docs entry points with the Server repo's cleaned docs structure.

## Current Status
- Root workspace docs now use the governed taxonomy `requirements/specs/plan/change/lessons`.
- Root entry docs now route readers to `docs/plan/` and `docs/specs/protocol_map.md`.
- This workflow uses the pure-migration strategy: canonical files move to new paths and in-repo references are updated instead of keeping compatibility stubs.

## Workflow Info
- Repository: `D:\project\MyFlowHub3`
- Branch: `chore/root-docs-governance`
- Base branch: `master`
- Base commit: `df9adb5`
- Worktree: `D:\project\MyFlowHub3\worktrees\root-docs-governance`
- Plan path: `D:\project\MyFlowHub3\worktrees\root-docs-governance\todo.md`
- Current stage: `4`
- Related worktree:
  - Repository: `MyFlowHub-Server`
  - Branch: `chore/server-docs-governance`
  - Worktree: `D:\project\MyFlowHub3\worktrees\server-docs-governance`
  - Plan path: `D:\project\MyFlowHub3\worktrees\server-docs-governance\todo.md`
  - Dependency: root entry links must point to the server worktree's final docs locations
  - Responsibility boundary: root worktree owns workspace navigation, global plan/change archives, and workspace-level spec entry points; server worktree owns server-local stable specs and server-local archive indexes

## Docs Governor Routing Check
- Category focus: index / entry maintenance, docs tree bootstrap, plan routing, change archive continuity
- Related requirements: none existing for product behavior
- Related specs: root workspace docs topology and protocol map location
- Requirements impact: `none`
- Specs impact: `clarify`
- Notes:
  - This workflow does not change runtime behavior or acceptance criteria.
  - This workflow changes the location and navigation of stable documentation, so spec entry points and references must be updated consistently.

## Task Checklist
- [x] ROOT-DOC-001 Create the governed root docs tree and category indexes
- [x] ROOT-DOC-002 Migrate `docs/plan_archive/` to `docs/plan/` and rebuild the plan index
- [x] ROOT-DOC-003 Move root `docs/protocol_map.md` into `docs/specs/` and rebuild spec indexes
- [x] ROOT-DOC-004 Rewrite root navigation docs and control-plane references (`docs/README.md`, `plan.md`, `repos.md`)
- [x] ROOT-DOC-005 Update in-repo references affected by the pure migration
- [x] ROOT-DOC-006 Validate root docs integrity and prepare review notes

## Executable Tasks
### ROOT-DOC-001
- Owner: `MainAgent`
- Worktree: `D:\project\MyFlowHub3\worktrees\root-docs-governance`
- Write set:
  - `docs/README.md`
  - `docs/requirements/**`
  - `docs/specs/**`
  - `docs/plan/**`
  - `docs/change/README.md`
  - `docs/lessons/**`
- Goal: create the standard docs taxonomy and replace the old entry model with category indexes.
- Acceptance:
  - Root `docs/` contains `requirements/`, `specs/`, `plan/`, `change/`, `lessons/`.
  - Each category has a `README.md`.
  - Root `docs/README.md` points to the new categories instead of `plan_archive`.
- Tests:
  - `Test-Path` confirms each category and index file exists.
  - Manual scan shows `docs/README.md` reading order matches the new taxonomy.
- Rollback:
  - Revert the newly created category files and directory moves in this worktree.

### ROOT-DOC-002
- Owner: `MainAgent`
- Worktree: `D:\project\MyFlowHub3\worktrees\root-docs-governance`
- Write set:
  - `docs/plan/**`
  - all former `docs/plan_archive/**`
- Goal: replace the legacy `plan_archive` category with `plan` while preserving archive content.
- Acceptance:
  - Historical workflow plans live under `docs/plan/`.
  - The plan index is rebuilt under `docs/plan/README.md`.
  - No root entry document still routes users to `docs/plan_archive/`.
- Tests:
  - `rg -n "docs/plan_archive|plan_archive/" . -S` only returns intentional historical body text if any remain; root entry docs should be clean.
  - `docs/plan/README.md` opens all migrated plan files.
- Rollback:
  - Move files back from `docs/plan/` to `docs/plan_archive/`.

### ROOT-DOC-003
- Owner: `MainAgent`
- Worktree: `D:\project\MyFlowHub3\worktrees\root-docs-governance`
- Write set:
  - `docs/specs/protocol_map.md`
  - `docs/specs/README.md`
  - any references to root `docs/specs/protocol_map.md`
- Goal: place the workspace protocol map under `specs` without editing protected generated blocks unsafely.
- Acceptance:
  - `docs/specs/protocol_map.md` exists and preserves the generated block markers.
  - Root indexes reference the new spec path.
  - No generated block text is modified outside safe path moves or header/path note updates.
- Tests:
  - `rg -n "BEGIN GENERATED|END GENERATED" docs/specs/protocol_map.md`
  - diff review confirms generated block boundaries remain intact.
- Rollback:
  - Move the file back to `docs/protocol_map.md` and restore references.

### ROOT-DOC-004
- Owner: `MainAgent`
- Worktree: `D:\project\MyFlowHub3\worktrees\root-docs-governance`
- Write set:
  - `docs/README.md`
  - `plan.md`
  - `repos.md`
- Goal: make control-plane documents route readers to the new docs topology and the server repo's final spec paths.
- Acceptance:
  - Root entry docs explain what belongs in each category.
  - `plan.md` and `repos.md` no longer advertise `docs/plan_archive/` or top-level `docs/protocol_map.md` as current entry points.
  - Links to server docs use the migrated server spec locations.
- Tests:
  - `rg -n "docs/plan_archive|docs/protocol_map\\.md|repo/MyFlowHub-Server/docs/[2-7]-|repo/MyFlowHub-Server/docs/core\\.md|repo/MyFlowHub-Server/docs/权限\\.md" plan.md repos.md docs/README.md -S`
- Rollback:
  - Revert the control-plane doc edits from this workflow.

### ROOT-DOC-005
- Owner: `MainAgent`
- Worktree: `D:\project\MyFlowHub3\worktrees\root-docs-governance`
- Write set:
  - root repo markdown files that reference migrated docs paths
- Goal: update workspace-internal links so the pure migration does not leave broken canonical references.
- Acceptance:
  - High-value references in root repo point to `docs/plan/` and `docs/specs/`.
  - No root README/index document points to removed legacy locations.
- Tests:
  - targeted `rg` checks for old canonical paths
  - spot-check migrated links from key documents
- Rollback:
  - Revert the link rewrite commits in this worktree.

### ROOT-DOC-006
- Owner: `MainAgent`
- Worktree: `D:\project\MyFlowHub3\worktrees\root-docs-governance`
- Write set:
  - `todo.md`
  - review notes generated during 3.3
- Goal: prepare validation evidence for review and archive stages.
- Acceptance:
  - Validation commands and residual risks are recorded.
  - Cross-repo dependency status with server docs is captured.
- Tests:
  - `git diff --stat`
  - final targeted `rg` checks recorded in the review
- Rollback:
  - N/A for planning notes; revert if the notes are materially wrong.

## Dependencies and Order
- `ROOT-DOC-001` before all root rewrites
- `ROOT-DOC-002` before `ROOT-DOC-004` and `ROOT-DOC-005`
- `ROOT-DOC-003` before `ROOT-DOC-004` and `ROOT-DOC-005`
- Server worktree path migration must settle before final root links are considered complete

## Risks and Attention Points
- Pure migration increases link-break risk across historical markdown files.
- Root `docs/specs/protocol_map.md` is a synced generated artifact; protected block markers must survive the move.
- Some historical archive bodies may still mention old paths as historical facts; review must distinguish factual history from current navigation.

## Out of Scope
- No product requirement changes
- No runtime code or protocol implementation changes
- No commit/merge/cleanup actions until the workflow reaches the end state
