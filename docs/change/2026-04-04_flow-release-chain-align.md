# 2026-04-04_flow-release-chain-align

## 变更背景 / 目标
- `flow` 的 Proto / SubProto / Server 文档与运行时能力已经在主线形成了新的事实契约，但正式 patch 发布链没有同步收口。
- 初始暴露症状包括：
  - `MyFlowHub-Server` 在 `GOWORK=off` 下仍解析到旧的 `myflowhub-proto v0.1.5` 与 `myflowhub-subproto/flow v0.1.2`
  - 新 push tag 刚发布后，经由代理校验会出现假性 `unknown revision`
  - `flow/v0.1.3` 首次发布后缺少 `myflowhub-proto v0.1.6` 的 `go.sum`，无法作为完整 module 发布点
- 本次目标是在不扩大到 `auth` 等无关脏改的前提下，完成 `Proto -> SubProto/flow -> Server` 的真实 semver 发布链收口，并把排障经验沉淀进 workspace lessons。

## 具体变更内容
### MyFlowHub-Proto
- 提交：`fe2f790`
- 发布：`v0.1.6`
- 修改：
  - `protocol/flow/types.go`
    - 对齐当前 `flow` canonical contract，补齐 `cancel_run`、`detail`、`list_runs`
    - 补齐 `cron` trigger、`branch/foreach/subflow/transform` 图结构所需字段
    - 明确 `flow.run` / `flow.read` 等权限常量，供下游模块与文档统一消费

### MyFlowHub-SubProto
- 提交：
  - `1fe8eeb`
  - `3f61a6e`
- 发布：
  - 首次推送：`flow/v0.1.3`
  - 最终有效 patch：`flow/v0.1.4`
- 修改：
  - `flow/go.mod`
    - `github.com/yttydcs/myflowhub-proto`：对齐到 `v0.1.6`
    - `github.com/yttydcs/myflowhub-subproto/broker`：对齐到 `v0.1.1`
  - `flow/go.sum`
    - 补齐 `myflowhub-proto v0.1.6` 等真实发布所需校验和
  - `flow/handler.go`
  - `flow/runtime_bindings.go`
  - `flow/graph_test.go`
  - `flow/orchestrator_test.go`
  - `flow/transform_test.go`
  - `flow/trigger_test.go`
    - 收口 `transform / branch / foreach / subflow / cron` 运行时与回归测试
  - `docs/change/2026-04-03_flow-transform-node-runtime.md`
  - `docs/change/README.md`
    - 增补 repo 内运行时变更归档
- 说明：
  - `flow/v0.1.3` 已公开但不完整，因此没有改写 tag，而是继续发布更高 patch `flow/v0.1.4` 覆盖。

### MyFlowHub-Server
- 提交：
  - `bd642d8`
  - `122d34a`
- 修改：
  - `go.mod`
    - `github.com/yttydcs/myflowhub-proto`：`v0.1.5 -> v0.1.6`
    - `github.com/yttydcs/myflowhub-subproto/flow`：`v0.1.2 -> v0.1.4`
  - `go.sum`
    - 同步新的 Proto / flow 校验和
  - `docs/requirements/flow_data_dag.md`
  - `docs/specs/flow.md`
    - 把 `transform / branch / foreach / subflow / cron` 收口为当前稳定 requirements/specs
- 无额外业务逻辑改动；`Server` 本轮只做契约文档与依赖消费升级。

## Requirements impact
`updated`

## Specs impact
`updated`

## Lessons impact
`updated`

## Related requirements
- `D:\project\MyFlowHub3\worktrees\server-release-align\docs\requirements\flow_data_dag.md`

## Related specs
- `D:\project\MyFlowHub3\worktrees\server-release-align\docs\specs\flow.md`

## Related lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)

## 对应 plan.md 任务映射
- `RELALIGN-0` - 控制面与版本校验
- `RELALIGN-1` - Proto canonical contract
- `RELALIGN-2` - SubProto flow release align
- `RELALIGN-3` - SubProto stream release align if needed
  - 结论：本轮无需执行；`Server` 真实依赖验证已不需要新增 `stream` patch
- `RELALIGN-4` - Server downstream align
- `RELALIGN-5` - Review / archive / release notes

## 经验 / 教训摘要
- `go.work` 联调通过不能当作可发布证据，收口必须回到真实的 `GOWORK=off` 下游验证。
- 刚 push 的 tag 立刻走公共代理校验，容易被 proxy lag 误判成 `unknown revision`；需要先区分“代理未刷新”还是“tag 真有问题”。
- 已公开的 semver tag 如果缺少关键 `go.sum` 或 module 自测完整性，不要改写历史；直接补发更高 patch 更安全、也更可审计。
- 发布链仍应严格按依赖方向推进：`Proto -> SubProto module -> Server`。

