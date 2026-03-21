# 变更说明：统一修复 Modal 遮罩未覆盖 Header

## 变更背景 / 目标
在 `Devices` 页点击节点弹出详情后，背景遮罩未覆盖顶部 Header（Connected / Active Profile），导致顶部仍可见且可点击。你要求：
1) 遮罩必须覆盖整个窗口（包含 Header），并阻止背景交互；
2) 统一修复项目内同类 Modal 遮罩实现（避免每页复制粘贴、重复踩坑）；
3) 支持 **Esc 关闭**（Modal + Profile 菜单），且多层 Overlay 并存时只关闭最上层。

## 具体变更内容（新增 / 修改 / 删除）
### 新增
- `frontend/src/components/ui/overlay/Overlay.vue`
  - 统一的全屏 Overlay 容器，默认使用 `Teleport to="body"`，规避祖先 `transform` 影响 `position: fixed` 的问题。
  - 支持：`closeOnBackdrop`（仅当点击遮罩空白处关闭）、`closeOnEsc`（Esc 关闭）、`teleport`（必要时可在局部禁用 Teleport）。
- `frontend/src/components/ui/overlay/index.ts`：导出 `Overlay` 组件。
- `frontend/src/lib/overlayStack.ts`
  - 维护 Overlay 栈并提供 **全局单一 keydown 监听**，保证 Esc 仅关闭最上层 Overlay。

### 修改（迁移现有弹窗遮罩）
- `frontend/src/pages/Devices.vue`
- `frontend/src/pages/File.vue`
- `frontend/src/pages/Flow.vue`
- `frontend/src/pages/Management.vue`
- `frontend/src/pages/VarPool.vue`
- `frontend/src/layout/AppShell.vue`
  - incoming transfer modal 迁移至 `Overlay`
  - Profile 菜单的透明 click-catcher 迁移至 `Overlay`（`teleport=false` + `bg-transparent`），并支持 Esc 关闭

> 说明：保持现有 “点击遮罩关闭” 行为不变——仅原本使用 `@click.self` 的弹窗（Devices/VarPool）开启 `closeOnBackdrop`；其他弹窗保持只通过按钮/逻辑关闭，但新增支持 Esc 关闭（按你确认的决策执行）。

## 对应 plan.md 任务映射
- `plan.md`
  - T2：新增通用 Overlay/Modal 容器组件（Teleport 到 body）
  - T3：逐页迁移到 Overlay 组件（统一修复）
  - T4：验收与回归检查（含 DevTools）

## 关键设计决策与权衡（性能 / 扩展性）
1) **Teleport 到 body（默认）**
   - 解决根因：祖先 `transform` 会改变 `fixed` 参照系，导致遮罩无法覆盖 Header。
   - 代价：DOM 位置变化；需通过 props 保持样式/交互可配置。
2) **Esc 关闭（默认开启）+ “只关闭最上层”**
   - 使用 Overlay 栈 + 单一 keydown 监听，避免多弹窗并存时 Esc 误关闭多个。
3) z-index 选择
   - Overlay 默认 `z-50`，ToastHost 为 `z-[100]`：确保弹窗开启时仍能看到 toast（错误/提示不被遮罩盖住）。
4) 可扩展性
   - 新增弹窗不再复制 `fixed inset-0 ...`，直接复用 `Overlay`，可统一收敛关闭策略与层级策略。

## 测试与验证方式 / 结果
已执行：
- `frontend/`：`npm run build`
- `MyFlowHub-Win`：`$env:GOWORK='off'; wails build -nopackage`（用于生成 bindings 并完整验证编译链路）

建议冒烟（人工）：
1) `Devices`：点击 Node 打开详情 → 确认 Header 被遮罩 → Esc 关闭 / 点击遮罩关闭
2) `File`：打开 settings/download/offer/add-node/preview → Esc 关闭（点击遮罩不关闭）
3) `Flow/Management/VarPool`：打开相关弹窗 → 确认遮罩覆盖 Header 且 Esc 行为正确
4) Header Profile 菜单：打开菜单 → 点击外部关闭 + Esc 关闭（透明 click-catcher 不应出现黑遮罩）

## 潜在影响与回滚方案
### 潜在影响
- 新增了 Esc 关闭交互：可能导致少数用户在输入时误触 Esc 关闭（但符合常见 Modal/Popover 预期，且为你明确确认的需求）。
- Teleport 改变 DOM 层级：理论上可能影响极少数依赖 DOM 层级的样式/定位（已通过 build 验证，且组件提供 `teleport` 可回退）。

### 回滚方案
- 直接 revert 本次变更（改动集中在少量页面与新增组件），或逐页回滚迁移点：
  - 先回退 `Overlay` 引用，再删除 `overlay/` 与 `overlayStack.ts`。

