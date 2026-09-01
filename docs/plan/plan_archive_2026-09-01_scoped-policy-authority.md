# Plan - Scoped Policy Definitions And Authority Bindings

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `refactor/scoped-policy-authority`
- Base: `master@70add9b3ac436ff509e77f2b13d64a265f5e2fee`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: canonical repository `docs/` in the active worktree
- Code Repos: canonical monorepo `MyFlowHub`
- Worktree: `D:\project\MyFlowHub3\worktrees\scoped-policy-authority`（已清理）
- Current Stage: `$m-archive` complete; implementation, validation, governed archive, local master integration and worktree/branch cleanup passed; unrelated user dirt is restored
- Publication: local-only; no push, release, publication or deployment is authorized

## Stage Records

### Initialization

- `guide.md` read: canonical docs live in repository `docs/`; worktrees stay under project-level `worktrees/`; canonical commands use `GOWORK=off`; commits use Chinese messages.
- Owning repo, docs root and dedicated branch/worktree are confirmed.
- Main checkout contains unrelated user-owned changes and remains untouched.
- Repository has no remote.

### Discuss - Discovery And Requirements Shaping

#### Goal

Restore reusable, persistent permission groups on top of the current Resource owner + Capability authorization model so Desktop Node 41 can be explicitly granted full authority-domain access without making Desktop a privileged product.

#### Current State

- `runtime/auth.PolicyState` stores only exact `(Subject, Capability, Resource Owner, Resource Name)` grants.
- `AllowAll` is an in-process Policy implementation, not a persistent subject role.
- Current management exposes only exact `system/policy/grant` and `system/policy/revoke` Commands.
- Desktop has generic Resource operation support and an Admission console, but no current Policy Definition/Binding console.
- Pre-vNext history had `superadmin:*`, `admin`, `node`, role editing and node overrides; those legacy permission strings do not carry current owner/topology/generation semantics.

#### Recommended Direction

- Keep every product and runtime as an ordinary Node.
- Add named Policy Definitions containing bounded Resource-name and Capability selectors.
- Add persistent Policy Bindings from one exact Subject NodeID to one Definition and one owner scope.
- Retain exact grants as a compatibility/minimum-permission layer.
- Evaluate dynamic scopes against the current authoritative topology; do not create a second permissions tree.
- Explicitly bind Desktop Node 41 to immutable `superadmin` within `authority-domain:1`.
- Do not auto-bind the first Enrollment, the first Desktop, a product name or a fixed NodeID during migration.

#### Rejected Options

- Expand exact grants whenever catalog changes: rejected as race-prone and unbounded.
- Add selectors without reusable Definitions: rejected as repetitive and hard to audit/manage.
- Special-case Desktop, Node 41 or an installation package as `AllowAll`: rejected because it creates product privilege.
- Restore legacy `auth.role_perms` strings: rejected because they lose Resource owner, topology and generation semantics.

#### Discussion Artifact

- `docs/intake/2026-09-01_scoped-policy-roles-and-bindings.md`

#### Issue List

- Blocking architecture questions: none.
- Approval is still required before runtime, protocol, SDK, UI, tests or live policy state are changed.

### Plan - Requirements And Architecture

#### Requirements Analysis

##### Accepted Requirements

1. Policy Definition is a reusable rule collection, not a Node/network role.
2. Policy Binding is the Authority-owned persistent fact assigning a Definition to an exact Subject and owner scope.
3. Owner scopes in phase one are `owner`, `subtree` and `authority-domain`.
4. Resource selectors support exact, segment-safe prefix and all; Capability selectors support exact sorted sets and all.
5. Exact grants and v1 `policy.json` survive migration without semantic loss.
6. Default deny remains the fallback; phase one has no explicit deny or role inheritance.
7. Any effective policy mutation increments policy generation and invalidates old generation-bound sessions/subscriptions.
8. Dynamic scope membership uses one atomic current topology snapshot; reparent cannot retain stale scope membership.
9. Only an effective Authority-domain `superadmin` Binding may mutate Definitions, Bindings or exact grants online in phase one.
10. The immutable built-in `superadmin` Definition exists after initialization/migration but has no automatic Binding.
11. Desktop, SDK and CLI consume the same policy Resources; Desktop does not hold a local authorization bypass.
12. Node 41 receives one explicit persistent `superadmin + authority-domain:1` Binding after automated gates pass.

