# powershell-args-automatic-variable

## Summary

PowerShell function parameters should not be named `$Args`. `$Args` is also an automatic variable, and binding a declared parameter with that name can silently produce an empty value under normal function invocation.

## Lookup Hints

- `param([string[]]$Args)`
- `$Args`
- `ValueFromRemainingArguments`
- forwarded arguments
- `--listen`
- `--mcp-path`
- `EnsureRunning`
- `start-myflowhub-mcp.ps1`

## Symptoms

- A wrapper script receives remaining CLI arguments correctly, but helper functions cannot read them.
- Explicit `--listen 127.0.0.1:<port>` is ignored and the script falls back to `127.0.0.1:17688`.
- Validation based on forwarded flags does not run, for example `--transport stdio` is not rejected.

## Impact

- Launcher logic can target the wrong endpoint.
- Smoke tests can accidentally start a server on the default port instead of the intended temporary port.
- Port reuse and conflict checks become misleading.

## Trigger Conditions

- A PowerShell helper uses a parameter like `param([string[]]$Args)`.
- The caller invokes the helper as `Get-ForwardArgValue -Args $ForwardArgs`.
- The helper relies on `.Count`, indexing, or `-join` over that parameter.

## Root Cause

`$Args` is a built-in automatic variable in PowerShell. Using the same name for a function parameter creates ambiguous behavior and can leave the helper reading the automatic variable rather than the intended passed collection.

## Investigation Trail

- `start-myflowhub-mcp.ps1 -EnsureRunning --listen 127.0.0.1:17889` started `127.0.0.1:17688`.
- Debug output confirmed the outer `ForwardArgs` contained `--listen`.
- A minimal function repro showed `param([string[]]$Args)` returned count `0`, while `param([string[]]$Items)` returned the expected values.

## Resolution

- Rename helper parameters from `$Args` to neutral names such as `$Items`.
- Re-run smoke with a temporary port and confirm the URL uses the requested listen address.
- Keep negative tests for flag validation, especially `--transport stdio` in HTTP-only ensure mode.

## Prevention / Guardrails

- Avoid PowerShell automatic variable names for function parameters, especially `$Args`, `$Input`, `$Host`, `$PSItem`, and `$Error`.
- When adding argument-forwarding helpers, test at least one non-default flag override.
- Prefer endpoint-level smoke tests over process-name assumptions.

## Related Docs

- [docs/change/2026-05-28_win-mcp-ensure-running.md](../change/2026-05-28_win-mcp-ensure-running.md)
- [frontend-and-powershell-preflight.md](frontend-and-powershell-preflight.md)
