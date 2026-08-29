# 2026-08-30 Desktop 使用 n 元分割树实现多面板停靠

## Status

Accepted for implementation。

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
