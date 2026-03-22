# Win 页面头部简化 Workflow Plan

## 项目目标与当前状态

- 目标：收敛 Win 主页面顶部的引导型头部卡片，去掉顶部小标题，只保留一个主标题和一行提示文案，降低重复感。
- 当前状态：
  - 独占分支已创建：`fix/win-single-hero-title`
  - 独占 worktree 已创建：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title`
  - 已完成需求分析与架构设计，确认本轮只调整主页面首屏头部卡片，不改功能子卡片、弹窗和运行时 window 标题栏。
  - 通过 `$docs-governor` 检查发现当前仓库 `docs/` 为旧结构，需先补齐标准索引后再归档本次变更。

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`fix/win-single-hero-title`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title`
- 当前阶段：`4 归档变更`
- 计划文档：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title\plan.md`

## 文档治理与影响检查

- 使用 `$docs-governor` 的结论：
  - 本次内容属于 workflow 执行与完成记录，计划落点为根 `plan.md`，归档落点为 `docs/change/2026-03-22_win-page-hero-simplify.md`
  - 当前仓缺少规范化 `docs/README.md`、`docs/requirements/`、`docs/specs/`、`docs/plan/`、`docs/lessons/` 索引，需本次补齐
- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none（本次为用户直接提出的视觉简化请求，当前仓尚无对应稳定 requirements 文档）`
- Related specs：`none（未变更技术契约、接口或数据结构）`
- 计划归档路径：`docs/change/2026-03-22_win-page-hero-simplify.md`
- Lessons 路径：`none（暂不预期形成可复用事故/教训文档）`

## 需求分析摘要

- 目标：
  - 将主页面首屏头部从“顶部小标题 + 主标题 + 提示语”收敛为“单一主标题 + 一行提示语”
  - 允许适度收敛主标题文案，使名称更短、更像页面主名称
- 范围：
  - 必须做：主页面头部卡片样式收敛；保留右侧现有操作区、状态徽标、tab 切换
  - 可选：微调主标题文字
  - 不做：内部业务卡片、弹窗标题、viewer/editor window 顶栏
- 验收标准：
  - 目标页面头部不再出现顶部小标题
  - 目标页面仍保留一个主标题和一行提示文案
  - 右侧按钮、badge、tab 布局正常
  - `frontend` 构建通过

## 架构设计摘要

- 采用方案：新增一个轻量可复用的页面头部组件，由页面传入标题、提示语和右侧操作区 slot
- 选型理由：
  - 统一骨架，避免同类模板散落在多个页面
  - 将文案与右侧 actions 解耦，后续页面复用成本更低
  - 只改展示层，不触碰业务状态与 API
- 备选方案：
  - 逐页直接内联修改
  - 不采用原因：短期快，但重复模板仍会继续扩散

## 可执行任务清单

- [x] `T1` 创建可复用页面头部组件
- [x] `T2` 迁移主页面头部到统一组件
- [x] `T3` 调整标题与提示文案
- [x] `T4` 验证构建并执行 Code Review
- [x] `T5` 归档变更并更新索引

## 任务详情

### `T1` 创建可复用页面头部组件

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title\plan.md`
- 目标：新增统一的主页面头部组件，支持主标题、提示语、右侧 slot
- 涉及模块 / 文件：
  - `frontend/src/components/PageHero.vue`
- Write set：
  - `frontend/src/components/PageHero.vue`
- 关键上下文引用：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/ModuleStub.vue`
- 依赖：无
- 验收条件：
  - 组件可渲染单一主标题和一行提示
  - 可通过 slot 保留右侧操作区
  - 响应式布局下不破坏现有按钮区换行
- 测试点：
  - `npm run build`
- 回滚点：
  - 删除新组件并恢复各页面原模板
- 风险与注意事项：
  - 仅做结构抽取，不在组件内耦合业务 store 或路由状态

### `T2` 迁移主页面头部到统一组件

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title\plan.md`
- 目标：将主页面首屏头部改为统一组件，去掉顶部小标题
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/ModuleStub.vue`
- Write set：
  - 上述页面文件
