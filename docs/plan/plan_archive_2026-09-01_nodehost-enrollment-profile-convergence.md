# Plan - NodeHost Enrollment Profile Convergence

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `refactor/nodehost-enrollment-convergence`
- Base: `master@a9eb27001438029ef772ea9f71d66e896407654e`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\nodehost-enrollment-convergence\docs`
- Code Repos: canonical monorepo `MyFlowHub`
- Worktree: `D:\project\MyFlowHub3\worktrees\nodehost-enrollment-convergence`
- Current Stage: `$m-archive` documentation complete and feature commits in progress; `QA01` passed, governed change/plan/lesson artifacts are prepared, and control-plane integration remains pending.

## Stage Records

### Initialization

- `guide.md` read: canonical protocol/runtime docs live in repository `docs/`; worktrees must live under project sibling `worktrees/`; commits use Chinese messages.
- Owning repo confirmed: `D:\project\MyFlowHub3\repo\MyFlowHub`.
- Governed docs root confirmed: the canonical monorepo `docs/` in the active worktree.
- Main checkout is control-plane only and contains unrelated user-owned docs/design changes. They remain untouched.
- Dedicated branch/worktree created from `master@a9eb270`.

### Discuss - Discovery And Requirements Shaping

#### Goal

Remove the documented Desktop authority Enrollment owning-runtime exception so every granted Desktop node runs through the same NodeHost and attached SDK path, without changing Enrollment wire or requiring re-enrollment.

#### Scope

- Runtime auth adapter from protected Enrollment Grant to a validated Node credential source.
- Generic NodeHost consumption of resolved credentials without creating a duplicate identity.
- Enrollment-only bootstrap facade that cannot own ordinary Node operations after Grant.
- Desktop Profile bootstrap → NodeHost handoff, idempotent connect, failure rollback and state projection.
- Stable documentation, architecture guards, regression tests and packaged Desktop/Hub integration evidence.

#### Assumptions

- One logical Admission Authority and the current MFHE/MFH4 contracts remain authoritative.
- The protected Enrollment credential remains the canonical source for authority identity, Grant, parent anchor and Authority anchor.
- Legacy version 2 Profiles remain supported through their existing NodeID/IdentityStore path.
- Existing public owning binding APIs remain deprecated compatibility surface unless a separate cleanup task is approved.

#### Options Considered

1. Keep authority Profiles on owning binding: rejected; preserves two lifecycle and send-ownership models.
2. Let NodeHost run with Node ID zero during Enrollment: rejected; violates MFH4 identity invariants and conflates pre-auth with ordinary Node runtime.
3. Copy the granted identity into `identity.dpapi`: rejected; creates two persistent identity truths and a cross-file atomicity problem.
4. Add an `EnrollmentHost` or `nodeclient` wrapper: rejected; user direction and existing architecture require one neutral Host.
5. Inject a validated credential source into the existing NodeHost: selected; Enrollment stays pre-Node, while granted nodes share one runtime model.

#### Recommended Direction

```text
Profile preparation
        ↓
Enrollment bootstrap (Device / Pending; no Node)
        ↓ signed Grant persisted atomically
Validated enrolled credential source
        ↓ identity + parent anchor
NodeHost.New → register resources → Start
        ↓
