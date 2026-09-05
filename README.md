# MyFlowHub

MyFlowHub is being rebuilt around one authoritative node tree. An authenticated parent-child link is simultaneously the routing edge and the runtime authority edge; each node owns `Variable`, `Stream`, and `Command` resources, while `Subscription` is a first-class relation over variables and streams.

This repository is the canonical monorepo. It intentionally does not preserve the former internal Go module paths, SubProto dispatcher API, or wire format.

The checkout is kept below the workspace's `repo/MyFlowHub` directory. Additional Git worktrees belong in the workspace-level sibling `worktrees/` directory, never inside this checkout.

## Core layout

- `protocol/`: versioned transport-neutral envelope and codec
- `runtime/link/`: byte-stream drivers and authenticated link sessions
- `runtime/tree/`: single-parent topology, routes, and topology epochs
- `runtime/auth/`: identity, admission, source proof, and policy hooks
- `runtime/resource/`: node-owned variable, stream, and command definitions
- `runtime/subscription/`: leases, aggregation, and bounded delivery
- `runtime/command/`: invocation, deduplication, timeout, and results
- `runtime/node/`: runtime composition and routing
- `transport/`: concrete byte-stream drivers
- `host/`, `sdk/`, and `cmd/`: composition roots and public entry points
- `apps/`: canonical Desktop, Windows Metrics and Windows Clipboard products (Clipboard also has a Web UI preview)

Android and embedded implementations are retired pending [redesign](docs/requirements/mobile-embedded-redesign.md). Ordinary Desktop/Metrics/Clipboard runtimes use NodeHost with an attached operation-only SDK; old runtime-owning SDK/binding entry points are removed.

## Local validation

```powershell
./scripts/mfh.ps1 -Action list
./scripts/mfh.ps1 -Action test -Target core
./scripts/mfh.ps1 -Action check -Target all -AllowUnavailable
```

The accepted architecture is indexed from [docs/README.md](docs/README.md), and the build contract is documented in [docs/specs/build-and-ci.md](docs/specs/build-and-ci.md). Exact legacy source baselines and dispositions are recorded under `migration/`.

## Local development

```powershell
./scripts/run-dev.ps1 -DryRun
./scripts/run-dev.ps1 -WaitHub
```

The script launches only canonical `cmd/mfh-hub`, `apps/desktop`, and `apps/nodes/metrics/windows` targets with `GOWORK=off`. It stores development state and logs below `.tmp/dev-state` unless `-StateRoot` is supplied. Background windows are hidden by default; use `-VisibleWindows` when an interactive console is useful.

Hub authorization is default-deny. A new device first prepares a durable key without a Node ID. Stop the Authority Hub before offline Permit issuance, then enroll through the target parent; the Authority returns the Node ID in its Grant:

```powershell
go run ./cmd/mfh-hub -state .tmp/dev-state/hub -identity
go run ./cmd/mfh-admin -state .tmp/dev-state/device -op identity
go run ./cmd/mfh-hub -state .tmp/dev-state/hub -issue-public-key '<device-raw-base64-ed25519-key>' -issue-target-id 1 -issue-role device > .tmp/device-permit.json
go run ./cmd/mfh-admin -state .tmp/dev-state/device -endpoint 127.0.0.1:7331 -op enroll -permit .tmp/device-permit.json
```

After enrollment, grant only the required resources to the assigned Node ID with `-policy grant`; the rule can be removed with `revoke`. `-issue-node-id` remains available only for Legacy Join. Online administration uses the authenticated management Commands documented in [docs/features/hub.md](docs/features/hub.md); offline mutation must never run concurrently with the Hub.
