# Win 左侧边栏收敛调整计划

## 项目目标与当前状态

- 目标：调整 Windows 端左侧边栏顶部结构，移除顶部 `Session` 卡片，并在侧边栏顶部增加菜单 icon，使左侧边栏可收起为窄栏，仅保留导航图标。
- 当前状态：已完成需求分析、架构设计、计划拆分、代码编写、Code Review 与变更归档，待用户确认是否结束本次 workflow。

## Workflow 信息

- 当前仓库：MyFlowHub-Win
- 当前分支：refactor/win-sidebar-toggle
- Base 分支：main
- Worktree 绝对路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle
- 当前阶段：4 归档变更（已完成）

## 1. 需求分析

### 目标

- 收敛 Win 主界面左侧边栏顶部信息层级，减少重复状态展示。
- 提供侧边栏收起能力，提升主内容区横向空间。

### 范围

- 必须：
  - 移除左侧边栏顶部 `Session` 卡片。
  - 在左侧边栏顶部增加菜单 icon。
  - 点击菜单 icon 后将左侧边栏收起为窄栏，仅保留图标导航。
  - 收起状态仅在当前运行期内生效，不做持久化。
- 可选：
  - 为收起态补充必要的可访问性文本和悬浮提示。
  - 为收起态同步优化品牌区与分组标题的视觉占位。
- 不做：
  - 不修改移动端顶部导航条。
  - 不新增本地存储、配置文件或后端接口。
  - 不调整页面路由结构和业务数据流。

### 使用场景

- 桌面端用户需要更大的主内容区域时，点击侧边栏顶部菜单 icon，将侧边栏收起为窄栏。
- 用户再次点击菜单 icon，可恢复完整侧边栏，查看分组标题、导航标签与描述。

### 功能需求

- 桌面端 `lg` 布局下提供侧边栏展开/收起切换。
- 展开态保持现有导航能力。
- 收起态保留所有导航入口的可点击性。
- 收起态隐藏文字标签，仅保留图标，并提供识别当前模块的必要提示。
- 顶部原 `Session` 卡片删除后，不影响现有头部连接状态展示。

### 非功能需求

- 性能：只引入本地 UI 状态切换，不增加额外 I/O、持久化读写或重复计算。
- 可读性：侧边栏展开/收起相关 class 与计算属性集中管理，避免模板内散落条件表达式。
- 可扩展性：后续若需要加入持久化或更多侧栏模式，应可在现有状态模型上扩展。
- 稳定性：不得影响窗口布局模式、移动端导航和现有 profile 菜单行为。
- 可维护性：变更范围尽量收敛在主布局组件。

### 输入输出

- 输入：
  - 用户点击侧边栏顶部菜单按钮。
- 输出：
  - 侧边栏在展开态和窄栏态之间切换。
  - 收起态下导航以图标形式展示。

### 边界异常

- 当前 route 变化后，不应重置本次运行期内的收起状态。
- 收起态下长列表仍需可滚动。
- 若图标按钮缺少文本，必须补 `aria-label` 或 `sr-only`，避免可访问性回退。
- 窄栏宽度下不得挤压主内容区造成横向溢出。

### 验收标准

- 左侧边栏顶部不再显示 `Session` 卡片。
- 左侧边栏顶部存在菜单 icon 按钮。
- 点击按钮后，桌面端侧边栏从完整宽度收起为窄栏，仅显示 logo/icon 与导航图标。
- 再次点击按钮后，可恢复完整宽度与导航文案。
- 头部连接状态、profile 菜单、页面路由跳转保持可用。
- 前端构建通过。

### 风险

- 收起态若直接删除文本节点，可能导致命中区域和垂直节奏失衡，需要同步调整容器对齐。
- 栅格宽度切换若写死在模板内，后续扩展多种侧栏模式会增加维护成本。

### 问题清单

- 阻塞：否

## 2. 架构设计（分析）

### 总体方案

- 方案：在 `frontend/src/layout/AppShell.vue` 内增加本地 `sidebarCollapsed` 状态与相关计算属性，统一控制桌面布局栅格宽度、侧边栏头部布局、导航项文案显示与分组标题显示。
- 选型理由：
  - 当前左侧边栏实现完全位于 `AppShell.vue`，状态只影响该组件视图层，无需提升到全局 store。
  - 不做持久化，使用局部 `ref` 可以避免引入额外存储逻辑和副作用。
  - 通过计算属性集中产出 class 和布尔标记，可减少模板中重复拼接，便于后续扩展到更多 sidebar mode。
