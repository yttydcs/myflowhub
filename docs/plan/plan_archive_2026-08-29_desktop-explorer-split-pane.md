# Plan - Desktop Explorer Node/Resource 上下分区

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `feat/desktop-explorer-split-pane`
- Base: `master@f79f165ccd90c328f5f46eee63af330a05ee019f`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane\docs`
- Code Repos: canonical `MyFlowHub` monorepo only
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane`
- Current Stage: `$m-archive` documentation/commit complete; safe local integration and cleanup blocked by protected dirty-main overlap
- Remote / publication: no remote configured; no push, release, or publication is authorized

## Stage Records

### Initialization

- `guide.md`: read from the main checkout. Applicable rules include Chinese commit messages, canonical `docs/`, sibling `worktrees/`, Windows frontend/Wails validation, and preserving unrelated user changes.
- Project/docs/code repo confirmation: `D:\project\MyFlowHub3` is the project root; `docs/` inside the canonical monorepo is the governed docs root; only `apps/desktop/frontend` and its stable docs participate.
- Base/worktree confirmation: clean dedicated worktree created at `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane` on `feat/desktop-explorer-split-pane` from `master@f79f165`.
- Main-checkout preservation boundary: do not stage, overwrite, restore, archive, or copy unrelated `guide.md`, `论文/**`, or `design-demos/**` changes.
- Prior verified UI delta: the main checkout contains a narrowly scoped uncommitted sidebar/ScrollArea containment fix in `frontend/src/style.css` plus regenerated `dist`. Execution must reproduce its semantic result in this branch so the fixed Profile footer and resource scrolling do not regress; it must not bulk-copy the dirty checkout.
- Subagents: none planned. Explorer state, accessibility, preferences, CSS, and tests share the same files and should be changed sequentially by the main agent.

### Discuss - Discovery And Requirements Shaping

#### Original Request

The user observed that Node and Resource rows mixed in one tree are difficult to browse. After comparing layouts, the user rejected the left/right design and confirmed an upper Node pane plus lower Resource pane, with a user-adjustable height split.

#### Problem / Opportunity

The existing Explorer gives Node and Resource the same tree-row treatment. With one Node exposing 25 Resources, Resource rows dominate the hierarchy, make Node relationships harder to scan, and incorrectly suggest that both object types share one navigation level. The current catalog/authority model already has a clean boundary: Nodes form the authoritative tree; Resources are direct children by ownership but resource path segments are presentation-only.

#### Confirmed Goals

- Show the authoritative Node hierarchy in an upper pane containing Node rows only.
- Show only the current Node's direct Resources in a lower pane.
- Group Resources by the first resource-name segment for presentation without creating a second authority or permission tree.
- Let the user resize the two panes vertically with pointer and keyboard interaction.
- Persist the split ratio per Profile with the existing non-secret UI preference mechanism.
- Keep Node focus/breadcrumb, arbitrary-depth traversal, Resource preview, drag/add, View persistence, themes, independent scrolling, and the fixed Profile/connection footer.
- Preserve real authority enforcement: availability and permissions continue to be decided by Hub calls, not by Explorer presentation.

#### Non-goals

- Do not grant permissions, mutate the local demo Hub policy, or add a Hub administration UI.
- Do not add effective-permission prediction or probe side-effecting capabilities.
- Do not redesign Resource renderers, Workspace widgets, View persistence, Settings, login, or Profile identity.
- Do not add server pagination/lazy loading or a second resource model.
- Do not design, replace, generate, or archive project icon assets.
- Do not add frontend dependencies.

#### Assumptions

- The Node pane and Resource pane remain inside the existing `资源` sidebar tab.
- Default split is approximately 35% Node / 65% Resource; exact pixel clamps are responsive to available height.
- Node and Resource each receive their own search field because they are distinct navigation domains. Node search retains ancestor paths; Resource search applies to the current Node.
- Closing the Inspector does not discard the current Node context in the Explorer.
- The Resource group prefix is derived only for display; stable Resource identity remains `(owner_node_id, full resource name)`.

