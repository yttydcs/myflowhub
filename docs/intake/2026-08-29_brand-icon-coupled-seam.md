# 2026-08-29 MyFlowHub 品牌图标定稿

## Source

- 来源：当前 Codex 任务中的用户直接请求与逐轮视觉反馈。
- 日期：2026-08-29。
- Project root：`D:/project/MyFlowHub3`。
- Canonical repository：`D:/project/MyFlowHub3/repo/MyFlowHub`。
- 相关独立边界：Desktop 生产界面 workflow 将 `ICON01` 明确排除并交给本任务。

## Request Text / Source-preserving Summary

用户先要求为整个 MyFlowHub 项目设计图标，并明确希望图标体现项目自身的节点、资源、可信连接与可扩展特征，而不是看起来像普通图标库组件。设计过程经历多轮方向探索：用户否决了过于简单的树/连线、曲线电路、外圆、四向圆点、六边形组合等方案，要求重新设计并参考公开品牌设计规范。

研究阶段吸收了 Apple、Microsoft、Google、Figma、Atlassian、IBM Carbon、Mozilla 与 Pentagram/Slack 的公开规范，最后将复杂架构压缩为单一隐喻：两个同源 `Keyed Cell` 在可信边界内形成 `Coupled Seam`。用户在 V4 明确认可该方向，并在 V5 光学校正后明确回复“好就这个，固定下来吧”，随后调用 `$m-archive` 要求定稿和归档。

## Confirmed Requirements

- 采用 V5 `Calibrated Seam` 作为 MyFlowHub canonical 品牌图标。
- 主标只表达“独立节点建立可信定向连接”，不把树、Resource、Subscription、Command 与 Transport 全部画入图标。
- 保留矿物蓝、浅矿物蓝、石墨灰与暖白色体系；绿色继续只表达连接状态。
- 品牌主标、compact、16px tray、单色、反白与 app icon source 分别生产。
- 平台遮罩、圆角和阴影由目标平台生成，不写入 app icon master。
- 保留 V1–V4 研究资产与淘汰记录，不能删除后只留下最终 SVG。
- 固定 canonical SVG 和 512px PNG；本 workflow 不直接修改 Desktop/Android/Web 平台派生资产，也不推送发布。

## Open Questions

- 正式字标是否继续使用 `Segoe UI Semibold`，或对 `M / F / H` 做轻量定制，留待独立品牌字标任务决定。
- 商标近似检索与法律清查尚未执行；它是正式公开发布前的外部流程，不阻塞本地设计定稿。
- Desktop、安装包、Web、Android 等平台派生和接入需单独执行与验证。

## Routed Docs

- [品牌身份长期需求](../requirements/brand-identity.md)
- [品牌资产技术规格](../specs/brand-identity-assets.md)
- [Coupled Seam 品牌决策](../decisions/2026-08-29_coupled-seam-brand-identity.md)
- [本 workflow 计划归档](../plan/plan_archive_2026-08-29_brand-icon-coupled-seam.md)

## Related Changes

- [品牌图标 Coupled Seam 定稿归档](../change/2026-08-29_brand-icon-coupled-seam.md)
