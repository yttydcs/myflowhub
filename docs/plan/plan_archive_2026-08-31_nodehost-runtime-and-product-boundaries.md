# Plan Archive - 通用 NodeHost、资源声明与产品运行时收敛

## Workflow Information

- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `refactor/nodehost-runtime`
- Base: `master` @ `54b46e6bc0c1bbea5b265a875e7c8132ae28e004`
- Project Root: `D:\project\MyFlowHub3`
- Docs Root: `D:\project\MyFlowHub3\worktrees\nodehost-runtime\docs`
- Code Repos: `D:\project\MyFlowHub3\worktrees\nodehost-runtime`（单一 canonical monorepo）
- Worktree: `D:\project\MyFlowHub3\worktrees\nodehost-runtime`
- Current Stage: `$m-archive` 文档归档已完成；control-plane integration 与 worktree cleanup 待本次 closeout 收口

## Stage Records

### Initialization

- 已完整读取 active worktree 的 `guide.md`，确认 `GOWORK=off`、canonical docs 与 sibling worktree 规则。
- 已确认真实 owning repo 是 `repo/MyFlowHub`，本轮只有一个代码仓库和一个 docs root。
- 已从 `master` 创建专用分支与 worktree；主检出目录的未提交文档和设计草案保持不动。
- 主检出目录存在与 Desktop/Metrics 独立性和 Agent Gateway 相关的未提交稳定文档草案；它们仅作为本轮讨论证据，执行时不得覆盖，归档集成时按路径协调。

### Discuss - Discovery And Requirements Shaping

#### Goal

为所有可运行完整 Go runtime 的产品提供一个可直接创建的通用 `NodeHost`，统一身份、Node、父连接、Listeners、SDK Client 和生命周期；让调用方可以在 Host 上便捷注册并更新 Variable，同时保持 Desktop、Metrics、Agent Gateway 等产品身份和进程完全独立。

#### Scope

- `host/nodehost` 纯 Go 运行时组合层；
- Parent-only、Parent + Listeners、Listeners-only、offline 四种角色；
- `host.Client()` 返回绑定同一 Node 的非拥有型 `*sdk.Client`；
- `host.Resources()` 提供 owner-bound Registry 与便捷 Variable 声明；
- Desktop、Metrics Windows/CLI、Metrics Android 和通用 Android leaf Client 的首批迁移；
- 当前 owning SDK/binding API 的显式兼容与弃用边界。

不在本轮实现 Agent Gateway Web 服务、Desktop 中继 UI、安装包整合、浏览器完整 Node、Embedded Go runtime 或所有旧 Host 的一次性迁移。

#### Assumptions

- 产品名不产生网络特权；Node identity、拓扑和 policy 才决定权限。
- Desktop 与 Metrics 继续是不同 Node、不同进程、不同状态目录和不同安装包。
- NodeHost 不依赖 Wails、Kotlin、Android API、具体 Transport 或产品 controller。
- Android/iOS 可通过 gomobile 复用同一个 Go NodeHost；原生 Service/Activity 继续拥有操作系统生命周期。
- Host Client 的所有远端操作继续进入 `Node.Operate/Subscribe → routeEnvelope → Session.Send`，不建立旁路。

#### Open Questions

- 无阻塞性产品问题。导出类型的最终字段名可按 Go 周边命名微调，但本计划中的所有权、生命周期和角色约束不可改变。

#### Options Considered

1. 每个产品继续各自 `OpenState → node.New → sdk.NewClient → ConnectManaged`：拒绝，重复装配、回滚和关闭语义持续漂移。
2. 由 SDK Client 创建并拥有 Node/连接：拒绝，SDK 同时承担“运行节点”和“使用节点”两种职责，`Close` 会误关共享 Host。
3. 把 Android/Wails/产品 API 放进 NodeHost：拒绝，平台生命周期和产品能力会污染通用运行时。
4. 纯 Go NodeHost + 薄平台/产品适配器：采用；Host 管运行，SDK 管操作，产品注册能力，原生层管 OS 生命周期。
5. 本轮一次迁移 Hub、Clipboard、Agent Gateway、Embedded：延期；先用 Desktop/Metrics/Android 验证公共 API，再迁移需要管理 feature 或异构实现的产品。

#### Rejected Options

- 合并 Desktop 与 Metrics 产品、Node identity 或安装单元。
- 将 `nodeclient` 作为“无 Listeners 的 Host”单独包装。
- 让 `sdk/bindings` 长期拥有 NodeHost 生命周期。
- 为 Android 定义第二套节点、权限或资源模型。

#### Recommended Direction

