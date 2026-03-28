# wails-binding-proto-drift

## Summary
- 当 Win Wails service 直接引用 shared proto 中尚未存在的新 req/resp 类型时，`GOWORK=off` 下的 `go test` 和 `wails generate module` 会直接编译失败。此类问题可通过 Win 本地 typed payload 保持 JSON 契约不变并隔离 proto 漂移。
- 另一类常见变体是：根 workspace `go.work` 把 Win 拉进了与 repo-local 依赖声明不一致的模块图，导致默认 `wails generate module` 失败，而同目录下 `GOWORK=off` 却能通过。

## Lookup Hints
- `undefined: flow.DetailReq`
- `undefined: flow.DetailResp`
- `undefined: flow.ActionDetail`
- `wails generate module`
- `GOWORK=off`
- `myflowhub-proto`
- `module github.com/yttydcs/myflowhub-proto provides package`
- `replaced but not required`
- `protocol/stream`

## Symptoms
- `wails generate module` 在 Go 编译阶段报未定义符号。
- `go test ./internal/services/<module>` 报 shared proto 中某个 req/resp 类型不存在。
- 开发者先看到 worktree 不在 `go.work` 的错误，切到 `GOWORK=off` 后才暴露真实缺失符号。
- `wails generate module` 在默认 `go.work` 模式下报 `module github.com/yttydcs/myflowhub-proto provides package .../protocol/stream and is replaced but not required`，但切到 `GOWORK=off` 后通过。

## Impact
- Wails bindings 无法生成。
- 前端依赖的 `frontend/wailsjs` 无法刷新。
- 相关 Go 包无法通过基础编译验证。

## Trigger Conditions
- 新增或修改 Win service public method 时，直接使用了 shared proto 中尚未发布的类型或常量。
- worktree 校验未显式使用 `GOWORK=off`，导致父级 `go.work` 先拦截真实错误。
- 根 workspace `go.work` 把 `repo/MyFlowHub-Proto` 等模块纳入联调，而 Win repo-local `go.mod` 同时又依赖另一个开发态 proto replace/worktree。

## Root Cause
- Win 仓库的实现节奏先于 shared proto 基线；service 层把“未来 proto 类型”直接暴露到当前可编译接口，导致 bindings 生成和 Go 编译都依赖一个不存在的符号集。
- 另一条根因是 workspace 模式下的模块图被根 `go.work` 污染：Win 默认继承 workspace 后，不再按 repo-local `go.mod` 的单模块图解析 Wails bindings 依赖，导致“本仓库可编译”和“根脚本启动失败”出现分叉。

## Investigation Trail
- 先在 worktree 中执行 `wails generate module`，看到父级 `go.work` 模块外错误。
- 切换到 `$env:GOWORK='off'` 后复现真实编译失败。
- 对 `repo/MyFlowHub-Proto/protocol/flow/types.go` 与当前 server 代码树做搜索，确认不存在 `DetailReq` / `DetailResp` / `ActionDetail*`。
- 对照已有 `internal/services/auth/authority.go`，确认本地 typed payload 是仓库内已采用的稳定修复模式。
- 如果默认 `wails generate module` 失败而 `GOWORK=off` 成功，继续检查根 `go.work` 是否纳入了与 Win repo-local replace 冲突的模块，例如 `repo/MyFlowHub-Proto`。

## Resolution
- 在 Win 侧新增本地 exported typed payload 和 action 常量。
- 保持 JSON 字段契约不变，只替换 service 方法的 Go 类型依赖。
- 补充最小单元测试，并用 `GOWORK=off` 执行 `go test ./...` 与 `wails generate module`。
- 若问题来自根脚本或 root workspace，优先让 Win 的 Wails 启动默认带 `GOWORK=off`，把 workspace 模式改为显式 opt-in，而不是继续让调用者手动记忆。

## Prevention / Guardrails
- 新增 Win Wails binding 前，先检查 shared proto 是否已定义对应 req/resp/action。
- worktree 下所有 Go / Wails 验证默认使用 `GOWORK=off`。
- 如果 shared proto 还没准备好，但前端/Win 需要先落地，优先使用 Win 本地 typed payload，并在 spec 中澄清这是实现边界而非协议扩展。
- 对 root 级启动脚本，如果目标是“稳定冒烟 / 默认可运行”，Win 默认应优先使用 `GOWORK=off`，不要隐式继承整个 workspace 的联调模块图。

## Related Docs
- [2026-03-26_win-flow-detail-bindings.md](../change/2026-03-26_win-flow-detail-bindings.md)
- [2026-03-26_win-authority-console-refactor.md](../change/2026-03-26_win-authority-console-refactor.md)
- [2026-03-29_root-run-dev-wails-gowork.md](../change/2026-03-29_root-run-dev-wails-gowork.md)
- [flow-editor-run-detail.md](../specs/flow-editor-run-detail.md)
