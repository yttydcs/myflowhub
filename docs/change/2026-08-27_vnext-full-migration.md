# 2026-08-27 vNext 全量迁移与旧仓退役

## 变更背景 / 目标

统一节点运行时第一阶段只证明了新核心闭环，Desktop、Android、MetricsNode、ClipboardNode、EmbeddedSDK、File、Flow、真实 Transport 与构建发布面仍分散在旧多仓/SubProto 体系。项目没有外部兼容用户，因此本 workflow 以 clean break 完成全部第一方源码迁移，使一个 canonical monorepo 成为唯一开发、构建和运行真相。

## 具体变更内容

- 固定 10 个旧仓的 commit/tree/file count，为全部 capability 和 build entry 记录 `migrate / replace / drop` 终态。
- 完成持久 identity、parent trust、一次性 admission permit、policy generation、revoke、审计和 default-deny Hub。
- 完成 supervised reconnect、reparent、跨子树授权路由、转发 Subscription 清理及 policy/topology generation 失效。
- 在统一 `link.Driver` 上实现 TCP、QUIC 和 RFCOMM；物理链路不进入 Resource 或权限模型。
- 以 Variable、Stream、Command 取代 VarStore、TopicBus、Exec 和 SubProto dispatcher，并完成 catalog、Management、Notification、File 与 Flow。
- 完成 Go SDK、generated contract、gomobile/Wails/Flutter 边界，以及 Hub、管理 CLI、Desktop、Android、MetricsNode、ClipboardNode 产品迁移。
- 完成 C99、MicroPython 和 ESP32 leaf profile；ESP-IDF/真板证据按主机可用性显式标记。
- 建立统一 `scripts/mfh.ps1`、无 path filter CI、工具链约束、artifact manifest 和 canonical `run-dev.ps1`。
- 完成跨产品 default-deny/allow/revoke、订阅与 Command 的集成、race、fuzz、soak 和真实进程门禁。
- 审计 777 份旧仓 Markdown，提炼长期知识后移除 10 个本地旧仓；不恢复旧 API 或多仓文档为当前真相。

## Docs root

- workflow 写入根：`D:/project/MyFlowHub3/worktrees/vnext-full-migration/docs`
- 本地合并后的 canonical 根：`D:/project/MyFlowHub3/docs`
- docs 与代码属于同一 Git 仓；本次 closeout 不配置远端、不推送、不发布。

## Intake impact

- `updated`
- 新增全量迁移来源记录，并回链本归档。

## Feature impact

- `updated`
- Hub、Desktop、Android、MetricsNode、ClipboardNode、Embedded、File、Flow 均建立迁移后的当前行为 dossier。

## Requirements impact

- `updated`
- 统一节点运行时、受控准入和节点显示名需求已映射到 canonical runtime/resource contract。

## Specs impact

- `updated`
- 节点树、仓库边界、wire、三资源、订阅、Command、lifecycle、catalog、Management、Notification、File、Flow、build/CI 与协议映射均已更新。

## Decision impact

- `updated`
- 单一 canonical monorepo、统一权威节点树/可插拔链路、单迁移分支分门禁切换三项决策进入已实施状态。

## Lessons impact

- `updated`
- 新增 Android/mobile binding、Embedded/toolchain、authority/routing/subscription、frontend/PowerShell、observable side effects/generated contracts 五类可复用经验；旧多仓 lessons 改写为 canonical 规则。

## Related intake

- [节点树、订阅与指令重构诉求](../intake/2026-08-27_node-tree-subscription-command-redesign.md)
- [vNext 全量迁移与最终切换](../intake/2026-08-27_vnext-full-migration.md)

## Related features

- [Hub](../features/hub.md)
- [Desktop](../features/desktop.md)
- [Android](../features/android.md)
- [MetricsNode](../features/metrics-node.md)
- [ClipboardNode](../features/clipboard-node.md)
- [Embedded leaf SDK](../features/embedded.md)
- [File transfer](../features/file-transfer.md)
- [Flow](../features/flow.md)

## Related requirements

- [统一节点运行时](../requirements/unified-node-runtime.md)
- [受控准入](../requirements/auth-controlled-admission.md)
- [节点显示名](../requirements/management-node-display-name.md)

## Related specs

- [节点树、链路与资源架构](../specs/node-tree-link-resource-architecture.md)
- [仓库与模块边界](../specs/repository-and-module-boundaries.md)
- [Wire Protocol vNext](../specs/wire-protocol-vnext.md)
- [Resource Model vNext](../specs/resource-model-vnext.md)
- [Subscription vNext](../specs/subscription-vnext.md)
- [Command vNext](../specs/command-vnext.md)
- [运行生命周期](../specs/operational-lifecycle.md)
- [构建与 CI](../specs/build-and-ci.md)

## Related decisions

- [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)
- [单迁移分支、分门禁实施与最终一次切换](../decisions/2026-08-27_gated-full-migration-cutover.md)

## Related lessons

- [Android runtime 与 mobile bindings](../lessons/android-runtime-and-mobile-bindings.md)
- [Embedded toolchain 与真板预检](../lessons/embedded-toolchain-and-board-preflight.md)
- [Authority、路由与订阅状态](../lessons/authority-routing-and-subscription-state.md)
- [Frontend 与 PowerShell 预检](../lessons/frontend-and-powershell-preflight.md)
- [可观察副作用与生成 contract](../lessons/observable-side-effects-and-generated-contracts.md)
- [会话替换的 generation cleanup](../lessons/session-replacement-generation-cleanup.md)

## 对应 plan.md 任务映射

