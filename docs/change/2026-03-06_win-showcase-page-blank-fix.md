# 变更归档：修复 Showcase 页面空白（Tooltip Provider 注入）

## 变更背景 / 目标
- 背景：`Showcase` 页面在最近引入 tooltip 组件后出现整页空白，导致页面不可用。
- 目标：
  - 修复 tooltip 组件注入链路，恢复 `Showcase` 页面渲染；
  - 保持现有 tooltip 调用点与业务逻辑不变。

## 根因分析
- `frontend/src/components/ui/tooltip/Tooltip.vue` 仅使用了 `TooltipRoot/TooltipTrigger/TooltipContent`，未包裹 `TooltipProvider`。
- `radix-vue` 的 Tooltip 实现中，`TooltipRoot` 会强制注入 `TooltipProvider` 上下文；缺失时抛出 `Injection ... not found` 异常，导致页面渲染中断。

## 具体变更内容

### 修改
- `frontend/src/components/ui/tooltip/Tooltip.vue`
  - 引入 `TooltipProvider`；
  - 在组件模板中使用 `TooltipProvider` 包裹 `TooltipRoot`，确保上下文注入完整。

### 新增
- 无。

### 删除
- 无。

## 对应 plan / todo 任务映射
- `SHC-BLANK-1`：修复 Tooltip 注入链路。
- `SHC-BLANK-2`：最小回归验证（执行 build 并记录环境失败原因）。
- `SHC-BLANK-3`：Code Review 与归档。

## 关键设计决策与权衡
- 采用“组件内部补 Provider”的最小修复方案，而非改动所有 Tooltip 使用点。
  - 优点：变更面最小、回归风险低、可立即恢复页面；
  - 权衡：每个 Tooltip 实例各自携带 Provider（轻量，当前可接受）。

### 性能要点
- 仅新增一层轻量 Provider 组件，不引入额外 I/O、轮询或计算热点。

### 可扩展性要点
- Tooltip 封装后续仍可统一扩展（如全局 delay、主题），调用方无需改造。

## 测试与验证方式 / 结果
- 差异审查：`git diff -- frontend/src/components/ui/tooltip/Tooltip.vue`（通过）。
- 构建验证：`npm --prefix frontend run build`（执行失败）。
  - 失败信息：`Could not resolve ../../wailsjs/go/session/SessionService from src/pages/Home.vue`。
  - 结论：失败由项目现有 `wailsjs` 生成物缺失引起，非本次 tooltip 修复引入。
- 运行时结论：根据 `radix-vue` 代码路径可确认缺 Provider 会触发注入异常，本次修复消除该风险路径。

## Code Review（阶段 3.3）结论
- 需求覆盖：通过
- 架构合理性：通过（最小改动、无外溢）
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：部分通过（静态与根因链路验证通过；全量构建受环境阻断）

## 潜在影响与回滚方案
- 潜在影响：
  - Tooltip 实例数增多时会有更多 Provider 实例，但成本很低；
  - 不影响业务逻辑、数据与接口。
- 回滚方案：
  - 回滚 `frontend/src/components/ui/tooltip/Tooltip.vue` 本次修改。
