# Plan - Resource Collections、Capability Actions 与 Desktop 交互

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `feat/resource-collections-actions`
- Base: `master` @ `eb1f722ca9c336f72bb4a207fd6b780301ada791`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\resource-collections-actions\docs`
- Code Repos: canonical monorepo `MyFlowHub` only
- Worktree: `D:\project\MyFlowHub3\worktrees\resource-collections-actions`
- Participating Modules: `protocol`、`runtime/resource`、`feature/flow`、新 `feature/filesystem`、`sdk/go`、
  `sdk/bindings`、`apps/desktop`、`apps/desktop/frontend`、canonical `docs/`
- Current Stage: `$m-archive`；`DOC01, COLL01, FLOW01, FS01, SDK01, DESK01, RENDER01, QA01, MAIN02` 已完成，
  archive docs 已生成，local integration/cleanup in progress
- Publication: local-only；未授权 remote、push、release 或 publication

## Stage Records

### Initialization

- `guide.md`: 已读取；中文 commit、canonical docs、`GOWORK=off` 与 sibling worktree 规则生效。
- Real owning repo: `D:\project\MyFlowHub3\repo\MyFlowHub`，不是外层 workspace。
- Dedicated branch/worktree: 已创建 `feat/resource-collections-actions` / 当前 worktree。
- Main checkout protection: 主 checkout 的 Metrics dist、Desktop docs、Agent Gateway/Desktop-Metrics docs、
  design demos、`guide.md` 等未提交内容均不属于本 workflow write set。
- Existing root `plan.md`/`todo.md` described an archived renderer workflow；该历史已有 `docs/plan` archive，
  root control files are replaced for this active workflow only.

### Discuss - Discovery And Requirements Shaping

#### Goal

把 Resource、Resource Capability、Collection member、独立 Command Resource 与 Desktop 交互收敛成一套模型：
普通操作不再占据独立资源节点；大量动态成员不膨胀全局 catalog；目录、Flow definitions/runs 等 provider 可以
复用 Collection contract；Desktop 可以从 Explorer、Inspector 和 View 安全触发同一 capability，并按内容
schema 使用专用 renderer。

#### Confirmed direction

- Resource 继续是全局身份、owner、授权与审计单位。
- Capability 表达对 Resource 的普通操作。
- Collection 是一个 Resource 管理多个 scoped members；member 默认不是全局 Resource。
- UI path namespace 只是展示，不是 Collection 或权限树。
- 独立 Command Resource 只用于没有自然目标且自身形成独立执行/安全/审计边界的操作。
- filesystem 每个需要单独授权的 root 默认注册为一个 Collection Resource。
- Flow 使用 `flow/definitions` 与 `flow/runs` Collections。
- Desktop 真实 Resource 提供右键/键盘操作入口；View Widget 提供常用动作和 schema/content renderer。
- 不暴露任意 Windows shell，不让 UI、presentation hint 或产品名赋予权限。

#### Rejected options

1. 每个 member 和每个 API endpoint 都注册为 Resource：拒绝，catalog/UI/权限管理会膨胀。
2. 把所有行为集中进全局 `commands/`：拒绝，破坏领域 owner、schema 与权限边界。
3. 只在 Desktop 折叠旧命令节点：拒绝，无法解决协议、SDK 和 provider 模型。
4. 将真实磁盘路径作为 ResourceID：拒绝，会泄漏部署细节并绑定平台。
5. 暴露 `system/exec.invoke(commandLine)`：拒绝，等价于高风险远程 shell。

#### Discussion status

- Coherent requirement: yes.
- External research required: no.
- Blocking questions: none for the bounded first implementation.
- Stable design: recorded in requirements/spec/decision; current APIs remain unchanged until the corresponding implementation and generated contracts land.

## Plan - Requirements And Architecture

### Requirements Analysis

#### Scope

Will implement:

- `mfh.collection` type and bounded generic member list/page contract.
- reusable multi-capability handler Resource while preserving Registry schema/size/copy/error enforcement.
- typed generic Collection SDK helpers and regenerated first-party contracts.
- clean-break Flow migration to two Collections.
- opt-in, read-only filesystem Collection provider supporting multiple independently authorized roots.
- Desktop shared Resource action model, accessible context menu, View action entry, Collection browser, safe text/code/image rendering.
- stable docs convergence and full proportional validation.

Will not imply:

- a second resource/authority tree;
- global identity for every member;
- arbitrary remote shell;
- permission granted by catalog metadata or UI controls;
- generic per-member policy selectors or effective-capability discovery in this phase;
- writable/delete filesystem operations or large-file download sessions;
- migration of every existing Management/Admission/Notification Command Resource.

#### Use cases

1. A product registers `storage/a` and `storage/b` from different local roots without registering each file.
2. A caller granted `list/read` on `storage/a` can browse and preview bounded files but cannot escape the root.
3. Flow callers list/get/create/update/archive/run definitions through `flow/definitions` and list/get/subscribe/cancel runs
   through `flow/runs`.
4. Desktop users right-click a real Resource or press Menu/Shift+F10 to preview, add to View, or open a supported operation.
5. A View Widget exposes safe common actions and renders Collection members using schema/content type; text is escaped,
   allowlisted raster images are decoded locally, and unknown/binary content falls back explicitly.
6. A pure Explorer namespace only expands/collapses and never exposes Resource actions.

#### Functional requirements

- Collection list requests and page/member responses are versioned, bounded, deterministic and schema-validated.
- Member descriptors contain opaque provider-scoped key, kind, label, optional content/schema/attributes and sorted
  capability subset; they never claim a Node owner or global ResourceID.
- A handler Resource rejects missing/extra handlers, unsupported capability, wrong schema and oversized input/output.
- Existing Variable/Stream/Topic/Command/File behavior remains unchanged outside explicit Flow migration.
- Outer authorization remains exact `subject + ResourceID + capability`; providers additionally validate member locator,
  caller ownership where applicable, root containment and domain rules.
- Flow old endpoint Resources disappear atomically with first-party SDK/contracts/tests; old grants do not silently map to
  broader new grants.
- Filesystem registration validates unique Resource names and roots before registering any mount; partial failure rolls back
  registrations. Physical roots never appear in catalog payloads.
- Filesystem list/read uses canonicalized paths, rejects absolute/parent/traversal and symlink/junction escape, bounds entries,
  cursor, metadata, file size and response payload.
- Desktop action derivation is a pure function of descriptor plus optional known authorization state. Without effective
  introspection it shows supported operations and preserves authoritative Forbidden feedback.
- Destructive/mutating/input-bearing actions open an explicit operation form; context-menu selection alone does not execute.
- Context menu has pointer and Menu/Shift+F10 entry, Escape close, focus restoration and accessible labels.
- Renderer selection remains provider-schema/Desktop-registry/user-View separated and never persists member content or drafts.

#### Non-functional requirements

- Core remains type-extensible; no product type switch is added to routing/auth.
- Collection listing is O(entries sorted for one page scope) with explicit `limit` and maximum scan/response bounds.
- No unbounded file read, event buffer, cursor, attribute map or renderer decode.
- No new data-plane queue; operations continue through Node/SDK generic operation and existing session lanes.
- Go APIs remain platform-neutral; Android and other bindings continue to use the same generic operation path.
- Accessibility and keyboard behavior are tested, not inferred from pointer smoke.
- Generated artifacts are deterministic and included in freshness gates.

#### Inputs / Outputs

Inputs:

- Resource descriptor and capability schemas;
- Collection list/member requests;
- Flow definition/run payloads;
- filesystem mount config and member locator;
- Desktop Resource selection, action selection, renderer selection and operation drafts.

Outputs:

- Collection page/member descriptors;
- domain-specific member payloads and Flow events;
- bounded filesystem content response;
- typed SDK results and regenerated binding contracts;
- Desktop menus, forms, action buttons and safe content views.

No physical root, Resource value, file content, operation draft, credential or permission decision is persisted in View settings.

#### Edge cases

- Empty collection, exact final page, stale/invalid cursor, duplicate/unsorted member capability and oversized attributes.
- Collection disappears or descriptor changes while Explorer/menu/Widget is open.
- Old Flow grant remains persisted but old Resource no longer exists.
- Flow run is cancelled by non-owner, terminal run is cancelled again, or definition is archived while active.
- Filesystem root is missing, relative, replaced, symlinked/junctioned, case-varied, unreadable or changes between list/read.
- File is empty, invalid UTF-8, too large, unsupported binary, HTML/SVG, or MIME/extension disagree.
- Right-click on a pure namespace, keyboard menu on a focused Resource, operation Forbidden after menu opened.
- Saved View refers to an old Flow command Resource; layout remains but Widget shows missing Resource rather than retargeting.

#### Acceptance criteria

- Catalog contains `mfh.collection` descriptors with validated standard list/page schemas.
- Registry/runtime tests prove arbitrary multi-capability dispatch without weakening schema, size, copy or unsupported errors.
- Flow catalog contains `flow/definitions` and `flow/runs` only for this domain; old five command Resources and
  `flow/events` are absent, and all first-party callers use the new capabilities.
- Three-mount filesystem fixture registers exactly three Resources; physical roots are absent from catalog; list/read work;
  traversal and symlink/junction escape are rejected.
- Desktop real Resource context menu and keyboard equivalent work; pure namespaces have no menu; selecting an input-bearing
  action opens the shared explicit form and makes no call until Execute.
- View Widget actions, Collection list/detail navigation and text/code/raster-image/fallback renderers work without path-name
  dispatch or persisted content.
- Focused/full Go tests, generated checks, frontend tests/build and Windows Wails production build pass; representative GUI
  interaction verifies menu, Flow actions, filesystem text preview, View restore, light/dark and Forbidden states.

### Architecture Design

#### Overall solution

```text
Node catalog: one Collection Resource
          │ descriptor capabilities + schemas
          ▼
