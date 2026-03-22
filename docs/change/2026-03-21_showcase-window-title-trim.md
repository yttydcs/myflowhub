# Showcase 窗口标题占位精简

## 变更背景 / 目标
- 背景：Showcase 已有独立 Editor 与独立 Viewer，但两个窗口头部仍保留 `Showcase Editor` 和 `Showcase Viewer` 静态标题，占用了首屏空间。
- 目标：进一步压缩窗口 chrome，只保留 screen 名称和必要状态信息，让界面更聚焦内容本身。

## 具体变更内容
### 修改
- `frontend/src/pages/Showcase.vue`
  - 移除编辑窗口头部的 `Showcase Editor` 标题。
  - 将 screen 标题 `h1` 的顶部 margin 一并收掉，避免标题删除后留下空白占位。
- `frontend/src/windows/ShowcaseWindow.vue`
  - 移除 viewer 窗口头部的 `Showcase Viewer` 标题。
  - 将 screen 标题 `h1` 的顶部 margin 一并收掉，确保 viewer 头部更紧凑。

### 未修改
- 未调整 screen name、连接状态、layout badge、时间信息。
- 未修改 store、路由、窗口打开逻辑、数据结构、保存逻辑和 widget 交互。

## 对应 plan.md 任务映射
- `T1` 明确本轮标题收口需求边界：完成。
- `T2` 完成架构设计与实现方案：完成。
- `T3` 实现窗口标题占位精简：完成。
- `T4` 进行 Code Review 与归档：完成。

## 关键设计决策与权衡
- 采用模板层直接删除，而不是 CSS 隐藏。
  - 原因：目标内容是纯静态标题，没有任何状态依赖。
  - 收益：DOM 结构更干净，不会留下无意义节点或隐藏样式耦合。
- 同步删除相邻 `mt-*` 间距，而不是只删文字。
  - 原因：用户目标是减少占位，不只是看不见标题。
  - 收益：头部高度真实收缩，不会出现“字没了但空白还在”的伪精简。

## 测试与验证方式 / 结果
- 文本搜索：
  - 命令：`rg -n "Showcase Editor|Showcase Viewer" frontend/src/pages/Showcase.vue frontend/src/windows/ShowcaseWindow.vue`
  - 结果：无命中，确认目标模板文案已移除。
- 依赖准备：
  - 命令：`npm ci`
  - 结果：成功。
- 前端构建：
  - 命令：`npm run build`
  - 结果：失败。
  - 失败原因：`src/pages/Home.vue` 无法解析 `../../wailsjs/go/session/SessionService`，属于现有环境 / 生成物缺失问题，与本次 Showcase 标题精简改动无关。

## 潜在影响与回滚方案
- 潜在影响：
  - Editor / Viewer 头部会比此前少一行静态模块标题。
  - 路由标题仍保留在 `frontend/src/router/index.ts`，仅影响窗口 / 文档标题，不影响页面占位。
- 回滚方案：
  - 回退 `frontend/src/pages/Showcase.vue`
  - 回退 `frontend/src/windows/ShowcaseWindow.vue`

## 子Agent执行轨迹
- 本轮未使用子Agent。
