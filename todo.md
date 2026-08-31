# Todo - Resource Collections、Capability Actions 与 Desktop 交互

## Approval

- [x] `$m-discuss` completed and stable direction recorded.
- [x] Dedicated branch/worktree and canonical docs root confirmed.
- [x] `$m-docs` routing, indexes and stable-doc impact recorded.
- [x] `$m-plan` drafted.
- [x] User explicitly approves `DOC01, COLL01, FLOW01, FS01, SDK01, DESK01, RENDER01, QA01`.
- [x] `$m-go` entry gate passes；stage 3.2 delegated execution started.

## Approved Execution Scope

- [x] DOC01 — converge stable docs, feature truth and protocol map.
  - [x] First pass：approved contract、deferred boundary 与 plan/todo gate convergence.
  - [x] Final pass：implementation 后更新 Flow/Desktop current feature truth 与 generated-backed protocol map.
- [x] COLL01 — add Collection protocol and multi-capability runtime foundation.
- [x] FLOW01 — migrate Flow to definitions/runs Collections.
- [x] FS01 — add opt-in read-only filesystem Collection provider.
- [x] SDK01 — add typed Collection/capability clients and regenerate contracts.
- [x] DESK01 — add shared Resource actions and accessible context menu.
- [x] RENDER01 — add Collection widgets, View actions and safe content renderers.
- [x] QA01 — run full tests, generated/build gates and packaged Desktop evidence.
- [x] MAIN02 — merge latest master and reconcile concurrent Profile/Resource docs and UI state without discarding either contract.
- [x] ARC01 — archive change/plan/lesson/evidence and perform local integration/cleanup.

## Will Not Execute In The Next Phase

- [ ] AUTHZ02 — member selector policy and filtered/effective-capability discovery; separate protocol/policy design.
- [ ] CMD02 — migrate remaining Management/Admission/Notification endpoint Commands; deferred by domain.
- [ ] FS02 — filesystem write/delete, virtual multi-root mounts and large-file download sessions; separate destructive/data-lane design.
- [ ] PUB01 — push/release/publication; not authorized.

## Acceptance Checklist

- [x] Collection type/list/page/member schemas are bounded, deterministic and generated.
- [x] Multi-capability Resource dispatch preserves Registry schema/size/copy/error rules.
- [x] Flow catalog exposes only definitions/runs Collections and all first-party callers use capabilities.
- [x] Filesystem three-root fixture registers independently authorized Resources without leaking physical paths.
- [x] Filesystem traversal/symlink/junction/oversize and unsafe content cases fail closed.
- [x] Real Resource context menu and keyboard entry work; pure namespaces expose no Resource operations.
- [x] Input/mutating actions do not execute before explicit Execute.
- [x] View actions and Collection/text/code/raster/fallback renderers share the same operation path.
- [x] No content/draft/credential/permission decision is persisted in View settings.
- [x] Focused/full Go, generated, frontend, Wails build and packaged GUI gates pass.

## Rollback Checkpoints

- [x] R0 — planning/stable-doc-only checkpoint.
- [x] R1 — COLL01 protocol/runtime checkpoint.
- [x] R2 — FLOW01/FS01 provider checkpoint.
- [x] R3 — SDK01 generated contract checkpoint.
- [x] R4 — DESK01/RENDER01 Desktop integration checkpoint.
- [x] R5 — QA01 verified archive-ready checkpoint.

## Current Gate

- Approved Task IDs: `DOC01, COLL01, FLOW01, FS01, SDK01, DESK01, RENDER01, QA01`.
- Blocked: no.
- Active phase: `$m-archive`；archive docs prepared，local integration/cleanup in progress.
- Deferred: `AUTHZ02, CMD02, FS02, PUB01`.
- `MAIN02` is complete：latest master、Profile entry/deactivation、Resource Inspector state 与并行资源方案已语义合并；
  `PUB01` remains unauthorized.
- DOC01/DESK01/RENDER01/QA01 are complete；final frontend 16 files/128 tests、TypeScript/Vite build、full Go/race/vet、
  generated freshness、Windows Wails package、dual-ABI Android AAR、offline Gradle unit/lint/assemble and real browser/packaged
  GUI smoke pass. No Android device was attached, so physical-device smoke is Unavailable. Post-mainline full Go/vet/generated
  and frontend 17-file/134-test/build gates also pass.
