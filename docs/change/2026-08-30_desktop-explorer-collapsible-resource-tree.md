# 2026-08-30 Desktop Explorer 可折叠分区与 Resource path tree

## 变更背景 / 目标

Desktop Explorer 原本把当前 Node 的 Resources 按首段分组为平铺列表，无法表达任意深度路径，也不能同时
表达“这个 path 本身是 Resource、同时还是父节点”的结构。Node 与 Resource 两区虽然可以调整比例，但不能
通过标题收起。本次将 Resource 展示升级为数据驱动的 path tree，并让两个分区成为可访问的 disclosure，
同时保留预览、拖放、添加、独立滚动、固定 Profile footer 和现有 View 行为。

## 具体变更内容

- 新增纯 TypeScript Resource tree 领域层，以 owner Node ID 与完整 Resource name 生成稳定 key，构建任意深度
  trie/index，并派生扁平可见行。
- 支持纯 namespace、普通 leaf、Resource/parent hybrid、不同 owner 隔离、numeric ordering、搜索祖先临时展开
  与默认首层展开；删除旧 `groupResources` 首段分组 presentation model。
- Resource 区改为单选 WAI-ARIA tree，使用 roving focus，支持方向键、Home/End、Enter/Space 和 `*`；焦点、
  选择、展开、拖动手柄与添加动作相互独立。
- Node/Resource heading 改为原生 disclosure button。最多收起一个 pane；收起时 separator 不存在，另一 pane
  占满高度；重新展开保留原 split ratio。
- `UIPreferences.version=1` 以 optional fields 增加 `expanded_resource_paths` 与
  `collapsed_explorer_pane`，保持旧文档兼容、损坏输入显式回退和 Profile 隔离。
- 更新矿物蓝/石墨灰的紧凑树行、缩进、折叠布局与浅/深色样式；从最终源码重新生成 production dist。

## Docs root

- `docs/`，位于 canonical `MyFlowHub` monorepo；本地版本化，无 remote、push、release 或 publication。

## Intake impact

- updated — 新增并索引本次可折叠 Explorer 与 Resource path tree 原始需求、方案比较和约束。

## Feature impact

- updated — `docs/features/desktop.md` 现在描述 disclosure、完整 Resource path tree、hybrid item、独立搜索/滚动
  与 per-Profile 展开状态。

## Requirements impact

- updated — `docs/requirements/desktop-resource-workspace.md` 增加树键盘语义、折叠/满高、性能复杂度、偏好兼容
  和真实 GUI 验收。

## Specs impact

- updated — `docs/specs/desktop-resource-workspace-v3.md` 增加 trie/index、扁平可见行、hybrid、搜索、disclosure
  和 preference contract。

## Decision impact

- none — 这是既有 Node ownership、Resource relative path 与 Desktop Explorer 边界内的可逆 UI/派生模型，
  没有新增难以逆转的架构决策。

## Lessons impact

- none — 未发现新的产品缺陷模式。Windows EOL/generated drift、前端 PowerShell 预检和新 worktree Wails bindings
  已由既有 lessons 覆盖；WebView accessibility element index 不稳定只影响本次测试工具操作，并未改变产品修复方式。

## Related intake

- [Desktop Explorer 可折叠分区与 Resource path tree](../intake/2026-08-30_desktop-explorer-collapsible-resource-tree.md)

## Related features

- [Desktop](../features/desktop.md)

## Related requirements

- [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)

## Related specs

- [Desktop Resource Workspace v3](../specs/desktop-resource-workspace-v3.md)

## Related decisions

