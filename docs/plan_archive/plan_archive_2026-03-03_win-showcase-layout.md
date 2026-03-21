# Plan - MyFlowHub-Win：Showcase 独立窗口 + 自定义布局（Columns）+ 实时同步

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`feat/win-showcase-layout`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-showcase-layout\MyFlowHub-Win`
- Base：`main`（包含已合并的 Showcase MVP）
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
  - `d:\project\MyFlowHub3` 根目录 `AGENTS.md`（阶段纪律、worktree 禁令等）

---

## 0) 当前状态（复用能力）
- 已有 Showcase MVP：
  - 页面：`frontend/src/pages/Showcase.vue`
  - store：`frontend/src/stores/showcase.ts`
  - Go 配置持久化：`app_showcase.go`（profile-scoped `showcase.config`）
- TopicBus：`TopicBusService.PublishSimple(...)`（payload Auto：合法 JSON → JSON；否则字符串）
- VarPool：`Get/Subscribe/Unsubscribe/Send/Set` + Wails events `varpool.changed/deleted`
- Window 模式示例（独立访问）：Logs 的 `#/log-window` + `window.open`（见 `frontend/src/pages/Logs.vue`、`frontend/src/router/index.ts`）

---

## 1) 需求分析（已确认）

### 目标
在现有 Showcase 基础上增强：
1) 每个 Screen 可被“独立访问”（像 Logs 一样单独打开窗口）。
2) 用户可自定义布局（V1 先做 Columns 多列布局：拖拽排序 + 控制宽度/列跨度）。
3) 具备响应式能力：窗口变窄时自动减少列数；窗口变宽时最多不超过用户设置的最大列数。
4) Viewer 窗口允许控制（TopicBus publish / VarStore set）。
5) 多窗口实时同步：Designer 保存配置后，已打开的 Viewer 窗口自动更新。

### 范围
- 必须（V1）：
  - 新增 Viewer 路由页面：`#/showcase-window?screenId=...`（`meta.layout=window`）。
  - Designer 页面支持：
    - 为每个 Screen 打开 Viewer 窗口（允许同一 Screen 同时打开多个窗口）。
    - Columns 布局设置（screen 级）：`maxColumns`、`minColumnWidth`（gap 固定默认值，先不暴露）。
    - 每个 widget 支持 `colSpan` 配置（默认 1，渲染时自动 clamp 到当前列数）。
    - 支持拖拽排序 widgets（必须）。
  - Viewer 页面：
    - 固定打开指定 `screenId`（不跟随 Designer 当前选择）。
    - 若 screenId 不存在：显示 “Screen not found” 并停止（不 fallback）。
  - 实时同步：
    - 后端 `SaveShowcaseConfig` 成功后广播事件 `showcase.config_changed`；
    - 所有窗口收到后 reload config；Viewer 需刷新订阅集合（leave/enter）。
  - 保持既有行为：变量订阅去重、`throttleMs=0` 语义、未登录禁用控制等不回退。

- 可选 / 不做（本轮）：
  - 绝对布局（absolute / x,y,w,h）
  - 画布缩放布局（scale / designWidth,designHeight + transform scale）
  - 更复杂的拖拽定位/缩放（仪表盘式）

### 默认值建议（可调）
- Columns：
  - `maxColumns=3`
  - `minColumnWidth=360px`
  - `gap=16px`（先固定，不暴露）
- Widget：
  - `colSpan=1`

### 验收标准（MVP）
1) 在 `#/showcase` 选择某个 Screen，点击“Open Window”能打开 `#/showcase-window?screenId=...`。
2) 同一 Screen 可开多个 Viewer 窗口（窗口互不覆盖）。
3) 在 Designer 中拖拽调整 widget 顺序并保存后：
   - Designer 自身顺序正确；
   - 已打开的 Viewer 窗口实时同步顺序（无需刷新）。
4) 在 Designer 中调整 Columns 配置（maxColumns/minColumnWidth）并保存后：
   - Viewer 响应式列数正确（变窄自动减列，变宽最多 maxColumns）。
