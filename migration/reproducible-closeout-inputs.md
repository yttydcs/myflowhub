# Reproducible closeout input audit

## Source

- Control checkout: `D:/project/MyFlowHub3/repo/MyFlowHub`
- Base commit: `078f4b75ee8316b0baaed67c4be4d6f7fbf301d9`
- Captured on: `2026-08-28`
- Destination worktree: `D:/project/MyFlowHub3/worktrees/reproducible-closeout`

The control checkout is intentionally treated as read-only evidence. This workflow does not stage, revert, move, or delete its existing changes.

## Selected Inputs

### Repository reproducibility

- `.gitignore`
- `.gitattributes`
- `build/toolchain.json`
- `apps/desktop/frontend/package.json.md5`
- `internal/archtest/architecture_test.go`
- `runtime/link/session_test.go`
- `sdk/bindings/generated/contracts.json` line-ending normalization only

### Repository and documentation routing

- `README.md`
- `repos.md`
- 12 tracked `docs/**/*.md` changes returned by `git diff --name-only -- docs`
- 82 untracked `docs/**/*.md` files returned by `git ls-files --others --exclude-standard -- docs`
- The workflow-created intake, plan, todo, change archive and reproducibility lesson in the active worktree

The document import is constrained to Markdown paths below `docs/`. A source path outside `docs/`, a non-Markdown candidate, or a missing source file is an explicit error.

## Excluded Inputs

- `论文/**`, including Markdown, Word, PDF, HTML, images, archives, and tracked deletions
- Workspace `附件/**`, local Android SDK files, and other project-root assets
- The dirty control-checkout `guide.md`; it contains machine-local operational information and credentials that must not be committed
- Any source outside the exact RC02/RC03 write sets in `plan.md`
- Push, release, signing, publishing, remote archive, stores, and external hardware certification

## Verification

- Compare the selected path set with `git status --short` before import.
- Require every selected docs candidate to resolve below `docs/` and end in `.md`.
- Run a repository-relative Markdown link checker and category-index coverage check after import.
- Scan the branch diff for excluded path prefixes and known local-secret markers before commit.
- Prove independence from the control checkout in a separate clean validation worktree created from the branch commit.

## Rollback

The import is isolated on `refactor/reproducible-closeout`. Reverting the task commits removes the selected inputs without changing the control checkout or excluded user assets.
