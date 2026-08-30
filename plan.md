# Plan - Desktop Profile Entry Production

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `feat/desktop-profile-entry`
- Base: `master` @ `5c1764858fe5b706567a21c186566cc3775a955a`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\desktop-profile-entry\docs`
- Code Repos: canonical monorepo `MyFlowHub` only
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-profile-entry`
- Participating Modules: `runtime/auth`、`apps/desktop`、`apps/desktop/frontend`、Wails generated bindings、canonical `docs/`
- Current Stage: `$m-test` heavy validation passed; QA01 is archive-ready
- Publication: local-only; no remote, push, release, or deployment is authorized

## Stage Records

### Initialization

- `guide.md`: read; Chinese commit messages, canonical repository docs, `GOWORK=off`, sibling-worktree, and generated-binding rules apply.
- Project/docs/code repo confirmation: `D:\project\MyFlowHub3` is the umbrella project; the canonical monorepo is both the code repo and the explicitly governed docs root.
- Base/worktree confirmation: created `feat/desktop-profile-entry` at committed `master@5c17648` under the required sibling `worktrees\` directory.
- Main checkout protection: the main checkout contains unrelated modified Desktop renderer/workspace output, docs, demos, and `guide.md`; none is copied, reverted, staged, or reformatted by this workflow.
- Root control files: the completed schema-widget workflow is already retained in `docs/plan`; root `plan.md` and `todo.md` are replaced only inside this dedicated worktree as the new active control plane.

### Discuss - Discovery And Requirements Shaping

#### Goal

Apply the confirmed two-entry login/Profile prototype to production React while using real protected credential state, preserving Authority Enrollment, Legacy compatibility, headless behavior, and the existing active-Profile/auto-connect contract.

#### Scope

- Replace the always-expanded 01/02/03 login form with “使用现有 Profile” and “首次连接” entry modes.
- Expose a sanitized, read-only per-Profile Enrollment lifecycle projection from protected credential state.
- Add an explicit “返回 Profile 选择” action that deactivates without deleting a Profile.
- Generate valid local Profile IDs automatically in the ordinary first-connection path.
- Preserve Permit, approval/TOFU, Pending retry, Enrolled reconnect, Legacy, error focus, theme, and accessibility behavior.

#### Assumptions

- `EnrollmentCredential` remains the single local truth for `device/pending/enrolled`, request ID, trust observation, and Grant; `settings.json` does not duplicate this lifecycle.
- `active_profile_id` continues to mean the locally active Profile. An active Profile may enter the workspace on restart and may auto-connect according to its existing flag.
- “返回 Profile 选择” is local Profile deactivation, not human-account logout and not credential deletion.
- Existing HTML prototype is interaction evidence only; production continues to use React, Wails, BrandMark, real API data, and current design tokens.

#### Open Questions

- Blocking: none.
- Deferred: forcing Profile selection on every application launch. This would redefine persisted activation and is excluded to preserve compatibility.
- Deferred: native Android/Embedded Enrollment UI and multi-Authority availability.

#### Options Considered

1. **Visual-only React rearrangement**: small diff, but still guesses Pending from `node_id` and leaves the normal Profile-selection boundary incomplete.
2. **Two-entry React plus credential status projection**: accurate lifecycle and no persisted schema migration, but without deactivation the chooser remains difficult to reach after activation.
3. **Two-entry React plus read-only status projection plus explicit deactivation**: selected; gives the UI an accurate state model and a real return path while preserving auto-connect and protocol behavior.
4. **Persist lifecycle fields in Profile/settings**: rejected because it creates a second truth source that can drift from the protected credential.
5. **Copy prototype HTML/JS into production**: rejected because it contains hard-coded Profiles, command-style DOM, simulated Toast results, and invalid Unicode-derived Profile IDs.

#### Recommended Direction

Add a non-mutating runtime inspection seam and a narrow Desktop `ProfileStatesJSON` projection. Render saved Profiles from that projection, not from `node_id` heuristics. Add `DeactivateProfile` as an additive Wails lifecycle action. Rebuild `LoginScreen` around production state transitions, generate a stable valid Profile ID once, and keep trust pins/Legacy fields behind advanced settings.

#### Research Summary

No external research was required. The decision is based on current repository implementation, canonical docs, archived packaged validation, and the existing prototype/tests.

#### Worktree / Branch / Docs Root Status

- Dedicated branch/worktree: ready and clean before planning writes.
- Docs root: confirmed at the worktree `docs/` tree; no separate private docs repository exists.
- Discussion intake: added and indexed as `docs/intake/2026-08-31_desktop-profile-entry-production.md`.
- Runtime implementation/tests: not started.

#### Issue List

- None blocking planning.

### Plan - Requirements And Architecture

#### Discussion Summary

The Enrollment runtime already persists the exact local lifecycle and stable request ID, but the React login screen receives only Profile settings. As a result it displays every authority Profile without a Node ID as “等待注册” and cannot distinguish identity-only, Pending, Enrolled-but-not-yet-hydrated, missing credentials, or corrupt credentials. The production change should expose a sanitized projection from the existing truth source, then organize actions around that state.

#### Accepted / Rejected Requirements

Accepted:

- Existing Profiles and first connection are separate top-level entry modes.
- Existing Profile rows use exact lifecycle state and state-specific primary actions.
- Pending retry reuses the persisted request/trust observation and does not repeat TOFU.
- First connection hides Node ID and normal public-key pins, generates a local Profile ID, and keeps Permit/approval explicit.
- Deactivation preserves Profile, credentials, Views, and next-use recovery.
- Existing active startup, auto-connect, headless, protocol, Authority, and Legacy behavior remain compatible.

Rejected or deferred:

- Inferring Pending from an empty Node ID.
- Duplicating credential lifecycle in settings or UI preference.
- Regenerating Profile ID whenever the display name changes.
- Generating keys as a side effect of merely listing Profiles.
- Replacing Profile identity with a human account/password concept.
- Always showing the chooser on startup, mobile/embedded UI migration, multi-Authority, publication, and deployment.

#### Requirements Analysis

##### Goal

Deliver a production Profile entry experience that is simpler for ordinary users and more correct for Device/Pending/Enrolled recovery, without altering network admission semantics or secret-storage boundaries.

##### Scope

- Go read-only Enrollment inspection and Desktop Profile-state facade.
- Local Profile deactivation lifecycle.
- React state/types/API integration.
- Two-entry LoginScreen, Profile-row actions, automatic Profile ID, advanced compatibility settings, styling, accessibility, tests, generated bindings, and stable docs.

##### Use Cases

1. A new installation with no Profiles opens directly to minimal first connection.
2. A signed-out installation with saved Profiles opens the existing Profile chooser and defaults to the previously relevant/first Profile.
3. A Device Profile continues first connection without receiving a fake Pending label.
4. A Pending Profile checks the same request and, once approved, stores the Authority-assigned Node ID and connects.
5. An Enrolled Profile reconnects without Permit or repeated TOFU.
6. A user prepares/copies the local public key for a Permit, pastes the one-time Permit, and completes registration without exposing the private key.
7. A user returns from the workspace to Profile selection without deleting identity or Views, then selects another Profile.
8. Legacy Profiles continue through their explicit compatible fields.

##### Functional Requirements

1. Read-only credential inspection must distinguish `not found`, `device`, `pending`, and `enrolled`, validate found state, and never create/save a key, Grant, request, or directory solely because the chooser was rendered.
2. `ProfileStatesJSON` must return one bounded summary per saved Profile with Profile ID, lifecycle, optional request/Node/parent/Authority IDs, and a sanitized actionable error; it must never include private keys, Permit bodies/signatures, Grant signatures, or full protected credential records.
3. Legacy Profiles are projected as `legacy` without opening Enrollment credentials.
4. Missing or invalid protected credentials are explicit `missing`/`error` states; corrupt state never silently becomes a new Device identity.
5. An Enrolled credential with an empty Profile `node_id` is still shown as Enrolled and is allowed to reconnect/hydrate; a non-empty Profile identity that conflicts with the Grant fails explicitly.
6. `DeactivateProfile` serializes with login/connect/switch/delete, clears `active_profile_id`, closes the active client and subscriptions, persists the cleared selection, and preserves Profile records, protected identity, runtime state directories, preferences, and Views.
7. Deactivation failure or partial cleanup must produce an actionable error and the frontend must reload authoritative settings instead of guessing the final state.
8. Existing Profiles default to `enrolled -> 连接父节点`, `pending -> 检查审批并连接`, `device/missing -> 继续首次连接`, `legacy -> 兼容连接`, and `error -> disabled recovery guidance`.
9. Pending retry calls the existing login path with the same Profile, no Permit, and no repeated TOFU; the binding uses the persisted trust observation.
10. First connection defaults to approval and shows only Profile name, endpoint, admission choice, auto-connect, and relevant actions. Node ID is never user-assigned in the authority path.
11. Permit mode exposes explicit identity preparation/copy and Permit input. Permit content remains component/session state and is cleared only after successful login.
12. Approval without configured pins requires explicit TOFU once. Existing Pending with stored observation does not.
13. Profile ID is generated once as a valid ASCII ID with bounded collision retry, remains stable while name/endpoint are edited, is locked after the Profile has saved identity/state, and is only exposed in advanced or diagnostic UI.
14. Existing Profile edit uses a locked ID and state-aware fields; changes are committed by the existing prepare/login boundary, preserving current inactive/activation semantics.
15. Existing active Profile startup and `auto_connect` behavior are unchanged. Disconnect remains distinct from returning to Profile selection.
16. Tabs, Profile selection, menus, forms, validation, errors, busy states, clipboard feedback, and theme controls remain keyboard- and screen-reader-accessible.

##### Non-functional Requirements

- No persisted settings version upgrade and no lifecycle duplication in localStorage/settings.
- Read-only state loading is bounded by the existing 64-Profile limit and occurs on boot/explicit mutation refresh, not high-frequency polling.
- Sensitive material stays in CredentialStore/DPAPI; frontend summaries and logs are allowlisted.
- Runtime and Wails boundaries validate IDs/status values and fail explicitly on corrupt or conflicting state.
- Existing React/design tokens/BrandMark are reused; no UI framework or dependency is added.
- Light/dark, 1024×768 minimum target, reduced motion, focus visibility, non-color status labels, and clear errors are required.
- Headless Enrollment, Authority management, MFHE/MFH4, Resource, View, and Renderer behavior remain unchanged.

##### Inputs / Outputs

Inputs:

- persisted `Settings.Profiles` and `active_profile_id`;
- protected per-Profile Enrollment credential or Legacy identity;
- current connection status, Permit text, explicit TOFU, user-selected Profile/admission mode, and generated local Profile ID.

Outputs:

- sanitized `ProfileState[]` for presentation;
- state-specific login/continue/retry actions;
- updated active Profile after successful login or empty active Profile after deactivation;
- ordinary production React UI state and actionable errors.

No private key, Permit, Grant signature, resource value, or credential record is output or persisted by the chooser.

##### Edge Cases

- Profile JSON exists but credential is missing, corrupt, wrong DPAPI user, unsupported version, or session-only state vanished after restart.
- Credential is Enrolled but Profile hydration was interrupted before `node_id` was written.
- Profile `node_id` conflicts with the protected Grant.
- Pending request expired/rejected, parent endpoint changed, Authority unavailable, or parent/Authority observation conflicts.
- Profile name is Chinese/empty/very long; generated ID collision; browser random source unavailable.
- User switches tabs after preparing identity, edits endpoint after preparation, or retries after Permit/login failure.
- Clipboard access is denied; public key remains selectable.
- Deactivation while connected, disconnected, connecting, or while the workspace has unsaved View changes.
- Last/selected Profile is deleted; active Profile is absent; Legacy Profile remains the only saved entry.
- Generated Wails TypeScript binding is stale or a new worktree lacks generated files/dependencies.

##### Acceptance Criteria

- With no Profile, “首次连接” is selected and the ordinary form has no editable Profile ID, Node ID, parent Node ID, parent public key, or Authority public key.
- With saved Profiles, “使用现有 Profile” is selected and Device, Pending, Enrolled, Legacy, missing, and error fixtures render distinct labels/actions.
- Merely loading Profile summaries leaves credential files and key material unchanged; the JSON response contains no private/Permit/Grant signature fields.
- Pending retry uses `permit=''` and `allowTOFU=false`; Enrolled reconnect uses no admission choice; Device continues first connection.
- Approval first attempt requires explicit TOFU only when no configured/stored trust exists; Permit preparation returns the same public key on repeated reads.
- Generated Profile IDs always match the backend pattern, remain fixed while the name changes, and do not collide with 64 existing IDs.
- “返回 Profile 选择” closes the current client, clears active selection across restart, preserves Profile/credential/View data, and is guarded by unsaved-work confirmation.
- Active Profile restart and auto-connect still enter the workspace as before unless the user explicitly deactivated it.
- Legacy login and Settings Profile editing remain functional.
- Focused Go/frontend tests, full Go tests/vet, Wails binding generation/freshness, TypeScript/Vite, Windows Wails package, and representative real UI smoke pass.

##### Risks

- A status query that reuses `LoadOrCreate` would mutate state and hide missing credentials; the inspection path must be read-only by construction.
- DPAPI/session-only behavior differs by build tag; Windows runtime tests and Linux compile-only coverage are both required.
- `active_profile_id` currently gates the entire workspace; deactivation must reset frontend resource/View state without deleting it from disk.
- Reusing `node_id` heuristics anywhere in Login/Settings can reintroduce ambiguous Pending labels.
- Generated ID randomness can make tests flaky; generation must accept a deterministic test seam and bounded collision handling.
- LoginScreen, App state, Settings, generated bindings, and CSS overlap heavily, so parallel editing is likely to cause conflicts.

#### Architecture Design

##### Overall Solution

```text
protected Enrollment credential
        │ read-only inspect; never LoadOrCreate
        ▼
