# Todo - 可扩展资源平台与 Desktop 工作区重构

## Approved Execution

- [x] DOC01 — target specs、supersession chain 与 feature transition skeleton
- [ ] RES01 — protocol Resource descriptor v2 与 generic operations/wire major
- [ ] RES02 — extensible registry 与 Variable/Stream/Command migration
- [ ] RES03 — Node-owned Topic runtime 与 SDK helpers
- [ ] RES04 — descriptor-driven auth/routing/subscription/catalog
- [ ] SES01 — resource sessions、control/data lanes 与 File migration
- [ ] SDK01 — SDK、bindings、generated 与 Embedded migration
- [ ] APP01 — first-party products migration
- [ ] UI01 — React/shadcn frontend foundation
- [ ] UI02 — Profile/CredentialStore/login/auto-connect/switch
- [ ] UI03 — Node/Resource Explorer 与 Renderer Registry
- [ ] UI04 — Preview/Workspace/View persistence
- [ ] UI05 — built-in renderers、minimal visual system 与 accessibility
- [ ] VAL01 — clean-break cleanup、full validation 与 current-truth docs切换

## Will Not Execute Now

- [ ] MED01 — production Media/WebRTC/codec/QoS/platform capture；需要单独计划
- [ ] SYNC01 — View同步与多Profile同时在线；首版local-first/单活动Profile
- [ ] MOB01 — Android/iOS通用资源工作区；Desktop范围外
- [ ] LEG01 — legacy wire/SubProto/TopicBus compatibility；明确拒绝
- [ ] PUB01 — push/release/sign/store/hardware certification；需要单独授权

## Gate

- Plan drafted: yes
- Stable requirements/decision added: yes
- Business logic changed: pending
- Blocked: no
- Implementation approved: yes (`$m-execute`, 2026-08-28)
- Current task: RES01
