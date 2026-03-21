# 变更归档：Showcase 工具栏 Tooltip 组件化与变量弹窗优化

## 变更背景 / 目标
- `Showcase` 顶部按钮此前使用原生 `title` 提示，且 screen 级操作与 widget 新增操作视觉上未分组。
- 变量新增弹窗里 `Target ID` 暴露给用户，易造成配置偏差。
- `Throttle (ms)` 说明常驻占位，`On/Off` 在非 switch 模式下显示会造成理解噪声。

目标：
1. 顶部按钮 tooltip 组件化，提示在按钮正下方显示；
2. 在 screen 操作与 Add Event/Add Var 之间增加分割线；
3. 新增变量默认 target=Hub NodeID，并在变量表单隐藏 target 输入；
4. `Throttle (ms)` 详细说明收敛到 tooltip；
5. `On Value/Off Value` 仅在 `switch` 模式显示并独立分区。

## 具体变更内容

### 新增
- `frontend/src/components/ui/tooltip/Tooltip.vue`
  - 新增可复用 Tooltip 组件（基于 `radix-vue`），默认 `side="bottom"`。
- `frontend/src/components/ui/tooltip/index.ts`
  - 导出 `Tooltip` 组件。

### 修改
- `frontend/src/pages/Showcase.vue`
  - 顶部 6 个操作按钮全部接入 `Tooltip` 组件，tooltip 文案沿用原按钮文本；
  - 顶部按钮按功能分组，并在两组之间增加竖向分割线；
  - 变量提交逻辑新增 `resolveVarTargetID`：
    - 新增 var：强制 target 使用 `hubId`；
    - 编辑 var：优先保留原 target，缺失时回退 `hubId`；
    - 无可用 Hub NodeID 时阻断并报错；
  - 变量弹窗表单：
    - `Target ID` 仅在 `topic_button` 时显示；
    - `Throttle (ms)` 改为标题 tooltip（移除常驻说明文本）；
    - `On/Off` 改为仅 `mode=switch` 显示，并新增 `Switch Settings` 分割区；
  - 变量提交校验：`On/Off` 仅在 `mode=switch` 时强校验。

## 对应 plan / todo 任务映射
- `SHC-UX-1`：Tooltip 组件 + 顶部按钮接入 + 分割线分组。
- `SHC-UX-2`：变量 target 逻辑收敛到 hub 且隐藏变量 target 输入。
- `SHC-UX-3`：Throttle 标题 tooltip 化 + switch 模式专属设置区。
- `SHC-UX-4`：验证、Code Review 与归档。

## 关键设计决策与权衡
- Tooltip 采用 UI 组件（`radix-vue` 封装），而非继续用 `title`：
  - 优点：位置可控（下方）、表现一致、可复用；
  - 成本：新增一个轻量 UI 组件文件夹。
- 变量 target 不再让用户手工输入：
  - 优点：减少误配，符合“变量走 Hub”默认链路；
  - 权衡：降低了灵活性，但满足当前产品期望与易用性优先策略。
- `On/Off` 仅在 switch 显示：
  - 优点：显著降低非 switch 模式噪声；
  - 风险：无，字段值仍保留在状态中，切换回 switch 可继续编辑。

### 性能要点
- 仅前端展示与输入逻辑改造，无新增网络请求、存储 I/O、轮询或高频计算。
- Tooltip 仅 hover/focus 时渲染浮层，不影响主渲染路径复杂度。

### 可扩展性要点
- 新增 `Tooltip` 组件可复用于其它页面（替代散落 `title`）。
- 变量 target 的解析封装在 `resolveVarTargetID`，后续若要支持策略化 target，可在该函数集中扩展。

## 测试与验证方式 / 结果
- 代码差异审查：`git diff -- frontend/src/pages/Showcase.vue frontend/src/components/ui/tooltip/*`（通过）。
- 依赖安装：`npm --prefix frontend install`（通过）。
- 构建验证：`npm --prefix frontend run build`（失败，非本次改动引入）。
  - 失败信息：`Could not resolve "../../wailsjs/go/auth/AuthService" from "src/pages/Home.vue"`。
  - 结论：构建被项目现有生成物/环境问题阻断，无法在当前环境完成全量构建验收。

## Code Review（阶段 3.3）结论
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1/重复计算/多余 I/O/锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：部分通过（静态审查通过；全量 build 受环境问题阻断，已记录）

## 潜在影响与回滚方案
- 潜在影响：
  - 顶部按钮 tooltip 从原生 `title` 切换到组件后，禁用态按钮的提示行为可能与浏览器原生略有不同；
  - 变量新增 target 固定 Hub，若极少数场景需要手工 target，将需后续再开放配置开关。
- 回滚方案：
  - 回滚 `frontend/src/pages/Showcase.vue`；
  - 删除 `frontend/src/components/ui/tooltip/`；
  - 回滚本归档文档。
