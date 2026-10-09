# 2026-10-09 统一 Variable 与 Desktop 通用交互

## Result

完成 DOC01、VAR01、PROV01、MET01、DESK01、LEGACY02、QA01：网络资源统一 Variable 契约、runtime/SDK/所有第一方提供者一次性替换、Desktop 通用消费。后续用户验收追加 shadcn 组件、简洁变量面板、控件加载/确认态、View 自动应用和分栏留白修复。EXT02 延期，不计入完成。

文档根为 canonical repo 的 docs/。[原计划及补充验收](../plan/plan_archive_2026-10-09_unified-variable-contract.md)、[Checklist](../plan/todo_archive_2026-10-09_unified-variable-contract.md)。计划标识 R2 + UI-20261009。

关键边界：节点提供变量，Desktop 按平台定义通用渲染，不按 Metrics 分支；write 经 owner/CAS/权限裁决。Inspector 手动 Apply，View 按控件完成编辑时自动提交，保留失败草稿。产品未发布，不保留旧变量格式、旧 API 或 Metrics /set。未改变身份/权限数据，也未设计 Flow。

若回滚，须整体回退同一提交的协议、提供者、SDK、前端和生成物，不可混用新旧 wire；先保存现有配置/视图，禁止将无兼容要求解释为删除真实数据。

## Evidence

- [原始权威执行报告](../../../../artifacts/m-execute/unified-variable-contract/REPORT.md)：AC-01..16、Task 映射、Go 全模块/vet/race、全产品构建、生成物和原生/浏览器实际联调，以及子代理审查轨迹。
- [最终验收与独立审查补充](../../../../artifacts/m-archive/2026-10-09_unified-variable-contract/REPORT.md)：最终候选内容指纹、复用边界、488 项完整前端测试、Desktop Go、TypeScript/Vite/Wails 生产构建和两类审查。
- [保全的 UI 逐次联调记录](../../../../artifacts/m-archive/2026-10-09_unified-variable-contract/retained-out/ui-refresh-20261009/REPORT.md)：后台浏览器控件确认、窄栏只读结构、十种布局场景，使用真实 TCP/Wails；中间阶段行为以后续条目为准。

Requirements / Standards 均通过；无未解决阻断项或豁免。实际音量写入/恢复通过；物理亮度设备未验证，未计为通过。最新原生包已重建，最新 UI 交互由后台浏览器验证；未重新操作原生窗口抢占用户输入。

## Documentation

- Intake updated：[原始需求](../intake/2026-09-06_unified-variable-contract.md)、[手动验收追加](../intake/2026-10-09_desktop-variable-manual-acceptance.md)及索引。
- Features updated：Desktop、Metrics、Clipboard、File、Hub 的当前实现说明。
- Requirements updated：[统一 Variable](../requirements/unified-variable-resource.md)、Desktop 工作区和可扩展资源平台及索引。
- Specs updated：[统一 Variable](../specs/unified-variable-resource.md)、Desktop schema rendering、resource platform、protocol map 及索引。
- Decisions updated：[平台 Variable 契约](../decisions/2026-09-06_platform-variable-contract.md)及索引。
- Lessons updated：[控件确认与分栏布局](../lessons/desktop-variable-confirmation-and-split-layout.md)，以及跨运行时边界、可观察副作用与生成契约两篇既有经验。
- Plan / Change 索引已更新；无未受影响的上述六类目录，docs 根路由保持不变。

## Closeout

本地提交、主线合入、工作树清理结果及未提交改动保全以 [收尾状态](../../../../artifacts/m-archive/2026-10-09_unified-variable-contract/closeout.json) 为准。本次不推送。主检出既有节点树刷新与产品文档改动单独保留，不计入本次提交。临时联调进程按路径关闭，配置、日志、截图与二进制保存在本地 artifacts，未加入 Git。
