# Plan - MyFlowHub-Win：Showcase 面板简化（仅 title + value）+ 设计页右键菜单

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`feat/showcase-panel-simplify`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-panel-simplify\MyFlowHub-Win`
- Base：`main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
  - `d:\project\MyFlowHub3` 根目录 `AGENTS.md`（阶段纪律、worktree 禁令等）

---

## 0) 当前状态（复用能力）

### Showcase 现有能力
- Designer 页面：`frontend/src/pages/Showcase.vue`
  - Widgets：支持 `topic_button` 与 `var`
  - Var 控制：`display/slider/switch`（`auto` 可基于 var type 推断）
  - 支持拖拽排序（DnD）与编辑弹窗（Overlay）
  - 当前卡片仍展示较多“元信息”（Target/Span/VarKey/payload/minmax 文案等）
- Viewer 窗口：`frontend/src/windows/ShowcaseWindow.vue`
  - 支持独立访问 `#/showcase-window?screenId=...`
  - 当前卡片同样展示较多“元信息”
- Store：`frontend/src/stores/showcase.ts`
  - TopicBus：`PublishSimple(sourceId, targetId, ...)`
  - VarPool：`SubscribeSimple/GetSimple/SetSimple/SendSimple`
  - 实时订阅：监听 `varpool.changed/deleted` 更新快照

### 约束（本 workflow 接受）
- 不改任何子协议/wire、不改数据结构 `ShowcaseConfig`，仅调整 UI 展示与交互入口（Edit/Remove 迁移至右键菜单）。
- 不引入第三方 UI 组件库（ContextMenu 先在页面内实现，避免依赖膨胀）。

---

## 1) 需求分析（已确认）

### 目标
1) 简化 Designer 与 Viewer 的 widget 卡片：只保留 **title + value（控制组件）**。
2) Designer：保留拖拽手柄；`Edit/Remove` 等“设计操作按钮”从卡片右上角移除，改为 **右键菜单**。

### 范围（必须 / 可选 / 不做）
- 必须：
  - Designer（`Showcase.vue`）：
    - 卡片仅保留 `title` 与 `value` 区域；
    - 右键菜单：`Edit`、`Remove`（Remove 仍需确认弹窗）；
    - 拖拽排序保持可用。
  - Viewer（`ShowcaseWindow.vue`）：
    - 卡片仅保留 `title` 与 `value`；
    - `topic_button` 不再显示 payload 预览，仅保留 Send 按钮；
    - `slider` 允许保留当前值的小 Badge。
- 不做：
  - 不改协议服务（TopicBus/VarPool/Session），不改存储结构，不新增后端 API。

### 验收标准
- 两处卡片不再出现：`Target/Span`、`Var(owner/name)`、payload 预览、slider 的 min/max/step/throttle 文案等。
- Designer：右键任意 widget 卡片可打开菜单并执行 `Edit/Remove`；拖拽排序仍可用。

---

## 2) 架构设计（分析结论）

### 方案 A（采用）：页面内轻量 ContextMenu（无外部依赖）
- `Showcase.vue` 内维护 `contextMenu` 状态（open/x/y/widgetId）。
- 卡片容器监听 `@contextmenu.prevent` 打开菜单；菜单用 `position: fixed` 渲染并做边界钳制。
- 关闭策略：点击空白处、`Esc`、`scroll`、`resize` 自动关闭，避免菜单悬挂。

### 关键权衡
- 不引入通用组件库：V1 只用于 Showcase Designer，先最小实现；若后续多页面复用再抽 `components/ui/ContextMenu.vue`。

---

## 3) 任务清单（Checklist）

### SC1 - Designer：实现右键菜单（Edit/Remove）
**目标**
- 移除卡片右上角 `Edit/Remove` 按钮；
- 右键菜单包含 `Edit` 与 `Remove`，并复用现有函数 `openEditWidget(widget)` / `removeWidget(widget)`。

**涉及文件**
- `frontend/src/pages/Showcase.vue`

**验收**
- 右键卡片弹出菜单；点击 `Edit` 打开编辑弹窗；点击 `Remove` 触发确认并删除 widget；
- 点击空白处或按 `Esc` 关闭菜单；
- 不出现系统默认右键菜单。

**测试点**
- 菜单打开位置不出屏；滚动/resize 后自动关闭；
- 拖拽手柄仍可拖拽排序（右键不会干扰拖拽）。

**回滚**
- 恢复原卡片右上角按钮与移除菜单逻辑（仅影响 UI，易回滚）。

---

### SC2 - Designer：卡片展示简化（仅 title + value）
**目标**
- 删除卡片中的元信息与辅助块，仅保留 `safeTitle(widget)` 与控制组件：
  - topic_button：Send
  - var：display/switch/slider（slider 可保留数值 Badge）

**涉及文件**
- `frontend/src/pages/Showcase.vue`

**验收**
- 卡片内不再出现 Target/Span、VarKey、payload、slider 文案等；
- 控制行为不变（Send/Slider/Switch 正常工作）。

**测试点**
- 变量值刷新后 UI 正常更新；
- slider 拖动仍会触发发送（throttle 逻辑保持不变）。

**回滚**
- 回滚该文件到合并前版本即可。

---

### SC3 - Viewer：卡片展示简化（仅 title + value）
**目标**
- Viewer 中移除卡片元信息与 payload/VarKey/minmax 文案，仅保留 `title + value`（与 Designer 保持一致）。

**涉及文件**
- `frontend/src/windows/ShowcaseWindow.vue`

**验收**
- Viewer 卡片展示与 Designer 一致；`topic_button` 仅保留 Send；`slider` 保留数值 Badge + 滑条。

**测试点**
- `#/showcase-window?screenId=...` 在独立窗口可正常控制并实时更新。

**回滚**
- 回滚该文件到合并前版本即可。

---

### SC4 - 本地验证（构建/类型检查 + 手工冒烟）
**目标**
- 确保前端可构建、无 TS/ESLint 明显错误（以仓库现有命令为准）。

**建议命令（推荐：Wails 生成 bindings + 构建）**
```powershell
cd d:\project\MyFlowHub3\worktrees\feat-showcase-panel-simplify\MyFlowHub-Win
$env:GOWORK='off'
wails build -debug -skipembedcreate -nopackage
```

如果已存在 `frontend/wailsjs`（bindings 已生成），也可单独：
```powershell
cd d:\project\MyFlowHub3\worktrees\feat-showcase-panel-simplify\MyFlowHub-Win\frontend
npm run build
```

**手工冒烟**
1) 启动 `wails dev`，打开 `#/showcase`；
2) 添加/编辑 widget，右键卡片操作 `Edit/Remove`；
3) Open Window → `#/showcase-window?...`，检查卡片简化与控制正常。

---

### SC5 - 归档变更（docs/change）
**目标**
- 在本仓库 `docs/change/` 记录变更背景、内容、验收与回滚方式。

**涉及文件（预期）**
- `docs/change/2026-03-03_showcase-panel-simplify.md`

**验收**
- 文档可脱离对话，直接复现“为什么改、改了什么、如何验证”。

---

## 4) 风险与注意事项
- WebView2/浏览器默认 context menu：必须 `@contextmenu.prevent`，否则会与自定义菜单叠加。
- 右键菜单与拖拽：拖拽只绑定在手柄上（现状如此），但仍需避免菜单覆盖手柄导致误触。

