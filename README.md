# MyFlowHub

MyFlowHub is being rebuilt around one authoritative node tree. An authenticated parent-child link is simultaneously the routing edge and the runtime authority edge; each node owns `Variable`, `Stream`, and `Command` resources, while `Subscription` is a first-class relation over variables and streams.

This repository is the canonical monorepo. It intentionally does not preserve the former internal Go module paths, SubProto dispatcher API, or wire format.

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

## Local validation

```sh
go test ./... -count=1
go vet ./...
```

The accepted architecture is indexed from [docs/README.md](docs/README.md). Exact legacy source baselines and deferred migration targets are recorded in [migration/sources.yaml](migration/sources.yaml).
