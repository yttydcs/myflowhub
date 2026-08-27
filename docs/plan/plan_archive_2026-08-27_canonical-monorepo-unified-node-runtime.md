# Plan - MyFlowHub Canonical Monorepo 与统一节点运行时第一阶段

## Workflow Information

- Repo: `D:/project/MyFlowHub3`
- Branch: `codex/refactor-monorepo`
- Base: `master` at `6ffd5d7`
- Project Root: `D:/project/MyFlowHub3`
- Docs Root: `D:/project/MyFlowHub3/docs`
- Code Repo: `D:/project/MyFlowHub3/worktrees/monorepo-rewrite`
- Legacy Source Repos: `D:/project/MyFlowHub3/repo/MyFlowHub-*`（只读输入，不是本轮写入仓库）
- Worktree: `D:/project/MyFlowHub3/worktrees/monorepo-rewrite`
- Current Stage: `4 archive complete; control-plane closeout delegated`
- Owner: main agent
- External Compatibility: none；允许破坏旧 module path、SubProto API 与 wire 协议

## Stage Records

### Initialization

- `guide.md`: 已读取；worktree 必须位于 `D:/project/MyFlowHub3/worktrees`，本 worktree 符合要求。
- Project/docs/code repo confirmation:
  - `project_root`: `D:/project/MyFlowHub3`
  - canonical private `docs_root`: `D:/project/MyFlowHub3/docs`
  - active code repo: 根 Git 仓的专用 worktree
  - legacy repos 仅作为已记录 commit 的源码参考，不在原仓直接编辑
- Base/worktree confirmation:
  - branch `codex/refactor-monorepo`
  - base `master@6ffd5d7`
  - active worktree clean at initialization
- Root main checkout and `MyFlowHub-MetricsNode` contain pre-existing uncommitted changes; they must not be overwritten, staged, reset, cleaned, or silently imported.

### Legacy Source Baselines

| Repository | Branch | Source commit | Working tree policy |
| --- | --- | --- | --- |
| MyFlowHub-Android | `main` | `93d2032530c97246cfcb5fc996a29c0f357b9ca9` | read-only committed baseline |
| MyFlowHub-ClipboardNode | `master` | `eec7d8acbacbecf4976e79ac4bb0e3a4b3957aca` | read-only committed baseline |
| MyFlowHub-Core | `master` | `82061f18e9d27e00b1ba4ac369ebecf3fb9afc22` | reusable transport/link source |
| MyFlowHub-EmbeddedSDK | `main` | `343d5b6eef9aeb78eb55b6662c0d4d5cf1b42c73` | deferred read-only baseline |
| MyFlowHub-MetricsNode | `main` | `c5e2f217afc8063d3e2bb1c6b21fb8fc878925c0` | import HEAD only later; preserve 16 dirty generated binding paths separately |
| MyFlowHub-Proto | `main` | `93ee3c823df43f6895c0a6862723e7f06d5defd7` | semantic reference; old SubProto map is not new core |
| MyFlowHub-SDK | `main` | `7c5b4f68aa5b2f612e22b0c6c0602ab39316b3ca` | client behavior reference |
| MyFlowHub-Server | `main` | `576a5af270337f15002d3acdd3e2978aab9f7047` | auth/runtime behavior reference |
| MyFlowHub-SubProto | `main` | `6c3b7aa0641254600ef53d57cbe642212b8dec03` | behavior/test reference; do not copy module architecture |
| MyFlowHub-Win | `main` | `606b747e8ecdeed368cf518f3629ac0b75c8ff60` | deferred read-only baseline |

### Discuss - Discovery And Requirements Shaping

#### Goal

重新建立 MyFlowHub 的核心模型：连接、路由和运行期 authority 使用同一棵节点树；节点拥有 Variable、Stream 和 Command；Subscription 是一级运行时关系；具体链路通过可插拔 Transport 接入。

#### Scope

- 将全部第一方源码逐步收敛到一个 canonical monorepo。
- 第一阶段只建立新 monorepo 基础、新协议和新运行时的最小纵向闭环。
- 旧仓保持可运行和只读，不在第一阶段切换现有应用。

#### Assumptions

- 当前不存在外部用户或第三方依赖方。
- 可以破坏旧 import path、module path、SubProto action 和 wire envelope。
- 旧仓历史和架构文档必须保留，但无需将全部 Git history 拼接进新仓。
- 第一阶段使用内存与 TCP Transport 验证抽象；真实 RFCOMM/QUIC 在后续迁移。

#### Open Questions