- [权威 Node tree 与可插拔 Link](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- [可扩展 Resource type system 与 Desktop workspace](../decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)

## Related lessons

- [Frontend 与 PowerShell 预检](../lessons/frontend-and-powershell-preflight.md)
- [新 worktree 缺 Wails bindings](../lessons/frontend-worktree-wailsjs-missing.md)
- [Windows clean checkout、EOL 与 generated drift](../lessons/windows-clean-checkout-eol-and-generated-drift.md)

## 对应 plan.md 任务映射

| Task | 结果 |
| --- | --- |
| DOC01 | intake、feature、requirement、spec 与索引保持可追溯一致。 |
| TREE01 | Resource trie/index、stable key、任意深度、hybrid、搜索祖先和扁平可见行完成。 |
| UI01 | Resource WAI-ARIA tree、类型图标、预览、拖拽、添加与两个 section disclosure 完成。 |
| PREF01 | collapse/resource expansion per-Profile preference、split ratio 保留和损坏回退完成。 |
| QA01 | 自动化、性能、Go、production build、Wails package 与真实 GUI 验证通过。 |
| ARC01 | change/plan/evidence 归档、索引和本地 closeout 由本阶段完成。 |

## 经验 / 教训摘要

- Resource path 不是文件系统：同一 path 可以同时拥有 Resource payload 和 children，不能假设 folder 不可选、
  leaf 一定没有子项。
- 搜索只应改变派生可见行与临时展开，不能污染用户持久化的 expanded set。
- 分区收起必须从布局与焦点路径移除 content/separator，不能只用负尺寸或视觉遮挡。
- 大树先统一成迭代扁平可见行；现有 API 无分页时，不应为伪 lazy loading 维护第二套树真相。

## 可复用排查线索

- 症状：`system/config/update` 不可见。触发条件：父 path `system/config` 本身也是 Resource。关键词：
  `hybrid Resource`、`resource-path`、`aria-expanded`。快速检查：确认 row 同时有 Resource actions 和 children。
- 症状：搜索清空后树仍保持临时展开。关键词：`deferredResourceQuery`、`expanded_resource_paths`。快速检查：
  搜索路径必须由 `visibleKeys` 派生，不能写回 preference。
- 症状：收起后键盘仍能进入隐藏内容或 separator。关键词：`collapsed_explorer_pane`、`hidden`、`separator`。
  快速检查：DOM 中不应存在受控 tree/search/separator 的可聚焦路径。
- 症状：大型 catalog 卡顿。关键词：`flattenResourceRows`、`10k`、`content-visibility`。快速检查：运行 10,000
  Resource fixture，并确认没有递归组件状态或热点 O(n²)。

## 关键设计决策与权衡

- 选择 owner/full-path trie，而不是 type-first tree、Node/Resource 混合树或首段 accordion；完整 identity 保持不变，
  presentation 可表达任意深度，但真正 server pagination/windowing 仍需后续 API 阶段。
- 保持 `UIPreferences.version=1` optional extension，避免为可安全缺省的 UI state 强制迁移；代价是读取边界必须严格
  校验 union、数组数量和 key 长度。
- 使用扁平行 + `aria-level/posinset/setsize`，避免递归组件各自保存展开状态；这为未来 DOM windowing 保留单一入口。
- 不引入新依赖；沿用 React、dnd-kit、Radix/shadcn primitives 和现有 CSS token。

## 测试与验证方式 / 结果

- `npm test`：9 个测试文件、53 个测试通过。
- 10,000 Resource synthetic fixture：578 ms，低于 750 ms 门槛。
- `$env:GOWORK='off'; go test ./... -count=1`：全仓通过。
- `npm run build`：TypeScript `--noEmit` 与 Vite production build 通过。
- Wails CLI v2.11.0 `wails build -clean -platform windows/amd64`：production build 通过；最终 executable
  SHA-256 为 `954D84C0BEDA843E1E6F01FE271CB8241CA19A157D2F4E5EB20538428182B456`。
- 真实 packaged GUI：Node/Resource 互斥折叠与满高、separator 移除、固定 footer、独立滚动、
  `system/config → system/config/update` hybrid、Tree `ArrowLeft`、浅/深色、重启 preference、预览/拖拽/添加均通过。
- [m-test evidence](../../artifacts/m-test/desktop-explorer-resource-tree/README.md)。
- `git diff --check` 通过；测试结束后恢复原四 Widget View，未保存测试 Widget。

## 潜在影响

- 10,000 Resource 的客户端派生已验证，但 server pagination、按需 catalog 和真正 DOM windowing 属于延期的
  `LAZY01`，当前 API 不支持。
- Resource tree 展开 key 包含 owner/full path；陈旧 key 只会在派生时无匹配，不改变 catalog、selection 或 View。
- 品牌图标、Agent Gateway、Metrics 独立产品、push、release 和 publication 均不属于本工作流。

## 回滚方案

- 回退 Resource tree 模块、Explorer/SplitPane/App/preferences、测试、样式和 regenerated dist；恢复旧 grouped list。
- optional preference fields 可被旧版本忽略，不需要 Profile 数据迁移或删除。
- 不修改用户 Profile、Resource、View、permit 或 Hub 数据。

## 子Agent执行轨迹

- 无。当前 host policy 与用户授权不允许主动委派，且 Tree/UI/preference 焦点契约共享写集；实现、测试、文档和
  GUI 验收由主 Agent 顺序完成。

## Closeout status

- Archive candidate complete；等待从 canonical control-plane 执行本地合并、恢复无关 dirty state 并清理 worktree。
- 无 remote；不推送、不发布。