建立 `host/nodehost` 作为唯一推荐的完整 Go 节点装配入口：`New` 创建持久状态和本地 Node、调用方注册资源、`Start` 激活 Parent/Listeners、`Close` 逆序幂等关闭。SDK Client 是 Host 创建的同一实例，只表达 Catalog/Snapshot/Operate/Invoke/Publish/Subscribe/Session 等操作，不拥有 Host。

#### Research Summary

- Desktop 当前通过 `sdk/bindings/desktop → sdk/bindings.Client` 间接创建状态、Node、SDK 和 ParentSupervisor。
- Metrics 当前在 `apps/nodes/metrics/runtime.go` 独立完成相同装配，并额外注册 controller 与 notification inbox。
- Android 同时存在通用 Client、in-process Hub Host 和 Metrics mobile Client 三条 runtime-owning 路径。
- Variable、Registry、revision、snapshot-first subscription 与更新通知已经实现；缺少 owner-bound 的便捷声明入口。
- `Session.Send` 已提供本地短路、路由复用、控制/数据有界队列、公平写循环和明确背压；Host Client 必须复用这条路径。

#### Worktree / Branch / Docs Root Status

- Dedicated worktree: ready。
- Branch: ready。
- Docs root: canonical repo `docs/`，ready。
- Main checkout dirty state: read-only and preserved；不阻塞本 worktree 规划与实现。

#### Issue List

- 无技术阻塞。

### Plan - Requirements And Architecture

#### Discussion Summary

NodeHost 是普通节点的运行时组合器，不是特权产品。配置决定叶子、中继、根或离线角色。产品 controller 通过 `host.Node()`/`host.Resources()` 注册能力；应用通过 `host.Client()` 使用本地或远端能力；外部 Codex 未来通过独立 Agent Gateway 的 Web/MCP API 访问，不直接实例化 Go Host。

#### Accepted / Rejected Requirements

- 接受：单一 Host 类型、可选一个 Parent、零个或多个 Listeners、同一 Node/Client、平台注入、动态资源注册、显式权限和有界生命周期。
- 接受：首批迁移 Desktop、Metrics 和相关 Android leaf/mobile 路径，保留产品独立。
- 拒绝：Desktop 默认获得中继能力、Metrics 依赖 Desktop、SDK 继续作为 canonical runtime owner、隐藏失败重试或静默权限回退。
- 延期：Hub/Clipboard 全迁移、Agent Gateway、浏览器/Embedded runtime、安装发布工作。

#### Requirements Analysis

##### Goal

调用方能够用一个稳定、平台中立的 Host API 创建节点、注册资源、启动拓扑连接并使用 SDK；首批产品不再重复 runtime 装配。

##### Scope

- 新增公共 Host contract 与资源声明辅助 API；
- 迁移首批产品并保持网络、权限、状态和用户行为兼容；
- 为后续 Hub/Clipboard/Agent Gateway 迁移保留扩展点，但不提前嵌入其 feature。

##### Use Cases

1. 叶节点配置 Parent、无 Listener，注册 Variable 后连接父节点。
2. 中继节点配置 Parent 和多个 Listener，复用同一 Node 路由上下游。
3. 根节点无 Parent、有 Listener，作为 authority root 接纳子节点。
4. 离线节点无 Parent/Listener，只运行本地资源与操作。
5. Desktop 从 `host.Client()` 发现和操作远端资源，不拥有 Metrics。
6. Metrics 在 Host 启动前注册指标资源，之后持续 `Variable.Set` 并向订阅者发布。
7. Android Kotlin Service 创建/关闭 gomobile facade，Go 内部仍是相同 NodeHost。

##### Functional Requirements

1. `NodeHost` 恰好拥有一个 `node.Node`、一个 Host 创建的 SDK Client、最多一个 ParentSupervisor 和任意数量 Listener。
2. `host.Client()` 每次返回同一非 owning SDK Client，不创建第二个 Node、连接或身份。
3. `host.Node()` 与 `host.Resources()` 指向同一 Node 的 Registry；注册会更新 `system/catalog`。
4. Parent 与 Listener 均为可选；四种角色组合必须通过 memory/TCP contract tests。
5. `New` 只完成配置、持久状态和本地 runtime 准备，不对外建立树边；资源可在 `Start` 前注册。
6. `Start` 一次性启动 Listeners 和 Parent supervision；部分失败必须逆序回滚并留下可行动错误。
7. `Close` 幂等，停止新工作、取消 Parent、关闭 Listeners/Node，并有界等待后台任务。
8. Parent trust/public key、permit、endpoint、Driver 和 supervisor 参数在边界严格验证；运行后不允许静默改 Parent identity。
9. Variable 便捷声明自动绑定 Host NodeID，同时要求显式 name、content type、schema、read permission 和 payload limit；远端可写必须显式声明 write permission。
10. `Variable.Set` 保留现有 revision、payload validation、watcher 与远端 subscription 行为。
11. SDK Client 不关闭或重配其所属 Host；旧 owning/connect API 只作为标记清晰的迁移兼容入口存在。
12. Desktop 与 Metrics 的 NodeID、state、credential、process 和 Resource ownership 保持独立。
13. Android 原生层继续负责 Foreground Service、权限、通知、Doze/network callback 和平台采集/执行；NodeHost 不导入平台包。