#### Open Questions

- None blocking. Exact minimum pane pixels and keyboard step size may be tuned during implementation while preserving the acceptance bounds below.

#### Options Considered

| Option | Result | Reason |
| --- | --- | --- |
| Keep mixed Node/Resource tree | Rejected | Resource-heavy Nodes obscure the authoritative hierarchy. |
| Fixed upper/lower panes | Rejected | Cannot adapt to profiles with many Nodes versus many Resources. |
| Resizable panes without bounds | Rejected | Users can accidentally make a pane inaccessible. |
| Left/right Node and Resource panes | Rejected by user | Consumes too much horizontal workspace and creates a four-column state when Inspector is open. |
| Bounded, persistent upper/lower split | Selected | Separates object types, preserves vertical list capacity, and adapts to catalog shape. |

#### Recommended Direction

Use a master/detail Explorer: a WAI-ARIA Node tree in the upper pane controls the current owner Node; a grouped Resource list in the lower pane displays that Node's direct resources. A semantic horizontal separator commits a clamped ratio into the active Profile's existing UI preference document.

#### Research Summary

No new web research was needed. The current implementation, WAI-ARIA requirements already recorded in the canonical spec, the existing Radix ScrollArea primitive, and project tests provide sufficient constraints.

#### Worktree / Branch / Docs Root Status

- Dedicated worktree: ready.
- Semantic branch: ready.
- Governed docs root: identified and local-only.
- Handoff to plan: unblocked.

#### Issue List

- None blocking.

### Plan - Requirements And Architecture

#### Discussion Summary

Adopt upper/lower panes. Node and Resource navigation become separate derived views over the same topology/catalog index. The split is bounded, accessible, resettable, and persisted per Profile. The design must not regress the previously verified sidebar containment fix.

#### Accepted / Rejected Requirements

- Accepted: upper Node pane, lower current-Node Resource pane, independent scrolling, resizable height, per-Profile persistence, keyboard equivalent, grouping by resource-name prefix.
- Rejected: left/right layout, unbounded dragging, fixed heights, global permission inference, new resizable-panel dependency.

#### Requirements Analysis

##### Goal

Make Node hierarchy and Resource discovery immediately distinguishable while preserving fast Resource preview/add workflows and all existing authority, accessibility, performance, and persistence boundaries.

##### Scope

- React Explorer composition and state.
- Explorer index/row derivation helpers.
- A dedicated accessible split-pane component or narrowly scoped Explorer splitter.
- UI preference schema validation and per-Profile persistence.
- Explorer/sidebar styling, light/dark/reduced-motion behavior, and tracked production `dist`.
- Focused frontend tests, Desktop build checks, real packaged Wails UI validation, and stable docs updates.

##### Use Cases

1. A user scans an arbitrary-depth Node-only tree, selects a Node, and sees its direct Resources below.
2. A user filters Nodes without Resource rows obscuring the tree, then filters the selected Node's Resources independently.
3. A user previews, drags, or adds a Resource from the lower pane and saves it to a View.
4. A user drags the separator to favor Node or Resource browsing; the ratio survives restart for that Profile.
5. A keyboard-only user adjusts the separator and navigates both panes without losing the tree's roving focus semantics.
6. When topology/catalog refresh removes the current Node or Resource, the Explorer selects a deterministic valid Node or shows a clear empty state without stale content.

##### Functional Requirements

