# Plan Archive - Desktop 多面板嵌套停靠

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `codex/desktop-nested-docking`
- Base: `master@0dd314244849a024bcc6eba8c6c08b3d12df2bbc`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\desktop-nested-docking\docs`
- Code Repos: canonical `MyFlowHub` monorepo only
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-nested-docking`
- Current Stage: `$m-test` passed; `$m-archive` closeout in progress
- Remote / publication: no remote configured; no push, release, or publication is authorized

## Stage Records

### Initialization

- `guide.md`: read. Applicable rules include Chinese commits, canonical repository `docs/`, sibling `worktrees/`, Windows/Wails validation, and preservation of unrelated main-checkout changes.
- Project/docs/code repo: `D:\project\MyFlowHub3` is the umbrella root; the canonical monorepo owns code and governed docs; only `apps/desktop` and Desktop docs participate.
- Dedicated worktree: `D:\project\MyFlowHub3\worktrees\desktop-nested-docking` on `codex/desktop-nested-docking`, created from committed `master@0dd3142`.
- Main preservation boundary: the main checkout contains uncommitted transitional two-panel edge-docking code plus unrelated Agent Gateway docs, `guide.md`, design demos, verification images, and generated assets. None were staged, overwritten, restored, or copied wholesale.
- Baseline: implement final View v3 from committed `master`; the dirty transitional implementation is not an implicit source of truth.
- Subagents: none. Current host policy does not authorize proactive delegation, and schema/domain/rendering/drag changes are tightly coupled.

### Discuss - Discovery And Requirements Shaping

#### Goal

Replace the three-or-more-widget flat grid with a data-driven docking layout supporting arbitrary horizontal/vertical nesting, exact insertion, smooth resizing, persistence, migration, and keyboard accessibility without dedicated layout buttons.

#### Scope

- View persistence schema and migration.
- Frontend layout domain, recursive rendering, separators, DnD targets/previews and styling.
- Stable feature/requirement/spec/decision docs.
- Automated, packaged Windows, persistence, accessibility, performance, and visual validation.

#### Assumptions

- One visible widget is one layout leaf; Resource reference, renderer and settings stay in `widgets`.
- Direct “添加到工作区” appends at the right of the whole workspace; dragging chooses a panel edge, divider, or workspace edge.
- Center drop is inactive in phase one; no tab stack is created.
- Existing limits remain 32 Views per Profile and 64 widgets per View.
- Window-size constraints never rewrite persisted topology.

#### Open Questions

- None blocking. Hit-zone thickness, pane minimum pixels and keyboard step size may be tuned within the acceptance criteria.

#### Options Considered

| Option | Result | Reason |
| --- | --- | --- |
| Extend flat 12-column grid | Rejected | Cannot express local cross-axis splits or one ratio authority. |
| Pure binary split tree | Rejected as primary | Correct, but adds artificial nesting for ordinary three-column insertion. |
| Project-owned n-ary split tree | Selected | Supports direct sibling insertion and cross-axis nesting with current React/dnd-kit/renderers. |
| Full docking library | Deferred | FlexLayout/Golden Layout/Lumino would also take over tabs, model, styling and lifecycle. |

#### Rejected Options

- Dedicated “左右 / 上下 / 交换” buttons.
- Infinite pixel canvas.
- Center tab stacks, floating/cross-window drag and automatic layout presets.

#### Recommended Direction

Use a canonical n-ary split tree. Same-axis panel drops insert into an existing parent, cross-axis drops wrap only the target leaf, divider drops insert at the exact sibling index, and workspace-edge drops split the root. Persist normalized weights and render each adjacent pair with an accessible separator.

#### Research Summary

- VS Code validates panel-edge docking and resizable horizontal/vertical groups.
- Golden Layout and FlexLayout validate serializable row/column trees with relative sizes.
- React Mosaic's multi-child split model supports n-ary same-axis insertion without forced binary nesting.
- WAI-ARIA Window Splitter supplies the separator role/value/orientation and keyboard baseline; packaged behavior must still be tested.
- Full sources and comparisons: `docs/intake/2026-08-30_desktop-nested-docking-layout.md`.

#### Worktree / Branch / Docs Root Status

- Worktree: ready.
- Branch: ready.
- Docs root: confirmed, repo-local and local-only.
- Main dirty state: preserved and excluded.
- Handoff to plan: complete.

