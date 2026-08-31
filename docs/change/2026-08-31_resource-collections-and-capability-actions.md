# 2026-08-31 Resource Collections、Capability Actions 与 Desktop 交互

## 变更背景 / 目标

既有 catalog 把领域操作分散成大量 endpoint-style Resources，无法自然表达“一个授权边界管理大量内部成员”，
Desktop 也缺少从真实 Resource 统一触发操作和展示内容的入口。本轮固定 Resource、Capability、Collection member、
独立 Command Resource 与 UI namespace 的边界，并以 Flow、只读 filesystem、SDK 与 Desktop 完成首个端到端实现。

## 具体变更内容

- 新增有界 `mfh.collection` list/member/page schema 与多 Capability runtime handler，继续复用既有 Registry、policy、
  route、schema validation、payload limit、deadline、错误和审计路径。
- Flow clean-break 为 `flow/definitions` 与 `flow/runs` 两个 Collections；旧 create/update/run/cancel/archive Commands
  与 `flow/events` 不再进入 catalog，旧 grant 不自动映射。
- 新增 opt-in、默认且实际只读的 filesystem provider；每个需要独立授权的 root 对应一个 Resource，支持安全
  `list/get/read`，并防御 traversal、symlink/junction、root 替换、oversize 与 snapshot/cursor 漂移。
- Go SDK 新增 generic Collection/Flow typed clients；bindings 保留 generic operation/subscription seam；Android sample
  使用两个 Flow Collections，并完成双 ABI AAR 与离线 Gradle unit/lint/assemble。
- Desktop 从 descriptor 推导共享 action model，在真实 Resource 上提供 pointer/Menu/Shift+F10 入口；Inspector 与
  View 共用显式 Execute、Collection browser 和 text/JSON/raster/non-executable fallback renderer。
- 合并最新 `master` 后，把 Desktop Profile entry/deactivation 与原子 Inspector state 合并，并从最终源码重建 tracked
  Vite assets；主线文档中的 Profile 约束和资源约束均保留。

## Docs root

- `D:\project\MyFlowHub3\worktrees\resource-collections-actions\docs`，属于 canonical MyFlowHub monorepo。
- 本轮只做本地归档、提交、主线集成与 cleanup；未授权 push、release、publication 或 deployment。

## Intake impact

- updated — 新增并索引资源目录、Collection、Capability、Desktop 操作入口与普通 Node 产品边界的原始讨论记录。

## Feature impact

- updated — Desktop、Flow 与 Android feature 已同步当前 Collection/action/renderer/调用方与验证边界。

## Requirements impact

- updated — 可扩展资源平台与 Desktop 资源工作区 requirement 已加入 Collection、双层授权、上下文操作和内容展示约束；
  同时保留主线 Profile entry/deactivation 要求。

## Specs impact

- updated — 新增 Resource Collections and Actions spec，并同步 protocol map、Flow vNext 与 Resource Platform v2。

## Decision impact

- updated — 新增 Collection Resource 与 Capability Action 决策；普通操作不再因为 UI 入口或 endpoint 形态自动成为 Resource。

## Lessons impact

- updated — 新增跨 Go/JSON/JavaScript/Wails 的原子状态、JSON 安全 revision、opaque cursor 与 envelope/content 分层经验。

## Related intake

- [Resource Collection、Capability 与 Desktop 操作入口](../intake/2026-08-31_resource-collections-and-actions.md)

## Related features

- [Desktop](../features/desktop.md)
- [Flow](../features/flow.md)
- [Android](../features/android.md)

## Related requirements

- [可扩展资源平台](../requirements/extensible-resource-platform.md)
- [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)

## Related specs

- [Resource Collections and Actions](../specs/resource-collections-and-actions.md)
- [Protocol map](../specs/protocol_map.md)
- [Flow vNext](../specs/flow-vnext.md)
- [Resource Platform v2](../specs/resource-platform-v2.md)

## Related decisions

- [Collection Resource 与 Capability Action](../decisions/2026-08-31_collection-resource-and-capability-actions.md)

## Related lessons

