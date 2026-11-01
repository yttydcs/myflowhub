# Desktop 多变量卡片与视图编辑模式

## 结果与范围

为紧凑桌面面板增加“名称＋值/控件”的多变量列表卡片，同时保留单变量卡片与两种既有布局。完成引用级重组、局部显示名称、编辑/查看会话与草稿保护；用户在第一版交付后明确要求合并并调用 `$m-archive`。仅本地收尾，不推送远端，不操作真实 Profile/设备。

- Docs root：canonical `repo/MyFlowHub/docs/`，来自现有 `docs/README.md`；无新 docs repo、remote、发布或备份策略。
- 产品候选：`4b3a216bdd708b710401e9c3464c8ad6c3bfc815`；固定比较 base：`cf1436115c10a6d3c6ecca3334c1a0e009aa7247`。
- 计划定义、Task/AC 写入范围、diff/blob manifest、两轴 review 与未验证事项由[唯一权威报告](../plan/desktop-variable-cards-review.md)承载；本归档不重复维护验收结论。

## 变更与任务映射

| Task | 交付 |
| --- | --- |
| T01 | View v3 可选 title/kind/variables/variable_id；旧视图读取、空列表和稳定引用身份；前后端校验与存储往返 |
| T02 | 纯函数添加/单卡合并/行排序/跨卡移动/拆出/显式转换；同卡去重、空容器保留 |
| T03 | 卡片独立标题/行别名、逐行状态错误、设置菜单等价入口；稳定 portal host 保护控件本地无效输入，结构 blur 不提交值 |
| T04 | 标题手柄布局、变量手柄重组；仅有效外部 drop 自动编辑；新建/重开/保存/取消、50 步结构撤销与异步保存保护；查看锁定结构但变量可操作 |
| T05 | 574 前端测试、Go 存储测试、TS/Vite/Wails 生产构建、Headless Edge 模拟交互、深浅色/窄窗口、文档归档与受控本地集成 |

关键权衡：列表是卡片而非第三布局；稳定引用 ID 与平铺 portal host 解耦编辑器和布局生命周期；模式只在会话内，View 不保存值/草稿；保持 v3 可选扩展读取旧文件，不承诺旧二进制前向兼容。

## 文档影响（显式使用 `$m-docs`）

| 类别 | Impact 与 canonical 去向 | 索引 |
| --- | --- | --- |
| Intake | updated：[原请求/确认约束/合并授权摘要](../intake/2026-10-31_desktop-variable-cards.md) | 更新 intake/README |
| Feature | updated：[Desktop 当前功能](../features/desktop.md)，卡片、模式、引用及回退边界 | 原有叶子入口继续有效，无需新增 feature 索引项 |
| Requirements | updated：[工作区需求](../requirements/desktop-resource-workspace.md) 第 20—23 及验收；未改 Flow/product-definition | 原有叶子入口继续有效 |
| Specs | updated：[View v3](../specs/desktop-resource-workspace-v3.md)，数据、限额、拖动优先级、会话/持久化与草稿合同 | 原有叶子入口继续有效 |
| Decision | updated：[n 元停靠 ADR 的本轮补充](../decisions/2026-08-30_desktop-n-ary-docking-layout.md)；区分变量正文合并和 center-no-stack | 原有叶子入口继续有效 |
| Lessons | updated：[变量确认与布局经验](../lessons/desktop-variable-confirmation-and-split-layout.md)，跨父 key/portal、结构 blur、窄窗口工具栏遮挡 | 更新 lessons/README 检索提示 |
| Plan / Change | updated：[计划](../plan/desktop-variable-cards.md)、[review](../plan/desktop-variable-cards-review.md)、[长期证据索引](../plan/desktop-variable-cards-evidence/README.md)和本归档 | 更新 plan/README、change/README 与证据目录 README |

顶层分类、全局阅读路径与 docs-root/publication 边界均未变，不修改根 docs/README。未改受保护生成文档区；frontend/dist 是既有跟踪的应用构建产物，不是新增文档发布渠道。

## 验证、审查及经验

Requirements review：passed；Standards review：passed，依据与候选/计划身份见[review](../plan/desktop-variable-cards-review.md)。收尾重跑前端 32 文件/574 测试、TypeScript/Vite 和 `GOWORK=off go test . -count=1` 通过；复用同内容第一版 Wails 构建及 Edge 模拟 API 交互证据。

模拟交互覆盖有效/取消/无效 drop、合并/去重/排序/跨卡/拆出、稳定输入 DOM、布局手柄、重命名、失败/成功保存、取消/新建/重开、窄窗口；结构操作零 write，查看模式显式变量操作能发出一次 mock write。没有真实 Profile/设备、真实 Wails GUI 或完整读屏验收；这些保持未验证，不因用户同意合并而伪称通过。

可复用经验已提升到 lesson，而非只放本归档：同 key 跨 React 父节点仍会 remount，共享 controller 不保存控件本地无效文本；先检查 input DOM 身份与 write 数，再看截图。结构手柄导致 blur 与变量提交意图需分离；窄窗口还要检查工具栏换行后是否遮住内容。检索关键词：跨卡草稿、portal host、结构 blur、toolbar 换行遮挡。

## 影响与回滚

- 不修改 Variable 协议/权限，不复制 owner 资源或改变网络行为；每卡 64 变量/每 View 512 引用、原 64 卡/布局上限保留。
- 新版可读取旧 v3，旧二进制严格 JSON 解码不能读取新增字段。试运行前备份该 Profile 的 `views.json`；回滚时关闭 Desktop、恢复旧二进制和该备份。不要拿 pre-v3 snapshot 当作本轮升级前备份，也不要直接删除新字段伪造兼容。
- 已写入真实设备的值不属于结构取消/代码回滚范围。当前未触碰真实数据。
- Vite 既有单 chunk 超 500 kB warning 仍在；首次 Wails/npm ci EPERM 已通过停专用 Vite、恢复依赖并使用已验证 dist 构建解决，无依赖锁文件漂移。
- 测试版 exe 已复制到工作区 `artifacts/desktop-variable-cards/MyFlowHub-Desktop-variable-cards.exe`，附 SHA-256；不纳入 Git，避免随 worktree 清理失效。

## 本地合并与清理记录

归档内容与当前两轴审查已完成，待从控制面执行本地合并及清理。主线初始有 255 个已修改/未跟踪文件，已在工作区 artifacts 记录路径/哈希；不批量暂存或提交它们。仅 `docs/intake/README.md` 与 `docs/plan/README.md` 两个索引与本轮相交，将窄范围暂存保护，合并后恢复并核对去除本轮新增链接后的内容一致。已有其他 stash 与其他 worktree 不在清理范围。

合并后需校验产品 manifest 每个 blob/删除与主线 tree 等价、链接有效、原有修改完整，再清理本轮 worktree/分支。最终实际状态在本节追加，不预先声称成功。远端状态始终不变：本次不 push。
