# 2026-03-23_win-showcase-rich-display-widgets

## 变更背景 / 目标

- 在现有 Win Showcase 基础上补充更丰富的单值展示组件。
- 目标是在保持当前简洁风格的前提下，让 Showcase 更适合做轻量状态面板。

## 具体变更内容

### 新增

- `docs/requirements/showcase-display-widgets.md`
  - 新增 Showcase 展示组件的长期需求、范围与验收标准。
- `docs/specs/showcase-display-widgets.md`
  - 新增 Showcase 展示模式的技术契约与兼容性约束。
- `frontend/src/components/showcase/ShowcaseWidgetCardContent.vue`
  - 新增 Editor / Viewer 共享 widget 渲染组件。

### 修改

- `app_showcase.go`
  - 扩展 `ShowcaseVarWidget.mode` 支持 `metric` / `badge` / `progress`。
  - `progress` 默认类型改为 `float64`。
- `app_showcase_test.go`
  - 增加新 mode 保留与 `progress` 默认类型回归测试。
- `frontend/src/stores/showcase.ts`
  - 扩展 `VarWidgetMode` 类型与前端 normalize 逻辑。
- `frontend/src/pages/Showcase.vue`
  - Widget 编辑对话框新增 `Metric` / `Badge` / `Progress` 模式。
  - `Progress` 复用范围配置，`Slider` 专属字段改为条件显示。
  - Editor 预览改为复用共享渲染组件。
- `frontend/src/windows/ShowcaseWindow.vue`
  - Viewer 改为复用共享渲染组件，与 Editor 保持一致。
- `docs/requirements/README.md`
  - 增加 Showcase requirement 导航。
- `docs/specs/README.md`
  - 增加 Showcase spec 导航。
- `docs/change/README.md`
  - 增加本次归档入口。
- `plan.md`
  - 记录阶段分析、任务映射、review 结论和归档状态。

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Related requirements

- `docs/requirements/showcase-display-widgets.md`

## Related specs

- `docs/specs/showcase-display-widgets.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\topicbus.md`

## 对应 plan.md 任务映射

- `SHW-1`
  - 补齐稳定 requirements/specs 与索引。
- `SHW-2`
  - 扩展 Go / store 的展示模式 schema 与 normalize。
- `SHW-3`
  - 接入共享渲染组件并完成 Editor / Viewer 新模式渲染。
- `SHW-4`
  - 执行 `go test`、`wails generate module`、`npm run build` 与 code review。
- `SHW-5`
  - 编写 change 归档并更新索引。

## 关键设计决策与权衡

- 选择“扩展 `var` mode”而不是新增 widget kind。
  - 优点：兼容现有订阅、存储模型和表单结构，改动面更小。
- 新增共享渲染组件，而不是在 Editor / Viewer 各自追加分支。
  - 优点：减少逻辑漂移，后续扩 mode 更容易审计。
- `progress` 复用现有 slider 范围配置，而不引入新配置块。
  - 优点：保持 schema 轻量。
  - 代价：需要在 UI 中明确“这里只是显示范围，不是交互 slider”。

## 测试与验证方式 / 结果

- Go：
  - `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过。
- Wails bindings：
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过，已生成 `frontend/wailsjs/**`。
- 前端构建：
  - `npm ci`
  - `npm run build`
  - 结果：通过。
- 手工 / UI：
  - 未执行 `chrome-devtools` 冒烟。
  - 原因：当前 turn 以内完成了 build-level 验证，但未启动带真实 Showcase 数据的 Wails 运行会话。

## 潜在影响

- 新增 display-only mode 会让 Showcase 配置里出现更多 `var.mode` 取值；旧客户端若不支持这些值，会回退为 `auto` 或表现不一致。
- 前端打包仍存在既有的大 chunk 告警，但本轮未新增新的构建失败或 runtime 依赖问题。

## 回滚方案

- 回退以下文件即可恢复为旧的 Showcase 展示能力：
  - `app_showcase.go`
  - `app_showcase_test.go`
  - `frontend/src/stores/showcase.ts`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/components/showcase/ShowcaseWidgetCardContent.vue`
- 如确认不保留长期文档，同时回退：
  - `docs/requirements/showcase-display-widgets.md`
  - `docs/specs/showcase-display-widgets.md`
  - 相关 README 索引修改

## 子Agent执行轨迹

- 未派发子Agent。
