# 2026-03-24_win-showcase-var-picker-watch-all

## 变更背景 / 目标

- 背景：
  - 上一轮将 Showcase 变量快捷选择的主候选改成了“当前 screen 变量上下文 + mine”，但用户最新确认希望主候选语义是“所有被 watch 的变量”。
  - 当前实现虽然避免了空弹窗，但会遗漏未订阅、仅 watch 的变量。
- 目标：
  - 让 Showcase 变量快捷选择展示所有有效 watched 变量，而不是只看订阅态或当前 screen 上下文。
  - 同步修正文案与分组标题，使 UI 语义与实际数据来源一致。

## 具体变更内容

### 修改

- `frontend/src/pages/Showcase.vue`
  - 移除上轮引入的 `quickPickSourceScreen` 与基于 `showcase.state.values` 的主候选链路。
  - `VarQuickPickItem` 的主标记从 `subscribed` 改为 `watched`。
  - `watchedVarQuickPickSourceItems` 改为直接基于 `varpool.state.keys` 构建，并仅要求 `name` 非空且 `owner > 0`。
  - 保留 `loadedMineVarQuickPickItems` 的显式拉取逻辑，并继续与 watched 候选按 `(owner:name)` 去重。
  - 弹窗和 tooltip 文案从 `Subscribed` / `subscribed` 收敛为 `Watched` / `watched`。
  - watched 分组空态文案改为 `No watched variables.`。

## Requirements impact

- `none`

## Specs impact

- `none`

## Related requirements

- `docs/requirements/showcase-display-widgets.md`

## Related specs

- `docs/specs/showcase-display-widgets.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`

## 对应 plan.md 任务映射

- `SHVW-1`：将快捷选择主候选改为 watched 语义。
- `SHVW-2`：同步更新文案、分组和空态。
- `SHVW-3`：验证、Code Review 与归档。

## 关键设计决策与权衡

- 主候选源切回 `varpool.state.keys`。
  - 原因：用户明确要的是“所有被 watch 的变量”，而不是“当前 screen 变量”或“已订阅变量”。
  - 收益：只要变量已进入 watch list，即使尚未订阅，也能在 Showcase 里直接选中。
- 继续保留 mine 显式拉取，而不是把 mine 写回 watch list。
  - 原因：watch 和 mine 是两种不同来源，混写会污染语义。
- 继续过滤 `owner<=0` 的 watched 条目。
  - 原因：快捷选择点击后需要明确回填 `Owner NodeID`，不能猜测 owner。

## 测试与验证方式 / 结果

- `git diff --check`
  - 结果：通过。
- `npm ci`
  - 结果：通过。
- `@vue/compiler-sfc` 定向解析 `frontend/src/pages/Showcase.vue`
  - 结果：通过。
- `npm run build`
  - 结果：失败。
  - 原因：仓库基线缺失 `frontend/wailsjs` 生成物，失败点为 `src/pages/Home.vue` 无法解析 `../../wailsjs/go/session/SessionService`；不是本次修改引入。

## 潜在影响

- 旧 watch 数据若缺少 `owner`，仍不会出现在快捷选择器中；这是当前回填契约下的显式降级。
- watched 分组现在与 `VarPool` watch list 强绑定，不再代表“当前 screen 变量”。

## 回滚方案

- 回滚 `frontend/src/pages/Showcase.vue` 本次改动。
- 删除本归档文档并回退 `docs/change/README.md` 更新。

## 子Agent执行轨迹

- 本轮未派发子Agent。
