# Showcase 界面进一步精简

## 变更背景 / 目标
- 在上一轮 `Showcase Center + 独立编辑窗口` 改造基础上继续做界面减法。
- 编辑页去掉 `Screen` 与 `Widget Outline` 两块次要信息区域，尽量只保留编辑本身。
- 列表页收敛为单行优先展示，不再显示 `screen_id`，时间字段不再带 `Updated` 文案。

## 具体变更内容

### 修改
- `frontend/src/pages/Showcase.vue`
  - 移除头部中的 `screen_id / Self / Hub` 信息行。
  - 移除单独的 `Screen` 卡片。
  - 移除单独的 `Widget Outline` 卡片。
  - 将 `Screen Name`、widget 数量、保存时间、layout 控制压缩进同一个主编辑卡片。
  - 保留预览区、保存/回退、打开 Viewer、添加 widget、layout 调整。
  - 增加更轻量的说明文案，提示直接在预览区编辑与右键操作。
- `frontend/src/pages/ShowcaseCenter.vue`
  - 单条 screen 改为单行优先布局。
  - 移除 `screen_id` 展示。
  - 时间字段仅显示时间值。
  - 保留 `Edit / View / Duplicate / Rename / Delete` 操作。

### 新增
- `plan.md`
  - 本轮 workflow 计划与 review 记录。

### 删除
- 无物理删除文件。
- 逻辑上删除了编辑页中的两个信息区块和列表页中的 `screen_id` 行。

## 对应 plan.md 任务映射
- `SS-01`
  - `frontend/src/pages/ShowcaseCenter.vue`
- `SS-02`
  - `frontend/src/pages/Showcase.vue`
- `SS-03`
  - `plan.md`
  - `docs/change/2026-03-21_showcase-ui-simplify.md`

## 关键设计决策与权衡
- 编辑页采用“单主体编辑卡片 + 预览区”方案，而不是保留弱化后的左栏。
  - 原因：用户明确表示不要 `Screen` 和 `Outline`，仅弱化样式不满足目标。
- 列表页保留 badge + 操作区，但压缩到单行优先，而不是做成真正表格。
  - 原因：既能满足紧凑展示，也能延续现有卡片式视觉语言，改动更小。
- 本轮不调整 store 或 Viewer。
  - 原因：需求集中在界面简化，继续限制变更面有利于稳定性。

## 测试与验证方式 / 结果
- Vue SFC 解析
  - 文件：
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/pages/ShowcaseCenter.vue`
  - 结果：通过。
- 前端完整构建
  - 命令：`npm run build`
  - 结果：失败。
  - 原因：当前 worktree 缺少 `../../wailsjs/go/main/App` 绑定，阻塞点出现在 `src/windows/ShowcaseWindow.vue`，不是本轮 UI 模板调整引入。
- 文案与残留检查
  - 已确认：
    - 不再出现 `Widget Outline`
    - 不再出现 `screen_id`
    - 不再出现 `Updated `

## 潜在影响与回滚方案
- 潜在影响
  - 去掉 `Widget Outline` 后，widget 编辑入口更依赖预览区右键菜单。
  - 编辑页顶部元信息更少，调试性信息展示被弱化。
- 回滚方案
  - 回滚以下文件即可恢复上一个版本：
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/pages/ShowcaseCenter.vue`

## 子Agent执行轨迹
- 本轮未使用子Agent。
- 原因：改动范围小，且两处页面都需要统一拿捏“简化程度”，拆分价值低。
