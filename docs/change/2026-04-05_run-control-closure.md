# 2026-04-05 run-control-closure

## 变更背景 / 目标
- 用户要求在同一个 workflow 内完成此前 flow 审查中剩余的三项收口：
  - `MyFlowHub-SubProto/flow` 修通单仓 `GOWORK=off go test ./...`
  - `MyFlowHub-Server` 补齐行为级 flow round-trip 集成测试
  - `MyFlowHub-Win` 移除 `cancel_run/list_runs` 的本地 compatibility DTO，直接收敛到 shared proto 类型
- 用户已明确确认：项目尚未上线，不需要保留旧数据 / legacy flow 的兼容加载。
- 目标是在不回退现有 flow runtime 方向的前提下，把三仓验证口径统一收敛到可审计、可复跑的 `GOWORK=off` 结果。

## 具体变更内容

### `MyFlowHub-SubProto/flow`
- 将主仓当前 flow 基线同步到 workflow worktree，保留已删除 legacy load compatibility 的现状，不把运行时再退回“读旧 kind”的兼容模式。
- 对齐 `flow/go.mod` / `flow/go.sum` 的真实单仓最低依赖：
  - `github.com/yttydcs/myflowhub-core v0.4.10`
  - `github.com/yttydcs/myflowhub-proto v0.1.6`
  - `github.com/yttydcs/myflowhub-subproto/broker v0.1.1`
  - `github.com/yttydcs/myflowhub-subproto/exec v0.1.2`
- 用 `GOWORK=off` 直接验证 repo-local module graph，不再依赖 workspace `go.work` 掩盖问题。

### `MyFlowHub-Server`
- 新增 `tests/integration_flow_round_trip_test.go`。
- 测试通过真实 `hubruntime`、TCP client 注册和 flow control frame 驱动完整闭环，而不是仅验证 handler 被装配。
- 覆盖行为链路：
  - `set -> run(2x) -> status(latest) -> detail(node) -> list_runs(limit=2) -> get -> list`
- 使用最小 `compose` graph，避免引入额外 capability / trigger 依赖，让断言聚焦在 flow runtime 自身。

### `MyFlowHub-Win`
- `internal/services/flow/run_control_types.go` 不再保留本地 struct 定义，只保留 shared proto alias：
  - `CancelRunReq/Resp`
  - `ListRunsReq/Resp`
  - `RunSummary`
- `internal/services/flow/service.go`、`internal/mcp/tools.go` 及对应 tests 全部切到 `protoflow.CancelRunReq/Resp` 与 `protoflow.ListRunsReq/Resp`。
- `go.mod` 的 proto replace 调整为：
  - `replace github.com/yttydcs/myflowhub-proto => ../../worktrees/proto-server-release-align`
- 之所以不是 child-only 的相对路径，是为了保证未来合并回真实 repo root 后仍然成立。
- 为当前 nested child worktree 的验证补了 helper junction：
  - `D:\project\MyFlowHub3\worktrees\worktrees\proto-server-release-align`
  - 指向 `D:\project\MyFlowHub3\worktrees\proto-server-release-align`
  - 用途仅是让 child worktree 继承 repo-local relative replace 时也能解析到正确 proto 基线

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- none
- 原因：
  - 本轮遇到的两类核心问题已有现成 lessons，可直接复用，不需要新增新的长期文档：
    - `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`
    - `D:\project\MyFlowHub3\docs\lessons\wails-binding-proto-drift.md`

## Related requirements
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`

## Related specs
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`

## Related lessons
- `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`
- `D:\project\MyFlowHub3\docs\lessons\wails-binding-proto-drift.md`

## 对应 `plan.md` 任务映射
- `SUB-1`
  - 同步当前 flow 基线并修通 `MyFlowHub-SubProto/flow` 单仓验证
- `SRV-1`
  - 新增行为级 flow round-trip 集成测试
- `WIN-1`
  - 移除 Win-local compatibility DTO，收敛到 shared proto 类型

## 经验 / 教训摘要
- nested child worktree 下，repo-local `replace ../../worktrees/...` 是否还能从真实 repo root 成立，必须和“当前 child worktree 能否验证”分开处理。验证需要的 helper junction 可以临时挂在 workspace 层，但不能把 child-only 相对路径直接写进将来要 merge 的 `go.mod`。
- `GOWORK=off` 是这次三项收口的共同验收口径；只要切回单仓模式仍能通过，才能说明真正解决了 module floor / replace / runtime 闭环问题。
- Server 侧要回答“flow 是否真的闭环”，需要行为级 round-trip 证据；仅补默认模块集包含 handler 的 targeted test 不够。