5) `screenId` 不存在时：Viewer 显示 “Screen not found” 并停止（不崩溃）。
6) Viewer 内 widget 控制可用（TopicBus publish、Var slider/switch 写入）；未连接/未登录时禁用并提示清晰。
7) 切换 Screen/关闭窗口后，不应产生订阅泄漏（Unsubscribe 正常）。

### 风险
- 多窗口同步：Designer save → Viewer reload 可能触发订阅抖动；V1 允许简单 leave/enter，后续可做差分订阅优化。
- Drag&Drop：原生 HTML5 DnD 细节较多，需要用“拖拽手柄”避免干扰 widget 内部交互。

---

## 2) 架构设计（分析）

### 总体方案（选型与理由）
V1 采用“Columns 网格布局 + 轻量拖拽排序”，不引入新布局库，避免依赖膨胀；为后续 absolute/scale 预留 schema。

### 模块职责
- Go（Wails backend）
  - 扩展 `ShowcaseConfig` schema（screen/layout + widget/colSpan），并保持向后兼容。
  - `SaveShowcaseConfig` 成功后 `EventsEmit("showcase.config_changed")` 广播同步事件。
- 前端 store：`frontend/src/stores/showcase.ts`
  - 解析/规范化 layout 字段与默认值。
  - 提供 viewer 所需的“固定 screenId 渲染”能力（override screenId，不写回配置）。
  - 监听 `showcase.config_changed`：reload config，并按需 leave/enter 更新订阅。
- 前端页面：
  - `frontend/src/pages/Showcase.vue`：Designer（打开窗口、布局参数编辑、DnD 排序）。
  - `frontend/src/windows/ShowcaseWindow.vue`（新增）：Viewer（固定 screenId 渲染 + 控制）。

### 数据 / 调用流
- Designer 保存配置：
  - `SaveShowcaseConfig` → 存储成功 → `EventsEmit("showcase.config_changed")`
  - 各窗口收到事件 → `store.load()` → `leave()` / `enter()`（Viewer 需）
- Columns 响应式列数：
  - Viewer/Designer 监听容器宽度 W → 计算 `cols = clamp(1,maxColumns,floor((W+gap)/(minColumnWidth+gap)))`
  - widget `colSpan` 渲染时 clamp 到 `cols`

### 接口草案
- Wails events：
  - `showcase.config_changed`（payload 可为空；V1 仅用作通知）
- Config schema（草案）：
  - `ShowcaseScreen.layout = { mode:"columns", columns:{maxColumns,minColumnWidth,gap} }`
  - `ShowcaseWidget.layout = { colSpan }`

### 错误与安全
- UI 强校验：
  - `maxColumns` 合法范围（建议 1~12）
  - `minColumnWidth` 合法范围（建议 200~1200）
  - `colSpan` 合法范围（建议 1~maxColumns）
- Viewer screenId 不存在：仅提示，不做 fallback。

### 性能与测试策略
- 性能关键点：
  - `showcase.config_changed` 事件触发 reload 可做最小节流（避免短时间多次 save 导致频繁 reload）。
  - DnD drop 才触发 save，dragover 不保存。
- 测试策略：
  - Go 单测：layout 默认值、`colSpan` clamp、版本迁移/向后兼容、事件广播不 panic。
  - 前端构建：`npm run build`
  - 手工：多窗口同步、响应式列数、DnD 排序与不干扰 widget 控制。

### 可扩展性设计点
- `layout.mode` 预留 `absolute/scale`，后续仅需扩展 layout 字段与渲染器。
- 可在 V2 做订阅差分与配置 diff 同步，减少 leave/enter 的抖动。

---

## 3.1) 计划拆分（Checklist）

> 进入 3.2 前必须：本 plan.md 获得确认（阻塞：是）。

### V1 - Go：扩展配置 schema + 广播 config_changed
- 目标：
  - 增加 screen/widget layout 字段与默认值；
  - `SaveShowcaseConfig` 成功后发出 `showcase.config_changed`。