##### Non-functional Requirements

- Fail closed on missing topology resolver, corrupt policy state, unknown Definition, invalid selector, scope outside the local Authority domain, revision conflict or exhausted generation.
- Persist atomically before publishing in-memory state or generation notifications.
- Bound selectors, Definition/rule counts, Binding/grant counts, page size, text length and evaluation cost; no regex or arbitrary policy language.
- Keep compiled indexes by Subject and Definition so common evaluation is exact-map lookup plus that Subject's bounded bindings/rules.
- Policy list/effective responses and audit events contain no keys, permits, credentials or payload bodies.
- Schema, contract and Wails/generated outputs must remain reproducible.
- Existing non-policy NodeHost, routing, Resource, Metrics and Enrollment behavior remains compatible.

##### Use Cases

- Desktop Node 41 accesses current Metrics Resources through an Authority-domain superadmin Binding.
- A new Node joins under Authority 1 and its new Resources become accessible to Node 41 without new exact grants.
- A narrower custom Definition can later be created and bound to another Desktop, CLI or Agent Gateway Node.
- Revoking a Binding removes access and expires generation-bound subscriptions.
- Reparenting a Resource owner outside a scoped subtree removes scope membership immediately under the new topology epoch.
- Existing exact grants continue to authorize their exact tuples after policy state migration.

##### Inputs / Outputs

- Definition input: stable ID, display label, expected revision for update, ordered validated rules.
- Rule input: Resource selector and Capability selector; owner is not embedded in the Definition.
- Binding input: stable/idempotent Binding ID, Subject NodeID, Definition ID, owner scope, optional expiry.
- Exact grant input: current Subject/Capability/ResourceID tuple.
- Evaluation output: allow/deny plus matched exact grant or Binding/Definition/rule/scope metadata suitable for safe UI explanation.
- Collection output: generation-backed bounded pages for Definitions, Bindings and exact grants.

##### Edge Cases

- v1 state has grants but no Definitions/Bindings.
- A Definition update races with another update or affects active Bindings.
- Built-in `superadmin` is edited or deleted.
- Definition deletion is requested while active Bindings reference it.
- Binding creation is retried with the same ID and same/different payload.
- Binding expires while a subscription is active.
- Scope anchor is unknown, detached or outside the local Authority domain.
- Resource prefix `metrics/` must not match `metrics-evil/...`.
- A caller has exact invoke/create permission on policy Resources but lacks a superadmin Binding.
- Topology changes during scope evaluation.
- Node 41 identity changes while an old Binding remains.
- Live Hub is running when an offline bootstrap mutation is attempted.

##### Acceptance Criteria

- A migrated v1 `policy.json` reopens as v2 with all exact grants intact and an unbound immutable `superadmin` Definition.
- No Subject gains access solely because the binary or state schema was upgraded.
- Node 41 with `superadmin + authority-domain:1` can read/subscribe/invoke current and later-added Resources in that Authority domain.
- A future Node/Resource requires no grant expansion; a Node outside the scope remains forbidden.
- Revocation and Definition change increase generation and expire old subscriptions/sessions.
- Reparent and stale topology tests prove scope membership follows current tree state.
- Policy mutation by a caller that only has exact access to the management Resource is rejected.
- Desktop provides a real Policy console backed by Authority Resources and shows the effective Node 41 Binding after restart.
- Hub + Desktop + Metrics real smoke passes with the persistent Desktop profile and no repeated Permit/enrollment.

#### Architecture Design

##### Domain Model

The implementation may refine names to match project style, but must preserve these boundaries:

```go
type PolicyDefinition struct {
    ID        string
    Revision  uint64
    Immutable bool
    Rules     []PolicyRule
}

type PolicyRule struct {
    Resource   ResourceSelector // exact, prefix, all
    Capability CapabilitySelector // sorted exact set, all
}

type PolicyBinding struct {
    ID           string
    Subject      protocol.NodeID
    DefinitionID string
    Scope        OwnerScope // owner, subtree, authority-domain
    CreatedBy    protocol.NodeID
    CreatedAtMS  int64
    ExpiresAtMS  int64
}
```

