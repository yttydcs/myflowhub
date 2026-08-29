# Plan Archive - MyFlowHub Coupled Seam 品牌图标

## Workflow Information

- Repo：`D:/project/MyFlowHub3/repo/MyFlowHub`
- Branch / checkout：canonical `master` 主检出；品牌文件在独立任务中以 path-scoped write set 处理。
- Base：`master@f79f165ccd90c328f5f46eee63af330a05ee019f`
- Project Root：`D:/project/MyFlowHub3`
- Docs Root：`D:/project/MyFlowHub3/repo/MyFlowHub/docs`
- Code Repos：canonical MyFlowHub monorepo only
- Worktree：未创建品牌专用 worktree；已有 `desktop-explorer-split-pane` worktree 属于另一任务，不在本 workflow 范围。
- Current Stage：Archive / closeout

## Goal

从多轮品牌图标探索中选择一个能够体现 MyFlowHub 节点与可信连接关系、同时适用于品牌展示、应用图标和 16px 托盘的稳定标志，并固定 canonical 资产、生产规范与决策记录。

## Scope

- 设计研究与公开规范分析；
- 多轮 HTML/SVG 方向探索；
- V4 Coupled Seam 方向选择；
- V5 键口、接缝与多尺寸光学校正；
- canonical SVG / 512px PNG 固定；
- 品牌 requirements/spec/decision/intake/change/lesson 归档与索引。

## Non-goals

- Desktop、Web、Android、安装包、PWA 或系统托盘的生产代码接入；
- 字标定制与商标法律清查；
- remote、push、release、publish 或 deployment；
- 修改另一个 Desktop implementation worktree。

## Docs Governance Routing Decision

- Docs root：`D:/project/MyFlowHub3/repo/MyFlowHub/docs`
- Intake impact：add
- Feature impact：none；本轮只固定品牌资产，尚未改变产品代码中的可见行为。
- Requirements impact：add `requirements/brand-identity.md`
- Specs impact：add `specs/brand-identity-assets.md`
- Decision impact：add `decisions/2026-08-29_coupled-seam-brand-identity.md`
- Lessons impact：add `lessons/brand-symbol-optical-size-variants.md`

## Related Docs

- Intake：[品牌图标定稿](../intake/2026-08-29_brand-icon-coupled-seam.md)
- Requirements：[Brand Identity](../requirements/brand-identity.md)
- Specs：[Brand Identity Assets](../specs/brand-identity-assets.md)
- Decision：[Coupled Seam Brand Identity](../decisions/2026-08-29_coupled-seam-brand-identity.md)
- Lesson：[Brand Symbol Optical Size Variants](../lessons/brand-symbol-optical-size-variants.md)

## Executable Task List

| Task ID | Title | Result | Acceptance |
| --- | --- | --- | --- |
| BRAND01 | 研究与方向探索 | Completed | V1–V4 研究板保留；单一母题形成 |
| BRAND02 | Coupled Seam V5 精修 | Completed | A 微调采用；B/C 有明确淘汰依据 |
| BRAND03 | 多尺寸生产资产 | Completed | full/mono/inverse/compact/tray/app source 齐备 |
| BRAND04 | Canonical 入口 | Completed | `myflowhub-icon.svg` 与 512px PNG 更新 |
| BRAND05 | 验证与归档 | Completed | 多视口、交互、资源、XML/PNG 与 docs 路由通过 |

## Key Decisions And Trade-offs

- 只表达“两个独立节点建立可信连接”，拒绝微型架构图。
- 采用 A Calibrated Seam；B 产生锯齿/拼图误读，C 退化为普通圆孔。
- 16px、20–31px 和 32px+ 使用独立接缝宽度，接受多资产维护成本换取真实可辨识度。
- 暂不把临时 `Segoe UI` 字标固化为矢量正式字标。

## Validation

- V5 HTML：Playwright 1440、768、390px，无横向溢出。
- 预览交互：color、mono、inverse 切换均指向正确资产和背景。
- 图片加载：broken images `0`；console/page/request errors `0`。
- 原尺寸：16、20、24、32、48、88px 视觉检查通过。
- canonical 512px PNG：从最终 SVG 内联透明渲染并肉眼检查。
- 文档：索引、相对链接、`git diff --check` 和 path-scoped status 检查。

## Rollback

- 将 `myflowhub-icon.svg` 和 `myflowhub-icon-512.png` 恢复到本次归档前版本，即可回滚 canonical 入口。
- V5 版本化资产与研究历史可保留，不影响旧产品代码。
- 如需整体回滚，revert 本 workflow 的单独 Git commit；不触碰另一个 Desktop worktree 或主检出无关 dirt。

## Sub-agent Trace

- None。用户要求图标工作留在当前任务；本 workflow 未使用子 Agent，也未向 Desktop 实现任务发送图标结果。
