# Plan - run-control-closure

## Workflow Information
- Repo: `MyFlowHub-SubProto`, `MyFlowHub-Server`, `MyFlowHub-Win`
- Branch:
  - `MyFlowHub-SubProto`: `chore/run-control-closure`
  - `MyFlowHub-Server`: `chore/run-control-closure`
  - `MyFlowHub-Win`: `chore/run-control-closure`
- Base:
  - `MyFlowHub-SubProto`: `main`
  - `MyFlowHub-Server`: `main`
  - `MyFlowHub-Win`: `main`
- Worktree:
  - Workflow root: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure`
  - `MyFlowHub-SubProto`: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-SubProto`
  - `MyFlowHub-Server`: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Server`
  - `MyFlowHub-Win`: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Win`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`: 已读。
- base/worktree confirmation:
  - 本次为跨 3 仓 workflow，已按规则在 `D:\project\MyFlowHub3\worktrees\` 下创建独立 repo worktree。
  - 主控制文档放在 workflow root `plan.md`，各 repo worktree 放最小 `todo.md` 作为归属声明。
  - 注意：`MyFlowHub-SubProto` 主仓当前有未提交 flow 相关改动；实现阶段需要把“当前主仓 flow 基线”显式同步到本次 worktree，避免丢掉已完成但未提交的 legacy-compat 删除结果。
  - 注意：`MyFlowHub-Win/go.mod` 存在 `../../worktrees/proto-stream-subproto` 的 repo-local `replace`；child worktree 下会解析到 `worktrees\worktrees\proto-stream-subproto`，实现阶段需通过最小安全方式恢复可验证路径。

### Stage 1 - Requirements Analysis
#### Goal
- 在一个 workflow 内完成 3 项收口工作：
  - `MyFlowHub-SubProto/flow` 在单仓 `GOWORK=off` 模式下可独立 `go test ./...`
  - `MyFlowHub-Server` 补齐行为级 flow round-trip 集成测试证据
  - `MyFlowHub-Win` 移除 `cancel_run/list_runs` 的本地 compatibility DTO，改为 shared proto 类型

#### Scope
- Must:
  - 不改变稳定 wire action、payload JSON shape 和权限语义。
  - 不回退当前已实现的 flow runtime / Win authoring 能力。
  - 对关键改动补测试并完成 `GOWORK=off` 验证。
- Optional:
  - 若实现中暴露出可顺手修复的小型测试 harness 缺口，可做最小补齐。
- Not In Scope:
  - 不新增新的 flow 产品能力。
  - 不重构 graph/node canonical contract。
  - 不做新的 semver 发布或 release tag 操作。

#### Use Cases
- 开发者在 `MyFlowHub-SubProto/flow` 目录单独执行 `GOWORK=off go test ./... -count=1 -p 1`，不再因 `go.sum` / 依赖声明不闭环而 setup fail。
- 开发者在 `MyFlowHub-Server` 可通过真实默认模块集验证 `flow set -> run -> status/detail/list_runs/get` 的闭环，而不是只验证 handler 是否被装配。
- Win Wails service 在 `cancel_run/list_runs` 上直接消费 shared proto 类型，减少本地 typed payload 漂移。

#### Functional Requirements
- `MyFlowHub-SubProto/flow` 必须按 repo-local `go.mod` 解析出实际需要的 `myflowhub-proto` / `myflowhub-subproto/exec` 版本，并补齐 `go.sum`。
- `MyFlowHub-Server` 的新增 flow 测试必须走行为级请求/响应链路，而不是只看 `SubProtoFlow` 是否存在。
- `MyFlowHub-Win` 的 `CancelRun*` / `ListRuns*` 方法必须改用 shared proto `flow.CancelRunReq/Resp` 与 `flow.ListRunsReq/Resp`。
- `MyFlowHub-Win` 的 MCP / frontend 调用链不能因类型收敛而退化或改 wire shape。

#### Non-functional Requirements
- 最小安全变更，不扩大无关模块写面。
- 所有验证默认使用 `GOWORK=off`，避免根 `go.work` 掩盖真实单仓问题。
- 明确记录 child worktree `replace` / release-chain 风险，不把本地联调绿灯误判成可发布证据。

#### Inputs / Outputs
- Inputs:
  - `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - `repo/MyFlowHub-Win/docs/requirements/flow-editor-visual-form.md`
  - 当前三仓源码与测试基线
- Outputs:
  - 修复后的 `SubProto/flow` module 依赖文件
  - Server 行为级 flow integration tests
  - Win 收敛后的 run-control service 类型与测试
  - workflow 归档文档

