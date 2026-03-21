# 2026-03-03 - Showcase：独立窗口 + Columns 布局 + 实时同步（V1）

## 背景 / 目标
在已落地的 Showcase MVP（Screen/Widget + TopicBus/Var 控制）基础上，补齐：
- Viewer 可独立访问（类似 Logs 的 Open Window）
- 用户可自定义布局（V1：Columns，多列响应式 + widget 列跨度）
- widget 支持拖拽排序（DnD）
- 多窗口实时同步（Designer 保存后 Viewer 自动更新）

## 变更内容

### 新增
- 前端：`frontend/src/windows/ShowcaseWindow.vue`（Viewer 窗口页，`#/showcase-window?screenId=...`）
- 前端：`frontend/src/lib/showcaseLayout.ts`（Columns 响应式列数/colSpan clamp 工具函数）
- 文档：`docs/design/2026-03-03_showcase-layout-architecture.md`（架构设计说明）

### 修改
- Go：`app_showcase.go`
  - 扩展 ShowcaseConfig schema：Screen/Widget 增加 `layout`（V1：`columns` + `colSpan`）
  - `SaveShowcaseConfig` 保存成功后广播 `showcase.config_changed`（Wails runtime event）
  - normalize：补齐默认值、clamp `maxColumns/minColumnWidth/gap/colSpan`、向后兼容旧配置
- Go：`app_showcase_test.go`
  - 新增/补齐 layout 默认值与 clamp 行为测试
- 前端：`frontend/src/stores/showcase.ts`
  - 扩展 types/normalize：支持 screen/widget layout
  - 增加 `fixedScreenId`（Viewer 固定 screenId 渲染/订阅）
  - 监听 `showcase.config_changed`：自动 reload config + leave/enter 刷新订阅（带最小 coalesce）
- 前端：`frontend/src/pages/Showcase.vue`
  - Screen 级 Columns 配置（`maxColumns`、`minColumnWidth`）
  - widget `colSpan` 编辑（创建/编辑对话框）
  - widgets 区域改为 Columns grid 预览 + `colSpan` 渲染
  - DnD 拖拽手柄排序（仅 drop 保存）
  - Open Window：打开 `#/showcase-window?screenId=...`（同 Screen 可开多个窗口）
- 前端路由：`frontend/src/router/index.ts`
  - 新增 `/showcase-window`（`meta.layout="window"`）

## Plan 任务映射
- V1：Go schema 扩展 + `showcase.config_changed` 广播（`app_showcase.go` / `app_showcase_test.go`）
- V2：前端 store 支持 layout + Viewer override + config_changed reload（`frontend/src/stores/showcase.ts`）
- V3：Designer Columns 配置 + widget colSpan + DnD + Open Window（`frontend/src/pages/Showcase.vue`）
- V4：Viewer 窗口页 + 路由（`frontend/src/windows/ShowcaseWindow.vue`、`frontend/src/router/index.ts`）
- V5：Wails bindings 同步 + 回归验证（`wails generate module`、`npm run build`、`go test`）

## 关键设计决策与权衡
- **实时同步机制**：后端直接 `runtime.EventsEmit("showcase.config_changed")`，前端监听后 reload（不引入新的子协议/服务）。
  - 优点：实现简单，复用现有跨窗口事件分发能力（Logs 已验证）。
  - 代价：V1 使用 leave/enter 全量刷新订阅，可能产生短暂的订阅抖动；后续可做订阅 diff 优化。
- **Columns 响应式**：使用 `ResizeObserver` 计算列数 `floor((w+gap)/(minWidth+gap))` 并 clamp 到 `maxColumns`。
  - 优点：同时满足“自动减列”与“最大列数上限”，并与 `colSpan` clamp 一致。
  - 代价：需要 JS 监听容器宽度（但实现局部、可复用）。
- **DnD 排序**：V1 采用原生 HTML5 Drag&Drop，拖拽手柄触发，drop 时保存。
  - 优点：不引入额外依赖；避免拖动过程频繁写入。
  - 代价：交互细节不如专业 sortable 库丰富（可在 V2 再评估替换）。

## 测试与验证
- Go：`GOWORK=off go test ./... -count=1 -p 1`
- Wails bindings：`GOWORK=off wails generate module`
- 前端构建：`cd frontend; npm ci; npm run build`
- 结果：以上命令在本 worktree 均执行通过。
- 手工建议：
  - `#/showcase` → Open Window 打开 Viewer
  - 同一 Screen 打开多个 Viewer，Designer 保存后 Viewer 自动同步（排序 / layout / widget 变更）
  - 调整窗口宽度验证自动减列；验证 `colSpan` 在窄窗口下会被 clamp
  - `screenId` 不存在时 Viewer 显示 “Screen not found”

## 潜在影响
- 多窗口同时编辑 ShowcaseConfig 时，后保存者会触发其他窗口 reload，可能覆盖其未保存的本地编辑状态（V1 未做冲突合并/提示）。
- `showcase.config_changed` 触发 reload 会导致 leave/enter，全量刷新订阅（可接受但需关注高频保存场景）。

## 回滚方案
- 回滚该分支变更即可恢复到 Showcase MVP（无独立窗口/布局/DnD/同步）：
  - 移除 `/showcase-window` 路由与 `ShowcaseWindow.vue`
  - 移除 layout schema 字段与 normalize
  - 移除 `showcase.config_changed` 广播与前端监听
