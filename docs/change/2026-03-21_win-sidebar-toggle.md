# 2026-03-21 Win 左侧边栏收起改造

## 变更背景 / 目标

- 背景：Win 主界面左侧边栏顶部存在单独的 `Session` 卡片，占用垂直空间，且缺少桌面端快速收起能力。
- 目标：
  - 移除左侧边栏顶部 `Session` 卡片。
  - 在顶部新增菜单 icon，实现侧边栏展开 / 收起。
  - 收起态保留图标导航，不做状态持久化。

## 具体变更内容

### 修改

- `frontend/src/layout/AppShell.vue`
  - 删除左侧顶部 `Session` 卡片及其相关显示计算属性。
  - 新增 `sidebarCollapsed` 本地状态与 `toggleSidebar()`。
  - 新增 `sidebarGridClass`、`sidebarAsideClass`、`sidebarHeaderClass`、`sidebarBrandClass`、`sidebarBrandMarkClass`、`sidebarNavClass`、`sidebarGroupClass`、`sidebarNavListClass`、`sidebarNavItemBaseClass`、`sidebarNavIconClass` 等计算属性，统一控制桌面侧边栏展开 / 收起样式。
  - 顶部新增菜单 icon 按钮，并显式使用原生 `button` 渲染，补充 `aria-label`、`title` 与 `sr-only` 文本。
  - 收起态隐藏品牌文案、导航分组标题与导航说明，仅保留图标和无障碍文本。

## 对应 plan.md 任务映射

- `T1` 调整左侧边栏结构与收起交互
  - 对应代码改动：`frontend/src/layout/AppShell.vue`
- `T2` 执行构建与交互验证
  - 对应验证：前端构建、浏览器交互核验
- `T3` 完成 Code Review 与变更归档
  - 对应产物：本变更文档、`plan.md` 执行结果与审查结论

## 关键设计决策与权衡

- 局部状态优先
  - 侧边栏收起仅影响 `AppShell.vue` 的桌面视图层，不需要提升到 store，避免引入额外状态同步复杂度。
- 计算属性集中样式
  - 通过计算属性输出关键 class，避免模板散落大量三元表达式，提高可读性与后续扩展性。
- 性能优先
  - 只引入单个局部 `ref` 和纯计算属性，不增加持久化、网络请求、额外 watcher 或重复计算。
- 可访问性补强
  - 收起态隐藏可视文案后，保留 `title` 和 `sr-only`，并让新菜单控件使用原生 `button`，避免纯图标交互不可达。

## 测试与验证方式 / 结果

- 构建验证
  - 命令：`npm run build`
  - 结果：通过
  - 备注：存在既有 chunk size warning，但构建成功。
- 交互验证
  - 方式：Chrome DevTools MCP 打开 `http://127.0.0.1:4173/#/home` 与 `#/presets`
  - 结果：
    - 展开态存在顶部菜单按钮。
    - 点击后侧栏收起为窄栏，仅保留图标。
    - 收起态下点击图标导航仍可跳转页面。
    - 再次点击后侧栏可恢复展开态。
- 验证环境说明
  - 当前 worktree 未自带 `frontend/node_modules` 与 `frontend/wailsjs`，验证时通过 junction 复用主仓对应目录。
  - 纯 Vite 环境缺少 Wails runtime，浏览器验证时注入最小 runtime/go stub 以渲染壳层 UI；该操作未修改源码。

## 潜在影响与回滚方案

- 潜在影响
  - 桌面端侧栏宽度从固定值改为可切换，若后续新增更复杂的品牌头部内容，需要同步复核收起态排版。
  - 当前移动端顶部导航未改动，未来若要求移动端统一交互，需要单独设计。
- 回滚方案
  - 回退 `frontend/src/layout/AppShell.vue` 到本次变更前版本即可恢复原始左侧边栏结构。

## 子Agent执行轨迹

- 本次 workflow 未使用子Agent。
- 原因：实现写集仅涉及 `frontend/src/layout/AppShell.vue`，布局状态、样式和交互强耦合，无法安全拆分为并行可独立验收任务。