1. Build one `ExplorerIndex` from topology and catalogs; do not duplicate ownership or hierarchy state.
2. The upper pane contains Node rows only. `aria-level`, `aria-posinset`, `aria-setsize`, and `aria-expanded` are computed only from Node relationships; Resources do not make a leaf Node expandable.
3. A current Node is always resolved deterministically from explicit Node selection, selected Resource owner, focused subtree root, or first visible root. It is distinct from transient Inspector visibility.
4. The lower pane contains only direct Resources owned by the current Node and preserves full Resource identity.
5. Resource presentation groups use the first `/` segment, have deterministic ordering, and never alter ownership, authorization, calls, drag payloads, or View references.
6. Node search and Resource search are independent. Node search retains matching ancestor paths; Resource search matches full name, label, and type within the current Node.
7. Selecting a Node updates the lower pane and opens the Node Inspector. Selecting a Resource opens its Inspector without losing current Node context.
8. Drag-to-Workspace and explicit `添加到工作区` remain available from every Resource row. Search/grouping must not change the descriptor passed to these actions.
9. The splitter supports pointer drag, Arrow Up/Down adjustment, Home/End bounds, and double-click reset. It exposes `role="separator"`, horizontal orientation, `aria-controls`, and value metadata.
10. Split ratio is clamped against both pane minimums, ignores non-finite input, and commits to `UIPreferences` only as a bounded non-secret numeric value.
11. Split ratio is isolated by Profile and restored on restart; corrupt preference data follows the existing explicit default-and-warning behavior.
12. Node and Resource panes scroll independently. The sidebar and all ScrollArea grid ancestors have `min-height: 0`/contained overflow so the Profile/connection footer stays fixed.
13. Empty topology, no Resources, filtered-empty results, disconnected refresh, and removed selection each have explicit copy and do not collapse the splitter unexpectedly.

##### Non-functional Requirements

- No new npm dependency; use React pointer/keyboard events and existing primitives.
- Preserve mineral-blue/graphite tokens, flat 1px pane boundaries, compact rows, full dark theme, visible focus, and reduced-motion behavior.
- Keep Node derivation iterative and data-driven for arbitrary depth; avoid recursive component state.
- Keep indexing and filtering O(nodes + resources) overall, with current-Node grouping O(resources owned by that Node).
- Keep more than 50 rows eligible for the existing `content-visibility` optimization; do not claim server lazy loading or true DOM windowing.
- Do not change Go protocol, Wails API, Hub policy, credential storage, or View file schema.

##### Inputs / Outputs

- Inputs: `Topology`, `ResourceDescriptor[]`, `WorkspaceSelection`, expanded/focused Node preferences, active Profile UI preferences, pointer/keyboard splitter events.
- Outputs: Node-only visible rows, current-Node grouped Resource rows, updated Inspector selection, unchanged Resource drag/add payloads, and a clamped `explorer_split_ratio` preference.

##### Edge Cases

- No topology roots or a focused Node that disappeared after refresh.
- A selected Resource whose owner disappears or whose catalog entry is removed.
- Same Resource local name on multiple Nodes.
- Resource name without `/`, multiple path segments, Unicode labels, and very long names.
- A pane container shorter than the sum of preferred minimum heights.
- Pointer release outside the splitter or loss of pointer capture.
- Explorer tab hidden while its measured height is zero.
- Ratio value is missing, `NaN`, infinite, or outside bounds.
- Switching Profiles where the same Node IDs exist but the saved ratios differ.
- Deep focused subtree and breadcrumb restoration after the information-architecture split.

##### Acceptance Criteria

- Node rows and Resource rows never appear in the same semantic tree.
- Selecting Node 1 in the real 2-Node/25-Resource environment shows its Resources in a separately labeled lower pane; selecting Node 2 shows its own empty/direct Resource set.
- Both panes display independent scrollbars when content exceeds their assigned height; scrolling either pane does not move the Profile/connection footer.
- Pointer drag changes pane heights within bounds; keyboard arrows also change the split; double-click restores the default.
- Split ratio reloads for the active Profile and remains isolated from another Profile.
- Node tree keyboard behavior, deep focus/breadcrumb, Resource preview, drag/add, View save, light/dark theme, and forbidden renderer states continue to work.
- Representative 2,000-Node/10,000-Resource data shaping remains within the existing 750 ms test budget.
- Vitest/Testing Library, TypeScript/Vite production build, Desktop Go tests, Wails Windows production build, and packaged GUI smoke pass.

##### Risks