Host.Client attached operation path
```

#### Research Summary

Local Git and code audit found no recent overwrite. The exception was introduced deliberately in `19b4cfe` and documented by `350e733`. Focused NodeHost/auth/enrollment/Desktop Go tests pass. The current frontend checkout has stale `node_modules`, while the same HEAD previously passed the complete clean frontend gate.

#### Issue List

- No blocking requirement question.
- Main checkout has concurrent docs changes; integration must reconcile rather than overwrite them.

### Plan - Requirements And Architecture

#### Discussion Summary

The user confirmed that NodeHost is the single ordinary-node runtime and that Enrollment only exists because a new device has no Node ID yet. Once the Authority Grant exists, continued use of an owning binding is no longer justified. Profile activation and network Enrollment must be separate concepts even if one Wails action orchestrates them for UI compatibility.

#### Accepted / Rejected Requirements

Accepted:

- Grant-before-Host ordering; no ordinary Node before a persisted Grant.
- One Parent-only NodeHost for both Legacy and enrolled authority Profiles.
- Grant-derived identity and parent trust; no second persisted identity file.
- Idempotent Connect over one Host supervisor.
- Existing authority Profiles reconnect without Permit or re-enrollment.
- Full credential/Profile consistency diagnostics and secret-safe projection.
- MFHE/MFH4, DPAPI, settings v2 and Legacy behavior remain compatible.

Rejected or deferred:

- Breaking removal of every owning SDK API, mobile/embedded Enrollment migration, settings v3, reparent/rekey and multi-Authority work.

#### Requirements Analysis

##### Functional Requirements

1. A missing/device/pending credential cannot create a NodeHost and cannot obtain a Node ID locally.
2. A signed enrolled credential yields exactly one validated Node identity and the Grant-bound parent anchor.
3. NodeHost must accept a credential source without using load-or-create identity behavior; missing/corrupt/mismatched credentials fail closed before external work.
4. Explicit NodeID/IdentityStore and credential-source modes are mutually exclusive or must match exactly; ambiguity is an error.
5. The Enrollment bootstrap exposes only status and Enroll operations; it does not expose Catalog/Operate/Subscribe or own the post-Grant Parent runtime.
6. Desktop closes the bootstrap before opening the granted Profile Host, preventing concurrent writers to the Profile state root.
7. Existing enrolled Profiles skip Enrollment and open NodeHost directly from the protected credential.
8. Repeated Connect while the Host is running or connecting waits for the existing supervisor instead of calling Start twice.
9. Profile settings never override credential-derived NodeID, parent or Authority truth. Any compatibility cache mismatch is explicit and secret-safe.
10. Candidate creation, failed login, Profile switching, deactivation, delete and shutdown preserve current atomic replacement and cleanup semantics.

##### Non-functional Requirements

- No private key, Permit, Grant signature or full public-key body enters settings, logs, frontend state or test artifacts.
- Credential reads clone sensitive buffers and do not mutate or create credential state.
- No duplicate runtime, Parent supervisor, SDK queue or state-directory writer is introduced.
- Existing public API and persisted formats remain backward compatible unless explicitly deprecated.
- Error messages distinguish missing/pending/corrupt credential, Grant mismatch, parent mismatch and lifecycle state.

##### Inputs / Outputs

- Input: protected `EnrollmentCredentialStore`, Profile endpoint/preferences, optional single-use Enrollment Permit/TOFU choice.
- Bootstrap output: `pending` outcome or atomically persisted signed Grant.
- Credential-source output: validated Node identity plus parent/Authority metadata; only identity and parent trust enter NodeHost.
- Runtime output: Parent-only NodeHost and attached Desktop operation facade.

##### Edge Cases

- Grant becomes durable but settings save fails.
- Existing Profile cache disagrees with Grant NodeID, parent or Authority.
- Bootstrap closes while Enrollment is blocked or returns pending.
- Duplicate Connect, auto-connect/manual-connect sequencing, and disconnect during connect.
- DPAPI credential is unreadable under another Windows user.
- Parent endpoint changes while the signed parent identity remains the same.
- Same Profile state root is already reserved by a Host.
- Deprecated owning API callers continue to compile while Desktop no longer uses them.

##### Acceptance Criteria

- After a successful authority login, `profileRuntime.host` is non-nil, RoleLeaf, listener-free, and its facade is attached/non-owning.
- Reopening an enrolled Profile creates NodeHost directly, uses the same Node ID/public key and does not call Enrollment or require Permit/TOFU.
- Repeated Connect succeeds or waits on the existing Parent supervisor; it never returns `binding client is already started`.
- No `identity.dpapi` or plaintext identity is created for an authority Profile; the protected Enrollment credential remains canonical.
- Pending and failed Enrollment create no ordinary Node, release bootstrap resources, preserve stable request/trust observation and remain retryable.
- Full Go/race/vet, generated freshness, Desktop frontend/build/Wails gates and real Hub + Desktop enrollment/restart smoke pass.

#### Architecture Design

##### Overall Solution

Add a small runtime/auth credential-source contract that loads a complete, already-enrolled Node credential from `EnrollmentCredentialStore`. The source validates the signed Grant and derives the Node identity from the original Device key without persisting another identity. NodeHost accepts that source as an alternative to its legacy NodeID + IdentityStore load-or-create path, initializes the normal trust/policy/admission state, and verifies/fills the configured parent identity from the Grant.

A narrow Enrollment bootstrap facade owns only the pre-auth handshake and credential mutation. Desktop uses it only while state is missing/device/pending. When state becomes enrolled, Desktop closes bootstrap, reopens the credential through NodeHost and installs a normal attached Desktop client. All active Profile runtimes therefore have the same ownership shape.

##### Interface Drafts

The implementation may adjust exported names to project conventions, but must preserve these contracts:

```go
type NodeCredential struct {
    Identity           Identity
    ParentNodeID       protocol.NodeID
    ParentPublicKey    ed25519.PublicKey
    AuthorityNodeID    protocol.NodeID
    AuthorityPublicKey ed25519.PublicKey
    EnrollmentID       string
}

