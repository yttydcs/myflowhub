# 变更归档：Showcase Throttle 标签与单位说明优化

## 变更背景 / 目标
- 背景：变量配置项当前显示 `Throttle (ms)`，需求希望标题简化为 `Throttle`，并把“毫秒单位”说明放到 tooltip 中。
- 目标：
  - 将展示标题从 `Throttle (ms)` 改为 `Throttle`；
  - 在 tooltip 文案中明确单位为 milliseconds（ms）；
  - 保持节流逻辑与数值行为不变。

## 具体变更内容

### 修改
- `frontend/src/pages/Showcase.vue`
  - `Throttle` 输入校验字段名从 `Throttle (ms)` 调整为 `Throttle`；
  - 标题文本从 `Throttle (ms)` 调整为 `Throttle`；
  - tooltip 文案新增单位说明：`Unit: milliseconds (ms). ...`。

### 新增
- 无。

### 删除
- 无。

## 对应 plan / todo 任务映射
- `SHC-THROTTLE-1`：Throttle 标签与 tooltip 调整。
- `SHC-THROTTLE-2`：一致性与最小回归验证。
- `SHC-THROTTLE-3`：Code Review 与归档。

## 关键设计决策与权衡
- 只做文案层变更，不调整 throttle 数据结构与事件节流策略。
  - 优点：变更最小、回归风险低；
  - 权衡：无。

### 性能要点
- 无新增计算、I/O、订阅或渲染结构变化，性能无影响。

### 可扩展性要点
- 单位说明集中在 tooltip 中，后续国际化可直接替换该文案，不影响逻辑。

## 测试与验证方式 / 结果
- 差异审查：`git diff -- frontend/src/pages/Showcase.vue`（通过）。
- 依赖安装：`npm --prefix frontend install`（通过）。
- 构建验证：`npm --prefix frontend run build`（失败）。
  - 失败信息：`Could not resolve "../../wailsjs/go/session/SessionService" from "src/pages/Home.vue"`。
  - 结论：失败为项目现有 `wailsjs` 生成物/环境问题，非本次文案变更引入。

## Code Review（阶段 3.3）结论
- 需求覆盖：通过
- 架构合理性：通过（纯文案改动）
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：部分通过（静态审查通过；全量 build 受环境问题阻断）

## 潜在影响与回滚方案
- 潜在影响：仅 UI 文案变化，不影响功能。
- 回滚方案：回滚 `frontend/src/pages/Showcase.vue` 本次修改。
