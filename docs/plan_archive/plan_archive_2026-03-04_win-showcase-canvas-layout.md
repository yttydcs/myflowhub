# Plan Archive - 2026-03-04 - MyFlowHub-Win Showcase canvas_percent

归档时间：2026-03-04  
分支：`feat/showcase-canvas-layout`  
Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-slider-right\MyFlowHub-Win`  
来源：`plan.md`

---

# Plan - MyFlowHub-Win：Showcase Canvas Percent 布局（自由拖拽/缩放）+ Slider 值右置

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`feat/showcase-canvas-layout`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-slider-right\MyFlowHub-Win`
- Base：`main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
  - `d:\project\MyFlowHub3` 根目录 `AGENTS.md`（阶段纪律、worktree 禁令等）

---

## 0) 当前状态（项目现状）
- Showcase 已存在：
  - Designer：`frontend/src/pages/Showcase.vue`
  - Viewer Window：`frontend/src/windows/ShowcaseWindow.vue`
- 现有 Screen Layout 仅支持 `columns`（maxColumns/minColumnWidth/gap；widget 仅 colSpan）。
- 近期已完成：widget 卡片 Key-Value 单行布局（不在本 workflow 改动范围内）。

---

## 1) 需求分析（已确认）

### 目标
1) Slider 模式：数值（Badge）显示在滑块右侧；极窄时允许退化为两行（第一行 slider，第二行右对齐 value）。
2) 新增一种 Screen 布局模式（暂名：`canvas_percent`）：用户可自由拖拽放置 widget，并可调整大小；位置/大小用百分比保存。
3) 缩放算法：非等比（x/y 分别缩小）、宽高都适配、上限固定为 1（不放大）。
4) Designer 右键菜单新增：置顶 / 置底（用于重叠时 z-order）。

### 范围（必须 / 不做）
- 必须：
  - 同时覆盖 Designer + Viewer Window；
  - 仅扩展 Showcase config 与 UI 渲染/交互；TopicBus/VarPool 数据流不改。
- 不做：
  - 不做像素绝对布局（px）持久化；
  - 不引入第三方拖拽缩放库（V1 自研最小交互）。

### 验收标准
- Slider：Designer + Viewer 均满足 value 在右；极窄退化可用。
- `canvas_percent`：
  - Designer：可拖拽移动 + resize 缩放；保存后 Viewer 实时同步；
  - Viewer：按百分比渲染，随窗口缩小可同时适配宽高；不放大；
  - 右键菜单：置顶/置底生效（数组顺序即 z-order）。
- columns 屏幕不回归。

### 风险
- 拖拽/缩放与控件交互冲突（需用专用移动手柄与 resize 手柄隔离事件）。
- 非等比缩小后 widget 可能变得过小，需要最小尺寸 clamp。

---

## 2) 架构设计（分析结论）

### 总体方案（采用）
- Screen 新增 `layout.mode = "canvas_percent"`：
  - Screen 级 `layout.canvas.baseWidth/baseHeight`（px，用于 scale 上限=1）。
  - Widget 级新增 `layout.canvasPercent = { xPct, yPct, wPct, hPct }`（0–100，保存 0.1% 精度）。
- 渲染（Designer/Viewer 一致）：
  - 测量容器 `cw/ch`；
  - `scaleX=min(1, cw/baseW)`，`scaleY=min(1, ch/baseH)`；
  - `canvasW=baseW*scaleX`，`canvasH=baseH*scaleY`，画布居中；
  - widget 用百分比换算为像素进行 absolute 定位与尺寸（不使用 transform，避免字体/控件被拉伸变形）。
- z-order：
  - 置顶：将 widget 移到数组末尾；置底：移到数组开头；Viewer/Designer 均以数组顺序决定覆盖层级。

---

## 3.1) 计划拆分（Checklist，需确认后进入编码）

### CAN1 - Slider：value 右置（Designer + Viewer）
**目标**
- `slider` value（Badge）显示在 `<input type="range">` 的右侧；
- 极窄时自动换行：第一行 slider，第二行右对齐 value。

**涉及文件**
- `frontend/src/pages/Showcase.vue`
- `frontend/src/windows/ShowcaseWindow.vue`

**验收**
- 两处 `slider` 展示一致；拖动/松手发送逻辑不回归。

**回滚**
- 回滚上述两文件的 slider 区块布局。

---

### CAN2 - 配置结构扩展（Go + TS normalize）
**目标**
- 扩展 ShowcaseConfig schema 支持 `canvas_percent`：
  - Go：`app_showcase.go` 增加 layout canvas 字段与 normalize/clamp；
  - TS：`frontend/src/stores/showcase.ts` 增加类型与 normalize/clamp；
- 旧配置兼容：未识别 mode 默认回退到 columns。

**涉及文件**
- `app_showcase.go`
- `frontend/src/stores/showcase.ts`

**验收**
- 旧 profile 配置可正常加载；
- 新字段缺失/越界会被 normalize（不崩溃）。

**测试点**
- `go test ./... -count=1`（覆盖基本解析/normalize 路径）。

**回滚**
- revert schema 相关提交。

---

### CAN3 - Viewer：渲染 `canvas_percent`
**目标**
- `ShowcaseWindow.vue` 支持按 `screen.layout.mode` 渲染：
  - columns：保持现状；
  - canvas_percent：画布居中 + absolute widgets + z-order。

**涉及文件**
- `frontend/src/windows/ShowcaseWindow.vue`

**验收**
- `#/showcase-window?screenId=...` 在 canvas 模式可正常展示与控制；
- 窗口缩小：宽高都适配且不放大。

