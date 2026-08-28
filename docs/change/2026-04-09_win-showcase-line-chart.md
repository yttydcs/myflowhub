# 2026-04-09_win-showcase-line-chart

## 变更背景 / 目标

- 在现有 Win Showcase `var` widget 基础上补充折线图展示能力。
- 用户要求先只实现折线图，并且可以调节显示的时间范围与粒度。
- 数据不做后端历史持久化，仅依赖当前 app session 内的前端内存采样。

## 具体变更内容

### 修改

- `docs/requirements/showcase-display-widgets.md`
  - 将 `line_chart`、时间范围/粒度、前端 session 内存采样与无后端历史查询写入长期 requirements。
- `docs/specs/showcase-display-widgets.md`
  - 更新 `var.mode`、`chart { rangeMs, bucketMs }`、采样 retention 与降级约束。
- `app_showcase.go`
  - 扩展 `ShowcaseVarWidget.mode` 支持 `line_chart`。
  - 新增 `ShowcaseVarChart` 持久化配置与 normalize。
- `app_showcase_test.go`
  - 增加 `line_chart` mode 保留、默认类型与图表配置 clamp 回归测试。
- `frontend/src/stores/showcase.ts`
  - 扩展 `VarWidgetMode`、widget normalize 与 `chart` 配置。
  - 在现有 `upsertSnapshot()` 路径上采集数值样本到前端内存。
  - 暴露 `lineChartState()` 供组件按可见窗口取数。
- `frontend/src/lib/showcaseChart.ts`
  - 新增图表 helper，集中处理配置归一化、样本 retention、bucket 聚合与空态判断。
- `frontend/src/components/showcase/ShowcaseWidgetCardContent.vue`
  - 在共享卡片组件内新增折线图分支，提供 `Range` / `Granularity` 轻量控件和 SVG 折线渲染。
- `frontend/src/pages/Showcase.vue`
  - Widget 对话框新增 `Line Chart` 模式及默认 `rangeMs` / `bucketMs` 配置项。
- `frontend/src/i18n/messages/showcase.ts`
  - 补充 `Line Chart`、`Range`、`Granularity` 与降级文案。
- `plan.md`
  - 回填 Stage 3.2 / 3.3 / 4 的实现、review 与归档状态。

### 新增

- `frontend/src/lib/showcaseChart.test.ts`
  - 覆盖配置归一化、样本裁剪、bucket 聚合与样本不足场景。
- `frontend/src/components/showcase/ShowcaseWidgetCardContent.test.ts`
  - 覆盖折线图控件与降级文案渲染。

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `docs/requirements/showcase-display-widgets.md`

## Related specs

- `docs/specs/showcase-display-widgets.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`

## Related lessons

- `D:\project\MyFlowHub3\worktrees\feat-showcase-chart-widgets\docs\lessons\frontend-worktree-wailsjs-missing.md`

## 对应 plan.md 任务映射

- `LCH-1`
  - 更新 Showcase requirements/specs，纳入 `line_chart` 和前端内存采样边界。
- `LCH-2`
  - 扩展 Go / 前端配置模型与 normalize，支持 `line_chart` 和 `chart` 配置持久化。
- `LCH-3`
  - 在前端 store 中实现按变量 key 复用的样本采集、裁剪与 bucket 聚合。
- `LCH-4`
  - 接入共享折线图渲染与界面控件，保持 Editor / Viewer 一致。
- `LCH-5`
  - 执行 `go test`、定向 Vitest、`npm run build` 与 Stage 3.3 review。
- `LCH-6`
  - 编写 change 归档并同步控制面 `docs/change`、`docs/plan` 与入口索引。

## 经验 / 教训摘要

- 折线图继续扩展现有 `var` mode，比新增 widget kind 更容易复用订阅、持久化和编辑表单。
- 图表样本不持久化时，最稳妥的语义是“配置持久化、数据不持久化”：默认范围/粒度写入 widget 配置，实际采样只留在前端 session 内存。
- 新 worktree 的前端验证仍要遵守既有链路：`npm ci`、`GOWORK=off wails generate module`、`npm run build`。

## 可复用排查线索

- 症状
  - `npm run build` 报 `Failed to resolve import "../../wailsjs/..."`
  - 折线图显示 `No value yet.`、`No numeric value yet.`、`No samples in selected range.` 或 `Need more samples to draw trend.`
- 触发条件
  - 新建 Win worktree 后尚未生成 `frontend/wailsjs`
  - 当前变量不是数值、当前 session 还没有累计到足够样本、或可见时间窗内没有有效点
- 关键词
  - `line_chart`
  - `chartHistory`
  - `frontend/wailsjs`
  - `Range`
  - `Granularity`
- 快速检查
  - 确认 `frontend/wailsjs` 已存在；如不存在先执行 `$env:GOWORK='off'; wails generate module`
  - 确认 widget `mode` 为 `line_chart`
  - 确认变量当前值可解析为数值，并等待至少两个可见 bucket

## 关键设计决策与权衡

- 选择“扩展 `var.mode`”而不是新增 widget kind。
  - 优点：复用现有 widget schema、订阅链路和编辑器表单，改动面最小。
- 选择轻量 SVG 而不是引入第三方图表库。
  - 优点：依赖面更小，样式更容易保持和现有 Showcase 一致。
- 选择“对话框配置持久化 + 卡片控件临时覆盖”而不是把卡片内调整直接写回配置。
  - 优点：满足界面内快速调节，同时避免普通查看动作意外改写 Showcase 配置。
  - 代价：卡片内的临时选择在重新打开后不会保留。

## 测试与验证方式 / 结果

- Go：
  - `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过。
- 前端定向测试：
  - `npm run test -- src/lib/showcaseChart.test.ts src/components/showcase/ShowcaseWidgetCardContent.test.ts`
  - 结果：通过。
- Wails bindings：
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过；控制台出现重复 `Not found: time.Time` 噪声，但未阻断生成。
- 前端构建：
  - `npm run build`
  - 结果：通过；保留既有的大 chunk warning，本轮未新增构建失败。
- 手工 / UI：
  - 未启动真实 Wails 会话做运行态冒烟。
  - 原因：当前 turn 内已完成 build 级验证，但未拉起带真实 Showcase 数据的运行环境。

## 潜在影响与回滚方案

### 潜在影响

- Showcase 配置里会出现新的 `var.mode = line_chart` 和 `chart` 配置字段；旧客户端若不支持，可能回退为 `auto` 或失去图表显示。
- 折线图只保留当前 session 内存样本，重开应用后不会显示上一次运行期间积累的历史。

### 回滚方案

- 回退以下文件即可恢复为原有 Showcase 展示能力：
  - `app_showcase.go`
  - `app_showcase_test.go`
  - `frontend/src/stores/showcase.ts`
  - `frontend/src/lib/showcaseChart.ts`
  - `frontend/src/lib/showcaseChart.test.ts`
  - `frontend/src/components/showcase/ShowcaseWidgetCardContent.vue`
  - `frontend/src/components/showcase/ShowcaseWidgetCardContent.test.ts`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/i18n/messages/showcase.ts`
- 如果确认不保留长期文档，同时回退：
  - `docs/requirements/showcase-display-widgets.md`
  - `docs/specs/showcase-display-widgets.md`
  - 对应索引改动

## 子Agent执行轨迹

- 未派发子Agent。
