# Todo - MyFlowHub vNext 全量迁移与最终切换

## Context

- Branch: `refactor/vnext-full-migration`
- Worktree: `D:/project/MyFlowHub3/worktrees/vnext-full-migration`
- Base: `master@87cd7b6`
- Detailed Plan: [plan.md](plan.md)
- Stage: `4.0 archive complete; local merge and worktree cleanup authorized by $m-archive`
- Implementation authorized: yes — explicit `$m-execute`

## Approved Execution Scope

- [x] FM00 — Freeze sources and build migration inventory
- [x] FM01 — Establish stable product and migration contracts
- [x] FM02 — Extend protocol schemas and built-in resource catalog
- [x] FM03 — Durable identity, admission, trust and policy
- [x] FM04 — Connection supervision, reconnect and subscription recovery
- [x] FM05 — Migrate QUIC and RFCOMM drivers
- [x] FM06 — Build production Hub and management resources
- [x] FM07 — Replace remaining SubProto features
- [x] FM08 — Complete Go SDK and platform binding contracts
- [x] FM09 — Migrate MetricsNode entire product source
- [x] FM10 — Migrate ClipboardNode entire product source
- [x] FM11 — Migrate Desktop application
- [x] FM12 — Migrate Android application
- [x] FM13 — Migrate Embedded C, MicroPython and ESP32
- [x] FM14 — Unify builds, generation, packaging and CI definitions
- [x] FM15 — Run cross-product security, integration and performance gates
- [x] FM16 — Eliminate legacy dependencies and switch local canonical entry

## Will Not Execute Now

- [ ] DX01 — New serial/USB/WebSocket transports; new feature, no migration source or product requirement
- [ ] DX02 — Legacy compatibility bridge; explicitly rejected
- [ ] DX03 — Remote push/release/sign/publish/remote archive; separately authorized external state
- [ ] DX04b — Delete/move main checkout dirt; not authorized and unrelated user changes must remain
- [ ] DX05 — Certify unavailable signed platforms/hardware; requires external hosts/devices/credentials

## Separately Authorized Closeout

- [x] DX04a — Remove the 10 local legacy `repo/MyFlowHub-*` checkouts after documentation extraction; recovery metadata retained in `migration/`

## Gate

- Blocked: no
- FM00–FM16 execution complete; every completed gate remains buildable
- Current gate: archive complete — `$m-archive` local merge/cleanup closeout in progress
- Do not dispatch implementation sub-agents
- No commit, merge, push, publish, remote archive, main-dirt deletion or worktree cleanup is authorized; legacy-repo deletion was separately authorized and completed after `$m-execute`
