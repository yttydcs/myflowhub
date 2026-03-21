# 2026-03-03 Showcase 面板简化（仅 title + value）+ Designer 右键菜单

## 背景 / 目标
- Showcase 的 widget 卡片目前展示了较多“元信息”（Target/Span、Var(owner/name)、payload 预览、slider 参数文案等），不利于作为“展示界面/控制面板”使用。
- 目标：在 **Designer（Showcase 页）** 与 **Viewer（ShowcaseWindow）** 中，统一将卡片简化为 **title + value（控制组件）**；并将 Designer 的 `Edit/Remove` 迁移到 **右键菜单**，保留拖拽手柄。

## 范围
- 仅调整 UI 展示与交互入口，不改任何协议/wire，不改 `ShowcaseConfig` 数据结构，不新增后端 API。
- `topic_button`：卡片仅保留 `Send` 按钮（不显示 payload）。
- `slider`：允许保留当前值的小 Badge；slider 的发送/节流逻辑保持不变。

## 具体变更
### Designer（Showcase 页）
- `frontend/src/pages/Showcase.vue`
  - widget 卡片仅保留：拖拽手柄 + `safeTitle(widget)` + value 区域（Send / Display / Switch / Slider）。
  - 移除卡片右上角 `Edit/Remove` 按钮，改为右键菜单：
    - `@contextmenu.prevent` 打开菜单；
    - 菜单关闭策略：点击空白处、`Esc`、scroll、resize；
    - 菜单位置：`nextTick()` 后测量并钳制到 viewport 内。

### Viewer（Showcase Window）
- `frontend/src/windows/ShowcaseWindow.vue`
  - widget 卡片仅保留 `safeTitle(widget)` + value 区域；
  - `topic_button` 不再显示 payload 预览，仅保留 `Send`；
  - `switch/slider` 移除元信息文案（ON/OFF 值、min/max/step/throttle 等），保留控制组件。

## 任务映射（plan.md）
- SC1：Designer 右键菜单（Edit/Remove）
- SC2：Designer 卡片简化（仅 title + value）
- SC3：Viewer 卡片简化（仅 title + value）
- SC4：本地验证（构建/冒烟）

## 关键设计决策与权衡
- ContextMenu 采用页面内轻量实现（不引第三方库），降低依赖与维护成本；后续如多页面复用再抽组件。
- 右键菜单为“设计操作入口”，因此在 Designer 中通过 `@contextmenu.prevent` 接管，避免系统菜单干扰。
- slider 仍采用“拖动实时发送 + 可配置节流 + 松手确认写入”的既有策略，避免引入新的交互语义。

## 测试与验证
### 构建/单测
- `GOWORK=off wails build -debug -skipembedcreate -nopackage`（✅ 通过）
  - 说明：该 worktree 不在上层 `go.work` 的 `use` 列表中，需 `GOWORK=off` 关闭 workspace 以避免构建报错。
- `GOWORK=off go test ./...`（✅ 通过）

### 手工冒烟建议
1) `wails dev` 打开 `#/showcase`：
   - 新增 widget（Event/Var）；
   - 右键卡片 → `Edit/Remove`；
   - 拖拽手柄排序正常。
2) `Open Window` 打开 `#/showcase-window?screenId=...`：
   - 卡片展示为 title + value；
   - Send / Switch / Slider 可用，变量值更新正常。

## 潜在影响
- Designer 中右键用于“设计操作”；若需要系统右键菜单（复制等）需在后续版本评估是否提供替代入口。
- 元信息被隐藏后，Target/Span/payload/owner 等需要通过 `Edit` 弹窗查看与修改。

## 回滚方案
- 回滚 `frontend/src/pages/Showcase.vue` 与 `frontend/src/windows/ShowcaseWindow.vue` 的相关改动即可（纯 UI 改动，回滚成本低）。