#### Edge Cases
- `MyFlowHub-SubProto/flow` 当前 worktree 基线若不继承主仓未提交 flow 改动，会把已完成的 legacy-compat 删除丢失。
- `MyFlowHub-Win` child worktree 的 repo-local `replace` 路径失效，导致 `go list` / `go test` / `wails generate module` 先因路径错误失败。
- Server 行为级测试若依赖远端 capability 或复杂 trigger，容易把测试目标变成外部依赖管理，而不是 flow 闭环本身。

#### Acceptance Criteria
- `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-SubProto\flow`
  - `GOWORK=off go test ./... -count=1 -p 1` 通过。
- `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Server`
  - 新增至少一组行为级 flow round-trip 集成测试，覆盖真实 `set -> run -> status/detail/list_runs/get` 中的关键路径。
- `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Win`
  - `CancelRunSimple` / `ListRunsSimple` 使用 shared proto 类型。
  - `GOWORK=off go test ./internal/services/flow ./internal/mcp ./internal/mcpapp` 通过。
  - 必要时 `GOWORK=off wails generate module` 通过。

#### Risks
- `cross-repo-semver-release`：`go.work` 或旧 module cache 可能掩盖真正的 semver / `go.sum` 问题。
- `wails-binding-proto-drift`：Win 若同时遇到 repo-local `replace` 失效和 shared proto 漂移，容易出现“默认构建失败但 `GOWORK=off` 行为不同”的假象。
- Server 行为级测试如果只复用 stubServer，可能仍不足以回答“DefaultHub/hubruntime 是否真的闭环”。

#### Issue List
- 无阻塞项。

### Stage 2 - Architecture Design
#### Overall Solution
- 方案 A（采用）：
  - `MyFlowHub-SubProto`
    - 先把主仓当前 flow 基线同步到 workflow worktree，再在该 worktree 内修复 `flow/go.mod` / `flow/go.sum`，用 `GOWORK=off` 直接验证 repo-local module。
  - `MyFlowHub-Server`
    - 在 `tests/` 新增基于真实 TCP server 与默认模块集的 flow round-trip integration test，优先选用 `compose` 之类无外部 capability 依赖的最小 flow，覆盖 `set/run/status/detail/list_runs/get`。
  - `MyFlowHub-Win`
    - 删除 `internal/services/flow/run_control_types.go` 的本地 DTO 定义，改为直接使用 shared proto `flow` 类型；同步 service tests、MCP tests 和必要的 Wails 验证。
- 不采用方案：
  - 只在 `MyFlowHub-Server/modules/*_test.go` 继续补“handler exists”类测试
    - 理由：无法回答真实行为闭环。
  - 继续保留 Win 本地 DTO 只改注释
    - 理由：问题本质是消费边界未收敛，不是文案问题。
  - 修改根 `go.work` 解决单仓 `go test`
    - 理由：会把单仓可验证性问题继续藏在 workspace 模式里。

#### Alternatives Considered
- 为 Server 行为级测试直接启动 `hubruntime`
  - 未采用：当前 `tests/` 已有成熟 TCP integration harness，使用真实 server.Options 更直接、变更面更小。
- 在 Win worktree 直接改 `go.mod replace`
  - 暂不作为首选：若只需临时恢复 child worktree 验证路径，优先用临时 junction，避免把 repo-local replace 结构永久改乱。

#### Module Responsibilities
- `MyFlowHub-SubProto/flow`
  - 负责 repo-local module 自洽和 flow runtime 单仓回归。
- `MyFlowHub-Server/tests`
  - 负责默认模块集上的行为级 flow 集成证据。
- `MyFlowHub-Win/internal/services/flow`
  - 负责 run-control req/resp 类型边界和传输调用。
- `MyFlowHub-Win/internal/mcp`
  - 负责 run-control MCP tool 的 JSON shape 不回归。
- `MyFlowHub-Win/frontend/src/stores/flow.ts`
  - 只做消费验证，不计划主动改业务逻辑，除非类型收敛暴露真实编译问题。

#### Data / Call Flow
- `SubProto/flow`:
  - `flow/go.mod` -> 解析 `myflowhub-proto` / `myflowhub-subproto/exec`
  - `GOWORK=off go test ./...` -> 直接验证 repo-local module graph 和 `go.sum`
- `Server`:
  - test client TCP -> `modules.DefaultHub` 注册的 flow handler
  - `set` 保存 flow -> `run` 生成 run -> `status/detail/list_runs/get` 读取结果
