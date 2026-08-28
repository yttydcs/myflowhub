# 2026-04-04_flow-completeness-review

## 变更背景 / 目标

- 用户要求按 `$m-autoflow` 对当前 `MyFlowHub3` 的 `flow` 能力做一次完备性审查，并允许使用子 agent。
- 本次不是实现 workflow，而是只读审查 workflow，目标是回答：
  - `flow` 的稳定契约是否收口
  - 后端 runtime 是否基本闭环
  - Win surface 是否完备
  - 现有测试与归档证据是否足以支撑“已完备”的结论

## 具体变更内容

### 新增 / 修改

- 新增本审查归档：
  - `docs/change/2026-04-04_flow-completeness-review.md`
- 更新变更索引：
  - `docs/change/README.md`
- 本轮没有修改任何产品代码、requirements 或 specs。

### 审查范围

- 稳定 requirements / specs：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
- 协议与实现：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Proto\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto\flow\*`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\modules\*`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\internal\services\flow\*`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\frontend\src\stores\flow*.ts`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\frontend\src\pages\Flow.vue`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\frontend\src\windows\FlowEditorWindow.vue`
- 现有测试与历史证据：
  - `MyFlowHub-SubProto/flow/*_test.go`
  - `MyFlowHub-Server/modules/*_test.go`
  - `MyFlowHub-Server/hubruntime/options_test.go`
  - `MyFlowHub-Win/internal/services/flow/service_test.go`
  - `MyFlowHub-Win/frontend/src/stores/flow.test.ts`
  - `MyFlowHub-Win/frontend/src/stores/flowProjects.test.ts`
  - `MyFlowHub-Win/frontend/src/windows/FlowEditorWindow.test.ts`
  - flow 相关 `docs/change/*.md`

## 审查结论

- 总结论：`flow` 目前应判为 `partial`，不是 `complete`。

### 契约面

- `set/delete/run/cancel_run/status/detail/list_runs/list/get` 这条主协议在 Server spec、Proto 和 workspace `protocol_map` 上已经基本一致。
- `cancel_run/detail/list_runs` 的 req/resp 结构、`flow.set/delete/run/read` 权限，以及 `max_active_runs` / `trigger.dedup_window_ms` 等 top-level wire 字段都已经进入 Proto。
- 但高阶节点与图约束仍然主要存在于 `repo/MyFlowHub-Server/docs/specs/flow.md` 与 `repo/MyFlowHub-Win/docs/specs/flow-editor-visual-form.md`：
  - `Node.Kind` 仍是 `string`
  - `Node.Spec` 仍是 `json.RawMessage`
  - `protocol_map.md` 只能同步 action / payload type / permission 常量，无法表达 graph / node contract
- 这意味着 action contract 已较完整，但 graph / node canonical contract 仍然偏 `spec-heavy`，契约面不能单独判为 complete。

### Runtime / Server 面

- `MyFlowHub-SubProto/flow` 已覆盖：
  - run control：`run/cancel_run/status/detail/list_runs/list/get`
  - trigger：`interval/cron/event/var_changed`
  - runtime 语义：`max_active_runs`、`dedup_window_ms`、`retry_backoff_ms`
  - archive：file / pg backend 与 retained reload
  - 高阶节点：`transform/branch/foreach/subflow`
- `MyFlowHub-Server` 默认装配也已有 targeted evidence：
  - `modules/hub_test.go` 覆盖默认集合包含 flow handler
  - `modules/defaultset/state_backends_test.go` 覆盖 PG flow backend / PG flow run archive backend 仍会包含 flow handler
  - `hubruntime/options_test.go` 覆盖默认 auth perms 含 `flow.run` / `flow.read`
- 但 runtime / integration 仍有两个未收口点：
  - `go test github.com/yttydcs/myflowhub-subproto/flow -run TestLoadFlowsFromDiskKeepsLegacyKindsForCompatibility -count=1` 在当前 workspace 模式下失败，说明 legacy kind compatibility 这条稳定性承诺未闭环
  - `MyFlowHub-Server` 侧目前看到的 flow 相关测试仍以“handler 被装进默认集合”和“backend 组合不丢 handler”为主，缺少通过 `DefaultHub` / `hubruntime` 做行为级 flow round-trip 的集成测试

### Win 面

- 当前 Win 应判为 `complete`，不再是旧结论里的缺口。
- authoring 已闭环：
  - docs 已定义 ordinary mode 的高级节点覆盖、deploy trigger、`max_active_runs`、`dedup_window_ms`
  - `flow.ts` / `flowProjects.ts` 已支持严格校验与 round-trip
  - `FlowNodeInspector` / `FlowEditorWindow` 已提供对应普通模式与 `foreach.body` 可视化入口
- run-consumption 也已闭环：
  - `internal/services/flow/service.go` 暴露 `RunSimple/CancelRunSimple/StatusSimple/ListRunsSimple/ListSimple/GetSimple/DetailSimple`
  - `frontend/src/stores/flow.ts` 已维护 `runHistory`、`statusRunId`、`nodeDetail`
  - `FlowEditorToolbar` 已暴露 `Run / Refresh Status / Run History / Cancel Run`
  - `FlowEditorWindow.vue` 已把这些动作接到真实用户入口，并提供 selected-run 下拉
  - `Current Deployments` 页负责 `list/get` 视图；run-control 则集中在 editor window
- 这一点与较早的旧审查草稿不同，应以 2026-04-04 当前源码为准。

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`

## Related specs

- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\docs\specs\protocol_map.md`

## Related lessons