- Bindings reference the current Definition by ID; Definition revision is used for optimistic updates and explanation, not to pin stale rule copies.
- `superadmin` is a canonical immutable Definition with all Resource names and Capabilities, but it remains inert until bound.
- Selectors are separate validated types; `*` is never accepted as a ResourceID or CapabilityID.

##### Evaluation Flow

```text
Request(Subject, Resource Owner/Name, Capability)
        ↓ normalize and validate
exact grant lookup ── matched ──→ allow
        ↓ no match
Subject Binding index
        ↓
current topology scope resolver
        ↓ owner in owner/subtree/authority-domain?
Definition rule match
        ↓
allow with explanation / default deny
```

- `PolicyState` owns persistent policy facts and compiled indexes.
- `runtime/tree.State` remains the sole topology fact source and exposes an atomic scope-membership query.
- Hub attaches a read-only topology scope resolver to `PolicyState` exactly once before opening external listeners.
- A binding-dependent authorization with no resolver fails closed; exact grants remain evaluable.
- Topology epoch handles membership changes; policy generation handles policy fact changes.

##### Persistent State And Migration

- Give policy state its own version constant instead of raising the auth package's shared identity/trust/admission version.
- v2 contains generation, exact grants, Definitions and Bindings.
- v1 load validates and normalizes all grants, inserts canonical `superadmin`, atomically writes v2 and does not create a Binding.
- Mutation is copy/validate/persist/swap/notify; persistence failure leaves memory and generation unchanged.
- Duplicate IDs, unknown Definition references, invalid revisions, expired/invalid times and non-canonical ordering fail explicitly.

##### Management Resource Model

Use Collection Resources so policy records are members and operations are Capabilities:

- `system/policy/definitions`: `list`, `get`, `create`, `update`, `delete`.
- `system/policy/bindings`: `list`, `get`, `create`, `revoke`, `evaluate`.
- `system/policy/grants`: `list`, `get`, `create`, `revoke`.
- Existing `system/policy/grant` and `system/policy/revoke` stay as exact-grant compatibility aliases.

Handlers use `resource.OperationRequest.Subject`; all policy mutations re-check an effective Authority-domain `superadmin` Binding server-side. An exact grant to a policy mutation capability alone is insufficient. Built-in Definition mutation and deletion of referenced Definitions return explicit Forbidden/Conflict errors.

List/get/evaluate remain independently authorizable Resource Capabilities. Pages use policy generation as collection revision and reject stale/repeated cursors. Mutation outcomes record actor, target Definition/Binding/grant key and status without payload contents.

##### SDK And Desktop

- Canonical Go SDK gains typed policy collection helpers; generic Operate remains the cross-platform base.
- Binding contract and generated Desktop schemas expose every new Resource/capability/schema and reject drift.
- Desktop adds a Policy section next to Admission under Authority-backed settings.
- Policy UI shows Definitions, Subject Bindings, exact grants and effective preview; immutable/all selectors and superadmin changes receive explicit warnings.
- UI never infers authorization from product type, descriptor visibility or local Profile fields; Authority Forbidden is displayed as the final result.

##### Bootstrap And Live State

- Extend the stopped-Hub offline CLI with policy show/bind/revoke-binding operations while keeping legacy exact grant/revoke flags compatible.
- Offline binding reports the stable Binding ID and resulting generation for audit and rollback.
- No migration path guesses the Desktop identity. The live step resolves the active persistent Hub state and Desktop NodeID, confirms Node 41, stops Hub, creates the one Binding, then restarts Hub/Metrics/Desktop.
- Existing exact grants remain intact in this workflow; cleanup is deferred so rollback only requires revoking the new Binding.

##### Error Handling

