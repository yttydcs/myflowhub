# Todo - MyFlowHub Monorepo 第一阶段

## Context

- Branch: `codex/refactor-monorepo`
- Worktree: `D:/project/MyFlowHub3/worktrees/monorepo-rewrite`
- Base: `master@6ffd5d7`
- Detailed plan: [plan.md](plan.md)
- Status: M00-M09 archived; control-plane closeout delegated to `$m-archive`

## Approved Execution

- [x] M00 — Baseline manifest and accepted docs handoff
- [x] M01 — Monorepo root module and dependency guardrails
- [x] M02 — vNext protocol envelope and codec
- [x] M03 — Pipe, Transport and LinkSession foundation
- [x] M04 — Authoritative tree, identity and adjudication
- [x] M05 — Resource Registry, Variable, Stream and Command definitions
- [x] M06 — Subscription lifecycle and bounded delivery
- [x] M07 — Command invocation and result lifecycle
- [x] M08 — Node runtime, minimal Hub and Go client
- [x] M09 — Cross-subtree vertical slice and focused validation

## Will Not Execute Now

- [ ] D01 — Real QUIC/RFCOMM/serial/platform Transport migration; deferred until LinkSession stabilizes
- [ ] D02 — File/Flow and remaining legacy feature migration; deferred until core primitives stabilize
- [ ] D03 — Desktop/Android migration; separate product wave
- [ ] D04 — MetricsNode/ClipboardNode migration; separate node-app wave and dirty binding reconciliation
- [ ] D05 — Embedded C/MicroPython migration; wait for vNext schema stability
- [ ] D06 — Legacy wire/SubProto bridge; not planned without an actual first-party migration blocker
- [ ] D07 — Remote/CI/release/archive/merge/cleanup; separate validation and archive phase

## Gate

- Blocked: no
- Approved: M00-M09 via explicit `$m-execute` invocation on 2026-08-27
- Execute only in the dedicated worktree
- Do not dispatch implementation sub-agents

## Validation

- [x] focused repaired packages, 100 repetitions
- [x] `GOWORK=off go test ./... -shuffle=on -count=20 -timeout=8m`
- [x] `GOWORK=off go test -race ./runtime/... ./tests/integration/... -count=10 -timeout=8m`
- [x] memory/TCP cross-subtree integration, 100 repetitions
- [x] protocol codec fuzz, 15 seconds / 3,527,777 executions
- [x] `GOWORK=off go vet ./...`
- [x] architecture/import/module/document/migration guards
- [x] `gofmt` scan
- [x] `git diff --check`

## Continue Result

- Passed: yes
- Repair iterations: 1
- Blocking issues: none
- Ready for explicit `$m-archive`: yes

## Archive

- [x] stable-doc impact and docs-root routing checked with `$m-docs`
- [x] `docs/change/2026-08-27_canonical-monorepo-unified-node-runtime.md`
- [x] `docs/lessons/session-replacement-generation-cleanup.md`
- [x] change, plan, and lessons indexes updated
- [x] stable intake/requirements/specs/decisions cross-linked
- [x] local commit/merge/cleanup authorized and delegated to the current `$m-archive` closeout
- [x] remote push/publication excluded because it was not authorized

## Phase Boundary

- No `docs/change` archive yet; `$m-continue` does not perform archive work.
- No commit, merge, push, old-repository archive, release, or worktree cleanup.
- D01-D07 remain deferred.
