# Plan - Win 前端：统一修复 Modal 遮罩未覆盖 Header

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`fix/win-overlay-mask`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-overlay-mask\MyFlowHub-Win`
- Base：`origin/main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
- 运行与验收（建议）
  1) 启动 server（默认 `:9000`）：
     - `cd ..\MyFlowHub-Server`
     - `go run ./cmd/hub_server`
  2) 启动 Win：
     - `cd ..\MyFlowHub-Win`
     - `wails dev`

## 背景（问题与根因）
- 现象：在 `Devices` 页点击节点后弹出的详情 Modal，遮罩未覆盖顶部 Header（Connected / Active Profile），导致顶部仍可见且可点。
- 根因（工程层面）：页面组件使用了带 `transform` 的 enter 动画（`tailwindcss-animate`），导致其后代 `position: fixed` 的参照系不再是 viewport，从而遮罩只覆盖页面内容区域而非全窗口。
- 影响面：同样写法（页面内 `fixed inset-0 ... bg-black/40`）的弹窗遮罩在其他页面也存在潜在同类问题，需要统一收敛。

## 目标
1) 任何页面打开 Modal 时，遮罩必须覆盖整个窗口（包含顶部 Header），并阻止背景交互。
2) 统一收敛遮罩实现，避免每个页面复制粘贴同类 DOM/样式。
3) 不修改路由切换动画与页面布局，仅修复遮罩覆盖与交互。

## 范围与约束
- 必须：
  - 统一修复前端所有 “Modal + 背景遮罩” 的实现，使其覆盖 Header。
  - 保持现有各页面的打开/关闭行为（点击遮罩空白处关闭、按钮关闭等）不变。
  - Profile 菜单的“全屏透明 click-catcher”也统一迁移到同一套 Overlay 机制（不加黑色背景，仅用于拦截点击并关闭菜单）。
- 不做：
  - 不引入新的 UI 框架/依赖（仅使用现有 Vue + Tailwind）。
  - 不重构路由动画（保留现有 `animate-in ...`）。

## 决策（已确认）

### D1. Esc 关闭默认开启
- 结论：**默认开启**（`closeOnEsc=true`）。
- 要求：同一时刻多个 Overlay 并存时，Esc **只关闭最上层 Overlay**（避免一次关闭多个）。

### D2. Esc 关闭范围：Modal + Profile 菜单
- 结论：Modal 与 Profile 菜单都支持 Esc 关闭（Profile 菜单等价 popover 行为）。

## 当前状态（事实，可审计）
- 存在页面内遮罩的点位（`fixed inset-0`）：
  - `frontend/src/pages/Devices.vue`（node info）
  - `frontend/src/pages/File.vue`（settings/download/offer/add-node 等多个 modal）
  - `frontend/src/pages/Flow.vue`（add-node）
  - `frontend/src/pages/Management.vue`（edit）
  - `frontend/src/pages/VarPool.vue`（多个 modal）
  - `frontend/src/layout/AppShell.vue`（incoming transfer modal；以及 profile menu click-catcher：透明全屏层，用于外部点击关闭）

---

## 3.1) 计划拆分（Checklist）

### T1. Workspace 准备（已完成）
- 目标：创建独占 worktree 与专业分支，确保不在 `repo/` 里做实现改动。
- 涉及：
  - `d:\project\MyFlowHub3\worktrees\win-overlay-mask\MyFlowHub-Win`
- 验收：
  - `git status -sb` 显示在 `fix/win-overlay-mask` 且工作区干净（或仅包含本 workflow 的变更）
- 回滚点：`git worktree remove` + `git worktree prune` + 删除分支（若未推送）。

### T2. 新增通用 Overlay/Modal 容器组件（Teleport 到 body）
- 目标：提供可复用的遮罩容器，确保永远以 viewport 为参照覆盖全屏（含 Header）。
- 涉及文件（预期）：
  - `frontend/src/components/ui/overlay/Overlay.vue`（新增）
  - `frontend/src/components/ui/overlay/index.ts`（新增导出）
- 设计要点：
  - 使用 `<Teleport to=\"body\">` 渲染遮罩节点，规避祖先 `transform` 影响。
  - 支持：点击遮罩空白处触发 `close`（等价于现有 `@click.self`）。
  - 支持：`Esc` 关闭（可配置；默认值由 D1/D2 决策决定）。
  - 多 Overlay 并存时：Esc 仅关闭“最上层 Overlay”（需要一个轻量的 stack/owner 机制，避免重复监听导致一次关闭多个）。
  - 提供 class 可配置（overlay 背景/间距/z-index），默认保持现有视觉（`bg-black/40 p-6`）。
- 验收：
  - 在任一页面打开 overlay 时，遮罩覆盖到窗口最顶端（包含 Header）。
  - 不引入常驻的全局事件监听：仅在 `open=true` 时绑定键盘监听，关闭即解绑。
- 测试点：
  - 快速打开/关闭多次，事件监听不会累积（无重复触发）。
- 回滚点：删除该组件与引用点，恢复页面内原写法。

### T3. 逐页迁移到 Overlay 组件（统一修复）
- 目标：将现有页面内 `fixed inset-0 ...` 遮罩统一替换为 `<Overlay>`，实现一致覆盖与行为。
- 涉及文件（预期）：
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/pages/File.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/Management.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/layout/AppShell.vue`（incoming transfer modal + profile menu click-catcher）
- 验收（逐页冒烟）：
  - 打开任一 modal 时：Header 被遮罩变暗，且背景不可点击。
  - 点击遮罩空白处：能关闭（与原行为一致）。
  - modal 内容区域点击：不触发关闭（与原行为一致）。
  - Profile 菜单打开时：背景不可点击；点击菜单外任意位置关闭；不出现“黑色遮罩”（保持透明拦截层）。
- 测试点：
  - `File` 页多 modal 并存时：分别打开/关闭互不干扰，z-index 不错乱。
- 回滚点：逐文件 revert（优先按页面拆分 commit，便于回滚）。

### T4. 验收与回归检查（含 DevTools）
- 目标：确保修复覆盖所有 modal 且无明显视觉/交互回归。
- 验收步骤：
  1) `Devices`：点击 Node 打开详情 → 确认 Header 被遮罩 → 点击遮罩关闭。
  2) `File`：依次打开 settings/download/offer/add-node → 检查覆盖与关闭。
  3) `Flow`：打开 add-node → 检查覆盖与关闭。
  4) `Management`：打开 edit → 检查覆盖与关闭。
  5) `VarPool`：打开各 modal → 检查覆盖与关闭。
- DevTools 检查点（可选）：
  - 遮罩 DOM 是否出现在 `document.body` 下（Teleport 生效）。
  - 遮罩的 `position: fixed; inset: 0` 的 containing block 是否为 viewport。
- 回滚点：若出现全局层级问题，优先回退到方案 B（仅取消页面动画 transform）作为兜底，但需重新走阶段 2/3.1 确认。

---

## 风险与注意事项
- z-index 冲突：现有 `z-40/z-50` 分散，Teleport 后应统一一个足够高的 z-index，避免被 Header/侧栏盖住。
- 行为一致性：部分页面可能依赖 `@click.self` 的细节（例如内部 stopPropagation）；迁移时需逐页验证。


