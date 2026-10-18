# Desktop 紧凑布局与视图模式切换

## Source and status

来源：本会话继统一 Variable 手动验收后的布局讨论；2026-10-09 实施，2026-10-19 重启联调、用户验收并提交。用户原话：“好的，紧凑优先，然后这个视图要能调整布局，比如现在这样的优先占满，或者你说的紧凑卡片网格”。用户随后确认“验收没问题”，要求提交并 push；代码已提交为 `e486e45`。

## Confirmed scope

- 解决单个下拉框被拉伸到整块工作区高度、产生大量留白的问题。
- 新视图紧凑优先；每个视图可在紧凑卡片与原有铺满分栏之间切换。
- 已实施并验收的交互包括卡片排序、从资源树拖入、宽高调整及恢复自动尺寸；保存后恢复模式和布局。
- 两种布局分别保留，模式切换不重建正在编辑的变量组件。既有未指定模式的视图保留原布局。
- 使用后台浏览器联调，保留用户输入设备控制权。本轮不改变资源协议、权限模型或 Flow。

## Related

- [Desktop](../features/desktop.md)、[工作区需求](../requirements/desktop-resource-workspace.md)、[View v3](../specs/desktop-resource-workspace-v3.md)
- [验收与审查记录](../plan/review_2026-10-19_desktop-compact-layout.md)、[变更归档](../change/2026-10-19_desktop-compact-layout.md)