##### Non-functional Requirements

- 安全：不产生产品名特权、同机权限旁路、明文私钥回退或未认证 Listener 路由。
- 正确性：本地调用短路、远端调用继续走现有路由/队列；失败启动和重复关闭不泄漏 goroutine/session。
- 兼容：首批产品用户行为、Profile/配置 schema、Metrics resource descriptor 和网络协议保持不变；弃用 API 有明确迁移说明。
- 可维护：依赖方向保持 `runtime → host/sdk → apps`，NodeHost 不依赖产品和具体 Transport。
- 可测试：核心使用 memory Driver；产品保留真实 TCP、Wails、gomobile/Gradle 门禁。
- 性能：不增加第二套发送队列、额外序列化或每次操作新建连接；资源声明辅助层只做一次性验证与注册。

##### Inputs / Outputs

- 输入：state directory、NodeID、可选 IdentityStore、Node limits、Parent config、Listener configs、Resource specs。
- 输出：Host、Node、SDK Client、Registry、实际 Listener endpoints、Parent connection snapshot/changes、明确错误。

##### Edge Cases

- Parent 和 Listener 都为空；多个 Listener 中途失败；Parent 已信任/公钥冲突；无效或过期 permit。
- `Start` 重复调用、`Close` 在未启动/启动失败/正在连接时调用、Client 被误关。
- 同名资源重复注册、foreign owner、非法 schema/permission、初始 payload 超限、revision exhaustion。
- Parent 断线重连、reparent、连接状态观察者慢消费、Host 关闭时 pending operation/subscription。
- Android sticky restart 无 live session、应用后台被回收、RFCOMM provider/权限不可用。
- Desktop 失败连接后重试、Profile 切换和 protected identity 保持稳定。

##### Acceptance Criteria

- 公共示例能够完成 `New Host → 声明 Variable → Start → Set → 远端 Snapshot/Subscribe`。
- memory 测试覆盖 leaf/root/relay/offline，并证明 `host.Client()` 与 `host.Node()` 共用发送与路由机制。
- Desktop 和 Metrics 不再直接执行完整的 `OpenState → node.New → sdk.NewClient → ConnectManaged` 装配链。
- Metrics Windows/CLI/Android 继续发布相同 Resource catalog、更新和 notification 行为。
- Desktop 登录、失败重试、Profile 切换、订阅和 operation 行为无回归。
- Android Go tests、真实 gomobile AAR 生成、Gradle unit/lint/assemble 在工具可用时通过；无设备时 device smoke 明确 Unavailable。
- `GOWORK=off go test ./...`、相关 race、vet、generated contract freshness 与 `git diff --check` 通过。

#### Architecture Design

##### Overall Solution

```text
Product / platform shell
        │ registers controllers/resources
        ▼
host/nodehost.Host
  ├── persistent auth state / IdentityStore
  ├── runtime/node.Node
  ├── ParentSupervisor (0..1)
  ├── Listeners (0..n)
  ├── resource.Registry
  └── sdk.Client (same Node, non-owning)
        │
        ▼
Node.routeEnvelope → Session.Send → Driver/Pipe
```

##### Alternatives Considered

- 复用 `host/hub.Hub`：拒绝作为通用基类；它附带 Management/File/Flow/Notification feature，不是中性 Host。
- 只在 SDK binding 中包装 Node：拒绝作为目标；跨语言 facade 可以包 Host，但 canonical ownership 必须位于 `host/nodehost`。
- 给每种角色建立 `LeafHost/RelayHost/RootHost`：拒绝；角色只是 Parent/Listeners 配置组合。

##### Module Responsibilities

