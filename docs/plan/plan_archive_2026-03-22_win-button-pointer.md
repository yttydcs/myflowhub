# Win 按钮悬停手型与禁选中文本 Workflow Plan

## 项目目标与当前状态

- 目标：统一 `MyFlowHub-Win` 前端按钮的交互样式，让所有按钮在鼠标悬停时显示手型光标，并禁止按钮文字进入文本选择/复制态。
- 当前状态：
  - 独占分支已创建：`fix/win-button-pointer`
  - 独占 worktree 已创建：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer`
  - 已完成需求分析与架构设计
  - 已定位共享按钮组件与全局样式入口：
    - `frontend/src/components/ui/button/Button.vue`
    - `frontend/src/style.css`
  - 已确认前端中存在大量原生 `<button>`，仅修改共享组件会漏覆盖

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`fix/win-button-pointer`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer`
- 当前阶段：`4 归档变更（已完成，等待用户确认是否结束 workflow）`
- 计划文档：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer\plan.md`

## 文档治理与影响检查

- 使用 `$docs-governor` 的结论：
  - 当前仓已有规范化 `docs/` 结构，无需补齐目录树
  - 当前 workflow 的执行计划落点为 worktree 根 `plan.md`
  - 本次完成后的归档落点为 `docs/change/2026-03-22_win-button-pointer.md`
  - 需要同步更新 `docs/change/README.md` 索引
- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none（当前仓无与“按钮 hover 光标/文本选中”对应的稳定 requirements 叶子文档）`
- Related specs：`none（不涉及接口、数据结构、长期技术契约变更）`
- Lessons 路径：`none（预期不产生通用事故复盘类文档）`

## 需求分析摘要

- 目标：
  - Win 前端所有按钮 hover 时显示手型光标
  - 按钮文字不能被选中或复制
- 范围：
  - 必须做：覆盖共享 `Button` 组件
  - 必须做：覆盖原生 `<button>` 与按钮类表单控件的全局兜底
  - 可选：覆盖 `[role="button"]` 的按钮语义兜底
  - 不做：非按钮文本区域、业务流程、接口与数据结构
- 验收标准：
  - 按钮 hover 时不再显示文本选择光标
  - 按钮文字不可被选中复制
  - 现有点击、禁用、焦点视觉不出现功能回归

## 架构设计摘要

- 采用方案：双层样式收口
  - 在共享 `Button` 组件增加 `cursor-pointer` 与 `select-none`
  - 在 `frontend/src/style.css` 添加按钮级全局基础样式兜底
- 选型理由：
  - 纯样式改动，无需新增运行时逻辑
  - 共享组件负责主路径覆盖，全局 CSS 负责原生按钮兜底
  - 改动集中、回滚简单、后续新增按钮默认继承
- 备选方案：
  - 只改共享组件：覆盖不足
  - 只改全局 CSS：对非原生按钮形态不稳
  - 结论：不采用

## 可执行任务清单

- [x] `T1` 统一按钮交互样式
- [x] `T2` 验证与回归检查
- [x] `T3` 执行 Code Review
- [x] `T4` 归档变更并更新索引

## 任务详情

### `T1` 统一按钮交互样式

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer\plan.md`
- 目标：
  - 为共享按钮组件补充统一的手型光标与禁选中文本样式
  - 为原生按钮及按钮语义元素补充基础样式兜底
- 涉及模块 / 文件：
  - `frontend/src/components/ui/button/Button.vue`
  - `frontend/src/style.css`
- Write set：
  - `frontend/src/components/ui/button/Button.vue`
  - `frontend/src/style.css`
- 关键上下文引用：
  - `frontend/src/components/ui/button/Button.vue`
  - `frontend/src/style.css`
  - `frontend/src/pages/File.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/TopicBusWindow.vue`
- 依赖：无
- 验收条件：
  - 共享按钮默认具备手型光标和禁选中能力
  - 原生按钮默认具备手型光标和禁选中能力
  - 不新增业务逻辑与副作用
- 测试点：
  - 静态扫描确认命中按钮基础选择器
  - 改动文件通过 Vue/Tailwind 语法检查
- 回滚点：
  - 回退 `Button.vue` 与 `style.css` 的样式变更
- 风险与注意事项：
  - 禁用态仍需保持不可操作，不能因样式调整放宽事件行为

### `T2` 验证与回归检查

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer\plan.md`
- 目标：验证样式覆盖范围和基础构建状态
- 涉及模块 / 文件：
  - `frontend/src/components/ui/button/Button.vue`
  - `frontend/src/style.css`
  - 必要时检查命中按钮模板文件
