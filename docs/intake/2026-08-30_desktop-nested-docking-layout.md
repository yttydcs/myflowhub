# Desktop 多面板嵌套停靠讨论记录

## 来源

- 日期：2026-08-30
- 阶段：`$m-discuss`
- 触发问题：三个资源面板排成一行后，无法把第三个放到某个面板下方，也无法插入两个既有面板之间；拖拽只能得到固定网格结果。
- 用户要求：充分规划，并参考成熟桌面/IDE 停靠方案；不要增加专门的布局按钮，布局应由拖拽完成。

## 现状与根因

当前 View v2 将组件保存为扁平 `widgets` 列表，只为双面板保存一个全局方向和一个分割比例。前端只对恰好两个组件启用方向与分隔条；三个及以上组件会进入固定行网格，因此模型无法表达以下不同拓扑：

- `A | C | B`：把 C 插入 A 与 B 中间。
- `A | (B / C)`：只把 C 放到 B 下方。
- `(A | B | C) / D`：把 D 放到整行下方并横跨工作区。

因此，继续扩大网格拖拽命中区只能缓解表象，不能形成可持久化、可调整比例的任意嵌套布局。

## 目标

- 三个及更多面板可以通过直接拖拽形成任意横向、纵向组合。
- 可以插入任意两个同级面板之间，而不是只能追加到末尾。
- 面板边缘、同级分隔线和工作区外缘表达不同的放置语义，并显示与最终结果一致的预览。
- 任意相邻面板之间均可连续调整比例；拖动过程平滑，释放时持久化。
- 布局随 View 保存、重启恢复，并可从现有 View v1/v2 无损迁移。
- 保留现有资源渲染器、资源拖拽入口、主题和组件体系，不为布局引入不必要的外部依赖。
- 分隔条继续满足键盘和辅助技术访问要求。

## 非目标（首期）

- 中心投放生成标签栈。
- 浮动窗口、跨窗口拖拽或像素坐标画布。
- 自动布局预设按钮或单独的“左右/上下/交换”控件。
- 借本次改造重写资源渲染器、连接模型或品牌资产。

## 方案比较

| 方案 | 能否表达任意嵌套 | 插入中间 | 迁移与集成 | 结论 |
| --- | --- | --- | --- | --- |
| 继续补丁扁平网格 | 弱 | 只能做视觉排序 | 低成本，但比例、持久化和局部上下分割仍含糊 | 拒绝 |
| 纯二叉分割树 | 强 | 可以，但同向三列会形成不必要的嵌套层级 | 中等；实现简单，比例语义不够直观 | 可行备选 |
| 项目内 n 元分割树 | 强 | 同向父节点直接按索引插入 | 中等偏高；与现有 React/dnd-kit 兼容 | 推荐 |
| 引入完整 Docking 库 | 强 | 通常支持 | 需要接管布局模型、拖拽、主题和生命周期，常伴随标签栈概念 | 当前不推荐 |

## 外部调研摘要