- Incorrect grid shrink constraints can recreate the missing-footer/no-scroll regression.
- Reusing mixed-tree ARIA metadata after separation can produce incorrect levels or expandability.
- Writing localStorage on every pointer move can cause excessive I/O; ratio should be previewed locally and committed on pointer release.
- A transient Resource selection must not become a second source of truth for Node ownership.
- Tracked Vite `dist` filenames and Windows EOL can create generated drift if build output is copied instead of regenerated in the worktree.

#### Architecture Design

##### Overall Solution

Refactor `Explorer` into a shell with two independently searchable/scrollable panes sharing one memoized `ExplorerIndex`. Keep current Node context inside the Explorer and reconcile it whenever topology, focus, or external selection changes. Add a narrowly scoped split-pane component that owns the live drag ratio and emits only validated commits. App passes the persisted ratio and updates the existing Profile UI preference document.

##### Alternatives Considered

- Install a resizable-panels package: rejected; the requirement is one bounded vertical separator and existing project guidance avoids unnecessary dependencies.
- Store pixel height: rejected; ratios survive window resizing and varying content heights better.
- Persist split state in Go settings/View store: rejected; it is non-secret per-Profile UI preference and already has a canonical browser-local mechanism.
- Infer usable Resources before listing: rejected; catalog discovery is not authorization and probes may have side effects.

##### Module Responsibilities

- `frontend/src/store.ts`: Node-only flatten/search helpers and deterministic current-Node Resource grouping over the existing `ExplorerIndex`.
- `frontend/src/components/Explorer.tsx`: Node pane, current Node reconciliation, Resource pane, selection, search, DnD/add actions, breadcrumb, and pane empty states.
- `frontend/src/components/ExplorerSplitPane.tsx` (or equivalent narrow component): live pointer ratio, capture cleanup, keyboard/double-click behavior, ARIA separator contract, clamping, and commit boundary.
- `frontend/src/preferences.ts`: optional bounded `explorer_split_ratio` field, default value, strict validation, and Profile isolation.
- `frontend/src/App.tsx`: pass split preference and persist committed updates; reset transient Explorer context across Profile changes.
- `frontend/src/style.css`: pane grid, headers, separator hit target/focus state, independent ScrollAreas, grouping rows, compact dark/light styling, and the previously verified sidebar containment constraints.
- `frontend/src/**/*.test.ts(x)`: model, accessibility, persistence, integration, DnD/add, and regression coverage.
- `frontend/dist/**`: regenerated only through the canonical Vite production build.

##### Data / Call Flow

```text
Topology + catalogs
       │
       ▼
single ExplorerIndex
       ├── Node query + expanded/focused state ──► Node-only visible rows
       └── current Node + Resource query ───────► grouped direct Resources

Profile UI preference ──► clamped split ratio ──► live pointer/keyboard resize
                                                     │
                                                     └── commit ──► per-Profile localStorage

Node click ──► current Node + Node Inspector
Resource click ──► Resource Inspector
Resource drag/add ──► unchanged Workspace/View flow
```

##### Interface Drafts

- `UIPreferences.explorer_split_ratio?: number`
- `Explorer` additions:
  - `splitRatio?: number`
  - `onSplitRatioChange(ratio: number): void`
- Splitter contract:
  - controlled committed ratio with internal live preview
  - `defaultRatio`, top/bottom minimum pixels, and pane IDs
  - pointer and keyboard handlers that clamp against measured height
- Store helpers return Node rows and grouped Resource descriptors while retaining original `ResourceDescriptor` objects.

##### Error Handling and Safety

- Validate ratio at both preference-load and interaction boundaries.
- If a measured container has zero/invalid height, keep the last valid ratio and do not commit.
- Reconcile a missing current Node to focused/root fallback; never display Resources from a detached owner.
- Continue rendering Hub `forbidden`, offline, expired, and renderer failures inside Inspector/Widget; Explorer does not convert them into authorization claims.
- Pointer capture cleanup must run on release/cancel/unmount; no window-global listener may leak.
- Preserve full resource names and Node IDs in drag/add callbacks; grouping labels are never used as identifiers.

