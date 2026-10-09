> 历史执行快照；下列阶段/工作树路径保留当时状态。最终状态见文末归档补充。

# Todo — 平台统一 Variable

- Workflow：2026-09-06 unified-variable-contract。
- Stage：$m-execute completed；本轮执行与验收完成，未归档、提交、合入或推送。
- Plan Revision：R2；用户明确产品未发布、无需任何兼容，取代 R1 兼容安排。
- Plan：[plan.md](plan_archive_2026-10-09_unified-variable-contract.md)。
- Worktree：D:/project/MyFlowHub3/worktrees/unified-variable-contract。
- Branch：feat/unified-variable-contract。
- Base：d77dc54b2fd6efe749547120683dcb913eea4714。
- Approval：用户 $m-execute 已批准以下 7 项；后续批准后台浏览器继续联调。

## 已完成的批准范围

- [x] DOC01 — 产品原则、Current R2 规格、Implemented R2 决策、features 与索引对齐。
- [x] VAR01 — 平台/runtime/SDK 标准变量闭环、统一校验、CAS、Catalog 原子发布。
- [x] PROV01 — 管理、文件摘要、Clipboard 的所有 Variable 迁移；保留原领域和持久化行为。
- [x] MET01 — 10 项真类型采样、config、实际同资源 write；删除 /set 并更新 UI/CLI。
- [x] DESK01 — 通用展示、持续订阅、共享会话、草稿、确认值与标准编辑。
- [x] LEGACY02 — 清除被替代的变量协议/API/renderer/生成引用，增加禁止双轨检查。
- [x] QA01 — AC-01..16 证据映射、独立审查、全产品门禁和隔离 Windows/浏览器联调。

## 本轮不执行

- [ ] EXT02 — 其他资源远程 schema 服务与异步执行模型；延期，不记为通过。

## Evidence

- [执行验收报告](../../../../artifacts/m-execute/unified-variable-contract/REPORT.md)：AC/Task、命令、截图、实际候选与计划身份。
- [候选内容指纹](../../../../artifacts/m-execute/unified-variable-contract/candidate-identity.json)：包含 tracked dirty、untracked、删除、模式和生成物；HEAD 不作为唯一身份。
- Requirements：passed；Standards：passed。AC-01..16 passed；无未解决发现或豁免。
- Go 全模块、vet、相关 race、canonical check/build all 通过；最终 Desktop 438 项前端测试、TS/Vite 与生产构建通过。生成物 15/15 一致。
- 陌生节点通用编辑、真实 Metrics 数据、订阅跨过 60 秒续租边界、保存视图与重载恢复通过；后续全部用后台浏览器，真实 Go/Wails/TCP 链路。
- Windows 音量/静音标准写入及恢复通过；物理亮度未测，计划要求的“音量/静音或亮度”由前者满足，不宣称全硬件验证。
- 本轮后台测试进程核实后停止，fixture 正常退出；保留证据和工作树，不修改既有手动联调进程、真实身份/策略或主检出。

回退需使用同批匹配源码与构建，不能混跑新旧 Variable。执行阶段不生成 docs/change，不提交、合入、推送或清理工作树。

## 2026-10-09 归档补充：用户手动验收范围

计划标识 R2 + UI-20261009。原 AC-01..16 与任务批准保持；用户后续直接请求扩展 DESK01/QA01，见 [追加 intake](../intake/2026-10-09_desktop-variable-manual-acceptance.md)。Inspector 显式 Apply 保持，View 自动应用按最后确认的控件提交时机执行，取代原计划中对所有呈现场景一概手动提交的限制。

| 补充验收 | 任务 | 最终行为 / 证据 |
| --- | --- | --- |
| UI-01 | DESK01, QA01 | shadcn 通用控件、统一标题/圆点、View 简洁布局；完整前端与 UI 报告 |
| UI-02 | DESK01, QA01 | View 本地提交意图、Enter/blur/滑块提交、控件加载态、owner 确认；VariablePanel/control tests 与真实 control-commit |
| UI-03 | DESK01, QA01 | 只读复杂值全宽有界显示、权限拒绝保留快照；catalog-preview 和组件测试 |
| UI-04 | DESK01, QA01 | 分栏比例与最小尺寸共同满足，布局结构变化丢弃旧预览；28 项布局测试及 10 组真实浏览器布局 |

完成状态与最新 488 项前端测试、生产构建、候选指纹、Requirements / Standards 审查见 [归档](../change/2026-10-09_unified-variable-contract.md)。EXT02 仍延期；无需物理亮度设备的替代验收已通过实际音量写入/恢复，不声称验证了物理亮度。
