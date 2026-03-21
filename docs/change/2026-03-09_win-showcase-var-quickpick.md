# 变更归档：Showcase 变量快捷选择（订阅 / Mine）

## 变更背景 / 目标
- 背景：在 `Showcase` 的 Var Widget 配置里，`Owner NodeID` 与 `Variable Name` 需要手工输入，效率低且容易填错。
- 目标：
  - 在 `Variable Name` 右侧提供快捷选择入口；
  - 从当前订阅变量与 mine 变量中快速选择；
  - 选择后自动回填 `Owner NodeID` 与 `Variable Name`。

## 具体变更内容

### 修改
- `frontend/src/pages/Showcase.vue`
  - 引入 `useVarPoolStore`，复用 VarPool 缓存数据；
  - 新增变量候选计算与过滤逻辑：
    - `Subscribed`：`subKnown && subscribed`；
    - `Mine`：`owner === selfNodeId`；
  - 在 `Variable Name` 右侧新增选择按钮；
  - 新增变量快捷选择弹窗（Overlay）：
    - 支持关键词过滤（name / owner）；
    - 分组展示 `Subscribed` 与 `Mine`；
    - 提供 `Refresh Mine` 刷新入口；
    - 点击条目回填 `Owner NodeID` 与 `Variable Name`。

### 新增
- 无。

### 删除
- 无。

## 对应 plan / todo 任务映射
- `SHC-QUICKPICK-1`：增加选择按钮与弹窗、点击回填。
- `SHC-QUICKPICK-2`：复用 VarPool 数据并按订阅/我的变量分组。
- `SHC-QUICKPICK-3`：回归验证、Code Review 与归档。

## 关键设计决策与权衡
- 采用“复用 VarPool 缓存”的方案，而非新增后端接口。
  - 优点：实现快、风险低、数据语义与 VarPool 页面一致；
  - 权衡：若缓存为空，候选为空，需要用户先刷新/使用 VarPool 产生缓存。
- 在弹窗内增加 `Refresh Mine`。
  - 优点：在不跳转页面的情况下补齐 mine 数据；
  - 风险：依赖当前登录态与绑定可用性，失败时给出错误提示。

### 性能要点
- 候选计算仅在前端内存中进行（Map 合并 + 过滤 + 排序），无新增高频 I/O。
- 不引入轮询或额外订阅。

### 可扩展性要点
- `VarQuickPickItem` 与分组计算可复用于其他变量输入场景。
- 后续如需增加“最近使用”或“按 owner 分组”，可在当前计算链路扩展。

## 测试与验证方式 / 结果
- 代码差异审查：`git diff -- frontend/src/pages/Showcase.vue`（通过）。
- 依赖安装：`npm --prefix frontend install`（通过）。
- 构建验证：`npm --prefix frontend run build`（失败）。
  - 失败信息：`Could not resolve "../../wailsjs/go/session/SessionService" from "src/pages/Home.vue"`。
  - 结论：失败由项目现有 `wailsjs` 生成物/环境问题引起，非本次改动引入。

## Code Review（阶段 3.3）结论
- 需求覆盖：通过
- 架构合理性：通过（复用现有 VarPool 口径）
- 性能风险（N+1/重复计算/多余 I/O/锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：部分通过（静态审查通过；全量构建受环境问题阻断）

## 潜在影响与回滚方案
- 潜在影响：
  - 候选列表依赖缓存，首次无缓存时可能为空；
  - `Refresh Mine` 在未登录场景会报错提示。
- 回滚方案：
  - 回滚 `frontend/src/pages/Showcase.vue` 本次改动。