##### Performance and Testing Strategy

- Reuse one `ExplorerIndex` memoized by topology/catalog inputs.
- Flatten only Node rows for the tree; filter/group only the current Node's Resources for ordinary browsing.
- Update the 2,000/10,000 benchmark to exercise the new derivation path under the existing 750 ms budget.
- Unit-test pure ratio clamping and grouping helpers.
- Testing Library covers ARIA tree metadata, current Node switching, independent search, separator keyboard/pointer behavior, preference persistence, and add-to-Workspace.
- Build/test from the dedicated worktree after canonical dependency/binding preflight; regenerate tracked `dist`, do not copy it from the main checkout.
- Heavy validation uses the existing isolated local Hub and packaged Wails executable at 1,266×813 or equivalent, including long lists, both scrollbars, fixed footer, resize persistence, light/dark, and real Resource preview/add.

##### Extensibility Design Points

- Resource grouping stays a presentation helper and can later support descriptor-provided labels without changing identity.
- The splitter component may later serve other Desktop panes if its contract remains generic enough, but this task does not broaden it prematurely.
- Effective permission/status badges remain a future backend-informed capability and are not baked into the split layout.

#### Issue List

- None blocking.

### Stage 3.1 - Planning

#### Project Goal and Current State

Current production UI has a mixed Node/Resource tree and a verified but uncommitted main-checkout CSS containment fix. This workflow will implement the confirmed split-pane design on a clean dedicated branch, include the containment behavior in the branch result, and later merge without touching unrelated main-checkout changes.

#### Docs Governance Routing Decision

- Docs root: `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane\docs`
- Root `plan.md` / `todo.md`: active workflow control-plane exception; archive later under `docs/plan/`.
- Original request evidence: new focused intake document.
- Current user-visible behavior: clarify Desktop feature.
- Durable capability intent: clarify Desktop Resource Workspace requirements.
- Technical contract: clarify Desktop Resource Workspace v2.
- Architecture decision: no ADR; this is a bounded UX realization within the existing Node ownership/Explorer decision.
- Workflow result: create `docs/change` only during `$m-archive` after validation.
- Lessons: none planned; add/update only if implementation exposes a reusable splitter/ScrollArea/Windows build failure pattern.

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Related intake: `docs/intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md`
- Planned intake: `docs/intake/2026-08-29_desktop-explorer-split-pane.md`
- Related feature: `docs/features/desktop.md`
- Related requirement: `docs/requirements/desktop-resource-workspace.md`
- Related spec: `docs/specs/desktop-resource-workspace-v2.md`
- Related decision: `docs/decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md`
- Related lessons: `docs/lessons/frontend-and-powershell-preflight.md`, `docs/lessons/windows-clean-checkout-eol-and-generated-drift.md`, `docs/lessons/frontend-worktree-wailsjs-missing.md`

#### Stable Docs Impact

- Intake impact: add.
- Feature impact: clarify.
- Requirements impact: clarify.
- Specs impact: clarify.
- Decision impact: none.
- Lessons impact known at planning time: none.

#### Executable Task List

- EX01 — Separate Node tree and current-Node Resource list.
- EX02 — Add bounded accessible split resizing and per-Profile persistence.
- DOC01 — Update canonical intake/feature/requirement/spec documentation.
- QA01 — Run automated, performance, production build, and packaged GUI validation.
- ARC01 — Archive evidence, commit in Chinese, integrate locally, and clean the worktree after gates pass.

#### Execution Scope After Approval

##### Will Execute

- EX01, EX02, DOC01, QA01, ARC01.

##### Will Not Execute Now

- PERM01 — effective permission/status discovery; requires a separate backend-informed contract and must not be inferred by the frontend.
- LAZY01 — server-side lazy loading/pagination or true DOM windowing; deferred protocol work already documented in the canonical spec.
- BRAND01 — project icon work remains in the separate brand task.
- PUB01 — push, release, and publication are not authorized and no remote is configured.