runtime/auth sanitized snapshot
        ▼
Desktop CredentialStore projection
        ▼
ProfileStatesJSON ────────────────┐
                                  ▼
Settings + ProfileState[] -> React entry state machine
                                  │
             ┌────────────────────┼─────────────────────┐
             ▼                    ▼                     ▼
         Existing             First connection      Credential error
 enrolled/pending/device      approval/permit        explicit recovery
             │                    │
             └────────── LoginJSON/PrepareProfileJSON ──> active workspace

workspace --DeactivateProfile--> saved Profiles remain, active selection cleared
```

##### Module Responsibilities

| Module | Responsibility |
| --- | --- |
| `runtime/auth/enrollment_client_store.go` | Validate and inspect existing Enrollment credential without creating or saving state; return a snapshot with no private key |
| Desktop platform credential backends | Provide non-creating access to an existing backend/credential; preserve DPAPI and session-only semantics |
| `apps/desktop/app.go` | Project per-Profile states, detect conflicts, expose `ProfileStatesJSON`, serialize and persist `DeactivateProfile` |
| `frontend/src/types.ts` / `api.ts` | Define the allowlisted Profile-state union and new Wails methods |
| `App.tsx` | Refresh settings/state together after mutations, own deactivation cleanup, and preserve active startup/auto-connect behavior |
| `LoginScreen.tsx` / `ProfileEditor.tsx` | Implement two-entry state machine, state actions, minimal first connection, locked/editable advanced fields, errors and accessibility |
| Profile ID helper | Generate/test valid stable collision-free local IDs independently of display names |
| `Settings.tsx` | Expose “返回 Profile 选择” while keeping Disconnect separate |
| `style.css` / BrandMark/UI primitives | Apply the confirmed compact production presentation in both themes |
| Tests/generated/docs | Protect credential safety, state transitions, Wails contract, UI behavior, and stable truth |

##### Data / Call Flow

1. App loads settings. If no Profile is active, it also loads `ProfileStatesJSON`; state refresh repeats after prepare, pending retry, successful login, delete, or deactivation.
2. `ProfileStatesJSON` snapshots the Profile list, maps Legacy directly, and asks the credential store to inspect authority state without creating it.
3. Found credentials are validated through `runtime/auth`; Desktop emits only allowlisted state/request/identity IDs. Missing/corrupt/conflicting states become explicit summaries instead of aborting the entire chooser.
4. LoginScreen selects Existing when Profiles exist, otherwise First Connection. A Device/missing row transfers its locked/generated Profile draft into First Connection; Pending/Enrolled/Legacy submit the existing login boundary directly.
5. First approval uses explicit TOFU only before any observation exists. First Permit preparation calls the existing protected `PrepareProfileJSON`, displays the public key, then submits Permit through `LoginJSON`.
6. Pending `LoginJSON` persists the inactive Profile and returns its current error/status; approval on a later retry hydrates the Grant into Profile settings and activates it.
7. `DeactivateProfile` clears the durable active ID and closes the current client under the lifecycle lock. React reloads settings, clears in-memory topology/resources/views/selection, and shows Existing Profiles.

##### Interface Drafts

Go runtime seam:

```go
func InspectEnrollmentClientState(store EnrollmentCredentialStore) (EnrollmentClientSnapshot, bool, error)
```

- `found=false` performs no save/create.
- found state is validated with the same invariants as normal load.
- snapshot contains no device private key.

Desktop DTO and facade:

```go
type profileState struct {
    ProfileID       string `json:"profile_id"`
    State           string `json:"state"` // legacy|missing|device|pending|enrolled|error
    RequestID       string `json:"request_id,omitempty"`
    NodeID          string `json:"node_id,omitempty"`
    ParentNodeID    string `json:"parent_node_id,omitempty"`
    AuthorityNodeID string `json:"authority_node_id,omitempty"`
    Message         string `json:"message,omitempty"`
}

