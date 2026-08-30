# Todo - Desktop Schema-driven Resource Widgets

## Approval

- [x] `$m-discuss` completed with provider/display/user ownership confirmed.
- [x] Dedicated branch/worktree and canonical docs root confirmed.
- [x] `$m-docs` routing and stable-doc impact recorded.
- [x] `$m-plan` drafted.
- [x] User explicitly approves execution Task IDs through `$m-execute`.
- [x] `$m-execute` entry gate passes.

## Will Execute After Approval

- [x] DOC01 — stabilize feature, requirement, spec, decision, intake/index links.
- [x] SCHEMA01 — add bounded provider-owned first-party schema definitions, deterministic Desktop artifact, coverage/parity/freshness gates.
- [x] RENDER01 — add `SchemaResolver`, bounded validation, versioned `RendererRegistry`, compatibility/ranking, legacy aliases, settings validation.
- [x] VALUE01 — add generic scalar/object/array displays and controls; implement staged, revision-safe Variable UX.
- [x] OP01 — add generated Command/Topic/generic operation forms, explicit Execute, typed results, and Advanced JSON fallback.
- [x] LIVE01 — add bounded event views, native File selection/in-flight status, and schema-ID-selected first-party structured adapters; per-transfer cancel remains behind a future transfer-handle binding.
- [x] VIEW01 — integrate renderer selection, View persistence, responsive pane density, legacy/incompatible fallback, and accessibility.
- [x] QA01 — full Go, generated post-commit check, frontend regression/build, production Wails, real packaged GUI, persistence, theme, security, and screenshot gates pass.

## Will Not Execute In The Next Phase

- [ ] REMOTE01 — owner-served schema discovery protocol; deferred for separate trust/cache/version/migration design.
- [ ] PLUGIN01 — executable third-party renderer plugins; out of scope and requires a separate sandbox/trust model.
- [ ] DASH01 — full dashboard query/history/chart engine; deferred because this workflow has no query/history contract.
- [x] ARC01 — original workflow and 2026-08-31 renderer-titlebar follow-up are archived and closed locally.
- [ ] PUB01 — push/release/publication; not authorized and no remote is assumed.

## Acceptance Checklist

- [x] Provider schema controls type/constraints; Desktop controls compatible components; user selection changes presentation only.
- [x] Bounded integer `0..100`, step `1` exposes native pointer/keyboard range semantics with exact min/max/step and numeric fallback.
- [x] String supports compatible single-line/multiline/code/text modes.
- [x] Read-only data never exposes mutating controls.
- [x] Writable drafts use Reset/Apply; failed/conflicting writes preserve the draft.
- [x] Renderer selection persists per View and legacy IDs still open safely.
- [x] Renderer selection is compactly placed in the Widget title bar without redundant body text or a vertical divider.
- [x] Commands use generated forms with explicit Execute and Advanced JSON fallback.
- [x] Streams/Topics have a 100-event bounded, frame-batched buffer with live controls and visible gap/expired/error state.
- [x] File uses native selection and exposes in-flight/errors; independent first-party transfer resources expose owner progress within current capabilities.
- [x] First-party catalog/topology/health/config/flow/audit/notification/file schemas have useful structured views.
- [x] Unknown/unsupported schema and incompatible saved renderer produce explicit safe fallback.
- [x] Light/dark, nested compact/normal/expanded panes, Explorer, View layout, Profile, and offline/Forbidden behavior have no regression.
- [x] Generated artifact is deterministic, LF-stable, covered, and fresh in focused regeneration/freshness tests.
- [x] `npm test`, `npm run build`, focused/full `GOWORK=off go test`, generated check, Wails production build, interaction, and packaged GUI smoke pass.

## Rollback Checkpoints

- [x] R0 — DOC01 contract-only checkpoint.
- [x] R1 — SCHEMA01 generated schema checkpoint.
- [x] R2 — RENDER01 pure domain checkpoint.
- [x] R3 — VALUE01/OP01/LIVE01 renderer-family checkpoints.
- [x] R4 — VIEW01 integration checkpoint.
- [x] R5 — QA01 verified archive-ready checkpoint.

## Current Gate

- Blocked: no.
- `DOC01`–`QA01`, the original archive, and the post-merge renderer-titlebar follow-up are complete.
- Next gate: none for this workflow; closeout is local-only and no push or publication is authorized.
- Do not dispatch implementation sub-agents because the approved implementation write sets overlap.