#### Issue List

- None blocking.

### Plan - Requirements And Architecture

#### Discussion Summary

The user confirmed the n-ary direction by entering `$m-plan`. The key layouts are three forms of one model:

- `A | C | B`: insert C before B in a horizontal split.
- `A | (B / C)`: replace B with a vertical split.
- `(A | B | C) / D`: replace the root with a vertical split.

#### Accepted / Rejected Requirements

- Accepted: arbitrary nesting, exact middle/root insertion, drag-only composition, smooth adjacent resizing, persistence, v1/v2 migration, strict validation, keyboard equivalence, actual-future-layout preview, bounded performance.
- Rejected/deferred: flat-grid patch, full docking dependency, tab stacks, floating windows, cross-window drag, presets, brand changes and publication.

#### Requirements Analysis

##### Goal

Make every View layout a deterministic persistent split topology so three or more panels remain as composable and adjustable as two panels.

##### Scope

- Go View document v3, migration and safety.
- TypeScript v3 types and pure immutable topology operations.
- Recursive React renderer and nested splitter interaction.
- dnd-kit collision/intent resolution, exact preview, add/move/remove and live announcements.
- CSS, canonical docs, tests, production assets, packaged Wails validation, archive and safe local convergence in their proper phases.

##### Use Cases

1. Drop C at B's left edge to produce `A | C | B`.
2. Drop C at B's bottom edge to produce `A | (B / C)`.
3. Drop D at the workspace bottom edge to produce `(A | B | C) / D`.
4. Drop on a divider to insert at that exact sibling boundary.
5. Resize any nested adjacent pair by pointer or keyboard and persist it.
6. Move/remove panes and normalize empty, one-child and same-axis artifacts.
7. Load v1/v2, preserve all widgets, migrate deterministically, save v3 and restore after restart.
8. Keep detached/forbidden/offline renderer states inside unchanged widget renderers.

##### Functional Requirements

1. View v3 stores one recursive `layout_root`; `widgets` retains identity, Resource, renderer and settings.
2. A leaf references one widget ID. A split has an axis, at least two children and one positive finite normalized weight per child.
3. Non-empty View: exactly one root and every widget exactly once. Empty View: no root.
4. Normalize zero/one-child splits and flatten same-axis nested splits; canonical output has no redundant adjacent same-axis split.
5. Same-axis edge drops insert beside the target; cross-axis drops wrap only the target; workspace-edge drops act on the root.
6. Divider intent carries drag-start parent path and insertion index. Same-parent source removal adjusts the index transactionally.
7. Uniform split insertion rebalances uniformly. Otherwise a panel-edge insertion splits the target weight; a divider insertion splits the following sibling. Same-parent reorder carries the child's weight.
8. First direct add creates the root leaf; later direct adds insert at the root right.
9. Collision priority: divider > workspace outer edge > panel edge > workspace background. Panel center is a dead zone.
10. Preview renders the hypothetical tree with the real layout tracks and highlights the future pane; identical intents do not rebuild state.
11. Each adjacent pair has a separator. Pointer movement updates local RAF-coalesced draft weights; release commits once; cancel restores.
12. Separators expose orientation, controls and values; Arrow/Shift+Arrow, Home/End, Enter and double-click are supported.
13. Widget title drag and dnd-kit keyboard drag are the layout controls; Chinese live instructions/announcements provide the accessible path without layout buttons.
14. Runtime minimums are computed recursively (initial target 200 px width and 120 px height per leaf, tuned during UI validation). Insufficient space scrolls instead of rewriting topology.
15. v1/v2 migrates in memory and writes v3 only on save. Before the first overwrite of an older file, create a permission-restricted one-time pre-v3 snapshot that is never silently replaced.
16. Migration is exact for 0/1/2 widgets and shipped 3+ grids. Other valid legacy grids preserve all widgets and group deterministically by `(y, x, original index)`; any visual approximation is documented.
17. Reject unknown versions/fields, invalid kinds/axes, bad weights/counts, duplicate/missing leaves, redundant nesting, excessive nodes/depth, invalid widget settings and legacy corruption without overwriting disk.

##### Non-functional Requirements

