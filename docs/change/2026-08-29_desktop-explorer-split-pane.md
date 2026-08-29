# Desktop Explorer 节点与资源上下分区

## 变更背景 / 目标

旧版 Explorer 把 Node 与其 Resources 混排在同一棵树中。资源较多时，节点关系、当前资源归属与可操作入口互相挤占；侧栏还缺少可调的上下空间分配。用户确认采用上下分区：上方只展示任意深度的 Node 权威树，下方只展示当前 Node 的 Resources，两区高度可调，并保持 Profile/连接状态固定在左下角。

本轮只实现 Desktop 界面与既有生产数据模型的映射。项目图标由独立任务处理；没有设计、替换、派生、定稿或归档品牌资产。

## 文档治理影响

- Docs root: `docs/`
- Intake impact: added
- Feature impact: clarified
- Requirements impact: clarified
- Specs impact: clarified
- Decision impact: none
- Lessons impact: none
- Related intake: [原始 Desktop 工作区重设计](../intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)、[本轮上下分区确认](../intake/2026-08-29_desktop-explorer-split-pane.md)
- Related feature: [Desktop](../features/desktop.md)
- Related requirement: [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)
- Related spec: [Desktop Resource Workspace v2](../specs/desktop-resource-workspace-v2.md)
- Related decision: [可扩展资源类型系统与 Desktop 工作区](../decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)；本轮是既有 Node 归属与 Explorer 决策的有界实现，不新增 ADR。
- Related lessons: [Frontend 与 PowerShell 预检](../lessons/frontend-and-powershell-preflight.md)、[Windows clean checkout 与生成物漂移](../lessons/windows-clean-checkout-eol-and-generated-drift.md)、[新 worktree 缺少 WailsJS](../lessons/frontend-worktree-wailsjs-missing.md)

## 具体变更内容

- Explorer 上区改为数据驱动的 Node-only WAI-ARIA tree，保留任意深度、祖先匹配、展开状态、深层聚焦、面包屑与返回。
- 下区只列出当前 Node 直接拥有的 Resources，并按资源类型分组；Node 与 Resource 搜索、选择、预览、拖拽和“添加到工作区”语义分离。
- 新增通用上下分区组件：支持 pointer capture、取消回滚、双击复位、方向键、`Shift` 步进和 `Home`/`End` 边界操作。
- 分隔比例受上下最小像素高度约束，并以可选 `explorer_split_ratio` 按 Profile 持久化；旧偏好文档无需迁移，非法值会被拒绝。
- 修正侧栏/ScrollArea containment，使 Node 区和 Resource 区独立滚动，左下 Profile 与连接状态不进入滚动区。
- 保留既有 Workspace/View 数据模型、资源预览与保存路径；未把原型演示数据复制进生产模型。
- 同步更新 React/Vitest 覆盖、canonical Desktop 文档与 tracked production `dist/**`；没有新增依赖、协议、后端权限或持久化 schema。

## 任务映射

| Task | 结果 |
| --- | --- |
| EX01 | 完成 Node-only 树、当前 Node Resource 分组列表、搜索/选择/预览/拖拽/添加与深层树行为。 |
| EX02 | 完成可访问的有界分隔条、按 Profile 持久化、独立滚动和固定页脚。 |
| DOC01 | 完成 focused intake 与 feature/requirement/spec 澄清；decision/lesson 均无新增。 |
| QA01 | 完成 27 个前端测试、全仓 Go 测试、性能预算、生产构建和真实 Wails GUI 验收。 |
| ARC01 | 完成 change、截图、plan 快照、索引、中文提交、基于最新 `master` 的 rebase、快进合并与 worktree/branch 清理。 |

## 关键设计决策与权衡

1. Node 树只表达权威/归属结构，Resource 使用当前 Node 的分组列表表达，避免在同一 ARIA tree 中混合两种实体与键盘语义。
2. Resource 列表只展示当前 Node 直接拥有的条目，不从前端推断跨 Node 权限或有效授权状态。
3. 分隔比例持久化为可选 UI preference，不进入 Go 配置或业务数据；Profile 隔离沿用既有 preference key 机制。
4. 上下最小高度以像素约束，持久值以比例表达，使小窗口仍可操作，同时在窗口尺寸变化后保持用户意图。
5. 大树继续使用扁平可见行与单次 ownership index，2,000 Node / 10,000 Resource 已受性能测试约束；服务端分页、按需加载和真正 DOM windowing 仍是明确分期边界。
6. 保留项目现有组件和依赖，不为一个分隔条引入新的布局或虚拟化库。

## 经验 / 教训摘要

上下分区解决的是信息模型和滚动边界，而不仅是 CSS 排版：Node 必须保持 tree 语义，Resource 必须保持 list/action 语义，当前 Node 则是两者唯一的选择桥梁。分隔比例只有在提交时持久化、读取时验证、窗口变化时重新按像素边界 clamp，才能兼顾可用性与兼容性。

本轮没有形成独立且可复用的新故障模式，因此不新增 lesson。Wails binding、Windows EOL/generated drift 与 packaged GUI 分层验证已由现有 lessons 覆盖。

