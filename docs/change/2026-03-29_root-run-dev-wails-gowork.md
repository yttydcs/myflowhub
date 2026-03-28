# 2026-03-29 root run-dev Win Wails GOWORK

## 变更背景 / 目标

- 用户从 workspace 根执行 `.\scripts\run-dev.ps1` 启动 Win 时，Wails CLI 在 `Generating bindings` 阶段失败：
  - `module github.com/yttydcs/myflowhub-proto provides package github.com/yttydcs/myflowhub-proto/protocol/stream and is replaced but not required`
- 同一台机器、同一目录下，`repo/MyFlowHub-Win` 执行 `$env:GOWORK='off'; wails generate module` 却可以通过。
- 这说明问题不在 Win repo-local `go.mod` 本身，而在 root `run-dev.ps1` 默认让 Win 继承了根 `go.work` 的 workspace 模式。
- 本次目标是让根脚本的默认 Win 启动路径回到稳定、可复现的单模块解析，同时保留显式 opt-out。

## 具体变更内容

### 修改
- `scripts/run-dev.ps1`
  - 新增 `-WinUseWorkspace` 参数，作为 Win 保留根 `go.work` 的显式 opt-out
  - 在未指定 `-WinUseWorkspace` 且未全局使用 `-GoWorkOff` 时，Win 进程默认附加 `GOWORK=off`
  - 启动日志中增加 `Win GOWORK：...` 输出，明确当前 Win 的模块解析模式
- `docs/lessons/wails-binding-proto-drift.md`
  - 增补“根 `go.work` 污染 Win Wails bindings”这一变体
  - 加入错误关键词 `replaced but not required` 和 `protocol/stream`
- `docs/lessons/README.md`
  - 更新索引摘要与关键词
- `docs/change/README.md`
  - 新增本条归档入口
- `plan.md`
  - 记录本轮 workflow 的计划、验证与 review

### 不变
- 不修改根 `go.work`
- 不修改 `repo/MyFlowHub-Win` 的业务代码
- 不改变 Server / MetricsNode 的默认 `GOWORK` 行为

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `updated`

## Related requirements

- none

## Related specs

- none

## Related lessons

- `docs/lessons/wails-binding-proto-drift.md`

## 对应 plan.md 任务映射

- `ROOTRUN-1`
  - 调整 `scripts/run-dev.ps1` 的 Win 启动环境
- `ROOTRUN-2`
  - 复现 `GOWORK=on/off` 下的 Wails bindings 差异
- `ROOTRUN-3`
  - 完成 3.3 checklist
- `ROOTRUN-4`
  - 更新 root `docs/change` 与 `docs/lessons`

## 经验 / 教训摘要

- 对 Win/Wails 这类 repo-local bindings 生成链路，workspace 联调并不总是“更完整”；当根 `go.work` 与 Win repo-local `replace` 指向不同 proto 基线时，workspace 反而会把启动链路带偏。
- 这种问题不能只看 `run-dev.ps1` 的外层报错，必须直接对比：
  - `wails generate module`
  - `$env:GOWORK='off'; wails generate module`
- 如果两者结果分叉，优先检查根 `go.work` 是否纳入了与 Win 当前开发态依赖冲突的模块。

## 可复用排查线索

- 症状
  - 根 `run-dev.ps1` 启动 Win 时，Wails 在 `Generating bindings` 失败
  - 错误文本包含 `module github.com/yttydcs/myflowhub-proto provides package .../protocol/stream and is replaced but not required`
- 触发条件
  - Win repo-local `go.mod` 依赖开发态 proto replace/worktree
  - 根 `go.work` 同时纳入了另一个 `github.com/yttydcs/myflowhub-proto` 本地模块图
  - Win 进程默认继承了 workspace 模式
- 关键词
  - `run-dev.ps1`
  - `wails generate module`
  - `GOWORK=off`
  - `replaced but not required`
  - `protocol/stream`
  - `go.work`
- 快速检查
  - 在 `repo/MyFlowHub-Win` 里先执行 `wails generate module`
  - 再执行 `$env:GOWORK='off'; wails generate module`
  - 若后者通过、前者失败，则优先检查根脚本是否需要让 Win 默认关闭 workspace

## 关键设计决策与权衡

- 决策：修根脚本默认行为，不改根 `go.work`
  - 原因：当前目标是恢复 Win 默认启动链路，而不是重写整个 workspace 联调拓扑
- 决策：把 Win 的 workspace 模式改成显式 opt-out
  - 原因：默认路径应优先稳定、可复现；需要特殊联调的人再显式打开 workspace 模式
- 决策：不把 Server / MetricsNode 一起切到 `GOWORK=off`
  - 原因：当前复现证据只指向 Win，扩大默认行为改动没有必要

## 测试与验证方式 / 结果

- `repo/MyFlowHub-Win`
  - 执行：`wails generate module`
  - 结果：失败
  - 说明：稳定复现用户报错 `module ... myflowhub-proto ... replaced but not required`
- `repo/MyFlowHub-Win`
  - 执行：`$env:GOWORK='off'; wails generate module`
  - 结果：通过
- root worktree
  - 执行：`pwsh -NoProfile -File .\scripts\run-dev.ps1 -SkipServer -SkipWin -SkipMetricsNode`
  - 结果：通过
  - 说明：脚本语法与参数解析正常

## 潜在影响与回滚方案

- 潜在影响
  - 默认 `.\scripts\run-dev.ps1` 启动 Win 时将不再继承根 `go.work`
  - 如果某些 Win 联调场景必须使用 workspace 模式，需要显式传 `-WinUseWorkspace`
- 回滚方案
  - 回退 `scripts/run-dev.ps1`
  - 回退 `docs/lessons/wails-binding-proto-drift.md`
  - 回退 `docs/lessons/README.md`
  - 回退 `docs/change/README.md`
  - 删除或回退本归档

## 子Agent执行轨迹

- 未使用子Agent
