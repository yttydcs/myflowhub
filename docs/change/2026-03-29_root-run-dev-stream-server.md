# 2026-03-29 root run-dev stream Server 路径修复

## 变更背景 / 目标

- 用户从 workspace 根执行 `.\scripts\run-dev.ps1` 后，Win `Stream` 页面创建本地 source 仍报：
  - `Failed to create local source.`
  - `stream announce: request timed out`
- Win 日志进一步显示：
  - `stream list_consumers sent ... tgt=1`
  - `stream list_sources sent ... tgt=1`
  - `stream announce sent ... tgt=1`
  - 随后统一 `context deadline exceeded`
- 这说明 timeout 发生在 Hub 侧控制面，而不是 Win 本地 owner 侧：
  - 根脚本默认启动的是 `repo/MyFlowHub-Server`
  - 该主线 Server 默认集合尚未注册 `stream` handler
  - 真正带 `stream` 的 Hub 位于 `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`
  - 且该 worktree 如果不显式 `GOWORK=off`，还会被根 `go.work` 污染回旧模块图
- 本次目标是修复 root `run-dev.ps1` 的默认 Server 启动路径，让它真正吃到 `stream` 更新，并在路径 / `GOWORK` 失配时直接输出诊断信息。

## 具体变更内容

### 修改
- `scripts/run-dev.ps1`
  - 新增 `-ServerProjectDir` 参数，允许显式指定要启动的 Server 项目目录
  - `Resolve-WorkspaceRoot()` 改为向上探测真实 workspace root，支持在 root worktree 中执行脚本验证
  - 新增 `Resolve-WorkspacePath()` / `Assert-ServerProjectLayout()` / `Test-ServerSupportsStream()` / `Resolve-ServerProjectSelection()`
  - 默认选择策略改为：
    - 显式参数优先
    - 若主线 `repo/MyFlowHub-Server` 已支持 `stream`，继续优先主线
    - 否则自动切到 `worktrees/server-stream-subproto-design`
    - 若仍未找到支持 `stream` 的候选，则保留主线并输出 warning
  - 新增启动摘要：
    - `Server 项目`
    - `Server 来源`
    - `Server stream 支持`
    - `Server GOWORK`
  - 当选中的 Server 不是主线 `repo/MyFlowHub-Server` 且用户未显式传 `-GoWorkOff` 时，仅对 Server 默认附加 `GOWORK=off`
  - 当最终选中的 Server 不支持 `stream` 时，启动前直接提示：
    - `Win Stream 页的 list_sources / list_consumers / announce 仍会超时`
- `docs/change/README.md`
  - 新增本条归档入口
- `docs/lessons/run-dev-stream-server-selection.md`
  - 新增 root 启动脚本路径 / `go.work` 污染的复用排查文档
- `docs/lessons/README.md`
  - 新增 lesson 索引与关键词
- `todo.md`
  - 记录本轮 workflow 的计划、验证结果与 code review

### 不变
- 不修改 `repo/MyFlowHub-Server` 主线代码
- 不修改 `Proto / SubProto / Win` 的 `stream` 协议或业务逻辑
- 不改根 `go.work` 内容

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `added`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\requirements\stream.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\specs\stream.md`

## Related lessons

- `docs/lessons/run-dev-stream-server-selection.md`
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\lessons\stream-control-plane-validation.md`
- `docs/lessons/wails-binding-proto-drift.md`

## 对应 plan.md 任务映射

- `RDS-1`
  - 新增 Server 项目目录解析、候选选择和 `stream` 支持探测
- `RDS-2`
  - 为非主线 Server 默认附加 `GOWORK=off`
- `RDS-3`
  - 补齐帮助示例、启动摘要和回退 warning 验证
- `RDS-4`
  - 归档本次 change / lesson / 索引更新

## 经验 / 教训摘要

- 对 root 级启动脚本，真正的故障点不一定在“被报错的应用页”；这次是 Win 报 `stream announce timeout`，但根因在脚本启动了错误的 Hub。
- 当一个能力仅在某个 Server worktree 可用时，根脚本不能继续把主线 repo 当作“默认正确路径”，否则会制造“本地明明修好了但 UI 仍超时”的假象。
- 对位于 workspace 子目录下的 alternate Server worktree，只切路径还不够；若不同时处理 `GOWORK=off`，Go 仍会顺着根 `go.work` 吃回旧版 `Proto / Server` 模块图。

## 可复用排查线索

- 症状
  - Win `Stream` 页创建本地 source 失败
  - 日志包含 `stream announce: request timed out`
  - 日志包含 `stream list_sources await failed: context deadline exceeded`
  - 日志包含 `stream list_consumers await failed: context deadline exceeded`
- 触发条件
  - 从 workspace 根使用 `run-dev.ps1` 默认启动链路
  - 主线 `repo/MyFlowHub-Server` 尚未合入 `stream`
  - 真正带 `stream` 的 Server 位于 worktree
  - 该 worktree 未显式 `GOWORK=off`
- 关键词
  - `run-dev.ps1`
  - `stream announce: request timed out`
  - `list_sources await failed`
  - `list_consumers await failed`
  - `server-stream-subproto-design`
  - `newStreamHandler`
  - `GOWORK=off`
  - `go.work`