- Invalid selector/schema/state: `malformed` or startup error with field context.
- Missing record: `not_found`.
- Revision/idempotency/reference conflict: `conflict`.
- Missing scope resolver, detached anchor or stale topology: fail closed; wire maps to `forbidden` or `stale_epoch` as appropriate.
- Non-superadmin mutation and out-of-scope management: `forbidden`.
- Generation exhaustion or persistence failure: explicit internal/startup error; never partial success.

##### Performance And Test Strategy

- Exact grants remain O(1).
- Bindings are indexed by Subject; Definitions by ID; prefix matching is bounded by configured rule counts and path-segment checks.
- No catalog expansion, per-new-node policy writes, regex engine or network call occurs inside authorization.
- Unit tests cover validation, migration, indexing, mutation atomicity, generation and topology matching.
- Integration tests cover future Node/Resource, reparent, revoke, subscription expiry, management escalation denial and SDK paths.
- Product gates cover generated freshness, Go/race/vet, Desktop tests/build/Wails and real Hub + Desktop + Metrics restart.

##### Extensibility Points

- Custom `network-admin` and `observer` Definitions can be created without runtime changes.
- Explicit deny, role inheritance, Subject groups, bounded delegation and cross-Authority federation require later decisions.
- Android/Agent Gateway can use the same generic/typed SDK contract without becoming privileged products.

#### Issue List

- Technical blocker: none.
- Execution blocker: explicit approval of the Will Execute Task IDs.

### Stage 3.1 - Planning

#### Docs Governance Routing Decision

- Docs root: `D:\project\MyFlowHub3\worktrees\scoped-policy-authority\docs`.
- Intake impact: add — discussion record exists and is indexed; active plan link is added during planning.
- Feature impact: clarify after approval — Hub and Desktop current behavior will gain persistent role/binding management.
- Requirements impact: add/clarify after approval — add scoped policy authorization requirement and preserve admission separation.
- Specs impact: add/clarify after approval — add policy contract and update topology/resource/protocol maps.
- Decision impact: add after approval — scoped bindings are selected over product privilege and catalog expansion.
- Lessons impact: clarify after execution if tests confirm new escalation/topology diagnostics; reuse `authority-local-admin-actions.md` rather than create a duplicate prematurely.
- Root `plan.md`/`todo.md` are active workflow control-plane exceptions; archive belongs in `docs/plan`/`docs/change` during `$m-archive`.

#### Related Docs

- Intake: `docs/intake/2026-09-01_scoped-policy-roles-and-bindings.md`.
- Features: `docs/features/hub.md`, `docs/features/desktop.md`.
- Requirements: `docs/requirements/auth-controlled-admission.md`, `docs/requirements/unified-node-runtime.md`.
- Specs: `docs/specs/node-tree-link-resource-architecture.md`, `docs/specs/resource-collections-and-actions.md`, `docs/specs/operational-lifecycle.md`, `docs/specs/protocol_map.md`.
- Decisions: `docs/decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md`, `docs/decisions/2026-08-31_collection-resource-and-capability-actions.md`, `docs/decisions/2026-08-30_generic-node-host-and-non-owning-sdk-client.md`.
- Lesson: `docs/lessons/authority-local-admin-actions.md`.

#### Executable Task List

- `DOC01` — freeze stable policy requirement/spec/ADR and current-product documentation.
- `PROTO01` — add bounded policy protocol schemas, selectors and generic capabilities.
- `AUTH01` — implement policy state v2, exact-grant migration, Definitions/Bindings and evaluator.
- `TREE01` — add atomic topology scope resolution and attach it at Hub authority startup.
- `MGMT01` — expose policy Collections, harden mutations and audit outcomes.
- `SDK01` — add typed SDK helpers and regenerate canonical binding/schema contracts.
- `DESK01` — implement the Authority-backed Desktop Policy console.
- `BOOT01` — add safe stopped-Hub policy inspection and Binding bootstrap CLI.
- `QA01` — run unit, race, generated, build and isolated integration gates.
- `LIVE01` — persist Node 41's Authority-domain superadmin Binding and run real Hub + Desktop + Metrics restart smoke.

#### Execution Scope After Approval

##### Will Execute

