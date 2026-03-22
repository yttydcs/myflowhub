# 2026-03-22 Win 卡片头收敛

## 变更背景 / 目标

- 背景：Win 端多个页面与独立窗口里存在重复的“顶部小标题 + 主标题 + 可选提示语”卡片头，首屏信息层级偏重复。
- 目标：遍历 `frontend/src/pages/**` 与 `frontend/src/windows/**` 中命中的卡片头，统一收敛为“主标题 + 可选提示语”，保留原有 badge、按钮和状态区，不补不存在的提示语。

## 文档治理与影响检查

- Requirements impact: `none`
- Specs impact: `none`
- Related requirements: `none`
- Related specs: `none`
- Lessons impact: `none`
- Index updates:
  - 已更新 `docs/change/README.md`
  - 未更新 `docs/plan/README.md`，原因是当前 workflow 的执行计划仍以 worktree 根 `plan.md` 为准

## 具体变更内容（新增 / 修改 / 删除）

### 新增

- `frontend/src/components/CardHeader.vue`
  - 新增轻量卡片头组件，统一承接 `title / description / titleTag / titleClass / descriptionClass` 与 `actions` slot。
  - 组件只负责卡片内部标题区，不抽取卡片外层容器，不耦合业务状态。

### 修改

- 页面类卡片头：
  - `frontend/src/pages/Home.vue`
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/pages/Presets.vue`
  - `frontend/src/pages/Settings.vue`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/VarPool.vue`
  - 将命中的卡片头替换为 `CardHeader`，删除顶部小标题，保留原有主标题、可选提示语和右侧操作区。
- 窗口类卡片头：
  - `frontend/src/windows/FileTasks.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/windows/LogWindow.vue`
  - `frontend/src/windows/TopicBusWindow.vue`
  - 同步完成窗口内业务卡片头收口，保留原按钮、badge、统计信息与窗口交互逻辑。
- `plan.md`
  - 更新任务状态、当前阶段、验证结果与 Code Review 结论。

### 未修改

- 未修改字段 `label`、列表项内部元信息、store / route / Wails 调用逻辑。
- 未新增任何原本不存在的提示语。
- 未改动已经是单行窗口顶栏的区域。

## 对应 plan.md 任务映射

- `T1` 新增共享卡片头组件：完成。
- `T2` 收敛页面类卡片头：完成。
- `T3` 收敛窗口类卡片头：完成。
- `T4` 验证并执行 Code Review：完成。
- `T5` 归档变更并更新索引：完成。

## 关键设计决策与权衡

- 采用共享 `CardHeader`，不在 13 个文件里直接散删小标题。
  - 收益：结构统一，后续新增卡片头时能继续复用同一套模板。
  - 代价：新增一个轻量展示组件，但组件本身不持有业务状态，维护成本低。
- 只抽取卡片内部头部，不抽取卡片容器。
  - 原因：不同页面和窗口的外层边框、padding、背景和滚动语义并不一致。
  - 收益：改动范围更小，不影响现有布局骨架。
- 没有提示语的区域不补描述。
  - 原因：本轮目标是精简视觉层级，不是重写信息文案。
  - 收益：避免引入计划外文案改动。
- 性能说明：
  - 本次为纯模板层重构，不新增请求、不新增循环、不新增重复计算。
  - `CardHeader` 只做常量级 slot 判断和条件渲染，运行时开销可忽略。

## 测试与验证方式 / 结果

- 文本扫描：
  - 命令：`rg -n "text-xs font-semibold uppercase tracking-\\[0\\.3em\\]" frontend/src/pages frontend/src/windows`
  - 结果：无命中，确认原 0.3em 小标题模式已从页面与窗口层清空。
- 模式复核：
  - 命令：`rg -U -n '<p class="text-xs font-semibold uppercase tracking-\\[0\\.[23]em\\] text-muted-foreground">[\\s\\S]{0,200}<h[123]' frontend/src/pages frontend/src/windows`
  - 结果：无命中，确认未残留同类“小标题 + 标题”组合。
- Vue SFC 解析：
  - 命令：使用 `@vue/compiler-sfc` 解析 `CardHeader.vue` 与所有改动过的 Vue 文件。
  - 结果：通过。
- 依赖准备：
  - 命令：`npm ci`
  - 结果：通过。
- 前端构建：
  - 命令：`npm run build`
  - 结果：失败。
  - 失败原因：`src/pages/Home.vue` 仍无法解析 `../../wailsjs/go/main/App`，属于现有 Wails 生成绑定缺失问题，与本次卡片头收敛改动无关。

## Code Review 结论

- 需求覆盖：通过。已覆盖命中的页面与窗口卡片头，并保持“无提示语不补”的边界。
- 架构合理性：通过。新增组件职责单一，未把页面级 `PageHero` 和卡片级头部语义混在一起。
- 性能风险：通过。纯展示层改动，无新增 I/O、重复计算或额外状态同步。
- 可读性与一致性：通过。各处头部结构收敛到统一组件，模板重复显著下降。
- 可扩展性与配置化：通过。后续卡片头只需传入标题、描述和 actions slot 即可复用。
- 稳定性与安全：通过。未触碰业务调用、权限逻辑或输入处理。
- 测试覆盖情况：有残余风险。静态解析和文本扫描通过，但完整生产构建仍受既有 `wailsjs` 生成物缺失影响。
- 子Agent治理与审计：通过。本轮未使用子Agent。

## 潜在影响与回滚方案

- 潜在影响：
  - 目标卡片头高度会缩短，首屏信息密度更直接。
  - 右侧操作区在窄宽度下仍通过 `flex-wrap` 换行，交互保持不变。
- 回滚方案：
  - 删除 `frontend/src/components/CardHeader.vue`
  - 回退上述页面与窗口文件
  - 回退 `docs/change/2026-03-22_win-card-header-simplify.md` 与 `docs/change/README.md`

## 子Agent执行轨迹

- 本轮未使用子Agent。