- `runtime/node`：协议、路由、LinkSession、Registry、operation/subscription 内核；不感知 Host 或产品。
- `runtime/resource`：Resource 实现与 owner-bound 声明辅助 API。
- `host/nodehost`：状态、Node、Parent、Listeners、生命周期、状态观察和 Client 所有权。
- `sdk/go`：稳定 typed operation facade；不拥有 Host。
- `sdk/bindings`：JSON/字符串/回调/跨语言转换；canonical 新路径不创建 Node。
- `apps/desktop`：Profile/CredentialStore、Host 生命周期、Wails facade 和 UI。
- `apps/nodes/metrics`：在 Host Node 上注册 controller，持有 collector/actuator/notification 产品逻辑。
- Kotlin/Windows shell：OS 生命周期、权限、系统通知和硬件 adapter。

##### Data / Call Flow

1. 产品读取并校验自己的配置，构造 NodeHost Config。
2. `nodehost.New` 打开状态、创建 Node 和 attached SDK Client。
3. 产品通过 `host.Node()` 或 `host.Resources()` 注册固定能力。
4. `host.Start` 启动 Listeners，再启动 Parent supervision；失败按逆序回滚。
5. 产品或 UI 通过 `host.Client()` 发起 typed operation；SDK 直接调用同一 Node。
6. 本地目标走 `handleLocal`；远端目标走 topology route 和现有 Session queues。
7. 产品更新 Variable 时触发 Registry observation/subscription；无需手工发送消息。
8. `host.Close` 先停止 supervision/外部入口，再关闭 Node 和 Host-owned task。

##### Interface Drafts

```go
type Config struct {
    StateDirectory string
    NodeID         protocol.NodeID
    IdentityStore  auth.IdentityStore // optional
    Runtime        RuntimeConfig
    Parent         *ParentConfig
    Listeners      []ListenerConfig
}

type Host struct { /* owns state, node, client, parent, listeners */ }

func New(ctx context.Context, config Config) (*Host, error)
func (h *Host) Start() error
func (h *Host) Node() *node.Node
func (h *Host) Client() *sdk.Client
func (h *Host) Resources() *resource.Registry
func (h *Host) Parent() ParentSnapshot
func (h *Host) Endpoints() []link.Endpoint
func (h *Host) Close() error
```

```go
host, err := nodehost.New(ctx, config)
if err != nil { return err }

status, err := host.Resources().Variable(resource.VariableSpec{
    Name:           "device/status",
    ContentType:    "application/json",
    Schema:         "device.status.v1",
    ReadPermission: "device.status.read",
    MaxPayload:     4096,
    Initial:        []byte(`{"online":true}`),
})
if err != nil { return err }

if err := host.Start(); err != nil { return err }
_, err = status.Set([]byte(`{"online":false}`))
```

最终字段名可服从周边惯例；以下语义固定：owner 自动取 Host NodeID、write permission 必须显式、Client 非 owning、Parent/Listeners 由 Host 拥有。

##### Error Handling and Safety

- 所有外部配置在创建 goroutine/listener 前验证；错误包含字段/Listener index/Parent endpoint。
- Listener 部分启动失败关闭已启动 Listener；Parent supervisor 创建失败关闭本轮 Host external state。
- SDK attached Client 的 `Close` 不得关闭 NodeHost；误用必须无状态破坏且返回明确错误或受限为 client-local cleanup。
- 不记录 private key、permit、payload；connection snapshot 只保留可操作摘要。
- 产品 controller 注册失败时先关闭已注册 controller，再关闭 Host；关闭顺序由 ownership 明确决定。
- 不引入静默重试、默认父节点、默认权限或平台能力 fallback。

##### Performance and Testing Strategy

- NodeHost 不增加数据面队列；operation 仍直接进入 Node。
- memory Driver 做四角色、回滚、重连、订阅和关闭 contract tests。
- focused Go tests 覆盖 Host、Resource、SDK、Desktop、Metrics 和 Android mobile。
- race 覆盖 Start/Close、连接变化、Variable Set/Subscribe 和产品 shutdown。
- 全仓 test/vet/generated，再执行 Desktop Wails 与 Metrics Android build matrix。

##### Extensibility Design Points

- 新 Transport 只提供 `link.Driver`；Host 配置不新增类型分支。
- iOS 可提供与 Android 相同的薄 lifecycle facade。
- Hub 后续可在 NodeHost 上注册 Management/File/Flow/Notification，而不是让 NodeHost 依赖这些 feature。
- Agent Gateway 后续作为普通 NodeHost 产品注册/使用资源，并在外层实现 token → 权限交集与审计。
- Embedded C/MicroPython 保持独立轻量实现，通过相同协议和 contract tests 对齐，不强行嵌入 Go。

#### Issue List

- 无。

### Stage 3.1 - Planning

#### Project Goal and Current State

