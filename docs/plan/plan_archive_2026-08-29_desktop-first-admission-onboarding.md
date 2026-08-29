# Plan - Desktop first-admission onboarding

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `fix/desktop-first-admission-onboarding`
- Base: `master@5de0cf2fa30ce4fc89ff54432712bb71bd2a9be4`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\desktop-first-admission-onboarding\docs`
- Code Repos: canonical `MyFlowHub` monorepo only
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-first-admission-onboarding`
- Current Stage: Archive complete; control-plane closeout in progress

## Stage Records

### Initialization

- `guide.md`: read; Chinese commits, sibling `worktrees/`, canonical `docs/`, real UI test required.
- Main checkout user changes: `guide.md`, `论文/**`, `design-demos/`; excluded and preserved.
- Dedicated worktree and semantic branch confirmed from clean `master` baseline.
- No subagents: security/API/UI changes overlap tightly and parallel file ownership would add conflict risk.

### Discuss - Discovery And Requirements Shaping

#### Goal

Make a fresh Desktop Profile capable of completing first admission without exposing private identity material or weakening Hub default-deny policy.

#### Scope

- Must: prepare/persist Profile identity, expose only public key, copy it, preserve inactive state, login with one-use Permit, real local Hub acceptance.
- Optional: compact field guidance that identifies the canonical local default endpoint `127.0.0.1:7331`.
- Not doing: Hub admin UI, Permit persistence, private-key export, protocol/schema change, icon work, publish.

#### Assumptions

- Windows DPAPI remains the production identity store.
- Parent identity and Permit are supplied by an operator or local bootstrap command.
- Existing `LoginJSON` remains the sole path that activates a new Profile after a successful connection.

#### Open Questions

- None blocking. The current repository already provides offline Hub identity, Permit issue and policy grant commands.

#### Options Considered

1. Add frontend-only calls to `SaveProfileJSON` then `IdentityJSON`: rejected because save marks the Profile active before admission and restart strands the user in a disconnected workspace without a Permit input.
2. Generate a temporary browser identity: rejected because it bypasses the credential store and would not match the production binding identity.
3. Add a narrow prepare API that persists identity and a non-active Profile: selected because it preserves credential ownership and activation semantics.

#### Recommended Direction

Add `PrepareProfileJSON`, returning `{profile, identity}`; make inactive Profile upsert explicit; add a two-step login UI; retain existing login/retry/permit behavior.

#### Research Summary

No web research used. Canonical feature/spec/requirement, current Go/React implementation and existing admission lesson were sufficient.

#### Worktree / Branch / Docs Root Status

- Ready: yes.
- Discuss handoff to plan: unblocked.

### Plan - Requirements And Architecture

#### Requirements Analysis

##### Use Cases

1. Fresh operator fills Profile and parent trust values, prepares an identity, copies the public key, receives a Permit, and logs in.
2. Operator reopens an unadmitted saved Profile and retrieves the same public key.
3. Invalid or changed Profile data fails explicitly and never silently activates or connects.

##### Functional Requirements

- Strictly validate Profile using the canonical Go boundary before persistence.
- Generate/load identity in the existing per-Profile credential store.
- Store prepared Profile without changing `active_profile_id`.
- Return only Node ID and public key; never return private key.
- Clear stale displayed identity when form data changes.
- Copy public key with an explicit success/error result.
- Preserve Permit after failed login and clear it only after success.

##### Non-functional Requirements

- No new dependency or protocol format.
- Lifecycle operations serialized with the existing mutex.
- Settings persistence remains atomic.
- Keyboard/focus and light/dark themes remain supported.

##### Inputs / Outputs

- Input: existing `Profile` JSON.
- Output: `{profile: Profile, identity: {node_id: string, public_key: string}}` JSON.
- Login input/output remains unchanged.

##### Edge Cases

- Existing Profile preparation reuses its DPAPI identity.
- Node ID mismatch against stored identity returns an explicit error.
- Prepare storage failure does not change in-memory settings.
- Identity-open failure does not persist a new Profile.
- Clipboard failure leaves the visible public key selectable.

##### Acceptance Criteria