- 最终公开远端、默认分支和发布命名在本地第一阶段完成后再决定，不阻塞实现。
- Embedded wire payload 的最终 schema/codegen 形式在 EmbeddedSDK 迁移前确认；第一阶段保持 payload 不透明并固定 envelope contract。

#### Options Considered

- 保留多仓原地重构：拒绝，继续承受跨仓和多 module 发布链。
- 只合并运行时、应用继续分仓：可行但不采用，第一方协议仍需跨仓同步。
- 全量第一方 monorepo：采用，以 package 和依赖测试维护边界。

#### Rejected Options

- 新内核继续以 SubProto handler 为顶层扩展模型。
- 为 Variable、Stream、Command、Subscription 分别建立 module。
- 通过长期 legacy adapter 同时维护两套运行时。
- 在当前 dirty 根 checkout 或 MetricsNode 工作树直接搬迁。

#### Recommended Direction

在专用根仓 worktree 中建立一个根 Go module，以新 package 直接实现 protocol、link、tree/auth、resource、subscription、command、node/host/client，并通过内存与 TCP 完成跨子树端到端验证。旧仓只提供经过选择的实现经验和测试用例。

#### Research Summary

- 未使用外部研究；决策基于当前仓库结构、依赖图、历史发布经验和已确认需求。
- 已知经验：[cross-repo-semver-release](D:/project/MyFlowHub3/docs/lessons/cross-repo-semver-release.md) 证明 `go.work` 会掩盖多仓真实发布链问题，进一步支持单一根 module。

#### Worktree / Branch / Docs Root Status

- Dedicated worktree: ready
- Branch: ready
- Docs root: identified and governed
- Legacy source repos: identified; no write authorization required because they are read-only inputs

#### Issue List

- None blocking planning.

## Plan - Requirements And Architecture

### Discussion Summary

采用全量第一方 monorepo 和 clean-break 策略。第一阶段不迁移现有产品，而是先证明新核心在真实 TCP 和内存 Transport 下可以完成节点加入、跨子树订阅、指令调用、权限裁决、断连与 reparent。

### Accepted / Rejected Requirements

Accepted:

- 权限树、协议连接树和节点路由树使用同一父子关系。
- 每个非根节点只有一个当前有效 authority parent。
- Variable 和 Stream 可订阅，Command 可调用。
- Transport 可插拔且不能污染上层资源语义。
- 新核心不受旧 SubProto 和 module path 兼容约束。
- 队列、重组、订阅和 pending command 必须有界。

Rejected:

- 同一 authority 域内多父控制。
- 独立资源树或独立权限树。
- 第一阶段实现所有旧功能和所有平台 Transport。
- 无限制队列、隐式 fallback 或错误吞掉。

### Requirements Analysis

#### Goal

在新 monorepo 内提供可测试、可运行的新节点内核纵向闭环，为后续应用和 EmbeddedSDK 迁移建立唯一基础。

#### Scope

第一阶段包括：

- 根 Go module 和依赖边界；
- versioned envelope、ID、资源寻址和 codec；
- Pipe/Transport/LinkSession；
- 单父节点树、认证、请求链、控制链与 LCA；
- Resource Registry、Variable、Stream、Command；
- Subscription 租约、聚合和有界背压；
- Command 请求、结果、拒绝、超时和去重；
- 最小 Hub、Node 与 Go Client；
- 内存与 TCP 纵向集成测试。

第一阶段不包括真实产品迁移、旧协议兼容、远端发布和旧仓归档。

#### Use Cases

1. 三层或更深节点通过不同 LinkSession 加入同一权威树。
2. 一个叶子节点跨子树订阅 Variable，并收到快照和后续修订。
3. 一个叶子节点跨子树订阅 Stream，并在队列溢出时收到显式 gap/overflow 结果。
4. 一个节点跨子树调用 Command，由 LCA 裁决后下发并返回结果。
5. 父节点直接控制子树资源，子节点验证当前父会话和 control phase 后执行。
6. 节点断连或 reparent 后，旧路由、旧控制和旧订阅不再有效。

#### Functional Requirements

- 只有完成身份认证和角色激活的 LinkSession 才能注册树边。
- 子连接只能代表自身及已证明后代；来自子连接的伪造 SourceID 必须拒绝。
- `parent-origin` 与 `control phase` 必须同时成立，才能跳过重复权限裁决。
- 目标不在下游时请求上送；到达覆盖双方的 LCA 后裁决并向下转换为 control。
- Variable 维护单调 revision；订阅返回 snapshot 后再传 changes。
- Stream 维护 sequence；溢出产生可观察 gap，而不是静默丢失。
- Command 使用 message/correlation ID，支持 timeout、duplicate detection 和结构化 error。
- 断连清理后代 route、link-bound subscriptions 和 pending requests。
- reparent 增加 topology epoch 并拒绝旧父延迟消息。