- No new runtime dependency; reuse React, dnd-kit, existing primitives, CSS tokens and renderers.
- Operations are O(layout nodes), bounded by 64 leaves and depth 32; rendering is O(nodes).
- Drag preview updates only on intent change; resize preview remains local and RAF-coalesced.
- Preserve mineral-blue/graphite flat panes, dark theme, reduced motion, compact headers and restrained radius.
- No silent disk/API fallback. Invalid View state gives an actionable error and keeps the last valid View.
- Preserve Profile isolation, revision conflicts, atomic save, Resource permissions and connection behavior.
- Regenerate `dist` canonically; never copy it or Wails bindings from dirty main.

##### Inputs / Outputs

- Inputs: v1/v2/v3 `views.json`, Resource descriptors, widgets, pointer/keyboard drag events, typed DockIntent, splitter events and workspace geometry.
- Outputs: canonical v3 tree, unchanged Resource payloads, one dirty View update per completed drop/resize, persisted topology/weights, live status and production assets.

##### Edge Cases

- Empty/single/64-leaf Views and depth limit.
- Self drop, adjacent divider drop, same-parent reorder, cross-parent move and root move.
- Removing a branch's last leaf and collapsing ancestors.
- Corners, center dead zone, overlapping root/panel/divider zones.
- Resource disappears or duplicates during drag.
- Tiny container, pointer capture loss/cancel, resize during window change and nested minimum pressure.
- Corrupt v3: unknown fields, one-child split, mismatched/non-finite weights, duplicate/missing/unused widgets, redundant axes and limit overflow.
- v1/v2 without optional layout, vertical two-pane layout, irregular legacy grid and repeated post-migration save.
- Read-only disk or backup/rename failure.
- Profile/View switch with unsaved layout changes.

##### Acceptance Criteria

- Dragging alone produces `A | C | B`, `A | (B / C)` and `(A | B | C) / D`.
- Three or more panes can be inserted, moved, removed and resized at arbitrary depth.
- Preview geometry matches the post-drop topology and distinguishes panel/divider/root scope.
- Save/reopen/restart restores topology, order and weights.
- v1/v2 retains every widget; malformed/unknown data stays unmodified with a clear error.
- Direct add still fills the first pane and appends later additions at root right.
- No “左右 / 上下 / 交换” buttons and no center-created hidden stack.
- Detached/offline/forbidden states, View revisions, themes and unsaved prompts do not regress.
- Frontend tests/build, Desktop and proportional full Go tests, Wails Windows build, `git diff --check`, generated review and packaged GUI drag/resize/restart smoke pass.

##### Risks

- v3 is backward-incompatible; an old binary needs the pre-v3 snapshot.
- Target overlap can select the wrong scope without explicit priority/dead zones.
- Divider paths shift on same-parent moves and require one atomic tested operation.
- Recursive minimums can overflow narrow views; topology must not be rewritten.
- Per-pointer React commits would reintroduce stutter.
- Dirty main overlaps source/docs/dist; archive cannot assume `git merge --ff-only` will work.
- Windows line endings and tracked generated output can create false drift.

#### Architecture Design

##### Overall Solution

Go and TypeScript share the following conceptual View v3 contract; `layout_root` is the only layout authority:

```ts
type ViewLayoutNode =
  | { kind: 'leaf'; widget_id: string }
  | {
      kind: 'split'
      axis: 'horizontal' | 'vertical'
      children: ViewLayoutNode[]
      weights: number[]
    }
```

A pure frontend module owns validation and immutable topology actions. `Workspace` recursively renders split nodes with CSS grid child/separator tracks. `App` resolves dnd-kit collisions into a `DockIntent`, previews a hypothetical tree, and commits one canonical View on drop.

##### Alternatives Considered

- Keep canonical `x/y/w/h`: rejected; it creates dual truth. These fields remain only in legacy decode/migration structs.
- Persist split IDs: rejected; persisted topology needs content, axis, order and weights. Render/drag paths are ephemeral.
- Silently repair invalid API input: rejected; Go validates external data, and frontend normalizes only its own valid mutations.
- Rewrite v2 spec in place: rejected; add v3 and explicitly mark v2 superseded.
- Persist window reflow: rejected; pixel minimums and overflow are presentation concerns.

##### Module Responsibilities