- 关键上下文引用：
  - `frontend/src/router/index.ts`
- 依赖：
  - `T1`
- 验收条件：
  - 页面首屏头部只保留一个主标题和一行提示
  - 原 badge、按钮、tab 切换仍在头部右侧工作
- 测试点：
  - 检查模板编译
  - 代码核对各页面 slot 保持原交互元素
- 回滚点：
  - 按页面恢复旧模板
- 风险与注意事项：
  - 只改首屏头部，不触碰内部业务区块

### `T3` 调整标题与提示文案

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title\plan.md`
- 目标：在不改业务语义的前提下，收敛大标题文案，让其更短、更稳定
- 涉及模块 / 文件：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/ModuleStub.vue`
- Write set：
  - 上述页面文件
- 依赖：
  - `T2`
- 验收条件：
  - 每个页面保留一行提示文案
  - 大标题文字更简洁，但不影响页面识别
- 测试点：
  - 人工核对文案与页面职责一致
- 回滚点：
  - 恢复原文案
- 风险与注意事项：
  - 文案只做轻量收敛，不引入新的业务名词

### `T4` 验证构建并执行 Code Review

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title\plan.md`
- 目标：完成本轮实现验证和强制 Code Review
- 涉及模块 / 文件：
  - `frontend/**`
  - `plan.md`
- Write set：
  - 无新增实现写集，允许只补充必要文档状态
- 依赖：
  - `T1`
  - `T2`
  - `T3`
- 验收条件：
  - 构建通过
  - Review 逐项给出通过/不通过结论
- 测试点：
  - `npm run build`
- 回滚点：
  - 按任务回退对应页面头部改动
- 风险与注意事项：
  - 若构建失败，回到 `3.2` 修正后再 Review

### `T5` 归档变更并更新索引

- Owner：`主Agent`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-single-hero-title\plan.md`
- 目标：生成 `docs/change` 归档并更新本仓 docs 索引
- 涉及模块 / 文件：
  - `docs/README.md`
  - `docs/change/README.md`
  - `docs/change/2026-03-22_win-page-hero-simplify.md`
- Write set：
  - 上述文档文件
- 依赖：
  - `T4`
- 验收条件：
  - 存在变更归档文档
  - 已记录 requirement/spec impact
  - docs 索引可导航到本次归档
- 测试点：
  - 人工核对索引链接与文档字段完整性
- 回滚点：
  - 删除本次新增归档并恢复索引到改动前状态
- 风险与注意事项：
  - 归档前必须再次显式执行 `$docs-governor` 检查

## 并行性评估

- 结论：本轮不使用子Agent
- 原因：
  - 目标页面头部改造共享同一个新组件，文件写集高度耦合
  - 当前执行策略要求显式用户授权后才可委派子Agent，本轮未获得该授权

## 回归与验证步骤

1. 在 `frontend/` 目录执行 `npm run build`
2. 核对目标页面模板中不再存在顶部小标题
3. 核对右侧操作区、tab、badge 仍保留并正常渲染
4. 完成强制 Code Review 后进入 `4`

## 当前结果

- 已完成 `T1` ~ `T5`
- 已新增 `frontend/src/components/PageHero.vue`，并接入：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/ModuleStub.vue`
- 已补齐受治理的 docs 索引：
  - `docs/README.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
  - `docs/plan/README.md`
  - `docs/change/README.md`
  - `docs/lessons/README.md`
- 验证结果：
  - `npm ci`：通过
  - `npm run build`：失败，原因为仓内既有 `wailsjs/go/main/App` 缺失，非本次改动引入
  - SFC 解析：`PageHero.vue` 与 5 个接入页面均通过
  - 旧头部文案检索：目标页面已无 `Workspace / Flow Project Center / TopicBus Console / VarPool Control Center / Tool Console`
- 残余风险：
  - 尚未完成带 Wails 绑定的全量前端构建
  - 未执行桌面运行态人工 UI 冒烟