- `DOC01`, `PROTO01`, `AUTH01`, `TREE01`, `MGMT01`, `SDK01`, `DESK01`, `BOOT01`, `QA01`, `LIVE01`.

##### Will Not Execute Now

- `DENY02` — explicit deny, role inheritance and Subject groups; deferred pending a separate precedence model decision.
- `DELEG02` — bounded delegation ceiling for non-superadmins; phase one intentionally uses the conservative superadmin-only mutation gate.
- `FED02` — cross-Authority federation or replicated policy state; outside the current single logical Authority model.
- `MOBL02` — Android/Embedded product-specific policy management UI; generic SDK/protocol compatibility is included, platform UI is separate.
- `CLEAN02` — delete Node 41's existing exact grants; deferred to preserve rollback and avoid unrelated policy cleanup.
- `ARC01` — archive, local merge and worktree cleanup; requires a later explicit `$m-archive` after QA/live evidence passes.
- `PUB01` — push, release, sign, publish or deploy; unauthorized and repository has no remote.

#### Task Details

##### DOC01 - Stable Policy Contracts

- Owner: execution worker after approval.
- Worktree: active worktree.
- Plan Path: root `plan.md`.
- Goal: make scoped Definitions/Bindings the durable contract without documenting unshipped behavior as already available.
- Files / Modules: `docs/features/{hub,desktop}.md`, new `docs/requirements/scoped-policy-authorization.md`, `auth-controlled-admission.md`, new `docs/specs/scoped-policy-authorization.md`, topology/resource/protocol specs, new ADR, affected indexes and existing authority admin lesson if warranted.
- Write Set: governed docs and nearest indexes only.
- Acceptance: docs explicitly separate identity/admission/authorization, ordinary Nodes/product neutrality, exact-grant compatibility, superadmin bootstrap and scope/generation semantics.
- Test Points: relative-link/index check and stale searches for “only exact triples”/automatic superuser claims.
- Rollback: revert DOC01 documentation independently.

##### PROTO01 - Policy Protocol And Selector Schemas

- Owner: execution worker after approval.
- Goal: define bounded wire payloads and collection capabilities for Definitions, Bindings, grants and effective evaluation.
- Files / Modules: `protocol/schema_policy.go` or project-consistent equivalent, `schema_catalog.go`, `data_schema.go`, focused protocol tests.
- Write Set: protocol schemas/constants/validation/tests; no runtime behavior.
- Acceptance: all structs validate version, IDs, canonical ordering, count/length/time bounds, segment-safe prefixes, scope kinds and optimistic revisions.
- Test Points: malformed/all/exact/prefix, duplicate capabilities/rules, invalid scope, JS-safe revisions, payload size.
- Rollback: remove new schemas/constants while old exact schemas remain.

##### AUTH01 - Persistent Policy Definitions, Bindings And Evaluator

- Owner: execution worker after approval.
- Goal: upgrade PolicyState without losing exact grants or auto-granting any Subject.
- Files / Modules: `runtime/auth/policy.go`, `policy_store.go`, new focused policy model/index files and tests.
- Write Set: runtime auth policy implementation and focused tests.
- Acceptance: v1→v2 migration is atomic/lossless; immutable unbound superadmin exists; union evaluation, expiry, idempotency, revision conflicts, generation watches and corrupt-state failure work.
- Test Points: exact O(1), subject binding index, prefix boundary, no resolver fail-closed, persistence failure rollback, concurrent mutation/race, generation exhaustion.
- Rollback: restore v1 code; migrated state requires the documented v2→v1 export/rollback fixture before downgrading.

##### TREE01 - Authoritative Topology Scope Resolver

- Owner: execution worker after approval.
- Goal: evaluate owner/subtree/authority-domain scopes from the single current Node tree.
- Files / Modules: `runtime/tree`, Hub startup wiring in `host/hub`, focused tree/Hub/node policy tests.
- Write Set: atomic read-only tree query, small auth resolver adapter/wiring, tests.
- Acceptance: self/descendant/domain matches are atomic; detached/unknown/outside owners deny; reparent changes membership under the new epoch; resolver is attached before listeners.
- Test Points: concurrent announce/withdraw/reparent, cycle/forged relation guard, root and relay cases, missing resolver.
- Rollback: remove resolver wiring; exact grants continue as before.