- `Win`:
  - `frontend/src/stores/flow.ts` -> `window.go.flow.FlowService.*Simple`
  - `internal/services/flow/service.go` -> shared proto `flow.CancelRun*` / `flow.ListRuns*`
  - `internal/mcp/tools.go` -> 相同 JSON wire shape

#### Interface Drafts
- `MyFlowHub-Win`
  - `CancelRun(ctx, sourceID, targetID, req flow.CancelRunReq) (flow.CancelRunResp, error)`
  - `CancelRunSimple(sourceID, targetID, req flow.CancelRunReq) (flow.CancelRunResp, error)`
  - `ListRuns(ctx, sourceID, targetID, req flow.ListRunsReq) (flow.ListRunsResp, error)`
  - `ListRunsSimple(sourceID, targetID, req flow.ListRunsReq) (flow.ListRunsResp, error)`
- `MyFlowHub-Server`
  - 新增 `tests/` 行为级 flow integration test，不新增产品接口。

#### Error Handling and Safety
- 不通过根 `go.work`、主仓 dirty 状态或 repo 外手工 replace 来掩盖单仓问题。
- 同步主仓 current flow baseline 时，仅同步本 workflow 需要的 flow 文件，不覆盖无关用户改动。
- 若 Win child worktree 的 `replace ../../worktrees/proto-stream-subproto` 失效，优先用临时路径修复验证链路，完成后清理。

#### Performance and Testing Strategy
- `SubProto`
  - 关键验证：`GOWORK=off go test ./... -count=1 -p 1`
- `Server`
  - 关键验证：`GOWORK=off go test ./tests -run TestIntegrationFlow -count=1 -p 1`
  - 补充保底：`GOWORK=off go test ./modules/... -run 'TestDefaultHub_ContainsFlow|TestDefaultHubWithPGFlowBackendIncludesFlowHandler|TestDefaultHubWithPGFlowRunArchiveBackendIncludesFlowHandler' -count=1 -p 1`
- `Win`
  - `GOWORK=off go test ./internal/services/flow ./internal/mcp ./internal/mcpapp`
  - 必要时 `GOWORK=off wails generate module`

#### Extensibility Design Points
- Server 新测试采用最小 flow graph，后续可以平滑扩展到 `cancel_run`、trigger 或 archive 场景。
- Win 类型收敛后，后续新增 flow action 时优先检查 shared proto，再决定是否真的需要本地 typed payload。
- `SubProto` 的 `GOWORK=off` 门禁继续作为 release-chain 基线，不与 workspace 联调结果混淆。

#### Issue List
- 无阻塞项。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标：
  - 在同一 workflow 内完成 `SubProto` 单仓验证修复、Server 行为级 flow 集成测试补齐、Win run-control shared proto 收敛。