- `none`

## 对应 plan.md 任务映射

- `FLOW-AUDIT-1` -> 契约面审查
- `FLOW-AUDIT-2` -> runtime / Server 装配与测试面审查
- `FLOW-AUDIT-3` -> Win surface 审查
- `FLOW-AUDIT-4` -> 证据整合与结论输出

## 经验 / 教训摘要

- 审查 `flow` 完备性时，不能只看 `Proto` 或 `protocol_map.md`。
- 当前主链路的真实状态是：
  - 顶层 action contract 较完整
  - runtime 与 Win surface 已明显成熟
  - 但 node / graph canonical contract 仍然分散在 specs
  - runtime 的模块级验证仍暴露失败测试
- 较早的 `flow-completeness-review` / `flow-completeness-repair` 草稿若还停留在 workspace 根的未提交改动中，只能当线索，不能直接复用结论。

## 可复用排查线索（症状 / 触发条件 / 关键词 / 快速检查）

- 症状：
  - action 看起来都齐了，但仍不敢判 `complete`
  - Win 旧归档说缺 `cancel_run/list_runs`，当前源码却已有
  - SubProto flow 单仓或 workspace 测试没有全部通过
- 触发条件：
  - 只按 `protocol_map.md` 判断 contract completeness
  - 只按旧 `docs/change` 结论判断 Win surface
  - 把 handler presence test 误当成 assembly behavior coverage
- 关键词：
  - `cancel_run`
  - `list_runs`
  - `max_active_runs`
  - `dedup_window_ms`
  - `transform`
  - `branch`
  - `foreach`
  - `subflow`
  - `DefaultHub`
  - `TestLoadFlowsFromDiskKeepsLegacyKindsForCompatibility`
- 快速检查：
  - 先看 `repo/MyFlowHub-Server/docs/specs/flow.md` 是否已定义目标 action / node / runtime 语义
  - 再看 `repo/MyFlowHub-Proto/protocol/flow/types.go` 是否只覆盖 top-level wire
  - 再看 `repo/MyFlowHub-SubProto/flow/*_test.go` 是否真有对应 runtime test
  - 再看 `repo/MyFlowHub-Win/internal/services/flow/service.go`、`frontend/src/stores/flow.ts`、`frontend/src/windows/FlowEditorWindow.vue` 是否真的暴露用户入口
  - 最后补跑当前主线测试，而不是只看历史归档

## 关键设计决策与权衡

- 本轮把“完备性”定义成系统级闭环，而不是“功能大体存在”：
  - 好处：能把 contract / runtime / Win / test evidence 四层同时纳入
  - 代价：结论更保守，不会因为某一层很强就直接给 `complete`
- Win 结论按当前源码而非旧草稿重判：
  - 好处：避免把已修复的问题继续记成缺口
  - 代价：需要额外核对 editor window、toolbar 与 tests
- runtime 结论同时纳入测试健康度：
  - 好处：能区分“代码里看起来有”和“当前真能稳定验证”
  - 代价：一条 failing test 就足以把整体从 `complete` 压回 `partial`

## 测试与验证方式 / 结果

- 只读证据审查：
  - requirements / specs / source / tests / historical changes
  - 结果：完成
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server`
  - `$env:GOWORK='off'; go test ./modules/... -run 'TestDefaultHub_ContainsFlow|TestDefaultHubWithPGFlowBackendIncludesFlowHandler|TestDefaultHubWithPGFlowRunArchiveBackendIncludesFlowHandler' -count=1 -p 1`
  - 结果：通过
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/flow -count=1`
  - 结果：通过
- `D:\project\MyFlowHub3\repo\MyFlowHub-Win\frontend`
  - `npx vitest run src/stores/flow.test.ts src/stores/flowProjects.test.ts src/windows/FlowEditorWindow.test.ts`
  - 结果：通过（3 files, 53 tests）
- `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto\flow`
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：失败
  - 原因：当前模块缺少 `go.sum` 依赖条目，不能独立通过模块级验证
- `D:\project\MyFlowHub3`
  - `go test github.com/yttydcs/myflowhub-subproto/flow -run TestLoadFlowsFromDiskKeepsLegacyKindsForCompatibility -count=1`
  - 结果：失败
  - 失败项：`flow_id_test.go:334 expected legacy flow to be loaded for runtime compatibility`
- `D:\project\MyFlowHub3`
  - `go test github.com/yttydcs/myflowhub-subproto/flow -run TestFlowDeleteFileFailureKeepsState -count=1`
  - 结果：通过

## 潜在影响与回滚方案

- 潜在影响：
  - 本次没有产品代码改动，不影响 runtime 行为。
  - 但审查结论会直接影响是否把 `flow` 对外表述为“已完备”。
  - 若忽略本次发现，容易高估：
    - contract 收口程度
    - runtime 验证健康度
    - release-ready 程度
- 回滚方案：
  1. 删除 `docs/change/2026-04-04_flow-completeness-review.md`
  2. 恢复 `docs/change/README.md` 的索引项

## 子Agent执行轨迹

- `FLOW-AUDIT-1 / Contract Coverage`
  - explorer 完成
  - 结论：顶层 action contract 基本完整，但 graph / node canonical contract 仍然 `spec-heavy`
- `FLOW-AUDIT-3 / Win Surface Coverage`
  - explorer 完成
  - 结论：Win surface 为 `complete`
- `FLOW-AUDIT-2 / Runtime / Server Coverage`
  - explorer 未在时限内返回最终结果
  - 由主 agent 继续完成源码审查与测试验证
