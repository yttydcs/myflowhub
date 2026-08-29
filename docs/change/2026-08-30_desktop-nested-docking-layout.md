# 2026-08-30 Desktop 多面板嵌套停靠

## 变更背景 / 目标

旧 View v2 只能为两个 Widget 保存一个全局方向与比例，三个以上则退化为扁平响应式网格，无法表达
`A | C | B`、`A | (B / C)` 或 `(A | B | C) / D`。本次将布局升级为可持久化的 n 元分割树，
让面板通过拖拽直接完成精确插入、上下/左右嵌套和 root 停靠，不再依赖“左右 / 上下 / 交换”按钮。

## 具体变更内容

- View document 升级为 v3，以 `leaf | split` 的递归 `layout_root` 作为唯一布局权威；Widget 不再保存
  canonical `x/y/w/h`。
- Go boundary 按版本严格解码，确定性迁移 v1/v2，递归校验 leaf、axis、weights、深度和节点上限；第一次
  v3 写回前创建不可静默覆盖的 `views.pre-v3.json`，原子提交失败不覆盖旧文件。
- 新增纯 TypeScript 布局领域层，统一执行 add、panel edge、divider、workspace edge、move、remove、resize
  和同轴规范化；所有 mutation 保持 O(layout nodes)。
- Workspace 递归渲染任意深度 split。每对相邻 child 均有语义化 separator，指针移动通过 RAF 只更新
  本地 preview，释放时提交一次；方向键、Shift+方向键、Home/End、Enter 与双击提供等价操作。
- dnd-kit 命中顺序固定为 divider、workspace outer edge、panel edge、background；中心区域首期无效，不
  隐式创建 tab stack。拖动预览使用 hypothetical tree 和真实 track 规则。
- 单面板占满可用区，直接加入后续资源默认停靠到整个 root 右侧。递归最小尺寸不足时 Workspace 滚动，
  不重写持久拓扑。
- 未增加运行时依赖；保留 Profile 隔离、revision 冲突、Resource renderer、权限错误和 detached 状态。

## 文档治理影响

- Docs root: `docs/`
- Intake impact: updated — 新增并索引本次拖拽需求与方案研究。
- Feature impact: updated — Desktop 当前行为改为 n 元嵌套停靠。
- Requirements impact: updated — 增加精确插入、平滑相邻调整、迁移与滚动验收。
- Specs impact: updated — 新增 current v3 spec，并将 v2 布局契约标为 superseded。
- Decision impact: updated — 新增项目自有 n 元分割树 ADR，部分取代响应式网格选择。
- Lessons impact: none — 未产生新的可复用故障模式；既有 Windows/Wails generated-drift lesson 足够覆盖构建注意项。
- Index impact: updated — intake、specs、decisions、plan 与 change 索引均可发现本工作流。

## Related docs

- Related plan: [Desktop 多面板嵌套停靠计划](../plan/plan_archive_2026-08-30_desktop-nested-docking-layout.md)
- Related intake: [Desktop 多面板嵌套停靠](../intake/2026-08-30_desktop-nested-docking-layout.md)
- Related feature: [Desktop](../features/desktop.md)
- Related requirements: [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)
- Related specs: [Desktop Resource Workspace v3](../specs/desktop-resource-workspace-v3.md)
- Related decisions: [Desktop n 元多面板停靠](../decisions/2026-08-30_desktop-n-ary-docking-layout.md)
- Related lessons: [Windows clean-checkout、EOL 与 generated drift](../lessons/windows-clean-checkout-eol-and-generated-drift.md)

## 对应 plan.md 任务映射