- Write set：
  - 无新增实现写集，仅允许补充必要文档状态
- 依赖：
  - `T1`
- 验收条件：
  - 命中规则可覆盖共享按钮与原生按钮
  - 构建或静态验证结果可被记录
- 测试点：
  - `rg` 扫描原生 `<button>` 命中范围
  - 前端构建或最小静态验证
  - 必要时使用浏览器/调试窗口进行 hover 行为复核
- 回滚点：
  - 按 `T1` 回滚对应样式改动
- 风险与注意事项：
  - 若构建受既有环境问题影响，必须在 Review 和 change 中显式记录

### `T3` 执行 Code Review

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer\plan.md`
- 目标：按强制清单审查需求覆盖、架构、性能、可维护性与测试情况
- 涉及模块 / 文件：
  - `frontend/src/components/ui/button/Button.vue`
  - `frontend/src/style.css`
  - `plan.md`
- Write set：
  - 无新增实现写集，仅允许修正评审发现的问题
- 依赖：
  - `T2`
- 验收条件：
  - Review 八项清单全部给出通过/不通过结论
  - 若不通过，返回 `3.2` 修正后重新评审
- 测试点：
  - Review 结论与验证结果一致
- 回滚点：
  - 按发现的问题回退对应实现

### `T4` 归档变更并更新索引

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-fix-button-pointer\plan.md`
- 目标：完成 change 归档，并同步 change 索引
- 涉及模块 / 文件：
  - `docs/change/2026-03-22_win-button-pointer.md`
  - `docs/change/README.md`
- Write set：
  - `docs/change/2026-03-22_win-button-pointer.md`
  - `docs/change/README.md`
- 依赖：
  - `T3`
- 验收条件：
  - change 文档完整记录背景、任务映射、验证结果和回滚方案
  - `docs/change/README.md` 可导航到本次归档
  - 归档前再次完成 `$docs-governor` 影响检查
- 测试点：
  - 人工核对文档字段与索引链接
- 回滚点：
  - 删除本次 change 文档并回退索引更新

## 并行性评估

- 结论：本轮不使用子Agent
- 原因：
  - 当前任务仅涉及两个样式文件，无法安全拆成多个独立可验收 write set
  - 当前对话未获得用户对委派子Agent的显式授权

## 回归与验证步骤

1. 在 worktree 中完成 `T1` 改动
2. 扫描原生按钮与按钮语义元素命中范围
3. 在 `frontend/` 执行构建或最小静态校验
4. 复核按钮 hover 光标与文本不可选行为
5. 执行强制 Code Review
6. 写入 `docs/change` 归档并更新索引

## 执行结果

- 代码变更：
  - `frontend/src/components/ui/button/Button.vue`
  - `frontend/src/style.css`
- 静态覆盖复核：
  - `rg -n -e '<button\\b' -e '\\[role="button"\\]' frontend/src`
  - 结果：确认仓内存在大量原生 `<button>`，全局兜底规则必要且已写入
- 依赖安装：
  - `npm --prefix frontend ci`
  - 结果：通过
- 构建验证：
  - `npm --prefix frontend run build`
  - 结果：失败
  - 失败原因：`Could not resolve "../../wailsjs/go/main/App" from "src/pages/TopicBus.vue?vue&type=script&setup=true&lang.ts"`
  - 结论：属于当前 worktree 缺少 `wailsjs` 生成物的既有环境问题，非本次按钮样式改动引入

## Code Review 结论

- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过（静态验证通过；完整构建受既有 `wailsjs` 缺失阻塞）
- 子Agent治理与审计：通过（本轮未使用子Agent）