底层 protocol、Node、Registry、SDK operation、routing 和 Session queue 已统一；Desktop、Metrics 和 Android 仍有多套 runtime 装配与所有权。当前 active worktree 只包含本 `plan.md`/`todo.md` 规划变更，没有业务实现。

#### Docs Governance Routing Decision

- `$m-docs` 分类结果：原始讨论进入 intake，长期技术 contract 进入 specs，SDK/Host 所有权选择进入 decisions，现有 requirement/feature 做最小澄清，执行过程留在根计划。
- 不在计划阶段提前创建 `docs/change`、`docs/plan` archive 或 lesson。
- 主检出目录现有未提交 Desktop/Metrics、Agent Gateway 草案不复制、不覆盖；DOC01 在 active branch 中按当前 master 与用户确认结论建立最小一致文档集，归档时再处理路径冲突。

#### Related Intake / Features / Requirements / Specs / Decisions / Lessons

- Intake: `docs/intake/2026-08-27_node-tree-subscription-command-redesign.md`；主检出草案 `docs/intake/2026-08-30_desktop-metrics-product-independence.md`
- Features: `docs/features/desktop.md`、`docs/features/metrics-node.md`、`docs/features/android.md`
- Requirements: `docs/requirements/unified-node-runtime.md`、`docs/requirements/extensible-resource-platform.md`；主检出草案 `docs/requirements/desktop-metrics-product-independence.md`
- Specs: `docs/specs/operational-lifecycle.md`、`docs/specs/node-tree-link-resource-architecture.md`、`docs/specs/resource-platform-v2.md`
- Decisions: `docs/decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md`；主检出草案 `docs/decisions/2026-08-30_desktop-metrics-independent-products.md`
- Lessons: `docs/lessons/android-runtime-and-mobile-bindings.md`、`docs/lessons/desktop-binding-reconnect-and-admission-diagnostics.md`

#### Stable Docs Impact

- Intake impact: add — 新增本轮 NodeHost/SDK/跨平台讨论保真摘要。
- Feature impact: clarify — Desktop、Metrics、Android 改为共享 NodeHost，但产品行为和独立边界不变。
- Requirements impact: clarify — 在统一节点运行时中明确 Host composition 与产品无特权原则。
- Specs impact: add/clarify — 新增稳定 `node-host-runtime.md`，并校准 lifecycle/resource platform 链接。
- Decision impact: add — 记录“通用 NodeHost + 非 owning SDK Client + 薄平台适配器”。
- Lessons known at planning time: 复用现有 Android lifecycle 与 Desktop reconnect lessons；执行/测试后再判断是否出现可复用新 lesson。

#### Executable Task List

| Task ID | Title | Scope | Depends On |
| --- | --- | --- | --- |
| DOC01 | 稳定文档与架构契约 | Will execute | none |
| HOST01 | 通用 NodeHost 生命周期与拓扑角色 | Will execute | DOC01 |
| RES01 | Owner-bound Variable 便捷声明 | Will execute | DOC01 |
| SDK01 | 非 owning Host Client 与 SDK/binding 边界 | Will execute | HOST01 |
| DESK01 | Desktop leaf runtime 迁移 | Will execute | HOST01, SDK01 |
| METR01 | Metrics Windows/CLI runtime 迁移 | Will execute | HOST01, RES01, SDK01 |
| ANDR01 | Android leaf/mobile Host 适配与构建契约 | Will execute | METR01, SDK01 |
| QA01 | 集成、race、生成与产品构建门禁 | Will execute | DESK01, METR01, ANDR01 |
| HUB01 | Hub 与 Android in-process Hub 迁移 | Will not execute now | Deferred: 需要先验证 Management/File/Flow 的 Host 扩展边界 |
| CLIP01 | Clipboard runtime 迁移 | Will not execute now | Deferred: 复用本轮稳定 API，独立控制同步/bridge 风险 |
| AGW01 | Agent Gateway Web/API/token 服务 | Will not execute now | Deferred: 产品已定义，等待接口和权限模型整体稳定 |
| WEB01 | 浏览器 WASM/WebSocket 完整 Node | Will not execute now | Research-only: 当前外部 Web 访问优先走 Gateway |
| EMB01 | C/MicroPython/ESP32 Runtime 统一实现 | Will not execute now | Out of scope: 保持异构实现和 wire contract 对齐 |
| INSTALL01 | Desktop/Metrics 独立安装升级卸载 | Will not execute now | Separate release workflow |
| ARC01 | change/plan 归档、提交、合并、清理 | Will not execute now | 仅在实现和测试通过后由 `$m-archive` 执行 |
| PUB01 | push/release/publication | Will not execute now | 未授权 |

