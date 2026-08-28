# 2026-04-04 flow-completeness-repair

## 变更背景 / 目标
- 基于上一轮 `flow` 完备性审查，当前主缺口已收敛为四类：
  - Win / MCP 缺少 `cancel_run` 与 `list_runs` 消费入口
  - Win deployments 对远端 `GetSimple` 失败时静默降级
  - Win `detail_types.go` 注释已过时
  - Server 缺少更直接的 flow 组装证据
- 本轮目标是在不触碰用户正在推进的 broad contract 变更前提下，修复这些可独立闭环的问题，并把残余验证阻塞显式归因。

## 具体变更内容

### Win run control / history surface
- `MyFlowHub-Win/internal/services/flow`
  - 新增 Win 本地兼容 `CancelRunReq/Resp`、`ListRunsReq/Resp`、`RunSummary` 与对应 action 常量。
  - `service.go` 补齐 `CancelRun*` / `ListRuns*`，并将 `extractCodeMsg` 扩展到新响应类型。
  - 保持 shared proto JSON 契约不变，但不再强依赖当前基线中不存在的 proto `cancel_run/list_runs` 类型。
- `MyFlowHub-Win/internal/mcpapp/runtime.go`
  - 补齐 `FlowCancelRun` / `FlowListRuns` 转发。
- `MyFlowHub-Win/internal/mcp/tools.go`
  - MCP backend/tool surface 新增 `myflowhub_flow_cancel_run` 与 `myflowhub_flow_list_runs`。
  - 参数归一化改用 Win 本地兼容类型，避免把未发布 proto 类型泄露进当前 worktree 编译链。
- `MyFlowHub-Win/frontend`
  - `stores/flow.ts` 新增 `listRunsFlow` / `cancelRunFlow`、`runHistory` / `runHistoryLoading`。
  - `runFlow` / `cancelRunFlow` 现在等待 follow-up `status/history` refresh 完成后再返回，但使用 `Promise.allSettled` 保持次级刷新失败不覆盖主请求成功语义。
  - `FlowEditorToolbar.vue`、`FlowEditorWindow.vue` 暴露 run history 与 cancel run 可见入口，并新增 selected-run 下拉。
  - `FlowEditorWindow.test.ts` 对齐新的 toolbar 事件和 window 行为。

### Win deployment enrichment / detail hygiene
- `frontend/src/stores/flowProjects.ts`
  - `GetSimple` 失败时不再静默吞掉 trigger enrichment 失败，而是保留 `triggerError`。
- `frontend/src/pages/Flow.vue`
  - 在部署列表显式展示 `Trigger details unavailable: ...`。
- `internal/services/flow/detail_types.go`
  - 注释修正为 Win 本地兼容 typed payload 语义。

### Server assembly evidence
- `MyFlowHub-Server/modules/hub_test.go`
  - 新增 `TestDefaultHub_ContainsFlow`。
- `MyFlowHub-Server/modules/defaultset/state_backends_test.go`
  - 新增默认 PG flow backend 与 run-archive backend 装配包含 flow handler 的 targeted tests。

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- updated

## Related requirements
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`

## Related specs
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\docs\specs\protocol_map.md`

## Related lessons
- `D:\project\MyFlowHub3\docs\lessons\wails-binding-proto-drift.md`
- `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`

## 对应 plan / todo 任务映射
- `FLOW-REPAIR-1`
  - Win / MCP `cancel_run` / `list_runs` surface
  - local compatibility payload restoration
  - window / toolbar / store wiring
- `FLOW-REPAIR-2`
  - deployment trigger enrichment error visibility
- `FLOW-REPAIR-3`
  - detail local type comment hygiene
- `FLOW-REPAIR-4`
  - Server flow assembly tests
- `FLOW-REPAIR-5`
  - targeted validation, code review, archive

## 经验 / 教训摘要
- Win Wails binding 面如果直接引用 shared proto 中尚未发布的 req/resp/action，会在 `GOWORK=off` 下暴露真实编译失败；这类场景应优先使用 Win 本地兼容 typed payload，保持 JSON 契约不变。
- 控制 worktree 下再嵌套 repo-specific child worktree 时，repo-local `replace ../../worktrees/...` 容易解析到不存在的 `worktrees/worktrees/...`。验证前要先判断这是路径问题还是代码问题。
- `Server/defaultset` 的 flow 组装证据可以通过 targeted tests 强化，但真正执行仍受 `Proto -> SubProto -> Server` 发布链一致性约束。

## 可复用排查线索
- 症状：
  - `undefined: flow.CancelRunReq`
  - `undefined: flow.ListRunsReq`
  - `replacement directory ../../worktrees/proto-stream-subproto does not exist`
  - `no required module provides package github.com/yttydcs/myflowhub-proto/protocol/stream`
- 触发条件：
  - Win child worktree 继承了 repo-local 相对 `replace`
  - 当前 consumed proto baseline 未包含 `cancel_run/list_runs/detail` 新类型
  - Server 在 `GOWORK=off` 下落回真实 semver 依赖链
