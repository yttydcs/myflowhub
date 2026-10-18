# Desktop 紧凑布局：验收与收尾审查

## Candidate and scope identity

- Repository / docs root：`D:/project/MyFlowHub3/repo/MyFlowHub` / `docs/`，直接在 canonical `master` 实施。
- 固定比较基线：`4886789573ec6d12c4e9a7f02ed2565dea6920f4`。
- 产品候选：`e486e451633292d1fd67e8a7ff4490ffd539b99f`，包含 18 个文件的代码、测试、构建产物和稳定文档。
- 本轮没有独立的事前 plan.md 或批准 Task ID；控制范围来自[用户确认的请求](../intake/2026-10-19_desktop-compact-layout.md)。下表 CL01–CL04 是收尾追溯编号，不冒充事前批准任务。此前统一 Variable 计划已归档；根 plan/todo 现由另一任务维护产品定义讨论，不将本轮误记为其未完成任务。
- 定义标识：COMPACT-AC-1，以下四行 AC 构成完整定义；源文件清单及证据 SHA-256 见[本地证据清单](../../../../artifacts/m-archive/2026-10-19_desktop-compact-layout/evidence.json)。
- 2026-10-19 审查时 `apps/desktop` 对候选无工作区差异，无本轮遗漏的未跟踪产品文件。其他产品文档、guide 和设计原型属于预先存在的无关修改，不纳入候选。归档阶段仅补充文档及索引。

## Acceptance / task / evidence

| Task / AC | 已确认要求 | 实现与证据 | 结论 |
| --- | --- | --- | --- |
| CL01 | 新建视图默认紧凑；可切换铺满分栏；既有视图保持原模式 | App.newView、Workspace 工具栏；2026-10-09 后台浏览器新增视图，2026-10-19 恢复既有视图并往返切换 | passed |
| CL02 | 按内容高度排列；卡片可拖入、排序和调尺寸，窗口收窄不覆盖存储尺寸 | CompactCard、workspace-layout、CSS；浏览器验证 1480/850 宽度、指针/键盘尺寸、Home、资源拖入和排序；下拉框 112px、数组 247px | passed |
| CL03 | 模式、两种布局独立状态可保存；切换保留编辑草稿；非法布局不得覆盖有效文档 | Workspace.test.tsx 同一 input DOM 和 45 草稿、单次 snapshot；workspace-layout.test.ts 独立顺序；views_compact_test.go 几何持久化、旧模式及边界；浏览器保存重载 | passed |
| CL04 | 用户可通过浏览器验收，资源网络正常连接 | 2026-10-19 真实 TCP/Wails 恢复检查；用户明确“验收没问题”并授权提交推送 | passed |

## Validation and freshness

- 2026-10-09：`npm test`，26 个文件、497 项通过；`GOWORK=off go test ./apps/desktop` 通过；TypeScript/Vite build、`scripts/mfh.ps1 -Action build -Target desktop` 的 Wails Windows 生产构建通过。
- 2026-10-09：后台 Edge 的真实 TCP/Wails 检查覆盖紧凑尺寸、响应式列数、模式切换、排序、资源树拖入、pointer/keyboard resize、恢复自动尺寸及保存重载，无 pageerror。
- 2026-10-19：`verify-restart-20261019.cjs` 确认连接、既有视图恢复、两种模式切换以及 247/112px 卡片高度。此脚本没有保存切换结果或覆盖用户视图；人工验收随后通过。
- 复用理由：候选提交仅收录上述已验证的产品内容，之后无产品代码/依赖变化；归档没有行为变更，因此不重复运行测试。日志、脚本与截图保全在[证据目录](../../../../artifacts/m-archive/2026-10-19_desktop-compact-layout/)。Wails 构建结果由当时工具成功输出佐证，原日志文件只保留调用行，不能把它描述为完整构建日志。

## Requirements review — passed

2026-10-19 按 COMPACT-AC-1 对照实际候选，检查 App 默认值、Workspace 模式切换、独立顺序/比例、CompactCard 内容高度与尺寸、Go 保存边界和原有分栏路径。需求与实现一致，既有视图、组件草稿及资源操作语义均保留，无未实现的本轮必需项。用户手动验收通过；未把前一轮 Flow/协议演进任务混入本轮完成范围。

## Standards review — passed

独立于需求结论，检查实际变更的模块边界、持久化校验、既有树不变量、ResizeObserver 清理、指针取消、错误传播和测试。未增加依赖，布局数据留在 Desktop View 层，未写入资源值/权限；Go 和前端均限制尺寸、拒绝非法模式；保持同一 renderer 组件实例。未发现当前范围内的阻断问题。归档补充旧 ADR 的模式适用范围，避免其“唯一布局”表述与现状冲突。

## Limits / unverified

- 紧凑视觉顺序由 CSS order 决定，DOM/Tab 顺序仍跟随 split tree；完整屏幕阅读器顺序一致性、触屏和所有 DPI 组合未做专项验证，不计为通过。
- 生产桌面包构建通过，但本轮最终交互复核使用浏览器，没有重新操控原生窗口。
- View v3 新增字段由旧程序严格解码时可能拒绝；回滚必须保全并适配视图数据，见[归档回滚说明](../change/2026-10-19_desktop-compact-layout.md)。
- 没有将上述范围外验证标为豁免或通过；无未解决的本轮阻断项，无子代理委派。
