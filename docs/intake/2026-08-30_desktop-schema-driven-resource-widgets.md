# 2026-08-30 Desktop Schema-driven Resource Widgets

## Source

- Source: user feedback with a running Desktop screenshot.
- Date: 2026-08-30.
- Workflow: `$m-discuss` on branch `feat/desktop-schema-driven-widgets`.

## Request Text / Source-preserving Summary

The current Workspace widgets are visually and operationally too primitive. Variable and operation resources mainly expose raw JSON, while a usable Desktop should present meaningful values, forms, controls, status, tables, trends, timelines, and file actions according to the resource contract.

## Observed State

- `ResourceRenderer` selects one renderer primarily from `presentation.renderer` or the broad resource type.
- writable Variable values use a JSON textarea and conditional write action;
- Command and Topic publish operations use a JSON textarea;
- Stream and Topic render a bounded raw event list;
- File upload uses two plain path inputs;
- unknown resources preserve discovery through descriptor/raw JSON fallback.

The limiting contract is not just visual styling. `CapabilityDescriptorV2` exposes schema IDs such as `mfh.flow.run.v1`, while `SchemaDescriptorV2` currently contains only `id` and `content_type`. The catalog does not provide property types, labels, required fields, enum values, numeric bounds, units, formats, defaults, or UI layout. The Desktop therefore cannot safely infer a switch, slider, selector, or structured form from the current remote descriptor alone.

## Problem / Opportunity

The Desktop already has the right resource and capability boundaries, but its Renderer layer stops at transport-level JSON. The opportunity is to make it an operational Resource workbench without turning it into a collection of resource-name-specific screens or allowing remote nodes to inject executable UI code.

## Confirmed Goals

1. Render current values as meaningful operational information rather than raw JSON by default.
2. Show editing controls only when the corresponding capability exists; read-only Variables must not look editable.
3. Choose controls from declared data shape and presentation metadata, never from resource path names.
4. Keep raw JSON as an explicit advanced/diagnostic fallback, not the primary experience.
5. Reuse the existing mineral-blue/graphite visual language, flat panes, restrained rounding, light/dark themes, and current component primitives.
6. Make widgets responsive to pane size and usable in the current nested split Workspace.
7. Preserve server-side permission, schema validation, expected-revision checks, size limits, and error authority.
8. Keep unknown and third-party resource types discoverable and safe.
9. Keep the provider-owned data contract separate from display-owned renderer choice: the provider defines the value type and valid domain, while Desktop chooses a compatible default renderer and lets the user switch among compatible renderers.

## Non-goals

- Do not infer units, ranges, destructive intent, or permissions from resource names.
- Do not download or execute renderer JavaScript from remote nodes.
- Do not replace runtime validation or authorization with client-side form validation.
- Do not turn every scalar into a chart or recreate a full Grafana query/dashboard editor.
- Do not introduce a second fixed Variable/Stream/Command protocol stack.
- Do not remove the raw payload/descriptor inspector.

## Recommended Product Behavior

### Provider contract, display policy, and user choice

The resource provider is authoritative for semantics and validity. For example, it may declare `integer`, `minimum: 0`, `maximum: 100`, and `multipleOf: 1`; it does not require Desktop to render a slider. String providers may declare constraints and semantic formats such as maximum length, pattern, date-time, JSON, or secret/write-only, but do not force a one-line or multiline control.

Desktop owns the component registry and presentation policy. It filters renderers by compatibility with the provider contract, ranks a safe default for the available pane size and capability, and exposes renderer selection in the widget's presentation menu. The selected renderer and display-only settings belong to the saved View, not to the resource value or remote node.

```text
provider-owned data schema
  type + constraints + semantics + capabilities
                    |
                    v
Desktop compatibility filter and default ranking
                    |
                    v
user-selected renderer + local display settings
```

Examples:

- a writable bounded integer with step 1 can use a slider, number stepper, or numeric text field; a read-only view can use a number, gauge, progress bar, or recent trend;
- a string can use a one-line input, multiline editor, or code editor when its constraints and semantic format allow it;
- a boolean can use a switch, checkbox, segmented choice, status indicator, or text, depending on whether it is writable and the user's selected presentation;
- a renderer that cannot faithfully enforce the provider's constraints is not offered.

Changing renderer never changes the underlying value. Editing continues to be staged and validated against the same provider-owned schema regardless of which control produced the draft.

### Variable

Read-only and writable states must be visually different.

| Declared value shape | Read presentation | Writable control |
| --- | --- | --- |
| boolean | state indicator with explicit text | switch, staged until Apply |
| enum | mapped label and optional semantic tone | select or segmented choice |
| bounded number | formatted stat, optional gauge/trend | number input plus slider when min/max exist |
| unbounded number | formatted stat with unit/precision | validated number input |
| short string | text/value display | text input |
| long string | wrapped or code/text display | textarea or code editor hint |
| date/time/duration | localized formatted value | matching date/time/duration control |
| object | key/value, grouped fields, or specialized view | generated field form |
| homogeneous array | table/list with empty state | repeatable rows when write is supported |
| unknown or unsupported schema | JSON tree/raw view | advanced JSON editor |