- 快速检查
  - 看启动摘要里的 `Server 项目` 是否仍是 `repo/MyFlowHub-Server`
  - 检查 `repo/MyFlowHub-Server/modules/defaultset/hub.go` 是否缺少 `newStreamHandler`
  - 检查 `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\modules\defaultset\hub.go` 是否包含 `newStreamHandler`
  - 在 `server-stream-subproto-design` 下执行 `go env GOWORK`
  - 再比较：
    - `go test ./tests -run TestStreamRootHubConnectDisconnect -count=1`
    - `$env:GOWORK='off'; go test ./tests -run TestStreamRootHubConnectDisconnect -count=1`

## 关键设计决策与权衡

- 决策：先修 root 启动脚本，不扩展到 Server 主线发布链
  - 原因：当前 blocker 是“本机没启动到正确 Hub”，不是 `stream` 业务实现缺失
- 决策：默认自动优先 `stream` worktree，同时保留 `-ServerProjectDir`
  - 原因：既解决默认路径错误，又不剥夺显式控制
- 决策：只对 Server worktree 默认关闭 `GOWORK`
  - 原因：问题证据指向 alternate Server 路径；不需要把 Win / MetricsNode 的既有默认行为再次扩大修改
- 决策：保留主线回退 warning，而不是在无 `stream` 支持时直接硬失败
  - 原因：脚本仍承担通用冒烟用途，但必须让 `stream` 不可用成为显式、可见的事实

## 测试与验证方式 / 结果

- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`
  - 执行：`go env GOWORK`
  - 结果：`D:\project\MyFlowHub3\go.work`
  - 说明：证明 alternate Server worktree 默认会被根 workspace 污染
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`
  - 执行：`$env:GOWORK='off'; go test ./tests -run TestStreamRootHubConnectDisconnect -count=1`
  - 结果：通过
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`
  - 执行：`go test ./tests -run TestStreamRootHubConnectDisconnect -count=1`
  - 结果：失败
  - 说明：报 `directory tests is contained in a module that is not one of the workspace modules listed in go.work`
- root worktree
  - 执行：`pwsh -NoLogo -ExecutionPolicy Bypass -File scripts/run-dev.ps1 -SkipServer -SkipWin -SkipMetricsNode`
  - 结果：通过
- root worktree
  - 执行：`Get-Help .\scripts\run-dev.ps1 -Detailed`
  - 结果：通过
  - 说明：帮助输出已包含 `-ServerProjectDir`
- root worktree
  - 执行：`pwsh -NoLogo -ExecutionPolicy Bypass -File scripts/run-dev.ps1 -ServerAddr ':9011' -SkipWin -SkipMetricsNode`
  - 结果：通过
  - 说明：输出确认
    - `Server 项目：D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`
    - `Server stream 支持：yes`
    - `Server GOWORK：off（Server worktree 默认）`
- root worktree
  - 执行：`pwsh -NoLogo -ExecutionPolicy Bypass -File scripts/run-dev.ps1 -ServerProjectDir 'repo\MyFlowHub-Server' -ServerAddr ':9012' -SkipWin -SkipMetricsNode`
  - 结果：通过
  - 说明：输出 warning，明确主线 Server 不支持 `stream`

## 3.3 Code Review 结论

- 需求覆盖：通过。默认启动路径、显式覆盖、失配 warning 和 `GOWORK` 防护都已覆盖。
- 架构合理性：通过。改动集中在 root 启动脚本，不触碰 `stream` 协议与业务层。
- 性能风险：通过。只增加少量本地文件存在性检查和文本探测，没有引入重型扫描。
- 可读性与一致性：通过。新增 helper 复用现有 PowerShell 结构，变量命名与既有脚本保持一致。
- 可扩展性与配置化：通过。显式 `-ServerProjectDir` 保留手动控制；主线未来合入 `stream` 后会自动回退优先主线。
- 稳定性与安全：通过。对错误路径、缺失 `go.mod/cmd\hub_server`、不支持 `stream` 的路径均给出显式反馈。
- 测试覆盖情况：通过。完成帮助验证、脚本行为验证、Server worktree 的 `GOWORK` 正反验证。
- 子Agent治理与审计：通过。本轮未使用子Agent。

## 潜在影响与回滚方案

- 潜在影响
  - 默认 `.\scripts\run-dev.ps1` 启动 Server 时，若主线尚未合入 `stream`，会自动切到 `worktrees/server-stream-subproto-design`
  - 如果用户显式指定主线 `repo/MyFlowHub-Server`，现在会看到新的 warning
  - 在 alternate Server worktree 启动场景下，Server 默认不再继承根 `go.work`
- 回滚方案
  - 回退 `scripts/run-dev.ps1`
  - 回退 `docs/change/README.md`
  - 删除 `docs/change/2026-03-29_root-run-dev-stream-server.md`
  - 删除 `docs/lessons/run-dev-stream-server-selection.md`
  - 回退 `docs/lessons/README.md`
  - 回退 `todo.md`

## 子Agent执行轨迹

- 未使用子Agent