#### Non-functional Requirements

- 单一根 module，禁止 nested `.git` 和无理由 nested `go.mod`。
- 依赖方向无循环；runtime 不依赖 host/apps，核心不硬编码具体 Transport。
- 所有外部输入在 codec、join、resource、subscribe 和 command 边界校验。
- control queue 与 data queue 分离或具备等价优先级保证。
- 内存、队列、payload、租约、pending 和重组缓冲均有配置上限。
- 错误必须结构化返回或记录，禁止影响正确性的静默 fallback。

#### Inputs / Outputs

Inputs:

- Transport byte stream；
- join/identity material；
- versioned protocol frame；
- resource registration；
- subscribe/unsubscribe request；
- variable update、stream event、command call。

Outputs:

- active authenticated tree edge；
- route/control decision；
- subscription snapshot/event/gap/expiry；
- command result/error/timeout；
- lifecycle events and bounded diagnostic state。

#### Edge Cases

- duplicate NodeID、duplicate child session、simultaneous reconnect；
- child-origin control message；
- stale topology epoch；
- parent disconnect during adjudication；
- subscribe/unsubscribe crossing reconnect；
- revision/sequence wrap or regression；
- short write, partial frame, oversize payload and malformed header；
- command result after timeout；
- queue full for control versus stream data。

#### Acceptance Criteria

- `go test ./...` passes from the new root module without legacy `go.work` or sibling `replace`.
- In-memory and TCP drivers pass the same link contract suite.
- Cross-subtree Variable, Stream and Command flows pass allow/deny tests.
- Child-origin forged control and stale-parent control are rejected.
- Reparent invalidates old epoch routes/subscriptions.
- No package imports `github.com/yttydcs/myflowhub-*` legacy modules.
- No old SubProto dispatcher participates in the integration path.

#### Risks

- Reusing too much Core code may carry old IConnection/ISubProcess assumptions into LinkSession.
- Defining the wire too broadly before Embedded validation may cause churn; phase one therefore keeps resource payload opaque and versioned.
- Parent absolute control increases subtree blast radius; auth/link code needs focused tests and small interfaces.
- A broad first phase could become another framework rewrite; acceptance remains limited to one vertical slice.

### Architecture Design

#### Overall Solution

```text
Transport Driver
    -> Pipe
    -> Frame Codec
    -> LinkSession (unauthenticated -> active)
    -> Node Tree + Auth/Policy
    -> Resource Registry
       -> Subscription (Variable / Stream)
       -> Command Dispatcher
    -> Node Runtime
    -> Hub / Go Client
```

#### Alternatives Considered

- 直接复制 Core + SubProto + Server：拒绝，会复制旧架构而非复用已验证能力。
- 先迁移所有应用再替换核心：拒绝，无法先证明核心模型。
- 只写抽象不做 TCP vertical slice：拒绝，容易形成不可运行的接口堆叠。

#### Module Responsibilities

| Module | Responsibility | Must not own |
| --- | --- | --- |
| `protocol` | IDs、envelope、phase/op、resource address、codec、wire errors | routing/business handlers |
| `runtime/link` | LinkSession state, framed send/receive, queues, liveness | tree policy/resources |
| `transport/memory` | deterministic tests and fault injection | runtime semantics |
| `transport/tcp` | TCP dial/listen and Pipe adaptation | routing/permissions |
| `runtime/tree` | parent/children, route index, LCA direction, epoch | payload decoding/business policy |
| `runtime/auth` | identity, join, source proof, request/control validation, RBAC hook | transport details |
| `runtime/resource` | registry, descriptor, Variable/Stream/Command definitions | network routing |
| `runtime/subscription` | lease, interest aggregation, snapshot/event delivery, backpressure | command execution |
| `runtime/command` | handler registry, invoke/result/error/timeout/dedup | subscription state |
| `runtime/node` | compose link/tree/auth/resource/subscription/command | UI/platform code |
| `host/hub` | composition root and configuration | duplicate runtime logic |
| `sdk/go` | typed Go client API | server/host implementation |

#### Data / Call Flow

Join:

1. Transport establishes Pipe.
2. LinkSession starts in unauthenticated state and accepts only handshake frames.
3. Auth verifies identity and desired role.
4. Tree validates single-parent/duplicate-child invariants.
5. Session becomes active and route index is updated atomically.

