# Todo - Desktop Explorer Node/Resource 上下分区

## Execution Status

- [x] EX01 — Node-only authority tree plus grouped current-Node Resource list
- [x] EX02 — bounded accessible vertical splitter, per-Profile persistence, and sidebar scroll containment
- [x] DOC01 — focused intake plus Desktop feature/requirement/spec clarification
- [x] QA01 — automated/performance/build checks and packaged Wails GUI acceptance
- [x] ARC01 — archive, rebase onto current master, fast-forward integration, post-merge validation, and workflow cleanup
  - [x] change record, plan snapshot, GUI evidence, and indexes
  - [x] implementation and archive commits use Chinese messages
  - [x] fast-forward `master` and remove this worktree/branch after overlapping main-checkout edits are safely resolved

## Will Not Execute Now

- [ ] PERM01 — effective permission/status discovery requires a separate backend-informed contract
- [ ] LAZY01 — server pagination/lazy loading and true DOM windowing remain deferred protocol work
- [ ] BRAND01 — icon work belongs to the separate brand task
- [ ] PUB01 — push/release/publication is not authorized and no remote is configured

## Gate

- Blocked: no
- Approved: EX01, EX02, DOC01, QA01, ARC01
- Completed stage: `$m-execute`
- Completed stage: `$m-test` (entered from `$m-archive` gate audit)
- Completed stage: `$m-archive`
- ARC01 is complete; the dedicated worktree/branch are removed after this finalized archive commit