- 当前状态：
  - `SubProto/flow` 的新 worktree 仍停留在 `HEAD` 基线，`GOWORK=off go test ./...` 直接因缺少 `go.sum` 条目失败。
  - `Server` 已有装配级 flow tests 和单条轻量 handler test，缺少真实 round-trip。
  - `Win` 仍保留 `run_control_types.go` 本地 compatibility DTO，且 child worktree 的 repo-local `replace` 当前失效。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验完成。
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`（优先复用已有 lessons；Stage 4 再判断是否需要新增）
- Stable truth:
  - requirements/specs 继续以 `repo/MyFlowHub-Server/docs/*` 与 `repo/MyFlowHub-Win/docs/*` 为准
- Workflow results:
  - 归档到 workspace `docs/change/`
- Reusable troubleshooting knowledge:
  - 优先复用 `docs/lessons/cross-repo-semver-release.md`
  - 优先复用 `docs/lessons/wails-binding-proto-drift.md`

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`
  - `D:\project\MyFlowHub3\docs\lessons\wails-binding-proto-drift.md`

#### Executable Task List
- [x] `SUB-1` 继承当前 flow 基线并修通 `MyFlowHub-SubProto/flow` 单仓 `GOWORK=off go test`
- [x] `SRV-1` 为 `MyFlowHub-Server` 增加行为级 flow round-trip 集成测试
- [x] `WIN-1` 将 `MyFlowHub-Win` 的 `cancel_run/list_runs` 改为 shared proto 类型并完成验证

#### Task Details
##### `SUB-1` - Sync flow baseline and repair standalone module validation
- Owner: 主 Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-SubProto`
- Plan Path: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\plan.md`
- Goal:
  - 把主仓当前 flow 基线安全同步到本次 worktree，并修通 `flow` module 的 repo-local `GOWORK=off` 测试。
- Files / Modules:
  - `MyFlowHub-SubProto/flow/go.mod`
  - `MyFlowHub-SubProto/flow/go.sum`
  - `MyFlowHub-SubProto/flow/*`（仅当同步主仓当前 flow 基线需要）
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-SubProto\flow\**`
- Acceptance:
  - `GOWORK=off go test ./... -count=1 -p 1` 通过
  - 不回退 call-only / drop-legacy-compat 方向
- Test Points:
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 必要时定向 `TestLoadFlowsFromDiskSkipsLegacyKinds`
- Rollback:
  - 回退 `flow/go.mod`、`flow/go.sum` 和本任务新增/同步的 flow 文件

##### `SRV-1` - Add behavior-level flow round-trip integration tests
- Owner: 子 Agent 或主 Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Server`
- Plan Path: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\plan.md`
- Goal:
  - 在默认模块集上提供真实 flow 闭环测试证据，至少覆盖 `set -> run -> status/detail/list_runs/get`。
- Files / Modules:
  - `MyFlowHub-Server/tests/*`
  - 如有必要，最小调整 `tests` helper
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Server\tests\**`
- Acceptance:
  - 新测试通过且不是纯 handler-presence 测试
  - 不改产品 runtime 逻辑，除非测试暴露真实 bug
- Test Points:
  - `GOWORK=off go test ./tests -run TestIntegrationFlow -count=1 -p 1`
- Rollback:
  - 回退本任务新增的测试文件与 helper 调整

##### `WIN-1` - Replace Win-local run-control DTOs with shared proto types
- Owner: 子 Agent 或主 Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Win`
- Plan Path: `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\plan.md`
- Goal:
  - 删除 `cancel_run/list_runs` 本地 compatibility DTO，直接消费 shared proto `flow` 类型。
- Files / Modules:
  - `MyFlowHub-Win/internal/services/flow/run_control_types.go`
  - `MyFlowHub-Win/internal/services/flow/service.go`
  - `MyFlowHub-Win/internal/services/flow/service_test.go`
  - `MyFlowHub-Win/internal/mcp/*`
  - 必要时 Wails 生成态相关验证路径
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Win\internal\services\flow\**`
  - `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Win\internal\mcp\**`
- Acceptance:
  - `CancelRun*` / `ListRuns*` 改用 shared proto 类型
  - Go tests 与 MCP tests 通过
  - 必要时 Wails 绑定生成通过
- Test Points:
  - `GOWORK=off go test ./internal/services/flow ./internal/mcp ./internal/mcpapp`
  - 必要时 `GOWORK=off wails generate module`
- Rollback:
  - 回退 run-control 类型收敛相关文件

#### Dependencies
- `SUB-1` 不依赖其他任务。
- `SRV-1` 与 `WIN-1` 可并行，但最终集成和验证由主 Agent 统一完成。
- `WIN-1` 的验证依赖 child worktree `replace` 路径先恢复到可用状态。

#### Risks and Notes
- `MyFlowHub-SubProto` 主仓 dirty 状态只可读，不得在主仓直接实现。
- `MyFlowHub-Win` 可能需要临时 junction 恢复 `../../worktrees/proto-stream-subproto` 路径；若使用，必须在最终归档里记录并清理。
- 本 workflow 不做根 workspace `go.work` 变更。

#### Parallelism Assessment
- 结论：`SRV-1` 与 `WIN-1` 写集可拆分，适合在 Stage `3.2` 使用子 Agent 并行执行。
- 不并行的部分：
  - `SUB-1` 需要主 Agent 先处理，因为它涉及当前主仓 dirty flow 基线的继承判断与 module graph 修复，属于关键路径。
- 计划并行方式：
  - 主 Agent：负责 `SUB-1`、总体集成、最终验证
  - 子 Agent A：负责 `SRV-1`
  - 子 Agent B：负责 `WIN-1`

#### Issue List
- 无阻塞项。

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### `SUB-1` Result
- 主 Agent 将 `repo/MyFlowHub-SubProto` 当前 flow 基线安全同步到 workflow worktree，保留已移除 legacy load 兼容的方向，不回滚到旧数据兼容逻辑。
- `flow/go.mod` / `flow/go.sum` 对齐到实际单仓最低依赖：
  - `github.com/yttydcs/myflowhub-core v0.4.10`
  - `github.com/yttydcs/myflowhub-proto v0.1.6`
  - `github.com/yttydcs/myflowhub-subproto/broker v0.1.1`
  - `github.com/yttydcs/myflowhub-subproto/exec v0.1.2`
- 验证：
  - `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-SubProto\flow`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过

#### `SRV-1` Result
- 子 Agent `James` 负责 `MyFlowHub-Server/tests/integration_flow_round_trip_test.go`。
- 新增行为级 flow round-trip 集成测试，使用真实 `hubruntime`、TCP 注册和 flow control frame，不再停留在 handler-presence 级别。
- 覆盖链路：
  - `set -> run(2x) -> status(latest) -> detail -> list_runs -> get -> list`
- 采用最小 `compose` flow，避免外部 capability 依赖把测试目标转移到无关模块。
- 主 Agent 已复核实现并重新执行验证：
  - `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Server`
  - `$env:GOWORK='off'; go test ./tests -run TestIntegrationFlowRoundTrip -count=1 -p 1`
  - 结果：通过

#### `WIN-1` Result
- 子 Agent `Confucius` 负责 `internal/services/flow` 与 `internal/mcp` 的 run-control 类型收敛。
- `cancel_run/list_runs` 的 Win-local compatibility DTO 已移除为 shared proto alias：
  - `CancelRunReq/Resp -> protoflow.CancelRunReq/Resp`
  - `ListRunsReq/Resp -> protoflow.ListRunsReq/Resp`
  - `RunSummary -> protoflow.RunSummary`
- `service.go`、`tools.go`、对应 tests 已切到 shared proto 类型，保留现有 JSON wire shape。
- 主 Agent 在集成时修正了一个 merge-safe 问题：
  - 不能把 `go.mod` 的 `replace` 改成只对 child worktree 成立、对真实 repo root 失效的相对路径。
  - 最终采用 `replace github.com/yttydcs/myflowhub-proto => ../../worktrees/proto-server-release-align`。
  - 为 child worktree 验证补了 helper junction：
    - `D:\project\MyFlowHub3\worktrees\worktrees\proto-server-release-align -> D:\project\MyFlowHub3\worktrees\proto-server-release-align`
- 验证：
  - `D:\project\MyFlowHub3\worktrees\chore-run-control-closure\MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/flow ./internal/mcp ./internal/mcpapp -count=1 -p 1`
  - `结果：通过`
  - `$env:GOWORK='off'; wails generate module`
  - `结果：通过`

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - `SUB-1/SRV-1/WIN-1` 均已完成，且未重新引入旧数据兼容。
- 架构合理性：通过
  - `SubProto` 用单仓依赖修正解决真实 module graph 问题；`Server` 用行为级 integration test；`Win` 直接收敛到 shared proto 类型。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 本轮仅增加测试与类型收敛，没有引入新的热路径或多余 I/O。
- 可读性与一致性：通过
  - `Win` run-control 边界统一到 shared proto；`Server` 测试使用最小 flow graph，断言链路直接对应行为目标。
- 可扩展性与配置化：通过
  - `Server` round-trip 测试可继续扩展到 `cancel_run` / archive / trigger；`Win` 后续新增 run-control action 可继续优先复用 shared proto。
- 稳定性与安全：通过
  - 验证全部在 `GOWORK=off` 下执行；`go.mod replace` 修正为 merge-safe 路径；child worktree 路径问题通过 workspace helper junction 隔离。
- 测试覆盖情况：通过
  - `SubProto`：单仓 `go test ./...`
  - `Server`：新增行为级 `TestIntegrationFlowRoundTrip`，并补跑默认模块集 targeted tests
  - `Win`：service / MCP / runtime go tests + `wails generate module`
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - `James` 仅写 `MyFlowHub-Server/tests/**`
  - `Confucius` 仅写 `MyFlowHub-Win/internal/services/flow/**`、`internal/mcp/**`、`go.mod`
  - 主 Agent 统一做结果复核、冲突处理、最终验证与归档

### Stage 4 - Change Archive
- 使用 `$m-docs` 完成路由检查与归档。
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
  - 原因：本轮复用已有 lessons 即可覆盖核心排查路径：
    - `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`
    - `D:\project\MyFlowHub3\docs\lessons\wails-binding-proto-drift.md`
- Archive:
  - `D:\project\MyFlowHub3\docs\change\2026-04-05_run-control-closure.md`
- Index updates:
  - `D:\project\MyFlowHub3\docs\change\README.md`

阻塞：否
Stage 4 已完成
等待用户确认是否结束 workflow
