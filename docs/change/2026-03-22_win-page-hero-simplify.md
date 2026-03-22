# 2026-03-22 Win 页面头部简化

## 变更背景 / 目标

- 用户反馈当前 Win 主页面中有多处“顶部小标题 + 主标题 + 提示语”的引导型头部，视觉上偏重复。
- 本次目标是在不影响功能区和交互布局的前提下，收敛主页面首屏头部，仅保留一个更明确的大标题和一行提示语。

## 具体变更内容（新增 / 修改 / 删除）

### 新增

- `frontend/src/components/PageHero.vue`
  - 新增统一的主页面头部组件。
  - 默认读取当前路由的 `meta.title` / `meta.subtitle` 作为主标题和提示语。
  - 通过 `actions` slot 保留页面右侧的 badge、tab、按钮等操作区。

### 修改

- `frontend/src/pages/Flow.vue`
  - 用 `PageHero` 替换原首屏头部。
  - 移除顶部小标题 `Workspace`，主标题改为路由级标题 `Flow`。
- `frontend/src/pages/ShowcaseCenter.vue`
  - 用 `PageHero` 替换原首屏头部。
  - 移除顶部小标题，保留原有的功能提示语和右侧 `screens / last sync / new blank` 操作区。
- `frontend/src/pages/TopicBus.vue`
  - 用 `PageHero` 替换原首屏头部。
  - 移除顶部小标题 `Workspace`，主标题改为更短的 `TopicBus`。
- `frontend/src/pages/VarPool.vue`
  - 用 `PageHero` 替换原首屏头部。
  - 移除顶部小标题 `Workspace`，主标题改为更短的 `VarPool`，并补齐提示语展示。
- `frontend/src/pages/ModuleStub.vue`
  - 用 `PageHero` 替换占位页面头部。
  - 保留原有 `backdrop-blur` 视觉效果和右侧示例按钮。
- `plan.md`
  - 切换为本次 workflow 计划，并记录任务完成情况、验证结果和残余风险。
- `docs/README.md`
  - 新增受治理文档入口，说明新旧 docs 结构边界。
- `docs/change/README.md`
  - 补齐 change 分类索引，并加入本次归档入口。
- `docs/requirements/README.md`
- `docs/specs/README.md`
- `docs/plan/README.md`
- `docs/lessons/README.md`
  - 补齐规范化 docs 分类索引，满足后续 workflow 的治理要求。

### 删除

- 无独立文件删除。

## 对应 plan.md 任务映射

- `T1`
  - 新增 `PageHero` 组件，统一主页面头部骨架。
- `T2`
  - 将 `Flow / ShowcaseCenter / TopicBus / VarPool / ModuleStub` 的首屏头部迁移到 `PageHero`。
- `T3`
  - 收敛主标题文案，保留单行提示语。
- `T4`
  - 执行 `npm ci`、`npm run build`、SFC 解析和旧文案检索，完成 Code Review。
- `T5`
  - 写入本次 `docs/change` 归档，并更新 docs 索引。

## 关键设计决策与权衡

- 采用共享组件而非逐页散改：
  - 优点：模板重复减少，后续同类页面继续收敛时成本更低。
  - 代价：新增一个轻量通用组件，需要页面通过 slot 传入右侧操作区。
- 标题优先使用路由 `meta.title`：
  - 优点：让页面名称更短、更稳定，避免继续在页面模板中散落 `Console / Center / Control Center` 之类的装饰性命名。
  - 代价：页面标题文案与路由元信息耦合，但这类信息本身就属于页面级元数据。
- 性能考虑：
  - 本次为纯展示层调整，不增加请求、不新增循环、不引入重复计算。
  - `PageHero` 只做常量级计算和 slot 渲染，运行时开销可忽略。
- 可扩展性考虑：
  - 后续其它主页面若也需要“单标题 + 单提示语”头部，只需接入 `PageHero`。
  - 个别页面如需保留自定义提示语，可继续通过 `description` 覆盖，而不必分叉组件。

## 测试与验证方式 / 结果

- `npm ci`
  - 结果：通过。
- `npm run build`
  - 结果：失败。
  - 失败原因：仓内既有 `frontend/src/pages/TopicBus.vue` 对 `../../wailsjs/go/main/App` 的引用在当前 worktree 中无法解析，`wailsjs` 目录整体缺失；该问题在本次改动前已存在，非 `PageHero` 改动引入。
- SFC 解析
  - 范围：
    - `frontend/src/components/PageHero.vue`
    - `frontend/src/pages/Flow.vue`
    - `frontend/src/pages/ShowcaseCenter.vue`
    - `frontend/src/pages/TopicBus.vue`
    - `frontend/src/pages/VarPool.vue`
    - `frontend/src/pages/ModuleStub.vue`
  - 结果：全部通过。
- 旧文案检索
  - 检索词：`Workspace`、`Flow Project Center`、`TopicBus Console`、`VarPool Control Center`、`Tool Console`
  - 范围：上述 5 个页面文件
  - 结果：无命中，说明旧头部小标题模式已从目标页面移除。

## Requirements / Specs 影响检查

- Requirements impact：`none`
- Specs impact：`none`
- Related requirements：`none`
- Related specs：`none`
- Lessons：`none`
  - 本次未暴露新的重复性故障模式，不新增 `docs/lessons`

## 潜在影响与回滚方案

- 潜在影响：
  - 页面头部标题会更短，少数页面会从 `* Console / * Center / * Control Center` 收敛为更直接的页面名。
  - 在窄屏下，右侧操作区仍会根据 `flex-wrap` 换行，但头部信息密度整体下降。
- 回滚方案：
  - 回退 `frontend/src/components/PageHero.vue`
  - 回退以下页面文件：
    - `frontend/src/pages/Flow.vue`
    - `frontend/src/pages/ShowcaseCenter.vue`
    - `frontend/src/pages/TopicBus.vue`
    - `frontend/src/pages/VarPool.vue`
    - `frontend/src/pages/ModuleStub.vue`
  - 如仅需回退 docs 治理补齐，可单独回退：
    - `docs/README.md`
    - `docs/change/README.md`
    - `docs/requirements/README.md`
    - `docs/specs/README.md`
    - `docs/plan/README.md`
    - `docs/lessons/README.md`
    - `plan.md`

## 子Agent执行轨迹

- 本轮未使用子Agent。
- 原因：
  - 页面头部调整共享同一 `PageHero` 组件，写集高度耦合。
  - 当前执行策略要求获得显式用户授权后才可委派子Agent。