- [Collection browser 的跨运行时契约边界](../lessons/collection-browser-cross-runtime-contract-boundaries.md)
- [Observable side effects 与 generated contracts](../lessons/observable-side-effects-and-generated-contracts.md)
- [Windows clean checkout、EOL 与 generated drift](../lessons/windows-clean-checkout-eol-and-generated-drift.md)

## Related plan snapshots

- [实施与验证计划](../plan/plan_archive_2026-08-31_resource-collections-and-capability-actions.md)
- [MAIN02 实施前 accepted-design 草案](../plan/plan_archive_2026-08-31_resource-collections-and-actions-accepted-design.md)

## 对应 plan.md 任务映射

| Task ID | 结果 |
| --- | --- |
| DOC01 | intake、features、requirements、specs、ADR 与索引收敛完成。 |
| COLL01 | Collection schemas、hard limits 与多 Capability runtime handler 完成。 |
| FLOW01 | definitions/runs Collections、clean-break 与 revision 边界完成。 |
| FS01 | opt-in 只读 filesystem provider、安全路径与分页快照完成。 |
| SDK01 | typed Go clients、generic bindings、generated contracts 与 Android caller 迁移完成。 |
| DESK01 | descriptor-driven action model、右键与键盘入口、显式 Execute 完成。 |
| RENDER01 | Collection browser、View actions 与安全 content renderers 完成。 |
| QA01 | full/race/vet/generated/frontend/Wails/AAR/Gradle/真实 GUI 门禁完成。 |
| MAIN02 | 与最新 master 合并、Profile/Resource 冲突及主检出并行文档方案完成语义收敛。 |
| ARC01 | change/plan/lesson/验证证据、local integration 与 cleanup 由本阶段处理。 |

## 经验 / 教训摘要

- Resource 是独立寻址、授权和审计边界；Capability 是对它的操作；Collection member 默认只在 owner 内定位。
- UI namespace、目录层级、presentation hint 与本地路径均不能扩张权限或创造第二棵资源树。
- descriptor 只说明“支持什么”，effective authorization 仍由 authority 裁决；`AUTHZ02` 前保留明确 Forbidden 反馈。
- 真实 GUI 能发现 jsdom 看不到的 stale embedded asset、状态重现和跨 JSON 数字边界问题，不能由 component tests 替代。
- wire envelope 与 decoded content 必须分层验证；HTML/SVG 不能因文件类型被当作可执行 UI。

## 可复用排查线索

- 症状：Return/Close 后旧 action/draft 重现。触发：selection/action 分裂状态或旧 packaged bundle。关键词：
  `focusedAction`、`go:embed`、asset hash。快速检查：原子更新 Inspector state，重建并二次启动真实 executable。
- 症状：翻页报 revision 非安全整数或 cursor stale。触发：`uint64` 直接进入 JavaScript number。关键词：
  `2^53-1`、`Number.isSafeInteger`、full fingerprint。快速检查：比较 Go JSON、浏览器值与 cursor 绑定三元组。
- 症状：JSON preview 显示 `Version/Encoding/Data`。触发：把 read envelope schema 用于 decoded JSON。关键词：
  `content_type`、envelope、inner renderer。快速检查：先校验 envelope，再按 decoded content 选择 renderer。
- 症状：filesystem 可以越过 root 或旧 cursor 读取新快照。触发：只做字符串前缀或只绑定 numeric revision。
  快速检查：canonical containment、open-handle identity、post-read check、parent/revision/full fingerprint cursor。

## 关键设计决策与权衡

- 采用 Collection 作为 Resource contract，而不是让每个文件、Flow definition/run 或 API endpoint 进入全局 catalog。
- filesystem 第一阶段保持“一 root 一 Resource”与只读；共享权限/owner/lifecycle 的 named multi-root、write/delete、
  download/session 延期到 `FS02`。
- Flow 使用 clean-break 而不是兼容重定向，避免旧 Resource/grant 被静默解释为新 capability。
- Desktop 使用 descriptor-driven action seam，但不把菜单可见性当授权；authoritative effective-capability discovery 延期到 `AUTHZ02`。
- mainline Profile 工作与资源工作在 `App.tsx`/requirements/dist 重叠；选择源码语义合并并重建 generated assets，而不是选择任一侧旧 bundle。