- 备选对比：
  - 备选 1：直接在模板中用多个三元表达式切换 class。
    - 缺点：可读性差，后续增加模式成本高。
  - 备选 2：将侧边栏状态放入独立 store。
    - 缺点：当前需求不需要跨页面共享或持久化，复杂度偏高。

### 模块职责

- `AppShell.vue`
  - 维护桌面侧边栏展开/收起状态。
  - 渲染顶部菜单按钮。
  - 负责桌面布局栅格和侧边栏导航表现切换。
- 现有 `sessionStore`
  - 继续仅负责头部连接状态展示与 VarPool 恢复流程，不承担侧边栏 UI 状态。

### 数据 / 调用流

- 用户点击菜单按钮。
- `toggleSidebar()` 翻转 `sidebarCollapsed`。
- 计算属性更新：
  - 根级 grid 列宽
  - 侧边栏容器宽度与内边距
  - 品牌头部排版
  - 分组标题可见性
  - 导航项文案、对齐方式和 tooltip/title
- Vue 响应式刷新视图，不触发网络请求或 store 写入。

### 接口草案

- 本地状态：
  - `const sidebarCollapsed = ref(false)`
- 本地方法：
  - `const toggleSidebar = () => { sidebarCollapsed.value = !sidebarCollapsed.value }`
- 本地计算属性：
  - `sidebarGridClass`
  - `sidebarAsideClass`
  - `sidebarHeaderClass`
  - `sidebarNavItemClass`
  - `sidebarNavIconClass`

### 错误与安全

- 该需求不新增外部输入字段、远程调用或敏感数据处理。
- 菜单按钮需补充 `type="button"`、`aria-label` 与收起/展开的可读文本，避免仅图标交互导致可访问性问题。

### 性能与测试策略

- 性能关键点：
  - UI 切换只依赖单个局部 `ref`，避免 store 级级联更新。
  - 不增加 watcher、I/O 或持久化。
  - 仍复用现有导航数组，不新增复制或重排逻辑。
- 测试策略：
  - 运行前端构建，验证模板与 class 变更无编译错误。
  - 启动前端页面后手动验证展开/收起交互、路由跳转与头部功能。
  - 如环境可用，使用 Chrome DevTools MCP 进行桌面视图截图/交互核验。

### 可扩展性设计点

- 将侧边栏模式收敛为显式状态，后续可扩展为 `expanded / compact / hidden` 多模式。
- 文案显示和布局 class 通过计算属性集中管理，未来加入持久化时只需在状态层增加读取/写回。

### 问题清单

- 阻塞：否

## 3.1 计划拆分

### 可执行任务清单

- [x] T1 调整左侧边栏结构与收起交互
- [x] T2 执行构建与交互验证
- [x] T3 完成 Code Review 与变更归档

### 任务详情

#### T1

- Task ID：T1
- 标题：调整左侧边栏结构与收起交互
- Owner：主Agent
- Worktree 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle
- Plan 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle/plan.md
- 目标：
  - 移除侧边栏顶部 `Session` 卡片。
  - 增加菜单 icon 按钮并实现展开/收起切换。
  - 收起态保留图标导航与可访问性信息。
- 涉及模块 / 文件：
  - `frontend/src/layout/AppShell.vue`
- Write set：
  - `frontend/src/layout/AppShell.vue`
- 验收条件：
  - 展开/收起行为符合需求分析。
  - 无计划外业务逻辑改动。
- 测试点：
  - 展开/收起切换。
  - 收起态导航可见且可点击。
  - 头部状态与 profile 菜单正常。
- 回滚点：
  - 回退 `AppShell.vue` 到本分支初始版本。
- 依赖关系：
  - 无前置代码依赖，可直接执行。
- 风险与注意事项：
  - 注意桌面布局列宽切换时主内容区不能抖动过大。
  - 注意 `title` / `sr-only` 补充，避免纯图标导航可访问性下降。
- 关键上下文引用：
  - `frontend/src/layout/AppShell.vue` 中桌面布局区块、顶部 header、navGroups 渲染。

#### T2

- Task ID：T2
- 标题：执行构建与交互验证
- Owner：主Agent
- Worktree 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle
- Plan 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle/plan.md
- 目标：
  - 通过构建验证静态正确性。
  - 对桌面端展开/收起交互做手动验证。
- 涉及模块 / 文件：
  - `frontend/package.json`
  - 前端运行产物
- Write set：
  - 无代码写入，允许生成临时构建产物
