# 通用变量确认时机与分栏剩余空间

## Summary

共享资源会话不等于共享提交意图。界面应区分草稿、已确认值和在途写入；CSS Grid 的 fr 比例则需考虑轨道触及最小尺寸后的重新分配。

## Lookup Hints

关键词：autoApply、draftVersion、Enter/blur、IME、subscription-before-response、minmax、fractional fr、右侧留白、卡片高度留白、模式切换丢草稿、compact、remount、跨卡草稿、portal host、结构 blur、toolbar 换行遮挡。先检查 VariablePanel/VariableEditor/controller、VariableCards/HostSlot、Workspace 工具栏与 splitGridStyle、CompactCard 及外层 grid 轨道。

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

## 2026-10-19：紧凑布局与状态保留

- 症状与触发：简单下拉框放到分栏里被拉伸到视口高度；将紧凑/分栏做成两套条件渲染后可能丢失输入草稿；窄窗口中卡片超出右边界。
- 根因：铺满布局与内容高度布局目标不同；切换组件树会卸载本地编辑状态；外层 grid 的隐式 auto 列受 toolbar 最小内容宽度影响，仅限制卡片宽度不够。
- 排查：查看 article 实际高度及所有祖先轨道；在切换前后断言同一 input DOM、未提交值及 snapshot 调用次数；分别测宽屏与窄屏末列的 right 边界。
- 解决：两种模式分别保存布局，保持 renderer 实例，用 display:contents/grid order 重新排布 leaf；内容测量决定卡片行跨度，长内容内部滚动。Workspace 显式使用 minmax(0, 1fr) 列，窗口缩放仅影响展示，不写回保存尺寸。
- 证据：真实浏览器下拉框从视口高度缩为约 112px，数组约 247px；1480/850 宽度、保存重载、拖拽排序、尺寸恢复通过；组件测试验证切换后仍为同一输入元素且 snapshot 只调用一次。
- 防回归：固定大小截图不足以覆盖此类问题，必须同时验证容器边界、交互与持久化。CSS order 不会改变 DOM/Tab 顺序；若新增读屏顺序要求，需专门设计，不能仅依赖视觉排序。

## 2026-11-01：跨卡组合仍需保留编辑器身份

- 症状/触发：跨卡移动、单卡转列表或拆出时，即使 reference key 不变，React 父级变化仍会卸载 editor；未通过数字校验的本地输入最容易丢失。点击结构手柄引发 blur 时还可能误提交真实变量。
- 根因：React key 只在同级集合标识身份，不能跨父节点复用实例；共享 controller 也不包含控件本地无效文本。结构编辑和变量写入共享焦点，但不是同一提交意图。
- 快速检查：操作前后比对 `beforeInput === afterInput`、原始无效文本与 mock write 计数，不只比较 controller draft 或截图。区分标题手柄、变量手柄与卡片正文目标。
- 解决：按稳定引用 ID 平铺 portal 内容，卡片 slot 只移动稳定 DOM host；结构 pointerdown 在 capture 阶段标记，避免随后 blur 触发 auto-apply。引用 ID 跟随跨卡/拆出移动，值和草稿不进入 View。
- 防回归：覆盖合并/行排序/跨卡/拆出/切换布局、取消/无效 drop、查看模式正常变量提交；结构路径计数为零，显式值操作计数为一。窄窗口同时核对工具栏换行和内容起始位置，不能只测右边界。
- 证据：[本轮权威审查](../plan/desktop-variable-cards-review.md)及其保留的浏览器日志/截图。真实设备、全部复杂值及读屏顺序未由该模拟验证覆盖。

## Related Docs

- [统一变量规格](../specs/unified-variable-resource.md)
- [Desktop 手动验收请求](../intake/2026-10-09_desktop-variable-manual-acceptance.md)
- [归档与证据](../change/2026-10-09_unified-variable-contract.md)
- [Desktop 工作区规格](../specs/desktop-resource-workspace-v3.md)
- [紧凑布局归档与验收](../change/2026-10-19_desktop-compact-layout.md)
