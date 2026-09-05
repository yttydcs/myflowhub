# Desktop Resource Workspace v3

## Status

Current。取代 [Desktop Resource Workspace v2](desktop-resource-workspace-v2.md) 中 View v2
双面板/三面板网格布局契约，并补充当前 Explorer 的可折叠分区与 Resource path tree 契约。Profile、登录、
Renderer 与安全边界未改变；本文的 v3 仍指 View document version，不要求 UI preference 升级 major。

## Scope

本规格定义 Desktop Explorer 的路径树派生/折叠状态，以及本地 View v3 的递归布局、迁移、校验、拖拽、
调整比例和可访问性。Desktop 仍通过 SDK/Wails JSON boundary 使用 Node-owned Resources，不解析 wire、
不自行裁决权限，也不把 Resource 正文复制进 View。

## Explorer disclosures and Resource path tree

Explorer 保持 master/detail：上区是 authoritative Node tree，下区只消费当前 Node 直接拥有的
`ResourceDescriptor[]`。Resource path tree 是 presentation-only 派生结构，不是 authority、ownership、
permission 或 routing tree。

每个 Resource name 按协议已验证的非空 `/` segment 构建 trie。path node 至少包含 owner、完整 path、当前
segment、可选 `resource` 与有序 `children`；key 必须包含 owner 和完整 path。一个 node 可以同时拥有
`resource` 和 children，例如 `system/config` 与 `system/config/update` 同时存在时只渲染一个 `config`
treeitem，但它同时可选择、拖动、添加和展开。纯 namespace 不声明选择状态，也不暴露 Resource 操作。

树使用一次派生索引和扁平可见行。稳定排序使用 segment 的 numeric-aware locale compare；每行显式包含
depth、parent key、position 和 sibling size。查询匹配完整 name、presentation label、type 与 segment，
匹配结果包含全部祖先并临时视为展开；清空查询恢复持久展开集合，不把搜索产生的展开写回 preference。
默认只展开第一层 path。

Resource tree 使用单选 WAI-ARIA tree 与 roving DOM focus：

- parent treeitem 只有在确有 children 时声明 `aria-expanded`；
- Right 打开 closed parent 或进入首个 child，Left 关闭 open parent 或返回 parent；
- Up/Down、Home/End 在当前可见行移动；`*` 可展开同级 parent；
- Enter/Space 对真实 Resource 执行选择，对纯 namespace 切换展开；
- 焦点与 `aria-selected` 分离，拖动手柄和添加按钮保留独立可访问名称与焦点路径。

Node 与 Resource heading 各是原生 disclosure button，使用 `aria-expanded` 和 `aria-controls`。状态表示为
可选 `collapsed_explorer_pane = node | resource`，因此最多收起一块；另一块填满剩余区域，separator 隐藏。
重新展开后恢复 `explorer_split_ratio`，收起动作不得覆盖比例。受控内容隐藏时不得留在 accessibility tree
或 Tab 顺序中，但组件状态保留。

现有 per-Profile `UIPreferences.version=1` 增加可选 `collapsed_explorer_pane` 与
`expanded_resource_paths`；旧文档无需迁移。字段必须校验 union、非空字符串、单项长度和集合上限，未知或
非法 preference 继续走现有显式默认值与警告路径。无匹配的陈旧 path key 只在派生时忽略，不修改 catalog。

Resource tree 构建必须保持 O(total path segments)，扁平化保持 O(visible rows)，搜索不得在每行重复扫描
整棵已加载树。超过 50 行继续使用现有离屏绘制策略。按需拓扑和独立 catalog 加载由当前
[拓扑发现合同](topology-discovery.md)接管；服务端分页与 DOM windowing 仍为独立后续工作。

## View v3 document

`ViewDocument.version` 固定为 `3`。一个 View 保存：

- `widgets`：Resource reference、renderer、可选非秘密 settings；
- `layout_root`：唯一权威的布局拓扑；
- revision 与创建/更新时间。

