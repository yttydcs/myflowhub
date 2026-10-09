# Desktop 通用变量手动验收追加请求

## Source and status

来源：统一 Variable 当前会话的连续手动验收；以用户最后一次明确修正为准。已实施，归入 DESK01 / QA01，未新增网络模型对象或节点专用展示。

## Confirmed requests

- 用户要求“其他的各种组件都要改，核心要义是简洁好看”，优先现成组件，随后明确使用 shadcn；统一输入、选择、开关、滑块等外观。
- 隐藏顶部技术信息；订阅用圆点，放在变量标题右侧。移除可写变量重复的当前值、根结构的重复标题和折叠开关。
- View 隐藏分隔线下面的详情/刷新/重置/应用，控件占满卡片宽度；Inspector 保留手动编辑入口。
- 用户先要求 View 自动应用，随后明确“给输入控件添加加载态……应用完成之后才正式变成对应的效果……编辑框就回车之后应用，或者失去焦点了之后应用”。最终行为取代中间的停顿提交：离散控件立即提交，滑块结束交互提交，单行 Enter/blur，多行 blur。
- 走查所有类型、嵌套对象/数组、只读数据与订阅拒绝；修复标题栏边线、窄栏嵌套挤压和分栏最小尺寸导致的右侧留白。
- 使用后台浏览器联调，不占用用户桌面输入。保持权限裁决；audit/catalog 订阅拒绝不通过扩大权限掩盖。

## Related

- [统一变量需求](../requirements/unified-variable-resource.md)
- [Desktop 功能](../features/desktop.md)
- [统一变量规格](../specs/unified-variable-resource.md)
- [归档](../change/2026-10-09_unified-variable-contract.md)