authority: subject + ResourceID + capability
          ▼
runtime HandlerResource dispatch
          ▼
provider validates member key + domain/root scope
          ▼
domain payload/event
          ▼
attached SDK / bindings generic operation
          ▼
Desktop shared action controller
       ├── Explorer context menu
       ├── Inspector operation form
       └── View Widget action + content renderer
```

#### Protocol contract

- Add `ResourceTypeCollection = "mfh.collection"`.
- Add reusable capability constants where semantics are generic: `list` and `get`; domain names such as `run`, `archive`
  and `cancel` remain extensible CapabilityIDs.
- Add bounded protocol payloads equivalent to:

```go
type CollectionListRequestV1 struct {
    Version int
    Parent  string
    Cursor  string
    Limit   int
}

type CollectionMemberV1 struct {
    Key          string
    Kind         string
    Label        string
    ContentType  string
    Schema       string
    Capabilities []CapabilityID
    Attributes   map[string]string
}

type CollectionPageV1 struct {
    Version    int
    Revision   uint64
    Parent     string
    Members    []CollectionMemberV1
    NextCursor string
}
```

- Exact field names/limits are finalized in the spec before generated use. Cursor remains opaque to callers.
- `list` uses the standard request/page schema. Other capabilities declare domain schemas explicitly.
- Capability `Permission` remains compatibility metadata in v2 and does not become a second authorization key; removing or
  renaming it requires the deferred catalog-version task.

#### Runtime responsibilities

- Add a normalized HandlerResource constructor with a map from CapabilityID to handler.
- Constructor verifies every operation capability has exactly one handler and no undeclared handler exists; event-only
  capability is owned by an explicit observable wrapper/provider.
- Registry remains the single point for descriptor normalization, input/output schema matching, bounds and defensive copies.
- No Collection logic is inserted into Node routing or auth switches.

#### Flow mapping

| Resource | Capability | Input | Output/Event |
| --- | --- | --- | --- |
| `flow/definitions` | `list` | collection list request | collection page |
| `flow/definitions` | `get` | member request | Flow definition |
| `flow/definitions` | `create` | Flow definition | Flow definition |
| `flow/definitions` | `update` | Flow definition | Flow definition |
| `flow/definitions` | `archive` | Flow archive | Flow archive |
| `flow/definitions` | `run` | Flow run | Flow run summary |
| `flow/runs` | `list` | collection list request | collection page |
| `flow/runs` | `get` | member request | Flow run summary |
| `flow/runs` | `subscribe` | subscription control | Flow event |
| `flow/runs` | `cancel` | Flow cancel | Flow run summary |

The existing initiator delegation, downstream policy intersection, dedupe, limits, persistence and event sequencing remain.

#### Filesystem provider

- New feature package owns configuration and Resource implementations; NodeHost/Core do not import it.
- API shape is equivalent to `filesystem.Register(node, []Mount)` so Desktop, Metrics, Agent Gateway or another product may
  opt in without installing or coupling another product.
- One mount is one Resource by default. Named virtual mounts are deferred because their shared permission meaning must remain
  explicit.
- First phase capabilities: `list`, `get` metadata and bounded `read`; provider is read-only by default and in implementation.
- Content response carries locator, content type, encoding/data, size and stable metadata needed for stale-read feedback.
- HTML/SVG is never rendered as executable content; unsupported binary stays metadata/raw-download-placeholder only.

#### SDK and generated contracts

- Add generic typed payload operation helper accepting ResourceID + CapabilityID, rather than adding one method per endpoint.
- Add a small CollectionClient for list/get plumbing; FlowClient calls the two Collection Resource IDs and explicit capabilities.
- Keep `host.Client()` and direct SDK Client on the same optimized Node path; no extra connection or queue is introduced.
- Update binding contract fixtures and generated Desktop schema artifact. Existing generic `OperateJSON` remains the Wails and
  Android-compatible transport surface unless implementation evidence requires a narrow additive facade.

#### Desktop interaction

- Introduce a pure ResourceAction model shared by Explorer, Inspector and Workspace.
- Use an accessible context-menu primitive; adding the focused Radix context-menu package is permitted if it avoids a custom
  focus/placement implementation and remains the only new UI dependency.
- Context menu actions select/open an operation controller. Only explicit Execute invokes a mutating/input-bearing capability.
- Widget renderer exposes a compact primary-action area and reuses the same operation form/result state.
- Collection renderer performs paged list, selected-member get/read, bounded loading/error/empty states and cancellation on
  unmount/resource/profile change.
- Content renderer selects by authoritative result schema/content type: escaped text/code/JSON, allowlisted raster image, or
  metadata/raw fallback. Resource name is never a dispatch key.

#### Permission model in this phase

- Network authority continues exact capability grants on the Collection Resource.
- Provider validates member containment and existing domain ownership rules.
- Separate roots are separate Resources when they require different grants.
- Generic member-prefix selectors and an authoritative effective-capabilities discovery API are deferred as `AUTHZ02`.
- Desktop therefore shows descriptor-supported actions, optionally honors known authorization state, and always handles
  authoritative Forbidden without silent retry or optimistic success.

#### Error handling and safety

- Invalid schema/version/cursor/member key: explicit malformed error.
- Missing member/resource: explicit not found/gone behavior.
- Unsupported capability: Registry unsupported error before provider mutation.
- Forbidden: preserved from authority and rendered at the originating menu/form/widget.
- Filesystem escape/symlink/junction/oversize/unsupported content: fail closed with actionable reason.
- Flow migration: no fallback alias; stale Widgets stay visibly missing, stale grants remain inert.
- Partial provider registration: remove already registered mounts in reverse order.
- UI async state: abort or ignore stale result by resource/action generation; drafts survive operation failure.

#### Performance and validation strategy

- Protocol/runtime/provider/SDK use focused table tests plus existing cross-node integration tests.
- Collection page and filesystem list enforce hard limits; Desktop does not load an unbounded member tree.
- Frontend action derivation, menu keyboard behavior, operation no-submit-before-Execute, collection state and content safety are
  unit/component tested.
- Full repository and production packaging gates run only after focused suites pass.

#### Extension points

- Other providers implement the standard list seam and domain capabilities without Core changes.
- A member may later be promoted to global Resource without changing Collection member keys already stored by the provider.
- Future AUTHZ02 may add member selectors/effective capability discovery without changing the rule that UI is not authority.
- Future FS02 may add atomic write/delete and download sessions without placing large bytes on the control lane.

### Stage 3.1 - Planning

#### Docs Governance Routing Decision

- Docs root: `D:\project\MyFlowHub3\worktrees\resource-collections-actions\docs`，canonical repo-local governed docs root.
- Intake impact: add — discussion evidence recorded.
- Feature impact: clarify — update Flow and Desktop current behavior only when implementation lands.
- Requirements impact: clarify — Collection/action and Desktop interaction requirements added.
- Specs impact: add/clarify — new Collection/action spec and Flow target contract added; protocol map/status updated during execution.
- Decision impact: add — Collection Resource + Capability Action ADR added.
- Lessons impact: none known at planning time; archive re-evaluates filesystem/path or generated-contract findings.
- Index status: intake/specs/decisions indexes updated; requirements existing leaves require no new index entry.

#### Related stable docs

- Intake: `docs/intake/2026-08-31_resource-collections-and-actions.md`
- Features: `docs/features/flow.md`、`docs/features/desktop.md`
- Requirements: `docs/requirements/extensible-resource-platform.md`、
  `docs/requirements/desktop-resource-workspace.md`
- Specs: `docs/specs/resource-platform-v2.md`、`docs/specs/resource-collections-and-actions.md`、
  `docs/specs/flow-vnext.md`、`docs/specs/desktop-schema-rendering.md`、
  `docs/specs/desktop-resource-workspace-v3.md`
- Decisions: `docs/decisions/2026-08-31_collection-resource-and-capability-actions.md`、
  `docs/decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md`、
  `docs/decisions/2026-08-30_provider-schema-desktop-renderer-ownership.md`
- Lessons: none required yet; consult `observable-side-effects-and-generated-contracts.md` and
  `windows-clean-checkout-eol-and-generated-drift.md` during execution.

#### Execution Scope After Approval

##### Will Execute

- DOC01
- COLL01
- FLOW01
- FS01
- SDK01
- DESK01
- RENDER01
- QA01

##### Will Not Execute Now

- AUTHZ02 — generic member selector policy、filtered catalog/effective-capability discovery；needs separate policy/schema design.
- CMD02 — migrate Management/Admission/Notification and every remaining endpoint-style Command；deferred by domain.
- FS02 — filesystem write/delete、large-file download/session、virtual multi-root mounts；requires separate destructive/data-lane design.
- MAIN02 — resolve the main checkout's concurrent uncommitted edit to
  `docs/requirements/desktop-resource-workspace.md`；the original unmerged index entry has disappeared externally, but the
  same-path overlap remains outside this worktree and must be resolved by its owning workflow before merge/archive.
- PUB01 — push、release、publication；not authorized and no remote is assumed.

#### Task Details

##### DOC01 - Converge Stable Contracts And Current Feature Truth

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: keep intake/requirements/spec/decision/indexes coherent, then update Flow/Desktop feature truth and protocol map when
  implementation is real.
- Files / Modules: `docs/intake`、`docs/features/flow.md`、`docs/features/desktop.md`、`docs/requirements`、
  `docs/specs/resource-collections-and-actions.md`、`flow-vnext.md`、`protocol_map.md`、`docs/decisions` and indexes.
- Write Set: governed docs only; no `docs/change` until archive.
- Acceptance: implemented/current vs accepted/pending status is accurate; Flow capability table matches code; deferred AUTHZ02/
  CMD02/FS02 remain explicit; all links resolve.
- Test Points: Markdown link/path inspection and `git diff --check`.
- Rollback: revert DOC01 docs without changing runtime.

##### COLL01 - Add Collection Protocol And Multi-capability Runtime Foundation

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: add bounded Collection schemas and a reusable Resource implementation that dispatches declared capabilities.
- Files / Modules: `protocol/schema_catalog.go`、new `protocol/schema_collection.go`、`protocol/data_schema.go` and tests；
  `runtime/resource` handler implementation/tests.
- Write Set: protocol/runtime foundation and focused tests only; no product migration.
- Acceptance: valid descriptors/pages pass; malformed/unsorted/oversized data fails; handlers exactly match declared operation
  capabilities; Registry schema/size/copy/error behavior remains.
- Test Points: protocol validation/generation fixtures, runtime handler dispatch, missing/extra handler, schema mismatch, payload
  boundaries, concurrent read-only dispatch.
- Rollback: remove new type/schema/helper; existing resource types remain unchanged.

##### FLOW01 - Migrate Flow To Definitions/Runs Collections

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: replace endpoint-style Flow Commands/Variables/Stream with two Collection Resources while preserving business behavior.
- Files / Modules: `protocol/schema_flow.go`、`feature/flow` and tests, relevant policy/integration fixtures.
- Write Set: Flow constants/schemas/resource wrappers/controller wiring/tests; clean-break removal of old Flow endpoint Resources.
- Acceptance: exact capability matrix in this plan; list/get paging; run events subscribe on runs; persistence/dedupe/limits/
  initiator delegation/cancel/archive semantics preserved; old Resources absent and old grants inert.
- Test Points: local and cross-node auth, grant per capability, downstream initiator policy, list/get pagination, event sequence,
  dedupe/retry/concurrency/restart/archive/cancel/overflow.
- Rollback: revert FLOW01 as one unit to old Flow Resources before SDK/Desktop migration is merged.

##### FS01 - Add Opt-in Read-only Filesystem Collection Provider

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: let any Node product register one or more local roots as independent Collection Resources without Core/product coupling.
- Files / Modules: new `feature/filesystem` package/tests and filesystem protocol/data-schema definitions.
- Write Set: provider, config, safe locator resolver, list/get/read payloads and tests; no default product mount.
- Acceptance: atomic multi-mount registration; no physical path in catalog; deterministic paged list; bounded metadata/read;
  traversal, absolute path, symlink/junction escape and oversize fail closed; read-only contract exposes no mutation capability.
- Test Points: three-root fixture, duplicate Resource name/root failure rollback, Unicode/case/path cases, empty/large directory,
  symlink/junction, MIME sniff/invalid UTF-8/HTML/SVG/binary/size, caller policy grant per root.
- Rollback: unregister/remove optional provider package; no persistent format migration.

##### SDK01 - Add Typed Collection/Capability Clients And Regenerate Contracts

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: keep callers on the existing optimized Node operation path while removing per-endpoint Flow client calls.
- Files / Modules: `sdk/go/client.go`、`sdk/go/features.go` and tests；`sdk/bindings/contract`、bindgen outputs、Desktop generated
  schemas/contracts and freshness gates.
- Write Set: typed payload helper, Collection client, Flow client migration, generated artifacts/tests; no new owning runtime.
- Acceptance: typed list/get/Flow operations use ResourceID + CapabilityID; host-attached/direct SDK behavior is identical;
  generic Wails/Android operation surface remains usable; generated outputs are deterministic and fresh.
- Test Points: memory/TCP typed contract tests, wrong schema/Forbidden/unknown capability, generation coverage/freshness.
- Rollback: revert SDK/generated outputs together with FLOW01; no persisted SDK state.

##### DESK01 - Create Shared Resource Actions And Accessible Context Menu

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: expose descriptor-supported actions consistently from Explorer, Inspector and Workspace without executing on visibility.
- Files / Modules: `apps/desktop/frontend/src/lib/resource-actions*`、`components/Explorer.tsx`、Inspector/operation controller、
  optional focused Radix context-menu primitive、styles/tests/types.
- Write Set: pure action model, accessible menu, focused operation state and tests; no provider/path special casing.
- Acceptance: real Resource only; pointer and Menu/Shift+F10; focus/Escape; add-to-View; input/mutating capability opens explicit
  form and performs zero calls before Execute; Forbidden remains actionable.
- Test Points: pure derivation, namespace exclusion, keyboard/focus restoration, menu close, stale descriptor, no-submit,
  same operation payload/result/error across entry points.
- Rollback: remove menu/action layer; existing select/add/Inspector and ResourceRenderer remain.

##### RENDER01 - Add Collection Widgets, View Actions And Safe Content Renderers

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: make Collections useful in Inspector/View and render filesystem member content by schema/content type.
- Files / Modules: renderer registry/schema/controller modules、`Renderer.tsx` split as needed、`Workspace.tsx`、CSS/component tests.
- Write Set: Collection controller/renderer, compact action buttons, text/code/JSON/raster-image/fallback views; View document
  version remains 3.
- Acceptance: paged list/detail/read, loading/empty/error/cancel states; buttons reuse shared action controller; no Resource-name
  dispatch; no HTML/SVG execution; no file/member content or draft persisted; saved old Widgets remain readable/missing explicitly.
- Test Points: list pagination, selection changes, text escaping, invalid UTF-8, JSON/code, safe raster Blob cleanup,
  unsupported binary fallback, unmount/profile cleanup, narrow pane, light/dark, save/reopen no-content persistence.
- Rollback: renderer registry falls back to generic operation/raw inspector; View layout/store unchanged.

##### QA01 - Run Full Validation And Packaged Desktop Evidence

- Owner: primary agent
- Worktree: current dedicated worktree
- Plan Path: root `plan.md`
- Goal: prove protocol/runtime/provider/SDK/Desktop correctness and regression safety before archive.
- Files / Modules: focused tests, deterministic generated outputs, frontend `dist`, temporary visual evidence later routed by
  `$m-archive`.
- Write Set: tests and deterministic generated/build artifacts only.
- Acceptance: all scoped checks pass; old Flow endpoints absent; no unreviewed generated drift; representative packaged GUI
  scenarios pass in light/dark and persist View layout without persisting content.
- Test Points:
  - focused `GOWORK=off go test` for protocol/runtime/resource/flow/filesystem/sdk/bindings;
  - `GOWORK=off go test ./... -count=1`;
  - generated contract check via canonical `scripts/mfh.ps1` target;
  - `npm test` and `npm run build` in Desktop frontend;
  - Windows `wails build -clean -trimpath -platform windows/amd64`;
  - browser/component interaction plus real packaged Wails smoke for context menu, Flow capability, filesystem text preview,
    View buttons, Forbidden, light/dark and restart restore.
- Rollback: stop before archive/merge, revert to the last passing task checkpoint, preserve failure signature/evidence.

#### Dependencies

```text
DOC01 -> COLL01 -> FLOW01 -> SDK01 --+
                 -> FS01 -----------+-> DESK01 -> RENDER01 -> QA01