v3 `ViewWidget` 不再保存 `x/y/w/h`。这些字段只存在于 v1/v2 解码结构和迁移输入中，不得作为 v3
并行布局真相。

```json
{
  "version": 3,
  "views": [{
    "id": "operations",
    "name": "Operations",
    "revision": 2,
    "widgets": [{
      "id": "catalog",
      "owner_node_id": "1",
      "resource_name": "system/catalog",
      "renderer": "mfh.variable"
    }],
    "layout_root": {
      "kind": "leaf",
      "widget_id": "catalog"
    },
    "created_at_unix_ms": 1,
    "updated_at_unix_ms": 2
  }]
}
```

布局节点是判别联合：

- leaf：`kind=leaf`，只允许非空 `widget_id`；
- split：`kind=split`、`axis=horizontal|vertical`、至少两个 `children`，以及等长
  `weights`。

每个 weight 必须是有限正数，持久化时归一化为总和 `1`。每个 widget ID 在树中恰好出现一次；非空
View 必须有 root，空 View 不得有 root。每个 View 最多 64 个 leaf、总节点不超过 127、深度不超过 32。
同轴 split 不得直接嵌套；删除或移动后必须折叠零/单子节点并合并同轴节点。

## Legacy migration and rollback

加载先读取 version，再使用对应版本的严格结构解码；未知字段、多个 JSON 值和未知版本都拒绝。

- v1/v2 空 View：无 root；
- 单 widget：单 leaf；
- 双 widget：v2 优先使用合法 `layout.direction/split_ratio`，否则从旧 `x/w` 推导横向比例；
- 三个及以上：按 `(y, x, original index)` 排序，同一 `y` 形成 horizontal split，多行形成
  vertical root；列权重来自旧 `w`，行权重来自该行最大 `h`。

迁移对已发布算法生成的网格保持形状；其他合法旧网格保证 widget 不丢失并使用上述确定性近似。迁移只在
内存中发生，用户保存后才写 v3。第一次以 v3 覆盖旧文件前，Desktop 必须创建权限受限、不可静默覆盖的
pre-v3 snapshot；snapshot 或原子提交失败时，旧 `views.json` 保持不变。

旧 Desktop 不能读取 v3。代码回滚后必须先停止 Desktop，再恢复 pre-v3 snapshot；恢复会丢失迁移后新增的
v3-only 嵌套布局。

## Layout operations

所有 UI mutation 通过同一纯布局领域层执行并重新验证不变量：

- direct add：空 View 建 root leaf；非空 View 在整个 root 右侧插入；
- panel edge：左/右使用 horizontal，上/下使用 vertical；
- same-axis panel drop：在目标同级前/后插入；
- cross-axis panel drop：只用新 split 包裹目标；
- divider drop：按父路径和 sibling index 精确插入；
- workspace edge：分割或加入整个 root；
- remove/move：事务性移除、索引修正、插入和规范化；
- self/no-op：不产生新 View revision 或 dirty state。

均匀 split 插入后继续均匀分配；用户已调整过的 split 在 panel-edge 插入时拆分目标 weight，在 divider
插入时拆分其后 child 的 weight。同父 reorder 让 weight 随 child 移动。

## Drag and preview

布局不提供“左右 / 上下 / 交换”按钮。Widget 标题手柄和 Resource drag 是组合布局的入口。

命中优先级固定为：

1. sibling divider；
2. workspace outer edge；
3. panel edge；
4. empty/background workspace。

面板中心是无效投放区，不创建隐藏 stack。拖动预览以 hypothetical layout tree 使用和最终 renderer 相同
的 track 规则渲染，必须显示实际未来 pane rectangle；相同 intent 的连续 pointer move 不重复提交状态。
取消拖动清理 preview，Resource 在 drop 前消失时显式失败或 no-op，不创建损坏 leaf。

拖动手柄保留 dnd-kit keyboard drag；界面通过中文 instructions/live announcements 描述当前目标、顺序和
完成/取消结果。未来 tab stack 必须新增显式 node kind 和独立可访问性契约，不能复用中心 drop 猜测。