- [VS Code Custom Layout](https://code.visualstudio.com/docs/configure/custom-layout) 与 [User Interface](https://code.visualstudio.com/docs/editing/userinterface) 使用面板四边拖拽建立横向或纵向编辑器组，并通过 sash 调整大小；这验证了“目标面板边缘决定局部分割”的交互惯例。
- [Golden Layout Structure](https://golden-layout.github.io/golden-layout/structure/) 将布局建模为行、列与叶节点组成的树；[Sizing components](https://golden-layout.github.io/golden-layout/sizing-components/) 使用相对尺寸分配空间。
- [React Mosaic](https://github.com/nomcopter/react-mosaic) 的近期模型支持一个 split 节点包含多个 children 和对应比例。由此可以推断，同向插入第三个兄弟节点时，n 元结构比强制二叉嵌套更直接。
- [FlexLayout](https://github.com/caplin/FlexLayout/blob/master/README.md) 使用可序列化的 Row/Tabset/Tab 树与相对权重，功能成熟，但其标签页和模型接管范围超出当前需求。
- [Lumino](https://github.com/jupyterlab/lumino) 提供桌面式 DockPanel，但属于更大的 widget 工具体系，不适合仅为本功能引入。
- [WAI-ARIA Window Splitter Pattern](https://www.w3.org/WAI/ARIA/apg/patterns/windowsplitter/) 要求可聚焦的 separator、方向和值属性及方向键调整。该 APG 模式仍标注为待完整评审，实现时应结合浏览器/辅助技术实测。

## 推荐模型

View v3 使用布局树作为唯一权威拓扑；组件列表继续保存资源身份、渲染参数等内容。

```ts
type LayoutNode =
  | { type: "leaf"; widgetId: string }
  | {
      type: "split";
      axis: "horizontal" | "vertical";
      children: LayoutNode[];
      weights: number[];
    };
```

采用 n 元 split，而不是强制二叉树：

- 同轴投放：若目标父节点方向相同，直接在目标前后插入同级 children，可自然得到 `A | C | B`。
- 交叉轴投放：若投放方向与父节点不同，用新的 split 包裹目标叶节点，可得到 `A | (B / C)`。
- 工作区外缘投放：分割整个根节点，可得到 `(A | B | C) / D`。

## 拖拽语义

命中优先级建议为：同级分隔线 > 工作区外缘 > 面板边缘 > 工作区空白。

1. 面板左/右边缘：在目标旁建立或加入横向 split。
2. 面板上/下边缘：在目标旁建立或加入纵向 split。
3. 同级分隔线：按该精确索引插入，解决“第三个放到第一、第二之间”。
4. 工作区外缘：分割整个根布局，让新面板横跨整行或整列。
5. 中心区域：首期不接受投放，避免无意引入标签栈或覆盖语义。

拖拽预览必须展示实际将占据的矩形区域，并通过 live region 描述“放到 X 左侧/下方/插入第 N 个位置”。不新增专门布局按钮。

## 比例调整与规范化

- 每对相邻 children 之间渲染分隔条，只改变相邻两项权重，其他兄弟节点保持不变。
- 拖动期间用 pointer capture 配合 CSS/`requestAnimationFrame` 更新预览；释放时一次性提交 store 和持久化，避免逐事件写状态造成卡顿。
- 如果父节点各项仍为均分，插入后重新均分；如果用户已调整比例，则拆分目标项权重并保留其他项。
- 删除或移动后折叠零/单子节点 split；合并同轴嵌套 split；权重必须为有限正数并归一化。
- 每个 widget ID 在树中恰好出现一次。沿用现有最多 64 个组件限制，并为树深设置防御性上限。
- 窗口过小时保留用户拓扑和权重，工作区允许滚动；不静默重排并覆盖已保存布局。实际最小宽高在计划/实现阶段依据资源渲染内容确定。
- 分隔条提供 `role="separator"`、方向、当前/最小/最大值及方向键调整。

## 迁移边界

建议将持久化格式升级为 View v3：

- 0 个组件：无布局根节点。
- 1 个组件：单 leaf。
- 2 个组件：根据 v2 的方向和比例生成 split。
- 3 个及以上：按旧 `x/y/w/h` 确定行，同一行生成横向 split；多行再由纵向根 split 组合。
- v3 由布局树决定渲染。旧坐标字段可保留一个迁移周期作为只读兼容输入，不再作为权威状态。
- Go 侧递归严格校验节点类型、深度、叶节点唯一性、权重数量/范围及组件集合一致性；损坏文件沿用现有安全拒绝与恢复策略。

## 测试重点

- 三面板同轴中间插入、局部交叉轴嵌套、根外缘横跨投放。
- 移动、删除后的折叠、同轴扁平化与权重保持。
- 分隔线、工作区外缘和面板边缘重叠时的命中优先级及真实预览。
- 嵌套分隔条的鼠标、键盘与辅助技术行为；连续拖动帧稳定性及仅释放时持久化。
- v1/v2 到 v3 迁移、重复叶、非法权重、过深树等损坏输入。
- 保存、重启恢复；浅色/深色、窄窗口及 6–8 个面板的 GUI 回归。

## 文档影响

用户确认推荐方向后，`$m-plan` 应同步规划：

- 更新 Desktop 工作区 requirements，替换“三个及以上使用固定响应式网格”的旧约束。
- 新增或升级 View v3 技术 spec，定义树结构、迁移、校验和拖拽状态机。
- 为持久化模型升级及“不引入完整 Docking 库”的边界记录 ADR。
- 在 feature 文档更新用户可见的多面板拖拽、调整比例和恢复行为。

本 intake 仅保存来源证据与讨论结论，不在用户确认前改写稳定 requirements/spec/decision。

## 工作树状态与交接门

本轮按 exploratory `$m-discuss` 处理，未创建实现工作树。主检出存在用户未提交改动，且当前三面板代码依赖上一轮尚未独立归档的停靠改动；进入 `$m-plan` 前必须先识别并保留这些改动，再在 `D:\project\MyFlowHub3\worktrees\` 下创建独立工作树。

讨论交接仍需用户确认：是否采用“项目内 n 元分割树 + 面板边缘局部分割 + 工作区外缘分割根布局 + 首期不做中心标签栈”的推荐方向。

## Routed Docs

- [Desktop 当前功能](../features/desktop.md)
- [Desktop 资源工作区需求](../requirements/desktop-resource-workspace.md)
- [Desktop Resource Workspace v3](../specs/desktop-resource-workspace-v3.md)
- [Desktop n 元多面板停靠决策](../decisions/2026-08-30_desktop-n-ary-docking-layout.md)
- Active workflow plan: `../../plan.md`

用户随后显式进入 `$m-plan` 并以 `$m-execute` 批准实现
`DOC01, VIEW01, LAYOUT01, RENDER01, DOCK01, QA01`。