| Task | 结果 |
| --- | --- |
| FM00–FM02 | 来源审计、处置矩阵、稳定产品/技术契约和协议 schema 完成 |
| FM03–FM05 | identity/admission/policy、supervisor、QUIC/RFCOMM 完成 |
| FM06–FM08 | 生产 Hub、Management/File/Flow/Notification、SDK/bindings 完成 |
| FM09–FM13 | Metrics、Clipboard、Desktop、Android、Embedded 全量迁移完成 |
| FM14–FM16 | build/CI、跨产品门禁、canonical 本地切换完成 |
| DX04a | 旧仓文档提炼和 10 个本地旧仓移除完成 |
| DX01–DX03、DX04b、DX05 | 新 Transport、compat bridge、远端发布、main dirt 清理、不可用硬件认证仍不在本次授权内 |

完整计划和 Checklist 快照见 [plan_archive_2026-08-27_vnext-full-migration.md](../plan/plan_archive_2026-08-27_vnext-full-migration.md)。

## 经验 / 教训摘要

- 可替换实现应位于 Driver、Store、Platform adapter 等窄边界；不能为“任意替换”复制 wire、权限、路由和业务 dispatcher。
- durable identity/config/desired subscription 与 live session/route/attached interest 必须分离；重连只复用前者。
- 转发必须保留 `Principal`、deadline、message/correlation ID、revision/sequence 和 generation；cache 或 next hop 不能成为 authority 证明。
- 所有替代 mutation path 必须保留 Variable revision、Stream sequence/gap、subscription delivery 和 audit side effects。
- 平台生成物只能由 canonical schema/facade 重建；复制旧 AAR/WailsJS 或依赖 stub 会制造假通过。
- 构建环境不可用、平台工具缺失和源码失败必须分开记录；没有设备就不伪报硬件证据。

## 可复用排查线索

- 症状：重连后订阅仍挂在旧链路、reparent 后旧父恢复控制、撤权后跨子树 owner 仍继续推送。
- 触发条件：desired/attached 状态混用、cleanup 未绑定 epoch/generation、转发记录缺少原始主体或 policy generation。
- 关键词：`TopologyEpoch`、`PolicyGeneration`、`Principal`、`CleanupLink`、`StreamGap`、`late result`、`join topology epoch mismatch`。
- 快速检查：先核对 `system/catalog`、session generation、资源 owner、授权主体和 pending/forwarded subscription 数量，再检查 UI/平台 adapter。
- 症状：前端类型缺方法、AAR 可编译但运行崩溃、Windows 脚本显示完成但没有产物。
- 触发条件：复制 foreign bindings、stub fallback、空/链接 `node_modules`、未检查 `$LASTEXITCODE`、PowerShell 5.1 编码差异。
- 快速检查：从干净 worktree 运行 canonical generate，核对 JNI ABI/导出方法/contract diff 和实际产物 hash。

## 关键设计决策与权衡

- 节点父子树同时承担当前 authority 和路由边；资源挂在节点下，Transport 是可替换承载，不建立第二棵资源/权限树。
- Subscription 与 Command 是核心 API，Variable/Stream 优先表达状态和事件，Command 只补足有界主动操作。
- 一个仓库、一个 Go module、一个版本图取代内部 SemVer/tag 链；package 依赖守卫承担边界约束。
- clean break 删除旧 wire/SubProto 兼容层，换取更小的长期认知和发布成本。
- ESP32 真板、Android/RFCOMM 设备、macOS/iOS 签名证据保持 Unavailable/Out of scope，不用 host mock 冒充。

## 测试与验证方式 / 结果

- `GOWORK=off go test ./...`、`go vet ./...` 通过。
- runtime/transport/feature/host/SDK/products/integration 的 race 门禁通过；supervisor/reconnect 和跨产品 focused soak 通过。
- protocol fuzz 10 秒执行 2,247,055 次；完整 frame golden fixture 在 Go/C/MicroPython 间 byte-for-byte 一致。
- Desktop Vitest/Vite/Wails production build、Metrics Wails、Android gomobile AAR/Gradle、Clipboard Flutter/Windows/Web/Android、C/MicroPython 构建测试通过。
- 三个 AAR 均验证 arm64-v8a/x86_64 `libgojni.so`；64 项 artifact manifest hash 复验通过。
- archive revalidation：`go test ./internal/archtest ./internal/migrationtest`、Markdown 相对链接检查和 `git diff --check` 通过。
- Unavailable：当前主机无 `idf.py`/ESP32 板卡、无可用 Android/RFCOMM 真机、无 macOS/iOS 签名环境；未计为失败或通过。

## 潜在影响

- 新 module、wire 和 API 与旧系统不兼容；这是没有外部用户前提下已接受的 clean break。
- 远端 push、tag/release、签名、商店发布和旧仓远端归档未执行，当前结果是本地 Git 状态。
- `master` 控制工作区存在 workflow 之前的无关脏文件；closeout 必须按路径保存并恢复，不能用 reset/checkout 覆盖。
- 旧 MetricsNode 的 29 个未提交 Wails 生成文件只保留 hash，不保留正文；它们不属于 canonical source。

## 回滚方案

- 合并后使用 Git revert 回滚本 workflow commit，不恢复旧 SubProto 或多 module 构建链。
- 旧仓已提交内容按 `migration/sources.yaml` 的 remote/commit 临时 checkout；9 个目录在回收站未清空时也可恢复。
- `MyFlowHub-Win` 本地目录因长路径永久删除，但其干净 commit 可从远端恢复。
- main checkout 的无关 dirt 在 closeout 前后按精确路径和 hash 验证，保留为未提交状态。

## 子Agent执行轨迹

- 未使用子Agent。
- 原因：用户未要求委派，且当前宿主策略禁止主动多 Agent 执行；FM00–FM16、DX04a、验证和归档均由主 Agent 串行完成。
