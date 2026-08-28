# 2026-08-29 可扩展资源平台与 Desktop 工作区重构

## 变更背景 / 目标

上一版已经把多仓系统收敛为单一节点运行时，但资源仍受固定类型和旧 wire 语义约束，Desktop 也只是面向少数功能的简陋页面。此次在没有外部用户和兼容负担的前提下做第二次 clean break：保留唯一 authoritative Node tree、父控子 authority 与可插拔链路，把“节点下的一切都是资源”落实为可扩展类型与 capability 模型，并把订阅和指令提升为一级能力；同时将 Desktop 重建为可登录、多 Profile、可浏览、预览、组合并保存 View 的通用资源工作台。

## 具体变更内容

- 协议与运行时改为 extensible `ResourceDescriptor`、generic operation/subscription/session wire；Variable、Stream、Topic、Command 和 File 通过统一 registry、auth、routing 与 catalog 工作。
- 保留唯一 Node tree 作为物理父子、authority 和资源归属边界；Transport 仍可替换为 Memory、TCP、QUIC、RFCOMM 等实现。
- Topic 改为 Node-owned、多 publisher、多 subscriber 资源，按 publisher 独立 sequence，默认不 replay；File 使用绑定 subject/resource/capability/link/generation/expiry 的 resource session 与 data lane。
- Go SDK、generated bindings、Android、Embedded C/MicroPython 和第一方产品 clean-break 迁移，不保留 fixed-kind、SubProto、TopicBus 或旧 wire bridge。
- Desktop 改为 React、TypeScript、Vite 与 shadcn-style/Radix primitives，提供受保护身份、多 Profile、首次准入、自动连接、Node/Resource Explorer、renderer registry、预览、拖放与键盘添加、12 列 View 布局和原子持久化。
- UI 重测补齐失败连接重试、未保存 View 保护、删除确认、深层搜索、部分目录错误、File 双提交保护、订阅停止状态、暗色对比度、滚动底部遮挡、键盘/ARIA 语义与大列表热点优化。
- current feature、requirements、specs 与 ADR 切换到新模型；生产 Media、View sync、多活动 Profile、移动资源工作区和 legacy compatibility 明确留在本轮范围外。

## Docs root

- `D:/project/MyFlowHub3/repo/MyFlowHub/docs`
- 代码与受治理文档位于同一个 canonical monorepo；本地没有配置 remote，本轮不 push、不发布。

## Intake impact

- Intake impact: updated
- 新增并落实 [可扩展资源与 Desktop 工作区重设计 intake](../intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)。

## Feature impact

- Feature impact: updated
- [Desktop](../features/desktop.md)、[File transfer](../features/file-transfer.md) 与 [Embedded](../features/embedded.md) 已改为当前可扩展资源模型的用户可见事实。

## Requirements impact

- Requirements impact: updated
- 新增 [可扩展资源平台](../requirements/extensible-resource-platform.md) 与 [Desktop 资源工作区](../requirements/desktop-resource-workspace.md) 需求，沿用统一节点树与准入边界。

## Specs impact

- Specs impact: updated
- resource model、catalog、platform、session、subscription、command、file、wire 与 Desktop workspace specs 已同步为 current truth，并更新 protocol map。

## Decision impact

- Decision impact: updated
- 新增 [可扩展资源类型系统与 Desktop 工作区](../decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md) ADR；不取代 authoritative Node tree 决策。

## Lessons impact

- Lessons impact: updated
- 新增 [Desktop binding 重连与准入诊断](../lessons/desktop-binding-reconnect-and-admission-diagnostics.md)，记录单次生命周期 client、失败重试、permit 与 last-error 保留规则。

## Related intake

- [可扩展资源与 Desktop 工作区重设计](../intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)

## Related features

- [Desktop](../features/desktop.md)
- [File transfer](../features/file-transfer.md)
- [Embedded](../features/embedded.md)

## Related requirements

- [可扩展资源平台](../requirements/extensible-resource-platform.md)
- [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)
- [统一节点运行时](../requirements/unified-node-runtime.md)
- [受控准入](../requirements/auth-controlled-admission.md)

## Related specs

- [Node tree、link 与 resource architecture](../specs/node-tree-link-resource-architecture.md)
- [Resource model vNext](../specs/resource-model-vnext.md)
- [Resource platform v2](../specs/resource-platform-v2.md)
- [Resource catalog](../specs/resource-catalog.md)
- [Subscription vNext](../specs/subscription-vnext.md)
- [Command vNext](../specs/command-vnext.md)
- [Resource sessions v2](../specs/resource-sessions-v2.md)
- [File transfer vNext](../specs/file-transfer-vnext.md)
- [Desktop resource workspace v2](../specs/desktop-resource-workspace-v2.md)
- [Wire protocol vNext](../specs/wire-protocol-vnext.md)
- [Protocol map](../specs/protocol_map.md)

## Related decisions

- [可扩展资源类型系统与 Desktop 工作区](../decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)
- [Authoritative Node tree 与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)

## Related lessons

- [Desktop binding 重连与准入诊断](../lessons/desktop-binding-reconnect-and-admission-diagnostics.md)
- [Authority routing 与 subscription state](../lessons/authority-routing-and-subscription-state.md)
- [Session replacement generation cleanup](../lessons/session-replacement-generation-cleanup.md)
- [Generated contracts 与 observable side effects](../lessons/observable-side-effects-and-generated-contracts.md)

## 对应 plan.md 任务映射

