# Todo - Scoped Policy Definitions And Authority Bindings

## Deferred Product Work

- [ ] ANDROID-REDESIGN — [重新设计 Android 节点](docs/requirements/mobile-embedded-redesign.md)；现有应用与移动绑定已移除。
- [ ] EMBEDDED-REDESIGN — [重新定义受限硬件实现](docs/requirements/mobile-embedded-redesign.md)；C/ESP32/MicroPython 旧实现已移除。

- [ ] FLOW-REDESIGN — [重新讨论通用自动化模型与语言](docs/requirements/flow-redesign.md)；旧 Flow 已移除，尚未选择替代方案。

## Approval Gate

- [x] User explicitly approves `DOC01, PROTO01, AUTH01, TREE01, MGMT01, SDK01, DESK01, BOOT01, QA01, LIVE01`.
- [x] User selects `$m-execute` after approval.
- [x] Dedicated branch/worktree exists from `master@70add9b`.
- [x] Main checkout user changes are identified and untouched.
- [x] `$m-discuss` brief and `$m-docs` routing/stable-doc impact are recorded.

## Approved Execution

- [x] DOC01 — stable requirement/spec/ADR and product documentation.
- [x] PROTO01 — bounded policy protocol schemas/selectors/capabilities.
- [x] AUTH01 — policy state v2, migration, Definitions/Bindings and evaluator.
- [x] TREE01 — authoritative topology scope resolver and Hub attachment.
- [x] MGMT01 — policy Collections, superadmin mutation guard and audit.
- [x] SDK01 — typed SDK helpers and generated contracts.
- [x] DESK01 — Authority-backed Desktop Policy console.
- [x] BOOT01 — stopped-Hub Binding inspection/bootstrap CLI.
- [x] QA01 — automated, race, generated, build and isolated integration gates.
- [x] LIVE01 — persist Node 41 Authority-domain superadmin Binding and real restart smoke.
- [x] ARC01 — governed archive, local master integration and worktree/branch cleanup.

## Will Not Execute Now

- [ ] DENY02 — explicit deny, inheritance and Subject groups; deferred.
- [ ] DELEG02 — bounded non-superadmin delegation; deferred.
- [ ] FED02 — cross-Authority policy federation; out of scope.
- [ ] MOBL02 — Android/Embedded policy management UI; separate product scope.
- [ ] CLEAN02 — remove Node 41 legacy exact grants; deferred for rollback safety.
- [ ] PUB01 — push/release/sign/publish/deploy; unauthorized.

## Dependency Order

- [x] DOC01 freezes stable terminology and boundaries.
- [x] PROTO01 freezes wire/schema contracts.
- [x] AUTH01 consumes PROTO01.
- [x] TREE01 attaches current-topology matching to AUTH01.
- [x] MGMT01 consumes PROTO01 + AUTH01 + TREE01.
- [x] SDK01 consumes protocol and management Resource contracts.
- [x] DESK01 consumes SDK/generated schemas and management Resources.
- [x] BOOT01 consumes AUTH01 and can complete before Desktop.
- [x] QA01 follows all implementation tasks.
- [x] LIVE01 runs only after QA01 passes.

## Current Status

- Phase: `$m-archive` complete; local master integrated, user dirt restored, feature worktree/branch removed.
- Blocked: no.
- Runtime/business logic changes: policy state v2, scoped Definition/Binding evaluation, topology resolver, management Collections, SDK, Desktop console and bootstrap CLI implemented.
- Live policy mutation: Subject 41 has persistent Binding `b3df4b135fdf4f3589f9c66aece5ce6a` to `superadmin` over `authority-domain:1`; policy generation 77; 70 legacy exact grants retained.
- Docs: requirement, specification, ADR, feature docs, protocol map and indexes updated.
- Implementation agents dispatched: none.
- Publication: local-only; repository has no remote and no push/release/publication was performed.

## Execution Evidence

- `GOWORK=off go test ./... -count=1` passed.
- `go vet ./...` passed.
- Focused race tests across protocol/auth/tree/management/Hub/SDK passed.
- Generated SDK binding freshness check passed.
- Desktop frontend passed 18 files / 136 tests and production build.
- Desktop Windows Wails production build passed.
- Isolated current/future Metrics Node scoped-binding integration passed.
- Live Hub + Metrics + Desktop restart passed: existing Desktop Profile auto-connected without Permit, Metrics read/subscribe succeeded, effective evaluation identified the Binding, unauthorized Subject remained denied, and a second stopped-Hub inspection confirmed persistence.