func (a *App) ProfileStatesJSON() (string, error)
func (a *App) DeactivateProfile() error
```

Frontend contract:

```ts
type ProfileState = {
  profile_id: string
  state: 'legacy' | 'missing' | 'device' | 'pending' | 'enrolled' | 'error'
  request_id?: string
  node_id?: string
  parent_node_id?: string
  authority_node_id?: string
  message?: string
}
```

Exact public names may be tightened during implementation, but the state set, sensitive-field allowlist, no-create rule, and additive Wails boundary are fixed acceptance requirements.

##### Error Handling And Safety

- An unreadable credential produces a disabled Profile row with an actionable local-credential message; it never generates a replacement identity.
- A state/Profile identity conflict fails closed. The recovery path is explicit edit/delete/re-enroll, not silent overwrite.
- Pending/rejected/expired/Authority-unavailable errors preserve the selected Profile, Permit where applicable, and focused alert.
- Clipboard failure leaves the public key visible/selectable and reports manual-copy guidance.
- Deactivation reloads authoritative settings on every outcome so a storage/close error cannot leave React claiming the wrong active state.
- Delete remains separately confirmed and destructive; deactivation never calls credential/View removal.
- The status response is allowlisted and tested against forbidden sensitive field names/content.

##### Performance And Testing Strategy

- Inspect at most 64 Profiles only on boot or explicit lifecycle refresh. No background polling of DPAPI/profile summaries.
- Unit-test pure status/action mapping and ID generation separately from React rendering.
- Go tests cover non-mutating inspection, corrupt/missing/found credentials, state conflicts, deactivation persistence, and reconnect recovery.
- Frontend tests cover tab defaults, keyboard semantics, all state rows/actions, Permit/approval behavior, error focus, deletion/edit, deactivation, unsaved View guard, and compatibility.
- Regenerate Wails bindings from this worktree; never copy another worktree's `wailsjs`.
- Run focused and full Go checks, frontend tests/build, deterministic generated check, Linux compile-only for non-Windows credential code, Windows Wails production build, and representative packaged GUI smoke.

##### Extensibility Design Points

- The Profile-state union can later add an explicit `revoked`/`expired` recovery state when the local credential contract exposes it; unknown states must still fail closed.
- The chooser can later be shared with Android/Embedded without changing Authority semantics, but no cross-platform UI abstraction is introduced now.
- A future “always choose Profile on startup” preference can sit above `active_profile_id`; it is not encoded into the current credential/status API.

#### Issue List

- No blocking architecture question remains.

### Stage 3.1 - Planning

#### Project Goal And Current State

The production Enrollment/Authority path is implemented and previously validated, while the shipped React login remains the three-stage form. The design prototype and Playwright evidence exist, but lifecycle status is not exposed to React and the active Profile has no non-destructive return-to-chooser operation.

#### Docs Governance Routing Decision

- Docs root: `D:\project\MyFlowHub3\worktrees\desktop-profile-entry\docs` (same canonical monorepo; local-only).
- Original request evidence: new `docs/intake/2026-08-31_desktop-profile-entry-production.md` plus intake index.
- Current user-visible truth after implementation: clarify `docs/features/desktop.md`.
- Durable intent after approval: clarify `docs/requirements/desktop-resource-workspace.md`; `auth-controlled-admission.md` remains the existing admission source.
- Technical contract after approval: add `docs/specs/desktop-profile-entry.md` and index it; link existing Enrollment/workspace specs.
- Architecture decisions: no new/superseding decision; centralized Authority and protected-credential truth remain unchanged.
- Lessons: reuse existing admission/binding/generated/frontend lessons; decide at test/archive whether a new reusable lesson emerged.

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake:
  - `docs/intake/2026-08-31_desktop-profile-entry-production.md`
  - `docs/intake/2026-08-30_node-enrollment-central-authority.md`
  - `docs/intake/2026-08-29_desktop-first-admission-onboarding.md`
- Feature: `docs/features/desktop.md`
- Requirements:
  - `docs/requirements/desktop-resource-workspace.md`
  - `docs/requirements/auth-controlled-admission.md`
- Specs:
  - planned `docs/specs/desktop-profile-entry.md`
  - `docs/specs/node-enrollment-and-admission-authority.md`
  - `docs/specs/desktop-resource-workspace-v3.md`
- Decision: `docs/decisions/2026-08-30_centralized-admission-authority.md`
- Lessons:
  - `docs/lessons/desktop-binding-reconnect-and-admission-diagnostics.md`
  - `docs/lessons/frontend-and-powershell-preflight.md`
  - `docs/lessons/frontend-worktree-wailsjs-missing.md`
  - `docs/lessons/wails-binding-proto-drift.md`

#### Stable Docs Impact

- Intake impact: add — completed during planning and indexed.
- Feature impact: clarify — Profile chooser/first connection/state/deactivation behavior after implementation.
- Requirements impact: clarify — exact chooser lifecycle, generated local ID, deactivation, compatibility acceptance.
- Specs impact: add — focused Desktop Profile entry/status projection contract and specs index.
- Decision impact: none — no Authority or persisted-state ownership decision changes.
- Lessons known at planning time: existing lessons cover credential preparation, reconnect, Wails generation, and Windows frontend preflight; new lesson deferred until evidence warrants it.

#### Executable Task List

| Task ID | Title | Scope | Dependencies |
| --- | --- | --- | --- |
| DOC01 | Stabilize Desktop Profile entry docs | Will execute | none |
| STATE01 | Add read-only Enrollment/Profile state projection | Will execute | DOC01 |
| SESSION01 | Add non-destructive Profile deactivation | Will execute | STATE01 |
| UI01 | Map the two-entry prototype to production React | Will execute | STATE01, SESSION01 |
| QA01 | Run regression, generated, package, and UI gates | Will execute | UI01 |
| STARTUP01 | Force chooser on every application launch | Will not execute now | deferred; preserves current active/auto-connect semantics |
| MOBILE01 | Add Android/Embedded native Enrollment chooser | Will not execute now | out of scope; separate product approval |
| PUB01 | Push, release, deploy, or publish | Will not execute now | not authorized |

#### Execution Scope After Approval

##### Will Execute

- `DOC01`, `STATE01`, `SESSION01`, `UI01`, `QA01`.

##### Will Not Execute Now

- `STARTUP01` — deferred because it would redefine persisted activation and automatic startup behavior.
- `MOBILE01` — out of scope; current Android/Embedded clients remain on their documented compatibility paths.
- `PUB01` — no remote, push, release, deployment, or publication authorization.

#### Task Details

##### DOC01 - Stabilize Desktop Profile Entry Docs

- Owner: primary agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-profile-entry`
- Plan Path: `plan.md`
- Goal: make the confirmed UI/lifecycle contract discoverable before code changes.
- Files / Modules: `docs/requirements/desktop-resource-workspace.md`, new `docs/specs/desktop-profile-entry.md`, `docs/specs/README.md`, links from the new intake where necessary.
- Write Set: governed documentation only.
- Acceptance: requirement/spec record exact states, trust/deactivation/no-secret boundaries; indexes and cross-links are correct; current feature truth is not changed before the UI lands.
- Test Points: link/path review, `git diff --check`, stable-doc impact recheck.
- Rollback: revert DOC01 files without touching the intake evidence or code.

