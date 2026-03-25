# Cross Repo Semver Release

## Summary
- 跨仓 API 变更如果只在 workspace 本地 `go.work` 联调通过，仍可能在真实下游仓库的 `GOWORK=off` 模式下失败。
- 这类问题通常不是业务逻辑 bug，而是“上游 tag 未发布”“下游 `go.mod` 未对齐”，或者“下游 CI 仍 checkout 到旧的 sibling repo 基线”的发布链问题。
- 对 `Server/defaultset` 这类跨多个 subproto module 的装配入口，最先报出的 1 到 2 个缺符号通常只是入口症状，根因往往藏在更上游的同链路 module 或协议版本。

## Lookup Hints
- 症状：
  - `go.work` 下测试全部通过
  - `GOWORK=off go test` 编译失败
  - 本地 worktree 测试通过，但远端 CI/tag release 仍打出旧行为
- 关键词：
  - `GOWORK=off`
  - `go list -m`
  - `undefined: filehandler.NewHandlerWithDeps`
  - `undefined: management.NewHandlerWithDeps`
  - `undefined: broker.SharedExecCapQueryBroker`
  - `undefined: protocol.ActionDelete`
  - `no required module provides package github.com/yttydcs/myflowhub-subproto/exec/runtimedeps`
  - `undefined: config.KeyParentJoinPermit`
  - `undefined: bootstrap.SelfRegisterOptions.JoinPermit`
  - `Checkout Server (for hubmobile replace)`
  - `actions/checkout`
  - `default branch`
- 快速检查：
  - 对目标版本执行 `go list -m`
  - 检查下游 `go.mod` 是否已升级到本轮发布版本
  - 检查下游 CI 是否用 `replace` 拉 sibling repo，以及它实际 checkout 的 ref 是什么

## Symptoms
- 代码在本地多仓联调环境可编译、可测试。
- 一旦关闭 `go.work` 或切到单仓环境，立即出现找不到新类型、常量或字段的错误。
- 常见表现是：修完首个缺符号后，继续暴露 sibling module 的 shared package 缺失，或继续暴露更上游 `Proto` 契约缺失。
- 或者本地 workflow worktree 测试通过，但远端 release workflow 仍因为 checkout 到旧 sibling repo 基线而产出旧行为。

## Impact
- 阻塞 semver 发布和下游依赖升级。
- 容易让 workflow 在“本地看起来全绿”时被过早判定完成。
- 如果不显式记录，后续团队成员会重复踩同一个坑。

## Trigger Conditions
- 本轮改动跨越 `Proto`、`Core`、`SubProto`、`Server` 等多个仓库。
- 下游仓库依赖了上游刚新增的 API，但上游对应 tag 还没有发布。
- 仅使用 workspace 本地 `go.work` 做联调验证，没有补 `GOWORK=off` 检查。
- 下游仓库通过 `replace` 依赖 sibling repo，而 CI/release workflow checkout 的是该 sibling repo 的远端默认分支或旧基线。

## Root Cause
- `go.work` 会把本地 worktree 直接接到编译链上，掩盖远端 semver 依赖尚未发布或下游版本尚未升级的问题。
- 同样地，`replace ../../SomeRepo` 在本地 workflow worktree 可以指向最新 sibling worktree，但远端 CI 如果只是 checkout 远端默认分支，就不会自动拿到你本地尚未公开的代码。
- 一旦切回真实依赖解析或真实 CI checkout 模式，隐藏问题就会立刻暴露。

## Investigation Trail
1. 在 workflow-local `go.work` 下确认功能与测试都通过。
2. 切换到 `GOWORK=off` 后，在真实下游复现缺符号或依赖不一致问题。
3. 反查缺失符号对应的上游来源，确认新 API 仍只存在于本地 worktree。
4. 若下游使用 sibling repo `replace`，继续检查 CI/release workflow 中 `actions/checkout` 拉取的是哪个 repo、哪个 ref。
5. 先发布上游 tag，或先把 CI 依赖的 sibling repo 基线公开，再执行下游 release。

## Resolution
- 发布：
  - `myflowhub-proto v0.1.2`
  - `myflowhub-core v0.4.8`
  - `myflowhub-subproto/auth v0.1.4`
  - `myflowhub-sdk v0.1.11`
  - `myflowhub-win v0.0.11`
- 将 `MyFlowHub-Server` 对齐到上述 auth admission 上游版本。
- 对关键下游补充 `GOWORK=off` 构建和测试验证，作为 workflow 收口条件。
- 对仍依赖 sibling repo `replace` 的应用仓，在确认 CI checkout 到正确上游基线前，不要直接发 release tag。

### Defaultset release-chain example
- 发布：
  - `myflowhub-proto v0.1.3`
  - `myflowhub-subproto/broker v0.1.1`
  - `myflowhub-subproto/exec v0.1.2`
  - `myflowhub-subproto/file v0.1.4`
  - `myflowhub-subproto/flow v0.1.2`
  - `myflowhub-subproto/topicbus v0.1.2`
  - `myflowhub-subproto/varstore v0.1.4`
  - `myflowhub-subproto/management v0.1.4`
- 将 `MyFlowHub-Server` 对齐到上述版本，并在 `GOWORK=off` 下执行 `go build ./...` 和 `go test ./... -count=1 -p 1`。
- 如果 first error 是 `NewHandlerWithDeps`，不要止步于 `file/management` 两个 module；继续检查 `broker`、`exec` 与 `Proto` 契约版本。

## Prevention / Guardrails
- 只要本轮改动触达上游公共 API，结束前必须至少执行一次真实下游的 `GOWORK=off go test`。
- 发布顺序按依赖方向执行：`Proto/Core -> SubProto module -> SDK/Server -> 应用层`。
- 同仓 shared package 新增后，继续沿依赖方向检查所有 sibling modules 的最小版本，而不是只修首个报错 module。
- 如果下游 CI 依赖 `replace` 到 sibling repo，必须检查该 sibling repo 的远端默认分支或 checkout ref 是否已经包含本轮代码。
- 已 push 的 semver tag 不重写；发现问题时发更高 patch 版本。
- 不要把 `go.work` 通过误判成“可发布”证据；它只能证明本地联调成功。

## Related Docs
- Requirements:
  - [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)
- Specs:
  - [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
- Changes:
  - [2026-03-24_auth-admission-downstream-followup.md](../change/2026-03-24_auth-admission-downstream-followup.md)
  - [2026-03-23_auth-controlled-admission.md](../change/2026-03-23_auth-controlled-admission.md)
  - [2026-03-24_auth-authority-strict-selection.md](../change/2026-03-24_auth-authority-strict-selection.md)
  - [2026-03-24_auth-admission-release-chain.md](../change/2026-03-24_auth-admission-release-chain.md)
  - [2026-03-25_defaultset-deps-release-chain.md](../change/2026-03-25_defaultset-deps-release-chain.md)
