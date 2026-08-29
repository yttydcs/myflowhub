# 2026-08-29 MyFlowHub 品牌图标 Coupled Seam 定稿

## 变更背景 / 目标

MyFlowHub 已经完成统一节点、资源、订阅、指令和可插拔链路的架构重构，但原有临时项目图标仍是多圆点树状图，既无法体现新的统一节点关系，也缺少 app、compact 和 tray 的生产边界。本 workflow 独立于 Desktop 生产界面任务，目标是选择并固定新的品牌 symbol，同时保留完整研究证据和可复用生产规范。

## 具体变更内容

- 建立 `Keyed Cell / 定向节点单元` 母形，并以两个同源单元形成 `Coupled Seam / 耦合缝` 主标。
- 保留 V1–V4 方向研究；V4 确立 Coupled Seam，V5 只进行键口和接缝光学校正。
- V5 采用 A `Calibrated Seam`：键口高度 12→13、入口 8→9、完整接缝 7→9；B/C 微调明确淘汰。
- 生成彩色、单色、反白、compact、16px tray 和未遮罩 app source。
- 将 `design-demos/brand/myflowhub-icon.svg` 与 512px PNG 更新为 V5，成为 canonical 稳定入口。
- 新增品牌资产 README、长期 requirement、技术 spec、accepted decision、intake、plan archive 和 reusable lesson。

## Docs root

- `D:/project/MyFlowHub3/repo/MyFlowHub/docs`
- 与 canonical monorepo 同仓；当前没有 Git remote，本 workflow 不 push、publish 或选择 backup。

## Intake impact

Updated：新增 [品牌图标定稿 intake](../intake/2026-08-29_brand-icon-coupled-seam.md)，保存用户逐轮反馈和最终确认。

## Feature impact

None：品牌资产已经固定，但尚未接入 Desktop、Web、Android 或安装包生产代码，因此不改写当前用户可见 feature truth。

## Requirements impact

Updated：新增 [Brand Identity](../requirements/brand-identity.md)，定义长期品牌、多尺寸、canonical 与研究保留要求。

## Specs impact

Updated：新增 [Brand Identity Assets](../specs/brand-identity-assets.md)，定义几何、色彩、资产路由与平台派生边界。

## Decision impact

Updated：新增 accepted ADR [Coupled Seam Brand Identity](../decisions/2026-08-29_coupled-seam-brand-identity.md)。

## Lessons impact

Updated：新增 [Brand Symbol Optical Size Variants](../lessons/brand-symbol-optical-size-variants.md)，记录为何 16px 不能机械缩放大尺寸主标。

## Related intake

- [2026-08-29 MyFlowHub 品牌图标定稿](../intake/2026-08-29_brand-icon-coupled-seam.md)

## Related features

- None。本次未修改产品 feature 文档；后续平台接入应更新对应 Desktop/Web/Android feature 与 change。

## Related requirements

- [Brand Identity](../requirements/brand-identity.md)

## Related specs

- [Brand Identity Assets](../specs/brand-identity-assets.md)

## Related decisions

- [Coupled Seam Brand Identity](../decisions/2026-08-29_coupled-seam-brand-identity.md)

## Related lessons

- [Brand Symbol Optical Size Variants](../lessons/brand-symbol-optical-size-variants.md)

## 对应 plan.md 任务映射

- 本品牌 workflow 是根级 Desktop plan 中明确排除的 `ICON01` 独立任务，不覆盖当前根 `plan.md`。
- 归档计划：[plan_archive_2026-08-29_brand-icon-coupled-seam.md](../plan/plan_archive_2026-08-29_brand-icon-coupled-seam.md)
- BRAND01：研究与多方向探索。
- BRAND02：V4 选择与 V5 精修。
- BRAND03：多尺寸生产资产。
- BRAND04：canonical SVG/PNG 固定。
- BRAND05：验证、stable docs、indexes 与 closeout。

## 经验 / 教训摘要

- 品牌 symbol 不能把完整架构压缩成一张拓扑图；一个不可替换的关系比五个准确但彼此竞争的元素更有识别度。
- 大尺寸正确不代表小尺寸可用。负空间必须在 20px 和 16px 主动放大，才能避免节点粘连。
- 第一轮 Keyed Cell 曾被读成 Pac-Man；增加说明文字不能修复错误轮廓，必须在无文字小样阶段直接淘汰。
- 精修阶段 B/C 虽强化语义，却分别引入锯齿/拼图和普通圆孔误读；停止添加比继续解释更重要。

## 可复用排查线索

- 症状：16px 看成一个黑点、两个节点粘连、中央孔消失。
- 触发条件：把 512px full symbol 直接缩成 tray/favicon。
- 关键词：`icon`, `tray`, `favicon`, `compact`, `negative space`, `optical correction`, `Coupled Seam`, `Pac-Man`, `chain link`。
- 快速检查：在真实 CSS 16/20/24px 下并排渲染，不放大截图；切换纯黑和纯白，确认仍能分辨两个主体。

## 关键设计决策与权衡

- 采用 V5 A Calibrated Seam，保留 V4 第一印象，仅增加键口和接缝稳定性。
- 接受 full/compact/tray 三套资产维护成本，拒绝“单 SVG 适配所有尺寸”的便利性。
- app icon source 保持未遮罩方形，平台负责圆角和阴影。
- 暂不接入任何应用模块，避免与仍在进行的 Desktop 工作任务形成写集冲突。

## 测试与验证方式 / 结果

- Playwright：V5 页面 1440px、768px、390px 均无横向溢出。
- 交互：color/mono/inverse 按钮的 `src`、背景模式和 `aria-pressed` 全部正确。
- 资源：所有本地 SVG natural size 有效；console、pageerror、requestfailed 均为 0。
- 视觉：hero、生产适配、16/20/24/32/48/88px、三种 app mask 和移动长页完成肉眼检查。
- canonical PNG：从最终 SVG 内联生成 512×512 透明 PNG，并完成预览检查。
- 文档：分类索引更新，`git diff --check` 通过。

## 潜在影响

- 后续平台接入时会改变应用窗口、安装包、任务栏、托盘、favicon 或移动 launcher 的用户可见品牌；必须另开 workflow 验证各平台派生。
- Coupled Seam 仍可能产生轻微链环联想；公开发布前建议完成商标近似检索。
- 临时字标尚未定稿，不能把研究板中的系统字体轮廓当作正式品牌资产。

## 回滚方案

- 回滚本次 commit 可恢复旧树状 canonical SVG/PNG 和移除新增品牌 stable docs。
- 若只回滚视觉入口，恢复 `myflowhub-icon.svg` / `myflowhub-icon-512.png` 即可；V5 研究与规范可继续保留。
- 本次没有修改 Desktop implementation worktree、平台打包资产或运行时数据，无需数据迁移。

## 子Agent执行轨迹

- None。所有设计、验证与归档均在当前任务完成；没有向独立 Desktop 实现任务发送图标结果。
