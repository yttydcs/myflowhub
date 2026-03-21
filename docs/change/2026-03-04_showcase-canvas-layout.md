# 2026-03-04 · Showcase：`canvas_percent` 自由布局 + Slider 值右置

## 变更背景 / 目标
- 为 Showcase Screen 增加一种可自由放置与缩放 widget 的布局模式：`canvas_percent`（位置/大小按百分比保存）。
- Slider 控件的数值展示（Badge）移到滑块右侧；极窄宽度下允许退化为两行以保证可用性。
- 通过右键菜单支持重叠场景下的 z-order 调整（置顶 / 置底）。

## 具体变更内容
### Go（配置结构 + normalize/clamp）
- 扩展 `ShowcaseScreenLayout`：新增 `canvas { baseWidth, baseHeight }`，并引入新布局模式 `canvas_percent`。
- 扩展 `ShowcaseWidgetLayout`：新增 `canvasPercent { xPct, yPct, wPct, hPct }`（0–100，保留 0.1% 精度）。
- 增加 normalize/clamp：
  - 未识别 `layout.mode` 自动回退到 `columns`；
  - `canvas.baseWidth/baseHeight` 默认值与边界 clamp；
  - `canvasPercent` 的边界 clamp（含最小尺寸：80×48 px 依据 baseW/baseH 换算为最小百分比）。
- 补充单测覆盖 canvas 默认与 clamp 路径。

### 前端（Designer + Viewer + Store）
- `Showcase.vue`（Designer）
  - Layout 增加 `columns/canvas_percent` 选择并保存；
  - 首次切换到 `canvas_percent` 且无历史 `canvasPercent` 时：按 2 列自动平铺初始化位置/大小；
  - 支持拖拽移动（Grip 手柄）与右下角 resize（pointerup 才保存，避免高频落盘）；
  - 右键菜单新增：Bring to Front / Send to Back（通过重排 widgets 数组实现 z-order）。
  - Slider：Badge 位于滑块右侧，极窄宽度自动换行保持可用。
- `ShowcaseWindow.vue`（Viewer Window）
  - 按 `screen.layout.mode` 渲染：`columns` 保持原逻辑；`canvas_percent` 使用居中画布 + absolute widgets。
  - 缩放算法：非等比缩小、宽高都适配、scale 上限为 1（不放大），不使用 `transform`。
  - Slider：Badge 位于滑块右侧。
- `stores/showcase.ts`
  - 扩展 layout 类型与 normalize：支持 `canvas_percent` 与 `canvas/canvasPercent` 字段，旧配置兼容。

## 对应 plan.md 任务映射
- CAN1：Slider 值右置（Designer + Viewer）
- CAN2：Go + TS schema/normalize 扩展
- CAN3：Viewer 渲染 `canvas_percent`
- CAN4：Designer 编辑 `canvas_percent`（拖拽/缩放 + 置顶/置底）
- CAN5：构建与测试（`go test`、`wails build`）

## 关键设计决策与权衡
- `canvas_percent` 使用百分比 rect + `baseWidth/baseHeight`：
  - 优点：布局与窗口尺寸解耦，缩放时可稳定换算；配置可读且便于扩展。
  - 代价：需要额外的 normalize/clamp 与交互坐标换算。
- 缩放采用“非等比缩小 + scale<=1（不放大）”，并通过计算后的画布尺寸直接渲染（不使用 `transform`）：
  - 优点：控件/字体不会被 transform 拉伸导致变形；交互坐标换算更直观。
- Designer 交互：移动/缩放只更新内存，pointerup 才 `save()`：
  - 优点：避免高频落盘与跨窗口频繁 reload。
  - 代价：拖动过程中 Viewer 不实时变化，但保存后可立即同步。
- z-order：复用 widgets 数组顺序（末尾覆盖在上层），通过“置顶/置底”重排实现：
  - 简单可靠、无需额外 zIndex 字段。

## 测试与验证方式 / 结果
- `go test ./... -count=1`：通过。
- `wails build -debug -skipembedcreate -nopackage`：通过。
- 建议手工冒烟：
  1) `wails dev` → `#/showcase`：切换布局、拖拽移动/缩放、右键置顶/置底；
  2) `Open Window`：观察保存后同步，验证缩放与 slider 右置效果。

## 潜在影响与回滚方案
- 潜在影响：
  - Showcase config 将出现新字段（`layout.canvas` / `layout.canvasPercent`）；旧数据仍可加载。
  - `columns` 布局逻辑保持不变；未识别 mode 会回退到 `columns`。
- 回滚方案：
  - 回滚本次变更文件；
  - 既有配置中的新字段在旧版本中会被忽略（mode 不支持时回退 `columns`）。