##### STATE01 - Add Read-only Enrollment/Profile State Projection

- Owner: primary agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-profile-entry`
- Plan Path: `plan.md`
- Goal: expose accurate Profile lifecycle without key creation or secret leakage.
- Files / Modules: `runtime/auth/enrollment_client_store.go` and tests; `apps/desktop/app.go`, platform credential backends and tests; `apps/desktop/frontend/src/types.ts`, `api.ts`; generated `apps/desktop/frontend/wailsjs`.
- Write Set: runtime read-only inspect helper, Desktop state DTO/facade, frontend type/API boundary, focused tests and generated bindings.
- Acceptance: legacy/missing/device/pending/enrolled/error map correctly; found state is validated; list read performs no save/create; conflicts fail closed; no private/Permit/signature fields leave Go.
- Test Points: runtime/auth table tests; Desktop missing/corrupt/session/DPAPI/state-conflict tests; JSON secret-field assertions; Wails generation/freshness.
- Rollback: remove additive inspect/ProfileStates surfaces and generated exports; existing credential format/settings remain unchanged.

##### SESSION01 - Add Non-destructive Profile Deactivation

- Owner: primary agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-profile-entry`
- Plan Path: `plan.md`
- Goal: provide a real path from active workspace back to saved Profile selection.
- Files / Modules: `apps/desktop/app.go`, `app_test.go`, Wails generated bindings, `frontend/src/api.ts`, `App.tsx`, `components/Settings.tsx`, `App.test.tsx`.
- Write Set: additive `DeactivateProfile`, frontend orchestration/reset, Settings action, focused lifecycle tests.
- Acceptance: active ID is durably cleared; client/subscriptions close; Profile/credential/View files remain; restart stays signed out; unsaved View guard and authoritative reload work; Disconnect remains separate.
- Test Points: Go connected/disconnected/save-failure/restart cases; frontend confirmation/state reset/API error cases.
- Rollback: remove the additive action/UI; no data migration or credential rollback required.