- `apps/desktop/views.go`: v3 structs, version-discriminated strict decode, v1/v2 migration, recursive validation, pre-v3 snapshot and atomic persistence.
- `apps/desktop/app_test.go`: migration, validation, backup, corruption, revision and restart fixtures.
- `frontend/src/types.ts`: canonical v3 types; current `ViewWidget` drops grid coordinates.
- New `frontend/src/workspace-layout.ts`: `DockIntent`, validation, normalize/add/dock/move/remove/resize, weights, paths and geometry/min-size helpers.
- New `frontend/src/workspace-layout.test.ts`: pure domain coverage.
- `frontend/src/components/Workspace.tsx` and optional narrow `WorkspaceLayout.tsx`: recursive split/leaf renderer, separators, local resize drafts, minimums, overflow and preview overlay.
- `frontend/src/App.tsx`: load boundary, active drag/intent, priority collision, default add, one-drop commit, cancel and live announcements.
- `frontend/src/store.ts`: retain Explorer/resource utilities; remove/narrow legacy flat-grid helpers.
- `frontend/src/style.css`: recursive tracks, pane borders, separator focus/hit targets, outer/panel targets, preview, overflow, themes and reduced motion.
- Frontend tests: domain, component, DnD integration, save/reopen and regressions.
- `frontend/dist/**`: canonical build output only.
- Stable docs: Desktop feature/requirement, new v3 spec, superseded v2 spec, new ADR/old ADR link, indexes and intake forward links.

##### Data / Call Flow

```text
v1/v2 views.json
      │ strict legacy decode + validate
      ▼
deterministic migration ──► canonical View v3 ──► recursive renderer
                                  │                     ├── local RAF resize ──► release commit
                                  │                     └── DockIntent ──► hypothetical preview ──► drop commit
                                  ▼
                    revision check + pre-v3 snapshot + atomic v3 save
```

##### Interface Drafts

- `ViewDocument.version = 3`.
- `ViewDefinition.layout_root?: ViewLayoutNode`, required iff widgets are non-empty.
- v3 `ViewWidget`: `id`, `owner_node_id`, `resource_name`, `renderer`, optional `settings`; no canonical `x/y/w/h`.
- `DockIntent` variants:
  - `panel-edge(targetWidgetID, side)`
  - `split-gap(parentPath, insertionIndex)`
  - `root-edge(side)`
  - `empty-workspace`
- `dockView(view, source, intent)`, where source is an existing widget ID or new widget.
- `resizeSplit(root, splitPath, dividerIndex, nextLeadingWeight, constraints)`.
- Limits: 64 leaves, depth 32, total nodes 127, split children 2–64.
- Weights: finite, positive, normalized to one within a documented tolerance.

##### Error Handling and Safety

- Read version first, then strictly decode a version-specific struct.
- Validate legacy before migration and v3 after decode; migration output passes the same validator as saves.
- Backup, serialization, sync, rename and revision errors preserve prior `views.json`.
- One-time snapshot is permission restricted and not overwritten. Restoring it intentionally loses later v3-only layout edits.
- Frontend invalid API state becomes a load/workspace error, not a synthesized grid.
- No-op/self drops return the original View without dirtying it.
- Drop targets accept only valid active resource/widget payloads.
- No Resource body, permission, identity, permit or secret enters layout nodes or announcements.

##### Performance and Testing Strategy

- Table-driven pure tests: same/cross-axis, root/gap, index adjustment, collapse/flatten, weights, no-op and limits.
- Go tests: v1/v2 0/1/2/3+/vertical/irregular fixtures, widget retention, v3 round trip, backup, strict fields, bad leaves/weights/depth, corruption and revisions.
- Component tests: recursive DOM, every separator's ARIA, pointer draft/commit/cancel, keyboard bounds/equalization, overflow, preview, no buttons, detached leaf and 64-leaf bounded render.
- App tests: direct add, resource/widget drag, priority, intent dedupe, unsaved state, save/reopen and switch confirmation.
- Execute validation: dependency/binding preflight, focused/full Vitest, TypeScript/Vite build, Desktop Go tests, proportional full Go, `git diff --check`, generated review.
- `$m-test`: packaged Wails with 6–8 panels, normal/narrow sizes, themes, three reference layouts, pointer/keyboard resize, save/restart/migration and visual evidence.

##### Extensibility Design Points

- Future `stack` can be a new explicit node kind; phase one does not overload center drop.
- N-ary splits avoid unnecessary depth for many same-axis siblings.
- DockIntent is renderer-independent and reusable by future commands/menus.
- Renderer leaves remain unaware of docking topology.
- Floating windows or full tab management trigger a future library re-evaluation.

