# 2026-08-31 NodeHost runtime 与产品边界收敛

## 变更背景 / 目标

Desktop、MetricsNode 与 Android 都需要创建持久 identity、Node、Parent supervisor、资源目录与 SDK Client，但它们仍是互相独立的普通节点产品。目标是把共同运行时装配收敛为平台中立的 NodeHost，同时保持产品、进程、安装、state directory、Resource owner 与权限边界独立；SDK Client 只提供同一 Node 上的操作能力，不成为第二套运行时或发送通道。

## 具体变更内容

- 新增 `host/nodehost`，统一持有 auth state、唯一 Node、Registry、attached SDK Client、可选 Parent 与 Listeners，并提供 `New`、单次 `Start`、幂等 `Close` 和 root/relay/leaf/offline 角色。
- 新增 `Registry.Variable(VariableSpec)`，自动绑定本地 owner，同时保留 schema、content type、read/write permission 与 payload limit 校验。
- 新增 non-owning SDK/binding attached Client 与只读 `ConnectionStatus`；旧 runtime-owning API 保留为兼容入口并标记弃用。所有 operation 继续经过原 Node route、Session queue 与 durable subscription。
- Desktop 的静态/既有 Profile 改为 Parent-only leaf NodeHost；Desktop binding 通常只管理 facade/subscription，不创建 Listener 或 MCP。注册前没有 Node ID 的 authority Enrollment Profile 暂时保留 owning binding 作为 bootstrap/reconnect 兼容路径。
- Metrics Windows/CLI/Android mobile 共用 NodeHost、Registry 与 attached SDK Client，并在 Host 启动前完成资源注册。
- Android 保留既有 gomobile ABI：导出的 `android.Client` 是平台 lifecycle wrapper，Host 构造集中在 `sdk/bindings/android/host.go`；Kotlin Service 继续拥有 OS lifecycle，并用 generation guard 防止停止后的陈旧异步启动。
- architecture test 把 Android Host 例外精确限制为 `sdk/bindings/android/host.go -> host/nodehost|host/hub`，其余 SDK 生产文件仍禁止依赖 `host/`。

## Docs root

- `D:\project\MyFlowHub3\worktrees\nodehost-runtime\docs`，属于 canonical MyFlowHub monorepo；本轮只做本地归档、提交与集成，不新增 remote，不 push/release/publish。

## Intake impact

- updated — 新增并索引 NodeHost、Desktop/Metrics 独立产品、Agent Gateway 普通节点定位与多平台边界的讨论记录。

## Feature impact

- updated — Desktop、MetricsNode 与 Android feature 已明确共用 Host contract，但产品身份、进程、安装和平台 lifecycle 独立。

## Requirements impact

- updated — 统一节点运行时 requirement 已加入通用 Host、产品无特权、attached Client 和异构平台边界。

## Specs impact

- updated — 新增 NodeHost runtime spec，并同步 lifecycle、resource helper 与 repository/module boundary。

## Decision impact

- updated — 新增通用 NodeHost、non-owning SDK Client 与薄平台适配 ADR；不改变既有 authority tree、Resource ownership 或 wire protocol。

## Lessons impact

- updated — 新增 Android Gradle single-use daemon loopback 环境故障 lesson，避免把环境不可用误报为产品测试通过或产品回归。

## Related intake

- [NodeHost、运行时能力共享与产品边界](../intake/2026-08-30_node-host-runtime-and-product-boundaries.md)

## Related features

- [Desktop](../features/desktop.md)
- [MetricsNode](../features/metrics-node.md)
- [Android](../features/android.md)

## Related requirements

- [统一节点运行时](../requirements/unified-node-runtime.md)

## Related specs

- [NodeHost Runtime](../specs/node-host-runtime.md)
- [Operational lifecycle](../specs/operational-lifecycle.md)
- [Resource Platform v2](../specs/resource-platform-v2.md)
- [仓库与模块边界](../specs/repository-and-module-boundaries.md)

