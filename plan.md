# Plan - Desktop Schema-driven Resource Widgets

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `feat/desktop-schema-driven-widgets`
- Base: `master` @ `54b46e6bc0c1bbea5b265a875e7c8132ae28e004`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\desktop-schema-driven-widgets\docs`
- Code Repos: canonical monorepo `MyFlowHub` only
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-schema-driven-widgets`
- Participating Modules: `protocol`、`sdk/bindings`、`apps/desktop`、`apps/desktop/frontend`、canonical `docs/`
- Current Stage: `$m-test` passed for `DOC01`–`QA01`; `$m-archive` documentation and closeout in progress
- Publication: local-only; no remote, push, release, or publication is authorized

## Stage Records

### Initialization

- `guide.md`: read; Chinese commit messages, canonical docs, `GOWORK=off`, and sibling-worktree rules apply.
- Project/docs/code repo confirmation: the canonical monorepo is both the owning code repo and governed docs root.
- Base/worktree confirmation: the dedicated semantic branch and worktree already exist under the project `worktrees\` directory.
- Main checkout protection: unrelated main-checkout changes such as `guide.md`, thesis files, design demos, and other task docs remain outside this worktree and must not be staged, overwritten, reverted, or archived by this workflow.
- Root control files: the previously completed Explorer control files are retained in `docs/plan`; root `plan.md` and `todo.md` now become this workflow's active control plane.

### Discuss - Discovery And Requirements Shaping

#### Goal

Replace raw-JSON-first Desktop widgets with schema-driven operational displays and controls while preserving generic Resource discovery, provider authority, permissions, revision safety, and View persistence.

#### Scope

- Provider-owned data type, constraints, semantics, and capability declarations.
- Desktop-owned compatible renderer set, default ranking, responsive presentation, and safe fallback.
- User-selectable renderer and non-secret display settings persisted per View widget.
- Rich first-party Variable, operation, event, File, and structured resource experiences.
- Generated first-party schema metadata for the current canonical protocol schemas without changing the wire descriptor in this phase.

#### Assumptions

- Invocation of `$m-plan` after the staged explanation confirms the phase-1 delivery boundary: built-in first-party schema provider now, owner-served schema discovery later.
- Current Go payload structs and `Validate` methods remain runtime enforcement authority.
- Existing `ViewWidget.renderer` and bounded `settings` fields are sufficient; View document version stays at v3.
- Presentation choices never grant permission or relax provider constraints.

#### Options Considered

1. Restyle JSON textareas: rejected because it does not add semantic controls.
2. Resource-name-specific screens: rejected because paths are not stable type contracts.
3. Full remote schema/plugin protocol now: deferred because trust, caching, compatibility, and migration require a separate protocol workflow.
4. Staged schema resolver with generated first-party metadata, generic primitives, and explicit specialized adapters: selected.

#### Recommended Direction

Implement a provider/display/user separation. A generated built-in schema provider adapts today's first-party schema IDs to a bounded JSON-Schema-compatible vocabulary. Desktop filters and ranks compatible renderers; the user may select any compatible renderer, and the selection is stored in the View without copying resource values.

#### Research Summary

- JSON Schema annotation and validation separation supports reusable data contracts.
- JSON Forms demonstrates separate data/UI schemas and a ranked renderer registry.
- Grafana demonstrates reusable unit/range/value mappings across display types.
- These patterns inform the architecture only; the project will not import a competing theme or full dashboard framework.

#### Worktree / Branch / Docs Root Status

- Dedicated worktree: ready.
- Discussion intake and index: drafted in this worktree.
- Runtime implementation: not started.
- Issue list: none blocking planning.

## Plan - Requirements And Architecture

### Discussion Summary

The provider defines what the data means and what values are valid. Desktop defines how valid data can be displayed. The user chooses among compatible displays. For example, a writable integer with `minimum=0`, `maximum=100`, and `multipleOf=1` may be edited through a slider, number stepper, or numeric input; a read-only rendering may be a number, gauge, progress bar, or bounded trend. A string may use a single-line, multiline, code, or read-only text renderer when its provider constraints allow it.

### Accepted / Rejected Requirements

Accepted:

- Controls are driven by schema ID, data shape, constraints, capabilities, and allowlisted presentation hints.
- Renderer changes are presentation-only and must never mutate a resource.
- Writable edits are staged; Apply performs the existing authoritative operation and preserves drafts on error or revision conflict.
- Unknown, unsupported, malformed, or incompatible schemas retain structured/raw access with an explicit explanation.
- All current first-party resource families gain useful non-JSON-first behavior in proportion to their contracts.
- Existing design tokens, light/dark themes, flat pane composition, nested split layout, and accessibility conventions remain.

Rejected or deferred:

- Resource-path matching, remotely supplied executable UI, automatic destructive inference, and controls that cannot enforce provider constraints.
- Full owner-served schema discovery, third-party executable renderer plugins, query-language dashboards, and production Media visualization in this phase.

### Requirements Analysis

#### Goal

Deliver a maintainable renderer platform that makes first-party resources directly usable and establishes the stable extension seam for future provider-served schemas.

#### Scope

- First-party schemas referenced by the current canonical binding contract and resource catalog.
- Generic scalar/object/array display and form generation.
- Variable, Command/generic operation, Stream/Topic, File, and selected structured first-party renderers.
- Renderer selection, persistence, responsive density, raw fallback, accessibility, and tests.

#### Use Cases

1. A user opens a numeric Variable and sees a meaningful value rather than JSON; writable bounded values offer safe controls.
2. The user switches a numeric widget between compatible presentations and sees the choice restored after saving/reopening the View.
3. A user invokes a Command through generated labeled fields and inspects structured output or raw JSON when necessary.
4. A user monitors Stream/Topic data as a table, timeline, or log and can pause, filter, clear, and recognize gaps.
5. A user selects a local file through the Desktop host, starts a transfer, and sees progress and actionable errors.
6. A first-party catalog, topology, health, configuration, flow, notification, audit, or transfer payload receives a structured display selected by stable schema ID.
7. An unknown third-party schema remains discoverable and usable through a safe raw fallback.

#### Functional Requirements

- Resolve input/output/event schemas through a typed provider interface.
- Support a bounded declarative schema vocabulary: `null`, `boolean`, `integer`, `number`, `string`, `object`, homogeneous `array`, properties, required, enum, numeric bounds/step, string length/pattern/format, item limits, labels/descriptions, units, precision, read/write sensitivity, and stable field order.
- Reject or fall back for unsupported keywords, recursive definitions, invalid references, excessive depth/fields, or invalid schema documents.
- Register renderers with stable versioned IDs, modes, compatibility predicates, ranking, minimum size, and allowed settings.
- Treat legacy `mfh.variable`, `mfh.stream`, `mfh.topic`, `mfh.command`, and `mfh.file` renderer IDs as automatic compatibility aliases.
- Offer only renderers compatible with both schema and resource capability.
- Keep a keyboard-accessible presentation selector in the widget chrome when more than one compatible renderer exists.
- Preserve resource payloads, drafts, credentials, and secrets outside `ViewWidget.settings`.
- Keep Advanced JSON/descriptor access available for all resources.

#### Non-functional Requirements

- No unnecessary visual-system or docking dependency; prefer project primitives and small internal modules.
- Schema resolution and renderer selection are deterministic, pure, and covered by unit tests.
- Schema/render trees have explicit depth, property, array, event-buffer, and payload bounds.
- Resize observation is batched; pane-size changes update responsive density without resaving the View or causing per-pixel request traffic.
- High-rate events use bounded buffers and batched UI updates.
- WAI-ARIA names, keyboard equivalents, focus behavior, non-color status, and error associations are required.
- Runtime owner validation, authority, permission, size, path, session, and revision enforcement remain final.

#### Inputs / Outputs

Inputs:

- `ResourceDescriptor`, capabilities, schema IDs, content types, presentation hints, current values/events/results, View widget renderer/settings, pane dimensions, and user actions.

Outputs:

- resolved bounded data schema, compatible renderer list, selected renderer, staged draft/validation state, structured display/control tree, operation payload, and display-only View settings.

No resource value, operation payload, event body, secret, file content, or permission result is persisted as renderer settings.

#### Edge Cases

- Missing or unknown schema ID/content type.
- Known schema whose generated definition is stale or invalid.
- Saved renderer removed or made incompatible after schema change.
- Read-only Variable viewed through a previously writable renderer choice.
- Nullable values, empty arrays/objects, very large arrays, deeply nested objects, unsupported unions, and opaque JSON fields.
- Revision conflict, Forbidden, Expired, Gap, disconnect/reconnect, resource disappearance, and schema mismatch while editing.
- Pane too small for a renderer, light/dark theme changes, reduced motion, keyboard-only operation, and multiple simultaneous widgets.
- File picker cancellation and upload/session failure.

#### Acceptance Criteria

- A bounded integer fixture (`0..100`, step `1`) offers compatible numeric displays/controls; keyboard and pointer edits honor bounds and step.
- A string fixture offers compatible one-line/multiline/code/text choices as allowed by its constraints; switching does not call the resource API.
- Read-only data never exposes mutating controls; writable drafts use Reset/Apply and remain intact after failed writes or revision conflicts.
- Saving and reopening a View restores explicit renderer choice and valid display settings; legacy renderer IDs still open safely.
- Commands render a generated form for supported fields, validate locally, submit only on explicit Execute, and show typed output; Advanced JSON remains available.
- Streams/Topics expose live state, pause/resume, clear, filter, count/rate, autoscroll, and visible gap/expired state using bounded storage.
- File upload uses a native picker boundary and shows destination/progress/cancel/error state where the current session contract supports it.
- Catalog/topology/health/config/flow/audit/notification/file payloads receive table, definition-list, status, timeline, or progress adapters keyed by schema ID, never by resource path.
- Unknown or unsupported schemas show an actionable fallback rather than a blank or misleading control.
- Light/dark, compact/normal/expanded panes, nested splits, offline/Forbidden states, and existing Explorer/View behavior do not regress.

#### Risks

- Handwritten Go validators and declarative schemas can drift; generation coverage and representative parity fixtures must fail closed.
- A complete JSON Schema engine would enlarge scope; this phase supports a documented bounded vocabulary and visibly falls back outside it.
- Generic nested forms can become unwieldy; opaque/conditional fields keep an Advanced JSON path rather than pretending full fidelity.
- Renderer settings can accidentally retain data; settings are allowlisted display metadata only and tested for serialization boundaries.
- Wails native dialog testing needs an injectable boundary because the real dialog cannot run in jsdom.

### Architecture Design

#### Overall Solution

```text
provider schema ID + capability + value/event/result
                         |
                         v
