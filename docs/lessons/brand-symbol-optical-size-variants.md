# Brand Symbol Optical Size Variants

## Summary

复杂品牌标志不能用一个大尺寸 SVG 机械缩放到所有 UI 场景。MyFlowHub Coupled Seam 在 32px 以上使用完整接缝，但 20–31px 和 16px 必须分别扩大负空间，否则两个节点会在真实显示密度下粘连。

## Lookup Hints

- 症状：tray/favicon 只剩一个黑点、中央孔闭合、两个主体看不出来、单色版本失去结构。
- 关键词：`icon`, `logo`, `tray`, `favicon`, `compact`, `negative space`, `optical correction`, `SVG scaling`。
- 快速检查：把候选同时放到真实 16、20、24、32px CSS 尺寸；用黑色和反白各看一次，不放大截图判断。

## Symptoms

- 512px 预览清楚，但系统托盘只显示不规则小块。
- 彩色版本尚能靠颜色区分，单色版本两个节点完全合并。
- 增加 stroke 或阴影后大尺寸变脏，小尺寸仍不稳定。
- 平台圆角和缩放后图标视觉偏心或负空间被吃掉。

## Impact

- 品牌识别在最常见的任务栏、托盘、favicon 和紧凑导航中失效。
- 产品团队可能为每个平台临时修改 canonical SVG，最终产生不可追溯的派生漂移。

## Trigger Conditions

- 直接把 512/1024px brand symbol 缩放到 16–24px。
- 主要识别依赖小孔、细线、窄接缝或双色边界。
- 只在放大截图或 Retina 2× 图中评审，没有检查 CSS 原尺寸。
- 把平台遮罩、阴影和圆角提前烘焙进 master。

## Root Cause

数学缩放保持比例，却不保持人眼在不同像素密度下感知到的负空间和视觉重量。低尺寸的像素取样会关闭窄缝、合并相邻轮廓；用阴影补救只是在增加新的不稳定细节。

## Investigation Trail

1. V4 full symbol 在 32px 以上成立，但 16/20px 单色预览接缝过紧。
2. 并排渲染 V4 baseline、Calibrated Seam、Directional Key 和 Tight Coupling。
3. B 的 L 键口产生锯齿/拼图误读；C 的紧密重叠将接缝退化为普通圆孔。
4. A 保留 full silhouette，同时把 full、compact、tray 接缝分别设为 9、24、30。
5. 在浅色、单色、反白、三种 app mask 和真实 CSS 像素阶梯复验。

## Resolution

- 建立 `full / compact / tray` 三套同源几何。
- 32px+ 使用 V5 full；20–31px 使用 compact；16px 使用单色 tray。
- app icon 使用未遮罩方形源层，由平台处理 mask/shadow。
- 用稳定资产路由表约束产品接入，禁止直接引用研究方向文件。

## Prevention / Guardrails

- 每次修改 canonical geometry 都必须同步检查 16/20/24/32px。
- 小尺寸验收必须包含单色和反白，不能只看品牌双色。
- 平台派生只能从受治理 master 生成，不能反向修改 master。
- 保留失败变体和误读原因，避免后续重新走 Pac-Man、拼图、普通链环和微型拓扑老路。

## Related Docs

- [Brand Identity requirement](../requirements/brand-identity.md)
- [Brand Identity Assets spec](../specs/brand-identity-assets.md)
- [Coupled Seam decision](../decisions/2026-08-29_coupled-seam-brand-identity.md)
- [Brand icon change](../change/2026-08-29_brand-icon-coupled-seam.md)
