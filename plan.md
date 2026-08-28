# Plan - 可扩展资源平台与 Desktop 工作区重构

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub`
- Branch: `refactor/resource-platform-desktop-workspace`
- Base: `master@6a07c08fe8ab531b057f148f467539de0eb05641`
- Project Root: `D:/project/MyFlowHub3`
- Docs Root: `D:/project/MyFlowHub3/worktrees/resource-platform-desktop-workspace/docs`
- Code Repos: canonical monorepo `MyFlowHub` only
- Worktree: `D:/project/MyFlowHub3/worktrees/resource-platform-desktop-workspace`
- Current Stage: `3.2 execute; DOC01 completed, RES01 in progress`
- Compatibility: clean break；不保留 fixed-three-kind wire、TopicBus、SubProto、旧 Desktop UI 或旧配置静默兼容

## Stage Records

### Initialization

- `guide.md`: read；中文 commit、UI 必须包含单测与 production build、docs 从 `docs/README.md` 进入、worktree 位于 workspace sibling、`GOWORK=off`。
- project/docs/code repo confirmation: `project_root=D:/project/MyFlowHub3`；canonical docs 与 code 都在单一 `repo/MyFlowHub`，本地工作区文档不推送或发布。
- base/worktree confirmation: dedicated branch/worktree 已由 `$m-discuss` 创建；主 checkout 的 `guide.md` 与 `论文/**` 用户改动不进入本工作树。
- sub-agent policy: 用户未请求委派；本 workflow 不派发实现 sub-agent。

### Discuss - Discovery And Requirements Shaping

#### Goal

把 Node/Resource 提升为唯一产品抽象，支持可扩展 Resource type，并把 Desktop 重建为可登录、多 Profile、可浏览、预览、组合和保存 View 的资源工作台。

#### Scope

- Resource descriptor/capability、generic operations、Topic、subscription/auth/routing generalization；
- session control 与 File 数据 lane 迁移；
- SDK/bindings/first-party products clean-break；
- Desktop React/shadcn、Profile、Explorer、Renderer、Workspace、View；
- 全产品和 UI 门禁、stable docs supersession。

#### Assumptions

- 用户直接调用 `$m-plan` 视为确认 discussion 的五项推荐默认值。
- 没有外部用户，不需要旧 wire、API、config 或 UI compatibility bridge。
- 唯一 authoritative Node tree、父控子、Resource ownership 和可插拔 Transport 不变。
- View 首版 local-first；生产级 Media、远端 View sync 与多 Profile 同时在线不在下一执行阶段。

#### Open Questions

- Blocking: none。
- 实现期非阻塞选择必须遵守本计划边界；任何扩大到 WebRTC/编解码器、云同步或多活动 Profile 的选择要回到新 plan。

#### Options Considered

- hard-coded type enum / extensible type+capability / all-Command；选择 extensible type+capability。
- fixed pages / infinite canvas / responsive grid；选择 responsive grid。
- Vanilla TS / React+shadcn；选择 React+shadcn。

#### Rejected Options

- 恢复 SubProto、全局匿名 TopicBus、独立权限树或资源树。
- 将 File/Media 大数据作为普通 Stream event。
- 明文持久化密码、私钥或一次性 permit。
- 同时保留两套 Desktop 前端。

#### Recommended Direction

- descriptor-driven Resource platform；Topic 为 Node-owned brokered Resource；File 使用 session/data lane；Desktop 使用 renderer registry 和 per-Profile responsive-grid Views。

#### Research Summary

- shadcn/ui 官方支持 Vite + React + TypeScript，并提供 Sidebar、Tabs、Resizable、Scroll Area、Command 等 primitives。
- Wails 官方支持 React/TypeScript/Vite 与 generated Go bindings。
- dnd-kit 提供 React 拖放和键盘/辅助技术路径；TanStack Virtual 仅在大型树测量证明需要时引入。
- Primary sources 已记录在 [discussion intake](docs/intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)。

#### Worktree / Branch / Docs Root Status

- branch/worktree ready；docs root identified；remote publication 未授权。

#### Issue List

- none。

### Plan - Requirements And Architecture

#### Discussion Summary

现有 fixed-three-kind 模型已经完成一次统一，但不应继续通过 Core enum 承载 Topic/File/Media；Desktop 的固定页面也不再满足通用资源平台。规划采用第二次 clean break，同时保留 Node tree 与 authority invariants。

#### Accepted / Rejected Requirements

- Accepted: extensible types、Topic、Profile/auto-login、Node/Resource tree、preview、drag-to-compose、saved Views、minimal shadcn UI。
- Accepted with boundary: File 迁移到 session/data lane；Media 只建立可扩展/session 控制边界，不交付生产视频。
- Rejected: legacy compatibility、全局匿名 topic、无限画布首版、明文 secret、UI 权限裁决。

#### Requirements Analysis

##### Goal

交付一个 descriptor-driven Resource runtime 和一个真正消费该模型的 Desktop workspace，使新增类型不再要求重写 Core、权限、路由和整个 UI shell。

##### Scope

- Protocol/runtime/auth/subscription/link/session/SDK/bindings/products/Desktop/docs/tests。

##### Use Cases

1. 发现/读取/写入/订阅 Variable。
2. 订阅 owner Stream 并处理 gap。
3. 多 Node 发布/订阅 Topic。
4. 调用 Command。
5. 打开 File session 并在独立 data lane 传输。
6. 登录/切换 Profile，浏览树并预览资源。
7. 将资源组合为 View，持久化后恢复。
8. 未知 type、断线、撤权、schema upgrade 的安全降级。

##### Functional Requirements

- 以 [可扩展资源平台需求](docs/requirements/extensible-resource-platform.md) 与 [Desktop 工作区需求](docs/requirements/desktop-resource-workspace.md) 为准。

##### Non-functional Requirements

- bounded memory/queues/payloads/sessions；control priority；no silent fallback；accessible UI；atomic local stores；clean checkout reproducibility。

##### Inputs / Outputs

- Inputs: resource descriptors/operations/events、profile credentials/config、Node topology/catalog、drag/keyboard workspace actions。
- Outputs: typed operation results/subscriptions、Topic deliveries、File session progress、rendered previews/widgets、versioned Views/profile state。

##### Edge Cases

- unknown type/capability/version；reparent/revoke during subscription/session；slow consumer；Topic duplicate/publisher reset；damaged profile/view；missing resource/renderer；large tree。

##### Acceptance Criteria

- requirements docs 的全部 acceptance；clean-break product matrix；Desktop visual/keyboard/persistence；no tracked generated drift。

##### Risks

- wire/runtime/SDK/Desktop 同时迁移面大；以可编译门禁和分阶段 commit 控制。
- generic capability 可能退化为字符串 RPC；通过 type-owned validation、schema 和 observable/session interfaces 控制。
- Topic 与 Stream 重叠；以 publisher ownership、publish permission 和 ordering contract 测试区分。
- data lane 可能饿死 control 或反向阻塞；使用独立有界队列和公平优先调度测试。
- shadcn primitives 不提供完整 tree/grid domain；将 domain state 与 UI component 分离并覆盖键盘测试。

#### Architecture Design

##### Overall Solution

```text
Transport Driver
      ↓
LinkSession: bounded control lane + bounded data sessions
      ↓
Authoritative Node Tree / Auth / Routing
      ↓
Resource Registry
  descriptor(type + capabilities + schemas + limits)
      ↓
Variable · Stream · Topic · Command · File(session) · extensions
      ↓
SDK / Bindings
      ↓
Desktop Renderer Registry + Explorer + Workspace Views
```

##### Alternatives Considered

- 见 ADR；不再在 planning 阶段重新打开已确认方向。

##### Module Responsibilities

- `protocol/`: Resource descriptor v2、generic resource operation/subscription/session payload、Topic/File schemas、wire major bump。
- `runtime/resource/`: registry、type/capability validation、generic dispatch、built-in type packages。
- `runtime/subscription/`: descriptor-driven observable subscription、lease、bounded delivery、gap/revoke/recovery。
- `runtime/auth/`: exact subject + resource + capability/action policy。
- `runtime/node/`: generic operation route/correlation、catalog/topology interaction。
- `runtime/link/`: bounded control/data lanes、session lifecycle、priority/fairness/backpressure。
- `feature/file/`: File Resource/session、checksum、atomic storage、progress observable。
- `sdk/go`, `sdk/bindings`: descriptor discovery、generic operation、typed helpers、generated contract。
- `apps/**`, `embedded/**`, `cmd/**`, `host/**`: first-party migration without local wire/type switches。
- `apps/desktop`: Profile/CredentialStore/Wails boundary and resource workspace product。

##### Data / Call Flow

- Discovery: subscribe/read owner `system/catalog` -> descriptor v2 -> client renderer/capability selection。
- Operation: client builds generic operation -> route to authority裁决 -> owner registry validates capability/schema -> type handler -> correlated result/error。
- Subscription: capability declares observable -> authority建立 lease -> type source emits typed events -> bounded delivery/gap -> durable client recovery。
- Topic publish: publisher invokes Topic `publish` capability -> owner validates publisher policy/rate/event -> broker fan-out to active subscriptions；默认不回放。
- File: `open` control -> scoped session/lease -> data chunks on bounded data lane -> checksum/atomic commit -> progress observable -> close/expire/revoke cleanup。
- Desktop: active Profile opens SDK -> topology/catalog store -> explorer selection or drag -> renderer registry -> transient preview or persistent View widget。

##### Interface Drafts

```go
type ResourceDescriptor struct {
    ID           ResourceID
    Type         ResourceTypeID
    TypeVersion  uint32
    Capabilities []CapabilityDescriptor
    Schemas      []SchemaDescriptor
    Limits       ResourceLimits
    Presentation PresentationHint
}

type Resource interface {
    Descriptor() ResourceDescriptor
    Operate(context.Context, OperationRequest) (OperationResult, error)
}

type Observable interface {
    Subscribe(context.Context, SubscriptionRequest) (EventSource, error)
}

type SessionResource interface {
    OpenSession(context.Context, SessionOpenRequest) (SessionGrant, error)
}
```

- 具体命名可在 RES01 内调整，但不得改变 descriptor-driven、type-owned validation、exact capability permission 和未知类型 fallback 四项约束。

##### Error Handling and Safety

- stable error code 至少覆盖 unsupported type/version/capability、invalid schema、forbidden、expired、resource gone、gap、rate/queue/session limit、credential unavailable、view corrupt。
- Owner 始终验证业务输入；authority approval 不替代 type validation。
- Session token 绑定 subject/resource/capability/link/topology/policy generation/expiry，不可跨 Node 或 Profile 复用。
- Profile switch 的 finally path 必须关闭 subscription/session/connection；失败不能进入半切换状态。

##### Performance and Testing Strategy

- control/data lane 分离，公平优先而非无限 control starvation；队列和 chunk 全部 bounded。
- Topic fan-out 不在 registry 锁内执行 subscriber delivery；slow consumer 产生明确 gap/expiry。
- Explorer 使用 normalized tree store 和 row-level memoization；只有规模基准超阈值才加入 virtualizer。
- tests: unit -> protocol/contract -> memory/TCP cross-subtree -> product matrix -> frontend component -> browser mocked bridge -> Wails GUI -> clean checkout full gate。

##### Extensibility Design Points

- type registration package不允许 Core import 产品包；unknown type discoverable。
- renderer registration与 Core type package分离；unknown renderer fallback。
- presentation hint 只影响默认 renderer，不影响 permission/operation。
- Media、future transports 和 synced Views 可以在不改 Node tree 的前提下增加。

#### Issue List

- none。

### Stage 3.1 - Planning

#### Project Goal and Current State

- Current: fixed Variable/Stream/Command runtime、chunked File、Vanilla TS Desktop 已可运行。
- Target: extensible Resource platform + Topic + session-oriented File + React/shadcn Desktop workspace。

#### Docs Governance Routing Decision

- Docs root: `D:/project/MyFlowHub3/worktrees/resource-platform-desktop-workspace/docs`，与 canonical code 同仓、仅本地，不配置 remote/publish。
- Original request -> intake；durable goals -> requirements；accepted hard-to-reverse direction -> decision；target interfaces -> specs during DOC01；implemented behavior -> feature docs after corresponding code passes；workflow result -> later change/archive。

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake: [extensible resources and Desktop redesign](docs/intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)
- Features: [Desktop](docs/features/desktop.md), [File transfer](docs/features/file-transfer.md)
- Requirements: [extensible resource platform](docs/requirements/extensible-resource-platform.md), [Desktop resource workspace](docs/requirements/desktop-resource-workspace.md), [unified node runtime](docs/requirements/unified-node-runtime.md), [auth admission](docs/requirements/auth-controlled-admission.md)
- Specs: [node-tree architecture](docs/specs/node-tree-link-resource-architecture.md), [resource model vNext](docs/specs/resource-model-vnext.md), [subscription vNext](docs/specs/subscription-vnext.md), [file transfer](docs/specs/file-transfer-vnext.md)
- Decisions: [extensible resource type and Desktop workspace](docs/decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md), [authoritative node tree](docs/decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- Lessons: current relevant build/UI lessons remain indexed；本阶段不新增 lesson。

#### Stable Docs Impact

- Intake impact: add（completed in discussion）。
- Feature impact: deprecate/rewrite Desktop and File current behavior after implementation; other product dossiers clarify migrated resource types。
- Requirements impact: add（resource platform + Desktop workspace completed in planning）。
- Specs impact: supersede resource/catalog/subscription/wire/file contracts during DOC01/related tasks。
- Decision impact: add/partial supersession（completed in planning）。
- Lessons impact: none known；execution/archive must reassess。

#### Executable Task List

- [x] DOC01 — target specs、supersession chain 与 feature transition skeleton。
- [ ] RES01 — protocol Resource descriptor v2 与 generic operations/wire major。
- [ ] RES02 — extensible registry/type interfaces 与 Variable/Stream/Command migration。
- [ ] RES03 — Node-owned Topic runtime 与 typed SDK helpers。
- [ ] RES04 — auth、routing、subscription、catalog descriptor-driven generalization。
- [ ] SES01 — bounded resource sessions/control-data lanes 与 File migration。
- [ ] SDK01 — Go SDK、bindings、generated contract 和 Embedded contract migration。
- [ ] APP01 — Hub、Management、Metrics、Clipboard、Notification、Flow 和 commands migration。
- [ ] UI01 — React/shadcn frontend foundation 与 Wails boundary model。
- [ ] UI02 — Profile、CredentialStore、login/admission、auto-connect 与切换。
- [ ] UI03 — Node/Resource Explorer、normalized store 与 Renderer Registry。
- [ ] UI04 — Preview、responsive workspace、drag/add/resize 与 View persistence。
- [ ] UI05 — built-in renderers、状态覆盖、minimal visual system 与 accessibility。
- [ ] VAL01 — clean-break cleanup、全产品/GUI/clean-checkout validation 与 docs current-truth切换。
- [ ] MED01 — production Media data plane（deferred）。
- [ ] SYNC01 — synced Views 与多 Profile 同时在线（deferred）。
- [ ] MOB01 — Android/iOS 通用资源工作区（deferred）。
- [ ] LEG01 — legacy wire/SubProto/TopicBus compatibility（rejected）。
- [ ] PUB01 — remote push/release/sign/store/hardware certification（separate authorization）。

#### Execution Scope After Approval

##### Will Execute

- DOC01, RES01, RES02, RES03, RES04, SES01, SDK01, APP01, UI01, UI02, UI03, UI04, UI05, VAL01。

##### Will Not Execute Now

- MED01：生产视频需要独立 codec、WebRTC/QUIC media QoS、平台 capture/playback 和安全计划；本轮只保留 type/session extension point。
- SYNC01：首版 View 明确 local-first、单活动 Profile；同步和并发连接单独规划。
- MOB01：用户当前要求 Desktop；移动工作区不是本轮验收。
- LEG01：无外部用户且已接受 clean break，兼容层会恢复被删除复杂度。
- PUB01：push、发布、签名、商店与真实硬件需要单独授权/凭据。

#### Task Details

##### DOC01 - Target specs and supersession
- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/resource-platform-desktop-workspace`
- Plan Path: `plan.md`
- Goal: 在代码迁移前建立 Resource platform、Topic、session/data lane、Desktop renderer/view 的 target specs，并明确旧 spec/ADR supersession。
- Files / Modules: `docs/specs`, `docs/features`, `docs/decisions`, category indexes
- Write Set: Markdown only
- Acceptance: target contract 可独立指导 RES/UI；旧文档不会与新 current truth 竞争；feature docs保留 current/transition 状态直到对应实现通过。
- Test Points: relative-link/index/archtest docs guards；`git diff --check`
- Rollback: revert DOC01 docs commit，不影响 runtime。

##### RES01 - Protocol Resource descriptor v2 and generic operations
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 用 type/capability/schema/limits/presentation descriptor 和 generic operation/subscription/session payload 替代 fixed kind wire。
- Files / Modules: `protocol/**`, `internal/protocoltest`, fixtures, generators
- Write Set: protocol schemas/codecs/tests only
- Acceptance: deterministic catalog；unknown type可解码/发现；unknown major/capability拒绝；old wire operation从 canonical contract移除。
- Test Points: protocol unit/fuzz/golden/roundtrip；generated contract drift
- Rollback: revert RES01 before downstream tasks；不提供双 wire。

##### RES02 - Extensible registry and built-in types
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: registry按 Resource interface/capability dispatch；迁移 Variable、Stream、Command，移除 type switch扩展点。
- Files / Modules: `runtime/resource`, `runtime/command`, supporting protocol adapters
- Write Set: runtime resource/command and focused tests
- Acceptance: built-ins保持既有 snapshot/sequence/invoke语义；custom test type无需修改 Core即可注册；foreign owner/duplicate capability失败。
- Test Points: unit/race/custom-type contract；panic/deadline/dedupe
- Rollback: revert RES02 + RES01 as one incompatible slice。

##### RES03 - Node-owned Topic
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 实现多 publisher/subscriber Topic、分权、per-publisher sequence、rate/queue limits 和默认无 replay。
- Files / Modules: `runtime/resource/topic` or equivalent, `protocol`, `sdk/go`, tests
- Write Set: Topic type, schemas, typed client, tests
- Acceptance: 2 publishers × 2 subscribers；unauthorized publish/subscribe拒绝；duplicate/slow consumer/reconnect语义明确；无 retained replay。
- Test Points: unit/race/memory+TCP cross-subtree integration
- Rollback: remove Topic registration/client and revert descriptor entries。

##### RES04 - Descriptor-driven auth, routing, subscription and catalog
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: policy绑定 exact resource+capability；routing/correlation generic；subscription消费 Observable capability；catalog v2。
- Files / Modules: `runtime/auth`, `runtime/node`, `runtime/subscription`, `runtime/tree`, `host/hub`
- Write Set: runtime control plane and tests
- Acceptance: revoke/topology epoch/policy generation/link cleanup覆盖 Variable/Stream/Topic；未知 capability不转发；no kind-specific route switch。
- Test Points: integration topology/policy/reparent/recovery/race
- Rollback: revert RES04 with RES01-03 incompatible slice。

##### SES01 - Resource sessions, control/data lanes and File migration
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 引入 subject/resource/capability-bound session grant；LinkSession提供独立有界 control/data queues和公平优先调度；File迁移到 session data lane。
- Files / Modules: `protocol`, `runtime/link`, `runtime/node`, `feature/file`, `sdk/go`, transports tests
- Write Set: session/link/file/SDK and tests
- Acceptance: file checksum/atomic/cancel/expiry保持；session revoke/reparent立即清理；大文件传输期间 heartbeat/command延迟门禁通过；无无界队列。
- Test Points: memory/TCP/QUIC/RFCOMM logical-lane tests；file integration；race；fault injection
- Rollback: revert SES01；不保留 command-chunk compatibility path。

##### SDK01 - SDK, bindings, generated and Embedded migration
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 提供 generic descriptor/operate/subscribe/session API和Variable/Stream/Topic/Command/File typed helpers；更新所有 bindings/fixtures。
- Files / Modules: `sdk/go`, `sdk/bindings`, `embedded`, generated contract, build scripts
- Write Set: SDK/bindings/embedded/generators/tests
- Acceptance: JSON/string/basic-type platform boundaries；unknown type descriptor可见；generated clean；C/MicroPython fixtures对齐。
- Test Points: SDK contract/race；gomobile；CMake/CTest；MicroPython；generated gate
- Rollback: revert SDK01 with protocol/runtime slice。

##### APP01 - First-party product migration
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Hub/Management/Metrics/Clipboard/Notification/Flow/cmd/apps使用新 descriptor/capability，不复制 wire或恢复TopicBus。
- Files / Modules: `feature/**`, `host/**`, `cmd/**`, `apps/android`, `apps/nodes/**`, product tests
- Write Set: non-Desktop first-party product sources/docs/tests
- Acceptance: all current product behaviors通过新 API；Clipboard仍订阅明确 peer Resource；Metrics/Flow/Management权限保持；无旧 kind/wire import。
- Test Points: product matrix；Android unit/lint；process E2E；race where supported
- Rollback: revert APP01 with preceding incompatible slice。

##### UI01 - React and shadcn foundation
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 用 React/TS/Vite/Tailwind/shadcn clean-break替换 Vanilla frontend，建立 bridge adapter、routing/shell、test harness和design tokens。
- Files / Modules: `apps/desktop/frontend`, `apps/desktop/wails.json`, generated bindings as needed
- Write Set: frontend/package lock/config/tests; delete old DOM imperative UI
- Acceptance: production build；shadcn primitives vendored/configured；mock Wails adapter；无第二套UI；system theme/CJK font基础。
- Test Points: Vitest/Testing Library/TypeScript/Vite；generated Wails build
- Rollback: revert UI01 commit恢复旧frontend，前提是尚未合并UI02-05。

##### UI02 - Profiles, credentials and login lifecycle
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: versioned Profile index/store、CredentialStore接口和Windows protected backend；login/admission、auto-connect、switch/delete/reset。
- Files / Modules: `apps/desktop` Go host/config/credential/profile packages, frontend auth/profile UI, bindings/tests
- Write Set: Desktop profile/security lifecycle only
- Acceptance: two profiles完全隔离；成功admission后不长期保存permit明文；重启auto-connect；切换清理旧subscription/session；credential failure明确。
- Test Points: Go temp-store/atomic/failure tests；frontend state/widget tests；Wails smoke
- Rollback: versioned store migration只允许显式reset；revert UI02不得删除既有identity数据。

##### UI03 - Node/Resource Explorer and Renderer Registry
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: normalized topology/catalog state；Tabs中的Explorer/View manager；Node/Resource selection/search/keyboard；renderer registry+fallback。
- Files / Modules: `apps/desktop` binding model, frontend `features/explorer`, `renderers`, state stores
- Write Set: Desktop discovery/UI domain/tests
- Acceptance: 跨子树Node和owner Resources正确展示；path grouping不改变authority；unknown renderer fallback；loading/offline/forbidden可见。
- Test Points: model reducers/selectors；component/keyboard tests；large-tree benchmark before virtualization
- Rollback: remove explorer/renderer feature while保留UI foundation/profile。

##### UI04 - Preview, workspace and View persistence
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: transient preview、drag/button/keyboard add、responsive layout、resize/reorder、per-Profile atomic View CRUD/recovery。
- Files / Modules: Desktop View store/bindings, frontend workspace/dnd/layout/view manager
- Write Set: View persistence and workspace UI/tests
- Acceptance: resource拖入并保存；重启布局一致；missing/denied/type-upgrade占位；损坏store不覆盖；拖放有键盘等价路径。
- Test Points: Go atomic/failure tests；dnd/layout reducer；browser interaction；persistence E2E
- Rollback: View schema versioned；revert UI04不删除profile/identity，保留可导出的view文件。

##### UI05 - Built-in renderers and minimal accessible visual system
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: Variable/Stream/Topic/Command/File renderers、Media placeholder、Node overview、minimal polished states和accessibility。
- Files / Modules: frontend renderers/components/styles/tests; Desktop typed bridge helpers
- Write Set: renderer/UI behavior only
- Acceptance: read/write/subscribe/publish/invoke/file session从UI可用且权限错误准确；统一spacing/type/color；light/dark/reduced motion；关键流程键盘可用。
- Test Points: component/state/a11y checks；browser screenshots at representative viewport；Wails live preview
- Rollback: renderer按注册表独立移除，fallback保持可用。

##### VAL01 - Clean-break convergence and full validation
- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 删除旧fixed-kind/wire/UI/config代码和生成物；更新current feature/spec/docs；跑全产品、GUI、clean checkout gate。
- Files / Modules: repository-wide necessary cleanup, docs, migration inventory, scripts/build as required
- Write Set: only obsolete paths and directly affected docs/tests/build inputs
- Acceptance: no old build/runtime fallback；all stable docs/indexes一致；clean checkout generated/build/test/run clean；主工作区用户改动不受影响。
- Test Points: `go test ./... -count=1`, `go vet ./...`, `mfh check all -AllowUnavailable`, `mfh check generated`, Desktop npm/test/build/Wails, Android/Metrics/Clipboard/Embedded, run-dev, clean checkout status, credential scan
- Rollback: revert whole feature branch before merge；不提供局部旧/新混跑。

#### Dependencies

```text
DOC01
  ↓
RES01 → RES02 → RES03 → RES04 → SES01 → SDK01 → APP01
                  │                         │
                  └──────────────┐          │
                                 ↓          ↓
UI01 → UI02 → UI03 → UI04 → UI05
                    ↓
                  VAL01
```

- UI01可在RES01-04期间只依赖mock descriptor并行设计，但当前host不派发sub-agent，主agent按门禁串行。
- UI03接真实SDK前依赖SDK01；UI mock contract必须最终由generated contract替换。
- VAL01依赖所有Will Execute tasks。

#### Risks and Notes

- 这是repository-wide clean break；不允许在主checkout直接实现或混入 `guide.md`/`论文/**`。
- 每个Task完成后先运行focused gate并形成中文commit，避免一个不可审查巨型提交。
- LinkSession data lane与Desktop profile credential是最高风险点；失败必须显式阻塞，不得用无保护文件或unbounded queue临时绕过。
- 不安装不必要依赖；shadcn组件按实际使用加入，Tree/Workspace domain由项目拥有。

#### Parallelism Assessment

- 用户未请求sub-agent，host policy不允许主动委派，因此不派发。
- 单agent仍按独立Write Set和门禁串行推进；未来若用户明确授权，可把UI mock-only、docs-link检查等只读/低耦合任务拆分。

#### Issue List

- Blocking issues: none。
- Approval required: 用户必须明确调用 `$m-execute` 或批准Will Execute Task IDs后才能写业务逻辑。