#### Execution Scope After Approval

##### Will Execute

- `DOC01, HOST01, RES01, SDK01, DESK01, METR01, ANDR01, QA01`

##### Will Not Execute Now

- `HUB01`：Hub 的 audit policy、Management/File/Flow 注册和多 Listener 扩展需在核心 API 验证后单独迁移。
- `CLIP01`：Clipboard 同步、bridge 和多平台生命周期另设迁移阶段。
- `AGW01`：产品定义已经确认，但本轮不实现 Web/MCP/API/token 服务。
- `WEB01`：浏览器完整 Node 尚需 WASM/WebSocket 与安全研究。
- `EMB01`：Embedded 不直接嵌入 Go Host，只保持协议一致性。
- `INSTALL01`：安装发布边界是单独 workflow。
- `ARC01`：属于完成后的 `$m-archive`。
- `PUB01`：没有 push/release 授权。

#### Task Details

##### DOC01 - 稳定文档与架构契约

- Owner: primary agent
- Worktree: `D:\project\MyFlowHub3\worktrees\nodehost-runtime`
- Plan Path: root `plan.md` / `todo.md`
- Goal: 固化 NodeHost、SDK ownership、平台适配与产品独立边界，并建立可追溯索引。
- Files / Modules: `docs/intake/**`、`docs/features/{desktop,metrics-node,android}.md`、`docs/requirements/unified-node-runtime.md`、`docs/specs/{node-host-runtime,operational-lifecycle,resource-platform-v2}.md`、`docs/decisions/**` 及分类 README。
- Write Set: 仅上述稳定文档与索引；不创建 change/plan archive。
- Acceptance: Host/SDK/Node/Resource/平台/产品职责无冲突；相关草案链接和 supersession 清楚；索引可导航。
- Test Points: relative link/path 检查、术语搜索、`git diff --check`。
- Rollback: 回退 DOC01 文档 patch，不影响 runtime。

##### HOST01 - 通用 NodeHost 生命周期与拓扑角色

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 实现可组合、单次启动、幂等关闭的纯 Go Host，并暴露同一 Node/Client/Registry。
- Files / Modules: 新增 `host/nodehost/**`；必要时最小调整 `host/config`、`runtime/node` 的只读状态/关闭接口；新增 focused tests。
- Write Set: Host 组合层和必要的 runtime seam；不导入产品、Wails、Android 或具体 Transport。
- Acceptance: root/relay/leaf/offline、multiple listeners、Parent supervision、status、rollback、IdentityStore 和 pointer identity 均通过。
- Test Points: memory Driver contract、listener partial failure、reconnect/reparent、Start/Close race、goroutine cleanup。
- Rollback: 删除新 package 和 seam；现有产品仍走旧装配。

##### RES01 - Owner-bound Variable 便捷声明

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 让调用方从 Host Registry 直接声明本节点 Variable，无需重复填写 owner 和完整 Descriptor boilerplate。
- Files / Modules: `runtime/resource/**`、`host/nodehost` 暴露 Registry 的薄入口、tests/examples。
- Write Set: Variable spec/builder、Registry registration helper 和测试；不改变 wire descriptor schema。
- Acceptance: read-only 默认、write 显式、owner 自动、catalog 原子更新、Set/revision/subscription 行为保持。
- Test Points: duplicate/foreign owner、schema/permission/limit、initial payload、dynamic registration、concurrent Set/Observe。
- Rollback: 移除 helper，底层 `NewVariable + Registry.Register` 保持可用。

##### SDK01 - 非 owning Host Client 与 SDK/binding 边界

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: Host Client 只负责操作面，不关闭 Host 或另建连接；为旧 owning API 提供显式、可删除的迁移边界。
- Files / Modules: `sdk/go/{client,connection,durable,features}.go`、`sdk/bindings/**`、generated contracts 与 tests。
- Write Set: attached/non-owning constructor 与 ownership guard、Host connection status adapter、binding operation facade；保留旧 API 时必须标注 deprecated 和 owner 语义。
- Acceptance: `host.Client()` 是同一 `*sdk.Client`；误调 Client close 不影响 Host；Catalog/Operate/Subscribe/File/Flow 行为不变；旧调用方编译且没有静默生命周期变化。
- Test Points: local/remote operation path、client-close guard、subscription cancellation、connection status、binding contract freshness。
- Rollback: Host 暂时直接使用 Node operation；恢复旧 SDK ownership，不迁移产品。

