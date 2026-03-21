# 2026-03-03 Showcase：Widget 卡片 Key-Value 单行布局

## 变更背景 / 目标
- 需求：Showcase 的 widget 卡片希望呈现为严格 Key-Value（key=title，value=展示/控制组件），尽量单行不换行；同时保持现有控制/订阅逻辑不变。
- 目标：Designer（`#/showcase`）与 Viewer Window（`#/showcase-window`）同步调整布局。

## 具体变更
- Designer：`frontend/src/pages/Showcase.vue`
  - 卡片内容改为两列 grid：左侧为拖拽手柄 + title（`truncate` + `title` tooltip），右侧为 value 区（右对齐）。
  - `display`：由多行 `<pre>` 改为单行截断文本，并以 `title` 属性展示完整值。
  - `switch`：value 区仅显示 checkbox（不显示 ON/OFF 文本）。
  - `slider`：value 区使用 `flex-wrap` + `min-width`，允许在极窄宽度下退化为两行（badge 与 slider 分行），保证可用性。
- Viewer：`frontend/src/windows/ShowcaseWindow.vue`
  - 与 Designer 保持一致的 Key-Value Row 渲染策略（无拖拽手柄）。

## 对应计划任务（plan.md）
- KV1：Designer Key-Value Row
- KV2：Viewer Key-Value Row
- KV3：本地验证
- KV4：归档变更

## 关键设计决策与权衡
- 仅调整 UI DOM 与样式类，不引入新协议/配置字段，避免影响 TopicBus/VarPool 数据流与既有行为。
- 不用 breakpoint 强行切换 1/2 列：采用 `flex-wrap + min-width` 让 slider 在“极窄”时自然退化，避免出现不可用的超窄滑条。
- 未抽象通用组件：目前仅两处使用，先以内联模板实现；后续如复用面扩大再抽 `KeyValueRow` 组件。

## 测试与验证
- 构建：
  - `GOWORK=off wails build -debug -skipembedcreate -nopackage`
- 单测：
  - `GOWORK=off go test ./... -count=1`
- 手工冒烟：
  - `wails dev` 打开 `#/showcase`，新增/编辑/右键菜单/拖拽；检查 display tooltip / switch / slider 窄宽退化。
  - `Open Window` 打开 `#/showcase-window?screenId=...`，检查布局与控制/实时更新。

## 潜在影响
- 仅影响 Showcase 的前端展示层（Designer/Viewer），不影响协议、后端服务、存储结构。

## 回滚方案
- revert 本次提交；或回滚以下文件：
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`

