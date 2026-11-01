# Desktop 多变量卡片与视图编辑模式

## 已确认交互
- 保留单变量卡片，增加多变量列表卡片，与现有紧凑/分栏布局正交。
- 卡片标题和组内行别名可覆盖默认名称，清空恢复自动名称，只影响视图引用。
- 标题手柄移动卡片，变量手柄移动引用；资源列表拖入为新增引用，卡片间为移动引用。
- 变量落在单变量卡片内容区将其转成列表，落在列表按行插入，落在布局区域独立成卡。
- 查看模式隐藏结构编辑入口/手柄/缩放，变量操作仍可用。新建进入编辑，打开已有视图进入查看，保存成功回到查看。
- 外部资源有效放下才进入编辑；无效释放/取消不改变模式。保存失败或保存期间继续修改保留编辑草稿。
- 取消只恢复视图结构，不回滚真实变量值。列表剩一项不自动降级，移空保留容器。
- 同卡不重复引用同一变量；保留引用 ID 和渲染实例以保护输入草稿，拖动不提交数值。

## Task / AC 定义及写入范围

所有源码路径相对 `apps/desktop/`；每项可同时修改对应测试。

| Task | 原实施项及允许写入范围 | AC |
| --- | --- | --- |
| T01 | View v3 可选字段、校验与往返；`views.go`、`views_variable_cards_test.go`、`frontend/src/types.ts`、`workspace-layout.ts` | AC01：旧文件可读；单卡/列表/空容器可保存重开；拒绝重复/非法绑定/越界，引用 ID 唯一 |
| T02 | 不可变引用操作；`frontend/src/workspace-variables.ts` 及单测 | AC02：添加/合并/排序/跨卡移动/拆出/转换保留 ID，失败不移除来源；同卡不重复；一行/空列表保留 |
| T03 | 稳定渲染与卡片/行设置；`VariableCards.tsx`、CSS、`WidgetSettings.tsx`、`variable/VariablePanel.tsx` | AC03：标题/别名清空恢复默认，逐行状态与错误；跨卡/拆分/切换布局保持同一无效输入 DOM，结构操作零 write；非拖拽等价操作 |
| T04 | 两级拖拽、模式会话和布局锁定；`App.tsx`、`Workspace.tsx`、`CompactCard.tsx` | AC04：标题拖布局、变量拖引用；有效外部 drop 才进入编辑；新建/打开/保存/取消及异步保存保护符合确认；查看模式可操作变量 |
| T05 | 测试、构建产物及文档；前端 `*.test.*`、`test/variable-cards*`、`test/verify-variable-cards.cjs`、`dist/` 和治理文档 | AC05：前端/Go 测试、TS/Vite/Wails 构建及模拟浏览器通过；浅深色/窄窗口反馈；不操作真实 Profile/设备；归档及主线脏文件保护 |

## 边界
- 不改变变量协议、权限和网络语义，不持久化实时值或变量草稿；无自动保存。
- 扩展字段仍使用版本 3，旧文件无修改即可读取；旧二进制严格 JSON 解码不认识新字段，试运行前应备份视图文件，回退时恢复该备份。
- 每卡最多 64 变量、每视图最多 512 引用，原每视图 64 卡片上限不变；结构撤销最多 50 步。
- 仅本次独立分支/worktree 在收尾范围，主线原有文档和其他 worktree 不属于本次提交/清理范围。不推送远端。
- 原始约束：[Intake](../intake/2026-10-31_desktop-variable-cards.md)；长期需求：[Desktop 资源工作区](../requirements/desktop-resource-workspace.md) 第 20—23 项。

## 交付状态（进度，不属于计划定义哈希）

- T01—T05 实施完成。2026-10-31 第一版未提交/合并，等待用户验收；2026-11-01 用户明确授权合并及 `$m-archive` 收尾，取代原“先不提交”暂停点。
- 唯一权威验收表、Requirements/Standards 两轴结论、候选/计划身份及长期证据：[review](desktop-variable-cards-review.md)。本文不另维护第二份验收报告。
- 原始日志与截图在工作区 `artifacts/`；需长期保留部分已纳入 `desktop-variable-cards-evidence/`，清理 worktree 不影响证据。
- 执行入口：前端 `npm test -- --maxWorkers=4`、`npm run build`；桌面 `go test . -count=1`；`wails build -s -m -nosyncgomod -platform windows/amd64 -o MyFlowHub-Desktop-variable-cards.exe`。
- 浏览器入口：前端 `test/variable-cards.html`、`test/verify-variable-cards.cjs`，Vite `127.0.0.1:62817`，通过 `PLAYWRIGHT_PATH` / `QA_URL` / `QA_ARTIFACTS` 设置运行环境，仅模拟 API。
- 首次普通 Wails 构建遇运行中 Vite 锁住 lightningcss 导致 npm ci EPERM，后使用已验证 frontend/dist 与 `-s` 成功；停止专用 Vite 并重新 npm ci 恢复依赖，未改 go.mod/package-lock、未迁移真实数据。
- [归档与最终合并/清理状态](../change/2026-11-01_desktop-variable-cards.md)。