## Related decisions

- [通用 NodeHost、non-owning SDK Client 与薄平台适配](../decisions/2026-08-30_generic-node-host-and-non-owning-sdk-client.md)
- [权威 Node tree 与可插拔 Link](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)

## Related lessons

- [Android Gradle daemon loopback 不可用](../lessons/android-gradle-loopback-daemon-unavailable.md)
- [Android runtime 与 mobile bindings](../lessons/android-runtime-and-mobile-bindings.md)
- [Windows clean checkout、EOL 与 generated drift](../lessons/windows-clean-checkout-eol-and-generated-drift.md)

## 对应 plan.md 任务映射

| Task ID | 结果 |
| --- | --- |
| DOC01 | intake、features、requirement、spec、ADR 与索引完成。 |
| HOST01 | 通用 NodeHost lifecycle、roles、Parent/Listens、状态与回滚完成。 |
| RES01 | owner-bound Variable helper 与边界测试完成。 |
| SDK01 | attached non-owning SDK/bindings Client、ConnectionStatus 与兼容路径完成。 |
| DESK01 | Desktop Parent-only leaf、Profile retry/rollback 与无 MCP/Listener 边界完成。 |
| METR01 | Metrics Windows/CLI runtime、资源和通知迁移完成。 |
| ANDR01 | Android generic/Metrics platform shell、typed bridge、generation guard 与双 AAR ABI 完成。 |
| QA01 | full test/vet/race、生成物、Wails、AAR 与文档门禁完成。 |
| ARC01 | change/plan/lesson 归档、索引、local integration 与 cleanup 由本阶段处理。 |

## 经验 / 教训摘要

- 产品名只表示默认能力组合，不代表网络角色或特权；Node 角色由 Parent、Listeners 和权限配置决定。
- Host 拥有生命周期，SDK Client 只拥有操作与订阅 facade；两者必须共享同一个 Node 和发送机制。
- 平台 composition root 可以持有 Host，但例外必须精确到文件/import pair，不能让普通 SDK 反向依赖产品 Host。
- generated AAR 成功与 Gradle app 构建是不同证据；环境故障必须记录为 Unavailable，而不是 Pass 或产品 Fail。

## 可复用排查线索

- 症状：SDK import architecture test 报 `SDK cannot import ".../host/nodehost"`。触发：把平台 Host composition 写进普通 binding 文件。关键词：`internal/archtest`、`sdk/bindings/android/host.go`。快速检查：生产 SDK 中 `host/` import 只能命中精确 Android `host.go` 例外。
- 症状：attached Client `Close` 意外终止整个 Node。关键词：`ErrAttachedClientClose`、`ownsRuntime`、`NewAttachedClient`。快速检查：Host pointer identity 与 Close 后本地 Snapshot 测试。
- 症状：Gradle 在任何 task 前失败。错误：`java.io.IOException: Unable to establish loopback connection`。快速检查：generic 与 Metrics 是否在 daemon 启动阶段同签名失败，同时 gomobile AAR 是否仍可生成。
- 症状：Android Stop 后旧协程重新启动 runtime。关键词：`RuntimeGeneration`、`NodeService`。快速检查：replacement/stop 后旧 generation 的 launch 必须失效。

## 关键设计决策与权衡

- 采用单一通用 NodeHost，而不是把 Metrics 合并进 Desktop或为 leaf/root/relay 建立不同 Host 类型；角色由组合配置表达。
- attached Client 的 `Connect`/`Close` 明确拒绝 lifecycle 操作，换取所有权可验证；旧 owning API 暂留以控制兼容风险。
- Android 为保 ABI 保留 lifecycle wrapper，但 Host 构造/import 集中在单文件 composition root；Android in-process Hub 迁移延期。
- Desktop authority Enrollment 暂留 owning runtime，避免在本轮同时改变 Enrollment identity/Grant persistence；该例外不改变产品权限，并在 NodeHost 可直接消费持久 Grant 后删除。
- 不迁移 Hub、ClipboardNode、Agent Gateway、Web/Embedded 或 installer，避免在首批 API 未稳定前扩大写集。

