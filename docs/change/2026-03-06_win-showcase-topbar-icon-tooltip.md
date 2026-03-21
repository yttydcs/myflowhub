# 变更归档：Showcase 顶部按钮图标化与 Tooltip

## 变更背景 / 目标
- 背景：`Showcase` 页面顶部操作区使用文字按钮，在空间利用与视觉一致性上与项目内其它工具栏（如 `File.vue`）存在差异。
- 目标：
  - 将顶部 6 个操作按钮改为图标按钮；
  - 为按钮提供 tooltip，文案沿用原按钮文字；
  - 保持原有功能行为与禁用逻辑不变。

## 具体变更内容

### 修改
- `frontend/src/pages/Showcase.vue`
  - 更新图标导入：新增 `RefreshCw`、`Plus`、`Pencil`、`Trash2`、`Rss`、`Database`。
  - 将顶部操作区 6 个按钮由文字按钮改为 `size="icon"` 图标按钮：
    - `Refresh Vars`
    - `New Screen`
    - `Rename Screen`
    - `Delete Screen`
    - `Add Event`
    - `Add Var`
  - 每个按钮增加：
    - `title`（作为 tooltip，文案与原按钮一致）；
    - `sr-only` 文本（可访问性文本）。

### 新增
- `docs/change/2026-03-06_win-showcase-topbar-icon-tooltip.md`（本文件）。

### 删除
- 无。

## 对应 plan.md / todo.md 任务映射
- `SHC-TOPBAR-1` 图标化顶部按钮并保留原行为
  - 对应改动：`Showcase.vue` 顶部按钮模板改造为 icon button。
- `SHC-TOPBAR-2` 增加 tooltip 与可访问性文本
  - 对应改动：为每个按钮补充 `title` 与 `sr-only`。
- `SHC-TOPBAR-3` 回归验证与归档准备
  - 对应改动：执行构建验证命令并记录结果；完成本归档文档。

## 关键设计决策与权衡
- 选择 `title` 原生 tooltip，而非新引入 Tooltip 组件。
  - 原因：需求仅要求显示现有按钮文字；`title` 成本低、变更面小、可维护性高。
  - 权衡：视觉样式不可深度定制，但满足当前需求且避免新增依赖。
- 沿用现有 `File.vue` 的图标按钮模式（`size="icon"` + `sr-only`）。
  - 原因：统一 UI 交互模式，降低学习成本。

### 性能要点
- 本次仅替换按钮展示层，不增加网络/存储/循环计算。
- 无新增 I/O、无状态订阅变更，不引入性能回归点。

### 可扩展性要点
- 按钮事件处理函数未改动，未来可继续按相同模式新增图标按钮。
- Tooltip 文案与按钮语义一一对应，后续国际化可集中替换文案常量。

## 测试与验证方式 / 结果
- 代码差异审查：`git diff -- frontend/src/pages/Showcase.vue`（通过）。
- 构建验证：`npm --prefix frontend run build`（未通过，环境限制）。
  - 失败信息：`'vite' is not recognized as an internal or external command`。
  - 结论：当前执行环境缺少前端依赖/可执行工具，无法完成构建级验证。
- 手工验证：未在本次自动化环境执行 UI 交互冒烟（需本地 `wails dev`/前端依赖完整环境）。

## 潜在影响与回滚方案
- 潜在影响：
  - UI 从文字改为图标后，首次使用者依赖 tooltip 理解功能；
  - 仅影响 `Showcase` 顶部操作区显示，不影响按钮业务逻辑。
- 回滚方案：
  - 回滚 `frontend/src/pages/Showcase.vue` 中顶部按钮区块为原文字按钮实现；
  - 保留或删除本归档文档按发布策略处理。