SchemaResolver -> built-in generated provider (phase 1)
                         |
                         v
bounded ResolvedDataSchema + validation result
                         |
                         v
RendererRegistry compatibility filter + rank
                         |
              +----------+-----------+
              |                      |
              v                      v
automatic safe default       user-selected compatible renderer
              |                      |
              +----------+-----------+
                         v
resource controller + display/control primitive
                         |
                         v
authoritative Desktop API operation/subscription/session
```

The resource controller owns transport lifecycle; value renderers never call Wails directly. This prevents every slider/table from duplicating subscription, revision, cancellation, and error behavior.

#### Canonical Schema Source And Generation

- Add a protocol-owned, bounded declarative schema model and first-party definitions keyed by existing schema constants.
- Existing Go payload structs and `Validate` methods remain enforcement authority; declarative definitions describe UI-visible shape and constraints.
- Extend the existing `go generate ./sdk/bindings` path to emit a deterministic Desktop schema artifact from the protocol definitions.
- Add freshness tests, schema-ID coverage against the canonical binding manifest, duplicate/sort/limit validation, and representative valid/invalid fixture parity.
- Add LF rules and generated-check tracking for the new artifact.
- Do not add schema documents to `SchemaDescriptorV2` or alter catalog/wire encoding in phase 1.

#### Module Responsibilities

| Module | Responsibility |
| --- | --- |
| `protocol` | First-party declarative schema definitions, IDs, limits, and validation of definitions |
| `sdk/bindings` generator | Deterministic cross-language schema artifact and freshness/coverage gates |
| `frontend/src/rendering/schema` | Typed resolver/provider interface, bounded runtime validation, defaults, and safe fallback reasons |
| `frontend/src/rendering/registry` | Versioned renderer definitions, compatibility, ranking, aliases, and settings validation |
| resource controllers | Snapshot/subscription/operation/session lifecycle, revisions, staged drafts, retry/cancel/error state |
| display/control primitives | Pure schema-aware value display and editing controls |
| specialized adapters | Schema-ID-selected tables/status/timelines/forms that compose generic primitives |
| Workspace/View integration | Renderer selector, pane density, settings persistence, dirty state, and detached/incompatible handling |
| Wails host | Native file picker and existing validated Desktop API boundary |

#### Data / Call Flow

1. Workspace resolves the current resource and View widget.
2. `SchemaResolver` resolves each capability's relevant schema ID through provider order.
3. The bounded validator accepts the schema or returns a reasoned unsupported/invalid result.
4. `RendererRegistry` filters by resource mode, capability, schema features, pane density, and minimum size.
5. A saved compatible renderer wins; an automatic/legacy alias uses deterministic ranking; an incompatible saved choice visibly falls back without erasing the stored preference.
6. The resource controller fetches or subscribes once and passes typed state to the selected presentation component.
7. Editors produce staged drafts and local errors; Apply/Execute converts the draft to the existing API payload.
8. Runtime errors are displayed in-widget. A successful write refreshes authoritative value/revision; a failed write preserves the draft.
9. Presentation choice/settings update the View and mark it dirty but never call a resource operation.

#### Interface Drafts

```ts
type SchemaResolution =
  | { status: 'resolved'; schema: ResolvedDataSchema; source: string }
  | { status: 'missing' | 'unsupported' | 'invalid'; schemaID?: string; reason: string }