Subscribe:

1. SDK creates request-phase subscribe frame with subscriber identity and resource address.
2. Node forwards upward until the resource owner is downstream of the current node.
3. LCA/auth hook allows or rejects.
4. Allowed request becomes control-phase subscribe and travels downward.
5. Owner returns Variable snapshot or establishes Stream start point.
6. Intermediate nodes maintain bounded aggregated interest and clean it on lease/link expiry.

Command:

1. Caller sends request-phase command with message ID and deadline.
2. LCA authenticates subject and scope, then emits control phase.
3. Owner validates input and invokes handler.
4. Result/error follows correlation route; late results after timeout are discarded or reported diagnostically.

Reparent:

1. New parent session authenticates but does not become active until tree transition succeeds.
2. Node increments topology epoch and atomically swaps the parent edge.
3. Old route, pending requests and cross-parent subscriptions are invalidated.
4. Old-parent delayed control fails parent/epoch checks.

#### Interface Drafts

```go
type Phase uint8 // request, control, response, event

type Envelope struct {
    Version       uint16
    Phase         Phase
    Operation     Operation
    MessageID     MessageID
    CorrelationID MessageID
    Source        NodeID
    Target        NodeID
    Resource      ResourceID
    TopologyEpoch uint64
    DeadlineUnixMS int64
    Payload       []byte
}
```

```go
type Driver interface {
    Dial(context.Context, Endpoint) (Pipe, error)
    Listen(context.Context, Endpoint) (Listener, error)
}

type Session interface {
    ID() SessionID
    Peer() Identity
    Role() LinkRole
    State() SessionState
    Send(context.Context, protocol.Envelope) error
    Close() error
    Done() <-chan struct{}
}
```

```go
type ResourceID struct {
    Owner NodeID
    Name  string
}

type Registry interface {
    Register(Resource) error
    Resolve(ResourceID) (Resource, bool)
    Remove(ResourceID) error
}
```

The exact wire widths and exported names are finalized in M02 and documented before downstream packages depend on them.

#### Error Handling and Safety

- Define stable error codes for malformed, unauthenticated, forbidden, not found, conflict, stale epoch, expired, overflow, timeout and internal failure.
- Protocol decode rejects oversize, truncated, unknown mandatory version/op and invalid phase combinations.
- Link closes after unrecoverable framing/auth errors; transient send errors propagate without silent retry unless policy explicitly owns retry.
- Tree mutations are atomic and rollback on partial activation.
- Resource/command handlers validate schema and limits even for parent-authorized control.
- Do not copy environment-specific credentials or local machine settings into the monorepo.

#### Performance and Testing Strategy

- Separate bounded control and data queues; control cannot be starved by Stream traffic.
- Variable updates may coalesce by resource/revision; Stream overflow emits gap metadata.
- Use contract suites for every Transport and codec implementation.
- Use deterministic in-memory fault injection for short writes, disconnects, delayed frames and reordering boundaries.
- Run focused package tests during tasks; final focused validation includes `go test ./...`, selected `-race`, `go vet ./...`, architecture dependency checks and `git diff --check`.

#### Extensibility Design Points

- New Transport implements Driver/Pipe only.
- New resource behavior composes Variable, Stream or Command rather than adding a new routing protocol.
- Policy is injected at the LCA/auth boundary without changing tree routing.
- Payload codec/schema is versioned independently from the fixed envelope.
- File and Flow remain features above core primitives.

#### Issue List

- None blocking plan approval.

## Stage 3.1 - Planning

### Project Goal and Current State

Current system remains in legacy multi-repo form. This plan creates a new root-module runtime in parallel; it does not switch existing products in the next execution phase.

### Docs Governance Routing Decision

Explicitly using `$m-docs`:

- Docs root: `D:/project/MyFlowHub3/docs`
- Intake impact: clarify — already records clean-break and monorepo decisions
- Feature impact: none — first phase has no production UI/product cutover
- Requirements impact: add — [unified-node-runtime.md](D:/project/MyFlowHub3/docs/requirements/unified-node-runtime.md)
- Specs impact: add/clarify — accepted architecture specs exist; M02/M05-M07 must add exact vNext wire and runtime contracts when finalized
- Decision impact: add — monorepo and authoritative tree ADRs are accepted
- Lessons impact: none at planning time; reuse existing cross-repo semver lesson

Related docs:

