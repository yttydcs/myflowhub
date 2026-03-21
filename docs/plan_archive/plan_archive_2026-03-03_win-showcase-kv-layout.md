# Plan - MyFlowHub-Win：Showcase 严格 Key-Value 展示（尽量单行）

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`feat/showcase-kv-layout`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-kv-layout\MyFlowHub-Win`
- Base：`main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
  - `d:\project\MyFlowHub3` 根目录 `AGENTS.md`（阶段纪律、worktree 禁令等）

---

## 0) 当前状态（项目现状）
- 已有 Showcase：
  - Designer 页面：`frontend/src/pages/Showcase.vue`
  - Viewer 窗口：`frontend/src/windows/ShowcaseWindow.vue`
- 当前（基于 `2026-03-03_showcase-panel-simplify`）widget 卡片已经做到 **title + value**，但布局仍以“上 title / 下 value”为主。
- 新需求：希望展示为**严格 Key-Value 形式**（key=title，value=展示/控制组件），并且**尽量不换行**。

---

## 1) 需求分析（已确认）

### 目标
1) Designer 与 Viewer 的 widget 卡片改为 **Key-Value 单行**展示（尽量不换行）。
2) `display` value 改为单行文本（不使用多行 `<pre>`），并提供 tooltip（`title` 属性）展示完整值。
3) `switch` value 仅显示开关本体（不显示 ON/OFF 文本）。
4) `slider` 在极窄情况下允许退化为两行（以保证可用性）；其他 widget 尽量保持单行。

### 范围（必须 / 不做）
- 必须：
  - Designer：`frontend/src/pages/Showcase.vue`
  - Viewer：`frontend/src/windows/ShowcaseWindow.vue`
  - 不修改发送/订阅逻辑（TopicBus publish、VarPool set/subscribe 等全部保持不变）。
- 不做：
  - 不改协议/wire，不改 `ShowcaseConfig` schema，不新增后端 API。

### 验收标准
- Designer + Viewer：widget 卡片展示为 key-value（title 与 value 尽量同一行，不自动换行）。
- `display` value：单行 + 省略号截断 + tooltip（title 属性）可查看完整值。
- `switch` value：仅开关本体。
- `slider`：正常情况下单行；极窄时可两行（允许 badge 与 slider 分行）。
- 既有交互不回归：Designer 拖拽手柄、右键菜单 Edit/Remove、Viewer 控制与实时刷新均正常。

### 风险
- 由于 Columns 布局下 card 宽度由列数决定，无法精确用 viewport breakpoint 判断“极窄”；需要通过 flex-wrap + min-width 策略自然退化，避免出现不可用的超窄 slider。

---

## 2) 架构设计（分析）

### 总体方案（采用）
- 保持现有数据流/协议调用不变，仅重排 widget 卡片内部 DOM 结构为 Key-Value Row：
  - key：`safeTitle(widget)`（必要时 `truncate`）
  - value：按 widget kind/mode 渲染（Send / 单行文本 / 开关 / slider）
- `display` value 使用单行 `<span>` + `:title="fullValue"`，并用 `truncate` 保证不换行。
- `slider` 的 value 区使用 `flex-wrap` 与 `min-width` 策略，使其在 card 极窄时自动换行（退化为两行），而不是把 slider 压缩到不可用宽度。

### 备选方案（不采用）
- 抽象通用 `KeyValueRow` 组件（复用更强，但 V1 仅两处页面，先以内联模板实现；若后续更多页面复用再抽组件）。

### 模块职责
- `Showcase.vue` / `ShowcaseWindow.vue`：仅负责展示与用户交互（布局与控件）。
- `frontend/src/stores/showcase.ts`：负责 TopicBus/VarPool 调用、订阅与快照维护（本次不改）。

### 数据/调用流（不变）
- 事件：`publishTopicButton(widget)` → TopicBus `PublishSimple(...)`
- 变量：
  - 进入 screen：收集 refs → `SubscribeSimple + GetSimple`
  - slider input：节流发送 `SendSimple(set, ...)`；松手 commit：`SetSimple(...)` await
  - switch toggle：`SetSimple(...)` await
  - 变更推送：`varpool.changed/deleted` → 更新快照 → UI 刷新

### 错误与安全
- 不新增外部输入面；保留现有 `busy/connected/selfNodeId` 的 disabled 策略。
- tooltip 使用 `title` 属性，仅展示字符串，不执行 HTML。

### 性能与测试策略
- 性能：仅调整 DOM 与 class，不新增 watchers/subscriptions；复杂度 O(widgets) 不变。
- 验证：
  - 构建：`GOWORK=off wails build -debug -skipembedcreate -nopackage`
  - 单测：`go test ./... -count=1`
  - 手工：窄窗口/多列布局下观察 slider 是否能自然换行且可用；display value tooltip 是否可读。

### 可扩展性设计点
- 若后续需要更多“展示组件”，可抽 `KeyValueRow` + “value renderer” 做可插拔扩展；本次仅做布局骨架一致化。

---

## 3.1) 计划拆分（Checklist，需确认后进入编码）

### KV1 - Designer：改为 Key-Value Row
**目标**
- `Showcase.vue` widget 卡片：title 与 value 同一行（尽量不换行）。
- 保留拖拽手柄与右键菜单（Edit/Remove）。

**涉及文件**
- `frontend/src/pages/Showcase.vue`

**验收**
- `topic_button`：title + Send 同行
- `display`：title + 单行 value（tooltip 可查看完整值）
- `switch`：title + 开关本体同行
- `slider`：title + badge + slider 尽量同行；极窄时可两行

**回滚**
- 回滚该文件到改动前版本（仅 UI）。

---

### KV2 - Viewer：改为 Key-Value Row
**目标**
- `ShowcaseWindow.vue` 与 Designer 保持一致布局策略。

**涉及文件**
- `frontend/src/windows/ShowcaseWindow.vue`

**验收**
- 与 KV1 一致，且独立窗口控制/更新正常。

**回滚**
- 回滚该文件到改动前版本（仅 UI）。

---

### KV3 - 本地验证（构建 + 冒烟）
**命令**
```powershell
cd d:\project\MyFlowHub3\worktrees\feat-showcase-kv-layout\MyFlowHub-Win
$env:GOWORK='off'
wails build -debug -skipembedcreate -nopackage
go test ./... -count=1
```

**手工冒烟**
1) `wails dev` → `#/showcase`：新增/编辑/右键/拖拽；观察单行布局与 tooltip
2) `Open Window` → `#/showcase-window?screenId=...`：观察单行布局；slider 极窄退化可用

---

### KV4 - 归档变更（docs/change）
**目标**
- 新增变更文档记录本次 Key-Value 布局调整与验证方式。

**涉及文件（预期）**
- `docs/change/2026-03-03_showcase-kv-layout.md`

---

## 4) 注意事项
- `rg`/lint 等工具不作为强制门槛，以仓库现有 `wails build` 与 `go test` 为准。
- slider 的“极窄退化”采用 flex-wrap + min-width 策略，避免引入复杂的 container query 依赖。