- 关键词：
  - `GOWORK=off`
  - `proto-stream-subproto`
  - `flow.CancelRunReq`
  - `flow.ListRunsReq`
  - `protocol/stream`
  - `RunArchiveStore`
- 快速检查：
  1. 在 Win child worktree 先执行 `$env:GOWORK='off'; go test ./internal/services/flow ./internal/mcp ./internal/mcpapp`
  2. 若先报 replace 目录不存在，确认 `go.mod` 的相对路径在当前嵌套 worktree 是否仍成立
  3. 若 replace 可达后仍报 `undefined: flow.*`，检查是否应回退为 Win 本地兼容 typed payload
  4. 在 Server child worktree 执行 `$env:GOWORK='off'; go test ./modules/... -run 'TestDefaultHub_ContainsFlow|TestDefaultHubWithPGFlowBackendIncludesFlowHandler|TestDefaultHubWithPGFlowRunArchiveBackendIncludesFlowHandler' -count=1`
  5. 若失败停在 `protocol/stream` / `RunArchiveStore` / `ActionCancelRun`，优先按发布链问题处理，不要误判为新增测试断言失败

## 关键设计决策与权衡
- 决策：Win `cancel_run/list_runs` 继续使用本地兼容 typed payload，而不是修改 shared proto 依赖。
  - 原因：用户当前正在推进 broader proto / docs 收口，本轮不能覆盖那条线；Win 只需要对齐当前稳定 JSON 契约并恢复可编译 surface。
- 决策：`runFlow/cancelRunFlow` 等待 follow-up refresh 完成再返回。
  - 原因：这样 `FlowEditorWindow` 的 busy 状态与实际状态刷新一致，避免 UI 先解锁、历史和状态仍旧旧值。
- 决策：Server 仅补装配证据测试，不对当前 semver 发布链做计划外对齐。
  - 原因：当前阻塞已经超出本轮 repair 范围，且需要更广的 Proto/SubProto 发布链协调。

## 测试与验证方式 / 结果
- Win frontend:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-review\MyFlowHub-Win\frontend`
  - `npx vitest run src/stores/flow.test.ts src/stores/flowProjects.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：通过（3 files, 53 tests）
- Win Go:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-review\MyFlowHub-Win`
  - `GOWORK=off go test ./internal/services/flow ./internal/mcp ./internal/mcpapp`
  - 结果：通过
  - 说明：验证前临时在 `D:\project\MyFlowHub3\worktrees\worktrees\proto-stream-subproto` 创建 junction 以满足 child worktree 下的 repo-local `replace`；验证后已移除。
- Server Go:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-review\MyFlowHub-Server`
  - `GOWORK=off go test ./modules/... -run 'TestDefaultHub_ContainsFlow|TestDefaultHubWithPGFlowBackendIncludesFlowHandler|TestDefaultHubWithPGFlowRunArchiveBackendIncludesFlowHandler' -count=1`
  - 结果：未通过
  - 阻塞：
    - `github.com/yttydcs/myflowhub-subproto/stream@v0.1.0` 需要 `github.com/yttydcs/myflowhub-proto/protocol/stream`
    - 当前本地 / 已发布模块链也未形成同时包含 `protocol/stream`、flow run-archive API、flow `cancel/detail/list_runs` 类型的统一基线
  - 结论：未观测到新增 server tests 本身的断言失败，当前仍是发布链阻塞

## 潜在影响与回滚方案
- 潜在影响：
  - Win run control surface 现在对当前 proto 基线保持本地兼容；未来 shared proto 发布对齐后，需要统一切回 canonical shared types，避免长期双轨。
  - `FlowEditorWindow` 的 run/cancel 按钮 busy 时间现在覆盖 follow-up refresh，行为更稳，但如果上游刷新异常，结束时间会晚于之前的 fire-and-forget 版本。
- 回滚方案：
  1. 回退 `MyFlowHub-Win/internal/services/flow` 的 `cancel_run/list_runs` compatibility types 与 service wiring
  2. 回退 `internal/mcp*`、`frontend/src/stores/flow.ts`、`FlowEditorToolbar.vue`、`FlowEditorWindow.vue`、`Flow.vue`
  3. 若只需回退装配证据，可单独回退 `MyFlowHub-Server/modules/*_test.go`

## 子Agent执行轨迹
- delegated Win worker（Maxwell）
  - 完成 `FLOW-REPAIR-1/2/3` 的主体改动：service/runtime/MCP/store/UI/flowProjects/comment fixes
- delegated Server worker
  - 完成 `FLOW-REPAIR-4`：`modules/hub_test.go` 与 `modules/defaultset/state_backends_test.go`
- delegated validation worker（Newton）
  - 完成 `FLOW-REPAIR-5` 的环境诊断与阻塞澄清，不修改 tracked repo 文件
- main agent
  - 集成子 agent 结果
  - 补 `FlowEditorWindow.test.ts`
  - 修正 store refresh 返回语义
  - 把 Win Go surface 从不存在的 shared proto 类型回退为本地兼容 typed payload
  - 复跑 targeted validation 并完成归档
