# 2026-03-23_win-showcase-var-picker-fix

## 变更背景 / 目标

- 背景：
  - Showcase Editor 的 `Variable Name` 选择按钮与输入框不在同一操作行，编辑效率较差。
  - 变量快捷选择弹窗错误依赖 `VarPool` watch list / subscribe 状态，导致用户未先使用 `#/varpool` 时弹窗经常为空。
- 目标：
  - 让变量名输入框与选择按钮并排显示。
  - 让快捷选择弹窗在 Showcase Editor 场景下稳定显示“当前 screen 变量 + mine 变量”候选。

## 具体变更内容

### 修改

- `frontend/src/pages/Showcase.vue`
  - 新增 `quickPickSourceScreen`，以当前编辑 screen 作为快捷选择候选的上下文来源。
  - 新增 `loadedMineVarQuickPickItems`，把 mine 列表改为弹窗内显式加载状态，而不是依赖 `varpool.state.keys` 副作用。
  - 新增 `mergeVarQuickPickItem`，统一按 `(owner:name)` 去重并合并 `mine` / `subscribed` 标记。
  - 将 `Subscribed` 候选改为基于当前 screen 的 `var` widgets，并只合并这些 widgets 对应的当前快照，避免跨 screen 旧值泄漏。
  - 将 mine 刷新逻辑从 `varpool.listMine()` 改为 `varpool.listOwnerNames(selfNodeId)`，直接拉取当前节点变量名列表。
  - 调整 `Variable Name` 表单结构：保留 label 在上方，输入框与选择按钮放入同一行。

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

- `SHVP-1`：修复 Showcase 变量快捷选择候选来源。
- `SHVP-2`：调整 `Variable Name` 输入区为同一行布局。
- `SHVP-3`：验证、Code Review 与归档。

## 关键设计决策与权衡

- 候选源改为“当前 screen + mine”，不再继续绑定 `VarPool` watch list。
  - 原因：Showcase Editor 的可用候选应由当前编辑上下文决定，而不是取决于用户是否预先去 VarPool 页面做过 watch。
  - 收益：在 Showcase 独立使用场景下，弹窗不再常态空白。
- mine 列表使用 `listOwnerNames(selfNodeId)` 显式拉取，而不是继续复用 `listMine()` 对全局 store 的写入副作用。
  - 原因：弹窗候选属于局部 UI 状态，显式加载更清晰，也更容易控制去重和错误处理。
- 当前快照只合并“当前 screen 已配置变量”的相关项。
  - 原因：`showcase.state.values` 不会在 `leave()` 时自动清空，若全量吞入可能带入历史 screen 旧值。

## 测试与验证方式 / 结果

- `git diff --check`
  - 结果：通过。
- `npm ci`
  - 结果：通过。
- `npm run build`
  - 结果：失败。
  - 原因：仓库基线缺失 `frontend/wailsjs` 生成物，失败点为 `src/pages/Home.vue` 无法解析 `../../wailsjs/go/auth/AuthService`；不是本次修改引入。
- `@vue/compiler-sfc` 定向解析 `frontend/src/pages/Showcase.vue`
  - 结果：通过。
  - 说明：已确认本次改动后的 SFC 脚本与模板可解析。

## 潜在影响

- 当前 screen 没有任何 `var` widget，且当前节点也没有 mine 变量时，弹窗仍会显示为空态；这是有效业务结果，不是错误。
- 快捷选择的 `Subscribed` 语义本轮明确收敛为“当前 screen 变量上下文”，不再代表全局 VarPool watch/subscription 列表。

## 回滚方案

- 回滚 `frontend/src/pages/Showcase.vue` 本次变更。
- 删除本归档文档并回退 `docs/change/README.md` 索引更新。

## 子Agent执行轨迹

- 本轮未派发子Agent。