**回滚**
- 回滚该文件对 canvas 模式渲染的新增代码。

---

### CAN4 - Designer：编辑 `canvas_percent`（拖拽/缩放 + 置顶/置底）
**目标**
- 在 Designer 增加布局模式选择（columns/canvas_percent）；
- 切换到 canvas_percent 时自动写入 `baseWidth/baseHeight`（取当时容器尺寸）；
- widget 支持：
  - 移动：拖拽手柄（不影响控件本体交互）；
  - 缩放：右下角 resize handle；
  - 右键菜单：Edit/Remove + 置顶/置底；
- pointerup 才触发 `showcase.save()`（避免高频落盘）。

**涉及文件**
- `frontend/src/pages/Showcase.vue`

**验收**
- 可拖拽/缩放并保存；
- “置顶/置底”可改变遮挡关系；
- 保存后已打开 Viewer 实时同步。

**回滚**
- 回滚该文件对 canvas 模式编辑器的新增代码。

---

### CAN5 - 本地验证（构建 + 冒烟）
**命令**
```powershell
cd d:\project\MyFlowHub3\worktrees\feat-showcase-slider-right\MyFlowHub-Win
$env:GOWORK='off'
wails build -debug -skipembedcreate -nopackage
go test ./... -count=1
```

**手工冒烟**
1) `wails dev` → `#/showcase`：切换 layout、拖拽移动/缩放、右键置顶/置底；
2) `Open Window`：观察实时同步；缩放与 slider 右置效果。

---

### CAN6 - 归档变更（docs/change + plan_archive）
**目标**
- 新增变更文档与 plan 归档，便于审计交接。

**涉及文件（预期）**
- `docs/change/2026-03-04_showcase-canvas-layout.md`
- `docs/plan_archive/plan_archive_2026-03-04_win-showcase-canvas-layout.md`

---

## 4) 注意事项
- `canvas_percent` 的拖拽/缩放坐标转换需基于画布实际像素尺寸（已应用 scaleX/scaleY 后）。
- 最小尺寸与边界 clamp 必须在 normalize 与交互更新两处都生效，避免保存坏数据。