##### DESK01 - Desktop leaf runtime 迁移

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: Desktop App/Profile runtime 显式拥有 NodeHost，desktop binding 退化为非 owning JSON/订阅 facade。
- Files / Modules: `apps/desktop/**`、`sdk/bindings/desktop/**`、必要的 `cmd/mfh-desktop` tests/docs。
- Write Set: Profile runtime factory、Host lifecycle、binding adapter、connection status和测试；不修改 UI 产品职责或加入 Listener。
- Acceptance: Desktop 仍是 Parent-only leaf；DPAPI identity、prepare/login、失败重试、切换 Profile、订阅/operation 和 shutdown 无回归。
- Test Points: existing Desktop tests、失败连接→修正→成功、Profile isolation、race、Wails production build。
- Rollback: 恢复 disposable binding client factory；Profile/state 数据格式不迁移。

##### METR01 - Metrics Windows/CLI runtime 迁移

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: `metrics.Runtime` 组合 NodeHost，在启动网络前注册 controller，并从 Host Client 建立 notification inbox。
- Files / Modules: `apps/nodes/metrics/{runtime,controller,notifications}.go`、Windows backend、`cmd/mfh-metrics`、tests。
- Write Set: runtime ownership/close/status，必要的 controller resource helper 使用；不改变指标 schema、采集器或 actuator 语义。
- Acceptance: Metrics 是独立 Parent-only Node；catalog、采样、配置、控制和 notification 行为保持；失败回滚无泄漏。
- Test Points: fake collector/actuator、真实 TCP process test、reconnect/durable notification、Windows platform smoke、race。
- Rollback: 恢复现有 Metrics runtime 装配；持久 state/config 保持兼容。

##### ANDR01 - Android leaf/mobile Host 适配与构建契约

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 通用 Android Client 和 Metrics mobile facade 使用同一 NodeHost contract，Kotlin Service 继续拥有 OS lifecycle。
- Files / Modules: `sdk/bindings/android/{client,provider_*}.go`、`apps/nodes/metrics/android/mobile/**`、Metrics `NodeService.kt`/tests、generated AAR contracts、`scripts/mfh.ps1` 必要最小调整。
- Write Set: gomobile-friendly facade、platform injection/lifecycle mapping、tests；不迁移 Android in-process Hub，不引入平台代码到 NodeHost。
- Acceptance: TCP/RFCOMM leaf、sticky restart、Start/Stop、action queue、identity/state 和 error mapping 无回归；真实 AAR 方法与 ABI 可验证。
- Test Points: Go Android cross-compile、gomobile AAR、Gradle unit/lint/assemble；有设备才执行 device smoke，否则标记 Unavailable。
- Rollback: 恢复旧 mobile wrappers；Kotlin settings 和 identity 不迁移。

##### QA01 - 集成、race、生成与产品构建门禁

- Owner: primary agent
- Worktree: active worktree
- Plan Path: root `plan.md`
- Goal: 验证公共 API、产品兼容、并发关闭和跨平台生成物。
- Files / Modules: focused tests、`tests/integration/**`、generated contracts/dist（仅 canonical 生成入口产生）、必要的测试证据。
- Write Set: 测试和由正式生成命令产生的 tracked artifact；不写 change archive、不修改主检出。
- Acceptance: 公共示例和全仓门禁通过；没有 runtime duplication regression、资源/权限旁路或 ownership leak。
- Test Points:
  - `$env:GOWORK='off'; go test ./host/nodehost/... ./runtime/resource/... ./sdk/... ./apps/desktop/... ./apps/nodes/metrics/... -count=1`
  - `$env:GOWORK='off'; go test -race ./host/nodehost/... ./runtime/resource/... ./sdk/go/... ./apps/nodes/metrics/...`
  - `$env:GOWORK='off'; go test ./... -count=1`
  - `$env:GOWORK='off'; go vet ./...`
  - `go generate ./sdk/bindings` + freshness check
  - `./scripts/mfh.ps1 -Action test -Target desktop`
  - `./scripts/mfh.ps1 -Action build -Target desktop`
  - `./scripts/mfh.ps1 -Action test -Target metrics`
  - `./scripts/mfh.ps1 -Action build -Target metrics`
  - Android/gomobile 工具或设备不可用时按 build spec 明确记录 Unavailable，不伪造通过
  - `git diff --check`
- Rollback: 按任务边界依次回退产品迁移、SDK、Resource helper、NodeHost；不触碰持久用户数据。

#### Dependencies

```text
DOC01
 ├── HOST01 ── SDK01 ── DESK01 ─┐
 │                 └── METR01 ── ANDR01 ── QA01
 └── RES01 ────────────┘
```