##### MGMT01 - Policy Collections, Mutation Guard And Audit

- Owner: execution worker after approval.
- Goal: provide network-manageable policy records without one Resource per CRUD operation or payload escalation.
- Files / Modules: `feature/management`, `runtime/resource.HandlerResource` usage, management tests.
- Write Set: policy Collection resources/handlers, legacy command adapters, audit helpers and tests.
- Acceptance: definitions/bindings/grants list/get/mutate/evaluate operate through capabilities; legacy exact commands remain compatible; every mutation requires effective superadmin and records safe outcome metadata.
- Test Points: pagination/generation, stale revision, referenced immutable definition, idempotent Binding, exact-invoke escalation denial, remote Principal preservation, audit redaction.
- Rollback: unregister new Collections and restore legacy handler path; state data remains readable by AUTH01.

##### SDK01 - Typed Clients And Generated Contracts

- Owner: execution worker after approval.
- Goal: make the new APIs convenient from Host-attached Go SDK and discoverable to all bindings.
- Files / Modules: `sdk/go/features.go`, `sdk/bindings/contract`, generated contracts/Desktop schemas, SDK/binding tests.
- Write Set: typed policy clients, contract declarations, generated outputs and tests.
- Acceptance: typed list/get/create/update/delete/bind/revoke/evaluate paths use the same attached Client; generic Operate remains valid; generated freshness passes.
- Test Points: owner validation, payload/result decode, pagination, error propagation, contract regeneration equality.
- Rollback: remove typed helpers/contracts; runtime Resources remain reachable through generic Operate.

##### DESK01 - Authority Policy Console

- Owner: execution worker after approval.
- Goal: ensure a real caller uses the APIs and can inspect/manage persistent permissions.
- Files / Modules: `apps/desktop/frontend/src/components/PolicyConsole.tsx`, Settings navigation, API/types/styles and focused frontend tests; Wails bindings only if exported Go surface changes.
- Write Set: Desktop frontend and focused tests/generated Wails outputs when necessary.
- Acceptance: role Definitions, Subject Bindings, exact grants and effective preview load from Authority; superadmin/all warnings, loading/empty/Forbidden/conflict states and revoke confirmation are explicit.
- Test Points: Node 41 effective result, create/update custom Definition, create/revoke Binding, unauthorized view, stale revision, keyboard/accessibility and no product-name inference.
- Rollback: remove Policy section; generic Resource operation remains available.

##### BOOT01 - Offline Superadmin Bootstrap

- Owner: execution worker after approval.
- Goal: create/revoke/inspect the first persistent Binding without weakening online authorization.
- Files / Modules: `cmd/mfh-hub`, README/run-dev help and focused CLI tests.
- Write Set: offline CLI flags/handlers/tests/docs; legacy exact grant/revoke flags remain.
- Acceptance: stopped-Hub show/bind/revoke-binding is atomic and outputs Binding ID/generation; invalid or duplicate/conflicting input fails; no automatic Subject selection.
- Test Points: fresh state, migrated state, idempotent retry, immutable Definition, rollback by Binding ID, running-state safety where supported by existing state ownership.
- Rollback: revoke the Binding before reverting CLI; keep legacy exact CLI.

##### QA01 - Automated And Isolated Integration Gates

- Owner: execution worker after approval.
- Goal: prove the model across protocol, runtime, management, SDK, Desktop and products before touching the user's live policy.
- Files / Modules: focused tests plus `tests/integration`; test fixtures only.
- Write Set: regression/integration tests and non-secret evidence; implementation repairs remain inside approved task write sets.
- Acceptance: default deny → scoped allow → future Node/Resource allow → revoke/expiry; reparent/out-of-domain deny; exact migration compatibility; policy management escalation denial; all build/generated gates pass.
- Test Points: `GOWORK=off go test ./...`, focused `-race`, `go vet ./...`, generated freshness, Desktop frontend tests/build, Wails production build, isolated TCP Hub + Desktop/SDK + Metrics test.
- Rollback: remove test-only fixtures; never relax assertions to obtain a pass.

