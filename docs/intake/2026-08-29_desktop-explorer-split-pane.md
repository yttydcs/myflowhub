# Desktop Explorer Node/Resource 上下分区

## 来源

用户在真实 Desktop 界面中发现：同一列同时展示 Node 与大量 Resource 时，Resource 行会淹没 Node
层级；左侧列表虽可滚动，但 Profile/连接状态入口和滚动边界需要保持稳定。用户最终选择上下布局，并要求
Node 列表与 Resource 列表的高度可以调整。

## 目标

- 上半区只展示 authoritative Node tree，下半区只展示当前 Node 的直接 Resources；
- Node 与 Resource 使用独立搜索和独立滚动区域，固定保留左下 Profile/连接状态入口；
- Resource 按完整名称的首段分组，但分组只用于显示，不改变 `(owner_node_id, full name)` 身份；
- 分隔条支持指针、键盘和双击复位，比例有边界并按 Profile 保存；
- 保留深层 Node 聚焦、breadcrumb、Resource preview、拖放/按钮添加和 View 保存能力；
- 不改变 Hub 权限裁决、Resource 协议或目录数据模型。

## 非目标与分期边界

- 不在 Explorer 中推断权限或把可见 Resource 声明为可用；真实操作仍可能返回 `forbidden`；
- 不实现服务端 topology/catalog 分页、按需加载或新的 Resource hierarchy；
- 不处理品牌图标、发布或推送；
- 真正 DOM windowing 继续作为 topology/catalog API 支持分页后的独立工作，本轮使用一次索引、扁平
  Node 可见行和浏览器离屏绘制优化。

## 已确认方案

采用 master/detail Explorer：上方 WAI-ARIA Node tree 控制当前 owner，下方分组 Resource list 展示其
直接资源。水平语义分隔条默认给 Node 区 35% 高度，限制在 20%–80%，并通过现有版本化、非秘密、
按 Profile 隔离的 UI preference 持久化。

## Stable Docs Impact

- Feature impact: clarify `docs/features/desktop.md`。
- Requirements impact: clarify `docs/requirements/desktop-resource-workspace.md`。
- Specs impact: clarify `docs/specs/desktop-resource-workspace-v2.md`。
- Decision impact: none；这是既有 authority/ownership 与 Explorer 决策内的交互实现。
- Lessons impact: none；本轮没有形成独立的可复用事故知识。