## 可复用排查线索
- 症状：
  - `GOWORK=off go test ./...` 在 `MyFlowHub-SubProto/flow` 失败
  - `undefined: protoflow.CancelRunReq`
  - `undefined: protoflow.ListRunsReq`
  - `replacement directory ../../worktrees/... does not exist`
  - Server 只有装配级 flow tests，但缺少真实 run/status/detail/list_runs/get/list 闭环证据
- 触发条件：
  - child worktree 继承了 repo-local 相对 `replace`
  - Win 使用的 consumed proto baseline 仍缺 run-control shared type
  - flow module 当前源码依赖的上游版本已经高于 `go.mod` 中声明的最低版本
- 关键词：
  - `GOWORK=off`
  - `proto-server-release-align`
  - `CancelRunReq`
  - `ListRunsReq`
  - `integration_flow_round_trip_test.go`
  - `hubruntime`
- 快速检查：
  1. `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-SubProto\flow`
     - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  2. `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Win`
     - `$env:GOWORK='off'; go test ./internal/services/flow ./internal/mcp ./internal/mcpapp -count=1 -p 1`
     - 若先报 replace 目录不存在，检查是否已建立 `worktrees\worktrees\proto-server-release-align` helper junction
  3. `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Server`
     - `$env:GOWORK='off'; go test ./tests -run TestIntegrationFlowRoundTrip -count=1 -p 1`

## 关键设计决策与权衡
- 决策：不恢复 legacy flow 加载兼容。
  - 原因：用户已明确项目尚未上线，不需要旧数据兼容；继续保留只会拖住当前 contract 收口。
- 决策：Win 直接收敛到 shared proto 类型，而不是继续维持本地 compatibility DTO。
  - 原因：shared proto 当前 consumed baseline 已经具备 `cancel_run/list_runs` 类型，继续保留本地 DTO 只会增加 drift 面。
- 决策：`go.mod` 采用 merge-safe 的 repo-root 相对路径，并在 workspace 层补 helper junction。
  - 原因：既要保证当前 nested child worktree 可验证，也不能把将来主仓会失效的相对路径提交进代码。
- 决策：Server 以行为级 round-trip test 作为主要证据，模块装配 test 作为补充保底。
  - 原因：前者回答真实闭环，后者回答默认模块集是否仍挂载 flow handler；两者目标不同。

## 测试与验证方式 / 结果
- `MyFlowHub-SubProto/flow`
  - 执行：`$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Server`
  - 执行：`$env:GOWORK='off'; go test ./tests -run TestIntegrationFlowRoundTrip -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Server`
  - 执行：`$env:GOWORK='off'; go test ./modules/... -run 'TestDefaultHub_ContainsFlow|TestDefaultHubWithPGFlowBackendIncludesFlowHandler|TestDefaultHubWithPGFlowRunArchiveBackendIncludesFlowHandler' -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win`
  - 执行：`$env:GOWORK='off'; go test ./internal/services/flow ./internal/mcp ./internal/mcpapp -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win`
  - 执行：`$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 说明：命令仍会打印既有 `Not found: time.Time` 噪声，但退出码为 `0`，不影响本次收口验收

## 潜在影响与回滚方案
- 潜在影响：
  - `MyFlowHub-Win` 本地开发的 proto replace 现在指向 `proto-server-release-align` 这条 proto 基线；后续若切换新的 proto worktree，需要同步维护 helper junction 与 repo-local replace 的对应关系。
  - 当前 helper junction 保留在 workspace 层，便于后续在同一 workflow 内重复验证；结束 workflow 时应一并清理。
- 回滚方案：
  1. `MyFlowHub-SubProto`
     - 回退 `flow/go.mod`、`flow/go.sum` 以及本 workflow 同步进来的 flow 基线文件
  2. `MyFlowHub-Server`
     - 删除 `tests/integration_flow_round_trip_test.go`
  3. `MyFlowHub-Win`
     - 回退 `go.mod`
     - 回退 `internal/services/flow/*` 与 `internal/mcp/*` 的 shared proto 收敛改动
     - 若无需继续在当前 workflow 验证，删除 helper junction `D:\project\MyFlowHub3\worktrees\worktrees\proto-server-release-align`

## 子Agent执行轨迹
- `James` (`019d5b3e-75dd-7d80-8e29-1e949aa9f78c`)
  - 负责 `SRV-1`
  - 交付 `MyFlowHub-Server/tests/integration_flow_round_trip_test.go`
  - 主 Agent 复核并重跑所有 Server 验证
- `Confucius` (`019d5b3e-782e-79f2-a856-7fe9909617a6`)
  - 负责 `WIN-1`
  - 交付 `internal/services/flow` / `internal/mcp` 的 shared proto 收敛和 tests
  - 主 Agent 复核后修正 `go.mod` 为 merge-safe replace 路径，并补 helper junction
- 主 Agent
  - 负责 `SUB-1`
  - 负责 cross-repo 集成、最终验证、Stage 3.3 review 与 Stage 4 archive
