# Win MCP Start Script

## 变更背景 / 目标

- 第一轮已交付 `myflowhub-mcp`，但仓内还缺少一个可直接给 MCP host 或本地开发环境使用的独立启动脚本。
- 目标是在 `scripts/` 下提供一个稳定入口，避免每次手动拼装 `go run ./cmd/myflowhub-mcp`。

## 具体变更内容

- 新增 `scripts/start-myflowhub-mcp.ps1`
  - 自动定位 repo root
  - 临时设置 `GOWORK=off`
  - 调用 `go run ./cmd/myflowhub-mcp`
  - 将脚本收到的剩余参数原样透传给 MCP CLI
- 更新 `README.md`
  - 增加脚本启动示例
  - 将 MCP host 示例调整为 `powershell.exe -File scripts/start-myflowhub-mcp.ps1`

## Requirements impact: none

## Specs impact: none

## Lessons impact: none

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\mcp-client.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\mcp-client.md`

## Related lessons

- none

## 对应 `plan.md` 任务映射

- `SCRIPT-1`
  - `scripts/start-myflowhub-mcp.ps1`
- `SCRIPT-2`
  - `README.md`
- `SCRIPT-3`
  - `docs/change/README.md`
  - `docs/change/2026-03-25_win-mcp-start-script.md`

## 经验 / 教训摘要

- 对于需要被外部 MCP host 拉起的命令，仓内最好提供一个稳定脚本入口，避免调用方依赖当前 shell 目录或手写 `go run` 细节。
- PowerShell 脚本只做“定位仓根 + 设置环境 + 参数透传”，不要复制 CLI 逻辑，否则后续参数扩展会出现双份维护。

## 可复用排查线索

- 症状：
  - MCP host 无法从 repo 内直接拉起 `myflowhub-mcp`
  - `go.work` 影响 `go run ./cmd/myflowhub-mcp`
- 触发条件：
  - 启动命令不在 repo root 下执行
  - 未显式关闭 workspace 模式
- 关键词：
  - `start-myflowhub-mcp.ps1`
  - `GOWORK=off`
  - `go run ./cmd/myflowhub-mcp`
- 快速检查：
  - 运行 `powershell -ExecutionPolicy Bypass -File scripts/start-myflowhub-mcp.ps1 --version`
  - 通过脚本喂 `initialize` / `tools/list`

## 关键设计决策与权衡

- 决策：脚本内部固定使用 `go run`
  - 原因：避免要求用户先手动 build，并始终运行当前 worktree 最新源码。
- 决策：参数全部透传，不在脚本层重复声明 `--endpoint` / `--config-dir`
  - 原因：避免脚本与 CLI 参数面重复维护。

## 测试与验证方式 / 结果

- `powershell -ExecutionPolicy Bypass -File scripts/start-myflowhub-mcp.ps1 --version`
  - workdir: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
  - 结果：通过
- 通过脚本执行进程级 smoke：
  - 输入：`initialize`、`tools/list`
  - 结果：返回正确 JSON-RPC 响应

## 潜在影响与回滚方案

- 潜在影响：
  - 脚本依赖本机安装 `go`
  - 首次 `go run` 会包含编译开销
- 回滚方案：
  - 删除 `scripts/start-myflowhub-mcp.ps1`
  - 回退 `README.md` 中脚本调用说明

## 子Agent执行轨迹

- none
