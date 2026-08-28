# Plan - MyFlowHub canonical local status

## Workflow Information

- Repository: `D:/project/MyFlowHub3/repo/MyFlowHub`
- Branch: `master`
- Docs Root: `D:/project/MyFlowHub3/repo/MyFlowHub/docs`
- Current Stage: `4.0 archive and local closeout complete`
- Compatibility: clean break；不保留 fixed-kind wire、SubProto、TopicBus、旧 Desktop UI 或旧配置兼容层

## Current Status

- 一切设备都是 Node，Node 下的一切都是 Resource；资源通过可扩展 type/capability、统一 catalog、operation、subscription 和 session 表达。
- 唯一 authoritative Node tree 同时承载物理父子、父控子 authority 与资源归属；Transport 保持 Memory、TCP、QUIC、RFCOMM 等可插拔实现，但不能绕过树上的信任与权限边界。
- Variable、Stream、Topic、Command 与 File 已迁移到统一资源平台；Go SDK、bindings、Android、Embedded 和第一方产品使用同一协议与运行时。
- Desktop 已重建为 React/Vite 资源工作区，提供受保护身份、多 Profile、自动连接、Node/Resource Explorer、预览、拖放/键盘组合与本地持久 View。
- canonical checkout 位于 `repo/MyFlowHub`，临时工作树统一位于 sibling `worktrees/`；已退役旧仓的来源和恢复信息保留在 `migration/`。

## Latest Closeout

- Active plan archive: [可扩展资源平台与 Desktop 工作区重构](docs/plan/plan_archive_2026-08-29_extensible-resource-platform-desktop-workspace.md)
- Change archive: [可扩展资源平台与 Desktop 工作区重构](docs/change/2026-08-29_extensible-resource-platform-desktop-workspace.md)
- Reusable lesson: [Desktop binding 重连与准入诊断](docs/lessons/desktop-binding-reconnect-and-admission-diagnostics.md)
- Source intake: [可扩展资源与 Desktop 工作区重设计](docs/intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)

## Completed Tasks

- [x] DOC01 — target specs、supersession chain 与 feature transition skeleton。
- [x] RES01-RES04 — extensible descriptor、registry、Topic 与 descriptor-driven auth/routing/subscription/catalog。
- [x] SES01 — resource session、control/data lanes 与 File 迁移。
- [x] SDK01、APP01 — SDK/bindings/generated/Embedded 与第一方产品 clean-break 迁移。
- [x] UI01-UI05 — React/shadcn-style shell、Profile、Explorer、Renderer、Workspace/View 与 accessibility。
- [x] VAL01、TST01-TST04 — 全产品门禁、真实 Wails UI 走查、重连修复与定量性能验收。

## Validation Baseline

- `go test ./... -count=1`
- `go vet ./...`
- Go race gate across runtime/features/SDK/transports/integration/Desktop
- link/subscription/integration repeated stability gate (`-count=10`)
- `scripts/mfh.ps1 -Action check -Target generated`
- Desktop Vitest、TypeScript/Vite production build、Windows Wails production build
- production Wails 登录/准入/自动连接/Explorer/预览/拖放/View 持久化/暗色模式走查
- Explorer 2,000 Nodes + 10,000 Resources 与 memory/TCP 跨子树订阅性能门禁

## Stable Docs Impact

- Intake impact: updated
- Feature impact: updated
- Requirements impact: updated
- Specs impact: updated
- Decision impact: updated
- Lessons impact: updated

## Deferred Or Separately Authorized Work

- MED01 — production Media/WebRTC/codec/QoS/platform capture。
- SYNC01 — View sync 与多 Profile 同时在线。
- MOB01 — Android/iOS 通用资源工作区。
- LEG01 — legacy compatibility bridge；已明确拒绝。
- PUB01 — remote push、release、sign、publish；需要单独授权。
- 真实蓝牙/QUIC/硬件与本机未安装的 Flutter 工具链需要相应设备或环境后单独验证。

## Gate

- Blocked: no
- Active refactor workflow: none
- Local archive/merge/cleanup: complete
- Remote publication: not authorized and not performed