```

- DOC01 establishes names and deferred boundaries used by code.
- COLL01 is required by Flow and filesystem providers.
- FLOW01 and FS01 may be developed independently after COLL01, but both update protocol schemas; sequential integration is
  safer unless execution creates non-overlapping commits.
- SDK01 consumes final Flow/filesystem contracts before Desktop integration.
- DESK01 owns the shared action seam; RENDER01 reuses it rather than duplicating calls.

#### Risks and notes

- Highest risk: Flow clean-break can leave stale policy grants and saved Widgets. Fail safely: no automatic permission widening,
  old Widgets remain visibly missing, and docs call out regranting.
- Filesystem path safety is platform-sensitive. Resolve canonical roots/targets, test Windows junction/symlink behavior, bound
  reads, and keep provider opt-in/read-only.
- Collection schema can become an accidental universal filesystem/database abstraction. Keep only list/page/member metadata
  generic; domain payloads remain provider schemas.
- Effective authorization discovery is not inferred from `Permission` strings. AUTHZ02 stays explicit and UI handles Forbidden.
- Desktop operation code already exists in `Renderer.tsx`; refactor to a shared controller before adding entry points to avoid
  three inconsistent execution paths.
- No unrelated Metrics UI, Desktop Profile/Enrollment, NodeHost, Agent Gateway, brand, Android lifecycle or design-demo changes.
- The main checkout no longer has the original unmerged index entry, but another concurrent workflow still has an ordinary
  uncommitted modification to `docs/requirements/desktop-resource-workspace.md`, overlapping this worktree. It does not block
  isolated implementation here, but merge/archive must stop if MAIN02 remains unresolved.

#### Parallelism assessment

- Initialization、docs routing 与 planning 已由 primary agent 完成；本节原有“尚未批准”判断已被用户明确批准的
  `$m-go` 取代。
- Implementation 按 Task ID 和依赖委派；FLOW01 与 FS01 只有在 COLL01 contract 稳定且 write set 不重叠时并行。
- 共享 protocol/generated 文件的任务必须顺序收敛，不能用并行覆盖其他 worker 结果。

#### Issue list

- No prerequisite blocks isolated implementation in this dedicated worktree.
- External closeout issue: MAIN02 must be resolved before merge/archive; this plan does not authorize touching the concurrent
  uncommitted main-checkout edit.
- Approval gate 已通过；MAIN02 仍只阻塞 merge/archive，不阻塞当前隔离 worktree 的已批准实施。

### Stage 3.2 - Approved Delegated Execution

- User approval: approved `DOC01, COLL01, FLOW01, FS01, SDK01, DESK01, RENDER01, QA01` for `$m-go`。
- Blocked: no for isolated implementation in this worktree。
- COLL01 completed: added bounded Collection schemas and descriptor-scoped member validation, plus generic exact-handler
  dispatch. Focused protocol/runtime tests, race detector, vet and scoped diff check passed.
- FS01 completed: added opt-in, independently authorized read-only filesystem Collections with atomic multi-mount registration,
  bounded deterministic list/get/read behavior, canonical path containment and built-in filesystem data schemas. Focused
  filesystem/protocol/Flow tests, filesystem race tests, vet and diff checks passed.
- FS01 known residuals: portable path APIs cannot eliminate every check-to-use race, so target resolution is reinforced by
  open-handle identity and post-read checks; Registry lacks atomic compare-and-remove, leaving an extremely narrow concurrent
  replacement window between Registration Close's identity check and removal.
- SDK01 completed: typed generic/Collection/Flow clients use the existing Node operation path；canonical binding/Desktop
  schemas were regenerated deterministically and focused memory/TCP/binding/race/vet checks passed。Android binding ABI was
  statically verified；Gradle could not start because the local JVM could not establish its loopback daemon connection，and
  the optional canonical AAR was absent。The generated check completed generation/tests but its final HEAD diff guard reported
  the two intentional uncommitted generated changes.
- DESK01 completed: real Resource rows expose pointer and Menu/Shift+F10 context actions with Escape/focus restoration；pure
  namespaces expose no Resource operation. Explorer、Inspector and Workspace reuse descriptor-derived action state and the
  explicit operation panel, without treating Resource name、permission compatibility strings or presentation as dispatch.
- RENDER01 completed: Inspector/View now share one descriptor-driven Collection browser with bounded pagination, stale-result
  suppression, domain `get` detail rendering and member + Resource descriptor double-gated `list/get/read`；members without
  `get` remain metadata-only with zero API calls. Text/JSON/raster/fallback rendering rejects
  invalid encodings, never executes HTML/SVG, revokes Blob URLs, and keeps member/content/action drafts transient. Workspace
  buttons reuse the DESK01 action model and explicit operation panel. A follow-up gate now keeps member-scoped `get` opt-in
  and treats mixed operation/event descriptors as explicit operations, matching HandlerResource. Focused Collection/renderer/
  workspace coverage plus the action-model checks, the final frontend 16-file/128-test suite, TypeScript/Vite build and scoped
  diff checks passed；QA01 retained the final deterministic `dist` bundle.
- DOC01 completed: the first contract/gate pass and final generated/code-backed Flow/Desktop current-truth pass are both
  complete；stable specs now record current Collection/Flow/filesystem/SDK/Desktop behavior and final QA evidence.
- QA01 completed: focused/full Go tests, race, vet, canonical generated freshness, frontend 16-file/128-test suite and build,
  Windows Wails package, dual-ABI Android AAR, offline Gradle unit/lint/assemble, real browser-mode interaction, and twice-launched
  packaged GUI smoke all passed. No Android device was attached, so physical-device smoke is recorded as Unavailable rather than
  passed. R5 is complete；temporary runtime/package artifacts were removed and final deterministic `dist` plus QA evidence remain.
- MAIN02 completed: merged latest `master@00bd333`, reconciled Profile entry/deactivation with atomic Resource Inspector state,
  retained both requirement sets and rebuilt tracked Vite assets from resolved source. Post-merge full Go/vet/generated freshness
  and frontend 17-file/134-test/build gates passed.
- Deferred unchanged outside this workflow: `AUTHZ02, CMD02, FS02, PUB01`；PUB01 remains unauthorized。

### Archive - Documentation And Closeout

- `$m-docs` impact complete：intake/features/requirements/specs/decision 已收敛并交叉链接，新增 change、lesson 与 plan snapshot。
- Governed QA evidence copied with SHA-256 equality into
  `docs/change/verification/2026-08-31_resource-collections-actions/`。
- Product commit：`46da972 feat: 引入资源集合与能力操作`；mainline merge commit：`946fc40`。
- Control-plane local integration、主检出用户改动恢复与 worktree/branch cleanup 由 ARC01 最后步骤处理。

## Approval Gate

- Plan status: approved for delegated execution.
- Approved Task IDs: `DOC01, COLL01, FLOW01, FS01, SDK01, DESK01, RENDER01, QA01`.
- Approval status: passed.
- Blocked: no.
- Active phase: `$m-archive` local integration/cleanup.
- MAIN02 is complete；push/release/publication remains unauthorized as `PUB01`.
