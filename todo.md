# Workflow Todo - Codex MCP Runtime Cleanup

## 项目目标与当前状态
- 目标：提供一个可手动执行、可双击运行的 Windows 清理脚本，用于清理 Codex 长时间使用后堆积的 MCP 相关 runtime。
- 当前状态：
  - 已确认主要增长类型为 `chrome-devtools-mcp`、`ace-tool`、`context7-mcp`。
  - 当前进程统计显示 `chrome-devtools-mcp` 增长最快，且其路径中包含额外 `watchdog` 子进程。
  - 当前 worktree 中尚无 `scripts/` 目录，需要在本 workflow 内补齐。

## Workflow 信息
- 仓库：MyFlowHub3
- 分支：`chore/codex-mcp-runtime-cleanup`
- Base：`master`
- Worktree：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup`
- 当前阶段：`3.1`
- 计划文档：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\todo.md`

## Checklist
- [ ] T1 统计并固化“主要涨得快”的 MCP 类型与默认清理策略
- [ ] T2 实现主 PowerShell 清理脚本
- [ ] T3 实现双击启动入口
- [ ] T4 验证脚本预览与参数路径
- [ ] T5 执行 Code Review 并归档变更

## 任务明细

### T1 统计并固化“主要涨得快”的 MCP 类型与默认清理策略
- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\todo.md`
- 目标：
  - 固化当前主要增长类型及其优先级，形成脚本默认策略。
- 涉及模块 / 文件：
  - 无代码文件写入，仅作为设计输入。
- 验收条件：
  - 明确默认优先处理 `chrome-devtools-mcp`。
  - 明确 `ace-tool` 与 `context7-mcp` 为可选扩大范围。
- 测试点：
  - 与当前系统进程统计一致。
- 回滚点：
  - 若统计依据有误，仅调整脚本默认参数，不影响整体结构。
- 依赖：
  - 无。
- 风险与注意事项：
  - 统计是当前现场快照，后续比例可能变化，但不影响脚本的可配置设计。

### T2 实现主 PowerShell 清理脚本
- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\todo.md`
- 目标：
  - 实现进程发现、分类、汇总、预览、清理主逻辑。
- 涉及模块 / 文件：
  - `D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\scripts\cleanup-codex-mcp.ps1`
- 验收条件：
  - 支持 `Preview` / `Kill`。
  - 支持 `MinAgeMinutes` 和 `Kinds` 参数。
  - 仅匹配 MCP 相关 `node.exe` / `cmd.exe`。
  - 输出汇总统计与目标清单。
- 测试点：
  - `Preview` 模式可正常列出命中。
  - `Kill` 模式命令可被解析并进入执行路径。
- 回滚点：
  - 删除新增脚本文件即可回滚。
- 依赖：
  - T1。
- 风险与注意事项：
  - 进程匹配规则要保守，避免误伤业务 Node 进程。

### T3 实现双击启动入口
- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\todo.md`
- 目标：
  - 提供无需手动输入 PowerShell 命令的双击入口。
- 涉及模块 / 文件：
  - `D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\scripts\cleanup-codex-mcp-preview.cmd`
  - `D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\scripts\cleanup-codex-mcp-chrome-safe.cmd`
  - `D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\scripts\cleanup-codex-mcp-all-safe.cmd`
- 验收条件：
  - 双击可调用主 PowerShell 脚本。
  - 默认参数偏保守。
- 测试点：
  - `.cmd` 能解析到同目录 `.ps1`。
- 回滚点：
  - 删除新增 `.cmd` 文件即可回滚。
- 依赖：
  - T2。
- 风险与注意事项：
  - 双击执行需要给出暂停或结果展示，避免窗口一闪而过。

### T4 验证脚本预览与参数路径
- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\todo.md`
- 目标：
  - 在不大规模影响现有会话的前提下，验证脚本基本可用。
- 涉及模块 / 文件：
  - 上述脚本文件。
- 验收条件：
  - `Preview` 模式输出命中汇总。
  - 至少验证一条保守的 `Kill` 命令参数可执行。
- 测试点：
  - 预览默认 `chrome`。
  - 预览 `chrome+ace+context7`。
- 回滚点：
  - 若发现规则过宽，收紧正则与默认年龄阈值。
- 依赖：
  - T2、T3。
- 风险与注意事项：
  - 默认不做激进清理；验证以预览为主。

### T5 执行 Code Review 并归档变更
- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup`
- Plan 路径：`D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\todo.md`
- 目标：
  - 完成本 workflow 的审查与变更归档。
- 涉及模块 / 文件：
  - `D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\todo.md`
  - `D:\project\MyFlowHub3\worktrees\codex-mcp-runtime-cleanup\docs\change\2026-03-21_codex-mcp-runtime-cleanup.md`
- 验收条件：
  - Review 各项检查给出通过 / 不通过。
  - 归档文档完整覆盖变更、验证、回滚方案。
- 测试点：
  - 文档存在且内容完整。
- 回滚点：
  - 删除归档文档并回退脚本文件。
- 依赖：
  - T1-T4。
- 风险与注意事项：
  - 归档需要如实记录“未根治，仅止血”的限制。

## 依赖关系
- T2 依赖 T1
- T3 依赖 T2
- T4 依赖 T2、T3
- T5 依赖 T1-T4

## 风险与注意事项
- 当前任务文件写集高度集中在 `scripts/` 与 `docs/change/`，不适合安全拆成多个子Agent并行修改。
- 不使用子Agent的原因：
  - 当前任务规模较小，脚本与入口文件强耦合。
  - 写集重叠明显，拆分并不能降低主路径时延，反而增加集成风险。