The entries above are compatible choices, not provider-mandated widgets. Desktop selects the initial choice and the user may switch within the compatible set.

Writable Variables default to staged editing with Reset and Apply. Apply continues to use `expected_revision`; a revision conflict must preserve the draft and offer reload/review instead of silently overwriting newer data. Immediate-write controls are not the default and would require an explicit trusted UI hint.

### Command and Generic Operations

- Generate fields from the input data schema with labels, descriptions, required markers, defaults, enums, bounds, formats, and inline validation.
- Keep one explicit Execute action; never submit on field change.
- Present output through the same data display system rather than a raw `<pre>` whenever the output schema is known.
- Keep an Advanced JSON mode for unsupported keywords, copy/paste, and diagnostics.
- Destructive styling or confirmation must come from declared action metadata; it must not be guessed from names such as `delete`, `revoke`, or `archive`.

### Stream and Topic

- Provide pause/resume, clear, filter/search, event count/rate, autoscroll control, and explicit gap/expired state.
- Render events as timeline, table, log, or structured details according to event schema and presentation hints.
- Topic publishing uses the same schema form as Command operations.
- Preserve publisher, publisher sequence, resource sequence/revision, timestamp when present, and raw payload access.

### File

- Use the Wails-native file picker/drop path rather than asking users to type a local path where supported.
- Show destination, size, progress, throughput, cancellation, completion, and actionable errors.
- Keep session and path validation at the Go boundary.

### First-party Specialized Views

Specialized renderers may be registered by stable schema ID or an explicit `presentation.renderer`, never by resource path. Initial high-value candidates are:

- catalog: searchable resource table/details;
- topology: node tree summary;
- health: status summary and checks;
- configuration: grouped key/value editor when writable through a declared operation;
- flow definitions/runs: tables with selection and structured details;
- flow events/audit/notification events: timeline/log;
- file transfers/progress: transfer table and progress view.

## Options Considered

| Option | Strength | Main limitation | Decision |
| --- | --- | --- | --- |
| A. Restyle current JSON renderers | Small and fast | Does not create semantic controls or meaningful displays | Reject |
| B. Hard-code first-party screens by resource name | Best short-term polish | Breaks extensibility and duplicates product logic in Desktop | Reject |
| C. Add remote schema discovery and a full plugin platform immediately | Fully generic end state | Expands protocol, trust, caching, compatibility, and migration scope before the UI foundation exists | Defer |
| D. Staged hybrid: schema resolver + first-party declarative registry + safe generic renderers | Immediate usable controls, preserves extension boundary, provides a path to remote discovery | Custom third-party nodes initially retain generic/raw fallback | Recommend |

## Recommended Technical Direction

Use a staged hybrid architecture:

```text
Resource descriptor + capability
            |
            v
SchemaResolver ----> built-in generated schema package (phase 1)
            |         owner schema provider (future phase)
            v
Data schema + UI presentation schema
            |
            v
ranked RendererRegistry
            |
            +--> generic display/control renderers
            +--> explicit specialized renderer
            `--> safe raw fallback
