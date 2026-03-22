# 2026-03-22 Win 左侧边栏二次收敛

## 变更背景 / 目标

- 背景：上一轮 Win 左侧边栏收起改造已经支持收起，但菜单按钮位置、侧栏滚动宽度、收起态选中样式和菜单提示交互仍不满足最新要求。
- 目标：
  - 将菜单按钮移到顶部连接状态左侧。
  - 修复侧栏滚动条引起的品牌区与导航区宽度不一致。
  - 收起态取消外层选中边框，仅保留颜色变化。
  - 收起态 hover 图标时显示菜单标题 tooltip。

## 具体变更内容

### 修改

- `frontend/src/layout/AppShell.vue`
  - 引入并复用 `Tooltip` 组件。
  - 将侧栏 toggle 按钮从侧栏头部迁移到顶部状态胶囊左侧，且仅在 `lg` 桌面布局显示。
  - 将侧栏滚动责任由 `nav` 内部滚动调整为 `aside` 整体滚动，并增加 `scrollbarGutter: "stable"`，使品牌区与导航区共享同一滚动宽度上下文。
  - 新增 `collapsedNavItemStyle`、`navItemStateClass()`、`navItemIconToneClass()`，拆分收起态与展开态 active / inactive 样式。
  - 收起态菜单项取消外层边框高亮，仅保留图标和文本颜色变化。
  - 收起态菜单项通过 `Tooltip` 在右侧显示菜单标题。

## 对应 plan.md 任务映射

- `T1` 调整顶部菜单按钮、侧栏滚动与收起态样式
  - 对应代码改动：`frontend/src/layout/AppShell.vue`
- `T2` 执行构建与交互验证
  - 对应验证：前端构建、Chrome DevTools 桌面交互核验
- `T3` 完成 Code Review 与变更归档
  - 对应产物：本归档文档与 `plan.md` 执行记录

## 关键设计决策与权衡

- 菜单按钮迁移到 header
  - 直接贴近连接状态区，减少用户在收起/展开时视线跨栏移动。
- 侧栏整体滚动
  - 选择让 `aside` 承担滚动并使用稳定 gutter，而不是只调整 `nav` 内部滚动补偿。
  - 优点：品牌区和导航区共享同一宽度上下文，对齐更稳。
- 收起态 tooltip 组件化
  - 复用现有 `Tooltip` 组件，而不是继续依赖原生 `title`。
  - 优点：交互一致、提示位置明确、后续可统一演进。
- 收起态 active 视觉收敛
  - 外层容器不再高亮，只通过图标和文字颜色表达当前项，避免窄栏状态下视觉过重。

## 测试与验证方式 / 结果

- 构建验证
  - 命令：`npm run build`
  - 结果：通过
  - 备注：仍存在既有 chunk size warning，但不影响构建成功。
- 交互验证
  - 方式：Chrome DevTools MCP 打开 `http://127.0.0.1:4174/#/home` 与 `#/presets`
  - 结果：
    - 顶部菜单按钮位于连接状态胶囊左侧。
    - 点击按钮后可正常展开 / 收起侧栏。
    - 收起态 hover 图标出现 tooltip，例如 `Presets`。
    - 收起态点击图标仍可跳转页面。
    - 收起态当前项无外层选中边框，仅保留颜色变化。
    - 展开态截图可见侧栏滚动情况下品牌区与导航区宽度一致。
- 验证环境说明
  - 当前 worktree 通过 junction 只读复用主仓 `frontend/node_modules` 与 `frontend/wailsjs`。
  - 浏览器验证时注入最小 Wails runtime/go stub 渲染壳层 UI；该操作未修改源码。

## 潜在影响与回滚方案

- 潜在影响
  - 侧栏整体滚动后，若未来需要品牌区固定不滚动，需要再单独设计 sticky 方案。
  - 收起态 tooltip 依赖 `Tooltip` 组件，若后续全局 tooltip 样式调整，会同步影响这里。
- 回滚方案
  - 回退 `frontend/src/layout/AppShell.vue` 到本次变更前版本即可恢复上一轮侧栏实现。

## 子Agent执行轨迹

- 本次 workflow 未使用子Agent。
- 原因：实现写集仅涉及 `frontend/src/layout/AppShell.vue`，顶部布局、侧栏滚动、收起态样式和 tooltip 逻辑强耦合，无法安全拆分为并行任务。
