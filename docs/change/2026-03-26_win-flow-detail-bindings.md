# 2026-03-26_win-flow-detail-bindings

## 变更背景 / 目标
- Win 仓库在 `GOWORK=off` 下执行 `wails generate module` 与 `go test ./internal/services/flow` 时，因 `internal/services/flow/service.go` 直接引用 shared proto 中并不存在的 `flow.DetailReq` / `flow.DetailResp` / `flow.ActionDetail*` 而编译失败。
- 本轮目标是在不扩大依赖升级范围的前提下，恢复 Wails bindings 生成与 Go 编译路径。

## 具体变更内容
- 后端：
  - 新增 `internal/services/flow/detail_types.go`
  - 在 Win 侧补齐本地 `DetailReq` / `DetailResp` 与 `detail` / `detail_resp` action 常量，保持现有 JSON 字段契约不变。
  - 更新 `internal/services/flow/service.go`
    - `Detail` / `DetailSimple` 改为使用本地 detail 类型
    - `extractCodeMsg(...)` 增加本地 `DetailResp` 分支
  - 新增 `internal/services/flow/service_test.go`
    - 覆盖 detail 输入校验
    - 覆盖校验通过后的空 session 错误
    - 覆盖 `extractCodeMsg(...)` 的 detail 分支
- 稳定文档：
  - 更新 `docs/specs/flow-editor-run-detail.md`
  - 澄清 Win `FlowService` 可以使用本地 typed payload，只要保持 `flow.detail` 的 JSON 字段契约一致。

## Requirements impact
- none

## Specs impact
- updated

## Lessons impact
- updated

## Related requirements
- [flow-editor-run-detail.md](../requirements/flow-editor-run-detail.md)

## Related specs
- [flow-editor-run-detail.md](../specs/flow-editor-run-detail.md)

## Related lessons
- [wails-binding-proto-drift.md](../lessons/wails-binding-proto-drift.md)

## 对应 plan.md 任务映射
- `DOC-1`
  - `docs/specs/flow-editor-run-detail.md`
- `FIX-1`
  - `internal/services/flow/detail_types.go`
  - `internal/services/flow/service.go`
- `TEST-1`
  - `internal/services/flow/service_test.go`

## 经验 / 教训摘要
- 对 Win Wails binding 来说，shared proto 中尚未落地的新 req/resp 类型不应直接写进 service public method；否则 `GOWORK=off` 下会立刻体现在 bindings 生成失败。
- 当 wire 契约已经明确、但 shared proto 尚未跟上时，Win 本地 typed payload 是比“顺手升级依赖”更小、更稳的修复面。
- 需要把 `GOWORK=off` 视为 worktree 下的标准验证入口，否则会被父级 `go.work` 错误掩盖真实编译问题。

## 可复用排查线索
- 症状：
  - `wails generate module` 报 `undefined: flow.DetailReq`
  - `go test ./internal/services/flow` 报 `undefined: flow.DetailResp`
  - `service.go` 中的 `ActionDetail` / `ActionDetailResp` 无法解析
- 触发条件：
  - Win service 直接引用了 shared proto 中尚未发布/不存在的 detail 类型或 action 常量
  - 在 worktree 中执行校验时，未先切到 `GOWORK=off`
- 关键词：
  - `flow.DetailReq`
  - `flow.DetailResp`
  - `ActionDetail`
  - `wails generate module`
  - `GOWORK=off`
- 快速检查：
  - 先执行 `$env:GOWORK='off'; go test ./internal/services/flow`
  - 再执行 `$env:GOWORK='off'; wails generate module`
  - 若仍提示 detail 类型不存在，检查 `repo/MyFlowHub-Proto/protocol/flow/types.go` 是否真的定义了对应符号

## 关键设计决策与权衡
- 决策：使用 Win 本地 `DetailReq` / `DetailResp`，而不是升级 `myflowhub-proto`
  - 原因：当前本地 proto 仓库也不存在 `flow.detail` 定义，升级路径在本轮不可用。
  - 代价：未来 shared proto 补齐后，需要再决定是否收敛回共享类型。
- 决策：仅澄清 spec，不改 requirement
  - 原因：用户可见行为与 JSON 契约未变化，变化的是 Win 实现边界。
- 决策：新增最小单元测试，而不是只依赖一次性 bindings 生成
  - 原因：可以把 detail guard 固化到 Go 测试里，后续回归更早暴露。

## 测试与验证方式 / 结果
- `$env:GOWORK='off'; go test ./internal/services/flow`
  - 结果：通过
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：命令仍打印重复 `Not found: time.Time` 警告，但退出码为 `0`
- `$env:GOWORK='off'; go test ./...`
  - 结果：通过
- `git diff --check`
  - 结果：通过

## 潜在影响
- 当前 fix 只恢复 Win 编译与 bindings 生成，不证明当前 server/runtime 已实际支持 `flow.detail`。
- 后续若 shared proto 新增同名类型，需要再评估是否收敛，避免本地与共享定义长期分叉。

## 回滚方案
- 回退 `internal/services/flow/detail_types.go`
- 回退 `internal/services/flow/service.go` 对本地 detail 类型的引用
- 回退 `internal/services/flow/service_test.go`
- 回退 `docs/specs/flow-editor-run-detail.md` 的澄清段落

## 子Agent执行轨迹
- none