## 可复用排查线索
- 症状：
  - `GOWORK=off go list -m` 仍解析到旧版本
  - 新 push tag 立刻校验时报 `unknown revision`
  - 下游仓库只在刚升级的某个 module 上失败，但继续追查会暴露更上游 Proto 契约或 `go.sum` 缺口
- 触发条件：
  - 上游公共 API 已在本地 worktree 落地，但 patch tag 尚未正式发布
  - 新 tag 刚推送完成，公共 proxy 还未刷新
  - 多 module 单仓中，module `go.mod/go.sum` 没有完全覆盖当前源码实际依赖
- 关键词：
  - `GOWORK=off`
  - `go list -m`
  - `GOPROXY=direct`
  - `GOPRIVATE=github.com/yttydcs/*`
  - `GONOSUMDB=github.com/yttydcs/*`
  - `unknown revision`
  - `flow/v0.1.3`
  - `flow/v0.1.4`
- 快速检查：
  - 先在真实下游设置：
    - `GOWORK=off`
    - `GOPROXY=direct`
    - `GOPRIVATE=github.com/yttydcs/*`
    - `GONOSUMDB=github.com/yttydcs/*`
  - 再执行：
    - `go list -m github.com/yttydcs/myflowhub-proto`
    - `go list -m github.com/yttydcs/myflowhub-subproto/flow`
  - 若 tag 已公开但 module 仍不完整，不改写 tag，直接发布更高 patch 后重验

## 关键设计决策与权衡
- 范围严格限制在 `flow` 发布链：
  - 不顺手带上 `MyFlowHub-SubProto/auth` 的无关脏改，避免把一次 release-chain 修复扩大成跨模块收口。
- `flow/v0.1.3` 不回收、不重写：
  - 虽然首次发布不完整，但 semver 历史一旦公开就保持只追加，改发 `flow/v0.1.4` 更符合当前 lessons 约束。
- `Server` 采用“依赖升级 + 契约文档同步”的最小消费策略：
  - 不在本轮混入额外 runtime 代码改动，降低回归面。
- 对 `Server` 验证采取“真实依赖链 + 定向模块测试”：
  - 当前全量 `go build ./...` 仍受无关 `bootstrap.SelfRegisterOptions.Dial` 问题影响，故把它记录为残留风险，而不扩展本轮 scope。

## 测试与验证方式 / 结果
- 首轮新 tag 校验环境：
  - `GOWORK=off`
  - `GOPROXY=direct`
  - `GOPRIVATE=github.com/yttydcs/*`
  - `GONOSUMDB=github.com/yttydcs/*`

### MyFlowHub-Proto
- `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过

### MyFlowHub-SubProto/flow
- `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
  - 结果：`v0.1.6`
- `go test -run 'TestValidateGraph|TestValidateTrigger|TestCronScheduleNextAfter|TestExecuteFlow|TestHandleTopic|TestHandleVarChanged|TestTryStart|TestTransform' -count=1 -p 1`
  - 结果：通过

### MyFlowHub-Server
- `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
  - 结果：`v0.1.6`
- `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/flow`
  - 结果：`v0.1.4`
- `go test ./modules/... -run 'TestDefaultHub_ContainsFlow|TestDefaultHubWithPGFlowBackendIncludesFlowHandler|TestDefaultHubWithPGFlowRunArchiveBackendIncludesFlowHandler' -count=1 -p 1`
  - 结果：通过
- `GOWORK=off go build ./...`
  - 结果：失败
  - 失败项：`hubruntime\\runtime.go:466:3: unknown field Dial in struct literal of type bootstrap.SelfRegisterOptions`
  - 判断：与本轮 flow release-chain 对齐无关，作为残留风险记录

## Code Review（3.3）结论
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计：通过（未使用子 agent）

## 潜在影响
- 正向影响：
  - `flow` 当前 canonical contract 已有正式 Proto / SubProto patch 可供下游拉取
  - `MyFlowHub-Server` 已能在真实 semver 依赖下消费 `Proto v0.1.6` 与 `flow v0.1.4`
- 注意事项：
  - 其他仍锁在旧 `flow` / `proto` patch 的下游仓库不会自动获得本轮能力
  - 新 tag 刚发布后的短时间内，如未绕开 proxy，验证结果可能出现假阴性

## 回滚方案
- 已 push 的 `v0.1.6` 与 `flow/v0.1.4` 不改写；若后续发现问题，只追加更高 patch 修复。
- `MyFlowHub-Server` 若需临时回退，可把 `go.mod/go.sum` 中的 `Proto/flow` 版本退回旧值后重新验证。
- 控制面归档若需撤回，可在 workflow 未结束前单独回退本 worktree 的 docs 提交；不影响已公开 tag。

## 子Agent执行轨迹
- 无子 agent；全部由主代理按同一 workflow 顺序完成。