- 涉及文件（预期）：
  - `app_showcase.go`
  - `app_showcase_test.go`
- 验收：
  - 旧配置（无 layout）加载后自动补齐 columns 默认值；
  - `colSpan` 缺失/非法时能兜底为 1；
  - `SaveShowcaseConfig` 保存后会触发 `showcase.config_changed`（手工在前端监听验证）。
- 测试点：
  - `GOWORK=off go test ./... -count=1 -p 1`
- 回滚点：
  - 移除新增 layout 字段与事件广播，保持原 Showcase 行为不变。

### V2 - 前端：showcase store 支持 layout + viewer override + config_changed reload
- 目标：
  - 扩展 types/normalize 支持 layout；
  - viewer 可固定 screenId 渲染（override，不写回配置）；
  - 监听 `showcase.config_changed` 自动 reload 并刷新订阅。
- 涉及文件（预期）：
  - `frontend/src/stores/showcase.ts`
- 验收：
  - `showcase.config_changed` 到达后，Viewer 画面自动更新；
  - leave/enter 不产生订阅泄漏；
  - DnD reorder 后（保存触发事件）Viewer 顺序同步。
- 测试点：
  - 手工：开两个 viewer 窗口同时观察同步。
- 回滚点：
  - 仅保留 Designer 行为，不影响原 `#/showcase` 使用。

### V3 - 前端：Designer（DnD 排序 + Columns 配置 + Open Window）
- 目标：
  - 增加 Columns 配置与 widget colSpan 配置入口；
  - 实现拖拽排序（必须，使用拖拽手柄）；
  - 可打开 viewer 窗口（screenId 固定）。
- 涉及文件（预期）：
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/router/index.ts`（若新增窗口路由）
- 验收：
  - 拖拽排序只在 drop 时保存；
  - Columns 响应式预览正确；
  - Open Window 打开 viewer。
- 测试点：
  - `cd frontend; npm ci; npm run build`
- 回滚点：
  - 移除 UI 编辑入口与 DnD，不影响既有 widgets 控制能力。

### V4 - 前端：Viewer 窗口页（固定 screenId 渲染）
- 目标：
  - 新增 `frontend/src/windows/ShowcaseWindow.vue`；
  - 支持 screen not found 提示；
  - 支持控制与自动同步。
- 涉及文件（预期）：
  - `frontend/src/windows/ShowcaseWindow.vue`（新增）
  - `frontend/src/router/index.ts`（新增路由 `/showcase-window`，meta.layout=window）
- 验收：
  - `#/showcase-window?screenId=...` 可独立访问；
  - screenId 不存在时提示并停止；
  - 多窗口同时打开同一 screenId 正常。
- 测试点：
  - `npm run build`
- 回滚点：
  - 删除窗口路由与页面文件。

### V5 - bindings 同步 + 回归验收
- 目标：
  - 生成并校验 Wails bindings；完成 MVP 验收条目。
- 验收：
  - `GOWORK=off wails generate module` + `npm run build` + `go test` 通过；
  - 手工验收 1~7 全部通过。
- 回滚点：
  - 回滚本分支全部提交。

---

## 3.3) Code Review（完成编码后执行）
- 需求覆盖：独立窗口、多窗口同屏、DnD 排序、Columns 响应式、实时同步、screen not found
- 架构合理性：schema 可扩展、事件同步机制合理、store 不产生副作用写回
- 性能风险：DnD drop 才保存；reload/leave/enter 不频繁；订阅去重
- 可读性与一致性：代码风格与现有 pages/stores 一致
- 稳定性与安全：输入校验、未登录禁用、异常提示清晰
- 测试覆盖：Go 单测 + 前端 build + 手工多窗口验收

---

## 4) 归档变更（完成 Review 后执行）
- 在 worktree 根目录创建 `docs/change/` 并新增文档：`docs/change/2026-03-03_showcase-layout.md`
- 内容必须包含：背景/目标、变更内容、plan 任务映射、关键决策与权衡（DnD/实时同步/响应式算法）、测试结果、影响与回滚方案
