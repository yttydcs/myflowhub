# Plan - 集中式 Admission Authority 与 Node Enrollment

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `feat/node-enrollment-admission`
- Base: `master@550dfeebd55da5af4abce8a4c718ff30b77f5981`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\node-enrollment-admission\docs`
- Code Repos: `D:\project\MyFlowHub3\worktrees\node-enrollment-admission`
- Worktree: `D:\project\MyFlowHub3\worktrees\node-enrollment-admission`
- Current Stage: `$m-continue` converged on 2026-08-30 after `$m-test` passed; the subsequent Desktop login UX refinement is revalidated and ready for `$m-archive`

## Stage Records

### Initialization

- `guide.md`: read and applied; canonical monorepo uses `GOWORK=off`, Chinese commit messages, sibling worktrees only.
- Owning repo: canonical `MyFlowHub` monorepo; retired sibling repositories are migration evidence only.
- Docs root: canonical repository `docs/`; no separate private docs root exists or is expected.
- Worktree: clean dedicated worktree on `feat/node-enrollment-admission`.
- Existing dirty main checkout changes are outside this worktree and must not be copied, reverted, or reformatted.

### Discuss - Discovery And Requirements Shaping

#### Goal

Replace the current client-preassigned `NodeID + parent NodeID + parent public key + Join Permit` onboarding with a first-enrollment lifecycle that:

1. has no formal Node ID before grant;
2. supports a pre-issued Permit path and a pending-approval path;
3. keeps direct-parent control and headless operation;
4. provides one global Permit/request/enrollment management view;
5. allows an authorized operator connected through node A to issue a Permit usable at target parent B;
6. prevents Permit double-consumption and Node ID duplication without distributed multi-writer merges.

#### Scope

- Single logical Admission Authority per authority domain, defaulting to the root node.
- Direct parent owns the physical link, pre-authentication challenge, local trust activation, disconnect, and subtree routing.
- Admission Authority owns Permit, pending request, enrollment, revocation, Node ID allocation, and audit state.
- New `Enroll` phase before existing `Join` for nodes without an assignment.
- Central management resources and Desktop/CLI administration.
- Compatibility for already-enrolled vNext nodes and the current `ProvisioningPermitV1` Join path during migration.

#### Assumptions

- An unregistered child can persist an Ed25519 device keypair before it has a Node ID.
- `device_id` is a versioned fingerprint derived from that public key, not an arbitrary client-supplied identity field.
- Authority-issued Permit delivery is out-of-band and must not be logged or persisted as ordinary settings.
- Existing enrolled nodes can reconnect from cached direct-parent trust without contacting the Authority on every Join.
- New enrollment, Permit consumption, approval, and global revocation fail closed while Authority is unavailable.
- The first implementation has one logical Authority writer. HA replication, if added later, must remain strongly consistent.

#### Open Questions

- None blocking for the first implementation. The following defaults are selected for planning:
  - Authority is the root unless explicitly configured.
  - Authority alone generates random positive 63-bit Node IDs, checks a durable unique index, retries conflicts, and retains tombstones.
  - Permit is bound to the child public-key fingerprint and a target parent or subtree scope.
  - Permit-assisted enrollment anchors the parent key through Authority-signed scope metadata; approval without a Permit uses explicit TOFU by default, with pinned-fingerprint mode available.
  - `admission_profile` records the enrollment class and constrains admission/management behavior; it does not silently grant arbitrary resource access. Resource permissions remain explicit `system/policy/*` grants.

#### Options Considered

1. Keep current Join Permit and preassigned Node ID: rejected because it has no unregistered state and requires manual parent identity input.
2. Let every direct parent issue/consume Permit and allocate IDs independently: rejected because global listing, cross-parent use, revocation, double-spend prevention, and ID uniqueness become distributed-consensus problems.
3. Restore the old `register -> login -> up_login` wire protocol: rejected as a literal implementation, but its `pending / approved / permit / post-approval Node ID` semantics are retained.
4. Single logical Admission Authority with distributed parent links: selected.
5. Federated multi-Authority permit merging or CRDT state: deferred; security state requires strong consistency.

#### Recommended Direction

- Add a transport-independent pre-auth enrollment frame family using a distinct magic/version from the normal `MFH4` Envelope.
- The listener dispatches the first frame to legacy/current Join or new Enrollment without weakening the invariant that ordinary Envelopes require non-zero Source and Target Node IDs.
- An unregistered child proves possession of its private key over a parent challenge and transcript.
- The direct parent submits the verified request to the configured Authority through a normal authenticated Command when the Authority is remote, or calls the same local Authority service when it is local.
- Permit consumption or approval produces one idempotent Authority Grant containing the assigned Node ID, parent binding, child key fingerprint, Authority identity, enrollment profile, generation, timestamps, and signature.
- After the client durably stores the binding, the same pipe may transition into normal Join; reconnect remains a safe fallback.
- Permit, request, enrollment, Node ID allocation, and revocation are a single Authority state machine, not separate mergeable stores.

#### Research Summary

- SSH host-key trust documents the unavoidable first-connection choice between a pre-known anchor and TOFU: <https://www.rfc-editor.org/rfc/rfc4251.html>.
- TLS 1.3 and Noise show certificate/PSK and transcript-binding patterns; they inform nonce/signature binding but do not require a root PKI for this scope: <https://www.rfc-editor.org/rfc/rfc8446.html>, <https://noiseprotocol.org/noise.html>.
- Full zero-touch PKI/voucher systems are intentionally deferred because the product requires direct-parent operation and a smaller deployment surface.

#### Worktree / Branch / Docs Root Status

- Worktree: ready and clean.
- Branch: ready.
- Docs root: confirmed in canonical repo.
- Discussion artifacts: captured in this root plan; stable docs remain unchanged until the plan is approved.

#### Issue List

- No planning blocker.
- User approved the `Will Execute` Task IDs by invoking `$m-execute` on 2026-08-30.

### Plan - Requirements And Architecture

#### Discussion Summary

The node tree stays distributed, but security credentials are not multi-writer. A logical Authority is the sole source of truth for enrollment. Operators may administer it from any authorized node through normal routed management Commands; the operator node is the audited actor, while the Authority is the cryptographic issuer. A Permit can target parent B even when issued from management node A. Parent B forwards consumption to the Authority, which atomically grants one Node ID or returns an idempotent prior result.

#### Accepted / Rejected Requirements

Accepted:

- Permit direct enrollment and pending approval are equal first-class paths.
- No formal Node ID exists before Authority Grant.
- Parent Node ID and public key are learned and persisted by the protocol, not entered as routine client fields.
- All Permit/request/enrollment administration has one listable Authority view.
- Existing enrolled nodes keep working during migration.
- Headless CLI and background retry are first-class; Desktop is not the only enrollment mechanism.
- Revocation is state-specific: active Permit revoke, pending reject, active enrollment revoke/disconnect/tombstone.

Rejected or deferred:

- No parent-local Permit merge.
- No client-selected Node ID.
- No generic unbound bearer Permit in the first version.
- No silent fallback to a local Authority when the configured Authority is unavailable.
- No federated Authority-domain merge, offline allocation ticket, Authority HA consensus, reparent migration, or parent-key rollover in the next execution phase.

#### Requirements Analysis

##### Goal

Deliver a secure and manageable enrollment control plane that preserves the authoritative node tree while removing manual identity bootstrapping fields and distributed Permit ambiguity.

##### Scope

- Protocol schemas and bounded enrollment framing.
- Device identity and enrollment binding persistence.
- Authority state and allocation transaction.
- Direct-parent enrollment broker and Authority routing.
- Management Resources, Go SDK, CLI, Desktop onboarding, and Desktop admission console.
- Migration and compatibility tests.

##### Use Cases

1. Provisioning pipeline prepares a device key, obtains an Authority Permit scoped to parent B, and the headless child enrolls immediately through B.
2. A child without a Permit connects to B, creates one pending request, and retries until an authorized operator approves it from Desktop connected through A.
3. An operator lists, filters, issues, and revokes all Authority-domain Permits from one interface.
4. A consumed Permit response is lost; the same child retries and receives the same Node ID and receipt.
5. Authority is unavailable; new enrollment fails explicitly while existing enrolled children continue cached Join behavior.
6. A malicious client rotates keys and floods requests; bounded handshake, stateless challenge, dedupe, rate, TTL, and record limits prevent unbounded state or Node ID allocation.
7. A Permit or enrollment is revoked; future Join is denied, an active direct session is disconnected, routes withdraw, and the Node ID remains tombstoned.

##### Functional Requirements

- Enrollment frames must be distinguishable from `MFH4` frames before constructing a normal `link.Session`.
- Enrollment payloads must have strict versions, types, sizes, nonces, time bounds, domain-separated signatures, and transcript binding.
- The client must not send a candidate Node ID; any such field is invalid.
- Pending request creation must allocate no Node ID and must deduplicate by Authority, target parent, and child fingerprint.
- Authority Grant must be idempotent for the same Permit/request and child key.
- Permit issue/consume/revoke and request approve/reject must be durable, concurrent-safe, bounded, and auditable.
- Permit scope must be checked against the actual direct parent identity.
- Authority Node ID allocation must be exclusive to the Authority and backed by a unique durable index plus tombstones.
- Parent local trust activation must require a verified active Authority Grant.
- Management list operations must be paginated and return explicit status, actor, target scope, timestamps, and non-secret metadata.
- Routed management must preserve the original Principal and enforce distinct read/issue/revoke/approve/reject permissions.
- Desktop and CLI must clear one-time Permit material only after the enrollment binding is durable.

##### Non-functional Requirements

- Fail closed on malformed signatures, stale nonce, wrong parent scope, expired/revoked/consumed-by-other Permit, Authority unavailability, and persistence failure.
- Never log private keys, raw Permit contents, or enrollment proof secrets.
- Bounds: payload size, handshake concurrency, pending requests, records, page size, TTL, and retry backoff.
- Existing normal Join and resource traffic must not acquire an Authority round trip.
- Authority storage recovery must converge incomplete grants idempotently before serving new enrollment.
- All state writes use the repository's atomic write/store patterns; no cross-file silent partial success.

##### Inputs / Outputs

Inputs:

- Child endpoint, device public key, client nonce, signed proof, optional Enrollment Permit, optional resume request ID.
- Operator issue/approve/reject/revoke/list commands.
- Parent and Authority configuration.

Outputs:

- Signed parent challenge.
- `pending`, `granted`, or `rejected` enrollment result with stable error codes.
- Authority-signed Enrollment Grant/Receipt.
- Paginated Permit/request/enrollment management views and audit events.
- Persisted client binding containing assigned Node ID, direct-parent identity, Authority identity, receipt, and generation.

##### Edge Cases

- Lost response after committed Permit consumption.
- Concurrent consume attempts against one Permit.
- Same child key enrolling simultaneously through two parents.
- Permit issued for B but presented at C.
- Pending approval expires immediately before approval.
- Authority allocates a generated ID that collides with an existing/tombstoned ID.
- Parent applies Grant but crashes before Join; client retries.
- Client saves Grant but crashes before Join; next startup joins normally.
- Revocation races with Join activation.
- Legacy Join and new Enroll arrive concurrently on the same listener.
- Authority or parent key changes unexpectedly.
- Pending flood uses many source IPs or rotated keys.

##### Acceptance Criteria

- An unregistered client can prepare a stable public identity without any Node ID.
- Permit and approval flows both result in the same durable grant/binding shape.
- Before grant, the client has no formal Node ID and cannot enter the normal node tree.
- A management client connected through A can issue an Authority Permit scoped to B, and a child can consume it only through B.
- One central interface lists active/consumed/revoked/expired Permits, pending/rejected/approved requests, and active/revoked enrollments.
- Concurrent/replayed consumption yields at most one assignment; identical retry returns the same assignment.
- Revoked enrollment cannot rejoin even with a saved receipt.
- Existing identity/trust/`ProvisioningPermitV1` Join tests continue to pass.
- Desktop new Profile no longer asks for local Node ID, parent Node ID, or parent public key.
- Headless enrollment works without a GUI.

##### Risks

- This touches the protocol, auth persistence, listener handshake, node lifecycle, management resources, bindings, CLI, and Desktop state migration.
- The current `auth.Identity` requires a Node ID; device key preparation must be separated without weakening enrolled-runtime invariants.
- Routed enrollment calls during a child handshake can deadlock or exhaust queues if not bounded and isolated from data lanes.
- Revocation propagation is not instant during a network partition; cached Join freshness policy must be explicit.
- Existing legacy Node IDs were manually assigned and are not all present in a central registry. Authority-generated random 63-bit IDs minimize hidden-legacy collision risk; active topology and imported records are still checked.

#### Architecture Design

##### Overall Solution

1. Add `protocol.EnrollmentCodec` and schemas with separate `MFHE` magic. Ordinary `Envelope.Validate` keeps non-zero Node ID invariants.
2. Add `auth.DeviceIdentity`, client `EnrollmentBinding`, and an Authority-owned `EnrollmentState` containing generation, Permits, pending requests, enrollments, revocations, used IDs, and tombstones.
3. Add `runtime/enrollment` for client/server transcript exchange, proof verification, trust bootstrap, retry/resume, first-frame dispatch, and bounded handshakes.
4. Configure each listening parent with one `AdmissionAuthorityID`. If local, call the local Authority service; if remote, invoke the Authority's enrollment Command through the authenticated node tree.
5. Authority returns a signed Grant. Parent persists a verified local grant/trust cache before accepting Join. Client persists its binding before constructing/activating a normal Node identity.
6. Extend management resources and SDK with centralized paginated list/issue/revoke/approve/reject/revoke-enrollment operations.
7. Update Hub/admin/headless CLI and Desktop onboarding/admin UI. Preserve legacy Join until non-Go/embedded clients are migrated later.

##### Alternatives Considered

- Special-case `OperationEnroll` with Source/Target zero: rejected because it weakens ordinary Envelope routing/authentication invariants.
- Parent-local Authority state with later merge: rejected because Permit single-use and Node ID uniqueness require strong consistency.
- Authority call on every Join: rejected because it makes the data plane and reconnect availability depend on the control plane.
- Full root certificate chain/BRSKI: deferred as unnecessary for the direct-parent product model.

##### Module Responsibilities

- `protocol/`: enrollment frame, payloads, management schemas, validation, stable errors.
- `runtime/auth/`: device identity, Authority state machine, Permit/request/enrollment lifecycle, allocator, receipt signing, persistence/recovery.
- `runtime/enrollment/`: pre-auth client/server handshake, transcript signatures, first-frame dispatch, trust bootstrap.
- `runtime/node/`: Authority routing adapter, parent grant cache, transition/adoption into normal Join, revocation/disconnect integration.
- `runtime/link/`: reusable prefixed/read-ahead pipe helper only; normal Session invariants remain unchanged.
- `feature/management/`: Authority Resources, permissions, pagination, audit, revocation orchestration.
- `host/config`, `host/hub`: Authority/local-parent configuration and composition.
- `sdk/go`, `sdk/bindings`: enrollment lifecycle, status, management client contracts, generated manifests.
- `cmd/`: Authority bootstrap/administration and headless enrollment.
- `apps/desktop`: Profile migration, enrollment UI/status, central admission console.
- `docs/`: stable requirement/spec/decision/feature truth and indexes.

##### Data / Call Flow

Permit flow:

`A management UI -> Authority issue -> child config -> B challenge/proof -> B routed consume -> Authority atomic Grant -> B trust cache -> child binding -> normal Join`

Approval flow:

`child -> B challenge/proof -> B routed submit -> Authority pending -> A approves -> Authority Grant -> child retry/resume through B -> binding -> Join`

Revocation flow:

`A -> Authority revoke -> Authority generation/audit -> target parent notification or polling -> local trust disable -> active session disconnect -> route withdrawal`

##### Interface Drafts

- Enrollment frame types: `init`, `challenge`, `proof`, `result`, `error`.
- Result statuses: `pending`, `granted`, `rejected`.
- Stable result/error codes: `authority_unavailable`, `pending`, `rejected`, `permit_invalid`, `permit_expired`, `permit_revoked`, `permit_consumed`, `parent_scope_mismatch`, `proof_invalid`, `conflict`, `rate_limited`, `state_unavailable`.
- Authority operations:
  - `system/admission/status`
  - `system/admission/list-permits`
  - `system/admission/issue`
  - `system/admission/revoke-permit`
  - `system/admission/list-requests`
  - `system/admission/approve`
  - `system/admission/reject`
  - `system/admission/list-enrollments`
  - `system/admission/revoke-enrollment`
  - internal/authorized `system/admission/submit-enrollment`
- Permission points separate read, submit, issue, revoke, approve, and reject.
- `EnrollmentPermitV1` is distinct from legacy `ProvisioningPermitV1`.
- `EnrollmentGrantV1` is proof of assignment, not a bearer authorization and cannot bypass current revocation state.

##### Error Handling and Safety

- Strict JSON decoding and bounded binary framing.
- Nonces and transcript hashes prevent replay/substitution.
- Device fingerprint is derived and checked against the proof public key.
- Active Permit can be revoked; consumed Permit points to its immutable grant for idempotent replay by the same key.
- Pending requests allocate no Node IDs and are bounded/deduplicated.
- Revoked enrollment retains fingerprint and Node ID tombstones.
- Local parent never trusts a child solely because it presents a receipt; it verifies Authority signature and current cached revocation generation.
- Authority-unavailable errors never fall back to parent-local issuance.

##### Performance and Testing Strategy

- Enrollment frames use a lower configurable payload limit than ordinary resource payloads.
- First-frame read, crypto verification, Authority calls, and persistence each have independent deadlines.
- Handshake and pending queues are bounded; list operations are paginated.
- Unit tests cover codecs, signatures, state transitions, durability, concurrency, and migration.
- Memory-transport and TCP integration tests cover root Authority, intermediate parent B, management node A, and child C.
- Targeted race tests cover consume/approve/revoke/Join races.
- Desktop tests cover Profile migration, onboarding branches, failure retry, Permit cleanup, and management lists/actions.
- Full gates: `GOWORK=off go test ./...`, targeted `-race`, binding contract generation/check, Desktop frontend tests/build, Wails production build in heavy test phase.

##### Extensibility Design Points

- Authority storage behind an interface so a future strong-consistency backend can replace the single local writer without changing Permit semantics.
- Node ID allocator behind an Authority-only interface; client and parent protocols never depend on sequential versus random allocation.
- Trust bootstrap mode supports Permit anchor, TOFU, and pinned fingerprint.
- Enrollment Grant includes Authority domain/generation for future reparent or key rollover work.
- Offline Allocation Tickets and federated domain identity remain separately versioned future protocols.

#### Issue List

- No unresolved architecture blocker for the planned first version.
- Execution approval received for the exact `Will Execute` Task IDs.

### Stage 3.1 - Planning

#### Project Goal and Current State

Current vNext has persistent Ed25519 `auth.Identity`, direct-parent trust, signed one-use `ProvisioningPermitV1`, and Join/JoinAck. It lacks an unregistered identity state, pending approval, central listable Authority state, routed Permit consumption, and Authority-assigned Node IDs. Desktop and bindings require client-provided Node ID, parent Node ID, and parent key.

#### Docs Governance Routing Decision

- Docs root: `D:\project\MyFlowHub3\worktrees\node-enrollment-admission\docs`.
- Original request evidence: add `docs/intake/2026-08-30_node-enrollment-central-authority.md`.
- User-visible truth: clarify `docs/features/hub.md` and `docs/features/desktop.md`; embedded/mobile keep legacy behavior explicitly until deferred migration.
- Durable intent: clarify `docs/requirements/auth-controlled-admission.md`.
- Technical contract: add `docs/specs/node-enrollment-and-admission-authority.md`, update operational/node-tree specs and generated protocol map through its generator.
- Architecture rationale: add `docs/decisions/2026-08-30_centralized-admission-authority.md`; it complements rather than supersedes the authoritative-tree ADR.
- Workflow result: `docs/change` only during `$m-archive` after implementation/test.
- Lessons: reuse current authority-routing and Desktop reconnect lessons; create a new lesson only if execution reveals a reusable failure pattern.

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake: `docs/intake/2026-08-29_desktop-first-admission-onboarding.md`
- Features: `docs/features/hub.md`, `docs/features/desktop.md`, `docs/features/embedded.md`, `docs/features/android.md`
- Requirements: `docs/requirements/auth-controlled-admission.md`, `docs/requirements/unified-node-runtime.md`
- Specs: `docs/specs/node-tree-link-resource-architecture.md`, `docs/specs/operational-lifecycle.md`, `docs/specs/protocol_map.md`
- Decisions: `docs/decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md`
- Lessons: `docs/lessons/authority-local-admin-actions.md`, `docs/lessons/desktop-binding-reconnect-and-admission-diagnostics.md`

#### Stable Docs Impact

- Intake impact: add
- Feature impact: clarify
- Requirements impact: clarify
- Specs impact: add and clarify
- Decision impact: add
- Lessons known at planning time: reuse existing; no new lesson yet

#### Executable Task List

##### DOC-ENROLL-1 - Establish stable enrollment truth

- Owner: primary implementation agent
- Worktree: `D:\project\MyFlowHub3\worktrees\node-enrollment-admission`
- Plan Path: `plan.md`
- Goal: update intake, feature, requirement, spec, ADR, and indexes before behavior changes land.
- Files / Modules: `docs/intake`, `docs/features/{hub,desktop,embedded,android}.md`, `docs/requirements/auth-controlled-admission.md`, new enrollment spec/ADR, category indexes.
- Write Set: governed docs only.
- Acceptance: docs state single logical Authority, two enrollment paths, no pre-grant Node ID, central management, compatibility, failure behavior, and deferred federation.
- Test Points: link/index audit; generated regions untouched.
- Rollback: revert the stable-doc commit independently.

##### PROTO-ENROLL-1 - Add bounded enrollment and management contracts

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: define `MFHE` framing, signed payloads, Authority grant/permit/list schemas, operations, and stable errors without weakening `MFH4` Envelope validation.
- Files / Modules: `protocol/codec*`, new `protocol/enrollment*`, `protocol/schema_provisioning.go`, `protocol/schema_management.go`, protocol tests and map generator inputs.
- Write Set: protocol package and generated contract/map outputs only.
- Acceptance: malformed/oversize/unknown frame rejection; signature fields and scopes validate; ordinary Envelope still rejects zero IDs.
- Test Points: codec round-trip, boundary, truncation, replay fields, fuzz/malformed fixtures, protocol map check.
- Rollback: remove new frame/schema versions; legacy MFH4 remains unchanged.

##### AUTH-STATE-1 - Implement device identity and Authority state machine

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: separate key preparation from enrolled identity and implement durable Permit/request/enrollment/revocation/allocation transactions.
- Files / Modules: `runtime/auth`, `internal/keystore` only if required by atomic recovery, focused auth tests.
- Write Set: new versioned state files/migrations and auth APIs.
- Acceptance: pending allocates no ID; one Grant per key/request/Permit; duplicate retry returns same Grant; revoke/tombstone survives restart; collisions retry; concurrent consumption is single-winner.
- Test Points: restart, corruption, migration, expiry, revoke, conflict injection, concurrency/race, no private material in errors.
- Rollback: keep legacy identity/admission loaders and version-gated new state; do not destructively rewrite legacy state without backup.

##### LINK-ENROLL-1 - Add pre-auth enrollment handshake and listener dispatch

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: perform transport-neutral challenge/proof/result exchange and dispatch MFHE versus legacy MFH4 on one listener.
- Files / Modules: new `runtime/enrollment`, minimal `runtime/link` read-ahead helper, `runtime/node/connection.go`, listener/handshake tests.
- Write Set: pre-auth/session boundary only.
- Acceptance: proof binds child key, both nonces, parent identity, target Authority and transcript; legacy Join remains accepted; bounds and timeouts are enforced.
- Test Points: memory Pipe contract, TCP handshake, wrong key/scope/nonce, partial frames, handshake limits, same-pipe handoff/reconnect fallback.
- Rollback: disable MFHE dispatcher and retain legacy MFH4 listener path.

##### NODE-AUTHORITY-1 - Route enrollment to the single logical Authority

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: make direct parent B submit to local or routed Authority, persist verified Grant cache, activate Join, and enforce revocation.
- Files / Modules: `runtime/node`, `runtime/tree` only if required for cleanup, `host/config`, focused node/supervisor tests.
- Write Set: enrollment broker/routing, parent cache, Join activation/revocation integration.
- Acceptance: A-issued/B-scoped Permit works only at B; Authority outage blocks new enrollment but not cached existing Join; revoke races cannot leave an active edge.
- Test Points: A-B-C memory/TCP topology, routed Principal preservation, Authority unreachable, restart, revoke/disconnect/route withdrawal, legacy Join.
- Rollback: configuration gate disables new enrollment and uses existing Join path.

##### MGMT-ADMISSION-1 - Build centralized admission Resources and SDK client

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: expose one paginated and permissioned Authority view for status, Permits, requests, enrollments, approval, and revocation.
- Files / Modules: `feature/management`, `protocol/schema_management.go`, `sdk/go/features*`, `sdk/bindings/contract`, audit tests.
- Write Set: management Resources, schemas, SDK methods, generated binding manifest.
- Acceptance: remote authorized actor can list/issue/approve/revoke; unauthorized actor is denied; actor/target/status are auditable; Permit body/secret is not listed.
- Test Points: permissions, pagination, filters, routed Principal, concurrent changes, revoke/expiry visibility, contract generation check.
- Rollback: remove new Resources while leaving Authority core unavailable to remote management.

##### HOST-CLI-1 - Configure Authority and provide headless administration/enrollment

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: configure root/local versus remote Authority, issue/list/approve/revoke centrally, and enroll a headless node without GUI or manual Node ID/parent key.
- Files / Modules: `host/config`, `host/hub`, `cmd/mfh-hub`, `cmd/mfh-admin`, `cmd/mfh-node`, relevant README/help/tests.
- Write Set: host composition and CLI surface.
- Acceptance: one root bootstrap path; A can administer Authority; child CLI takes endpoint, trust mode, optional Permit only; errors are actionable and secret-safe.
- Test Points: CLI parsing, offline stopped-only guard, remote Authority unavailable, first bootstrap, stdout JSON contracts.
- Rollback: retain legacy CLI flags through deprecation period; new commands are additive.

##### BINDING-COMPAT-1 - Add enrollment lifecycle to Go bindings with legacy migration

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: expose prepare/enroll/status/join APIs, migrate stored identity/binding safely, and preserve current TrustParent/Start APIs for enrolled clients.
- Files / Modules: `sdk/bindings`, `sdk/bindings/desktop`, protected identity interfaces, generated contracts/tests.
- Write Set: binding APIs, credential schemas, migration adapters.
- Acceptance: unregistered key survives retry; binding saves before Join; legacy state opens unchanged; Permit clears only after durable Grant; no private key export.
- Test Points: protected/session-only stores, migration, failed retry, lost result recovery, contract manifest and Wails binding generation.
- Rollback: versioned adapters preserve v1/v2 state and legacy entry points.

##### DESKTOP-ENROLL-1 - Replace manual login fields and add Authority admission console

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: make new Profile enrollment endpoint-first and provide one UI for all Authority Permits, pending requests, approvals, and enrollments.
- Files / Modules: `apps/desktop/{app.go,config.go,credential_*}`, frontend Login/Profile/Settings/new admission components, tests, generated Wails bindings/dist.
- Write Set: Desktop Profile schema migration, host API, React UI, styles and generated production assets.
- Acceptance: new Profile asks for name/endpoint/trust mode/optional Permit only; pending status survives retries; admin view lists and acts on Authority state from any authorized A; legacy Profiles reconnect.
- Test Points: React state/action tests, Go Profile migration, DPAPI/session-only paths, forbidden/loading/empty/error states, production build and packaged Wails smoke in heavy test phase.
- Rollback: Profile migration keeps backup and legacy binding adapter; admin UI is additive and can be hidden by capability discovery.

##### TEST-ENROLL-1 - Run integration, race, build, and security review gates

- Owner: primary implementation agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: validate the complete A-Authority-B-child flows, compatibility, failure modes, and production artifacts.
- Files / Modules: all changed modules; test/evidence artifacts only.
- Write Set: focused missing tests and test evidence; no unrelated fixes.
- Acceptance: all required gates pass or the workflow returns to execution with explicit failures.
- Test Points: `GOWORK=off go test ./...`, targeted `go test -race`, protocol map/binding generation checks, frontend test/build, Wails build/smoke, `git diff --check`, secret scan of persisted/config outputs.
- Rollback: not applicable; failures block archive and identify the owning Task.

#### Execution Scope After Approval

##### Will Execute

- `DOC-ENROLL-1`
- `PROTO-ENROLL-1`
- `AUTH-STATE-1`
- `LINK-ENROLL-1`
- `NODE-AUTHORITY-1`
- `MGMT-ADMISSION-1`
- `HOST-CLI-1`
- `BINDING-COMPAT-1`
- `DESKTOP-ENROLL-1`
- `TEST-ENROLL-1`

##### Will Not Execute Now

- `FEDERATED-AUTH-1`: multi-Authority domain merge, cross-domain Permit exchange, consensus/HA, and composite domain identity are deferred; separate architecture approval is required.
- `OFFLINE-TICKET-1`: Authority-preallocated offline enrollment tickets are deferred; first version fails closed for new enrollment when Authority is unavailable.
- `NATIVE-CLIENT-1`: Android native, Clipboard Flutter bridge, Metrics mobile, C, MicroPython, and ESP32 migration to MFHE are deferred; they retain the legacy Join path in the next phase.
- `REPARENT-REKEY-1`: enrollment transfer across parents/Authorities and parent/Authority key rollover are deferred; unexpected identity change remains fail closed.
- `ARCHIVE-ENROLL-1`: `docs/change`, archived full plan, merge, and cleanup belong to later `$m-archive`, only after execution and tests pass.

#### Dependencies

- `DOC-ENROLL-1` establishes stable truth before implementation commits.
- `PROTO-ENROLL-1` precedes all runtime, management, binding, CLI, and UI work.
- `AUTH-STATE-1` and protocol contracts precede listener/Authority routing.
- `LINK-ENROLL-1` and `NODE-AUTHORITY-1` form the runtime vertical slice.
- `MGMT-ADMISSION-1` depends on Authority state and enables CLI/Desktop administration.
- `BINDING-COMPAT-1` depends on the client handshake and state contracts.
- `HOST-CLI-1` and `DESKTOP-ENROLL-1` depend on management/bindings.
- `TEST-ENROLL-1` closes the gate after all executing Tasks.

#### Risks and Notes

- This is a staged architectural migration, not a small login-form change.
- The implementation must keep legacy Join callable until deferred native/embedded clients migrate.
- Current main checkout has unrelated changes; all work stays in the active worktree.
- Stable docs are local to the feature branch until normal merge; no remote/push/publication is authorized.
- Plan approval authorizes only the `Will Execute` Task IDs, not deferred work, Git push, merge, release, or cleanup.

#### Parallelism Assessment

- No implementation sub-agents were dispatched during planning.
- Protocol and Authority state are shared foundations and should be implemented serially first.
- After their contracts stabilize, management SDK/CLI and Desktop UI can be parallelized only if a later `$m-go` or explicit delegation request authorizes sub-agents and assigns non-overlapping write sets.
- Final integration, generated contracts, migrations, and race review must reconverge under one owner.

#### Issue List

- Blocked: no.
- `$m-execute` completed all approved `Will Execute` Task IDs.
- Implementation sub-agents remain disabled because the user did not invoke `$m-go` or explicitly authorize delegation.

### Execute - Implementation Result

#### Completed Task IDs

- [x] `DOC-ENROLL-1` — stable requirement, feature, specification, ADR, intake, index, and README truth updated.
- [x] `PROTO-ENROLL-1` — bounded `MFHE` enrollment framing and versioned enrollment/admission schemas added without weakening `MFH4` validation.
- [x] `AUTH-STATE-1` — pre-ID device identity plus durable single-writer Permit/request/Enrollment/revocation/Node-ID state machine implemented.
- [x] `LINK-ENROLL-1` — listener prefix dispatch, challenge/proof transcript, timeout, cancellation, and bounded handshake paths implemented.
- [x] `NODE-AUTHORITY-1` — local/remote Authority broker, parent trust activation, ordinary Join transition, and routed revocation implemented.
- [x] `MGMT-ADMISSION-1` — permissioned paginated Authority resources, stable cursors/epochs, lifecycle audit targets/status, and central A-to-B flow implemented.
- [x] `HOST-CLI-1` — local/remote Authority host configuration, offline Permit issuance compatibility, and headless identity/enroll/status/admin operations implemented.
- [x] `BINDING-COMPAT-1` — enrollment lifecycle bindings and generated contract updated while retaining legacy identity/Join entry points.
- [x] `DESKTOP-ENROLL-1` — endpoint-first Authority Profile, explicit TOFU/Permit/Pending login, protected credential persistence, and centralized admission console implemented.
- [x] `TEST-ENROLL-1` — full regression, targeted race, static analysis, generation, frontend, cross-platform compile, packaged build, smoke, and secret-persistence gates passed.

#### Validation Evidence

- `GOWORK=off go test ./...`: passed.
- `GOWORK=off go test -race ./runtime/auth ./runtime/enrollment ./runtime/node ./feature/management ./host/hub ./sdk/bindings`: passed.
- `GOWORK=off go vet ./...`: passed.
- Binding contract generation: passed and idempotent.
- Desktop frontend: 49 tests passed; TypeScript and Vite production build passed.
- Desktop non-Windows credential path: `linux/amd64` cross-compilation passed.
- Wails Windows production package: clean build passed; packaged executable launch smoke passed.
- `git diff --check`: passed after implementation; only repository line-ending notices were emitted.
- Security checks: Permit body/signature are absent from settings, logs, management lists, and audit events; DPAPI stores Enrollment credentials separately on Windows; parent/Authority pinning, request binding, parent-key authentication, bounded state, replay/idempotency, collision, expiry, and tombstone tests passed.

#### Deferred Scope Remains Unchanged

`FEDERATED-AUTH-1`, `OFFLINE-TICKET-1`, `NATIVE-CLIENT-1`, `REPARENT-REKEY-1`, and `ARCHIVE-ENROLL-1` were not executed. No merge, push, release, archive, or worktree cleanup was performed.

### Continue - Heavy Validation Result

#### Iteration Summary

- Iteration: 1 complete validation cycle.
- Recovered state: `validation-needed`; all approved Task IDs were implemented, but `$m-continue` required fresh heavyweight and UI evidence.
- Execute repairs: none. No failing check mapped to an approved Task ID.
- Final state: `converged` / `Passed`; all approved Task IDs and acceptance criteria are satisfied.
- Delegation: no sub-agents were used because host policy required explicit user authorization for delegation; validation remained in the active worktree under the primary agent.

#### Heavy Validation Evidence

- Fresh full regression: `GOWORK=off go test -count=1 ./...` passed.
- Race and static analysis: targeted `go test -race -count=1` and `go vet ./...` passed.
- Reordered/repeated security-state tests: Authority state tests passed three shuffled runs, enrollment transcript/trust tests passed five shuffled runs, central/remote management tests passed three shuffled runs, and protocol Enrollment/Admission validation tests passed five shuffled runs.
- Frontend: all 49 Vitest tests passed; TypeScript/Vite production build passed.
- Generated contracts: `go generate ./sdk/bindings` was idempotent at SHA-256 `0D5931116EC8A0CED71A9DF30482242EA1F3D0F05C4A24D50BEE5E2E256FC1A5`.
- Platform/package: Linux `amd64` Desktop credential paths passed compile-only validation; Wails generated bindings and clean Windows `amd64` production packaging passed.
- Actual packaged UI flow: opened the Wails executable with an isolated config root, created an Authority Profile using only name/endpoint, prepared a DPAPI-protected device key while the client still had no Node ID, exercised explicit TOFU and the actionable connection-failure state, then consumed an Authority-issued Permit through a live local Hub. The Authority assigned the Node ID and returned parent/Authority bindings. Closing and relaunching the packaged application restored the DPAPI credential and reconnected without the Permit.
- Actual Admission UI flow: from the enrolled Desktop client, opened the centralized Admission Console, loaded the approved request, consumed Permit, and active enrollment from Authority Node 1, then issued a new Permit bound to a device fingerprint and target Node 1. The list returned metadata only while the one-time Permit body remained in the issuance result.
- Secret persistence: the consumed Permit ID and signature were absent from every persisted Desktop settings/profile/log/credential file; Windows enrollment credentials remained in a separate DPAPI file.
- UI evidence:
  - `C:\Users\HelloWorld\.codex\visualizations\2026\08\29\01a04eda-1e1a-7813-b788-1632cca5a9d4\node-enrollment-profile-assigned.png`
  - `C:\Users\HelloWorld\.codex\visualizations\2026\08\29\01a04eda-1e1a-7813-b788-1632cca5a9d4\node-enrollment-admission-console.png`

The first Linux cross-platform attempt used `go test`, which compiled and then tried to execute Linux test binaries on Windows. It failed with `not a valid Win32 application`; the correct compile-only check (`go test -c -o NUL`) passed for both Desktop packages. This was a validation-command correction, not a product failure.

#### Mandatory Review Checklist

- [x] 需求覆盖 — Permit direct enrollment, pending approval, post-grant Node ID, central administration, headless flow, compatibility, revocation, and restart behavior are covered.
- [x] 架构合理性 — the direct parent owns the link while the single logical Authority owns security-state writes and allocation; no distributed Permit merge was introduced.
- [x] 性能风险 — bounded payloads/handshakes/pending queues/records and paginated reads prevent unbounded growth; no new N+1 network loop or lock inversion was found.
- [x] 性能指标 / 阈值 — 64 KiB enrollment frames, configured handshake deadlines, page limits, pending quotas, per-parent quotas, record caps, and collision retry limits are explicit and tested. No latency SLO exists for this control-plane path.
- [x] 可用性 / 用户路径 — actual packaged UI creation, identity preparation, failure recovery, Permit enrollment, restart, and central issuance were operated successfully.
- [x] 可读性与一致性 — protocol, auth, enrollment, management, bindings, CLI, and Desktop responsibilities remain separated and use existing repository conventions.
- [x] 可扩展性与配置化 — Authority routing, trust mode, caps, TTLs, pagination, and admission profiles are versioned/configurable; federation remains explicitly deferred.
- [x] 稳定性与安全 — replay/idempotency, expiry, collision, tombstone, corruption, cancellation, DPAPI, and revocation gates passed.
- [x] 安全边界 / 权限 / 数据暴露 — explicit remote permissions, TOFU/pinning, authenticated parent identity, secret-free audit/list/settings persistence, and default deny passed.
- [x] 测试覆盖情况 — fresh full, race, repeated/shuffled focused, frontend, generation, cross-platform compile, package, secret scan, and actual UI evidence passed.
- [x] 整体流程 / 联调验证 — live root Authority / Desktop child enrollment, central list/issue, and restart reconnect passed.
- [x] 子Agent治理与审计 — no delegation occurred; Task mapping, worktree ownership, evidence, and result review stayed with the primary agent.

#### Residual Risks And Rollback

- The first release still has one logical Authority writer and therefore a control-plane availability/capacity concentration. Existing enrolled Join remains available from cached trust, but new enrollment fails closed while the Authority is unavailable.
- Authority persistence rewrites a bounded atomic snapshot on mutation. This is acceptable for the current low-frequency control plane, but the configured 100,000-record ceiling is a capacity guard rather than a demonstrated high-throughput target; a future strongly consistent backend needs separate planning.
- Native/embedded clients, Authority federation/HA, offline allocation tickets, and reparent/rekey remain the explicitly deferred Task IDs listed above.
- Rollback remains task-local: disable MFHE dispatch and Authority enrollment surfaces while preserving legacy MFH4 Join and versioned client/profile state. No destructive migration, merge, push, archive, or worktree cleanup was performed.

#### Continue Decision

- Blocked: no.
- Result: `Passed`.
- Next permitted phase: `$m-archive`.

### Post-Validation Desktop Login UX Refinement

- Reorganized the login surface into three user-facing stages: connect to the parent, prepare or inspect the device identity, and choose an explicit admission method.
- Replaced the implicit “empty Permit means approval” behavior with two visible choices: direct registration with an existing Permit, or an approval request with explicit TOFU confirmation when no trust pin is configured.
- Moved Authority/parent pins and the Legacy compatibility switch behind “高级与兼容连接设置”; ordinary Authority enrollment no longer presents Node ID or public-key inputs.
- Enrolled Authority Profiles no longer show Permit/approval controls and instead present a direct parent reconnect action. Saved pending Profiles are described as retaining a local device identity rather than appearing uninitialized.
- Security and protocol semantics are unchanged: the Authority still allocates Node ID, Permit enrollment still sends the one-time Permit only for the direct path, approval enrollment sends no Permit, and TOFU remains explicit.
- Focused revalidation: all 50 frontend Vitest tests passed, TypeScript/Vite production build passed, final Windows `amd64` Wails packaging passed, and the packaged login window was operated with an isolated config root in both approval and Permit modes.
- Result remains `Passed`; next permitted phase remains `$m-archive`.

### Post-Validation Login Information Architecture Prototype

- Created `design-demos/login-profile-flow.html` to separate the dominant “use an existing Profile” path from first connection.
- The prototype covers registered and pending Profile states plus approval/Permit first-connection branches, keeps Node ID Authority-assigned, and moves trust pins/Legacy controls behind advanced settings.
- Three Playwright checks passed with zero page errors, including the compact 1024×768 Desktop viewport.
- The prototype is design evidence only. The user invoked `$m-archive` before approving production React mapping, so the current validated three-stage React login remains the shipped behavior and the two-tab prototype is explicitly deferred.

### Archive - Documentation And Closeout Preparation

- `$m-docs` confirmed the canonical docs root is this repository's `docs/` tree.
- Intake impact: updated.
- Feature impact: updated.
- Requirements impact: updated.
- Specs impact: updated.
- Decision impact: updated.
- Lessons impact: updated with the Windows Go cross-compile compile-only rule.
- Change archive: `docs/change/2026-08-30_node-enrollment-admission-authority.md`.
- Plan archive: `docs/plan/plan_archive_2026-08-30_node-enrollment-admission-authority.md`.
- Sub-agent trace: none; no delegation occurred.
- Publication: local-only; no remote/push/release authorization was inferred.
