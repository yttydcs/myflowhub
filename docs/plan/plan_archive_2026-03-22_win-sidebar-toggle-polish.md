# Win 左侧边栏二次收敛计划

## 项目目标与当前状态

- 目标：继续优化 Win 主界面侧边栏与顶部状态栏的关系，修复侧栏滚动引起的视觉不齐，收敛收起态选中样式，并补齐收起态 tooltip 交互。
- 当前状态：已完成需求分析、架构设计、计划拆分、代码编写、Code Review、变更归档，并在用户确认后完成 workflow 收敛。

## Workflow 信息

- 当前仓库：MyFlowHub-Win
- 当前分支：refactor/win-sidebar-toggle-polish
- Base 分支：main
- Worktree 绝对路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle-polish
- 当前阶段：已结束（代码已合并至 `repo/MyFlowHub-Win` 的 `main`，归档已迁移到根控制面）

## 1. 需求分析

### 目标

- 将侧栏收起按钮从侧边栏内部移到顶部连接状态区左侧。
- 消除侧栏滚动条导致的品牌区与导航内容宽度不一致问题。
- 优化收起态选中视觉，使其只通过颜色变化表达当前项。
- 在收起态为图标导航补充 tooltip，显示菜单标题。

### 范围

- 必须：
  - 菜单按钮放到顶部 `Connected to xxxxx` 状态胶囊左侧。
  - 侧栏滚动布局调整后，品牌 logo 与导航区宽度保持一致，不再因滚动条出现错位。
  - 收起态选中项移除外层边框，仅保留颜色变化。
  - 收起态 hover 到图标时显示 tooltip，内容为菜单标题。
- 可选：
  - 根据现有布局对顶部按钮和状态胶囊的间距做轻量微调。
  - 对侧栏滚动容器做稳定 gutter 或滚动区域重构，只要结果满足对齐目标。
- 不做：
  - 不引入侧栏状态持久化。
  - 不改移动端横向导航行为。
  - 不调整菜单信息架构、路由或业务逻辑。

### 使用场景

- 桌面端用户在顶部状态区直接点击菜单按钮展开/收起侧栏。
- 收起态下用户通过图标快速导航，并通过 hover tooltip 识别菜单名称。
- 侧栏条目较多出现滚动时，顶部品牌区与导航列宽仍保持一致。

### 功能需求

- 顶部 header 左侧区域包含菜单按钮与连接状态胶囊，菜单按钮位于连接状态胶囊左边。
- 左侧栏内部不再显示菜单按钮。
- 侧栏可展开/收起，保持当前已有逻辑。
- 收起态下菜单项无外层选中边框，但当前项仍有明确颜色状态。
- 收起态下每个图标 hover 时显示对应 `label` tooltip。

### 非功能需求

- 性能：不增加网络请求、持久化或复杂 watcher；tooltip 与收起逻辑仅使用前端局部状态。
- 可读性：顶部按钮位置调整、侧栏滚动容器和选中样式逻辑应集中在 `AppShell.vue`，避免模板条件分支失控。
- 可扩展性：tooltip 和选中样式应复用现有组件和 class 体系，后续可扩展到更多侧栏模式。
- 稳定性：不影响顶部 profile 菜单、主内容滚动、现有桌面路由切换和移动端导航。
- 可维护性：变更范围尽量收敛到单文件布局组件。

### 输入输出

- 输入：
  - 用户点击顶部菜单按钮。
  - 用户在收起态 hover 某个图标菜单。
- 输出：
  - 侧栏展开/收起切换。
  - 收起态显示 tooltip。
  - 滚动情况下侧栏头部和导航列宽一致。

### 边界异常

- 顶部菜单按钮在移动端侧栏不存在时不应形成无意义交互。
- tooltip 只应在收起态启用，展开态不应重复显示。
- 侧栏滚动修复后，不能引入新的横向溢出或裁剪。
- 收起态切换路由时，选中态仍应可辨识。

### 验收标准

- 顶部连接状态胶囊左侧出现菜单按钮。
- 左侧栏内部不再出现菜单按钮。
- 侧栏出现滚动时，logo/品牌区与导航内容左右边界对齐，不再因为滚动条导致宽度不一致。
- 收起态选中菜单项没有外层边框，但仍有颜色变化。
- 收起态 hover 图标能出现 tooltip，内容为菜单标题。
- 前端构建通过。

### 风险

- 调整侧栏滚动容器时若处理不当，可能影响品牌区是否固定、滚动行为或内容裁剪。
- tooltip 若包裹 `RouterLink` 结构不当，可能影响点击命中或 active 样式。

### 问题清单

- 阻塞：否

## 2. 架构设计（分析）

### 总体方案