- Fresh login page can prepare an identity and show/copy a stable public key without leaving the login screen.
- `active_profile_id` stays empty until successful `LoginJSON`.
- Permit is never persisted and no private key is exposed.
- Real production Wails flow reaches a connected workspace against a local default-deny Hub.

#### Architecture Design

##### Module Responsibilities

- `apps/desktop/config.go`: non-activating Profile upsert helper.
- `apps/desktop/app.go`: serialized prepare boundary and safe rollback ordering.
- `frontend/src/api.ts`: typed Wails facade.
- `LoginScreen.tsx` / `App.tsx`: two-step state, copy feedback and settings refresh.
- Go/Vitest tests: inactive persistence, identity stability, public-only output and UI workflow.

##### Data / Call Flow

`Profile form → PrepareProfileJSON → validate → credential.Open → IdentityJSON → atomic non-active settings save → return public identity → operator issues Permit → LoginJSON → connect → activate Profile`.

##### Error Handling and Safety

- Never log the Permit or private identity.
- Close the temporary client after preparation.
- Do not persist settings until identity preparation succeeds.
- Keep login failure retryable and preserve typed Permit.

##### Performance and Testing Strategy

- Identity preparation is one bounded credential/store operation; no polling or network I/O.
- Focused Go tests, React component/App tests, full frontend suite/build, Desktop Go tests, Wails build, actual local Hub onboarding and screenshot evidence.

##### Extensibility Design Points

- The public preparation response can later support guided remote operator handoff without embedding Hub administration in Desktop.

### Stage 3.1 - Planning

#### Docs Governance Routing Decision

- Intake impact: add.
- Feature impact: clarify.
- Requirements impact: clarify.
- Specs impact: clarify.
- Decision impact: none; existing CredentialStore/default-deny decisions remain authoritative.
- Lessons: update existing Desktop admission diagnostic lesson with bootstrap circularity prevention.

#### Related Docs

- Intake: `docs/intake/2026-08-29_desktop-first-admission-onboarding.md`
- Feature: `docs/features/desktop.md`
- Requirement: `docs/requirements/desktop-resource-workspace.md`
- Spec: `docs/specs/desktop-resource-workspace-v2.md`
- Lesson: `docs/lessons/desktop-binding-reconnect-and-admission-diagnostics.md`

#### Execution Scope After Approval

##### Will Execute

- ADM01, UI01, DOC01, VAL01, ARC01.

##### Will Not Execute Now

- ICON01: separate brand task.
- ADMIN01: Hub/Permit administration UI is outside Desktop trust boundary.
- PUB01: no remote and no publish authorization.

#### Task Details

##### ADM01 - Prepare inactive Profile identity
- Owner: main agent
- Files / Modules: `apps/desktop/config.go`, `apps/desktop/app.go`, Go tests, generated Wails binding
- Goal: prepare or reuse identity and save Profile without activation or network connection.
- Acceptance: public-only response; active ID unchanged; stable identity; invalid input/identity mismatch explicit.
- Tests: focused Go unit/integration tests and Desktop package suite.
- Rollback: remove the prepare method/helper and generated binding additions.

##### UI01 - Two-step first admission UI
- Owner: main agent
- Files / Modules: `frontend/src/api.ts`, `App.tsx`, `LoginScreen.tsx`, `style.css`, Vitest tests
- Goal: prepare, display and copy public key before Permit login.
- Acceptance: remains on login page, stale identity clears on edits, keyboard-accessible copy, Permit retry retained.
- Tests: App/Testing Library workflow plus production Wails operation.
- Rollback: remove prepare facade/controls while retaining existing login.

##### DOC01 - Stable docs and guidance
- Owner: main agent
- Files / Modules: Desktop feature/requirement/spec, admission lesson, intake/index
- Goal: make onboarding and security boundaries canonical.
- Acceptance: consistent docs; indexes correct; no icon or protocol claims.
- Tests: docs diff and link/path review.
- Rollback: revert scoped clarifications and intake record.

##### VAL01 - Real local Hub acceptance
- Owner: main agent
- Files / Modules: generated executable and isolated local runtime state only
- Goal: verify prepare → permit issue → Hub start → login → connected workspace.
- Acceptance: production Wails connected UI screenshot; permit not present in settings.
- Tests: frontend suite/build, Go tests, Wails build, actual UI path.
- Rollback: stop test Hub and remove only the explicitly isolated demo state if cleanup is requested.