##### LIVE01 - Persist Node 41 Binding And Real Product Smoke

- Owner: execution worker after QA01 passes.
- Goal: apply the user-requested persistent superadmin Binding to the current local Authority and verify it survives restart.
- Files / Modules: resolved local Hub policy state and runtime processes; no source file cleanup.
- Write Set: one `superadmin + authority-domain:1` Binding for Subject 41 plus secret-free local evidence.
- Acceptance: Hub stopped before mutation; Binding ID/generation captured; Hub, Metrics and Desktop restart; existing Profile auto-connects without Permit; Metrics read/subscribe succeeds; effective preview identifies the Binding; restart preserves it.
- Test Points: verify a Resource not covered by prior exact grants in an isolated/future-node path; verify an out-of-domain/unauthorized Subject remains denied; inspect logs for secrets and errors.
- Rollback: stop Hub, revoke the recorded Binding ID, restart; existing exact grants are left untouched.

#### Dependencies

```text
DOC01 ─┐
PROTO01 → AUTH01 → TREE01 → MGMT01 → SDK01 → DESK01 ─┐
                    └──────────────→ BOOT01 ──────────┤
                                                     ↓
                                                   QA01 → LIVE01
```

#### Risks And Notes

- `superadmin` all-Capability semantics intentionally cover future Capabilities; UI and bootstrap output must make that explicit.
- Mutation authorization must inspect the real routed Subject; `CommandHandler` paths that discard `OperationRequest.Subject` cannot be reused unchanged.
- Policy state needs its own schema version; changing shared `stateVersion` would incorrectly invalidate identity/trust/admission files.
- Scope resolution must not reconstruct topology from asynchronously refreshed management snapshots.
- LIVE01 changes user runtime policy and therefore runs only after automated gates and exact target/state checks.
- Old exact grants remain as rollback protection, so the live proof must use effective explanation and a not-previously-granted future Resource path.

#### Parallelism Assessment

- Protocol/docs and some UI scaffolding are technically parallelizable after contract freeze, but this session has no explicit user authorization for sub-agent delegation.
- Execution should proceed in dependency order in one agent unless the user separately authorizes delegated workers.
- Shared files `protocol/data_schema.go`, `feature/management/controller.go`, generated contracts and docs indexes must remain single-writer even if delegation is later approved.

#### Approval Gate

- Approved: `DOC01, PROTO01, AUTH01, TREE01, MGMT01, SDK01, DESK01, BOOT01, QA01, LIVE01`.
- Blocking: no.
- Implementation is complete; QA01 and LIVE01 passed. Archive/merge remains gated on a later explicit `$m-archive`.
- No implementation sub-agents are dispatched because user authorization for delegation is absent.

### Archive - Documentation And Integration

- Entry gate: execution complete; full/vet/race/generated/frontend/Wails/integration/LIVE01 evidence passed; changed files and Task IDs are known.
- `$m-docs` routing: canonical docs root remains repository `docs/`; intake, Hub/Desktop features, scoped authorization/admission requirements, Policy spec/protocol map and ADR were updated before the change archive.
- Archive targets: `docs/change/2026-09-01_scoped-policy-authority.md`, `docs/plan/plan_archive_2026-09-01_scoped-policy-authority.md` and `docs/lessons/scoped-policy-state-migration-and-live-proof.md`; affected indexes are updated.
- Implementation commit: `408afb5 feat: 引入作用域策略定义与权限绑定`.
- Archive commit: `4213452 docs: 归档作用域策略权限模型`.
- Integration: local master fast-forwarded from `70add9b` to `4213452`; unrelated user changes were replayed with matching patch-id and 24 matching untracked-file hashes.
- Cleanup: detached preview, feature worktree and merged feature branch were removed; live Hub/Metrics/Desktop were restarted from the independent artifact working directory.
- Remaining state: eight user-owned tracked modifications and 24 user-owned untracked files remain unstaged; two pre-existing stashes remain untouched.
- Outcome: `ARC01` complete; no blocker remains.
- Publication: local-only; no remote/push/release/publication/deployment authorization.