##### UI01 - Map The Two-entry Prototype To Production React

- Owner: primary agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-profile-entry`
- Plan Path: `plan.md`
- Goal: implement the approved compact Existing Profile / First Connection experience with real state and existing admission APIs.
- Files / Modules: `frontend/src/components/LoginScreen.tsx`, `ProfileEditor.tsx`, optional focused Profile ID/state helper and tests, `App.tsx`, `App.test.tsx`, `style.css`, existing UI primitives/BrandMark, `docs/features/desktop.md`.
- Write Set: React composition/state mapping, stable ID generator, state-aware edit/submit actions, CSS and focused frontend tests.
- Acceptance: correct default tab; exact lifecycle labels/actions; minimal first form; Permit/public-key and approval/TOFU paths; pending retry; locked identity; Legacy advanced path; no 01/02/03 or routine subtitles; 1024×768/light/dark/accessibility behavior; Desktop feature doc describes the implemented current behavior.
- Test Points: Vitest/Testing Library state/action matrix, keyboard/focus/error/clipboard tests, prototype-equivalent interaction smoke, TypeScript/Vite build.
- Rollback: restore previous LoginScreen/ProfileEditor/styles while retaining additive backend APIs if desired; no profile data conversion occurred.

##### QA01 - Run Regression, Generated, Package, And UI Gates

- Owner: primary agent
- Worktree: `D:\project\MyFlowHub3\worktrees\desktop-profile-entry`
- Plan Path: `plan.md`
- Goal: prove security, compatibility, generation freshness, UI behavior, and production packaging.
- Files / Modules: focused tests, generated Wails bindings, tracked frontend `dist`; temporary visual evidence routed during archive if retained.
- Write Set: tests/fixes necessary for planned behavior and deterministic generated/build output only.
- Acceptance: all planned checks pass; no sensitive material in summaries/settings/logs; no unrelated main-checkout files; packaged UI proves Device -> Pending -> Enrolled and deactivation -> chooser recovery.
- Test Points:
  - `$env:GOWORK='off'; go test ./runtime/auth ./apps/desktop/... -count=1`
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `$env:GOWORK='off'; go vet ./...`
  - `./scripts/mfh.ps1 -Action generate -Target generated` followed by generated freshness/diff check
  - Linux `apps/desktop` compile-only gate for `credential_other.go`; do not execute cross-target binaries on Windows
  - `npm ci`, `npm test`, `npm run build` in `apps/desktop/frontend`
  - `wails build -clean -trimpath -platform windows/amd64 -o mfh-desktop.exe`
  - real packaged Wails smoke for existing/device/pending/enrolled/Legacy/deactivate and representative light/dark 1024×768 UI
- Rollback: stop before archive/merge, revert to the latest passing task checkpoint, retain failure evidence without modifying persisted user credentials.

#### Dependencies

```text
DOC01 -> STATE01 -> SESSION01 -> UI01 -> QA01
```

- Documentation fixes state/interface vocabulary before public code names are generated.
- UI depends on the exact state DTO and deactivation contract.
- `App.tsx`, `app.go`, generated bindings, tests, and styles overlap; execute sequentially in one worktree.

#### Risks And Notes

- The main checkout is dirty with unrelated work; all edits and validation remain in this worktree until a later archive/integration phase.
- Status inspection must not call a create-on-missing path. This is the highest-priority security/correctness invariant.
- No settings schema migration is planned. If implementation appears to require one, stop and return to `$m-discuss`/`$m-plan` rather than improvising.
- No external dependency is expected.
- Production `dist` and Wails bindings are tracked generated artifacts and must be regenerated from this worktree only.
- A completed implementation is not authorization to merge, push, publish, or clean the worktree; `$m-archive` owns closeout.

#### Parallelism Assessment

- Initialization and planning were performed by the primary agent.
- No sub-agent was requested, and the implementation write sets overlap across `app.go`, Wails exports, `App.tsx`, LoginScreen, shared tests, and styles. Sequential primary-agent execution is the safe default.
- Do not dispatch implementation sub-agents before approval.

#### Issue List

- No technical blocker remains.
- No implementation blocker remains; heavy validation is intentionally routed to `$m-test`.

### Execute - Implementation

- Approval: the user explicitly invoked `$m-execute`, approving `DOC01`, `STATE01`, `SESSION01`, `UI01`, and `QA01`.
- Parallelism: skipped because the approved write sets overlap across Go lifecycle/state, generated Wails bindings, React orchestration, LoginScreen, shared tests, styles, and tracked `dist`; no delegation was requested.
- `DOC01`: completed — focused spec added/indexed, requirements clarified, intake linked, and feature truth updated after implementation.
- `STATE01`: completed — added validated non-creating Enrollment inspection, sanitized per-Profile state projection, DPAPI zero-write construction, frontend types/API, generated bindings, and conflict/secret/read-only tests.
- `SESSION01`: completed — added durable non-destructive deactivation, authoritative frontend reload/reset, Settings action, restart/preservation tests, and generated binding.
- `UI01`: completed — replaced numbered steps with Existing Profile / First Connection Tabs, exact lifecycle actions, hidden generated Profile ID, compact approval/Permit paths, pending retry, error recovery, keyboard semantics, styling, and tests.
- `QA01`: completed — full repository Go tests/vet, generated freshness, clean frontend install/test/build, Linux compile-only, Windows production package, security boundary review, and real packaged GUI smoke pass.
- Scope protection: `STARTUP01`, `MOBILE01`, `PUB01`, merge, push, release, deployment, archive, and worktree cleanup were not executed.

### Test - Heavy Validation

- Full regression: `$env:GOWORK='off'; go test ./... -count=1` and `$env:GOWORK='off'; go vet ./...` passed.
- Generated/build gates: `scripts/mfh.ps1 -Action generate -Target generated`, clean `npm ci`, 87/87 Vitest tests, TypeScript/Vite production build, Linux `apps/desktop` compile-only, and Windows Wails production packaging passed.
- Security boundary: protected-state inspection is non-creating and validated; the frontend projection remains allowlisted and excludes private keys, public-key bodies, Permit content, Grants, and signatures. Corrupt/conflicting credentials fail closed.
- Production GUI: the packaged Windows executable was exercised with isolated DPAPI state for missing/new, Device, Pending, Enrolled, and Legacy Profiles. Permit identity preparation, approval resume, Pending retry, Enrolled reconnect action, keyboard tab navigation, light/dark theme, and non-destructive deactivation all behaved as specified.
- Deactivation evidence: `active_profile_id` was durably cleared while both Profile records and both protected credential directories remained present.
- Minimum layout: the packaged window was resized to 1024×768; chooser controls, state labels, row actions, and the primary action remained visible without overlap or clipping.
- Visual evidence: `artifacts/m-test/desktop-first-connection.png` and `artifacts/m-test/desktop-profile-selection-1024x768-dark.png`.
- Sub-agent governance: no sub-agent was dispatched because neither the user nor repository/skill instructions authorized delegation for this phase; the primary agent performed the mandatory real-app validation.
- Cleanup: the temporary lifecycle seeding test source was removed before the final diff; no temporary test helper is part of the change set.

## Approval Gate

- Plan status: approved and implemented through `UI01`.
- Approval status: `DOC01`, `STATE01`, `SESSION01`, `UI01`, and `QA01` explicitly approved via `$m-execute`.
- Blocked: no.
- Next stage: `$m-archive`; do not merge, push, publish, or clean this worktree from `$m-test`.
