# Todo - Desktop Explorer Node/Resource 上下分区

## Execution Status

- [x] EX01 — Node-only authority tree plus grouped current-Node Resource list
- [x] EX02 — bounded accessible vertical splitter, per-Profile persistence, and sidebar scroll containment
- [x] DOC01 — focused intake plus Desktop feature/requirement/spec clarification
- [x] QA01 — automated/performance/build checks and packaged Wails GUI acceptance
- [ ] ARC01 — change/plan archive and Chinese commits complete; local integration/worktree cleanup blocked by protected dirty-main overlap
  - [x] change record, plan snapshot, GUI evidence, and indexes
  - [x] implementation and archive commits use Chinese messages
  - [ ] fast-forward `master` and remove this worktree/branch after overlapping main-checkout edits are safely resolved

## Will Not Execute Now

- [ ] PERM01 — effective permission/status discovery requires a separate backend-informed contract
- [ ] LAZY01 — server pagination/lazy loading and true DOM windowing remain deferred protocol work
- [ ] BRAND01 — icon work belongs to the separate brand task
- [ ] PUB01 — push/release/publication is not authorized and no remote is configured

## Gate

- Blocked: yes — main checkout has protected uncommitted `style.css`/`dist` paths that overlap the branch write set
- Approved: EX01, EX02, DOC01, QA01, ARC01
- Completed stage: `$m-execute`
- Completed stage: `$m-test` (entered from `$m-archive` gate audit)
- Active stage: `$m-archive`
- `$m-archive` documentation and commits are complete; ARC01 remains open only for safe local integration and cleanup