- 方案：继续在 `frontend/src/layout/AppShell.vue` 内完成本轮收敛，使用现有 `sidebarCollapsed` 状态；将 toggle 控件移动到 header 左侧状态区，并把侧栏滚动责任从 `nav` 内部滚动改成侧栏整体滚动容器，使品牌区和导航共享同一滚动宽度；收起态菜单项使用分支化 active class 和 `Tooltip` 组件包装。
- 选型理由：
  - 当前所有相关状态和布局都集中在 `AppShell.vue`，无需再拆到 store 或新组件。
  - 项目已存在可复用 `Tooltip` 组件，直接接入比原生 `title` 更符合交互要求。
  - 通过让同一侧栏容器承担滚动，可直接消除顶部品牌区和导航区因滚动条宽度不同而产生的错位。
- 备选对比：
  - 备选 1：继续让 `nav` 独立滚动，只给 `nav` 加额外右边距或 `scrollbar-gutter`。
    - 缺点：头部和导航仍分属不同宽度上下文，对齐不稳，且平台差异较大。
  - 备选 2：继续使用原生 `title` 显示提示。
    - 缺点：交互一致性和视觉质量都低于现有 `Tooltip` 组件。

### 模块职责

- `AppShell.vue`
  - 调整 header 左侧按钮与状态胶囊布局。
  - 负责侧栏整体滚动容器和收起态 active 样式。
  - 在收起态接入 tooltip。
- `Tooltip` 组件
  - 提供收起态图标 hover 的提示能力，不改组件实现，只复用。

### 数据 / 调用流

- 用户点击 header 左侧菜单按钮。
- `toggleSidebar()` 翻转 `sidebarCollapsed`。
- 计算属性更新：
  - header 左侧按钮显隐与间距
  - 侧栏滚动容器样式
  - 收起态 active class
  - 收起态 tooltip 是否启用
- 用户 hover 收起态图标时，`Tooltip` 渲染菜单标题。

### 接口草案

- 复用状态：
  - `sidebarCollapsed`
  - `toggleSidebar()`
- 新增 / 调整计算属性：
  - `sidebarAsideClass`
  - `sidebarNavClass`
  - `sidebarNavItemActiveClass`
  - `sidebarNavItemInactiveClass`
  - `showCollapsedNavTooltip`

### 错误与安全

- 不新增远程调用、用户持久化输入或敏感数据处理。
- 顶部菜单按钮保留 `aria-label` / `sr-only`。
- tooltip 仅作视图增强，不参与业务状态写入。

### 性能与测试策略

- 性能关键点：
  - 继续复用局部 `ref` 和纯计算属性，不引入额外 store 更新。
  - Tooltip 只在收起态分支渲染，避免展开态无意义包装。
  - 滚动容器调整不增加额外 DOM 深度以外的计算成本。
- 测试策略：
  - 执行 `npm run build` 验证模板和依赖引用。
  - 浏览器中手动验证 header 按钮位置、侧栏滚动对齐、收起态 tooltip 和选中态视觉。

### 可扩展性设计点

- 收起态 active / inactive class 独立拆分后，未来可支持更多视觉模式而不污染模板。
- Tooltip 接入点可扩展为未来的快捷键、权限状态等图标提示。

### 问题清单

- 阻塞：否

## 3.1 计划拆分

### 可执行任务清单

- [x] T1 调整顶部菜单按钮、侧栏滚动与收起态样式
- [x] T2 执行构建与交互验证
- [x] T3 完成 Code Review 与变更归档

### 任务详情

#### T1

- Task ID：T1
- 标题：调整顶部菜单按钮、侧栏滚动与收起态样式
- Owner：主Agent
- Worktree 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle-polish
- Plan 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle-polish/plan.md
- 目标：
  - 菜单按钮迁移到顶部状态区左侧。
  - 修复侧栏滚动造成的列宽不齐。
  - 收起态 active 去外边框。
  - 收起态图标接入 tooltip。
- 涉及模块 / 文件：
  - `frontend/src/layout/AppShell.vue`
- Write set：
  - `frontend/src/layout/AppShell.vue`
- 验收条件：
  - 满足四项用户要求。
  - 不影响 header 连接状态、profile 菜单与路由切换。
- 测试点：
  - 顶部按钮位置。
  - 侧栏滚动对齐。
  - 收起态选中样式。
  - 收起态 tooltip。
- 回滚点：
  - 回退 `frontend/src/layout/AppShell.vue` 到本分支初始版本。
- 依赖关系：
  - 无前置依赖，可直接执行。
- 风险与注意事项：
  - 注意 tooltip 包裹后不能破坏 RouterLink 的可点击区域。
  - 注意 header 按钮不要在移动端产生无意义视觉噪声。
- 关键上下文引用：
  - `frontend/src/layout/AppShell.vue`
  - `frontend/src/components/ui/tooltip/Tooltip.vue`

