# Wails Bindings Cross-Project Drift

## Summary
- MetricsNode Windows may fail frontend type-checking when `windows/frontend/wailsjs/**` is replaced or polluted by bindings from another Wails app.

## Lookup Hints
- Symptoms:
  - `TS2305` missing export `BootstrapGet`
  - `TS2305` missing export `MetricsSettingsGet`
  - `TS2305` missing export `StartReporting`
- Keywords:
  - `AboutState`
  - `FlowProjectsState`
  - `SaveHomeState`
  - `App.d.ts`
  - `wailsjs`
- Quick checks:
  - open `windows/frontend/wailsjs/go/main/App.d.ts`
  - confirm it exports `BootstrapGet`, `Status`, `MetricsSettingsGet`
  - if it exposes `AboutState` or `FlowProjectsState`, rerun `scripts/build-windows.ps1`

## Symptoms
- `npm run build` fails inside `windows/frontend`.
- The missing exports are methods that definitely exist in `windows/app.go`.
- `App.d.ts` lists a different app surface than the one consumed by `windows/frontend/src/App.vue`.

## Impact
- Frontend build and Wails packaging are blocked.
- The failure looks like a Vue/TypeScript regression even though the real issue is generated artifacts drift.

## Trigger Conditions
- A working copy contains stale `wailsjs` files from another Wails application.
- Someone debugs the frontend directly against `wailsjs` output without first validating the generated bindings.
- The recovery flow skips the repo-local clean-and-generate script.

## Root Cause
- `windows/frontend/wailsjs/**` is generated output, but local drift can still overwrite it with a foreign binding surface.
- When `App.d.ts` no longer reflects the MetricsNode Go `App`, `vue-tsc` fails on imports that are actually valid for this repo.

## Investigation Trail
- Compare `windows/frontend/src/App.vue` imports with `windows/frontend/wailsjs/go/main/App.d.ts`.
- Compare both with the bound methods in `windows/app.go` and `windows/main.go`.
- Validate the same repo from a clean worktree.
- Run `scripts/build-windows.ps1` to clean bindings, regenerate with Wails, and rebuild.

## Resolution
- Use a clean worktree as the execution environment.
- Run `powershell -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1`.
- Confirm the regenerated `App.d.ts` contains MetricsNode exports such as `BootstrapGet`, `Status`, and `MetricsSettingsGet`.
- Keep the frontend code unchanged unless the clean worktree proves the backend binding contract actually changed.

## Prevention / Guardrails
- Prefer `scripts/build-windows.ps1` over ad hoc generation commands when validating the Windows build.
- Fail fast if `App.d.ts` shows foreign exports such as `AboutState`, `FlowProjectsState`, or `SaveHomeState`.
- Treat `TS2305` on known Go exports as a binding-surface check first, not a UI rewrite trigger.
- Keep binding validation in the dedicated worktree; do not silently overwrite dirty control-plane generated files mid-investigation.

## Related Docs
- [../change/2026-03-26_metricsnode-wails-bindings-sync.md](../change/2026-03-26_metricsnode-wails-bindings-sync.md)
- [../change/2026-03-03_metricsnode-settings-ui.md](../change/2026-03-03_metricsnode-settings-ui.md)