#### Task Details

##### EX01 - Separate Node and Resource navigation

- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane`
- Plan Path: `plan.md`
- Goal: replace mixed tree rows with a Node-only tree and a grouped current-Node Resource list without changing authority or Workspace actions.
- Files / Modules: `frontend/src/store.ts`, `frontend/src/components/Explorer.tsx`, relevant tests.
- Write Set: Node row derivation/search, current Node reconciliation, Resource grouping/filtering, list semantics, selection, drag/add wiring.
- Acceptance: Node ARIA metadata excludes Resources; Node selection updates Resource pane; Resource preview/add/drag works; deep focus/breadcrumb and arbitrary depth remain intact.
- Test Points: deep Node fixture, matching ancestors, same Resource name on different Nodes, empty owner, grouping, resource search, keyboard navigation, App add/save integration.
- Rollback: restore the mixed `flattenExplorerRows` rendering path and its tests; no persisted backend data changes.

##### EX02 - Accessible persistent vertical splitter

- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane`
- Plan Path: `plan.md`
- Goal: give Node/Resource panes bounded adjustable heights and remember the committed ratio per Profile.
- Files / Modules: `frontend/src/components/ExplorerSplitPane.tsx` or equivalent, `preferences.ts`, `App.tsx`, `style.css`, tests, regenerated `dist/**`.
- Write Set: pointer/keyboard/double-click splitter, ARIA contract, ratio validation/persistence, pane grid/CSS, independent ScrollAreas, prior sidebar containment fix.
- Acceptance: pointer and keyboard resize stay within minimums; double-click resets; ratio persists per Profile; footer remains fixed; both panes independently scroll in light/dark themes.
- Test Points: clamp helper, zero-height container, pointer cancel/release, separator ARIA, keyboard steps/bounds, corrupted ratio, two Profile isolation, long-list scroll regression.
- Rollback: remove optional preference field and splitter component, restore fixed Explorer layout; old preference documents remain readable because the field is optional.

##### DOC01 - Clarify canonical Explorer behavior

- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane`
- Plan Path: `plan.md`
- Goal: make the confirmed Node/Resource separation and resizable preference the current documented truth.
- Files / Modules: new focused intake plus `docs/intake/README.md`, `docs/features/desktop.md`, `docs/requirements/desktop-resource-workspace.md`, `docs/specs/desktop-resource-workspace-v2.md`.
- Write Set: request evidence, user-visible behavior, durable requirements, technical/accessibility contract, index entry.
- Acceptance: docs consistently distinguish Node authority tree from Resource presentation list; splitter persistence and responsive bounds are documented; no permission, lazy-load, or icon claims are added.
- Test Points: docs diff review, index/link checks, stable-doc impact re-check.
- Rollback: revert the focused intake/index and scoped clarifications.

##### QA01 - Validate behavior and production artifact

- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane`
- Plan Path: `plan.md`
- Goal: prove the split Explorer is accessible, performant, buildable, and usable in the packaged Windows app.
- Files / Modules: test files, tracked `frontend/dist/**`, Wails build output, isolated verification screenshots.
- Write Set: focused tests and generated production frontend assets; no test Hub policy expansion.
- Acceptance: all focused/full frontend tests, production build, Desktop Go tests, Wails Windows build, and real GUI acceptance pass; Node/Resource panes scroll independently and footer remains pinned.
- Test Points: `npm test`, `npm run build`, `GOWORK=off go test ./apps/desktop/...`, `wails build -platform windows/amd64`, `git diff --check`, packaged GUI pointer/keyboard/restart/theme smoke.
- Rollback: delete only generated build binaries/evidence created for this workflow; retain source branch for diagnosis if any gate fails.

##### ARC01 - Archive, integrate, and clean up

- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-explorer-split-pane`
- Plan Path: `plan.md`
- Goal: record verified results, commit with Chinese messages, integrate locally without losing dirty-main user changes, and remove only this workflow worktree/branch.
- Files / Modules: `docs/change`, `docs/plan`, root plan/todo, Git branch/worktree metadata.
- Write Set: archive/index/evidence, implementation/docs commits, local fast-forward or safe convergence procedure.
- Acceptance: archive is indexed; test evidence linked; main-checkout `guide.md`, `论文/**`, `design-demos/**`, and unrelated modifications remain untouched; no push/publish; workflow worktree cleaned only after successful integration.
- Test Points: pre/post status comparison, branch ancestry, main dirty-path preservation, final diff/commit inspection.
- Rollback: keep branch/commit hashes until integration is verified; if dirty-main overlap blocks fast-forward, stop and report instead of stashing or discarding user changes.

#### Dependencies

- EX01 → EX02 → QA01 → ARC01.
- DOC01 begins after EX01 semantics are stable, is finalized after EX02, and must pass the archive impact re-check before ARC01.
- QA01 regenerates `dist` only after source/tests are stable.

#### Risks and Notes

- Main checkout has overlapping uncommitted `style.css`/`dist` changes from the prior verified scroll fix. The branch will reproduce the same semantic fix; archive must compare exact content before integration and must not use a broad stash/reset/checkout.
- A real Hub with 25 catalog entries is available for UI validation, but its default-deny policy remains unchanged; only already authorized resources are expected to render successfully.
- Existing Wails process may hold the canonical executable path. QA must identify and stop only the exact workflow-owned/known Desktop process before overwriting a build artifact.
- Browser-local UI preferences are non-secret but still strictly validated; this task does not migrate them into Go settings.

#### Parallelism Assessment

- No subagents. EX01 and EX02 overlap in `Explorer.tsx`, state semantics, CSS, and tests; sequential execution minimizes merge and accessibility-contract risk. DOC01 is also finalized by the main agent to keep stable docs synchronized with the implemented behavior.

#### Issue List

- None blocking.

#### Approval Gate

- Approved by explicit user invocation of `$m-execute`: EX01, EX02, DOC01, QA01, and ARC01.
- Blocked: no.
- Enter execution: EX01, EX02, DOC01, and lightweight QA01.
- ARC01 remains gated to the later `$m-archive` phase; do not create `docs/change`, merge, or clean the worktree from `$m-execute`.
- No implementation subagents: Explorer state, preferences, CSS, tests, and docs are tightly coupled, and current host policy does not authorize proactive delegation.

### Execute - Node/Resource Split Implementation

- EX01 complete: Explorer now derives one authority/ownership index, renders Node-only WAI-ARIA tree rows in the upper pane, and renders grouped direct Resources for the current Node in the lower pane. Node and Resource search, roving focus, selection, preview, drag, and add actions remain distinct.
- EX02 complete: added a bounded horizontal separator with pointer capture, cancel rollback, Arrow/Shift+Arrow, Home/End, double-click reset, pixel minimums, and per-Profile `explorer_split_ratio` persistence. Sidebar/ScrollArea containment keeps the Profile/Connection footer outside both scroll regions.
- DOC01 complete: added focused intake and updated the canonical Desktop feature, requirement, and spec. Decision impact remains none; lessons impact remains none.
- QA01 execute-stage evidence: `npm test` passed 27/27; `npm run build` passed TypeScript and Vite production build; `GOWORK=off go test ./apps/desktop/... ./sdk/bindings/desktop` passed; `git diff --check` passed; 2,000-Node/10,000-Resource benchmark remains under the 750 ms test budget.
- QA01 remaining: packaged Windows Wails build and real GUI pointer/keyboard/restart/light-dark/independent-scroll smoke belong to `$m-test`.
- Generated frontend `dist/**` was refreshed from the final source. No commit, merge, archive, worktree cleanup, push, release, or publication was performed in `$m-execute`.
- Stable docs impact result: intake added; feature, requirements, and specs clarified; decision none; lessons none.
- Issue list: none blocking.

### Test - Packaged Windows GUI Validation

- Entry route: direct `$m-archive` invocation was audited against its entry gate; because QA01 still lacked packaged Wails and actual UI evidence, the workflow returned to `$m-test` rather than treating missing validation as accepted risk.
- Scope: QA01 packaged Windows build, actual Explorer operation, independent scroll/footer containment, pointer and keyboard splitter behavior, Profile preference persistence, light/dark review, security/performance/code-review checklist.
- Parallelism: no subagents. GUI automation and screenshot evidence are one stateful host sequence, and current host policy does not authorize proactive delegation.
- Windows production build: `GOWORK=off wails build -clean -platform windows/amd64` passed and produced `apps/desktop/build/bin/myflowhub-desktop.exe`.
- Actual UI path: the worktree executable connected to the isolated default-deny Hub at `127.0.0.1:7331`, loaded Node-owned catalogs, switched Node 1/Node 2, previewed `system/catalog`, and added it to the unsaved test Workspace.
- Split behavior: pointer drag, keyboard `End`, double-click reset, sidebar Tab round-trip, independent Resource scrolling, and fixed Profile/Connection footer passed in the packaged application.
- Persistence/theme: a non-default split and dark theme survived a process restart; light/dark visual review passed. The temporary test instance was disconnected and stopped; the pre-existing main-checkout window was reconnected and its unsaved View was preserved.
- Evidence: `D:\project\MyFlowHub3\.tmp\m-test-desktop-explorer-split-pane\evidence\desktop-explorer-dark-persisted.jpg` and `desktop-explorer-light-scrolled.jpg`; archive copies are created only in `$m-archive`.
- Automated regression: `npm test` passed 27/27; `GOWORK=off go test ./... -count=1` passed all packages and integration tests; `git diff --check` passed; the 2,000-Node/10,000-Resource benchmark passed under the 750 ms budget.
- Review checklist: requirements, architecture, performance risk/threshold, usability, readability, extensibility, stability/security, permission/data boundaries, coverage, integration, and subagent governance all passed. No authorization, secret, schema, backend policy, or migration boundary changed.
- Result: blocking no; QA01 complete; proceed to `$m-archive`.

### Archive - Documentation And Safe Closeout

- `$m-docs` impact review complete: focused intake added; Desktop feature, requirement, and spec clarified; decision impact none; lessons impact none.
- Change record: `docs/change/2026-08-29_desktop-explorer-split-pane.md`.
- GUI evidence: `docs/change/verification/2026-08-29_desktop-explorer-split-pane-dark-persisted.jpg` and `2026-08-29_desktop-explorer-split-pane-light-scrolled.jpg`.
- Plan snapshot: `docs/plan/plan_archive_2026-08-29_desktop-explorer-split-pane.md`.
- Implementation commit: `ca4e54f` (`实现 Desktop 节点资源分区浏览`).
- Archive commit: created after the finalized plan snapshot; Chinese commit message, no push or publication.
- Integration audit: `master@f79f165` is an ancestor of the implementation, but the dirty main checkout overlaps the branch write set at `apps/desktop/frontend/src/style.css`, `dist/index.html`, and two tracked generated assets. Its replacement generated assets are also untracked.
- Safe merge evidence: `git merge --ff-only feat/desktop-explorer-split-pane` aborted before modification because it would overwrite `dist/index.html` and `src/style.css`; `master` stayed at `f79f165` and the complete porcelain status was identical before/after.
- Safety result: do not stash, reset, checkout, overwrite, or force-merge. Preserve `guide.md`, `论文/**`, `design-demos/**`, and all existing Desktop edits; retain `feat/desktop-explorer-split-pane` plus its worktree until the overlap is resolved or a separately authorized convergence is performed.
- ARC01 result: documentation/archive and Chinese commits complete; local merge and worktree/branch cleanup blocked by protected main-checkout overlap.
- Brand boundary: no icon asset or brand decision changed or archived.