## Recursive rendering and resizing

split 使用递归 CSS grid，child tracks 与 separator tracks 交替。每个相邻 child pair 都有 separator：

- horizontal split 使用 vertical separator 和 Left/Right；
- vertical split 使用 horizontal separator 和 Up/Down；
- separator 暴露 `role=separator`、`aria-orientation`、`aria-controls`、当前/最小/最大值和可读
  `aria-valuetext`；
- Shift+Arrow 使用较大步进，Home/End 到边界，Enter 与双击均分相邻 pair。

pointer resize 仅在目标 split 内以 `requestAnimationFrame` 合并本地 preview；pointer release 才提交一次
View，cancel 恢复已持久权重。不得将像素或窗口 resize 结果写回拓扑。

初始可用最小值为 leaf 宽 200 px、高 120 px，并由嵌套树递归聚合；实施可在 GUI 验证中微调常量，但不得
低于 Resource renderer 可操作区域。空间不足时 Workspace 滚动，不能静默重排或覆盖用户布局。

## Validation and failure behavior

Go boundary 必须拒绝：

- layout_root 与 widgets 空/非空关系不匹配；
- unknown kind/axis 或 node kind 间字段混用；
- 少于两个 child、weights 数量不等、非有限/非正/未归一化 weight；
- duplicate/missing/unused widget leaf；
- 同轴直接嵌套、深度/节点/leaf 超限；
- legacy grid、widget identity、settings、revision 或 JSON 本身非法。

加载/迁移/保存错误必须包含可操作上下文，不得静默回退为 flat grid，也不得覆盖原文件。已有 revision
conflict、每 Profile 隔离、32 View 上限、原子 temp/sync/backup/rename 和 corrupt-store 保护继续生效。

## Performance

- 领域操作和验证是 O(layout nodes)，上限 127 nodes；
- UI 递归 DOM 是 O(nodes)；
- drag preview 只在 DockIntent 改变时重算；
- resize pointer move 只更新当前 split 的本地 draft，最多每 animation frame 一次；
- 64-leaf synthetic render/operation 必须在前端单测中保持有界，不新增布局依赖或 N² 扫描。

## Acceptance

- Node/Resource heading 可通过指针、Enter/Space 收起和恢复，最多一个收起；另一块满高且原比例可恢复；
- Resource name 任意深度、纯 namespace、普通 leaf 和 Resource/parent 混合项均有正确树语义；
- Resource 搜索保留祖先、临时展开且清空后恢复；选择、preview、拖拽、添加与类型图标无回归；
- Resource path 展开与 Explorer 收起状态按 Profile 保存，旧 preference 保持可读；
- 拖拽生成 `A | C | B`、`A | (B / C)`、`(A | B | C) / D`；
- 三个以上 pane 可任意插入、移动、删除和调整相邻比例；
- divider、workspace edge、panel edge preview 与实际 drop 结果一致；
- no layout buttons、center no stack；
- pointer/keyboard resize 平滑，release-only commit；
- v1/v2 保留所有 widgets 迁移，v3 保存/重开/重启一致；
- malformed/unknown View 不覆盖，pre-v3 rollback snapshot 可用；
- light/dark、narrow overflow、detached/forbidden 和 Profile/View 行为无回归。

## Related docs

- [Desktop schema rendering](desktop-schema-rendering.md)
- [Desktop feature](../features/desktop.md)
- [Desktop requirements](../requirements/desktop-resource-workspace.md)
- [n-ary docking decision](../decisions/2026-08-30_desktop-n-ary-docking-layout.md)
- [Explorer Resource tree request](../intake/2026-08-30_desktop-explorer-collapsible-resource-tree.md)
- [request and research](../intake/2026-08-30_desktop-nested-docking-layout.md)
- [Explorer Resource tree change](../change/2026-08-30_desktop-explorer-collapsible-resource-tree.md)