## 测试与验证方式 / 结果

- focused Go、focused `-race`、`go test ./... -count=1`、`go vet ./...`：通过。
- architecture test、docs relative links、bindings generation freshness、`git diff --check`：通过。
- Desktop：Go tests、54 个 frontend tests、TypeScript/Vite 与 Windows/amd64 Wails production build 通过。
- Metrics：Go tests、2 个 frontend tests、TypeScript/Vite 与 Windows/amd64 Wails production build 通过。
- generic Android 与 Metrics gomobile AAR：arm64-v8a、x86_64 和 `javap` Java ABI 通过。
- Gradle `testDebugUnitTest`、`lintDebug`、`assembleDebug`：Unavailable；generic 与 Metrics 均在 daemon 启动时失败于 `java.io.IOException: Unable to establish loopback connection`，不记为通过。
- Android device smoke：Unavailable；本机没有 `adb`/设备证据。

## 潜在影响

- generic Android Client 与 Metrics RuntimeConfig 仍使用默认 identity persistence；Keystore-backed `IdentityStore` 注入是后续 seam。
- Android Gradle/JVM 与真实设备 lifecycle 需要在可用环境补验。
- 旧 owning SDK/binding 与新 attached API 在迁移期并存；除 Desktop authority Enrollment 等已记录兼容入口外，新增产品应只走 NodeHost attached path。
- 没有 wire、Resource schema、Profile/config 或持久数据格式迁移。

## 回滚方案

- 建议按 Android adapter/generated artifact → Desktop/Metrics 迁移 → SDK attached boundary → Variable helper → NodeHost 的逆依赖顺序回退。
- 回退不会要求删除用户 identity、Profile、Metrics config 或 Resource 数据；本轮没有数据格式迁移。
- 若集成与主线冲突，保留 feature commit/worktree，不覆盖主检出未提交内容。

## 子Agent执行轨迹

- `doc01_contract`（Locke）：DOC01 stable docs、ADR、索引与 Android ownership/Keystore 事实校正。
- `host01_nodehost`：HOST01 NodeHost lifecycle、roles、Parent/Listeners 与测试。
- `res01_variable`：RES01 Registry Variable helper 与测试。
- `sdk01_attached_client`：SDK01 attached Client、ConnectionStatus、bindings 与测试。
- `desk01_nodehost`：DESK01 Desktop migration、Profile retry/rollback 与产品构建。
- `metr01_nodehost`：METR01 Metrics runtime/controller/notifications/CLI migration。
- `andr01_platform_shell`（Mencius）：ANDR01 Android composition root、Kotlin lifecycle、gomobile ABI；修复 QA 发现的 SDK import boundary。
- `qa01_full_gate`（Epicurus）：QA01 heavy gate、architecture regression detection、最终 plan/todo 状态与审查。
- 主 Agent：需求/权限边界协调、独立复核、docs governance、archive 与 control-plane integration。

## Closeout status

- Archive docs：完成。
- Feature commits：`19b4cfe feat: 收敛通用 NodeHost 与产品运行时`、`350e733 docs: 记录 Enrollment 兼容边界`。
- Master integration：在临时 detached worktree 中把主检出 tracked 修改快照重放到 `350e733`，四个重叠 docs 路径均自动合并；随后 `master` 从 `5c17648` fast-forward 到 `350e733`。
- Preservation：主检出 autostash 恢复后与预演结果逐文件比对，tracked/untracked 用户改动保留差异为 0；这些未提交内容仍留在主检出，未被纳入本归档提交。
- Cleanup：临时集成 worktree、`worktrees/nodehost-runtime` 与已合并的 `refactor/nodehost-runtime` 分支均已删除；既有 `feat/desktop-profile-entry` worktree 未改动。
- Publication：local-only；未授权 push、release 或 publish。