type NodeCredentialSource interface {
    LoadNodeCredential() (NodeCredential, error)
}
```

`auth.EnrollmentCredentialSource` is read-only and fail-closed. `nodehost.Config` accepts either the existing legacy identity inputs or a `NodeCredentialSource`; the credential source derives the effective local Node ID and parent identity, while endpoint/Driver remain product configuration. Authority identity validates provenance but is not automatically promoted to arbitrary Resource permission.

##### Module Responsibilities

- `runtime/auth`: validate/derive enrolled credential; open normal state from a resolved identity without create-on-missing behavior.
- `host/nodehost`: own the single post-Grant Node/runtime/Parent/client lifecycle; reject contradictory credential and Parent inputs.
- `sdk/bindings`: provide a narrow Enrollment bootstrap; retain deprecated owning API only as compatibility surface.
- `apps/desktop`: orchestrate Profile preparation, bootstrap, handoff, activation, retry/rollback and safe status projection.
- `sdk/bindings/desktop`: remain an attached JSON/subscription facade for active runtime operations.

##### Error Handling And Safety

- Credential source errors are terminal until user repairs/deletes the Profile; they never generate a replacement identity.
- Grant/Node/parent/Authority mismatches report the mismatched category without exposing key bodies or signatures.
- Bootstrap is always closed before Host creation; failed Host creation releases state-directory reservation and leaves the Grant durable.
- Candidate replacement keeps the previous active Host until the new Profile can be opened, except same-Profile exclusive-state replacement, which retains the existing rollback/reopen rule.

##### Performance And Testing Strategy

- Credential load occurs at Profile open/activation, not per SDK operation.
- No additional message serialization, queues or network connection are added.
- Tests cover runtime/auth contracts, Host modes, binding ownership, Desktop end-to-end enrollment/restart, race cleanup and packaged GUI behavior.

##### Extensibility Design Points

- The credential-source seam can later support platform Keystore/Secure Enclave without changing NodeHost roles.
- Android/Embedded may adopt the same Grant → Host handoff later while keeping platform lifecycle facades.
- Settings v3 may remove authority cache fields after a separate migration decision.

#### Issue List

- Blocking: none.
- Approval required before any runtime, SDK, Desktop, test or stable-contract implementation edit.

### Stage 3.1 - Planning

#### Project Goal and Current State

Current code has one complete NodeHost but two Desktop runtime ownership paths. This workflow converges enrolled authority Profiles onto NodeHost while preserving registration and persistence behavior.

#### Docs Governance Routing Decision

- Docs root: `D:\project\MyFlowHub3\worktrees\nodehost-enrollment-convergence\docs`.
- Original request evidence: new intake record created during planning and indexed.
- Current user-visible truth: `docs/features/desktop.md` will change only after approval as part of DOC01.
- Durable intent: existing unified runtime, admission and Desktop requirements will be clarified.
- Technical contracts: NodeHost, Enrollment, Desktop Profile and operational lifecycle specs will be updated.
- Architecture decision: add a dated ADR that removes the Desktop authority owning-runtime exception without replacing centralized Authority.
- Reusable lesson: update the existing Desktop binding reconnect/admission lesson; do not create a duplicate lesson unless execution reveals a distinct failure pattern.

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake: `docs/intake/2026-09-01_nodehost-enrollment-profile-convergence.md`.
- Feature: `docs/features/desktop.md`.
- Requirements: `unified-node-runtime.md`, `auth-controlled-admission.md`, `desktop-resource-workspace.md`.
- Specs: `node-host-runtime.md`, `node-enrollment-and-admission-authority.md`, `desktop-profile-entry.md`, `operational-lifecycle.md`.
- Decisions: `2026-08-30_generic-node-host-and-non-owning-sdk-client.md`, `2026-08-30_centralized-admission-authority.md`.
- Lesson: `desktop-binding-reconnect-and-admission-diagnostics.md`.

#### Stable Docs Impact

- Intake impact: add — completed for source evidence during planning.
- Feature impact: clarify — authority Profile current runtime changes from owning binding to NodeHost.
- Requirements impact: clarify — Grant → Host convergence and canonical fact-source acceptance.
- Specs impact: clarify — credential-source contract, bootstrap handoff and idempotent Connect.
- Decision impact: add — close the explicit compatibility exception; centralized Authority decision remains unchanged.
- Lessons impact: clarify — owning binding retry guidance becomes bootstrap-only; active runtime guidance becomes Host-owned.

#### Executable Task List

- `DOC01` — converge stable contracts and record the exception removal decision.
- `AUTH01` — add enrolled Node credential source and state-opening contract.
- `HOST01` — let the neutral NodeHost consume resolved credentials safely.
- `BOOT01` — split narrow Enrollment bootstrap from post-Grant runtime ownership.
- `DESK01` — hand Desktop authority Profiles from bootstrap to NodeHost.
- `REG01` — add ownership, idempotency, migration and architecture regression guards.
- `QA01` — run full generated/build/race and real enrollment/restart integration gates.

#### Execution Scope After Approval

##### Will Execute

- `DOC01`, `AUTH01`, `HOST01`, `BOOT01`, `DESK01`, `REG01`, `QA01`.

##### Will Not Execute Now

- `LEGACY02` — remove all deprecated runtime-owning SDK/binding APIs; deferred pending downstream inventory and separate breaking-change approval.
- `MOBL02` — Android/Embedded MFHE Enrollment and Grant → Host migration; deferred because product/platform scope is separate.
- `SETV3` — remove authority Node/parent/Authority cache fields from Desktop settings; deferred to an explicit persisted-schema migration.
- `MAIN01` — reconcile and integrate with the main checkout's concurrent user docs/design edits; archive/integration phase only.
- `ARC01` — archive docs, merge and worktree cleanup; requires a later explicit `$m-archive` after QA passes.
- `PUB01` — push, release, publish or deploy; unauthorized and out of scope.

#### Task Details

##### DOC01 - Converge Stable Contracts

- Owner: execution worker assigned after approval.
- Worktree: active worktree.
- Plan Path: root `plan.md`.
- Goal: make Grant → NodeHost handoff and bootstrap-only Enrollment the durable documented contract.
- Files / Modules: Desktop feature; unified runtime/admission/Desktop requirements; NodeHost/Enrollment/Profile/lifecycle specs; NodeHost ADR and decision index; existing reconnect lesson and affected indexes.
- Write Set: `docs/features/desktop.md`, the listed `docs/requirements/*`, `docs/specs/*`, new `docs/decisions/2026-09-01_enrollment-bootstrap-nodehost-handoff.md`, the existing NodeHost ADR, `docs/lessons/desktop-binding-reconnect-and-admission-diagnostics.md`, nearest indexes.
- Acceptance: no stable doc still describes post-Grant authority reconnect through owning binding; wire and Legacy compatibility remain explicit.
- Test Points: relative-link check, index check, terminology search for stale exception text.
- Rollback: revert DOC01 docs without changing runtime.

##### AUTH01 - Enrolled Node Credential Source

- Owner: execution worker assigned after approval.
- Worktree: active worktree.
- Plan Path: root `plan.md`.
- Goal: derive a complete post-Grant Node credential from protected Enrollment state without create-on-missing or duplicate persistence.
- Files / Modules: `runtime/auth/enrollment_client_store.go`, `runtime/auth/state.go`, new focused auth file if appropriate, auth tests.
- Write Set: `runtime/auth/*enrollment*`, the minimal state-opening seam, focused `_test.go` files.
- Acceptance: enrolled state yields cloned validated identity/parent metadata; missing/device/pending/corrupt/mismatched state fails closed and does not write files.
- Test Points: Grant signature, Node/key/parent/Authority mismatch, immutability, no identity file, concurrent read safety.
- Rollback: remove the source/seam; existing Enrollment store remains readable and unchanged.

##### HOST01 - Credential-backed Neutral NodeHost

- Owner: execution worker assigned after approval.
- Worktree: active worktree.
- Plan Path: root `plan.md`.
- Goal: let the existing Host open a resolved credential while preserving one Node, one Parent supervisor and one attached Client.
- Files / Modules: `host/nodehost/config.go`, `host/nodehost/host.go`, `host/nodehost/host_test.go` and focused config tests.
- Write Set: `host/nodehost/**` plus only the auth seam required by AUTH01.
- Acceptance: legacy and credential-source modes are unambiguous; credential derives NodeID/parent, contradictions fail before listeners/dials, and Host lifecycle remains unchanged.
- Test Points: leaf role, pointer identity, no Listener, duplicate state-root reservation, failure rollback, Start/Close race, parent mismatch.
- Rollback: revert credential-source branch while retaining the original NodeID/IdentityStore mode.

##### BOOT01 - Narrow Enrollment Bootstrap

- Owner: execution worker assigned after approval.
- Worktree: active worktree.
- Plan Path: root `plan.md`.
- Goal: expose status/Enroll/Close without allowing the bootstrap to become the post-Grant operation runtime.
- Files / Modules: `sdk/bindings/enrollment.go`, minimal Desktop binding adapter if required, binding tests.
- Write Set: Enrollment binding files and their tests; no NodeHost or Desktop app files.
- Acceptance: narrow bootstrap cannot Catalog/Operate/Subscribe/Start ordinary Parent runtime; deprecated APIs remain source-compatible but are not canonical.
- Test Points: device/pending/granted flows, idempotent retry, Close cancellation, no Node before Grant, no secret JSON.
- Rollback: Desktop can temporarily remain on the old compatibility entry; credential format is unchanged.

##### DESK01 - Desktop Bootstrap To NodeHost Handoff

- Owner: execution worker assigned after approval.
- Worktree: active worktree.
- Plan Path: root `plan.md`.
- Goal: make every active/granted Desktop Profile own a Parent-only NodeHost and attached client.
- Files / Modules: `apps/desktop/app.go`, `config.go`, credential adapters, `sdk/bindings/desktop`, Wails boundary/generated files only if signatures change.
- Write Set: Desktop Go composition/profile runtime and minimal generated binding outputs; no frontend redesign.
- Acceptance: bootstrap closes before Host opens; enrolled restart skips Enrollment; repeated Connect is idempotent; legacy, pending, switch/deactivate/delete and rollback remain compatible.
- Test Points: permit and pending paths, persisted Grant restart, same/different Profile replacement, full cache mismatch diagnostics, DPAPI, no duplicate identity, no Permit/settings/log leakage.
- Rollback: revert composition to the documented compatibility path without deleting Enrollment credentials or Profile state.

##### REG01 - Regression And Architecture Guards

- Owner: execution worker assigned after approval.
- Worktree: active worktree.
- Plan Path: root `plan.md`.
- Goal: prevent a future merge from restoring the owning authority runtime or split fact sources.
- Files / Modules: auth/NodeHost/binding/Desktop tests, `internal/archtest` if the repository's architecture gate owns this rule, frontend Profile tests where behavior is visible.
- Write Set: focused test files and architecture assertions; production changes only if a test exposes an approved-scope defect.
- Acceptance: tests prove authority active runtime has a Host, Desktop production code does not call `StartEnrolledTCP`, Connect twice succeeds, and Profile state projection is credential-backed.
- Test Points: Go unit/integration/race, frontend existing/pending/enrolled flows, grep/import architecture guard, generated freshness.
- Rollback: remove individual guards only with the matching production rollback.

##### QA01 - Full Validation And Product Evidence

- Owner: execution tester assigned after implementation.
- Worktree: active worktree and clean detached validation worktree when needed.
- Plan Path: root `plan.md`.
- Goal: validate the entire registration → NodeHost → restart path and detect unrelated regressions.
- Files / Modules: no planned production writes; evidence under governed change verification at archive time or project artifacts during testing.
- Write Set: test/evidence outputs only; any repair returns to its owning Task ID.
- Acceptance: full Go/vet/race, canonical generated check, Desktop npm/test/build/Wails, Hub + packaged Desktop Permit and Pending enrollment, repeat Connect, restart without Permit, Profile switch/deactivate, and secret scan pass.
- Test Points: `scripts/mfh.ps1 -Action check/test -Target core`, generated check, Desktop target test/build, focused race, real GUI/TCP smoke.
- Rollback: no runtime rollback; failing signatures route to AUTH01/HOST01/BOOT01/DESK01/REG01.

#### Dependencies

```text
DOC01 ─┬─ AUTH01 ─ HOST01 ─┐
       └─ BOOT01 ──────────┼─ DESK01 ─ REG01 ─ QA01
                           ┘
```

- DOC01 fixes names and invariants first.
- AUTH01 and BOOT01 may execute in parallel because their production write sets are disjoint.
- HOST01 depends on AUTH01's credential contract.
- DESK01 depends on AUTH01, HOST01 and BOOT01.
- REG01 follows production convergence; QA01 is the final gate.

#### Risks and Notes

- Highest risk is creating two identity truths or a non-atomic Grant/identity migration; prohibited by design.
- Parent and Authority are different concepts. Only the Grant-bound direct parent becomes NodeHost parent trust; Authority provenance does not grant Resource permission.
- Public owning APIs may have unknown downstream consumers; Desktop stops using them, but deletion is deferred.
- Wails generated files must be regenerated deterministically if exported methods change.
- Main checkout docs have concurrent user changes overlapping likely DOC01 indexes/feature files; MAIN01 must use semantic merge and preservation checks.
- Worktree frontend must run `npm ci`; the main checkout's partial `node_modules` is not valid test evidence.

#### Parallelism Assessment

- No sub-agent is dispatched during planning.
- After explicit approval, AUTH01 and BOOT01 are safe parallel candidates with disjoint write sets.
- HOST01 starts after AUTH01; DESK01 and REG01 remain sequential because they touch shared lifecycle contracts/tests.
- Any execution worker must receive: this plan, the new intake, relevant stable docs, repository `AGENTS.md`/`guide.md`, exact write set, and the instruction to preserve main-checkout user changes.

#### Issue List

- Blocking: none for implementation.
- `DOC01`, `AUTH01`, `HOST01`, `BOOT01`, `DESK01`, and `REG01` are complete.
- `QA01` automated validation passed; actual packaged GUI operation and screenshot evidence remain in the heavy test phase.
- Approved execution remains limited to `DOC01`, `AUTH01`, `HOST01`, `BOOT01`, `DESK01`, `REG01`, and `QA01`.

### Execute - Implementation And Lightweight Validation

- `DOC01`: stable feature/requirement/spec/lesson contracts now define Enrollment as bootstrap-only and record the exception-removal ADR.
- `AUTH01`: protected Enrollment state exposes a read-only, cloned, fail-closed `NodeCredentialSource`; resolved identities can open normal auth state without identity persistence.
- `HOST01`: neutral NodeHost accepts credential-backed mode, resolves Node/Parent truth before external work, rejects ambiguous or conflicting configuration, and preserves Legacy mode.
- `BOOT01`: `EnrollmentBootstrap` exposes only status, Enroll and Close; Close cancels active handshakes. Deprecated owning APIs remain compatible.
- `DESK01`: authority Profiles close bootstrap before opening a Parent-only Host; enrolled restart and repeated Connect use the Host-owned supervisor and attached Client.
- `REG01`: tests cover missing/pending/corrupt credentials, cloned key material, no duplicate identity, ownership shape, full Profile cache mismatch, Permit/Pending handoff, restart, repeated Connect, cancellation and production architecture markers.
- Automated evidence: `go test ./...`, `go vet ./...`, focused `-race`, generated freshness, 134 Desktop frontend tests, Vite production build and Wails Windows production build passed.
- Heavy evidence remaining: launch and operate the packaged Desktop GUI against Hub, capture screenshots, and review the user-visible login/restart path under `$m-test`.

### Continue - Test Iteration 1

- Recovered state: `validation-needed`; all implementation Task IDs remain complete and mapped.
- Product startup evidence: a fresh Hub state started successfully at `127.0.0.1:17431`; the packaged `mfh-desktop.exe` started with an isolated `MFH_DESKTOP_CONFIG_DIR`; both processes were responsive before controlled shutdown.
- UI operation result: blocked before the first interaction. Windows application control failed once, then failed again after its prescribed kernel reset. The supported browser-control fallback failed with the same signature.
- Normalized failure signature: `QA01 / missing UI operation and screenshot evidence / control runtime initialization / failed to write kernel assets (os error 3)`.
- Progress evidence: new production-startup evidence exists, but no valid packaged-GUI interaction or screenshot evidence was produced.
- Hard blocker: the host UI-control runtime must be repaired or made available. Creating a custom UI automation bypass is outside the approved workflow and would not satisfy the required evidence gate.
- Test states and logs are isolated under `artifacts/qa01-gui/`; generated credential state must not be published or archived as evidence.

### Continue - Test Iteration 2

- Recovered state: `validation-needed`; implementation Task IDs remained complete and the previously unavailable supported Windows GUI-control runtime was available.
- Permit path: the packaged Desktop created a Profile, submitted an offline-issued one-time Permit, handed the granted credential to NodeHost, disconnected/reconnected without another Permit, and auto-connected after a production-process restart.
- Pending path: a second independent Profile submitted an approval request, remained outside the ordinary workspace while pending, was approved from the packaged Desktop admission UI, then connected on approval polling without another trust prompt and auto-connected after restart.
- Authority boundary: the test authority received only four admission-management permissions in the isolated Hub state. Both enrolled Profiles were still denied `system/topology`, confirming that Enrollment does not imply resource access.
- Ownership evidence: both Profiles persisted protected Enrollment credentials, resolved their granted Node/Parent identities, and had no `identity.dpapi`; the Enrollment credential remained the single identity fact source.
- Product screenshots and the sanitized run record are under `artifacts/qa01-gui/iteration-2/iteration-2.md`. Raw state and the one-time Permit in that directory are sensitive test material and must not be published or archived.
- A first-form Permit submission was rejected because PowerShell startup text contaminated the harness value. The GUI's bounded-JSON rejection was correct; secure field replacement passed without a product-code repair.
- Shutdown evidence: the packaged Desktop and Hub exited and port `17432` was released.
- Outcome: `QA01` passed after two test iterations across `$m-continue` invocations; iteration 2 required no code repair loop. The next permitted phase is a later explicit `$m-archive`.

### Archive - Documentation And Integration

- Entry gate: execution complete, QA/code review passed, Task IDs and changed files known, active plan current.
- `$m-docs` routing: canonical docs root remains repository `docs/`; intake, Desktop feature, admission/runtime requirements, Profile/Enrollment/NodeHost/lifecycle specs and ADR were updated before the change archive.
- Archive artifacts: `docs/change/2026-09-01_nodehost-enrollment-profile-convergence.md` and `docs/plan/plan_archive_2026-09-01_nodehost-enrollment-profile-convergence.md`; affected indexes are updated.
- Lessons: existing Desktop reconnect diagnostics and Windows frontend/PowerShell preflight entries now cover the reusable lifecycle, control-runtime and shell-output failure signatures.
- Sensitive QA material: the exact untracked `artifacts/qa01-gui/` directory was dry-run verified and removed before staging; Permit, DPAPI test credentials, isolated policy state, logs and screenshots were not committed.
- Implementation commit: `b94fe34 refactor: 收敛 Enrollment 与 NodeHost 生命周期`.
- Integration: pending archive commit, preservation of unrelated main-checkout dirt, local master merge, closeout record update and safe worktree/branch cleanup.
- Publication: local-only; no push, release, publication or deployment authorization.