## 可复用排查线索

- 症状：Resource 仍出现在 `role=treeitem` 中。触发条件：复用了旧混合树派生。关键词：`NodeExplorerRow`、`aria-level`、Node-only tree。快速检查：tree 的可见行与 ARIA 元数据是否只来自 Node。
- 症状：切换 Node 后仍看到前一个 Node 的资源。触发条件：current Node reconciliation 或 ownership index 过期。快速检查：同名 Resource fixture 是否按 owner Node 隔离。
- 症状：拖动后下区消失或小窗口无法复位。触发条件：只保存比例、未按容器像素最小值 clamp。关键词：`explorer_split_ratio`、separator、min pane。快速检查：方向键 `Home`/`End` 与双击复位。
- 症状：左下 Profile/连接状态随资源列表滚走。触发条件：侧栏 grid/flex child 缺少 `min-height: 0` 或 footer 位于 ScrollArea 内。快速检查：只滚动 Resource 区并观察 footer 与 Node 区。
- 症状：新 worktree 构建提示 `wailsjs` 缺失。快速检查：按 canonical generate 入口生成绑定，不从其他工作副本复制。

## 测试与验证

- `npm test`: 6 个文件、27 个测试通过。
- `npm run build`: TypeScript 与 Vite production build 通过，tracked `dist/**` 已刷新。
- `GOWORK=off go test ./... -count=1`: 全部 package 与 integration tests 通过。
- `git diff --check`: 通过；只有 Windows checkout 的 LF/CRLF 提示，无 whitespace error。
- 2,000 Node / 10,000 Resource 派生基准：通过 750 ms 测试预算。
- `GOWORK=off wails build -clean -platform windows/amd64`: 通过，生成生产 `myflowhub-desktop.exe`。
- 真实链路：worktree executable 连接隔离 default-deny Hub `127.0.0.1:7331`，加载 2 Nodes / 25 Resources，切换 Node，预览并把 Node 2 的 `system/catalog` 加入临时未保存 Workspace。
- GUI 交互：pointer 拖动、键盘 `End`、双击复位、Resource/View Tab 往返、独立 Resource 滚动与固定 footer 均通过。
- 持久化与主题：非默认分隔比例及深色主题跨进程重启保持；浅色/深色视觉审查通过。
- 测试实例未保存 View；测试进程已停止。原主检出窗口重新连接，原有未保存 View 保持不变。
- 窗口证据：[深色主题与持久化分隔比例](verification/2026-08-29_desktop-explorer-split-pane-dark-persisted.jpg)、[浅色主题与独立 Resource 滚动](verification/2026-08-29_desktop-explorer-split-pane-light-scrolled.jpg)。

## 潜在影响、迁移边界与回滚

- 现有 Profile、连接、View、Workspace 与 Resource contract 无需迁移；新增偏好字段是可选字段，旧文档保持可读。
- 未实现有效权限/连接状态的前端推断，未改变 Hub policy、Permit、CredentialStore、wire protocol 或 Resource schema。
- 服务端 lazy loading/pagination 和真正 DOM virtual scrolling 未在本轮实现，分期边界已写入 canonical spec。
- 回滚可恢复 rebase 后实现提交 `1d6feaa` 涉及的 Explorer、preference、CSS、测试、dist 与稳定文档；不会回滚用户业务数据。
- 本轮未推送、发布、创建 remote 或处理品牌图标。

## 本地集成与保留状态

- 用户授权安全收敛后，先把原主检出 Desktop 差异与生成物备份到仓库外 `D:\project\MyFlowHub3\.tmp\desktop-explorer-merge-preservation-2026-08-29`；binary patch 与各文件 SHA-256 均已记录。
- 对比证明原 `style.css` 的 sidebar/ScrollArea containment 三处修改均被 Explorer 分支完整包含；`dist/**` 由最终源码重新生成，不手工拼接带哈希产物。
- 收敛期间发现另一任务已将品牌提交 `947d4ae` 合入 `master`。Explorer 两笔提交因此 rebase 到该提交之上，三个索引冲突均同时保留品牌与 Explorer 条目；品牌资产内容未由本任务修改。
- rebase 后实现提交为 `1d6feaa`（`实现 Desktop 节点资源分区浏览`），归档提交为 `fc47dc6`（`归档 Desktop 节点资源分区工作流`）。
- `master` 已从 `947d4ae` 快进到 `fc47dc6`；快进前后的完整 porcelain dirty status 一致。`guide.md`、`论文/**`、原型与验证截图均未暂存、覆盖、还原或归档。
- 合并后再次通过 27/27 前端测试、TypeScript/Vite production build 与 `GOWORK=off go test ./... -count=1`，生成物保持干净。
- 本记录提交后移除专用 worktree 与本地 feature branch；没有 push、release、remote 或品牌资产派生操作。

## 子 Agent 执行轨迹

- 无。Explorer 状态、preference、CSS、测试与 GUI 自动化是同一套有状态契约面；当前执行策略也未授权主动委派，因此由主 agent 顺序完成。