#### Issue List

- None blocking.

### Stage 3.1 - Planning

#### Project Goal and Current State

Committed production supports exact direction/ratio only for two widgets and a flat grid for three-plus. Dirty main contains a transitional four-edge interaction limited to one/two widgets; it is preserved but excluded. This worktree will implement canonical View v3.

#### Docs Governance Routing Decision

- Docs root: `D:\project\MyFlowHub3\worktrees\desktop-nested-docking\docs`.
- Root `plan.md` / `todo.md`: active workflow control-plane exception; archive later under `docs/plan/`.
- Intake: existing focused record, later forward-linked.
- Feature/requirements: clarify current behavior and durable acceptance.
- Specs: add `desktop-resource-workspace-v3.md`, mark v2 superseded, update index.
- Decision: add n-ary/no-library ADR; mark the old response-grid choice partially superseded.
- Change: create only during `$m-archive`.
- Lessons: none known; add only if reusable migration/DnD/resize/generated failure knowledge emerges.

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake: `docs/intake/2026-08-30_desktop-nested-docking-layout.md`
- Feature: `docs/features/desktop.md`
- Requirement: `docs/requirements/desktop-resource-workspace.md`
- Specs: v2, planned v3 and `docs/specs/build-and-ci.md`
- Decision: `docs/decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md`
- Planned decision: `docs/decisions/2026-08-30_desktop-n-ary-docking-layout.md`
- Lessons: `frontend-worktree-wailsjs-missing.md`, `frontend-build-empty-node-modules.md`, `frontend-and-powershell-preflight.md`, `windows-clean-checkout-eol-and-generated-drift.md`, `wails-embed-dist-placeholder.md`

#### Stable Docs Impact

- Intake impact: clarify.
- Feature impact: clarify.
- Requirements impact: clarify.
- Specs impact: add v3 and supersede v2.
- Decision impact: add and partially supersede the 2026-08-28 response-grid choice.
- Lessons impact known at planning time: none.

#### Executable Task List

- DOC01 — View v3 stable docs and ADR.
- VIEW01 — Go v3 persistence/migration/validation/rollback snapshot.
- LAYOUT01 — TypeScript split-tree domain.
- RENDER01 — Recursive panes and nested separators.
- DOCK01 — Drag-only docking and exact previews.
- QA01 — Execute-stage automated/build/generated validation.
- TEST01 — Later `$m-test` packaged GUI/migration/accessibility acceptance.
- ARC01 — Later `$m-archive` docs/commits/integration/cleanup.
- STACK01 — Center tab stacks.
- FLOAT01 — Floating/cross-window/preset layouts.
- LIB01 — Full docking framework adoption.
- BRAND01 — Brand/icon work.
- PUB01 — Push/release/publication.

#### Execution Scope After Approval

##### Will Execute

- DOC01, VIEW01, LAYOUT01, RENDER01, DOCK01, QA01.

##### Will Not Execute Now

- TEST01 — phase-gated until execute checks pass, then `$m-test`.
- ARC01 — phase-gated until `$m-test` passes, then `$m-archive`.
- STACK01 — out of scope; center remains inactive.
- FLOAT01 — out of scope; separate product/window-lifecycle work.
- LIB01 — rejected for current scope.
- BRAND01 — separate task.
- PUB01 — unauthorized and no remote.

#### Task Details

##### DOC01 - Establish View v3 stable truth

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: replace three-plus flat-grid truth and record the n-ary decision.
- Files / Modules: Desktop feature/requirement; new v3 and superseded v2 specs; new/old ADR; indexes; intake links.
- Write Set: behavior, acceptance, schema/migration/accessibility contract, alternatives and links.
- Acceptance: exactly one current spec; all stable layers agree on DnD, no buttons, migration, validation and non-goals.
- Test Points: links/indexes, docs diff, impact re-check, no brand/Agent Gateway/publication changes.
- Rollback: revert scoped stable docs/indexes; retain intake source evidence.

