# 2026-10-19 Desktop 紧凑布局与模式切换

## Result

完成用户确认的紧凑优先工作区：新建视图采用内容高度的卡片网格；每个视图可切换原有铺满分栏，两种布局分别保存顺序/尺寸或树/权重。支持资源拖入、卡片排序、宽高调整和自动尺寸恢复，切换保留组件草稿。代码及对应测试、构建产物、稳定文档已于 `e486e45` 提交并推送 `origin/master`。

Docs root：canonical `D:/project/MyFlowHub3/repo/MyFlowHub/docs`。本轮采用会话确认范围，未建立事前独立计划；[权威验收记录](../plan/review_2026-10-19_desktop-compact-layout.md)以 CL01–CL04 回溯映射，不伪造计划批准历史。

保留既有 n 元分割树并增加独立紧凑顺序/尺寸，避免模式切换覆盖另一模式布局；复用 React/dnd-kit/CSS，不新增 docking 依赖。网络模型、资源值及权限语义无变化。

## Evidence

验收与两类审查、候选/基线身份、复用理由和未验证事项均以[验收记录](../plan/review_2026-10-19_desktop-compact-layout.md)为唯一入口：497 项前端测试、Desktop Go、TypeScript/Vite、Wails 生产构建及真实 TCP/Wails 后台浏览器通过；用户 2026-10-19 明确验收通过。Requirements / Standards 均 passed；完整读屏顺序、触屏、所有 DPI 组合及最终原生窗口交互未专项验证。

## Documentation

| 类别 | Impact | 关联与索引 |
| --- | --- | --- |
| Intake | updated | [确认请求](../intake/2026-10-19_desktop-compact-layout.md)，本次补齐索引 |
| Feature | updated | [Desktop](../features/desktop.md)，已随产品提交，既有索引有效 |
| Requirements | updated | [工作区](../requirements/desktop-resource-workspace.md)，已随产品提交，既有索引有效 |
| Specs | updated | [View v3](../specs/desktop-resource-workspace-v3.md)，已随产品提交，既有索引有效 |
| Decision | updated | [n 元分割树 ADR](../decisions/2026-08-30_desktop-n-ary-docking-layout.md)追加两种模式适用范围，既有索引有效 |
| Lessons | updated | [布局及状态保留经验](../lessons/desktop-variable-confirmation-and-split-layout.md)补充高度留白、模式切换 remount、窄窗口隐式 grid 轨道溢出，更新检索线索 |

Plan / Change 分类索引已补齐；根 plan/todo 正由另一任务维护产品定义讨论，本轮不改写其活跃状态。docs 根路由未改变，无需重复更新根 docs README。未委派子代理。

## Rollback

回退前备份各 Profile 的 `views.json`；旧二进制严格解码可能拒绝 `layout_mode` / `compact_size`。若必须回退产品提交，恢复升级前视图备份，或仅在备份副本中显式移除这两个新增字段，保留 widgets 与合法 layout_root，再由旧程序校验。此操作会丢失紧凑模式偏好和尺寸，应明确告知；禁止直接删除用户 Profile、凭据或所有视图。不修改现有视图仅切回“铺满分栏”即可停用紧凑展示。

## Closeout

- 产品代码已在 canonical master，远端包含 `e486e45`，不需要另行合并。本次归档文档单独本地提交，不新增推送授权或改变 remote。
- 本轮未创建 worktree/feature branch，因此无可清理对象；其余三个 worktree 属于其他任务，保留。
- 24 小时验收服务继续运行；临时身份、permit 和日志不加入 Git。可复用证据已复制到本地 artifacts，不依赖临时服务存活。
- 其他产品定义文档、guide 和设计原型修改保持原样。本次归档提交后本地比 origin/master 多一个文档提交。
