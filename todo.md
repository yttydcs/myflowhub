# Todo - 可扩展资源平台与 Desktop 工作区重构

## Approved Execution

- [x] DOC01 — target specs、supersession chain 与 feature transition skeleton
- [x] RES01 — protocol Resource descriptor v2 与 generic operations/wire major
- [x] RES02 — extensible registry 与 Variable/Stream/Command migration
- [x] RES03 — Node-owned Topic runtime 与 SDK helpers
- [x] RES04 — descriptor-driven auth/routing/subscription/catalog
- [x] SES01 — resource sessions、control/data lanes 与 File migration
- [x] SDK01 — SDK、bindings、generated 与 Embedded migration
- [x] APP01 — first-party products migration
- [x] UI01 — React/shadcn frontend foundation
- [x] UI02 — Profile/CredentialStore/login/auto-connect/switch
- [x] UI03 — Node/Resource Explorer 与 Renderer Registry
- [x] UI04 — Preview/Workspace/View persistence
- [x] UI05 — built-in renderers、minimal visual system 与 accessibility
- [x] VAL01 — clean-break cleanup、full validation 与 current-truth docs切换

## Test Remediation - Return to `$m-execute`

- [ ] TST01 — 将 admission/permit 拒绝和超时映射为明确、可操作的登录错误
- [ ] TST02 — 让空 Workspace 状态跨越完整工作区宽度并增加视觉回归覆盖
- [ ] TST03 — 隔离 Explorer 滚动区域，避免资源列表绘制到底部状态栏下方
- [ ] TST04 — 定义并执行大资源树、订阅延迟/吞吐量的代表性性能阈值，或显式批准延期

## Will Not Execute Now

- [ ] MED01 — production Media/WebRTC/codec/QoS/platform capture；需要单独计划
- [ ] SYNC01 — View同步与多Profile同时在线；首版local-first/单活动Profile
- [ ] MOB01 — Android/iOS通用资源工作区；Desktop范围外
- [ ] LEG01 — legacy wire/SubProto/TopicBus compatibility；明确拒绝
- [ ] PUB01 — push/release/sign/store/hardware certification；需要单独授权

## Gate

- Plan drafted: yes
- Stable requirements/decision added: yes
- Business logic changed: completed
- Blocked: no
- Implementation approved: yes (`$m-execute`, 2026-08-28)
- Heavy test passed: no (`$m-test`, 2026-08-29)
- Archive ready: no
- Current task: return to `$m-execute` for TST01-TST04

## Validation Summary

- [x] Full Go test and vet
- [x] Generated contract drift gate
- [x] Desktop Go, Vitest, TypeScript/Vite production build, Wails production build and startup smoke
- [x] Android gomobile, unit tests and lint
- [x] Metrics and Clipboard Go/Windows product gates
- [x] Embedded C/CTest and MicroPython protocol gates
- [x] Go race gate across runtime/features/SDK/transports/integration/Desktop
- [x] Link/subscription/integration repeated stability gate (`-count=10`)
- [x] Actual Wails login/profile/auto-connect/preview/drag/View persistence path
- [ ] Actual Wails visual and admission usability acceptance — TST01-TST03 failed
- [ ] Quantitative performance threshold acceptance — TST04 unresolved
- [ ] Clipboard Flutter tests — tool unavailable on this host; explicitly skipped by `-AllowUnavailable`