##### VIEW01 - Go View v3 store and migration

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: safely persist the n-ary tree.
- Files / Modules: `apps/desktop/views.go`, `apps/desktop/app_test.go`; no Wails method signature change expected.
- Write Set: version-specific structs, migration, validation, snapshot and atomic save errors.
- Acceptance: migration retains all widgets; v3 round-trips; invalid topology never overwrites; revision/profile limits remain.
- Test Points: 0/1/2/3+, vertical, irregular, backup failure/idempotence, unknown fields/version, leaf set, weights, limits, corruption, revision.
- Rollback: before v3 save, code rollback; after save, restore pre-v3 snapshot before running v2, accepting loss of later v3-only edits.

##### LAYOUT01 - TypeScript layout domain

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: centralize canonical mutations and remove flat-grid authority from components.
- Files / Modules: `types.ts`, new `workspace-layout.ts/.test.ts`, scoped `store.ts/.test.ts` cleanup.
- Write Set: types, validation, normalization, add/dock/move/remove/resize, weights, paths and geometry/min-size helpers.
- Acceptance: three reference topologies and exact gap insertion work; invariants always hold; invalid/no-op operations are explicit.
- Test Points: all insertion/move/remove cases, weights, limits, non-finite values and immutability.
- Rollback: restore v2 frontend only together with VIEW01.

##### RENDER01 - Recursive panes and nested resizing

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: render full-area nested panes with smooth accessible resizing.
- Files / Modules: `Workspace.tsx`, optional `WorkspaceLayout.tsx`, component tests and `style.css`.
- Write Set: recursive grids, pane IDs, separators, RAF drafts, commit/cancel/reset, minimums and overflow.
- Acceptance: arbitrary tree fills space; every separator is smooth/keyboard operable; no layout buttons; detached widgets unchanged.
- Test Points: DOM/order, separator ARIA, pointer/keyboard, path isolation, overflow and theme/reduced-motion selectors.
- Rollback: revert only in lockstep with LAYOUT01/VIEW01.

##### DOCK01 - Drag-only docking and exact preview

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: panel/divider/workspace dragging is the sole composition interaction.
- Files / Modules: `App.tsx`, Workspace layout components, App/Workspace tests and styles.
- Write Set: drag payload, target metadata, priority collision, dead zone, hypothetical preview, DragOverlay, add/move, announcements and cancel/error paths.
- Acceptance: all reference layouts plus exact gap insertion work; overlap priority is correct; preview equals result; no buttons.
- Test Points: priority, corners/center, intent dedupe, resource/widget/self/disappearing payloads, keyboard announcements, dirty/save.
- Rollback: revert together with v2 schema; never save partially supported v3.

##### QA01 - Execute-stage validation and production assets

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: prove integrated schema/domain/UI before heavy GUI test.
- Files / Modules: tests, tracked `frontend/dist/**`, canonical bindings only if preflight requires.
- Write Set: focused tests and canonical generated frontend assets.
- Acceptance: frontend suites/build, Desktop and proportional full Go tests, migration tests, diff/generated checks pass.
- Test Points: `npm ci`, `npm test`, `npm run build`, `GOWORK=off go test ./apps/desktop/... -count=1`, proportional `go test ./... -count=1`, generated review.
- Rollback: remove workflow-generated binaries/evidence only; retain branch on failure.

##### TEST01 - Packaged GUI and migration acceptance

- Scope: later `$m-test`.
- Acceptance: actual Wails passes reference layouts, 6–8 panes, exact insertion, pointer/keyboard resize, narrow overflow, themes, save/restart and v2 migration.
- Safety: isolated config root and workflow-owned processes/evidence only.

##### ARC01 - Archive and safe local convergence

- Scope: later `$m-archive`.
- Acceptance: docs impact/change/plan archive indexed; Chinese commits; unrelated dirty main byte-identical; no push; cleanup only after verified integration.
- Safety: preserve branch/commits and an external binary patch/hash manifest before overlapping main convergence. Stop with worktree intact if equivalence cannot be proven.

##### STACK01 / FLOAT01 / LIB01 / BRAND01 / PUB01

- Scope: will not execute now.
- Reasons: respectively separate tab-stack decision; separate window lifecycle; rejected dependency; separate brand task; unauthorized publication.

#### Dependencies

- `DOC01 → VIEW01 → LAYOUT01 → RENDER01 → DOCK01 → QA01`.
- `TEST01` starts after execute gates pass.
- `ARC01` starts after `TEST01` and final `$m-docs` impact review.

#### Risks and Notes