- 验收条件：
  - `npm run build` 成功。
  - 手动验证结果记录完整。
- 测试点：
  - 桌面布局下点击菜单按钮展开/收起。
  - 收起态路由跳转。
- 回滚点：
  - 删除临时构建产物，不影响源码。
- 依赖关系：
  - 依赖 T1 完成。
- 风险与注意事项：
  - 若本地依赖未安装，需要先确认 `node_modules` 是否存在。

#### T3

- Task ID：T3
- 标题：完成 Code Review 与变更归档
- Owner：主Agent
- Worktree 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle
- Plan 路径：D:/project/MyFlowHub3/worktrees/refactor-win-sidebar-toggle/plan.md
- 目标：
  - 按规范完成 3.3 审查。
  - 产出 `docs/change` 归档文档。
- 涉及模块 / 文件：
  - `plan.md`
  - `docs/change/YYYY-MM-DD_win-sidebar-toggle.md`
- Write set：
  - `plan.md`
  - `docs/change/`
- 验收条件：
  - Review 结论完整。
  - 归档文档覆盖任务映射、测试、风险与回滚。
- 测试点：
  - 文档内容与实际改动一致。
- 回滚点：
  - 删除新增文档，不影响实现代码。
- 依赖关系：
  - 依赖 T1、T2 完成。
- 风险与注意事项：
  - 归档必须准确反映未使用子Agent的原因。

## 3.2 执行结果

### T1 执行结果

- 结果：完成
- 实际变更：
  - 在 `frontend/src/layout/AppShell.vue` 新增局部 `sidebarCollapsed` 状态、toggle 方法和一组集中管理的侧栏 class 计算属性。
  - 删除左侧顶部 `Session` 卡片。
  - 新增顶部菜单 icon 按钮，支持桌面侧栏展开/收起切换。
  - 收起态隐藏品牌文案、分组标题和导航说明，仅保留图标与无障碍文本。
- 偏差说明：无计划外改动。

### T2 执行结果

- 结果：完成
- 验证记录：
  - `npm run build` 通过。
  - 使用 Chrome DevTools MCP 在 `http://127.0.0.1:4173/#/home` 与 `#/presets` 验证：
    - 初始展开态存在顶部菜单按钮。
    - 点击后侧栏收起为窄栏，仅保留图标。
    - 收起态下点击导航图标仍可路由跳转。
    - 再次点击可恢复展开态。
- 环境说明：
  - 当前 worktree 缺少独立的 `frontend/node_modules` 与 `frontend/wailsjs`，验证时通过 junction 复用主仓对应目录。
  - 纯 Vite 环境缺少 Wails runtime，浏览器验证时注入最小 runtime/go stub 以渲染壳层 UI；该操作未修改源码。

## 3.3 Code Review

- 需求覆盖：通过
  - 已移除顶部 `Session` 卡片，并新增顶部菜单 icon 完成展开/收起。
- 架构合理性：通过
  - 侧栏状态保持在 `AppShell.vue` 局部，未把纯视图状态错误提升到全局 store。
- 性能风险：通过
  - 无新增 I/O、watcher、重复计算或额外数据复制。
- 可读性与一致性：通过
  - 侧栏样式切换集中在计算属性，模板结构仍清晰。
- 可扩展性与配置化：通过
  - 当前状态模型可自然扩展到更多 sidebar mode。
- 稳定性与安全：通过
  - 头部连接状态、profile 菜单与桌面路由跳转验证正常。
  - 新增 toggle 使用原生 `button`，避免可访问性回退。
- 测试覆盖情况：通过
  - 完成前端构建。
  - 完成展开/收起与图标导航跳转的浏览器验证。
- 子Agent治理与审计：通过
  - 本 workflow 未使用子Agent，原因是单一写集、实现与验证强耦合，不具备安全并行拆分条件。

### Review 结论

- 结论：通过
- 阻塞：否
- 下一阶段：已完成 4 归档变更，等待用户确认是否结束本次 workflow。

## 并行性评估

- 当前任务拆分结果：3 个任务中，只有 T1 为实现改动，T2 与 T3 依赖 T1 的最终结果。
- 结论：本 workflow 不派发子Agent。
- 原因：
  - 当前实现写集集中在 `frontend/src/layout/AppShell.vue`，不可安全拆分。
  - 验证与归档依赖实现最终形态，不能并行推进到可独立验收程度。

## 当前结论

- 计划文档已确认并已同步执行结果。
- 阻塞：否
- 下一阶段：等待用户确认是否结束本次 workflow
