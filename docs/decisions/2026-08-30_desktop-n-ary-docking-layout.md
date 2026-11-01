# 2026-08-30 Desktop 使用 n 元分割树实现多面板停靠

## Status

Accepted for implementation。

2026-10-19 补充：本 ADR 的分割树、边缘停靠及中心投放规则适用于“铺满分栏”。用户已确认并验收紧凑优先及模式切换，见下方补充决策。

## Context

View v2 只为两个 Widget 保存一个全局方向和比例；三个以上退化为扁平响应式网格。因此它不能区分
`A | C | B`、`A | (B / C)` 和 `(A | B | C) / D`，也不能给任意嵌套相邻 pane 保存比例。
用户同时明确拒绝专门的方向/交换按钮，要求拖拽本身表达布局。

## Options Considered

1. 继续扩展 12-column grid：迁移小，但没有可组合拓扑。
2. 纯二叉 split tree：可表达布局，但普通三列产生额外嵌套，插入和比例语义不直观。
3. 项目自有 n-ary split tree：同轴直接插 sibling，交叉轴只包裹目标。
4. 引入 FlexLayout、Golden Layout、Lumino 或其他完整 docking framework：能力成熟，但也接管 tab、
   theme、model、drag lifecycle，超过当前边界。

## Decision

- View v3 使用 `leaf | split` n 元树，`layout_root` 是唯一布局权威。
- Widget 继续保存 Resource/renderer/settings；v3 不保存 canonical `x/y/w/h`。
- panel edge 作用于目标，divider 作用于同级精确索引，workspace edge 作用于 root。
- center drop 首期无效，不引入 stack。
- 使用现有 React、dnd-kit、项目 CSS 与 renderer，不增加 docking 依赖。
- split 使用相对 weights；同轴扁平化、单子节点折叠、leaf/widget 一一对应是持久不变量。
- v1/v2 经 version-specific strict decode 迁移；第一次 v3 写回前保留不可覆盖的 pre-v3 snapshot。

## Consequences

### 2026-10-19 双模式补充

- 新建 View 默认 `compact`；已有未指定模式的 View 继续 `split`，用户可通过工具栏选择。
- `layout_root` 仍是分栏布局的唯一权威。紧凑模式另用 widgets 数组顺序和可选 compact_size，两者共用资源成员，新增/删除同步维护。
- 紧凑模式允许在卡片中心选择前/后插入，背景追加末尾；不受 split 的中心投放限制，也不引入 tab stack。
- 两种模式独立保留几何和顺序，复用 renderer 实例以保留草稿及在途操作。CSS 视觉顺序与 split DOM 顺序可能不同，完整读屏顺序不作已验证承诺。
- View document 继续为 v3；旧版严格解码对新增字段的回滚限制见[本轮归档](../change/2026-10-19_desktop-compact-layout.md)。当前规范以 [View v3](../specs/desktop-resource-workspace-v3.md) 为准。

### 2026-11-01 变量卡片组合补充

- 单卡/多变量列表是两种卡片类型，不增加第三种布局模式；列表占一个 leaf，可在两种布局中混排。
- 区分标题手柄（整卡布局）和变量手柄（引用重组）。变量卡片正文接受引用并合并/排序，不改变整卡拖动的 center-no-stack 决策。
- 采用稳定引用 ID 与平铺 portal host，布局只迁移宿主而不重建变量编辑器，保留无效输入与在途状态；代价是需显式维护宿主生命周期。
- 编辑/查看属于临时会话，不进入 View 文件；仅有效外部 drop 自动进入编辑，结构变动不提交真实变量。
- 保持 View v3 可选字段扩展以读取旧文件，但不承诺旧二进制前向兼容；试运行前备份 `views.json`。
- 不引入 docking/状态库依赖。可访问菜单提供添加/移动/拆出；完整读屏遍历与所有嵌套控件仍需专门人工验证，不能以视觉排序或浏览器模拟代替。
- 请求与归档：[确认范围](../intake/2026-10-31_desktop-variable-cards.md)、[验收/回滚](../change/2026-11-01_desktop-variable-cards.md)。

### 原分栏决策的后果

- 支持任意深度的左右/上下组合和同轴中间插入。
- View 文件升级为 version 3，旧 Desktop 无法直接读取；回滚必须恢复 snapshot。
- 前端从 flat-grid/two-panel 特例转为独立领域层和递归 renderer，初始实现量增加，但后续操作共享一套
  可测试 mutation。
- n-ary split 控制同轴深度；64 leaf/32 depth/127 node 上限保持运行时与校验有界。
- tab stack、floating/cross-window docking 仍需单独决策；如果这些成为核心需求，再重新评估完整 docking
  framework。

## Confidence

High。模型与用户交互一一对应；主要风险集中在 legacy migration、DnD target priority、nested minimums 和
Windows packaged GUI 验证，均有明确测试门。

## Supersedes / Superseded By

- 部分取代 [可扩展 Resource type system 与 Desktop workspace](2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)
  中“响应式网格 Workspace”的布局实现选择。
- 取代 [Desktop Resource Workspace v2](../specs/desktop-resource-workspace-v2.md) 的 View v2 布局契约。

## Related Features

- [Desktop](../features/desktop.md)

## Related Requirements

- [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)

## Related Specs

- [Desktop Resource Workspace v3](../specs/desktop-resource-workspace-v3.md)

## Related Intake

- [Desktop 多面板嵌套停靠](../intake/2026-08-30_desktop-nested-docking-layout.md)