- HOST01 先固定 ownership/lifecycle；RES01 可在其接口确定后实现。
- SDK01 是产品迁移的共同前置，避免 Desktop/Metrics 各自定义 Client ownership。
- Android 依赖 Metrics 与 SDK facade 收敛后再生成 AAR，避免重复 binding drift。

#### Risks and Notes

- `New` 与 `Start` 两阶段必须避免“Node 已创建但产品以为完全停止”的模糊状态；spec 和状态枚举必须一致。
- attached Client 与旧 owning Client 共存期间最容易出现误关 Host；必须有 ownership tests 和弃用说明。
- Desktop reconnect 不能复用失败后的单次生命周期对象，也不能丢失 ParentSupervisor 的最后错误。
- Android 编译成功不等于真实 AAR/JNI/Service 可用；必须检查 class、ABI 和 lifecycle。
- 当前主检出有未提交文档草案；本分支不得覆盖，归档/merge 需要路径级协调。
- 不为延期的 Hub/Clipboard 预先加入 product-specific callback；缺口由后续迁移反馈驱动。

#### Parallelism Assessment

- 本节记录规划时的初始判断，已由用户显式批准的 `$m-go` delegated execution 取代。
- 实施按 Task ID、依赖与不重叠 write set 委派；共享 exported API 的 HOST01/SDK01/产品迁移仍按依赖收敛后再并行。

#### Issue List

- 无。

### Stage 3.2 - Approved Delegated Execution

- User approval: `批准以上 Task IDs，使用 $m-go 执行`。
- Approved execution Task IDs: `DOC01, HOST01, RES01, SDK01, DESK01, METR01, ANDR01, QA01`。
- Technical blockers: none。
- Execution mode: `$m-go`；所有实现编辑按 Task ID 委派，主 agent 负责协调、审计与自动 `$m-test` 循环。
- Deferred scope remains: `HUB01, CLIP01, AGW01, WEB01, EMB01, INSTALL01, ARC01, PUB01`。
- DOC01: completed；intake、NodeHost spec、ADR、feature/requirement clarification、分类索引与 approval gate 已验证。

### Stage 3.3 - Heavy Validation Result

- Completed Task IDs: `DOC01, HOST01, RES01, SDK01, DESK01, METR01, ANDR01, QA01`。
- Focused/full validation: focused Go、focused race、`go test ./... -count=1`、`go vet ./...`、architecture/docs links、generated binding freshness 与 `git diff --check` 全部通过。
- Desktop product: Go tests、54 个 frontend tests、TypeScript/Vite build 与 Windows/amd64 Wails production build 通过。
- Metrics product: Go tests、2 个 frontend tests、TypeScript/Vite build、Windows/amd64 Wails production build 通过。
- Android artifacts: generic Android 与 Metrics gomobile AAR 生成通过；两者均包含 `arm64-v8a` 与 `x86_64`，Java ABI 已用 `javap` 核对。
- Environment Unavailable: generic Android 与 Metrics 的 Gradle `testDebugUnitTest`、`lintDebug`、`assembleDebug` 均在 single-use daemon 启动时失败，exact cause 为 `java.io.IOException: Unable to establish loopback connection`；不记为通过。
- Device Unavailable: 本机未安装 `adb`，没有 Android 设备 smoke 证据；不记为通过。
- Residual: generic Android Client 与 Metrics RuntimeConfig 当前仍使用默认 identity persistence；Keystore-backed `IdentityStore` 注入仍是后续 seam。
- Review verdict: 无实现阻塞，允许携带上述明确环境盲区和后续 seam 进入 `$m-archive`；本阶段不 archive、commit、merge、push 或清理 worktree。

## Approval Gate

- Planned execution Task IDs: `DOC01, HOST01, RES01, SDK01, DESK01, METR01, ANDR01, QA01`
- Technical blockers: none
- Approved execution Task IDs: `DOC01, HOST01, RES01, SDK01, DESK01, METR01, ANDR01, QA01`
- Blocked: no
- Implementation completed: yes — `$m-go` delegated execution and heavy validation
- Archive readiness: completed by `$m-archive`
- Archive documentation completed: yes
- Local integration/cleanup: completed
- Push/publication: not authorized

## Archive Routing

- Change: [NodeHost runtime 与产品边界收敛](../change/2026-08-31_nodehost-runtime-and-product-boundaries.md)
- Lesson: [Android Gradle daemon loopback 不可用](../lessons/android-gradle-loopback-daemon-unavailable.md)
- Stable docs: intake、Desktop/Metrics/Android features、统一运行时 requirement、NodeHost/lifecycle/resource/repository specs 与 NodeHost ADR 均已更新。
