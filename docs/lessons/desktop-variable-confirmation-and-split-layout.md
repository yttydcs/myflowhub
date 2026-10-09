# 通用变量确认时机与分栏剩余空间

## Summary

共享资源会话不等于共享提交意图。界面应区分草稿、已确认值和在途写入；CSS Grid 的 fr 比例则需考虑轨道触及最小尺寸后的重新分配。

## Lookup Hints

关键词：autoApply、draftVersion、Enter/blur、IME、subscription-before-response、minmax、fractional fr、右侧留白。先检查 VariablePanel/VariableEditor/controller、ui/slider 和 Workspace.splitGridStyle。

## Symptoms

数字输入闪现“未应用”；开关在远端确认前变色；View 提交 Inspector 草稿；四列缩窄后最右侧出现空白；只读嵌套字段逐层挤到右侧。

## Impact

可能误报操作成功、提交其他界面的未确认意图，或让有空间的工作区仍挤压控件。

## Trigger Conditions

多个面板共用资源会话；Enter 后紧随 blur；订阅推送早于 write 返回；不等比分栏触及最小宽高；复用适合标量的键值行样式展示递归结构。

## Root Cause

草稿和确认值合并会抹去异步边界。Radix 滑块键盘 commit 先于 change，不能直接假设事件顺序。CSS fr 之和低于 1 时只消费部分剩余空间；原来归一化为总和 1 的比例，在部分轨道固定到 min 后会触发此规则。

## Investigation Trail

后台浏览器延迟 write 响应 700ms，验证加载态与确认值；分别测试数字 Enter/blur 和订阅先到。四列从 1280 缩到 1200 的窗口真实复现 80px 留白，再覆盖横纵混合布局和增删组件。

## Resolution

提交意图绑定 controller、instance 和 draft generation；字段显式触发提交，去重并取消失效意图；离散控件保留确认效果，控件内显示加载态，错误显式恢复。滑块 commit 延后到 change 后消费。分栏渲染将权重同比放大至最小 fr 为 1，持久化仍保留原比例；拖动预览绑定来源节点，结构变更即失效。只读复杂值全宽分组，保留深度和数量上限。

## Prevention / Guardrails

测试失败不循环重试、输入法确认、Enter+blur、组件释放、Profile 切换、共享草稿归属和推送先到。布局检查末项与容器末边差及子项最小尺寸，覆盖窄宽、高低、横纵嵌套和增删。不要用固定容器尺寸的截图代替这些断言。

## Related Docs

- [统一变量规格](../specs/unified-variable-resource.md)
- [Desktop 手动验收请求](../intake/2026-10-09_desktop-variable-manual-acceptance.md)
- [归档与证据](../change/2026-10-09_unified-variable-contract.md)