- DOC01 must establish v3 and explicit v2/ADR supersession before completion.
- Dirty main overlaps App/Workspace/store/tests/styles/docs/dist; normal fast-forward is expected to refuse until safe convergence.
- Agent Gateway docs/indexes, `guide.md`, demos and verification are unrelated and must remain untouched.
- Wails methods stay JSON-string boundaries, so method bindings should not change; preflight still checks missing/foreign bindings and drift.
- A source rollback alone is invalid after v3 data persistence.
- No subagent work is planned or authorized.

#### Parallelism Assessment

- Execute sequentially. VIEW01/LAYOUT01 share schema; RENDER01/DOCK01 share state, metadata, CSS and tests. Parallel edits add contract risk.

#### Issue List

- None blocking.

#### Approval Gate

- Plan drafted: yes.
- User implementation approval: granted by explicit `$m-execute` invocation.
- Approved Task IDs: `DOC01, VIEW01, LAYOUT01, RENDER01, DOCK01, QA01`.
- Deferred phase-gated Task IDs: `TEST01, ARC01`.
- Blocked: no.
- Enter `$m-execute`; do not enter `$m-test` or `$m-archive` until their gates are satisfied.

### Execute - Implementation And Lightweight Validation

- Completed Task IDs: `DOC01, VIEW01, LAYOUT01, RENDER01, DOCK01, QA01`.
- Implemented the governed View v3 contract, strict v1/v2 migration with one-time pre-v3 snapshot, canonical TypeScript n-ary split-tree mutations, recursive full-area rendering, nested pointer/keyboard separators, drag-only edge/divider docking, dead centers, and hypothetical-layout preview.
- Removed the v2 flat-grid frontend authority and dedicated `左右 / 上下 / 交换` layout controls without adding dependencies or changing the Wails method boundary.
- Lightweight validation passed: 44 frontend tests, TypeScript/Vite production build, Desktop Go tests, full repository Go tests, canonical Wails Windows production build, governed-doc link resolution, and `git diff --check`.
- Wails bindings were regenerated by the canonical build and are unchanged. Production `frontend/dist/**` was regenerated and is intentionally changed.
- Heavy packaged GUI, restart/migration fixture, 6–8 pane usability, theme/narrow-window and pointer smoothness acceptance remains `TEST01` under `$m-test`.
- No commit, merge, archive, push, publication, or worktree cleanup was performed.

### Test - Packaged GUI And Regression Validation

- Completed Task ID: `TEST01`.
- Iterations: one implementation baseline and one validation pass; the only setup correction was moving the legacy fixture from the config root to the active Profile directory. No product defect signature repeated.
- Automated gates passed: Vitest 8 files / 44 tests, `go test ./...`, TypeScript/Vite production build and canonical Wails Windows production build.
- Packaged GUI passed against a workflow-owned local Hub and isolated cloned Profile config:
  - v2 four-pane migration retained all Widgets;
  - pointer and keyboard separators adjusted the intended pair and created one dirty View update;
  - real drag created a vertical root with a full-width top pane and a nested three-column lower split;
  - six panes rendered in light/dark themes;
  - at 986 px, eight panes preserved recursive minimums, exposed a horizontal scrollbar and scrolled to the right-side panes;
  - first v3 save created an unchanged v2 `views.pre-v3.json`, and save/reopen/restart restored topology and weights;
  - detached and real Forbidden Resource states remained explicit in Inspector and Widget.
- Final package: `apps/desktop/build/bin/myflowhub-desktop.exe`, SHA-256 `3C94D2DB1B474D43D03B0529B472BA41D39CF9E23313CC17F0504C59269DE560`.
- Evidence: `artifacts/m-test/desktop-nested-docking/README.md` and screenshots in the same directory.
- Test result: Passed. Archive-ready: yes.

### Archive - Documentation And Local Closeout

- Completed Task ID: `ARC01`.
- `$m-docs` impact review confirmed the stable chain: intake updated; feature and requirements clarified; v3 spec added and v2 superseded; n-ary ADR added; no new lesson required.
- Change entry: `docs/change/2026-08-30_desktop-nested-docking-layout.md`.
- Plan archive: `docs/plan/plan_archive_2026-08-30_desktop-nested-docking-layout.md`.
- No brand/icon asset or brand ADR was created, changed or archived by this workflow.
- No remote exists and no push, release or publication was performed.
- The final integration and cleanup results are recorded in the root closeout plan after the control-plane merge.
