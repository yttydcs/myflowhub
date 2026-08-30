# Todo - Desktop Profile Entry Production

## Approval

- [x] `$m-discuss` completed with the two-entry direction and credential-state source confirmed.
- [x] Dedicated `feat/desktop-profile-entry` branch/worktree created from committed `master@5c17648`.
- [x] `$m-docs` routing and stable-doc impact recorded.
- [x] `$m-plan` drafted.
- [x] User explicitly approves execution Task IDs through `$m-execute` or `$m-go`.
- [x] Execution entry gate passes.

## Will Execute After Approval

- [x] DOC01 — clarify the Desktop requirement and add/index the focused Profile entry spec.
- [x] STATE01 — add non-mutating Enrollment inspection and sanitized per-Profile lifecycle projection.
- [x] SESSION01 — add non-destructive Profile deactivation and frontend state cleanup.
- [x] UI01 — implement Existing Profile / First Connection production React UI, valid stable Profile IDs, state-aware actions, and current feature truth.
- [x] QA01 — run focused/full Go, vet, generated, frontend, cross-platform compile, Wails package, security, and GUI gates.

## Will Not Execute In The Next Phase

- [ ] STARTUP01 — force Profile selection on every launch; deferred to preserve persisted active Profile and auto-connect behavior.
- [ ] MOBILE01 — Android/Embedded native Enrollment chooser; separate product approval required.
- [x] ARC01 — stable docs, reusable lesson, governed evidence, plan snapshot, local merge and cleanup closeout.
- [ ] PUB01 — push/release/deployment/publication; not authorized.

## Acceptance Checklist

- [x] Profile summaries are read-only, validated, bounded, and contain no private key, Permit, or Grant signature.
- [x] Legacy, missing, Device, Pending, Enrolled, and error states render distinct labels and actions.
- [x] Pending retry reuses its stable request/trust state without another TOFU confirmation.
- [x] Ordinary first connection has no editable Profile ID, Node ID, parent ID, or public-key pin fields.
- [x] Generated Profile IDs are valid, stable across name edits, and collision-safe.
- [x] Permit identity preparation/copy and approval/TOFU paths work; Permit clears only on success.
- [x] Returning to Profile selection clears the active Profile/client but preserves Profile, credential, preference, and View data.
- [x] Existing active startup, auto-connect, Settings, Legacy, headless, protocol, Resource, Renderer, and View behavior do not regress.
- [x] Light/dark, keyboard, focus, error, reduced-motion, and 1024×768 UI gates pass.
- [x] Go/frontend/generated/Wails package and real GUI validation pass.

## Rollback Checkpoints

- [x] R0 — DOC01 contract-only checkpoint.
- [x] R1 — STATE01 additive read-only API checkpoint.
- [x] R2 — SESSION01 deactivation checkpoint.
- [x] R3 — UI01 production interaction checkpoint.
- [x] R4 — QA01 verified archive-ready checkpoint.

## Current Gate

- Blocked: no — `DOC01`, `STATE01`, `SESSION01`, `UI01`, and `QA01` were explicitly approved via `$m-execute`.
- `$m-archive` documentation and post-mainline validation are complete; only the authorized local merge/worktree cleanup remains.
- No implementation sub-agents were dispatched because the write sets overlapped and no delegation was requested.