- DOC01：target specs、supersession chain 与 feature transition skeleton。
- RES01-RES04：descriptor/wire major、registry、Topic、auth/routing/subscription/catalog 泛化。
- SES01：resource sessions、control/data lanes 与 File 迁移。
- SDK01、APP01：SDK/bindings/generated/Embedded 与第一方产品迁移。
- UI01-UI05：React/shadcn foundation、Profile/credential/login、Explorer/renderer、Workspace/View、内置 renderer 与 accessibility。
- VAL01：旧实现清理、全产品门禁、current-truth docs 切换。
- TST01-TST04：准入诊断、空画布布局、Explorer 滚动隔离与定量性能门禁。
- 完整计划：[plan_archive_2026-08-29_extensible-resource-platform-desktop-workspace.md](../plan/plan_archive_2026-08-29_extensible-resource-platform-desktop-workspace.md)

## 经验 / 教训摘要

- 可扩展类型不能只替换 enum；descriptor、capability、auth、routing、catalog、subscription 和 SDK 必须共享同一事实源，否则扩展点仍会退化为跨层 type switch。
- 大数据传输不能伪装成普通事件；File 需要可撤销、有界、与 topology/policy generation 绑定的 session data lane。
- UI 的“能够冷启动连接”不足以证明登录生命周期正确，必须覆盖失败后修正并重试、成功后清理 permit、重启自动连接和 Profile 切换。
- 大规模树浏览器既要测构建/过滤耗时，也要在真实窗口检查高度、overflow、底部状态栏和暗色继承色；组件测试无法替代 production Wails 走查。

## 可复用排查线索

- 症状：连接失败后重试出现 `parent trust cannot change after the binding client starts`；触发：复用已启动 binding client；关键词：`last_error`、`permit`、`context deadline exceeded`；快速检查：确认每次 `Connect` 使用新候选 client，并在成功前保留 permit。
- 症状：长 Resource 列表末行被状态栏遮挡；触发：Tabs/ScrollArea/Explorer footer 共享错误高度或 overflow；关键词：`min-height: 0`、`overflow`、`padding-bottom`、`z-index`；快速检查：真实 production 窗口滚动到底部。
- 症状：Explorer 大数据集卡顿；触发：render/filter 热路径反复线性查找；关键词：O(n²)、`Map`、2,000 Nodes、10,000 Resources；快速检查：固定代表性 fixture 并记录 build+filter 阈值。
- 症状：大文件传输影响 heartbeat/command；触发：control/data 共用无界或不公平队列；关键词：bounded queue、fair priority、session revoke；快速检查：memory/TCP 跨子树事件延迟和 file concurrency gate。

## 关键设计决策与权衡

- 权限、物理链路父子和资源归属继续使用同一棵 authoritative Node tree，以保留父控子和可强制执行的信任边界；逻辑资源位于 Node 下，不另建会与 authority 漂移的资源树。
- Transport 接口保持可插拔，但 link 的建立方式不改变 Node authority；多路径/新链路实现不能绕过 parent trust、policy generation 或 session cleanup。
- Topic 是有明确 owner 的资源，不恢复全局匿名总线；默认无 replay，持久历史需要以后作为显式能力设计。
- Desktop 首版使用 local-first 单活动 Profile 与 12 列响应式布局；暂不引入云同步、无限画布或多 Profile 同时在线，避免扩大一致性与凭据边界。
- 使用项目持有的 shadcn-style primitives 和 design tokens，不保留未使用的 Tailwind 构建层；renderer registry 允许后续资源类型在不重写 shell 的情况下加入预览与控件。

## 测试与验证方式 / 结果

- `go test ./... -count=1`、`go vet ./...`：通过。
- `go test -race ./runtime/... ./feature/... ./sdk/go/... ./sdk/bindings/... ./transport/... ./tests/integration/... ./apps/desktop -count=1`：通过。
- `go test ./runtime/link ./runtime/subscription ./tests/integration -count=10`：通过。
- Desktop Vitest 10/10、TypeScript/Vite production build、Windows Wails production build：通过。
- `./scripts/mfh.ps1 -Action check -Target generated`：通过；此前完整 `check all -AllowUnavailable` 通过，只有本机未安装 Flutter 被明确标记为 unavailable。
- 2,000 Nodes + 10,000 Resources 的 Explorer build+filter 样本约 37-39ms，低于 750ms 门禁。
- 50 次跨子树 Variable 事件三轮复验最慢：memory p95 1.1056ms、total 11.0432ms；TCP loopback p95 4.5976ms、total 111.9591ms，均低于计划预算。
- production Wails 实际验收通过：首次登录/准入、自动连接、2 Nodes/25 Resources、列表到底部、搜索/预览、按钮与拖放添加、View 保存/重启恢复、空 View、未保存切换取消、删除取消和暗色主题。
- `git diff --check` 与生成一致性检查通过；临时测试进程已停止，隔离证据没有进入产品状态。

## 潜在影响

- 这是 repository-wide clean break；旧 fixed-kind wire、SubProto、TopicBus、旧 Desktop UI 与旧配置不再兼容。
- Profile/View store 有明确版本；遇到旧或损坏数据必须显式 reset，不能静默降级或覆盖。
- 真实蓝牙/QUIC/硬件、production Media、View sync、多活动 Profile、移动工作区和 Flutter 工具链不在本轮完成范围。

## 回滚方案

- 本地合并前可删除功能分支恢复到 `master@6a07c08`；合并后可按提交范围反向回退 `c400648..9998c77` 与归档提交。
- 协议、运行时、SDK 和产品是不可分割的 clean-break slice，不支持只回退其中一层后与新层混跑。
- Desktop View/Profile 数据不由代码回滚自动删除；需要回退 schema 时先导出或显式 reset，避免破坏用户身份数据。
- 本轮没有 remote publication；不存在远端回滚动作。

## 子Agent执行轨迹

- 未使用子Agent。用户未显式要求委派，宿主策略不允许主动派发；设计、实现、测试、真实 UI 走查与归档均由主 agent 串行完成。