##### ARC01 - Archive, merge and cleanup
- Owner: main agent
- Files / Modules: `docs/change`, `docs/plan`, root plan/todo, Git worktree/branch
- Goal: record evidence, commit in Chinese, fast-forward merge, preserve unrelated main-checkout changes and clean this worktree.
- Acceptance: archive indexed, master advanced, worktree/branch removed, no push.
- Rollback: branch remains recoverable by commit hash; no remote mutation.

#### Dependencies

- ADM01 → UI01 → VAL01 → ARC01; DOC01 may progress alongside implementation but is finalized after validation.

#### Risks and Notes

- Highest risk is accidentally treating preparation as authentication; guarded by inactive settings tests.
- Hub offline mutation must occur only while the test Hub is stopped.
- The running pre-change Desktop executable must be closed before rebuilding the same output path.

#### Parallelism Assessment

- No subagents. Go API, generated binding, React flow and security tests are tightly sequential and share contract surfaces.

#### Gate

- Plan confirmation: covered by the user's explicit “好的，请继续” after the proposed onboarding scope and the standing instruction to display the confirmation gate then continue.
- Blocked: no.
- Enter execution: ADM01, UI01, DOC01, VAL01, ARC01.

### Execute - Implementation And Lightweight Validation

- ADM01: implemented `PrepareProfileJSON`, non-activating Profile upsert and public-only/stable/inactive Go coverage.
- UI01: implemented staged identity preparation, public-key copy feedback, stale identity invalidation and canonical `7331` endpoint guidance.
- DOC01: intake created and Desktop feature/requirement/spec/admission lesson clarified.
- Lightweight validation: focused Go tests passed; full frontend 20 tests passed; TypeScript/Vite production build passed; `go test ./apps/desktop/...` passed; `git diff --check` passed.
- Heavy validation remaining: Wails binding regeneration/build, actual local default-deny Hub admission, UI screenshot and security state inspection.

### Test - Heavy Validation And Review

- Generated Wails bindings regenerated and `wails build -clean -platform windows/amd64` produced the Windows production executable.
- `npm test`: 4 files / 20 tests passed, including inactive identity preparation, copy feedback and stale identity invalidation.
- `npm run build`: TypeScript and Vite production build passed; tracked `dist` refreshed.
- `GOWORK=off go test ./apps/desktop/...`: passed.
- `GOWORK=off go test ./... -count=1`: all repository packages and integration tests passed.
- `gofmt -d` on changed Go files and `git diff --check`: clean.
- Real Wails acceptance used an isolated default-deny Hub at `127.0.0.1:7331`: prepared stable Node 2 identity, issued a one-use Permit, granted only test-resource read/subscribe policies, logged in, loaded 2 Nodes / 25 Resources and previewed `system/health`.
- Security inspection of `%APPDATA%/MyFlowHub/desktop-resource-workspace/settings.json`: active Profile became `local-dev` only after login; Permit and private key were absent.
- Screenshot evidence: `docs/change/verification/2026-08-29_desktop-first-admission-connected.png`.
- Review result: passed. Preparation does not authenticate, activate or connect; existing default-deny, CredentialStore and LoginJSON boundaries remain authoritative.
- Residual runtime state: isolated local demo Hub remains running for the user's live preview; it is outside the repository and is not product data.

### Archive - Documentation And Closeout

- `$m-docs` impact review complete: intake/feature/requirements/specs/lessons updated; decision impact remains none.
- Change record: `docs/change/2026-08-29_desktop-first-admission-onboarding.md`.
- Test evidence: `docs/change/verification/2026-08-29_desktop-first-admission-connected.png`.
- Plan snapshot: `docs/plan/plan_archive_2026-08-29_desktop-first-admission-onboarding.md`.
- Implementation commit: `de4ecfb` (`实现 Desktop 首次准入身份准备`).
- Closeout policy: fast-forward local `master`, preserve unrelated main-checkout changes, remove this worktree/branch, no push or publish.
- Brand boundary: no icon asset or brand decision changed or archived.
