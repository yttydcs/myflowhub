# Todo - NodeHost Enrollment Profile Convergence

## Approval Gate

- [x] User explicitly approves `DOC01, AUTH01, HOST01, BOOT01, DESK01, REG01, QA01`.
- [x] User selects `$m-execute` after approval.
- [x] Dedicated branch/worktree created from `master@a9eb270`.
- [x] Main checkout user changes identified and left untouched.
- [x] `$m-docs` routing and stable-doc impact recorded.

## Will Execute After Approval

- [x] DOC01 — converge stable contracts and add the exception-removal ADR.
- [x] AUTH01 — add fail-closed enrolled Node credential source.
- [x] HOST01 — add credential-backed mode to the neutral NodeHost.
- [x] BOOT01 — split narrow Enrollment bootstrap from post-Grant runtime ownership.
- [x] DESK01 — hand authority Profiles from bootstrap to Parent-only NodeHost.
- [x] REG01 — add ownership/idempotency/migration/architecture regression guards.
- [x] QA01 — automated generated/build/race and real TCP gates plus packaged GUI Permit, Pending/approval, reconnect and restart evidence passed.

## Will Not Execute Now

- [ ] LEGACY02 — remove all deprecated owning SDK/binding APIs; deferred pending downstream inventory and breaking-change approval.
- [ ] MOBL02 — migrate Android/Embedded Enrollment; separate product/platform scope.
- [ ] SETV3 — remove authority identity cache fields from settings; requires persisted-schema migration.
- [ ] MAIN01 — reconcile/integrate main checkout concurrent docs/design edits; archive phase only.
- [x] ARC01 — governed archive, local commits, master integration, user-dirt restoration and worktree/branch cleanup completed.
- [ ] PUB01 — push/release/publish/deploy; unauthorized.

## Dependency Order

- [x] DOC01 completes contract naming.
- [x] AUTH01 and BOOT01 may run in parallel.
- [x] HOST01 consumes AUTH01.
- [x] DESK01 consumes AUTH01 + HOST01 + BOOT01.
- [x] REG01 follows DESK01.
- [x] QA01 is the final execution gate.

## Current Status

- Phase: `$m-archive` complete; local master integrated and feature worktree/branch cleaned.
- Blocked: no.
- Runtime/business logic changes: credential source, credential-backed NodeHost, narrow Enrollment bootstrap, Desktop bootstrap-to-Host handoff and regression guards implemented.
- Validation: full Go/vet, focused race, generated freshness, Desktop frontend tests/build, Wails build, real Hub TCP paths, and packaged GUI Permit/Pending/approval/restart paths passed.
- Test iteration 1: product startup passed but GUI control was blocked by the host runtime.
- Test iteration 2: supported GUI control recovered; both independent Profile paths, authority approval, default deny, Enrollment-owned identity, reconnect and restart auto-connect passed without a code repair iteration.
- Archive boundary: raw iteration-2 credentials, Permit and policy state were removed before staging and were not published.
- Archive commits: `b94fe34` implementation and `a0c2fe7` governed archive; final closeout state is recorded by the subsequent master closeout commit.
- Integration: local `master` fast-forward completed; detached preview, feature worktree and merged feature branch were removed.
- Preservation: 8 tracked modifications and 24 untracked user docs/design files remain unstaged; exact replay checks passed and the transient stash was dropped. Two pre-existing stashes remain untouched.
- Publication: local-only; no push, release, publication or deployment was performed.
- Implementation agents dispatched: none.