- Intake: `D:/project/MyFlowHub3/docs/intake/2026-08-27_node-tree-subscription-command-redesign.md`
- Requirement: `D:/project/MyFlowHub3/docs/requirements/unified-node-runtime.md`
- Specs:
  - `D:/project/MyFlowHub3/docs/specs/node-tree-link-resource-architecture.md`
  - `D:/project/MyFlowHub3/docs/specs/repository-and-module-boundaries.md`
- Decisions:
  - `D:/project/MyFlowHub3/docs/decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md`
  - `D:/project/MyFlowHub3/docs/decisions/2026-08-27_single-canonical-monorepo.md`
- Lesson: `D:/project/MyFlowHub3/docs/lessons/cross-repo-semver-release.md`

### Executable Task List

#### Approved For Execution

- M00 — Baseline manifest and accepted docs handoff
- M01 — Monorepo root module and dependency guardrails
- M02 — vNext protocol envelope and codec
- M03 — Pipe, Transport and LinkSession foundation
- M04 — Authoritative tree, identity and adjudication
- M05 — Resource Registry, Variable, Stream and Command definitions
- M06 — Subscription lifecycle and bounded delivery
- M07 — Command invocation and result lifecycle
- M08 — Node runtime, minimal Hub and Go client
- M09 — Cross-subtree vertical slice and focused validation

#### Will Not Execute Now

- D01 — Port real QUIC, RFCOMM, serial and platform providers; deferred until LinkSession contract passes phase-one tests.
- D02 — Port File, Flow and remaining legacy feature behavior; deferred until core primitives stabilize.
- D03 — Migrate Desktop and Android apps; deferred to a separate product migration wave.
- D04 — Migrate MetricsNode and ClipboardNode; deferred, with MetricsNode dirty generated bindings preserved and reconciled later.
- D05 — Migrate C and MicroPython EmbeddedSDK; deferred until vNext envelope and schemas stabilize.
- D06 — Add a legacy wire/SubProto compatibility bridge; not planned because there are no external consumers, only reconsider if first-party migration is otherwise blocked.
- D07 — Configure remote, CI/release publication, archive old repositories, merge or clean worktrees; requires later validation and explicit archive/release phase.

### Task Details