## 测试与验证方式 / 结果

- 实现阶段：full Go、focused race、full vet、canonical generated freshness、frontend 16 files/128 tests/build 通过。
- 产品阶段：clean Windows/amd64 Wails package、真实 browser-mode 与两次 packaged GUI 启动通过；pointer/keyboard menu、
  pagination、Flow create/run、Forbidden draft retention、safe content、theme/View restore 均使用真实 Hub/policy/provider。
- Android：双 ABI AAR、11 个 generated Java binding classes、离线 JDK 21 Gradle `testDebugUnitTest`、`lintDebug`、
  `assembleDebug` 通过；没有连接设备，device smoke 记为 unavailable。
- 合并最新 master 后：`go test ./... -count=1`、`go vet ./...`、generated freshness、frontend 17 files/134 tests、
  TypeScript/Vite production build 通过。
- `git diff --check` 对源码与非 `dist` 文件通过；Vite/esbuild 生成的 minified bundle 含构建器产生的空白行，
  由确定性 rebuild/freshness 与真实 package 验证约束，不手改生成物。
- [完整 QA 证据与截图说明](verification/2026-08-31_resource-collections-actions/README.md)

## 潜在影响

- Flow 是 intentional clean-break；旧 Resource 名称与旧 grant 不再工作，需要调用方使用新的 Collection Resource/capability。
- filesystem provider 必须由产品显式注册，不会让 Desktop、Metrics 或 Core 默认暴露本机目录。
- `AUTHZ02`、`CMD02`、`FS02` 仍是独立后续范围；本归档不宣称它们已完成。
- Android 真实设备、write/delete 与 large-file data lane 仍没有当前证据。

## 回滚方案

- 建议按 Desktop renderer/action → Android/SDK/generated contracts → filesystem/Flow providers → runtime/protocol 的逆依赖顺序回退。
- Flow clean-break 回退涉及 catalog/grant 语义，必须整体回退协议、provider、generated contract 和第一方 caller，不能只恢复旧 UI 名称。
- filesystem 没有写入用户文件；关闭显式 provider 注册即可停止暴露，回滚不删除物理 root 内容。
- 若主线集成失败，保留 feature commits/worktree 和主检出 tracked/untracked 快照，不覆盖并行用户修改。

## 子Agent执行轨迹

- `andr01_platform_shell`（Mencius）：SDK/Android/Desktop platform shell、双 ABI/Gradle，以及真实 GUI 发现后的三轮实现修复。
- `doc01_contract`（Locke）：稳定文档、协议现状、deferred boundary 与最终 revision/content 事实校正。
- `qa01_full_gate`（Epicurus）：全量门禁、真实 browser/packaged GUI 证据、状态与 archive-ready 复核。
- 主 Agent：需求/权限边界协调、冲突语义合并、独立验证、docs governance、archive 与 control-plane integration。

## Closeout status

- Archive docs：完成；change/plan/lesson、21 个真实 UI 证据文件与实施前 accepted-design 草案均已进入受治理目录。
- Feature commits：`46da972 feat: 引入资源集合与能力操作`、`946fc40 Merge branch 'master' into feat/resource-collections-actions`。
- Master integration：`master` 从 `00bd333` fast-forward 到 `cd9fa7c`；主检出 tracked 快照在临时 detached worktree
  预演后重放，实际 tree `3f7f7d7` 与预演逐字节一致。
- Preservation：8 个实质 tracked 用户修改恢复为未暂存状态；其余原状态项只有 EOL/index normalization、没有内容差异。
  其他 untracked 文件保持原位；同名旧 spec 已归档为 accepted-design 快照，未用旧草案覆盖 current contract。
- Cleanup：tracked stash、临时预演 worktree、功能 worktree/branch 与可逆备份将在本归档提交后清理并复核。
- Publication：local-only；未授权 push、release、publish 或 deployment。