interface SchemaProvider {
  id: string
  resolve(schemaID: string): SchemaResolution | undefined
}

interface RendererDefinition {
  id: string
  mode: 'value' | 'operation' | 'event' | 'file' | 'structured'
  supports(context: RendererContext): Compatibility
  rank(context: RendererContext): number
  validateSettings(value: unknown): RendererSettings
}
```

Renderer IDs are versioned and resource-mode specific, for example `mfh.variable.slider.v1`, `mfh.variable.number.v1`, `mfh.value.table.v1`, `mfh.event.timeline.v1`, and `mfh.operation.form.v1`. Exact names are finalized in the spec before code use.

#### Initial Schema / Renderer Matrix

| Shape / mode | Compatible presentations |
| --- | --- |
| Boolean read/write | text/status; switch or checkbox for staged writable drafts |
| Enum read/write | badge/text; select or segmented choice within bounded option count |
| Bounded numeric | number/stat/progress/gauge/trend; numeric input/stepper/slider when writable |
| Unbounded numeric | number/stat/trend; numeric input/stepper when writable |
| String | text/code; one-line, multiline, or code textarea when constraints/format allow |
| Date/time/duration | localized display; matching bounded input when writable |
| Object | definition list, grouped display, generated form, or JSON tree/raw fallback |
| Homogeneous array | list/table, repeatable form rows within limits, or JSON fallback |
| Command/operation | generated input form + explicit Execute + structured output + Advanced JSON |
| Stream/Topic | log/table/timeline with bounded buffer, filter, pause, clear, rate, autoscroll, gap |
| File | native source selection, destination, progress/status, cancel/error |
| Unknown/unsupported | descriptor + JSON tree/raw viewer; no invented editing control |

#### First-party Structured Coverage

- `mfh.catalog.v2`: searchable resource table and details.
- management topology/health/config: hierarchy/table, status checks, and grouped key/value display.
- management audit and notification events: timeline/log with stable metadata columns.
- file transfers/progress: transfer table and progress/status presentation.
- flow definitions/runs/events: definition/run tables and event timeline; deeply conditional flow-editing fields may retain Advanced JSON where the bounded dialect cannot express them faithfully.
- management and flow Commands: generated forms where supported, with schema-driven raw fallback for opaque fields.

#### Error Handling And Safety

- Missing/invalid schema, unsupported vocabulary, incompatible saved renderer, and malformed settings have distinct messages and safe fallbacks.
- Unknown schema never receives an inferred mutating control.
- Local validation is advisory UX; owner/runtime errors remain authoritative and are not swallowed.
- Revision conflicts preserve the user's draft and offer authoritative reload/reset.
- `readOnly`, `writeOnly`, and sensitive annotations suppress unsafe echo/persistence.
- Renderer settings are bounded, versioned, allowlisted, and contain presentation only.
- Remote HTML, JS, arbitrary CSS, component names outside the registry, and side-effecting URLs are never evaluated.
- Subscription and session effects are cleaned up on unmount, resource change, profile switch, and connection loss.

#### Performance And Testing Strategy

- Pure unit tests cover definition validation, resolver precedence, compatibility/ranking, schema validation, aliases, settings, and fallback.
- Component tests cover keyboard/pointer controls, staged edits, switch-without-mutation, errors/conflicts, operation forms, event buffering, and responsive variants.
- Go tests cover schema-definition validity, generator freshness/coverage, View settings bounds, native dialog boundary, and app regressions.
- Synthetic deep/wide object and high-rate event fixtures verify depth/field/buffer limits and batching.
- Production validation uses `GOWORK=off`, Vitest, TypeScript/Vite, generated-contract check, full Go tests, Wails production build, browser interaction, and real packaged GUI smoke.

#### Extensibility Design Points

- `SchemaResolver` accepts ordered providers; a future owner-served provider can be inserted without replacing renderers.
- Renderer registry remains local and allowlisted; future third-party data schemas do not imply third-party executable UI.
- Schema/version mismatch falls back without losing View topology or resource reference.
- Specialized adapters compose the same generic primitives and are selected by schema or explicit renderer ID, not path.

## Stage 3.1 - Planning

### Project Goal And Current State

The current renderer layer is a single `Renderer.tsx` with JSON textareas/pre blocks and broad type dispatch. View v3 already stores renderer/settings and supports arbitrary nested panes. The plan retains that layout/store contract while creating the missing schema and renderer domains.

### Docs Governance Routing Decision

Using `$m-docs`:

- Docs root: `D:\project\MyFlowHub3\worktrees\desktop-schema-driven-widgets\docs`
- Intake impact: clarify; discussion brief and intake index already updated.
- Feature impact: clarify `docs/features/desktop.md` with current schema-driven widget behavior.
- Requirements impact: clarify `docs/requirements/desktop-resource-workspace.md` with provider/display/user ownership and renderer-switch acceptance.
- Specs impact: add `docs/specs/desktop-schema-rendering.md`; link it from the spec index and Desktop workspace v3.
- Decision impact: add an ADR for provider-owned data schemas and Desktop-owned renderer selection; update the decision index.
- Lessons known at planning time: reference generated-contract drift and observable-side-effect lessons; no new lesson is justified before implementation evidence.
- Archive/change impact: `$m-archive` will later retain the approved plan, test evidence, stable-doc impact, and change record.
- Root docs index impact: none; category topology and reading order do not change.

### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake: `docs/intake/2026-08-30_desktop-schema-driven-resource-widgets.md`
- Feature: `docs/features/desktop.md`
- Requirements: `docs/requirements/desktop-resource-workspace.md`, `docs/requirements/extensible-resource-platform.md`
- Specs: `docs/specs/desktop-resource-workspace-v3.md`, `docs/specs/resource-platform-v2.md`, `docs/specs/build-and-ci.md`
- Decisions: `docs/decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md`, `docs/decisions/2026-08-30_desktop-n-ary-docking-layout.md`
- Lessons: `docs/lessons/observable-side-effects-and-generated-contracts.md`, `docs/lessons/wails-binding-proto-drift.md`, `docs/lessons/frontend-and-powershell-preflight.md`

### Stable Docs Impact

- Intake impact: clarify
- Feature impact: clarify
- Requirements impact: clarify
- Specs impact: add
- Decision impact: add
- Lessons impact: none planned; reassess after test/debug evidence

### Executable Task List

| Task ID | Title | Scope | Primary files / modules | Acceptance cue |
| --- | --- | --- | --- | --- |
| DOC01 | Stabilize schema-rendering contracts | Will execute | governed docs | Stable ownership, behavior, spec, ADR, and indexes agree |
| SCHEMA01 | Add provider-owned built-in schema metadata and generation | Will execute | `protocol`, `sdk/bindings`, generated artifact | Deterministic coverage/freshness/parity gates pass |
| RENDER01 | Create schema resolver and renderer registry domains | Will execute | frontend `rendering/*`, types/settings | Pure compatibility/ranking/fallback tests pass |
| VALUE01 | Implement generic value controls and Variable UX | Will execute | value primitives, Variable controller/renderer | Numeric/string/bool/object/array choices and staged writes work |
| OP01 | Implement structured operation and result UX | Will execute | operation form/result renderers | Supported forms validate and submit explicitly; raw fallback remains |
| LIVE01 | Implement event, File, and first-party structured views | Will execute | event/file/specialized renderers, Wails file boundary | Bounded live UX, picker/progress, schema-ID adapters work |
| VIEW01 | Integrate selection, persistence, responsiveness, and accessibility | Will execute | Workspace/App/View/settings/styles | Choices persist; aliases/fallback/density/keyboard behavior pass |
| QA01 | Run proportional full validation and GUI evidence | Will execute | tests/build/generated/Wails | Unit, full Go, generated, frontend, production, GUI gates pass |
| REMOTE01 | Owner-served schema discovery protocol | Will not execute now | future protocol/catalog/cache work | Deferred: separate trust/version/cache migration decision |
| PLUGIN01 | Executable third-party renderer plugins | Will not execute now | future plugin platform | Out of scope and unsafe without sandbox/trust model |
| DASH01 | Full dashboard query/chart engine | Will not execute now | future analytics layer | Deferred: no query/history contract and unnecessary dependency scope |
| ARC01 | Archive, merge, and cleanup | Will not execute now | `docs/plan`, `docs/change`, Git/worktree | Owned by later `$m-archive` after test gate |
| PUB01 | Push, release, or publication | Will not execute now | remote/release infrastructure | Not authorized; repository may have no remote |

### Execution Scope After Approval

#### Will Execute

- `DOC01`, `SCHEMA01`, `RENDER01`, `VALUE01`, `OP01`, `LIVE01`, `VIEW01`, `QA01`

#### Will Not Execute Now

- `REMOTE01`: deferred to a protocol workflow because owner-served definitions need wire discovery, trust, cache, compatibility, and invalidation rules.
- `PLUGIN01`: remotely executable UI remains out of scope and is not implied by declarative schemas.
- `DASH01`: no full Grafana-like query/history platform in this widget usability phase.
- `ARC01`: only after implementation and tests pass, through an explicit `$m-archive` invocation.
- `PUB01`: no remote/push/release authorization.

### Task Details

#### DOC01 - Stabilize Schema-rendering Contracts

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: make the provider/display/user ownership model and phase boundary canonical before runtime changes.
- Files / Modules: `docs/intake`, `docs/features/desktop.md`, `docs/requirements/desktop-resource-workspace.md`, new `docs/specs/desktop-schema-rendering.md`, related category indexes, new decision record.
- Write Set: governed documentation only.
- Acceptance: no competing truth; phase-1 wire non-change and future remote provider seam are explicit; all new leaves are indexed and cross-linked.
- Test Points: Markdown link/path inspection and `git diff --check`.
- Rollback: revert DOC01 files without affecting runtime.

#### SCHEMA01 - Add Provider-owned Built-in Schema Metadata And Generation

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: provide deterministic first-party data schemas keyed by existing protocol schema constants.
- Files / Modules: `protocol/data_schema*.go`, protocol tests, `sdk/bindings/contract`, `sdk/bindings/cmd/mfh-bindgen`, generated Desktop schema artifact, `.gitattributes`, `scripts/mfh.ps1` generated tracking.
- Write Set: declarative definitions, generator, generated artifact, and focused tests; no `SchemaDescriptorV2` wire change.
- Acceptance: all schemas used by the canonical first-party binding manifest resolve or are explicitly classified as opaque/session-only; definitions are sorted, bounded, unique, and deterministic.
- Test Points: protocol definition tests, representative valid/invalid fixtures, generator freshness, `check/generated`.
- Rollback: remove the new definitions/output/generator extension; existing catalog and runtime remain unchanged.

#### RENDER01 - Create Schema Resolver And Renderer Registry Domains

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: separate schema resolution, validation, compatibility/ranking, and renderer settings from React transport components.
- Files / Modules: `apps/desktop/frontend/src/rendering/schema/*`, `rendering/registry/*`, generated-schema loader, `types.ts`, focused unit tests.
- Write Set: pure domain modules and tests.
- Acceptance: deterministic provider lookup; bounded schema validation; stable renderer IDs/aliases; safe missing/invalid/incompatible results; settings reject unknown or unsafe state.
- Test Points: resolver precedence, schema limits, compatibility matrix, default ranking, alias migration, settings round-trip/fallback.
- Rollback: remove rendering domain and retain old `Renderer.tsx` dispatch until integration tasks land.

#### VALUE01 - Implement Generic Value Controls And Variable UX

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: make scalar and common structured Variables useful while preserving revision and permission safety.
- Files / Modules: resource controllers, value display/control primitives, Variable renderers, UI primitives/styles, component tests.
- Write Set: React components/styles/tests; no server permission change.
- Acceptance: bool/enum/number/string/date/object/array matrix works; bounded slider/stepper honors provider constraints; read-only is distinct; Reset/Apply and conflict draft preservation work; raw mode remains.
- Test Points: pointer/keyboard bounds/step, invalid draft, read-only, successful write, Forbidden/conflict, unmount cleanup, renderer switch without API mutation.
- Rollback: registry can route Variables to the legacy JSON renderer alias.

#### OP01 - Implement Structured Operation And Result UX

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: replace JSON-only Command/Topic publish/generic operation entry with generated forms and structured output.
- Files / Modules: operation controller, form generator, result display, Advanced JSON mode, tests.
- Write Set: frontend only.
- Acceptance: labels/descriptions/required/enums/bounds/formats render; local errors associate with fields; Execute is explicit; server errors remain visible; opaque/unsupported fields use raw mode.
- Test Points: form payload conversion, no submit-on-change, output schema rendering, error handling, size/unsupported fallback.
- Rollback: route operations to the Advanced JSON renderer without changing API contracts.

#### LIVE01 - Implement Event, File, And First-party Structured Views

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: provide bounded operational monitoring and useful schema-ID-specific views across existing first-party resources.
- Files / Modules: event controller/renderers, specialized adapters, `apps/desktop/app.go`, frontend API facade/types, File renderer, tests/styles.
- Write Set: frontend plus a narrow Wails native file-picker method and its generated binding output if public API changes.
- Acceptance: event pause/resume/filter/clear/rate/autoscroll/gap states; native file selection/cancel; transfer progress/errors; catalog/topology/health/config/flow/audit/notification/file adapters selected only by schema/renderer ID.
- Test Points: bounded event buffer/batching, gap/expired, subscription cleanup, picker cancellation/mock, schema-ID dispatch, detached/Forbidden/session failure.
- Rollback: keep generic event/raw and typed-path File fallback while removing optional specialized registrations; no protocol rollback required.

#### VIEW01 - Integrate Selection, Persistence, Responsiveness, And Accessibility

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: make renderer choice a first-class View behavior without changing layout version.
- Files / Modules: `Workspace.tsx`, `App.tsx`, `store.ts`, `types.ts`, `views.go`, renderer settings integration, CSS, Workspace/App/Go tests.
- Write Set: View/widget integration and validation; View document remains v3.
- Acceptance: accessible selector appears only for multiple compatible choices; selection marks View dirty, persists, restores, and does not mutate data; legacy IDs auto-resolve; incompatible saved choice visibly falls back; compact/normal/expanded modes behave inside nested panes.
- Test Points: save/reopen, legacy fixture, malformed settings, no resource API on switch, ResizeObserver batching, focus/keyboard labels, light/dark and narrow pane.
- Rollback: retain stored values but map them through legacy aliases/raw fallback; no layout migration rollback.

#### QA01 - Run Proportional Full Validation And GUI Evidence

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: prove renderer correctness, generated-contract freshness, regression safety, and packaged Desktop usability.
- Files / Modules: focused tests, generated output, production `dist`; temporary visual evidence later routed by `$m-archive`.
- Write Set: tests and deterministic build/generated outputs only.
- Acceptance: all planned checks pass; no unreviewed generated diff; representative light/dark and multi-pane GUI scenarios are visually verified.
- Test Points:
  - `GOWORK=off go test ./protocol ./sdk/bindings ./apps/desktop/... -count=1`
  - `GOWORK=off go test ./... -count=1`
  - `npm ci`, `npm test`, `npm run build` in Desktop frontend
  - `./scripts/mfh.ps1 -Action check -Target generated`
  - `wails build -clean -trimpath -platform windows/amd64 -o mfh-desktop.exe`
  - browser interaction for forms/renderer switching and real packaged Wails GUI smoke
- Rollback: stop before merge/archive, revert failing task group to its prior passing checkpoint, and preserve evidence/error signature.

### Dependencies

```text
DOC01 -> SCHEMA01 -> RENDER01 -> VALUE01 --+
                              -> OP01 -----+-> VIEW01 -> QA01
                              -> LIVE01 ---+
```

- `DOC01` fixes terminology and contracts used by all code tasks.
- `SCHEMA01` provides the generated inputs for `RENDER01`.
- `VALUE01`, `OP01`, and `LIVE01` may proceed independently after the registry contract is stable.
- `VIEW01` integrates all renderer families before the full validation gate.

### Risks And Notes

- The largest risk is semantic drift between Go validation and declarative schemas; freshness, coverage, and parity tests are mandatory.
- The second risk is scope explosion in nested/conditional forms; unsupported constructs must fall back rather than grow an incomplete schema engine.
- The third risk is lifecycle duplication; transport stays in resource controllers, not presentation primitives.
- No dependency should be added unless a focused comparison proves it smaller and safer than the bounded internal implementation. A full component/theme/dashboard package is disallowed.
- Brand/icon work is not touched by this workflow.
- Main-checkout unrelated changes remain protected.

### Parallelism Assessment

- Planning and initialization are performed by the primary agent only.
- No implementation sub-agent is dispatched before approval.
- After approval, `VALUE01`, `OP01`, and `LIVE01` are conceptually parallel after `RENDER01`, but their shared registry/styles make sequential primary-agent execution the safer default unless the execution skill explicitly establishes non-overlapping write sets.

### Issue List

- No blocking prerequisite remains.
- The user invoked `$m-execute`, approving `DOC01`–`QA01`; deferred tasks remain outside the write set.

## Execute - Implementation Result

- Completed: `DOC01`, `SCHEMA01`, `RENDER01`, `VALUE01`, `OP01`, `LIVE01`, `VIEW01`.
- Completed before heavy validation: focused Go tests, 81 frontend tests, TypeScript/Vite production frontend build, deterministic regeneration, Wails binding generation, and `git diff --check`.
- Mainline integration: merged `master` @ `67bf3c3` before closeout and extended the canonical generated schema set for its centralized admission/enrollment payloads; provider-owned constraints and sensitive input annotations remain enforced by the same generation/freshness gates.
- File boundary: native picker and blocking upload state are implemented. The current upload binding exposes no transfer handle, so per-transfer cancel and byte progress are not fabricated; owner-reported `file/progress` and `file/transfers` remain available as independent Widgets.
- Security boundary: remotely executable renderers remain excluded; phase-1 renderer settings allowlist is empty; sensitive write-only fields render as password inputs and are never displayed as output.
- Parallelism result: no implementation sub-agent was used because schema, registry, renderer, Workspace, styles, and generated artifacts formed overlapping write sets and the user invoked `$m-execute`, not delegated `$m-go`.

## Test - Heavy Validation Result

- `GOWORK=off go test ./... -count=1`: all repository packages and integration tests passed.
- `GOWORK=off .\scripts\mfh.ps1 -Action check -Target generated`: deterministic protocol/Desktop bindings remained fresh and the worktree stayed clean.
- `npm test`: 13 files / 81 tests passed, including numeric bounds/step, renderer registry/fallback, command form, centralized admission UI, 10k Resource-tree budget, View persistence, nested layout, and WAI-ARIA navigation.
- `npm run build`: TypeScript and Vite production build passed.
- `wails build -clean -trimpath -platform windows/amd64 -o mfh-desktop.exe`: Windows production executable built successfully.
- Real packaged Wails acceptance used an isolated default-deny Hub at `127.0.0.1:7443` and isolated Desktop config. The Profile completed one-use admission, loaded 2 Nodes / 24 root resources, and showed persistent `已连接` state.
- After merging current `master`, a second packaged smoke used a fresh centralized Admission Authority state: the prepared device completed Enrollment Permit registration, received its Authority-assigned Node ID, restored the saved two-pane View, showed persistent `已连接`, and surfaced the intentionally ungranted topology read as explicit `Forbidden`.
- Actual UI operations passed: `system/health` specialized display, switch to Raw JSON and back, `flow/create` schema-generated numeric/text/object/array controls, dynamic array rows, explicit invoke, actionable provider validation error, horizontal ratio drag, View save, restart restore, and light/dark themes.
- Persisted View v3 retained renderer IDs and the dragged split weights `0.6484375 / 0.3515625`; operation draft and error state were not persisted.
- Security inspection found no Permit, signature, or private key in settings/View JSON. The DPAPI identity remained protected, renderer settings stayed empty, and remotely executable UI remained excluded.
- Evidence: `docs/change/verification/2026-08-30_desktop-schema-widgets-light.png` and `docs/change/verification/2026-08-30_desktop-schema-widgets-dark.png`.
- Review result: passed. No severity-threshold issue or regression remains; `QA01` and rollback checkpoint `R5` are complete.

## Archive - Documentation And Closeout

- `$m-docs` impact review: intake/feature/requirements/spec/decision are already canonical and indexed; no new reusable lesson is justified beyond existing generated-contract, Wails-binding, and frontend preflight lessons.
- Change record: `docs/change/2026-08-30_desktop-schema-driven-resource-widgets.md`.
- Plan snapshot: `docs/plan/plan_archive_2026-08-30_desktop-schema-driven-resource-widgets.md`.
- Test evidence: two governed screenshots under `docs/change/verification/`.
- Closeout policy: commit archive records, safely fast-forward local `master` while preserving unrelated main-checkout changes, remove this worktree/branch, and do not push or publish.
- Brand boundary: no icon asset, brand decision, or platform icon was changed by this workflow.

## Approval Gate

- Plan status: confirmed.
- Approval status: `DOC01`–`QA01` approved by explicit `$m-execute` invocation.
- Blocked: no.
- `QA01` and rollback checkpoint `R5` passed; `$m-archive` is authorized and in progress.
- Implementation sub-agents: not dispatched because the shared schema/registry/styles create overlapping write sets and `$m-go` delegation was not requested.