#### M00 - Baseline manifest and accepted docs handoff

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/monorepo-rewrite`
- Plan Path: `plan.md`
- Goal: Record legacy source commits and bring only accepted stable architecture docs into the target worktree without copying unrelated dirty state.
- Files / Modules: `migration/sources.yaml`, `docs/intake`, `docs/requirements`, `docs/specs`, `docs/decisions`, category indexes
- Write Set: `migration/`, selected governed docs in active worktree
- Acceptance:
  - every legacy source repo maps to an exact commit and future target area;
  - MetricsNode dirty generated paths are recorded as excluded/preserved;
  - no nested `.git`, legacy `go.mod`, binaries or secrets are imported;
  - accepted docs and indexes have no broken relative links.
- Test Points: docs link checker; manifest schema/unit check; `git status --short` review
- Rollback: revert `migration/` and selected docs additions in the active worktree only.

#### M01 - Monorepo root module and dependency guardrails

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Establish one root Go module and enforce target dependency boundaries before runtime code grows.
- Files / Modules: `go.mod`, `go.sum`, `README.md`, `internal/archtest`, optional root build scripts
- Write Set: root module metadata and architecture tests
- Acceptance:
  - module path is `github.com/yttydcs/myflowhub`;
  - repository builds without legacy `go.work`, sibling `replace`, nested module or nested Git repository;
  - architecture tests reject forbidden dependency directions and legacy `myflowhub-*` imports.
- Test Points: `go list ./...`; architecture tests; repository scan
- Rollback: revert root module and guardrail files.

#### M02 - vNext protocol envelope and codec

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Define the transport-neutral, versioned frame contract needed by link/tree/resource operations without SubProto routing.
- Files / Modules: `protocol/`, `docs/specs/wire-protocol-vnext.md`
- Write Set: `protocol`, focused spec/index updates
- Acceptance:
  - typed NodeID, MessageID, ResourceID, Phase, Operation, Envelope and structured error contract;
  - bounded deterministic encode/decode and length validation;
  - request/control phase combinations validated;
  - payload remains opaque and content/schema metadata is explicit;
  - no `SubProto` field or handler dispatch dependency.
- Test Points: roundtrip, malformed/truncated, oversize, unknown version/op, fuzz seed corpus
- Rollback: revert protocol/spec additions.

#### M03 - Pipe, Transport and LinkSession foundation

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Reuse proven byte-stream lessons while replacing IConnection/ISubProcess coupling with a narrow LinkSession state machine.
- Files / Modules: `runtime/link`, `transport/memory`, `transport/tcp`
- Write Set: link and two transport packages
- Acceptance:
  - memory and TCP implement the same Driver/Pipe contract;
  - LinkSession supports unauthenticated/active/closing/closed states;
  - bounded prioritized send queues, framing, liveness and cancellation;
  - short writes and partial frames handled correctly;
  - Transport names/types do not appear in resource/tree code.
- Test Points: shared transport contract suite, short-write adapter, cancel/close, queue full, frame limit, TCP loopback
- Rollback: revert link/transport packages; no legacy repo changes.

#### M04 - Authoritative tree, identity and adjudication

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Implement the single-parent node tree and original parent-control/LCA request-control semantics.
- Files / Modules: `runtime/tree`, `runtime/auth`
- Write Set: tree/auth packages and tests
- Acceptance:
  - authenticated join activates exactly one parent edge;
  - duplicate parent/child and cycle attempts fail explicitly;
  - source proof restricts child to itself/known descendants;
  - request routes upward and control routes downward;
  - child trusts only current parent + control phase + valid topology epoch;
  - reparent invalidates old parent, routes and link-bound authority.
- Test Points: tree/LCA table tests, forged source/control, stale epoch, disconnect cleanup, concurrent reparent
- Rollback: revert tree/auth packages.

#### M05 - Resource Registry, Variable, Stream and Command definitions

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Establish node-owned resources as the only core domain surface.
- Files / Modules: `runtime/resource`, `docs/specs/resource-model-vnext.md`
- Write Set: resource package and stable contract
- Acceptance:
  - unique owner + local name addressing;
  - descriptor kind/schema/permission metadata;
  - Variable current value with monotonic revision;
  - Stream sequence/event source without mandatory current value;
  - Command descriptor/handler contract;
  - duplicate registration, invalid names, revision regression and oversize values fail explicitly.
- Test Points: registry lifecycle, concurrency, revision/sequence boundaries, schema/limit validation
- Rollback: revert resource/spec additions.

#### M06 - Subscription lifecycle and bounded delivery

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Make Subscription a first-class runtime relation for Variable and Stream.
- Files / Modules: `runtime/subscription`, `docs/specs/subscription-vnext.md`
- Write Set: subscription package and spec
- Acceptance:
  - subscribe/unsubscribe, lease expiry and link cleanup;
  - Variable snapshot-before-change ordering;
  - Stream event sequence and explicit gap/overflow signal;
  - bounded per-link/per-resource queues and variable coalescing;
  - intermediate interest aggregation preserves subscriber authorization and lease boundaries.
- Test Points: snapshot race, multiple subscribers, aggregation, expiry, disconnect, slow consumer, overflow/gap
- Rollback: revert subscription/spec additions.

#### M07 - Command invocation and result lifecycle

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Implement Command as the imperative complement to subscriptions.
- Files / Modules: `runtime/command`, `docs/specs/command-vnext.md`
- Write Set: command package and spec
- Acceptance:
  - register/invoke/result/error lifecycle;
  - LCA authorization before cross-subtree control;
  - direct parent-to-child control semantics;
  - input validation, deadline, cancellation, duplicate message detection and late-result handling;
  - structured forbidden/not-found/conflict/timeout/internal errors.
- Test Points: allow/deny, direct control, timeout, duplicate, late response, handler panic isolation
- Rollback: revert command/spec additions.

#### M08 - Node runtime, minimal Hub and Go client

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Compose the new packages into a usable runtime without legacy Server/defaultset assembly.
- Files / Modules: `runtime/node`, `host/hub`, `sdk/go`, `cmd/mfh-hub`, `cmd/mfh-node`
- Write Set: composition, CLI and public Go client surface
- Acceptance:
  - start/stop Node and Hub with context cancellation;
  - explicit configuration and dependency injection;
  - Go client supports connect/join, subscribe Variable/Stream and invoke Command;
  - startup failures are actionable and clean partial state;
  - no imports from legacy modules.
- Test Points: lifecycle, invalid config, startup rollback, SDK await/cancel, local loopback
- Rollback: revert node/host/sdk/cmd additions.

#### M09 - Cross-subtree vertical slice and focused validation

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Prove the phase-one architecture end to end and update stable docs with final exact contracts.
- Files / Modules: `tests/integration`, affected specs/indexes, architecture tests
- Write Set: integration tests and planning-time stable doc clarifications only
- Acceptance:
  - root with two branches and multiple leaves works over memory and TCP;
  - cross-subtree Variable snapshot/change, Stream event/gap and Command allow/deny/timeout pass;
  - disconnect and reparent invalidate old state;
  - control traffic remains deliverable under saturated Stream data;
  - no legacy SubProto dispatcher/module participates;
  - stable specs match implemented interfaces.
- Test Points:
  - focused package tests during implementation;
  - `go test ./... -count=1`;
  - selected `go test -race ./runtime/... ./tests/integration/... -count=1`;
  - `go vet ./...`;
  - architecture import scan;
  - docs link check;
  - `git diff --check`.
- Rollback: revert integration/doc clarification changes or reset the dedicated branch; legacy products remain untouched.

### Dependencies

```text
M00 -> M01 -> M02 -> M03 -> M04 -> M05 -> M06 -> M07 -> M08 -> M09
```

- M02 must stabilize envelope types before LinkSession.
- M03 must provide authenticated-session hooks before tree join.
- M04 must establish authority and routing before network resource operations.
- M05 resource contracts precede Subscription and Command runtime behavior.
- M08 composes only after focused packages pass.
- Deferred D01-D07 do not block phase-one acceptance.

### Risks and Notes

- The active worktree is based on root HEAD and does not automatically contain accepted uncommitted docs from the main checkout; M00 must port only the selected stable docs and verify them.
- Do not copy entire legacy repositories or their `.git`, `go.mod`, build artifacts, generated bindings or local config.
- MetricsNode dirty generated Wails bindings remain untouched; later migration should regenerate rather than blindly copy them.
- No remote, push, tag, repository archive or destructive cleanup is authorized in this phase.
- No business/application migration is implied by the new CLI integration harness.

### Parallelism Assessment

- Execution is assigned to the main agent only. No implementation sub-agents will be dispatched unless the user explicitly requests delegation in a later turn.
- Package work remains sequential through M05 because the core contracts are intentionally being stabilized.
- M06 and M07 could be developed independently after M05, but the current execution policy keeps them sequential to reduce interface churn.

### Issue List

- None blocking execution.

## Approved Execution Scope

### Will Execute

- M00, M01, M02, M03, M04, M05, M06, M07, M08, M09

### Will Not Execute Now

- D01, D02, D03, D04, D05, D06, D07 for the reasons recorded above.

## Approval Gate

- Blocked: no — the user explicitly approved execution by invoking `$m-execute` on 2026-08-27.
- Approved Task IDs: M00-M09.
- Enter implementation in the dedicated worktree only.
- Do not modify legacy source repositories.
- Do not dispatch implementation sub-agents.

## Stage 3.2 Execution Record

### Task Status

- M00 — Completed: pinned 10 legacy source commits, preserved 16 MetricsNode dirty generated paths, and handed off accepted architecture docs.
- M01 — Completed: established the single root module and architecture/import/document guards; removed the legacy workspace files from the canonical target.
- M02 — Completed: implemented the bounded `MFH3` envelope, operations, opaque payload metadata, codec, structured errors, and fuzz seeds.
- M03 — Completed: implemented Driver/Pipe/Listener, memory and TCP contract adapters, and a cancellable LinkSession with separate bounded control/data queues and liveness.
- M04 — Completed: implemented Ed25519 join/ack, explicit trust, one-parent topology, descendant source proof, route epochs, request/control validation, and reparent invalidation.
- M05 — Completed: implemented node-owned Registry, Variable, Stream, and Command with bounded values and monotonic revisions/sequences.
- M06 — Completed: implemented leases, link cleanup, independent interest boundaries, snapshot ordering, Variable coalescing, and explicit Stream gaps.
- M07 — Completed: implemented bounded invocation, authorization origins, timeout/cancellation, duplicate detection, late-result observation, and panic isolation.
- M08 — Completed: composed Node, Hub, Go SDK, and minimal CLI entry points without legacy runtime imports.
- M09 — Completed: proved cross-subtree Variable, Stream, Command, deny/timeout, disconnect/reparent, and control-queue behavior over memory and TCP.

### Validation Evidence

- `GOWORK=off go test ./... -count=1`: Passed.
- `GOWORK=off go test -race ./runtime/... ./tests/integration/... -count=1`: Passed.
- `GOWORK=off go vet ./...`: Passed.
- Architecture/import/module/document/migration guards: Passed as part of `go test ./...`.
- `git diff --check`: Passed; Git reports only expected Windows LF-to-CRLF checkout warnings.

The explicit `GOWORK=off` is required only because this dedicated worktree is nested below the untouched legacy meta-workspace whose parent `go.work` is auto-discovered by Go. The canonical repository itself contains no `go.work`, sibling `replace`, nested module, or nested Git repository.

### Residual Scope

- D01-D07 remain deferred exactly as approved.
- No legacy repository, dirty main checkout, or MetricsNode generated binding was modified.
- No remote, commit, merge, archive, release, or worktree cleanup was performed.
- Rollback remains the dedicated branch/worktree back to `master@6ffd5d7`; legacy products remain untouched.

## Stage 3.3 Continue and Test Record

### Recovered State

- `$m-continue` classified the existing M00-M09 implementation as `validation-needed`; no deferred D01-D07 work was added.
- Validation remained local because this run did not authorize sub-agent delegation. Independent test lanes were executed concurrently where safe.

### Repair Iteration 1

- M04/M06: replacement LinkSessions now clean subscriptions bound to the old link even when the peer identity is unchanged.
- M04: parent activation commits the exact signed topology epoch, while parent/child teardown matches both peer identity and session epoch so a stale session cannot remove a replacement edge.
- M06: Variable delivery keeps capacity in the internal bounded queue and no longer stages an intermediate revision in a second buffered output queue.
- M07: command completion uses an unbuffered result handoff so a deadline-winning result is always classified as late and observed exactly through the late-result hook.
- M08: all Node-owned background operations register under the closed-state lock and are awaited before the diagnostics channel closes.
- M02 test contract: fuzz comparison treats nil and zero-length payload slices as the same wire value and retains an explicit empty-payload seed.
- Exact reconnect/epoch cleanup behavior was added to the accepted architecture and wire specifications.

### Independent Validation Evidence

- Focused repair regression: `GOWORK=off go test ./runtime/tree ./runtime/subscription ./runtime/command ./runtime/node -count=100 -timeout=5m` — Passed.
- Race validation: `GOWORK=off go test -race ./runtime/... ./tests/integration/... -count=10 -timeout=8m` — Passed.
- Full randomized validation: `GOWORK=off go test ./... -shuffle=on -count=20 -timeout=8m` — Passed.
- Cross-subtree memory/TCP vertical slice: `GOWORK=off go test ./tests/integration -run TestCrossSubtreeVerticalSlice -count=100 -timeout=8m` — Passed in 12.824 seconds.
- Protocol fuzzing: `GOWORK=off go test ./protocol -run '^$' -fuzz FuzzCodecRoundTrip -fuzztime=15s` — Passed after 3,527,777 executions.
- `GOWORK=off go vet ./...`, architecture/document/migration guards, `gofmt` scan, and `git diff --check` — Passed; only expected Windows LF-to-CRLF checkout warnings remain.

### Test Decision

- Terminal result: `Passed`.
- Blockers: none.
- M00-M09 are ready for an explicit `$m-archive` pass.
- No archive document, commit, merge, push, release, legacy-repository archive, or worktree cleanup was performed by `$m-continue`.

## Stage 4 Archive Record

### `$m-docs` Routing And Impact

- Docs root: `D:/project/MyFlowHub3/docs`
- Intake impact: updated — original request now links to the completed change.
- Feature impact: none — no production application or UI was cut over.
- Requirements impact: updated — unified Node runtime phase-one acceptance is implemented and linked to the archive.
- Specs impact: updated — tree/link/resource, repository, wire, resource, subscription, and command contracts link to the completed change.
- Decision impact: updated — both 2026-08-27 Accepted ADRs link to the completed change.
- Lessons impact: updated — session replacement and asynchronous cleanup knowledge was promoted for symptom-first lookup.

### Archive Artifacts

- Change: `docs/change/2026-08-27_canonical-monorepo-unified-node-runtime.md`
- Plan archive: `docs/plan/plan_archive_2026-08-27_canonical-monorepo-unified-node-runtime.md`
- Lesson: `docs/lessons/session-replacement-generation-cleanup.md`
- Updated indexes: `docs/change/README.md`, `docs/plan/README.md`, `docs/lessons/README.md`

### Closeout Boundary

- Archive content and stable-doc links are complete.
- Local commit, control-plane fast-forward merge, worktree removal, and feature-branch deletion are authorized by the explicit `$m-archive` invocation and delegated to the post-archive control-plane closeout; their actual terminal state is reported by that closeout rather than this immutable plan snapshot.
- Main-checkout unrelated dirt must remain untouched. Only paths overlapping the workflow commit may enter a named protection stash during integration; the protection reference is retained unless restoration is proven complete.
- No remote, push, publication, release, legacy-repository archive, or external backup was authorized.