#### T2

- Task ID：T2
- 标题：执行构建与交互验证
- Owner：主Agent
- Worktree 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle-polish
- Plan 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle-polish/plan.md
- 目标：
  - 验证模板构建与关键交互行为。
- 涉及模块 / 文件：
  - `frontend/package.json`
- Write set：
  - 无源码写入；允许临时运行验证辅助文件或进程
- 验收条件：
  - `npm run build` 成功。
  - 浏览器验证记录完整。
- 测试点：
  - 顶部按钮位置和收起展开。
  - 收起态 tooltip。
  - 收起态 active 无外层边框。
  - 滚动侧栏视觉对齐。
- 回滚点：
  - 删除临时验证产物，不影响源码。
- 依赖关系：
  - 依赖 T1 完成。
- 风险与注意事项：
  - 若 worktree 缺少 `node_modules` / `wailsjs`，按现有做法只读复用主仓目录辅助验证。

#### T3

- Task ID：T3
- 标题：完成 Code Review 与变更归档
- Owner：主Agent
- Worktree 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle-polish
- Plan 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle-polish/plan.md
- 目标：
  - 完成 3.3 审查并生成 `docs/change` 归档。
- 涉及模块 / 文件：
  - `plan.md`
  - `docs/change/2026-03-22_win-sidebar-toggle-polish.md`
- Write set：
  - `plan.md`
  - `docs/change/`
- 验收条件：
  - 审查结论完整。
  - 文档与实际改动一致。
- 测试点：
  - 文档任务映射、验证记录和回滚说明完整。
- 回滚点：
  - 删除本次新增文档，不影响实现代码。
- 依赖关系：
  - 依赖 T1、T2 完成。

## 并行性评估

- 结论：本 workflow 不派发子Agent。
- 原因：
  - 写集仅涉及 `frontend/src/layout/AppShell.vue`，布局、滚动、tooltip 和选中样式强耦合。
  - 验证与归档依赖单一最终实现结果，缺少可安全并行的独立 write set。

## 3.2 执行结果

### T1 执行结果

- 结果：完成
- 实际变更：
  - 菜单按钮从侧栏内部迁移到顶部连接状态胶囊左侧，且仅在桌面端显示。
  - 侧栏滚动改由 `aside` 整体承载，并增加稳定 gutter，消除品牌区与导航区宽度错位。
  - 收起态 active 样式去除外层边框，保留图标与文本颜色变化。
  - 收起态图标通过现有 `Tooltip` 组件显示菜单标题。
- 偏差说明：无计划外改动。

### T2 执行结果

- 结果：完成
- 构建验证：
  - `npm run build` 通过。
- 浏览器验证：
  - 使用 Chrome DevTools MCP 打开 `http://127.0.0.1:4174/#/home` 与 `#/presets`。
  - 验证菜单按钮位于顶部连接状态左侧。
  - 验证收起态图标 hover 可出现 tooltip。
  - 验证收起态点击图标后仍可切换到 `Presets` 页面。
  - 验证收起态当前项无外层选中边框，仅通过颜色变化识别。
  - 验证展开态侧栏滚动时品牌区与导航区处于同一宽度上下文。
- 环境说明：
  - 当前 worktree 通过 junction 只读复用主仓 `frontend/node_modules` 与 `frontend/wailsjs`。
  - 浏览器验证时注入最小 Wails runtime/go stub，仅用于渲染壳层 UI；未修改源码。

## 3.3 Code Review

- 需求覆盖：通过
  - 四项用户要求均已覆盖。
- 架构合理性：通过
  - 变更继续收敛在 `AppShell.vue`，未把纯布局状态提升到全局 store。
- 性能风险：通过
  - 无新增 I/O、持久化、额外 watcher 或重复计算。
- 可读性与一致性：通过
  - 顶部按钮、滚动策略、收起态 active class 和 tooltip 分支职责清晰。
- 可扩展性与配置化：通过
  - 收起态 active / inactive 逻辑和 tooltip 接入点后续可继续扩展。
- 稳定性与安全：通过
  - 顶部 profile 菜单、路由跳转和桌面收起/展开验证正常。
  - 菜单按钮保留 `aria-label` 与 `sr-only`。
- 测试覆盖情况：通过
  - 已完成前端构建与关键浏览器交互验证。
- 子Agent治理与审计：通过
  - 本 workflow 未使用子Agent，原因是写集单一且布局改动强耦合。

### Review 结论

- 结论：通过
- 阻塞：否
- 下一阶段：4 已完成，等待并已完成 workflow 收敛。

## 当前结论

- 计划文档已确认。
- 阻塞：否
- workflow 已结束，代码与归档已收敛到根控制面。