| Task | 结果 |
| --- | --- |
| DOC01 | v3 feature / requirement / spec / ADR 与索引一致。 |
| VIEW01 | Go v3 store、严格迁移、递归校验、一次性 pre-v3 snapshot 与原子保存完成。 |
| LAYOUT01 | n 元树领域操作与 13 个纯布局测试完成。 |
| RENDER01 | 递归满区 pane、所有相邻 separator、RAF preview 与键盘交互完成。 |
| DOCK01 | 无专用按钮的 panel/divider/root 拖拽与真实未来布局 preview 完成。 |
| QA01 | 自动化、TypeScript/Vite、全仓 Go、generated review 和 Wails production build 通过。 |
| TEST01 | 真实 packaged GUI、迁移、6–8 pane、主题、窄窗口、保存/重启和错误状态通过。 |
| ARC01 | change/plan 归档、中文提交、本地安全合并与 worktree 清理完成。 |

## 关键设计决策与权衡

- 选择项目自有 n 元 split tree，而不是继续修补 12-column grid、强制二叉树或引入完整 docking framework；
  它能以最小依赖表达当前需求，但 tab stack、floating 与跨窗口停靠仍需未来单独决策。
- View v3 不保留并行像素/grid 真相，避免窗口 resize 反向污染用户拓扑；代价是旧 Desktop 必须通过
  pre-v3 snapshot 回滚。
- 拖动中心首期明确无效，避免用户误以为创建了标签栈。
- 每次 pointer move 不持久化，既消除旧 1/12 跳格，也控制 React 更新和磁盘写入。

## 测试与验证方式 / 结果

- `npm test`：8 个测试文件、44 个测试通过。
- `go test ./...`：全仓通过。
- `npm run build`：TypeScript `--noEmit` 与 Vite production build 通过。
- `wails build -clean -platform windows/amd64`：Windows production build 通过；最终
  `myflowhub-desktop.exe` SHA-256 为
  `3C94D2DB1B474D43D03B0529B472BA41D39CF9E23313CC17F0504C59269DE560`。
- 真实 Wails GUI：v2 四面板迁移、pointer/keyboard separator、拖拽生成交叉轴嵌套、六面板浅/深色、
  986 px 窄窗口、八面板横向滚动、v3/pre-v3 文件、保存/重开/重启和真实 Forbidden Inspector/Widget 均通过。
- 证据索引：[m-test evidence](../../artifacts/m-test/desktop-nested-docking/README.md)。
- `git diff --check` 与归档后回归通过。

## 经验 / 教训摘要与可复用排查线索

- 症状：旧 View 第一次启动为空。快速检查：测试 fixture 必须位于 Profile 自己的 `profiles/<id>/views.json`，
  不是 Desktop config root；这是隔离测试配置错误，不是迁移失败。
- 症状：八面板在窄窗口中部分不可见。快速检查：确认 `widget-layout-scroll` 出现横向滚动条，而不是让 CSS
  自动重排或缩到 renderer 不可操作。
- 症状：旧二进制拒绝 v3。快速检查：停止 Desktop 后恢复 `views.pre-v3.json`；不要只回滚代码。
- 关键词：`View v3`、`layout_root`、`split-gap`、`workspace-root-edge`、`pre-v3`、`requestAnimationFrame`。
- 上述内容是本次变更的操作说明，不新增独立 lesson。

## 潜在影响与回滚方案

- v3 对旧 Desktop 不向后兼容。回滚前先停止 Desktop，再把 `views.pre-v3.json` 恢复为 `views.json`；
  这会丢失迁移后新增的 v3-only 嵌套布局。
- v2 的非标准重叠/自由网格按 `(y, x, original index)` 做确定性近似，但所有 Widget 都保留。
- 可回滚本变更涉及的 View store、布局领域、Workspace/App、测试、tracked dist 和稳定文档；不删除用户
  Profile、Resource 或 Hub 数据。
- 品牌图标、Agent Gateway、Metrics 独立产品文档、push、release 和 publication 均不属于本工作流。

## 子Agent执行轨迹

- 无。当前 host policy 未授权主动委派，schema、领域、DnD、渲染、测试、文档和 GUI 验收由主 agent 顺序完成。