```

### Phase 1 boundary for the next plan

1. Add a typed `SchemaResolver` abstraction in Desktop.
2. Provide a generated or otherwise canonical built-in schema package for all current `protocol/schema_*.go` payloads; schema IDs remain the lookup key.
3. Use a JSON-Schema-compatible provider data vocabulary for type, constraints, and semantic annotations. Keep widget choice and display-only settings in a separate Desktop-owned presentation model. Provider presentation metadata may be treated as a default hint, never as executable UI or an exclusive component mandate.
4. Build the generic display/control primitives with the existing React/Radix-style component system.
5. Add a small set of high-value specialized first-party renderers selected by schema or explicit renderer ID.
6. Keep unsupported schemas in structured/raw fallback and clearly explain why a richer renderer is unavailable.
7. Persist only non-secret renderer settings in `ViewWidget.settings`.
8. Do not change the wire protocol in phase 1; make the resolver provider boundary explicit so owner-served schema discovery can be added without replacing the Renderer system.

The built-in phase-1 provider is a compatibility adapter for today's first-party schema IDs, not a claim that Desktop owns resource semantics. The future owner-served provider replaces that lookup source while preserving the same resolved schema and renderer-selection contracts.

### Future protocol boundary

A later workflow may define an owner-authoritative schema discovery Resource. It must return bounded, versioned, cacheable declarative data/UI schemas and never executable code. Trust, schema conflicts, offline caching, compatibility, and invalid-definition behavior require a separate protocol decision.

## Responsive Widget Model

Renderers should react to their pane container, not only the application viewport:

- compact: primary value/status and one safe primary action;
- normal: value plus metadata/control or a short event/table view;
- expanded: history, details, filters, advanced output, and raw inspector.

Below the renderer's documented minimum, the Workspace continues to scroll rather than hiding controls or silently changing layout.

## Accessibility and Safety

- Every generated field has a programmatic label, description, required state, and inline error association.
- Boolean state is expressed in text as well as color.
- Sliders always have a number-input or keyboard-equivalent path.
- Event live regions announce connection/gap state without announcing every high-rate payload.
- `readOnly`, `writeOnly`, and sensitive fields are respected; write-only values are never echoed from stored drafts.
- Remote schemas cannot introduce HTML, scripts, URLs with side effects, component names outside an allowlist, or arbitrary CSS.
- Client validation improves feedback but the Go/runtime boundary remains authoritative.

## Research Summary

- JSON Schema separates validation from annotations such as `title`, `description`, `default`, examples, `readOnly`, and `writeOnly`, which is appropriate for reusable data contracts: <https://json-schema.org/understanding-json-schema/reference/annotations>.
- JSON Forms demonstrates the useful separation between a data schema and a UI schema, then selects custom renderers through a registry rather than embedding component code in the data contract: <https://jsonforms.io/docs/> and <https://jsonforms.io/docs/tutorial/custom-renderers/>.
- Grafana keeps display configuration separate from the data and standardizes unit, min/max, decimals, thresholds, value mappings, and raw inspection across several visualization types: <https://grafana.com/docs/grafana/latest/visualizations/panels-visualizations/configure-standard-options/> and <https://grafana.com/docs/grafana/latest/visualizations/panels-visualizations/visualizations/table/>.

These sources support the architecture pattern; they do not mandate adopting their component libraries. The next plan should evaluate a small internal schema/rendering layer against a focused dependency, and must reject a large theme/component package that conflicts with the existing Desktop design system.

## Constraints and Risks

- The current Go structs are canonical payload definitions but are not machine-readable JSON Schema documents; generation and drift detection need an explicit source-of-truth decision.
- A schema ID can remain stable while an incorrect definition changes; generated artifacts need deterministic validation and contract tests.
- `oneOf`/conditional schemas, deep objects, large arrays, recursive refs, and unknown formats require bounded fallback behavior.
- Container-responsive widgets must avoid observing/reserializing on every pixel change.
- High-rate event rendering needs bounded buffers and batched UI updates.
- A polished form must not imply the user has permission; Forbidden remains an expected runtime state.
- Renderer compatibility must be recomputed when a provider schema changes. An incompatible saved renderer must fall back visibly to a safe renderer without mutating the resource or silently discarding the saved preference.

## Open Questions

Resolved for this workflow:

1. Implement the provider/display/user separation through the built-in first-party compatibility provider now; owner-served schema discovery is a later protocol workflow.
2. Use a bounded internal schema/renderer domain and existing project primitives. Do not add a competing form, theme, dashboard, or docking framework; reconsider only if implementation evidence shows the focused internal vocabulary cannot be maintained safely.

## Rejected Ideas

- Resource-name matching such as special-casing `system/health` directly in the generic renderer.
- Inferring sliders from the current numeric value without declared bounds.
- Making writable boolean switches send immediately by default.
- Rendering remotely supplied HTML/Markdown as trusted operational UI.
- Removing the raw inspector after specialized renderers exist.
- Using decorative cards, gauges, or charts for every resource regardless of semantics.
- Letting the provider hard-code a Desktop component, or letting a user-selected component weaken provider constraints.

## Handoff Criteria for `$m-plan`

The discussion is ready for planning when the user confirms the staged hybrid boundary. The plan must then:

- trace every task to this intake and the stable Desktop/resource contracts;
- decide the canonical schema-generation source and drift test;
- list the phase-1 schema/control matrix and specialized renderer coverage;
- define responsive, accessibility, permission/error, revision-conflict, and raw-fallback acceptance;
- record remote owner schema discovery as deferred unless the user explicitly expands scope;
- use this dedicated worktree and create confirmed root `plan.md` / `todo.md` before execution.

## Workflow Status

- `project_root`: `D:\project\MyFlowHub3`
- `docs_root`: `D:\project\MyFlowHub3\worktrees\desktop-schema-driven-widgets\docs`
- `code_repo`: `D:\project\MyFlowHub3\repo\MyFlowHub`
- `base`: `master` at `54b46e6`
- `branch`: `feat/desktop-schema-driven-widgets`
- `active_worktree`: `D:\project\MyFlowHub3\worktrees\desktop-schema-driven-widgets`
- implementation: approved through `$m-execute`
- planning: confirmed in root `plan.md`; tasks `DOC01`–`QA01` are active

## Routed Docs

- [Desktop feature](../features/desktop.md)
- [Desktop requirements](../requirements/desktop-resource-workspace.md)
- [Desktop Resource Workspace v3](../specs/desktop-resource-workspace-v3.md)
- [Resource Platform v2](../specs/resource-platform-v2.md)
